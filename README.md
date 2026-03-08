# dbtui

A modern terminal-based MySQL/MariaDB client built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/license-MIT-blue)

## Features

- **Split-pane TUI** — sidebar, data grid, and query editor in one view
- **Table browser** — navigate tables with keyboard, view schema info
- **Table search** — press `/` in the sidebar to filter tables by name
- **Data grid** — scroll, filter, and paginate through table data
- **Query editor** — execute SQL with history navigation and multi-line support
- **SQL syntax highlighting** — keywords, strings, and numbers colored when editor is unfocused
- **Query history persistence** — history saved to disk and restored across sessions
- **Server-side pagination** — efficiently browse large tables page by page
- **Database switching** — `USE dbname;` or `Ctrl+D` to switch databases without reconnecting
- **Quick filter** — filter displayed rows by column or across all columns (`column | value`)
- **Export data** — export query results as CSV (`Ctrl+S`) or JSON (`Ctrl+J`)
- **Copy to clipboard** — copy cell (`c`) or row (`y`) from the data view
- **Dark/Light themes** — toggle with `Ctrl+T`
- **Query cancellation** — `Ctrl+C` cancels a running query, press again to quit
- **Auto-reconnect** — automatically reconnects up to 3 times on connection drop
- **Auto-refresh** — sidebar updates after DDL operations (CREATE, DROP, ALTER)

## Installation

```bash
go install github.com/alaa/dbtui@latest
```

Or build from source:

```bash
git clone https://github.com/alaa/dbtui.git
cd dbtui
go build -o dbtui .
```

## Usage

```bash
# Connect with flags
dbtui -u root -p secret mydb

# Connect with DSN
dbtui --dsn "root:secret@tcp(127.0.0.1:3306)/mydb"

# Show version
dbtui --version
```

### Flags

| Flag        | Description                          | Default     |
|-------------|--------------------------------------|-------------|
| `-u`        | MySQL user                           | (required)  |
| `-p`        | MySQL password                       |             |
| `-h`        | MySQL host                           | `127.0.0.1` |
| `-P`        | MySQL port                           | `3306`      |
| `-c`        | Connection profile name from config  |             |
| `--dsn`     | Full DSN string (overrides others)   |             |
| `--tls`     | TLS mode: `true`, `skip-verify`, or CA cert path |  |
| `--tls-cert`| Path to client certificate (mutual TLS) |          |
| `--tls-key` | Path to client key (mutual TLS)      |             |
| `--version` | Show version                         |             |

### Connection Profiles

Save named connection profiles in your config file to avoid typing credentials every time:

```toml
[[connections]]
name = "local"
host = "127.0.0.1"
port = 3306
user = "root"
password = "secret"
database = "mydb"

[[connections]]
name = "production"
host = "db.example.com"
port = 3306
user = "app"
password = "prod_pass"
database = "appdb"
```

Then connect using a profile name:

```bash
# Use a saved profile
dbtui -c local

# Use a profile but override the password
dbtui -c local -p newpass

# Use a profile but connect to a different database
dbtui -c production otherdb
```

### TLS/SSL

Connect to MySQL servers that require TLS:

```bash
# Use system CA certificates
dbtui -u root -p pass --tls true mydb

# Skip certificate verification (self-signed certs)
dbtui -u root -p pass --tls skip-verify mydb

# Use a specific CA certificate
dbtui -u root -p pass --tls /path/to/ca.pem mydb

# Mutual TLS with client certificate and key
dbtui -u root -p pass --tls /path/to/ca.pem --tls-cert /path/to/client.pem --tls-key /path/to/client-key.pem mydb
```

TLS can also be configured in connection profiles:

```toml
[[connections]]
name = "secure"
host = "db.example.com"
user = "app"
password = "secret"
database = "appdb"
tls = "/path/to/ca.pem"
tls_cert = "/path/to/client.pem"
tls_key = "/path/to/client-key.pem"
```

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
| `Ctrl+S`      | Export data as CSV         |
| `Ctrl+J`      | Export data as JSON        |
| `Ctrl+X`      | Explain current query     |
| `Ctrl+D`      | Switch database           |
| `F1`          | Toggle help overlay       |

### Sidebar

| Key           | Action                    |
|---------------|---------------------------|
| `j` / `k`     | Navigate tables           |
| `Enter`       | Select table, load data   |
| `i`           | Toggle schema info        |
| `g` / `G`     | First / last table        |
| `/`           | Filter tables by name     |
| `Escape`      | Clear filter              |

### Data View

| Key           | Action                    |
|---------------|---------------------------|
| Arrows / `hjkl` | Scroll grid             |
| `/` / `Ctrl+F`  | Activate filter         |
| `Escape`      | Clear filter              |
| `n` / `p`     | Next / prev server page   |
| `PgUp` / `PgDn` | Scroll viewport up/down |
| `Home` / `End`  | First / last row        |
| `c`           | Copy cell to clipboard    |
| `y`           | Copy row to clipboard     |
| `d`           | Toggle row detail view    |

### Query Editor

| Key           | Action                    |
|---------------|---------------------------|
| `Enter`       | Execute (requires `;`) or newline |
| `Ctrl+E`      | Force execute without `;` |
| `Up` / `Down` | Navigate query history    |
| `Escape`      | Clear input               |

## Configuration

dbtui looks for a TOML config file in:

1. `$XDG_CONFIG_HOME/dbtui/config.toml`
2. `~/.config/dbtui/config.toml`
3. `~/.dbtui.toml`

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

- Press `Ctrl+D` to open the database switcher, or type `USE dbname;` in the editor
- Use `column | value` format in the filter for column-specific search
- `Ctrl+C` cancels a running query — press again to quit
- Query history is saved to `~/.config/dbtui/history` by default
- Exports are saved to the current working directory as `dbtui_export_YYYYMMDD_HHMMSS.csv/json`
- SQL keywords are syntax-highlighted when you tab away from the editor

## Project Structure

```
dbtui/
├── main.go                         # Entry point
├── cmd/root.go                     # CLI flag parsing
└── internal/
    ├── config/                     # TOML config loading
    ├── database/                   # MySQL connection, queries, executor
    ├── stringutil/                 # Shared string utilities
    └── tui/                        # Bubble Tea TUI
        ├── app.go                  # Root model, layout, message routing
        ├── clipboard.go            # Clipboard integration
        ├── export.go               # CSV/JSON export
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
# Set test database credentials (defaults to root:root@127.0.0.1:3306/dbtui_test)
export DBTUI_TEST_USER=root
export DBTUI_TEST_PASS=root
export DBTUI_TEST_DB=dbtui_test

go test ./... -v
```

## License

MIT
