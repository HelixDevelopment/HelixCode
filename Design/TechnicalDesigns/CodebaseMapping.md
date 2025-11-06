# CodebaseMapping - Technical Design Document

## Overview

The CodebaseMapping module provides comprehensive codebase analysis using tree-sitter parsers, enabling semantic understanding of code structure across 30+ programming languages. This design is inspired by Aider's repomap.py and Plandex's tree-sitter integration, with enhancements for caching, token counting, and incremental updates.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      CodebaseMapping                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │    Mapper    │  │ TreeSitter   │  │   Language   │         │
│  │              │──│   Parser     │──│   Registry   │         │
│  └──────┬───────┘  └──────┬───────┘  └──────────────┘         │
│         │                  │                                    │
│  ┌──────┴──────────────────┴──────────┐                        │
│  │         CacheManager               │                        │
│  │  (.helix.cache/ with versioning)   │                        │
│  └──────┬──────────────┬──────────────┘                        │
│         │              │                                        │
│  ┌──────┴─────┐  ┌─────┴──────┐                               │
│  │   Token    │  │  Import    │                               │
│  │  Counter   │  │  Analyzer  │                               │
│  └────────────┘  └────────────┘                               │
│                                                                 │
│  ┌─────────────────────────────────────────────────┐          │
│  │         Definition Extractor                    │          │
│  │  (Functions, Classes, Methods, Interfaces)     │          │
│  └─────────────────────────────────────────────────┘          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
                           │
                           ▼
                   Tree-sitter Parsers
                   (30+ languages)
```

## Core Interfaces

### Mapper Interface

```go
// Mapper generates codebase maps
type Mapper interface {
    // MapCodebase maps an entire codebase
    MapCodebase(ctx context.Context, root string, opts *MapOptions) (*CodebaseMap, error)

    // MapFile maps a single file
    MapFile(ctx context.Context, path string) (*FileMap, error)

    // MapFiles maps multiple files
    MapFiles(ctx context.Context, paths []string) ([]*FileMap, error)

    // UpdateMap updates a codebase map incrementally
    UpdateMap(ctx context.Context, cmap *CodebaseMap, changes []string) error

    // GetDefinitions extracts definitions from code
    GetDefinitions(ctx context.Context, path string) ([]*Definition, error)

    // GetReferences finds references to a definition
    GetReferences(ctx context.Context, def *Definition, scope *CodebaseMap) ([]*Reference, error)
}

// CodebaseMap represents a complete codebase map
type CodebaseMap struct {
    Root         string
    Files        map[string]*FileMap
    Languages    map[string]int // Language -> file count
    TotalFiles   int
    TotalLines   int
    TotalTokens  int
    Definitions  map[string]*Definition // Qualified name -> Definition
    Dependencies map[string][]string    // File -> dependencies
    CreatedAt    time.Time
    UpdatedAt    time.Time
    Version      string
}

// FileMap represents a single file's map
type FileMap struct {
    Path           string
    Language       string
    Size           int64
    Lines          int
    Tokens         int
    Definitions    []*Definition
    Imports        []*Import
    Exports        []*Export
    Comments       []*Comment
    Complexity     int
    Checksum       string
    ParsedAt       time.Time
}

// Definition represents a code definition
type Definition struct {
    Type          DefinitionType
    Name          string
    QualifiedName string
    FilePath      string
    StartLine     int
    EndLine       int
    StartByte     int
    EndByte       int
    Signature     string
    DocComment    string
    Visibility    Visibility
    Parameters    []*Parameter
    ReturnType    string
    Parent        string // Parent class/namespace
    Children      []string // Nested definitions
    Metadata      map[string]interface{}
}

// DefinitionType represents the type of definition
type DefinitionType int

const (
    DefFunction DefinitionType = iota
    DefMethod
    DefClass
    DefInterface
    DefStruct
    DefEnum
    DefType
    DefVariable
    DefConstant
    DefModule
    DefNamespace
)

