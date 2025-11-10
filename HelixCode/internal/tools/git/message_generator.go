package git

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"dev.helix.code/internal/llm"
	"github.com/google/uuid"
)

// MessageGenerator generates commit messages using LLM
type MessageGenerator struct {
	llmProvider llm.Provider
	analyzer    *DiffAnalyzer
	cache       *MessageCache
	mu          sync.RWMutex
}

// MessageRequest configures message generation
type MessageRequest struct {
	Diffs          []*Diff
	Format         MessageFormat
	Language       string
	Context        CommitContext
	MaxLength      int
	IncludeDetails bool
}

// Message represents a generated commit message
type Message struct {
	Subject    string
	Body       string
	Footer     string
	Format     MessageFormat
	Confidence float64
	Analysis   *DiffAnalysis
}

// MessageFormat specifies commit message format
type MessageFormat int

const (
	FormatConventional MessageFormat = iota // Conventional Commits
	FormatSemantic                          // Semantic Commit Messages
	FormatAngular                           // Angular style
	FormatCustom                            // Custom template
)

// Diff represents a file diff
type Diff struct {
	Path  string
	Hunks []*DiffHunk
}

// DiffHunk represents a hunk in a diff
type DiffHunk struct {
	Header string
	Lines  []DiffLine
}

// DiffLine represents a line in a diff
type DiffLine struct {
	Type    LineType
	Content string
	LineNo  int
}

// LineType represents the type of diff line
type LineType int

const (
	LineContext LineType = iota
	LineAdd
	LineDelete
)

// DiffAnalysis contains analysis results
type DiffAnalysis struct {
	Files      []*FileAnalysis
	Summary    *ChangeSummary
	ChangeType ChangeType
	Scope      string
}

// FileAnalysis contains per-file analysis
type FileAnalysis struct {
	Path       string
	Language   string
	Summary    *ChangeSummary
	Functions  []*FunctionChange
	Complexity int
}

// ChangeSummary summarizes changes
type ChangeSummary struct {
	LinesAdded   int
	LinesDeleted int
	FilesChanged int

	// Semantic changes
	FunctionsAdded    []string
	FunctionsModified []string
	FunctionsDeleted  []string

	// Code characteristics
	TestsAdded    bool
	DocsModified  bool
	ConfigChanged bool
}

// ChangeType categorizes the change
type ChangeType string

const (
	TypeFeat     ChangeType = "feat"     // New feature
	TypeFix      ChangeType = "fix"      // Bug fix
	TypeDocs     ChangeType = "docs"     // Documentation
	TypeStyle    ChangeType = "style"    // Formatting
	TypeRefactor ChangeType = "refactor" // Code refactoring
	TypePerf     ChangeType = "perf"     // Performance
	TypeTest     ChangeType = "test"     // Tests
	TypeBuild    ChangeType = "build"    // Build system
	TypeCI       ChangeType = "ci"       // CI/CD
	TypeChore    ChangeType = "chore"    // Other
)

// FunctionChange represents a function modification
type FunctionChange struct {
	Name         string
	Type         string // added, modified, deleted
	LinesChanged int
}

// NewMessageGenerator creates a new message generator
func NewMessageGenerator(llmProvider llm.Provider) *MessageGenerator {
	return &MessageGenerator{
		llmProvider: llmProvider,
		analyzer:    NewDiffAnalyzer(),
		cache:       NewMessageCache(15 * time.Minute),
	}
}

// Generate creates a commit message from diffs
func (mg *MessageGenerator) Generate(ctx context.Context, req MessageRequest) (*Message, error) {
	mg.mu.Lock()
	defer mg.mu.Unlock()

	// Check cache
	diffHash := computeDiffHash(req.Diffs)
	if cached, ok := mg.cache.Get(diffHash); ok {
		return cached, nil
	}

	// Analyze diffs
	analysis, err := mg.analyzer.Analyze(ctx, req.Diffs)
	if err != nil {
		return nil, fmt.Errorf("analyze diffs: %w", err)
	}

	// Build prompt for LLM
	prompt := mg.buildPrompt(analysis, req)

	// Generate message using LLM
	llmReq := &llm.LLMRequest{
		ID:    uuid.New(),
		Model: "llama-3-8b",
		Messages: []llm.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   500,
		Temperature: 0.3, // Lower temperature for consistent output
	}

	llmResp, err := mg.llmProvider.Generate(ctx, llmReq)
	if err != nil {
		return nil, fmt.Errorf("generate with LLM: %w", err)
	}

	// Parse response
	message := mg.parseResponse(llmResp.Content, analysis, req.Format)

	// Cache result
	mg.cache.Set(diffHash, message)

	return message, nil
}

