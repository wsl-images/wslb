BINARY_NAME = wslb
VERSION ?= 0.1.0

LDFLAGS = -s -w

ifeq ($(OS),Windows_NT)
    BIN_DIR = bin
    LDFLAGS = -s -w -X github.com/wsl-images/wslb/internal/version.Version=$(VERSION)
else
    BIN_DIR = ./bin
    LDFLAGS = -s -w -X github.com/wsl-images/wslb/internal/version.Version=$(VERSION)
endif

.PHONY: all linux windows clean

all: linux windows

linux:
ifeq ($(OS),Windows_NT)
	@echo "Building Linux binary on Windows host..."
	if not exist $(BIN_DIR) mkdir $(BIN_DIR)
	cmd /C "set CGO_ENABLED=0&&set GOOS=linux&&set GOARCH=amd64&&go build -ldflags=-s -ldflags=-w -o $(BIN_DIR)\$(BINARY_NAME) ."
else
	@echo "Building Linux binary (using Linux shell)..."
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) .
endif

windows:
ifeq ($(OS),Windows_NT)
	@echo "Building Windows binary on Windows host (native)..."
	if not exist $(BIN_DIR) mkdir $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)\$(BINARY_NAME).exe .
else
	@echo "Building Windows binary (using Linux shell)..."
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME).exe .
endif

clean:
ifeq ($(OS),Windows_NT)
	@echo "Cleaning bin directory..."
	if exist $(BIN_DIR) rmdir /s /q $(BIN_DIR)
else
	@echo "Cleaning bin directory..."
	rm -rf $(BIN_DIR)
endif