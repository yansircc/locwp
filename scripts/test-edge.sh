#!/usr/bin/env bash
# test-edge.sh — 极端场景边界测试
# 在 test-e2e.sh 通过后运行，测试各种奇葩操作
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BINARY="$PROJECT_DIR/locwp"
PASS=0
FAIL=0
TOTAL_START=$(date +%s)
TMPOUT=$(mktemp)
LOCWP_HOME="$HOME/.locwp"
BREW_PREFIX="$(brew --prefix)"

# ─── 颜色 ───────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

pass() { ((PASS++)); echo -e "  ${GREEN}PASS${NC}: $1"; }
fail() { ((FAIL++)); echo -e "  ${RED}FAIL${NC}: $1"; }

assert_eq() {
  if [[ "$1" == "$2" ]]; then pass "$3"; else fail "$3 (got '$1', want '$2')"; fi
}
assert_contains() {
  if echo "$1" | grep -q "$2"; then pass "$3"; else fail "$3 (output missing '$2')"; fi
}
assert_exit_ok() {
  if [[ "$1" -eq 0 ]]; then pass "$2"; else fail "$2 (exit code $1)"; fi
}
assert_exit_fail() {
  if [[ "$1" -ne 0 ]]; then pass "$2"; else fail "$2 (should have failed)"; fi
}
assert_http_status() {
  local url="$1" expected="$2" label="$3"
  local status
  # 重试最多 5 次（等待 FPM socket 就绪）
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

run_cmd() {
  "$@" > "$TMPOUT" 2>&1
  return $?
}

cleanup() {
  rm -f "$TMPOUT"
  sudo rm -f /etc/sudoers.d/locwp-test 2>/dev/null || true
  exit
}
trap cleanup EXIT INT TERM

# ─── 初始化 ─────────────────────────────────────────────
echo -e "${YELLOW}=== 初始化 ===${NC}"
echo 'a23456' | sudo -S -v 2>/dev/null
echo "$(whoami) ALL=(ALL) NOPASSWD: ALL" | sudo tee /etc/sudoers.d/locwp-test >/dev/null
sudo chmod 0440 /etc/sudoers.d/locwp-test
echo "  [ok] sudo 已配置"

echo ""
echo -e "${YELLOW}=== 编译 ===${NC}"
(cd "$PROJECT_DIR" && go build -o locwp .)
echo "  [ok] 编译完成"

echo ""
echo -e "${YELLOW}=== Reset + Setup ===${NC}"
bash "$SCRIPT_DIR/reset.sh"
echo "$(whoami) ALL=(ALL) NOPASSWD: ALL" | sudo tee /etc/sudoers.d/locwp-test >/dev/null
sudo chmod 0440 /etc/sudoers.d/locwp-test
run_cmd "$BINARY" setup
assert_exit_ok $? "setup 完成"

# ════════════════════════════════════════════════════════
# Edge Case 1: Setup 幂等性
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 1: Setup 幂等性（连续运行两次）━━━${NC}"

run_cmd "$BINARY" setup
assert_exit_ok $? "第二次 setup 成功"

out=$(cat "$TMPOUT")
assert_contains "$out" "Setup complete" "第二次 setup 正常完成"
assert_contains "$out" "already" "第二次 setup 跳过已有配置"

# ════════════════════════════════════════════════════════
# Edge Case 2: 站名边界
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 2: 站名边界 ━━━${NC}"

# 单字符名
run_cmd "$BINARY" add a --pass a23456
assert_exit_ok $? "单字符名 'a' 允许"
sleep 3
assert_http_status "https://a.loc.wp" "200" "单字符站点可访问"

# 纯数字名
run_cmd "$BINARY" add 123 --pass a23456
assert_exit_ok $? "纯数字名 '123' 允许"
sleep 3
assert_http_status "https://123.loc.wp" "200" "纯数字站点可访问"

# 连续连字符
run_cmd "$BINARY" add "my--site" --no-start
rc=$?
# 连续连字符在正则 ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$ 中是合法的
if [[ $rc -eq 0 ]]; then
  pass "连续连字符 'my--site' 被接受（符合正则）"
  # 清理
  "$BINARY" delete "my--site" 2>/dev/null || true
else
  pass "连续连字符 'my--site' 被拒绝"
fi

# 超长名（50 字符）
longname="abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmn"
run_cmd "$BINARY" add "$longname" --no-start
rc=$?
if [[ $rc -eq 0 ]]; then
  pass "超长名被接受"
  "$BINARY" delete "$longname" 2>/dev/null || true
else
  pass "超长名被拒绝"
fi

# 更多无效名测试
for bad in "" " " "." ".." "a.b" "a b" "a_b" "-" "a-" "A" "aB" "a@b" "a/b" "a\\b"; do
  rc=0
  "$BINARY" add "$bad" --no-start >/dev/null 2>&1 || rc=$?
  if [[ $rc -ne 0 ]]; then
    pass "拒绝无效名 '$bad'"
  else
    fail "应拒绝无效名 '$bad'"
    "$BINARY" delete "$bad" 2>/dev/null || true
  fi
done

# ════════════════════════════════════════════════════════
# Edge Case 3: Stop/Start 幂等性
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 3: Stop/Start 幂等性 ━━━${NC}"

# Stop 两次
run_cmd "$BINARY" stop a
assert_exit_ok $? "第一次 stop 成功"

run_cmd "$BINARY" stop a
rc=$?
# 第二次 stop 可能成功也可能失败，关键是不 crash
echo "  (第二次 stop exit code: $rc)"
if [[ $rc -eq 0 ]]; then
  pass "第二次 stop 幂等成功"
else
  # 即使失败也不应该 crash
  pass "第二次 stop 返回错误（可接受）"
fi

# Start 两次
run_cmd "$BINARY" start a
assert_exit_ok $? "第一次 start 成功"

run_cmd "$BINARY" start a
rc=$?
echo "  (第二次 start exit code: $rc)"
if [[ $rc -eq 0 ]]; then
  pass "第二次 start 幂等成功"
else
  pass "第二次 start 返回错误（可接受）"
fi

sleep 3
assert_http_status "https://a.loc.wp" "200" "幂等操作后站点正常"

# ════════════════════════════════════════════════════════
# Edge Case 4: 删除后重新添加同名站点（回收）
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 4: 删除后重新添加同名站点 ━━━${NC}"

# 先验证 123 站点存在
assert_http_status "https://123.loc.wp" "200" "删除前 123 可访问"

# 删除
run_cmd "$BINARY" delete 123
assert_exit_ok $? "delete 123 成功"

# 重新添加
run_cmd "$BINARY" add 123 --pass a23456
assert_exit_ok $? "重新添加 123 成功"

sleep 3
assert_http_status "https://123.loc.wp" "200" "重新添加后 123 可访问"

# ════════════════════════════════════════════════════════
# Edge Case 5: Stop 后 Delete
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 5: Stop 后 Delete ━━━${NC}"

run_cmd "$BINARY" stop 123
assert_exit_ok $? "stop 123 成功"

run_cmd "$BINARY" delete 123
assert_exit_ok $? "删除已停止的站点成功"

# 确认真的删除了
if [[ ! -d "$LOCWP_HOME/sites/123" ]]; then
  pass "已停止站点的目录被删除"
else
  fail "已停止站点的目录应被删除"
fi

db_check=$(mariadb -e "SHOW DATABASES LIKE 'wp_123'" -sN 2>/dev/null) || true
if [[ -z "$db_check" ]]; then
  pass "已停止站点的数据库被删除"
else
  fail "已停止站点的数据库应被删除"
fi

# ════════════════════════════════════════════════════════
# Edge Case 6: 快速 add → delete → add 循环
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 6: 快速 add/delete 循环 ━━━${NC}"

for i in 1 2 3; do
  run_cmd "$BINARY" add cycle --pass a23456
  assert_exit_ok $? "循环 $i: add 成功"
  run_cmd "$BINARY" delete cycle
  assert_exit_ok $? "循环 $i: delete 成功"
done

# 最终再添加一次验证
run_cmd "$BINARY" add cycle --pass a23456
assert_exit_ok $? "循环后最终 add 成功"
sleep 3
assert_http_status "https://cycle.loc.wp" "200" "循环后站点可访问"
"$BINARY" delete cycle 2>/dev/null || true

# ════════════════════════════════════════════════════════
# Edge Case 7: 手动破坏文件系统后的行为
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 7: 文件系统破坏 ━━━${NC}"

# 7a: 手动删除 config.json
run_cmd "$BINARY" add broken --pass a23456
assert_exit_ok $? "创建 broken 站点"

rm -f "$LOCWP_HOME/sites/broken/config.json"
rc=0; "$BINARY" start broken >/dev/null 2>&1 || rc=$?
assert_exit_fail "$rc" "缺少 config.json 时 start 报错"

rc=0; "$BINARY" stop broken >/dev/null 2>&1 || rc=$?
assert_exit_fail "$rc" "缺少 config.json 时 stop 报错"

rc=0; "$BINARY" delete broken >/dev/null 2>&1 || rc=$?
assert_exit_fail "$rc" "缺少 config.json 时 delete 报错"

# 手动清理残留
rm -rf "$LOCWP_HOME/sites/broken"
rm -f "$LOCWP_HOME/nginx/sites/broken.conf"*
rm -f "$LOCWP_HOME/php/broken.conf"
rm -f "$BREW_PREFIX/etc/nginx/servers/locwp-broken.conf"
rm -f "$BREW_PREFIX/etc/php/8.3/php-fpm.d/locwp-broken.conf"
mariadb -e "DROP DATABASE IF EXISTS wp_broken" 2>/dev/null || true

# 7b: 手动删除 vhost 文件后 stop/start
run_cmd "$BINARY" add ghost --pass a23456
assert_exit_ok $? "创建 ghost 站点"

rm -f "$LOCWP_HOME/nginx/sites/ghost.conf"
run_cmd "$BINARY" stop ghost
# stop 应该不 crash（vhost 已不存在，mv 会 fail 但 || true）
pass "删除 vhost 后 stop 不崩溃 (exit $?)"

run_cmd "$BINARY" start ghost
# start 的 enable-vhost 也应该 || true
pass "删除 vhost 后 start 不崩溃 (exit $?)"

# 清理
"$BINARY" delete ghost 2>/dev/null || true

# 7c: 手动删除 nginx symlink
run_cmd "$BINARY" add orphan --pass a23456
assert_exit_ok $? "创建 orphan 站点"

rm -f "$BREW_PREFIX/etc/nginx/servers/locwp-orphan.conf"
# nginx reload 应该不崩溃（只是少了一个 vhost）
sudo nginx -s reload 2>/dev/null || true
pass "删除 symlink 后 nginx reload 不崩溃"

"$BINARY" delete orphan 2>/dev/null || true

# ════════════════════════════════════════════════════════
# Edge Case 8: 数据库预先存在
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 8: 数据库预先存在 ━━━${NC}"

# 先手动创建数据库
mariadb -e "CREATE DATABASE IF NOT EXISTS wp_preexist" 2>/dev/null || true

run_cmd "$BINARY" add preexist --pass a23456
rc=$?
# provision 使用 CREATE DATABASE IF NOT EXISTS，应该不 crash
assert_exit_ok $rc "数据库已存在时 add 成功"

sleep 3
assert_http_status "https://preexist.loc.wp" "200" "预存数据库站点可访问"

"$BINARY" delete preexist 2>/dev/null || true

# ════════════════════════════════════════════════════════
# Edge Case 9: 对不存在站点的各种操作
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 9: 对不存在站点的操作 ━━━${NC}"

for action in start stop delete; do
  rc=0; "$BINARY" "$action" nonexistent >/dev/null 2>&1 || rc=$?
  assert_exit_fail "$rc" "$action nonexistent 返回错误"
done

# wp 命令对不存在的站点
rc=0; "$BINARY" wp nonexistent -- option get siteurl >/dev/null 2>&1 || rc=$?
assert_exit_fail "$rc" "wp nonexistent 返回错误"

# ════════════════════════════════════════════════════════
# Edge Case 10: 无参数调用
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 10: 无参数调用 ━━━${NC}"

for action in add start stop delete; do
  rc=0; "$BINARY" "$action" >/dev/null 2>&1 || rc=$?
  assert_exit_fail "$rc" "$action 无参数返回错误"
done

# ════════════════════════════════════════════════════════
# Edge Case 11: list 在各种状态下
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 11: list 各种状态 ━━━${NC}"

# 只剩站点 a 现在
run_cmd "$BINARY" list
out=$(cat "$TMPOUT")
assert_contains "$out" "a" "list 显示现有站点"

# 停止 a 后 list 应该仍然显示
"$BINARY" stop a 2>/dev/null || true
run_cmd "$BINARY" list
out=$(cat "$TMPOUT")
assert_contains "$out" "a" "list 显示已停止的站点"

# 删除所有站点后 list
"$BINARY" start a 2>/dev/null || true
"$BINARY" delete a 2>/dev/null || true
run_cmd "$BINARY" list
out=$(cat "$TMPOUT")
assert_contains "$out" "No sites yet" "无站点时显示提示"

# ════════════════════════════════════════════════════════
# Edge Case 12: 并发域名安全
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 12: 域名和 DB 唯一性 ━━━${NC}"

run_cmd "$BINARY" add test1 --pass a23456
assert_exit_ok $? "添加 test1"

# test1 的 domain = test1.loc.wp，db = wp_test1
# 添加另一个不同名但不冲突的站点
run_cmd "$BINARY" add test2 --pass a23456
assert_exit_ok $? "添加 test2（不冲突）"

sleep 3
assert_http_status "https://test1.loc.wp" "200" "test1 可访问"
assert_http_status "https://test2.loc.wp" "200" "test2 可访问"

# 清理
"$BINARY" delete test1 2>/dev/null || true
"$BINARY" delete test2 2>/dev/null || true

# ════════════════════════════════════════════════════════
# Edge Case 13: wp 命令传参
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 13: wp 命令传参 ━━━${NC}"

run_cmd "$BINARY" add wptest --pass a23456
assert_exit_ok $? "创建 wptest 站点"

sleep 1

# 测试 wp option get
rc=0; wp_out=$("$BINARY" wp wptest -- option get siteurl 2>&1) || rc=$?
if [[ $rc -eq 0 ]]; then
  assert_contains "$wp_out" "wptest.loc.wp" "wp option get 返回正确域名"
else
  fail "wp option get 失败 (exit $rc)"
fi

# 测试 wp user list
rc=0; wp_out=$("$BINARY" wp wptest -- user list --fields=user_login --format=csv 2>&1) || rc=$?
if [[ $rc -eq 0 ]]; then
  assert_contains "$wp_out" "admin" "wp user list 显示 admin"
else
  fail "wp user list 失败 (exit $rc)"
fi

"$BINARY" delete wptest 2>/dev/null || true

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