// buildPrompt creates the LLM prompt
func (mg *MessageGenerator) buildPrompt(analysis *DiffAnalysis, req MessageRequest) string {
	var prompt strings.Builder

	prompt.WriteString("Generate a clear, concise git commit message for the following changes.\n\n")

	// Format specification
	switch req.Format {
	case FormatConventional:
		prompt.WriteString("Use Conventional Commits format: <type>(<scope>): <subject>\n\n")
		prompt.WriteString("Types: feat, fix, docs, style, refactor, perf, test, build, ci, chore\n\n")
	case FormatSemantic:
		prompt.WriteString("Use semantic commit format with clear imperative mood.\n\n")
	case FormatAngular:
		prompt.WriteString("Use Angular commit format: <type>(<scope>): <subject>\n\n")
	}

	// Change summary
	prompt.WriteString("Change Summary:\n")
	prompt.WriteString(fmt.Sprintf("- Files changed: %d\n", analysis.Summary.FilesChanged))
	prompt.WriteString(fmt.Sprintf("- Lines added: %d\n", analysis.Summary.LinesAdded))
	prompt.WriteString(fmt.Sprintf("- Lines deleted: %d\n", analysis.Summary.LinesDeleted))

	// Semantic changes
	if len(analysis.Summary.FunctionsAdded) > 0 {
		prompt.WriteString(fmt.Sprintf("\nFunctions added: %s\n", strings.Join(analysis.Summary.FunctionsAdded, ", ")))
	}
	if len(analysis.Summary.FunctionsModified) > 0 {
		prompt.WriteString(fmt.Sprintf("Functions modified: %s\n", strings.Join(analysis.Summary.FunctionsModified, ", ")))
	}
	if len(analysis.Summary.FunctionsDeleted) > 0 {
		prompt.WriteString(fmt.Sprintf("Functions deleted: %s\n", strings.Join(analysis.Summary.FunctionsDeleted, ", ")))
	}

	// Context
	if req.Context.IssueRef != "" {
		prompt.WriteString(fmt.Sprintf("\nRelated issue: %s\n", req.Context.IssueRef))
	}
	if req.Context.BranchName != "" {
		prompt.WriteString(fmt.Sprintf("Branch: %s\n", req.Context.BranchName))
	}

	// File details
	if len(analysis.Files) > 0 {
		prompt.WriteString("\nChanged files:\n")
		for _, file := range analysis.Files {
			prompt.WriteString(fmt.Sprintf("- %s (%d lines changed)\n", file.Path, file.Summary.LinesAdded+file.Summary.LinesDeleted))
		}
	}

	prompt.WriteString("\n\nGenerate a commit message with:\n")
	prompt.WriteString("1. Subject line (max 72 characters, imperative mood)\n")
	prompt.WriteString("2. Optional body with details (if needed)\n")
	prompt.WriteString("3. Use the detected change type: ")
	prompt.WriteString(string(analysis.ChangeType))
	prompt.WriteString("\n")

	if analysis.Scope != "" {
		prompt.WriteString("4. Use scope: ")
		prompt.WriteString(analysis.Scope)
		prompt.WriteString("\n")
	}

	return prompt.String()
}

// parseResponse extracts commit message parts from LLM response
func (mg *MessageGenerator) parseResponse(response string, analysis *DiffAnalysis, format MessageFormat) *Message {
	// Clean up response
	response = strings.TrimSpace(response)

	// Split into parts: subject, body, footer
	parts := strings.Split(response, "\n\n")

	msg := &Message{
		Subject:    strings.TrimSpace(parts[0]),
		Format:     format,
		Confidence: 0.8,
		Analysis:   analysis,
	}

	// Validate and fix subject line
	msg.Subject = mg.validateSubject(msg.Subject, analysis, format)

	if len(parts) > 1 {
		msg.Body = strings.TrimSpace(parts[1])
	}
	if len(parts) > 2 {
		msg.Footer = strings.TrimSpace(parts[2])
	}

	return msg
}

