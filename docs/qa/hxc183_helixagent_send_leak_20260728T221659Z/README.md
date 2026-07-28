# HXC-183 — helixagent GenerateStream goroutine + connection leak: captured RED/GREEN evidence

**Item:** HXC-183 (Bug, High) — "One AI provider still leaks a background worker when the caller stops listening"
**Fix commit:** `1352caa5`
**Defect sites:** `helix_code/internal/llm/providers/helixagent/helixagent.go` :600 (per-delta send), :625 (finish_reason send) — both cited line numbers CONFIRMED exact.
**Guard:** `helix_code/internal/llm/providers/helixagent/helixagent_send_leak_test.go`, driving the SHARED harness `helix_code/internal/llm/streamleak`.

All exit codes below were taken directly from the invoking shell, not inferred from log text.

## Root cause of the coverage gap (verified empirically, not assumed)

`providers/*` sub-packages import `internal/llm` for the shared Provider types, so a
`package llm` test file cannot import them back to drive them:

```
$ go test -run TestZZCycleProbe ./internal/llm
# dev.helix.code/internal/llm
package dev.helix.code/internal/llm
	imports dev.helix.code/internal/llm/providers/helixagent from zz_cycleprobe_test.go
	imports dev.helix.code/internal/llm from embeddings.go: import cycle not allowed in test
FAIL	dev.helix.code/internal/llm [setup failed]
```

That language-level wall — not oversight — is why the `905a0b0a` / `97d5ad2b` fan-out
stopped at the package boundary. The harness was therefore hoisted into a generic,
dependency-free package (`internal/llm/streamleak`) that imports nothing from
`dev.helix.code`, giving both sides the same seam. It required **no `go.mod` change**.

## §11.4.115 polarity matrix — all four cells

### 1. RED_MODE=1 on the UNFIXED artifact — defect must be PRESENT

```
$ RED_MODE=1 go test -count=1 -v -run TestGenerateStream_NoGoroutineLeak_OnCtxCancelWithoutDrain ./internal/llm/providers/helixagent
=== RUN   TestGenerateStream_NoGoroutineLeak_OnCtxCancelWithoutDrain
--- PASS: TestGenerateStream_NoGoroutineLeak_OnCtxCancelWithoutDrain (0.16s)
PASS
ok  	dev.helix.code/internal/llm/providers/helixagent	0.166s
RED_MODE1_EXIT=0
```

### 2. RED_MODE=0 (the STANDING GUARD) on the UNFIXED artifact — must FAIL

This is the load-bearing cell: it proves the guard can actually fail on broken code.

```
$ RED_MODE=0 go test -count=1 -v -run TestGenerateStream_NoGoroutineLeak_OnCtxCancelWithoutDrain ./internal/llm/providers/helixagent
=== RUN   TestGenerateStream_NoGoroutineLeak_OnCtxCancelWithoutDrain
    helixagent_send_leak_test.go:87:
        	Error Trace:	internal/llm/streamleak/streamleak.go:219
        	            				internal/llm/providers/helixagent/helixagent_send_leak_test.go:87
        	Error:      	Should be false
        	Messages:   	helixagent: GREEN failed: a goroutine is still parked in
        	            	dev.helix.code/internal/llm/providers/helixagent.(*Provider).GenerateStream's
        	            	closure (blocked in chan-send state) 1.5s after ctx cancellation with the
        	            	channel never drained — an unguarded blocking send leaks a goroutine (and the
        	            	open HTTP response body it holds) forever.
--- FAIL: TestGenerateStream_NoGoroutineLeak_OnCtxCancelWithoutDrain (1.66s)
FAIL	dev.helix.code/internal/llm/providers/helixagent	1.670s
RED_MODE0_UNFIXED_EXIT=1
```

### 3. RED_MODE=0 on the FIXED artifact — standing GREEN guard must PASS

```
$ go test -count=1 -v -run TestGenerateStream_NoGoroutineLeak_OnCtxCancelWithoutDrain ./internal/llm/providers/helixagent
=== RUN   TestGenerateStream_NoGoroutineLeak_OnCtxCancelWithoutDrain
--- PASS: TestGenerateStream_NoGoroutineLeak_OnCtxCancelWithoutDrain (1.66s)
PASS
ok  	dev.helix.code/internal/llm/providers/helixagent	1.668s
GREEN_EXIT=0
```

### 4. RED_MODE=1 on the FIXED artifact — must now FAIL (polarity confirmed)

```
$ RED_MODE=1 go test -count=1 -run TestGenerateStream_NoGoroutineLeak_OnCtxCancelWithoutDrain ./internal/llm/providers/helixagent
        	Error:      	Should be true
        	Messages:   	helixagent: RED expectation failed: no goroutine found parked in
        	            	...(*Provider).GenerateStream's closure (blocked in chan-send state)
        	            	within 1.5s of ctx cancellation on this artifact
FAIL	dev.helix.code/internal/llm/providers/helixagent	1.670s
RED_ON_FIXED_EXIT=1
```

## Regression sweep

```
$ go test -count=1 -race ./internal/llm
ok  	dev.helix.code/internal/llm	247.857s
LLM_RACE_EXIT=0

$ go test -count=1 ./internal/llm/providers/helixagent ./internal/llm/streamleak
ok  	dev.helix.code/internal/llm/providers/helixagent	1.679s
?   	dev.helix.code/internal/llm/streamleak	[no test files]
PKG_EXIT=0
```

Harness refactor proven behaviour-preserving — the 5 pre-existing guards
(anthropic / azure / gemini / groq / vertexai) pass identically before and after:

```
BEFORE refactor: ok  dev.helix.code/internal/llm  11.634s   BASELINE_EXIT=0
AFTER  refactor: ok  dev.helix.code/internal/llm  12.072s   REFACTOR_EXIT=0
```

## Scope check — was helixagent really the only one missed?

Zero unguarded sends in the sibling sub-packages, so the item's scope claim holds:

```
cerebras: 0    cohere: 0    huggingface: 0    replicate: 0    together: 0
```

## Honest gaps (§11.4.6)

- `internal/llm/streamleak` has **no test files of its own**. It is exercised transitively
  by all seven guards that call it, and its ability to fail is proven by cell 2 above, but
  it carries no dedicated unit test.
- The guard proves the goroutine no longer parks. It does **not** separately assert that
  `resp.Body` was closed; body closure is inferred from the `defer` now being reached.
