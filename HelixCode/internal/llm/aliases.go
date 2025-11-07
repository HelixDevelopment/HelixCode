package llm

import (
	"fmt"
	"strings"
	"sync"
)

// ModelAlias represents a model alias configuration
type ModelAlias struct {
	Alias       string   `yaml:"alias" json:"alias"`
	TargetModel string   `yaml:"target_model" json:"target_model"`
	Provider    string   `yaml:"provider" json:"provider"`
	Description string   `yaml:"description" json:"description"`
	Tags        []string `yaml:"tags" json:"tags"`
}

// AliasManager manages model aliases
type AliasManager struct {
	aliases          map[string]*ModelAlias // alias -> ModelAlias
	fuzzyThreshold   float64
	mu               sync.RWMutex
	providerAliases  map[string][]string // provider -> list of aliases
	caseInsensitive  bool
}

// NewAliasManager creates a new alias manager
func NewAliasManager(fuzzyThreshold float64) *AliasManager {
	if fuzzyThreshold <= 0 || fuzzyThreshold > 1 {
		fuzzyThreshold = 0.7 // Default 70% match threshold
	}

	return &AliasManager{
		aliases:         make(map[string]*ModelAlias),
		fuzzyThreshold:  fuzzyThreshold,
		providerAliases: make(map[string][]string),
		caseInsensitive: true,
	}
}

// AddAlias adds a new model alias
func (am *AliasManager) AddAlias(alias *ModelAlias) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if alias.Alias == "" {
		return fmt.Errorf("alias name cannot be empty")
	}

	if alias.TargetModel == "" {
		return fmt.Errorf("target model cannot be empty")
	}

	key := am.normalizeKey(alias.Alias)

	// Check for duplicate
	if existing, exists := am.aliases[key]; exists {
		return fmt.Errorf("alias '%s' already exists (targets: %s)", alias.Alias, existing.TargetModel)
	}

	am.aliases[key] = alias

	// Track by provider
	if alias.Provider != "" {
		am.providerAliases[alias.Provider] = append(am.providerAliases[alias.Provider], alias.Alias)
	}

	return nil
}

// RemoveAlias removes an alias
func (am *AliasManager) RemoveAlias(alias string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	key := am.normalizeKey(alias)

	modelAlias, exists := am.aliases[key]
	if !exists {
		return fmt.Errorf("alias '%s' not found", alias)
	}

	// Remove from provider aliases
	if modelAlias.Provider != "" {
		providerAliases := am.providerAliases[modelAlias.Provider]
		for i, a := range providerAliases {
			if am.normalizeKey(a) == key {
				am.providerAliases[modelAlias.Provider] = append(providerAliases[:i], providerAliases[i+1:]...)
				break
			}
		}
	}

	delete(am.aliases, key)
	return nil
}

// Resolve resolves an alias to the target model
func (am *AliasManager) Resolve(aliasOrModel string) (targetModel string, provider string, resolved bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	key := am.normalizeKey(aliasOrModel)

	// Exact match
	if alias, exists := am.aliases[key]; exists {
		return alias.TargetModel, alias.Provider, true
	}

	// Fuzzy match
	if am.fuzzyThreshold > 0 {
		bestMatch, score := am.fuzzyMatch(aliasOrModel)
		if score >= am.fuzzyThreshold && bestMatch != nil {
			return bestMatch.TargetModel, bestMatch.Provider, true
		}
	}

	// Not an alias, return as-is
	return aliasOrModel, "", false
}

// ResolveWithProvider resolves an alias for a specific provider
func (am *AliasManager) ResolveWithProvider(aliasOrModel string, preferredProvider string) (targetModel string, provider string, resolved bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	key := am.normalizeKey(aliasOrModel)

	// First try exact match with preferred provider
	if preferredProvider != "" {
		for _, alias := range am.aliases {
			if am.normalizeKey(alias.Alias) == key && alias.Provider == preferredProvider {
				return alias.TargetModel, alias.Provider, true
			}
		}
	}

	// Fall back to general resolution
	return am.Resolve(aliasOrModel)
}

// GetAlias retrieves alias details
func (am *AliasManager) GetAlias(alias string) (*ModelAlias, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	key := am.normalizeKey(alias)
	modelAlias, exists := am.aliases[key]
	return modelAlias, exists
}

// ListAliases returns all aliases
func (am *AliasManager) ListAliases() []*ModelAlias {
	am.mu.RLock()
	defer am.mu.RUnlock()

	aliases := make([]*ModelAlias, 0, len(am.aliases))
	for _, alias := range am.aliases {
		aliases = append(aliases, alias)
	}
	return aliases
}

// ListAliasesByProvider returns aliases for a specific provider
func (am *AliasManager) ListAliasesByProvider(provider string) []*ModelAlias {
	am.mu.RLock()
	defer am.mu.RUnlock()

	aliasNames := am.providerAliases[provider]
	aliases := make([]*ModelAlias, 0, len(aliasNames))

	for _, name := range aliasNames {
		if alias, exists := am.aliases[am.normalizeKey(name)]; exists {
			aliases = append(aliases, alias)
		}
	}

	return aliases
}

