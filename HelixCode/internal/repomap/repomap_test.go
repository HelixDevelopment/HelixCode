package repomap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Test RepoMap creation and initialization
func TestNewRepoMap(t *testing.T) {
	tempDir := t.TempDir()

	config := DefaultConfig()
	rm, err := NewRepoMap(tempDir, config)

	if err != nil {
		t.Fatalf("Failed to create RepoMap: %v", err)
	}

	if rm == nil {
		t.Fatal("RepoMap is nil")
	}

	if rm.rootPath != tempDir {
		t.Errorf("Expected root path %s, got %s", tempDir, rm.rootPath)
	}
}

func TestNewRepoMapInvalidPath(t *testing.T) {
	config := DefaultConfig()
	_, err := NewRepoMap("/nonexistent/path/to/nowhere", config)

	if err == nil {
		t.Fatal("Expected error for invalid path, got nil")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.MaxFiles <= 0 {
		t.Error("MaxFiles should be positive")
	}

	if config.TokenBudget <= 0 {
		t.Error("TokenBudget should be positive")
	}

	if len(config.LanguagesSupported) == 0 {
		t.Error("LanguagesSupported should not be empty")
	}

	expectedLanguages := []string{"go", "python", "javascript", "typescript", "java", "c", "cpp", "rust", "ruby"}
	if len(config.LanguagesSupported) != len(expectedLanguages) {
		t.Errorf("Expected %d languages, got %d", len(expectedLanguages), len(config.LanguagesSupported))
	}
}

// Test language detection
func TestDetectLanguage(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultConfig()
	rm, _ := NewRepoMap(tempDir, config)

	tests := []struct {
		filename string
		expected string
	}{
		{"test.go", "go"},
		{"test.py", "python"},
		{"test.js", "javascript"},
		{"test.jsx", "javascript"},
		{"test.ts", "typescript"},
		{"test.tsx", "typescript"},
		{"test.java", "java"},
		{"test.c", "c"},
		{"test.h", "c"},
		{"test.cpp", "cpp"},
		{"test.cc", "cpp"},
		{"test.cxx", "cpp"},
		{"test.hpp", "cpp"},
		{"test.rs", "rust"},
		{"test.rb", "ruby"},
		{"test.txt", ""},
		{"test.md", ""},
	}

	for _, tt := range tests {
		result := rm.detectLanguage(tt.filename)
		if result != tt.expected {
			t.Errorf("detectLanguage(%s) = %s, expected %s", tt.filename, result, tt.expected)
		}
	}
}

// Test file discovery
func TestDiscoverFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	createTestFile(t, tempDir, "main.go", "package main\n\nfunc main() {}\n")
	createTestFile(t, tempDir, "utils.go", "package main\n\nfunc helper() {}\n")
	createTestFile(t, tempDir, "README.md", "# Test\n")

	// Create subdirectory
	subdir := filepath.Join(tempDir, "pkg")
	os.MkdirAll(subdir, 0755)
	createTestFile(t, subdir, "lib.go", "package pkg\n\nfunc Lib() {}\n")

	config := DefaultConfig()
	rm, _ := NewRepoMap(tempDir, config)

	files, err := rm.discoverFiles()
	if err != nil {
		t.Fatalf("Failed to discover files: %v", err)
	}

	// Should find 3 Go files, not the markdown file
	if len(files) != 3 {
		t.Errorf("Expected 3 files, found %d", len(files))
	}

	// Verify all files are Go files
	for _, file := range files {
		if !strings.HasSuffix(file, ".go") {
			t.Errorf("Found non-Go file: %s", file)
		}
	}
}

func TestDiscoverFilesIgnoresCommonDirs(t *testing.T) {
	tempDir := t.TempDir()

	// Create files in ignored directories
	ignoredDirs := []string{".git", "node_modules", "vendor", ".helix", "__pycache__", "dist", "build", "target"}

	for _, dir := range ignoredDirs {
		dirPath := filepath.Join(tempDir, dir)
		os.MkdirAll(dirPath, 0755)
		createTestFile(t, dirPath, "test.go", "package test")
	}

	// Create a file in the root
	createTestFile(t, tempDir, "main.go", "package main")

	config := DefaultConfig()
	rm, _ := NewRepoMap(tempDir, config)

	files, err := rm.discoverFiles()
	if err != nil {
		t.Fatalf("Failed to discover files: %v", err)
	}

	// Should only find the root file
	if len(files) != 1 {
		t.Errorf("Expected 1 file, found %d (ignored directories not working)", len(files))
	}
}

