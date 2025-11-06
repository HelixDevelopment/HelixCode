package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dev.helix.code/internal/llm"
	"github.com/google/uuid"
)

// MockLLMProvider implements llm.Provider for testing
type MockLLMProvider struct {
	response *llm.LLMResponse
	err      error
}

func (m *MockLLMProvider) GetType() llm.ProviderType {
	return llm.ProviderTypeLocal
}

func (m *MockLLMProvider) GetName() string {
	return "mock"
}

func (m *MockLLMProvider) GetModels() []llm.ModelInfo {
	return []llm.ModelInfo{
		{
			Name:     "mock-model",
			Provider: llm.ProviderTypeLocal,
		},
	}
}

func (m *MockLLMProvider) GetCapabilities() []llm.ModelCapability {
	return []llm.ModelCapability{llm.CapabilityCodeGeneration}
}

func (m *MockLLMProvider) Generate(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *MockLLMProvider) GenerateStream(ctx context.Context, request *llm.LLMRequest, ch chan<- llm.LLMResponse) error {
	return nil
}

func (m *MockLLMProvider) IsAvailable(ctx context.Context) bool {
	return true
}

func (m *MockLLMProvider) GetHealth(ctx context.Context) (*llm.ProviderHealth, error) {
	return &llm.ProviderHealth{
		Status: "healthy",
	}, nil
}

func (m *MockLLMProvider) Close() error {
	return nil
}

// setupTestRepo creates a temporary git repository for testing
func setupTestRepo(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to config git user.name: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to config git user.email: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// TestNewAutoCommitCoordinator tests creating a new coordinator
func TestNewAutoCommitCoordinator(t *testing.T) {
	t.Run("valid repository", func(t *testing.T) {
		repoPath, cleanup := setupTestRepo(t)
		defer cleanup()

		mockProvider := &MockLLMProvider{
			response: &llm.LLMResponse{
				ID:      uuid.New(),
				Content: "feat: add new feature",
			},
		}

		acc, err := NewAutoCommitCoordinator(repoPath, mockProvider)
		if err != nil {
			t.Fatalf("NewAutoCommitCoordinator() error = %v", err)
		}

		if acc == nil {
			t.Fatal("NewAutoCommitCoordinator() returned nil")
		}

		if acc.repoPath != repoPath {
			t.Errorf("repoPath = %v, want %v", acc.repoPath, repoPath)
		}
	})

	t.Run("invalid repository", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "not-git-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		mockProvider := &MockLLMProvider{}

		_, err = NewAutoCommitCoordinator(tmpDir, mockProvider)
		if err == nil {
			t.Error("NewAutoCommitCoordinator() expected error for non-git directory")
		}
	})
}

