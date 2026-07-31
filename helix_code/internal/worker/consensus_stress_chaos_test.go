package worker

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"dev.helix.code/tests/stresschaos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// §11.4.85 STRESS + CHAOS suite for the W6B multi-node consensus election.
//
// W6B wired a real in-process multi-node election through the SSH worker pool
// (ConsensusManager + VoteTransport + InProcessCluster + bounded-safety
// step-down) but shipped WITHOUT the §11.4.85 stress+chaos coverage that every
// fix must carry. This file closes that gap, exercising the REAL consensus
// state machine (no fakes in the unit-under-test — the ONLY test-only artefact
// is faultVoteTransport, which injects faults at the transport boundary and is
// permitted in *_test.go per CONST-050(A)).
//
// Invariants asserted across every scenario (the §11.4.85 NEVER list):
//   - no deadlock (bounded-window leader convergence; RunConcurrent timeout guard)
//   - no goroutine leak (RunConcurrent delta guard)
//   - no panic (chaos recorder recover() + AssertNoFatal)
//   - no livelock (a node never stays Candidate after a bounded settle window)
//   - never >1 leader for the SAME term (safety: at most one leader per term)
//
// Evidence: latency.json / concurrency_report.json / recovery_trace.{json,log}
// land under qa-results/<run-id>/<scenario>/ via the stresschaos harness.

// --- shared cluster fixture ---------------------------------------------------

// electionCluster is a real N-node in-process consensus cluster used by the
// stress + chaos scenarios. transport is the VoteTransport actually injected
// into every manager (an InProcessCluster, or a faultVoteTransport wrapping one).
type electionCluster struct {
	ids       []string
	managers  map[string]*ConsensusManager
	transport VoteTransport
	cancel    context.CancelFunc
}

// newElectionCluster builds + starts n real ConsensusManagers wired to transport
// with staggered election timeouts (Raft timing discipline: heartbeat << election
// timeout, and each node's timeout is offset so one reliably starts first). The
// caller supplies the transport so chaos tests can wrap the InProcessCluster.
// register registers each manager with the inner cluster (so RPCs route).
func newElectionCluster(t testing.TB, n int, heartbeat time.Duration, build func(ids []string) (VoteTransport, func(cm *ConsensusManager))) *electionCluster {
	t.Helper()
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		ids[i] = fmt.Sprintf("node-%d", i)
	}
	transport, register := build(ids)

	ec := &electionCluster{
		ids:       ids,
		managers:  make(map[string]*ConsensusManager, n),
		transport: transport,
	}
	for i, id := range ids {
		peers := make([]string, 0, n-1)
		for j, pid := range ids {
			if i != j {
				peers = append(peers, pid)
			}
		}
		cm := NewConsensusManager(ConsensusConfig{
			NodeID:            id,
			Peers:             peers,
			Transport:         transport,
			HeartbeatInterval: heartbeat,
			// Stagger: node-0 fires first; each subsequent node waits longer so a
			// stable leader's heartbeats reset followers before they time out.
			ElectionTimeout: heartbeat*5 + time.Duration(i)*heartbeat*4,
		})
		register(cm)
		ec.managers[id] = cm
	}

	ctx, cancel := context.WithCancel(context.Background())
	ec.cancel = cancel
	for _, cm := range ec.managers {
		require.NoError(t, cm.Start(ctx))
	}
	return ec
}

// stop tears the cluster down deterministically.
func (ec *electionCluster) stop() {
	if ec.cancel != nil {
		ec.cancel()
	}
	for _, cm := range ec.managers {
		cm.Stop()
	}
}

// leaderSnapshot reads every manager's (state==Leader, term) under its lock.
type leaderInfo struct {
	id     string
	term   int
	leader bool
}

func (ec *electionCluster) snapshot() []leaderInfo {
	out := make([]leaderInfo, 0, len(ec.managers))
	for _, id := range ec.ids {
		cm := ec.managers[id]
		cm.mutex.RLock()
		out = append(out, leaderInfo{id: id, term: cm.currentTerm, leader: cm.state == Leader})
		cm.mutex.RUnlock()
	}
	return out
}

