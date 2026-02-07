package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDevcontainerSuperset(t *testing.T) {
	jsonText := `{
  "name": "Ubuntu Dev",
  "image": "ubuntu:24.04",
  "remoteUser": "dev",
  "features": {
    "ghcr.io/devcontainers/features/common-utils:2": {
      "username": "dev"
    },
    "wslb:feature/persist-home": {
      "MOUNT_POINT": "/home"
    }
  },
  "wslb": {
    "workspaceName": "dev-workspace",
    "imageId": "ubuntu-dev",
    "target": "wsl",
    "distroName": "UbuntuDev",
    "managed": true,
    "outputDir": "./.wslb-out",
    "installDir": "./distros/ubuntu-dev",
    "state": {
      "mode": "windows-dir",
      "path": "./state/ubuntu-dev-home",
      "mountPoint": "/home",
      "fsLabel": "WSLB_STATE"
    },
    "wslconf": {
      "user": {
        "default": "dev"
      },
      "boot": {
        "systemd": true,
        "command": "echo booted"
      },
      "automount": {
        "enabled": true,
        "mountFsTab": true,
        "options": "metadata"
      },
      "interop": {
        "enabled": true,
        "appendWindowsPath": true
      }
    },
    "distribution": {
      "oobe": {
        "command": "echo setup"
      },
      "shortcut": {
        "enabled": true,
        "icon": {
          "simpleIcon": "ubuntu",
          "color": "E95420"
        }
      }
    }
  }
}`

	m, err := ParseDevcontainerSuperset([]byte(jsonText), filepath.Join(t.TempDir(), "devcontainer.json"))
	if err != nil {
		t.Fatalf("ParseDevcontainerSuperset failed: %v", err)
	}
	if m.Version != 1 {
		t.Fatalf("version=%d", m.Version)
	}
	if len(m.Images) != 1 {
		t.Fatalf("images=%d", len(m.Images))
	}
	img := m.Images[0]
	if img.Target != "wsl" {
		t.Fatalf("target=%q", img.Target)
	}
	if img.WSL == nil {
		t.Fatalf("expected wsl config")
	}
	if img.WSL.DistroName != "UbuntuDev" {
		t.Fatalf("distro=%q", img.WSL.DistroName)
	}
	if len(img.WSL.Features) != 2 {
		t.Fatalf("features=%d", len(img.WSL.Features))
	}
	if img.WSL.WSLConf.Boot == nil || img.WSL.WSLConf.Boot.Systemd == nil || !*img.WSL.WSLConf.Boot.Systemd {
		t.Fatalf("expected boot.systemd=true")
	}
	if img.WSL.WSLConf.Automount == nil || img.WSL.WSLConf.Automount.Enabled == nil || !*img.WSL.WSLConf.Automount.Enabled {
		t.Fatalf("expected automount.enabled=true")
	}
	if img.WSL.Distribution == nil || img.WSL.Distribution.Shortcut == nil {
		t.Fatalf("expected distribution shortcut config")
	}
	if got := img.WSL.Distribution.Shortcut.Icon.SimpleIcon; got != "ubuntu" {
		t.Fatalf("shortcut icon simpleIcon=%q", got)
	}
}

func TestParseDevcontainerSupersetLegacyWSLConfBooleans(t *testing.T) {
	jsonText := `{
  "name": "Ubuntu Dev",
  "image": "ubuntu:24.04",
  "wslb": {
    "target": "wsl",
    "wslconf": {
      "systemd": false,
      "automount": false
    }
  }
}`
	m, err := ParseDevcontainerSuperset([]byte(jsonText), filepath.Join(t.TempDir(), "devcontainer.json"))
	if err != nil {
		t.Fatalf("ParseDevcontainerSuperset failed: %v", err)
	}
	img := m.Images[0]
	if img.WSL.WSLConf.Boot == nil || img.WSL.WSLConf.Boot.Systemd == nil || *img.WSL.WSLConf.Boot.Systemd {
		t.Fatalf("expected boot.systemd=false from legacy shorthand")
	}
	if img.WSL.WSLConf.Automount == nil || img.WSL.WSLConf.Automount.Enabled == nil || *img.WSL.WSLConf.Automount.Enabled {
		t.Fatalf("expected automount.enabled=false from legacy shorthand")
	}
}

func TestLoadFallsBackToDevcontainerPath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir tmp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	if err := os.MkdirAll(".devcontainer", 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := `{"name":"demo","image":"ubuntu:24.04","wslb":{"target":"wsl"}}`
	if err := os.WriteFile(filepath.Join(".devcontainer", "devcontainer.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write devcontainer: %v", err)
	}

	m, abs, err := Load(DefaultManifestPath)
	if err != nil {
		t.Fatalf("load fallback: %v", err)
	}
	if filepath.Base(abs) != "devcontainer.json" {
		t.Fatalf("abs=%q", abs)
	}
	if len(m.Images) != 1 {
		t.Fatalf("images=%d", len(m.Images))
	}
}
