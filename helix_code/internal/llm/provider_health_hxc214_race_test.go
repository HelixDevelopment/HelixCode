package llm

// HXC-214 — standing regression guard for the provider GetHealth data race and
// live-pointer aliasing defect.
//
// # THE DEFECT
//
// Six LLM providers in this package implement a method NAMED as a query —
// GetHealth — that in practice MUTATES shared provider state and then hands the
// caller a pointer to that very state:
//
//	anthropic_provider.go:991          ap.lastHealth = health     (then `return health`)
//	openai_provider.go:222 / :498      op.lastHealth.<field> = …  (then `return op.lastHealth`)
//	deepseek_provider.go:194 / :471    dp.lastHealth.<field> = …  (then `return dp.lastHealth`)
//	mistral_provider.go:180 / :448     mp.lastHealth.<field> = …  (then `return mp.lastHealth`)
//	openrouter_provider.go:174 / :561  orp.lastHealth.<field> = … (then `return orp.lastHealth`)
//	openai_compatible_provider.go:343 / :725
//
// (The last two were NOT in the reported site list — that sweep keyed on the
// presence of a catalogMu field, which those two types do not declare. They
// carry the identical defect and are covered here so the package is not left
// half-fixed.)
//
// Two distinct faults fall out of that one line:
//
//  1. UNGUARDED CONCURRENT WRITE. Four of the six types DECLARE a mutex
//     (catalogMu) — but it guards only the model catalogue, never lastHealth,
//     and the other two declare no mutex at all. Because the method reads as a
//     query, callers invoke it freely from concurrent paths (health monitors,
//     status endpoints, IsAvailable), so concurrent verdicts interleave
//     field-by-field and overwrite each other. A provider can be reported
//     healthy when the probe found it failing, or failing when it is healthy.
//     The error counter is worse than interleaved: the old code computed
//     `lastHealth.ErrorCount+1` at the CALL SITE with no lock held, so two
//     concurrent failures both read N and both store N+1, losing one error.
//
//  2. ALIASING. The returned pointer IS the provider's record. Anything the
//     caller does to the value it was handed silently mutates provider state —
//     which no caller of a method called "GetHealth" could reasonably expect.
//     This half bites with or without the race detector.
//
// # WHY A COPY IS SAFE HERE (and was NOT in the HXC-205 twin)
//
// HXC-205's GetStatus returned `*v`, a SHALLOW copy, and that was insufficient
// because HealthStatus carried pointer and map fields. ProviderHealth does not:
// every field is a value type — see missing_types.go:114-121 (string, time.Time,
// time.Duration, int, int, string). So `*lastHealth` is a COMPLETE copy and the
// returned pointer shares nothing whatsoever with provider state. That fact is
// load-bearing for the fix and is asserted directly by
// TestProviderHealth_HXC214_ProviderHealthIsFlat below, so that adding a
// pointer/slice/map field to ProviderHealth later BREAKS this guard instead of
// silently re-opening the aliasing hole.
//
// # POLARITY SWITCH — RED_MODE (§11.4.115, repo convention)
//
// A data race has no in-process assertion API: the verdict is the race
// detector's own report plus the non-zero exit code. So, exactly as in
// auto_llm_manager_hxc205_race_test.go and
// tests/unit/local_llm_manager_hxc203_race_test.go, the polarity is carried by
// WHICH synchronization is exercised, and the evidence is the -race output.
//
//   - RED_MODE unset / "0" (DEFAULT — the standing GREEN regression guard,
//     §11.4.135): drives real concurrent GetHealth traffic against every
//     affected provider, and separately asserts that a caller mutating the
//     returned record cannot reach provider state. On the shipped artifact this
//     is race-free and non-aliasing. It goes RED if anyone drops the locking or
//     the copy back out of the health path. Captured going RED on the genuine
//     pre-fix artifact in docs/qa/hxc214_provider_health_race_*/.
//
//   - RED_MODE=1 (harness self-validation — the §11.4.107(10) golden-bad
//     fixture): concurrently performs the SAME unsynchronized write the pre-fix
//     code performed — `h.Status = …; h.LastCheck = …` on one shared
//     *ProviderHealth — with no lock. This provokes the defect class on ANY
//     artifact, including the fixed one, and so proves the harness can actually
//     SEE an unsynchronized write to this data class: that a GREEN result means
//     "no race", not "blind test".
//     EXPECTED OUTCOME under -race: a DATA RACE report and a non-zero exit.
//     If RED_MODE=1 ever completes CLEANLY under -race, this guard has lost its
//     teeth and every GREEN result from it is worthless.
//
// # WHY THIS GUARD IS IN-PACKAGE
//
// The aliasing half must observe whether a caller's write reached the
// provider's OWN record. lastHealth is unexported and no accessor exposes it,
// so the assertion is only expressible from inside the package. No production
// API was widened to make these types testable.
//
// Run the GREEN guard (default):
//
//	go test -race -count=3 -timeout 300s -run TestProviderHealth_HXC214 ./internal/llm/
//
// Run the harness self-validation (expect a DATA RACE + non-zero exit):
//
//	RED_MODE=1 go test -race -count=1 -timeout 120s \
//	  -run TestProviderHealth_HXC214_HarnessSelfValidation ./internal/llm/

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"
)

