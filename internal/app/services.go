package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/wsl-images/wslb/internal/engine"
	"github.com/wsl-images/wslb/internal/features"
	dcprovider "github.com/wsl-images/wslb/internal/features/providers/devcontainer"
	intprovider "github.com/wsl-images/wslb/internal/features/providers/internalfeatures"
	platformwindows "github.com/wsl-images/wslb/internal/platform/windows"
	wslTarget "github.com/wsl-images/wslb/internal/targets/wsl"
	"github.com/wsl-images/wslb/internal/workspace"
)

type Services struct {
	Workspace WorkspaceService
	WSL       WSLService
	Doctor    DoctorService
	Catalog   CatalogService
}

func NewServices() *Services {
	router := features.NewRouter(
		dcprovider.New(filepath.Join(os.TempDir(), "wslb-feature-cache")),
		intprovider.New(filepath.Join(os.TempDir(), "wslb-internal-features")),
	)
	ws := &workspaceSvc{router: router}
	wsl := &wslSvc{router: router, target: wslTarget.NewService()}
	doc := &doctorSvc{}
	catalog := &catalogSvc{router: router}
	return &Services{Workspace: ws, WSL: wsl, Doctor: doc, Catalog: catalog}
}

type workspaceSvc struct {
	router *features.Router
}

func (s *workspaceSvc) LoadAndValidate(_ context.Context, manifestPath string) (*workspace.Manifest, string, workspace.ValidationResult, error) {
	m, abs, err := workspace.Load(manifestPath)
	if err != nil {
		return nil, abs, workspace.ValidationResult{}, err
	}
	vr := workspace.Validate(m, abs)
	return m, abs, vr, nil
}

func (s *workspaceSvc) Plan(ctx context.Context, manifestPath, imageID string) (workspace.Plan, workspace.ValidationResult, error) {
	m, abs, vr, err := s.LoadAndValidate(ctx, manifestPath)
	if err != nil {
		return workspace.Plan{}, vr, err
	}
	if !vr.Valid {
		return workspace.Plan{}, vr, fmt.Errorf("workspace validation failed")
	}
	p := workspace.BuildPlan(m, abs, imageID)
	return p, vr, nil
}

func (s *workspaceSvc) Build(ctx context.Context, manifestPath, imageID, preferredEngine string) error {
	m, abs, vr, err := s.LoadAndValidate(ctx, manifestPath)
	if err != nil {
		return err
	}
	if !vr.Valid {
		return fmt.Errorf("workspace validation failed")
	}

	for _, img := range m.Images {
		if imageID != "" && img.ID != imageID {
			continue
		}
		outDir := workspace.ResolvePath(abs, m.Workspace.OutputDir)
		if img.Target != "wsl" {
			continue
		}
		if _, err := wslTarget.BuildArtifact(ctx, abs, img, outDir, preferredEngine, s.router); err != nil {
			return err
		}
	}
	return nil
}

func (s *workspaceSvc) Publish(ctx context.Context, manifestPath, imageID string) error {
	m, _, vr, err := s.LoadAndValidate(ctx, manifestPath)
	if err != nil {
		return err
	}
	if !vr.Valid {
		return fmt.Errorf("workspace validation failed")
	}
	for _, img := range m.Images {
		if imageID != "" && img.ID != imageID {
			continue
		}
		if img.Target != "wsl" {
			continue
		}
	}
	return nil
}

type wslSvc struct {
	router *features.Router
	target *wslTarget.Service
}

func (s *wslSvc) loadImage(ctx context.Context, manifestPath, imageID string) (*workspace.Manifest, string, workspace.Image, error) {
	m, abs, err := workspace.Load(manifestPath)
	if err != nil {
		return nil, "", workspace.Image{}, err
	}
	vr := workspace.Validate(m, abs)
	if !vr.Valid {
		return nil, abs, workspace.Image{}, fmt.Errorf("workspace validation failed")
	}
	img, err := workspace.FindImage(m, imageID)
	if err != nil {
		return nil, abs, workspace.Image{}, err
	}
	if img.Target != "wsl" {
		return nil, abs, workspace.Image{}, fmt.Errorf("image %s target is %s, expected wsl", imageID, img.Target)
	}
	return m, abs, *img, nil
}

func (s *wslSvc) StateCreate(ctx context.Context, manifestPath, imageID string, fallbackWindowsDir bool, nonInteractive bool) (workspace.Image, error) {
	m, abs, img, err := s.loadImage(ctx, manifestPath, imageID)
	if err != nil {
		return workspace.Image{}, err
	}
	updated, err := s.target.StateCreate(ctx, abs, img, wslTarget.StateCreateOptions{FallbackWindowsDir: fallbackWindowsDir, NonInteractive: nonInteractive})
	if err != nil {
		return workspace.Image{}, err
	}
	for i := range m.Images {
		if m.Images[i].ID == updated.ID {
			m.Images[i] = updated
		}
	}
	return updated, nil
}

func (s *wslSvc) Install(ctx context.Context, manifestPath, imageID, preferredEngine string, fallbackWindowsDir bool, nonInteractive bool) (string, error) {
	m, abs, img, err := s.loadImage(ctx, manifestPath, imageID)
	if err != nil {
		return "", err
	}
	outDir := workspace.ResolvePath(abs, m.Workspace.OutputDir)
	build, err := wslTarget.BuildArtifact(ctx, abs, img, outDir, preferredEngine, s.router)
	if err != nil {
		return "", err
	}
	if img.WSL != nil && img.WSL.Managed && img.WSL.State != nil {
		updated, err := s.target.StateCreate(ctx, abs, img, wslTarget.StateCreateOptions{
			FallbackWindowsDir: fallbackWindowsDir,
			NonInteractive:     nonInteractive,
		})
		if err != nil {
			return "", err
		}
		img = updated
	}
	if err := s.target.Install(ctx, abs, img, build.ArtifactPath); err != nil {
		return "", err
	}
	return build.ArtifactPath, nil
}

