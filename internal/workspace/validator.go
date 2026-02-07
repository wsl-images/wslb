package workspace

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/wsl-images/wslb/internal/features"
)

type ValidationIssue struct {
	Path        string `json:"path"`
	Code        string `json:"code"`
	Message     string `json:"message"`
	Remediation string `json:"remediation,omitempty"`
}

type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Issues []ValidationIssue `json:"issues"`
}

func Validate(m *Manifest, manifestPath string) ValidationResult {
	issues := make([]ValidationIssue, 0)

	if m.Version != 1 {
		issues = append(issues, ValidationIssue{
			Path:        "version",
			Code:        "WSLB_WORKSPACE_VERSION_INVALID",
			Message:     "version must be 1",
			Remediation: "set `version: 1`",
		})
	}
	if strings.TrimSpace(m.Workspace.Name) == "" {
		issues = append(issues, ValidationIssue{Path: "workspace.name", Code: "WSLB_WORKSPACE_NAME_REQUIRED", Message: "workspace.name is required"})
	}
	if strings.TrimSpace(m.Workspace.OutputDir) == "" {
		issues = append(issues, ValidationIssue{Path: "workspace.outputDir", Code: "WSLB_WORKSPACE_OUTPUTDIR_REQUIRED", Message: "workspace.outputDir is required"})
	}

	seen := map[string]struct{}{}
	for i := range m.Images {
		img := &m.Images[i]
		prefix := fmt.Sprintf("images[%d]", i)
		if img.ID == "" {
			issues = append(issues, ValidationIssue{Path: prefix + ".id", Code: "WSLB_IMAGE_ID_REQUIRED", Message: "image id is required"})
		} else {
			if _, exists := seen[img.ID]; exists {
				issues = append(issues, ValidationIssue{Path: prefix + ".id", Code: "WSLB_IMAGE_ID_DUPLICATE", Message: "image id must be unique", Remediation: "change one image id"})
			}
			seen[img.ID] = struct{}{}
		}

		if img.Target != "wsl" {
			issues = append(issues, ValidationIssue{Path: prefix + ".target", Code: "WSLB_IMAGE_TARGET_INVALID", Message: "target must be wsl"})
		}
		if img.Base == "" {
			issues = append(issues, ValidationIssue{Path: prefix + ".base", Code: "WSLB_IMAGE_BASE_REQUIRED", Message: "base is required"})
		}

		switch img.Target {
		case "wsl":
			if img.WSL == nil {
				issues = append(issues, ValidationIssue{Path: prefix + ".wsl", Code: "WSLB_WSL_CONFIG_REQUIRED", Message: "wsl config is required for wsl target"})
				continue
			}
			if img.WSL.DistroName == "" {
				issues = append(issues, ValidationIssue{Path: prefix + ".wsl.distroName", Code: "WSLB_WSL_DISTRO_REQUIRED", Message: "wsl.distroName is required"})
			}
			if img.WSL.InstallDir == "" {
				issues = append(issues, ValidationIssue{Path: prefix + ".wsl.installDir", Code: "WSLB_WSL_INSTALLDIR_REQUIRED", Message: "wsl.installDir is required"})
			}
			effectiveMode := EffectiveOOBEMode(img.WSL)
			if effectiveMode == OOBEModePredefined && strings.TrimSpace(img.WSL.DefaultUser.Name) == "" {
				issues = append(issues, ValidationIssue{Path: prefix + ".wsl.defaultUser.name", Code: "WSLB_WSL_DEFAULTUSER_REQUIRED", Message: "default user name is required"})
			}
			if img.WSL.OOBE != nil {
				mode := strings.ToLower(strings.TrimSpace(img.WSL.OOBE.Mode))
				if mode != "" && mode != OOBEModeAuto && mode != OOBEModePredefined && mode != OOBEModeInteractive {
					issues = append(issues, ValidationIssue{
						Path:        prefix + ".wsl.oobe.mode",
						Code:        "WSLB_WSL_OOBE_MODE_INVALID",
						Message:     "oobe.mode must be one of auto|predefined|interactive",
						Remediation: "set oobe.mode to auto, predefined, or interactive",
					})
				}
				strategy := strings.ToLower(strings.TrimSpace(img.WSL.OOBE.Strategy))
				if strategy != "" && strategy != OOBEStrategyHybrid && strategy != OOBEStrategyWSLBOnly && strategy != OOBEStrategyNative {
					issues = append(issues, ValidationIssue{
						Path:        prefix + ".wsl.oobe.strategy",
						Code:        "WSLB_WSL_OOBE_STRATEGY_INVALID",
						Message:     "oobe.strategy must be one of hybrid|wslb-only|native-only",
						Remediation: "set oobe.strategy to hybrid, wslb-only, or native-only",
					})
				}
			}
			if img.WSL.WSLConf.User != nil && strings.TrimSpace(img.WSL.WSLConf.User.Default) == "" {
				issues = append(issues, ValidationIssue{Path: prefix + ".wsl.wslconf.user.default", Code: "WSLB_WSLCONF_DEFAULTUSER_REQUIRED", Message: "wslconf.user.default cannot be empty when user section is provided"})
			}
			if effectiveMode == OOBEModeInteractive && img.WSL.WSLConf.User != nil && strings.TrimSpace(img.WSL.WSLConf.User.Default) != "" {
				issues = append(issues, ValidationIssue{
					Path:        prefix + ".wsl.wslconf.user.default",
					Code:        "WSLB_WSLCONF_DEFAULTUSER_CONFLICT",
					Message:     "interactive oobe mode conflicts with wslconf.user.default",
					Remediation: "remove wslconf.user.default or switch oobe.mode to predefined",
				})
			}
			if img.WSL.Managed && img.WSL.State == nil {
				issues = append(issues, ValidationIssue{Path: prefix + ".wsl.state", Code: "WSLB_WSL_STATE_REQUIRED", Message: "managed WSL requires state config"})
			}
			for section, kv := range img.WSL.WSLConf.ExtraSections {
				if strings.TrimSpace(section) == "" {
					issues = append(issues, ValidationIssue{
						Path:        prefix + ".wsl.wslconf",
						Code:        "WSLB_WSLCONF_SECTION_INVALID",
						Message:     "wslconf extra section name cannot be empty",
						Remediation: "rename section to a non-empty identifier",
					})
					continue
				}
				for key, value := range kv {
					if !isScalarConfigValue(value) {
						issues = append(issues, ValidationIssue{
							Path:        fmt.Sprintf("%s.wsl.wslconf.%s.%s", prefix, section, key),
							Code:        "WSLB_WSLCONF_VALUE_INVALID",
							Message:     "wslconf section values must be string/number/boolean",
							Remediation: "replace nested objects/arrays with scalar values",
						})
					}
				}
			}
			if img.WSL.State != nil {
				if img.WSL.State.Mode != "vhdx" && img.WSL.State.Mode != "windows-dir" {
					issues = append(issues, ValidationIssue{Path: prefix + ".wsl.state.mode", Code: "WSLB_WSL_STATE_MODE_INVALID", Message: "state.mode must be vhdx or windows-dir"})
				}
				if strings.TrimSpace(img.WSL.State.Path) == "" {
					issues = append(issues, ValidationIssue{Path: prefix + ".wsl.state.path", Code: "WSLB_WSL_STATE_PATH_REQUIRED", Message: "state.path is required"})
				}
				if img.WSL.State.MountPoint == "" {
					issues = append(issues, ValidationIssue{Path: prefix + ".wsl.state.mountPoint", Code: "WSLB_WSL_STATE_MOUNTPOINT_REQUIRED", Message: "state.mountPoint is required"})
				}
				if img.WSL.State.Mode == "vhdx" && strings.TrimSpace(img.WSL.State.FSLabel) == "" {
					issues = append(issues, ValidationIssue{Path: prefix + ".wsl.state.fsLabel", Code: "WSLB_WSL_STATE_FSLABEL_REQUIRED", Message: "fsLabel is required for vhdx mode"})
				}
			}
			if img.WSL.WSLConfig != nil {
				if img.WSL.WSLConfig.WSL2 != nil {
					for key, value := range img.WSL.WSLConfig.WSL2 {
						if !isScalarConfigValue(value) {
							issues = append(issues, ValidationIssue{
								Path:        fmt.Sprintf("%s.wsl.wslconfig.wsl2.%s", prefix, key),
								Code:        "WSLB_WSLCONFIG_VALUE_INVALID",
								Message:     "wslconfig.wsl2 values must be string/number/boolean",
								Remediation: "replace nested objects/arrays with scalar values",
							})
						}
					}
				}
				if img.WSL.WSLConfig.Experimental != nil {
					for key, value := range img.WSL.WSLConfig.Experimental {
						if !isScalarConfigValue(value) {
							issues = append(issues, ValidationIssue{
								Path:        fmt.Sprintf("%s.wsl.wslconfig.experimental.%s", prefix, key),
								Code:        "WSLB_WSLCONFIG_VALUE_INVALID",
								Message:     "wslconfig.experimental values must be string/number/boolean",
								Remediation: "replace nested objects/arrays with scalar values",
							})
						}
					}
				}
			}
			for j, f := range img.WSL.Features {
				if strings.TrimSpace(f.Ref) == "" {
					issues = append(issues, ValidationIssue{Path: fmt.Sprintf("%s.wsl.features[%d].ref", prefix, j), Code: "WSLB_FEATURE_REF_REQUIRED", Message: "feature ref is required"})
					continue
				}
				if _, err := features.ParseRef(f.Ref); err != nil {
					issues = append(issues, ValidationIssue{
						Path:        fmt.Sprintf("%s.wsl.features[%d].ref", prefix, j),
						Code:        "WSLB_FEATURE_REF_INVALID",
						Message:     err.Error(),
						Remediation: "use an OCI ref or `wslb:feature/<id>`",
					})
				}
			}
			if img.WSL.Distribution != nil && img.WSL.Distribution.Shortcut != nil {
				icon := img.WSL.Distribution.Shortcut.Icon
				if strings.TrimSpace(icon.Path) != "" && strings.TrimSpace(icon.SimpleIcon) != "" {
					issues = append(issues, ValidationIssue{
						Path:        prefix + ".wsl.distribution.shortcut.icon",
						Code:        "WSLB_WSL_DISTRIBUTION_ICON_CONFLICT",
						Message:     "shortcut icon must set either `path` or `simpleIcon`, not both",
						Remediation: "remove one of the fields from distribution.shortcut.icon",
					})
				}
				if strings.TrimSpace(icon.Path) == "" && strings.TrimSpace(icon.SimpleIcon) == "" && img.WSL.Distribution.Shortcut.Enabled != nil {
					issues = append(issues, ValidationIssue{
						Path:        prefix + ".wsl.distribution.shortcut.icon",
						Code:        "WSLB_WSL_DISTRIBUTION_ICON_REQUIRED",
						Message:     "shortcut enabled expects icon.path or icon.simpleIcon",
						Remediation: "set distribution.shortcut.icon.path to an .ico file or icon.simpleIcon to a Simple Icons slug",
					})
				}
			}
			if img.WSL.Distribution != nil {
				for section, kv := range img.WSL.Distribution.ExtraSections {
					if strings.TrimSpace(section) == "" {
						issues = append(issues, ValidationIssue{
							Path:        prefix + ".wsl.distribution",
							Code:        "WSLB_DISTRIBUTION_SECTION_INVALID",
							Message:     "distribution extra section name cannot be empty",
							Remediation: "rename section to a non-empty identifier",
						})
						continue
					}
					for key, value := range kv {
						if !isScalarConfigValue(value) {
							issues = append(issues, ValidationIssue{
								Path:        fmt.Sprintf("%s.wsl.distribution.%s.%s", prefix, section, key),
								Code:        "WSLB_DISTRIBUTION_VALUE_INVALID",
								Message:     "distribution section values must be string/number/boolean",
								Remediation: "replace nested objects/arrays with scalar values",
							})
						}
					}
				}
			}
		}

		if img.WSL != nil {
			img.WSL.InstallDir = filepath.Clean(ResolvePath(manifestPath, img.WSL.InstallDir))
			if img.WSL.State != nil {
				img.WSL.State.Path = filepath.Clean(ResolvePath(manifestPath, img.WSL.State.Path))
			}
			if img.WSL.Distribution != nil && img.WSL.Distribution.Shortcut != nil {
				iconPath := strings.TrimSpace(img.WSL.Distribution.Shortcut.Icon.Path)
				if iconPath != "" && !strings.Contains(iconPath, "://") && !strings.HasPrefix(iconPath, "/") {
					img.WSL.Distribution.Shortcut.Icon.Path = filepath.Clean(ResolvePath(manifestPath, iconPath))
				}
			}
		}
	}

	return ValidationResult{Valid: len(issues) == 0, Issues: issues}
}

func isScalarConfigValue(v interface{}) bool {
	switch v.(type) {
	case string, bool, float64, float32, int, int32, int64, uint, uint32, uint64:
		return true
	default:
		return false
	}
}
