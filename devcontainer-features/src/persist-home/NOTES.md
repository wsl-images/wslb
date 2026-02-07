# Persist Home

Creates mount helper wiring so `/home` (or custom `mountPoint`) can bind to an external state source.

When `systemd=true`, a oneshot unit is created at:

- `/etc/systemd/system/wslb-persist-home.service`
