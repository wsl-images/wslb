package wsl

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/wsl-images/wslb/internal/platform/windows"
	"github.com/wsl-images/wslb/internal/workspace"
)

type StateCreateOptions struct {
	FallbackWindowsDir bool
	NonInteractive     bool
}

type VersionRecord struct {
	VersionID     string    `json:"versionId"`
	CreatedAt     time.Time `json:"createdAt"`
	ArtifactPath  string    `json:"artifactPath"`
	BackupPath    string    `json:"backupPath,omitempty"`
	StableName    string    `json:"stableName"`
	CandidateName string    `json:"candidateName,omitempty"`
	StateMode     string    `json:"stateMode"`
	StatePath     string    `json:"statePath"`
	LastKnownGood bool      `json:"lastKnownGood"`
}

type History struct {
	ImageID  string          `json:"imageId"`
	Versions []VersionRecord `json:"versions"`
}

type Service struct{}

func NewService() *Service { return &Service{} }

func (s *Service) StateCreate(ctx context.Context, manifestPath string, image workspace.Image, opts StateCreateOptions) (workspace.Image, error) {
	if runtime.GOOS != "windows" {
		return image, errors.New("wsl commands are only supported on Windows")
	}
	if image.WSL == nil || image.WSL.State == nil {
		return image, fmt.Errorf("image %s missing wsl.state config", image.ID)
	}
	state := image.WSL.State
	state.Path = workspace.ResolvePath(manifestPath, state.Path)

	switch state.Mode {
	case "windows-dir":
		if err := os.MkdirAll(state.Path, 0o755); err != nil {
			return image, err
		}
		image.WSL.State.Path = state.Path
		return image, nil
	case "vhdx":
		if _, err := os.Stat(state.Path); err == nil {
			image.WSL.State.Path = state.Path
			return image, nil
		}
		admin := isAdmin(ctx)
		if !admin {
			if !opts.NonInteractive && isInteractiveTTY() {
				if askYesNo("WSL VHDX creation likely needs elevation. Fallback to windows-dir for this run? [y/N]: ") {
					if !opts.FallbackWindowsDir {
						opts.FallbackWindowsDir = true
					}
				} else {
					return image, elevationErr(state.Path)
				}
			} else if !opts.FallbackWindowsDir {
				return image, elevationErr(state.Path)
			}
		}
		if opts.FallbackWindowsDir {
			fallback := state.Path + "-dir"
			if err := os.MkdirAll(fallback, 0o755); err != nil {
				return image, err
			}
			image.WSL.State.Mode = "windows-dir"
			image.WSL.State.Path = fallback
			return image, nil
		}

		if err := createVHDX(state.Path); err != nil {
			return image, fmt.Errorf("failed to create VHDX: %w", err)
		}
		if err := mountVHDX(ctx, state.Path); err != nil {
			return image, fmt.Errorf("created VHDX but could not mount it. run elevated and format ext4 manually: %w", err)
		}
		image.WSL.State.Path = state.Path
		return image, nil
	default:
		return image, fmt.Errorf("unsupported state mode: %s", state.Mode)
	}
}

func (s *Service) Install(ctx context.Context, manifestPath string, image workspace.Image, artifactPath string) error {
	if image.WSL == nil {
		return fmt.Errorf("image %s is not WSL target", image.ID)
	}
	installDir := workspace.ResolvePath(manifestPath, image.WSL.InstallDir)
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return err
	}
	installed := false
	if err := tryInstallFromWSLPackage(ctx, image.WSL.DistroName, installDir, artifactPath); err == nil {
		installed = true
	} else if isFromFileUnsupported(err) {
		if _, impErr := runCombined(ctx, "wsl", "--import", image.WSL.DistroName, installDir, artifactPath, "--version", "2"); impErr != nil {
			return impErr
		}
		installed = true
	} else {
		return err
	}
	if !installed {
		return fmt.Errorf("failed to install distro %s", image.WSL.DistroName)
	}
	if err := configureWSLForDistro(ctx, manifestPath, image, image.WSL.DistroName); err != nil {
		return err
	}
	ok, err := verifyDistro(ctx, image, image.WSL.DistroName)
	if err != nil {
		return fmt.Errorf("post-install verification failed: %w", err)
	}
	if !ok {
		return fmt.Errorf("post-install verification failed")
	}
	if err := ensureWindowsShortcut(ctx, manifestPath, image); err != nil {
		return err
	}
	return nil
}

