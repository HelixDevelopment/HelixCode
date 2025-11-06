// Package git provides intelligent auto-commit functionality for HelixCode.
//
// This package implements LLM-powered commit message generation, semantic diff
// analysis, and safe git operations with co-author attribution support.
//
// # Overview
//
// The git package enables automated git commits with intelligent message
// generation based on semantic analysis of code changes. It integrates with
// HelixCode's LLM providers to generate meaningful commit messages following
// conventional commit formats.
//
// # Key Features
//
//   - LLM-powered commit message generation
//   - Semantic diff analysis with function-level detection
//   - Multiple commit message formats (Conventional, Semantic, Angular)
//   - Co-author attribution (including Claude attribution)
//   - Safe amend detection (prevents amending pushed or foreign commits)
//   - Multi-language support for commit messages
//   - Message caching for performance
//   - Pre-commit hook integration
//
// # Architecture
//
// The package is organized into several core components:
//
//   - AutoCommitCoordinator: Main orchestrator for auto-commit workflow
//   - MessageGenerator: LLM-powered commit message generation
//   - DiffAnalyzer: Semantic analysis of git diffs
//   - AttributionManager: Co-author and attribution management
//   - AmendDetector: Safe amend detection and validation
//
// # Usage
//
// Basic auto-commit with LLM-generated message:
//
//	// Create coordinator with LLM provider
//	coordinator, err := git.NewAutoCommitCoordinator(
//	    "/path/to/repo",
//	    llmProvider,
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Perform auto-commit
//	result, err := coordinator.AutoCommit(context.Background(), git.CommitOptions{
//	    Files: []string{"main.go", "handler.go"},
//	    Author: git.Person{
//	        Name:  "John Doe",
//	        Email: "john@example.com",
//	    },
//	    Attributions: []git.Attribution{
//	        {
//	            Type:  git.AttributionCoAuthor,
//	            Name:  "Claude",
//	            Email: "noreply@anthropic.com",
//	        },
//	    },
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Committed: %s\n", result.Hash)
//	fmt.Printf("Message: %s\n", result.Message)
//
// Generate commit message without committing:
//
//	message, err := coordinator.GenerateMessage(context.Background(), git.MessageOptions{
//	    Format:   git.FormatConventional,
//	    Language: "en",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Println("Generated message:", message)
//
// Safe amend of last commit:
//
//	// Check if safe to amend
//	canAmend, reason := coordinator.amendDetector.CanAmend(context.Background())
//	if !canAmend {
//	    log.Printf("Cannot amend: %s", reason)
//	    return
//	}
//
//	// Amend with updated message
//	err := coordinator.Amend(context.Background(), git.AmendOptions{
//	    UpdateMessage: true,
//	    AddFiles:      []string{"new_file.go"},
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Message Formats
//
// The package supports multiple commit message formats:
//
// Conventional Commits:
//
//	feat(api): add user authentication
//
//	Implemented JWT-based authentication for API endpoints.
//	Includes middleware and token validation.
//
//	Co-authored-by: Claude <noreply@anthropic.com>
//
// Semantic Commits:
//
//	Add user authentication to API
//
//	Implemented JWT-based authentication system with middleware
//	and token validation for secure API access.
//
// Angular Style:
//
//	feat(api): add user authentication
//
//	- Implemented JWT middleware
//	- Added token validation
//	- Updated API documentation
//
// # Change Type Detection
//
// The DiffAnalyzer automatically detects change types based on semantic
// analysis:
//
//   - feat: New feature (function additions)
//   - fix: Bug fix (small modifications)
//   - docs: Documentation changes (README, markdown files)
//   - test: Test additions (_test.go, .spec.js files)
//   - refactor: Code refactoring (modifications with additions and deletions)
//   - perf: Performance improvements
//   - build: Build system changes (config files)
//   - ci: CI/CD changes
//   - chore: Other changes
//
// # Attribution System
//
// The attribution system supports multiple attribution types:
//
//   - Co-authored-by: Co-author attribution
//   - Signed-off-by: Sign-off (DCO compliance)
//   - Reviewed-by: Code reviewer attribution
//   - Tested-by: Tester attribution
//
// Example with multiple attributions:
//
//	attrs := []git.Attribution{
//	    {
//	        Type:  git.AttributionCoAuthor,
//	        Name:  "Claude",
//	        Email: "noreply@anthropic.com",
//	    },
//	    {
//	        Type:  git.AttributionReviewed,
//	        Name:  "Jane Smith",
//	        Email: "jane@example.com",
//	    },
//	}
//
//	message := am.AddAttribution(message, attrs)
//
// # Amend Safety
//
// The AmendDetector ensures safe amend operations by checking:
//
//   - Commit not pushed to remote (prevents rewriting published history)
//   - Authored by current user (prevents amending others' commits)
//   - Not on protected branches (main, master, develop, production)
//
// These checks prevent common git mistakes and maintain repository integrity.
//
// # Configuration
//
// Configuration can be customized using the Config struct:
//
//	config := &git.Config{
//	    Message: git.MessageConfig{
//	        Provider:         "anthropic",
//	        Model:            "claude-3-5-sonnet-20241022",
//	        Format:           git.FormatConventional,
//	        Language:         "en",
//	        MaxSubjectLength: 72,
//	        IncludeBody:      true,
//	        IncludeFooter:    true,
//	    },
//	    Attribution: git.AttributionConfig{
//	        EnableCoAuthors:   true,
//	        ClaudeAttribution: true,
//	        ClaudeName:        "Claude",
//	        ClaudeEmail:       "noreply@anthropic.com",
//	    },
//	    Amend: git.AmendConfig{
//	        Enabled:            true,
//	        NeverAmendPushed:   true,
//	        NeverAmendForeign:  true,
//	        NeverAmendBranches: []string{"main", "master"},
//	    },
//	    Safety: git.SafetyConfig{
//	        DryRun: false,
//	    },
//	}
//
//	coordinator, err := git.NewAutoCommitCoordinator(
//	    repoPath,
//	    llmProvider,
//	    git.WithConfig(config),
//	)
//
// # Performance
//
// The package includes several performance optimizations:
//
//   - Message caching: Generated messages are cached for 15 minutes
//   - Efficient diff parsing: Incremental parsing of git diff output
//   - Thread-safe operations: Mutex protection for concurrent access
//   - Lazy initialization: Components initialized only when needed
//
// # Integration with HelixCode
//
// This package integrates seamlessly with HelixCode's architecture:
//
//   - Uses existing llm.Provider interface for message generation
//   - Follows HelixCode's error handling patterns (fmt.Errorf wrapping)
//   - Context-based cancellation support
//   - Compatible with all HelixCode LLM providers
//
// # Error Handling
//
// Errors are wrapped with context using fmt.Errorf:
//
//	if err != nil {
//	    return fmt.Errorf("generate message: %w", err)
//	}
//
// This preserves the error chain for debugging and error handling.
//
// # Testing
//
// The package includes comprehensive tests:
//
//   - Unit tests for all components
//   - Integration tests with mock git repositories
//   - Mock LLM provider for deterministic testing
//   - Benchmark tests for performance validation
//
// Run tests with:
//
//	go test -v ./internal/tools/git
//	go test -cover ./internal/tools/git
//
// # Thread Safety
//
// All public methods are thread-safe through mutex protection:
//
//   - AutoCommitCoordinator uses sync.RWMutex
//   - MessageCache uses sync.RWMutex for concurrent access
//   - MessageGenerator uses sync.RWMutex for LLM calls
//
// # Examples
//
// See the test file (git_test.go) for comprehensive examples of:
//
//   - Setting up test repositories
//   - Mocking LLM providers
//   - Testing various commit scenarios
//   - Verifying message generation
//   - Testing attribution management
//
// # References
//
// This implementation is inspired by:
//
//   - Aider's auto-commit functionality
//   - Conventional Commits specification (conventionalcommits.org)
//   - Angular commit guidelines
//   - GitHub co-author attribution
//
// # License
//
// Part of HelixCode, an enterprise-grade distributed AI development platform.
package git
