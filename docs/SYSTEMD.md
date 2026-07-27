# Helix Platform — systemd Boot Integration

**Revision:** 1
**Last modified:** 2026-07-27
**Maintainer:** HelixCode platform

The whole Helix platform — HelixCode, HelixAgent, HelixLLM and their
infrastructure — is installed as systemd **user** units that start on host boot
and survive restarts.

---

## Quick reference

```bash
./setup.sh                                  # install everything (units included)
./setup.sh --start                          # ... and start it now

systemctl --user start   helix.target       # bring the whole platform up
systemctl --user stop    helix.target       # bring it all down
systemctl --user status  helixagent         # one service
systemctl --user list-units 'helix*'        # all of them
journalctl --user -u helixagent -f          # follow one service's logs

scripts/install_systemd_units.sh            # re-install after editing a unit
scripts/install_systemd_units.sh --start    # install + enable + start
scripts/install_systemd_units.sh --uninstall
```

---

## Units

`helix.target` is the umbrella; each service declares `WantedBy=helix.target`,
so enabling a service wires it in automatically.

| Unit | Type | Port | Purpose |
|---|---|---|---|
| `helix.target` | target | — | Umbrella — start/stop the platform as one |
| `helixcode-infra.service` | oneshot | see below | Postgres, Redis, Weaviate, Qdrant, ChromaDB, Cognee, Ollama, Selenium, memcached via podman compose |
| `llmsverifier.service` | simple | 8100 | LLMsVerifier — model/provider verification + scoring API |
| `helixllm-coder.service` | oneshot | 18434 | Local Qwen3-Coder-30B model container |
| `helixllm-gateway.service` | simple | 8443 (TLS) | HelixLLM multi-provider LLM router |
| `helixagent.service` | simple | 7061 | HelixAgent runtime (HTTP/1.1+2 on TCP, HTTP/3 on UDP) |
| `helixcode-server.service` | simple | 8081 | HelixCode API server |

Start ordering: `infra` → `llmsverifier` → `llm-coder` / `llm-gateway` →
`helixagent` → `helixcode-server`.

Dependency strength is deliberate per service:

- `helixagent` uses `Requires=helixcode-infra.service` — its startup dependency
  verification is mandatory and hard-fails, so starting without infra is
  pointless.
- `helixllm-gateway` uses `Wants=llmsverifier.service` — the gateway polls
  `{verifier}/api/scores` and falls back to a static score table when it is
  unreachable. That fallback is a real working mode, so a down verifier must
  not prevent the gateway from starting and routing.
- `helixcode-server` uses `Wants=` for the same reason: it has its own
  infra-boot fallback path.

### Port map

| Port | Service | Notes |
|---|---|---|
| 8081 | helixcode-server | `config/replica-8081.yaml` |
| 8443 | helixllm-gateway | **TLS** — use `https://`, health route is `/v1/models` |
| 7061 | helixagent | TCP *and* UDP (QUIC) |
| 8100 | llmsverifier | the address the gateway looks for; the binary's own default is 8080, so the unit passes `--port 8100` explicitly |
| 18434 | helixllm-coder | local model |
| 5433 | infra postgres | not 5432 — 5432 belongs to an unrelated stack on this host |
| 6380 | infra redis | not 6379, same reason |
| 8083 | infra weaviate | moved off 8081, which helixcode-server owns |
| 8082 | infra chromadb | |
| 8000 | infra cognee | |
| 6333/6334 | infra qdrant | |
| 11434 | infra ollama | |
| 11211 | infra memcached | |
| 4444/7900 | infra selenium | |
| 9222 | infra chromedp | |

---

## Why user scope, and why linger

These are **user** units, not system units, because the services run rootless
podman (§11.4.161) as the invoking user and read that user's `~/.config/cdi`
GPU specs.

User units normally start at *login*. `loginctl enable-linger $USER` is what
makes them start at *boot* instead — the installer enables it and reports
whether it succeeded. Without linger the platform only comes up once someone
logs in.

Verify:

```bash
loginctl show-user "$USER" -p Linger      # expect Linger=yes
```

---

## Unit templating

The units in `scripts/systemd/` are **templates** containing `@HELIX_ROOT@` and
`@HELIXLLM_BIN@`. `scripts/install_systemd_units.sh` expands them for the
current checkout when installing into `~/.config/systemd/user/`.

Never hand-edit the installed copies — edit the template in `scripts/systemd/`
and re-run the installer, otherwise the change is lost on the next install and
a fresh clone cannot reproduce it. The installer refuses to install a unit that
still contains an unexpanded placeholder.

---

## Secrets

`helixagent` needs `DB_PASSWORD` and `JWT_SECRET`; both are read from `.env`
via `EnvironmentFile=` and are **never** inlined into unit files, which are
world-readable and version-controlled (CONST-042 / §12.1). `.env` is mode 0600
and gitignored.

