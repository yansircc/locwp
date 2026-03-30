# LOCWP

Local WordPress site manager for macOS. Zero sudo, zero configuration — just `locwp add` and go.

```bash
locwp setup          # one-time: install PHP, Caddy, WP-CLI
locwp add            # creates http://localhost:10001 with WordPress ready to go
```

Built on [pawl](https://github.com/yansircc/pawl) — all site lifecycle operations are declarative JSON workflows with built-in retry, progress display, and error handling.

## Features

- **Zero sudo** — no root required, everything runs as your user
- **One-command setup** — `locwp setup` installs and configures everything
- **Port-based** — each site gets its own `http://localhost:<port>` (auto-assigned from 10001)
- **SQLite database** — no daemon, no service, DB is just a file inside the site directory
- **Per-site PHP** — choose PHP 8.1, 8.2, or 8.3 per site
- **Full lifecycle** — add, start, stop, delete with clean teardown
- **WP-CLI passthrough** — run any wp command against any site
- **Editable workflows** — pawl JSON workflows are plain files you can customize

## Requirements

- macOS with [Homebrew](https://brew.sh)
- [pawl](https://github.com/yansircc/pawl) (`cargo install pawl`)
- Go 1.23+ (for building from source)

## Install

```bash
go install github.com/yansircc/locwp@latest
```

Or build from source:

```bash
git clone https://github.com/yansircc/locwp.git
cd locwp
go build -o locwp .
```

## Quick Start

```bash
# 1. One-time setup (installs PHP, Caddy, WP-CLI)
locwp setup

# 2. Create your first site
locwp add --pass secret123

# 3. Done! Open in browser
open http://localhost:10001
```

WordPress admin login: `http://localhost:10001/wp-admin/` (default user: `admin`)

### What `setup` does

- Installs Homebrew packages: PHP, Caddy, WP-CLI
- Writes a Caddyfile that imports per-site configs
- Starts Caddy and PHP-FPM as user-level services (no sudo needed)

## Usage

### Create a site

```bash
locwp add                           # create + provision + start
locwp add --pass secret123          # set WordPress admin password
locwp add --php 8.2                 # use PHP 8.2
locwp add --no-start                # create config only, don't provision
```

Each site is identified by its port number (auto-assigned starting from 10001).

| Flag | Description | Default |
|---|---|---|
| `--php` | PHP version | `8.3` |
| `--user` | WordPress admin username | `admin` |
| `--pass` | WordPress admin password | `admin` |
| `--email` | WordPress admin email | `admin@loc.wp` |
| `--no-start` | Skip provisioning | `false` |

### Manage sites

```bash
locwp list                          # list all sites with status (alias: ls)
locwp stop 10001                    # stop a site
locwp start 10001                   # start a stopped site
locwp delete 10001                  # delete site and all configs (alias: rm)
```

### WP-CLI

Run any WP-CLI command against a site:

```bash
locwp wp 10001 -- option get siteurl
locwp wp 10001 -- plugin list
locwp wp 10001 -- theme activate twentytwentyfour
locwp wp 10001 -- user list
```

## How It Works

```
~/.locwp/
  caddy/
    sites/
      10001.caddy                  # Caddy site config (port 10001)
  sites/
    10001/
      config.json                  # site configuration
      wordpress/                   # WordPress files
      logs/                        # Caddy & PHP logs
      .pawl/workflows/
        provision.json
        start.json
        stop.json
        destroy.json
  php/
    10001.conf                     # PHP-FPM pool config
```

Each site gets:
- A Caddy site block on its own port (`http://localhost:<port>`)
- A dedicated PHP-FPM pool with Unix socket (`/tmp/locwp-<port>.sock`)
- A SQLite database (`wp-content/database/.ht.sqlite`)
- WordPress installed via the [SQLite Database Integration](https://wordpress.org/plugins/sqlite-database-integration/) plugin
- Four pawl workflows for its full lifecycle

Workflows are plain JSON — edit them to add custom steps without touching Go code.

### Environment Variables

| Variable | Description | Default |
|---|---|---|
| `LOCWP_HOME` | Data directory | `~/.locwp` |

## Testing

```bash
# Unit tests
go test ./...

# E2E tests (full setup → add → verify → lifecycle)
bash scripts/test-e2e.sh

# Edge case tests (idempotency, invalid args, rapid cycles, etc.)
bash scripts/test-edge.sh
```

## License

[MIT](LICENSE)
