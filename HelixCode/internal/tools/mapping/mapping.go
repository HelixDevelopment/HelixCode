package mapping

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"
	"unicode"
)

// CodebaseMap represents a complete codebase map
type CodebaseMap struct {
	Root         string                `json:"root"`
	Files        map[string]*FileMap   `json:"files"`
	Languages    map[string]int        `json:"languages"`
	TotalFiles   int                   `json:"total_files"`
	TotalLines   int                   `json:"total_lines"`
	TotalTokens  int                   `json:"total_tokens"`
	Definitions  map[string]*Definition `json:"definitions"`
	Dependencies map[string][]string   `json:"dependencies"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
	Version      string                `json:"version"`
}

// FileMap represents a single file's map
type FileMap struct {
	Path        string        `json:"path"`
	Language    string        `json:"language"`
	Size        int64         `json:"size"`
	Lines       int           `json:"lines"`
	Tokens      int           `json:"tokens"`
	Definitions []*Definition `json:"definitions"`
	Imports     []*Import     `json:"imports"`
	Exports     []*Export     `json:"exports,omitempty"`
	Comments    []*Comment    `json:"comments,omitempty"`
	Complexity  int           `json:"complexity"`
	Checksum    string        `json:"checksum"`
	ParsedAt    time.Time     `json:"parsed_at"`
}

// MapOptions configures codebase mapping
type MapOptions struct {
	UseCache      bool     `json:"use_cache"`
	Concurrency   int      `json:"concurrency"`
	MaxFileSize   int64    `json:"max_file_size"`
	ExcludeDirs   []string `json:"exclude_dirs"`
	IncludeHidden bool     `json:"include_hidden"`
	Languages     []string `json:"languages,omitempty"`
}

// DefaultMapOptions returns default mapping options
func DefaultMapOptions() *MapOptions {
	return &MapOptions{
		UseCache:    true,
		Concurrency: 10,
		MaxFileSize: 1 * 1024 * 1024, // 1 MB
		ExcludeDirs: []string{
			".git",
			"node_modules",
			"vendor",
			".helix.cache",
			"build",
			"dist",
			"target",
			".venv",
			"venv",
			"__pycache__",
			".next",
			".nuxt",
			"coverage",
		},
		IncludeHidden: false,
	}
}

// NewCodebaseMap creates a new codebase map
func NewCodebaseMap(root string) *CodebaseMap {
	return &CodebaseMap{
		Root:         root,
		Files:        make(map[string]*FileMap),
		Languages:    make(map[string]int),
		Definitions:  make(map[string]*Definition),
		Dependencies: make(map[string][]string),
		CreatedAt:    time.Now(),
		Version:      CacheVersion,
	}
}

// AddFile adds a file map to the codebase map
func (cm *CodebaseMap) AddFile(fileMap *FileMap) {
	cm.Files[fileMap.Path] = fileMap
	cm.TotalFiles++
	cm.TotalLines += fileMap.Lines
	cm.TotalTokens += fileMap.Tokens
	cm.Languages[fileMap.Language]++

	// Add definitions
	for _, def := range fileMap.Definitions {
		if def.QualifiedName != "" {
			cm.Definitions[def.QualifiedName] = def
		}
	}

	cm.UpdatedAt = time.Now()
}

// RemoveFile removes a file map from the codebase map
func (cm *CodebaseMap) RemoveFile(path string) {
	fileMap, ok := cm.Files[path]
	if !ok {
		return
	}

	cm.TotalFiles--
	cm.TotalLines -= fileMap.Lines
	cm.TotalTokens -= fileMap.Tokens
	cm.Languages[fileMap.Language]--

	// Remove definitions
	for _, def := range fileMap.Definitions {
		if def.QualifiedName != "" {
			delete(cm.Definitions, def.QualifiedName)
		}
	}

	delete(cm.Files, path)
	delete(cm.Dependencies, path)

	cm.UpdatedAt = time.Now()
}

// GetDefinition gets a definition by qualified name
func (cm *CodebaseMap) GetDefinition(qualifiedName string) (*Definition, bool) {
	def, ok := cm.Definitions[qualifiedName]
	return def, ok
}

// GetFilesByLanguage returns all files for a specific language
func (cm *CodebaseMap) GetFilesByLanguage(language string) []*FileMap {
	var files []*FileMap
	for _, fileMap := range cm.Files {
		if fileMap.Language == language {
			files = append(files, fileMap)
		}
	}
	return files
}

// GetTopLanguages returns the top N languages by file count
func (cm *CodebaseMap) GetTopLanguages(n int) []string {
	type langCount struct {
		lang  string
		count int
	}

	var counts []langCount
	for lang, count := range cm.Languages {
		counts = append(counts, langCount{lang, count})
	}

	// Simple bubble sort (good enough for small n)
	for i := 0; i < len(counts); i++ {
		for j := i + 1; j < len(counts); j++ {
			if counts[i].count < counts[j].count {
				counts[i], counts[j] = counts[j], counts[i]
			}
		}
	}

	var result []string
	for i := 0; i < n && i < len(counts); i++ {
		result = append(result, counts[i].lang)
	}

	return result
}

// TokenCounter counts tokens in source code
type TokenCounter struct{}

// NewTokenCounter creates a new token counter
func NewTokenCounter() *TokenCounter {
	return &TokenCounter{}
}

// Count counts tokens in source code
// This is a simplified implementation. For production, you'd use tiktoken or similar
func (tc *TokenCounter) Count(source []byte, language string) int {
	// Rough estimate: split by whitespace and common delimiters
	text := string(source)

	// Remove comments (simplified)
	text = tc.removeComments(text, language)

	// Split into tokens
	tokens := strings.FieldsFunc(text, func(r rune) bool {
		return unicode.IsSpace(r) ||
			r == '(' || r == ')' ||
			r == '{' || r == '}' ||
			r == '[' || r == ']' ||
			r == ';' || r == ',' ||
			r == '.' || r == ':'
	})

	return len(tokens)
}

// removeComments removes comments from source code
func (tc *TokenCounter) removeComments(text, language string) string {
	// This is a very simplified implementation
	// Production code would use the parser's comment extraction

	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		// Skip line comments
		if strings.Contains(line, "//") {
			line = strings.Split(line, "//")[0]
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// CountDefinition counts tokens in a definition
func (tc *TokenCounter) CountDefinition(def *Definition, source []byte) int {
	if def.StartByte >= 0 && def.EndByte <= len(source) && def.StartByte < def.EndByte {
		defSource := source[def.StartByte:def.EndByte]
		return tc.Count(defSource, "")
	}
	return 0
}

// ImportAnalyzer analyzes imports and dependencies
type ImportAnalyzer struct{}

// NewImportAnalyzer creates a new import analyzer
func NewImportAnalyzer() *ImportAnalyzer {
	return &ImportAnalyzer{}
}

// ResolveDependencies resolves file dependencies
func (ia *ImportAnalyzer) ResolveDependencies(fileMap *FileMap, cmap *CodebaseMap) []string {
	deps := make([]string, 0)

	for _, imp := range fileMap.Imports {
		// Try to resolve import to a file in the codebase
		resolved := ia.resolveImport(imp, fileMap.Path, cmap.Root)
		if resolved != "" && cmap.Files[resolved] != nil {
			deps = append(deps, resolved)
		}
	}

	return deps
}

// resolveImport resolves an import to a file path
func (ia *ImportAnalyzer) resolveImport(imp *Import, currentFile, root string) string {
	// This is a simplified implementation
	// Production code would handle language-specific import resolution

	if imp.IsRelative {
		// Relative import
		// For now, just return the path as-is
		return imp.Path
	}

	// Absolute import - would need language-specific resolution
	return ""
}

// BuildDependencyGraph builds a dependency graph
func (ia *ImportAnalyzer) BuildDependencyGraph(cmap *CodebaseMap) *DependencyGraph {
	graph := &DependencyGraph{
		Nodes: make(map[string]*DependencyNode),
		Edges: make(map[string][]string),
	}

	// Create nodes
	for path := range cmap.Files {
		graph.Nodes[path] = &DependencyNode{
			Path: path,
		}
	}

	// Create edges
	for path, deps := range cmap.Dependencies {
		graph.Edges[path] = deps
	}

	return graph
}

// DependencyGraph represents a dependency graph
type DependencyGraph struct {
	Nodes map[string]*DependencyNode `json:"nodes"`
	Edges map[string][]string        `json:"edges"`
}

// DependencyNode represents a node in the dependency graph
type DependencyNode struct {
	Path string `json:"path"`
}

// GetDependents returns files that depend on this file
func (dg *DependencyGraph) GetDependents(path string) []string {
	var dependents []string
	for node, deps := range dg.Edges {
		for _, dep := range deps {
			if dep == path {
				dependents = append(dependents, node)
				break
			}
		}
	}
	return dependents
}

// GetDependencies returns files that this file depends on
func (dg *DependencyGraph) GetDependencies(path string) []string {
	return dg.Edges[path]
}

// HasCycle checks if the dependency graph has cycles
func (dg *DependencyGraph) HasCycle() bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for node := range dg.Nodes {
		if dg.hasCycleDFS(node, visited, recStack) {
			return true
		}
	}

	return false
}

// hasCycleDFS performs DFS to detect cycles
func (dg *DependencyGraph) hasCycleDFS(node string, visited, recStack map[string]bool) bool {
	if recStack[node] {
		return true
	}

	if visited[node] {
		return false
	}

	visited[node] = true
	recStack[node] = true

	for _, dep := range dg.Edges[node] {
		if dg.hasCycleDFS(dep, visited, recStack) {
			return true
		}
	}

	recStack[node] = false
	return false
}

// CalculateChecksum calculates SHA-256 checksum of data
func CalculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// CountLines counts the number of lines in source code
func CountLines(source []byte) int {
	return bytes.Count(source, []byte{'\n'}) + 1
}

// ExtractRelativeIndentation extracts code with relative indentation
// This is useful for context generation (like Aider's repomap)
func ExtractRelativeIndentation(source []byte, startLine, endLine int) string {
	lines := bytes.Split(source, []byte{'\n'})

	if startLine < 0 || endLine >= len(lines) || startLine > endLine {
		return ""
	}

	// Find minimum indentation
	minIndent := -1
	for i := startLine; i <= endLine; i++ {
		line := lines[i]
		if len(bytes.TrimSpace(line)) == 0 {
			continue // Skip empty lines
		}

		indent := 0
		for _, b := range line {
			if b == ' ' || b == '\t' {
				indent++
			} else {
				break
			}
		}

		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent == -1 {
		minIndent = 0
	}

	// Extract lines with relative indentation
	var result []string
	for i := startLine; i <= endLine; i++ {
		line := lines[i]
		if len(bytes.TrimSpace(line)) == 0 {
			result = append(result, "")
			continue
		}

		// Remove minimum indentation
		if len(line) > minIndent {
			result = append(result, string(line[minIndent:]))
		} else {
			result = append(result, string(line))
		}
	}

	return strings.Join(result, "\n")
}
