package wsl

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wsl-images/wslb/internal/workspace"
)

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

func TestImageToMultiSizeICO(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, color.RGBA{R: 233, G: 84, B: 32, A: 255})
		}
	}

	icoBytes, err := imageToMultiSizeICO(img, []int{16, 32, 64, 128, 256})
	if err != nil {
		t.Fatalf("imageToMultiSizeICO error: %v", err)
	}
	if len(icoBytes) < 6 {
		t.Fatalf("ico output too small")
	}
	count := binary.LittleEndian.Uint16(icoBytes[4:6])
	if count != 5 {
		t.Fatalf("expected 5 icon frames, got %d", count)
	}
	maybeWriteIconDebugFrames(t, icoBytes, "image-multisize")
}

func TestApplySVGColor(t *testing.T) {
	raw := []byte(`<svg role="img" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path d="M0 0h24v24z"/></svg>`)
	colored := applySVGColor(raw, "E95420")
	if string(colored) == string(raw) {
		t.Fatalf("expected svg to be colorized")
	}
	if got := string(colored); !containsAll(got, `fill="#E95420"`, "<svg ") {
		t.Fatalf("unexpected colorized svg: %s", got)
	}
}

func TestSimpleIconToICOFlatPipeline(t *testing.T) {
	svg := []byte(`<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path d="M2 2h20v20H2z"/></svg>`)
	icoBytes, err := simpleIconToICO(svg, workspace.WSLDIcon{Style: "flat"}, 256)
	if err != nil {
		t.Fatalf("simpleIconToICO error: %v", err)
	}
	if len(icoBytes) < 6 {
		t.Fatalf("ico output too small")
	}
	count := binary.LittleEndian.Uint16(icoBytes[4:6])
	if int(count) != len(defaultIconSizes) {
		t.Fatalf("expected %d icon frames, got %d", len(defaultIconSizes), count)
	}
	maybeWriteIconDebugFrames(t, icoBytes, "simpleicon-flat")
}

func TestSimpleIconUbuntuFidelity(t *testing.T) {
	svgPath := filepath.Join("testdata", "simpleicons-ubuntu.svg")
	svg, err := os.ReadFile(svgPath)
	if err != nil {
		t.Fatalf("read test svg: %v", err)
	}
	colored := applySVGColor(svg, "E95420")
	icoBytes, err := simpleIconToICO(colored, workspace.WSLDIcon{Style: "flat"}, 256)
	if err != nil {
		t.Fatalf("simpleIconToICO ubuntu error: %v", err)
	}
	count := int(binary.LittleEndian.Uint16(icoBytes[4:6]))
	if count != len(defaultIconSizes) {
		t.Fatalf("expected %d frames got %d", len(defaultIconSizes), count)
	}
	frame16, err := decodeICOFramePNGBySize(icoBytes, 16)
	if err != nil {
		t.Fatalf("decode 16x16 frame: %v", err)
	}
	nonTransparent := 0
	orangeish := 0
	opaqueBlack := 0
	b := frame16.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := frame16.At(x, y).RGBA()
			if a == 0 {
				continue
			}
			nonTransparent++
			ru, gu, bu := uint8(r>>8), uint8(g>>8), uint8(bl>>8)
			if ru > 180 && gu > 40 && gu < 140 && bu < 110 {
				orangeish++
			}
			if ru < 20 && gu < 20 && bu < 20 {
				opaqueBlack++
			}
		}
	}
	if nonTransparent < 20 {
		t.Fatalf("unexpectedly sparse 16x16 icon: nonTransparent=%d", nonTransparent)
	}
	if orangeish == 0 {
		t.Fatalf("expected orange pixels in 16x16 icon")
	}
	if opaqueBlack > nonTransparent/5 {
		t.Fatalf("too many black opaque pixels in 16x16 icon: black=%d total=%d", opaqueBlack, nonTransparent)
	}
	minX, minY, maxX, maxY, found := imageOpaqueBounds(frame16)
	if found {
		t.Logf("16x16 opaque bounds: (%d,%d)-(%d,%d) size=%dx%d", minX, minY, maxX, maxY, maxX-minX+1, maxY-minY+1)
	}
	maybeWriteIconDebugFrames(t, icoBytes, "simpleicon-ubuntu")
}