// Test symbol extraction
func TestExtractFileSymbols(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	code := `package main

// MyFunc is a test function
func MyFunc() {
	println("hello")
}

// MyStruct is a test struct
type MyStruct struct {
	Field string
}

// MyMethod is a method on MyStruct
func (m *MyStruct) MyMethod() {
	println(m.Field)
}
`

	createTestFile(t, tempDir, "test.go", code)

	config := DefaultConfig()
	config.CacheEnabled = false // Disable cache for testing
	rm, _ := NewRepoMap(tempDir, config)

	symbols, err := rm.extractFileSymbols(testFile)
	if err != nil {
		t.Fatalf("Failed to extract symbols: %v", err)
	}

	// Should find function, struct, and method
	if len(symbols) < 2 {
		t.Errorf("Expected at least 2 symbols, got %d", len(symbols))
	}

	// Check for MyFunc
	foundFunc := false
	for _, sym := range symbols {
		if sym.Name == "MyFunc" && sym.Type == SymbolTypeFunction {
			foundFunc = true
			if sym.LineStart <= 0 {
				t.Error("Symbol should have line number")
			}
		}
	}

	if !foundFunc {
		t.Error("Did not find MyFunc symbol")
	}
}

// Test GetOptimalContext
func TestGetOptimalContext(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	createTestFile(t, tempDir, "main.go", `package main

func main() {
	helper()
}
`)

	createTestFile(t, tempDir, "helper.go", `package main

func helper() {
	println("helper")
}
`)

	config := DefaultConfig()
	config.CacheEnabled = false
	config.TokenBudget = 10000
	rm, _ := NewRepoMap(tempDir, config)

	contexts, err := rm.GetOptimalContext("helper", []string{})
	if err != nil {
		t.Fatalf("GetOptimalContext failed: %v", err)
	}

	if len(contexts) == 0 {
		t.Error("Expected at least one context")
	}

	// Should find helper.go with higher relevance
	foundHelper := false
	for _, ctx := range contexts {
		if strings.Contains(ctx.FilePath, "helper.go") {
			foundHelper = true
		}
	}

	if !foundHelper {
		t.Error("Expected to find helper.go in context")
	}
}

func TestGetOptimalContextWithChangedFiles(t *testing.T) {
	tempDir := t.TempDir()

	file1 := filepath.Join(tempDir, "file1.go")

	createTestFile(t, tempDir, "file1.go", "package main\nfunc File1() {}")
	createTestFile(t, tempDir, "file2.go", "package main\nfunc File2() {}")

	config := DefaultConfig()
	config.CacheEnabled = false
	rm, _ := NewRepoMap(tempDir, config)

	contexts, err := rm.GetOptimalContext("", []string{file1})
	if err != nil {
		t.Fatalf("GetOptimalContext failed: %v", err)
	}

	// Changed file should appear first
	if len(contexts) > 0 && contexts[0].FilePath != file1 {
		t.Error("Changed file should have highest relevance")
	}
}

func TestGetOptimalContextTokenBudget(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple large files
	largeContent := strings.Repeat("package main\nfunc Test() {}\n", 1000)

	for i := 0; i < 10; i++ {
		createTestFile(t, tempDir, filepath.Join("file", string(rune('0'+i))+".go"), largeContent)
	}

	config := DefaultConfig()
	config.CacheEnabled = false
	config.TokenBudget = 100 // Very small budget
	rm, _ := NewRepoMap(tempDir, config)

	contexts, err := rm.GetOptimalContext("", []string{})
	if err != nil {
		t.Fatalf("GetOptimalContext failed: %v", err)
	}

	// Should respect token budget
	totalTokens := 0
	for _, ctx := range contexts {
		totalTokens += ctx.TokenCount
	}

	if totalTokens > config.TokenBudget {
		t.Errorf("Total tokens %d exceeds budget %d", totalTokens, config.TokenBudget)
	}
}

// Test cache functionality
func TestCacheEnabled(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	createTestFile(t, tempDir, "test.go", "package main\nfunc Test() {}")

	config := DefaultConfig()
	config.CacheEnabled = true
	rm, _ := NewRepoMap(tempDir, config)

	// First call - not cached
	symbols1, err := rm.extractFileSymbols(testFile)
	if err != nil {
		t.Fatalf("Failed to extract symbols: %v", err)
	}

	// Second call - should be cached
	symbols2, err := rm.extractFileSymbols(testFile)
	if err != nil {
		t.Fatalf("Failed to extract symbols: %v", err)
	}

	if len(symbols1) != len(symbols2) {
		t.Error("Cached symbols differ from original")
	}
}

func TestInvalidateFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	createTestFile(t, tempDir, "test.go", "package main\nfunc Test() {}")

	config := DefaultConfig()
	config.CacheEnabled = true
	rm, _ := NewRepoMap(tempDir, config)

	// Cache the file
	rm.extractFileSymbols(testFile)

	// Wait for async save to complete
	time.Sleep(100 * time.Millisecond)

	// Invalidate
	err := rm.InvalidateFile(testFile)
	if err != nil {
		t.Fatalf("Failed to invalidate file: %v", err)
	}

	// Wait for async removal to complete
	time.Sleep(100 * time.Millisecond)
}

func TestRefreshCache(t *testing.T) {
	tempDir := t.TempDir()

	createTestFile(t, tempDir, "test.go", "package main\nfunc Test() {}")

	config := DefaultConfig()
	config.CacheEnabled = true
	rm, _ := NewRepoMap(tempDir, config)

	err := rm.RefreshCache()
	if err != nil {
		t.Fatalf("RefreshCache failed: %v", err)
	}

	// Wait for async save operations to complete
	time.Sleep(100 * time.Millisecond)
}

func TestRefreshCacheDisabled(t *testing.T) {
	tempDir := t.TempDir()

	config := DefaultConfig()
	config.CacheEnabled = false
	rm, _ := NewRepoMap(tempDir, config)

	err := rm.RefreshCache()
	if err == nil {
		t.Error("Expected error when cache is disabled")
	}
}

// Test statistics
func TestGetStatistics(t *testing.T) {
	tempDir := t.TempDir()

	createTestFile(t, tempDir, "test.go", "package main\nfunc Test() {}")
	createTestFile(t, tempDir, "test.py", "def test():\n    pass")

	config := DefaultConfig()
	config.CacheEnabled = false
	rm, _ := NewRepoMap(tempDir, config)

	stats, err := rm.GetStatistics()
	if err != nil {
		t.Fatalf("GetStatistics failed: %v", err)
	}

	if stats.TotalFiles != 2 {
		t.Errorf("Expected 2 files, got %d", stats.TotalFiles)
	}

	if len(stats.Languages) == 0 {
		t.Error("Expected language statistics")
	}

	if stats.IndexingDuration == 0 {
		t.Error("Expected non-zero indexing duration")
	}
}

// Test TreeSitterParser
func TestNewTreeSitterParser(t *testing.T) {
	parser := NewTreeSitterParser()

	if parser == nil {
		t.Fatal("Parser is nil")
	}

	langs := parser.SupportedLanguages()
	if len(langs) < 8 {
		t.Errorf("Expected at least 8 supported languages, got %d", len(langs))
	}
}

func TestParseFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	createTestFile(t, tempDir, "test.go", "package main\n\nfunc main() {}\n")

	parser := NewTreeSitterParser()
	tree, err := parser.ParseFile(testFile, "go")

	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if tree == nil {
		t.Fatal("Tree is nil")
	}
}

func TestParseFileUnsupportedLanguage(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.unknown")

	createTestFile(t, tempDir, "test.unknown", "unknown content")

	parser := NewTreeSitterParser()
	_, err := parser.ParseFile(testFile, "unknown")

	if err == nil {
		t.Error("Expected error for unsupported language")
	}
}

func TestExtractSymbols(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	code := `package main

func MyFunction() {
	println("test")
}

type MyStruct struct {
	Field string
}
`

	createTestFile(t, tempDir, "test.go", code)

	parser := NewTreeSitterParser()
	tree, _ := parser.ParseFile(testFile, "go")

	symbols, err := parser.ExtractSymbols(tree, testFile, "go")
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}

	if len(symbols) == 0 {
		t.Error("Expected to extract symbols")
	}

	// Should find at least the function
	foundFunc := false
	for _, sym := range symbols {
		if sym.Name == "MyFunction" {
			foundFunc = true
		}
	}

	if !foundFunc {
		t.Error("Did not find MyFunction")
	}
}

// Test TagExtractor
func TestNewTagExtractor(t *testing.T) {
	extractor := NewTagExtractor("go")

	if extractor == nil {
		t.Fatal("Extractor is nil")
	}

	if extractor.language != "go" {
		t.Error("Language not set correctly")
	}
}

