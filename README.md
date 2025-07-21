# GMM MKV Subtitles Extract

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
- Real-time progress indication with elapsed time
- Detailed logging for troubleshooting
- Cross-platform support (macOS, Windows, Linux)

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

```sh
./gmmmkvsubsextract -x /path/to/yourfile.mkv
```

### GUI Version
1. Click "Select MKV File" to choose your MKV file
2. Click "Select Output Directory" to choose where to save the extracted subtitles
3. Click "Load Tracks" to see available subtitle tracks
4. Select the subtitle tracks you want to extract/convert
5. Click "Start Extract" to begin the process
6. Monitor the progress in the application window

## Troubleshooting

- Ensure Deno, mkvmerge, and mkvextract are in your PATH
- Check the conversion logs in the output directory
- For permission issues, try running the application with administrator privileges

```sh
./gmmmkvsubsextract -x example.mkv
```

This command will extract all subtitle tracks from `example.mkv` and save them with appropriate file names based on track properties.

## Converting Subtitles

If you need to convert the extracted subtitle files to other formats such as `.srt`, you can use online tools like [Subtitle Tools](https://subtitletools.com/). This website allows you to upload your subtitle files and convert them to various formats easily.

## License

This project is licensed under the MIT License. See the `LICENSE.md` file for details.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request on GitHub.

## Acknowledgements

- [MKVToolNix](https://mkvtoolnix.download/)
- [gocmd](https://github.com/devfacet/gocmd)
- [logrus](https://github.com/sirupsen/logrus)