func tryInstallFromWSLPackage(ctx context.Context, distroName, installDir, artifactPath string) error {
	_, err := runCombined(ctx, "wsl", "--install", "--from-file", artifactPath, "--name", distroName, "--location", installDir, "--no-launch")
	return err
}

func isFromFileUnsupported(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "from-file") && (strings.Contains(msg, "unrecognized") || strings.Contains(msg, "unknown") || strings.Contains(msg, "invalid"))
}

func (s *Service) Upgrade(ctx context.Context, manifestPath, outputDir string, image workspace.Image, artifactPath string) (string, error) {
	if image.WSL == nil {
		return "", fmt.Errorf("image %s is not WSL target", image.ID)
	}
	stable := image.WSL.DistroName
	plan := BuildSwapPlan(stable, false, time.Now())

	candidateInstallDir := filepath.Join(workspace.ResolvePath(manifestPath, image.WSL.InstallDir), "candidate")
	_ = os.MkdirAll(candidateInstallDir, 0o755)
	if _, err := runCombined(ctx, "wsl", "--import", plan.CandidateName, candidateInstallDir, artifactPath, "--version", "2"); err != nil {
		return "", err
	}
	if err := configureWSLForDistro(ctx, manifestPath, image, plan.CandidateName); err != nil {
		return "", fmt.Errorf("candidate configuration failed: %w", err)
	}

	verified, verr := verifyDistro(ctx, image, plan.CandidateName)
	plan.AllowSwap = verified
	if verr != nil {
		return "", fmt.Errorf("candidate verification failed: %w", verr)
	}
	if !plan.AllowSwap {
		return "", fmt.Errorf("candidate verification did not pass; refusing to unregister stable distro")
	}

	backupDir := filepath.Join(workspace.ResolvePath(manifestPath, outputDir), "state", image.ID, "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}
	backupTar := filepath.Join(backupDir, plan.BackupName+".tar")
	if _, err := runCombined(ctx, "wsl", "--export", stable, backupTar); err != nil {
		return "", fmt.Errorf("failed to export stable backup: %w", err)
	}

	_, _ = runCombined(ctx, "wsl", "--terminate", stable)
	if _, err := runCombined(ctx, "wsl", "--unregister", stable); err != nil {
		return "", fmt.Errorf("failed to unregister stable after successful verification: %w", err)
	}

	stableInstallDir := workspace.ResolvePath(manifestPath, image.WSL.InstallDir)
	if err := os.MkdirAll(stableInstallDir, 0o755); err != nil {
		return "", err
	}
	if _, err := runCombined(ctx, "wsl", "--import", stable, stableInstallDir, artifactPath, "--version", "2"); err != nil {
		return "", err
	}
	if err := configureWSLForDistro(ctx, manifestPath, image, stable); err != nil {
		return "", err
	}
	if err := ensureWindowsShortcut(ctx, manifestPath, image); err != nil {
		return "", err
	}
	_, _ = runCombined(ctx, "wsl", "--unregister", plan.CandidateName)

	if err := appendHistory(outputDir, image, VersionRecord{
		VersionID:     fmt.Sprintf("%d", time.Now().UTC().Unix()),
		CreatedAt:     time.Now().UTC(),
		ArtifactPath:  artifactPath,
		BackupPath:    backupTar,
		StableName:    stable,
		CandidateName: plan.CandidateName,
		StateMode:     image.WSL.State.Mode,
		StatePath:     image.WSL.State.Path,
		LastKnownGood: true,
	}); err != nil {
		return backupTar, err
	}
	return backupTar, nil
}

func (s *Service) Rollback(ctx context.Context, manifestPath, outputDir string, image workspace.Image, backupPath string) error {
	if image.WSL == nil {
		return fmt.Errorf("image %s is not WSL target", image.ID)
	}
	if backupPath == "" {
		h, err := readHistory(outputDir, image.ID)
		if err != nil {
			return err
		}
		for i := len(h.Versions) - 1; i >= 0; i-- {
			if h.Versions[i].BackupPath != "" {
				backupPath = h.Versions[i].BackupPath
				break
			}
		}
		if backupPath == "" {
			return fmt.Errorf("no backup available for rollback")
		}
	} else if _, err := os.Stat(backupPath); err != nil {
		h, herr := readHistory(outputDir, image.ID)
		if herr != nil {
			return fmt.Errorf("backup path not found and history unavailable: %w", err)
		}
		found := ""
		for _, v := range h.Versions {
			if v.VersionID == backupPath && v.BackupPath != "" {
				found = v.BackupPath
				break
			}
		}
		if found == "" {
			return fmt.Errorf("backup %q not found as path or version id", backupPath)
		}
		backupPath = found
	}

	stable := image.WSL.DistroName
	_, _ = runCombined(ctx, "wsl", "--terminate", stable)
	_, _ = runCombined(ctx, "wsl", "--unregister", stable)
	installDir := workspace.ResolvePath(manifestPath, image.WSL.InstallDir)
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return err
	}
	if _, err := runCombined(ctx, "wsl", "--import", stable, installDir, backupPath, "--version", "2"); err != nil {
		return err
	}
	if err := configureWSLForDistro(ctx, manifestPath, image, stable); err != nil {
		return err
	}
	if err := ensureWindowsShortcut(ctx, manifestPath, image); err != nil {
		return err
	}
	return nil
}

