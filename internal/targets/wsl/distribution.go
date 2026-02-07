package wsl

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	ico "github.com/Kodeworks/golang-image-ico"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/rasterizer"
	"github.com/wsl-images/wslb/internal/platform/windows"
	"github.com/wsl-images/wslb/internal/workspace"
	xdraw "golang.org/x/image/draw"
)

const (
	defaultIconInstallPath  = "/usr/lib/wsl/icons/wslb-icon.ico"
	defaultWTProfilePath    = "/usr/share/wsl/terminal-profile.json"
	simpleIconSVGURLPattern = "https://simpleicons.org/icons/%s.svg"
)

var simpleIconSlugSanitizer = regexp.MustCompile(`[^a-z0-9]`)
var defaultIconSizes = []int{16, 32, 48, 64, 128, 256}

func renderDistributionConf(ctx context.Context, manifestPath string, image workspace.Image, distro string) (string, error) {
	assets, err := resolveDistributionAssets(ctx, manifestPath, image)
	if err != nil {
		return "", err
	}
	if len(assets.IconBytes) > 0 {
		if err := writeBinaryFileToDistro(ctx, distro, assets.IconPath, assets.IconBytes, 0o644); err != nil {
			return "", err
		}
	}
	if strings.TrimSpace(assets.WTProfileTemplateJSON) != "" && strings.TrimSpace(assets.WTProfileTemplatePath) != "" {
		if err := writeTextFileToDistro(ctx, distro, assets.WTProfileTemplatePath, assets.WTProfileTemplateJSON, 0o644); err != nil {
			return "", err
		}
	}
	return assets.Conf, nil
}

type distributionAssets struct {
	Conf                  string
	IconPath              string
	IconBytes             []byte
	WTProfileTemplatePath string
	WTProfileTemplateJSON string
}

