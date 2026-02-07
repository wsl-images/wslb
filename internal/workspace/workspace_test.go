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