func TestTintRasterImage(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 1))
	src.SetRGBA(0, 0, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	src.SetRGBA(1, 0, color.RGBA{R: 10, G: 20, B: 30, A: 128})

	dst := tintRasterImage(src, color.RGBA{R: 233, G: 84, B: 32, A: 255})
	c0 := color.RGBAModel.Convert(dst.At(0, 0)).(color.RGBA)
	c1 := color.RGBAModel.Convert(dst.At(1, 0)).(color.RGBA)

	if c0.R != 233 || c0.G != 84 || c0.B != 32 || c0.A != 255 {
		t.Fatalf("unexpected tinted pixel 0: %#v", c0)
	}
	if c1.R != 233 || c1.G != 84 || c1.B != 32 || c1.A != 128 {
		t.Fatalf("unexpected tinted pixel 1: %#v", c1)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}

func maybeWriteIconDebugFrames(t *testing.T, icoBytes []byte, prefix string) {
	t.Helper()
	dir := strings.TrimSpace(os.Getenv("WSLB_ICON_DEBUG_DIR"))
	if dir == "" {
		return
	}
	if len(icoBytes) < 6 {
		t.Logf("icon debug: ico too small")
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Logf("icon debug: mkdir failed: %v", err)
		return
	}
	count := int(binary.LittleEndian.Uint16(icoBytes[4:6]))
	for i := 0; i < count; i++ {
		o := 6 + (16 * i)
		if o+16 > len(icoBytes) {
			continue
		}
		size := int(binary.LittleEndian.Uint32(icoBytes[o+8 : o+12]))
		off := int(binary.LittleEndian.Uint32(icoBytes[o+12 : o+16]))
		if size <= 0 || off < 0 || off+size > len(icoBytes) {
			continue
		}
		out := filepath.Join(dir, fmt.Sprintf("%s-frame-%02d.png", prefix, i))
		if err := os.WriteFile(out, icoBytes[off:off+size], 0o644); err != nil {
			t.Logf("icon debug: write %s failed: %v", out, err)
			continue
		}
	}
	t.Logf("icon debug: wrote frames under %s", dir)
}

func decodeICOFramePNGBySize(icoBytes []byte, sizePx int) (image.Image, error) {
	if len(icoBytes) < 6 {
		return nil, fmt.Errorf("ico too small")
	}
	count := int(binary.LittleEndian.Uint16(icoBytes[4:6]))
	for i := 0; i < count; i++ {
		o := 6 + (16 * i)
		if o+16 > len(icoBytes) {
			continue
		}
		w := int(icoBytes[o])
		h := int(icoBytes[o+1])
		if w == 0 {
			w = 256
		}
		if h == 0 {
			h = 256
		}
		if w != sizePx || h != sizePx {
			continue
		}
		size := int(binary.LittleEndian.Uint32(icoBytes[o+8 : o+12]))
		off := int(binary.LittleEndian.Uint32(icoBytes[o+12 : o+16]))
		if size <= 0 || off < 0 || off+size > len(icoBytes) {
			return nil, fmt.Errorf("invalid frame bounds for size %d", sizePx)
		}
		return png.Decode(bytes.NewReader(icoBytes[off : off+size]))
	}
	return nil, fmt.Errorf("frame size %d not found", sizePx)
}

func imageOpaqueBounds(img image.Image) (minX, minY, maxX, maxY int, found bool) {
	b := img.Bounds()
	minX, minY = b.Max.X, b.Max.Y
	maxX, maxY = b.Min.X-1, b.Min.Y-1
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a == 0 {
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
	return minX, minY, maxX, maxY, found
}
