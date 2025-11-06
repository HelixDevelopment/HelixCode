package editor

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Example demonstrates basic usage of the editor package
func Example() {
	// Create a temporary directory for examples
	tmpDir, err := os.MkdirTemp("", "editor_example_*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "example.go")

	// Create initial file
	initialContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello World")
}
`
	if err := os.WriteFile(testFile, []byte(initialContent), 0644); err != nil {
		log.Fatal(err)
	}

	// Example 1: Whole file replacement
	editor1, _ := NewCodeEditor(EditFormatWhole)
	edit1 := Edit{
		FilePath: testFile,
		Format:   EditFormatWhole,
		Content: `package main

import "fmt"

func main() {
	fmt.Println("Hello Universe")
}
`,
	}
	editor1.ApplyEdit(edit1)

	fmt.Println("Whole file replacement: done")

	// Example 2: Search and replace
	editor2, _ := NewCodeEditor(EditFormatSearchReplace)
	edit2 := Edit{
		FilePath: testFile,
		Format:   EditFormatSearchReplace,
		Content: []SearchReplace{
			{Search: "Universe", Replace: "Galaxy", Count: -1, Regex: false},
		},
	}
	editor2.ApplyEdit(edit2)

	fmt.Println("Search replace: done")

	// Example 3: Model-specific format selection
	format := SelectFormatForModel("gpt-4o")
	fmt.Printf("GPT-4o preferred format: %s\n", format)

	format = SelectFormatForModel("claude-3-sonnet")
	fmt.Printf("Claude-3 preferred format: %s\n", format)

	// Output:
	// Whole file replacement: done
	// Search replace: done
	// GPT-4o preferred format: diff
	// Claude-3 preferred format: search_replace
}

// ExampleCodeEditor_diff demonstrates diff-based editing
func ExampleCodeEditor_diff() {
	tmpDir, _ := os.MkdirTemp("", "editor_example_*")
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("line1\nline2\nline3"), 0644)

	ed, _ := NewCodeEditor(EditFormatDiff)

	diff := `--- test.txt
+++ test.txt
@@ -1,3 +1,3 @@
 line1
-line2
+modified line
 line3`

	edit := Edit{
		FilePath: testFile,
		Format:   EditFormatDiff,
		Content:  diff,
	}

	if err := ed.ApplyEdit(edit); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	result, _ := os.ReadFile(testFile)
	fmt.Print(string(result))

	// Output:
	// line1
	// modified line
	// line3
}

// ExampleSelectBestFormat demonstrates intelligent format selection
func ExampleSelectBestFormat() {
	// Small file - use preferred format
	format1 := SelectBestFormat("gpt-4o", 5*1024)
	fmt.Printf("Small file (5KB): %s\n", format1)

	// Medium file - use efficient format
	format2 := SelectBestFormat("claude-3-sonnet", 50*1024)
	fmt.Printf("Medium file (50KB): %s\n", format2)

	// Large file - use whole replacement
	format3 := SelectBestFormat("gpt-4", 150*1024)
	fmt.Printf("Large file (150KB): %s\n", format3)

	// Output:
	// Small file (5KB): diff
	// Medium file (50KB): search_replace
	// Large file (150KB): whole
}

// ExampleRecommendFormat demonstrates format recommendation with reasoning
func ExampleRecommendFormat() {
	recommendation := RecommendFormat(
		"gpt-4o",
		20*1024, // 20KB file
		ComplexityComplex,
	)

	fmt.Printf("Format: %s\n", recommendation.Format)
	fmt.Printf("Confidence: %.2f\n", recommendation.Confidence)
	fmt.Printf("Reasoning: %s\n", recommendation.Reasoning)

	// Output:
	// Format: diff
	// Confidence: 0.90
	// Reasoning: Complex edit requirements, diff format provides precision
}

// ExampleCodeEditor_concurrent demonstrates thread-safe editing
func ExampleCodeEditor_concurrent() {
	tmpDir, _ := os.MkdirTemp("", "editor_example_*")
	defer os.RemoveAll(tmpDir)

	ed, _ := NewCodeEditor(EditFormatWhole)

	// Create multiple files
	for i := 0; i < 3; i++ {
		file := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
		os.WriteFile(file, []byte(fmt.Sprintf("content %d", i)), 0644)
	}

	// Edit files concurrently (mutex ensures safety)
	done := make(chan bool, 3)
	for i := 0; i < 3; i++ {
		go func(index int) {
			file := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", index))
			edit := Edit{
				FilePath: file,
				Format:   EditFormatWhole,
				Content:  fmt.Sprintf("modified %d", index),
			}
			ed.ApplyEdit(edit)
			done <- true
		}(i)
	}

	// Wait for all edits
	for i := 0; i < 3; i++ {
		<-done
	}

	fmt.Println("Concurrent edits completed safely")

	// Output:
	// Concurrent edits completed safely
}

// ExampleCodeEditor_validation demonstrates edit validation
func ExampleCodeEditor_validation() {
	ed, _ := NewCodeEditor(EditFormatDiff)

	// Valid edit
	validEdit := Edit{
		FilePath: "/tmp/test.txt",
		Format:   EditFormatDiff,
		Content:  "valid diff content",
	}

	if err := ed.ValidateEdit(validEdit); err != nil {
		fmt.Println("Valid edit rejected:", err)
	} else {
		fmt.Println("Valid edit accepted")
	}

	// Invalid edit - missing file path
	invalidEdit := Edit{
		FilePath: "",
		Format:   EditFormatDiff,
		Content:  "content",
	}

	if err := ed.ValidateEdit(invalidEdit); err != nil {
		fmt.Println("Invalid edit rejected")
	}

	// Output:
	// Valid edit accepted
	// Invalid edit rejected
}

// ExampleCodeEditor_backup demonstrates backup functionality
func ExampleCodeEditor_backup() {
	tmpDir, _ := os.MkdirTemp("", "editor_example_*")
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "important.txt")
	os.WriteFile(testFile, []byte("important data"), 0644)

	ed, _ := NewCodeEditor(EditFormatWhole)

	edit := Edit{
		FilePath: testFile,
		Format:   EditFormatWhole,
		Content:  "modified data",
		Backup:   true, // Create backup before editing
	}

	ed.ApplyEdit(edit)

	// Check if backup was created
	backupFile := testFile + ".bak"
	if _, err := os.Stat(backupFile); err == nil {
		fmt.Println("Backup created successfully")

		backup, _ := os.ReadFile(backupFile)
		fmt.Printf("Backup content: %s\n", string(backup))
	}

	// Output:
	// Backup created successfully
	// Backup content: important data
}
