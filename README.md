# WSLB - Windows Subsystem for Linux Builder

WSLB is a command-line tool that simplifies building and installing Windows Subsystem for Linux (WSL) distributions from Docker images.

## Features

- Build custom WSL distributions from any Docker image
- Install WSL distributions directly from Docker image URLs
- Install pre-built WSL distribution files

## Installation

### Prerequisites

- Windows operating system with WSL enabled
- Docker installed and running
- Go 1.16 or later (for building from source)

### Building from source

```bash
git clone https://github.com/wsl-images/wslb.git
cd wslb
make windows
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

## Examples

Build Ubuntu 22.04 and save to a specific directory:

```bash
wslb build ubuntu:22.04 -o ~/wsl-images
```

Install Debian with a custom name:

```bash
wslb install debian:bullseye -n MyDebianWSL
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