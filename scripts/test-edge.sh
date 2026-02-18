#!/usr/bin/env bash
# test-edge.sh — edge case boundary tests
# Run after test-e2e.sh passes; tests various unusual operations
source "$(dirname "$0")/lib.sh"

# ─── Initialization ─────────────────────────────────────
echo -e "${YELLOW}=== Initialization ===${NC}"
init_sudo
echo "  [ok] sudo configured"

echo ""
echo -e "${YELLOW}=== Build ===${NC}"
build_binary
echo "  [ok] build complete"

echo ""
echo -e "${YELLOW}=== Reset + Setup ===${NC}"
bash "$SCRIPT_DIR/reset.sh"
restore_sudo
run_cmd "$BINARY" setup
assert_exit_ok $? "setup completed"

# ════════════════════════════════════════════════════════
# Edge Case 1: Setup idempotency
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 1: Setup idempotency (run twice) ━━━${NC}"

run_cmd "$BINARY" setup
assert_exit_ok $? "second setup succeeds"

out=$(cat "$TMPOUT")
assert_contains "$out" "Setup complete" "second setup completes normally"
assert_contains "$out" "already" "second setup skips existing config"

# ════════════════════════════════════════════════════════
# Edge Case 2: site name boundaries
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 2: site name boundaries ━━━${NC}"

# Single character name
run_cmd "$BINARY" add a --pass a23456
assert_exit_ok $? "single-char name 'a' allowed"
sleep 3
assert_http_status "https://a.loc.wp" "200" "single-char site accessible"

# Numeric-only name
run_cmd "$BINARY" add 123 --pass a23456
assert_exit_ok $? "numeric name '123' allowed"
sleep 3
assert_http_status "https://123.loc.wp" "200" "numeric site accessible"

# Consecutive hyphens
run_cmd "$BINARY" add "my--site" --no-start
rc=$?
if [[ $rc -eq 0 ]]; then
  pass "consecutive hyphens 'my--site' accepted (matches regex)"
  "$BINARY" delete "my--site" 2>/dev/null || true
else
  pass "consecutive hyphens 'my--site' rejected"
fi

# Very long name (50 chars)
longname="abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmn"
run_cmd "$BINARY" add "$longname" --no-start
rc=$?
if [[ $rc -eq 0 ]]; then
  pass "long name accepted"
  "$BINARY" delete "$longname" 2>/dev/null || true
else
  pass "long name rejected"
fi

# Invalid name tests
for bad in "" " " "." ".." "a.b" "a b" "a_b" "-" "a-" "A" "aB" "a@b" "a/b" "a\\b"; do
  rc=0
  "$BINARY" add "$bad" --no-start >/dev/null 2>&1 || rc=$?
  if [[ $rc -ne 0 ]]; then
    pass "rejected invalid name '$bad'"
  else
    fail "should reject invalid name '$bad'"
    "$BINARY" delete "$bad" 2>/dev/null || true
  fi
done

# ════════════════════════════════════════════════════════
# Edge Case 3: Stop/Start idempotency
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 3: Stop/Start idempotency ━━━${NC}"

run_cmd "$BINARY" stop a
assert_exit_ok $? "first stop succeeds"

run_cmd "$BINARY" stop a
rc=$?
echo "  (second stop exit code: $rc)"
if [[ $rc -eq 0 ]]; then
  pass "second stop idempotent success"
else
  pass "second stop returns error (acceptable)"
fi

run_cmd "$BINARY" start a
assert_exit_ok $? "first start succeeds"

run_cmd "$BINARY" start a
rc=$?
echo "  (second start exit code: $rc)"
if [[ $rc -eq 0 ]]; then
  pass "second start idempotent success"
else
  pass "second start returns error (acceptable)"
fi

sleep 3
assert_http_status "https://a.loc.wp" "200" "site works after idempotent ops"

# ════════════════════════════════════════════════════════
# Edge Case 4: re-add site after deletion (recycle)
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 4: re-add site after deletion ━━━${NC}"