func resolveDistributionAssets(ctx context.Context, manifestPath string, image workspace.Image) (distributionAssets, error) {
	if image.WSL == nil || image.WSL.Distribution == nil {
		return distributionAssets{}, nil
	}

	dist := image.WSL.Distribution
	iconPath := ""
	iconBytes := []byte{}
	if dist.Shortcut != nil {
		resolved, bytes, err := resolveShortcutIcon(ctx, manifestPath, dist.Shortcut.Icon)
		if err != nil {
			return distributionAssets{}, err
		}
		iconPath = resolved
		iconBytes = bytes
	}

	var b strings.Builder
	if dist.OOBE != nil && (strings.TrimSpace(dist.OOBE.Command) != "" || strings.TrimSpace(dist.OOBE.DefaultName) != "" || dist.OOBE.DefaultUID != nil) {
		b.WriteString("[oobe]\n")
		if strings.TrimSpace(dist.OOBE.DefaultName) != "" {
			b.WriteString("defaultName=" + strings.TrimSpace(dist.OOBE.DefaultName) + "\n")
		}
		if dist.OOBE.DefaultUID != nil {
			b.WriteString(fmt.Sprintf("defaultUid=%d\n", *dist.OOBE.DefaultUID))
		}
		if strings.TrimSpace(dist.OOBE.Command) != "" {
			b.WriteString("command=" + strings.TrimSpace(dist.OOBE.Command) + "\n")
		}
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
	wtTemplate := ""
	wtTemplatePath := ""
	if iconPath != "" {
		wtTemplatePath = defaultWTProfilePath
		wtTemplate = fmt.Sprintf("{\n  \"profiles\": [\n    {\n      \"icon\": %q\n    }\n  ]\n}\n", iconPath)
		b.WriteString("[windowsterminal]\n")
		b.WriteString("ProfileTemplate=" + wtTemplatePath + "\n")
	}
	return distributionAssets{
		Conf:                  b.String(),
		IconPath:              iconPath,
		IconBytes:             iconBytes,
		WTProfileTemplatePath: wtTemplatePath,
		WTProfileTemplateJSON: wtTemplate,
	}, nil
}

func resolveShortcutIcon(ctx context.Context, manifestPath string, icon workspace.WSLDIcon) (string, []byte, error) {
	pathValue := strings.TrimSpace(icon.Path)
	slugValue := strings.TrimSpace(icon.SimpleIcon)
	if pathValue == "" && slugValue == "" {
		return "", nil, nil
	}

	iconBytes := []byte{}
	var err error

	switch {
	case slugValue != "":
		slug := normalizeSimpleIconSlug(slugValue)
		if slug == "" {
			return "", nil, fmt.Errorf("invalid simpleIcon slug %q", slugValue)
		}
		iconURL := fmt.Sprintf(simpleIconSVGURLPattern, slug)
		svg, fetchErr := readIconBytes(ctx, iconURL)
		if fetchErr != nil {
			return "", nil, fmt.Errorf("failed to fetch simpleIcon %q: %w", slug, fetchErr)
		}
		if strings.TrimSpace(icon.Color) != "" {
			svg = applySVGColor(svg, strings.TrimSpace(icon.Color))
		}
		iconBytes, err = simpleIconToICO(svg, icon, 256)
		if err != nil {
			return "", nil, fmt.Errorf("failed to convert simpleIcon %q svg to ico: %w", slug, err)
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
			return "", nil, err
		}

		if isSVG(pathValue, raw) {
			iconBytes, err = svgToICO(raw, 256)
			if err != nil {
				return "", nil, fmt.Errorf("failed to convert icon svg to ico: %w", err)
			}
		} else if isICO(pathValue, raw) {
			iconBytes = raw
		} else {
			decoded, _, decErr := image.Decode(bytes.NewReader(raw))
			if decErr != nil {
				return "", nil, fmt.Errorf("icon at %q is not .ico/.svg and failed image decode: %w", pathValue, decErr)
			}
			if strings.TrimSpace(icon.Color) != "" {
				decoded = tintRasterImage(decoded, parseHexColor(icon.Color, color.RGBA{R: 0xE9, G: 0x54, B: 0x20, A: 0xFF}))
			}
			iconBytes, err = imageToMultiSizeICO(decoded, defaultIconSizes)
			if err != nil {
				return "", nil, fmt.Errorf("icon at %q failed image conversion: %w", pathValue, err)
			}
		}
	}

	if len(iconBytes) == 0 {
		return "", nil, nil
	}
	return defaultIconInstallPath, iconBytes, nil
}

func simpleIconToICO(svg []byte, icon workspace.WSLDIcon, size int) ([]byte, error) {
	if size <= 0 {
		size = 256
	}
	style := strings.ToLower(strings.TrimSpace(icon.Style))
	if style == "" {
		style = "flat"
	}
	if style == "flat" {
		// Match Simple Icons PNG generation semantics:
		// render the SVG onto a fixed square canvas with no custom crop/fit.
		pngBytes, err := simpleIconSVGToPNG(svg, 640)
		if err != nil {
			return nil, err
		}
		img, _, err := image.Decode(bytes.NewReader(pngBytes))
		if err != nil {
			return nil, err
		}
		return imageToMultiSizeICO(img, defaultIconSizes)
	}

	fill := parseHexColor(icon.Color, color.RGBA{R: 0xE9, G: 0x54, B: 0x20, A: 0xFF})
	fgSVG := bytes.ReplaceAll(svg, []byte("currentColor"), []byte("#FFFFFF"))

	frames := make([]icoFrame, 0, len(defaultIconSizes))
	for _, s := range defaultIconSizes {
		img, err := renderBadgeSimpleIcon(fgSVG, fill, s)
		if err != nil {
			return nil, err
		}
		png, err := encodePNG(img)
		if err != nil {
			return nil, err
		}
		frames = append(frames, icoFrame{Size: s, PNG: png})
	}
	return encodeICOFrames(frames)
}

func simpleIconSVGToPNG(svg []byte, baseSize int) ([]byte, error) {
	if baseSize < 64 {
		baseSize = 640
	}
	dst, err := renderSVGToRGBAWithCanvas(svg, baseSize)
	if err != nil {
		// Fallback for environments where canvas rasterization fails unexpectedly.
		dst, err = renderSVGToRGBA(svg, baseSize)
		if err != nil {
			return nil, err
		}
	}
	return encodePNG(dst)
}

func renderSVGToRGBAWithCanvas(svg []byte, size int) (*image.RGBA, error) {
	c, err := canvas.ParseSVG(bytes.NewReader(svg))
	if err != nil {
		return nil, err
	}
	w, _ := c.Size()
	if w <= 0 {
		return nil, fmt.Errorf("invalid svg size")
	}
	dpm := float64(size) / w
	return rasterizer.Draw(c, canvas.Resolution(dpm), canvas.DefaultColorSpace), nil
}

func renderBadgeSimpleIcon(fgSVG []byte, fill color.RGBA, size int) (*image.RGBA, error) {
	bg := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(bg, bg.Bounds(), image.Transparent, image.Point{}, draw.Src)
	drawFilledCircle(bg, float64(size)/2, float64(size)/2, float64(size)*0.47, fill)

	iconGlyph, err := oksvg.ReadIconStream(bytes.NewReader(fgSVG), oksvg.WarnErrorMode)
	if err != nil {
		return nil, err
	}
	padding := float64(size) * 0.2
	iconGlyph.SetTarget(padding, padding, float64(size)-padding*2, float64(size)-padding*2)
	scanner := rasterx.NewScannerGV(size, size, bg, bg.Bounds())
	dasher := rasterx.NewDasher(size, size, scanner)
	iconGlyph.Draw(dasher, 1.0)
	return bg, nil
}

func parseHexColor(hex string, fallback color.RGBA) color.RGBA {
	s := strings.TrimSpace(strings.TrimPrefix(hex, "#"))
	if s == "" {
		return fallback
	}
	switch len(s) {
	case 3:
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	case 6:
	default:
		return fallback
	}
	var r, g, b uint8
	_, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b)
	if err != nil {
		return fallback
	}
	return color.RGBA{R: r, G: g, B: b, A: 0xFF}
}

