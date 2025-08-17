# Simple Makefile for the syncai project

# Variables (can be overridden: e.g., `make run CONFIG=myconfig.json`)
GO ?= go
BUILD_DIR ?= bin
BINARY_NAME ?= syncai
CONFIG ?= syncai.json
LDFLAGS ?= -s -w

.PHONY: all build run clean

all: build

# Build the binary into ./bin/syncai from the cmd entrypoint
build:
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd

# Run the application with the default configuration file (syncai.json)
run: build
	$(GO) run ./cmd -config $(CONFIG)

# Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