func (d DefinitionType) String() string {
    return [...]string{
        "Function", "Method", "Class", "Interface", "Struct",
        "Enum", "Type", "Variable", "Constant", "Module", "Namespace",
    }[d]
}

// Visibility represents code visibility
type Visibility int

const (
    VisibilityPublic Visibility = iota
    VisibilityPrivate
    VisibilityProtected
    VisibilityInternal
)

// Parameter represents a function parameter
type Parameter struct {
    Name    string
    Type    string
    Default string
}

// Import represents an import statement
type Import struct {
    Path       string
    Alias      string
    Items      []string
    IsRelative bool
    StartLine  int
}

// Export represents an export statement
type Export struct {
    Name      string
    Type      string
    IsDefault bool
    StartLine int
}

// Comment represents a comment
type Comment struct {
    Text      string
    StartLine int
    EndLine   int
    IsDoc     bool
}

// Reference represents a reference to a definition
type Reference struct {
    DefinitionID  string
    FilePath      string
    Line          int
    Column        int
    Context       string
}
```

### TreeSitterParser Interface

```go
// TreeSitterParser parses code using tree-sitter
type TreeSitterParser interface {
    // Parse parses source code
    Parse(ctx context.Context, source []byte, language string) (*ParsedTree, error)

    // ParseFile parses a file
    ParseFile(ctx context.Context, path string) (*ParsedTree, error)

    // GetSupportedLanguages returns supported languages
    GetSupportedLanguages() []string

    // IsSupported checks if a language is supported
    IsSupported(language string) bool
}

// ParsedTree represents a parsed syntax tree
type ParsedTree struct {
    Language    string
    Root        *Node
    Source      []byte
    ParseErrors []*ParseError
}

// Node represents a syntax tree node
type Node struct {
    Type       string
    Text       string
    StartByte  int
    EndByte    int
    StartPoint Point
    EndPoint   Point
    Children   []*Node
    Parent     *Node
}

// Point represents a position in source code
type Point struct {
    Row    int
    Column int
}

// ParseError represents a parsing error
type ParseError struct {
    Message   string
    StartByte int
    EndByte   int
    StartLine int
    EndLine   int
}
```

### LanguageRegistry Interface

```go
// LanguageRegistry manages language parsers
type LanguageRegistry interface {
    // Register registers a language parser
    Register(lang string, parser LanguageParser) error

    // Get gets a language parser
    Get(lang string) (LanguageParser, error)

    // GetByExtension gets a parser by file extension
    GetByExtension(ext string) (LanguageParser, error)

    // List lists all registered languages
    List() []string
}

// LanguageParser parses code for a specific language
type LanguageParser interface {
    // Parse parses source code
    Parse(source []byte) (*ParsedTree, error)

    // ExtractDefinitions extracts definitions from a tree
    ExtractDefinitions(tree *ParsedTree) ([]*Definition, error)

    // ExtractImports extracts imports from a tree
    ExtractImports(tree *ParsedTree) ([]*Import, error)

    // CalculateComplexity calculates code complexity
    CalculateComplexity(tree *ParsedTree) int

    // GetQueries returns tree-sitter queries for this language
    GetQueries() *LanguageQueries
}

// LanguageQueries contains tree-sitter queries for a language
type LanguageQueries struct {
    Functions   string
    Classes     string
    Methods     string
    Imports     string
    Exports     string
    Comments    string
}
```

### CacheManager Interface

```go
// CacheManager manages codebase map cache
type CacheManager interface {
    // Load loads a cached map
    Load(root string) (*CodebaseMap, error)

    // Save saves a map to cache
    Save(cmap *CodebaseMap) error

    // Invalidate invalidates cache for specific files
    Invalidate(files []string) error

    // Clear clears all cache
    Clear() error

    // GetCacheDir returns the cache directory
    GetCacheDir() string

    // GetCacheStats returns cache statistics
    GetCacheStats() (*CacheStats, error)
}