// hxc214RedMode reports whether the golden-bad self-validation arm is selected.
func hxc214RedMode() bool { return os.Getenv("RED_MODE") == "1" }

// hxc214Server serves both wire shapes the affected providers probe:
//   - GET  <any>/models              -> an OpenAI-shaped model list (the five
//     shape-A providers' health probe)
//   - POST <any>                     -> a minimal Anthropic messages response
//     (AnthropicProvider.GetHealth issues a real Generate call)
//
// Every response is well-formed, so GetHealth takes its SUCCESS path — which is
// the path that writes lastHealth and returns the live pointer. Driving the
// error paths instead would exercise a different, less interesting branch.
func hxc214Server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"id":"msg_hxc214","type":"message","role":"assistant",` +
				`"model":"claude-3-5-haiku-latest",` +
				`"content":[{"type":"text","text":"Hi"}],` +
				`"stop_reason":"end_turn",` +
				`"usage":{"input_tokens":1,"output_tokens":1}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"model-a"},{"id":"model-b"},{"id":"model-c"}]}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// hxc214Target is one provider under test, reduced to the two things this guard
// needs: a way to invoke the query-named method, and a way to read the record
// the provider kept for itself.
type hxc214Target struct {
	name string
	// getHealth invokes the provider's GetHealth.
	getHealth func(context.Context) (*ProviderHealth, error)
	// stored returns the provider's OWN record. Used ONLY from the
	// single-goroutine aliasing subtest — reading an unexported field while
	// other goroutines write it would be a race introduced by the test.
	stored func() *ProviderHealth
}

