#!/usr/bin/env bash
# lib.sh — shared test library
# Usage: source "$(dirname "$0")/lib.sh"

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[1]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BINARY="$PROJECT_DIR/locwp"
PASS=0
FAIL=0
TOTAL_START=$(date +%s)
TMPOUT=$(mktemp)
LOCWP_HOME="$HOME/.locwp"
BREW_PREFIX="$(brew --prefix)"

# ─── Colors ──────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# ─── Assertion functions ─────────────────────────────────
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
assert_exit_ok() {
  if [[ "$1" -eq 0 ]]; then pass "$2"; else fail "$2 (exit code $1)"; fi
}
assert_exit_fail() {
  if [[ "$1" -ne 0 ]]; then pass "$2"; else fail "$2 (should have failed)"; fi
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

# ─── Command execution ───────────────────────────────────
# Run command and capture output to $TMPOUT (stdout+stderr)
run_cmd() {
  "$@" > "$TMPOUT" 2>&1
  return $?
}

# Run command and capture output to $TMPOUT, also print to terminal
run_capture() {
  "$@" > "$TMPOUT" 2>&1
  local rc=$?
  cat "$TMPOUT"
  return $rc
}

# ─── Lifecycle ───────────────────────────────────────────
cleanup() {
  rm -f "$TMPOUT"
  sudo rm -f /etc/sudoers.d/locwp-test 2>/dev/null || true
  exit
}
trap cleanup EXIT INT TERM

# Initialize sudo
init_sudo() {
  echo 'a23456' | sudo -S -v 2>/dev/null
  if [[ $? -ne 0 ]]; then
    echo -e "${RED}sudo password incorrect, exiting${NC}"
    exit 1
  fi
  echo "$(whoami) ALL=(ALL) NOPASSWD: ALL" | sudo tee /etc/sudoers.d/locwp-test >/dev/null
  sudo chmod 0440 /etc/sudoers.d/locwp-test
}

# Restore sudo (reset.sh removes locwp-test)
restore_sudo() {
  echo "$(whoami) ALL=(ALL) NOPASSWD: ALL" | sudo tee /etc/sudoers.d/locwp-test >/dev/null
  sudo chmod 0440 /etc/sudoers.d/locwp-test
}

# Build binary
build_binary() {
  (cd "$PROJECT_DIR" && go build -o locwp .)
}

# Print test summary
print_summary() {
  local total_end elapsed
  total_end=$(date +%s)
  elapsed=$((total_end - TOTAL_START))
  echo ""
  echo "════════════════════════════════════════════════"
  echo -e "  ${GREEN}PASS: $PASS${NC}  ${RED}FAIL: $FAIL${NC}  Elapsed: ${elapsed}s"
  echo "════════════════════════════════════════════════"
  if [[ $FAIL -gt 0 ]]; then
    echo ""
    echo -e "${RED}$FAIL test(s) failed!${NC}"
    exit 1
  else
    echo ""
    echo -e "${GREEN}All passed!${NC}"
  fi
}
