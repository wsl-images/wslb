package wsl

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wsl-images/wslb/internal/engine"
	"github.com/wsl-images/wslb/internal/features"
	"github.com/wsl-images/wslb/internal/workspace"
)

type ArtifactBuildResult struct {
	ArtifactPath string
	Engine       engine.Name
	ImageTag     string
}

func BuildArtifact(ctx context.Context, manifestPath string, image workspace.Image, outDir, preferredEngine string, router *features.Router) (*ArtifactBuildResult, error) {
	eng, _, err := engine.Detect(ctx, preferredEngine)
	if err != nil {
		return nil, err
	}
	r := engine.NewRunner(eng)

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	tmpDir, err := os.MkdirTemp("", "wslb-build-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	df := strings.Builder{}
	df.WriteString("FROM " + image.Base + "\n")
	df.WriteString("SHELL [\"/bin/sh\", \"-c\"]\n")

	if image.WSL != nil {
		distAssets, err := resolveDistributionAssets(ctx, manifestPath, image)
		if err != nil {
			return nil, err
		}
		if distAssets.Conf != "" || len(distAssets.IconBytes) > 0 || strings.TrimSpace(distAssets.OOBEScript) != "" {
			distDir := filepath.Join(tmpDir, "wsl-distribution")
			if err := os.MkdirAll(distDir, 0o755); err != nil {
				return nil, err
			}
			if distAssets.BackupNativeConf {
				df.WriteString("RUN if [ -f /etc/wsl-distribution.conf ]; then cp /etc/wsl-distribution.conf /etc/wsl-distribution.conf.wslb-native; fi\n")
			}
			if strings.TrimSpace(distAssets.OOBEScript) != "" && strings.TrimSpace(distAssets.OOBEScriptPath) != "" {
				oobePath := filepath.Join(distDir, "wslb-oobe.sh")
				if err := os.WriteFile(oobePath, []byte(distAssets.OOBEScript), 0o755); err != nil {
					return nil, err
				}
				df.WriteString("RUN mkdir -p " + filepath.ToSlash(filepath.Dir(distAssets.OOBEScriptPath)) + "\n")
				df.WriteString("COPY wsl-distribution/wslb-oobe.sh " + distAssets.OOBEScriptPath + "\n")
				df.WriteString("RUN chmod +x " + distAssets.OOBEScriptPath + "\n")
			}
			if distAssets.Conf != "" {
				confPath := filepath.Join(distDir, "wsl-distribution.conf")
				if err := os.WriteFile(confPath, []byte(distAssets.Conf), 0o644); err != nil {
					return nil, err
				}
				df.WriteString("COPY wsl-distribution/wsl-distribution.conf /etc/wsl-distribution.conf\n")
			}
			if len(distAssets.IconBytes) > 0 {
				iconPath := filepath.Join(distDir, "wslb-icon.ico")
				if err := os.WriteFile(iconPath, distAssets.IconBytes, 0o644); err != nil {
					return nil, err
				}
				df.WriteString("RUN mkdir -p /usr/lib/wsl/icons\n")
				df.WriteString("COPY wsl-distribution/wslb-icon.ico " + defaultIconInstallPath + "\n")
			}
			if strings.TrimSpace(distAssets.WTProfileTemplateJSON) != "" && strings.TrimSpace(distAssets.WTProfileTemplatePath) != "" {
				templatePath := filepath.Join(distDir, "terminal-profile.json")
				if err := os.WriteFile(templatePath, []byte(distAssets.WTProfileTemplateJSON), 0o644); err != nil {
					return nil, err
				}
				df.WriteString("RUN mkdir -p " + filepath.ToSlash(filepath.Dir(distAssets.WTProfileTemplatePath)) + "\n")
				df.WriteString("COPY wsl-distribution/terminal-profile.json " + distAssets.WTProfileTemplatePath + "\n")
			}
		}

		for i, fa := range image.WSL.Features {
			prov, ref, err := router.ProviderFor(fa.Ref)
			if err != nil {
				return nil, err
			}
			rp, ok := prov.(features.ResolveProvider)
			if !ok {
				continue
			}
			script, err := rp.Resolve(ctx, ref, fa.Options)
			if err != nil {
				return nil, err
			}

			sourceDir := filepath.Dir(script.Path)
			targetDir := filepath.Join(tmpDir, "features", fmt.Sprintf("%02d", i))
			if err := copyDirContents(sourceDir, targetDir); err != nil {
				return nil, err
			}

			df.WriteString(fmt.Sprintf("COPY features/%02d/ /tmp/wslb/features/%02d/\n", i, i))
			df.WriteString(fmt.Sprintf("RUN chmod +x /tmp/wslb/features/%02d/install.sh &&", i))
			keys := make([]string, 0, len(script.Env))
			for k := range script.Env {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				v := script.Env[k]
				df.WriteString(fmt.Sprintf(" %s=%q", k, v))
			}
			df.WriteString(fmt.Sprintf(" /tmp/wslb/features/%02d/install.sh\n", i))
		}
		df.WriteString(`RUN set -eu; \
  mkdir -p /etc/profile.d; \
  cat >/etc/profile.d/00-wslb-devcontainer-path.sh <<'EOF'
export NVM_DIR="${NVM_DIR:-/usr/local/share/nvm}"
if [ -s "$NVM_DIR/nvm.sh" ]; then
  . "$NVM_DIR/nvm.sh"
fi
if [ -d /usr/local/share/nvm/current/bin ]; then
  case ":$PATH:" in
    *:/usr/local/share/nvm/current/bin:*) ;;
    *) export PATH="/usr/local/share/nvm/current/bin:$PATH" ;;
  esac
fi
case ":$PATH:" in
  *:/usr/local/go/bin:*) ;;
  *) export PATH="/usr/local/go/bin:$PATH" ;;
esac
EOF
RUN set -eu; \
  chmod 0644 /etc/profile.d/00-wslb-devcontainer-path.sh; \
  for b in node npm npx corepack pnpm yarn; do \
    if [ -x "/usr/local/share/nvm/current/bin/$b" ]; then \
      ln -sf "/usr/local/share/nvm/current/bin/$b" "/usr/local/bin/$b"; \
    fi; \
  done; \
  if [ -x /usr/local/go/bin/go ]; then ln -sf /usr/local/go/bin/go /usr/local/bin/go; fi; \
  if [ -x /usr/local/go/bin/gofmt ]; then ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt; fi
`)
	}

	dockerfile := filepath.Join(tmpDir, "Dockerfile")
	if err := os.WriteFile(dockerfile, []byte(df.String()), 0o644); err != nil {
		return nil, err
	}

	tag := fmt.Sprintf("wslb-%s:%d", strings.ReplaceAll(image.ID, "_", "-"), time.Now().UTC().Unix())
	if err := r.BuildImage(ctx, tag, dockerfile, tmpDir); err != nil {
		return nil, err
	}

	containerName := strings.ReplaceAll(tag, ":", "-")
	if err := r.CreateContainer(ctx, containerName, tag); err != nil {
		return nil, err
	}
	defer r.RemoveContainer(ctx, containerName)

	tmpTar := filepath.Join(tmpDir, image.ID+".tar")
	if err := r.ExportContainer(ctx, containerName, tmpTar); err != nil {
		return nil, err
	}

	outPath := filepath.Join(outDir, image.ID+".wsl")
	if err := filterTar(tmpTar, outPath); err != nil {
		return nil, err
	}

	return &ArtifactBuildResult{ArtifactPath: outPath, Engine: eng, ImageTag: tag}, nil
}

func copyDirContents(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		mode := info.Mode().Perm()
		if mode == 0 {
			mode = 0o644
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
		if err != nil {
			return err
		}
		defer out.Close()
		if _, err := io.Copy(out, in); err != nil {
			return err
		}
		return nil
	})
}

func filterTar(inputTar, outputTar string) error {
	in, err := os.Open(inputTar)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(outputTar)
	if err != nil {
		return err
	}
	defer out.Close()

	tr := tar.NewReader(in)
	tw := tar.NewWriter(out)
	defer tw.Close()

	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if h.Name == "etc/resolv.conf" || h.Name == ".dockerenv" {
			continue
		}
		if err := tw.WriteHeader(h); err != nil {
			return err
		}
		if _, err := io.Copy(tw, tr); err != nil {
			return err
		}
	}
}