If `.env` is missing, `EnvironmentFile=-` (leading `-`) makes it optional, so
the service fails with a clear dependency error rather than a unit-file parse
error.

---

## Two traps worth knowing

### `podman ps` is not a reachability oracle

`helixcode-infra.service` sets `KillMode=process`. `podman compose up -d`
returns immediately but leaves `conmon` and `rootlessport` (the host-side port
forwarders) in the unit's cgroup. Under the default
`KillMode=control-group`, any stop *or failure* makes systemd SIGKILL the whole
cgroup — killing the port forwarders of every container in the stack while the
containers themselves survive.

When that happens `podman ps` still reports `Up (healthy)
0.0.0.0:5433->5432/tcp`, because the container is alive; only the forwarder is
gone. Nothing on the host can connect.

**Always confirm reachability with `ss -ltn` or `curl`, never with `podman ps`.**

```bash
# what is ACTUALLY reachable
for p in 5433 6380 8083 8082 8000 6333 11434; do
  ss -ltn | grep -q ":$p " && echo "ok $p" || echo "DEAD $p"
done
```

### `helixagent` serves HTTP/3, so it binds UDP too

`ss -ltn` (TCP only) will not show the QUIC listener. Use `ss -ltn` *and*
`ss -lun`, or `ss -an | grep 7061`.

---

## Troubleshooting

```bash
systemctl --user status <unit> -l          # why it failed
journalctl --user -u <unit> -n 50          # recent log
journalctl --user -u <unit> --since today  # today's log
systemctl --user reset-failed              # clear failed state before retrying
systemctl --user cat <unit>                # the installed (expanded) unit
```

**A unit is `active (running)` but nothing is listening.** Unit state is not
proof the service works (§11.4.108). Check the journal for where startup
stalled, and probe the port directly.

**Infra fails with `address already in use`.** Something else on the host holds
the port. Identify the owner *before* acting — it may belong to another project
(§11.4.174):

```bash
ss -ltnp | grep ':<port>'
```

---

## Known gaps

Recorded honestly rather than papered over (§11.4.6):

- **`distributed_locks` table has no schema.** HelixAgent references it from
  three Go files but no migration or `.sql` file defines it, so it logs
  `Error cleaning locks: relation "distributed_locks" does not exist` every
  30 s. Non-fatal — the service runs normally. Needs a migration authored by
  someone who can specify the intended columns; not invented here.
- **LLMsVerifier publishes no scores yet.** `llmsverifier.service` is running
  and `GET :8100/api/scores` returns `200 {"scores":{}}` — an honest "nothing
  measured yet", because no verification run has populated its database. The
  gateway reads that correctly and keeps its static score table, logging
  `verifier returned empty scores, using static scores`. To populate it, run a
  verification pass (`llm-verifier --help` for the model/provider verification
  commands) with provider API keys configured; `/api/scores` then publishes the
  per-provider aggregate and the gateway adopts it.

  This is distinct from the earlier state, where the gateway logged
  `verifier unreachable … connection refused` because nothing served `:8100`
  at all.
- **`helixllm-gateway` lists no models** (`{"object":"list","data":null}`)
  until provider API keys are configured in `.env`. The API itself works.
- **QUIC receive-buffer warning** from `helixllm-gateway`
  (`failed to sufficiently increase receive buffer size`) — a host sysctl
  tuning matter (`net.core.rmem_max`), not a service defect.

---

## Sources verified 2026-07-27: PRIMARILY INTERNALLY DERIVED — this document describes THIS repository's own systemd user-unit installation, so its authoritative sources are the tracked unit definitions and installer, cross-referenced in this pass against: `scripts/systemd/` — all seven referenced units confirmed present (`helix.target`, `helixcode-infra.service`, `helixcode-server.service`, `helixagent.service`, `helixllm-gateway.service`, `helixllm-coder.service`, `llmsverifier.service`), matching the `systemctl --user` unit names used in the Quick reference above; and `scripts/install_systemd_units.sh` (confirmed present and executable, mode 0755), the installer the document instructs the operator to run, including its `--start` / `--uninstall` flags. Honest boundary (§11.4.6 / §11.4.99(B)) — two explicit gaps, recorded rather than papered over: (1) the upstream systemd manual pages (`systemd.unit`, `systemd.service`, `systemd.target`, `journalctl`) were NOT re-fetched in this pass, so the generic systemd semantics described here rest on the in-repo unit files alone and carry no external-currency claim; (2) no `systemctl --user` command was executed while writing this footer — the platform was deliberately not started or queried, because other work is live on this shared host (§11.4.174) — so this footer attests that the units and installer EXIST as documented, NOT that they were observed running in this pass. The runtime observations recorded in the sections above (verifier empty-scores, empty model list, QUIC warning) come from earlier captured runs, not from this verification pass. Re-verify against the official systemd documentation per the §11.4.99 180-day staleness window before this guide is used as external-facing operator authority.
