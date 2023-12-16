# Halsey

Halsey is a command-line tool for downloading HLS streams from a given URL.

## Installation

To install Halsey, use the following `go install` command:

```bash
go install github.com/trugamr/halsey
```

Make sure your Go bin directory is in your system's PATH to execute halsey from any location.

## Usage

```bash
halsey download [url] [flags]
```

### Flags

- `--directory` or `-d`: The directory to save the downloaded files to. Defaults to "playlist".


## Examples

Download an HLS stream from a given URL:

```bash
halsey download https://example.com/playlist.m3u8
```

Download an HLS stream from a given URL and save it to a specific directory:

```bash
halsey download https://example.com/playlist.m3u8 --directory /path/to/directory
```

## Contributing

If you find any issues or have suggestions for improvements, feel free to open an issue or create a pull request.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE.md) file for more information.