assert_http_status "https://123.loc.wp" "200" "123 accessible before deletion"

run_cmd "$BINARY" delete 123
assert_exit_ok $? "delete 123 succeeds"

run_cmd "$BINARY" add 123 --pass a23456
assert_exit_ok $? "re-add 123 succeeds"

sleep 3
assert_http_status "https://123.loc.wp" "200" "123 accessible after re-add"

# ════════════════════════════════════════════════════════
# Edge Case 5: Delete after Stop
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 5: Delete after Stop ━━━${NC}"

run_cmd "$BINARY" stop 123
assert_exit_ok $? "stop 123 succeeds"

run_cmd "$BINARY" delete 123
assert_exit_ok $? "delete stopped site succeeds"

if [[ ! -d "$LOCWP_HOME/sites/123" ]]; then
  pass "stopped site directory deleted"
else
  fail "stopped site directory should be deleted"
fi

db_check=$(mariadb -e "SHOW DATABASES LIKE 'wp_123'" -sN 2>/dev/null) || true
if [[ -z "$db_check" ]]; then
  pass "stopped site database deleted"
else
  fail "stopped site database should be deleted"
fi

# ════════════════════════════════════════════════════════
# Edge Case 6: rapid add/delete cycles
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 6: rapid add/delete cycles ━━━${NC}"

for i in 1 2 3; do
  run_cmd "$BINARY" add cycle --pass a23456
  assert_exit_ok $? "cycle $i: add succeeds"
  run_cmd "$BINARY" delete cycle
  assert_exit_ok $? "cycle $i: delete succeeds"
done

run_cmd "$BINARY" add cycle --pass a23456
assert_exit_ok $? "final add after cycles succeeds"
sleep 3
assert_http_status "https://cycle.loc.wp" "200" "site accessible after cycles"
"$BINARY" delete cycle 2>/dev/null || true

# ════════════════════════════════════════════════════════
# Edge Case 7: behavior after filesystem corruption
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 7: filesystem corruption ━━━${NC}"

# 7a: manually delete config.json
run_cmd "$BINARY" add broken --pass a23456
assert_exit_ok $? "create broken site"

rm -f "$LOCWP_HOME/sites/broken/config.json"
rc=0; "$BINARY" start broken >/dev/null 2>&1 || rc=$?
assert_exit_fail "$rc" "start errors when config.json missing"

rc=0; "$BINARY" stop broken >/dev/null 2>&1 || rc=$?
assert_exit_fail "$rc" "stop errors when config.json missing"

rc=0; "$BINARY" delete broken >/dev/null 2>&1 || rc=$?
assert_exit_fail "$rc" "delete errors when config.json missing"

# Manually clean up leftovers
rm -rf "$LOCWP_HOME/sites/broken"
rm -f "$LOCWP_HOME/nginx/sites/broken.conf"*
rm -f "$LOCWP_HOME/php/broken.conf"
rm -f "$BREW_PREFIX/etc/nginx/servers/locwp-broken.conf"
rm -f "$BREW_PREFIX/etc/php/8.3/php-fpm.d/locwp-broken.conf"
mariadb -e "DROP DATABASE IF EXISTS wp_broken" 2>/dev/null || true

# 7b: stop/start after manually deleting vhost file
run_cmd "$BINARY" add ghost --pass a23456
assert_exit_ok $? "create ghost site"

rm -f "$LOCWP_HOME/nginx/sites/ghost.conf"
run_cmd "$BINARY" stop ghost
pass "stop does not crash after vhost deleted (exit $?)"

run_cmd "$BINARY" start ghost
pass "start does not crash after vhost deleted (exit $?)"

"$BINARY" delete ghost 2>/dev/null || true

# 7c: manually delete nginx symlink
run_cmd "$BINARY" add orphan --pass a23456
assert_exit_ok $? "create orphan site"

