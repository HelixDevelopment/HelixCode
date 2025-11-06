package git

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"dev.helix.code/internal/llm"
)

// AutoCommitCoordinator orchestrates the auto-commit workflow
type AutoCommitCoordinator struct {
	repoPath       string
	msgGenerator   *MessageGenerator
	attributionMgr *AttributionManager
	amendDetector  *AmendDetector
	config         *Config
	mu             sync.RWMutex
}

// Config represents auto-commit configuration
type Config struct {
	Message     MessageConfig     `yaml:"message"`
	Attribution AttributionConfig `yaml:"attribution"`
	Amend       AmendConfig       `yaml:"amend"`
	Safety      SafetyConfig      `yaml:"safety"`
}

// MessageConfig configures message generation
type MessageConfig struct {
	Provider         string        `yaml:"provider"`
	Model            string        `yaml:"model"`
	Format           MessageFormat `yaml:"format"`
	Language         string        `yaml:"language"`
	MaxSubjectLength int           `yaml:"max_subject_length"`
	IncludeBody      bool          `yaml:"include_body"`
	IncludeFooter    bool          `yaml:"include_footer"`
}

// AmendConfig configures amend behavior
type AmendConfig struct {
	Enabled            bool     `yaml:"enabled"`
	NeverAmendPushed   bool     `yaml:"never_amend_pushed"`
	NeverAmendForeign  bool     `yaml:"never_amend_foreign"`
	NeverAmendBranches []string `yaml:"never_amend_branches"`
}

// SafetyConfig configures safety features
type SafetyConfig struct {
	ConfirmBeforeCommit bool `yaml:"confirm_before_commit"`
	DryRun              bool `yaml:"dry_run"`
	BackupEnabled       bool `yaml:"backup_enabled"`
}

// CommitOptions configures commit behavior
type CommitOptions struct {
	Files        []string
	Message      string
	Author       Person
	Committer    *Person
	Amend        bool
	SignOff      bool
	GPGSign      bool
	Attributions []Attribution
	SkipHooks    bool
}

// Person represents a git person
type Person struct {
	Name  string
	Email string
}

// CommitResult contains the result of a commit operation
type CommitResult struct {
	Hash      string
	Message   string
	Timestamp time.Time
	Files     []string
	Analysis  *DiffAnalysis
}

// MessageOptions configures message generation
type MessageOptions struct {
	Format         MessageFormat
	Language       string
	Context        CommitContext
	MaxLength      int
	IncludeDetails bool
}

// CommitContext provides additional context
type CommitContext struct {
	IssueRef      string
	PreviousMsg   string
	BranchName    string
	ChangedFiles  []string
}

// AmendOptions configures amend behavior
type AmendOptions struct {
	UpdateMessage bool
	AddFiles      []string
	NoEdit        bool
}

// NewAutoCommitCoordinator creates a new coordinator
func NewAutoCommitCoordinator(repoPath string, llmProvider llm.Provider, opts ...Option) (*AutoCommitCoordinator, error) {
	// Verify git repository
	if !isGitRepository(repoPath) {
		return nil, fmt.Errorf("not a git repository: %s", repoPath)
	}

	acc := &AutoCommitCoordinator{
		repoPath:       repoPath,
		msgGenerator:   NewMessageGenerator(llmProvider),
		attributionMgr: NewAttributionManager(),
		amendDetector:  NewAmendDetector(repoPath),
		config:         DefaultConfig(),
	}

	for _, opt := range opts {
		opt(acc)
	}

	return acc, nil
}

// Option is a functional option for AutoCommitCoordinator
type Option func(*AutoCommitCoordinator)

// WithConfig sets the configuration
func WithConfig(config *Config) Option {
	return func(acc *AutoCommitCoordinator) {
		acc.config = config
	}
}

// WithMessageGenerator sets the message generator
func WithMessageGenerator(mg *MessageGenerator) Option {
	return func(acc *AutoCommitCoordinator) {
		acc.msgGenerator = mg
	}
}

