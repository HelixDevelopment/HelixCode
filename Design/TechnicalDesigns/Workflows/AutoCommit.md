# Intelligent Git Auto-Commit - Technical Design

## Overview

Intelligent auto-commit automatically generates meaningful git commit messages using LLM analysis of code changes, provides co-author attribution, and safely handles git operations with pre-commit hook integration.

## Architecture

### Component Diagram

```
┌────────────────────────────────────────────────────────────┐
│                    AutoCommitCoordinator                    │
│  - Orchestrates commit workflow                            │
│  - Manages git operations                                   │
└────────────┬───────────────────────────────┬───────────────┘
             │                               │
             ▼                               ▼
┌────────────────────────┐      ┌────────────────────────┐
│   MessageGenerator     │      │   GitOperations        │
│  - LLM-powered         │      │  - Staging             │
│  - Diff analysis       │      │  - Committing          │
│  - Multi-language      │      │  - Pre-commit hooks    │
└──────────┬─────────────┘      └────────┬───────────────┘
           │                             │
           ▼                             ▼
┌────────────────────┐        ┌──────────────────────┐
│ AttributionManager │        │   AmendDetector      │
│ - Co-authored-by   │        │  - Author check      │
│ - Sign-off         │        │  - Safety rules      │
└────────────────────┘        └──────────────────────┘
           │                             │
           └─────────────┬───────────────┘
                         ▼
                 ┌──────────────┐
                 │  Repository  │
                 └──────────────┘
```

### Core Components

#### 1. AutoCommitCoordinator

```go
package autocommit

import (
    "context"
    "time"
)

// AutoCommitCoordinator orchestrates the auto-commit workflow
type AutoCommitCoordinator struct {
    msgGenerator      *MessageGenerator
    gitOps            *GitOperations
    attributionMgr    *AttributionManager
    amendDetector     *AmendDetector
    config            *Config
}

// NewAutoCommitCoordinator creates a new coordinator
func NewAutoCommitCoordinator(repoPath string, opts ...Option) (*AutoCommitCoordinator, error) {
    repo, err := git.PlainOpen(repoPath)
    if err != nil {
        return nil, fmt.Errorf("open repository: %w", err)
    }

    acc := &AutoCommitCoordinator{
        msgGenerator:   NewMessageGenerator(),
        gitOps:         NewGitOperations(repo),
        attributionMgr: NewAttributionManager(),
        amendDetector:  NewAmendDetector(repo),
        config:         DefaultConfig(),
    }

    for _, opt := range opts {
        opt(acc)
    }

    return acc, nil
}

// AutoCommit performs an automatic commit
func (acc *AutoCommitCoordinator) AutoCommit(ctx context.Context, opts CommitOptions) (*CommitResult, error)

// GenerateMessage generates a commit message without committing
func (acc *AutoCommitCoordinator) GenerateMessage(ctx context.Context, opts MessageOptions) (string, error)

// Amend amends the last commit if safe
func (acc *AutoCommitCoordinator) Amend(ctx context.Context, opts AmendOptions) error
```

#### 2. MessageGenerator

```go
// MessageGenerator generates commit messages using LLM
type MessageGenerator struct {
    llmClient      LLMClient
    diffAnalyzer   *DiffAnalyzer
    templateEngine *TemplateEngine
    cache          *MessageCache
}

// Generate creates a commit message from diffs
func (mg *MessageGenerator) Generate(ctx context.Context, req MessageRequest) (*Message, error) {
    // Analyze diffs
    analysis, err := mg.diffAnalyzer.Analyze(ctx, req.Diffs)
    if err != nil {
        return nil, fmt.Errorf("analyze diffs: %w", err)
    }

    // Generate message using LLM
    prompt := mg.buildPrompt(analysis, req)
    message, err := mg.llmClient.Generate(ctx, prompt)
    if err != nil {
        return nil, fmt.Errorf("generate message: %w", err)
    }

    // Format according to template
    formatted := mg.templateEngine.Format(message, req.Format)

    return &Message{
        Subject:     formatted.Subject,
        Body:        formatted.Body,
        Footer:      formatted.Footer,
        Format:      req.Format,
        Confidence:  message.Confidence,
        Analysis:    analysis,
    }, nil
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

// CommitContext provides additional context
type CommitContext struct {
    IssueRef      string
    PreviousMsg   string
    BranchName    string
    ChangedFiles  []string
}
```

#### 3. DiffAnalyzer

