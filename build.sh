#!/bin/bash

# Check if version argument is provided
if [ -z "$1" ]; then
    echo "Error: Version argument is required"
    echo "Usage: $0 <version>"
    exit 1
fi

VERSION="$1"
BINARY_NAME="wslb"
BIN_DIR="bin"
LDFLAGS="-X github.com/wsl-images/wslb/internal/version.Version=$VERSION"

# Create bin directory if it doesn't exist
if [ ! -d "$BIN_DIR" ]; then
    mkdir -p "$BIN_DIR"
    echo "Created bin directory"
fi

# Build Linux binary
echo "Building WSLB Linux binary..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$LDFLAGS" -o "$BIN_DIR/$BINARY_NAME" .

# Build Windows binary
echo "Building WSLB Windows binary..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$LDFLAGS" -o "$BIN_DIR/${BINARY_NAME}.exe" .

echo "Build completed successfully!"