#!/usr/bin/env bash
# reset.sh — 还原 locwp 到全新状态（保留 brew 包）
set -euo pipefail

LOCWP_HOME="${LOCWP_HOME:-$HOME/.locwp}"
BREW_PREFIX="$(brew --prefix 2>/dev/null || echo /opt/homebrew)"
PHP_VER="${PHP_VER:-8.3}"

echo "=== locwp reset ==="
echo "  LOCWP_HOME: $LOCWP_HOME"
echo "  BREW_PREFIX: $BREW_PREFIX"

# 1. 停止 root 级别的服务
echo ""
echo "--- 停止系统服务 ---"
sudo brew services stop nginx 2>/dev/null || true
sudo nginx -s quit 2>/dev/null || true
sudo pkill -9 nginx 2>/dev/null || true
sleep 1
# 确认无残留进程
sudo pkill -9 nginx 2>/dev/null || true
sudo brew services stop dnsmasq 2>/dev/null || true
sleep 1

# 2. 遍历 sites，DROP 对应数据库
echo ""
echo "--- 清理数据库 ---"
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
  echo "  (无站点目录)"
fi

# 3. 删除 ~/.locwp/ 整个目录
echo ""
echo "--- 删除 $LOCWP_HOME ---"
rm -rf "$LOCWP_HOME"

# 4. 删除 nginx 服务器配置
echo ""
echo "--- 清理 nginx 配置 ---"
rm -f "$BREW_PREFIX"/etc/nginx/servers/locwp-* 2>/dev/null || true

# 5. 删除 FPM pool 配置（包括旧格式 wp-local-*）
echo ""
echo "--- 清理 PHP-FPM pool 配置 ---"
rm -f "$BREW_PREFIX"/etc/php/"$PHP_VER"/php-fpm.d/locwp-* 2>/dev/null || true
rm -f "$BREW_PREFIX"/etc/php/"$PHP_VER"/php-fpm.d/wp-local-* 2>/dev/null || true

# 6. 删除 /etc/resolver/wp
echo ""
echo "--- 清理 DNS resolver ---"
sudo rm -f /etc/resolver/wp 2>/dev/null || true

# 7. 删除 /etc/sudoers.d/locwp
echo ""
echo "--- 清理 sudoers ---"
sudo rm -f /etc/sudoers.d/locwp 2>/dev/null || true

# 8. 从 dnsmasq.conf 中移除 locwp 行
echo ""
echo "--- 清理 dnsmasq 配置 ---"
DNSMASQ_CONF="$BREW_PREFIX/etc/dnsmasq.conf"
if [[ -f "$DNSMASQ_CONF" ]]; then
  # 移除包含 .loc.wp 的行
  sudo sed -i '' '/\.loc\.wp/d' "$DNSMASQ_CONF" 2>/dev/null || true
  echo "  已清理 dnsmasq.conf"
else
  echo "  (dnsmasq.conf 不存在)"
fi

# 9. 恢复 nginx.conf（setup 会覆盖，这里不做）
echo ""
echo "--- 跳过 nginx.conf 恢复（setup 会覆盖）---"

# 10. 重启用户级服务
echo ""
echo "--- 重启用户级服务 ---"
brew services restart mariadb 2>/dev/null || true
brew services restart "php@$PHP_VER" 2>/dev/null || true

# 等待 MariaDB 就绪
echo -n "  等待 MariaDB 就绪"
for i in $(seq 1 15); do
  if mariadb -e "SELECT 1" &>/dev/null; then
    echo " OK"
    break
  fi
  echo -n "."
  sleep 1
done

echo ""
echo "=== Reset 完成 ==="