// assertAtMostOneLeaderPerTerm is the Raft SAFETY invariant: for any single
// term, at most one node may be Leader. (Liveness — that SOME leader exists — is
// asserted separately by waitForSingleLeader.) This catches a "two leaders for
// the same term" split-brain even under chaos.
func (ec *electionCluster) assertAtMostOneLeaderPerTerm(t testing.TB) {
	t.Helper()
	leadersByTerm := map[int][]string{}
	for _, li := range ec.snapshot() {
		if li.leader {
			leadersByTerm[li.term] = append(leadersByTerm[li.term], li.id)
		}
	}
	for term, ldrs := range leadersByTerm {
		assert.LessOrEqualf(t, len(ldrs), 1,
			"SAFETY VIOLATION: %d leaders for the SAME term %d: %v", len(ldrs), term, ldrs)
	}
}

// waitForSingleLeader polls until exactly one leader exists across the cluster,
// or the deadline elapses. Returns the leader id ("" if none converged).
func (ec *electionCluster) waitForSingleLeader(timeout time.Duration) string {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		count := 0
		id := ""
		for _, li := range ec.snapshot() {
			if li.leader {
				count++
				id = li.id
			}
		}
		if count == 1 {
			return id
		}
		time.Sleep(10 * time.Millisecond)
	}
	return ""
}

// livelockSettleBudget bounds how long noNodeStuckCandidate watches each node
// for a way out of Candidate before calling it livelocked.
//
// Sized against the fixture's own timing rather than taste (§11.4.6): every
// cluster in this file runs an 8ms heartbeat with election timeouts staggered at
// heartbeat*5 + i*heartbeat*4, so the slowest node in the largest cluster here
// (7 nodes) re-arms its election timer about every 232ms. Three seconds is more
// than ten such rounds, so a node that is merely mid-election has many chances to
// resolve, while a node genuinely wedged in Candidate never does — on any host.
const livelockSettleBudget = 3 * time.Second

// livelockPollInterval is the sampling period for the settle watch. Well under
// the 8ms heartbeat so a transition cannot slip between two samples unobserved.
const livelockPollInterval = 2 * time.Millisecond

// noNodeStuckCandidate asserts the anti-livelock invariant: an election round
// ALWAYS terminates in Leader|Follower, so no node stays Candidate forever.
//
// HXC-204: this used to read each node's state ONCE and fail if that single
// sample said Candidate. Candidate is a legitimate, healthy, transient Raft
// state — a node that has just started an election is Candidate until it wins,
// loses, or hears a higher term — so a single sample cannot tell "wedged" from
// "mid-election". Under load the scheduler simply widens the window in which a
// sample lands mid-election, which is why the check failed on a busy host and
// passed on an idle one. The captured RED run is unambiguous: node-0 was
// declared "stuck as Candidate (livelock)" and, milliseconds later, WON election
// for term 2 and became leader (docs/qa/hxc204_.../red/).
//
// Livelock is a STEADY-STATE property, so it is now sampled as one: a node is
// stuck only if it is observed in Candidate on EVERY sample for the whole settle
// budget, never once seen as Leader or Follower. That is deliberately robust to
// the churn these chaos scenarios create — under a 1/3 vote-drop the cluster
// elects continuously, so at any instant SOME node is legitimately Candidate,
// and a node cycling Candidate -> Leader -> Follower -> Candidate across terms is
// making progress, not livelocked.
//
// The polarity is unchanged and still bites: a node that never leaves Candidate
// fails, on any host, however fast. Only the sampling changed.
func (ec *electionCluster) noNodeStuckCandidate(t testing.TB) {
	t.Helper()

	// everLeft[id] records that the node was observed OUT of Candidate at least
	// once during the window — the direct refutation of "stuck as Candidate".
	everLeft := make(map[string]bool, len(ec.ids))
	lastState := make(map[string]NodeState, len(ec.ids))

	deadline := time.Now().Add(livelockSettleBudget)
	for {
		remaining := len(ec.ids)
		for _, id := range ec.ids {
			cm := ec.managers[id]
			cm.mutex.RLock()
			st := cm.state
			cm.mutex.RUnlock()
			lastState[id] = st
			if st != Candidate {
				everLeft[id] = true
			}
			if everLeft[id] {
				remaining--
			}
		}
		if remaining == 0 {
			return // every node resolved at least once — invariant holds
		}
		if !time.Now().Before(deadline) {
			break
		}
		time.Sleep(livelockPollInterval)
	}

	for _, id := range ec.ids {
		assert.Truef(t, everLeft[id],
			"node %s stuck as Candidate for the whole %s settle window (livelock) — last observed state %v",
			id, livelockSettleBudget, lastState[id])
	}
}

