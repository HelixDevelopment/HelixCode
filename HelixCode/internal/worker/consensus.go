// Package worker provides distributed worker management with consensus
package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// ConsensusManager handles distributed consensus for worker coordination
type ConsensusManager struct {
	nodeID      string
	peers       []string
	currentTerm int
	votedFor    string
	log         []LogEntry
	commitIndex int
	lastApplied int
	state       NodeState
	votes       map[string]bool
	mutex       sync.RWMutex

	// Channels for Raft protocol
	voteRequests   chan VoteRequest
	voteResponses  chan VoteResponse
	appendRequests chan AppendRequest
	heartbeatTimer *time.Timer
	electionTimer  *time.Ticker

	// Callbacks
	onLeaderElected func(string)
	onStateChanged  func(NodeState)
}

// NodeState represents the current state of a Raft node
type NodeState int

const (
	Follower NodeState = iota
	Candidate
	Leader
)

// LogEntry represents a log entry in Raft
type LogEntry struct {
	Term    int         `json:"term"`
	Index   int         `json:"index"`
	Command interface{} `json:"command"`
}

// VoteRequest represents a vote request in Raft
type VoteRequest struct {
	Term         int    `json:"term"`
	CandidateID  string `json:"candidate_id"`
	LastLogIndex int    `json:"last_log_index"`
	LastLogTerm  int    `json:"last_log_term"`
}

// VoteResponse represents a vote response in Raft
type VoteResponse struct {
	Term        int  `json:"term"`
	VoteGranted bool `json:"vote_granted"`
}

// AppendRequest represents an append entries request in Raft
type AppendRequest struct {
	Term         int        `json:"term"`
	LeaderID     string     `json:"leader_id"`
	PrevLogIndex int        `json:"prev_log_index"`
	PrevLogTerm  int        `json:"prev_log_term"`
	Entries      []LogEntry `json:"entries"`
	LeaderCommit int        `json:"leader_commit"`
}

// ConsensusConfig represents configuration for the consensus manager
type ConsensusConfig struct {
	NodeID            string
	Peers             []string
	HeartbeatInterval time.Duration
	ElectionTimeout   time.Duration
	OnLeaderElected   func(string)
	OnStateChanged    func(NodeState)
}

// NewConsensusManager creates a new consensus manager
func NewConsensusManager(config ConsensusConfig) *ConsensusManager {
	cm := &ConsensusManager{
		nodeID:          config.NodeID,
		peers:           config.Peers,
		state:           Follower,
		votes:           make(map[string]bool),
		voteRequests:    make(chan VoteRequest, 100),
		voteResponses:   make(chan VoteResponse, 100),
		appendRequests:  make(chan AppendRequest, 100),
		onLeaderElected: config.OnLeaderElected,
		onStateChanged:  config.OnStateChanged,
	}

	if config.HeartbeatInterval == 0 {
		config.HeartbeatInterval = 100 * time.Millisecond
	}
	if config.ElectionTimeout == 0 {
		config.ElectionTimeout = 500 * time.Millisecond
	}

	cm.heartbeatTimer = time.NewTimer(config.HeartbeatInterval)
	cm.electionTimer = time.NewTicker(config.ElectionTimeout)

	return cm
}

// Start begins the consensus protocol
func (cm *ConsensusManager) Start(ctx context.Context) error {
	log.Printf("Starting Raft consensus for node %s", cm.nodeID)

	go cm.run(ctx)
	return nil
}

// Stop stops the consensus protocol
func (cm *ConsensusManager) Stop() {
	if cm.heartbeatTimer != nil {
		cm.heartbeatTimer.Stop()
	}
	if cm.electionTimer != nil {
		cm.electionTimer.Stop()
	}
}

// IsLeader returns true if this node is the current leader
func (cm *ConsensusManager) IsLeader() bool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.state == Leader
}

// GetLeader returns the current leader ID
func (cm *ConsensusManager) GetLeader() string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	if cm.state == Leader {
		return cm.nodeID
	}
	// In a real implementation, this would track the known leader
	return ""
}

// GetCurrentTerm returns the current term
func (cm *ConsensusManager) GetCurrentTerm() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.currentTerm
}

// run is the main consensus loop
func (cm *ConsensusManager) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-cm.electionTimer.C:
			cm.startElection()
		case <-cm.heartbeatTimer.C:
			if cm.IsLeader() {
				cm.sendHeartbeats()
			}
		case req := <-cm.voteRequests:
			cm.handleVoteRequest(req)
		case resp := <-cm.voteResponses:
			cm.handleVoteResponse(resp)
		case req := <-cm.appendRequests:
			cm.handleAppendRequest(req)
		}
	}
}

