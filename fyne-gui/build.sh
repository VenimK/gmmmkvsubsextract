#!/bin/bash
# Build script for Subtitle Forge application

# Create build directory if it doesn't exist
mkdir -p build

# Build for macOS
echo "Building for macOS..."
go build -o build/subtitle-forge-mac
if [ $? -eq 0 ]; then
    echo "✅ macOS build successful"
else
    echo "❌ macOS build failed"
    exit 1
fi

# Build for Windows (requires CGO)
echo "Building for Windows..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -o build/subtitle-forge.exe
if [ $? -eq 0 ]; then
    echo "✅ Windows build successful"
else
    echo "❌ Windows build failed (you may need to install MinGW for cross-compilation)"
    echo "   Install with: brew install mingw-w64"
fi

# Build for Linux (requires CGO)
echo "Building for Linux..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-linux-musl-gcc go build -o build/subtitle-forge-linux
if [ $? -eq 0 ]; then
    echo "✅ Linux build successful"
else
    echo "❌ Linux build failed (you may need to install musl-cross for cross-compilation)"
    echo "   Install with: brew install FiloSottile/musl-cross/musl-cross"
fi

echo "Creating distribution packages..."

# Create macOS package
mkdir -p build/macos
cp build/subtitle-forge-mac build/macos/
cp -r ../scripts build/macos/
cp README.md build/macos/ 2>/dev/null || echo "No README.md found"
cp LICENSE build/macos/ 2>/dev/null || echo "No LICENSE found"
tar -czf build/subtitle-forge-macos.tar.gz -C build macos

# Create Windows package
mkdir -p build/windows
cp build/subtitle-forge.exe build/windows/
cp -r ../scripts build/windows/
cp README.md build/windows/ 2>/dev/null || echo "No README.md found"
cp LICENSE build/windows/ 2>/dev/null || echo "No LICENSE found"
zip -r build/subtitle-forge-windows.zip build/windows

# Create Linux package
mkdir -p build/linux
cp build/subtitle-forge-linux build/linux/
cp -r ../scripts build/linux/
cp README.md build/linux/ 2>/dev/null || echo "No README.md found"
cp LICENSE build/linux/ 2>/dev/null || echo "No LICENSE found"
tar -czf build/subtitle-forge-linux.tar.gz -C build linux

echo "Build process completed. Check the 'build' directory for binaries and packages."
