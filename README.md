# locwp

Local WordPress site manager for macOS. Create and manage WordPress development sites with HTTPS, custom domains, and zero configuration — all powered by native Homebrew services.

```bash
locwp setup              # one-time: install PHP, MariaDB, Nginx, dnsmasq, mkcert
locwp add mysite         # creates https://mysite.loc.wp with WordPress ready to go
```

Built on [pawl](https://github.com/yansircc/pawl) — all site lifecycle operations are declarative JSON workflows with built-in retry, progress display, and error handling.

## Features

- **One-command setup** — `locwp setup` installs and configures everything
- **HTTPS by default** — wildcard SSL via mkcert (`*.loc.wp`), no browser warnings
- **Custom domains** — each site gets `<name>.loc.wp`, resolved via local dnsmasq
- **Per-site PHP** — choose PHP 8.1, 8.2, or 8.3 per site
- **Isolated services** — dedicated Nginx vhost + PHP-FPM pool per site
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
# 1. One-time setup (installs PHP, MariaDB, Nginx, dnsmasq, mkcert, configures SSL + DNS)
locwp setup

# 2. Create your first site
locwp add mysite --pass secret123

# 3. Done! Open in browser
open https://mysite.loc.wp
```

WordPress admin login: `https://mysite.loc.wp/wp-admin/` (default user: `admin`)

### What `setup` does

- Installs Homebrew packages: PHP, MariaDB, Nginx, WP-CLI, dnsmasq, mkcert
- Generates a `*.loc.wp` wildcard SSL certificate (trusted by your browser)
- Configures dnsmasq so all `*.loc.wp` domains resolve to `127.0.0.1`
- Sets up macOS DNS resolver (`/etc/resolver/wp`)
- Configures passwordless `sudo nginx -s reload` so adding sites never prompts for password
- Starts Nginx (ports 80/443) and dnsmasq (port 53) as root services

> **Note:** `setup` requires `sudo` for DNS resolver, dnsmasq, and Nginx. You'll be prompted once.

### Proxy users (Surge, ClashX, etc.)

If you use a system proxy, add `*.loc.wp` to your skip-proxy list. Otherwise HTTPS requests to local sites will be routed through the proxy and fail.

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
| `--email` | WordPress admin email | `admin@local.test` |
| `--no-start` | Skip provisioning | `false` |

### Manage sites

```bash
locwp list                          # list all sites with status (alias: ls)
locwp stop <name>                   # stop a site (nginx vhost disabled)
locwp start <name>                  # start a stopped site
locwp delete <name>                 # delete site, database, and all configs (alias: rm)
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
  ssl/
    _wildcard.loc.wp.pem          # SSL certificate
    _wildcard.loc.wp-key.pem      # SSL key
  sites/
    mysite/
      config.json                  # site configuration
      wordpress/                   # WordPress files
      logs/                        # Nginx & PHP logs
      .pawl/workflows/
        provision.json             # initial setup workflow
        start.json                 # start site workflow
        stop.json                  # stop site workflow
        destroy.json               # teardown workflow
  nginx/
    sites/
      mysite.conf                  # Nginx vhost (HTTPS)
  php/
    mysite.conf                    # PHP-FPM pool config
```

Each site gets:
- An Nginx vhost with HTTPS (`*.loc.wp` wildcard cert)
- A dedicated PHP-FPM pool with Unix socket (`/tmp/locwp-<name>.sock`)
- A MariaDB database (`wp_<name>`, hyphens converted to underscores)
- WordPress installed and configured
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
