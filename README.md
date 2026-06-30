# Noxie-Sort

> Automatic file sorting daemon for Linux. Watches directories and moves files into categorized folders based on configurable extension rules.

---

## Features

- **Daemon mode** — runs silently in the background, reacts to new files instantly via `fsnotify`
- **Multi-directory scanning** — watch one primary dir or a list of directories simultaneously
- **Rule-based sorting** — map file extensions to target folders with full control
- **Interactive TUI** — configure everything visually without touching a config file (`-i` flag)
- **Structured logging** — per-level log files with rotation powered by lumberjack
- **OpenTelemetry** — built-in traces and metrics export via OTLP
- **Graceful shutdown** — handles `SIGINT` / `SIGTERM` cleanly

---

## Stack

| Layer | Library |
|---|---|
| Language | Go 1.25 |
| File watching | `fsnotify/fsnotify` |
| Config format | TOML via `pelletier/go-toml/v2` |
| Log rotation | `lumberjack.v2` |
| File locking | `gofrs/flock` |
| TUI framework | `charmbracelet/bubbletea` |
| TUI styling | `charmbracelet/lipgloss` |
| Observability | OpenTelemetry SDK (OTLP HTTP) |

---

## Installation

```bash
git clone https://github.com/Nox1KCL/Noxie-Sort.git
cd Noxie-Sort
go build -o nxe-sort ./cmd/nxe-sort
```

Place the binary somewhere on your `$PATH`:

```bash
mv nxe-sort /usr/local/bin/
```

---

## Configuration

### Quick start — interactive TUI

The easiest way to set up a config is through the built-in editor. Requires a terminal at least **96×39** characters wide.

```bash
nxe-sort -i
```

The TUI will guide you through all fields and validate them before saving. When the badge in the top-right corner shows **TUNED**, the config is valid and ready.

### Manual config file

Create a `config.toml` in one of the locations below (checked in order):

| Priority | Path |
|---|---|
| 1 | `./config.toml` (current directory) |
| 2 | `$HOME/Noxie-Sort/config.toml` |
| 3 | `/etc/Noxie-Sort/config.toml` |
| 4 | `$XDG_CONFIG_HOME/Noxie-Sort/config.toml` |

You can also override the path with a flag or an environment variable:

```bash
nxe-sort --config /path/to/my.toml
IFS_CONFIG_PATH=/path/to/my.toml nxe-sort
```

### Config reference

```toml
# Primary directory to watch (single dir shorthand)
scan_dir = "$HOME/Downloads/"

# Additional directories to watch
scan_dirs = ["$HOME/Downloads/", "$HOME/Documents"]

# Directory where log files are written
logs_dir = "./logs"

# Sorting rules — each key is the rule name
[rules.Images]
target_dir = "Images"
extensions = [".jpg", ".png", ".gif"]

[rules.Documents]
target_dir = "Documents"
extensions = [".pdf", ".docx", ".txt"]

# Log rotation settings
[logger]
max_size    = 10   # MB before rotation
max_age     = 28   # days to keep old logs
max_backups = 4    # number of rotated files to keep
compress    = true # gzip rotated logs
```

> **Note:** `scan_dir` and `scan_dirs` can be used together. At least one of them must be set. Extensions must be unique across all rules.

---

## Usage

```bash
# Start the sorter (foreground)
nxe-sort

# Open the interactive config editor
nxe-sort -i

# Start the sorter with a specific config file
nxe-sort --config /path/to/config.toml

# Run as a background process
nxe-sort --background

# Run a one-time sort and exit (using scan_dir from config)
nxe-sort --once default

# Run a one-time sort on a specific directory and exit
nxe-sort --once /path/to/folder

# Manage the system daemon
nxe-sort --daemon install
nxe-sort --daemon uninstall
```

### All flags

| Flag | Type | Description |
|---|---|---|
| `-i` | bool | Launch the interactive TUI config editor |
| `--config` | string | Path to config file |
| `--background` | bool | Spawn a detached child process |
| `--once` | string | One-time sort (`default` for config.ScanDir or absolute path) |
| `--daemon` | string | `install` or `uninstall` system daemon |
| `--stop` | bool | Stop the running process |

---

## Project structure

```
cmd/
  nxe-sort/      Entry point, flag routing
  server/        Ping server (health check)
internal/
  config/        Config struct, loading, validation, path discovery
  tui/           Interactive config editor (Bubble Tea)
    backstage.go   Styles, colours, types  (frontend)
    tui.go         AppModel logic          (frontend)
    core.go        RunTUI, GenerateReport  (bridge)
  files/         File moving and sorting logic
  watcher/       Directory scanner and worker pool
  daemon/        System service management
  background/    Child process management
  logger/        Structured multi-file logger
  telemetry/     OpenTelemetry setup
  syncutils/     WaitGroup helpers
  ping/          Health check endpoint
```

---

## Docker

```bash
docker compose up --build
```

The compose file starts the ping server by default. To run the sorter instead, update the `CMD` in `Dockerfile`.

---

## License

MIT