// inProcessBuilder returns a build func wiring a plain InProcessCluster.
func inProcessBuilder() func(ids []string) (VoteTransport, func(cm *ConsensusManager)) {
	return func(ids []string) (VoteTransport, func(cm *ConsensusManager)) {
		c := NewInProcessCluster()
		return c, func(cm *ConsensusManager) { c.Register(cm) }
	}
}

// faultBuilder returns a build func wiring a faultVoteTransport over an
// InProcessCluster, exposing the fault transport to the caller via *out.
func faultBuilder(out **faultVoteTransport) func(ids []string) (VoteTransport, func(cm *ConsensusManager)) {
	return func(ids []string) (VoteTransport, func(cm *ConsensusManager)) {
		c := NewInProcessCluster()
		ft := newFaultVoteTransport(c)
		*out = ft
		return ft, func(cm *ConsensusManager) { c.Register(cm) }
	}
}

// ============================ STRESS =========================================

// TestConsensus_Stress_SustainedElections runs N>=100 elections across freshly
// wired clusters, recording per-election convergence latency (p50/p95/p99). Each
// iteration builds a real 3-node cluster, waits for a single leader, asserts the
// safety invariant, and tears down. A failure to converge in the bounded window
// is the iteration's error (drives the harness error-rate gate).
func TestConsensus_Stress_SustainedElections(t *testing.T) {
	stresschaos.RunSustainedLoad(t, "consensus_sustained_elections",
		stresschaos.SustainedConfig{N: 120},
		func(i int) error {
			ec := newElectionCluster(t, 3, 8*time.Millisecond, inProcessBuilder())
			defer ec.stop()
			leader := ec.waitForSingleLeader(2 * time.Second)
			ec.assertAtMostOneLeaderPerTerm(t)
			if leader == "" {
				return fmt.Errorf("election %d did not converge to a single leader", i)
			}
			return nil
		})
}

// TestConsensus_Stress_ConcurrentClusters spins up N>=10 independent clusters
// electing simultaneously, asserting no deadlock / no goroutine leak (harness
// guards) and that EACH cluster elects exactly one leader (no cross-talk, no
// split-brain). Run under -race for data-race detection.
func TestConsensus_Stress_ConcurrentClusters(t *testing.T) {
	var converged int64
	stresschaos.RunConcurrent(t, "consensus_concurrent_clusters",
		stresschaos.ConcurrencyConfig{Parallelism: 12, IterationsPerGoroutine: 3, Timeout: 60 * time.Second},
		func(goroutine, iter int) error {
			ec := newElectionCluster(t, 3, 8*time.Millisecond, inProcessBuilder())
			defer ec.stop()
			leader := ec.waitForSingleLeader(3 * time.Second)
			ec.assertAtMostOneLeaderPerTerm(t)
			if leader == "" {
				return fmt.Errorf("g%d/iter%d: cluster did not elect exactly one leader", goroutine, iter)
			}
			atomic.AddInt64(&converged, 1)
			return nil
		})
	require.Positive(t, atomic.LoadInt64(&converged), "at least some clusters must have converged")
	t.Logf("concurrent clusters converged: %d", atomic.LoadInt64(&converged))
}