```go
// DiffAnalyzer analyzes diffs to understand changes
type DiffAnalyzer struct {
    parsers        map[string]LanguageParser
    classifier     *ChangeClassifier
}

// Analyze examines diffs and categorizes changes
func (da *DiffAnalyzer) Analyze(ctx context.Context, diffs []*Diff) (*DiffAnalysis, error) {
    analysis := &DiffAnalysis{
        Files:   make([]*FileAnalysis, 0, len(diffs)),
        Summary: &ChangeSummary{},
    }

    for _, diff := range diffs {
        fileAnalysis, err := da.analyzeFile(ctx, diff)
        if err != nil {
            return nil, err
        }
        analysis.Files = append(analysis.Files, fileAnalysis)
        analysis.Summary.Merge(fileAnalysis.Summary)
    }

    // Classify overall change type
    analysis.ChangeType = da.classifier.Classify(analysis.Summary)

    return analysis, nil
}

// DiffAnalysis contains analysis results
type DiffAnalysis struct {
    Files      []*FileAnalysis
    Summary    *ChangeSummary
    ChangeType ChangeType
    Scope      string
}

// FileAnalysis contains per-file analysis
type FileAnalysis struct {
    Path        string
    Language    string
    Summary     *ChangeSummary
    Functions   []*FunctionChange
    Imports     []*ImportChange
    Complexity  int
}

// ChangeSummary summarizes changes
type ChangeSummary struct {
    LinesAdded     int
    LinesDeleted   int
    FilesModified  int
    FilesCreated   int
    FilesDeleted   int
    FilesRenamed   int

    // Semantic changes
    FunctionsAdded   []string
    FunctionsModified []string
    FunctionsDeleted  []string
    ImportsAdded     []string
    ImportsRemoved   []string

    // Code characteristics
    TestsAdded       bool
    DocsModified     bool
    ConfigChanged    bool
    DependenciesChanged bool
}

// ChangeType categorizes the change
type ChangeType string

const (
    TypeFeat      ChangeType = "feat"     // New feature
    TypeFix       ChangeType = "fix"      // Bug fix
    TypeDocs      ChangeType = "docs"     // Documentation
    TypeStyle     ChangeType = "style"    // Formatting
    TypeRefactor  ChangeType = "refactor" // Code refactoring
    TypePerf      ChangeType = "perf"     // Performance
    TypeTest      ChangeType = "test"     // Tests
    TypeBuild     ChangeType = "build"    // Build system
    TypeCI        ChangeType = "ci"       // CI/CD
    TypeChore     ChangeType = "chore"    // Other
)

// FunctionChange represents a function modification
type FunctionChange struct {
    Name      string
    Type      string // added, modified, deleted
    Signature string
    LinesChanged int
}

// ImportChange represents an import modification
type ImportChange struct {
    Package string
    Type    string // added, removed
}
```

#### 4. ChangeClassifier

```go
// ChangeClassifier classifies changes into categories
type ChangeClassifier struct {
    rules []ClassificationRule
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

// ClassificationRule defines classification logic
type ClassificationRule struct {
    Type     ChangeType
    Priority int
    Match    func(*ChangeSummary) bool
}

// Default classification rules
var defaultRules = []ClassificationRule{
    {
        Type:     TypeTest,
        Priority: 10,
        Match: func(s *ChangeSummary) bool {
            return s.TestsAdded && s.FilesModified == 1
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
        Type:     TypeFeat,
        Priority: 8,
        Match: func(s *ChangeSummary) bool {
            return len(s.FunctionsAdded) > 0
        },
    },
    {
        Type:     TypeFix,
        Priority: 7,
        Match: func(s *ChangeSummary) bool {
            // Heuristic: small changes might be fixes
            return s.LinesAdded+s.LinesDeleted < 50 && len(s.FunctionsModified) > 0
        },
    },
    {
        Type:     TypeRefactor,
        Priority: 6,
        Match: func(s *ChangeSummary) bool {
            return len(s.FunctionsModified) > 0 && s.LinesAdded > 0 && s.LinesDeleted > 0
        },
    },
}
```

#### 5. TemplateEngine

```go
// TemplateEngine formats messages according to templates
type TemplateEngine struct {
    templates map[MessageFormat]*template.Template
}

// Format formats a message using the specified format
func (te *TemplateEngine) Format(message *Message, format MessageFormat) *FormattedMessage {
    tmpl := te.templates[format]
    if tmpl == nil {
        tmpl = te.templates[FormatConventional]
    }

    var buf bytes.Buffer
    data := map[string]interface{}{
        "Type":    message.Analysis.ChangeType,
        "Scope":   message.Analysis.Scope,
        "Subject": message.Subject,
        "Body":    message.Body,
        "Footer":  message.Footer,
    }

    if err := tmpl.Execute(&buf, data); err != nil {
        return &FormattedMessage{Subject: message.Subject}
    }

    return parseFormattedMessage(buf.String())
}

// FormattedMessage contains formatted parts
type FormattedMessage struct {
    Subject string
    Body    string
    Footer  string
}

// Conventional Commits template
var conventionalTemplate = `{{.Type}}{{if .Scope}}({{.Scope}}){{end}}: {{.Subject}}

