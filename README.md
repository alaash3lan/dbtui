# dbplus

A modern terminal-based MySQL/MariaDB client built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/license-MIT-blue)

## Features

- **Split-pane TUI** — sidebar, data grid, and query editor in one view
- **Table browser** — navigate tables with keyboard, view schema info
- **Data grid** — scroll, filter, and paginate through table data
- **Query editor** — execute SQL with history navigation and multi-line support
- **Server-side pagination** — efficiently browse large tables page by page
- **Database switching** — `USE dbname;` to switch databases without reconnecting
- **Quick filter** — filter displayed rows by column or across all columns (`column | value`)
- **Dark/Light themes** — toggle with `Ctrl+T`
- **Query cancellation** — `Ctrl+C` cancels a running query, press again to quit
- **Auto-refresh** — sidebar updates after DDL operations (CREATE, DROP, ALTER)

## Installation

```bash
go install github.com/alaa/dbplus@latest
```

Or build from source:

```bash
git clone https://github.com/alaa/dbplus.git
cd dbplus
go build -o dbplus .
```

## Usage

```bash
# Connect with flags


 -u root -p secret mydb

# Connect with DSN
dbplus --dsn "root:secret@tcp(127.0.0.1:3306)/mydb"

# Show version
dbplus --version
```

### Flags

| Flag        | Description                          | Default     |
|-------------|--------------------------------------|-------------|
| `-u`        | MySQL user                           | (required)  |
| `-p`        | MySQL password                       |             |
| `-h`        | MySQL host                           | `127.0.0.1` |
| `-P`        | MySQL port                           | `3306`      |
| `--dsn`     | Full DSN string (overrides others)   |             |
| `--version` | Show version                         |             |

## Keyboard Shortcuts

### Global

| Key           | Action                    |
|---------------|---------------------------|
| `Ctrl+C`      | Cancel query / Quit       |
| `Tab`         | Next pane                 |
| `Shift+Tab`   | Previous pane             |
| `Ctrl+Left`   | Shrink sidebar            |
| `Ctrl+Right`  | Grow sidebar              |
| `Ctrl+T`      | Toggle dark/light theme   |
| `Ctrl+R`      | Refresh tables & data     |
| `F1`          | Toggle help overlay       |

### Sidebar

| Key           | Action                    |
|---------------|---------------------------|
| `j` / `k`     | Navigate tables           |
| `Enter`       | Select table, load data   |
| `i`           | Toggle schema info        |
| `g` / `G`     | First / last table        |

### Data View

| Key           | Action                    |
|---------------|---------------------------|
| Arrows / `hjkl` | Scroll grid             |
| `/` / `Ctrl+F`  | Activate filter         |
| `Escape`      | Clear filter              |
| `n` / `p`     | Next / prev server page   |
| `PgUp` / `PgDn` | Scroll viewport up/down |
| `Home` / `End`  | First / last row        |

### Query Editor

| Key           | Action                    |
|---------------|---------------------------|
| `Enter`       | Execute (requires `;`) or newline |
| `Ctrl+E`      | Force execute without `;` |
| `Up` / `Down` | Navigate query history    |
| `Escape`      | Clear input               |

## Configuration

dbplus looks for a TOML config file in:

1. `$XDG_CONFIG_HOME/dbplus/config.toml`
2. `~/.config/dbplus/config.toml`
3. `~/.dbplus.toml`

### Example config

```toml
[display]
theme = "dark"           # "dark" or "light"
page_size = 100          # rows per page
sidebar_width = 20       # sidebar width percentage
editor_height = 8        # editor height in lines

[query]
timeout_seconds = 30     # query timeout

[history]
max_entries = 500        # max stored queries
save_to_file = true
```

## Tips

- Type `USE dbname;` in the editor to switch databases
- Use `column | value` format in the filter for column-specific search
- `Ctrl+C` cancels a running query — press again to quit

## Project Structure

```
dbplus/
├── main.go                         # Entry point
├── cmd/root.go                     # CLI flag parsing
└── internal/
    ├── config/                     # TOML config loading
    ├── database/                   # MySQL connection, queries, executor
    ├── stringutil/                 # Shared string utilities
    └── tui/                        # Bubble Tea TUI
        ├── app.go                  # Root model, layout, message routing
        ├── keys.go                 # Global keybindings
        ├── messages.go             # Message types
        ├── styles.go               # Lipgloss styles
        ├── theme.go                # Dark/light theme definitions
        └── components/
            ├── sidebar/            # Table list navigation
            ├── dataview/           # Data grid with filtering
            ├── editor/             # Query editor with history
            ├── statusbar/          # Connection info, query stats
            └── titlebar/           # App title, row count
```

## Running Tests

```bash
# Set test database credentials (defaults to root:root@127.0.0.1:3306/dbplus_test)
export DBPLUS_TEST_USER=root
export DBPLUS_TEST_PASS=root
export DBPLUS_TEST_DB=dbplus_test

go test ./... -v
```

## License

MIT
