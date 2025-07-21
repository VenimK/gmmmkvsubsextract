# Subtitle Forge

A tool for extracting and converting subtitles from MKV files, available in both command-line (CLI) and graphical user interface (GUI) versions.

## Overview

This project provides two applications:
1. **CLI Version** - Command-line tool for extracting subtitles from MKV files
2. **GUI Version** - Fyne-based graphical application with enhanced features including PGS to SRT conversion

## Features

### CLI Version
- Extract subtitles from MKV files
- Support for multiple subtitle formats including SRT, ASS, and SUP
- Automatic naming of extracted subtitle files based on track properties

### GUI Version
- User-friendly graphical interface
- Extract subtitle tracks from MKV files
- Convert PGS/SUP subtitles to SRT format using OCR
- Enhanced progress reporting:
  - Detailed progress bar showing percentage complete
  - Real-time frame processing status
  - Elapsed time tracking
  - Estimated time remaining calculation
- Detailed logging for troubleshooting
- Cross-platform support (macOS, Windows, Linux)
- Automatic dependency checking at startup
- Drag-and-drop support for MKV files
- Automatic output directory setting (defaults to MKV file location)

## Requirements

### CLI Version
- Go 1.16 or later
- `mkvmerge` and `mkvextract` tools from the MKVToolNix package
- `gocmd` library

### GUI Version
- Go 1.18 or later
- Fyne v2.6.1 or later
- [Deno](https://deno.land/) (for running the PGS to SRT conversion script)
- [mkvmerge and mkvextract](https://mkvtoolnix.download/) (part of MKVToolNix)
- [Tesseract OCR](https://github.com/tesseract-ocr/tesseract) (used by the PGS-to-SRT conversion script)
- PGS-to-SRT conversion script

## Installation

### CLI Version
1. Install Go from [golang.org](https://golang.org/dl/)
2. Install MKVToolNix from [mkvtoolnix.download](https://mkvtoolnix.download/)
3. Clone the repository and navigate to the project directory:
    ```sh
    git clone https://github.com/rhaseven7h/gmmmkvsubsextract.git
    cd gmmmkvsubsextract
    ```
4. Build the CLI version:
    ```sh
    go build -o gmmmkvsubsextract
    ```

### GUI Version

#### macOS
1. Extract the `gmmmkvsubsextract-macos.tar.gz` archive
2. Install Deno: `brew install deno`
3. Install MKVToolNix: `brew install mkvtoolnix`
4. Run the application: `./gmmmkvsubsextract-mac`

#### Windows
1. Extract the `gmmmkvsubsextract-windows.zip` archive
2. Install Deno: [Deno Installation](https://deno.land/#installation)
3. Install MKVToolNix: [MKVToolNix Download](https://mkvtoolnix.download/downloads.html)
4. Add both to your PATH environment variable
5. Run the application by double-clicking `gmmmkvsubsextract.exe`

#### Linux
1. Extract the `gmmmkvsubsextract-linux.tar.gz` archive
2. Install Deno: `curl -fsSL https://deno.land/x/install/install.sh | sh`
3. Install MKVToolNix: Use your distribution's package manager (e.g., `apt install mkvtoolnix`)
4. Run the application: `./gmmmkvsubsextract-linux`

#### Building from Source
1. Clone the repository
2. Navigate to the `fyne-gui` directory
3. Install Fyne dependencies: [Fyne Getting Started](https://developer.fyne.io/started/)
4. Run the build script: `./build.sh`

## Usage

### CLI Version
To extract subtitles from an MKV file, use the `-x` or `--extract` flag followed by the path to the MKV file:

## PGS to SRT Conversion Process

The application includes a powerful feature to convert PGS/SUP subtitle files (image-based subtitles) to SRT format (text-based subtitles) using Optical Character Recognition (OCR). This process involves several steps:

### How It Works

1. **Extraction**: First, the PGS subtitles are extracted from the MKV file using `mkvextract` as .sup files

2. **OCR Processing**: The extracted .sup files are then processed using a Deno-based script that:
   - Decodes the PGS/SUP format to extract individual subtitle frames
   - Uses Tesseract OCR to convert the subtitle images to text
   - Preserves timing information from the original subtitles
   - Formats the output as a standard SRT file

3. **Real-time Feedback**: During conversion, the application provides:
   - Progress updates
   - Elapsed time tracking
   - Detailed logs of the conversion process

### Requirements for OCR

- **Deno Runtime**: Required to execute the conversion script
- **Tesseract OCR**: The underlying OCR engine used for text recognition
- **Tessdata Files**: Language training data for Tesseract (English data included by default)

### Performance Considerations

- OCR conversion is CPU-intensive and may take significant time for longer subtitle tracks
- The quality of the OCR results depends on several factors:
  - Resolution and clarity of the original PGS subtitles
  - Font style used in the original subtitles
  - Language of the subtitles (English works best with the default configuration)

### Troubleshooting OCR Conversion

- If conversion fails, check that Deno is properly installed and in your PATH
- Verify that the Tesseract language data files are available
- For poor OCR quality, you may need to adjust the conversion parameters in the script
- The application creates detailed logs that can help diagnose conversion issues

./gmmmkvsubsextract -x /path/to/yourfile.mkv

### GUI Version
1. Load an MKV file using one of these methods:
   - Click "Select MKV File" to choose your MKV file using the file dialog
   - Or simply drag and drop an MKV file onto the application window
2. The output directory is automatically set to the same location as your MKV file
   - You can change it by clicking "Change Output Directory" if needed
3. Click "Load Tracks" to see available subtitle tracks
4. Select the subtitle tracks you want to extract/convert
5. Click "Start Extract" to begin the process
6. Monitor the progress in the application window

## Building from Source

### Prerequisites

- Go 1.18 or later
- Fyne dependencies: [Fyne Getting Started](https://developer.fyne.io/started/)

### Build Steps

1. Clone the repository
2. Navigate to the `fyne-gui` directory
3. Run the build script: `./build.sh`

For cross-compilation, you may need additional tools:
- For Windows builds on macOS: `brew install mingw-w64`
- For Linux builds on macOS: `brew install FiloSottile/musl-cross/musl-cross`

## Troubleshooting

- The application automatically checks for required dependencies at startup
- Missing dependencies will be clearly indicated in the application window
- Ensure Deno, mkvmerge, and mkvextract are in your PATH
- Check the conversion logs in the output directory
- For permission issues, try running the application with administrator privileges

## License

[MIT License](LICENSE)