// hxc214Targets constructs every provider that carries the defect, pointed at
// the local test server. Construction is real: these are the production
// constructors, with a real *http.Client talking to a real HTTP server.
func hxc214Targets(t *testing.T, endpoint string) []hxc214Target {
	t.Helper()

	cfg := func(ep string) ProviderConfigEntry {
		return ProviderConfigEntry{APIKey: "hxc214-test-key", Endpoint: ep}
	}

	openai, err := NewOpenAIProvider(cfg(endpoint))
	if err != nil {
		t.Fatalf("NewOpenAIProvider: %v", err)
	}
	deepseek, err := NewDeepSeekProvider(cfg(endpoint))
	if err != nil {
		t.Fatalf("NewDeepSeekProvider: %v", err)
	}
	mistral, err := NewMistralProvider(cfg(endpoint))
	if err != nil {
		t.Fatalf("NewMistralProvider: %v", err)
	}
	openrouter, err := NewOpenRouterProvider(cfg(endpoint))
	if err != nil {
		t.Fatalf("NewOpenRouterProvider: %v", err)
	}
	// AnthropicProvider's endpoint is the FULL messages URL, not a base.
	anthropic, err := NewAnthropicProvider(cfg(endpoint + "/v1/messages"))
	if err != nil {
		t.Fatalf("NewAnthropicProvider: %v", err)
	}
	compat, err := NewOpenAICompatibleProvider("hxc214-compat", OpenAICompatibleConfig{
		BaseURL: endpoint,
		APIKey:  "hxc214-test-key",
		Timeout: 30 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewOpenAICompatibleProvider: %v", err)
	}

	return []hxc214Target{
		{"openai", openai.GetHealth, func() *ProviderHealth { return openai.lastHealth }},
		{"deepseek", deepseek.GetHealth, func() *ProviderHealth { return deepseek.lastHealth }},
		{"mistral", mistral.GetHealth, func() *ProviderHealth { return mistral.lastHealth }},
		{"openrouter", openrouter.GetHealth, func() *ProviderHealth { return openrouter.lastHealth }},
		{"anthropic", anthropic.GetHealth, func() *ProviderHealth { return anthropic.lastHealth }},
		{"openai_compatible", compat.GetHealth, func() *ProviderHealth { return compat.lastHealth }},
	}
}

// TestProviderHealth_HXC214_ProviderHealthIsFlat pins the fact the fix rests on:
// ProviderHealth has NO pointer, slice, map, channel or func field, so a value
// copy is a COMPLETE copy. If a future change adds such a field, `*lastHealth`
// silently becomes a shallow copy and the aliasing hole re-opens through that
// field alone — this test fails first and says so.
func TestProviderHealth_HXC214_ProviderHealthIsFlat(t *testing.T) {
	rt := reflect.TypeOf(ProviderHealth{})
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		switch f.Type.Kind() {
		case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func, reflect.UnsafePointer:
			t.Fatalf("ProviderHealth.%s is %s: a value copy of ProviderHealth is no longer a "+
				"complete copy, so GetHealth's returned copy still aliases provider state "+
				"through this field. Deep-copy it explicitly (see HXC-205, where exactly "+
				"this shallow-copy assumption was the bug).", f.Name, f.Type.Kind())
		}
	}
}

// TestProviderHealth_HXC214_NoAliasing is the half that bites WITHOUT the race
// detector: it proves a caller cannot reach provider state through the value
// GetHealth handed it.
//
// On the pre-fix artifact `return p.lastHealth` (and, for anthropic,
// `ap.lastHealth = health; return health`) hands back the provider's own
// record, so the caller's write lands in provider state and this FAILS.
func TestProviderHealth_HXC214_NoAliasing(t *testing.T) {
	if hxc214RedMode() {
		t.Skip("SKIP-OK: RED_MODE=1 selects the harness self-validation arm only")
	}
	srv := hxc214Server(t)
	ctx := context.Background()

	for _, tgt := range hxc214Targets(t, srv.URL) {
		t.Run(tgt.name, func(t *testing.T) {
			returned, err := tgt.getHealth(ctx)
			if err != nil {
				t.Fatalf("GetHealth returned an error against the local test server: %v", err)
			}
			if returned == nil {
				t.Fatal("GetHealth returned a nil health record")
			}

			stored := tgt.stored()
			if stored == nil {
				t.Fatal("provider kept no health record, so GetHealth's write never happened")
			}

			// Identity check first: the clearest statement of the defect.
			if returned == stored {
				t.Errorf("GetHealth returned the provider's OWN *ProviderHealth (%p): every "+
					"caller can mutate provider state through it", returned)
			}

			before := *stored

			// Now behave like an ordinary caller that treats the returned value
			// as its own — the thing a query-named method invites.
			returned.Status = "CORRUPTED-BY-CALLER"
			returned.ErrorCount = 987654
			returned.ModelCount = -321
			returned.Message = "caller scribbled here"

			after := *tgt.stored()
			if after != before {
				t.Errorf("a caller's writes to the value returned by GetHealth changed the "+
					"provider's own health record.\n  before: %+v\n  after:  %+v", before, after)
			}
		})
	}
}

