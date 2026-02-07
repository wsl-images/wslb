package internalfeatures

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wsl-images/wslb/internal/features"
)

type Provider struct {
	cache string
}

func New(cache string) *Provider {
	if cache == "" {
		cache = filepath.Join(os.TempDir(), "wslb-internal-features")
	}
	_ = os.MkdirAll(cache, 0o755)
	return &Provider{cache: cache}
}

func (p *Provider) Kind() string { return "internal" }

func (p *Provider) Describe(_ context.Context, ref features.Ref) (features.Metadata, error) {
	meta, _, err := p.feature(ref.Name)
	return meta, err
}

func (p *Provider) Resolve(_ context.Context, ref features.Ref, options map[string]interface{}) (features.ScriptSpec, error) {
	meta, script, err := p.feature(ref.Name)
	if err != nil {
		return features.ScriptSpec{}, err
	}
	dir := filepath.Join(p.cache, ref.Name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return features.ScriptSpec{}, err
	}
	path := filepath.Join(dir, "install.sh")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		return features.ScriptSpec{}, err
	}
	env := map[string]string{}
	for k, v := range options {
		env[k] = fmt.Sprintf("%v", v)
	}
	return features.ScriptSpec{Name: ref.Raw, Path: path, Env: env, Metadata: meta}, nil
}

