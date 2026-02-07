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

func TestParseDevcontainerSupersetConventionDefaults(t *testing.T) {
	jsonText := `{
  "$schema": "https://raw.githubusercontent.com/wsl-images/wslb/main/schemas/wslb-workspace.schema.json",
  "name": "My Dev",
  "image": "ubuntu:24.04",
  "remoteUser": "steve",
  "wslb": {
    "distroName": "MyDevDistro",
    "state": {
      "mode": "windows-dir"
    }
  }
}`
	m, err := ParseDevcontainerSuperset([]byte(jsonText), filepath.Join(t.TempDir(), "devcontainer.json"))
	if err != nil {
		t.Fatalf("ParseDevcontainerSuperset failed: %v", err)
	}
	img := m.Images[0]
	if img.Target != "wsl" {
		t.Fatalf("target=%q", img.Target)
	}
	if img.ID != "mydevdistro" {
		t.Fatalf("id=%q", img.ID)
	}
	if img.WSL.InstallDir == "" {
		t.Fatalf("expected default installDir")
	}
	if img.WSL.DefaultUser.Name != "steve" {
		t.Fatalf("default user=%q", img.WSL.DefaultUser.Name)
	}
	if img.WSL.State == nil || img.WSL.State.MountPoint != "/home" {
		t.Fatalf("expected merged default state mountPoint")
	}
}

func TestParseDevcontainerSupersetAdvancedWSLSettings(t *testing.T) {
	jsonText := `{
  "name": "Advanced WSL",
  "image": "ubuntu:24.04",
  "wslb": {
    "distroName": "AdvancedWSL",
    "wslconf": {
      "boot": {
        "systemd": true,
        "protectBinfmt": true
      },
      "automount": {
        "enabled": true,
        "mountFsTab": true,
        "crossDistro": true
      },
      "gpu": {
        "enabled": true
      },
      "time": {
        "useWindowsTimezone": true
      }
    },
    "wslconfig": {
      "apply": false,
      "wsl2": {
        "memory": "4GB",
        "processors": 2
      },
      "experimental": {
        "sparseVhd": true
      }
    },
    "distribution": {
      "windowsterminal": {
        "enabled": true,
        "ProfileTemplate": "/usr/lib/wsl/terminal-profile.json"
      }
    }
  }
}`
	m, err := ParseDevcontainerSuperset([]byte(jsonText), filepath.Join(t.TempDir(), "devcontainer.json"))
	if err != nil {
		t.Fatalf("ParseDevcontainerSuperset failed: %v", err)
	}
	img := m.Images[0]
	if img.WSL == nil {
		t.Fatalf("expected wsl config")
	}
	if img.WSL.WSLConf.Boot == nil || img.WSL.WSLConf.Boot.ProtectBinfmt == nil || !*img.WSL.WSLConf.Boot.ProtectBinfmt {
		t.Fatalf("expected boot.protectBinfmt=true")
	}
	if img.WSL.WSLConf.Automount == nil || img.WSL.WSLConf.Automount.CrossDistro == nil || !*img.WSL.WSLConf.Automount.CrossDistro {
		t.Fatalf("expected automount.crossDistro=true")
	}
	if img.WSL.WSLConf.GPU == nil || img.WSL.WSLConf.GPU.Enabled == nil || !*img.WSL.WSLConf.GPU.Enabled {
		t.Fatalf("expected gpu.enabled=true")
	}
	if img.WSL.WSLConf.Time == nil || img.WSL.WSLConf.Time.UseWindowsTimezone == nil || !*img.WSL.WSLConf.Time.UseWindowsTimezone {
		t.Fatalf("expected time.useWindowsTimezone=true")
	}
	if img.WSL.WSLConfig == nil || img.WSL.WSLConfig.Apply == nil || *img.WSL.WSLConfig.Apply {
		t.Fatalf("expected wslconfig.apply=false")
	}
	if img.WSL.Distribution == nil || img.WSL.Distribution.WindowsTerminal == nil {
		t.Fatalf("expected distribution.windowsterminal")
	}
	if img.WSL.Distribution.WindowsTerminal.ProfileTemplate != "/usr/lib/wsl/terminal-profile.json" {
		t.Fatalf("unexpected ProfileTemplate=%q", img.WSL.Distribution.WindowsTerminal.ProfileTemplate)
	}
}

