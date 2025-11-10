package mapping

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

// LanguageInfo contains information about a supported language
type LanguageInfo struct {
	Name           string   `json:"name"`
	Extensions     []string `json:"extensions"`
	Aliases        []string `json:"aliases,omitempty"`
	TreeSitterName string   `json:"tree_sitter_name"`
}

// LanguageQueries contains tree-sitter queries for a language
type LanguageQueries struct {
	Functions string `json:"functions"`
	Classes   string `json:"classes"`
	Methods   string `json:"methods"`
	Imports   string `json:"imports"`
	Exports   string `json:"exports"`
	Comments  string `json:"comments"`
}

// LanguageParser defines the interface for language-specific parsers
type LanguageParser interface {
	// Parse parses source code into a ParsedTree
	Parse(source []byte) (*ParsedTree, error)

	// ExtractDefinitions extracts definitions from a parsed tree
	ExtractDefinitions(tree *ParsedTree) ([]*Definition, error)

	// ExtractImports extracts imports from a parsed tree
	ExtractImports(tree *ParsedTree) ([]*Import, error)

	// CalculateComplexity calculates cyclomatic complexity
	CalculateComplexity(tree *ParsedTree) int

	// GetQueries returns tree-sitter queries for this language
	GetQueries() *LanguageQueries
}

// LanguageRegistry manages language parsers
type LanguageRegistry interface {
	// Register registers a language parser
	Register(lang string, parser LanguageParser) error

	// Get gets a language parser by name
	Get(lang string) (LanguageParser, error)

	// GetByExtension gets a parser by file extension
	GetByExtension(ext string) (LanguageParser, error)

	// List lists all registered languages
	List() []string

	// GetLanguageInfo gets information about a language
	GetLanguageInfo(lang string) (*LanguageInfo, error)
}

// DefaultLanguageRegistry implements LanguageRegistry
type DefaultLanguageRegistry struct {
	parsers   map[string]LanguageParser
	languages map[string]*LanguageInfo
	mu        sync.RWMutex
}

// NewDefaultLanguageRegistry creates a new language registry
func NewDefaultLanguageRegistry() *DefaultLanguageRegistry {
	registry := &DefaultLanguageRegistry{
		parsers:   make(map[string]LanguageParser),
		languages: make(map[string]*LanguageInfo),
	}

	// Register default language information
	registry.registerLanguageInfo()

	return registry
}

// Register registers a language parser
func (r *DefaultLanguageRegistry) Register(lang string, parser LanguageParser) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	lang = strings.ToLower(lang)
	r.parsers[lang] = parser
	return nil
}

// Get gets a language parser by name
func (r *DefaultLanguageRegistry) Get(lang string) (LanguageParser, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	lang = strings.ToLower(lang)
	parser, ok := r.parsers[lang]
	if !ok {
		return nil, fmt.Errorf("language parser not found: %s", lang)
	}
	return parser, nil
}

// GetByExtension gets a parser by file extension
func (r *DefaultLanguageRegistry) GetByExtension(ext string) (LanguageParser, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ext = strings.ToLower(ext)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	// Find language by extension
	for langName, langInfo := range r.languages {
		for _, langExt := range langInfo.Extensions {
			if langExt == ext {
				return r.parsers[langName], nil
			}
		}
	}

	return nil, fmt.Errorf("no parser found for extension: %s", ext)
}

// List lists all registered languages
func (r *DefaultLanguageRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	languages := make([]string, 0, len(r.parsers))
	for lang := range r.parsers {
		languages = append(languages, lang)
	}
	return languages
}

// GetLanguageInfo gets information about a language
func (r *DefaultLanguageRegistry) GetLanguageInfo(lang string) (*LanguageInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	lang = strings.ToLower(lang)
	info, ok := r.languages[lang]
	if !ok {
		return nil, fmt.Errorf("language not found: %s", lang)
	}
	return info, nil
}

