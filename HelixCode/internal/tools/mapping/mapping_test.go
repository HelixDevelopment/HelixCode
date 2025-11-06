package mapping

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test sample code snippets for various languages
const (
	goSample = `package main

import "fmt"

// Add adds two numbers
func Add(a, b int) int {
	return a + b
}

type Calculator struct {
	result int
}

func (c *Calculator) Multiply(a, b int) int {
	return a * b
}

func main() {
	fmt.Println(Add(1, 2))
}
`

	pythonSample = `import math

def fibonacci(n):
    """Calculate fibonacci number"""
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)

class Point:
    def __init__(self, x, y):
        self.x = x
        self.y = y

    def distance(self):
        return math.sqrt(self.x**2 + self.y**2)
`

	jsSample = `const lodash = require('lodash');

function factorial(n) {
    if (n <= 1) return 1;
    return n * factorial(n - 1);
}

class Rectangle {
    constructor(width, height) {
        this.width = width;
        this.height = height;
    }

    area() {
        return this.width * this.height;
    }
}

module.exports = { factorial, Rectangle };
`

	tsSample = `import { Observable } from 'rxjs';

interface User {
    id: number;
    name: string;
    email: string;
}

export class UserService {
    private users: User[] = [];

    constructor() {}

    addUser(user: User): void {
        this.users.push(user);
    }

    getUser(id: number): User | undefined {
        return this.users.find(u => u.id === id);
    }
}

export function greet(name: string): string {
    return 'Hello, ' + name;
}
`

	rustSample = `use std::fmt;

pub struct Point {
    x: f64,
    y: f64,
}

impl Point {
    pub fn new(x: f64, y: f64) -> Self {
        Point { x, y }
    }

    pub fn distance(&self) -> f64 {
        (self.x.powi(2) + self.y.powi(2)).sqrt()
    }
}

pub fn add(a: i32, b: i32) -> i32 {
    a + b
}

fn main() {
    let p = Point::new(3.0, 4.0);
    println!("Distance: {}", p.distance());
}
`

	javaSample = `package com.example;

import java.util.ArrayList;
import java.util.List;

public class Calculator {
    private int result;

    public Calculator() {
        this.result = 0;
    }

    public int add(int a, int b) {
        result = a + b;
        return result;
    }

    public int multiply(int a, int b) {
        result = a * b;
        return result;
    }

    public static void main(String[] args) {
        Calculator calc = new Calculator();
        System.out.println(calc.add(1, 2));
    }
}
`
)

// TestDefinitionType tests the DefinitionType enum
func TestDefinitionType(t *testing.T) {
	tests := []struct {
		name     string
		defType  DefinitionType
		expected string
	}{
		{"Function", DefFunction, "Function"},
		{"Method", DefMethod, "Method"},
		{"Class", DefClass, "Class"},
		{"Interface", DefInterface, "Interface"},
		{"Struct", DefStruct, "Struct"},
		{"Enum", DefEnum, "Enum"},
		{"Type", DefType, "Type"},
		{"Variable", DefVariable, "Variable"},
		{"Constant", DefConstant, "Constant"},
		{"Module", DefModule, "Module"},
		{"Namespace", DefNamespace, "Namespace"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.defType.String())
		})
	}
}

// TestVisibility tests the Visibility enum
func TestVisibility(t *testing.T) {
	tests := []struct {
		name       string
		visibility Visibility
		expected   string
	}{
		{"Public", VisibilityPublic, "Public"},
		{"Private", VisibilityPrivate, "Private"},
		{"Protected", VisibilityProtected, "Protected"},
		{"Internal", VisibilityInternal, "Internal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.visibility.String())
		})
	}
}

