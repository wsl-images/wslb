package wsl

import "testing"

func TestNormalizeSimpleIconSlug(t *testing.T) {
	cases := map[string]string{
		"Ubuntu":               "ubuntu",
		"Visual Studio Code":   "visualstudiocode",
		"GitHub-Actions":       "githubactions",
		"  Docker  ":           "docker",
		"UPPER_and-symbols!!!": "upperandsymbols",
	}
	for in, want := range cases {
		if got := normalizeSimpleIconSlug(in); got != want {
			t.Fatalf("normalizeSimpleIconSlug(%q)=%q want=%q", in, got, want)
		}
	}
}

func TestIsICOHeader(t *testing.T) {
	if !isICO("icon.bin", []byte{0x00, 0x00, 0x01, 0x00}) {
		t.Fatalf("expected ico header to be detected")
	}
}
