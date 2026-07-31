package llm

import "time"

// Shared health-verdict machinery for the provider implementations (HXC-214).
//
// Six providers — anthropic, openai, deepseek, mistral, openrouter and
// openai_compatible — expose a method NAMED as a query, GetHealth, that in
// practice MUTATES the provider's stored health record. Because it reads as a
// query, callers invoke it freely from concurrent paths (health monitors,
// status endpoints, IsAvailable), so before this change:
//
//   - the verdict was written field-by-field with no lock held, letting two
//     concurrent verdicts interleave — a provider could be reported healthy
//     when the probe had found it failing, or the reverse; and
//   - the method returned the provider's OWN *ProviderHealth, so anything a
//     caller did to the value it was handed silently mutated provider state.
//
// The two helpers below give all six providers ONE verdict shape, so the fix
// cannot drift between them.

// errCountAdjust says how a verdict should adjust ProviderHealth.ErrorCount.
//
// The pre-fix code computed the successor at the CALL SITE —
// `updateHealth(status, latency, p.lastHealth.ErrorCount+1)` — reading the
// counter with no lock held. Two concurrent failures therefore both read N and
// both stored N+1, losing one error. Passing the INTENT instead, and applying
// it inside the same critical section as the write, makes the counter correct
// under concurrency.
type errCountAdjust int

const (
	// errCountKeep leaves ErrorCount unchanged. Used by the "degraded" path,
	// where the probe reached the provider and only the response body was
	// unreadable — that is not a new connectivity failure.
	errCountKeep errCountAdjust = iota
	// errCountIncrement records one more consecutive failure.
	errCountIncrement
	// errCountReset clears the failure streak after a successful probe.
	errCountReset
)

// modelCountUnchanged tells applyProviderHealth to leave ModelCount as it was.
// Only a successful probe learns a model count; the failure and degraded paths
// deliberately keep the count from the last good probe rather than zeroing it,
// which is the behaviour the pre-fix code had by omission.
const modelCountUnchanged = -1

// applyProviderHealth writes a health verdict into h and returns a COPY of the
// updated record.
//
// The caller MUST already hold the mutex guarding h. The entire verdict is
// applied in that ONE critical section, so no concurrent reader can observe it
// half-applied.
//
// The returned pointer is safe to hand to an arbitrary caller: ProviderHealth
// is a flat value struct — every field is a value type, see missing_types.go —
// so `*h` is a COMPLETE copy sharing nothing with provider state. That property
// is load-bearing and is pinned by
// TestProviderHealth_HXC214_ProviderHealthIsFlat, which fails if a pointer,
// slice, map, chan or func field is ever added to ProviderHealth. (The HXC-205
// twin is the cautionary case: there the equivalent record DID carry pointer
// and map fields, so a value copy was only a shallow copy and callers still
// reached manager state through them.)
func applyProviderHealth(
	h *ProviderHealth,
	status string,
	latency time.Duration,
	adj errCountAdjust,
	modelCount int,
) *ProviderHealth {
	h.Status = status
	h.Latency = latency
	h.LastCheck = time.Now()

	switch adj {
	case errCountIncrement:
		h.ErrorCount++
	case errCountReset:
		h.ErrorCount = 0
	case errCountKeep:
		// Deliberately unchanged.
	}

	if modelCount != modelCountUnchanged {
		h.ModelCount = modelCount
	}

	snapshot := *h
	return &snapshot
}
