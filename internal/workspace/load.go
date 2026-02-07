package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tailscale/hujson"
)

const DefaultManifestPath = DefaultDevcontainerPath
const DefaultDevcontainerJSONCPath = ".devcontainer/devcontainer.jsonc"

func Load(path string) (*Manifest, string, error) {
	if path == "" {
		path = DefaultManifestPath
	}

	abs, err := resolveManifestPath(path)
	if err != nil {
		return nil, "", err
	}

	b, err := os.ReadFile(abs)
	if err != nil {
		return nil, abs, err
	}

	normalized, err := normalizeJSONOrJSONC(b)
	if err != nil {
		return nil, abs, err
	}
	m, err := ParseDevcontainerSuperset(normalized, abs)
	if err != nil {
		return nil, abs, err
	}

	if m.Workspace.OutputDir == "" {
		m.Workspace.OutputDir = "./.wslb-out"
	}

	return m, abs, nil
}

func FindImage(m *Manifest, imageID string) (*Image, error) {
	for i := range m.Images {
		if m.Images[i].ID == imageID {
			return &m.Images[i], nil
		}
	}
	return nil, errors.New("image not found: " + imageID)
}

func ResolvePath(manifestPath, p string) string {
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(filepath.Dir(manifestPath), p))
}

func resolveManifestPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); err == nil {
		return abs, nil
	}
	// For the default path, auto-discover root devcontainer.json as fallback.
	if path == DefaultManifestPath {
		for _, candidate := range []string{DefaultDevcontainerJSONCPath, "devcontainer.jsonc", "devcontainer.json"} {
			cAbs, cErr := filepath.Abs(candidate)
			if cErr != nil {
				continue
			}
			if _, statErr := os.Stat(cAbs); statErr == nil {
				return cAbs, nil
			}
		}
	}
	if errors.Is(err, os.ErrNotExist) {
		return abs, fmt.Errorf("manifest not found: %s (expected devcontainer superset JSON/JSONC)", abs)
	}
	return abs, err
}

func normalizeJSONOrJSONC(raw []byte) ([]byte, error) {
	normalized, err := hujson.Standardize(raw)
	if err != nil {
		return nil, errors.New("manifest must be valid JSON/JSONC devcontainer superset")
	}
	if len(strings.TrimSpace(string(normalized))) == 0 {
		return nil, errors.New("manifest is empty")
	}
	return normalized, nil
}
