package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Config holds the converter configuration
type Config struct {
	InputFile  string
	OutputFile string
	Title      string
	Theme      string
}

// TOCEntry represents a table of contents entry
type TOCEntry struct {
	ID    string
	Title string
	Level int
}

// HTMLTemplate holds the data for HTML generation
type HTMLTemplate struct {
	Title     string
	Content   string
	TOC       []TOCEntry
	Timestamp string
	Version   string
	CSSInline bool
	JSInline  bool
}

const (
	cssTemplate = `/* HelixCode Manual Styles */
:root {
    --primary-500: #0ea5e9;
    --primary-600: #0284c7;
    --accent-1: #10b981;
    --accent-2: #f59e0b;
    --accent-3: #ef4444;
    --bg-primary: #ffffff;
    --bg-secondary: #f8fafc;
    --bg-tertiary: #f1f5f9;
    --text-primary: #0f172a;
    --text-secondary: #475569;
    --border-light: #e2e8f0;
}

[data-theme="dark"] {
    --bg-primary: #0f172a;
    --bg-secondary: #1e293b;
    --bg-tertiary: #334155;
    --text-primary: #f8fafc;
    --text-secondary: #cbd5e1;
    --border-light: #334155;
}

body {
    font-family: 'Inter', -apple-system, sans-serif;
    line-height: 1.7;
    color: var(--text-primary);
    background: var(--bg-primary);
    margin: 0;
    padding: 0;
}

.container { max-width: 900px; margin: 2rem auto; padding: 0 2rem; }
h1, h2, h3 { margin-top: 2rem; color: var(--text-primary); }
code { background: var(--bg-tertiary); padding: 0.2rem 0.4rem; border-radius: 4px; font-size: 0.875em; }
pre { background: var(--bg-tertiary); padding: 1rem; border-radius: 8px; overflow-x: auto; }
pre code { background: none; padding: 0; }
a { color: var(--primary-600); text-decoration: none; }
a:hover { text-decoration: underline; }
table { width: 100%; border-collapse: collapse; margin: 1rem 0; }
th, td { padding: 0.75rem; border: 1px solid var(--border-light); text-align: left; }
th { background: var(--bg-tertiary); font-weight: 600; }`
)

