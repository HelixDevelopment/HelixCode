package web

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// Parser converts HTML to markdown
type Parser struct {
	config *Config
}

// ParsedContent contains parsed content
type ParsedContent struct {
	Markdown string
	Metadata Metadata
	BaseURL  string
}

// Metadata contains document metadata
type Metadata struct {
	Title       string
	Description string
	Author      string
	Published   time.Time
	Modified    time.Time
	Keywords    []string
	Image       string
	Language    string
}

// NewParser creates a new parser
func NewParser(config *Config) *Parser {
	return &Parser{
		config: config,
	}
}

// Parse converts HTML to markdown
func (p *Parser) Parse(htmlContent []byte, baseURL string) (*ParsedContent, error) {
	// Parse HTML
	doc, err := html.Parse(bytes.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	// Extract metadata
	metadata := p.extractMetadata(doc)

	// Clean document
	if p.config.RemoveScripts {
		p.removeElements(doc, "script")
	}
	if p.config.RemoveStyles {
		p.removeElements(doc, "style")
	}
	if p.config.RemoveNavigation {
		p.removeElements(doc, "nav")
		p.removeElements(doc, "header")
		p.removeElements(doc, "footer")
		p.removeElements(doc, "aside")
	}

	// Convert to markdown
	markdown := p.convertToMarkdown(doc)

	// Clean markdown
	markdown = p.cleanMarkdown(markdown)

	return &ParsedContent{
		Markdown: markdown,
		Metadata: metadata,
		BaseURL:  baseURL,
	}, nil
}

// extractMetadata extracts metadata from HTML
func (p *Parser) extractMetadata(doc *html.Node) Metadata {
	metadata := Metadata{
		Keywords: []string{},
	}

	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				if n.FirstChild != nil {
					metadata.Title = n.FirstChild.Data
				}
			case "meta":
				p.extractMetaTag(n, &metadata)
			case "html":
				for _, attr := range n.Attr {
					if attr.Key == "lang" {
						metadata.Language = attr.Val
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}

	if p.config.ExtractMetadata {
		extract(doc)
	}

	return metadata
}

// extractMetaTag extracts information from meta tags
func (p *Parser) extractMetaTag(n *html.Node, metadata *Metadata) {
	var name, property, content string

	for _, attr := range n.Attr {
		switch attr.Key {
		case "name":
			name = attr.Val
		case "property":
			property = attr.Val
		case "content":
			content = attr.Val
		}
	}

	switch name {
	case "description":
		metadata.Description = content
	case "author":
		metadata.Author = content
	case "keywords":
		metadata.Keywords = strings.Split(content, ",")
		for i := range metadata.Keywords {
			metadata.Keywords[i] = strings.TrimSpace(metadata.Keywords[i])
		}
	}

	switch property {
	case "og:title":
		if metadata.Title == "" {
			metadata.Title = content
		}
	case "og:description":
		if metadata.Description == "" {
			metadata.Description = content
		}
	case "og:image":
		metadata.Image = content
	}
}

// removeElements removes all elements with the given tag name
func (p *Parser) removeElements(n *html.Node, tagName string) {
	var toRemove []*html.Node
	var find func(*html.Node)

	find = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == tagName {
			toRemove = append(toRemove, node)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}

	find(n)

	for _, node := range toRemove {
		if node.Parent != nil {
			node.Parent.RemoveChild(node)
		}
	}
}

// convertToMarkdown converts HTML node to markdown
func (p *Parser) convertToMarkdown(n *html.Node) string {
	var buf bytes.Buffer
	p.nodeToMarkdown(n, &buf, 0)
	return buf.String()
}

// nodeToMarkdown recursively converts HTML nodes to markdown
func (p *Parser) nodeToMarkdown(n *html.Node, buf *bytes.Buffer, depth int) {
	switch n.Type {
	case html.TextNode:
		text := strings.TrimSpace(n.Data)
		if text != "" {
			buf.WriteString(text)
			buf.WriteString(" ")
		}

	case html.ElementNode:
		switch n.Data {
		case "h1":
			buf.WriteString("\n# ")
			p.writeChildren(n, buf, depth)
			buf.WriteString("\n\n")
		case "h2":
			buf.WriteString("\n## ")
			p.writeChildren(n, buf, depth)
			buf.WriteString("\n\n")
		case "h3":
			buf.WriteString("\n### ")
			p.writeChildren(n, buf, depth)
			buf.WriteString("\n\n")
		case "h4":
			buf.WriteString("\n#### ")
			p.writeChildren(n, buf, depth)
			buf.WriteString("\n\n")
		case "h5":
			buf.WriteString("\n##### ")
			p.writeChildren(n, buf, depth)
			buf.WriteString("\n\n")
		case "h6":
			buf.WriteString("\n###### ")
			p.writeChildren(n, buf, depth)
			buf.WriteString("\n\n")
		case "p":
			buf.WriteString("\n")
			p.writeChildren(n, buf, depth)
			buf.WriteString("\n\n")
		case "br":
			buf.WriteString("\n")
		case "strong", "b":
			text := p.getChildText(n)
			if text != "" {
				buf.WriteString("**")
				buf.WriteString(strings.TrimSpace(text))
				buf.WriteString("**")
			}
		case "em", "i":
			buf.WriteString("*")
			p.writeChildren(n, buf, depth)
			buf.WriteString("*")
		case "code":
			buf.WriteString("`")
			p.writeChildren(n, buf, depth)
			buf.WriteString("`")
		case "pre":
			buf.WriteString("\n```\n")
			p.writeChildren(n, buf, depth)
			buf.WriteString("\n```\n\n")
		case "a":
			href := ""
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					href = attr.Val
					break
				}
			}
			text := p.getChildText(n)
			if text != "" && href != "" {
				buf.WriteString("[")
				buf.WriteString(strings.TrimSpace(text))
				buf.WriteString("](")
				buf.WriteString(href)
				buf.WriteString(")")
			} else {
				p.writeChildren(n, buf, depth)
			}
		case "img":
			alt := ""
			src := ""
			for _, attr := range n.Attr {
				if attr.Key == "alt" {
					alt = attr.Val
				} else if attr.Key == "src" {
					src = attr.Val
				}
			}
			buf.WriteString("![")
			buf.WriteString(alt)
			buf.WriteString("](")
			buf.WriteString(src)
			buf.WriteString(")")
		case "ul":
			buf.WriteString("\n")
			p.writeChildren(n, buf, depth)
			buf.WriteString("\n")
		case "ol":
			buf.WriteString("\n")
			p.writeChildren(n, buf, depth)
			buf.WriteString("\n")
		case "li":
			buf.WriteString(strings.Repeat("  ", depth))
			buf.WriteString("- ")
			p.writeChildren(n, buf, depth+1)
			buf.WriteString("\n")
		case "blockquote":
			buf.WriteString("\n> ")
			p.writeChildren(n, buf, depth)
			buf.WriteString("\n\n")
		case "hr":
			buf.WriteString("\n---\n\n")
		case "table":
			buf.WriteString("\n")
			p.writeChildren(n, buf, depth)
			buf.WriteString("\n")
		case "tr":
			buf.WriteString("| ")
			p.writeChildren(n, buf, depth)
			buf.WriteString("\n")
		case "td", "th":
			p.writeChildren(n, buf, depth)
			buf.WriteString(" | ")
		default:
			p.writeChildren(n, buf, depth)
		}

	default:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			p.nodeToMarkdown(c, buf, depth)
		}
	}
}