func TestParseDevcontainerSupersetWindowsTerminalAliasAndTemplate(t *testing.T) {
	jsonText := `{
  "name": "WT Alias",
  "image": "ubuntu:24.04",
  "wslb": {
    "distribution": {
      "windowsTerminal": {
        "enabled": true,
        "template": {
          "profiles": [
            {
              "name": "WSLB",
              "hidden": false
            }
          ]
        }
      }
    }
  }
}`

	m, err := ParseDevcontainerSuperset([]byte(jsonText), filepath.Join(t.TempDir(), "devcontainer.json"))
	if err != nil {
		t.Fatalf("ParseDevcontainerSuperset failed: %v", err)
	}
	img := m.Images[0]
	if img.WSL == nil || img.WSL.Distribution == nil || img.WSL.Distribution.WindowsTerminal == nil {
		t.Fatalf("expected distribution windows terminal config")
	}
	wt := img.WSL.Distribution.WindowsTerminal
	if wt.Enabled == nil || !*wt.Enabled {
		t.Fatalf("expected windows terminal enabled=true")
	}
	if len(wt.Template) == 0 {
		t.Fatalf("expected windows terminal template payload")
	}
}

func TestParseDevcontainerSupersetWindowsTerminalAliasConflict(t *testing.T) {
	jsonText := `{
  "name": "WT Alias Conflict",
  "image": "ubuntu:24.04",
  "wslb": {
    "distribution": {
      "windowsterminal": { "enabled": true },
      "windowsTerminal": { "enabled": true }
    }
  }
}`

	if _, err := ParseDevcontainerSuperset([]byte(jsonText), filepath.Join(t.TempDir(), "devcontainer.json")); err == nil {
		t.Fatalf("expected error when both windowsterminal and windowsTerminal are set")
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

func TestLoadFallsBackToJSONC(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir tmp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	content := `{
  "name": "demo-jsonc",
  "image": "ubuntu:24.04",
  "wslb": {
    "distroName": "DemoJSONC"
  }
}`
	if err := os.WriteFile("devcontainer.jsonc", []byte(content), 0o644); err != nil {
		t.Fatalf("write devcontainer jsonc: %v", err)
	}

	m, abs, err := Load(DefaultManifestPath)
	if err != nil {
		t.Fatalf("load fallback jsonc: %v", err)
	}
	if filepath.Base(abs) != "devcontainer.jsonc" {
		t.Fatalf("abs=%q", abs)
	}
	if len(m.Images) != 1 {
		t.Fatalf("images=%d", len(m.Images))
	}
}

func TestParseDevcontainerSupersetInteractiveOOBE(t *testing.T) {
	jsonText := `{
  "name": "Interactive OOBE",
  "image": "ubuntu:24.04",
  "wslb": {
    "oobe": {
      "mode": "interactive",
      "strategy": "hybrid",
      "promptForPassword": true
    },
    "wslconf": {
      "boot": {
        "systemd": true
      }
    }
  }
}`
	m, err := ParseDevcontainerSuperset([]byte(jsonText), filepath.Join(t.TempDir(), "devcontainer.json"))
	if err != nil {
		t.Fatalf("ParseDevcontainerSuperset failed: %v", err)
	}
	img := m.Images[0]
	if img.WSL == nil || img.WSL.OOBE == nil {
		t.Fatalf("expected wsl.oobe config")
	}
	if img.WSL.OOBE.Mode != OOBEModeInteractive {
		t.Fatalf("oobe.mode=%q", img.WSL.OOBE.Mode)
	}
	if img.WSL.WSLConf.User != nil && img.WSL.WSLConf.User.Default != "" {
		t.Fatalf("interactive mode should not auto-populate wslconf.user.default")
	}
}

func TestParseDevcontainerSupersetPassThroughSections(t *testing.T) {
	jsonText := `{
  "name": "Pass Through",
  "image": "ubuntu:24.04",
  "wslb": {
    "wslconf": {
      "boot": { "systemd": true },
      "customSection": {
        "foo": "bar",
        "enabled": true
      }
    },
    "distribution": {
      "shortcut": {
        "enabled": true,
        "icon": { "simpleIcon": "ubuntu" }
      },
      "customDistro": {
        "flag": true
      }
    }
  }
}`
	m, err := ParseDevcontainerSuperset([]byte(jsonText), filepath.Join(t.TempDir(), "devcontainer.json"))
	if err != nil {
		t.Fatalf("ParseDevcontainerSuperset failed: %v", err)
	}
	img := m.Images[0]
	if img.WSL == nil {
		t.Fatalf("expected wsl config")
	}
	if img.WSL.WSLConf.ExtraSections == nil || img.WSL.WSLConf.ExtraSections["customSection"]["foo"] != "bar" {
		t.Fatalf("expected wslconf custom section passthrough")
	}
	if img.WSL.Distribution == nil || img.WSL.Distribution.ExtraSections == nil || img.WSL.Distribution.ExtraSections["customDistro"]["flag"] != true {
		t.Fatalf("expected distribution custom section passthrough")
	}
}