{{if .Body}}{{.Body}}

{{end}}{{if .Footer}}{{.Footer}}{{end}}`

// Angular template
var angularTemplate = `{{.Type}}({{.Scope}}): {{.Subject}}

{{.Body}}

{{.Footer}}`
```

#### 6. GitOperations

```go
// GitOperations handles git operations
type GitOperations struct {
    repo       *git.Repository
    precommit  *PreCommitRunner
}

// Stage stages files for commit
func (go *GitOperations) Stage(ctx context.Context, files []string) error {
    worktree, err := go.repo.Worktree()
    if err != nil {
        return err
    }

    for _, file := range files {
        if _, err := worktree.Add(file); err != nil {
            return fmt.Errorf("stage %s: %w", file, err)
        }
    }

    return nil
}

// Commit creates a commit
func (go *GitOperations) Commit(ctx context.Context, opts CommitOptions) (*plumbing.Hash, error) {
    // Run pre-commit hooks if enabled
    if opts.RunPreCommit {
        if err := go.precommit.Run(ctx); err != nil {
            return nil, fmt.Errorf("pre-commit failed: %w", err)
        }
    }

    worktree, err := go.repo.Worktree()
    if err != nil {
        return nil, err
    }

    commitOpts := &git.CommitOptions{
        Author: &object.Signature{
            Name:  opts.Author.Name,
            Email: opts.Author.Email,
            When:  time.Now(),
        },
    }

    // Set committer if different
    if opts.Committer != nil {
        commitOpts.Committer = &object.Signature{
            Name:  opts.Committer.Name,
            Email: opts.Committer.Email,
            When:  time.Now(),
        }
    }

    // Allow amending if specified
    commitOpts.Amend = opts.Amend

    hash, err := worktree.Commit(opts.Message, commitOpts)
    if err != nil {
        return nil, fmt.Errorf("commit: %w", err)
    }

    return &hash, nil
}

// GetDiff retrieves current diff
func (go *GitOperations) GetDiff(ctx context.Context) ([]*Diff, error) {
    worktree, err := go.repo.Worktree()
    if err != nil {
        return nil, err
    }

    status, err := worktree.Status()
    if err != nil {
        return nil, err
    }

    var diffs []*Diff
    for file := range status {
        diff, err := go.getDiffForFile(file)
        if err != nil {
            return nil, err
        }
        diffs = append(diffs, diff)
    }

    return diffs, nil
}

// CommitOptions configures commit behavior
type CommitOptions struct {
    Message       string
    Author        Person
    Committer     *Person
    Amend         bool
    RunPreCommit  bool
    SignOff       bool
    GPGSign       bool
}

// Person represents a git person
type Person struct {
    Name  string
    Email string
}
```

#### 7. AttributionManager

```go
// AttributionManager handles commit attribution
type AttributionManager struct {
    config *AttributionConfig
}

// AddAttribution adds attribution to commit message
func (am *AttributionManager) AddAttribution(message string, attrs []Attribution) string {
    var footer strings.Builder

    // Extract existing footer
    parts := strings.Split(message, "\n\n")
    body := message
    existingFooter := ""
    if len(parts) > 1 {
        body = strings.Join(parts[:len(parts)-1], "\n\n")
        existingFooter = parts[len(parts)-1]
    }

    // Add co-authors
    for _, attr := range attrs {
        switch attr.Type {
        case AttributionCoAuthor:
            footer.WriteString(fmt.Sprintf("Co-authored-by: %s <%s>\n", attr.Name, attr.Email))
        case AttributionSignedOff:
            footer.WriteString(fmt.Sprintf("Signed-off-by: %s <%s>\n", attr.Name, attr.Email))
        case AttributionReviewed:
            footer.WriteString(fmt.Sprintf("Reviewed-by: %s <%s>\n", attr.Name, attr.Email))
        }
    }

    // Combine
    result := body
    if existingFooter != "" {
        result += "\n\n" + existingFooter
    }
    if footer.Len() > 0 {
        result += "\n\n" + footer.String()
    }

    return result
}

// Attribution represents a commit attribution
type Attribution struct {
    Type  AttributionType
    Name  string
    Email string
}

// AttributionType specifies attribution type
type AttributionType int

const (
    AttributionCoAuthor AttributionType = iota
    AttributionSignedOff
    AttributionReviewed
    AttributionTested
)

// AttributionConfig configures attribution
type AttributionConfig struct {
    EnableCoAuthors bool
    ClaudeAttribution bool
    ClaudeName      string
    ClaudeEmail     string
    AutoSignOff     bool
}
```

