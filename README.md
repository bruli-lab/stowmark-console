# Stowmark Console

[![CI](https://github.com/bruli-lab/stowmark-console/actions/workflows/release.yml/badge.svg)](https://github.com/bruli-lab/stowmark-console/actions/workflows/release.yml)
[![Release](https://img.shields.io/github/v/release/bruli/stowmark-console)](https://github.com/bruli-lab/stowmark-console/releases)
[![License](https://img.shields.io/github/license/bruli/stowmark-console)](LICENSE)
[![Go](https://img.shields.io/github/go-mod/go-version/bruli/stowmark-console)](go.mod)

A terminal user interface for browsing and managing local [Stowmark](https://github.com/bruli-lab/stowmark) snapshot repositories.

Stowmark Console provides a compact, keyboard-driven interface for inspecting repository contents, previewing and editing text files, creating snapshots, verifying their integrity, and restoring them.

## Features

- Browse a Stowmark repository without leaving the terminal
- Automatically initialize the repository when it does not exist
- Preview text files with line numbers and scrolling
- Open text files in your preferred terminal editor
- Create snapshots from a selected source directory
- Verify the integrity of a snapshot
- Restore a snapshot after an explicit confirmation
- Navigate using arrow keys or Vim-style key bindings

> [!NOTE]
> The current implementation operates on local repositories.

## Requirements

- Linux
- Go 1.26.5 or newer when building from source
- A terminal with color support
- An optional terminal editor for the edit action

The editor is selected in this order: `VISUAL`, `EDITOR`, `nvim`, `vim`, `nano`, then `vi`.

## Installation

### Debian package

Download the package for your architecture from the [latest release](https://github.com/bruli-lab/stowmark-console/releases/latest), then install it with:

```bash
sudo apt install ./stowmark_console_*_linux_amd64.deb
```

Use the `arm64` package instead when appropriate.

### Build from source

```bash
git clone https://github.com/bruli-lab/stowmark-console.git
cd stowmark-console
go build -o stowmark_console ./cmd/console
```

Optionally install the binary in a directory on your `PATH`:

```bash
sudo install -m 0755 stowmark_console /usr/local/bin/stowmark_console
```

## Usage

Set `STOWMARK_REPOSITORY` to the directory containing the Stowmark repository and start the console:

```bash
export STOWMARK_REPOSITORY="$HOME/backups/stowmark"
stowmark_console
```

You can also set the variable for a single invocation:

```bash
STOWMARK_REPOSITORY="$HOME/backups/stowmark" stowmark_console
```

If the path does not contain a Stowmark repository yet, the console initializes one there automatically.

## Keyboard controls

### Repository browser

| Key | Action |
| --- | --- |
| `â†‘` / `k` | Select the previous entry |
| `â†“` / `j` | Select the next entry |
| `Enter` | Open a directory or preview a file |
| `Backspace` / `Esc` / `â†` / `h` | Go to the parent directory |
| `e` | Edit the selected text file |
| `r` | Refresh the current directory |
| `g` / `Home` | Select the first entry |
| `G` / `End` | Select the last entry |
| `:q` / `Ctrl+C` | Exit |

### Snapshot directory

The following actions are available while browsing the repository's `snapshots` directory:

| Key | Action |
| --- | --- |
| `c` | Create a snapshot |
| `v` | Verify the selected snapshot |
| `x` | Restore the selected snapshot |

Creating a snapshot opens a small form for the source directory. Restoring requires confirmation before any files are written.

## Development

Run the test suite:

```bash
go test ./...
```

Format the code:

```bash
gofmt -w .
```

Build a local snapshot release with [GoReleaser](https://goreleaser.com/):

```bash
goreleaser release --snapshot --clean
```

## Related project

- [Stowmark](https://github.com/bruli-lab/stowmark) â€” immutable snapshot backup engine used by this console.

## License

Licensed under the [Apache License 2.0](LICENSE).