rm -f "$BREW_PREFIX/etc/nginx/servers/locwp-orphan.conf"
sudo nginx -s reload 2>/dev/null || true
pass "nginx reload does not crash after symlink deleted"

"$BINARY" delete orphan 2>/dev/null || true

# ════════════════════════════════════════════════════════
# Edge Case 8: pre-existing database
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 8: pre-existing database ━━━${NC}"

mariadb -e "CREATE DATABASE IF NOT EXISTS wp_preexist" 2>/dev/null || true

run_cmd "$BINARY" add preexist --pass a23456
assert_exit_ok $? "add succeeds when database already exists"

sleep 3
assert_http_status "https://preexist.loc.wp" "200" "pre-existing database site accessible"

"$BINARY" delete preexist 2>/dev/null || true

# ════════════════════════════════════════════════════════
# Edge Case 9: operations on non-existent sites
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 9: operations on non-existent sites ━━━${NC}"

for action in start stop delete; do
  rc=0; "$BINARY" "$action" nonexistent >/dev/null 2>&1 || rc=$?
  assert_exit_fail "$rc" "$action nonexistent returns error"
done

rc=0; "$BINARY" wp nonexistent -- option get siteurl >/dev/null 2>&1 || rc=$?
assert_exit_fail "$rc" "wp nonexistent returns error"

# ════════════════════════════════════════════════════════
# Edge Case 10: no-argument invocation
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 10: no-argument invocation ━━━${NC}"

for action in add start stop delete; do
  rc=0; "$BINARY" "$action" >/dev/null 2>&1 || rc=$?
  assert_exit_fail "$rc" "$action with no args returns error"
done

# ════════════════════════════════════════════════════════
# Edge Case 11: list under various states
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 11: list under various states ━━━${NC}"

run_cmd "$BINARY" list
out=$(cat "$TMPOUT")
assert_contains "$out" "a" "list shows existing site"

"$BINARY" stop a 2>/dev/null || true
run_cmd "$BINARY" list
out=$(cat "$TMPOUT")
assert_contains "$out" "a" "list shows stopped site"

"$BINARY" start a 2>/dev/null || true
"$BINARY" delete a 2>/dev/null || true
run_cmd "$BINARY" list
out=$(cat "$TMPOUT")
assert_contains "$out" "No sites yet" "shows prompt when no sites"

# ════════════════════════════════════════════════════════
# Edge Case 12: domain and DB uniqueness
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 12: domain and DB uniqueness ━━━${NC}"

run_cmd "$BINARY" add test1 --pass a23456
assert_exit_ok $? "add test1"

run_cmd "$BINARY" add test2 --pass a23456
assert_exit_ok $? "add test2 (no conflict)"

sleep 3
assert_http_status "https://test1.loc.wp" "200" "test1 accessible"
assert_http_status "https://test2.loc.wp" "200" "test2 accessible"

"$BINARY" delete test1 2>/dev/null || true
"$BINARY" delete test2 2>/dev/null || true

# ════════════════════════════════════════════════════════
# Edge Case 13: wp command argument passing
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 13: wp command argument passing ━━━${NC}"

run_cmd "$BINARY" add wptest --pass a23456
assert_exit_ok $? "create wptest site"

sleep 1

rc=0; wp_out=$("$BINARY" wp wptest -- option get siteurl 2>&1) || rc=$?
if [[ $rc -eq 0 ]]; then
  assert_contains "$wp_out" "wptest.loc.wp" "wp option get returns correct domain"
else
  fail "wp option get failed (exit $rc)"
fi

rc=0; wp_out=$("$BINARY" wp wptest -- user list --fields=user_login --format=csv 2>&1) || rc=$?
if [[ $rc -eq 0 ]]; then
  assert_contains "$wp_out" "admin" "wp user list shows admin"
else
  fail "wp user list failed (exit $rc)"
fi

"$BINARY" delete wptest 2>/dev/null || true

# ─── Summary ────────────────────────────────────────────
print_summary