func drawFilledCircle(dst *image.RGBA, cx, cy, r float64, fill color.RGBA) {
	minX := int(cx - r)
	maxX := int(cx + r)
	minY := int(cy - r)
	maxY := int(cy + r)
	rr := r * r
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if x < 0 || y < 0 || x >= dst.Bounds().Dx() || y >= dst.Bounds().Dy() {
				continue
			}
			dx := float64(x) - cx
			dy := float64(y) - cy
			if dx*dx+dy*dy <= rr {
				dst.SetRGBA(x, y, fill)
			}
		}
	}
}

func normalizeSimpleIconSlug(in string) string {
	s := strings.ToLower(strings.TrimSpace(in))
	s = simpleIconSlugSanitizer.ReplaceAllString(s, "")
	return s
}

func applySVGColor(svg []byte, colorHex string) []byte {
	clean := strings.TrimSpace(strings.TrimPrefix(colorHex, "#"))
	if clean == "" {
		return svg
	}
	color := "#" + clean
	if bytes.Contains(svg, []byte("currentColor")) {
		return bytes.ReplaceAll(svg, []byte("currentColor"), []byte(color))
	}
	if bytes.Contains(svg, []byte("fill=")) {
		return svg
	}
	// Simple Icons raw SVGs typically omit fill attributes; set fill on root svg.
	return bytes.Replace(svg, []byte("<svg "), []byte(`<svg fill="`+color+`" `), 1)
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

func svgToMultiSizeICO(svg []byte, sizes []int) ([]byte, error) {
	const baseSize = 1024
	src, err := renderSVGToRGBA(svg, baseSize)
	if err != nil {
		return nil, err
	}
	srcRect := nonTransparentBounds(src)
	if srcRect.Empty() {
		srcRect = src.Bounds()
	}
	// Grow crop very slightly so edges are not clipped in small icon sizes.
	padX := maxInt(1, srcRect.Dx()/80)
	padY := maxInt(1, srcRect.Dy()/80)
	srcRect = image.Rect(
		maxInt(src.Bounds().Min.X, srcRect.Min.X-padX),
		maxInt(src.Bounds().Min.Y, srcRect.Min.Y-padY),
		minInt(src.Bounds().Max.X, srcRect.Max.X+padX),
		minInt(src.Bounds().Max.Y, srcRect.Max.Y+padY),
	)

	frames := make([]icoFrame, 0, len(sizes))
	for _, size := range sizes {
		dst := image.NewRGBA(image.Rect(0, 0, size, size))

		targetEdge := int(float64(size) * 0.96)
		if targetEdge < 1 {
			targetEdge = 1
		}
		srcW := srcRect.Dx()
		srcH := srcRect.Dy()
		targetW := targetEdge
		targetH := targetEdge
		if srcW > srcH {
			targetH = maxInt(1, int(float64(targetEdge)*(float64(srcH)/float64(srcW))))
		} else if srcH > srcW {
			targetW = maxInt(1, int(float64(targetEdge)*(float64(srcW)/float64(srcH))))
		}
		scaled := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
		xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), src, srcRect, draw.Over, nil)
		ox := (size - targetW) / 2
		oy := (size - targetH) / 2
		draw.Draw(dst, image.Rect(ox, oy, ox+targetW, oy+targetH), scaled, image.Point{}, draw.Over)
		png, err := encodePNG(dst)
		if err != nil {
			return nil, err
		}
		frames = append(frames, icoFrame{Size: size, PNG: png})
	}
	return encodeICOFrames(frames)
}

func renderSVGToRGBA(svg []byte, size int) (*image.RGBA, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(svg), oksvg.WarnErrorMode)
	if err != nil {
		return nil, err
	}
	icon.SetTarget(0, 0, float64(size), float64(size))
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	scanner := rasterx.NewScannerGV(size, size, dst, dst.Bounds())
	r := rasterx.NewDasher(size, size, scanner)
	icon.Draw(r, 1.0)
	return dst, nil
}

