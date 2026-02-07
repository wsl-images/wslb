#!/usr/bin/env bash
set -euo pipefail

test -x /usr/local/sbin/wslb-mount-state.sh
test -f /etc/systemd/system/wslb-persist-home.service