// AutoCommit performs an automatic commit
func (acc *AutoCommitCoordinator) AutoCommit(ctx context.Context, opts CommitOptions) (*CommitResult, error) {
	acc.mu.Lock()
	defer acc.mu.Unlock()

	// Get current diff
	diffs, err := acc.getDiff(ctx, opts.Files)
	if err != nil {
		return nil, fmt.Errorf("get diff: %w", err)
	}

	if len(diffs) == 0 {
		return nil, fmt.Errorf("no changes to commit")
	}

	// Generate commit message if not provided
	message := opts.Message
	var analysis *DiffAnalysis
	if message == "" {
		analysis, err = acc.msgGenerator.analyzer.Analyze(ctx, diffs)
		if err != nil {
			return nil, fmt.Errorf("analyze diffs: %w", err)
		}

		req := MessageRequest{
			Diffs:    diffs,
			Format:   acc.config.Message.Format,
			Language: acc.config.Message.Language,
			Context:  acc.getCommitContext(ctx),
		}

		msg, err := acc.msgGenerator.Generate(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("generate message: %w", err)
		}

		message = msg.FormatMessage()
	}

	// Add attribution
	if len(opts.Attributions) > 0 {
		message = acc.attributionMgr.AddAttribution(message, opts.Attributions)
	}

	// Check if we should amend
	shouldAmend := opts.Amend
	if !shouldAmend && acc.config.Amend.Enabled {
		canAmend, _ := acc.amendDetector.CanAmend(ctx)
		shouldAmend = canAmend
	}

	// Stage files
	if len(opts.Files) > 0 {
		if err := acc.stageFiles(ctx, opts.Files); err != nil {
			return nil, fmt.Errorf("stage files: %w", err)
		}
	}

	// Dry run check
	if acc.config.Safety.DryRun {
		return &CommitResult{
			Message:   message,
			Timestamp: time.Now(),
			Files:     opts.Files,
			Analysis:  analysis,
		}, nil
	}

	// Perform commit
	hash, err := acc.commit(ctx, message, opts, shouldAmend)
	if err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &CommitResult{
		Hash:      hash,
		Message:   message,
		Timestamp: time.Now(),
		Files:     opts.Files,
		Analysis:  analysis,
	}, nil
}

// GenerateMessage generates a commit message without committing
func (acc *AutoCommitCoordinator) GenerateMessage(ctx context.Context, opts MessageOptions) (string, error) {
	acc.mu.RLock()
	defer acc.mu.RUnlock()

	// Get current diff
	diffs, err := acc.getDiff(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("get diff: %w", err)
	}

	if len(diffs) == 0 {
		return "", fmt.Errorf("no changes to analyze")
	}

	req := MessageRequest{
		Diffs:    diffs,
		Format:   opts.Format,
		Language: opts.Language,
		Context:  opts.Context,
	}

	msg, err := acc.msgGenerator.Generate(ctx, req)
	if err != nil {
		return "", fmt.Errorf("generate message: %w", err)
	}

	return msg.FormatMessage(), nil
}

// Amend amends the last commit if safe
func (acc *AutoCommitCoordinator) Amend(ctx context.Context, opts AmendOptions) error {
	acc.mu.Lock()
	defer acc.mu.Unlock()

	// Check if safe to amend
	canAmend, reason := acc.amendDetector.CanAmend(ctx)
	if !canAmend {
		return fmt.Errorf("cannot amend: %s", reason)
	}

	// Stage additional files if specified
	if len(opts.AddFiles) > 0 {
		if err := acc.stageFiles(ctx, opts.AddFiles); err != nil {
			return fmt.Errorf("stage files: %w", err)
		}
	}

	// Generate new message if requested
	var message string
	if opts.UpdateMessage {
		diffs, err := acc.getDiff(ctx, nil)
		if err != nil {
			return fmt.Errorf("get diff: %w", err)
		}

		req := MessageRequest{
			Diffs:    diffs,
			Format:   acc.config.Message.Format,
			Language: acc.config.Message.Language,
			Context:  acc.getCommitContext(ctx),
		}

		msg, err := acc.msgGenerator.Generate(ctx, req)
		if err != nil {
			return fmt.Errorf("generate message: %w", err)
		}

		message = msg.FormatMessage()
	}

	// Perform amend
	args := []string{"commit", "--amend"}
	if opts.NoEdit {
		args = append(args, "--no-edit")
	}
	if message != "" {
		args = append(args, "-m", message)
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = acc.repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit --amend failed: %w, output: %s", err, output)
	}

	return nil
}

// getDiff retrieves current diff
func (acc *AutoCommitCoordinator) getDiff(ctx context.Context, files []string) ([]*Diff, error) {
	args := []string{"diff", "--cached", "--unified=3"}
	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = acc.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	// Parse diff output
	return parseDiff(string(output)), nil
}

// stageFiles stages files for commit
func (acc *AutoCommitCoordinator) stageFiles(ctx context.Context, files []string) error {
	for _, file := range files {
		cmd := exec.CommandContext(ctx, "git", "add", file)
		cmd.Dir = acc.repoPath
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git add %s failed: %w, output: %s", file, err, output)
		}
	}
	return nil
}

