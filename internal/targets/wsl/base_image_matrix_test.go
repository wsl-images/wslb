package wsl

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wsl-images/wslb/internal/features"
	internalfeatures "github.com/wsl-images/wslb/internal/features/providers/internalfeatures"
	"github.com/wsl-images/wslb/internal/workspace"
)

type baseImageTarget struct {
	Image      string
	Deprecated bool
	WSLCompat  bool
}

// Sources (user-provided list, verified by live pull during test run):
// - Docker Official Images catalog: https://hub.docker.com/categories/operating-systems
// - Vendor official namespaces on Docker Hub.
var baseImageMatrix = []baseImageTarget{
	{Image: "ubuntu:latest", WSLCompat: true},
	{Image: "debian:latest", WSLCompat: true},
	{Image: "alpine:latest", WSLCompat: true},
	{Image: "fedora:latest", WSLCompat: true},
	{Image: "amazonlinux:2023", WSLCompat: true},
	{Image: "oraclelinux:9", WSLCompat: true},
	{Image: "rockylinux:9", WSLCompat: true},
	{Image: "almalinux:9", WSLCompat: true},
	{Image: "archlinux:latest", WSLCompat: true},
	{Image: "photon:latest", WSLCompat: true},
	{Image: "mageia:latest", WSLCompat: true},
	{Image: "alt:latest", WSLCompat: true},
	{Image: "cirros:latest", WSLCompat: true},
	{Image: "centos:7", Deprecated: true, WSLCompat: true},
	{Image: "clearlinux:latest", Deprecated: true, WSLCompat: true},
	{Image: "sl:latest", Deprecated: true, WSLCompat: true},
	{Image: "clefos:latest", Deprecated: true, WSLCompat: false},
	{Image: "opensuse/leap:15.6", WSLCompat: true},
	{Image: "opensuse/tumbleweed:latest", WSLCompat: true},
	{Image: "kalilinux/kali-rolling:latest", WSLCompat: true},
	{Image: "gentoo/stage3:latest", WSLCompat: true},
	{Image: "voidlinux/voidlinux:latest", WSLCompat: true},
	{Image: "parrotsec/core:latest", WSLCompat: true},
	{Image: "redhat/ubi8:latest", WSLCompat: true},
	{Image: "redhat/ubi8-minimal:latest", WSLCompat: true},
}

func TestBaseImageMatrixWSLPrereqsAndOOBE(t *testing.T) {
	if strings.TrimSpace(strings.ToLower(strings.TrimSpace(os.Getenv("WSLB_BASE_IMAGE_MATRIX")))) != "1" {
		t.Skip("set WSLB_BASE_IMAGE_MATRIX=1 to run docker-backed base image matrix")
	}

	prereqsScript := resolveInternalFeatureScript(t, "wslb:feature/wsl-prereqs")
	oobeScript := renderWSLBOOBEScript(
		&workspace.WSLImageConfig{
			DefaultUser: workspace.WSLDefaultUser{
				Name: "wslbtest",
				UID:  2000,
				GID:  2000,
			},
		},
		workspace.OOBEModePredefined,
		workspace.OOBEStrategyWSLBOnly,
		false,
		"wslbtest",
		2000,
	)
	if strings.TrimSpace(oobeScript) == "" {
		t.Fatalf("generated oobe script is empty")
	}

	for _, item := range baseImageMatrix {
		item := item
		t.Run(item.Image, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
			defer cancel()

			if out, err := runCmd(ctx, "docker", "pull", item.Image); err != nil {
				if item.Deprecated || !item.WSLCompat {
					t.Skipf("image outside supported WSL compatibility profile unavailable: %v\n%s", err, out)
				}
				t.Fatalf("docker pull failed: %v\n%s", err, out)
			}

			containerScript := "set -eu\n" +
				"cat >/tmp/wslb-prereqs.sh <<'__WSLB_PREREQS__'\n" + prereqsScript + "\n__WSLB_PREREQS__\n" +
				"chmod +x /tmp/wslb-prereqs.sh\n" +
				"/tmp/wslb-prereqs.sh\n" +
				"cat >/tmp/wslb-oobe.sh <<'__WSLB_OOBE__'\n" + oobeScript + "\n__WSLB_OOBE__\n" +
				"chmod +x /tmp/wslb-oobe.sh\n" +
				"/tmp/wslb-oobe.sh\n" +
				"id -u wslbtest >/dev/null\n" +
				"grep -q '^\\[user\\]' /etc/wsl.conf\n" +
				"grep -q '^default=wslbtest$' /etc/wsl.conf\n"

			if out, err := runCmd(ctx, "docker", "run", "--rm", "--entrypoint", "sh", item.Image, "-lc", containerScript); err != nil {
				if item.Deprecated || !item.WSLCompat {
					t.Skipf("image outside supported WSL compatibility profile failed matrix run: %v\n%s", err, out)
				}
				t.Fatalf("matrix run failed: %v\n%s", err, out)
			}
		})
	}
}

func resolveInternalFeatureScript(t *testing.T, rawRef string) string {
	t.Helper()
	ref, err := features.ParseRef(rawRef)
	if err != nil {
		t.Fatalf("parse feature ref %q: %v", rawRef, err)
	}
	cacheDir := filepath.Join(t.TempDir(), "internal-feature-cache")
	provider := internalfeatures.New(cacheDir)
	spec, err := provider.Resolve(context.Background(), ref, map[string]interface{}{})
	if err != nil {
		t.Fatalf("resolve feature %q: %v", rawRef, err)
	}
	b, err := os.ReadFile(spec.Path)
	if err != nil {
		t.Fatalf("read feature script: %v", err)
	}
	return string(b)
}
