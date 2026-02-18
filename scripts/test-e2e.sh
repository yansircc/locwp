#!/usr/bin/env bash
# test-e2e.sh — 自动化闭环 E2E 测试
# 真实运行 locwp setup + add，验证完整流程
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BINARY="$PROJECT_DIR/locwp"
PASS=0
FAIL=0
TOTAL_START=$(date +%s)
TMPOUT=$(mktemp)

# ─── 颜色 ───────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { ((PASS++)); echo -e "  ${GREEN}PASS${NC}: $1"; }
fail() { ((FAIL++)); echo -e "  ${RED}FAIL${NC}: $1"; }

assert_eq() {
  if [[ "$1" == "$2" ]]; then pass "$3"; else fail "$3 (got '$1', want '$2')"; fi
}
assert_contains() {
  if echo "$1" | grep -q "$2"; then pass "$3"; else fail "$3 (output missing '$2')"; fi
}
assert_file_exists() {
  if [[ -f "$1" ]]; then pass "$2"; else fail "$2 ($1 not found)"; fi
}
assert_dir_exists() {
  if [[ -d "$1" ]]; then pass "$2"; else fail "$2 ($1 not found)"; fi
}
assert_exit_code() {
  if [[ "$1" -eq "$2" ]]; then pass "$3"; else fail "$3 (exit code $1, want $2)"; fi
}
assert_http_status() {
  local url="$1" expected="$2" label="$3"
  local status
  for attempt in 1 2 3 4 5; do
    status=$(curl -sk --noproxy '*' -o /dev/null -w '%{http_code}' --connect-timeout 3 --max-time 5 "$url" 2>/dev/null) || true
    [[ "$status" == "$expected" ]] && break
    sleep 1
  done
  if [[ "$status" == "$expected" ]]; then
    pass "$label (HTTP $status)"
  else
    fail "$label (got HTTP $status, want $expected)"
  fi
}

# 运行命令并捕获输出（保持 stdin 连接到 tty）
run_capture() {
  "$@" > "$TMPOUT" 2>&1
  local rc=$?
  cat "$TMPOUT"
  return $rc
}

cleanup() {
  rm -f "$TMPOUT"
  # 移除临时 NOPASSWD 规则
  sudo rm -f /etc/sudoers.d/locwp-test 2>/dev/null || true
  exit
}
trap cleanup EXIT INT TERM

# ─── 缓存 sudo 并设置 NOPASSWD ─────────────────────────
echo -e "${YELLOW}=== 设置 sudo ===${NC}"
echo 'a23456' | sudo -S -v 2>/dev/null
if [[ $? -ne 0 ]]; then
  echo -e "${RED}sudo 密码错误，退出${NC}"
  exit 1
fi
# 设置临时 NOPASSWD 规则，确保子进程中 sudo 也不需要密码
echo "$(whoami) ALL=(ALL) NOPASSWD: ALL" | sudo tee /etc/sudoers.d/locwp-test >/dev/null
sudo chmod 0440 /etc/sudoers.d/locwp-test
echo "  [ok] sudo NOPASSWD 已配置"

# ─── 编译 ──────────────────────────────────────────────
echo ""
echo -e "${YELLOW}=== 编译最新二进制 ===${NC}"
(cd "$PROJECT_DIR" && go build -o locwp .)
if [[ $? -ne 0 ]]; then
  echo -e "${RED}编译失败，退出${NC}"
  exit 1
fi
echo "  [ok] 已编译 $BINARY"

# ─── Reset ──────────────────────────────────────────────
echo ""
echo -e "${YELLOW}=== 执行 Reset ===${NC}"
bash "$SCRIPT_DIR/reset.sh"

# reset 会删除 /etc/sudoers.d/locwp-test，重新设置
echo "$(whoami) ALL=(ALL) NOPASSWD: ALL" | sudo tee /etc/sudoers.d/locwp-test >/dev/null
sudo chmod 0440 /etc/sudoers.d/locwp-test

# ─── Test 1: locwp setup ────────────────────────────────
echo ""
echo -e "${YELLOW}=== Test 1: locwp setup ===${NC}"
setup_rc=0; run_capture "$BINARY" setup || setup_rc=$?
setup_out=$(cat "$TMPOUT")

assert_exit_code "$setup_rc" 0 "setup 退出码为 0"
assert_contains "$setup_out" "Setup complete" "setup 输出 Setup complete"

# 验证 setup 产物
LOCWP_HOME="$HOME/.locwp"
BREW_PREFIX="$(brew --prefix)"

assert_dir_exists "$LOCWP_HOME/ssl" "SSL 目录存在"
assert_file_exists "$LOCWP_HOME/ssl/_wildcard.loc.wp.pem" "通配符证书存在"
assert_file_exists "$LOCWP_HOME/ssl/_wildcard.loc.wp-key.pem" "通配符证书密钥存在"
assert_file_exists "/etc/resolver/wp" "DNS resolver 配置存在"
assert_file_exists "/etc/sudoers.d/locwp" "sudoers 配置存在"

