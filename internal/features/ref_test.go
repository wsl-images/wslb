package features

import "testing"

type fakeProvider struct{ kind string }

func (p fakeProvider) Kind() string { return p.kind }

func TestParseRef(t *testing.T) {
	tests := []struct {
		in       string
		provider string
		wantErr  bool
	}{
		{in: "wslb:feature/first-boot-user", provider: "internal"},
		{in: "ghcr.io/devcontainers/features/common-utils:2", provider: "devcontainer"},
		{in: "https://example.com/feature.tgz", provider: "devcontainer"},
		{in: "not-a-feature", wantErr: true},
	}
	for _, tc := range tests {
		got, err := ParseRef(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("ParseRef(%q) expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseRef(%q) unexpected error: %v", tc.in, err)
		}
		if got.Provider != tc.provider {
			t.Fatalf("ParseRef(%q) provider=%q want=%q", tc.in, got.Provider, tc.provider)
		}
	}
}

func TestRouterProviderFor(t *testing.T) {
	r := NewRouter(fakeProvider{kind: "internal"}, fakeProvider{kind: "devcontainer"})
	p, parsed, err := r.ProviderFor("wslb:feature/persist-home")
	if err != nil {
		t.Fatalf("ProviderFor: %v", err)
	}
	if p.Kind() != "internal" {
		t.Fatalf("provider kind=%q", p.Kind())
	}
	if parsed.Name != "persist-home" {
		t.Fatalf("parsed name=%q", parsed.Name)
	}
}
