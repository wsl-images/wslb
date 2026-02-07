# First Boot User

Creates or updates the default Linux user expected by WSL:

- ensures group and user exist
- ensures home directory ownership
- writes `[user] default=<username>` in `/etc/wsl.conf`