// validateSubject validates and fixes the subject line
func (mg *MessageGenerator) validateSubject(subject string, analysis *DiffAnalysis, format MessageFormat) string {
	// Remove trailing period
	subject = strings.TrimSuffix(subject, ".")

	// Ensure it's not too long
	if len(subject) > 72 {
		subject = subject[:69] + "..."
	}

	// Ensure conventional commits format if required
	if format == FormatConventional {
		if !hasConventionalPrefix(subject) {
			// Add type prefix
			typePrefix := string(analysis.ChangeType)
			if analysis.Scope != "" {
				subject = fmt.Sprintf("%s(%s): %s", typePrefix, analysis.Scope, subject)
			} else {
				subject = fmt.Sprintf("%s: %s", typePrefix, subject)
			}
		}
	}

	// Ensure imperative mood (starts with lowercase verb)
	subject = ensureImperativeMood(subject)

	return subject
}

// FormatMessage formats the message into a single string
func (m *Message) FormatMessage() string {
	var result strings.Builder

	result.WriteString(m.Subject)

	if m.Body != "" {
		result.WriteString("\n\n")
		result.WriteString(m.Body)
	}

	if m.Footer != "" {
		result.WriteString("\n\n")
		result.WriteString(m.Footer)
	}

	return result.String()
}

// hasConventionalPrefix checks if the subject has a conventional commits prefix
func hasConventionalPrefix(subject string) bool {
	conventionalTypes := []string{
		"feat", "fix", "docs", "style", "refactor",
		"perf", "test", "build", "ci", "chore",
	}

	for _, t := range conventionalTypes {
		if strings.HasPrefix(strings.ToLower(subject), t+":") ||
			strings.HasPrefix(strings.ToLower(subject), t+"(") {
			return true
		}
	}

	return false
}

// ensureImperativeMood ensures the subject uses imperative mood
func ensureImperativeMood(subject string) string {
	// Extract the actual message part (after type and scope if present)
	parts := strings.SplitN(subject, ":", 2)
	if len(parts) != 2 {
		return subject
	}

	prefix := parts[0]
	message := strings.TrimSpace(parts[1])

	// Common past tense to imperative conversions
	conversions := map[string]string{
		"added":       "add",
		"fixed":       "fix",
		"updated":     "update",
		"removed":     "remove",
		"deleted":     "delete",
		"created":     "create",
		"implemented": "implement",
		"improved":    "improve",
		"refactored":  "refactor",
		"changed":     "change",
	}

	// Check if first word needs conversion
	words := strings.Fields(message)
	if len(words) > 0 {
		firstWord := strings.ToLower(words[0])
		if replacement, ok := conversions[firstWord]; ok {
			words[0] = replacement
			message = strings.Join(words, " ")
		}
	}

	return prefix + ": " + message
}

// DiffAnalyzer analyzes diffs to understand changes
type DiffAnalyzer struct {
	classifier *ChangeClassifier
}

// NewDiffAnalyzer creates a new diff analyzer
func NewDiffAnalyzer() *DiffAnalyzer {
	return &DiffAnalyzer{
		classifier: NewChangeClassifier(),
	}
}

// Analyze examines diffs and categorizes changes
func (da *DiffAnalyzer) Analyze(ctx context.Context, diffs []*Diff) (*DiffAnalysis, error) {
	analysis := &DiffAnalysis{
		Files:   make([]*FileAnalysis, 0, len(diffs)),
		Summary: &ChangeSummary{},
	}

	for _, diff := range diffs {
		fileAnalysis := da.analyzeFile(diff)
		analysis.Files = append(analysis.Files, fileAnalysis)

		// Merge into summary
		analysis.Summary.LinesAdded += fileAnalysis.Summary.LinesAdded
		analysis.Summary.LinesDeleted += fileAnalysis.Summary.LinesDeleted
		analysis.Summary.FilesChanged++
		analysis.Summary.FunctionsAdded = append(analysis.Summary.FunctionsAdded, fileAnalysis.Summary.FunctionsAdded...)
		analysis.Summary.FunctionsModified = append(analysis.Summary.FunctionsModified, fileAnalysis.Summary.FunctionsModified...)
		analysis.Summary.FunctionsDeleted = append(analysis.Summary.FunctionsDeleted, fileAnalysis.Summary.FunctionsDeleted...)

		// Update flags
		if fileAnalysis.Summary.TestsAdded {
			analysis.Summary.TestsAdded = true
		}
		if fileAnalysis.Summary.DocsModified {
			analysis.Summary.DocsModified = true
		}
		if fileAnalysis.Summary.ConfigChanged {
			analysis.Summary.ConfigChanged = true
		}
	}

	// Classify overall change type
	analysis.ChangeType = da.classifier.Classify(analysis.Summary)

	// Detect scope from file paths
	analysis.Scope = detectScope(diffs)

	return analysis, nil
}