func (s *wslSvc) Upgrade(ctx context.Context, manifestPath, imageID, preferredEngine string, fallbackWindowsDir bool, nonInteractive bool) (string, error) {
	m, abs, img, err := s.loadImage(ctx, manifestPath, imageID)
	if err != nil {
		return "", err
	}
	if img.WSL != nil && img.WSL.Managed && img.WSL.State != nil {
		updated, err := s.target.StateCreate(ctx, abs, img, wslTarget.StateCreateOptions{
			FallbackWindowsDir: fallbackWindowsDir,
			NonInteractive:     nonInteractive,
		})
		if err != nil {
			return "", err
		}
		img = updated
	}
	outDir := workspace.ResolvePath(abs, m.Workspace.OutputDir)
	build, err := wslTarget.BuildArtifact(ctx, abs, img, outDir, preferredEngine, s.router)
	if err != nil {
		return "", err
	}
	return s.target.Upgrade(ctx, abs, outDir, img, build.ArtifactPath)
}

func (s *wslSvc) Rollback(ctx context.Context, manifestPath, imageID, backupPath string, fallbackWindowsDir bool, nonInteractive bool) error {
	m, abs, img, err := s.loadImage(ctx, manifestPath, imageID)
	if err != nil {
		return err
	}
	if img.WSL != nil && img.WSL.Managed && img.WSL.State != nil {
		updated, err := s.target.StateCreate(ctx, abs, img, wslTarget.StateCreateOptions{
			FallbackWindowsDir: fallbackWindowsDir,
			NonInteractive:     nonInteractive,
		})
		if err != nil {
			return err
		}
		img = updated
	}
	outDir := workspace.ResolvePath(abs, m.Workspace.OutputDir)
	return s.target.Rollback(ctx, abs, outDir, img, backupPath)
}

func (s *wslSvc) Status(ctx context.Context, manifestPath, imageID string) (string, error) {
	_, _, img, err := s.loadImage(ctx, manifestPath, imageID)
	if err != nil {
		return "", err
	}
	return s.target.Status(ctx, img)
}

type doctorSvc struct{}

func (s *doctorSvc) Run(ctx context.Context) DoctorReport {
	checks := []DoctorCheck{}

	addCmdCheck := func(id string, cmd []string, remediation string, required bool) {
		if len(cmd) == 0 {
			return
		}
		if _, err := exec.LookPath(cmd[0]); err != nil {
			status := "warn"
			if required {
				status = "fail"
			}
			checks = append(checks, DoctorCheck{ID: id, Status: status, Message: fmt.Sprintf("%s not found", cmd[0]), Remediation: remediation})
			return
		}
		out, err := exec.CommandContext(ctx, cmd[0], cmd[1:]...).CombinedOutput()
		msg := strings.TrimSpace(platformwindows.DecodePossiblyUTF16(out))
		if err != nil {
			status := "warn"
			if required {
				status = "fail"
			}
			checks = append(checks, DoctorCheck{ID: id, Status: status, Message: msg, Remediation: remediation})
			return
		}
		checks = append(checks, DoctorCheck{ID: id, Status: "ok", Message: msg})
	}

	addCmdCheck("wsl.version", []string{"wsl", "--version"}, "Install/update WSL from Microsoft Store: `wsl --update`", true)
	if runtime.GOOS == "windows" {
		if err := exec.CommandContext(ctx, "net", "session").Run(); err != nil {
			checks = append(checks, DoctorCheck{ID: "windows.admin", Status: "warn", Message: "admin privileges not detected", Remediation: "Run terminal as Administrator for VHD mount operations."})
		} else {
			checks = append(checks, DoctorCheck{ID: "windows.admin", Status: "ok", Message: "admin privileges detected"})
		}
	}

	_, statuses, _ := engine.Detect(ctx, "")
	for _, st := range statuses {
		status := "warn"
		if st.Installed && st.Ready {
			status = "ok"
		}
		if !st.Installed {
			status = "warn"
		}
		checks = append(checks, DoctorCheck{ID: "engine." + string(st.Name), Status: status, Message: strings.TrimSpace(strings.Join([]string{st.Version, st.Message}, " ")), Remediation: st.Remediation})
	}

	addCmdCheck("node", []string{"node", "--version"}, "Install Node.js 20+", false)
	addCmdCheck("playwright", []string{"npx", "playwright", "--version"}, "Install Playwright: `npm i -D @playwright/test`", false)

	ok := true
	for _, c := range checks {
		if c.Status == "fail" {
			ok = false
			break
		}
	}
	return DoctorReport{OK: ok, Checks: checks}
}

type catalogSvc struct {
	router *features.Router
}

func (s *catalogSvc) DescribeFeature(ctx context.Context, ref string) (features.Metadata, error) {
	p, parsed, err := s.router.ProviderFor(ref)
	if err != nil {
		return features.Metadata{}, err
	}
	dp, ok := p.(features.DescribeProvider)
	if !ok {
		return features.Metadata{}, fmt.Errorf("provider %s does not support describe", parsed.Provider)
	}
	return dp.Describe(ctx, parsed)
}
