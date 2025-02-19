# GMM MKV Subtitles Extract

`gmmmkvsubsextract` is a command-line tool written in Go for extracting subtitles from MKV files.

## Features

- Extract subtitles from MKV files.
- Supports multiple subtitle formats including SRT, ASS, and SUP.
- Automatically names the extracted subtitle files based on track properties.

## Requirements

- Go 1.16 or later
- `mkvmerge` and `mkvextract` tools from the MKVToolNix package
- `gocmd` library

## Installation

1. Install Go from [golang.org](https://golang.org/dl/).
2. Install MKVToolNix from [mkvtoolnix.download](https://mkvtoolnix.download/).
3. Clone the repository and navigate to the project directory:
    ```sh
    git clone https://github.com/rhaseven7h/gmmmkvsubsextract.git
    cd gmmmkvsubsextract
    ```
4. Build the project:
    ```sh
    go build -o gmmmkvsubsextract
    ```

## Usage

To extract subtitles from an MKV file, use the `-x` or `--extract` flag followed by the path to the MKV file:

```sh
./gmmmkvsubsextract -x /path/to/yourfile.mkv
```

## Example

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