// TestDefinitionGetSignature tests Definition.GetSignature
func TestDefinitionGetSignature(t *testing.T) {
	tests := []struct {
		name     string
		def      *Definition
		expected string
	}{
		{
			name: "Function with parameters",
			def: &Definition{
				Type: DefFunction,
				Name: "add",
				Parameters: []*Parameter{
					{Name: "a", Type: "int"},
					{Name: "b", Type: "int"},
				},
				ReturnType: "int",
			},
			expected: "add(a int, b int) int",
		},
		{
			name: "Method without return type",
			def: &Definition{
				Type: DefMethod,
				Name: "print",
				Parameters: []*Parameter{
					{Name: "msg", Type: "string"},
				},
			},
			expected: "print(msg string)",
		},
		{
			name: "Class",
			def: &Definition{
				Type: DefClass,
				Name: "Calculator",
			},
			expected: "Calculator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.def.GetSignature())
		})
	}
}

// TestDefinitionLineCount tests Definition.LineCount
func TestDefinitionLineCount(t *testing.T) {
	def := &Definition{
		StartLine: 10,
		EndLine:   20,
	}
	assert.Equal(t, 11, def.LineCount())
}

// TestDetectLanguage tests language detection
func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"Go file", "main.go", "go"},
		{"JavaScript file", "index.js", "javascript"},
		{"TypeScript file", "app.ts", "typescript"},
		{"Python file", "script.py", "python"},
		{"Rust file", "main.rs", "rust"},
		{"Java file", "Main.java", "java"},
		{"C file", "main.c", "c"},
		{"C++ file", "main.cpp", "cpp"},
		{"Ruby file", "script.rb", "ruby"},
		{"PHP file", "index.php", "php"},
		{"Unknown extension", "file.xyz", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, DetectLanguage(tt.path))
		})
	}
}

// TestIsSupported tests language support check
func TestIsSupported(t *testing.T) {
	tests := []struct {
		name     string
		language string
		expected bool
	}{
		{"Go", "go", true},
		{"JavaScript", "javascript", true},
		{"TypeScript", "typescript", true},
		{"Python", "python", true},
		{"Unsupported", "brainfuck", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsSupported(tt.language))
		})
	}
}

// TestLanguageRegistry tests the language registry
func TestLanguageRegistry(t *testing.T) {
	registry := NewDefaultLanguageRegistry()

	t.Run("List languages", func(t *testing.T) {
		languages := registry.List()
		// Empty initially since no parsers are registered
		assert.NotNil(t, languages)
	})

	t.Run("Get language info", func(t *testing.T) {
		info, err := registry.GetLanguageInfo("go")
		require.NoError(t, err)
		assert.Equal(t, "go", info.Name)
		assert.Contains(t, info.Extensions, ".go")
	})

	t.Run("Get by extension", func(t *testing.T) {
		_, err := registry.GetByExtension(".go")
		// Should error since no parser registered
		if err == nil {
			t.Skip("Parser already registered")
		}
	})
}

// TestTokenCounter tests token counting
func TestTokenCounter(t *testing.T) {
	counter := NewTokenCounter()

	tests := []struct {
		name     string
		source   string
		language string
		minCount int
	}{
		{"Go code", goSample, "go", 20},
		{"Python code", pythonSample, "python", 20},
		{"JavaScript code", jsSample, "javascript", 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := counter.Count([]byte(tt.source), tt.language)
			assert.Greater(t, count, tt.minCount, "Token count should be greater than minimum")
		})
	}
}

// TestCountLines tests line counting
func TestCountLines(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected int
	}{
		{"Single line", "hello", 1},
		{"Two lines", "hello\nworld", 2},
		{"Three lines with trailing newline", "a\nb\nc\n", 4},
		{"Empty", "", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, CountLines([]byte(tt.source)))
		})
	}
}

// TestCalculateChecksum tests checksum calculation
func TestCalculateChecksum(t *testing.T) {
	data1 := []byte("hello")
	data2 := []byte("world")
	data3 := []byte("hello")

	checksum1 := CalculateChecksum(data1)
	checksum2 := CalculateChecksum(data2)
	checksum3 := CalculateChecksum(data3)

	assert.NotEqual(t, checksum1, checksum2, "Different data should have different checksums")
	assert.Equal(t, checksum1, checksum3, "Same data should have same checksums")
	assert.Len(t, checksum1, 64, "SHA-256 checksum should be 64 hex characters")
}