func (s *Service) Status(ctx context.Context, image workspace.Image) (string, error) {
	if image.WSL == nil {
		return "", fmt.Errorf("image %s is not WSL target", image.ID)
	}
	out, err := runCombined(ctx, "wsl", "-l", "-v")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, image.WSL.DistroName) {
			return strings.TrimSpace(line), nil
		}
	}
	return "not installed", nil
}

func configureWSLForDistro(ctx context.Context, manifestPath string, image workspace.Image, distro string) error {
	if image.WSL == nil {
		return nil
	}
	if err := applyGlobalWSLConfig(ctx, image); err != nil {
		return err
	}
	if err := ensureDistroDefaultUser(ctx, image, distro); err != nil {
		return err
	}

	wslConf := renderWSLConf(image)
	if err := writeTextFileToDistro(ctx, distro, "/etc/wsl.conf", wslConf, 0o644); err != nil {
		return err
	}

	distributionConf, err := renderDistributionConf(ctx, manifestPath, image, distro)
	if err != nil {
		return err
	}
	if distributionConf != "" {
		if err := writeTextFileToDistro(ctx, distro, "/etc/wsl-distribution.conf", distributionConf, 0o644); err != nil {
			return err
		}
	}
	if distributionConf == "" && image.WSL.Distribution != nil {
		if _, err := runCombined(ctx, "wsl", "-d", distro, "-u", "root", "--", "sh", "-lc", "rm -f /etc/wsl-distribution.conf"); err != nil {
			return err
		}
	}
	if _, err := runCombined(ctx, "wsl", "-d", distro, "-u", "root", "--", "sh", "-lc", "test -s /etc/wsl.conf"); err != nil {
		return err
	}

	if image.WSL.Managed && image.WSL.State != nil {
		if err := configureManagedStateMount(ctx, distro, image); err != nil {
			return err
		}
	}
	_, _ = runCombined(ctx, "wsl", "--terminate", distro)
	return nil
}

