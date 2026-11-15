# Kova

A minimal text editor for macOS.

## Command line

Install the tool from **Kova → Install Command Line Tool…**, then:

```sh
kova notes.txt  # open a file
kova .          # open a folder
```

See [docs/cli.md](docs/cli.md) for the full argument rules, the install
locations, and current limitations.

## Settings

Kova reads editor settings from `settings.json` in the platform config
directory and creates it with defaults on first launch. See
[docs/settings.md](docs/settings.md) for the file locations, supported fields,
and validation rules.

## Development

Building from source, running with hot reload, and releasing: see
[docs/development.md](docs/development.md).

## License

[MIT](LICENSE) © [Nilambar Sharma](https://www.nilambar.net)