// startElection starts a new election
func (cm *ConsensusManager) startElection() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if cm.state == Leader {
		return // Already leader
	}

	cm.state = Candidate
	cm.currentTerm++
	cm.votedFor = cm.nodeID
	cm.votes = make(map[string]bool)
	cm.votes[cm.nodeID] = true

	log.Printf("Node %s starting election for term %d", cm.nodeID, cm.currentTerm)

	// In a real implementation, send to peers
	// For now, simulate quorum
	if len(cm.peers) == 0 {
		// Single node cluster - become leader immediately
		cm.becomeLeader()
	}
}

// handleVoteRequest handles incoming vote requests
func (cm *ConsensusManager) handleVoteRequest(req VoteRequest) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	resp := VoteResponse{
		Term:        cm.currentTerm,
		VoteGranted: false,
	}

	if req.Term > cm.currentTerm {
		cm.currentTerm = req.Term
		cm.state = Follower
		cm.votedFor = ""
	}

	if req.Term == cm.currentTerm &&
		(cm.votedFor == "" || cm.votedFor == req.CandidateID) &&
		cm.isLogUpToDate(req.LastLogIndex, req.LastLogTerm) {

		cm.votedFor = req.CandidateID
		resp.VoteGranted = true
		log.Printf("Node %s granted vote to %s for term %d", cm.nodeID, req.CandidateID, req.Term)
	}

	// Send response (in real implementation)
	_ = resp
}

// handleVoteResponse handles incoming vote responses
func (cm *ConsensusManager) handleVoteResponse(resp VoteResponse) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if resp.Term > cm.currentTerm {
		cm.currentTerm = resp.Term
		cm.state = Follower
		cm.votedFor = ""
		return
	}

	if cm.state == Candidate && resp.Term == cm.currentTerm && resp.VoteGranted {
		// In a real implementation, track by responding node ID
		// For now, just increment a counter
		currentVotes := 0
		for _, granted := range cm.votes {
			if granted {
				currentVotes++
			}
		}
		currentVotes++ // Count this vote

		// Check if we have quorum
		voteCount := currentVotes

		totalNodes := len(cm.peers) + 1 // peers + self
		if voteCount > totalNodes/2 {
			cm.becomeLeader()
		}
	}
}

// handleAppendRequest handles incoming append entries requests
func (cm *ConsensusManager) handleAppendRequest(req AppendRequest) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if req.Term > cm.currentTerm {
		cm.currentTerm = req.Term
		cm.state = Follower
		cm.votedFor = ""
	}

	// Reset election timer
	cm.electionTimer.Reset(500 * time.Millisecond)
}

// sendHeartbeats sends heartbeats to followers (when leader)
func (cm *ConsensusManager) sendHeartbeats() {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if cm.state != Leader {
		return
	}

	// Send append entries requests to all peers
	appendReq := AppendRequest{
		Term:         cm.currentTerm,
		LeaderID:     cm.nodeID,
		PrevLogIndex: len(cm.log) - 1,
		PrevLogTerm:  0,
		Entries:      []LogEntry{}, // Empty for heartbeat
		LeaderCommit: cm.commitIndex,
	}

	if len(cm.log) > 0 {
		appendReq.PrevLogTerm = cm.log[len(cm.log)-1].Term
	}

	// In a real implementation, send to peers
	log.Printf("Leader %s sending heartbeat for term %d", cm.nodeID, cm.currentTerm)
	_ = appendReq
}

// becomeLeader transitions this node to leader state
func (cm *ConsensusManager) becomeLeader() {
	cm.state = Leader
	log.Printf("Node %s became leader for term %d", cm.nodeID, cm.currentTerm)

	if cm.onLeaderElected != nil {
		cm.onLeaderElected(cm.nodeID)
	}

	if cm.onStateChanged != nil {
		cm.onStateChanged(Leader)
	}

	// Reset heartbeat timer
	cm.heartbeatTimer.Reset(100 * time.Millisecond)
}

// isLogUpToDate checks if candidate's log is at least as up-to-date as ours
func (cm *ConsensusManager) isLogUpToDate(lastLogIndex, lastLogTerm int) bool {
	ourLastIndex := len(cm.log) - 1
	ourLastTerm := 0
	if ourLastIndex >= 0 {
		ourLastTerm = cm.log[ourLastIndex].Term
	}

	if lastLogTerm > ourLastTerm {
		return true
	}
	if lastLogTerm == ourLastTerm && lastLogIndex >= ourLastIndex {
		return true
	}
	return false
}

// ProposeTask proposes a task for consensus (leader only)
func (cm *ConsensusManager) ProposeTask(task interface{}) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if cm.state != Leader {
		return fmt.Errorf("not leader - cannot propose tasks")
	}

	// Add to log
	entry := LogEntry{
		Term:    cm.currentTerm,
		Index:   len(cm.log),
		Command: task,
	}
	cm.log = append(cm.log, entry)

	log.Printf("Leader %s proposed task: %+v", cm.nodeID, task)
	return nil
}

// GetQuorumSize returns the quorum size for the current cluster
func (cm *ConsensusManager) GetQuorumSize() int {
	return (len(cm.peers)+1)/2 + 1
}
