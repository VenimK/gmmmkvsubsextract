#!/bin/bash
# Build script for Subtitle Forge application

# Parse command line arguments
BUILD_ALL=false
if [ "$1" == "--all" ]; then
    BUILD_ALL=true
fi

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

# Windows build removed as requested
echo "Windows build disabled"

# Only build for Linux if --all flag is provided
if [ "$BUILD_ALL" = true ]; then
    # Build for Linux (requires CGO)
    echo "Building for Linux..."
    GOOS=linux GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-linux-musl-gcc go build -o build/subtitle-forge-linux
    if [ $? -eq 0 ]; then
        echo "✅ Linux build successful"
    else
        echo "❌ Linux build failed (you may need to install musl-cross for cross-compilation)"
        echo "   Install with: brew install FiloSottile/musl-cross/musl-cross"
    fi
else
    echo "Skipping Linux build (use --all flag to build for all platforms)"
fi

echo "Creating distribution packages..."

# Check if README.md exists in the project root
README_PATH="../README.md"
if [ ! -f "$README_PATH" ]; then
    README_PATH="README.md"
    if [ ! -f "$README_PATH" ]; then
        echo "Warning: README.md not found"
        README_PATH=""
    fi
fi

# Create macOS package
if [ -f "build/subtitle-forge-mac" ]; then
    echo "Creating macOS package..."
    mkdir -p build/macos
    cp build/subtitle-forge-mac build/macos/
    
    # Copy scripts if they exist
    if [ -d "../scripts" ]; then
        cp -r ../scripts build/macos/
    elif [ -d "scripts" ]; then
        cp -r scripts build/macos/
    else
        echo "Warning: scripts directory not found"
    fi
    
    # Copy README and LICENSE if they exist
    if [ -n "$README_PATH" ]; then
        cp "$README_PATH" build/macos/
    fi
    
    if [ -f "../LICENSE" ]; then
        cp ../LICENSE build/macos/
    elif [ -f "LICENSE" ]; then
        cp LICENSE build/macos/
    else
        echo "Warning: LICENSE file not found"
    fi
    
    tar -czf build/subtitle-forge-macos.tar.gz -C build macos
    echo "✅ macOS package created"
fi

# Windows package creation removed as requested
echo "Windows packaging disabled"

# Create Linux package only if Linux build exists
if [ -f "build/subtitle-forge-linux" ]; then
    echo "Creating Linux package..."
    mkdir -p build/linux
    cp build/subtitle-forge-linux build/linux/
    
    # Copy scripts if they exist
    if [ -d "../scripts" ]; then
        cp -r ../scripts build/linux/
    elif [ -d "scripts" ]; then
        cp -r scripts build/linux/
    else
        echo "Warning: scripts directory not found"
    fi
    
    # Copy README and LICENSE if they exist
    if [ -n "$README_PATH" ]; then
        cp "$README_PATH" build/linux/
    fi
    
    if [ -f "../LICENSE" ]; then
        cp ../LICENSE build/linux/
    elif [ -f "LICENSE" ]; then
        cp LICENSE build/linux/
    else
        echo "Warning: LICENSE file not found"
    fi
    
    tar -czf build/subtitle-forge-linux.tar.gz -C build linux
    echo "✅ Linux package created"
fi

echo "Build process completed. Check the 'build' directory for binaries and packages."