// CacheStats contains cache statistics
type CacheStats struct {
    TotalFiles      int
    TotalSize       int64
    HitRate         float64
    LastUpdated     time.Time
    Version         string
}
```

## Implementation

### Mapper Implementation

```go
// DefaultMapper implements Mapper
type DefaultMapper struct {
    parser       TreeSitterParser
    registry     LanguageRegistry
    cache        CacheManager
    tokenCounter *TokenCounter
    importAnalyzer *ImportAnalyzer
}

// NewDefaultMapper creates a new mapper
func NewDefaultMapper(
    parser TreeSitterParser,
    registry LanguageRegistry,
    cache CacheManager,
) *DefaultMapper {
    return &DefaultMapper{
        parser:         parser,
        registry:       registry,
        cache:          cache,
        tokenCounter:   NewTokenCounter(),
        importAnalyzer: NewImportAnalyzer(),
    }
}

// MapCodebase maps an entire codebase
func (m *DefaultMapper) MapCodebase(ctx context.Context, root string, opts *MapOptions) (*CodebaseMap, error) {
    if opts == nil {
        opts = DefaultMapOptions()
    }

    // Try to load from cache
    if opts.UseCache {
        if cached, err := m.cache.Load(root); err == nil {
            // Check if cache is still valid
            if m.isCacheValid(cached, root) {
                return cached, nil
            }
        }
    }

    // Create new map
    cmap := &CodebaseMap{
        Root:         root,
        Files:        make(map[string]*FileMap),
        Languages:    make(map[string]int),
        Definitions:  make(map[string]*Definition),
        Dependencies: make(map[string][]string),
        CreatedAt:    time.Now(),
        Version:      "1.0.0",
    }

    // Find all source files
    files, err := m.findSourceFiles(root, opts)
    if err != nil {
        return nil, fmt.Errorf("failed to find source files: %w", err)
    }

    // Map files concurrently
    fileMaps := make(chan *FileMap, len(files))
    errors := make(chan error, len(files))

    var wg sync.WaitGroup
    semaphore := make(chan struct{}, opts.Concurrency)

    for _, file := range files {
        wg.Add(1)
        go func(path string) {
            defer wg.Done()

            semaphore <- struct{}{}
            defer func() { <-semaphore }()

            fileMap, err := m.MapFile(ctx, path)
            if err != nil {
                errors <- err
                return
            }

            fileMaps <- fileMap
        }(file)
    }

    go func() {
        wg.Wait()
        close(fileMaps)
        close(errors)
    }()

    // Collect results
    for fileMap := range fileMaps {
        cmap.Files[fileMap.Path] = fileMap
        cmap.TotalFiles++
        cmap.TotalLines += fileMap.Lines
        cmap.TotalTokens += fileMap.Tokens
        cmap.Languages[fileMap.Language]++

        // Add definitions
        for _, def := range fileMap.Definitions {
            cmap.Definitions[def.QualifiedName] = def
        }

        // Add dependencies
        deps := m.importAnalyzer.ResolveDependencies(fileMap, cmap)
        cmap.Dependencies[fileMap.Path] = deps
    }

    // Check for errors
    for err := range errors {
        if err != nil {
            return nil, err
        }
    }

    cmap.UpdatedAt = time.Now()

    // Save to cache
    if opts.UseCache {
        if err := m.cache.Save(cmap); err != nil {
            // Log warning but don't fail
            log.Printf("Warning: failed to save cache: %v", err)
        }
    }

    return cmap, nil
}

