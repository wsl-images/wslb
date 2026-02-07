package wsl

import (
	"reflect"
	"strings"
	"testing"

	"github.com/wsl-images/wslb/internal/workspace"
)

func TestWindowsPathToWSLArg(t *testing.T) {
	in := `D:\code\wslb\examples\devcontainer\state\ubuntu-devcontainer-home`
	got := windowsPathToWSLArg(in)
	want := "D:/code/wslb/examples/devcontainer/state/ubuntu-devcontainer-home"
	if got != want {
		t.Fatalf("windowsPathToWSLArg() = %q, want %q", got, want)
	}
}

func TestExpectedFeatureTools(t *testing.T) {
	image := workspace.Image{
		WSL: &workspace.WSLImageConfig{
			Features: []workspace.FeatureAssignment{
				{Ref: "ghcr.io/devcontainers/features/node:1"},
				{Ref: "ghcr.io/devcontainers/features/go:1"},
				{Ref: "ghcr.io/devcontainers/features/python:1"},
				{Ref: "ghcr.io/devcontainers/features/git:1"},
				{Ref: "ghcr.io/devcontainers/features/github-cli:1"},
			},
		},
	}
	got := expectedFeatureTools(image)
	want := []string{"gh", "git", "go", "node", "npm", "python3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expectedFeatureTools() = %#v, want %#v", got, want)
	}
}

func TestRenderWSLConfSections(t *testing.T) {
	trueVal := true
	falseVal := false
	image := workspace.Image{
		WSL: &workspace.WSLImageConfig{
			DefaultUser: workspace.WSLDefaultUser{Name: "dev"},
			WSLConf: workspace.WSLConfConfig{
				User: &workspace.WSLConfUser{Default: "dev"},
				Boot: &workspace.WSLConfBoot{
					Systemd: &trueVal,
					Command: "echo boot",
				},
				Automount: &workspace.WSLConfAutomount{
					Enabled:    &trueVal,
					MountFsTab: &trueVal,
					Options:    "metadata",
				},
				Network: &workspace.WSLConfNetwork{
					Hostname:      "devbox",
					GenerateHosts: &falseVal,
				},
				Interop: &workspace.WSLConfInterop{
					Enabled:           &trueVal,
					AppendWindowsPath: &falseVal,
				},
			},
		},
	}
	conf := renderWSLConf(image)
	required := []string{
		"[user]",
		"default=dev",
		"[boot]",
		"systemd=true",
		"command=echo boot",
		"[automount]",
		"options=metadata",
		"[network]",
		"hostname=devbox",
		"generateHosts=false",
		"[interop]",
		"appendWindowsPath=false",
	}
	for _, token := range required {
		if !strings.Contains(conf, token) {
			t.Fatalf("renderWSLConf missing %q in:\n%s", token, conf)
		}
	}
}

func TestRenderWSLConfAdditionalSections(t *testing.T) {
	trueVal := true
	image := workspace.Image{
		WSL: &workspace.WSLImageConfig{
			DefaultUser: workspace.WSLDefaultUser{Name: "dev"},
			WSLConf: workspace.WSLConfConfig{
				User: &workspace.WSLConfUser{Default: "dev"},
				Boot: &workspace.WSLConfBoot{
					Systemd:       &trueVal,
					ProtectBinfmt: &trueVal,
				},
				Automount: &workspace.WSLConfAutomount{
					Enabled:     &trueVal,
					MountFsTab:  &trueVal,
					CrossDistro: &trueVal,
				},
				GPU:  &workspace.WSLConfGPU{Enabled: &trueVal},
				Time: &workspace.WSLConfTime{UseWindowsTimezone: &trueVal},
			},
		},
	}
	conf := renderWSLConf(image)
	required := []string{
		"protectBinfmt=true",
		"crossDistro=true",
		"[gpu]",
		"[time]",
		"useWindowsTimezone=true",
	}
	for _, token := range required {
		if !strings.Contains(conf, token) {
			t.Fatalf("renderWSLConf missing %q in:\n%s", token, conf)
		}
	}
}

func TestRenderGlobalWSLConfig(t *testing.T) {
	cfg := workspace.WSLGlobalConfig{
		WSL2: map[string]interface{}{
			"memory":     "4GB",
			"processors": float64(2),
		},
		Experimental: map[string]interface{}{
			"sparseVhd": true,
		},
	}
	out, err := renderGlobalWSLConfig(cfg)
	if err != nil {
		t.Fatalf("renderGlobalWSLConfig() error = %v", err)
	}
	required := []string{
		"[wsl2]",
		"memory=4GB",
		"processors=2",
		"[experimental]",
		"sparseVhd=true",
	}
	for _, token := range required {
		if !strings.Contains(out, token) {
			t.Fatalf("renderGlobalWSLConfig missing %q in:\n%s", token, out)
		}
	}
}

func TestManagedUserHome(t *testing.T) {
	cases := []struct {
		mount string
		user  string
		want  string
	}{
		{mount: "/home", user: "dev", want: "/home/dev"},
		{mount: "/workspace/home", user: "alice", want: "/workspace/home/alice"},
		{mount: "/", user: "dev", want: "/dev"},
		{mount: "", user: "", want: "/home/dev"},
	}
	for _, c := range cases {
		got := managedUserHome(c.mount, c.user)
		if got != c.want {
			t.Fatalf("managedUserHome(%q,%q)=%q want=%q", c.mount, c.user, got, c.want)
		}
	}
}
