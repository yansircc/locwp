# locwp

Local WordPress site manager for macOS. Create and manage WordPress development sites using native Homebrew services (PHP, MariaDB, Nginx).

## Features

- One-command WordPress site provisioning
- Per-site PHP version selection (8.1, 8.2, 8.3)
- Auto-configured Nginx vhosts and PHP-FPM pools
- Start/stop individual sites without affecting others
- WP-CLI passthrough for each site
- Integrated [pawl](https://github.com/yansircc/pawl) workflow for automated setup

## Requirements

- macOS with [Homebrew](https://brew.sh)
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
# Install dependencies (PHP, MariaDB, Nginx, WP-CLI)
locwp setup

# Create a new WordPress site
locwp add mysite

# That's it! Visit http://localhost:8081
```

## Usage

```bash
locwp add <name>              # Create a new site
locwp add <name> --port 9090  # Specify port
locwp add <name> --php 8.2    # Specify PHP version
locwp add <name> --no-start   # Create without provisioning

locwp list                    # List all sites (alias: ls)
locwp start <name>            # Start a site
locwp stop <name>             # Stop a site
locwp delete <name>           # Delete a site and its database (alias: rm)

locwp wp <name> -- plugin list       # Run WP-CLI commands
locwp wp <name> -- theme activate twentytwentyfour

locwp setup                   # Install Homebrew dependencies
locwp setup --php 8.2         # Install a specific PHP version
```

## How It Works

`locwp` manages WordPress sites using native macOS services:

```
~/.locwp/
  sites/
    mysite/
      config.json          # Site configuration
      wordpress/           # WordPress files
      logs/                # Nginx & PHP logs
      .pawl/config.json    # Provisioning workflow
  nginx/
    sites/
      mysite.conf          # Nginx vhost
  php/
    mysite.conf            # PHP-FPM pool
```

Each site gets:
- A dedicated Nginx vhost (port auto-assigned from 8081)
- A dedicated PHP-FPM pool with Unix socket
- A MariaDB database (`wp_<name>`)
- WordPress admin credentials (default: `admin` / `admin`)

### Environment Variables

| Variable | Description | Default |
|---|---|---|
| `LOCWP_HOME` | Data directory | `~/.locwp` |

### Add Command Flags

| Flag | Description | Default |
|---|---|---|
| `--port` | Port number | Auto-assigned from 8081 |
| `--php` | PHP version | `8.3` |
| `--user` | WordPress admin username | `admin` |
| `--pass` | WordPress admin password | `admin` |
| `--email` | WordPress admin email | `admin@local.test` |
| `--no-start` | Skip provisioning | `false` |

## Testing

```bash
# Unit tests
go test ./...

# E2E tests
bash tests/e2e.sh
```

## License

[MIT](LICENSE)
