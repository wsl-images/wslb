package devcontainer

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/wsl-images/wslb/internal/features"
)

type Provider struct {
	CacheDir string
}

func New(cacheDir string) *Provider {
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "wslb-feature-cache")
	}
	_ = os.MkdirAll(cacheDir, 0o755)
	return &Provider{CacheDir: cacheDir}
}

func (p *Provider) Kind() string { return "devcontainer" }

func (p *Provider) Describe(ctx context.Context, ref features.Ref) (features.Metadata, error) {
	dir, meta, err := p.fetch(ctx, ref)
	if err != nil {
		return features.Metadata{}, err
	}
	_ = dir
	return meta, nil
}

func (p *Provider) Resolve(ctx context.Context, ref features.Ref, options map[string]interface{}) (features.ScriptSpec, error) {
	dir, meta, err := p.fetch(ctx, ref)
	if err != nil {
		return features.ScriptSpec{}, err
	}

	script := filepath.Join(dir, "install.sh")
	if _, err := os.Stat(script); err != nil {
		return features.ScriptSpec{}, fmt.Errorf("install.sh missing in feature bundle: %w", err)
	}

	env := map[string]string{}
	for k, def := range meta.Options {
		envKey := toFeatureEnv(k)
		if v, ok := options[k]; ok {
			env[envKey] = fmt.Sprintf("%v", v)
			continue
		}
		if def.Default != nil {
			env[envKey] = fmt.Sprintf("%v", def.Default)
		}
	}

	return features.ScriptSpec{
		Name:     ref.Raw,
		Path:     script,
		Env:      env,
		Metadata: meta,
	}, nil
}

type featureFile struct {
	ID            string                            `json:"id"`
	Version       string                            `json:"version"`
	Name          string                            `json:"name"`
	Description   string                            `json:"description"`
	Options       map[string]features.Option        `json:"options"`
	DependsOn     map[string]map[string]interface{} `json:"dependsOn"`
	InstallsAfter []string                          `json:"installsAfter"`
}

func (p *Provider) fetch(ctx context.Context, ref features.Ref) (string, features.Metadata, error) {
	safe := strings.NewReplacer("/", "_", ":", "_", "@", "_", ".", "_").Replace(ref.Raw)
	dir := filepath.Join(p.CacheDir, safe)
	metaPath := filepath.Join(dir, "devcontainer-feature.json")

	if b, err := os.ReadFile(metaPath); err == nil {
		var ff featureFile
		if err := json.Unmarshal(b, &ff); err == nil {
			return dir, toMetadata(ff), nil
		}
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", features.Metadata{}, err
	}

	n, err := name.ParseReference(ref.Raw, name.WeakValidation)
	if err != nil {
		return "", features.Metadata{}, err
	}
	desc, err := remote.Get(n, remote.WithContext(ctx))
	if err != nil {
		return "", features.Metadata{}, err
	}
	img, err := desc.Image()
	if err != nil {
		return "", features.Metadata{}, err
	}
	layers, err := img.Layers()
	if err != nil {
		return "", features.Metadata{}, err
	}
	if len(layers) == 0 {
		return "", features.Metadata{}, fmt.Errorf("feature image has no layers")
	}

	picked := layers[0]
	for _, l := range layers {
		mt, err := l.MediaType()
		if err != nil {
			continue
		}
		if strings.Contains(string(mt), "devcontainers.layer.v1+tar") {
			picked = l
			break
		}
	}

	rc, err := picked.Uncompressed()
	if err != nil {
		return "", features.Metadata{}, err
	}
	defer rc.Close()

	if err := untar(rc, dir); err != nil {
		return "", features.Metadata{}, err
	}

	b, err := os.ReadFile(metaPath)
	if err != nil {
		return "", features.Metadata{}, fmt.Errorf("feature metadata missing after extraction: %w", err)
	}
	var ff featureFile
	if err := json.Unmarshal(b, &ff); err != nil {
		return "", features.Metadata{}, err
	}
	return dir, toMetadata(ff), nil
}

func untar(r io.Reader, dir string) error {
	tr := tar.NewReader(r)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dir, filepath.Clean(h.Name))
		if !strings.HasPrefix(target, dir) {
			return fmt.Errorf("invalid tar path: %s", h.Name)
		}
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
}

func toMetadata(ff featureFile) features.Metadata {
	return features.Metadata{
		ID:            ff.ID,
		Name:          ff.Name,
		Version:       ff.Version,
		Description:   ff.Description,
		Options:       ff.Options,
		DependsOn:     ff.DependsOn,
		InstallsAfter: ff.InstallsAfter,
	}
}

var nonAlnum = regexp.MustCompile(`[^A-Z0-9]+`)

func toFeatureEnv(k string) string {
	up := strings.ToUpper(k)
	up = nonAlnum.ReplaceAllString(up, "_")
	return strings.Trim(up, "_")
}
