# Editor Package

A comprehensive multi-format code editing system for HelixCode that supports various edit formats optimized for different LLM models.

## Features

- **Multiple Edit Formats**: Diff, Whole file, Search/Replace, and Line-based editing
- **Model-Specific Optimization**: Automatic format selection based on LLM model capabilities
- **Thread-Safe**: Concurrent editing with mutex protection
- **Validation**: Built-in edit validation before application
- **Backup Support**: Optional file backup before editing
- **Syntax Validation**: Basic syntax checking for Go, JSON, and YAML files
- **Large File Support**: Efficient handling of files up to several MB

## Edit Formats

### 1. Diff Format (`EditFormatDiff`)
Unix unified diff format for precise, contextual edits.

**Best for:**
- Models: GPT-4, Gemini Pro, Llama 70B+, DeepSeek Coder
- Complex edits with multiple changes
- Large files where context is important
- Precise line-by-line modifications

**Example:**
```go
editor, _ := editor.NewCodeEditor(editor.EditFormatDiff)
edit := editor.Edit{
    FilePath: "test.go",
    Format:   editor.EditFormatDiff,
    Content: `--- test.go
+++ test.go
@@ -1,3 +1,3 @@
 line1
-line2
+modified line
 line3`,
}
editor.ApplyEdit(edit)
```

### 2. Whole File Format (`EditFormatWhole`)
Complete file replacement.

**Best for:**
- Models: Gemini Pro, Llama 3 8B, CodeLlama 34B, O1 models
- Small to medium files
- Complete rewrites
- Simple transformations

**Example:**
```go
editor, _ := editor.NewCodeEditor(editor.EditFormatWhole)
edit := editor.Edit{
    FilePath: "config.json",
    Format:   editor.EditFormatWhole,
    Content:  `{"key": "new value"}`,
}
editor.ApplyEdit(edit)
```

### 3. Search/Replace Format (`EditFormatSearchReplace`)
Pattern-based search and replacement with regex support.

**Best for:**
- Models: Claude (all versions), GPT-3.5, Mistral, Phi-3
- Multiple similar changes
- Pattern-based edits
- Refactoring operations

**Example:**
```go
editor, _ := editor.NewCodeEditor(editor.EditFormatSearchReplace)
edit := editor.Edit{
    FilePath: "code.go",
    Format:   editor.EditFormatSearchReplace,
    Content: []editor.SearchReplace{
        {Search: "oldFunc", Replace: "newFunc", Count: -1, Regex: false},
        {Search: `test\d+`, Replace: "exam", Count: -1, Regex: true},
    },
}
editor.ApplyEdit(edit)
```

### 4. Line-Based Format (`EditFormatLines`)
Edit specific line ranges.

**Best for:**
- Models: GPT-4, Claude, Gemini, CodeLlama
- Targeted line modifications
- Insertion/deletion operations
- Non-overlapping edits

**Example:**
```go
editor, _ := editor.NewCodeEditor(editor.EditFormatLines)
edit := editor.Edit{
    FilePath: "file.txt",
    Format:   editor.EditFormatLines,
    Content: []editor.LineEdit{
        {StartLine: 5, EndLine: 7, NewContent: "new\ncontent\nhere"},
    },
}
editor.ApplyEdit(edit)
```

## Model-Specific Format Selection

The package includes a comprehensive model-to-format mapping:

```go
// Automatic selection
format := editor.SelectFormatForModel("gpt-4o")
// Returns: EditFormatDiff

format = editor.SelectFormatForModel("claude-3-sonnet")
// Returns: EditFormatSearchReplace

// Check if a model supports a format
supported := editor.SupportsFormat("gpt-4", editor.EditFormatDiff)
// Returns: true

// Get full capabilities
capability := editor.GetModelCapability("claude-3-opus")
// Returns: ModelCapability with all supported formats
```

## Intelligent Format Recommendation

The package provides smart format selection based on multiple factors:

```go
// Based on file size
format := editor.SelectBestFormat("gpt-4", 150*1024) // 150KB
// Returns: EditFormatWhole (large file optimization)

// Based on edit complexity
format = editor.SelectFormatByComplexity("claude-3-sonnet", editor.ComplexityComplex)
// Returns: EditFormatDiff

// Full recommendation with reasoning
recommendation := editor.RecommendFormat(
    "gpt-4o",
    20*1024,                    // 20KB file
    editor.ComplexityMedium,
)
// Returns: FormatRecommendation{
//   Format: EditFormatDiff,
//   Confidence: 0.95,
//   Reasoning: "Small file size, using model's preferred format"
// }
```

## Supported Models