func TestSymbolTypes(t *testing.T) {
	types := []SymbolType{
		SymbolTypeFunction,
		SymbolTypeMethod,
		SymbolTypeClass,
		SymbolTypeInterface,
		SymbolTypeStruct,
		SymbolTypeEnum,
		SymbolTypeTrait,
		SymbolTypeModule,
		SymbolTypeVariable,
		SymbolTypeConstant,
		SymbolTypeImport,
		SymbolTypeExport,
	}

	for _, st := range types {
		if string(st) == "" {
			t.Errorf("Symbol type %v is empty", st)
		}
	}
}

// Test FileRanker
func TestNewFileRanker(t *testing.T) {
	ranker := NewFileRanker()

	if ranker == nil {
		t.Fatal("Ranker is nil")
	}

	if ranker.weights.SymbolMatch <= 0 {
		t.Error("Symbol match weight should be positive")
	}
}

func TestRankFiles(t *testing.T) {
	ranker := NewFileRanker()

	symbols := []Symbol{
		{
			Name:     "TestFunc",
			Type:     SymbolTypeFunction,
			FilePath: "/test/file1.go",
		},
		{
			Name:     "HelperFunc",
			Type:     SymbolTypeFunction,
			FilePath: "/test/file2.go",
		},
	}

	scores := ranker.RankFiles(symbols, "test", []string{})

	if len(scores) != 2 {
		t.Errorf("Expected 2 scores, got %d", len(scores))
	}

	// file1.go should rank higher due to "TestFunc" matching "test"
	var file1Score, file2Score float64
	for _, score := range scores {
		if strings.Contains(score.FilePath, "file1.go") {
			file1Score = score.Score
		} else if strings.Contains(score.FilePath, "file2.go") {
			file2Score = score.Score
		}
	}

	if file1Score <= file2Score {
		t.Error("Expected file1.go to have higher score due to symbol match")
	}
}

func TestRankFilesWithChangedFiles(t *testing.T) {
	ranker := NewFileRanker()

	symbols := []Symbol{
		{Name: "Func", Type: SymbolTypeFunction, FilePath: "/test/file1.go"},
		{Name: "Func", Type: SymbolTypeFunction, FilePath: "/test/file2.go"},
	}

	changedFiles := []string{"/test/file2.go"}
	scores := ranker.RankFiles(symbols, "", changedFiles)

	// file2.go should rank higher because it was changed
	var file1Score, file2Score float64
	for _, score := range scores {
		if score.FilePath == "/test/file1.go" {
			file1Score = score.Score
		} else if score.FilePath == "/test/file2.go" {
			file2Score = score.Score
		}
	}

	if file2Score <= file1Score {
		t.Error("Changed file should have higher score")
	}
}

func TestTokenizeQuery(t *testing.T) {
	ranker := NewFileRanker()

	tests := []struct {
		query    string
		expected int
	}{
		{"simple query", 2},
		{"test_function_name", 3},
		{"TestCamelCase", 1},
		{"test-kebab-case", 3},
		{"", 0},
	}

	for _, tt := range tests {
		tokens := ranker.tokenizeQuery(tt.query)
		if len(tokens) != tt.expected {
			t.Errorf("tokenizeQuery(%q) returned %d tokens, expected %d", tt.query, len(tokens), tt.expected)
		}
	}
}

func TestTokenizeSymbolName(t *testing.T) {
	ranker := NewFileRanker()

	tests := []struct {
		name     string
		expected int
	}{
		{"MyFunction", 2},
		{"my_function", 2},
		{"my-function", 2},
		{"myfunction", 1},
		{"HTTPServer", 1},
	}

	for _, tt := range tests {
		tokens := ranker.tokenizeSymbolName(tt.name)
		if len(tokens) < 1 {
			t.Errorf("tokenizeSymbolName(%q) returned %d tokens, expected at least 1", tt.name, len(tokens))
		}
	}
}

// Test Cache
func TestNewRepoCache(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")

	cache, err := NewRepoCache(cacheDir, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	if cache == nil {
		t.Fatal("Cache is nil")
	}

	// Check directory was created
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Error("Cache directory was not created")
	}
}

func TestCacheGetSet(t *testing.T) {
	tempDir := t.TempDir()
	cache, _ := NewRepoCache(filepath.Join(tempDir, "cache"), 1*time.Hour)

	key := "test-key"
	value := "test-value"

	// Set value
	cache.Set(key, value)

	// Wait for async save to complete
	time.Sleep(50 * time.Millisecond)

	// Get value
	retrieved, found := cache.Get(key)
	if !found {
		t.Error("Value not found in cache")
	}

	if retrieved.(string) != value {
		t.Errorf("Retrieved value %v != original value %v", retrieved, value)
	}
}

