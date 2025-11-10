package repomap

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// RepoMap represents the semantic codebase mapping system
type RepoMap struct {
	rootPath string
	parser   *TreeSitterParser
	cache    *RepoCache
	ranker   *FileRanker
	config   RepoMapConfig
	mu       sync.RWMutex
}

// RepoMapConfig configures the behavior of RepoMap
type RepoMapConfig struct {
	MaxFiles           int
	TokenBudget        int
	CacheEnabled       bool
	LanguagesSupported []string
	CacheTTL           time.Duration
	MaxConcurrency     int
}

// FileContext represents contextual information about a file
type FileContext struct {
	FilePath   string
	Symbols    []Symbol
	Content    string
	Relevance  float64
	TokenCount int
}

// RepoMapStats provides statistics about the repository mapping
type RepoMapStats struct {
	TotalFiles       int
	TotalSymbols     int
	CachedFiles      int
	Languages        map[string]int
	LastRefresh      time.Time
	IndexingDuration time.Duration
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() RepoMapConfig {
	return RepoMapConfig{
		MaxFiles:           100,
		TokenBudget:        8000,
		CacheEnabled:       true,
		LanguagesSupported: []string{"go", "python", "javascript", "typescript", "java", "c", "cpp", "rust", "ruby"},
		CacheTTL:           24 * time.Hour,
		MaxConcurrency:     4,
	}
}

// NewRepoMap creates a new RepoMap instance
func NewRepoMap(rootPath string, config RepoMapConfig) (*RepoMap, error) {
	// Validate root path
	if _, err := os.Stat(rootPath); err != nil {
		return nil, fmt.Errorf("invalid root path: %w", err)
	}

	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Initialize parser
	parser := NewTreeSitterParser()

	// Initialize cache if enabled
	var cache *RepoCache
	if config.CacheEnabled {
		cacheDir := filepath.Join(absPath, ".helix", "cache")
		cache, err = NewRepoCache(cacheDir, config.CacheTTL)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize cache: %w", err)
		}
	}

	// Initialize ranker
	ranker := NewFileRanker()

	rm := &RepoMap{
		rootPath: absPath,
		parser:   parser,
		cache:    cache,
		ranker:   ranker,
		config:   config,
	}

	return rm, nil
}

// GetOptimalContext returns the most relevant file contexts for a given query
func (rm *RepoMap) GetOptimalContext(query string, changedFiles []string) ([]FileContext, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Get all relevant files
	files, err := rm.discoverFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to discover files: %w", err)
	}

	// Extract symbols from all files
	allSymbols := make([]Symbol, 0)
	symbolsByFile := make(map[string][]Symbol)

	for _, file := range files {
		symbols, err := rm.extractFileSymbols(file)
		if err != nil {
			// Skip files that can't be parsed
			continue
		}
		allSymbols = append(allSymbols, symbols...)
		symbolsByFile[file] = symbols
	}

	// Rank files based on relevance
	fileScores := rm.ranker.RankFiles(allSymbols, query, changedFiles)

	// Sort by score (highest first)
	sort.Slice(fileScores, func(i, j int) bool {
		return fileScores[i].Score > fileScores[j].Score
	})

	// Build file contexts within token budget
	contexts := make([]FileContext, 0)
	currentTokens := 0

	for i := 0; i < len(fileScores) && i < rm.config.MaxFiles; i++ {
		score := fileScores[i]

		// Estimate token count (rough approximation: 1 token per 4 chars)
		content, err := rm.readFileContent(score.FilePath)
		if err != nil {
			continue
		}

		tokenCount := len(content) / 4
		if currentTokens+tokenCount > rm.config.TokenBudget {
			break
		}

		ctx := FileContext{
			FilePath:   score.FilePath,
			Symbols:    symbolsByFile[score.FilePath],
			Content:    content,
			Relevance:  score.Score,
			TokenCount: tokenCount,
		}

		contexts = append(contexts, ctx)
		currentTokens += tokenCount
	}

	return contexts, nil
}

// RefreshCache invalidates and rebuilds the cache
func (rm *RepoMap) RefreshCache() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.cache == nil {
		return fmt.Errorf("cache is not enabled")
	}

	// Discover all files
	files, err := rm.discoverFiles()
	if err != nil {
		return fmt.Errorf("failed to discover files: %w", err)
	}

	// Re-parse and cache all files
	for _, file := range files {
		symbols, err := rm.extractFileSymbols(file)
		if err != nil {
			continue
		}

		cacheKey := rm.getCacheKey(file)
		rm.cache.Set(cacheKey, symbols)
	}

	return nil
}

