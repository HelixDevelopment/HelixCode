# HXC-249 — captured verification: the complete-validation runner's false green, and its fix

Captured 2026-08-10T18:47:13Z from the live tree, main repo HEAD `be5d56be`.
helix_agent submodule HEAD: `d2d70206`

## 1. The three fix commits exist in helix_agent

```
1a55c8aa fix(validation): the complete-validation runner reported PASSED on tests that never ran
886f72c0 test(validation): the readiness helper had no self-test — three mutations inverted its verdicts silently
ad3b5590 fix(validation): `grep -c … || echo "0"` rendered "0 0 packages" — the footgun 1a55c8aa documented, still live in the file it fixed
```

## 2. The fixed readiness + required-flag path is present in the live runner

scripts/run_complete_validation.sh, `run_integration_tests()`:

```bash
    # actually bound", not "something answered :8100". The old check used bare
    # `curl -s`, which exits 0 on the 404 a foreign occupant returns, so it
    # passed without ever reaching this server.
    log_info "Waiting for HelixAgent to be ready..."
    local HELIX_URL
    if ! HELIX_URL="$(helix_wait_ready "$HELIX_PID" 60)"; then
        log_error "HelixAgent failed to become ready (check logs/helixagent.log)"
        kill $HELIX_PID 2>/dev/null || true
        return 1
    fi
    # Point the tests at the server we just verified. Without this they keep
    # their :8100 default and assert against whatever holds that port.
    export HELIXAGENT_URL="$HELIX_URL"

    log_success "HelixAgent is running (PID: $HELIX_PID) at $HELIX_URL"

    # Check providers endpoint
    log_info "Checking providers endpoint..."
    curl -sf "$HELIX_URL/v1/providers" | jq '.' > logs/providers_list.json 2>/dev/null || true

    # This script STARTED the server and just proved its identity, so a test
    # that cannot find it is a real failure here, not an honest environment
    # skip. Without this, the guarded tests skip and `go test` still exits 0.
    export HELIXAGENT_REQUIRED=1
```

## 3. Zero-executed-tests is now fatal (a suite that ran nothing is not a suite that passed)

```bash
    log_info "Running integration tests..."
    local integration_exit=0
    nice -n 19 ionice -c 3 go test ./tests/integration/... -v -timeout 10m 2>&1 \
        | tee logs/test_integration.log | tail -50
    integration_exit=${PIPESTATUS[0]}

    # A suite that ran nothing is not a suite that passed: `go test` exits 0
    # for an all-skipped package, so exit status alone cannot tell "everything
    # passed" from "nothing executed".
    if ! helix_assert_tests_executed logs/test_integration.log "Integration tests"; then
        log_error "Integration tests executed ZERO test cases — nothing was validated"
        kill $HELIX_PID 2>/dev/null || true
        return 1
    fi

```
