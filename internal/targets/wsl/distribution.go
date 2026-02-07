package wsl

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	ico "github.com/Kodeworks/golang-image-ico"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"github.com/wsl-images/wslb/internal/platform/windows"
	"github.com/wsl-images/wslb/internal/workspace"
)

const (
	defaultIconInstallPath  = "/usr/lib/wsl/icons/wslb-icon.ico"
	simpleIconCDNURLPattern = "https://cdn.jsdelivr.net/npm/simple-icons@v15/icons/%s.svg"
)

var simpleIconSlugSanitizer = regexp.MustCompile(`[^a-z0-9]`)

func renderDistributionConf(ctx context.Context, manifestPath string, image workspace.Image, distro string) (string, error) {
	if image.WSL == nil || image.WSL.Distribution == nil {
		return "", nil
	}

	dist := image.WSL.Distribution
	iconPath := ""
	if dist.Shortcut != nil {
		resolved, err := resolveShortcutIcon(ctx, manifestPath, distro, dist.Shortcut.Icon)
		if err != nil {
			return "", err
		}
		iconPath = resolved
	}

	var b strings.Builder
	if dist.OOBE != nil && strings.TrimSpace(dist.OOBE.Command) != "" {
		b.WriteString("[oobe]\n")
		b.WriteString("command=" + strings.TrimSpace(dist.OOBE.Command) + "\n")
	}
	if dist.Shortcut != nil {
		enabledSet := dist.Shortcut.Enabled != nil
		if enabledSet || iconPath != "" {
			b.WriteString("[shortcut]\n")
			if enabledSet {
				b.WriteString("enabled=" + boolToString(*dist.Shortcut.Enabled) + "\n")
			}
			if iconPath != "" {
				b.WriteString("icon=" + iconPath + "\n")
			}
		}
	}
	return b.String(), nil
}

func resolveShortcutIcon(ctx context.Context, manifestPath, distro string, icon workspace.WSLDIcon) (string, error) {
	pathValue := strings.TrimSpace(icon.Path)
	slugValue := strings.TrimSpace(icon.SimpleIcon)
	if pathValue == "" && slugValue == "" {
		return "", nil
	}

	iconBytes := []byte{}
	var err error

	switch {
	case slugValue != "":
		slug := normalizeSimpleIconSlug(slugValue)
		if slug == "" {
			return "", fmt.Errorf("invalid simpleIcon slug %q", slugValue)
		}
		iconURL := fmt.Sprintf(simpleIconCDNURLPattern, slug)
		svg, fetchErr := readIconBytes(ctx, iconURL)
		if fetchErr != nil {
			return "", fmt.Errorf("failed to fetch simpleIcon %q: %w", slug, fetchErr)
		}
		if strings.TrimSpace(icon.Color) != "" {
			svg = bytes.ReplaceAll(svg, []byte("currentColor"), []byte("#"+strings.TrimPrefix(strings.TrimSpace(icon.Color), "#")))
		}
		iconBytes, err = svgToICO(svg, 256)
		if err != nil {
			return "", fmt.Errorf("failed to convert simpleIcon %q svg to ico: %w", slug, err)
		}
	case pathValue != "":
		isURL := false
		if u, parseErr := url.Parse(pathValue); parseErr == nil && u != nil && (u.Scheme == "http" || u.Scheme == "https") {
			isURL = true
		}

		raw := []byte{}
		if isURL {
			raw, err = readIconBytes(ctx, pathValue)
		} else {
			resolvedPath := pathValue
			if !filepath.IsAbs(resolvedPath) {
				resolvedPath = workspace.ResolvePath(manifestPath, pathValue)
			}
			raw, err = os.ReadFile(resolvedPath)
		}
		if err != nil {
			return "", err
		}

		if isSVG(pathValue, raw) {
			iconBytes, err = svgToICO(raw, 256)
			if err != nil {
				return "", fmt.Errorf("failed to convert icon svg to ico: %w", err)
			}
		} else if isICO(pathValue, raw) {
			iconBytes = raw
		} else {
			iconBytes, err = imageToICO(raw)
			if err != nil {
				return "", fmt.Errorf("icon at %q is not .ico/.svg and failed image conversion: %w", pathValue, err)
			}
		}
	}

	if len(iconBytes) == 0 {
		return "", nil
	}
	if err := writeBinaryFileToDistro(ctx, distro, defaultIconInstallPath, iconBytes, 0o644); err != nil {
		return "", err
	}
	return defaultIconInstallPath, nil
}

func normalizeSimpleIconSlug(in string) string {
	s := strings.ToLower(strings.TrimSpace(in))
	s = simpleIconSlugSanitizer.ReplaceAllString(s, "")
	return s
}

func readIconBytes(ctx context.Context, ref string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ref, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d for %s", resp.StatusCode, ref)
	}
	return io.ReadAll(resp.Body)
}

func isICO(ref string, data []byte) bool {
	if strings.HasSuffix(strings.ToLower(ref), ".ico") {
		return true
	}
	if len(data) >= 4 {
		// ICO magic: 00 00 01 00
		return data[0] == 0x00 && data[1] == 0x00 && data[2] == 0x01 && data[3] == 0x00
	}
	return false
}

func isSVG(ref string, data []byte) bool {
	if strings.HasSuffix(strings.ToLower(ref), ".svg") {
		return true
	}
	head := strings.ToLower(string(bytes.TrimSpace(data)))
	return strings.Contains(head, "<svg")
}

func svgToICO(svg []byte, size int) ([]byte, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(svg), oksvg.WarnErrorMode)
	if err != nil {
		return nil, err
	}
	if size <= 0 {
		size = 256
	}
	icon.SetTarget(0, 0, float64(size), float64(size))
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	scanner := rasterx.NewScannerGV(size, size, dst, dst.Bounds())
	r := rasterx.NewDasher(size, size, scanner)
	icon.Draw(r, 1.0)

	var out bytes.Buffer
	if err := ico.Encode(&out, dst); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func imageToICO(raw []byte) ([]byte, error) {
	decoded, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	b := decoded.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return nil, fmt.Errorf("invalid icon image dimensions")
	}

	side := b.Dx()
	if b.Dy() > side {
		side = b.Dy()
	}
	if side > 256 {
		side = 256
	}
	dst := image.NewRGBA(image.Rect(0, 0, side, side))
	draw.Draw(dst, dst.Bounds(), image.Transparent, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), decoded, b.Min, draw.Over)

	var out bytes.Buffer
	if err := ico.Encode(&out, dst); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func writeTextFileToDistro(ctx context.Context, distro, targetPath, content string, mode os.FileMode) error {
	return writeBinaryFileToDistro(ctx, distro, targetPath, []byte(content), mode)
}

func writeBinaryFileToDistro(ctx context.Context, distro, targetPath string, data []byte, mode os.FileMode) error {
	targetDir := path.Dir(targetPath)
	cmdText := fmt.Sprintf("set -eu; mkdir -p %s; cat > %s; chmod %04o %s", shQuote(targetDir), shQuote(targetPath), mode.Perm(), shQuote(targetPath))
	cmd := exec.CommandContext(ctx, "wsl", "-d", distro, "-u", "root", "--", "sh", "-lc", cmdText)
	cmd.Stdin = bytes.NewReader(data)
	out, err := cmd.CombinedOutput()
	decoded := windows.DecodePossiblyUTF16(out)
	if err != nil {
		return fmt.Errorf("failed writing %s in distro %s: %w\n%s", targetPath, distro, err, decoded)
	}
	return nil
}

func shQuote(in string) string {
	if in == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(in, "'", `'"'"'`) + "'"
}