// SearchAliases searches aliases by tags or description
func (am *AliasManager) SearchAliases(query string) []*ModelAlias {
	am.mu.RLock()
	defer am.mu.RUnlock()

	query = strings.ToLower(query)
	matches := make([]*ModelAlias, 0)

	for _, alias := range am.aliases {
		// Check description
		if strings.Contains(strings.ToLower(alias.Description), query) {
			matches = append(matches, alias)
			continue
		}

		// Check tags
		for _, tag := range alias.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				matches = append(matches, alias)
				break
			}
		}
	}

	return matches
}

// Autocomplete provides alias name autocompletion
func (am *AliasManager) Autocomplete(partial string) []string {
	am.mu.RLock()
	defer am.mu.RUnlock()

	partial = strings.ToLower(partial)
	matches := make([]string, 0)

	for key, alias := range am.aliases {
		if strings.HasPrefix(key, partial) || strings.HasPrefix(strings.ToLower(alias.Alias), partial) {
			matches = append(matches, alias.Alias)
		}
	}

	return matches
}

// normalizeKey normalizes alias keys for comparison
func (am *AliasManager) normalizeKey(key string) string {
	if am.caseInsensitive {
		return strings.ToLower(strings.TrimSpace(key))
	}
	return strings.TrimSpace(key)
}

// fuzzyMatch performs fuzzy matching on aliases
func (am *AliasManager) fuzzyMatch(query string) (*ModelAlias, float64) {
	query = strings.ToLower(query)
	var bestMatch *ModelAlias
	var bestScore float64

	for _, alias := range am.aliases {
		score := am.calculateSimilarity(query, strings.ToLower(alias.Alias))
		if score > bestScore {
			bestScore = score
			bestMatch = alias
		}

		// Also check against target model
		targetScore := am.calculateSimilarity(query, strings.ToLower(alias.TargetModel))
		if targetScore > bestScore {
			bestScore = targetScore
			bestMatch = alias
		}
	}

	return bestMatch, bestScore
}

// calculateSimilarity calculates similarity between two strings (0.0 - 1.0)
func (am *AliasManager) calculateSimilarity(s1, s2 string) float64 {
	// Exact match
	if s1 == s2 {
		return 1.0
	}

	// Contains match
	if strings.Contains(s2, s1) || strings.Contains(s1, s2) {
		longer := s1
		shorter := s2
		if len(s2) > len(s1) {
			longer = s2
			shorter = s1
		}
		return float64(len(shorter)) / float64(len(longer))
	}

	// Levenshtein distance-based similarity
	distance := levenshteinDistance(s1, s2)
	maxLen := maxInt(len(s1), len(s2))
	if maxLen == 0 {
		return 0
	}

	return 1.0 - (float64(distance) / float64(maxLen))
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}

			matrix[i][j] = minInt3(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// maxInt returns the maximum of two integers
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// minInt3 returns the minimum of three integers
func minInt3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// SetFuzzyThreshold sets the fuzzy matching threshold
func (am *AliasManager) SetFuzzyThreshold(threshold float64) error {
	if threshold <= 0 || threshold > 1 {
		return fmt.Errorf("threshold must be between 0 and 1")
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	am.fuzzyThreshold = threshold
	return nil
}

// GetFuzzyThreshold returns the current fuzzy matching threshold
func (am *AliasManager) GetFuzzyThreshold() float64 {
	am.mu.RLock()
	defer am.mu.RUnlock()

	return am.fuzzyThreshold
}

// ExportAliases exports all aliases for backup/migration
func (am *AliasManager) ExportAliases() []*ModelAlias {
	return am.ListAliases()
}

// ImportAliases imports aliases from backup/migration
func (am *AliasManager) ImportAliases(aliases []*ModelAlias, overwrite bool) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	for _, alias := range aliases {
		key := am.normalizeKey(alias.Alias)

		if !overwrite {
			if _, exists := am.aliases[key]; exists {
				continue // Skip existing
			}
		}

		am.aliases[key] = alias

		// Track by provider
		if alias.Provider != "" {
			// Remove from old provider list if exists
			for provider, providerAliases := range am.providerAliases {
				for i, a := range providerAliases {
					if am.normalizeKey(a) == key {
						am.providerAliases[provider] = append(providerAliases[:i], providerAliases[i+1:]...)
						break
					}
				}
			}

			// Add to new provider list
			am.providerAliases[alias.Provider] = append(am.providerAliases[alias.Provider], alias.Alias)
		}
	}

	return nil
}

// Clear removes all aliases
func (am *AliasManager) Clear() {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.aliases = make(map[string]*ModelAlias)
	am.providerAliases = make(map[string][]string)
}

// Count returns the total number of aliases
func (am *AliasManager) Count() int {
	am.mu.RLock()
	defer am.mu.RUnlock()

	return len(am.aliases)
}