// GetStatistics returns statistics about the repository mapping
func (rm *RepoMap) GetStatistics() (RepoMapStats, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	startTime := time.Now()

	files, err := rm.discoverFiles()
	if err != nil {
		return RepoMapStats{}, fmt.Errorf("failed to discover files: %w", err)
	}

	stats := RepoMapStats{
		TotalFiles:       len(files),
		Languages:        make(map[string]int),
		IndexingDuration: time.Since(startTime),
	}

	totalSymbols := 0
	cachedFiles := 0

	for _, file := range files {
		lang := rm.detectLanguage(file)
		if lang != "" {
			stats.Languages[lang]++
		}

		cacheKey := rm.getCacheKey(file)
		if rm.cache != nil {
			if _, found := rm.cache.Get(cacheKey); found {
				cachedFiles++
			}
		}

		symbols, err := rm.extractFileSymbols(file)
		if err == nil {
			totalSymbols += len(symbols)
		}
	}

	stats.TotalSymbols = totalSymbols
	stats.CachedFiles = cachedFiles
	stats.LastRefresh = time.Now()

	return stats, nil
}

// discoverFiles finds all relevant source files in the repository
func (rm *RepoMap) discoverFiles() ([]string, error) {
	files := make([]string, 0)

	err := filepath.Walk(rm.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible files
		}

		// Skip directories and hidden files
		if info.IsDir() {
			// Skip common directories to ignore
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" ||
				name == ".helix" || name == "__pycache__" || name == "dist" ||
				name == "build" || name == "target" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file is a supported language
		if rm.isSupportedFile(path) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// isSupportedFile checks if a file is in a supported language
func (rm *RepoMap) isSupportedFile(filePath string) bool {
	lang := rm.detectLanguage(filePath)
	if lang == "" {
		return false
	}

	for _, supported := range rm.config.LanguagesSupported {
		if supported == lang {
			return true
		}
	}

	return false
}

// detectLanguage detects the programming language from file extension
func (rm *RepoMap) detectLanguage(filePath string) string {
	ext := filepath.Ext(filePath)

	langMap := map[string]string{
		".go":   "go",
		".py":   "python",
		".js":   "javascript",
		".jsx":  "javascript",
		".ts":   "typescript",
		".tsx":  "typescript",
		".java": "java",
		".c":    "c",
		".h":    "c",
		".cpp":  "cpp",
		".cc":   "cpp",
		".cxx":  "cpp",
		".hpp":  "cpp",
		".rs":   "rust",
		".rb":   "ruby",
	}

	return langMap[ext]
}

// extractFileSymbols extracts symbols from a file with caching
func (rm *RepoMap) extractFileSymbols(filePath string) ([]Symbol, error) {
	// Check cache first
	if rm.cache != nil {
		cacheKey := rm.getCacheKey(filePath)
		if cached, found := rm.cache.Get(cacheKey); found {
			if symbols, ok := cached.([]Symbol); ok {
				return symbols, nil
			}
		}
	}

	// Parse file and extract symbols
	lang := rm.detectLanguage(filePath)
	if lang == "" {
		return nil, fmt.Errorf("unsupported file type")
	}

	tree, err := rm.parser.ParseFile(filePath, lang)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	symbols, err := rm.parser.ExtractSymbols(tree, filePath, lang)
	if err != nil {
		return nil, fmt.Errorf("failed to extract symbols: %w", err)
	}

	// Cache the symbols
	if rm.cache != nil {
		cacheKey := rm.getCacheKey(filePath)
		rm.cache.Set(cacheKey, symbols)
	}

	return symbols, nil
}

// readFileContent reads the content of a file
func (rm *RepoMap) readFileContent(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// getCacheKey generates a cache key for a file
func (rm *RepoMap) getCacheKey(filePath string) string {
	relPath, err := filepath.Rel(rm.rootPath, filePath)
	if err != nil {
		return filePath
	}

	// Include file modification time in cache key
	info, err := os.Stat(filePath)
	if err != nil {
		return relPath
	}

	return fmt.Sprintf("%s:%d", relPath, info.ModTime().Unix())
}

// InvalidateFile invalidates the cache for a specific file
func (rm *RepoMap) InvalidateFile(filePath string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.cache == nil {
		return nil
	}

	cacheKey := rm.getCacheKey(filePath)
	rm.cache.Invalidate(cacheKey)
	return nil
}

// GetSupportedLanguages returns the list of supported languages
func (rm *RepoMap) GetSupportedLanguages() []string {
	return rm.config.LanguagesSupported
}
