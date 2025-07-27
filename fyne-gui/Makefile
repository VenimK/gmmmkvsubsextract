# Makefile for Subtitle Forge

# Variables
APP_NAME = subtitle-forge
BUILD_DIR = build
SCRIPTS_DIR = ../scripts

# Default target
.PHONY: all
all: build

# Build for current platform
.PHONY: build
build:
	@echo "Building for current platform..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME)

# Build for all platforms
.PHONY: build-all
build-all:
	@echo "Building for all platforms..."
	@chmod +x build.sh
	./build.sh

# Build for macOS
.PHONY: build-mac
build-mac:
	@echo "Building for macOS..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME)-mac

# Create a proper macOS .app bundle
.PHONY: bundle-mac
bundle-mac: build-mac
	@echo "Creating macOS app bundle..."
	@mkdir -p $(BUILD_DIR)/$(APP_NAME).app/Contents/MacOS
	@mkdir -p $(BUILD_DIR)/$(APP_NAME).app/Contents/Resources
	@cp $(BUILD_DIR)/$(APP_NAME)-mac $(BUILD_DIR)/$(APP_NAME).app/Contents/MacOS/$(APP_NAME)
	@echo '<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><dict><key>CFBundleExecutable</key><string>$(APP_NAME)</string><key>CFBundleIconFile</key><string>icon.icns</string><key>CFBundleIdentifier</key><string>com.gmm.subtitleforge</string><key>CFBundleName</key><string>Subtitle Forge</string><key>CFBundlePackageType</key><string>APPL</string><key>CFBundleShortVersionString</key><string>1.2</string><key>NSHighResolutionCapable</key><true/></dict></plist>' > $(BUILD_DIR)/$(APP_NAME).app/Contents/Info.plist

# Build for Windows
.PHONY: build-windows
build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -o $(BUILD_DIR)/$(APP_NAME).exe

# Build for Linux
.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-linux-musl-gcc go build -o $(BUILD_DIR)/$(APP_NAME)-linux

# Package the application
.PHONY: package
package: build-all
	@echo "Creating packages..."
	@mkdir -p $(BUILD_DIR)/dist
	@tar -czf $(BUILD_DIR)/dist/$(APP_NAME)-macos.tar.gz -C $(BUILD_DIR) $(APP_NAME)-mac
	@zip -j $(BUILD_DIR)/dist/$(APP_NAME)-windows.zip $(BUILD_DIR)/$(APP_NAME).exe
	@tar -czf $(BUILD_DIR)/dist/$(APP_NAME)-linux.tar.gz -C $(BUILD_DIR) $(APP_NAME)-linux

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go get fyne.io/fyne/v2@latest

# Run the application
.PHONY: run
run: build
	@echo "Running application..."
	./$(BUILD_DIR)/$(APP_NAME)

# Help target
.PHONY: help
help:
	@echo "GMM MKV Subtitles Extract Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  all         - Default target, builds for current platform"
	@echo "  build       - Build for current platform"
	@echo "  build-all   - Build for all platforms (macOS, Windows, Linux)"
	@echo "  build-mac   - Build for macOS only"
	@echo "  build-windows - Build for Windows only"
	@echo "  build-linux - Build for Linux only"
	@echo "  package     - Create distribution packages for all platforms"
	@echo "  clean       - Remove build artifacts"
	@echo "  deps        - Install dependencies"
	@echo "  run         - Build and run the application"
	@echo "  help        - Show this help message"