// analyzeFile analyzes a single file diff
func (da *DiffAnalyzer) analyzeFile(diff *Diff) *FileAnalysis {
	fa := &FileAnalysis{
		Path: diff.Path,
		Summary: &ChangeSummary{
			FunctionsAdded:    []string{},
			FunctionsModified: []string{},
			FunctionsDeleted:  []string{},
		},
	}

	// Detect language
	fa.Language = detectLanguage(diff.Path)

	// Analyze hunks
	for _, hunk := range diff.Hunks {
		for _, line := range hunk.Lines {
			switch line.Type {
			case LineAdd:
				fa.Summary.LinesAdded++
				// Check for function additions
				if funcName := extractFunctionName(line.Content, fa.Language); funcName != "" {
					fa.Summary.FunctionsAdded = append(fa.Summary.FunctionsAdded, funcName)
				}
			case LineDelete:
				fa.Summary.LinesDeleted++
				// Check for function deletions
				if funcName := extractFunctionName(line.Content, fa.Language); funcName != "" {
					fa.Summary.FunctionsDeleted = append(fa.Summary.FunctionsDeleted, funcName)
				}
			}
		}
	}

	// Detect file characteristics
	fa.Summary.TestsAdded = isTestFile(diff.Path)
	fa.Summary.DocsModified = isDocFile(diff.Path)
	fa.Summary.ConfigChanged = isConfigFile(diff.Path)

	return fa
}

// detectLanguage detects the programming language from file path
func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	languages := map[string]string{
		".go":   "go",
		".js":   "javascript",
		".ts":   "typescript",
		".py":   "python",
		".java": "java",
		".c":    "c",
		".cpp":  "cpp",
		".rs":   "rust",
		".rb":   "ruby",
		".php":  "php",
		".sh":   "shell",
	}
	return languages[ext]
}

