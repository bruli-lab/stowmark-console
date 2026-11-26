# Stowmark Console

[![CI](https://github.com/bruli-lab/stowmark-console/actions/workflows/release.yml/badge.svg)](https://github.com/bruli-lab/stowmark-console/actions/workflows/release.yml)
[![Release](https://img.shields.io/github/v/release/bruli-lab/stowmark-console)](https://github.com/bruli-lab/stowmark-console/releases)
[![License](https://img.shields.io/github/license/bruli-lab/stowmark-console)](LICENSE)
[![Go](https://img.shields.io/github/go-mod/go-version/bruli-lab/stowmark-console)](go.mod)

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

While the console is running, Stowmark log output is written to a log file (see `STOWMARK_CONSOLE_LOG_FILE`) so it does not corrupt the screen.

### Configuration

The console is configured with environment variables:

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `STOWMARK_REPOSITORY` | yes | | Path (or URL) of the Stowmark repository |
| `STOWMARK_PRIVATE_KEY` | no | | Private key used to create, verify and restore snapshots of an encrypted repository |
| `STOWMARK_PUBLIC_KEY` | no | | Public key used to encrypt a repository when it is initialized |
| `STOWMARK_COMPRESSION` | no | `none` | Compression of a new repository: `none`, `zstd`, `lz4`, `xz` or `gzip` |
| `STOWMARK_COMPRESSION_LEVEL` | no | | Compression level of a new repository |
| `STOWMARK_CONSOLE_LOG_FILE` | no | `~/.cache/stowmark-console/console.log` | File that receives Stowmark log output |

The initialization options only apply when the console creates a new repository, and they are shown in the confirmation dialog. Variables read by Stowmark itself, such as `STOWMARK_LOG_LEVEL`, are honored as well.

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

The project uses [Task](https://taskfile.dev/). Run `task` to list every task.

| Task | Description |
| --- | --- |
| `task run` | Start the console against the local `sandbox/repository` |
| `task build` | Build the binary into `bin/` |
| `task test` | Run the tests with race detector and coverage, reported with tparse |
| `task mocks` | Regenerate the interface mocks with moq |
| `task fmt` | Format the code with gofumpt |
| `task lint` | Run golangci-lint |
| `task security` | Run govulncheck |
| `task check` | Run fmt, fix, security, lint and test in order |

Build a local snapshot release with [GoReleaser](https://goreleaser.com/):

```bash
goreleaser release --snapshot --clean
```

## Related project

- [Stowmark](https://github.com/bruli-lab/stowmark) — immutable snapshot backup engine used by this console.

## License

Licensed under the [Apache License 2.0](LICENSE).