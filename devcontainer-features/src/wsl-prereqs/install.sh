#!/usr/bin/env bash
set -euo pipefail

PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:${PATH:-}"
export PATH

INSTALL_SUDO="${INSTALLSUDO:-${INSTALL_SUDO:-true}}"
INSTALL_BASH="${INSTALLBASH:-${INSTALL_BASH:-true}}"
INSTALL_CA_CERTS="${INSTALLCACERTS:-${INSTALL_CA_CERTS:-true}}"

bool_true() {
  case "${1:-}" in
    1|true|TRUE|True|yes|YES|Yes|on|ON|On) return 0 ;;
  esac
  return 1
}

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

install_with_apt() {
  export DEBIAN_FRONTEND=noninteractive
  apt-get update
  local pkgs="passwd"
  if bool_true "$INSTALL_SUDO"; then pkgs="$pkgs sudo"; fi
  if bool_true "$INSTALL_BASH"; then pkgs="$pkgs bash"; fi
  if bool_true "$INSTALL_CA_CERTS"; then pkgs="$pkgs ca-certificates"; fi
  apt-get install -y --no-install-recommends $pkgs || apt-get install -y $pkgs
  rm -rf /var/lib/apt/lists/*
}

install_with_dnf() {
  local pkgs="shadow-utils passwd procps-ng"
  if bool_true "$INSTALL_SUDO"; then pkgs="$pkgs sudo"; fi
  if bool_true "$INSTALL_BASH"; then pkgs="$pkgs bash"; fi
  if bool_true "$INSTALL_CA_CERTS"; then pkgs="$pkgs ca-certificates"; fi
  dnf -y install $pkgs
}

install_with_microdnf() {
  local pkgs="shadow-utils passwd"
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
  local pkgs="shadow"
  if bool_true "$INSTALL_SUDO"; then pkgs="$pkgs sudo"; fi
  if bool_true "$INSTALL_BASH"; then pkgs="$pkgs bash"; fi
  if bool_true "$INSTALL_CA_CERTS"; then pkgs="$pkgs ca-certificates"; fi
  apk add --no-cache $pkgs
}

install_with_zypper() {
  local pkgs="shadow"
  if bool_true "$INSTALL_SUDO"; then pkgs="$pkgs sudo"; fi
  if bool_true "$INSTALL_BASH"; then pkgs="$pkgs bash"; fi
  if bool_true "$INSTALL_CA_CERTS"; then pkgs="$pkgs ca-certificates"; fi
  zypper --non-interactive install --no-recommends $pkgs
}

install_with_pacman() {
  local pkgs="shadow"
  if bool_true "$INSTALL_SUDO"; then pkgs="$pkgs sudo"; fi
  if bool_true "$INSTALL_BASH"; then pkgs="$pkgs bash"; fi
  if bool_true "$INSTALL_CA_CERTS"; then pkgs="$pkgs ca-certificates"; fi
  pacman -Sy --noconfirm --needed $pkgs
}

install_with_xbps() {
  local pkgs="shadow"
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
