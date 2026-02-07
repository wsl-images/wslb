package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDefaultsOutputDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devcontainer.json")
	content := `{"name":"test","image":"ubuntu:24.04","wslb":{"target":"wsl"}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	m, abs, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if abs == "" {
		t.Fatalf("expected absolute path")
	}
	if got, want := m.Workspace.OutputDir, "./.wslb-out"; got != want {
		t.Fatalf("outputDir=%q want=%q", got, want)
	}
}

func TestValidateManagedWSLRequiresState(t *testing.T) {
	m := &Manifest{
		Version: 1,
		Workspace: WorkspaceConfig{
			Name:      "demo",
			OutputDir: "./.wslb-out",
		},
		Images: []Image{
			{
				ID:     "img",
				Target: "wsl",
				Base:   "ubuntu:24.04",
				WSL: &WSLImageConfig{
					Managed:    true,
					DistroName: "Demo",
					InstallDir: "./distros/demo",
					DefaultUser: WSLDefaultUser{
						Name: "dev",
						UID:  1000,
						GID:  1000,
					},
				},
			},
		},
	}

	res := Validate(m, filepath.Join(t.TempDir(), "devcontainer.json"))
	if res.Valid {
		t.Fatalf("expected invalid result")
	}
	found := false
	for _, i := range res.Issues {
		if i.Code == "WSLB_WSL_STATE_REQUIRED" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected WSLB_WSL_STATE_REQUIRED issue, got: %+v", res.Issues)
	}
}

func TestBuildPlanDeterministicNames(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "devcontainer.json")
	m := &Manifest{
		Version: 1,
		Workspace: WorkspaceConfig{
			Name:      "demo",
			OutputDir: "./.wslb-out",
		},
		Images: []Image{
			{
				ID:          "dev-ubuntu",
				Target:      "wsl",
				DisplayName: "Dev Ubuntu",
				Base:        "ubuntu:24.04",
				WSL: &WSLImageConfig{
					DistroName: "DemoDistro",
				},
			},
		},
	}
	plan := BuildPlan(m, manifestPath, "")
	if len(plan.Images) != 1 {
		t.Fatalf("expected 1 image plan, got %d", len(plan.Images))
	}
	ip := plan.Images[0]
	if got, want := ip.CandidateName, "DemoDistro__candidate"; got != want {
		t.Fatalf("candidate=%q want=%q", got, want)
	}
	if !strings.Contains(ip.BackupPattern, "DemoDistro__backup__") {
		t.Fatalf("backup pattern=%q", ip.BackupPattern)
	}
	if !filepath.IsAbs(ip.DeterministicDir) {
		t.Fatalf("expected absolute deterministic dir, got %q", ip.DeterministicDir)
	}
}

func TestLoadRejectsYAMLManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wslb-workspace.yaml")
	content := "version: 1\nworkspace:\n  name: test\nimages: []\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if _, _, err := Load(path); err == nil {
		t.Fatalf("expected YAML manifest to be rejected")
	}
}

func TestLoadJSONCManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devcontainer.jsonc")
	content := `{
  // comment
  "name": "jsonc",
  "image": "ubuntu:24.04",
  "wslb": {
    "distroName": "JsoncDistro",
    "state": {
      "mode": "windows-dir",
    },
  },
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	m, _, err := Load(path)
	if err != nil {
		t.Fatalf("load jsonc: %v", err)
	}
	if len(m.Images) != 1 || m.Images[0].WSL == nil {
		t.Fatalf("unexpected parsed manifest")
	}
	if m.Images[0].WSL.State == nil || m.Images[0].WSL.State.MountPoint != "/home" {
		t.Fatalf("expected merged default mountpoint for jsonc")
	}
}

func TestValidateDistributionIconConflict(t *testing.T) {
	m := &Manifest{
		Version: 1,
		Workspace: WorkspaceConfig{
			Name:      "demo",
			OutputDir: "./.wslb-out",
		},
		Images: []Image{
			{
				ID:     "img",
				Target: "wsl",
				Base:   "ubuntu:24.04",
				WSL: &WSLImageConfig{
					Managed:    false,
					DistroName: "Demo",
					InstallDir: "./distros/demo",
					DefaultUser: WSLDefaultUser{
						Name: "dev",
						UID:  1000,
						GID:  1000,
					},
					WSLConf: DefaultWSLConfConfig(),
					Distribution: &WSLDistributionConfig{
						Shortcut: &WSLDistributionShortcut{
							Icon: WSLDIcon{
								Path:       "./icon.ico",
								SimpleIcon: "ubuntu",
							},
						},
					},
				},
			},
		},
	}

	res := Validate(m, filepath.Join(t.TempDir(), "devcontainer.json"))
	if res.Valid {
		t.Fatalf("expected invalid result")
	}
	found := false
	for _, i := range res.Issues {
		if i.Code == "WSLB_WSL_DISTRIBUTION_ICON_CONFLICT" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected icon conflict issue, got: %+v", res.Issues)
	}
}