func (p *Provider) feature(name string) (features.Metadata, string, error) {
	switch name {
	case "first-boot-user":
		return features.Metadata{
				ID:          "first-boot-user",
				Name:        "First Boot User",
				Version:     "1.0.0",
				Description: "ensure user exists and default user is configured",
				Options: map[string]features.Option{
					"USERNAME": {Type: "string", Default: "dev", Description: "username"},
					"USER_UID": {Type: "string", Default: "1000", Description: "uid"},
					"USER_GID": {Type: "string", Default: "1000", Description: "gid"},
					"HOME":     {Type: "string", Default: "/home/dev", Description: "home"},
				},
			}, `#!/bin/sh
set -eu
USERNAME="${USERNAME:-dev}"
USER_UID="${USER_UID:-1000}"
USER_GID="${USER_GID:-1000}"
USER_HOME="${HOME:-/home/${USERNAME}}"
if ! getent group "$USER_GID" >/dev/null 2>&1; then
  groupadd -g "$USER_GID" "$USERNAME" || true
fi
if ! id -u "$USERNAME" >/dev/null 2>&1; then
  useradd -m -u "$USER_UID" -g "$USER_GID" -s /bin/bash "$USERNAME" || true
fi
mkdir -p "$USER_HOME"
chown -R "$USER_UID":"$USER_GID" "$USER_HOME" || true
mkdir -p /etc
grep -q '^\[user\]' /etc/wsl.conf 2>/dev/null || echo '[user]' >> /etc/wsl.conf
if grep -q '^default=' /etc/wsl.conf 2>/dev/null; then
  sed -i "s/^default=.*/default=${USERNAME}/" /etc/wsl.conf
else
  echo "default=${USERNAME}" >> /etc/wsl.conf
fi
`, nil
	case "persist-home":
		return features.Metadata{
				ID:          "persist-home",
				Name:        "Persist Home",
				Version:     "1.0.0",
				Description: "configure persistent home mount wiring",
				Options: map[string]features.Option{
					"MOUNT_POINT":  {Type: "string", Default: "/home", Description: "mount point"},
					"STATE_SOURCE": {Type: "string", Default: "/mnt/wsl/wslb-state", Description: "state source"},
					"SYSTEMD":      {Type: "string", Default: "true", Description: "enable systemd unit"},
				},
			}, `#!/bin/sh
set -eu
MOUNT_POINT="${MOUNT_POINT:-/home}"
STATE_SOURCE="${STATE_SOURCE:-/mnt/wsl/wslb-state}"
SYSTEMD_FLAG="${SYSTEMD:-true}"
mkdir -p "$MOUNT_POINT"
mkdir -p /usr/local/sbin
cat >/usr/local/sbin/wslb-mount-state.sh <<'EOF'
#!/bin/sh
set -eu
SRC="${STATE_SOURCE:-/mnt/wsl/wslb-state}"
DST="${MOUNT_POINT:-/home}"
mkdir -p "$DST"
if [ -d "$SRC" ] && [ ! -L "$DST" ]; then
  mountpoint -q "$DST" || mount --bind "$SRC" "$DST" || true
fi
EOF
chmod +x /usr/local/sbin/wslb-mount-state.sh
if [ "$SYSTEMD_FLAG" = "true" ]; then
  mkdir -p /etc/systemd/system
  cat >/etc/systemd/system/wslb-persist-home.service <<'EOF'
[Unit]
Description=WSLB Persist Home Mount
After=local-fs.target

[Service]
Type=oneshot
ExecStart=/usr/local/sbin/wslb-mount-state.sh
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF
fi
`, nil
	case "wsl-prereqs":
		return features.Metadata{
				ID:          "wsl-prereqs",
				Name:        "WSL Prerequisites",
				Version:     "1.0.0",
				Description: "install/prepare user-management prerequisites across common Linux distros for WSL first-launch setup",
				Options: map[string]features.Option{
					"INSTALL_SUDO":     {Type: "string", Default: "true", Description: "install sudo when package manager supports it"},
					"INSTALL_BASH":     {Type: "string", Default: "true", Description: "install bash when package manager supports it"},
					"INSTALL_CA_CERTS": {Type: "string", Default: "true", Description: "install ca-certificates when package manager supports it"},
				},
			}, `#!/bin/sh
set -eu
PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:${PATH:-}"
export PATH

INSTALL_SUDO="${INSTALL_SUDO:-true}"
INSTALL_BASH="${INSTALL_BASH:-true}"
INSTALL_CA_CERTS="${INSTALL_CA_CERTS:-true}"

need_user_tools() {
  command -v id >/dev/null 2>&1 || return 0
  command -v passwd >/dev/null 2>&1 || return 0
  if command -v useradd >/dev/null 2>&1 && command -v groupadd >/dev/null 2>&1; then
    return 1
  fi
  if command -v adduser >/dev/null 2>&1 && command -v addgroup >/dev/null 2>&1; then
    return 1
  fi
  return 0
}

bool_true() {
  case "${1:-}" in
    1|true|TRUE|True|yes|YES|Yes|on|ON|On) return 0 ;;
  esac
  return 1
}

pkg_line() {
  line="$1"
  for p in "$@"; do :; done
  echo "$line"
}

install_with_apt() {
  export DEBIAN_FRONTEND=noninteractive
  apt-get update
  pkgs="passwd"
  if bool_true "$INSTALL_SUDO"; then pkgs="$pkgs sudo"; fi
  if bool_true "$INSTALL_BASH"; then pkgs="$pkgs bash"; fi
  if bool_true "$INSTALL_CA_CERTS"; then pkgs="$pkgs ca-certificates"; fi
  apt-get install -y --no-install-recommends $pkgs || apt-get install -y $pkgs
  rm -rf /var/lib/apt/lists/*
}

install_with_dnf() {
  pkgs="shadow-utils passwd"
  if bool_true "$INSTALL_SUDO"; then pkgs="$pkgs sudo"; fi
  if bool_true "$INSTALL_BASH"; then pkgs="$pkgs bash"; fi
  if bool_true "$INSTALL_CA_CERTS"; then pkgs="$pkgs ca-certificates"; fi
  pkgs="$pkgs procps-ng"
  dnf -y install $pkgs
}

install_with_microdnf() {
  pkgs="shadow-utils passwd"
  if bool_true "$INSTALL_SUDO"; then pkgs="$pkgs sudo"; fi
  if bool_true "$INSTALL_BASH"; then pkgs="$pkgs bash"; fi
  if bool_true "$INSTALL_CA_CERTS"; then pkgs="$pkgs ca-certificates"; fi
  microdnf -y install $pkgs
}

install_with_yum() {
  yum -y install shadow-utils || true
  if ! command -v useradd >/dev/null 2>&1 || ! command -v groupadd >/dev/null 2>&1; then
    yum -y install shadow || true
    yum -y install shadow-tools || true
  fi
  if bool_true "$INSTALL_SUDO"; then yum -y install sudo || true; fi
  if bool_true "$INSTALL_BASH"; then yum -y install bash || true; fi
  if bool_true "$INSTALL_CA_CERTS"; then yum -y install ca-certificates || true; fi
}

install_with_apk() {
  pkgs="shadow"
  if bool_true "$INSTALL_SUDO"; then pkgs="$pkgs sudo"; fi
  if bool_true "$INSTALL_BASH"; then pkgs="$pkgs bash"; fi
  if bool_true "$INSTALL_CA_CERTS"; then pkgs="$pkgs ca-certificates"; fi
  apk add --no-cache $pkgs
}

install_with_zypper() {
  pkgs="shadow"
  if bool_true "$INSTALL_SUDO"; then pkgs="$pkgs sudo"; fi
  if bool_true "$INSTALL_BASH"; then pkgs="$pkgs bash"; fi
  if bool_true "$INSTALL_CA_CERTS"; then pkgs="$pkgs ca-certificates"; fi
  zypper --non-interactive install --no-recommends $pkgs
}

install_with_pacman() {
  pkgs="shadow"
  if bool_true "$INSTALL_SUDO"; then pkgs="$pkgs sudo"; fi
  if bool_true "$INSTALL_BASH"; then pkgs="$pkgs bash"; fi
  if bool_true "$INSTALL_CA_CERTS"; then pkgs="$pkgs ca-certificates"; fi
  pacman -Sy --noconfirm --needed $pkgs
}

install_with_xbps() {
  pkgs="shadow"
  if bool_true "$INSTALL_SUDO"; then pkgs="$pkgs sudo"; fi
  if bool_true "$INSTALL_BASH"; then pkgs="$pkgs bash"; fi
  if bool_true "$INSTALL_CA_CERTS"; then pkgs="$pkgs ca-certificates"; fi
  xbps-install -Sy -y $pkgs
}

install_with_tdnf() {
  tdnf install -y shadow-utils || true
  if ! command -v useradd >/dev/null 2>&1 || ! command -v groupadd >/dev/null 2>&1; then
    tdnf install -y shadow || true
    tdnf install -y shadow-tools || true
  fi
  if bool_true "$INSTALL_SUDO"; then tdnf install -y sudo || true; fi
  if bool_true "$INSTALL_BASH"; then tdnf install -y bash || true; fi
  if bool_true "$INSTALL_CA_CERTS"; then tdnf install -y ca-certificates || true; fi
}

install_prereqs() {
  if command -v apt-get >/dev/null 2>&1; then install_with_apt; return 0; fi
  if command -v dnf >/dev/null 2>&1; then install_with_dnf; return 0; fi
  if command -v microdnf >/dev/null 2>&1; then install_with_microdnf; return 0; fi
  if command -v tdnf >/dev/null 2>&1; then install_with_tdnf; return 0; fi
  if command -v yum >/dev/null 2>&1; then install_with_yum; return 0; fi
  if command -v apk >/dev/null 2>&1; then install_with_apk; return 0; fi
  if command -v zypper >/dev/null 2>&1; then install_with_zypper; return 0; fi
  if command -v pacman >/dev/null 2>&1; then install_with_pacman; return 0; fi
  if command -v xbps-install >/dev/null 2>&1; then install_with_xbps; return 0; fi
  return 1
}

ensure_getent_fallback() {
  if command -v getent >/dev/null 2>&1; then
    return 0
  fi
  mkdir -p /usr/local/bin
  cat >/usr/local/bin/getent <<'EOF'
#!/bin/sh
set -eu
db="${1:-}"
key="${2:-}"
case "$db" in
  passwd)
    if [ -z "$key" ]; then cat /etc/passwd; exit 0; fi
    if echo "$key" | grep -Eq '^[0-9]+$'; then
      awk -F: -v k="$key" '$3==k { print; found=1; exit } END { exit(found ? 0 : 2) }' /etc/passwd
    else
      awk -F: -v k="$key" '$1==k { print; found=1; exit } END { exit(found ? 0 : 2) }' /etc/passwd
    fi
    ;;
  group)
    if [ -z "$key" ]; then cat /etc/group; exit 0; fi
    if echo "$key" | grep -Eq '^[0-9]+$'; then
      awk -F: -v k="$key" '$3==k { print; found=1; exit } END { exit(found ? 0 : 2) }' /etc/group
    else
      awk -F: -v k="$key" '$1==k { print; found=1; exit } END { exit(found ? 0 : 2) }' /etc/group
    fi
    ;;
  *)
    exit 2
    ;;
esac
EOF
  chmod +x /usr/local/bin/getent
}

if need_user_tools; then
  install_prereqs || true
fi

ensure_getent_fallback

# Validate minimum capabilities expected by WSLB OOBE logic.
if ! command -v id >/dev/null 2>&1; then
  echo "wsl-prereqs: missing id command after bootstrap" >&2
  exit 42
fi
if ! command -v passwd >/dev/null 2>&1; then
  echo "wsl-prereqs: missing passwd command after bootstrap" >&2
  exit 42
fi
if command -v useradd >/dev/null 2>&1 && command -v groupadd >/dev/null 2>&1; then
  exit 0
fi
if command -v adduser >/dev/null 2>&1 && command -v addgroup >/dev/null 2>&1; then
  exit 0
fi
echo "wsl-prereqs: missing user creation tools (useradd/groupadd or adduser/addgroup)" >&2
exit 42
`, nil
	default:
		return features.Metadata{}, "", fmt.Errorf("unknown internal feature: %s", name)
	}
}
