package compression

import (
	"time"
)

// RetentionPolicy defines rules for what messages to retain during compression
type RetentionPolicy struct {
	rules       []RetentionRule
	recentCount int
	minAge      time.Duration
}

// RetentionRule defines a single retention rule
type RetentionRule struct {
	Priority int
	Match    func(*Message, MessagePosition) bool
	Action   RetentionAction
	Reason   string
}

// RetentionAction specifies what to do with a message
type RetentionAction int

const (
	// ActionRetain keeps the message
	ActionRetain RetentionAction = iota
	// ActionCompress allows the message to be compressed
	ActionCompress
	// ActionRemove explicitly removes the message
	ActionRemove
)

// String returns the string representation of a retention action
func (ra RetentionAction) String() string {
	switch ra {
	case ActionRetain:
		return "retain"
	case ActionCompress:
		return "compress"
	case ActionRemove:
		return "remove"
	default:
		return "unknown"
	}
}

// MessagePosition provides context about a message's position in the conversation
type MessagePosition struct {
	Index       int
	IsFirst     bool
	IsLast      bool
	AgeDuration time.Duration
	IsRecent    bool
}

// DefaultRetentionPolicy returns a default retention policy
func DefaultRetentionPolicy() *RetentionPolicy {
	return &RetentionPolicy{
		rules:       defaultRetentionRules(),
		recentCount: 10,
		minAge:      30 * time.Minute,
	}
}

// NewRetentionPolicy creates a new retention policy with custom rules
func NewRetentionPolicy(recentCount int, minAge time.Duration, customRules []RetentionRule) *RetentionPolicy {
	rules := defaultRetentionRules()
	rules = append(rules, customRules...)

	return &RetentionPolicy{
		rules:       rules,
		recentCount: recentCount,
		minAge:      minAge,
	}
}

// ShouldRetain checks if a message should be retained based on the policy
func (rp *RetentionPolicy) ShouldRetain(msg *Message, position MessagePosition) bool {
	// Sort rules by priority (higher priority first)
	// This is done on the fly, but could be optimized by sorting once
	for _, rule := range rp.getSortedRules() {
		if rule.Match(msg, position) {
			return rule.Action == ActionRetain
		}
	}

	// Default to not retaining if no rule matches
	return false
}

// GetMatchingRule returns the first matching rule for a message
func (rp *RetentionPolicy) GetMatchingRule(msg *Message, position MessagePosition) *RetentionRule {
	for _, rule := range rp.getSortedRules() {
		if rule.Match(msg, position) {
			return &rule
		}
	}
	return nil
}

// AddRule adds a custom retention rule
func (rp *RetentionPolicy) AddRule(rule RetentionRule) {
	rp.rules = append(rp.rules, rule)
}

// RemoveRule removes a rule by reason
func (rp *RetentionPolicy) RemoveRule(reason string) {
	var filtered []RetentionRule
	for _, rule := range rp.rules {
		if rule.Reason != reason {
			filtered = append(filtered, rule)
		}
	}
	rp.rules = filtered
}

// GetRules returns all retention rules
func (rp *RetentionPolicy) GetRules() []RetentionRule {
	return rp.rules
}

// SetRecentCount sets the number of recent messages to retain
func (rp *RetentionPolicy) SetRecentCount(count int) {
	rp.recentCount = count
}

// GetRecentCount returns the number of recent messages to retain
func (rp *RetentionPolicy) GetRecentCount() int {
	return rp.recentCount
}

// SetMinAge sets the minimum age before a message can be compressed
func (rp *RetentionPolicy) SetMinAge(duration time.Duration) {
	rp.minAge = duration
}

// GetMinAge returns the minimum age before compression
func (rp *RetentionPolicy) GetMinAge() time.Duration {
	return rp.minAge
}

