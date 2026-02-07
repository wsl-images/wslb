#!/usr/bin/env bash
set -euo pipefail

id -u ubuntu >/dev/null
grep -q '^\[user\]' /etc/wsl.conf
grep -q '^default=ubuntu$' /etc/wsl.conf
