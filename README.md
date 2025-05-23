# WSLB - Windows Subsystem for Linux Builder

WSLB is a command-line tool that simplifies building and installing Windows Subsystem for Linux (WSL) distributions from Docker images.

## Features

- Build custom WSL distributions from any Docker image
- Install WSL distributions directly from Docker image URLs
- Manage WSL distributions (list, stop, remove)

## Installation

### Prerequisites

- Windows operating system with WSL enabled
- Docker installed and running
- Go 1.16 or later (for building from source)


### Install With Go
```bash
go install -a github.com/wsl-images/wslb@latest
```

### Install With Scoop
```bash
scoop bucket add wsl-images-bucket https://github.com/wsl-images/wsl-images-bucket
scoop install wsl-images-bucket/wslb 
```

### Install With NPM (Windows x86_64 Only)

[@wsl-images/wslb-cli](https://www.npmjs.com/package/@wsl-images/wslb-cli)
```bash
npm install -g @wsl-images/wslb-cli
```

### Antivirus and Anti Malware Problems with Scoop
```bash
Add-MpPreference -ExclusionPath "$($env:programdata)\scoop", "$($env:scoop)"
```

To Undo this change
```bash
Remove-MpPreference -ExclusionPath "$($env:programdata)\scoop", "$($env:scoop)"
```

### Building from source

```bash
git clone https://github.com/wsl-images/wslb.git
cd wslb
./build.sh
```

The executable will be available in the `bin` directory.

## Usage

### Build a WSL distribution from a Docker image

```bash
wslb build ubuntu:20.04
```

This will create a `.wsl` file in the current directory.

Options:
- `-o, --output`: Specify output directory (default: current directory)

### Install a WSL distribution

From Docker image URL:

```bash
wslb install ubuntu:20.04
```

From a pre-built .wsl file:

```bash
wslb install -f ./ubuntu.wsl
```

Options:
- `-n, --name`: Specify a custom name for the WSL distribution
- `-f, --file`: Path to a pre-built .wsl file

### List WSL distributions

```bash
wslb ls
# or
wslb list
```

Options:
- `--all`: List all distributions, including those being installed or uninstalled
- `--running`: List only distributions that are currently running
- `-q, --quiet`: Only show distribution names
- `-v, --verbose`: Show detailed information about all distributions
- `-o, --online`: Display a list of available distributions for install

### Remove a WSL distribution

```bash
wslb rm <Distro>
```

Unregisters the distribution and deletes the root filesystem.

### Stop a WSL distribution

```bash
wslb stop <Distro>
```

Terminates the specified WSL distribution.

### Shutdown all WSL distributions

```bash
wslb shutdown
```

Immediately terminates all running distributions and the WSL 2 lightweight utility virtual machine.

### Show WSL status

```bash
wslb status
```

Shows the status of Windows Subsystem for Linux.

### Display version information

```bash
wslb version
```

Prints the version information of WSLB.

## Examples

Build Ubuntu 22.04 and save to a specific directory:

```bash
wslb build ubuntu:22.04 -o ~/wsl-images
```

Install Debian with a custom name:

```bash
wslb install debian:bullseye -n MyDebianWSL
```

List all running WSL distributions:

```bash
wslb ls --running
```

[//]: # (## Configuration)

[//]: # ()
[//]: # (WSLB looks for a configuration file at `$HOME/.wslb/wslb.yaml`, or you can specify a custom config file:)

[//]: # ()
[//]: # (```bash)

[//]: # (wslb --config /path/to/config.yaml build ubuntu:20.04)

[//]: # (```)

## Logs

Logs are stored in `$HOME/.wslb/logs/wslb.log`