// TestAutoCommit tests the auto-commit functionality
func TestAutoCommit(t *testing.T) {
	t.Run("successful commit", func(t *testing.T) {
		repoPath, cleanup := setupTestRepo(t)
		defer cleanup()

		// Create and stage a test file
		testFile := filepath.Join(repoPath, "test.go")
		content := `package main

func NewFunction() {
	// New feature
}
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		// Stage the file manually first
		cmd := exec.Command("git", "add", testFile)
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		mockProvider := &MockLLMProvider{
			response: &llm.LLMResponse{
				ID:      uuid.New(),
				Content: "feat: add new function\n\nImplemented NewFunction for testing",
			},
		}

		acc, err := NewAutoCommitCoordinator(repoPath, mockProvider)
		if err != nil {
			t.Fatal(err)
		}

		opts := CommitOptions{
			Files: []string{},
			Author: Person{
				Name:  "Test User",
				Email: "test@example.com",
			},
		}

		result, err := acc.AutoCommit(context.Background(), opts)
		if err != nil {
			t.Fatalf("AutoCommit() error = %v", err)
		}

		if result == nil {
			t.Fatal("AutoCommit() returned nil result")
		}

		if result.Hash == "" {
			t.Error("AutoCommit() returned empty hash")
		}

		if !strings.Contains(result.Message, "feat:") {
			t.Errorf("AutoCommit() message = %v, want to contain 'feat:'", result.Message)
		}
	})

	t.Run("no changes", func(t *testing.T) {
		repoPath, cleanup := setupTestRepo(t)
		defer cleanup()

		mockProvider := &MockLLMProvider{}

		acc, err := NewAutoCommitCoordinator(repoPath, mockProvider)
		if err != nil {
			t.Fatal(err)
		}

		opts := CommitOptions{
			Files: []string{},
		}

		_, err = acc.AutoCommit(context.Background(), opts)
		if err == nil {
			t.Error("AutoCommit() expected error for no changes")
		}
	})

	t.Run("with attribution", func(t *testing.T) {
		repoPath, cleanup := setupTestRepo(t)
		defer cleanup()

		testFile := filepath.Join(repoPath, "test.go")
		if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
			t.Fatal(err)
		}

		// Stage the file first
		cmd := exec.Command("git", "add", testFile)
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		mockProvider := &MockLLMProvider{
			response: &llm.LLMResponse{
				ID:      uuid.New(),
				Content: "feat: add feature",
			},
		}

		acc, err := NewAutoCommitCoordinator(repoPath, mockProvider)
		if err != nil {
			t.Fatal(err)
		}

		opts := CommitOptions{
			Files: []string{},
			Attributions: []Attribution{
				{
					Type:  AttributionCoAuthor,
					Name:  "Claude",
					Email: "noreply@anthropic.com",
				},
			},
		}

		result, err := acc.AutoCommit(context.Background(), opts)
		if err != nil {
			t.Fatalf("AutoCommit() error = %v", err)
		}

		if !strings.Contains(result.Message, "Co-authored-by: Claude") {
			t.Errorf("AutoCommit() message should contain attribution")
		}
	})
}

// TestGenerateMessage tests message generation without committing
func TestGenerateMessage(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create and stage a test file
	testFile := filepath.Join(repoPath, "test.go")
	content := `package main

func TestFunction() {
	// Test code
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Stage the file
	cmd := exec.Command("git", "add", testFile)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	mockProvider := &MockLLMProvider{
		response: &llm.LLMResponse{
			ID:      uuid.New(),
			Content: "test: add test function",
		},
	}

	acc, err := NewAutoCommitCoordinator(repoPath, mockProvider)
	if err != nil {
		t.Fatal(err)
	}

	opts := MessageOptions{
		Format:   FormatConventional,
		Language: "en",
	}

	message, err := acc.GenerateMessage(context.Background(), opts)
	if err != nil {
		t.Fatalf("GenerateMessage() error = %v", err)
	}

	if message == "" {
		t.Error("GenerateMessage() returned empty message")
	}
}

// TestAmendDetector tests amend detection
func TestAmendDetector(t *testing.T) {
	t.Run("can amend unpushed commit", func(t *testing.T) {
		repoPath, cleanup := setupTestRepo(t)
		defer cleanup()

		// Rename branch to feature (not protected)
		cmd := exec.Command("git", "checkout", "-b", "feature")
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		// Create initial commit
		testFile := filepath.Join(repoPath, "test.txt")
		if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}

		cmd = exec.Command("git", "add", testFile)
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		cmd = exec.Command("git", "commit", "-m", "initial commit")
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		detector := NewAmendDetector(repoPath)
		canAmend, reason := detector.CanAmend(context.Background())

		if !canAmend {
			t.Errorf("CanAmend() = false, reason = %v, want true", reason)
		}
	})

	t.Run("cannot amend on protected branch", func(t *testing.T) {
		repoPath, cleanup := setupTestRepo(t)
		defer cleanup()

		// Create initial commit
		testFile := filepath.Join(repoPath, "test.txt")
		if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}

		cmd := exec.Command("git", "add", testFile)
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		cmd = exec.Command("git", "commit", "-m", "initial commit")
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		// Rename branch to main
		cmd = exec.Command("git", "branch", "-M", "main")
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		detector := NewAmendDetector(repoPath)
		canAmend, reason := detector.CanAmend(context.Background())

		if canAmend {
			t.Error("CanAmend() = true on main branch, want false")
		}

		if !strings.Contains(reason, "protected") {
			t.Errorf("CanAmend() reason = %v, want to contain 'protected'", reason)
		}
	})
}

// TestAttributionManager tests attribution management
func TestAttributionManager(t *testing.T) {
	t.Run("add co-author attribution", func(t *testing.T) {
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

		if !strings.Contains(result, "Co-authored-by: Claude <noreply@anthropic.com>") {
			t.Errorf("AddAttribution() = %v, want to contain co-author", result)
		}
	})

	t.Run("add multiple attributions", func(t *testing.T) {
		am := NewAttributionManager()

		message := "feat: add feature"
		attrs := []Attribution{
			{
				Type:  AttributionCoAuthor,
				Name:  "Claude",
				Email: "noreply@anthropic.com",
			},
			{
				Type:  AttributionReviewed,
				Name:  "Reviewer",
				Email: "reviewer@example.com",
			},
		}

		result := am.AddAttribution(message, attrs)

		if !strings.Contains(result, "Co-authored-by: Claude") {
			t.Error("AddAttribution() missing co-author")
		}

		if !strings.Contains(result, "Reviewed-by: Reviewer") {
			t.Error("AddAttribution() missing reviewer")
		}
	})
}

// TestDiffAnalyzer tests diff analysis
func TestDiffAnalyzer(t *testing.T) {
	t.Run("analyze feature addition", func(t *testing.T) {
		analyzer := NewDiffAnalyzer()

		diffs := []*Diff{
			{
				Path: "api/handler.go",
				Hunks: []*DiffHunk{
					{
						Lines: []DiffLine{
							{Type: LineAdd, Content: "func NewHandler() *Handler {"},
							{Type: LineAdd, Content: "	return &Handler{}"},
							{Type: LineAdd, Content: "}"},
						},
					},
				},
			},
		}

		analysis, err := analyzer.Analyze(context.Background(), diffs)
		if err != nil {
			t.Fatalf("Analyze() error = %v", err)
		}

		if analysis.ChangeType != TypeFeat {
			t.Errorf("Analyze() ChangeType = %v, want %v", analysis.ChangeType, TypeFeat)
		}

		if len(analysis.Summary.FunctionsAdded) == 0 {
			t.Error("Analyze() should detect function addition")
		}

		if analysis.Scope != "api" {
			t.Errorf("Analyze() Scope = %v, want api", analysis.Scope)
		}
	})

	t.Run("analyze test file", func(t *testing.T) {
		analyzer := NewDiffAnalyzer()

		diffs := []*Diff{
			{
				Path: "handler_test.go",
				Hunks: []*DiffHunk{
					{
						Lines: []DiffLine{
							{Type: LineAdd, Content: "func TestHandler(t *testing.T) {"},
							{Type: LineAdd, Content: "}"},
						},
					},
				},
			},
		}

		analysis, err := analyzer.Analyze(context.Background(), diffs)
		if err != nil {
			t.Fatalf("Analyze() error = %v", err)
		}

		if !analysis.Summary.TestsAdded {
			t.Error("Analyze() should detect test file")
		}

		if analysis.ChangeType != TypeTest {
			t.Errorf("Analyze() ChangeType = %v, want %v", analysis.ChangeType, TypeTest)
		}
	})

	t.Run("analyze documentation", func(t *testing.T) {
		analyzer := NewDiffAnalyzer()

		diffs := []*Diff{
			{
				Path: "README.md",
				Hunks: []*DiffHunk{
					{
						Lines: []DiffLine{
							{Type: LineAdd, Content: "# Documentation"},
							{Type: LineAdd, Content: "New section"},
						},
					},
				},
			},
		}

		analysis, err := analyzer.Analyze(context.Background(), diffs)
		if err != nil {
			t.Fatalf("Analyze() error = %v", err)
		}

		if !analysis.Summary.DocsModified {
			t.Error("Analyze() should detect documentation")
		}

		if analysis.ChangeType != TypeDocs {
			t.Errorf("Analyze() ChangeType = %v, want %v", analysis.ChangeType, TypeDocs)
		}
	})
}

// TestMessageGenerator tests message generation
func TestMessageGenerator(t *testing.T) {
	t.Run("generate conventional commit message", func(t *testing.T) {
		mockProvider := &MockLLMProvider{
			response: &llm.LLMResponse{
				ID:      uuid.New(),
				Content: "add user authentication",
			},
		}

		mg := NewMessageGenerator(mockProvider)

		diffs := []*Diff{
			{
				Path: "auth/auth.go",
				Hunks: []*DiffHunk{
					{
						Lines: []DiffLine{
							{Type: LineAdd, Content: "func Authenticate() {}"},
						},
					},
				},
			},
		}

		req := MessageRequest{
			Diffs:    diffs,
			Format:   FormatConventional,
			Language: "en",
		}

		msg, err := mg.Generate(context.Background(), req)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		if msg == nil {
			t.Fatal("Generate() returned nil")
		}

		// Should have conventional format
		if !strings.Contains(msg.Subject, ":") {
			t.Errorf("Generate() subject = %v, should contain ':'", msg.Subject)
		}
	})

	t.Run("cache messages", func(t *testing.T) {
		mockProvider := &MockLLMProvider{
			response: &llm.LLMResponse{
				ID:      uuid.New(),
				Content: "test message",
			},
		}

		mg := NewMessageGenerator(mockProvider)

		diffs := []*Diff{
			{
				Path: "test.go",
				Hunks: []*DiffHunk{
					{
						Lines: []DiffLine{
							{Type: LineAdd, Content: "test"},
						},
					},
				},
			},
		}

		req := MessageRequest{
			Diffs:  diffs,
			Format: FormatConventional,
		}

		// First call
		msg1, err := mg.Generate(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}

		// Second call should use cache
		msg2, err := mg.Generate(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}

		if msg1.Subject != msg2.Subject {
			t.Error("Generate() should return cached message")
		}
	})
}

// TestParseDiff tests diff parsing
func TestParseDiff(t *testing.T) {
	t.Run("parse simple diff", func(t *testing.T) {
		diffOutput := `diff --git a/test.go b/test.go
index 1234567..abcdefg 100644
--- a/test.go
+++ b/test.go
@@ -1,3 +1,4 @@
 package main

+func New() {}
 func Old() {}`

		diffs := parseDiff(diffOutput)

		if len(diffs) != 1 {
			t.Fatalf("parseDiff() got %d diffs, want 1", len(diffs))
		}

		if diffs[0].Path != "test.go" {
			t.Errorf("parseDiff() path = %v, want test.go", diffs[0].Path)
		}

		if len(diffs[0].Hunks) != 1 {
			t.Fatalf("parseDiff() got %d hunks, want 1", len(diffs[0].Hunks))
		}
	})

	t.Run("parse empty diff", func(t *testing.T) {
		diffs := parseDiff("")

		if diffs != nil {
			t.Errorf("parseDiff(\"\") = %v, want nil", diffs)
		}
	})
}

// TestMessageFormat tests message formatting
func TestMessageFormat(t *testing.T) {
	tests := []struct {
		name    string
		message *Message
		want    string
	}{
		{
			name: "subject only",
			message: &Message{
				Subject: "feat: add feature",
			},
			want: "feat: add feature",
		},
		{
			name: "subject and body",
			message: &Message{
				Subject: "feat: add feature",
				Body:    "Detailed description",
			},
			want: "feat: add feature\n\nDetailed description",
		},
		{
			name: "full message with footer",
			message: &Message{
				Subject: "feat: add feature",
				Body:    "Detailed description",
				Footer:  "Closes #123",
			},
			want: "feat: add feature\n\nDetailed description\n\nCloses #123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.message.FormatMessage()
			if got != tt.want {
				t.Errorf("FormatMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestExtractFunctionName tests function name extraction
func TestExtractFunctionName(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		language string
		want     string
	}{
		{
			name:     "go function",
			line:     "func NewHandler() *Handler {",
			language: "go",
			want:     "NewHandler",
		},
		{
			name:     "go method",
			line:     "func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {",
			language: "go",
			want:     "ServeHTTP",
		},
		{
			name:     "javascript function",
			line:     "function handleClick() {",
			language: "javascript",
			want:     "handleClick",
		},
		{
			name:     "python function",
			line:     "def process_data(data):",
			language: "python",
			want:     "process_data",
		},
		{
			name:     "no function",
			line:     "var x = 10",
			language: "go",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFunctionName(tt.line, tt.language)
			if got != tt.want {
				t.Errorf("extractFunctionName() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDetectLanguage tests language detection
func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"main.go", "go"},
		{"app.js", "javascript"},
		{"app.ts", "typescript"},
		{"script.py", "python"},
		{"Main.java", "java"},
		{"app.rb", "ruby"},
		{"unknown.txt", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := detectLanguage(tt.path)
			if got != tt.want {
				t.Errorf("detectLanguage(%v) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// TestAttributionParsing tests attribution parsing
func TestAttributionParsing(t *testing.T) {
	t.Run("parse co-author", func(t *testing.T) {
		line := "Co-authored-by: Claude <noreply@anthropic.com>"
		attr, err := ParseAttribution(line)
		if err != nil {
			t.Fatalf("ParseAttribution() error = %v", err)
		}

		if attr.Type != AttributionCoAuthor {
			t.Errorf("ParseAttribution() Type = %v, want %v", attr.Type, AttributionCoAuthor)
		}

		if attr.Name != "Claude" {
			t.Errorf("ParseAttribution() Name = %v, want Claude", attr.Name)
		}

		if attr.Email != "noreply@anthropic.com" {
			t.Errorf("ParseAttribution() Email = %v, want noreply@anthropic.com", attr.Email)
		}
	})

	t.Run("extract multiple attributions", func(t *testing.T) {
		message := `feat: add feature

Some description

Co-authored-by: Claude <noreply@anthropic.com>
Reviewed-by: John Doe <john@example.com>`

		attrs := ExtractAttributions(message)

		if len(attrs) != 2 {
			t.Fatalf("ExtractAttributions() got %d attributions, want 2", len(attrs))
		}

		if attrs[0].Type != AttributionCoAuthor {
			t.Errorf("ExtractAttributions() attrs[0].Type = %v, want %v", attrs[0].Type, AttributionCoAuthor)
		}

		if attrs[1].Type != AttributionReviewed {
			t.Errorf("ExtractAttributions() attrs[1].Type = %v, want %v", attrs[1].Type, AttributionReviewed)
		}
	})
}

// TestConfig tests configuration
func TestConfig(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		config := DefaultConfig()

		if config == nil {
			t.Fatal("DefaultConfig() returned nil")
		}

		if config.Message.Format != FormatConventional {
			t.Errorf("DefaultConfig().Message.Format = %v, want %v", config.Message.Format, FormatConventional)
		}

		if config.Amend.Enabled != true {
			t.Error("DefaultConfig().Amend.Enabled should be true")
		}

		if !config.Attribution.ClaudeAttribution {
			t.Error("DefaultConfig().Attribution.ClaudeAttribution should be true")
		}
	})
}

// Benchmark tests
func BenchmarkDiffAnalysis(b *testing.B) {
	analyzer := NewDiffAnalyzer()

	diffs := []*Diff{
		{
			Path: "test.go",
			Hunks: []*DiffHunk{
				{
					Lines: []DiffLine{
						{Type: LineAdd, Content: "func Test() {}"},
						{Type: LineAdd, Content: "func Another() {}"},
						{Type: LineDelete, Content: "func Old() {}"},
					},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = analyzer.Analyze(context.Background(), diffs)
	}
}

func BenchmarkMessageCache(b *testing.B) {
	cache := NewMessageCache(15 * time.Minute)
	msg := &Message{
		Subject: "test message",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash := "test-hash"
		cache.Set(hash, msg)
		_, _ = cache.Get(hash)
	}
}