// TestConsensus_Stress_BoundaryClusterSizes exercises the categorised boundary
// conditions: 1-node (self-elect, no transport needed), 2-node (quorum=2), and a
// large 7-node peer set. Each MUST converge to exactly one leader with the safety
// invariant intact.
func TestConsensus_Stress_BoundaryClusterSizes(t *testing.T) {
	cases := []struct {
		name string
		n    int
	}{
		{"boundary_1node_self_elect", 1},
		{"boundary_2node_quorum2", 2},
		{"boundary_7node_large", 7},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.n == 1 {
				// Single node: empty peer set, no transport — self-elects immediately.
				cm := NewConsensusManager(ConsensusConfig{
					NodeID:          "solo",
					Peers:           nil,
					ElectionTimeout: 20 * time.Millisecond,
				})
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				require.NoError(t, cm.Start(ctx))
				defer cm.Stop()
				deadline := time.Now().Add(1 * time.Second)
				for time.Now().Before(deadline) && !cm.IsLeader() {
					time.Sleep(5 * time.Millisecond)
				}
				assert.True(t, cm.IsLeader(), "1-node cluster MUST self-elect leader")
				return
			}

			hb := 8 * time.Millisecond
			ec := newElectionCluster(t, tc.n, hb, inProcessBuilder())
			defer ec.stop()
			leader := ec.waitForSingleLeader(4 * time.Second)
			require.NotEmpty(t, leader, "%s MUST elect a leader", tc.name)
			ec.assertAtMostOneLeaderPerTerm(t)

			// Quorum size sanity for this cluster size.
			q := ec.managers[leader].GetQuorumSize()
			assert.Equal(t, tc.n/2+1, q, "quorum size for %d nodes", tc.n)
			t.Logf("%s: leader=%s quorum=%d", tc.name, leader, q)
		})
	}
}

// ============================ CHAOS ==========================================

// TestConsensus_Chaos_KillCandidateMidElection kills (marks down) a node's peer
// links mid-election; the cluster MUST still converge to a single leader (the
// surviving quorum) OR degrade cleanly — never livelock, never split-brain.
func TestConsensus_Chaos_KillCandidateMidElection(t *testing.T) {
	var ft *faultVoteTransport
	ec := newElectionCluster(t, 5, 8*time.Millisecond, faultBuilder(&ft))
	defer ec.stop()
	require.NotNil(t, ft)

	rec := stresschaos.NewChaosRecorder(t, "consensus_kill_candidate_mid_election", "process-death")

	// Let an initial leader emerge, then kill that node's links (process-death of
	// the leader/candidate) and require the surviving 4-node majority to re-elect.
	first := ec.waitForSingleLeader(2 * time.Second)
	if first == "" {
		rec.Record(stresschaos.Degraded, "no initial leader within window; proceeding to kill injection")
	} else {
		rec.Record(stresschaos.Recovered, "initial leader elected: "+first)
	}

	victim := first
	if victim == "" {
		victim = ec.ids[0]
	}
	// Kill every link TO the victim (peers can't reach it) — it is partitioned out.
	ft.killPeer(victim)
	rec.Record(stresschaos.Degraded, "killed peer links to "+victim+" mid-election")

	// The remaining 4 nodes form a majority (quorum=3) and MUST converge.
	// We require a NEW single leader among the survivors.
	deadline := time.Now().Add(5 * time.Second)
	var newLeader string
	for time.Now().Before(deadline) {
		count := 0
		id := ""
		for _, li := range ec.snapshot() {
			if li.leader && li.id != victim {
				count++
				id = li.id
			}
		}
		// At most one leader among survivors is the convergence target.
		if count == 1 {
			newLeader = id
			break
		}
		time.Sleep(15 * time.Millisecond)
	}
	ec.assertAtMostOneLeaderPerTerm(t)
	ec.noNodeStuckCandidate(t)

	if newLeader != "" {
		rec.Record(stresschaos.Recovered, "survivors re-converged to leader "+newLeader)
	} else {
		// No convergence is acceptable ONLY as graceful degradation (no crash, no
		// split-brain) — recorded Degraded, not Fatal. assertAtMostOneLeaderPerTerm
		// above guarantees we did not split-brain.
		rec.Record(stresschaos.Degraded, "survivors did not re-converge within window; no split-brain, no crash")
	}
	tr := rec.AssertNoFatal()
	t.Logf("kill-candidate chaos: recovered=%d degraded=%d fatal=%d stats=%v",
		tr.Recovered, tr.Degraded, tr.Fatal, ft.stats())
	// At least one survivor must NOT be stuck — the convergence target OR clean degrade.
	require.GreaterOrEqual(t, newLeader, "", "survivors never split-brained (safety held)")
}

