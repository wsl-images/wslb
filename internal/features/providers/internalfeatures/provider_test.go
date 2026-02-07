package internalfeatures

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/wsl-images/wslb/internal/features"
)

func TestResolveWSLPrereqsFeature(t *testing.T) {
	p := New(t.TempDir())
	ref, err := features.ParseRef("wslb:feature/wsl-prereqs")
	if err != nil {
		t.Fatalf("parse ref: %v", err)
	}
	spec, err := p.Resolve(context.Background(), ref, map[string]interface{}{
		"INSTALL_SUDO": "false",
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if strings.TrimSpace(spec.Path) == "" {
		t.Fatalf("expected script path")
	}
	b, err := os.ReadFile(spec.Path)
	if err != nil {
		t.Fatalf("read script: %v", err)
	}
	txt := string(b)
	required := []string{
		"install_prereqs",
		"ensure_getent_fallback",
		"missing user creation tools",
	}
	for _, token := range required {
		if !strings.Contains(txt, token) {
			t.Fatalf("expected script to contain %q", token)
		}
	}
	if got := spec.Env["INSTALL_SUDO"]; got != "false" {
		t.Fatalf("env INSTALL_SUDO=%q", got)
	}
}