#### 8. AmendDetector

```go
// AmendDetector detects when it's safe to amend
type AmendDetector struct {
    repo *git.Repository
}

// CanAmend checks if it's safe to amend the last commit
func (ad *AmendDetector) CanAmend(ctx context.Context) (bool, string) {
    // Check if commit is pushed
    if pushed, err := ad.isCommitPushed(); err != nil || pushed {
        return false, "commit already pushed"
    }

    // Check authorship
    if foreign, err := ad.isForeignCommit(); err != nil || foreign {
        return false, "not authored by current user"
    }

    // Check if on main/master
    if protected, err := ad.isProtectedBranch(); err != nil || protected {
        return false, "on protected branch"
    }

    return true, ""
}

// isCommitPushed checks if the last commit is pushed to remote
func (ad *AmendDetector) isCommitPushed() (bool, error) {
    head, err := ad.repo.Head()
    if err != nil {
        return false, err
    }

    // Check if local is ahead of remote
    remote, err := ad.repo.Remote("origin")
    if err != nil {
        return false, nil // No remote
    }

    refs, err := remote.List(&git.ListOptions{})
    if err != nil {
        return false, err
    }

    // Check if HEAD exists in remote refs
    for _, ref := range refs {
        if ref.Hash() == head.Hash() {
            return true, nil
        }
    }

    return false, nil
}

// isForeignCommit checks if last commit is by another author
func (ad *AmendDetector) isForeignCommit() (bool, error) {
    head, err := ad.repo.Head()
    if err != nil {
        return false, err
    }

    commit, err := ad.repo.CommitObject(head.Hash())
    if err != nil {
        return false, err
    }

    // Get current user
    cfg, err := ad.repo.Config()
    if err != nil {
        return false, err
    }

    currentEmail := cfg.User.Email
    return commit.Author.Email != currentEmail, nil
}

// isProtectedBranch checks if on main/master
func (ad *AmendDetector) isProtectedBranch() (bool, error) {
    head, err := ad.repo.Head()
    if err != nil {
        return false, err
    }

    branchName := head.Name().Short()
    protected := []string{"main", "master", "develop", "production"}

    for _, p := range protected {
        if branchName == p {
            return true, nil
        }
    }

    return false, nil
}
```

### State Machine

```
┌─────────┐
│  Idle   │
└────┬────┘
     │ AutoCommit()
     ▼
┌──────────────┐
│ Analyzing    │
│ - Get diffs  │
│ - Classify   │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Generating   │
│ - LLM call   │
│ - Format msg │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Validating   │
│ - Check amend│
│ - Verify msg │
└──────┬───────┘
       │
       ├─────────────┐
       │             │
       ▼             ▼
┌──────────┐   ┌──────────┐
│ Staging  │   │ Amending │
└────┬─────┘   └────┬─────┘
     │              │
     └──────┬───────┘
            ▼
    ┌──────────────┐
    │ Pre-Commit   │
    │ - Run hooks  │
    └──────┬───────┘
           │
           ├──Success──┐
           │           ▼
           │     ┌───────────┐
           │     │Committing │
           │     └─────┬─────┘
           │           │
           │           ▼
           │     ┌───────────┐
           │     │ Committed │
           │     └───────────┘
           │
           └──Error────┐
                       ▼
                 ┌───────────┐
                 │  Failed   │
                 │ - Rollback│
                 └───────────┘
```

## LLM Integration

### Prompt Engineering

```go
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
    }

    // Change summary
    prompt.WriteString(fmt.Sprintf("Change Summary:\n"))
    prompt.WriteString(fmt.Sprintf("- Files modified: %d\n", analysis.Summary.FilesModified))
    prompt.WriteString(fmt.Sprintf("- Files created: %d\n", analysis.Summary.FilesCreated))
    prompt.WriteString(fmt.Sprintf("- Files deleted: %d\n", analysis.Summary.FilesDeleted))
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

    // Diff details
    prompt.WriteString("\n\nDiff:\n```\n")
    for _, file := range analysis.Files {
        prompt.WriteString(fmt.Sprintf("--- %s\n", file.Path))
        // Include relevant diff hunks (truncated if too long)
    }
    prompt.WriteString("```\n\n")

    prompt.WriteString("Generate commit message (subject line, optional body, optional footer):\n")

    return prompt.String()
}
```

### LLM Client

```go
// LLMClient interfaces with LLM services
type LLMClient interface {
    Generate(ctx context.Context, prompt string) (*GeneratedMessage, error)
}

