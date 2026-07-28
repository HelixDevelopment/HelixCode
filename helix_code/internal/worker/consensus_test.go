package worker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsensusManager_NewConsensusManager(t *testing.T) {
	config := ConsensusConfig{
		NodeID:            "node-1",
		Peers:             []string{"node-2", "node-3"},
		HeartbeatInterval: 50 * time.Millisecond,
		ElectionTimeout:   100 * time.Millisecond,
	}

	cm := NewConsensusManager(config)

	assert.Equal(t, "node-1", cm.nodeID)
	assert.Equal(t, []string{"node-2", "node-3"}, cm.peers)
	assert.Equal(t, Follower, cm.state)
	assert.Equal(t, 0, cm.currentTerm)
	assert.NotNil(t, cm.heartbeatTimer)
	assert.NotNil(t, cm.electionTimer)
}

func TestConsensusManager_StartStop(t *testing.T) {
	cm := NewConsensusManager(ConsensusConfig{
		NodeID:          "node-1",
		Peers:           []string{},
		ElectionTimeout: 50 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := cm.Start(ctx)
	require.NoError(t, err)

	// Should become leader in single-node cluster
	time.Sleep(100 * time.Millisecond)
	assert.True(t, cm.IsLeader())
	assert.Equal(t, "node-1", cm.GetLeader())

	cm.Stop()
}

func TestConsensusManager_SingleNodeLeaderElection(t *testing.T) {
	cm := NewConsensusManager(ConsensusConfig{
		NodeID:          "node-1",
		Peers:           []string{},
		ElectionTimeout: 50 * time.Millisecond,
	})

	ctx := context.Background()
	err := cm.Start(ctx)
	require.NoError(t, err)
	defer cm.Stop()

	// Wait for election
	time.Sleep(100 * time.Millisecond)

	assert.True(t, cm.IsLeader())
	assert.Equal(t, "node-1", cm.GetLeader())
	assert.Equal(t, Leader, cm.state)
}

func TestConsensusManager_VoteRequestHandling(t *testing.T) {
	cm := NewConsensusManager(ConsensusConfig{
		NodeID: "node-1",
		Peers:  []string{"node-2"},
	})

	// Test vote request with higher term
	req := VoteRequest{
		Term:         2,
		CandidateID:  "node-2",
		LastLogIndex: 0,
		LastLogTerm:  0,
	}

	cm.handleVoteRequest(req)

	assert.Equal(t, 2, cm.currentTerm)
	assert.Equal(t, Follower, cm.state)
	// votedFor should be set to the candidate since we grant the vote
	assert.Equal(t, "node-2", cm.votedFor)
}

func TestConsensusManager_VoteResponseHandling(t *testing.T) {
	cm := NewConsensusManager(ConsensusConfig{
		NodeID: "node-1",
		Peers:  []string{"node-2", "node-3"},
	})

	// Set up as candidate
	cm.state = Candidate
	cm.currentTerm = 1
	cm.votes = make(map[string]bool)
	cm.votes["node-1"] = true

	// Test vote response
	resp := VoteResponse{
		Term:        1,
		VoteGranted: true,
	}

	cm.handleVoteResponse(resp)

	// In simplified implementation, we don't track peer votes
	// Only self vote is tracked
	voteCount := 0
	for _, granted := range cm.votes {
		if granted {
			voteCount++
		}
	}
	assert.Equal(t, 1, voteCount) // Only self vote tracked in simplified version
}

func TestConsensusManager_AppendRequestHandling(t *testing.T) {
	cm := NewConsensusManager(ConsensusConfig{
		NodeID: "node-1",
		Peers:  []string{},
	})

	// Test append request with higher term
	req := AppendRequest{
		Term:     2,
		LeaderID: "node-2",
	}

	cm.handleAppendRequest(req)

	assert.Equal(t, 2, cm.currentTerm)
	assert.Equal(t, Follower, cm.state)
}

func TestConsensusManager_LogUpToDate(t *testing.T) {
	cm := NewConsensusManager(ConsensusConfig{
		NodeID: "node-1",
	})

	// Test with empty log
	assert.True(t, cm.isLogUpToDate(0, 0))

	// Add some log entries
	cm.log = append(cm.log, LogEntry{Term: 1, Index: 0})
	cm.log = append(cm.log, LogEntry{Term: 1, Index: 1})

	// Test with higher term
	assert.True(t, cm.isLogUpToDate(1, 2))

	// Test with same term, higher index
	assert.True(t, cm.isLogUpToDate(2, 1))

	// Test with lower term
	assert.False(t, cm.isLogUpToDate(0, 0))
}

func TestConsensusManager_ProposeTask(t *testing.T) {
	cm := NewConsensusManager(ConsensusConfig{
		NodeID: "node-1",
	})

	// Should fail when not leader. Round-184 CONST-046: the literal
	// "not leader - cannot propose tasks" is now resolved through the
	// internal/worker i18n Translator seam (NoopTranslator echoes the
	// message ID); assert on the ID substring instead.
	err := cm.ProposeTask("test task")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "internal_worker_not_leader")

	// Set as leader
	cm.state = Leader

	// Should succeed when leader
	err = cm.ProposeTask("test task")
	assert.NoError(t, err)
	assert.Len(t, cm.log, 1)
	assert.Equal(t, "test task", cm.log[0].Command)
}

func TestConsensusManager_GetQuorumSize(t *testing.T) {
	tests := []struct {
		name     string
		peers    []string
		expected int
	}{
		{"single node", []string{}, 1},
		{"two nodes", []string{"node-2"}, 2},
		{"three nodes", []string{"node-2", "node-3"}, 2},
		{"four nodes", []string{"node-2", "node-3", "node-4"}, 3},
		{"five nodes", []string{"node-2", "node-3", "node-4", "node-5"}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := NewConsensusManager(ConsensusConfig{
				NodeID: "node-1",
				Peers:  tt.peers,
			})

			assert.Equal(t, tt.expected, cm.GetQuorumSize())
		})
	}
}

