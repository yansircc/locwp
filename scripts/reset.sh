#!/usr/bin/env bash
# reset.sh — reset locwp to a clean state (keeps brew packages)
set -euo pipefail

LOCWP_HOME="${LOCWP_HOME:-$HOME/.locwp}"
BREW_PREFIX="$(brew --prefix 2>/dev/null || echo /opt/homebrew)"
PHP_VER="${PHP_VER:-8.3}"

echo "=== locwp reset ==="
echo "  LOCWP_HOME: $LOCWP_HOME"
echo "  BREW_PREFIX: $BREW_PREFIX"

# 1. Stop Caddy
echo ""
echo "--- Stopping Caddy ---"
brew services stop caddy 2>/dev/null || true
sleep 1

# 2. Remove the entire ~/.locwp/ directory (includes site data, SQLite DBs, caddy configs)
echo ""
echo "--- Removing $LOCWP_HOME ---"
rm -rf "$LOCWP_HOME"

# 3. Remove Caddyfile
echo ""
echo "--- Cleaning up Caddyfile ---"
rm -f "$BREW_PREFIX/etc/Caddyfile" 2>/dev/null || true

# 4. Remove FPM pool configs (including legacy wp-local-* format)
echo ""
echo "--- Cleaning up PHP-FPM pool configs ---"
rm -f "$BREW_PREFIX"/etc/php/"$PHP_VER"/php-fpm.d/locwp-* 2>/dev/null || true
rm -f "$BREW_PREFIX"/etc/php/"$PHP_VER"/php-fpm.d/wp-local-* 2>/dev/null || true

# 5. Restart user-level services
echo ""
echo "--- Restarting user-level services ---"
brew services restart "php@$PHP_VER" 2>/dev/null || true

echo ""
echo "=== Reset complete ==="