// MapFile maps a single file
func (m *DefaultMapper) MapFile(ctx context.Context, path string) (*FileMap, error) {
    // Read file
    source, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read file: %w", err)
    }

    // Detect language
    language := m.detectLanguage(path)
    if language == "" {
        return nil, fmt.Errorf("unsupported language for file: %s", path)
    }

    // Parse file
    tree, err := m.parser.Parse(ctx, source, language)
    if err != nil {
        return nil, fmt.Errorf("failed to parse file: %w", err)
    }

    // Get language parser
    langParser, err := m.registry.Get(language)
    if err != nil {
        return nil, fmt.Errorf("failed to get language parser: %w", err)
    }

    // Extract definitions
    definitions, err := langParser.ExtractDefinitions(tree)
    if err != nil {
        return nil, fmt.Errorf("failed to extract definitions: %w", err)
    }

    // Extract imports
    imports, err := langParser.ExtractImports(tree)
    if err != nil {
        return nil, fmt.Errorf("failed to extract imports: %w", err)
    }

    // Calculate complexity
    complexity := langParser.CalculateComplexity(tree)

    // Count tokens
    tokens := m.tokenCounter.Count(source, language)

    // Create file map
    fileMap := &FileMap{
        Path:        path,
        Language:    language,
        Size:        int64(len(source)),
        Lines:       countLines(source),
        Tokens:      tokens,
        Definitions: definitions,
        Imports:     imports,
        Complexity:  complexity,
        Checksum:    calculateChecksum(source),
        ParsedAt:    time.Now(),
    }

    return fileMap, nil
}

// GetDefinitions extracts definitions from code
func (m *DefaultMapper) GetDefinitions(ctx context.Context, path string) ([]*Definition, error) {
    fileMap, err := m.MapFile(ctx, path)
    if err != nil {
        return nil, err
    }
    return fileMap.Definitions, nil
}

// findSourceFiles finds all source files in a directory
func (m *DefaultMapper) findSourceFiles(root string, opts *MapOptions) ([]string, error) {
    var files []string

    err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        // Skip directories
        if info.IsDir() {
            // Check if directory should be excluded
            if m.shouldExcludeDir(path, opts.ExcludeDirs) {
                return filepath.SkipDir
            }
            return nil
        }

        // Check if file should be included
        if m.shouldIncludeFile(path, opts) {
            files = append(files, path)
        }

        return nil
    })

    return files, err
}

// detectLanguage detects the language of a file
func (m *DefaultMapper) detectLanguage(path string) string {
    ext := filepath.Ext(path)
    if ext == "" {
        return ""
    }

    // Map extensions to languages
    langMap := map[string]string{
        ".go":    "go",
        ".js":    "javascript",
        ".ts":    "typescript",
        ".py":    "python",
        ".rs":    "rust",
        ".java":  "java",
        ".c":     "c",
        ".cpp":   "cpp",
        ".h":     "c",
        ".hpp":   "cpp",
        ".rb":    "ruby",
        ".php":   "php",
        ".swift": "swift",
        ".kt":    "kotlin",
        ".scala": "scala",
        ".cs":    "csharp",
    }

    return langMap[ext]
}

// shouldExcludeDir checks if a directory should be excluded
func (m *DefaultMapper) shouldExcludeDir(path string, excludeDirs []string) bool {
    basename := filepath.Base(path)

    for _, exclude := range excludeDirs {
        if basename == exclude {
            return true
        }
    }

    return false
}

// shouldIncludeFile checks if a file should be included
func (m *DefaultMapper) shouldIncludeFile(path string, opts *MapOptions) bool {
    // Check if language is supported
    lang := m.detectLanguage(path)
    if lang == "" {
        return false
    }

    // Check file size
    info, err := os.Stat(path)
    if err != nil || info.Size() > opts.MaxFileSize {
        return false
    }

    return true
}

// isCacheValid checks if cached map is still valid
func (m *DefaultMapper) isCacheValid(cached *CodebaseMap, root string) bool {
    // Check if any files have been modified
    for path, fileMap := range cached.Files {
        info, err := os.Stat(path)
        if err != nil {
            return false
        }

        // Check if file has been modified
        currentChecksum := calculateChecksum([]byte{}) // Simplified
        if currentChecksum != fileMap.Checksum {
            return false
        }
    }

    return true
}

// countLines counts lines in source code
func countLines(source []byte) int {
    return bytes.Count(source, []byte{'\n'}) + 1
}

