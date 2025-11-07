package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// TestCase represents a single test case
type TestCase struct {
	ID          string
	Name        string
	Package     string
	File        string
	Description string
	Steps       []string
	Category    string
	LineNumber  int
}

func main() {
	fmt.Println("🔍 Generating Test Catalog...")

	testCases := []TestCase{}

	// Scan all test directories
	testDirs := []string{
		"./internal",
		"./test",
		"./tests",
		"./cmd",
		"./benchmarks",
		"./applications",
	}

	for _, dir := range testDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if !info.IsDir() && strings.HasSuffix(path, "_test.go") {
				cases := parseTestFile(path)
				testCases = append(testCases, cases...)
			}

			return nil
		})

		if err != nil {
			fmt.Printf("Error walking %s: %v\n", dir, err)
		}
	}

	// Sort test cases by category and name
	sort.Slice(testCases, func(i, j int) bool {
		if testCases[i].Category == testCases[j].Category {
			return testCases[i].Name < testCases[j].Name
		}
		return testCases[i].Category < testCases[j].Category
	})

	// Generate catalog
	err := generateCatalog(testCases)
	if err != nil {
		fmt.Printf("Error generating catalog: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Generated catalog with %d test cases\n", len(testCases))
}

func parseTestFile(path string) []TestCase {
	cases := []TestCase{}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return cases
	}

	category := determineCategory(path)
	pkg := node.Name.Name

	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil {
			continue
		}

		name := fn.Name.Name
		if !strings.HasPrefix(name, "Test") && !strings.HasPrefix(name, "Benchmark") {
			continue
		}

		// Extract description from comments
		description := extractDescription(fn.Doc)

		// Extract test steps
		steps := extractSteps(fn.Body)

		testCase := TestCase{
			ID:          generateID(path, name),
			Name:        name,
			Package:     pkg,
			File:        path,
			Description: description,
			Steps:       steps,
			Category:    category,
			LineNumber:  fset.Position(fn.Pos()).Line,
		}

		cases = append(cases, testCase)
	}

	return cases
}

func determineCategory(path string) string {
	// Normalize path separators and check without leading slash
	normalizedPath := filepath.ToSlash(path)

	if strings.Contains(normalizedPath, "internal/") {
		return "Unit Test"
	} else if strings.Contains(normalizedPath, "test/integration/") {
		return "Integration Test"
	} else if strings.Contains(normalizedPath, "test/e2e/") || strings.Contains(normalizedPath, "tests/e2e/") {
		return "E2E Test"
	} else if strings.Contains(normalizedPath, "test/automation/") {
		return "Automation Test"
	} else if strings.Contains(normalizedPath, "test/load/") {
		return "Load Test"
	} else if strings.Contains(normalizedPath, "benchmarks/") {
		return "Benchmark"
	} else if strings.Contains(normalizedPath, "applications/") {
		return "Application Test"
	} else if strings.Contains(normalizedPath, "cmd/") {
		return "Command Test"
	}
	return "Other Test"
}

func generateID(path, name string) string {
	// Generate unique ID based on path and name
	normalizedPath := filepath.ToSlash(path)
	pathParts := strings.Split(normalizedPath, "/")
	var prefix string

	if strings.Contains(normalizedPath, "internal/") {
		prefix = "UT"
	} else if strings.Contains(normalizedPath, "integration/") {
		prefix = "IT"
	} else if strings.Contains(normalizedPath, "e2e/") {
		prefix = "E2E"
	} else if strings.Contains(normalizedPath, "automation/") {
		prefix = "AT"
	} else if strings.Contains(normalizedPath, "load/") {
		prefix = "LT"
	} else if strings.Contains(normalizedPath, "benchmark") {
		prefix = "BM"
	} else {
		prefix = "OT"
	}

	// Use package name as part of ID
	pkgName := ""
	for i := len(pathParts) - 1; i >= 0; i-- {
		if !strings.HasSuffix(pathParts[i], "_test.go") {
			pkgName = pathParts[i]
			break
		}
	}

	return fmt.Sprintf("%s-%s-%s", prefix, strings.ToUpper(pkgName[:min(3, len(pkgName))]), name)
}

