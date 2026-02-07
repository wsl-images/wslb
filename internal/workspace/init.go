package workspace

import (
	"os"
	"path/filepath"
)

const defaultDevcontainerSuperset = `{
  "name": "WSLB Dev Container",
  "image": "ubuntu:24.04",
  "features": {
    "ghcr.io/devcontainers/features/common-utils:2": {
      "username": "dev",
      "upgradePackages": "false"
    },
    "wslb:feature/first-boot-user": {
      "USERNAME": "dev",
      "USER_UID": "1000",
      "USER_GID": "1000",
      "HOME": "/home/dev"
    },
    "wslb:feature/persist-home": {
      "MOUNT_POINT": "/home",
      "SYSTEMD": "true"
    }
  },
  "remoteUser": "dev",
  "wslb": {
    "version": 1,
    "workspaceName": "wslb-devcontainer",
    "imageId": "wslb-dev",
    "target": "wsl",
    "distroName": "WSLBDev",
    "managed": true,
    "outputDir": "./.wslb-out",
    "installDir": "./distros/wslb-dev",
    "state": {
      "mode": "windows-dir",
      "path": "./state/wslb-dev-home",
      "mountPoint": "/home",
      "fsLabel": "WSLB_STATE"
    },
    "wslconf": {
      "user": {
        "default": "dev"
      },
      "boot": {
        "systemd": true
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
        "command": "usermod --shell /bin/bash dev"
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
