# Stowmark Console

[![CI](https://github.com/bruli-lab/stowmark-console/actions/workflows/release.yml/badge.svg)](https://github.com/bruli-lab/stowmark-console/actions/workflows/release.yml)
[![Release](https://img.shields.io/github/v/release/bruli/stowmark-console)](https://github.com/bruli-lab/stowmark-console/releases)
[![License](https://img.shields.io/github/license/bruli/stowmark-console)](LICENSE)
[![Go](https://img.shields.io/github/go-mod/go-version/bruli/stowmark-console)](go.mod)

A terminal user interface for browsing and managing local [Stowmark](https://github.com/bruli-lab/stowmark) snapshot repositories.

Stowmark Console provides a compact, keyboard-driven interface for inspecting repository contents, previewing text files, creating snapshots, verifying their integrity, and restoring them.

## Features

- Browse a Stowmark repository without leaving the terminal
- Offer to initialize the repository when it does not exist
- Preview text files with line numbers and scrolling
- Create snapshots from a selected source directory
- Verify the integrity of a snapshot
- Restore a snapshot to a chosen destination after an explicit confirmation
- Navigate using arrow keys or Vim-style key bindings

> [!NOTE]
> The current implementation operates on local repositories.

## Requirements

- Linux
- Go 1.27.4 or newer when building from source
- A terminal with color support

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

If the path does not contain a Stowmark repository yet, the console asks for confirmation before initializing one there. Declining quits without touching the disk.

While the console is running, Stowmark log output is written to `~/.cache/stowmark-console/console.log` (or `$XDG_CACHE_HOME/stowmark-console/console.log`) so it does not corrupt the screen.

## Keyboard controls

### Repository browser

| Key | Action |
| --- | --- |
| `↑` / `k` | Select the previous entry |
| `↓` / `j` | Select the next entry |
| `Enter` | Open a directory or preview a file |
| `Backspace` / `Esc` / `←` / `h` | Go to the parent directory |
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

Creating a snapshot opens a small form for the source directory.

Restoring asks for a destination directory first. It defaults to `~/stowmark-restore/<snapshot-id>`, never to the original source. The confirmation shows the destination and warns when it already contains files, which would be overwritten. Only `y` confirms the restore.

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

- [Stowmark](https://github.com/bruli-lab/stowmark) — immutable snapshot backup engine used by this console.

## License

Licensed under the [Apache License 2.0](LICENSE).