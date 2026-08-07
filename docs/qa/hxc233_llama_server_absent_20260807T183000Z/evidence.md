# HXC-233 investigation + fix — evidence

**Scope:** `helix_code` meta-repo, submodule `submodules/helix_llm`
(module `github.com/HelixDevelopment/HelixLLM`).
**Method:** `superpowers:systematic-debugging` Phase 1 (root-cause first,
no speculative patch before FACT). Platform was live/reachable throughout
(6/6 units, gateway on :8443).

## 1. What expects `llama-server`, with file:line

- `submodules/helix_llm/cmd/helixllm/main.go:213-263` (pre-fix line numbers) —
  the gateway's `main()`, gated by `cfg.LLM.LlamaServerEmbed`
  (`HELIX_LLAMA_SERVER_EMBEDDED`, **default `true`** —
  `internal/shared/config/config.go:76`), builds a
  `brain.LlamaServerConfig` and calls `llamaSrv.Start(ctx)`.
- `submodules/helix_llm/internal/brain/server.go:94-138` — `LlamaServer.Start`
  spawns `exec.CommandContext(ctx, binary, ...)` where `binary` is
  `cfg.BinaryPath` if set, else the literal string `"llama-server"`
  resolved via `$PATH`. Pre-fix, `main.go` never set `BinaryPath`, so
  resolution was always PATH-only with no override available.

## 2. What capability is actually lost

**More than the ticket's original text assumed.** With no cloud provider
keys configured in this deployment's `.env` (confirmed: grep for
`ANTHROPIC|OPENAI|CHUTES|OPENROUTER|HUGGINGFACE|NVIDIA|CEREBRAS|SAMBANOVA|TOGETHER`
against `.env` returns nothing), the `llamacpp` provider — pointed at the
embedded server's `127.0.0.1:8080` — is the Brain's **only** registered
provider (`internal/brain/brain.go:64-70`: `llamacpp` is registered
whenever `LlamaCppURL != ""`, which it always is), and it is also the
`fallback` chain's explicit "local safety net" (`cmd/helixllm/main.go:339`
comment: *"llamacpp always placed last as the local safety net"*).

Live pre-fix reproduction (see `02_pre_fix_live_curl.txt`):

- `GET /v1/models` → `{"object":"list","data":null}` (zero models loaded)
- `POST /v1/chat/completions` → **HTTP 500**, `"brain error: all providers
  exhausted, last error: llamacpp: send request: Post
  \"https://127.0.0.1:8080\": tls: failed to verify certificate: x509:
  cannot validate certificate for 127.0.0.1 because it doesn't contain
  any IP SANs"`

So the finding is not "a local backend is unavailable, nothing else
affected" — it is **every single completion request through this gateway
fails**, because nothing else is configured to catch the fall-through.
The TLS-handshake wording is itself a symptom of a second, distinct bug
(see §4.1): `main.go` was pointing the "local" route at 127.0.0.1:8080
even though the embedded server never started, and *something unrelated*
answers on that port on this host (traced to a rootless-podman
port-forwarder for one of the other infra containers, not to
`helixllm-coder`, which uses host networking and only binds :18434 —
`podman inspect helixllm-coder` confirms `NetworkMode=host`,
`podman top helixllm-coder` shows exactly one process, `llama-server`
on `--port 18434`, nothing on 8080).

`helixllm-coder` (Qwen3-Coder-30B, systemd unit
`helixllm-coder.service`, `active exited`/`RemainAfterExit=yes`) is a
**separate, already-working** local-model backend — its own llama-server
binary lives *inside its container image*, launched directly by
`scripts/boot_coder_cdi.sh`, completely independent of the gateway's
embedded-server code path. It does not currently help: the gateway never
routes to it (`HELIX_LLM_LOCAL_RPC_PORT` defaults to `50052`, and gets
overridden to `LlamaServerPort` (8080) whenever an embed is *attempted* —
never to 18434 by any existing code path).

## 3. Required / optional / dead — verdict