func renderWSLConf(image workspace.Image) string {
	cfg := image.WSL.WSLConf
	defaultUser := image.WSL.DefaultUser.Name
	if cfg.User == nil {
		cfg.User = &workspace.WSLConfUser{}
	}
	if strings.TrimSpace(cfg.User.Default) == "" {
		cfg.User.Default = defaultUser
	}
	if cfg.Boot == nil {
		cfg.Boot = &workspace.WSLConfBoot{}
	}
	if cfg.Boot.Systemd == nil {
		sys := true
		cfg.Boot.Systemd = &sys
	}
	if cfg.Automount == nil {
		cfg.Automount = &workspace.WSLConfAutomount{}
	}
	if cfg.Automount.Enabled == nil {
		enabled := true
		cfg.Automount.Enabled = &enabled
	}
	if cfg.Automount.MountFsTab == nil {
		mfstab := true
		cfg.Automount.MountFsTab = &mfstab
	}

	var b strings.Builder
	b.WriteString("[user]\n")
	b.WriteString("default=" + cfg.User.Default + "\n")
	b.WriteString("[boot]\n")
	b.WriteString("systemd=" + boolToString(*cfg.Boot.Systemd) + "\n")
	if strings.TrimSpace(cfg.Boot.Command) != "" {
		b.WriteString("command=" + cfg.Boot.Command + "\n")
	}
	if cfg.Boot.ProtectBinfmt != nil {
		b.WriteString("protectBinfmt=" + boolToString(*cfg.Boot.ProtectBinfmt) + "\n")
	}
	b.WriteString("[automount]\n")
	b.WriteString("enabled=" + boolToString(*cfg.Automount.Enabled) + "\n")
	b.WriteString("mountFsTab=" + boolToString(*cfg.Automount.MountFsTab) + "\n")
	if strings.TrimSpace(cfg.Automount.Root) != "" {
		b.WriteString("root=" + cfg.Automount.Root + "\n")
	}
	if strings.TrimSpace(cfg.Automount.Options) != "" {
		b.WriteString("options=" + cfg.Automount.Options + "\n")
	}
	if cfg.Automount.CrossDistro != nil {
		b.WriteString("crossDistro=" + boolToString(*cfg.Automount.CrossDistro) + "\n")
	}
	if cfg.Network != nil {
		if strings.TrimSpace(cfg.Network.Hostname) != "" || cfg.Network.GenerateHosts != nil || cfg.Network.GenerateResolvConf != nil {
			b.WriteString("[network]\n")
			if strings.TrimSpace(cfg.Network.Hostname) != "" {
				b.WriteString("hostname=" + cfg.Network.Hostname + "\n")
			}
			if cfg.Network.GenerateHosts != nil {
				b.WriteString("generateHosts=" + boolToString(*cfg.Network.GenerateHosts) + "\n")
			}
			if cfg.Network.GenerateResolvConf != nil {
				b.WriteString("generateResolvConf=" + boolToString(*cfg.Network.GenerateResolvConf) + "\n")
			}
		}
	}
	if cfg.Interop != nil {
		if cfg.Interop.Enabled != nil || cfg.Interop.AppendWindowsPath != nil {
			b.WriteString("[interop]\n")
			if cfg.Interop.Enabled != nil {
				b.WriteString("enabled=" + boolToString(*cfg.Interop.Enabled) + "\n")
			}
			if cfg.Interop.AppendWindowsPath != nil {
				b.WriteString("appendWindowsPath=" + boolToString(*cfg.Interop.AppendWindowsPath) + "\n")
			}
		}
	}
	if cfg.GPU != nil && cfg.GPU.Enabled != nil {
		b.WriteString("[gpu]\n")
		b.WriteString("enabled=" + boolToString(*cfg.GPU.Enabled) + "\n")
	}
	if cfg.Time != nil && cfg.Time.UseWindowsTimezone != nil {
		b.WriteString("[time]\n")
		b.WriteString("useWindowsTimezone=" + boolToString(*cfg.Time.UseWindowsTimezone) + "\n")
	}
	return b.String()
}

func applyGlobalWSLConfig(ctx context.Context, image workspace.Image) error {
	if image.WSL == nil || image.WSL.WSLConfig == nil {
		return nil
	}
	cfg := image.WSL.WSLConfig
	if cfg.Apply != nil && !*cfg.Apply {
		return nil
	}
	if len(cfg.WSL2) == 0 && len(cfg.Experimental) == 0 {
		return nil
	}

	userProfile := strings.TrimSpace(os.Getenv("USERPROFILE"))
	if userProfile == "" {
		return fmt.Errorf("USERPROFILE is required to apply .wslconfig")
	}
	targetPath := filepath.Join(userProfile, ".wslconfig")

	rendered, err := renderGlobalWSLConfig(*cfg)
	if err != nil {
		return err
	}
	if strings.TrimSpace(rendered) == "" {
		return nil
	}

	existing, readErr := os.ReadFile(targetPath)
	if readErr == nil && strings.TrimSpace(string(existing)) == strings.TrimSpace(rendered) {
		return nil
	}
	if readErr == nil {
		backupPath := targetPath + ".wslb.bak"
		_ = os.WriteFile(backupPath, existing, 0o644)
	}
	if err := os.WriteFile(targetPath, []byte(rendered), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", targetPath, err)
	}
	if _, err := runCombined(ctx, "wsl", "--shutdown"); err != nil {
		return fmt.Errorf("updated .wslconfig but failed to restart WSL VM: %w", err)
	}
	return nil
}

