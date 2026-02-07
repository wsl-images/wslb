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
	default:
		return features.Metadata{}, "", fmt.Errorf("unknown internal feature: %s", name)
	}
}