**Required-but-currently-absent, not dead.** `HELIX_LLAMA_SERVER_EMBEDDED`
defaults to `true` and, in this deployment's actual configuration (no
cloud keys), is the *only* completion path — genuinely required for the
gateway to serve any request at all, not merely a nice-to-have local
option. It is not a dead reference: the code path is live, reachable, and
exercised on every boot (confirmed via `18_boot_warning_census.txt` in
the prior live-boot evidence and reproduced again pre-fix here). §11.4.124
does not apply (nothing is being removed) and §11.4.112 does not apply
(this is not a proven structural impossibility — the binary can be
installed, or the gateway can be pointed at an already-running
OpenAI-compatible local server).

The *download* of a model for the embedded server to serve, however, is
genuinely wasteful whenever the binary cannot be resolved — that half is
fixed (§4.2).

## 4. Fix

Both changes committed in the `helix_llm` submodule, commit
`8990df4d62bbabdaed2b97f411b6953ac7b2bde3` (see
`03_fix_commit_stat.txt` / `04_fix_commit_full_diff.txt`). **Neither
installs anything or changes what capability exists on this host** — both
make the existing absence honest and non-wasteful, per the task's
explicit instruction not to bluff a fix that doesn't change what the
software can do.

### 4.1 Stop misrouting every request at a server that never started

`cmd/helixllm/main.go`: pre-fix, `llamaSrv` (a non-nil `*brain.LlamaServer`
struct pointer, allocated by `NewLlamaServer` *before* `Start` is ever
called) was tested with a bare `if llamaSrv != nil` **after** the
Start/WaitReady block, regardless of whether `Start()` actually returned
an error. `NewLlamaServer` unconditionally returns a non-nil pointer, so
this check was always true whenever an embed was attempted — including
when `cmd.Start()` failed synchronously because the binary does not
exist. The override
(`cfg.LLM.LocalRPCHost = "127.0.0.1"; cfg.LLM.LocalRPCPort =
cfg.LLM.LlamaServerPort`) therefore fired even with **no process
running**, and the `llamacpp` provider went on to attempt real HTTP
requests against a port where nothing the gateway spawned was listening
— producing the confusing TLS-handshake failure captured in §2 instead
of a clean "no provider available" error.

Fix: on a genuine `Start()` failure, `llamaSrv` is now reset to `nil`
before the override check, so the override only fires when a real
process is confirmed running. A `Start()` success followed by a
`WaitReady` timeout still overrides (the process exists and may still
finish loading) — that behaviour is unchanged.

### 4.2 Stop wasting the 37-minute download when the binary can't be found

Added `brain.ResolveLlamaServerBinary(explicitPath string) (string,
error)` (`internal/brain/server.go`) — a pure, side-effect-free
`exec.LookPath` pre-flight mirroring `LlamaServer.Start`'s own
`"" -> "llama-server"` fallback. `main.go` now calls it **before** the
model auto-download loop; when it fails, both the download and the
`Start()` attempt are skipped for that run, with one clear log line
naming the exact remediation options (install the binary, set
`HELIX_LLAMA_SERVER_BINARY_PATH`, or set
`HELIX_LLAMA_SERVER_EMBEDDED=false`).

Added `LlamaServerBinaryPath` /
`HELIX_LLAMA_SERVER_BINARY_PATH` config field
(`internal/shared/config/config.go`), wired into
`LlamaServerConfig.BinaryPath` — pre-fix this struct field existed but
**nothing ever set it**, so there was no way for an operator to point
HelixLLM at an installed-but-off-PATH `llama-server` without editing the
Go source. This is the concrete mechanism the ticket's own text asked
for ("making sure the service can find it").

### 4.3 Tests

