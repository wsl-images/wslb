package wsl

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/wsl-images/wslb/internal/workspace"
)

// Latest official images discovered from:
// https://github.com/orgs/wsl-images/packages?repo_name=images
var wslImagesLatest = []string{
	"ghcr.io/wsl-images/almalinux-10:latest",
	"ghcr.io/wsl-images/almalinux-8:latest",
	"ghcr.io/wsl-images/almalinux-9:latest",
	"ghcr.io/wsl-images/almalinux-kitten-10:latest",
	"ghcr.io/wsl-images/archlinux:latest",
	"ghcr.io/wsl-images/debian:latest",
	"ghcr.io/wsl-images/elxr:latest",
	"ghcr.io/wsl-images/fedoralinux-42:latest",
	"ghcr.io/wsl-images/fedoralinux-43:latest",
	"ghcr.io/wsl-images/kali-linux:latest",
	"ghcr.io/wsl-images/opensuse-leap-15.6:latest",
	"ghcr.io/wsl-images/opensuse-leap-16.0:latest",
	"ghcr.io/wsl-images/opensuse-tumbleweed:latest",
	"ghcr.io/wsl-images/suse-linux-enterprise-15-sp5:latest",
	"ghcr.io/wsl-images/suse-linux-enterprise-15-sp6:latest",
	"ghcr.io/wsl-images/suse-linux-enterprise-15-sp7:latest",
	"ghcr.io/wsl-images/suse-linux-enterprise-16.0:latest",
	"ghcr.io/wsl-images/ubuntu:latest",
	"ghcr.io/wsl-images/ubuntu-24.04:latest",
}

func TestWSLImagesOOBEMatrix(t *testing.T) {
	if strings.TrimSpace(strings.ToLower(strings.TrimSpace(os.Getenv("WSLB_OOBE_MATRIX")))) != "1" {
		t.Skip("set WSLB_OOBE_MATRIX=1 to run docker-backed distro matrix")
	}

	script := renderWSLBOOBEScript(
		&workspace.WSLImageConfig{
			DefaultUser: workspace.WSLDefaultUser{
				Name: "wslbtest",
				UID:  1000,
				GID:  1000,
			},
		},
		workspace.OOBEModePredefined,
		workspace.OOBEStrategyWSLBOnly,
		false,
		"wslbtest",
		1000,
	)
	if strings.TrimSpace(script) == "" {
		t.Fatalf("generated oobe script is empty")
	}

	for _, image := range wslImagesLatest {
		image := image
		t.Run(image, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
			defer cancel()

			if out, err := runCmd(ctx, "docker", "pull", image); err != nil {
				t.Fatalf("docker pull failed: %v\n%s", err, out)
			}

			containerScript := "set -eu\n" +
				"cat >/tmp/wslb-oobe <<'EOF'\n" + script + "\nEOF\n" +
				"chmod +x /tmp/wslb-oobe\n" +
				"/tmp/wslb-oobe\n" +
				"id -u wslbtest >/dev/null\n" +
				"grep -q '^\\[user\\]' /etc/wsl.conf\n" +
				"grep -q '^default=wslbtest$' /etc/wsl.conf\n"

			if out, err := runCmd(ctx, "docker", "run", "--rm", "--entrypoint", "sh", image, "-lc", containerScript); err != nil {
				t.Fatalf("oobe smoke failed: %v\n%s", err, out)
			}
		})
	}
}

func runCmd(ctx context.Context, bin string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}
