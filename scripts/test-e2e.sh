#!/usr/bin/env bash
# test-e2e.sh — automated end-to-end tests
# Runs locwp setup + add for real and verifies the full workflow
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
assert_file_exists "$BREW_PREFIX/etc/Caddyfile" "Caddyfile exists"

if grep -q 'auto_https off' "$BREW_PREFIX/etc/Caddyfile"; then
  pass "Caddyfile disables auto_https"
else
  fail "Caddyfile missing auto_https off"
fi

# ─── Test 2: locwp add testsite ─────────────────────────
echo ""
echo -e "${YELLOW}=== Test 2: locwp add testsite ===${NC}"
add_rc=0; run_capture "$BINARY" add testsite --pass a23456 || add_rc=$?
add_out=$(cat "$TMPOUT")

assert_exit_code "$add_rc" 0 "add testsite exits with code 0"
assert_contains "$add_out" "configured" "add outputs configured"

assert_file_exists "$LOCWP_HOME/sites/testsite/config.json" "config.json exists"
assert_dir_exists "$LOCWP_HOME/sites/testsite/wordpress" "wordpress directory exists"
assert_file_exists "$LOCWP_HOME/caddy/sites/testsite.caddy" "Caddy site config exists"
assert_file_exists "$LOCWP_HOME/php/testsite.conf" "FPM pool config exists"

cfg_name=$(python3 -c "import json; print(json.load(open('$LOCWP_HOME/sites/testsite/config.json'))['name'])")
cfg_port=$(site_port testsite)
assert_eq "$cfg_name" "testsite" "config name = testsite"
assert_eq "$cfg_port" "10001" "config port = 10001"

# Verify SQLite database was created
assert_file_exists "$LOCWP_HOME/sites/testsite/wordpress/wp-content/db.php" "SQLite db.php drop-in exists"
assert_dir_exists "$LOCWP_HOME/sites/testsite/wordpress/wp-content/database" "SQLite database directory exists"
assert_file_exists "$LOCWP_HOME/sites/testsite/wordpress/wp-content/database/.ht.sqlite" "SQLite database file exists"

assert_file_exists "$LOCWP_HOME/sites/testsite/wordpress/wp-config.php" "wp-config.php exists"
assert_file_exists "$LOCWP_HOME/sites/testsite/wordpress/index.php" "WordPress index.php exists"

echo ""
echo "  Waiting for services..."
sleep 3
assert_http_status "$(site_url testsite)" "200" "HTTP access testsite"

# ─── Test 3: duplicate site name ────────────────────────
echo ""
echo -e "${YELLOW}=== Test 3: duplicate site name ===${NC}"
dup_rc=0; dup_out=$("$BINARY" add testsite --pass a23456 2>&1) || dup_rc=$?
assert_exit_code "$dup_rc" 1 "duplicate add returns error code"
assert_contains "$dup_out" "already exists" "error message contains already exists"

# ─── Test 4: invalid site names ─────────────────────────
echo ""
echo -e "${YELLOW}=== Test 4: invalid site names ===${NC}"

invalid_names=("My.Site" "MyBlog" "-badname" "bad name" "bad_name" "UPPER")
for iname in "${invalid_names[@]}"; do
  inv_rc=0; inv_out=$("$BINARY" add "$iname" --no-start 2>&1) || inv_rc=$?
  if [[ $inv_rc -ne 0 ]]; then
    pass "rejected invalid name '$iname'"
  else
    fail "should reject invalid name '$iname'"
  fi
done

# ─── Test 5: add multiple sites consecutively ───────────
echo ""
echo -e "${YELLOW}=== Test 5: add multiple sites ===${NC}"

add2_rc=0; run_capture "$BINARY" add blog --pass a23456 || add2_rc=$?
assert_exit_code "$add2_rc" 0 "add blog exits with code 0"

add3_rc=0; run_capture "$BINARY" add shop --pass a23456 || add3_rc=$?
assert_exit_code "$add3_rc" 0 "add shop exits with code 0"

assert_file_exists "$LOCWP_HOME/sites/blog/config.json" "blog config.json exists"
assert_file_exists "$LOCWP_HOME/sites/shop/config.json" "shop config.json exists"

# Verify ports are sequential
blog_port=$(site_port blog)
shop_port=$(site_port shop)
assert_eq "$blog_port" "10002" "blog port = 10002"
assert_eq "$shop_port" "10003" "shop port = 10003"

sleep 2
assert_http_status "$(site_url blog)" "200" "HTTP access blog"
assert_http_status "$(site_url shop)" "200" "HTTP access shop"

# ─── Test 6: locwp list ────────────────────────────────
echo ""
echo -e "${YELLOW}=== Test 6: locwp list ===${NC}"
list_out=$("$BINARY" list 2>&1) || true
echo "$list_out"
assert_contains "$list_out" "testsite" "list shows testsite"
assert_contains "$list_out" "blog" "list shows blog"
assert_contains "$list_out" "shop" "list shows shop"

# ─── Test 7: stop and start ─────────────────────────────
echo ""
echo -e "${YELLOW}=== Test 7: stop and start ===${NC}"

"$BINARY" stop testsite 2>&1 || true
sleep 1
if [[ ! -f "$LOCWP_HOME/caddy/sites/testsite.caddy" ]] && [[ -f "$LOCWP_HOME/caddy/sites/testsite.caddy.disabled" ]]; then
  pass "caddy conf disabled after stop"
else
  fail "caddy conf state unexpected after stop"
fi

"$BINARY" start testsite 2>&1 || true
sleep 2

if [[ -f "$LOCWP_HOME/caddy/sites/testsite.caddy" ]]; then
  pass "caddy conf restored after start"
else
  fail "caddy conf should be restored after start"
fi

assert_http_status "$(site_url testsite)" "200" "HTTP accessible after restart"

# ─── Test 8: delete ─────────────────────────────────────
echo ""
echo -e "${YELLOW}=== Test 8: delete ===${NC}"

"$BINARY" delete shop 2>&1 || true
sleep 1

if [[ ! -d "$LOCWP_HOME/sites/shop" ]]; then
  pass "shop directory deleted (including SQLite DB)"
else
  fail "shop directory should be deleted"
fi

assert_file_exists "$LOCWP_HOME/sites/testsite/config.json" "testsite unaffected"
assert_file_exists "$LOCWP_HOME/sites/blog/config.json" "blog unaffected"

# ─── Test 9: site name with hyphens ─────────────────────
echo ""
echo -e "${YELLOW}=== Test 9: site name with hyphens ===${NC}"
add_hyphen_rc=0; run_capture "$BINARY" add my-site --pass a23456 || add_hyphen_rc=$?
assert_exit_code "$add_hyphen_rc" 0 "add my-site exits with code 0"
sleep 2
assert_http_status "$(site_url my-site)" "200" "HTTP access my-site"

# ─── Summary ────────────────────────────────────────────
print_summary