# 验证 dnsmasq 配置
if grep -q 'address=/.loc.wp/127.0.0.1' "$BREW_PREFIX/etc/dnsmasq.conf"; then
  pass "dnsmasq 配置包含 .loc.wp"
else
  fail "dnsmasq 配置缺少 .loc.wp"
fi

# 验证 DNS 解析（等待 dnsmasq 就绪）
sleep 2
dns_result=$(dig +short testdns.loc.wp @127.0.0.1 2>/dev/null) || true
if [[ "$dns_result" == *"127.0.0.1"* ]]; then
  pass "DNS 解析 .loc.wp -> 127.0.0.1"
else
  fail "DNS 解析失败 (got '$dns_result')"
fi

# 验证 nginx 正在运行
if sudo nginx -t 2>/dev/null; then
  pass "nginx 配置语法正确"
else
  fail "nginx 配置语法有误"
fi

# ─── Test 2: locwp add testsite ─────────────────────────
echo ""
echo -e "${YELLOW}=== Test 2: locwp add testsite ===${NC}"
add_rc=0; run_capture "$BINARY" add testsite --pass a23456 || add_rc=$?
add_out=$(cat "$TMPOUT")

assert_exit_code "$add_rc" 0 "add testsite 退出码为 0"
assert_contains "$add_out" "configured" "add 输出 configured"

# 验证文件系统
assert_file_exists "$LOCWP_HOME/sites/testsite/config.json" "config.json 存在"
assert_dir_exists "$LOCWP_HOME/sites/testsite/wordpress" "wordpress 目录存在"
assert_file_exists "$LOCWP_HOME/nginx/sites/testsite.conf" "nginx vhost 存在"
assert_file_exists "$LOCWP_HOME/php/testsite.conf" "FPM pool 配置存在"

# 验证 nginx symlink
nginx_link="$BREW_PREFIX/etc/nginx/servers/locwp-testsite.conf"
if [[ -L "$nginx_link" ]]; then
  pass "nginx symlink 存在"
else
  fail "nginx symlink 不存在 ($nginx_link)"
fi

# 验证 config.json 内容
cfg_name=$(python3 -c "import json; print(json.load(open('$LOCWP_HOME/sites/testsite/config.json'))['name'])")
cfg_domain=$(python3 -c "import json; print(json.load(open('$LOCWP_HOME/sites/testsite/config.json'))['domain'])")
cfg_db=$(python3 -c "import json; print(json.load(open('$LOCWP_HOME/sites/testsite/config.json'))['db_name'])")
assert_eq "$cfg_name" "testsite" "config name = testsite"
assert_eq "$cfg_domain" "testsite.loc.wp" "config domain = testsite.loc.wp"
assert_eq "$cfg_db" "wp_testsite" "config db_name = wp_testsite"

# 验证数据库存在
db_exists=$(mariadb -e "SHOW DATABASES LIKE 'wp_testsite'" -sN 2>/dev/null) || true
if [[ "$db_exists" == "wp_testsite" ]]; then
  pass "数据库 wp_testsite 存在"
else
  fail "数据库 wp_testsite 不存在 (got '$db_exists')"
fi

# 验证 WordPress 文件
assert_file_exists "$LOCWP_HOME/sites/testsite/wordpress/wp-config.php" "wp-config.php 存在"
assert_file_exists "$LOCWP_HOME/sites/testsite/wordpress/index.php" "WordPress index.php 存在"

# 等待服务就绪后验证 HTTPS
echo ""
echo "  等待服务就绪..."
sleep 3

# 验证 HTTPS 访问
assert_http_status "https://testsite.loc.wp" "200" "HTTPS 访问 testsite.loc.wp"

# ─── Test 3: 重复添加同名站点 ──────────────────────────
echo ""
echo -e "${YELLOW}=== Test 3: 重复添加同名站点 ===${NC}"
dup_rc=0; dup_out=$("$BINARY" add testsite --pass a23456 2>&1) || dup_rc=$?
assert_exit_code "$dup_rc" 1 "重复添加返回错误码"
assert_contains "$dup_out" "already exists" "错误消息包含 already exists"

# ─── Test 4: 无效站名 ──────────────────────────────────
echo ""
echo -e "${YELLOW}=== Test 4: 无效站名 ===${NC}"

invalid_names=("My.Site" "MyBlog" "-badname" "bad name" "bad_name" "UPPER")
for iname in "${invalid_names[@]}"; do
  inv_rc=0; inv_out=$("$BINARY" add "$iname" --no-start 2>&1) || inv_rc=$?
  if [[ $inv_rc -ne 0 ]]; then
    pass "拒绝无效名称 '$iname'"
  else
    fail "应拒绝无效名称 '$iname'"
  fi
done

# ─── Test 5: 连续添加多个站点 ───────────────────────────
echo ""
echo -e "${YELLOW}=== Test 5: 连续添加多个站点 ===${NC}"

add2_rc=0; run_capture "$BINARY" add blog --pass a23456 || add2_rc=$?
assert_exit_code "$add2_rc" 0 "add blog 退出码为 0"

