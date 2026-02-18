#!/usr/bin/env bash
# E2E tests for locwp CLI
set -euo pipefail

PASS=0
FAIL=0
BINARY=""
export LOCWP_HOME=""

pass() { ((PASS++)); echo "  PASS: $1"; }
fail() { ((FAIL++)); echo "  FAIL: $1"; }
assert_eq() {
  if [[ "$1" == "$2" ]]; then pass "$3"; else fail "$3 (got '$1', want '$2')"; fi
}
assert_contains() {
  if echo "$1" | grep -q "$2"; then pass "$3"; else fail "$3 (output missing '$2')"; fi
}
assert_file_exists() {
  if [[ -f "$1" ]]; then pass "$2"; else fail "$2 ($1 not found)"; fi
}
assert_file_not_exists() {
  if [[ ! -f "$1" ]]; then pass "$2"; else fail "$2 ($1 should not exist)"; fi
}

setup() {
  echo "Building locwp..."
  BINARY="$(cd "$(dirname "$0")/.." && pwd)/locwp"
  (cd "$(dirname "$0")/.." && go build -o locwp .)
  export LOCWP_HOME="$(mktemp -d)"
  echo "Using temp dir: $LOCWP_HOME"
}

cleanup() {
  if [[ -n "$LOCWP_HOME" && -d "$LOCWP_HOME" ]]; then
    rm -rf "$LOCWP_HOME"
  fi
}
trap cleanup EXIT

# ─── Test: help ───────────────────────────────────────────
test_help() {
  echo ""
  echo "=== test_help ==="
  local out
  out=$("$BINARY" --help 2>&1)
  assert_contains "$out" "locwp" "root help shows locwp"
  assert_contains "$out" "add" "root help shows add command"
  assert_contains "$out" "list" "root help shows list command"
  assert_contains "$out" "start" "root help shows start command"
  assert_contains "$out" "stop" "root help shows stop command"
  assert_contains "$out" "delete" "root help shows delete command"
}

# ─── Test: list with no sites ─────────────────────────────
test_list_empty() {
  echo ""
  echo "=== test_list_empty ==="
  local out
  out=$("$BINARY" list 2>&1)
  assert_contains "$out" "No sites yet" "list with no sites shows hint"
}

# ─── Test: add with defaults (--no-start) ─────────────────
test_add_defaults() {
  echo ""
  echo "=== test_add_defaults ==="
  local out
  out=$("$BINARY" add testsite --no-start 2>&1)

  assert_contains "$out" "configured" "add outputs configured"
  assert_contains "$out" "8.3" "add shows default PHP"

  assert_file_exists "$LOCWP_HOME/sites/testsite/config.json" "config.json created"
  assert_file_exists "$LOCWP_HOME/nginx/sites/testsite.conf" "nginx vhost created"
  assert_file_exists "$LOCWP_HOME/php/testsite.conf" "FPM pool created"
  assert_file_exists "$LOCWP_HOME/sites/testsite/.pawl/workflows/provision.json" "pawl provision workflow created"

  local name php
  name=$(python3 -c "import json; print(json.load(open('$LOCWP_HOME/sites/testsite/config.json'))['name'])")
  php=$(python3 -c "import json; print(json.load(open('$LOCWP_HOME/sites/testsite/config.json'))['php'])")
  assert_eq "$name" "testsite" "config name is testsite"
  assert_eq "$php" "8.3" "config php defaults to 8.3"
}

# ─── Test: add with flags ────────────────────────────────
test_add_with_flags() {
  echo ""
  echo "=== test_add_with_flags ==="
  local out
  out=$("$BINARY" add blog --port 9090 --php 8.2 --no-start 2>&1)

  assert_contains "$out" "9090" "add shows specified port"
  assert_contains "$out" "8.2" "add shows specified PHP"

  local port php
  port=$(python3 -c "import json; print(json.load(open('$LOCWP_HOME/sites/blog/config.json'))['port'])")
  php=$(python3 -c "import json; print(json.load(open('$LOCWP_HOME/sites/blog/config.json'))['php'])")
  assert_eq "$port" "9090" "config port is 9090"
  assert_eq "$php" "8.2" "config php is 8.2"
}

# ─── Test: add duplicate should fail ─────────────────────
test_add_duplicate() {
  echo ""
  echo "=== test_add_duplicate ==="
  set +e
  local out rc
  out=$("$BINARY" add testsite --no-start 2>&1)
  rc=$?
  set -e
  if [[ $rc -ne 0 ]]; then pass "duplicate add returns error"; else fail "duplicate add should fail"; fi
  assert_contains "$out" "already exists" "duplicate add error message"
}