// calculateChecksum calculates SHA-256 checksum
func calculateChecksum(data []byte) string {
    hash := sha256.Sum256(data)
    return fmt.Sprintf("%x", hash)
}
```

### Language Parser Example (Go)

```go
// GoParser implements LanguageParser for Go
type GoParser struct {
    language *sitter.Language
}

// NewGoParser creates a new Go parser
func NewGoParser() *GoParser {
    return &GoParser{
        language: golang.GetLanguage(),
    }
}

// Parse parses Go source code
func (p *GoParser) Parse(source []byte) (*ParsedTree, error) {
    parser := sitter.NewParser()
    parser.SetLanguage(p.language)

    tree := parser.Parse(nil, source)
    if tree == nil {
        return nil, fmt.Errorf("failed to parse Go code")
    }

    return &ParsedTree{
        Language: "go",
        Root:     convertNode(tree.RootNode()),
        Source:   source,
    }, nil
}

// ExtractDefinitions extracts definitions from Go code
func (p *GoParser) ExtractDefinitions(tree *ParsedTree) ([]*Definition, error) {
    var definitions []*Definition

    // Query for functions
    funcQuery := `
    (function_declaration
        name: (identifier) @func.name
        parameters: (parameter_list) @func.params
        result: (_)? @func.result) @func.def
    `

    matches := p.query(tree, funcQuery)
    for _, match := range matches {
        def := &Definition{
            Type:      DefFunction,
            Name:      match["func.name"],
            StartLine: match["func.def.start_line"],
            EndLine:   match["func.def.end_line"],
        }
        definitions = append(definitions, def)
    }

    // Query for methods
    methodQuery := `
    (method_declaration
        receiver: (parameter_list) @method.receiver
        name: (identifier) @method.name
        parameters: (parameter_list) @method.params
        result: (_)? @method.result) @method.def
    `

    matches = p.query(tree, methodQuery)
    for _, match := range matches {
        def := &Definition{
            Type:      DefMethod,
            Name:      match["method.name"],
            StartLine: match["method.def.start_line"],
            EndLine:   match["method.def.end_line"],
        }
        definitions = append(definitions, def)
    }

    // Query for types
    typeQuery := `
    (type_declaration
        (type_spec
            name: (type_identifier) @type.name
            type: (_) @type.type)) @type.def
    `

    matches = p.query(tree, typeQuery)
    for _, match := range matches {
        def := &Definition{
            Type:      DefType,
            Name:      match["type.name"],
            StartLine: match["type.def.start_line"],
            EndLine:   match["type.def.end_line"],
        }
        definitions = append(definitions, def)
    }

    return definitions, nil
}

// ExtractImports extracts imports from Go code
func (p *GoParser) ExtractImports(tree *ParsedTree) ([]*Import, error) {
    var imports []*Import

    query := `
    (import_declaration
        (import_spec
            path: (interpreted_string_literal) @import.path
            name: (package_identifier)? @import.name)) @import.def
    `

    matches := p.query(tree, query)
    for _, match := range matches {
        imp := &Import{
            Path:      strings.Trim(match["import.path"], `"`),
            Alias:     match["import.name"],
            StartLine: match["import.def.start_line"],
        }
        imports = append(imports, imp)
    }

    return imports, nil
}

// CalculateComplexity calculates cyclomatic complexity
func (p *GoParser) CalculateComplexity(tree *ParsedTree) int {
    complexity := 1 // Base complexity

    // Count decision points
    query := `
    [
        (if_statement)
        (for_statement)
        (switch_statement)
        (case_clause)
        (||)
        (&&)
    ] @decision
    `

    matches := p.query(tree, query)
    complexity += len(matches)

    return complexity
}

// GetQueries returns tree-sitter queries
func (p *GoParser) GetQueries() *LanguageQueries {
    return &LanguageQueries{
        Functions: `(function_declaration) @function`,
        Classes:   ``, // Go doesn't have classes
        Methods:   `(method_declaration) @method`,
        Imports:   `(import_declaration) @import`,
        Comments:  `(comment) @comment`,
    }
}