// writeChildren writes all children of a node
func (p *Parser) writeChildren(n *html.Node, buf *bytes.Buffer, depth int) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		p.nodeToMarkdown(c, buf, depth)
	}
}

// getChildText extracts text from child nodes
func (p *Parser) getChildText(n *html.Node) string {
	var buf bytes.Buffer
	var extract func(*html.Node)
	extract = func(node *html.Node) {
		if node.Type == html.TextNode {
			buf.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(n)
	return buf.String()
}

// cleanMarkdown cleans up markdown output
func (p *Parser) cleanMarkdown(markdown string) string {
	// Remove excessive blank lines
	markdown = regexp.MustCompile(`\n{3,}`).ReplaceAllString(markdown, "\n\n")

	// Trim whitespace from lines
	lines := strings.Split(markdown, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	markdown = strings.Join(lines, "\n")

	// Remove excessive spaces
	markdown = regexp.MustCompile(` {2,}`).ReplaceAllString(markdown, " ")

	return strings.TrimSpace(markdown)
}

// ExtractText extracts plain text from HTML
func (p *Parser) ExtractText(htmlContent []byte) (string, error) {
	doc, err := html.Parse(bytes.NewReader(htmlContent))
	if err != nil {
		return "", fmt.Errorf("parse html: %w", err)
	}

	var buf bytes.Buffer
	var extract func(*html.Node)

	extract = func(n *html.Node) {
		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				buf.WriteString(text)
				buf.WriteString(" ")
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}

	extract(doc)
	return strings.TrimSpace(buf.String()), nil
}
