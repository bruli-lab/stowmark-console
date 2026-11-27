# Stowmark Console

[![CI](https://github.com/bruli-lab/stowmark-console/actions/workflows/release.yml/badge.svg)](https://github.com/bruli-lab/stowmark-console/actions/workflows/release.yml)
[![Codecov](https://codecov.io/gh/bruli-lab/stowmark-console/branch/main/graph/badge.svg)](https://codecov.io/gh/bruli-lab/stowmark-console)
[![Release](https://img.shields.io/github/v/release/bruli-lab/stowmark-console)](https://github.com/bruli-lab/stowmark-console/releases)
[![License](https://img.shields.io/github/license/bruli-lab/stowmark-console)](LICENSE)
[![Go](https://img.shields.io/github/go-mod/go-version/bruli-lab/stowmark-console)](go.mod)

A terminal user interface for browsing and managing [Stowmark](https://github.com/bruli-lab/stowmark) snapshot repositories.

Stowmark Console provides a compact, keyboard-driven interface for listing snapshots, browsing the files they contain, previewing text files, creating snapshots, verifying their integrity, and restoring them.

## Features

- List the snapshots of a Stowmark repository, newest first
- Browse the files of a snapshot by directory, straight from its manifest
- Offer to initialize the repository when it does not exist, choosing compression, level, format version and encryption
- Preview text files stored in a snapshot with line numbers and scrolling
- Create snapshots from a selected source directory
- Verify the integrity of a snapshot
- Check the integrity of the whole repository (`:check`): structure only, all the data or a part of it
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

If the path does not contain a Stowmark repository yet, the console opens an init form, prefilled with the repository configuration: compression (`none`, `zstd`, `lz4`, `gzip`, `xz`), its level when the algorithm has levels (empty uses Stowmark's default), format version (1–3) and encryption. `Tab`/`↑↓` move between fields, `←→` change the selected value and `Enter` initializes the repository. Compression and format cannot change once the repository exists. `Esc` quits without touching the disk, or goes back to the repositories list when several are configured.

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
    retention:
      keep_last: 7
      keep_daily: 14
      keep_monthly: 12
    exclude:
      - "*.tmp"
      - node_modules/
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
| `exclude` | no | Exclude patterns that prefill the create form (same syntax as `.gitignore`) |
| `skip_ignore_files` | no | `true` to ignore the `.stowmarkignore` files of the origin by default |
| `retention` | no | Keep rules that prefill `:forget`: `keep_last`, `keep_daily`, `keep_weekly`, `keep_monthly`, `keep_yearly` and `keep_within` (`30d`, `2w`, `36h`) |
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

The initialization options only apply when the console creates a new repository, and they prefill the init form. Variables read by Stowmark itself, such as `STOWMARK_LOG_LEVEL`, are honored as well.

### Encrypted repositories

A repository is encrypted when it is initialized with a public key. Reading it, creating snapshots and restoring them need the matching private key, so set both keys for the repository. Format version 3 is recommended for encrypted repositories: it also encrypts the manifests.

- The header and the repositories list show a 🔒 for repositories configured with a key, and the panel at the top shows the private key and its fingerprint (the same `SHA256:` fingerprint Stowmark stores in the repository).
- A format version 3 repository opened without its private key is reported as such, instead of offering to initialize it. A wrong or unreadable private key gives an error that names the repository and the setting to fix.
- The console refuses to create snapshots in a repository configured with a public key but no private key: Stowmark would store the new files without encryption in a format version 2 repository.
- When the repository to initialize has no public key configured, the init form offers the encryption `new key pair`: it generates a new key pair in a folder (by default the console's keys folder), initializes the repository with the chosen options (format version 3 by default) and that public key, and opens it with the private key right away. It then shows the lines to add to the repositories file (or the `STOWMARK_*` variables for the `default` repository) so the next session can open it.
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

Press `:` in any view, type a command and press `Enter`. While you type, the footer lists the matching commands and `Tab` completes the suggested one. `Esc` closes the command line. The panel at the top also lists the main commands; `:rewrap` and `:rekey` only appear when the open repository is encrypted.

| Command | Action |
| --- | --- |
| `:q`, `:quit` | Exit |
| `:snap`, `:snapshots`, `:s` | Go back to the snapshots list |
| `:repo`, `:repos`, `:repositories` | List the configured repositories and switch to another one |
| `:keygen` | Generate a key pair for encrypted repositories |
| `:rewrap` | Wrap the key of the open repository with a new public key |
| `:rekey` | Encrypt every object of the open repository with a new key |
| `:check` | Check the integrity of the open repository: structure, all data or a part of it |
| `:forget` | Forget snapshots with a retention policy, after a preview |
| `:prune` | Free the space of the objects no snapshot uses, after showing the plan |
| `:unlock` | Remove the stale locks of the open repository |
| `:unlock!` | Remove every lock of the open repository, after a confirmation |

### Creating and restoring

Creating a snapshot opens a form for the origin directory, prefilled with the source of the selected snapshot. While you type, it suggests the subdirectories of the path and the origins of earlier snapshots: `Tab` accepts the highlighted suggestion and `↑`/`↓` move between them. The form tells you whether the path is an existing directory and only accepts directories.

`Enter` then shows the snapshot options, prefilled with the `exclude` and `skip_ignore_files` of the repository:

- **Exclude**: patterns separated by commas, with the syntax of `.gitignore` (`*.tmp`, `cache/`, `/build`, `!keep.log`). They are added to the `.stowmarkignore` files of the origin.
- **Read .stowmarkignore files**: toggle it with `Space` to ignore those files for this snapshot. The form says whether the origin has one.

Stowmark stores regular files only (and, from format version 3, directories and symbolic links). When the origin has entries it cannot store, such as sockets, named pipes or devices, the message counts them and a dialog lists them.

Restoring asks for a destination directory first. It defaults to `~/stowmark-restore/<snapshot-id>`, never to the original source. The confirmation shows the destination and warns when it already contains files, which would be overwritten. Only `y` confirms the restore.

While an operation is running, `Esc` cancels it. Creating, verifying and restoring snapshots and `:rekey` show a progress bar in the footer with the stage, the files (or objects) and bytes done, the speed and the estimated remaining time.

### Retention and space

`:forget` opens a form with the keep rules, prefilled with the `retention` of the repository when the repositories file has one. A snapshot is kept when any rule matches it, and Stowmark applies the rules to the snapshots of each source on its own:

- **Last**: the newest snapshots.
- **Daily**, **Weekly**, **Monthly**, **Yearly**: the newest snapshot of each of the last N days, weeks, months or years that have snapshots.
- **Within**: every snapshot newer than a duration, such as `30d`, `2w` or `36h`.

`Enter` shows the plan: every snapshot, grouped by source, marked as kept (with the rules that keep it) or forgotten. Nothing changes until you press `f` (forget) or `p` (forget and prune) and confirm. Forgotten snapshots cannot be restored. In S3 and GCS repositories with object lock, snapshots still under retention are kept and the result says until when.

Forgetting only removes the snapshots from the list: their contents stay in the repository until a prune. `:prune` lists every object (slow on S3 and GCS) and shows how many objects no snapshot uses, the space they take and the old key generations left by `:rekey`, and removes them after a confirmation. If some objects used by snapshots are missing, it says so: those snapshots are damaged. Prune takes an exclusive lock on the repository.

Stowmark has no command to forget a single snapshot, so the console does not offer one.

### Repository check

`:check` asks what to check:

- **Structure only** (`s`): reads every manifest and lists the objects to find the missing and unreferenced ones. It is fast and does not need the private key.
- **All data** (`d`): also reads and verifies every object, with a progress bar. It reads the whole repository, and an encrypted repository needs its private key.
- **Part of the data** (`p`): reads a subset of the objects, such as `10%` (a random sample) or `2/5` (the second of five slices, to spread a full read over several runs).

The result screen shows the totals, the damaged snapshots (corrupt manifests or missing objects), the errors and the warnings. Warnings have shortcuts: `p` plans a prune when there are unreferenced objects or old key generations, and `u` removes stale locks. `r` checks again with the same choice. Stowmark removes stale locks itself when the check takes its lock, so they rarely show up.

### Locks

Stowmark locks the repository while it works on it: several readers can share it, but a command that changes the repository (creating a snapshot, rotating keys, pruning) needs it alone. A lock becomes stale 30 minutes after its last refresh, for example when a process crashed, and Stowmark removes stale locks by itself on the next command.

When the repository is locked by another process, the error shows who holds the lock (host and process id). `:unlock` removes the stale locks and reports how many locks are still active. `:unlock!` removes every lock, also those of commands that are still running: use it only when no other Stowmark process works with the repository, because breaking a running command can corrupt the repository.

## Architecture

The project follows a hexagonal architecture:

- `internal/domain`: entities, aggregates, value objects, events and ports (`repository`, `snapshot`, `encryption`, `retention`, `progress`). Events and aggregates are built on [go-core](https://github.com/bruli-lab/go-core) `event`.
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