# ─── Test: list with sites ────────────────────────────────
test_list_with_sites() {
  echo ""
  echo "=== test_list_with_sites ==="
  local out
  out=$("$BINARY" list 2>&1)
  assert_contains "$out" "testsite" "list shows testsite"
  assert_contains "$out" "blog" "list shows blog"
}

# ─── Test: ls alias ───────────────────────────────────────
test_ls_alias() {
  echo ""
  echo "=== test_ls_alias ==="
  local out
  out=$("$BINARY" ls 2>&1)
  assert_contains "$out" "testsite" "ls alias works"
}

# ─── Test: stop ───────────────────────────────────────────
test_stop() {
  echo ""
  echo "=== test_stop ==="
  "$BINARY" stop testsite 2>&1 || true
  assert_file_not_exists "$LOCWP_HOME/nginx/sites/testsite.conf" "vhost disabled after stop"
  assert_file_exists "$LOCWP_HOME/nginx/sites/testsite.conf.disabled" "vhost .disabled exists"
}

# ─── Test: stop idempotent ────────────────────────────────
test_stop_idempotent() {
  echo ""
  echo "=== test_stop_idempotent ==="
  "$BINARY" stop testsite 2>&1 || true
  assert_file_exists "$LOCWP_HOME/nginx/sites/testsite.conf.disabled" "stop is idempotent"
}

# ─── Test: start ──────────────────────────────────────────
test_start() {
  echo ""
  echo "=== test_start ==="
  "$BINARY" start testsite 2>&1 || true
  assert_file_exists "$LOCWP_HOME/nginx/sites/testsite.conf" "vhost re-enabled after start"
  assert_file_not_exists "$LOCWP_HOME/nginx/sites/testsite.conf.disabled" "disabled file removed"
}

# ─── Test: start idempotent ───────────────────────────────
test_start_idempotent() {
  echo ""
  echo "=== test_start_idempotent ==="
  "$BINARY" start testsite 2>&1 || true
  assert_file_exists "$LOCWP_HOME/nginx/sites/testsite.conf" "start is idempotent"
}

# ─── Test: delete (no confirmation) ──────────────────────
test_delete() {
  echo ""
  echo "=== test_delete ==="
  "$BINARY" delete blog 2>&1 || true

  if [[ ! -d "$LOCWP_HOME/sites/blog" ]]; then
    pass "blog directory removed"
  else
    fail "blog directory should be removed"
  fi
  assert_file_not_exists "$LOCWP_HOME/nginx/sites/blog.conf" "blog nginx conf removed"
  assert_file_not_exists "$LOCWP_HOME/php/blog.conf" "blog FPM pool removed"
  assert_file_exists "$LOCWP_HOME/sites/testsite/config.json" "testsite still exists"
}

# ─── Test: rm alias ───────────────────────────────────────
test_rm_alias() {
  echo ""
  echo "=== test_rm_alias ==="
  "$BINARY" add rmtest --no-start 2>&1
  "$BINARY" rm rmtest 2>&1 || true
  if [[ ! -d "$LOCWP_HOME/sites/rmtest" ]]; then
    pass "rm alias works"
  else
    fail "rm alias should delete site"
  fi
}

# ─── Test: error on nonexistent site ──────────────────────
test_nonexistent() {
  echo ""
  echo "=== test_nonexistent ==="
  set +e
  "$BINARY" start nonexistent 2>&1; [[ $? -ne 0 ]] && pass "start nonexistent errors" || fail "should error"
  "$BINARY" stop nonexistent 2>&1; [[ $? -ne 0 ]] && pass "stop nonexistent errors" || fail "should error"
  "$BINARY" delete nonexistent 2>&1; [[ $? -ne 0 ]] && pass "delete nonexistent errors" || fail "should error"
  set -e
}

# ─── Test: add requires name ─────────────────────────────
test_add_no_args() {
  echo ""
  echo "=== test_add_no_args ==="
  set +e
  "$BINARY" add 2>&1; [[ $? -ne 0 ]] && pass "add without args errors" || fail "should error"
  set -e
}

# ─── Run ──────────────────────────────────────────────────
setup

test_help
test_list_empty
test_add_defaults
test_add_with_flags
test_add_duplicate
test_list_with_sites
test_ls_alias
test_stop
test_stop_idempotent
test_start
test_start_idempotent
test_delete
test_rm_alias
test_nonexistent
test_add_no_args

echo ""
echo "════════════════════════════════════"
echo "  PASS: $PASS  FAIL: $FAIL"
echo "════════════════════════════════════"

if [[ $FAIL -gt 0 ]]; then exit 1; fi
