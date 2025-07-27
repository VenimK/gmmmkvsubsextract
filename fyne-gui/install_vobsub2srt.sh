#!/bin/bash

# Script to install VobSub2SRT from leonard-slass fork
echo "Installing VobSub2SRT from leonard-slass fork..."

# Create a temporary directory for the build
TEMP_DIR=$(mktemp -d)
echo "Using temporary directory: $TEMP_DIR"

# Navigate to the temporary directory
cd "$TEMP_DIR" || exit 1

# Clone the repository
echo "Cloning the repository..."
git clone https://github.com/leonard-slass/VobSub2SRT.git
if [ $? -ne 0 ]; then
    echo "Failed to clone repository."
    exit 1
fi

# Navigate to the cloned repository
cd VobSub2SRT || exit 1

# Create build directory
echo "Creating build directory..."
mkdir -p build
cd build || exit 1

# Run cmake
echo "Running cmake..."
cmake -DCMAKE_POLICY_VERSION_MINIMUM=3.5 ..
if [ $? -ne 0 ]; then
    echo "Failed to configure with cmake."
    exit 1
fi

# Build and install
echo "Building and installing..."
make
if [ $? -ne 0 ]; then
    echo "Failed to build."
    exit 1
fi

# Install (may require sudo)
echo "Installing (may require sudo)..."
sudo make install
if [ $? -ne 0 ]; then
    echo "Failed to install. Try running this script with sudo."
    exit 1
fi

# Verify installation
echo "Verifying installation..."
if command -v vobsub2srt &> /dev/null; then
    echo "VobSub2SRT installed successfully!"
    vobsub2srt --version
else
    echo "VobSub2SRT installation could not be verified."
    exit 1
fi

# Clean up
echo "Cleaning up temporary files..."
cd
rm -rf "$TEMP_DIR"

echo "Installation complete!"
exit 0