func TestCacheInvalidate(t *testing.T) {
	tempDir := t.TempDir()
	cache, _ := NewRepoCache(filepath.Join(tempDir, "cache"), 1*time.Hour)

	key := "test-key"
	cache.Set(key, "value")

	// Wait for async save to complete
	time.Sleep(50 * time.Millisecond)

	cache.Invalidate(key)

	// Wait for async removal to complete
	time.Sleep(50 * time.Millisecond)

	_, found := cache.Get(key)
	if found {
		t.Error("Value should not be found after invalidation")
	}
}

func TestCacheExpiration(t *testing.T) {
	tempDir := t.TempDir()
	cache, _ := NewRepoCache(filepath.Join(tempDir, "cache"), 1*time.Millisecond)

	key := "test-key"
	cache.Set(key, "value")

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	_, found := cache.Get(key)
	if found {
		t.Error("Value should have expired")
	}
}

func TestCacheSize(t *testing.T) {
	tempDir := t.TempDir()
	cache, _ := NewRepoCache(filepath.Join(tempDir, "cache"), 1*time.Hour)

	if cache.Size() != 0 {
		t.Error("New cache should be empty")
	}

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	// Wait for async save to complete
	time.Sleep(50 * time.Millisecond)

	if cache.Size() != 2 {
		t.Errorf("Expected cache size 2, got %d", cache.Size())
	}
}

func TestCacheCleanup(t *testing.T) {
	tempDir := t.TempDir()
	cache, _ := NewRepoCache(filepath.Join(tempDir, "cache"), 1*time.Millisecond)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	time.Sleep(10 * time.Millisecond)

	removed := cache.Cleanup()
	if removed != 2 {
		t.Errorf("Expected to remove 2 entries, removed %d", removed)
	}

	if cache.Size() != 0 {
		t.Error("Cache should be empty after cleanup")
	}
}

func TestCacheGetStats(t *testing.T) {
	tempDir := t.TempDir()
	cache, _ := NewRepoCache(filepath.Join(tempDir, "cache"), 1*time.Hour)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	// Wait for async save to complete
	time.Sleep(50 * time.Millisecond)

	stats := cache.GetStats()

	if stats.TotalEntries != 2 {
		t.Errorf("Expected 2 total entries, got %d", stats.TotalEntries)
	}

	if stats.TotalSize <= 0 {
		t.Error("Expected positive total size")
	}
}

func TestCacheGetOrCompute(t *testing.T) {
	tempDir := t.TempDir()
	cache, _ := NewRepoCache(filepath.Join(tempDir, "cache"), 1*time.Hour)

	computeCount := 0
	compute := func() (interface{}, error) {
		computeCount++
		return "computed", nil
	}

	// First call - should compute
	value1, _ := cache.GetOrCompute("key", compute)
	if value1.(string) != "computed" {
		t.Error("Expected computed value")
	}
	if computeCount != 1 {
		t.Error("Compute should have been called once")
	}

	// Wait for async save to complete
	time.Sleep(50 * time.Millisecond)

	// Second call - should use cache
	value2, _ := cache.GetOrCompute("key", compute)
	if value2.(string) != "computed" {
		t.Error("Expected cached value")
	}
	if computeCount != 1 {
		t.Error("Compute should not have been called again")
	}
}

func TestCacheHas(t *testing.T) {
	tempDir := t.TempDir()
	cache, _ := NewRepoCache(filepath.Join(tempDir, "cache"), 1*time.Hour)

	if cache.Has("key") {
		t.Error("Cache should not have key")
	}

	cache.Set("key", "value")

	// Wait for async save to complete
	time.Sleep(50 * time.Millisecond)

	if !cache.Has("key") {
		t.Error("Cache should have key")
	}
}

func TestCacheKeys(t *testing.T) {
	tempDir := t.TempDir()
	cache, _ := NewRepoCache(filepath.Join(tempDir, "cache"), 1*time.Hour)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	// Wait for async save to complete
	time.Sleep(50 * time.Millisecond)

	keys := cache.Keys()

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}
}

func TestGetLanguageQueries(t *testing.T) {
	languages := []string{"go", "python", "javascript", "typescript", "java", "c", "cpp", "rust", "ruby"}

	for _, lang := range languages {
		queries := GetLanguageQueries(lang)
		if queries == nil {
			t.Errorf("Expected queries for language %s, got nil", lang)
		}
	}

	// Test unsupported language
	queries := GetLanguageQueries("unknown")
	if queries != nil {
		t.Error("Expected nil for unsupported language")
	}
}

// Helper functions

func createTestFile(t *testing.T, dir, filename, content string) {
	path := filepath.Join(dir, filename)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
}