// TestCodebaseMap tests CodebaseMap operations
func TestCodebaseMap(t *testing.T) {
	cmap := NewCodebaseMap("/test/root")

	t.Run("Add file", func(t *testing.T) {
		fileMap := &FileMap{
			Path:     "/test/root/main.go",
			Language: "go",
			Lines:    100,
			Tokens:   500,
			Definitions: []*Definition{
				{
					Name:          "Add",
					QualifiedName: "main.Add",
					Type:          DefFunction,
				},
			},
		}

		cmap.AddFile(fileMap)

		assert.Equal(t, 1, cmap.TotalFiles)
		assert.Equal(t, 100, cmap.TotalLines)
		assert.Equal(t, 500, cmap.TotalTokens)
		assert.Equal(t, 1, cmap.Languages["go"])
		assert.NotNil(t, cmap.Definitions["main.Add"])
	})

	t.Run("Get definition", func(t *testing.T) {
		def, ok := cmap.GetDefinition("main.Add")
		assert.True(t, ok)
		assert.Equal(t, "Add", def.Name)
	})

	t.Run("Remove file", func(t *testing.T) {
		cmap.RemoveFile("/test/root/main.go")
		assert.Equal(t, 0, cmap.TotalFiles)
		assert.Equal(t, 0, cmap.TotalLines)
		assert.Equal(t, 0, cmap.TotalTokens)
	})
}

// TestDiskCacheManager tests cache operations
func TestDiskCacheManager(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewDiskCacheManager(tmpDir)

	cmap := NewCodebaseMap(tmpDir)
	cmap.TotalFiles = 10
	cmap.TotalLines = 1000

	t.Run("Save and load", func(t *testing.T) {
		err := cache.Save(cmap)
		require.NoError(t, err)

		loaded, err := cache.Load(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, cmap.TotalFiles, loaded.TotalFiles)
		assert.Equal(t, cmap.TotalLines, loaded.TotalLines)
	})

	t.Run("Cache stats", func(t *testing.T) {
		stats, err := cache.GetCacheStats()
		require.NoError(t, err)
		assert.Greater(t, stats.TotalSize, int64(0))
		assert.Equal(t, CacheVersion, stats.Version)
	})

	t.Run("Clear cache", func(t *testing.T) {
		err := cache.Clear()
		require.NoError(t, err)

		_, err = cache.Load(tmpDir)
		assert.Error(t, err)
	})
}

// TestMapFile tests file mapping
func TestMapFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		filename string
		content  string
		language string
	}{
		{"Go file", "test.go", goSample, "go"},
		{"Python file", "test.py", pythonSample, "python"},
		{"JavaScript file", "test.js", jsSample, "javascript"},
		{"TypeScript file", "test.ts", tsSample, "typescript"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			path := filepath.Join(tmpDir, tt.filename)
			err := os.WriteFile(path, []byte(tt.content), 0644)
			require.NoError(t, err)

			// Create mapper
			mapper := NewMapper(tmpDir)
			ctx := context.Background()

			// Map file (will create basic file map without parser)
			fileMap, err := mapper.MapFile(ctx, path)

			// May error without actual tree-sitter parsers, which is expected
			if err != nil {
				t.Skipf("Skipping test without tree-sitter parser: %v", err)
				return
			}

			// Verify
			assert.Equal(t, path, fileMap.Path)
			assert.Equal(t, tt.language, fileMap.Language)
			assert.Greater(t, fileMap.Lines, 0)
			assert.Greater(t, fileMap.Tokens, 0)
			assert.NotEmpty(t, fileMap.Checksum)
		})
	}
}