// query executes a tree-sitter query
func (p *GoParser) query(tree *ParsedTree, queryStr string) []map[string]interface{} {
    // This is a simplified implementation
    // Real implementation would use tree-sitter query API
    return nil
}

// convertNode converts tree-sitter node to our Node type
func convertNode(node *sitter.Node) *Node {
    if node == nil {
        return nil
    }

    converted := &Node{
        Type:      node.Type(),
        StartByte: int(node.StartByte()),
        EndByte:   int(node.EndByte()),
        StartPoint: Point{
            Row:    int(node.StartPoint().Row),
            Column: int(node.StartPoint().Column),
        },
        EndPoint: Point{
            Row:    int(node.EndPoint().Row),
            Column: int(node.EndPoint().Column),
        },
    }

    // Convert children
    childCount := int(node.ChildCount())
    converted.Children = make([]*Node, childCount)
    for i := 0; i < childCount; i++ {
        child := convertNode(node.Child(i))
        if child != nil {
            child.Parent = converted
            converted.Children[i] = child
        }
    }

    return converted
}
```

### Cache Manager Implementation

```go
// DiskCacheManager implements CacheManager
type DiskCacheManager struct {
    cacheDir string
    mu       sync.RWMutex
}

// NewDiskCacheManager creates a new disk cache manager
func NewDiskCacheManager(workspaceRoot string) *DiskCacheManager {
    cacheDir := filepath.Join(workspaceRoot, ".helix.cache")
    os.MkdirAll(cacheDir, 0755)

    return &DiskCacheManager{
        cacheDir: cacheDir,
    }
}

// Load loads a cached map
func (cm *DiskCacheManager) Load(root string) (*CodebaseMap, error) {
    cm.mu.RLock()
    defer cm.mu.RUnlock()

    cachePath := cm.getCachePath(root)

    data, err := os.ReadFile(cachePath)
    if err != nil {
        return nil, err
    }

    var cmap CodebaseMap
    if err := json.Unmarshal(data, &cmap); err != nil {
        return nil, fmt.Errorf("failed to unmarshal cache: %w", err)
    }

    return &cmap, nil
}

// Save saves a map to cache
func (cm *DiskCacheManager) Save(cmap *CodebaseMap) error {
    cm.mu.Lock()
    defer cm.mu.Unlock()

    cachePath := cm.getCachePath(cmap.Root)

    data, err := json.MarshalIndent(cmap, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal cache: %w", err)
    }

    // Write to temp file first, then rename (atomic)
    tmpPath := cachePath + ".tmp"
    if err := os.WriteFile(tmpPath, data, 0644); err != nil {
        return fmt.Errorf("failed to write cache: %w", err)
    }

    if err := os.Rename(tmpPath, cachePath); err != nil {
        return fmt.Errorf("failed to rename cache: %w", err)
    }

    return nil
}

// Invalidate invalidates cache for specific files
func (cm *DiskCacheManager) Invalidate(files []string) error {
    // For now, just clear the entire cache
    // A more sophisticated implementation would update the cache incrementally
    return cm.Clear()
}

// Clear clears all cache
func (cm *DiskCacheManager) Clear() error {
    cm.mu.Lock()
    defer cm.mu.Unlock()

    return os.RemoveAll(cm.cacheDir)
}

// GetCacheDir returns the cache directory
func (cm *DiskCacheManager) GetCacheDir() string {
    return cm.cacheDir
}

// GetCacheStats returns cache statistics
func (cm *DiskCacheManager) GetCacheStats() (*CacheStats, error) {
    cm.mu.RLock()
    defer cm.mu.RUnlock()

    var totalFiles int
    var totalSize int64

    err := filepath.Walk(cm.cacheDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if !info.IsDir() {
            totalFiles++
            totalSize += info.Size()
        }
        return nil
    })

    if err != nil {
        return nil, err
    }

    return &CacheStats{
        TotalFiles:  totalFiles,
        TotalSize:   totalSize,
        LastUpdated: time.Now(),
        Version:     "1.0.0",
    }, nil
}