`internal/brain/server_test.go`: four new unit tests —
`TestResolveLlamaServerBinary_{ExplicitPath,PATHLookup}_{Found,NotFound}`
— reproduce both the explicit-path and default-PATH-lookup success/failure
combinations deterministically (each test sets its own scratch `$PATH` /
`t.TempDir()`, never depending on the host's actual PATH state). The
`_PATHLookup_NotFound` case pins the exact HXC-233 condition (llama-server
genuinely absent from every PATH directory) as a permanent regression
guard.

Full submodule suite, post-fix (`05_post_fix_go_test_all.txt`): all
packages `ok`. `go build ./...`
(`06_post_fix_go_build_all.txt`): exit 0.

### 4.4 What was deliberately NOT done

- **Live redeployment of the fixed binary was attempted and stopped at
  the operator's explicit rejection of the tool call** that would have
  copied the new binary over `/home/milos/.local/bin/helixllm` and
  restarted `helixllm-gateway.service`. The running gateway is still the
  pre-fix binary (confirmed: `md5sum /home/milos/.local/bin/helixllm` =
  `47cd08ad2762f54bb5944bdc5a90d633`, service `active`, same PID as
  before this session). The fix is committed and unit-tested but **not
  yet live**.
- **Installing `llama-server` on the host** was not attempted — the task
  explicitly reserves this as an operator decision (no system package
  installs without approval).
- **Redirecting the gateway's "local" route at the already-running
  `helixllm-coder` container (127.0.0.1:18434)** was considered
  (§2) but deliberately not implemented: it would silently repurpose the
  general-purpose local-model fallback to serve a code-specialised 30B
  model the operator stood up for a different purpose. That is a design
  decision, not a bug fix, and is flagged below for the operator rather
  than applied unilaterally (§11.4.101/§11.4.122 posture: reversible and
  evidence-backed changes are fine to make autonomously; a change in
  what a shared feature *means* is not).

## 5. Needing an operator decision

1. **Deploy the fix.** Copy the rebuilt binary to
   `/home/milos/.local/bin/helixllm` and `systemctl --user restart
   helixllm-gateway.service` (a helix unit, in scope per the task's
   constraints) — or the operator's preferred deployment path — to make
   the fix live. Until this happens, the gateway keeps failing every
   completion request with the confusing TLS error, not the fixed clean
   error.
2. **Whether to actually restore local model serving**, and how:
   - (a) install `llama-server` on the host (system package / build from
     source — requires approval per the task's constraints), or
   - (b) point `HELIX_LLM_LOCAL_RPC_HOST=localhost` /
     `HELIX_LLM_LOCAL_RPC_PORT=18434` (and adjust `HELIX_LLM_LOCAL_MODEL`
     to the coder's actual served model id) at the already-running
     `helixllm-coder` container instead of embedding a second one — no
     install required, but repurposes a dedicated coder backend as the
     general gateway's fallback, or
   - (c) leave `HELIX_LLAMA_SERVER_EMBEDDED=false` and rely entirely on
     cloud providers once keys are configured in `.env`.
3. **Secondary, unrelated-but-adjacent finding** worth a look separately:
   `cfg.LLM.DefaultProvider` defaults to `"local"`
   (`config.go:69`) but the router registers the local provider under the
   name `"llamacpp"` (`brain.go:65-70`), so `Router.Route`'s step-4
   explicit-fallback lookup (`r.providers[r.fallback]`) never matches —
   it's currently harmless only because step-5 ("any available
   provider") happens to cover the same ground, but the fallback name
   mismatch is real and independent of this ticket. Not fixed here
   (scope discipline); flagging for its own ticket if desired.
4. **Unrelated finding, also flagged but not touched:** something other
   than `helixllm-coder` is listening on host port 8080 answering TLS
   handshakes (likely a rootless-podman port-forwarder for one of the
   other infra containers) — worth identifying if that port is ever
   needed for something else, but it is not a defect in itself and out
   of scope for HXC-233.

## Files

- Pre-fix host PATH check: `01_pre_fix_host_path_check.txt`
- Pre-fix live curl reproduction: `02_pre_fix_live_curl.txt`
- Fix commit (stat / full diff): `03_fix_commit_stat.txt`,
  `04_fix_commit_full_diff.txt`
- Post-fix `go test ./...`: `05_post_fix_go_test_all.txt`
- Post-fix `go build ./...`: `06_post_fix_go_build_all.txt`