// GeneratedMessage contains LLM output
type GeneratedMessage struct {
    Subject    string
    Body       string
    Footer     string
    Confidence float64
    Reasoning  string
}

// ClaudeClient implements LLMClient for Claude
type ClaudeClient struct {
    apiKey string
    model  string
    client *anthropic.Client
}

// Generate generates a commit message using Claude
func (cc *ClaudeClient) Generate(ctx context.Context, prompt string) (*GeneratedMessage, error) {
    req := anthropic.MessageRequest{
        Model: cc.model,
        Messages: []anthropic.Message{
            {
                Role:    "user",
                Content: prompt,
            },
        },
        MaxTokens: 500,
        Temperature: 0.3, // Lower temperature for consistent output
    }

    resp, err := cc.client.CreateMessage(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("create message: %w", err)
    }

    // Parse response
    return cc.parseResponse(resp)
}

// parseResponse extracts commit message parts
func (cc *ClaudeClient) parseResponse(resp *anthropic.MessageResponse) (*GeneratedMessage, error) {
    content := resp.Content[0].Text

    // Parse format: subject\n\nbody\n\nfooter
    parts := strings.Split(content, "\n\n")

    msg := &GeneratedMessage{
        Subject:    strings.TrimSpace(parts[0]),
        Confidence: 0.8, // Default confidence
    }

    if len(parts) > 1 {
        msg.Body = strings.TrimSpace(parts[1])
    }
    if len(parts) > 2 {
        msg.Footer = strings.TrimSpace(parts[2])
    }

    // Validate subject line
    if len(msg.Subject) > 72 {
        msg.Subject = msg.Subject[:69] + "..."
    }

    return msg, nil
}
```

## Configuration Schema

```yaml
# auto_commit.yaml

auto_commit:
  # Enable auto-commit
  enabled: true

  # Message generation
  message:
    # LLM provider
    provider: claude  # claude, openai, local
    model: claude-3-5-sonnet-20241022

    # Message format
    format: conventional  # conventional, semantic, angular, custom
    template: |
      {{.Type}}{{if .Scope}}({{.Scope}}){{end}}: {{.Subject}}

      {{.Body}}

      {{.Footer}}

    # Language
    language: en  # en, es, fr, de, etc.

    # Options
    max_subject_length: 72
    include_body: true
    include_footer: true

  # Attribution
  attribution:
    # Co-author attribution
    co_authors: true
    claude_attribution: true
    claude_name: "Claude"
    claude_email: "noreply@anthropic.com"

    # Sign-off
    sign_off: false

  # Amend behavior
  amend:
    # Enable amend detection
    enabled: true

    # Never amend rules
    never_amend_pushed: true
    never_amend_foreign: true
    never_amend_branches:
      - main
      - master
      - develop
      - production

  # Pre-commit hooks
  hooks:
    enabled: true
    timeout: 5m

  # Analysis
  analysis:
    # Classify change type
    classify: true

    # Detect scope
    detect_scope: true

    # Include file details
    include_files: true
    max_files_shown: 10

    # Semantic analysis
    parse_functions: true
    parse_imports: true

  # Safety
  safety:
    # Require confirmation
    confirm_before_commit: false

    # Dry run
    dry_run: false

    # Backup
    backup_enabled: false
```

```go
// Config represents auto-commit configuration
type Config struct {
    Enabled     bool              `yaml:"enabled"`
    Message     MessageConfig     `yaml:"message"`
    Attribution AttributionConfig `yaml:"attribution"`
    Amend       AmendConfig       `yaml:"amend"`
    Hooks       HooksConfig       `yaml:"hooks"`
    Analysis    AnalysisConfig    `yaml:"analysis"`
    Safety      SafetyConfig      `yaml:"safety"`
}

// MessageConfig configures message generation
type MessageConfig struct {
    Provider          string        `yaml:"provider"`
    Model             string        `yaml:"model"`
    Format            MessageFormat `yaml:"format"`
    Template          string        `yaml:"template"`
    Language          string        `yaml:"language"`
    MaxSubjectLength  int           `yaml:"max_subject_length"`
    IncludeBody       bool          `yaml:"include_body"`
    IncludeFooter     bool          `yaml:"include_footer"`
}