func TestConsensusManager_StateTransitions(t *testing.T) {
	// §11.4.120 reconciliation: this test previously asserted that
	// startElection() with peers but no transport LEFT the node Candidate.
	// That was the pre-fix LIVELOCK behaviour (a node stuck Candidate
	// forever). The bounded-safety fix now steps such a node DOWN to
	// Follower, so the test is reconciled to assert the NEW correct
	// mechanism while still exercising the full Follower→Candidate→Leader
	// transition via the candidacy + becomeLeader path.
	cm := NewConsensusManager(ConsensusConfig{
		NodeID: "node-1",
		Peers:  []string{"node-2"},
	})

	// Start as follower
	assert.Equal(t, Follower, cm.state)

	// A multi-peer election with NO transport cannot gather a quorum, so
	// the round MUST end with a clean step-down to Follower — NEVER leave
	// the node stuck as Candidate (the old livelock).
	cm.startElection()
	assert.Equal(t, Follower, cm.state, "bounded-safety: no-transport multi-peer election steps down, never stuck Candidate")
	assert.Equal(t, 1, cm.currentTerm, "election round still bumped the term")

	// The candidate→leader transition itself is exercised directly: mark
	// candidate, then win.
	cm.mutex.Lock()
	cm.state = Candidate
	cm.mutex.Unlock()
	assert.Equal(t, Candidate, cm.state)

	cm.mutex.Lock()
	cm.becomeLeader()
	cm.mutex.Unlock()
	assert.Equal(t, Leader, cm.state)
	assert.True(t, cm.IsLeader())
}

