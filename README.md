# LOCWP

Local WordPress site manager for macOS. Create and manage WordPress development sites on localhost with per-site ports — zero sudo, zero configuration, powered by native Homebrew services.

```bash
locwp setup              # one-time: install PHP, Caddy, WP-CLI
locwp add mysite         # creates http://localhost:10001 with WordPress ready to go
```

Built on [pawl](https://github.com/yansircc/pawl) — all site lifecycle operations are declarative JSON workflows with built-in retry, progress display, and error handling.

## Features

- **Zero sudo** — no root required, everything runs as your user
- **One-command setup** — `locwp setup` installs and configures everything
- **Per-site ports** — each site gets its own `http://localhost:<port>` (starting from 10001)
- **Per-site PHP** — choose PHP 8.1, 8.2, or 8.3 per site
- **SQLite database** — no daemon, no service, DB is just a file inside the site directory
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
locwp add mysite --pass secret123

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
locwp add <name>                    # create + provision + start
locwp add <name> --pass secret123   # set WordPress admin password
locwp add <name> --php 8.2          # use PHP 8.2
locwp add <name> --no-start         # create config only, don't provision
```

Site names: lowercase letters, digits, and hyphens. Cannot start or end with a hyphen.

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
locwp stop <name>                   # stop a site (Caddy conf disabled)
locwp start <name>                  # start a stopped site
locwp delete <name>                 # delete site and all configs (alias: rm)
```

### WP-CLI

Run any WP-CLI command against a site:

```bash
locwp wp <name> -- option get siteurl
locwp wp <name> -- plugin list
locwp wp <name> -- theme activate twentytwentyfour
locwp wp <name> -- user list
```

## How It Works

```
~/.locwp/
  caddy/
    sites/
      mysite.caddy                 # Caddy site config (port 10001)
  sites/
    mysite/
      config.json                  # site configuration (includes port)
      wordpress/                   # WordPress files
      logs/                        # Caddy & PHP logs
      .pawl/workflows/
        provision.json             # initial setup workflow
        start.json                 # start site workflow
        stop.json                  # stop site workflow
        destroy.json               # teardown workflow
  php/
    mysite.conf                    # PHP-FPM pool config
```

Each site gets:
- A Caddy site block on its own port (`http://localhost:<port>`)
- A dedicated PHP-FPM pool with Unix socket (`/tmp/locwp-<name>.sock`)
- A SQLite database (`wp-content/database/.ht.sqlite`)
- WordPress installed and configured via the [SQLite Database Integration](https://wordpress.org/plugins/sqlite-database-integration/) plugin
- Four pawl workflows for its full lifecycle

Workflows are plain JSON — edit them to add custom steps (install plugins, import data, seed content) without touching Go code.

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

# Edge case tests (boundary names, idempotency, filesystem corruption, etc.)
bash scripts/test-edge.sh
```

## License

[MIT](LICENSE)