// AmendConfig configures amend behavior
type AmendConfig struct {
    Enabled             bool     `yaml:"enabled"`
    NeverAmendPushed    bool     `yaml:"never_amend_pushed"`
    NeverAmendForeign   bool     `yaml:"never_amend_foreign"`
    NeverAmendBranches  []string `yaml:"never_amend_branches"`
}

// HooksConfig configures pre-commit hooks
type HooksConfig struct {
    Enabled bool          `yaml:"enabled"`
    Timeout time.Duration `yaml:"timeout"`
}

// AnalysisConfig configures diff analysis
type AnalysisConfig struct {
    Classify        bool `yaml:"classify"`
    DetectScope     bool `yaml:"detect_scope"`
    IncludeFiles    bool `yaml:"include_files"`
    MaxFilesShown   int  `yaml:"max_files_shown"`
    ParseFunctions  bool `yaml:"parse_functions"`
    ParseImports    bool `yaml:"parse_imports"`
}

// SafetyConfig configures safety features
type SafetyConfig struct {
    ConfirmBeforeCommit bool `yaml:"confirm_before_commit"`
    DryRun              bool `yaml:"dry_run"`
    BackupEnabled       bool `yaml:"backup_enabled"`
}
```

## Testing Strategy

### Unit Tests

```go
func TestMessageGenerator_Generate(t *testing.T) {
    mg := NewMessageGenerator()
    mg.llmClient = &MockLLMClient{
        response: &GeneratedMessage{
            Subject: "feat(api): add user authentication",
            Body:    "Implemented JWT-based auth",
        },
    }

    analysis := &DiffAnalysis{
        ChangeType: TypeFeat,
        Scope:      "api",
        Summary: &ChangeSummary{
            FilesModified: 3,
            FunctionsAdded: []string{"Authenticate", "ValidateToken"},
        },
    }

    req := MessageRequest{
        Format:   FormatConventional,
        Language: "en",
    }

    msg, err := mg.Generate(context.Background(), analysis, req)
    require.NoError(t, err)
    assert.Equal(t, "feat(api): add user authentication", msg.Subject)
    assert.Contains(t, msg.Body, "JWT")
}

func TestDiffAnalyzer_Analyze(t *testing.T) {
    da := NewDiffAnalyzer()

    diff := &Diff{
        Path: "auth.go",
        Hunks: []*DiffHunk{
            {
                Lines: []DiffLine{
                    {Type: LineAdd, Content: "func Authenticate(token string) bool {"},
                    {Type: LineAdd, Content: "    return validateToken(token)"},
                    {Type: LineAdd, Content: "}"},
                },
            },
        },
    }

    analysis, err := da.Analyze(context.Background(), []*Diff{diff})
    require.NoError(t, err)
    assert.Len(t, analysis.Summary.FunctionsAdded, 1)
    assert.Equal(t, "Authenticate", analysis.Summary.FunctionsAdded[0])
}

func TestAmendDetector_CanAmend(t *testing.T) {
    // Create test repo
    repo := createTestRepo(t)
    ad := NewAmendDetector(repo)

    // Make a commit
    commitTestFile(t, repo, "test.txt", "test content")

    // Should be safe to amend (not pushed, same author, not protected)
    canAmend, reason := ad.CanAmend(context.Background())
    assert.True(t, canAmend)
    assert.Empty(t, reason)
}

func TestAttributionManager_AddAttribution(t *testing.T) {
    am := NewAttributionManager()

    message := "feat: add feature\n\nImplemented new feature"
    attrs := []Attribution{
        {
            Type:  AttributionCoAuthor,
            Name:  "Claude",
            Email: "noreply@anthropic.com",
        },
    }

    result := am.AddAttribution(message, attrs)
    assert.Contains(t, result, "Co-authored-by: Claude <noreply@anthropic.com>")
}
```

### Integration Tests

```go
func TestAutoCommitCoordinator_EndToEnd(t *testing.T) {
    // Setup test repository
    repoDir := t.TempDir()
    repo := initTestRepo(t, repoDir)

    // Create coordinator
    acc, err := NewAutoCommitCoordinator(repoDir)
    require.NoError(t, err)

    // Mock LLM client
    acc.msgGenerator.llmClient = &MockLLMClient{
        response: &GeneratedMessage{
            Subject: "feat: add test feature",
        },
    }

    // Make changes
    testFile := filepath.Join(repoDir, "test.go")
    err = os.WriteFile(testFile, []byte("package test\n\nfunc Test() {}\n"), 0644)
    require.NoError(t, err)

    // Auto commit
    result, err := acc.AutoCommit(context.Background(), CommitOptions{
        Author: Person{
            Name:  "Test User",
            Email: "test@example.com",
        },
    })
    require.NoError(t, err)
    assert.NotNil(t, result.Hash)
    assert.Equal(t, "feat: add test feature", result.Message)

    // Verify commit
    commits, err := repo.Log(&git.LogOptions{})
    require.NoError(t, err)

    commit, err := commits.Next()
    require.NoError(t, err)
    assert.Contains(t, commit.Message, "feat: add test feature")
}