// getCachePath returns the cache file path for a root
func (cm *DiskCacheManager) getCachePath(root string) string {
    // Use hash of root path as cache filename
    hash := sha256.Sum256([]byte(root))
    filename := fmt.Sprintf("%x.json", hash)
    return filepath.Join(cm.cacheDir, filename)
}
```

### Token Counter

```go
// TokenCounter counts tokens in source code
type TokenCounter struct {
    // Use tiktoken or similar for accurate token counting
}

// NewTokenCounter creates a new token counter
func NewTokenCounter() *TokenCounter {
    return &TokenCounter{}
}

// Count counts tokens in source code
func (tc *TokenCounter) Count(source []byte, language string) int {
    // Simplified implementation
    // Real implementation would use tiktoken or language-specific tokenizers

    // Rough estimate: split by whitespace and common delimiters
    text := string(source)
    tokens := strings.FieldsFunc(text, func(r rune) bool {
        return unicode.IsSpace(r) || r == '(' || r == ')' || r == '{' || r == '}' || r == '[' || r == ']'
    })

    return len(tokens)
}

// CountDefinition counts tokens in a definition
func (tc *TokenCounter) CountDefinition(def *Definition, source []byte) int {
    // Extract definition source
    defSource := source[def.StartByte:def.EndByte]
    return tc.Count(defSource, "")
}
```

### Import Analyzer

```go
// ImportAnalyzer analyzes imports and dependencies
type ImportAnalyzer struct{}

// NewImportAnalyzer creates a new import analyzer
func NewImportAnalyzer() *ImportAnalyzer {
    return &ImportAnalyzer{}
}

// ResolveDependencies resolves file dependencies
func (ia *ImportAnalyzer) ResolveDependencies(fileMap *FileMap, cmap *CodebaseMap) []string {
    var deps []string

    for _, imp := range fileMap.Imports {
        // Resolve import path to actual file
        resolved := ia.resolveImport(imp, fileMap.Path, cmap.Root)
        if resolved != "" {
            deps = append(deps, resolved)
        }
    }

    return deps
}

// resolveImport resolves an import to a file path
func (ia *ImportAnalyzer) resolveImport(imp *Import, currentFile, root string) string {
    if imp.IsRelative {
        // Resolve relative import
        dir := filepath.Dir(currentFile)
        resolved := filepath.Join(dir, imp.Path)
        return resolved
    }

    // For absolute imports, would need to check module systems
    // This is language-specific
    return ""
}

// BuildDependencyGraph builds a dependency graph
func (ia *ImportAnalyzer) BuildDependencyGraph(cmap *CodebaseMap) *DependencyGraph {
    graph := &DependencyGraph{
        Nodes: make(map[string]*DependencyNode),
        Edges: make(map[string][]string),
    }

    for path := range cmap.Files {
        graph.Nodes[path] = &DependencyNode{
            Path: path,
        }
    }

    for path, deps := range cmap.Dependencies {
        graph.Edges[path] = deps
    }

    return graph
}

// DependencyGraph represents a dependency graph
type DependencyGraph struct {
    Nodes map[string]*DependencyNode
    Edges map[string][]string
}

// DependencyNode represents a node in the dependency graph
type DependencyNode struct {
    Path string
}
```

## Testing Strategy

### Unit Tests

```go
// TestFileMapping tests file mapping
func TestFileMapping(t *testing.T) {
    mapper := setupTestMapper(t)

    t.Run("map go file", func(t *testing.T) {
        tmpfile := createTestFile(t, "test.go", `
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}

type User struct {
    Name string
    Age  int
}
`)
        defer os.Remove(tmpfile)

        fileMap, err := mapper.MapFile(context.Background(), tmpfile)
        require.NoError(t, err)

        assert.Equal(t, "go", fileMap.Language)
        assert.Greater(t, len(fileMap.Definitions), 0)
        assert.Greater(t, len(fileMap.Imports), 0)
    })
}