// TestMapCodebase tests codebase mapping
func TestMapCodebase(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := map[string]string{
		"main.go":    goSample,
		"script.py":  pythonSample,
		"index.js":   jsSample,
		"app.ts":     tsSample,
		"README.md":  "# Test Project",
		"config.txt": "some config",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Create mapper
	mapper := NewMapper(tmpDir)
	ctx := context.Background()

	t.Run("Map entire codebase", func(t *testing.T) {
		cmap, err := mapper.MapCodebase(ctx, tmpDir, nil)

		// May have errors without actual tree-sitter parsers
		if err != nil && cmap == nil {
			t.Skipf("Skipping test without tree-sitter parsers: %v", err)
			return
		}

		// If no files were mapped, skip (no parsers available)
		if cmap.TotalFiles == 0 {
			t.Skip("No files mapped - tree-sitter parsers not available")
			return
		}

		// Should only map source files (not README.md or config.txt)
		assert.Equal(t, 4, cmap.TotalFiles)
		assert.Contains(t, cmap.Files, filepath.Join(tmpDir, "main.go"))
		assert.Contains(t, cmap.Files, filepath.Join(tmpDir, "script.py"))
		assert.Contains(t, cmap.Files, filepath.Join(tmpDir, "index.js"))
		assert.Contains(t, cmap.Files, filepath.Join(tmpDir, "app.ts"))

		// Check languages
		assert.Equal(t, 1, cmap.Languages["go"])
		assert.Equal(t, 1, cmap.Languages["python"])
		assert.Equal(t, 1, cmap.Languages["javascript"])
		assert.Equal(t, 1, cmap.Languages["typescript"])

		// Check totals
		assert.Greater(t, cmap.TotalLines, 0)
		assert.Greater(t, cmap.TotalTokens, 0)
	})

	t.Run("Map with custom options", func(t *testing.T) {
		opts := &MapOptions{
			UseCache:    false,
			Concurrency: 2,
			MaxFileSize: 1024 * 1024,
			Languages:   []string{"go", "python"},
		}

		cmap, err := mapper.MapCodebase(ctx, tmpDir, opts)

		// May have errors without actual tree-sitter parsers
		if err != nil && cmap == nil {
			t.Skipf("Skipping test without tree-sitter parsers: %v", err)
			return
		}

		// If no files were mapped, skip (no parsers available)
		if cmap.TotalFiles == 0 {
			t.Skip("No files mapped - tree-sitter parsers not available")
			return
		}

		// Should only map Go and Python files
		assert.Equal(t, 2, cmap.TotalFiles)
		assert.Contains(t, cmap.Files, filepath.Join(tmpDir, "main.go"))
		assert.Contains(t, cmap.Files, filepath.Join(tmpDir, "script.py"))
	})

	t.Run("Use cache", func(t *testing.T) {
		opts := DefaultMapOptions()

		// First map
		cmap1, err := mapper.MapCodebase(ctx, tmpDir, opts)
		if err != nil && cmap1 == nil {
			t.Skipf("Skipping test without tree-sitter parsers: %v", err)
			return
		}

		// Second map (should use cache)
		cmap2, err := mapper.MapCodebase(ctx, tmpDir, opts)
		if err != nil && cmap2 == nil {
			t.Skipf("Skipping test without tree-sitter parsers: %v", err)
			return
		}

		assert.Equal(t, cmap1.TotalFiles, cmap2.TotalFiles)
	})
}

// TestDependencyGraph tests dependency graph operations
func TestDependencyGraph(t *testing.T) {
	graph := &DependencyGraph{
		Nodes: map[string]*DependencyNode{
			"a.go": {Path: "a.go"},
			"b.go": {Path: "b.go"},
			"c.go": {Path: "c.go"},
		},
		Edges: map[string][]string{
			"a.go": {"b.go"},
			"b.go": {"c.go"},
		},
	}

	t.Run("Get dependencies", func(t *testing.T) {
		deps := graph.GetDependencies("a.go")
		assert.Equal(t, []string{"b.go"}, deps)
	})

	t.Run("Get dependents", func(t *testing.T) {
		dependents := graph.GetDependents("b.go")
		assert.Contains(t, dependents, "a.go")
	})

	t.Run("No cycle", func(t *testing.T) {
		assert.False(t, graph.HasCycle())
	})

	t.Run("Has cycle", func(t *testing.T) {
		graph.Edges["c.go"] = []string{"a.go"} // Create cycle
		assert.True(t, graph.HasCycle())
	})
}

// TestExtractRelativeIndentation tests relative indentation extraction
func TestExtractRelativeIndentation(t *testing.T) {
	source := []byte(`    func main() {
        fmt.Println("hello")
        if true {
            fmt.Println("world")
        }
    }`)

	result := ExtractRelativeIndentation(source, 0, 4)
	assert.NotEmpty(t, result)

	// Should have removed the common 4-space indentation
	lines := strings.Split(result, "\n")
	assert.True(t, lines[0] == "func main() {" || strings.HasPrefix(lines[0], "func"))
}

// TestNodeOperations tests Node operations
func TestNodeOperations(t *testing.T) {
	root := &Node{
		Type: "program",
		Children: []*Node{
			{Type: "function", Text: "func1"},
			{Type: "class", Text: "Class1"},
			{Type: "function", Text: "func2"},
		},
	}

	t.Run("FindChild", func(t *testing.T) {
		node := root.FindChild("class")
		require.NotNil(t, node)
		assert.Equal(t, "Class1", node.Text)
	})

	t.Run("FindChildren", func(t *testing.T) {
		nodes := root.FindChildren("function")
		assert.Len(t, nodes, 2)
	})

	t.Run("Walk", func(t *testing.T) {
		count := 0
		root.Walk(func(n *Node) bool {
			count++
			return true
		})
		assert.Equal(t, 4, count) // root + 3 children
	})
}

// TestParsedTreeQuery tests tree querying
func TestParsedTreeQuery(t *testing.T) {
	tree := &ParsedTree{
		Language: "go",
		Root: &Node{
			Type: "program",
			Children: []*Node{
				{Type: "function_declaration", Text: "func1"},
				{Type: "type_declaration", Text: "type1"},
				{Type: "function_declaration", Text: "func2"},
			},
		},
	}

	results := tree.Query("function_declaration")
	assert.Len(t, results, 2)
}

// TestMapOptions tests MapOptions
func TestMapOptions(t *testing.T) {
	t.Run("Default options", func(t *testing.T) {
		opts := DefaultMapOptions()
		assert.True(t, opts.UseCache)
		assert.Equal(t, 10, opts.Concurrency)
		assert.Greater(t, opts.MaxFileSize, int64(0))
		assert.Contains(t, opts.ExcludeDirs, ".git")
		assert.Contains(t, opts.ExcludeDirs, "node_modules")
	})

	t.Run("Custom options", func(t *testing.T) {
		opts := &MapOptions{
			UseCache:      false,
			Concurrency:   5,
			MaxFileSize:   500 * 1024,
			ExcludeDirs:   []string{"custom"},
			IncludeHidden: true,
			Languages:     []string{"go"},
		}

		assert.False(t, opts.UseCache)
		assert.Equal(t, 5, opts.Concurrency)
		assert.Equal(t, int64(500*1024), opts.MaxFileSize)
	})
}

// TestImportAnalyzer tests import analysis
func TestImportAnalyzer(t *testing.T) {
	analyzer := NewImportAnalyzer()
	cmap := NewCodebaseMap("/test")

	fileMap := &FileMap{
		Path:     "/test/main.go",
		Language: "go",
		Imports: []*Import{
			{Path: "fmt", IsRelative: false},
			{Path: "./utils", IsRelative: true},
		},
	}

	t.Run("Resolve dependencies", func(t *testing.T) {
		deps := analyzer.ResolveDependencies(fileMap, cmap)
		// deps can be empty slice, which is valid
		assert.NotNil(t, deps)
		assert.IsType(t, []string{}, deps)
	})

	t.Run("Build dependency graph", func(t *testing.T) {
		cmap.AddFile(fileMap)
		graph := analyzer.BuildDependencyGraph(cmap)
		assert.NotNil(t, graph)
		assert.Contains(t, graph.Nodes, "/test/main.go")
	})
}

// Benchmark tests

func BenchmarkTokenCounting(b *testing.B) {
	counter := NewTokenCounter()
	source := []byte(goSample)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counter.Count(source, "go")
	}
}

func BenchmarkChecksumCalculation(b *testing.B) {
	data := []byte(goSample)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateChecksum(data)
	}
}

func BenchmarkLanguageDetection(b *testing.B) {
	path := "main.go"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectLanguage(path)
	}
}