// extractFunctionName extracts function name from a line of code
func extractFunctionName(line, language string) string {
	var patterns []string

	switch language {
	case "go":
		patterns = []string{
			`func\s+(\w+)\s*\(`,                    // func Name(
			`func\s+\(\w+\s+\*?\w+\)\s+(\w+)\s*\(`, // func (r *Type) Name(
		}
	case "javascript", "typescript":
		patterns = []string{
			`function\s+(\w+)\s*\(`,     // function name(
			`(\w+)\s*:\s*function\s*\(`, // name: function(
			`const\s+(\w+)\s*=\s*\(`,    // const name = (
		}
	case "python":
		patterns = []string{
			`def\s+(\w+)\s*\(`, // def name(
		}
	case "java":
		patterns = []string{
			`\w+\s+(\w+)\s*\([^\)]*\)\s*\{`, // type name() {
		}
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(line); len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

// isTestFile checks if a file is a test file
func isTestFile(path string) bool {
	return strings.Contains(path, "_test.") ||
		strings.Contains(path, ".test.") ||
		strings.HasSuffix(path, "_test.go") ||
		strings.HasSuffix(path, ".spec.js") ||
		strings.HasSuffix(path, ".spec.ts")
}

// isDocFile checks if a file is a documentation file
func isDocFile(path string) bool {
	return strings.HasSuffix(path, ".md") ||
		strings.HasSuffix(path, ".rst") ||
		strings.HasSuffix(path, ".txt") ||
		strings.Contains(path, "README") ||
		strings.Contains(path, "CHANGELOG")
}

// isConfigFile checks if a file is a configuration file
func isConfigFile(path string) bool {
	return strings.HasSuffix(path, ".yaml") ||
		strings.HasSuffix(path, ".yml") ||
		strings.HasSuffix(path, ".json") ||
		strings.HasSuffix(path, ".toml") ||
		strings.HasSuffix(path, ".ini") ||
		strings.HasSuffix(path, ".conf") ||
		strings.Contains(path, "config")
}

// detectScope detects the scope from file paths
func detectScope(diffs []*Diff) string {
	if len(diffs) == 0 {
		return ""
	}

	// Extract common directory prefix
	if len(diffs) == 1 {
		path := diffs[0].Path
		parts := strings.Split(path, "/")
		if len(parts) > 1 {
			return parts[0]
		}
	}

	// Find common prefix for multiple files
	commonParts := strings.Split(diffs[0].Path, "/")
	for _, diff := range diffs[1:] {
		parts := strings.Split(diff.Path, "/")
		newCommon := []string{}
		for i := 0; i < len(commonParts) && i < len(parts); i++ {
			if commonParts[i] == parts[i] {
				newCommon = append(newCommon, parts[i])
			} else {
				break
			}
		}
		commonParts = newCommon
	}

	if len(commonParts) > 0 {
		return commonParts[0]
	}

	return ""
}

// ChangeClassifier classifies changes into categories
type ChangeClassifier struct {
	rules []ClassificationRule
}

// ClassificationRule defines classification logic
type ClassificationRule struct {
	Type     ChangeType
	Priority int
	Match    func(*ChangeSummary) bool
}

// NewChangeClassifier creates a new classifier
func NewChangeClassifier() *ChangeClassifier {
	return &ChangeClassifier{
		rules: defaultRules(),
	}
}

// Classify determines the change type
func (cc *ChangeClassifier) Classify(summary *ChangeSummary) ChangeType {
	for _, rule := range cc.rules {
		if rule.Match(summary) {
			return rule.Type
		}
	}
	return TypeChore
}

// defaultRules returns default classification rules
func defaultRules() []ClassificationRule {
	return []ClassificationRule{
		{
			Type:     TypeTest,
			Priority: 10,
			Match: func(s *ChangeSummary) bool {
				return s.TestsAdded && len(s.FunctionsAdded) > 0
			},
		},
		{
			Type:     TypeDocs,
			Priority: 9,
			Match: func(s *ChangeSummary) bool {
				return s.DocsModified && len(s.FunctionsModified) == 0
			},
		},
		{
			Type:     TypeBuild,
			Priority: 8,
			Match: func(s *ChangeSummary) bool {
				return s.ConfigChanged
			},
		},
		{
			Type:     TypeFeat,
			Priority: 7,
			Match: func(s *ChangeSummary) bool {
				return len(s.FunctionsAdded) > 0
			},
		},
		{
			Type:     TypeFix,
			Priority: 6,
			Match: func(s *ChangeSummary) bool {
				// Heuristic: small changes might be fixes
				return s.LinesAdded+s.LinesDeleted < 50 && len(s.FunctionsModified) > 0
			},
		},
		{
			Type:     TypeRefactor,
			Priority: 5,
			Match: func(s *ChangeSummary) bool {
				return len(s.FunctionsModified) > 0 && s.LinesAdded > 0 && s.LinesDeleted > 0
			},
		},
	}
}

// MessageCache caches generated messages
type MessageCache struct {
	mu    sync.RWMutex
	cache map[string]*CachedMessage
	ttl   time.Duration
}

// CachedMessage represents a cached message
type CachedMessage struct {
	Message   *Message
	Timestamp time.Time
}

// NewMessageCache creates a new message cache
func NewMessageCache(ttl time.Duration) *MessageCache {
	return &MessageCache{
		cache: make(map[string]*CachedMessage),
		ttl:   ttl,
	}
}

// Get retrieves a cached message
func (mc *MessageCache) Get(diffHash string) (*Message, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	cached, ok := mc.cache[diffHash]
	if !ok {
		return nil, false
	}

	// Check TTL
	if time.Since(cached.Timestamp) > mc.ttl {
		return nil, false
	}

	return cached.Message, true
}

// Set stores a message in cache
func (mc *MessageCache) Set(diffHash string, message *Message) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.cache[diffHash] = &CachedMessage{
		Message:   message,
		Timestamp: time.Now(),
	}
}
