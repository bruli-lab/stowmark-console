# Stowmark Console

[![CI](https://github.com/bruli-lab/stowmark-console/actions/workflows/release.yml/badge.svg)](https://github.com/bruli-lab/stowmark-console/actions/workflows/release.yml)
[![Release](https://img.shields.io/github/v/release/bruli-lab/stowmark-console)](https://github.com/bruli-lab/stowmark-console/releases)
[![License](https://img.shields.io/github/license/bruli-lab/stowmark-console)](LICENSE)
[![Go](https://img.shields.io/github/go-mod/go-version/bruli-lab/stowmark-console)](go.mod)

A terminal user interface for browsing and managing [Stowmark](https://github.com/bruli-lab/stowmark) snapshot repositories.

Stowmark Console provides a compact, keyboard-driven interface for listing snapshots, browsing the files they contain, previewing text files, creating snapshots, verifying their integrity, and restoring them.

## Features

- List the snapshots of a Stowmark repository, newest first
- Browse the files of a snapshot by directory, straight from its manifest
- Offer to initialize the repository when it does not exist
- Preview text files stored in a snapshot with line numbers and scrolling
- Create snapshots from a selected source directory
- Verify the integrity of a snapshot
- Restore a snapshot to a chosen destination after an explicit confirmation
- Work with encrypted repositories, generate key pairs and rotate keys (`:keygen`, `:rewrap`, `:rekey`)
- Cancel a running operation with `Esc`
- Navigate using arrow keys or Vim-style key bindings

Every repository type supported by Stowmark works: local paths, SSH/SFTP, S3, SMB, WebDAV and Google Cloud Storage.

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

The version shown on the start screen is taken from the git tag stamped by `go build` (for example `v1.0.3` on a tag, or a pseudo-version such as `v1.0.3-0.20261126222033-0a91d20a48c6+dirty` between tags). Release builds made by GoReleaser inject it with `-X main.version`. Builds without version control information, such as `go run`, show `dev`.

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

To work with several repositories, list them in a repositories file (see [Several repositories](#several-repositories)) and switch between them with `:repo`.

If the path does not contain a Stowmark repository yet, the console asks for confirmation before initializing one there. Declining quits without touching the disk, or goes back to the repositories list when several are configured.

On start, the console shows the Stowmark logo and its version for a moment while the repositories load; press any key to skip it.

While the console is running, Stowmark log output is written to a log file (see `STOWMARK_CONSOLE_LOG_FILE`) so it does not corrupt the screen.

### Remote repositories

`STOWMARK_REPOSITORY` also accepts the remote URLs understood by Stowmark, and the credentials are read by Stowmark from its own environment variables:

| Backend | Repository URL | Credentials |
| --- | --- | --- |
| SSH/SFTP | `ssh://<user>@<host>[:<port>]/<absolute-path>` | `STOWMARK_SSH_PRIVATE_KEY`, `STOWMARK_SSH_KNOWN_HOSTS` or the SSH agent |
| S3 | `s3://<bucket>/<path>` | `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION` (and `STOWMARK_S3_ENDPOINT`, `STOWMARK_S3_PATH_STYLE` for S3 compatible services) |
| SMB | `smb://<user>@<host>[:<port>]/<share>[/<path>]` | `STOWMARK_SMB_PASSWORD` |
| WebDAV | `webdav://` or `webdavs://<host>[:<port>]/<path>` | `STOWMARK_WEBDAV_USERNAME`, `STOWMARK_WEBDAV_PASSWORD` |
| GCS | `gcs://<bucket>/<path>` | `GOOGLE_APPLICATION_CREDENTIALS` (and `STOWMARK_GCS_ENDPOINT` for emulators) |

See the [Stowmark documentation](https://github.com/bruli-lab/stowmark) for every option. Snapshot sources and restore destinations are always local directories.

### Several repositories

The console reads a YAML file with the repositories to choose from, `~/.config/stowmark-console/repositories.yaml` by default (`STOWMARK_CONSOLE_CONFIG` changes the path). Each repository has its own initialization options and credentials:

```yaml
repositories:
  - name: laptop
    url: ~/backups/stowmark
    compression: zstd
    compression_level: 3
  - name: nas
    url: smb://pablo@nas:445/backups/stowmark
    env:
      STOWMARK_SMB_PASSWORD: secret
  - name: cloud
    url: s3://my-bucket/stowmark
    format_version: 3
    private_key: ~/.config/stowmark/keys/1a2b3c4d-private.pem
    public_key: ~/.config/stowmark/keys/1a2b3c4d-public.pem
    env:
      AWS_ACCESS_KEY_ID: AKIA...
      AWS_SECRET_ACCESS_KEY: ...
      AWS_REGION: eu-west-1
```

| Field | Required | Description |
| --- | --- | --- |
| `name` | yes | Unique name shown in the repositories list |
| `url` | yes | Path or URL of the repository (`~` is expanded for local paths) |
| `private_key`, `public_key` | no | Same as `STOWMARK_PRIVATE_KEY` and `STOWMARK_PUBLIC_KEY` |
| `compression`, `compression_level` | no | Same as `STOWMARK_COMPRESSION` and `STOWMARK_COMPRESSION_LEVEL` |
| `format_version` | no | Same as `STOWMARK_FORMAT_VERSION` |
| `env` | no | Environment variables set while the repository is open, such as the credentials of a remote backend |

When `STOWMARK_REPOSITORY` is set, it is added first as the `default` repository with the options of the environment variables. With a single repository, the console opens it on start. With several, it starts on the repositories list so you can choose which one to open. `:repo` lists them all again, marks the current one and opens the selected one: its `env` variables replace those of the previous repository. Keep the file readable only by you (`chmod 600`) when it holds credentials.

### Configuration

The console is configured with environment variables. At least one repository must be configured, either with `STOWMARK_REPOSITORY` or in the repositories file:

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `STOWMARK_REPOSITORY` | no | | Path (or URL) of the `default` Stowmark repository |
| `STOWMARK_PRIVATE_KEY` | no | | Private key used to create, verify and restore snapshots of an encrypted repository |
| `STOWMARK_PUBLIC_KEY` | no | | Public key used to encrypt a repository when it is initialized |
| `STOWMARK_COMPRESSION` | no | `none` | Compression of a new repository: `none`, `zstd`, `lz4`, `xz` or `gzip` |
| `STOWMARK_COMPRESSION_LEVEL` | no | | Compression level of a new repository |
| `STOWMARK_FORMAT_VERSION` | no | `2` | Format version of a new repository: `1`, `2` or `3` |
| `STOWMARK_CONSOLE_LOG_FILE` | no | `~/.cache/stowmark-console/console.log` | File that receives Stowmark log output |
| `STOWMARK_CONSOLE_CONFIG` | no | `~/.config/stowmark-console/repositories.yaml` | Repositories file |

The initialization options only apply when the console creates a new repository, and they are shown in the confirmation dialog. Variables read by Stowmark itself, such as `STOWMARK_LOG_LEVEL`, are honored as well.

### Encrypted repositories

A repository is encrypted when it is initialized with a public key. Reading it, creating snapshots and restoring them need the matching private key, so set both keys for the repository. Format version 3 is recommended for encrypted repositories: it also encrypts the manifests.

- The header and the repositories list show a 🔒 for repositories configured with a key, and the sidebar shows the private key and its fingerprint (the same `SHA256:` fingerprint Stowmark stores in the repository).
- A format version 3 repository opened without its private key is reported as such, instead of offering to initialize it. A wrong or unreadable private key gives an error that names the repository and the setting to fix.
- The console refuses to create snapshots in a repository configured with a public key but no private key: Stowmark would store the new files without encryption in a format version 2 repository.
- `:keygen` generates a key pair in a folder and shows the `public_key` / `private_key` lines to add to the repositories file.
- `:rewrap` wraps the repository key with a new public key (asymmetric rotation, fast). `:rekey` encrypts every object again with a new key generation (symmetric rotation, slow on big repositories; `Esc` cancels it and Stowmark discards the incomplete generation), and asks you to type the repository name to confirm. Both take an exclusive lock on the repository.
- The console never writes the repositories file: after a key change it shows the lines to update. Restart the console once they are updated.

## Keyboard controls

### Snapshots

| Key | Action |
| --- | --- |
| `↑` / `k`, `↓` / `j` | Select the previous or next snapshot |
| `PgUp` / `b`, `PgDown` / `f`, `u` / `d` | Move one page or half a page |
| `g` / `Home`, `G` / `End` | Select the first or last snapshot |
| `Enter` / `→` / `l` | Browse the files of the selected snapshot |
| `c` | Create a snapshot |
| `v` | Verify the selected snapshot |
| `x` | Restore the selected snapshot |
| `r` | Refresh the list |
| `:` | Open the command line |
| `Ctrl+C` | Exit |

### Files of a snapshot

| Key | Action |
| --- | --- |
| `↑` / `k`, `↓` / `j` | Select the previous or next entry |
| `Enter` / `→` / `l` | Open a directory or preview a file |
| `o` | Open the selected file with the system application |
| `Esc` / `Backspace` / `←` / `h` | Go to the parent directory, or back to the snapshots |
| `v` / `x` | Verify or restore the snapshot being browsed |
| `c` | Create a snapshot |
| `r` | Reload the snapshot |

### File preview

| Key | Action |
| --- | --- |
| `↑` / `k`, `↓` / `j` | Scroll one line |
| `←` / `h`, `→` / `l` | Scroll long lines horizontally |
| `PgUp` / `b`, `PgDown` / `f` / `Space` | Scroll one page |
| `g` / `Home`, `G` / `End` | Go to the top or the bottom |
| `o` | Open the file with the system application |
| `Esc` / `q` | Close the preview |

The preview adapts to the content of the file:

- **Text** is shown with line numbers. At most the first 1 MiB is read.
- **Images** (PNG, JPEG and GIF up to 32 MiB) are drawn with half-block characters scaled to the panel, so they need a terminal with true color support.
- **Other binary files** are shown in hexadecimal, like `xxd`, with the offset, the bytes and their printable characters. At most the first 1 MiB is read.

`o` extracts the whole file from the snapshot to a temporary directory and opens it with `xdg-open` (`open` on macOS), so it needs a graphical session. The temporary directory is deleted when the console exits.

### Command line

Press `:` in any view, type a command and press `Enter`. While you type, the footer lists the matching commands and `Tab` completes the suggested one. `Esc` closes the command line. The sidebar also lists the main commands; `:rewrap` and `:rekey` only appear when the open repository is encrypted.

| Command | Action |
| --- | --- |
| `:q`, `:quit` | Exit |
| `:snap`, `:snapshots`, `:s` | Go back to the snapshots list |
| `:repo`, `:repos`, `:repositories` | List the configured repositories and switch to another one |
| `:keygen` | Generate a key pair for encrypted repositories |
| `:rewrap` | Wrap the key of the open repository with a new public key |
| `:rekey` | Encrypt every object of the open repository with a new key |

### Creating and restoring

Creating a snapshot opens a form for the origin directory, prefilled with the source of the selected snapshot. While you type, it suggests the subdirectories of the path and the origins of earlier snapshots: `Tab` accepts the highlighted suggestion and `↑`/`↓` move between them. The form tells you whether the path is an existing directory and only accepts directories.

Restoring asks for a destination directory first. It defaults to `~/stowmark-restore/<snapshot-id>`, never to the original source. The confirmation shows the destination and warns when it already contains files, which would be overwritten. Only `y` confirms the restore.

While an operation is running, `Esc` cancels it.

## Architecture

The project follows a hexagonal architecture:

- `internal/domain`: entities, aggregates, value objects, events and ports (`repository`, `snapshot`, `encryption`). Events and aggregates are built on [go-core](https://github.com/bruli-lab/go-core) `event`.
- `internal/app`: one query or command handler per use case on go-core `cqs`, subscribed to a command bus and a query bus (`app.NewBuses`). Commands return domain events.
- `internal/infra/store`: adapter that implements the ports on top of the Stowmark library, with a single handler per session.
- `internal/infra/console`: the terminal user interface built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles) and [Lip Gloss](https://github.com/charmbracelet/lipgloss) v2, which only talks to the application layer. A root model keeps a stack of screens (snapshots, files, preview) and the active dialog; screens emit intent messages and the root model runs the use cases.
- `internal/config`, `internal/buildinfo`, `internal/fixtures`, `internal/mocks`: configuration, version of the binary, test data builders and moq mocks.

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

### Local remote backends

`docker/compose.yaml` starts one server per remote backend, bound to `127.0.0.1` on ports that do not clash with the Stowmark project's own test servers:

| Backend | Port | Environment file |
| --- | --- | --- |
| SSH/SFTP (OpenSSH) | 12222 | `docker/env/ssh.env` |
| S3 (moto) | 15000 | `docker/env/s3.env` |
| SMB (Samba) | 11445 | `docker/env/smb.env` |
| WebDAV | 18081 | `docker/env/webdav.env` |
| GCS (fake-gcs-server) | 14443 | `docker/env/gcs.env` |

| Task | Description |
| --- | --- |
| `task env:up` | Start the backends, create the buckets and the SSH key and `known_hosts` in `docker/.local/` |
| `task env:down` | Stop the backends keeping their data |
| `task env:clean` | Stop the backends and delete their data and keys |
| `task run:remote BACKEND=ssh` | Start the console against one backend (`ssh`, `s3`, `smb`, `webdav` or `gcs`) |
| `task test:integration` | Run the store integration tests against every backend (build tag `integration`) |

Build a local snapshot release with [GoReleaser](https://goreleaser.com/):

```bash
goreleaser release --snapshot --clean
```

## Related project

- [Stowmark](https://github.com/bruli-lab/stowmark) — immutable snapshot backup engine used by this console.

## License

Licensed under the [Apache License 2.0](LICENSE).