// getSortedRules returns rules sorted by priority (highest first)
func (rp *RetentionPolicy) getSortedRules() []RetentionRule {
	// Create a copy to avoid modifying the original
	sorted := make([]RetentionRule, len(rp.rules))
	copy(sorted, rp.rules)

	// Simple bubble sort by priority (descending)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].Priority < sorted[j].Priority {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}

// defaultRetentionRules returns the default set of retention rules
func defaultRetentionRules() []RetentionRule {
	return []RetentionRule{
		{
			Priority: 10,
			Match: func(msg *Message, pos MessagePosition) bool {
				return msg.Role == RoleSystem
			},
			Action: ActionRetain,
			Reason: "system_messages",
		},
		{
			Priority: 9,
			Match: func(msg *Message, pos MessagePosition) bool {
				return msg.Pinned
			},
			Action: ActionRetain,
			Reason: "pinned_messages",
		},
		{
			Priority: 8,
			Match: func(msg *Message, pos MessagePosition) bool {
				return msg.Important
			},
			Action: ActionRetain,
			Reason: "important_messages",
		},
		{
			Priority: 7,
			Match: func(msg *Message, pos MessagePosition) bool {
				return msg.Metadata.Type == TypeCommand
			},
			Action: ActionRetain,
			Reason: "command_messages",
		},
		{
			Priority: 6,
			Match: func(msg *Message, pos MessagePosition) bool {
				return pos.IsRecent
			},
			Action: ActionRetain,
			Reason: "recent_messages",
		},
		{
			Priority: 5,
			Match: func(msg *Message, pos MessagePosition) bool {
				return msg.Metadata.HasError
			},
			Action: ActionRetain,
			Reason: "error_messages",
		},
		{
			Priority: 4,
			Match: func(msg *Message, pos MessagePosition) bool {
				return msg.Metadata.Type == TypeToolCall || msg.Metadata.Type == TypeToolResult
			},
			Action: ActionRetain,
			Reason: "tool_messages",
		},
		{
			Priority: 3,
			Match: func(msg *Message, pos MessagePosition) bool {
				return len(msg.Metadata.FilePaths) > 0
			},
			Action: ActionRetain,
			Reason: "messages_with_files",
		},
		{
			Priority: 2,
			Match: func(msg *Message, pos MessagePosition) bool {
				return msg.Metadata.CodeBlocks > 0
			},
			Action: ActionRetain,
			Reason: "messages_with_code",
		},
	}
}

// ConservativePolicy returns a policy that retains more messages
func ConservativePolicy() *RetentionPolicy {
	return &RetentionPolicy{
		rules:       defaultRetentionRules(),
		recentCount: 30,
		minAge:      1 * time.Hour,
	}
}

// AggressivePolicy returns a policy that retains fewer messages
func AggressivePolicy() *RetentionPolicy {
	// Aggressive policy only keeps essential messages
	rules := []RetentionRule{
		{
			Priority: 10,
			Match: func(msg *Message, pos MessagePosition) bool {
				return msg.Role == RoleSystem
			},
			Action: ActionRetain,
			Reason: "system_messages",
		},
		{
			Priority: 9,
			Match: func(msg *Message, pos MessagePosition) bool {
				return msg.Pinned
			},
			Action: ActionRetain,
			Reason: "pinned_messages",
		},
		{
			Priority: 5,
			Match: func(msg *Message, pos MessagePosition) bool {
				return pos.IsRecent
			},
			Action: ActionRetain,
			Reason: "recent_messages",
		},
	}

	return &RetentionPolicy{
		rules:       rules,
		recentCount: 5,
		minAge:      10 * time.Minute,
	}
}

// BalancedPolicy returns a balanced retention policy (same as default)
func BalancedPolicy() *RetentionPolicy {
	return DefaultRetentionPolicy()
}

// PolicyPreset represents different policy presets
type PolicyPreset string

const (
	// PresetConservative retains more messages
	PresetConservative PolicyPreset = "conservative"
	// PresetBalanced is the default, balanced approach
	PresetBalanced PolicyPreset = "balanced"
	// PresetAggressive retains fewer messages for maximum compression
	PresetAggressive PolicyPreset = "aggressive"
	// PresetCustom allows for fully custom policies
	PresetCustom PolicyPreset = "custom"
)

// GetPolicyByPreset returns a policy based on a preset
func GetPolicyByPreset(preset PolicyPreset) *RetentionPolicy {
	switch preset {
	case PresetConservative:
		return ConservativePolicy()
	case PresetBalanced:
		return BalancedPolicy()
	case PresetAggressive:
		return AggressivePolicy()
	case PresetCustom:
		return &RetentionPolicy{
			rules:       []RetentionRule{},
			recentCount: 10,
			minAge:      30 * time.Minute,
		}
	default:
		return DefaultRetentionPolicy()
	}
}

// CalculateMessagePosition calculates the position context for a message
func CalculateMessagePosition(index int, totalMessages int, msgTime time.Time, recentCount int) MessagePosition {
	now := time.Now()
	age := now.Sub(msgTime)
	isRecent := index >= totalMessages-recentCount

	return MessagePosition{
		Index:       index,
		IsFirst:     index == 0,
		IsLast:      index == totalMessages-1,
		AgeDuration: age,
		IsRecent:    isRecent,
	}
}

// EvaluatePolicy evaluates how many messages would be retained/compressed with this policy
func EvaluatePolicy(policy *RetentionPolicy, messages []*Message) (retained, compressed int) {
	retained = 0
	compressed = 0

	for i, msg := range messages {
		position := CalculateMessagePosition(i, len(messages), msg.Timestamp, policy.recentCount)
		if policy.ShouldRetain(msg, position) {
			retained++
		} else {
			compressed++
		}
	}

	return retained, compressed
}

// PolicyStats provides statistics about policy application
type PolicyStats struct {
	TotalMessages      int
	RetainedMessages   int
	CompressedMessages int
	RetentionRate      float64
	RuleMatches        map[string]int
}

// AnalyzePolicy analyzes how a policy would affect a set of messages
func AnalyzePolicy(policy *RetentionPolicy, messages []*Message) *PolicyStats {
	stats := &PolicyStats{
		TotalMessages: len(messages),
		RuleMatches:   make(map[string]int),
	}

	for i, msg := range messages {
		position := CalculateMessagePosition(i, len(messages), msg.Timestamp, policy.recentCount)

		if policy.ShouldRetain(msg, position) {
			stats.RetainedMessages++

			// Track which rule matched
			rule := policy.GetMatchingRule(msg, position)
			if rule != nil {
				stats.RuleMatches[rule.Reason]++
			}
		} else {
			stats.CompressedMessages++
		}
	}

	if stats.TotalMessages > 0 {
		stats.RetentionRate = float64(stats.RetainedMessages) / float64(stats.TotalMessages)
	}

	return stats
}

// PolicyBuilder helps construct custom retention policies
type PolicyBuilder struct {
	policy *RetentionPolicy
}

// NewPolicyBuilder creates a new policy builder
func NewPolicyBuilder() *PolicyBuilder {
	return &PolicyBuilder{
		policy: &RetentionPolicy{
			rules:       []RetentionRule{},
			recentCount: 10,
			minAge:      30 * time.Minute,
		},
	}
}

// WithRecentCount sets the number of recent messages to retain
func (pb *PolicyBuilder) WithRecentCount(count int) *PolicyBuilder {
	pb.policy.recentCount = count
	return pb
}

// WithMinAge sets the minimum age before compression
func (pb *PolicyBuilder) WithMinAge(duration time.Duration) *PolicyBuilder {
	pb.policy.minAge = duration
	return pb
}

// AddRule adds a retention rule
func (pb *PolicyBuilder) AddRule(rule RetentionRule) *PolicyBuilder {
	pb.policy.rules = append(pb.policy.rules, rule)
	return pb
}

// WithDefaultRules includes the default retention rules
func (pb *PolicyBuilder) WithDefaultRules() *PolicyBuilder {
	pb.policy.rules = append(pb.policy.rules, defaultRetentionRules()...)
	return pb
}

// Build returns the constructed policy
func (pb *PolicyBuilder) Build() *RetentionPolicy {
	return pb.policy
}
