# HXC-193 + HXC-195 — one root cause, filed as two items

**Commits:** `30c81925` (submodule) · `46a96298` (gate + pointer)
**Gate:** `CM-MCP-SERVER-PORT-AGREEMENT` — `gate_run.txt`, exit 0.

The service read NO environment configuration at all, which is simultaneously why
the port knob was inert AND why the health check could never succeed. Filed
separately, fixed as one — splitting them would have put two agents on one fix.

A third cause neither ticket named: the health check invoked `curl`, absent from
`node:20-alpine`. Even with something listening, that probe could never pass.

Bind observed AT RUNTIME under the container's own environment, not inferred from
config: pre-fix `state=exited health=starting restarts=2`; post-fix
`running healthy restarts=0`, host receives `{"status":"healthy"}`.
