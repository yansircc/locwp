#!/usr/bin/env bash
# test-e2e.sh — automated end-to-end tests
source "$(dirname "$0")/lib.sh"

# ─── Initialization ──────────────────────────────────────
echo ""
echo -e "${YELLOW}=== Building latest binary ===${NC}"
build_binary || { echo -e "${RED}Build failed${NC}"; exit 1; }
echo "  [ok] Built $BINARY"

echo ""
echo -e "${YELLOW}=== Running Reset ===${NC}"
bash "$SCRIPT_DIR/reset.sh"

# ─── Test 1: locwp setup ────────────────────────────────
echo ""
echo -e "${YELLOW}=== Test 1: locwp setup ===${NC}"
setup_rc=0; run_capture "$BINARY" setup || setup_rc=$?
setup_out=$(cat "$TMPOUT")

assert_exit_code "$setup_rc" 0 "setup exits with code 0"
assert_contains "$setup_out" "Setup complete" "setup outputs Setup complete"
assert_dir_exists "$LOCWP_HOME/caddy/sites" "Caddy sites directory exists"

# ─── Test 2: locwp add ──────────────────────────────────
echo ""
echo -e "${YELLOW}=== Test 2: locwp add ===${NC}"
add_rc=0; run_capture "$BINARY" add --pass a23456 || add_rc=$?
add_out=$(cat "$TMPOUT")

assert_exit_code "$add_rc" 0 "add exits with code 0"
assert_contains "$add_out" "configured" "add outputs configured"
assert_contains "$add_out" "10001" "add shows port 10001"

assert_file_exists "$LOCWP_HOME/sites/10001/config.json" "config.json exists"
assert_dir_exists "$LOCWP_HOME/sites/10001/wordpress" "wordpress directory exists"
assert_file_exists "$LOCWP_HOME/caddy/sites/10001.caddy" "Caddy site config exists"
assert_file_exists "$LOCWP_HOME/php/10001.conf" "FPM pool config exists"

# Verify SQLite
assert_file_exists "$LOCWP_HOME/sites/10001/wordpress/wp-content/db.php" "SQLite db.php drop-in exists"
assert_file_exists "$LOCWP_HOME/sites/10001/wordpress/wp-content/database/.ht.sqlite" "SQLite database file exists"

echo ""
echo "  Waiting for services..."
sleep 3
assert_http_status "$(site_url 10001)" "200" "HTTP access port 10001"

# ─── Test 3: locwp add (second site) ────────────────────
echo ""
echo -e "${YELLOW}=== Test 3: locwp add (second site) ===${NC}"
add2_rc=0; run_capture "$BINARY" add --pass a23456 || add2_rc=$?
assert_exit_code "$add2_rc" 0 "add second site exits with code 0"

assert_file_exists "$LOCWP_HOME/sites/10002/config.json" "second site config.json exists"
sleep 2
assert_http_status "$(site_url 10002)" "200" "HTTP access port 10002"

# ─── Test 4: locwp list ─────────────────────────────────
echo ""
echo -e "${YELLOW}=== Test 4: locwp list ===${NC}"
list_out=$("$BINARY" list 2>&1) || true
echo "$list_out"
assert_contains "$list_out" "10001" "list shows port 10001"
assert_contains "$list_out" "10002" "list shows port 10002"
assert_contains "$list_out" "PORT" "list header shows PORT"

# ─── Test 5: stop and start ─────────────────────────────
echo ""
echo -e "${YELLOW}=== Test 5: stop and start ===${NC}"

"$BINARY" stop 10001 2>&1 || true
sleep 1
if [[ ! -f "$LOCWP_HOME/caddy/sites/10001.caddy" ]] && [[ -f "$LOCWP_HOME/caddy/sites/10001.caddy.disabled" ]]; then
  pass "caddy conf disabled after stop"
else
  fail "caddy conf state unexpected after stop"
fi

"$BINARY" start 10001 2>&1 || true
sleep 2

if [[ -f "$LOCWP_HOME/caddy/sites/10001.caddy" ]]; then
  pass "caddy conf restored after start"
else
  fail "caddy conf should be restored after start"
fi

assert_http_status "$(site_url 10001)" "200" "HTTP accessible after restart"

# ─── Test 6: delete ─────────────────────────────────────
echo ""
echo -e "${YELLOW}=== Test 6: delete ===${NC}"

"$BINARY" delete 10002 2>&1 || true
sleep 1

if [[ ! -d "$LOCWP_HOME/sites/10002" ]]; then
  pass "site 10002 directory deleted"
else
  fail "site 10002 directory should be deleted"
fi

assert_file_exists "$LOCWP_HOME/sites/10001/config.json" "site 10001 unaffected"

# ─── Test 7: wp command ─────────────────────────────────
echo ""
echo -e "${YELLOW}=== Test 7: wp command ===${NC}"

rc=0; wp_out=$("$BINARY" wp 10001 -- option get siteurl 2>&1) || rc=$?
if [[ $rc -eq 0 ]]; then
  assert_contains "$wp_out" "localhost:10001" "wp option get returns correct URL"
else
  fail "wp option get failed (exit $rc)"
fi

# ─── Summary ────────────────────────────────────────────
print_summary
