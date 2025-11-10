package mentions

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// MentionParser parses and resolves @ mentions in user input
type MentionParser struct {
	handlers     map[MentionType]MentionHandler
	mentionRegex *regexp.Regexp
}

// NewMentionParser creates a new mention parser
func NewMentionParser() *MentionParser {
	return &MentionParser{
		handlers:     make(map[MentionType]MentionHandler),
		mentionRegex: regexp.MustCompile(`@([a-zA-Z0-9-_]+)(?:\[([^\]]*)\])?(?:\(([^\)]*)\))?`),
	}
}

// RegisterHandler registers a mention handler
func (mp *MentionParser) RegisterHandler(handler MentionHandler) {
	mp.handlers[handler.Type()] = handler
}

// Parse extracts all mentions from the input text
func (mp *MentionParser) Parse(text string) []string {
	matches := mp.mentionRegex.FindAllString(text, -1)
	return matches
}

// ParseAndResolve parses and resolves all mentions in the input text
func (mp *MentionParser) ParseAndResolve(ctx context.Context, text string) (*MentionResult, error) {
	result := &MentionResult{
		OriginalText:  text,
		ProcessedText: text,
		Contexts:      make([]MentionContext, 0),
		TotalTokens:   0,
	}

	// Find all mentions
	matches := mp.mentionRegex.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return result, nil
	}

	// Process each mention
	for _, match := range matches {
		fullMatch := match[0]
		_ = match[1] // mentionType - determined by handler.CanHandle
		target := ""
		options := make(map[string]string)

		// Extract target and options
		if len(match) > 2 && match[2] != "" {
			target = match[2]
		}
		if len(match) > 3 && match[3] != "" {
			// Parse options from parentheses
			opts := strings.Split(match[3], ",")
			for _, opt := range opts {
				parts := strings.SplitN(strings.TrimSpace(opt), "=", 2)
				if len(parts) == 2 {
					options[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
				}
			}
		}

		// Find appropriate handler
		var handler MentionHandler
		for _, h := range mp.handlers {
			if h.CanHandle(fullMatch) {
				handler = h
				break
			}
		}

		if handler == nil {
			continue // Skip unknown mentions
		}

		// Resolve the mention
		mentionCtx, err := handler.Resolve(ctx, target, options)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve mention %s: %w", fullMatch, err)
		}

		// Add to results
		result.Contexts = append(result.Contexts, *mentionCtx)
		result.TotalTokens += mentionCtx.TokenCount

		// Replace mention with resolved content in processed text
		replacement := fmt.Sprintf("\n\n--- %s: %s ---\n%s\n--- End of %s ---\n\n",
			mentionCtx.Type, mentionCtx.Target, mentionCtx.Content, mentionCtx.Type)
		result.ProcessedText = strings.Replace(result.ProcessedText, fullMatch, replacement, 1)
	}

	return result, nil
}

// ExtractMentionInfo extracts information from a mention string
func (mp *MentionParser) ExtractMentionInfo(mention string) (mentionType string, target string, options map[string]string) {
	matches := mp.mentionRegex.FindStringSubmatch(mention)
	if len(matches) == 0 {
		return "", "", nil
	}

	mentionType = matches[1]
	options = make(map[string]string)

	if len(matches) > 2 && matches[2] != "" {
		target = matches[2]
	}

	if len(matches) > 3 && matches[3] != "" {
		opts := strings.Split(matches[3], ",")
		for _, opt := range opts {
			parts := strings.SplitN(strings.TrimSpace(opt), "=", 2)
			if len(parts) == 2 {
				options[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	return mentionType, target, options
}