func renderGlobalWSLConfig(cfg workspace.WSLGlobalConfig) (string, error) {
	var b strings.Builder
	writeSection := func(name string, kv map[string]interface{}) error {
		if len(kv) == 0 {
			return nil
		}
		keys := make([]string, 0, len(kv))
		for k := range kv {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteString("[" + name + "]\n")
		for _, k := range keys {
			v := kv[k]
			if v == nil {
				continue
			}
			rendered, err := renderWSLGlobalValue(v)
			if err != nil {
				return fmt.Errorf(".wslconfig %s.%s: %w", name, k, err)
			}
			b.WriteString(k + "=" + rendered + "\n")
		}
		b.WriteString("\n")
		return nil
	}
	if err := writeSection("wsl2", cfg.WSL2); err != nil {
		return "", err
	}
	if err := writeSection("experimental", cfg.Experimental); err != nil {
		return "", err
	}
	return strings.TrimSpace(b.String()) + "\n", nil
}

func renderWSLGlobalValue(v interface{}) (string, error) {
	switch t := v.(type) {
	case bool:
		return boolToString(t), nil
	case string:
		return t, nil
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), nil
	case float32:
		return strconv.FormatFloat(float64(t), 'f', -1, 32), nil
	case int:
		return strconv.Itoa(t), nil
	case int64:
		return strconv.FormatInt(t, 10), nil
	case int32:
		return strconv.FormatInt(int64(t), 10), nil
	case uint:
		return strconv.FormatUint(uint64(t), 10), nil
	case uint64:
		return strconv.FormatUint(t, 10), nil
	case uint32:
		return strconv.FormatUint(uint64(t), 10), nil
	case json.Number:
		return t.String(), nil
	default:
		return "", fmt.Errorf("unsupported value type %T (use string/number/bool)", v)
	}
}

func ensureDistroDefaultUser(ctx context.Context, image workspace.Image, distro string) error {
	if image.WSL == nil {
		return nil
	}
	u := strings.TrimSpace(image.WSL.DefaultUser.Name)
	if u == "" {
		u = "dev"
	}
	uid := image.WSL.DefaultUser.UID
	if uid == 0 {
		uid = 1000
	}
	gid := image.WSL.DefaultUser.GID
	if gid == 0 {
		gid = 1000
	}
	homeBase := "/home"
	if image.WSL.State != nil && strings.TrimSpace(image.WSL.State.MountPoint) != "" {
		homeBase = strings.TrimSpace(image.WSL.State.MountPoint)
	}
	home := managedUserHome(homeBase, u)
	cmdText := fmt.Sprintf(
		`set -eu; u=%s; target_uid=%d; target_gid=%d; `+
			`if ! getent group "$target_gid" >/dev/null 2>&1; then groupadd -g "$target_gid" "$u" || true; fi; `+
			`if ! id -u "$u" >/dev/null 2>&1; then useradd -m -u "$target_uid" -g "$target_gid" -s /bin/bash "$u" || useradd -m -s /bin/bash "$u" || true; fi; `+
			`uid="$(id -u "$u" 2>/dev/null || echo "$target_uid")"; gid="$(id -g "$u" 2>/dev/null || echo "$target_gid")"; `+
			`mkdir -p %s; chown -R "$uid:$gid" %s || true`,
		shQuote(u), uid, gid, shQuote(home), shQuote(home),
	)
	if _, err := runCombined(ctx, "wsl", "-d", distro, "-u", "root", "--", "sh", "-lc", cmdText); err != nil {
		return fmt.Errorf("failed to ensure default user %s: %w", u, err)
	}
	return nil
}

