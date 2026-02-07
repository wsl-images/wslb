#!/usr/bin/env bash
set -euo pipefail

MOUNT_POINT="${MOUNT_POINT:-${mountPoint:-/home}}"
STATE_SOURCE="${STATE_SOURCE:-${stateSource:-/mnt/wsl/wslb-state}}"
SYSTEMD_FLAG="${SYSTEMD:-${systemd:-true}}"

mkdir -p "$MOUNT_POINT"
mkdir -p /usr/local/sbin

cat >/usr/local/sbin/wslb-mount-state.sh <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
SRC="${STATE_SOURCE:-/mnt/wsl/wslb-state}"
DST="${MOUNT_POINT:-/home}"
mkdir -p "$DST"
if [ -d "$SRC" ] && [ ! -L "$DST" ]; then
  mountpoint -q "$DST" || mount --bind "$SRC" "$DST" || true
fi
EOF
chmod +x /usr/local/sbin/wslb-mount-state.sh

if [[ "$SYSTEMD_FLAG" == "true" || "$SYSTEMD_FLAG" == "1" ]]; then
  mkdir -p /etc/systemd/system
  cat >/etc/systemd/system/wslb-persist-home.service <<EOF
[Unit]
Description=WSLB Persist Home Mount
After=local-fs.target

[Service]
Type=oneshot
Environment=MOUNT_POINT=${MOUNT_POINT}
Environment=STATE_SOURCE=${STATE_SOURCE}
ExecStart=/usr/local/sbin/wslb-mount-state.sh
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF
fi