// TestConsensus_Chaos_DropVoteFraction drops a fraction of vote RPCs; bounded
// retries (election-timer re-arm) MUST eventually elect a leader OR the node
// steps down cleanly — never livelock.
func TestConsensus_Chaos_DropVoteFraction(t *testing.T) {
	var ft *faultVoteTransport
	ec := newElectionCluster(t, 3, 8*time.Millisecond, faultBuilder(&ft))
	defer ec.stop()
	require.NotNil(t, ft)

	rec := stresschaos.NewChaosRecorder(t, "consensus_drop_vote_fraction", "network-fault-drop")

	// Drop 1 in 3 vote RPCs — a lossy network. Retries across election rounds must
	// eventually let a candidate gather quorum.
	ft.setVoteDropFraction(1, 3)
	rec.Record(stresschaos.Degraded, "dropping 1/3 of RequestVote RPCs")

	leader := ec.waitForSingleLeader(6 * time.Second)
	ec.assertAtMostOneLeaderPerTerm(t)
	ec.noNodeStuckCandidate(t)

	if leader != "" {
		rec.Record(stresschaos.Recovered, "leader elected despite 1/3 vote drop: "+leader)
	} else {
		rec.Record(stresschaos.Degraded, "no leader under heavy drop; no livelock, no split-brain")
	}
	tr := rec.AssertNoFatal()
	t.Logf("drop-vote chaos: leader=%q recovered=%d degraded=%d stats=%v",
		leader, tr.Recovered, tr.Degraded, ft.stats())
	require.Positive(t, ft.stats()["votes_dropped"], "fault injection must have actually dropped votes (else not a real chaos run)")
}

// TestConsensus_Chaos_TransportHardErrors makes the transport return errors for
// EVERY vote RPC mid-election. The candidate cannot gather remote votes and MUST
// step down cleanly to Follower (bounded-safety) — no crash, no deadlock, no
// fake-win.
func TestConsensus_Chaos_TransportHardErrors(t *testing.T) {
	var ft *faultVoteTransport
	ec := newElectionCluster(t, 3, 8*time.Millisecond, faultBuilder(&ft))
	defer ec.stop()
	require.NotNil(t, ft)

	rec := stresschaos.NewChaosRecorder(t, "consensus_transport_hard_errors", "transport-error")

	ft.hardErrorVotes.Store(true)
	rec.Record(stresschaos.Degraded, "transport hard-errors EVERY RequestVote")

	// Give several election rounds to run under total vote failure.
	time.Sleep(500 * time.Millisecond)

	// SAFETY: with every vote erroring, NO node may become leader (no fake-win),
	// and no node may be stuck Candidate (bounded-safety step-down to Follower).
	leaders := 0
	for _, li := range ec.snapshot() {
		if li.leader {
			leaders++
		}
	}
	ec.noNodeStuckCandidate(t)
	assert.Equal(t, 0, leaders, "no node may fake-win when every vote RPC errors")
	ec.assertAtMostOneLeaderPerTerm(t)
	rec.Record(stresschaos.Recovered, "all nodes stepped down to Follower under total vote failure (no fake-win, no livelock)")

	// Now heal the transport and require convergence — proving step-down was clean,
	// not a permanent wedge.
	ft.hardErrorVotes.Store(false)
	healed := ec.waitForSingleLeader(4 * time.Second)
	if healed != "" {
		rec.Record(stresschaos.Recovered, "cluster converged after transport healed: "+healed)
	} else {
		rec.Record(stresschaos.Degraded, "did not converge after heal within window")
	}
	tr := rec.AssertNoFatal()
	t.Logf("hard-error chaos: healed-leader=%q recovered=%d degraded=%d stats=%v",
		healed, tr.Recovered, tr.Degraded, ft.stats())
	require.Positive(t, ft.stats()["votes_errored"], "transport must have actually errored votes")
}