func boolToString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func verifyDistro(ctx context.Context, image workspace.Image, distro string) (bool, error) {
	if image.WSL == nil {
		return false, fmt.Errorf("missing WSL config")
	}
	if image.WSL.Managed && image.WSL.State != nil {
		mp := image.WSL.State.MountPoint
		if mp == "" {
			mp = "/home"
		}
		test := fmt.Sprintf("set -eu; mkdir -p %q; mountpoint -q %q || { echo \"managed mountpoint not active: %s\" >&2; exit 32; }; touch %q/.wslb-verify; cat %q/.wslb-verify >/dev/null", mp, mp, mp, mp, mp)
		if _, err := runCombined(ctx, "wsl", "-d", distro, "-u", "root", "--", "sh", "-lc", test); err != nil {
			return false, err
		}
	}
	userHome := managedUserHome("/home", image.WSL.DefaultUser.Name)
	if image.WSL.State != nil && image.WSL.State.MountPoint != "" {
		userHome = managedUserHome(image.WSL.State.MountPoint, image.WSL.DefaultUser.Name)
	}
	userCheck := fmt.Sprintf("set -eu; mkdir -p %q; touch %q/.wslb-user-verify; cat %q/.wslb-user-verify >/dev/null", userHome, userHome, userHome)
	if _, err := runCombined(ctx, "wsl", "-d", distro, "-u", image.WSL.DefaultUser.Name, "--", "sh", "-lc", userCheck); err != nil {
		return false, err
	}
	tools := expectedFeatureTools(image)
	if len(tools) > 0 {
		quoted := make([]string, 0, len(tools))
		for _, t := range tools {
			quoted = append(quoted, fmt.Sprintf("%q", t))
		}
		cmd := fmt.Sprintf("set -eu; missing=''; for t in %s; do command -v \"$t\" >/dev/null 2>&1 || missing=\"$missing $t\"; done; [ -z \"$missing\" ] || { echo \"missing feature tools:$missing\" >&2; exit 33; }", strings.Join(quoted, " "))
		if _, err := runCombined(ctx, "wsl", "-d", distro, "--", "sh", "-lc", cmd); err != nil {
			return false, err
		}
	}
	return true, nil
}