func nonTransparentBounds(img *image.RGBA) image.Rectangle {
	b := img.Bounds()
	minX, minY := b.Max.X, b.Max.Y
	maxX, maxY := b.Min.X, b.Min.Y
	found := false
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if img.RGBAAt(x, y).A == 0 {
				continue
			}
			found = true
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x > maxX {
				maxX = x
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	if !found {
		return image.Rectangle{}
	}
	return image.Rect(minX, minY, maxX+1, maxY+1)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func tintRasterImage(src image.Image, fill color.RGBA) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a16 := src.At(x, y).RGBA()
			if a16 == 0 {
				continue
			}
			dst.SetRGBA(x-b.Min.X, y-b.Min.Y, color.RGBA{
				R: fill.R,
				G: fill.G,
				B: fill.B,
				A: uint8(a16 >> 8),
			})
		}
	}
	return dst
}

func imageToICO(raw []byte) ([]byte, error) {
	decoded, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	return imageToMultiSizeICO(decoded, defaultIconSizes)
}

func imageToMultiSizeICO(src image.Image, sizes []int) ([]byte, error) {
	b := src.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return nil, fmt.Errorf("invalid icon image dimensions")
	}

	frames := make([]icoFrame, 0, len(sizes))
	for _, size := range sizes {
		dst := image.NewRGBA(image.Rect(0, 0, size, size))
		draw.Draw(dst, dst.Bounds(), image.Transparent, image.Point{}, draw.Src)

		sw := b.Dx()
		sh := b.Dy()
		targetW := size
		targetH := size
		if sw > sh {
			targetH = int(float64(size) * (float64(sh) / float64(sw)))
			if targetH < 1 {
				targetH = 1
			}
		} else if sh > sw {
			targetW = int(float64(size) * (float64(sw) / float64(sh)))
			if targetW < 1 {
				targetW = 1
			}
		}

		scaled := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
		xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), src, b, draw.Over, nil)
		ox := (size - targetW) / 2
		oy := (size - targetH) / 2
		draw.Draw(dst, image.Rect(ox, oy, ox+targetW, oy+targetH), scaled, image.Point{}, draw.Over)

		png, err := encodePNG(dst)
		if err != nil {
			return nil, err
		}
		frames = append(frames, icoFrame{Size: size, PNG: png})
	}
	return encodeICOFrames(frames)
}

type icoFrame struct {
	Size int
	PNG  []byte
}

func encodePNG(img image.Image) ([]byte, error) {
	var out bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&out, img); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func encodeICOFrames(frames []icoFrame) ([]byte, error) {
	if len(frames) == 0 {
		return nil, fmt.Errorf("no icon frames provided")
	}
	sort.Slice(frames, func(i, j int) bool { return frames[i].Size < frames[j].Size })

	var out bytes.Buffer
	if err := binary.Write(&out, binary.LittleEndian, uint16(0)); err != nil {
		return nil, err
	}
	if err := binary.Write(&out, binary.LittleEndian, uint16(1)); err != nil {
		return nil, err
	}
	if err := binary.Write(&out, binary.LittleEndian, uint16(len(frames))); err != nil {
		return nil, err
	}

	dataOffset := 6 + (16 * len(frames))
	for _, frame := range frames {
		if frame.Size < 1 || frame.Size > 256 {
			return nil, fmt.Errorf("invalid icon frame size %d", frame.Size)
		}
		w := byte(frame.Size)
		h := byte(frame.Size)
		if frame.Size == 256 {
			w = 0
			h = 0
		}
		if err := out.WriteByte(w); err != nil {
			return nil, err
		}
		if err := out.WriteByte(h); err != nil {
			return nil, err
		}
		if err := out.WriteByte(0); err != nil {
			return nil, err
		}
		if err := out.WriteByte(0); err != nil {
			return nil, err
		}
		if err := binary.Write(&out, binary.LittleEndian, uint16(1)); err != nil {
			return nil, err
		}
		if err := binary.Write(&out, binary.LittleEndian, uint16(32)); err != nil {
			return nil, err
		}
		if err := binary.Write(&out, binary.LittleEndian, uint32(len(frame.PNG))); err != nil {
			return nil, err
		}
		if err := binary.Write(&out, binary.LittleEndian, uint32(dataOffset)); err != nil {
			return nil, err
		}
		dataOffset += len(frame.PNG)
	}

	for _, frame := range frames {
		if _, err := out.Write(frame.PNG); err != nil {
			return nil, err
		}
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
