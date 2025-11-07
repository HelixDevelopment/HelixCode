package scheduler

import (
	"sort"

	"dev.helix.code/tests/e2e/orchestrator/pkg"
)

// Scheduler handles test scheduling and filtering
type Scheduler struct{}

// NewScheduler creates a new scheduler
func NewScheduler() *Scheduler {
	return &Scheduler{}
}

// Schedule orders tests by priority (higher priority first)
func (s *Scheduler) Schedule(tests []*pkg.TestCase) []*pkg.TestCase {
	scheduled := make([]*pkg.TestCase, len(tests))
	copy(scheduled, tests)
	
	// Sort by priority (descending)
	sort.SliceStable(scheduled, func(i, j int) bool {
		return scheduled[i].Priority > scheduled[j].Priority
	})
	
	return scheduled
}

// FilterByTags filters tests that match ANY of the specified tags
func (s *Scheduler) FilterByTags(tests []*pkg.TestCase, tags []string) []*pkg.TestCase {
	if len(tags) == 0 {
		return tests
	}
	
	filtered := make([]*pkg.TestCase, 0)
	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}
	
	for _, test := range tests {
		for _, testTag := range test.Tags {
			if tagSet[testTag] {
				filtered = append(filtered, test)
				break
			}
		}
	}
	
	return filtered
}

// FilterByIDs filters tests by their IDs
func (s *Scheduler) FilterByIDs(tests []*pkg.TestCase, ids []string) []*pkg.TestCase {
	if len(ids) == 0 {
		return tests
	}
	
	filtered := make([]*pkg.TestCase, 0)
	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}
	
	for _, test := range tests {
		if idSet[test.ID] {
			filtered = append(filtered, test)
		}
	}
	
	return filtered
}