add3_rc=0; run_capture "$BINARY" add shop --pass a23456 || add3_rc=$?
assert_exit_code "$add3_rc" 0 "add shop 退出码为 0"

# 验证所有站点都存在
assert_file_exists "$LOCWP_HOME/sites/blog/config.json" "blog config.json 存在"
assert_file_exists "$LOCWP_HOME/sites/shop/config.json" "shop config.json 存在"

# 验证多站点数据库
db2_exists=$(mariadb -e "SHOW DATABASES LIKE 'wp_blog'" -sN 2>/dev/null) || true
db3_exists=$(mariadb -e "SHOW DATABASES LIKE 'wp_shop'" -sN 2>/dev/null) || true
assert_eq "$db2_exists" "wp_blog" "数据库 wp_blog 存在"
assert_eq "$db3_exists" "wp_shop" "数据库 wp_shop 存在"

# 验证多站点 HTTPS
sleep 2
assert_http_status "https://blog.loc.wp" "200" "HTTPS 访问 blog.loc.wp"
assert_http_status "https://shop.loc.wp" "200" "HTTPS 访问 shop.loc.wp"

# ─── Test 6: locwp list ────────────────────────────────
echo ""
echo -e "${YELLOW}=== Test 6: locwp list ===${NC}"
list_out=$("$BINARY" list 2>&1) || true
echo "$list_out"
assert_contains "$list_out" "testsite" "list 显示 testsite"
assert_contains "$list_out" "blog" "list 显示 blog"
assert_contains "$list_out" "shop" "list 显示 shop"

# ─── Test 7: stop 和 start ──────────────────────────────
echo ""
echo -e "${YELLOW}=== Test 7: stop 和 start ===${NC}"

"$BINARY" stop testsite 2>&1 || true
sleep 1
# stop 后 vhost 应该被禁用
if [[ ! -f "$LOCWP_HOME/nginx/sites/testsite.conf" ]] && [[ -f "$LOCWP_HOME/nginx/sites/testsite.conf.disabled" ]]; then
  pass "stop 后 vhost 被禁用"
else
  fail "stop 后 vhost 状态异常"
fi

# stop 后 nginx symlink 应被移除
if [[ ! -L "$BREW_PREFIX/etc/nginx/servers/locwp-testsite.conf" ]]; then
  pass "stop 后 nginx symlink 已移除"
else
  fail "stop 后 nginx symlink 应被移除"
fi

"$BINARY" start testsite 2>&1 || true
sleep 2

# start 后 symlink 应重建
if [[ -L "$BREW_PREFIX/etc/nginx/servers/locwp-testsite.conf" ]]; then
  pass "start 后 nginx symlink 已重建"
else
  fail "start 后 nginx symlink 应重建"
fi

# start 后应可访问
assert_http_status "https://testsite.loc.wp" "200" "重新启动后 HTTPS 可访问"

# ─── Test 8: delete ─────────────────────────────────────
echo ""
echo -e "${YELLOW}=== Test 8: delete ===${NC}"

"$BINARY" delete shop 2>&1 || true
sleep 1

if [[ ! -d "$LOCWP_HOME/sites/shop" ]]; then
  pass "shop 目录已删除"
else
  fail "shop 目录应已删除"
fi

db_shop_after=$(mariadb -e "SHOW DATABASES LIKE 'wp_shop'" -sN 2>/dev/null) || true
if [[ -z "$db_shop_after" ]]; then
  pass "shop 数据库已删除"
else
  fail "shop 数据库应已删除"
fi

# 确认其他站点未受影响
assert_file_exists "$LOCWP_HOME/sites/testsite/config.json" "testsite 未受影响"
assert_file_exists "$LOCWP_HOME/sites/blog/config.json" "blog 未受影响"

# ─── Test 9: 带连字符的站名 ────────────────────────────
echo ""
echo -e "${YELLOW}=== Test 9: 带连字符的站名 ===${NC}"
add_hyphen_rc=0; run_capture "$BINARY" add my-site --pass a23456 || add_hyphen_rc=$?
assert_exit_code "$add_hyphen_rc" 0 "add my-site 退出码为 0"

cfg_db_hyphen=$(python3 -c "import json; print(json.load(open('$LOCWP_HOME/sites/my-site/config.json'))['db_name'])")
assert_eq "$cfg_db_hyphen" "wp_my_site" "连字符站名的 db_name 正确转换"
sleep 2
assert_http_status "https://my-site.loc.wp" "200" "HTTPS 访问 my-site.loc.wp"

# ─── 汇总 ──────────────────────────────────────────────
TOTAL_END=$(date +%s)
ELAPSED=$((TOTAL_END - TOTAL_START))

echo ""
echo "════════════════════════════════════════════════"
echo -e "  ${GREEN}PASS: $PASS${NC}  ${RED}FAIL: $FAIL${NC}  耗时: ${ELAPSED}s"
echo "════════════════════════════════════════════════"

if [[ $FAIL -gt 0 ]]; then
  echo ""
  echo -e "${RED}有 $FAIL 个测试失败！${NC}"
  exit 1
else
  echo ""
  echo -e "${GREEN}全部通过！${NC}"
fi