// TestDefinitionExtraction tests definition extraction
func TestDefinitionExtraction(t *testing.T) {
    parser := NewGoParser()

    source := []byte(`
package main

func Add(a, b int) int {
    return a + b
}

type Calculator struct {
    result int
}

func (c *Calculator) Multiply(a, b int) int {
    return a * b
}
`)

    tree, err := parser.Parse(source)
    require.NoError(t, err)

    definitions, err := parser.ExtractDefinitions(tree)
    require.NoError(t, err)

    assert.GreaterOrEqual(t, len(definitions), 2) // At least function and method

    // Check function
    funcDef := findDefinition(definitions, "Add")
    require.NotNil(t, funcDef)
    assert.Equal(t, DefFunction, funcDef.Type)

    // Check method
    methodDef := findDefinition(definitions, "Multiply")
    require.NotNil(t, methodDef)
    assert.Equal(t, DefMethod, methodDef.Type)
}

// TestCaching tests cache functionality
func TestCaching(t *testing.T) {
    tmpDir, err := os.MkdirTemp("", "cache-test-*")
    require.NoError(t, err)
    defer os.RemoveAll(tmpDir)

    cache := NewDiskCacheManager(tmpDir)

    cmap := &CodebaseMap{
        Root:       tmpDir,
        TotalFiles: 10,
        Version:    "1.0.0",
    }

    // Save to cache
    err = cache.Save(cmap)
    require.NoError(t, err)

    // Load from cache
    loaded, err := cache.Load(tmpDir)
    require.NoError(t, err)
    assert.Equal(t, cmap.TotalFiles, loaded.TotalFiles)
}
```

## Configuration

```go
// MapOptions configures codebase mapping
type MapOptions struct {
    UseCache      bool
    Concurrency   int
    MaxFileSize   int64
    ExcludeDirs   []string
    IncludeHidden bool
    Languages     []string // Filter by languages
}

// DefaultMapOptions returns default options
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
        },
        IncludeHidden: false,
    }
}
```

## Supported Languages

The following languages are supported through tree-sitter:

1. Go
2. JavaScript/TypeScript
3. Python
4. Rust
5. Java
6. C/C++
7. C#
8. Ruby
9. PHP
10. Swift
11. Kotlin
12. Scala
13. Elixir
14. Haskell
15. OCaml
16. Lua
17. Perl
18. R
19. Julia
20. Dart
21. Zig
22. Nim
23. Crystal
24. F#
25. Clojure
26. Erlang
27. Elm
28. PureScript
29. ReasonML
30. Solidity

## References

### Aider's repomap.py

- **Location**: `aider/repomap.py`
- **Features**:
  - Tree-sitter based parsing
  - Token counting for context management
  - Relative indentation preservation
  - Smart caching

### Plandex's Tree-sitter Integration

- **Features**:
  - Multi-language support
  - Definition extraction
  - Dependency analysis

### Libraries

- `github.com/smacker/go-tree-sitter` - Tree-sitter bindings for Go
- Language parsers: `tree-sitter-go`, `tree-sitter-javascript`, etc.

## Future Enhancements

1. **Semantic Search**: Search by semantic meaning, not just text
2. **Call Graph**: Build complete call graphs
3. **Type Inference**: Infer types for dynamically typed languages
4. **Cross-language Support**: Handle polyglot codebases
5. **Documentation Generation**: Auto-generate documentation from code
6. **Code Metrics**: Advanced metrics (maintainability index, etc.)
7. **Incremental Parsing**: Update only changed files
8. **LSP Integration**: Integrate with Language Server Protocol
9. **Symbol Resolution**: Resolve symbols across files
10. **Refactoring Support**: Safe refactoring operations
