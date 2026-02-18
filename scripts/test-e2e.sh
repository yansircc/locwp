#!/usr/bin/env bash
# test-e2e.sh — automated end-to-end tests
# Runs locwp setup + add for real and verifies the full workflow
source "$(dirname "$0")/lib.sh"

# ─── Initialization ──────────────────────────────────────
echo -e "${YELLOW}=== Setting up sudo ===${NC}"
init_sudo
echo "  [ok] sudo NOPASSWD configured"

echo ""
echo -e "${YELLOW}=== Building latest binary ===${NC}"
build_binary || { echo -e "${RED}Build failed${NC}"; exit 1; }
echo "  [ok] Built $BINARY"

echo ""
echo -e "${YELLOW}=== Running Reset ===${NC}"
bash "$SCRIPT_DIR/reset.sh"
restore_sudo

# ─── Test 1: locwp setup ────────────────────────────────
echo ""
echo -e "${YELLOW}=== Test 1: locwp setup ===${NC}"
setup_rc=0; run_capture "$BINARY" setup || setup_rc=$?
setup_out=$(cat "$TMPOUT")

assert_exit_code "$setup_rc" 0 "setup exits with code 0"
assert_contains "$setup_out" "Setup complete" "setup outputs Setup complete"

assert_dir_exists "$LOCWP_HOME/ssl" "SSL directory exists"
assert_file_exists "$LOCWP_HOME/ssl/_wildcard.loc.wp.pem" "wildcard certificate exists"
assert_file_exists "$LOCWP_HOME/ssl/_wildcard.loc.wp-key.pem" "wildcard certificate key exists"
assert_file_exists "/etc/resolver/wp" "DNS resolver config exists"
assert_file_exists "/etc/sudoers.d/locwp" "sudoers config exists"

if grep -q 'address=/.loc.wp/127.0.0.1' "$BREW_PREFIX/etc/dnsmasq.conf"; then
  pass "dnsmasq config contains .loc.wp"
else
  fail "dnsmasq config missing .loc.wp"
fi

sleep 2
dns_result=$(dig +short testdns.loc.wp @127.0.0.1 2>/dev/null) || true
if [[ "$dns_result" == *"127.0.0.1"* ]]; then
  pass "DNS resolves .loc.wp -> 127.0.0.1"
else
  fail "DNS resolution failed (got '$dns_result')"
fi

if sudo nginx -t 2>/dev/null; then
  pass "nginx config syntax valid"
else
  fail "nginx config syntax invalid"
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
assert_file_exists "$LOCWP_HOME/nginx/sites/testsite.conf" "nginx vhost exists"
assert_file_exists "$LOCWP_HOME/php/testsite.conf" "FPM pool config exists"

nginx_link="$BREW_PREFIX/etc/nginx/servers/locwp-testsite.conf"
if [[ -L "$nginx_link" ]]; then
  pass "nginx symlink exists"
else
  fail "nginx symlink missing ($nginx_link)"
fi

cfg_name=$(python3 -c "import json; print(json.load(open('$LOCWP_HOME/sites/testsite/config.json'))['name'])")
cfg_domain=$(python3 -c "import json; print(json.load(open('$LOCWP_HOME/sites/testsite/config.json'))['domain'])")
cfg_db=$(python3 -c "import json; print(json.load(open('$LOCWP_HOME/sites/testsite/config.json'))['db_name'])")
assert_eq "$cfg_name" "testsite" "config name = testsite"
assert_eq "$cfg_domain" "testsite.loc.wp" "config domain = testsite.loc.wp"
assert_eq "$cfg_db" "wp_testsite" "config db_name = wp_testsite"

db_exists=$(mariadb -e "SHOW DATABASES LIKE 'wp_testsite'" -sN 2>/dev/null) || true
if [[ "$db_exists" == "wp_testsite" ]]; then
  pass "database wp_testsite exists"
else
  fail "database wp_testsite missing (got '$db_exists')"
fi

assert_file_exists "$LOCWP_HOME/sites/testsite/wordpress/wp-config.php" "wp-config.php exists"
assert_file_exists "$LOCWP_HOME/sites/testsite/wordpress/index.php" "WordPress index.php exists"

echo ""
echo "  Waiting for services..."
sleep 3
assert_http_status "https://testsite.loc.wp" "200" "HTTPS access testsite.loc.wp"

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

db2_exists=$(mariadb -e "SHOW DATABASES LIKE 'wp_blog'" -sN 2>/dev/null) || true
db3_exists=$(mariadb -e "SHOW DATABASES LIKE 'wp_shop'" -sN 2>/dev/null) || true
assert_eq "$db2_exists" "wp_blog" "database wp_blog exists"
assert_eq "$db3_exists" "wp_shop" "database wp_shop exists"

sleep 2
assert_http_status "https://blog.loc.wp" "200" "HTTPS access blog.loc.wp"
assert_http_status "https://shop.loc.wp" "200" "HTTPS access shop.loc.wp"

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
if [[ ! -f "$LOCWP_HOME/nginx/sites/testsite.conf" ]] && [[ -f "$LOCWP_HOME/nginx/sites/testsite.conf.disabled" ]]; then
  pass "vhost disabled after stop"
else
  fail "vhost state unexpected after stop"
fi

if [[ ! -L "$BREW_PREFIX/etc/nginx/servers/locwp-testsite.conf" ]]; then
  pass "nginx symlink removed after stop"
else
  fail "nginx symlink should be removed after stop"
fi

"$BINARY" start testsite 2>&1 || true
sleep 2

if [[ -L "$BREW_PREFIX/etc/nginx/servers/locwp-testsite.conf" ]]; then
  pass "nginx symlink restored after start"
else
  fail "nginx symlink should be restored after start"
fi

assert_http_status "https://testsite.loc.wp" "200" "HTTPS accessible after restart"

# ─── Test 8: delete ─────────────────────────────────────
echo ""
echo -e "${YELLOW}=== Test 8: delete ===${NC}"

"$BINARY" delete shop 2>&1 || true
sleep 1

if [[ ! -d "$LOCWP_HOME/sites/shop" ]]; then
  pass "shop directory deleted"
else
  fail "shop directory should be deleted"
fi

db_shop_after=$(mariadb -e "SHOW DATABASES LIKE 'wp_shop'" -sN 2>/dev/null) || true
if [[ -z "$db_shop_after" ]]; then
  pass "shop database deleted"
else
  fail "shop database should be deleted"
fi

assert_file_exists "$LOCWP_HOME/sites/testsite/config.json" "testsite unaffected"
assert_file_exists "$LOCWP_HOME/sites/blog/config.json" "blog unaffected"

# ─── Test 9: site name with hyphens ─────────────────────
echo ""
echo -e "${YELLOW}=== Test 9: site name with hyphens ===${NC}"
add_hyphen_rc=0; run_capture "$BINARY" add my-site --pass a23456 || add_hyphen_rc=$?
assert_exit_code "$add_hyphen_rc" 0 "add my-site exits with code 0"

cfg_db_hyphen=$(python3 -c "import json; print(json.load(open('$LOCWP_HOME/sites/my-site/config.json'))['db_name'])")
assert_eq "$cfg_db_hyphen" "wp_my_site" "hyphenated site name db_name correctly converted"
sleep 2
assert_http_status "https://my-site.loc.wp" "200" "HTTPS access my-site.loc.wp"

# ─── Summary ────────────────────────────────────────────
print_summary