func TestConsensusManager_HeartbeatMechanism(t *testing.T) {
	cm := NewConsensusManager(ConsensusConfig{
		NodeID:            "node-1",
		Peers:             []string{},
		HeartbeatInterval: 10 * time.Millisecond,
		ElectionTimeout:   20 * time.Millisecond,
	})

	// The ctx budget bounds the lifetime of the run() goroutine (consensus.go:172
	// starts it on a child of this ctx; consensus.go:305 returns on ctx.Done()).
	// It MUST exceed the poll budget below, otherwise a run() goroutine that the
	// host scheduler starts late is cancelled before it can ever fire the
	// election timer — that was the actual failure mode of the previous
	// 100ms-ctx / 50ms-fixed-sleep form.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := cm.Start(ctx)
	require.NoError(t, err)
	defer cm.Stop()

	// The assertion is state-shaped ("this peerless node elects itself"), so
	// poll for the state transition instead of sleeping a fixed window. A fixed
	// sleep asserts the far stronger and unintended claim "the transition
	// happens within exactly 50ms of wall clock", which host load breaks while
	// the code under test is correct (§11.4.50). The bounded poll fails only if
	// leadership is never reached — the property actually under test — and it
	// returns as soon as the transition is observed, so it is also faster on a
	// quiet host. Same fixed-sleep→poll migration the package already applies in
	// tests/stresschaos/stresschaos.go settleGoroutines() for HXC-144.
	require.Eventually(t, cm.IsLeader, 2*time.Second, 2*time.Millisecond,
		"a single-node (peerless) cluster must elect itself leader")

	// Leadership is terminal for a peerless node: nothing can step it down, so
	// these post-conditions are deterministic once Eventually has observed the
	// transition.
	assert.True(t, cm.IsLeader())
	assert.Equal(t, "node-1", cm.GetLeader(), "a leader advertises its own node ID")
}

func TestConsensusManager_GetCurrentTerm(t *testing.T) {
	cm := NewConsensusManager(ConsensusConfig{
		NodeID: "node-1",
		Peers:  []string{},
	})

	// Initial term should be 0
	assert.Equal(t, 0, cm.GetCurrentTerm())

	// After election, term should increase
	cm.currentTerm = 5
	assert.Equal(t, 5, cm.GetCurrentTerm())
}

func TestConsensusManager_GetLeader_NoLeader(t *testing.T) {
	cm := NewConsensusManager(ConsensusConfig{
		NodeID: "node-1",
		Peers:  []string{"node-2"},
	})

	// Initially no leader
	leader := cm.GetLeader()
	assert.Equal(t, "", leader)
}

func TestConsensusManager_VoteResponseHigherTerm(t *testing.T) {
	cm := NewConsensusManager(ConsensusConfig{
		NodeID: "node-1",
		Peers:  []string{"node-2", "node-3"},
	})

	// Set up as candidate
	cm.state = Candidate
	cm.currentTerm = 1
	cm.votes = make(map[string]bool)

	// Test vote response with higher term (should revert to follower)
	resp := VoteResponse{
		Term:        5,
		VoteGranted: false,
	}

	cm.handleVoteResponse(resp)

	// Should become follower due to higher term
	assert.Equal(t, Follower, cm.state)
	assert.Equal(t, 5, cm.currentTerm)
}

func TestConsensusManager_BecomeLeader_WithPeers(t *testing.T) {
	// §11.4.120 reconciliation: previously asserted startElection() left the
	// node Candidate (the pre-fix livelock). The bounded-safety fix steps a
	// no-transport multi-peer candidate down to Follower. The test now
	// asserts that NEW correct behaviour, then exercises becomeLeader().
	cm := NewConsensusManager(ConsensusConfig{
		NodeID:            "node-1",
		Peers:             []string{"node-2", "node-3"},
		HeartbeatInterval: 10 * time.Millisecond,
		ElectionTimeout:   50 * time.Millisecond,
	})

	// No-transport multi-peer election → clean step-down to Follower.
	cm.startElection()
	assert.Equal(t, Follower, cm.state, "bounded-safety: steps down, not stuck Candidate")

	// Drive the winning transition explicitly.
	cm.mutex.Lock()
	cm.state = Candidate
	cm.becomeLeader()
	cm.mutex.Unlock()

	assert.Equal(t, Leader, cm.state)
	// GetLeader() returns nodeID when state is Leader
	assert.Equal(t, "node-1", cm.GetLeader())
}

func TestConsensusManager_SendHeartbeats(t *testing.T) {
	cm := NewConsensusManager(ConsensusConfig{
		NodeID:            "node-1",
		Peers:             []string{"node-2"},
		HeartbeatInterval: 10 * time.Millisecond,
	})

	cm.state = Leader
	cm.currentTerm = 1

	// sendHeartbeats should not panic with mock peers
	// In real usage, this would send RPC to peers
	cm.sendHeartbeats()

	// Verify state unchanged
	assert.Equal(t, Leader, cm.state)
	// Leader should be identified via GetLeader()
	assert.Equal(t, "node-1", cm.GetLeader())
}
