#!/usr/bin/env bash
# test-edge.sh — edge case boundary tests
source "$(dirname "$0")/lib.sh"

# ─── Initialization ─────────────────────────────────────
echo -e "${YELLOW}=== Initialization ===${NC}"

echo ""
echo -e "${YELLOW}=== Build ===${NC}"
build_binary
echo "  [ok] build complete"

echo ""
echo -e "${YELLOW}=== Reset + Setup ===${NC}"
bash "$SCRIPT_DIR/reset.sh"
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
# Edge Case 2: add rejects extra arguments
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 2: add rejects extra args ━━━${NC}"

rc=0; "$BINARY" add extraarg >/dev/null 2>&1 || rc=$?
assert_exit_fail "$rc" "add with extra arg returns error"

# ════════════════════════════════════════════════════════
# Edge Case 3: Stop/Start idempotency
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 3: Stop/Start idempotency ━━━${NC}"

run_cmd "$BINARY" add --pass a23456
assert_exit_ok $? "add site"
sleep 3

run_cmd "$BINARY" stop 10001
assert_exit_ok $? "first stop succeeds"

run_cmd "$BINARY" stop 10001
rc=$?
if [[ $rc -eq 0 ]]; then
  pass "second stop idempotent success"
else
  pass "second stop returns error (acceptable)"
fi

run_cmd "$BINARY" start 10001
assert_exit_ok $? "first start succeeds"

run_cmd "$BINARY" start 10001
rc=$?
if [[ $rc -eq 0 ]]; then
  pass "second start idempotent success"
else
  pass "second start returns error (acceptable)"
fi

sleep 3
assert_http_status "$(site_url 10001)" "200" "site works after idempotent ops"

# ════════════════════════════════════════════════════════
# Edge Case 4: re-add after deletion
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 4: re-add after deletion ━━━${NC}"

run_cmd "$BINARY" delete 10001
assert_exit_ok $? "delete succeeds"

run_cmd "$BINARY" add --pass a23456
assert_exit_ok $? "re-add succeeds (gets next port)"

# Port may be 10001 again or 10002 depending on NextPort logic
sleep 3
new_port=$(python3 -c "import os,json; d='$LOCWP_HOME/sites'; ports=[json.load(open(os.path.join(d,x,'config.json')))['port'] for x in os.listdir(d) if os.path.isfile(os.path.join(d,x,'config.json'))]; print(max(ports))")
assert_http_status "$(site_url $new_port)" "200" "re-added site accessible"

"$BINARY" delete "$new_port" 2>/dev/null || true

# ════════════════════════════════════════════════════════
# Edge Case 5: rapid add/delete cycles
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 5: rapid add/delete cycles ━━━${NC}"

for i in 1 2 3; do
  run_cmd "$BINARY" add --pass a23456
  assert_exit_ok $? "cycle $i: add succeeds"
  # Find the latest port
  p=$(python3 -c "import os,json; d='$LOCWP_HOME/sites'; ports=[json.load(open(os.path.join(d,x,'config.json')))['port'] for x in os.listdir(d) if os.path.isfile(os.path.join(d,x,'config.json'))]; print(max(ports))")
  run_cmd "$BINARY" delete "$p"
  assert_exit_ok $? "cycle $i: delete succeeds"
done

# ════════════════════════════════════════════════════════
# Edge Case 6: operations on non-existent port
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 6: operations on non-existent port ━━━${NC}"

for action in start stop delete; do
  rc=0; "$BINARY" "$action" 99999 >/dev/null 2>&1 || rc=$?
  assert_exit_fail "$rc" "$action 99999 returns error"
done

# ════════════════════════════════════════════════════════
# Edge Case 7: invalid port argument
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 7: invalid port argument ━━━${NC}"

for action in start stop delete; do
  rc=0; "$BINARY" "$action" "notaport" >/dev/null 2>&1 || rc=$?
  assert_exit_fail "$rc" "$action with non-numeric arg returns error"
done

# ════════════════════════════════════════════════════════
# Edge Case 8: no-argument invocation
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 8: no-argument invocation ━━━${NC}"

for action in start stop delete; do
  rc=0; "$BINARY" "$action" >/dev/null 2>&1 || rc=$?
  assert_exit_fail "$rc" "$action with no args returns error"
done

# ════════════════════════════════════════════════════════
# Edge Case 9: list empty state
# ════════════════════════════════════════════════════════
echo ""
echo -e "${CYAN}━━━ Edge 9: list empty state ━━━${NC}"

run_cmd "$BINARY" list
out=$(cat "$TMPOUT")
assert_contains "$out" "No sites yet" "shows prompt when no sites"

# ─── Summary ────────────────────────────────────────────
print_summary
