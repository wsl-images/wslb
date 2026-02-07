# WSL Prerequisites

Installs a baseline of user-management tools used by WSL OOBE flows:

- user/group management (`useradd`/`groupadd` or `adduser`/`addgroup`)
- `passwd`
- optional `sudo`, `bash`, and `ca-certificates`
- `getent` fallback shim when distro images do not include it
