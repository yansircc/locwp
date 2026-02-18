#!/usr/bin/env bash
# reset.sh — reset locwp to a clean state (keeps brew packages)
set -euo pipefail

LOCWP_HOME="${LOCWP_HOME:-$HOME/.locwp}"
BREW_PREFIX="$(brew --prefix 2>/dev/null || echo /opt/homebrew)"
PHP_VER="${PHP_VER:-8.3}"

echo "=== locwp reset ==="
echo "  LOCWP_HOME: $LOCWP_HOME"
echo "  BREW_PREFIX: $BREW_PREFIX"

# 1. Stop root-level services
echo ""
echo "--- Stopping system services ---"
sudo brew services stop nginx 2>/dev/null || true
sudo nginx -s quit 2>/dev/null || true
sudo pkill -9 nginx 2>/dev/null || true
sleep 1
# Ensure no lingering processes
sudo pkill -9 nginx 2>/dev/null || true
sudo brew services stop dnsmasq 2>/dev/null || true
sleep 1

# 2. Iterate sites and DROP corresponding databases
echo ""
echo "--- Cleaning up databases ---"
if [[ -d "$LOCWP_HOME/sites" ]]; then
  for cfg in "$LOCWP_HOME"/sites/*/config.json; do
    [[ -f "$cfg" ]] || continue
    db_name=$(python3 -c "import json; print(json.load(open('$cfg'))['db_name'])" 2>/dev/null || true)
    if [[ -n "$db_name" ]]; then
      echo "  DROP DATABASE IF EXISTS \`$db_name\`"
      mariadb -e "DROP DATABASE IF EXISTS \`$db_name\`" 2>/dev/null || true
    fi
  done
else
  echo "  (no sites directory)"
fi

# 3. Remove the entire ~/.locwp/ directory
echo ""
echo "--- Removing $LOCWP_HOME ---"
rm -rf "$LOCWP_HOME"

# 4. Remove nginx server configs
echo ""
echo "--- Cleaning up nginx configs ---"
rm -f "$BREW_PREFIX"/etc/nginx/servers/locwp-* 2>/dev/null || true

# 5. Remove FPM pool configs (including legacy wp-local-* format)
echo ""
echo "--- Cleaning up PHP-FPM pool configs ---"
rm -f "$BREW_PREFIX"/etc/php/"$PHP_VER"/php-fpm.d/locwp-* 2>/dev/null || true
rm -f "$BREW_PREFIX"/etc/php/"$PHP_VER"/php-fpm.d/wp-local-* 2>/dev/null || true

# 6. Remove /etc/resolver/wp
echo ""
echo "--- Cleaning up DNS resolver ---"
sudo rm -f /etc/resolver/wp 2>/dev/null || true

# 7. Remove /etc/sudoers.d/locwp
echo ""
echo "--- Cleaning up sudoers ---"
sudo rm -f /etc/sudoers.d/locwp 2>/dev/null || true

# 8. Remove locwp lines from dnsmasq.conf
echo ""
echo "--- Cleaning up dnsmasq config ---"
DNSMASQ_CONF="$BREW_PREFIX/etc/dnsmasq.conf"
if [[ -f "$DNSMASQ_CONF" ]]; then
  # Remove lines containing .loc.wp
  sudo sed -i '' '/\.loc\.wp/d' "$DNSMASQ_CONF" 2>/dev/null || true
  echo "  cleaned dnsmasq.conf"
else
  echo "  (dnsmasq.conf not found)"
fi

# 9. Restore nginx.conf (setup will overwrite, skip here)
echo ""
echo "--- Skipping nginx.conf restore (setup will overwrite) ---"

# 10. Restart user-level services
echo ""
echo "--- Restarting user-level services ---"
brew services restart mariadb 2>/dev/null || true
brew services restart "php@$PHP_VER" 2>/dev/null || true

# Wait for MariaDB to be ready
echo -n "  Waiting for MariaDB"
for i in $(seq 1 15); do
  if mariadb -e "SELECT 1" &>/dev/null; then
    echo " OK"
    break
  fi
  echo -n "."
  sleep 1
done

echo ""
echo "=== Reset complete ==="