### OpenAI
- GPT-4o, GPT-4 Turbo → Diff
- GPT-4 → Diff
- GPT-3.5 Turbo → Search/Replace
- O1 Preview/Mini → Whole

### Anthropic
- Claude 3 Opus, Sonnet, Haiku → Search/Replace
- Claude 3.5 Sonnet → Search/Replace
- Claude Sonnet 4 → Search/Replace

### Google
- Gemini Pro → Whole
- Gemini Ultra, 1.5 Pro → Diff
- Gemini 1.5 Flash → Search/Replace

### Meta
- Llama 2 (7B-13B) → Whole
- Llama 2 70B → Diff
- Llama 3 8B → Whole
- Llama 3 70B+ → Diff

### Code-Specific
- CodeLlama (7B-34B) → Whole/Search
- CodeLlama 70B → Diff
- DeepSeek Coder → Diff
- StarCoder → Whole
- WizardCoder → Search/Replace

### Others
- Mistral, Mixtral → Search/Diff
- Qwen 2.5 → Diff
- xAI Grok → Diff
- Microsoft Phi → Search/Replace

## Advanced Features

### Validation
All edits are validated before application:

```go
editor, _ := editor.NewCodeEditor(editor.EditFormatDiff)
edit := editor.Edit{
    FilePath: "test.go",
    Format:   editor.EditFormatDiff,
    Content:  diffContent,
}

// Validate without applying
if err := editor.ValidateEdit(edit); err != nil {
    log.Printf("Invalid edit: %v", err)
    return
}

// Apply (also validates)
editor.ApplyEdit(edit)
```

### Backup Creation
Automatically create backups before editing:

```go
edit := editor.Edit{
    FilePath: "important.go",
    Format:   editor.EditFormatWhole,
    Content:  newContent,
    Backup:   true, // Creates important.go.bak
}
editor.ApplyEdit(edit)
```

### Syntax Validation
Automatic syntax validation for supported file types:

- **Go files**: Package declaration, bracket balance
- **JSON files**: Structure, bracket/brace balance, string closure
- **YAML files**: Indentation, key-value pairs

### Concurrent Editing
Thread-safe editing with internal mutex:

```go
editor, _ := editor.NewCodeEditor(editor.EditFormatWhole)

// Safe to call from multiple goroutines
go editor.ApplyEdit(edit1)
go editor.ApplyEdit(edit2)
go editor.ApplyEdit(edit3)
```

### Custom Validators
Implement custom validation logic:

```go
type MyValidator struct{}

func (v *MyValidator) Validate(edit editor.Edit) error {
    // Custom validation logic
    return nil
}

editor, _ := editor.NewCodeEditor(editor.EditFormatDiff)
editor.SetValidator(&MyValidator{})
```

## Error Handling

The package provides detailed error messages:

```go
err := editor.ApplyEdit(edit)
if err != nil {
    // Errors include context about what failed
    log.Printf("Edit failed: %v", err)
    // Examples:
    // - "validation failed: file path is required"
    // - "apply failed: hunk context mismatch at line 42"
    // - "syntax validation failed: unbalanced braces"
}
```

## Performance Considerations

### File Size Guidelines
- **Small (<10KB)**: Use preferred format
- **Medium (10-100KB)**: Use Diff or Search/Replace
- **Large (>100KB)**: Consider Whole file replacement

### Memory Usage
- Diff: O(n) where n is file size
- Whole: O(n) where n is new content size
- Search/Replace: O(n*m) where m is number of operations
- Lines: O(n) where n is file size

### Concurrent Operations
The editor uses a mutex for thread safety. For high-throughput scenarios, consider:
- Creating multiple editor instances
- Batching edits to the same file
- Using appropriate format for file size

## Testing

The package includes 276+ comprehensive tests covering:
- Valid and invalid inputs for each format
- Error handling and edge cases
- Large file handling (10K+ lines)
- Concurrent editing scenarios
- Integration tests across formats
- Model format selection
- Syntax validation

Run tests:
```bash
go test -v ./internal/editor/...
```

## Usage in HelixCode

The editor package integrates with HelixCode's LLM system:

1. **Model Detection**: When an LLM generates code edits, the system detects the model
2. **Format Selection**: Automatically selects the optimal edit format
3. **Validation**: Validates the edit before application
4. **Application**: Applies the edit with optional backup
5. **Error Recovery**: Provides detailed error messages for debugging

## Future Enhancements

Potential additions:
- More syntax validators (Python, JavaScript, Rust, etc.)
- Incremental diff application for very large files
- Edit history and rollback
- Dry-run mode with preview
- Edit statistics and metrics
- AST-based editing for structural changes

## License

Part of the HelixCode project.