func expectedFeatureTools(image workspace.Image) []string {
	if image.WSL == nil {
		return nil
	}
	set := map[string]struct{}{}
	for _, fa := range image.WSL.Features {
		ref := strings.ToLower(strings.TrimSpace(fa.Ref))
		switch {
		case strings.Contains(ref, "/features/node:"):
			set["node"] = struct{}{}
			set["npm"] = struct{}{}
		case strings.Contains(ref, "/features/go:"):
			set["go"] = struct{}{}
		case strings.Contains(ref, "/features/python:"):
			set["python3"] = struct{}{}
		case strings.Contains(ref, "/features/git:"):
			set["git"] = struct{}{}
		case strings.Contains(ref, "/features/github-cli:"):
			set["gh"] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func configureManagedStateMount(ctx context.Context, distro string, image workspace.Image) error {
	if image.WSL == nil || image.WSL.State == nil {
		return fmt.Errorf("missing managed state configuration")
	}
	dst := image.WSL.State.MountPoint
	if dst == "" {
		dst = "/home"
	}
	dstEsc := strings.ReplaceAll(dst, " ", `\040`)
	userName := strings.TrimSpace(image.WSL.DefaultUser.Name)
	if userName == "" {
		userName = "dev"
	}
	uid := image.WSL.DefaultUser.UID
	gid := image.WSL.DefaultUser.GID
	if uid == 0 {
		uid = 1000
	}
	if gid == 0 {
		gid = 1000
	}

	switch image.WSL.State.Mode {
	case "windows-dir":
		windowsSrc := windowsPathToWSLArg(image.WSL.State.Path)
		linuxSrc, err := resolveManagedStateSource(ctx, distro, image)
		if err != nil {
			return err
		}
		srcEsc := strings.ReplaceAll(windowsSrc, " ", `\040`)
		mountOpts := fmt.Sprintf("metadata,uid=%d,gid=%d", uid, gid)
		userHome := managedUserHome(dst, image.WSL.DefaultUser.Name)
		cmd := fmt.Sprintf(
			`set -eu; if [ ! -d %q ]; then echo "state source does not exist: %s" >&2; exit 21; fi; `+
				`u=%s; `+
				`mkdir -p %q; if [ -f /etc/fstab ]; then grep -v ' # wslb-state$' /etc/fstab > /etc/fstab.wslb || true; else : > /etc/fstab.wslb; fi; mv /etc/fstab.wslb /etc/fstab; `+
				`echo '%s %s drvfs %s 0 0 # wslb-state' >> /etc/fstab; `+
				`mountpoint -q %q || mount -a || mount -t drvfs %q %q -o %s; `+
				`mountpoint -q %q || { echo "managed state mount not active at %s" >&2; exit 22; }; `+
				`mkdir -p %q; chown %d:%d %q || true`,
			linuxSrc, windowsSrc, shQuote(userName), dst, srcEsc, dstEsc, mountOpts, dst, windowsSrc, dst, mountOpts, dst, dst, userHome, uid, gid, userHome,
		)
		if _, err := runCombined(ctx, "wsl", "-d", distro, "-u", "root", "--", "sh", "-lc", cmd); err != nil {
			return fmt.Errorf("failed to configure managed state mount: %w", err)
		}
		return nil
	case "vhdx":
		src, err := resolveManagedStateSource(ctx, distro, image)
		if err != nil {
			return err
		}
		srcEsc := strings.ReplaceAll(src, " ", `\040`)
		userHome := managedUserHome(dst, image.WSL.DefaultUser.Name)
		cmd := fmt.Sprintf(
			`set -eu; if [ ! -d %q ]; then echo "state source does not exist: %s" >&2; exit 21; fi; `+
				`u=%s; `+
				`mkdir -p %q; if [ -f /etc/fstab ]; then grep -v ' # wslb-state$' /etc/fstab > /etc/fstab.wslb || true; else : > /etc/fstab.wslb; fi; mv /etc/fstab.wslb /etc/fstab; `+
				`echo '%s %s none bind 0 0 # wslb-state' >> /etc/fstab; `+
				`mountpoint -q %q || mount -a || mount --bind %q %q; `+
				`mountpoint -q %q || { echo "managed state mount not active at %s" >&2; exit 22; }; `+
				`mkdir -p %q; chown %d:%d %q || true`,
			src, src, shQuote(userName), dst, srcEsc, dstEsc, dst, src, dst, dst, dst, userHome, uid, gid, userHome,
		)
		if _, err := runCombined(ctx, "wsl", "-d", distro, "-u", "root", "--", "sh", "-lc", cmd); err != nil {
			return fmt.Errorf("failed to configure managed state mount: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported state mode for managed mount: %s", image.WSL.State.Mode)
	}
}

func resolveManagedStateSource(ctx context.Context, distro string, image workspace.Image) (string, error) {
	if image.WSL == nil || image.WSL.State == nil {
		return "", fmt.Errorf("missing managed state configuration")
	}
	if image.WSL.State.Mode == "vhdx" {
		label := strings.TrimSpace(image.WSL.State.FSLabel)
		if label == "" {
			return "", fmt.Errorf("state.fsLabel is required for vhdx managed mode")
		}
		return "/mnt/wsl/" + label, nil
	}
	src := image.WSL.State.Path
	if image.WSL.State.Mode != "windows-dir" {
		return src, nil
	}

	// Fast path for drive-letter Windows paths.
	if linuxPath, ok := windowsPathToLinuxDrive(src); ok {
		return linuxPath, nil
	}

	// WSL argument parsing strips backslashes in Windows paths; convert to slash form before wslpath.
	arg := windowsPathToWSLArg(src)
	out, err := runCombined(ctx, "wsl", "-d", distro, "-u", "root", "--", "wslpath", arg)
	if err != nil {
		return "", fmt.Errorf("failed to resolve windows-dir path %q in distro %s: %w", src, distro, err)
	}
	linuxPath := strings.TrimSpace(out)
	if linuxPath == "" {
		return "", fmt.Errorf("resolved empty Linux path for state source %q", src)
	}
	return linuxPath, nil
}

func managedUserHome(mountPoint, user string) string {
	mp := strings.TrimSpace(mountPoint)
	if mp == "" {
		mp = "/home"
	}
	u := strings.TrimSpace(user)
	if u == "" {
		u = "dev"
	}
	if mp == "/" {
		return "/" + u
	}
	return strings.TrimRight(mp, "/") + "/" + u
}

func windowsPathToWSLArg(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

var windowsDrivePathPattern = regexp.MustCompile(`(?i)^([a-z]):[\\/](.*)$`)

func windowsPathToLinuxDrive(path string) (string, bool) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", false
	}
	normalized := strings.ReplaceAll(trimmed, "\\", "/")
	m := windowsDrivePathPattern.FindStringSubmatch(normalized)
	if len(m) != 3 {
		return "", false
	}
	drive := strings.ToLower(m[1])
	rest := strings.TrimLeft(m[2], "/")
	if rest == "" {
		return "/mnt/" + drive, true
	}
	return "/mnt/" + drive + "/" + rest, true
}

func createVHDX(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	script := fmt.Sprintf("create vdisk file=\"%s\" maximum=32768 type=expandable", path)
	tmp, err := os.CreateTemp("", "wslb-diskpart-*.txt")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(script + "\n"); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()
	out, err := exec.Command("diskpart", "/s", tmp.Name()).CombinedOutput()
	if err != nil {
		return fmt.Errorf("diskpart failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func mountVHDX(ctx context.Context, path string) error {
	_, err := runCombined(ctx, "wsl", "--mount", "--vhd", path, "--bare")
	return err
}

func elevationErr(path string) error {
	return fmt.Errorf("mounting VHDX requires elevated privileges. rerun as Administrator or fallback to windows-dir.\nadmin retry: wsl --mount --vhd \"%s\" --bare\nfallback retry: wslb wsl state create --fallback-windows-dir <imageId>", path)
}

func isAdmin(ctx context.Context) bool {
	return exec.CommandContext(ctx, "net", "session").Run() == nil
}

func isInteractiveTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func askYesNo(prompt string) bool {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}

func runCombined(ctx context.Context, bin string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	out, err := cmd.CombinedOutput()
	decoded := windows.DecodePossiblyUTF16(out)
	if err != nil {
		return decoded, fmt.Errorf("%s %s failed: %w\n%s", bin, strings.Join(args, " "), err, decoded)
	}
	return decoded, nil
}

func ensureWindowsShortcut(ctx context.Context, manifestPath string, image workspace.Image) error {
	if runtime.GOOS != "windows" {
		return nil
	}
	if image.WSL == nil || image.WSL.Distribution == nil || image.WSL.Distribution.Shortcut == nil {
		return nil
	}
	enabled := true
	if image.WSL.Distribution.Shortcut.Enabled != nil {
		enabled = *image.WSL.Distribution.Shortcut.Enabled
	}
	if !enabled {
		return nil
	}

	assets, err := resolveDistributionAssets(ctx, manifestPath, image)
	if err != nil {
		return err
	}
	if len(assets.IconBytes) == 0 {
		return nil
	}

	localAppData := os.Getenv("LOCALAPPDATA")
	appData := os.Getenv("APPDATA")
	if localAppData == "" || appData == "" {
		return fmt.Errorf("LOCALAPPDATA/APPDATA not available for shortcut creation")
	}

	iconDir := filepath.Join(localAppData, "wslb", "icons")
	if err := os.MkdirAll(iconDir, 0o755); err != nil {
		return err
	}
	iconPath := filepath.Join(iconDir, image.WSL.DistroName+".ico")
	if err := os.WriteFile(iconPath, assets.IconBytes, 0o644); err != nil {
		return err
	}

	startMenuDir := filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "WSLB")
	if err := os.MkdirAll(startMenuDir, 0o755); err != nil {
		return err
	}
	linkPath := filepath.Join(startMenuDir, image.WSL.DistroName+".lnk")
	args := fmt.Sprintf("-d %s --cd ~", image.WSL.DistroName)
	ps := fmt.Sprintf(
		`$w=New-Object -ComObject WScript.Shell; $s=$w.CreateShortcut('%s'); $s.TargetPath='wsl.exe'; $s.Arguments='%s'; $s.IconLocation='%s,0'; $s.WorkingDirectory='%s'; $s.Save()`,
		psSingleQuote(linkPath),
		psSingleQuote(args),
		psSingleQuote(iconPath),
		psSingleQuote(os.Getenv("USERPROFILE")),
	)
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", ps)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed creating Windows shortcut: %w\n%s", err, windows.DecodePossiblyUTF16(out))
	}
	return nil
}

func psSingleQuote(in string) string {
	return strings.ReplaceAll(in, `'`, `''`)
}

func historyPath(outputDir, imageID string) string {
	return filepath.Join(outputDir, "state", imageID, "versions.json")
}

func appendHistory(outputDir string, image workspace.Image, rec VersionRecord) error {
	path := historyPath(outputDir, image.ID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	h := History{ImageID: image.ID, Versions: []VersionRecord{}}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &h)
	}
	h.Versions = append(h.Versions, rec)
	b, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func readHistory(outputDir, imageID string) (History, error) {
	path := historyPath(outputDir, imageID)
	b, err := os.ReadFile(path)
	if err != nil {
		return History{}, err
	}
	var h History
	if err := json.Unmarshal(b, &h); err != nil {
		return History{}, err
	}
	return h, nil
}
