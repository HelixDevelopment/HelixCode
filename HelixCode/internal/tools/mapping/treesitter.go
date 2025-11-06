package mapping

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// TreeSitterParser defines the interface for tree-sitter parsing
type TreeSitterParser interface {
	// Parse parses source code into a ParsedTree
	Parse(ctx context.Context, source []byte, language string) (*ParsedTree, error)

	// ParseFile parses a file into a ParsedTree
	ParseFile(ctx context.Context, path string) (*ParsedTree, error)

	// GetSupportedLanguages returns a list of supported languages
	GetSupportedLanguages() []string

	// IsSupported checks if a language is supported
	IsSupported(language string) bool
}

// ParsedTree represents a parsed syntax tree
type ParsedTree struct {
	Language    string        `json:"language"`
	Root        *Node         `json:"root"`
	Source      []byte        `json:"-"`
	ParseErrors []*ParseError `json:"parse_errors,omitempty"`
}

// Node represents a syntax tree node
type Node struct {
	Type       string  `json:"type"`
	Text       string  `json:"text,omitempty"`
	StartByte  int     `json:"start_byte"`
	EndByte    int     `json:"end_byte"`
	StartPoint Point   `json:"start_point"`
	EndPoint   Point   `json:"end_point"`
	Children   []*Node `json:"children,omitempty"`
	Parent     *Node   `json:"-"`
}

// Point represents a position in source code (line, column)
type Point struct {
	Row    int `json:"row"`
	Column int `json:"column"`
}

// ParseError represents a parsing error
type ParseError struct {
	Message   string `json:"message"`
	StartByte int    `json:"start_byte"`
	EndByte   int    `json:"end_byte"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// DefaultTreeSitterParser implements TreeSitterParser
// Note: This is a placeholder implementation. In production, you would use
// github.com/smacker/go-tree-sitter and language-specific parsers
type DefaultTreeSitterParser struct {
	registry LanguageRegistry
}

// NewDefaultTreeSitterParser creates a new tree-sitter parser
func NewDefaultTreeSitterParser(registry LanguageRegistry) *DefaultTreeSitterParser {
	return &DefaultTreeSitterParser{
		registry: registry,
	}
}

// Parse parses source code into a ParsedTree
func (p *DefaultTreeSitterParser) Parse(ctx context.Context, source []byte, language string) (*ParsedTree, error) {
	language = strings.ToLower(language)

	// Check if language is supported
	if !IsSupported(language) {
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	// Get language parser
	parser, err := p.registry.Get(language)
	if err != nil {
		return nil, fmt.Errorf("failed to get language parser: %w", err)
	}

	// Parse the source
	tree, err := parser.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source: %w", err)
	}

	return tree, nil
}

// ParseFile parses a file into a ParsedTree
func (p *DefaultTreeSitterParser) ParseFile(ctx context.Context, path string) (*ParsedTree, error) {
	// Detect language from file extension
	language := DetectLanguage(path)
	if language == "" {
		return nil, fmt.Errorf("could not detect language for file: %s", path)
	}

	// Read file
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse
	return p.Parse(ctx, source, language)
}

// GetSupportedLanguages returns a list of supported languages
func (p *DefaultTreeSitterParser) GetSupportedLanguages() []string {
	return p.registry.List()
}

// IsSupported checks if a language is supported
func (p *DefaultTreeSitterParser) IsSupported(language string) bool {
	return IsSupported(language)
}

// GetText returns the text content of a node from source
func (n *Node) GetText(source []byte) string {
	if n.StartByte >= 0 && n.EndByte <= len(source) && n.StartByte < n.EndByte {
		return string(source[n.StartByte:n.EndByte])
	}
	return n.Text
}

// IsNamed checks if a node is named (not anonymous)
func (n *Node) IsNamed() bool {
	// Anonymous nodes typically have types that start with special characters
	return len(n.Type) > 0 && n.Type[0] != '"' && n.Type[0] != '\''
}

// FindChild finds a child node by type
func (n *Node) FindChild(nodeType string) *Node {
	for _, child := range n.Children {
		if child.Type == nodeType {
			return child
		}
	}
	return nil
}

// FindChildren finds all child nodes by type
func (n *Node) FindChildren(nodeType string) []*Node {
	var result []*Node
	for _, child := range n.Children {
		if child.Type == nodeType {
			result = append(result, child)
		}
	}
	return result
}

// Walk traverses the tree depth-first and calls the callback for each node
func (n *Node) Walk(callback func(*Node) bool) {
	if !callback(n) {
		return
	}
	for _, child := range n.Children {
		child.Walk(callback)
	}
}

// Query executes a simple query on the tree
// Note: This is a simplified implementation. Production code should use
// tree-sitter's query language
func (t *ParsedTree) Query(nodeType string) []*Node {
	var results []*Node
	t.Root.Walk(func(node *Node) bool {
		if node.Type == nodeType {
			results = append(results, node)
		}
		return true
	})
	return results
}

// GetNodeAtPosition finds the node at a specific position
func (t *ParsedTree) GetNodeAtPosition(line, column int) *Node {
	var result *Node
	t.Root.Walk(func(node *Node) bool {
		if node.StartPoint.Row <= line && line <= node.EndPoint.Row {
			if line == node.StartPoint.Row && column < node.StartPoint.Column {
				return false
			}
			if line == node.EndPoint.Row && column > node.EndPoint.Column {
				return false
			}
			result = node
			return true
		}
		return false
	})
	return result
}

// ExtractComments extracts comments from the tree
func (t *ParsedTree) ExtractComments() []*Comment {
	var comments []*Comment

	commentTypes := []string{"comment", "line_comment", "block_comment", "doc_comment"}

	for _, commentType := range commentTypes {
		nodes := t.Query(commentType)
		for _, node := range nodes {
			text := node.GetText(t.Source)
			isDoc := strings.Contains(commentType, "doc") ||
				strings.HasPrefix(text, "/**") ||
				strings.HasPrefix(text, "///")

			comments = append(comments, &Comment{
				Text:      text,
				StartLine: node.StartPoint.Row + 1,
				EndLine:   node.EndPoint.Row + 1,
				IsDoc:     isDoc,
			})
		}
	}

	return comments
}

// HasErrors checks if the tree has parse errors
func (t *ParsedTree) HasErrors() bool {
	return len(t.ParseErrors) > 0
}

// GetErrorCount returns the number of parse errors
func (t *ParsedTree) GetErrorCount() int {
	return len(t.ParseErrors)
}

// SimpleLexer provides basic lexical analysis for languages without tree-sitter support
type SimpleLexer struct{}

// NewSimpleLexer creates a new simple lexer
func NewSimpleLexer() *SimpleLexer {
	return &SimpleLexer{}
}

// ParseBasic performs basic parsing without tree-sitter
// This is a fallback for languages that don't have tree-sitter support yet
func (l *SimpleLexer) ParseBasic(source []byte, language string) (*ParsedTree, error) {
	tree := &ParsedTree{
		Language: language,
		Source:   source,
		Root: &Node{
			Type:      "program",
			StartByte: 0,
			EndByte:   len(source),
			StartPoint: Point{Row: 0, Column: 0},
			EndPoint: Point{
				Row:    countLines(source) - 1,
				Column: 0,
			},
			Children: []*Node{},
		},
	}

	// This would be implemented with language-specific lexing rules
	// For now, it's just a placeholder

	return tree, nil
}

// countLines counts the number of lines in source code
func countLines(source []byte) int {
	count := 1
	for _, b := range source {
		if b == '\n' {
			count++
		}
	}
	return count
}