// registerLanguageInfo registers default language information
func (r *DefaultLanguageRegistry) registerLanguageInfo() {
	languages := []LanguageInfo{
		{
			Name:           "go",
			Extensions:     []string{".go"},
			TreeSitterName: "go",
		},
		{
			Name:           "javascript",
			Extensions:     []string{".js", ".jsx", ".mjs", ".cjs"},
			Aliases:        []string{"js", "node"},
			TreeSitterName: "javascript",
		},
		{
			Name:           "typescript",
			Extensions:     []string{".ts", ".tsx"},
			Aliases:        []string{"ts"},
			TreeSitterName: "typescript",
		},
		{
			Name:           "python",
			Extensions:     []string{".py", ".pyw"},
			Aliases:        []string{"py"},
			TreeSitterName: "python",
		},
		{
			Name:           "rust",
			Extensions:     []string{".rs"},
			TreeSitterName: "rust",
		},
		{
			Name:           "java",
			Extensions:     []string{".java"},
			TreeSitterName: "java",
		},
		{
			Name:           "c",
			Extensions:     []string{".c", ".h"},
			TreeSitterName: "c",
		},
		{
			Name:           "cpp",
			Extensions:     []string{".cpp", ".cxx", ".cc", ".hpp", ".hxx", ".hh"},
			Aliases:        []string{"c++", "cplusplus"},
			TreeSitterName: "cpp",
		},
		{
			Name:           "csharp",
			Extensions:     []string{".cs"},
			Aliases:        []string{"cs", "c#"},
			TreeSitterName: "c_sharp",
		},
		{
			Name:           "ruby",
			Extensions:     []string{".rb"},
			Aliases:        []string{"rb"},
			TreeSitterName: "ruby",
		},
		{
			Name:           "php",
			Extensions:     []string{".php", ".phtml"},
			TreeSitterName: "php",
		},
		{
			Name:           "swift",
			Extensions:     []string{".swift"},
			TreeSitterName: "swift",
		},
		{
			Name:           "kotlin",
			Extensions:     []string{".kt", ".kts"},
			TreeSitterName: "kotlin",
		},
		{
			Name:           "scala",
			Extensions:     []string{".scala", ".sc"},
			TreeSitterName: "scala",
		},
		{
			Name:           "elixir",
			Extensions:     []string{".ex", ".exs"},
			TreeSitterName: "elixir",
		},
		{
			Name:           "haskell",
			Extensions:     []string{".hs", ".lhs"},
			TreeSitterName: "haskell",
		},
		{
			Name:           "ocaml",
			Extensions:     []string{".ml", ".mli"},
			TreeSitterName: "ocaml",
		},
		{
			Name:           "lua",
			Extensions:     []string{".lua"},
			TreeSitterName: "lua",
		},
		{
			Name:           "perl",
			Extensions:     []string{".pl", ".pm"},
			TreeSitterName: "perl",
		},
		{
			Name:           "r",
			Extensions:     []string{".r", ".R"},
			TreeSitterName: "r",
		},
		{
			Name:           "julia",
			Extensions:     []string{".jl"},
			TreeSitterName: "julia",
		},
		{
			Name:           "dart",
			Extensions:     []string{".dart"},
			TreeSitterName: "dart",
		},
		{
			Name:           "zig",
			Extensions:     []string{".zig"},
			TreeSitterName: "zig",
		},
		{
			Name:           "nim",
			Extensions:     []string{".nim"},
			TreeSitterName: "nim",
		},
		{
			Name:           "crystal",
			Extensions:     []string{".cr"},
			TreeSitterName: "crystal",
		},
		{
			Name:           "fsharp",
			Extensions:     []string{".fs", ".fsx"},
			Aliases:        []string{"f#"},
			TreeSitterName: "fsharp",
		},
		{
			Name:           "clojure",
			Extensions:     []string{".clj", ".cljs", ".cljc"},
			TreeSitterName: "clojure",
		},
		{
			Name:           "erlang",
			Extensions:     []string{".erl", ".hrl"},
			TreeSitterName: "erlang",
		},
		{
			Name:           "elm",
			Extensions:     []string{".elm"},
			TreeSitterName: "elm",
		},
		{
			Name:           "solidity",
			Extensions:     []string{".sol"},
			TreeSitterName: "solidity",
		},
	}

	for _, lang := range languages {
		r.languages[lang.Name] = &lang
		// Also register by aliases
		for _, alias := range lang.Aliases {
			r.languages[alias] = &lang
		}
	}
}

// DetectLanguage detects the language from a file path
func DetectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return ""
	}

	// Map extensions to language names
	extMap := map[string]string{
		".go":    "go",
		".js":    "javascript",
		".jsx":   "javascript",
		".mjs":   "javascript",
		".cjs":   "javascript",
		".ts":    "typescript",
		".tsx":   "typescript",
		".py":    "python",
		".pyw":   "python",
		".rs":    "rust",
		".java":  "java",
		".c":     "c",
		".h":     "c",
		".cpp":   "cpp",
		".cxx":   "cpp",
		".cc":    "cpp",
		".hpp":   "cpp",
		".hxx":   "cpp",
		".hh":    "cpp",
		".cs":    "csharp",
		".rb":    "ruby",
		".php":   "php",
		".phtml": "php",
		".swift": "swift",
		".kt":    "kotlin",
		".kts":   "kotlin",
		".scala": "scala",
		".sc":    "scala",
		".ex":    "elixir",
		".exs":   "elixir",
		".hs":    "haskell",
		".lhs":   "haskell",
		".ml":    "ocaml",
		".mli":   "ocaml",
		".lua":   "lua",
		".pl":    "perl",
		".pm":    "perl",
		".r":     "r",
		".R":     "r",
		".jl":    "julia",
		".dart":  "dart",
		".zig":   "zig",
		".nim":   "nim",
		".cr":    "crystal",
		".fs":    "fsharp",
		".fsx":   "fsharp",
		".clj":   "clojure",
		".cljs":  "clojure",
		".cljc":  "clojure",
		".erl":   "erlang",
		".hrl":   "erlang",
		".elm":   "elm",
		".sol":   "solidity",
	}

	return extMap[ext]
}

// IsSupported checks if a language is supported
func IsSupported(language string) bool {
	language = strings.ToLower(language)
	supported := []string{
		"go", "javascript", "typescript", "python", "rust",
		"java", "c", "cpp", "csharp", "ruby", "php",
		"swift", "kotlin", "scala", "elixir", "haskell",
		"ocaml", "lua", "perl", "r", "julia", "dart",
		"zig", "nim", "crystal", "fsharp", "clojure",
		"erlang", "elm", "solidity",
	}

	for _, lang := range supported {
		if lang == language {
			return true
		}
	}
	return false
}