func extractDescription(commentGroup *ast.CommentGroup) string {
	if commentGroup == nil {
		return "No description provided"
	}

	var lines []string
	for _, comment := range commentGroup.List {
		text := strings.TrimPrefix(comment.Text, "//")
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSuffix(text, "*/")
		text = strings.TrimSpace(text)
		if text != "" {
			lines = append(lines, text)
		}
	}

	if len(lines) > 0 {
		return strings.Join(lines, " ")
	}

	return "Tests functionality"
}

func extractSteps(body *ast.BlockStmt) []string {
	steps := []string{}

	if body == nil {
		return steps
	}

	// Extract key operations from function body
	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
				step := fmt.Sprintf("Call %s", sel.Sel.Name)
				if len(steps) < 10 { // Limit to 10 steps
					steps = append(steps, step)
				}
			}
		}
		return true
	})

	if len(steps) == 0 {
		steps = append(steps, "Execute test logic")
	}

	return steps
}

func generateCatalog(testCases []TestCase) error {
	// Create output directory
	err := os.MkdirAll("Documentation/Testing", 0755)
	if err != nil {
		return err
	}

	file, err := os.Create("Documentation/Testing/Tests_Catalog.md")
	if err != nil {
		return err
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "# HelixCode Test Catalog\n\n")
	fmt.Fprintf(file, "**Generated**: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "**Total Tests**: %d\n\n", len(testCases))
	fmt.Fprintf(file, "---\n\n")

	// Write summary
	fmt.Fprintf(file, "## Test Summary by Category\n\n")
	categoryCounts := make(map[string]int)
	for _, tc := range testCases {
		categoryCounts[tc.Category]++
	}

	fmt.Fprintf(file, "| Category | Count |\n")
	fmt.Fprintf(file, "|----------|-------|\n")
	for cat, count := range categoryCounts {
		fmt.Fprintf(file, "| %s | %d |\n", cat, count)
	}
	fmt.Fprintf(file, "\n---\n\n")

	// Write test cases by category
	currentCategory := ""
	for _, tc := range testCases {
		if tc.Category != currentCategory {
			currentCategory = tc.Category
			fmt.Fprintf(file, "## %s\n\n", currentCategory)
		}

		fmt.Fprintf(file, "### %s\n\n", tc.Name)
		fmt.Fprintf(file, "**ID**: `%s`\n\n", tc.ID)
		fmt.Fprintf(file, "**Package**: `%s`\n\n", tc.Package)
		fmt.Fprintf(file, "**File**: `%s:%d`\n\n", tc.File, tc.LineNumber)
		fmt.Fprintf(file, "**Description**: %s\n\n", tc.Description)

		if len(tc.Steps) > 0 {
			fmt.Fprintf(file, "**Test Steps**:\n")
			for i, step := range tc.Steps {
				if i < 5 { // Limit displayed steps
					fmt.Fprintf(file, "%d. %s\n", i+1, step)
				}
			}
			if len(tc.Steps) > 5 {
				fmt.Fprintf(file, "   ... and %d more steps\n", len(tc.Steps)-5)
			}
			fmt.Fprintf(file, "\n")
		}

		fmt.Fprintf(file, "---\n\n")
	}

	// Write footer
	fmt.Fprintf(file, "## Usage\n\n")
	fmt.Fprintf(file, "To run a specific test:\n\n")
	fmt.Fprintf(file, "```bash\n")
	fmt.Fprintf(file, "# Run by package\n")
	fmt.Fprintf(file, "go test -v ./internal/package_name\n\n")
	fmt.Fprintf(file, "# Run specific test\n")
	fmt.Fprintf(file, "go test -v ./path/to/package -run TestName\n\n")
	fmt.Fprintf(file, "# Run all tests\n")
	fmt.Fprintf(file, "./run_all_tests.sh\n")
	fmt.Fprintf(file, "```\n\n")

	fmt.Fprintf(file, "## Automatic Updates\n\n")
	fmt.Fprintf(file, "This catalog is automatically generated by running:\n\n")
	fmt.Fprintf(file, "```bash\n")
	fmt.Fprintf(file, "go run scripts/generate-test-catalog.go\n")
	fmt.Fprintf(file, "```\n\n")
	fmt.Fprintf(file, "The catalog should be regenerated whenever new tests are added.\n\n")

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