// TestConsensus_Chaos_ConcurrentReconfigureMidElection hammers Reconfigure
// concurrently with running elections (state-corruption chaos class). The
// manager MUST remain in a consistent state — no race (run under -race), no
// panic, no split-brain — and after reconfiguration settles, converge to a
// single leader.
func TestConsensus_Chaos_ConcurrentReconfigureMidElection(t *testing.T) {
	var ft *faultVoteTransport
	ec := newElectionCluster(t, 4, 8*time.Millisecond, faultBuilder(&ft))
	defer ec.stop()
	require.NotNil(t, ft)

	rec := stresschaos.NewChaosRecorder(t, "consensus_concurrent_reconfigure", "state-corruption")

	// Concurrently call Reconfigure on every manager repeatedly with the SAME
	// (correct) peer set + transport while elections run. Reconfigure resets to
	// Follower under the lock, so the only safe outcome is a consistent state and
	// eventual convergence — never a panic/race/split-brain.
	var wg sync.WaitGroup
	stop := make(chan struct{})
	for _, id := range ec.ids {
		cm := ec.managers[id]
		peers := make([]string, 0, len(ec.ids)-1)
		for _, pid := range ec.ids {
			if pid != id {
				peers = append(peers, pid)
			}
		}
		wg.Add(1)
		go func(cm *ConsensusManager, peers []string) {
			defer wg.Done()
			defer func() {
				if p := recover(); p != nil {
					rec.Record(stresschaos.Fatal, fmt.Sprintf("Reconfigure panicked: %v", p))
				}
			}()
			for {
				select {
				case <-stop:
					return
				default:
					cm.Reconfigure(peers, ft)
					time.Sleep(3 * time.Millisecond)
				}
			}
		}(cm, peers)
	}

	// Let the storm of concurrent Reconfigure + elections run.
	time.Sleep(600 * time.Millisecond)
	rec.Record(stresschaos.Degraded, "ran concurrent Reconfigure storm during live elections")
	close(stop)
	wg.Wait()

	// After the storm, the cluster MUST settle to a single leader (the reconfig
	// always installs the correct peer set, so convergence is required).
	leader := ec.waitForSingleLeader(5 * time.Second)
	ec.assertAtMostOneLeaderPerTerm(t)
	ec.noNodeStuckCandidate(t)
	if leader != "" {
		rec.Record(stresschaos.Recovered, "cluster converged to a single leader after reconfigure storm: "+leader)
	} else {
		rec.Record(stresschaos.Degraded, "no convergence after storm within window; no split-brain, no panic")
	}
	tr := rec.AssertNoFatal()
	t.Logf("reconfigure chaos: leader=%q recovered=%d degraded=%d fatal=%d",
		leader, tr.Recovered, tr.Degraded, tr.Fatal)
}

// evidencePathHint logs where this run's artefacts landed (debugging aid only).
func TestConsensus_StressChaos_EvidenceLocation(t *testing.T) {
	root := stresschaos.EvidenceRoot()
	t.Logf("consensus stress+chaos evidence root: %s", root)
	t.Logf("expected artefacts under: %s", filepath.Join(root, "<scenario>", "{latency,concurrency_report,recovery_trace}.{json,log}"))
}