// TestProviderHealth_HXC214_ConcurrentGetHealth drives the REAL production
// concurrency: several goroutines calling the query-named GetHealth at once,
// which is exactly what a health monitor, a status endpoint and IsAvailable do.
//
// On the pre-fix artifact every one of those calls writes Status, Latency,
// LastCheck and ModelCount on one shared record with no lock, and reads
// ErrorCount to compute its successor — so the detector reports it.
//
// The verdict is the race detector's report plus the process exit code; there is
// no in-process assertion for a data race.
func TestProviderHealth_HXC214_ConcurrentGetHealth(t *testing.T) {
	if hxc214RedMode() {
		t.Skip("SKIP-OK: RED_MODE=1 selects the harness self-validation arm only")
	}
	srv := hxc214Server(t)

	const (
		goroutines = 8
		iterations = 12
	)

	for _, tgt := range hxc214Targets(t, srv.URL) {
		t.Run(tgt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			var wg sync.WaitGroup
			start := make(chan struct{})
			errs := make(chan error, goroutines*iterations)

			for g := 0; g < goroutines; g++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start // release all goroutines together, to overlap the writes
					for i := 0; i < iterations; i++ {
						h, err := tgt.getHealth(ctx)
						if err != nil {
							errs <- err
							return
						}
						// Read every field, as a real caller would. Post-fix this
						// touches a private copy; pre-fix it touches shared state
						// concurrently with other goroutines' writes.
						_ = fmt.Sprintf("%s/%d/%d/%v/%v/%s",
							h.Status, h.ErrorCount, h.ModelCount, h.Latency, h.LastCheck, h.Message)
					}
				}()
			}

			close(start)
			wg.Wait()
			close(errs)

			for err := range errs {
				t.Fatalf("GetHealth failed against the local test server: %v", err)
			}
		})
	}
}

// TestProviderHealth_HXC214_HarnessSelfValidation is the §11.4.107(10)
// golden-bad fixture. It performs the SAME unsynchronized write the pre-fix
// providers performed — concurrent field writes on one shared *ProviderHealth,
// with a concurrent read of ErrorCount to compute its successor, exactly as
// `updateHealth(…, p.lastHealth.ErrorCount+1)` did — and MUST be reported by
// the race detector.
//
// This runs ONLY under RED_MODE=1 and is EXPECTED TO FAIL there. Its purpose is
// to prove that a GREEN result from the guards above means "no race" rather
// than "the harness cannot see races at all".
func TestProviderHealth_HXC214_HarnessSelfValidation(t *testing.T) {
	if !hxc214RedMode() {
		t.Skip("SKIP-OK: golden-bad fixture; runs only under RED_MODE=1 (§11.4.107(10))")
	}

	t.Log("RED_MODE=1: reproducing the pre-fix unsynchronized write on a shared " +
		"*ProviderHealth. Under -race this MUST report a DATA RACE and exit non-zero. " +
		"A clean run here means the guard is blind and its GREEN results are worthless.")

	shared := &ProviderHealth{Status: "unknown", LastCheck: time.Now()}

	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				// Verbatim pre-fix shape: read-then-write ErrorCount across no
				// lock, plus the field-by-field verdict write.
				next := shared.ErrorCount + 1
				shared.Status = "unhealthy"
				shared.Latency = time.Duration(n) * time.Millisecond
				shared.ErrorCount = next
				shared.LastCheck = time.Now()
				shared.ModelCount = i
			}
		}(g)
	}
	wg.Wait()

	// Unreachable as a meaningful assertion under -race (the detector aborts the
	// process first); kept so the body has an observable effect without -race.
	if b, _ := json.Marshal(shared); len(b) == 0 {
		t.Fatal("shared health record did not marshal")
	}
}