func TestAutoCommitCoordinator_WithPreCommit(t *testing.T) {
    repoDir := t.TempDir()
    repo := initTestRepo(t, repoDir)

    // Create pre-commit hook
    hookPath := filepath.Join(repoDir, ".git", "hooks", "pre-commit")
    hookContent := `#!/bin/sh
echo "Running pre-commit hook"
exit 0
`
    err := os.WriteFile(hookPath, []byte(hookContent), 0755)
    require.NoError(t, err)

    acc, err := NewAutoCommitCoordinator(repoDir)
    require.NoError(t, err)

    // Make changes
    testFile := filepath.Join(repoDir, "test.go")
    err = os.WriteFile(testFile, []byte("package test"), 0644)
    require.NoError(t, err)

    // Commit with pre-commit
    result, err := acc.AutoCommit(context.Background(), CommitOptions{
        RunPreCommit: true,
    })
    require.NoError(t, err)
    assert.NotNil(t, result.Hash)
}
```

### Message Quality Tests

```go
func TestMessageQuality(t *testing.T) {
    tests := []struct {
        name     string
        diffs    []*Diff
        expected struct {
            type_   ChangeType
            scope   string
            subject string
        }
    }{
        {
            name: "feature addition",
            diffs: []*Diff{
                {
                    Path: "api/user.go",
                    Hunks: []*DiffHunk{
                        {Lines: []DiffLine{
                            {Type: LineAdd, Content: "func CreateUser() {}"},
                        }},
                    },
                },
            },
            expected: struct {
                type_   ChangeType
                scope   string
                subject string
            }{
                type_:   TypeFeat,
                scope:   "api",
                subject: "add user creation",
            },
        },
        {
            name: "bug fix",
            diffs: []*Diff{
                {
                    Path: "auth/token.go",
                    Hunks: []*DiffHunk{
                        {Lines: []DiffLine{
                            {Type: LineDelete, Content: "if token == \"\" {"},
                            {Type: LineAdd, Content: "if token == \"\" || token == \"invalid\" {"},
                        }},
                    },
                },
            },
            expected: struct {
                type_   ChangeType
                scope   string
                subject string
            }{
                type_:   TypeFix,
                scope:   "auth",
                subject: "fix token validation",
            },
        },
    }

    mg := NewMessageGenerator()
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            analysis, _ := mg.diffAnalyzer.Analyze(context.Background(), tt.diffs)
            assert.Equal(t, tt.expected.type_, analysis.ChangeType)
            assert.Equal(t, tt.expected.scope, analysis.Scope)
        })
    }
}
```

## Performance Considerations

### Caching

```go
// MessageCache caches generated messages
type MessageCache struct {
    mu    sync.RWMutex
    cache map[string]*CachedMessage
    ttl   time.Duration
}

