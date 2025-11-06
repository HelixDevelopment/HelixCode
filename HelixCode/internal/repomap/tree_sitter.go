package repomap

import (
	"context"
	"fmt"
	"os"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/ruby"
)

// TreeSitterParser handles parsing of source code using Tree-sitter
type TreeSitterParser struct {
	languages map[string]*sitter.Language
}

// NewTreeSitterParser creates a new Tree-sitter parser
func NewTreeSitterParser() *TreeSitterParser {
	return &TreeSitterParser{
		languages: map[string]*sitter.Language{
			"go":         golang.GetLanguage(),
			"python":     python.GetLanguage(),
			"javascript": javascript.GetLanguage(),
			"typescript": typescript.GetLanguage(),
			"java":       java.GetLanguage(),
			"c":          c.GetLanguage(),
			"cpp":        cpp.GetLanguage(),
			"rust":       rust.GetLanguage(),
			"ruby":       ruby.GetLanguage(),
		},
	}
}

// ParseFile parses a source file and returns its syntax tree
func (tsp *TreeSitterParser) ParseFile(filePath string, language string) (*sitter.Tree, error) {
	lang, ok := tsp.languages[language]
	if !ok {
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Create parser
	parser := sitter.NewParser()
	parser.SetLanguage(lang)

	// Parse content
	ctx := context.Background()
	tree, err := parser.ParseCtx(ctx, nil, content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}

	return tree, nil
}

// ExtractSymbols extracts symbols from a parsed syntax tree
func (tsp *TreeSitterParser) ExtractSymbols(tree *sitter.Tree, filePath string, language string) ([]Symbol, error) {
	if tree == nil {
		return nil, fmt.Errorf("tree is nil")
	}

	extractor := NewTagExtractor(language)
	symbols := extractor.Extract(tree, filePath)

	return symbols, nil
}

// SupportedLanguages returns the list of supported languages
func (tsp *TreeSitterParser) SupportedLanguages() []string {
	langs := make([]string, 0, len(tsp.languages))
	for lang := range tsp.languages {
		langs = append(langs, lang)
	}
	return langs
}

// Query helpers for different language constructs

// goQueries defines Tree-sitter queries for Go
var goQueries = map[string]string{
	"functions": `
		(function_declaration
			name: (identifier) @name) @definition
	`,
	"methods": `
		(method_declaration
			receiver: (parameter_list) @receiver
			name: (field_identifier) @name) @definition
	`,
	"types": `
		(type_declaration
			(type_spec
				name: (type_identifier) @name)) @definition
	`,
	"interfaces": `
		(type_declaration
			(type_spec
				name: (type_identifier) @name
				type: (interface_type))) @definition
	`,
	"structs": `
		(type_declaration
			(type_spec
				name: (type_identifier) @name
				type: (struct_type))) @definition
	`,
}

// pythonQueries defines Tree-sitter queries for Python
var pythonQueries = map[string]string{
	"functions": `
		(function_definition
			name: (identifier) @name) @definition
	`,
	"classes": `
		(class_definition
			name: (identifier) @name) @definition
	`,
	"methods": `
		(class_definition
			body: (block
				(function_definition
					name: (identifier) @name))) @definition
	`,
}

// javascriptQueries defines Tree-sitter queries for JavaScript/TypeScript
var javascriptQueries = map[string]string{
	"functions": `
		(function_declaration
			name: (identifier) @name) @definition
	`,
	"classes": `
		(class_declaration
			name: (type_identifier) @name) @definition
	`,
	"methods": `
		(method_definition
			name: (property_identifier) @name) @definition
	`,
	"arrow_functions": `
		(variable_declarator
			name: (identifier) @name
			value: (arrow_function)) @definition
	`,
}

// javaQueries defines Tree-sitter queries for Java
var javaQueries = map[string]string{
	"classes": `
		(class_declaration
			name: (identifier) @name) @definition
	`,
	"methods": `
		(method_declaration
			name: (identifier) @name) @definition
	`,
	"interfaces": `
		(interface_declaration
			name: (identifier) @name) @definition
	`,
}

// cppQueries defines Tree-sitter queries for C/C++
var cppQueries = map[string]string{
	"functions": `
		(function_definition
			declarator: (function_declarator
				declarator: (identifier) @name)) @definition
	`,
	"classes": `
		(class_specifier
			name: (type_identifier) @name) @definition
	`,
	"structs": `
		(struct_specifier
			name: (type_identifier) @name) @definition
	`,
}

// rustQueries defines Tree-sitter queries for Rust
var rustQueries = map[string]string{
	"functions": `
		(function_item
			name: (identifier) @name) @definition
	`,
	"structs": `
		(struct_item
			name: (type_identifier) @name) @definition
	`,
	"enums": `
		(enum_item
			name: (type_identifier) @name) @definition
	`,
	"traits": `
		(trait_item
			name: (type_identifier) @name) @definition
	`,
	"impls": `
		(impl_item
			type: (type_identifier) @name) @definition
	`,
}

// rubyQueries defines Tree-sitter queries for Ruby
var rubyQueries = map[string]string{
	"methods": `
		(method
			name: (identifier) @name) @definition
	`,
	"classes": `
		(class
			name: (constant) @name) @definition
	`,
	"modules": `
		(module
			name: (constant) @name) @definition
	`,
}

// GetLanguageQueries returns the appropriate queries for a language
func GetLanguageQueries(language string) map[string]string {
	switch language {
	case "go":
		return goQueries
	case "python":
		return pythonQueries
	case "javascript", "typescript":
		return javascriptQueries
	case "java":
		return javaQueries
	case "c", "cpp":
		return cppQueries
	case "rust":
		return rustQueries
	case "ruby":
		return rubyQueries
	default:
		return nil
	}
}