func TestValidateWSLConfigRequiresScalarValues(t *testing.T) {
	m := &Manifest{
		Version: 1,
		Workspace: WorkspaceConfig{
			Name:      "demo",
			OutputDir: "./.wslb-out",
		},
		Images: []Image{
			{
				ID:     "img",
				Target: "wsl",
				Base:   "ubuntu:24.04",
				WSL: &WSLImageConfig{
					Managed:    false,
					DistroName: "Demo",
					InstallDir: "./distros/demo",
					DefaultUser: WSLDefaultUser{
						Name: "dev",
						UID:  1000,
						GID:  1000,
					},
					WSLConf: DefaultWSLConfConfig(),
					WSLConfig: &WSLGlobalConfig{
						WSL2: map[string]interface{}{
							"memory": "4GB",
							"nested": map[string]interface{}{"bad": true},
						},
					},
				},
			},
		},
	}

	res := Validate(m, filepath.Join(t.TempDir(), "devcontainer.json"))
	if res.Valid {
		t.Fatalf("expected invalid result")
	}
	found := false
	for _, i := range res.Issues {
		if i.Code == "WSLB_WSLCONFIG_VALUE_INVALID" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected WSLB_WSLCONFIG_VALUE_INVALID issue, got: %+v", res.Issues)
	}
}

func TestValidateWSLConfExtraSectionRequiresScalarValues(t *testing.T) {
	m := &Manifest{
		Version: 1,
		Workspace: WorkspaceConfig{
			Name:      "demo",
			OutputDir: "./.wslb-out",
		},
		Images: []Image{
			{
				ID:     "img",
				Target: "wsl",
				Base:   "ubuntu:24.04",
				WSL: &WSLImageConfig{
					Managed:    false,
					DistroName: "Demo",
					InstallDir: "./distros/demo",
					DefaultUser: WSLDefaultUser{
						Name: "dev",
						UID:  1000,
						GID:  1000,
					},
					WSLConf: WSLConfConfig{
						Boot: &WSLConfBoot{Systemd: boolPtr(true)},
						ExtraSections: map[string]map[string]interface{}{
							"custom": {
								"nested": map[string]interface{}{"bad": true},
							},
						},
					},
				},
			},
		},
	}
	res := Validate(m, filepath.Join(t.TempDir(), "devcontainer.json"))
	if res.Valid {
		t.Fatalf("expected invalid result")
	}
	found := false
	for _, i := range res.Issues {
		if i.Code == "WSLB_WSLCONF_VALUE_INVALID" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected WSLB_WSLCONF_VALUE_INVALID issue, got: %+v", res.Issues)
	}
}

func TestValidateDistributionExtraSectionRequiresScalarValues(t *testing.T) {
	m := &Manifest{
		Version: 1,
		Workspace: WorkspaceConfig{
			Name:      "demo",
			OutputDir: "./.wslb-out",
		},
		Images: []Image{
			{
				ID:     "img",
				Target: "wsl",
				Base:   "ubuntu:24.04",
				WSL: &WSLImageConfig{
					Managed:    false,
					DistroName: "Demo",
					InstallDir: "./distros/demo",
					DefaultUser: WSLDefaultUser{
						Name: "dev",
						UID:  1000,
						GID:  1000,
					},
					WSLConf: DefaultWSLConfConfig(),
					Distribution: &WSLDistributionConfig{
						ExtraSections: map[string]map[string]interface{}{
							"custom": {
								"nested": map[string]interface{}{"bad": true},
							},
						},
					},
				},
			},
		},
	}
	res := Validate(m, filepath.Join(t.TempDir(), "devcontainer.json"))
	if res.Valid {
		t.Fatalf("expected invalid result")
	}
	found := false
	for _, i := range res.Issues {
		if i.Code == "WSLB_DISTRIBUTION_VALUE_INVALID" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected WSLB_DISTRIBUTION_VALUE_INVALID issue, got: %+v", res.Issues)
	}
}

func boolPtr(v bool) *bool { return &v }
