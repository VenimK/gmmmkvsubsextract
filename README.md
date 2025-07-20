# GMM MKV Subtitles Extract

A GUI application for extracting and converting PGS subtitles from MKV files to SRT format.

## Features

- Extract subtitle tracks from MKV files
- Convert PGS/SUP subtitles to SRT format using OCR
- Real-time progress indication with elapsed time
- Detailed logging for troubleshooting
- Cross-platform support (macOS, Windows, Linux)

## Requirements

- [Deno](https://deno.land/) (for running the PGS to SRT conversion script)
- [mkvmerge](https://mkvtoolnix.download/) (part of MKVToolNix)

## Installation

### macOS

1. Extract the `gmmmkvsubsextract-macos.tar.gz` archive
2. Install Deno: `brew install deno`
3. Install MKVToolNix: `brew install mkvtoolnix`
4. Run the application: `./gmmmkvsubsextract-mac`

### Windows

1. Extract the `gmmmkvsubsextract-windows.zip` archive
2. Install Deno: [Deno Installation](https://deno.land/#installation)
3. Install MKVToolNix: [MKVToolNix Download](https://mkvtoolnix.download/downloads.html)
4. Add both to your PATH environment variable
5. Run the application by double-clicking `gmmmkvsubsextract.exe`

### Linux

1. Extract the `gmmmkvsubsextract-linux.tar.gz` archive
2. Install Deno: `curl -fsSL https://deno.land/x/install/install.sh | sh`
3. Install MKVToolNix: Use your distribution's package manager (e.g., `apt install mkvtoolnix`)
4. Run the application: `./gmmmkvsubsextract-linux`

## Usage

1. Click "Select MKV File" to choose your MKV file
2. Click "Select Output Directory" to choose where to save the extracted subtitles
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

- Ensure Deno and mkvmerge are in your PATH
- Check the conversion logs in the output directory
- For permission issues, try running the application with administrator privileges

## License

[MIT License](LICENSE)