type CachedMessage struct {
    Message   *Message
    Timestamp time.Time
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

// computeDiffHash creates a hash of diffs for caching
func computeDiffHash(diffs []*Diff) string {
    h := sha256.New()
    for _, diff := range diffs {
        h.Write([]byte(diff.Path))
        for _, hunk := range diff.Hunks {
            for _, line := range hunk.Lines {
                h.Write([]byte(line.Content))
            }
        }
    }
    return fmt.Sprintf("%x", h.Sum(nil))
}
```

### Batch Processing

```go
// BatchCommit commits multiple changes as separate commits
func (acc *AutoCommitCoordinator) BatchCommit(ctx context.Context, changes []ChangeSet) ([]*CommitResult, error) {
    results := make([]*CommitResult, 0, len(changes))

    for _, change := range changes {
        result, err := acc.AutoCommit(ctx, CommitOptions{
            Files: change.Files,
        })
        if err != nil {
            return results, err
        }
        results = append(results, result)
    }

    return results, nil
}

// ChangeSet represents a group of related changes
type ChangeSet struct {
    Files   []string
    Context CommitContext
}
```

## Security Considerations

### Sensitive Data Detection

```go
// SensitiveDataDetector checks for sensitive data
type SensitiveDataDetector struct {
    patterns []*regexp.Regexp
}

// Detect checks commit for sensitive data
func (sdd *SensitiveDataDetector) Detect(diffs []*Diff) ([]string, error) {
    var findings []string

    for _, diff := range diffs {
        for _, hunk := range diff.Hunks {
            for _, line := range hunk.Lines {
                if line.Type != LineAdd {
                    continue
                }

                for _, pattern := range sdd.patterns {
                    if pattern.MatchString(line.Content) {
                        findings = append(findings, fmt.Sprintf(
                            "%s:%d: potential sensitive data",
                            diff.Path, line.LineNo,
                        ))
                    }
                }
            }
        }
    }

    return findings, nil
}

// Default patterns for sensitive data
var defaultPatterns = []string{
    `(?i)password\s*=\s*["'].*["']`,
    `(?i)api[_-]?key\s*=\s*["'].*["']`,
    `(?i)secret\s*=\s*["'].*["']`,
    `(?i)token\s*=\s*["'].*["']`,
    `-----BEGIN.*PRIVATE KEY-----`,
}
```

### Commit Signing

```go
// GPGSigner signs commits with GPG
type GPGSigner struct {
    keyID      string
    passphrase string
}

// Sign signs a commit message
func (gs *GPGSigner) Sign(message string) (string, error) {
    // Create GPG signature
    cmd := exec.Command("gpg", "--detach-sign", "--armor", "-u", gs.keyID)
    cmd.Stdin = strings.NewReader(message)

    var out bytes.Buffer
    cmd.Stdout = &out

    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("gpg sign: %w", err)
    }

    return out.String(), nil
}
```

## References

### Aider Auto-Commit

- **Repository**: `aider/repo.py` - `commit()` method
- **Implementation**: Lines 150-250
- **Features**:
  - Automatic commit message generation
  - Git operations wrapper
  - Pre-commit hook support
  - Amend detection

### Aider Commit Message Generation

- **Repository**: `aider/coders/base_coder.py` - `auto_commit()` method
- **Features**:
  - LLM-powered message generation
  - Diff analysis
  - Conventional commits format
  - Context awareness

### Key Insights

1. **Semantic Understanding**: Use LLM to understand semantic meaning of changes
2. **Safety First**: Never amend others' commits or pushed commits
3. **Attribution**: Always attribute AI contributions
4. **Hook Integration**: Support pre-commit hooks for validation
5. **Multiple Formats**: Support various commit message formats

## Usage Examples

```go
// Example 1: Basic auto-commit
func ExampleBasicAutoCommit() {
    acc, _ := NewAutoCommitCoordinator(".")

    result, _ := acc.AutoCommit(context.Background(), CommitOptions{
        Author: Person{
            Name:  "John Doe",
            Email: "john@example.com",
        },
    })

    fmt.Printf("Committed: %s\n", result.Message)
    fmt.Printf("Hash: %s\n", result.Hash)
}

// Example 2: Generate message without committing
func ExampleGenerateMessage() {
    acc, _ := NewAutoCommitCoordinator(".")

    message, _ := acc.GenerateMessage(context.Background(), MessageOptions{
        Format: FormatConventional,
        Language: "en",
    })

    fmt.Println(message)
    // Output: feat(api): add user authentication
}

// Example 3: With co-author attribution
func ExampleWithAttribution() {
    acc, _ := NewAutoCommitCoordinator(".")

    result, _ := acc.AutoCommit(context.Background(), CommitOptions{
        Attributions: []Attribution{
            {
                Type:  AttributionCoAuthor,
                Name:  "Claude",
                Email: "noreply@anthropic.com",
            },
        },
    })

    fmt.Println(result.Message)
    // Output:
    // feat: add feature
    //
    // Co-authored-by: Claude <noreply@anthropic.com>
}

// Example 4: Amend last commit
func ExampleAmend() {
    acc, _ := NewAutoCommitCoordinator(".")

    // Check if safe to amend
    canAmend, reason := acc.amendDetector.CanAmend(context.Background())
    if !canAmend {
        fmt.Printf("Cannot amend: %s\n", reason)
        return
    }

    // Amend
    acc.Amend(context.Background(), AmendOptions{
        UpdateMessage: true,
    })
}
```

## Future Enhancements

1. **Multi-Language Support**: Generate messages in multiple languages
2. **Custom Templates**: User-defined message templates
3. **Learning**: Learn from user's commit history
4. **Interactive Mode**: Allow editing generated messages
5. **Emoji Support**: Optional emoji in commit messages (gitmoji)