func main() {
	var config Config

	flag.StringVar(&config.InputFile, "input", "", "Input Markdown file (required)")
	flag.StringVar(&config.OutputFile, "output", "", "Output HTML file (required)")
	flag.StringVar(&config.Title, "title", "HelixCode User Manual", "HTML document title")
	flag.StringVar(&config.Theme, "theme", "light", "Theme (light/dark)")
	flag.Parse()

	if config.InputFile == "" || config.OutputFile == "" {
		fmt.Println("Usage: md-to-html -input=README.md -output=manual.html")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if err := convertMarkdownToHTML(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Converted %s to %s\n", config.InputFile, config.OutputFile)
}

func convertMarkdownToHTML(config Config) error {
	// Read input file
	content, err := os.ReadFile(config.InputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Parse Markdown and generate HTML
	htmlContent, toc := parseMarkdown(string(content))

	// Generate full HTML document
	html := generateHTML(HTMLTemplate{
		Title:     config.Title,
		Content:   htmlContent,
		TOC:       toc,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Version:   "1.0",
	})

	// Write output file
	if err := os.WriteFile(config.OutputFile, []byte(html), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

func parseMarkdown(md string) (string, []TOCEntry) {
	var buf bytes.Buffer
	var toc []TOCEntry
	scanner := bufio.NewScanner(strings.NewReader(md))

	inCodeBlock := false
	codeLanguage := ""

	for scanner.Scan() {
		line := scanner.Text()

		// Handle code blocks
		if strings.HasPrefix(line, "```") {
			if !inCodeBlock {
				inCodeBlock = true
				codeLanguage = strings.TrimPrefix(line, "```")
				buf.WriteString(fmt.Sprintf("<pre><code class=\"language-%s\">", codeLanguage))
			} else {
				inCodeBlock = false
				buf.WriteString("</code></pre>\n")
			}
			continue
		}

		if inCodeBlock {
			buf.WriteString(template.HTMLEscapeString(line))
			buf.WriteString("\n")
			continue
		}

		// Parse headings
		if strings.HasPrefix(line, "#") {
			level := 0
			for _, c := range line {
				if c == '#' {
					level++
				} else {
					break
				}
			}

			if level > 0 && level <= 6 {
				title := strings.TrimSpace(strings.TrimPrefix(line, strings.Repeat("#", level)))
				id := generateID(title)

				// Add to TOC
				if level <= 3 {
					toc = append(toc, TOCEntry{
						ID:    id,
						Title: title,
						Level: level,
					})
				}

				buf.WriteString(fmt.Sprintf("<h%d id=\"%s\"><span class=\"anchor\">#</span>%s</h%d>\n",
					level, id, template.HTMLEscapeString(title), level))
				continue
			}
		}

		// Parse lists
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			item := strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* ")
			buf.WriteString("<li>" + parseInline(item) + "</li>\n")
			continue
		}

		// Parse numbered lists
		if matched, _ := regexp.MatchString(`^\d+\. `, line); matched {
			re := regexp.MustCompile(`^\d+\. `)
			item := re.ReplaceAllString(line, "")
			buf.WriteString("<li>" + parseInline(item) + "</li>\n")
			continue
		}

		// Parse paragraphs
		if line != "" {
			buf.WriteString("<p>" + parseInline(line) + "</p>\n")
		} else {
			buf.WriteString("\n")
		}
	}

	return buf.String(), toc
}

func parseInline(text string) string {
	// Parse bold
	boldRe := regexp.MustCompile(`\*\*([^*]+)\*\*`)
	text = boldRe.ReplaceAllString(text, "<strong>$1</strong>")

	// Parse italic
	italicRe := regexp.MustCompile(`\*([^*]+)\*`)
	text = italicRe.ReplaceAllString(text, "<em>$1</em>")

	// Parse inline code
	codeRe := regexp.MustCompile("`([^`]+)`")
	text = codeRe.ReplaceAllString(text, "<code>$1</code>")

	// Parse links
	linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	text = linkRe.ReplaceAllString(text, `<a href="$2">$1</a>`)

	return text
}

func generateID(title string) string {
	// Convert title to URL-safe ID
	id := strings.ToLower(title)
	id = regexp.MustCompile(`[^a-z0-9\s-]`).ReplaceAllString(id, "")
	id = regexp.MustCompile(`\s+`).ReplaceAllString(id, "-")
	return id
}

func generateHTML(data HTMLTemplate) string {
	var buf bytes.Buffer

	// Write HTML header
	buf.WriteString(`<!DOCTYPE html>
<html lang="en" data-theme="auto">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + data.Title + `</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&family=JetBrains+Mono:wght@400;500;600&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/github-dark.min.css">
    <style>` + cssTemplate + `</style>
</head>
<body>
    <div class="header">
        <div class="header-content">
            <h1>` + data.Title + `</h1>
            <button class="theme-toggle" id="themeToggle">🌙</button>
        </div>
    </div>
`)

	// Write TOC
	if len(data.TOC) > 0 {
		buf.WriteString(`    <nav class="toc">
        <h2>Table of Contents</h2>
        <ul>
`)
		for _, entry := range data.TOC {
			indent := strings.Repeat("    ", entry.Level-1)
			buf.WriteString(fmt.Sprintf(`%s<li><a href="#%s">%s</a></li>
`, indent, entry.ID, entry.Title))
		}
		buf.WriteString(`        </ul>
    </nav>
`)
	}

	// Write main content
	buf.WriteString(`    <main class="container">
` + data.Content + `
        <footer>
            <p><em>Generated on: ` + data.Timestamp + `</em></p>
        </footer>
    </main>

    <script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/highlight.min.js"></script>
    <script>
        hljs.highlightAll();

        // Theme toggle
        const themeToggle = document.getElementById('themeToggle');
        const html = document.documentElement;

        themeToggle.addEventListener('click', () => {
            const current = html.getAttribute('data-theme');
            const newTheme = current === 'dark' ? 'light' : 'dark';
            html.setAttribute('data-theme', newTheme);
            themeToggle.textContent = newTheme === 'dark' ? '☀️' : '🌙';
        });
    </script>
</body>
</html>`)

	return buf.String()
}

// Helper function to read file content
func readFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Helper function to ensure directory exists
func ensureDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
}