// commit creates a commit
func (acc *AutoCommitCoordinator) commit(ctx context.Context, message string, opts CommitOptions, amend bool) (string, error) {
	args := []string{"commit", "-m", message}

	if amend {
		args = []string{"commit", "--amend", "-m", message}
	}

	if opts.SignOff {
		args = append(args, "--signoff")
	}

	if opts.GPGSign {
		args = append(args, "--gpg-sign")
	}

	if opts.SkipHooks {
		args = append(args, "--no-verify")
	}

	// Set author if specified
	if opts.Author.Name != "" && opts.Author.Email != "" {
		args = append(args, "--author", fmt.Sprintf("%s <%s>", opts.Author.Name, opts.Author.Email))
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = acc.repoPath

	// Set environment for committer if specified
	if opts.Committer != nil {
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("GIT_COMMITTER_NAME=%s", opts.Committer.Name),
			fmt.Sprintf("GIT_COMMITTER_EMAIL=%s", opts.Committer.Email),
		)
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git commit failed: %w, output: %s", err, output)
	}

	// Get commit hash
	cmd = exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = acc.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// getCommitContext retrieves context for commit message generation
func (acc *AutoCommitCoordinator) getCommitContext(ctx context.Context) CommitContext {
	context := CommitContext{}

	// Get branch name
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = acc.repoPath
	if output, err := cmd.Output(); err == nil {
		context.BranchName = strings.TrimSpace(string(output))
	}

	// Get previous commit message
	cmd = exec.CommandContext(ctx, "git", "log", "-1", "--pretty=%B")
	cmd.Dir = acc.repoPath
	if output, err := cmd.Output(); err == nil {
		context.PreviousMsg = strings.TrimSpace(string(output))
	}

	// Get changed files
	cmd = exec.CommandContext(ctx, "git", "diff", "--cached", "--name-only")
	cmd.Dir = acc.repoPath
	if output, err := cmd.Output(); err == nil {
		files := strings.Split(strings.TrimSpace(string(output)), "\n")
		context.ChangedFiles = files
	}

	return context
}

// isGitRepository checks if a path is a git repository
func isGitRepository(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	return err == nil && info.IsDir()
}

// parseDiff parses git diff output into Diff structures
func parseDiff(diffOutput string) []*Diff {
	if diffOutput == "" {
		return nil
	}

	var diffs []*Diff
	var currentDiff *Diff
	var currentHunk *DiffHunk

	lines := strings.Split(diffOutput, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			// New file diff
			if currentDiff != nil {
				diffs = append(diffs, currentDiff)
			}
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				currentDiff = &Diff{
					Path: strings.TrimPrefix(parts[3], "b/"),
				}
			}
			currentHunk = nil
		} else if strings.HasPrefix(line, "@@") {
			// New hunk
			if currentDiff != nil {
				currentHunk = &DiffHunk{
					Header: line,
				}
				currentDiff.Hunks = append(currentDiff.Hunks, currentHunk)
			}
		} else if currentHunk != nil {
			// Diff line
			var lineType LineType
			content := line
			if len(line) > 0 {
				switch line[0] {
				case '+':
					lineType = LineAdd
					content = line[1:]
				case '-':
					lineType = LineDelete
					content = line[1:]
				default:
					lineType = LineContext
				}
			}
			currentHunk.Lines = append(currentHunk.Lines, DiffLine{
				Type:    lineType,
				Content: content,
			})
		}
	}

	if currentDiff != nil {
		diffs = append(diffs, currentDiff)
	}

	return diffs
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

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Message: MessageConfig{
			Provider:         "local",
			Model:            "llama-3-8b",
			Format:           FormatConventional,
			Language:         "en",
			MaxSubjectLength: 72,
			IncludeBody:      true,
			IncludeFooter:    true,
		},
		Attribution: AttributionConfig{
			EnableCoAuthors:   true,
			ClaudeAttribution: true,
			ClaudeName:        "Claude",
			ClaudeEmail:       "noreply@anthropic.com",
			AutoSignOff:       false,
		},
		Amend: AmendConfig{
			Enabled:            true,
			NeverAmendPushed:   true,
			NeverAmendForeign:  true,
			NeverAmendBranches: []string{"main", "master", "develop", "production"},
		},
		Safety: SafetyConfig{
			ConfirmBeforeCommit: false,
			DryRun:              false,
			BackupEnabled:       false,
		},
	}
}
