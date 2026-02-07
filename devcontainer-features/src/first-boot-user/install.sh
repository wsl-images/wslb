#!/usr/bin/env bash
set -euo pipefail

USERNAME="${USERNAME:-${username:-dev}}"
USER_UID="${USER_UID:-${USERUID:-${userUid:-1000}}}"
USER_GID="${USER_GID:-${USERGID:-${userGid:-1000}}}"
USER_HOME="${USER_HOME:-${home:-/home/${USERNAME}}}"

if ! id -u "$USERNAME" >/dev/null 2>&1; then
  existing_user="$(getent passwd "$USER_UID" 2>/dev/null | cut -d: -f1 || true)"
  if [ -n "$existing_user" ]; then
    USERNAME="$existing_user"
  fi
fi

if ! getent group "$USER_GID" >/dev/null 2>&1; then
  groupadd -g "$USER_GID" "$USERNAME" || true
fi

if ! id -u "$USERNAME" >/dev/null 2>&1; then
  useradd -m -u "$USER_UID" -g "$USER_GID" -s /bin/bash "$USERNAME" || true
fi

mkdir -p "$USER_HOME"
chown -R "$USER_UID":"$USER_GID" "$USER_HOME" || true

mkdir -p /etc
if ! grep -q '^\[user\]' /etc/wsl.conf 2>/dev/null; then
  printf '[user]\n' >>/etc/wsl.conf
fi

if grep -q '^default=' /etc/wsl.conf 2>/dev/null; then
  sed -i "s/^default=.*/default=${USERNAME}/" /etc/wsl.conf
else
  printf 'default=%s\n' "$USERNAME" >>/etc/wsl.conf
fi
