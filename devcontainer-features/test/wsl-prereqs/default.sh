#!/usr/bin/env bash
set -euo pipefail

command -v id
command -v passwd
if command -v useradd >/dev/null 2>&1 && command -v groupadd >/dev/null 2>&1; then
  exit 0
fi
if command -v adduser >/dev/null 2>&1 && command -v addgroup >/dev/null 2>&1; then
  exit 0
fi
echo "Missing user creation tooling" >&2
exit 1
