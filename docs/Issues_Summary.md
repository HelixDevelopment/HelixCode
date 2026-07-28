# HelixCode — Issues Summary

> Generated **mechanically** from `docs/Issues.md` by `scripts/generate_issues_summary.sh` per Constitution §11.4.91 (summary clarity) + §11.4.12 (CM-ISSUES-SUMMARY-SYNC). Do not hand-edit — re-run the generator. Title column carries each item's H2 heading (self-contained, ≥40 chars per §11.4.91); Notes is the first Closure/Evidence/Resolution line.
>
> **Prefix convention:** IDs are scope-prefixed (`HXC`=root project; `HXA`=HelixAgent; `HXL`=HelixLLM; `HXQ`=HelixQA; `HXV`=LLMsVerifier; `VEN`=VisionEngine; `PAN`=panoptic; `OPS`=LLMOps). See `docs/Issues.md` "Prefix convention" for the legacy `ISSUE-NNN` mapping.

| ID | Title | Type | Status | Discovered | Notes |
|---|---|---|---|---|---|
| HXC-153 | TestGuard_GetSystemStatus_WithDB_StillReports is named for a database-present case it never exercises | Bug | Queued |  | TestGuard_GetSystemStatus_WithDB_StillReports is named for a database-present case it never exercises |
| HXC-154 | TestStartQASession_Success asserts on a session field a background goroutine is concurrently changing | Bug | Queued |  | TestStartQASession_Success asserts on a session field a background goroutine is concurrently changing |
| HXC-155 | Standardise the divergent polarity-switch environment variable convention across the server test package | Task | Queued |  | Standardise the divergent polarity-switch environment variable convention across the server test package |
| HXC-156 | harmony_os background system monitor used an unsynchronised on/off flag as its stop signal, racing application shutdown | Bug | Queued |  | harmony_os background system monitor used an unsynchronised on/off flag as its stop signal, racing application shutdown |
| HXC-157 | harmony_os and aurora_os changed on-screen elements directly from background threads, which the Fyne UI toolkit forbids | Bug | Queued |  | harmony_os and aurora_os changed on-screen elements directly from background threads, which the Fyne UI toolkit forbids |
| HXC-158 | No test builds the harmony_os or aurora_os screens, so their widget-threading fixes rest on code review rather than captured runtime proof | Bug | Queued |  | No test builds the harmony_os or aurora_os screens, so their widget-threading fixes rest on code review rather than captured runtime proof |

**Counts**: 6 tracked item-sections in `docs/Issues.md` — **6 open** (non-terminal status) / **0 closed** (terminal `(→ Fixed.md)` status; retained as §11.4.19 migration tombstones).

*Last regenerated: 2026-07-29 by `scripts/generate_issues_summary.sh`. HTML/PDF exports via `scripts/regenerate-tracker-exports.sh`.*
