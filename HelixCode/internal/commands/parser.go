package commands

import (
	"regexp"
	"strings"
)

// Parser parses slash commands from user input
type Parser struct {
	commandRegex *regexp.Regexp
}

// NewParser creates a new command parser
func NewParser() *Parser {
	return &Parser{
		// Matches /command or /command args
		commandRegex: regexp.MustCompile(`^/([a-zA-Z0-9_-]+)(?:\s+(.*))?$`),
	}
}

// Parse parses a command from input
func (p *Parser) Parse(input string) (commandName string, args []string, flags map[string]string, isCommand bool) {
	input = strings.TrimSpace(input)

	// Check if it's a command
	if !strings.HasPrefix(input, "/") {
		return "", nil, nil, false
	}

	// Match command pattern
	matches := p.commandRegex.FindStringSubmatch(input)
	if len(matches) < 2 {
		return "", nil, nil, false
	}

	commandName = matches[1]
	args = make([]string, 0)
	flags = make(map[string]string)

	// Parse arguments if present
	if len(matches) > 2 && matches[2] != "" {
		argsStr := matches[2]
		args, flags = p.parseArgs(argsStr)
	}

	return commandName, args, flags, true
}

// parseArgs parses command arguments and flags
func (p *Parser) parseArgs(argsStr string) ([]string, map[string]string) {
	args := make([]string, 0)
	flags := make(map[string]string)

	// Split by spaces, respecting quotes
	parts := p.splitRespectingQuotes(argsStr)

	for i := 0; i < len(parts); i++ {
		part := strings.TrimSpace(parts[i])
		if part == "" {
			continue
		}

		// Check if it's a flag (--key=value or --key value)
		if strings.HasPrefix(part, "--") {
			// Check if flag has = separator
			if strings.Contains(part, "=") {
				key, value := p.parseFlag(part)
				flags[key] = value
			} else {
				// Flag without =, check next part for value
				key := strings.TrimPrefix(part, "--")
				// Look ahead to see if next part is a value (not another flag)
				if i+1 < len(parts) && !strings.HasPrefix(parts[i+1], "--") {
					flags[key] = parts[i+1]
					i++ // Skip next part as we consumed it as a value
				} else {
					// Boolean flag
					flags[key] = "true"
				}
			}
		} else {
			// Regular argument
			args = append(args, part)
		}
	}

	return args, flags
}

// parseFlag parses a flag (--key=value or --key)
func (p *Parser) parseFlag(flag string) (string, string) {
	flag = strings.TrimPrefix(flag, "--")

	// Check for = separator
	if strings.Contains(flag, "=") {
		parts := strings.SplitN(flag, "=", 2)
		return parts[0], parts[1]
	}

	// Flag without value (treat as boolean true)
	return flag, "true"
}

// splitRespectingQuotes splits a string by spaces but respects quotes
func (p *Parser) splitRespectingQuotes(s string) []string {
	var result []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)

	for _, char := range s {
		switch {
		case char == '"' || char == '\'':
			if !inQuotes {
				inQuotes = true
				quoteChar = char
			} else if char == quoteChar {
				inQuotes = false
				quoteChar = 0
			} else {
				current.WriteRune(char)
			}
		case char == ' ' && !inQuotes:
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// IsCommand checks if the input is a command
func (p *Parser) IsCommand(input string) bool {
	input = strings.TrimSpace(input)
	return strings.HasPrefix(input, "/")
}

// ExtractCommandName extracts just the command name from input
func (p *Parser) ExtractCommandName(input string) string {
	commandName, _, _, isCommand := p.Parse(input)
	if !isCommand {
		return ""
	}
	return commandName
}
