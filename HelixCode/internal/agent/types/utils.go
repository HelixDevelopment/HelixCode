package types

// countLines counts the number of lines in a string
func countLines(code string) int {
	if code == "" {
		return 0
	}
	lines := 1
	for _, c := range code {
		if c == '\n' {
			lines++
		}
	}
	return lines
}
