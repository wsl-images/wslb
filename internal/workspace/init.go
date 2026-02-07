package workspace

import (
	"os"
	"path/filepath"
)

const defaultDevcontainerSuperset = `{
  "$schema": "https://raw.githubusercontent.com/wsl-images/wslb/main/schemas/wslb-workspace.schema.json",
  "name": "WSLB Dev Container",
  "image": "ubuntu:24.04",
  "features": {
    "ghcr.io/devcontainers/features/common-utils:2": {
      "username": "dev",
      "upgradePackages": "false"
    }
  },
  "remoteUser": "dev",
  "wslb": {
    "version": 1,
    "distroName": "WSLBDev",
    "managed": true,
    "oobe": {
      "mode": "auto",
      "strategy": "hybrid",
      "promptForPassword": true
    },
    "state": {
      "mode": "windows-dir",
      "mountPoint": "/home"
    },
    "wslconf": {
      "boot": { "systemd": true },
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
      "shortcut": {
        "enabled": true,
        "icon": {
          "simpleIcon": "ubuntu",
          "color": "E95420",
          "style": "flat"
        }
      },
      "windowsTerminal": {
        "enabled": true
      }
    }
  }
}
`

func InitFile(path string) (string, error) {
	return InitDevcontainerFile(path)
}

func InitDevcontainerFile(path string) (string, error) {
	if path == "" {
		path = DefaultDevcontainerPath
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); err == nil {
		return abs, nil
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(abs, []byte(defaultDevcontainerSuperset), 0o644); err != nil {
		return "", err
	}
	return abs, nil
}
