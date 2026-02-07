package wsl

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
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
	defaultOOBEScriptPath   = "/usr/lib/wsl/wslb-oobe"
	simpleIconSVGURLPattern = "https://simpleicons.org/icons/%s.svg"
)

var simpleIconSlugSanitizer = regexp.MustCompile(`[^a-z0-9]`)
var defaultIconSizes = []int{16, 32, 48, 64, 128, 256}

func renderDistributionConf(ctx context.Context, manifestPath string, image workspace.Image, distro string) (string, error) {
	assets, err := resolveDistributionAssets(ctx, manifestPath, image)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(assets.OOBEScript) != "" && strings.TrimSpace(assets.OOBEScriptPath) != "" {
		if err := writeTextFileToDistro(ctx, distro, assets.OOBEScriptPath, assets.OOBEScript, 0o755); err != nil {
			return "", err
		}
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
	OOBEScriptPath        string
	OOBEScript            string
	BackupNativeConf      bool
}

func resolveDistributionAssets(ctx context.Context, manifestPath string, image workspace.Image) (distributionAssets, error) {
	if image.WSL == nil {
		return distributionAssets{}, nil
	}

	dist := image.WSL.Distribution
	if dist == nil {
		dist = &workspace.WSLDistributionConfig{}
	}

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

	oobeCfg := workspace.NormalizeOOBEConfig(image.WSL.OOBE)
	effectiveMode := workspace.EffectiveOOBEMode(image.WSL)
	oobeDefaultUID := image.WSL.DefaultUser.UID
	if oobeDefaultUID <= 0 {
		oobeDefaultUID = 1000
	}
	oobeDefaultName := ""
	if effectiveMode == workspace.OOBEModePredefined {
		oobeDefaultName = strings.TrimSpace(image.WSL.DefaultUser.Name)
	}
	if dist.OOBE != nil {
		if strings.TrimSpace(dist.OOBE.DefaultName) != "" {
			oobeDefaultName = strings.TrimSpace(dist.OOBE.DefaultName)
		}
		if dist.OOBE.DefaultUID != nil && *dist.OOBE.DefaultUID > 0 {
			oobeDefaultUID = *dist.OOBE.DefaultUID
		}
	}
	oobeCommand := ""
	oobeScriptPath := ""
	oobeScript := ""
	backupNativeConf := false
	if dist.OOBE != nil && strings.TrimSpace(dist.OOBE.Command) != "" {
		oobeCommand = strings.TrimSpace(dist.OOBE.Command)
	} else {
		strategy := strings.ToLower(strings.TrimSpace(oobeCfg.Strategy))
		if strategy != workspace.OOBEStrategyNative {
			promptPassword := true
			if oobeCfg.PromptForPassword != nil {
				promptPassword = *oobeCfg.PromptForPassword
			}
			oobeScriptPath = defaultOOBEScriptPath
			oobeCommand = oobeScriptPath
			oobeScript = renderWSLBOOBEScript(image.WSL, effectiveMode, strategy, promptPassword, oobeDefaultName, oobeDefaultUID)
			backupNativeConf = strategy == workspace.OOBEStrategyHybrid
		}
	}

	var b strings.Builder
	if strings.TrimSpace(oobeCommand) != "" || strings.TrimSpace(oobeDefaultName) != "" || oobeDefaultUID > 0 {
		b.WriteString("[oobe]\n")
		if strings.TrimSpace(oobeDefaultName) != "" {
			b.WriteString("defaultName=" + strings.TrimSpace(oobeDefaultName) + "\n")
		}
		if oobeDefaultUID > 0 {
			b.WriteString(fmt.Sprintf("defaultUid=%d\n", oobeDefaultUID))
		}
		if strings.TrimSpace(oobeCommand) != "" {
			b.WriteString("command=" + strings.TrimSpace(oobeCommand) + "\n")
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
	wtEnabledSet := false
	wtEnabled := false
	if dist.WindowsTerminal != nil {
		if dist.WindowsTerminal.Enabled != nil {
			wtEnabledSet = true
			wtEnabled = *dist.WindowsTerminal.Enabled
		}
		if strings.TrimSpace(dist.WindowsTerminal.ProfileTemplate) != "" {
			wtTemplatePath = strings.TrimSpace(dist.WindowsTerminal.ProfileTemplate)
		}
		if len(bytes.TrimSpace(dist.WindowsTerminal.Template)) > 0 {
			resolvedTemplate, err := resolveWTTemplate(manifestPath, dist.WindowsTerminal.Template)
			if err != nil {
				return distributionAssets{}, err
			}
			wtTemplate = resolvedTemplate
		}
	}
	if wtTemplatePath == "" && wtTemplate != "" {
		wtTemplatePath = defaultWTProfilePath
	}
	if iconPath != "" && wtTemplate == "" {
		if wtTemplatePath == "" {
			wtTemplatePath = defaultWTProfilePath
		}
		wtTemplatePath = defaultWTProfilePath
		wtTemplate = fmt.Sprintf("{\n  \"profiles\": [\n    {\n      \"icon\": %q\n    }\n  ]\n}\n", iconPath)
	}
	if wtEnabledSet || wtTemplatePath != "" {
		b.WriteString("[windowsterminal]\n")
		if wtEnabledSet {
			b.WriteString("enabled=" + boolToString(wtEnabled) + "\n")
		}
		if wtTemplatePath != "" {
			b.WriteString("ProfileTemplate=" + wtTemplatePath + "\n")
		}
	}
	if len(dist.ExtraSections) > 0 {
		sectionNames := make([]string, 0, len(dist.ExtraSections))
		for section := range dist.ExtraSections {
			sectionNames = append(sectionNames, section)
		}
		sort.Strings(sectionNames)
		for _, section := range sectionNames {
			kv := dist.ExtraSections[section]
			if len(kv) == 0 {
				continue
			}
			b.WriteString("[" + section + "]\n")
			keys := make([]string, 0, len(kv))
			for k := range kv {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, key := range keys {
				val, err := renderDistributionScalarValue(kv[key])
				if err != nil {
					continue
				}
				b.WriteString(key + "=" + val + "\n")
			}
		}
	}
	return distributionAssets{
		Conf:                  b.String(),
		IconPath:              iconPath,
		IconBytes:             iconBytes,
		WTProfileTemplatePath: wtTemplatePath,
		WTProfileTemplateJSON: wtTemplate,
		OOBEScriptPath:        oobeScriptPath,
		OOBEScript:            oobeScript,
		BackupNativeConf:      backupNativeConf,
	}, nil
}

func renderDistributionScalarValue(v interface{}) (string, error) {
	switch t := v.(type) {
	case bool:
		return boolToString(t), nil
	case string:
		return t, nil
	case float64:
		return fmt.Sprintf("%v", t), nil
	case float32:
		return fmt.Sprintf("%v", t), nil
	case int:
		return fmt.Sprintf("%d", t), nil
	case int64:
		return fmt.Sprintf("%d", t), nil
	case int32:
		return fmt.Sprintf("%d", t), nil
	case uint:
		return fmt.Sprintf("%d", t), nil
	case uint64:
		return fmt.Sprintf("%d", t), nil
	case uint32:
		return fmt.Sprintf("%d", t), nil
	default:
		return "", fmt.Errorf("unsupported value type %T", v)
	}
}

func renderWSLBOOBEScript(cfg *workspace.WSLImageConfig, mode, strategy string, promptPassword bool, defaultName string, defaultUID int) string {
	if cfg == nil {
		return ""
	}
	userName := strings.TrimSpace(defaultName)
	if userName == "" {
		userName = strings.TrimSpace(cfg.DefaultUser.Name)
	}
	uid := defaultUID
	if uid <= 0 {
		uid = cfg.DefaultUser.UID
	}
	if uid <= 0 {
		uid = 1000
	}
	gid := cfg.DefaultUser.GID
	if gid <= 0 {
		gid = uid
	}
	if mode == "" {
		mode = workspace.OOBEModeAuto
	}
	if strategy == "" {
		strategy = workspace.OOBEStrategyHybrid
	}
	prompt := "false"
	if promptPassword {
		prompt = "true"
	}
return fmt.Sprintf(`#!/bin/sh
set -eu
PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:${PATH:-}"
export PATH

WSLB_MODE=%q
WSLB_STRATEGY=%q
WSLB_PROMPT_PASSWORD=%q
WSLB_DEFAULT_USER=%q
WSLB_DEFAULT_UID=%d
WSLB_DEFAULT_GID=%d
WSLB_SCRIPT_PATH=%q

ensure_wslconf_default() {
  user="$1"
  [ -n "$user" ] || return 0
  mkdir -p /etc
  touch /etc/wsl.conf
  if grep -q '^\[user\]' /etc/wsl.conf 2>/dev/null; then
    if grep -q '^default=' /etc/wsl.conf 2>/dev/null; then
      sed -i "s/^default=.*/default=${user}/" /etc/wsl.conf || true
    else
      printf '\ndefault=%%s\n' "$user" >> /etc/wsl.conf
    fi
  else
    printf '\n[user]\ndefault=%%s\n' "$user" >> /etc/wsl.conf
  fi
}

have_getent() {
  command -v getent >/dev/null 2>&1
}

get_passwd_entry() {
  key="$1"
  if have_getent; then
    getent passwd "$key" 2>/dev/null || true
    return 0
  fi
  if [ ! -f /etc/passwd ]; then
    return 0
  fi
  if echo "$key" | grep -Eq '^[0-9]+$'; then
    awk -F: -v k="$key" '$3==k { print; exit }' /etc/passwd || true
  else
    awk -F: -v k="$key" '$1==k { print; exit }' /etc/passwd || true
  fi
}

get_group_entry() {
  key="$1"
  if have_getent; then
    getent group "$key" 2>/dev/null || true
    return 0
  fi
  if [ ! -f /etc/group ]; then
    return 0
  fi
  if echo "$key" | grep -Eq '^[0-9]+$'; then
    awk -F: -v k="$key" '$3==k { print; exit }' /etc/group || true
  else
    awk -F: -v k="$key" '$1==k { print; exit }' /etc/group || true
  fi
}

existing_user_uid_1000() {
  entry="$(get_passwd_entry "$WSLB_DEFAULT_UID")"
  [ -n "$entry" ] || return 1
  echo "$entry" | cut -d: -f1
}

create_group_if_needed() {
  user="$1"
  if [ -n "$(get_group_entry "$WSLB_DEFAULT_GID")" ]; then
    return 0
  fi
  if command -v groupadd >/dev/null 2>&1; then
    groupadd -g "$WSLB_DEFAULT_GID" "$user" || true
  elif command -v addgroup >/dev/null 2>&1; then
    addgroup --gid "$WSLB_DEFAULT_GID" "$user" >/dev/null 2>&1 || addgroup -g "$WSLB_DEFAULT_GID" "$user" >/dev/null 2>&1 || true
  fi
}

detect_default_groups() {
  groups=""
  for g in sudo wheel adm cdrom dip plugdev audio video netdev docker; do
    if [ -n "$(get_group_entry "$g")" ]; then
      groups="${groups:+$groups,}$g"
    fi
  done
  echo "$groups"
}

create_user() {
  user="$1"
  groups="$(detect_default_groups)"
  create_group_if_needed "$user"

  if command -v useradd >/dev/null 2>&1; then
    if [ -n "$groups" ]; then
      useradd -m -u "$WSLB_DEFAULT_UID" -g "$WSLB_DEFAULT_GID" -G "$groups" -s /bin/bash "$user"
    else
      useradd -m -u "$WSLB_DEFAULT_UID" -g "$WSLB_DEFAULT_GID" -s /bin/bash "$user"
    fi
    return 0
  fi

  if command -v adduser >/dev/null 2>&1; then
    if adduser --help 2>&1 | grep -q -- '--uid'; then
      adduser --uid "$WSLB_DEFAULT_UID" --gid "$WSLB_DEFAULT_GID" --gecos '' "$user"
    elif adduser --help 2>&1 | grep -q -- '\-D'; then
      adduser -D -u "$WSLB_DEFAULT_UID" -G "$user" "$user" || adduser -D "$user"
    else
      adduser "$user" || true
    fi
    if [ -n "$groups" ] && command -v usermod >/dev/null 2>&1; then
      usermod -aG "$groups" "$user" || true
    fi
    id -u "$user" >/dev/null 2>&1 && return 0
  fi

  echo "no supported user creation utility found (useradd/adduser)" >&2
  return 40
}

run_native_oobe_if_configured() {
  if [ "$WSLB_STRATEGY" = "wslb-only" ]; then
    return 0
  fi
  native_conf="/etc/wsl-distribution.conf.wslb-native"
  [ -f "$native_conf" ] || return 0
  native_cmd="$(awk '
    BEGIN { in_oobe=0 }
    /^\[/ {
      line=tolower($0)
      in_oobe=(line=="[oobe]")
      next
    }
    in_oobe {
      if (tolower($0) ~ /^command[[:space:]]*=/) {
        sub(/^[^=]*=/, "", $0)
        print $0
        exit
      }
    }
  ' "$native_conf" | tr -d '\r')"
  [ -n "$native_cmd" ] || return 0
  if [ "$native_cmd" = "$WSLB_SCRIPT_PATH" ]; then
    return 0
  fi
  sh -lc "$native_cmd"
}

if [ "$WSLB_STRATEGY" = "native-only" ]; then
  run_native_oobe_if_configured
  exit 0
fi

run_native_oobe_if_configured || true

existing_user="$(existing_user_uid_1000 || true)"
if [ -n "$existing_user" ]; then
  ensure_wslconf_default "$existing_user"
  exit 0
fi

target_user=""
if [ "$WSLB_MODE" = "interactive" ]; then
  while true; do
    printf 'Enter new UNIX username: '
    read -r target_user || true
    if [ -n "$target_user" ]; then
      break
    fi
  done
else
  target_user="$WSLB_DEFAULT_USER"
fi

if [ -z "$target_user" ]; then
  echo "no username provided for OOBE" >&2
  exit 41
fi

if ! id -u "$target_user" >/dev/null 2>&1; then
  create_user "$target_user"
fi

if [ "$WSLB_PROMPT_PASSWORD" = "true" ]; then
  passwd "$target_user" || true
fi

ensure_wslconf_default "$target_user"
`, mode, strategy, prompt, userName, uid, gid, defaultOOBEScriptPath)
}

func resolveWTTemplate(manifestPath string, raw json.RawMessage) (string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return "", nil
	}

	var pathValue string
	if err := json.Unmarshal(trimmed, &pathValue); err == nil {
		pathValue = strings.TrimSpace(pathValue)
		if pathValue == "" {
			return "", nil
		}
		resolvedPath := pathValue
		if !filepath.IsAbs(resolvedPath) {
			resolvedPath = workspace.ResolvePath(manifestPath, pathValue)
		}
		b, err := os.ReadFile(resolvedPath)
		if err != nil {
			return "", fmt.Errorf("windowsterminal.template path %q: %w", pathValue, err)
		}
		if !json.Valid(b) {
			return "", fmt.Errorf("windowsterminal.template file %q is not valid JSON", pathValue)
		}
		var out bytes.Buffer
		if err := json.Indent(&out, b, "", "  "); err != nil {
			return "", fmt.Errorf("windowsterminal.template file %q: %w", pathValue, err)
		}
		out.WriteByte('\n')
		return out.String(), nil
	}

	if !json.Valid(trimmed) {
		return "", fmt.Errorf("windowsterminal.template must be JSON object or string path")
	}
	var out bytes.Buffer
	if err := json.Indent(&out, trimmed, "", "  "); err != nil {
		return "", fmt.Errorf("windowsterminal.template: %w", err)
	}
	out.WriteByte('\n')
	return out.String(), nil
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
