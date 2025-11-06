package shell

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// SecurityError represents a security-related error
type SecurityError struct {
	Type    string
	Message string
	Command string
}

func (e *SecurityError) Error() string {
	return fmt.Sprintf("security error [%s]: %s (command: %s)", e.Type, e.Message, e.Command)
}

// AllowlistMode defines the allowlist behavior
type AllowlistMode int

const (
	AllowlistStrict   AllowlistMode = iota // Only exact matches allowed
	AllowlistPrefix                        // Prefix matches allowed
	AllowlistPattern                       // Pattern matches allowed
	AllowlistDisabled                      // Allowlist disabled (blocklist only)
)

func (m AllowlistMode) String() string {
	return [...]string{"Strict", "Prefix", "Pattern", "Disabled"}[m]
}

// AllowlistManager manages allowed commands
type AllowlistManager struct {
	exactCommands  map[string]bool
	prefixCommands []string
	patterns       []*regexp.Regexp
	mode           AllowlistMode
}

// NewAllowlistManager creates a new allowlist manager
func NewAllowlistManager(mode AllowlistMode, exact map[string]bool, prefixes []string, patterns []string) *AllowlistManager {
	am := &AllowlistManager{
		exactCommands:  make(map[string]bool),
		prefixCommands: prefixes,
		mode:           mode,
	}

	// Copy exact commands
	for cmd := range exact {
		am.exactCommands[cmd] = true
	}

	// Compile patterns
	for _, pattern := range patterns {
		if re, err := regexp.Compile(pattern); err == nil {
			am.patterns = append(am.patterns, re)
		}
	}

	return am
}

// IsAllowed checks if a command is allowed
func (am *AllowlistManager) IsAllowed(command string) bool {
	if am.mode == AllowlistDisabled {
		return true
	}

	// Extract base command (first word)
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}
	baseCmd := filepath.Base(parts[0])

	// Check exact matches
	if am.exactCommands[baseCmd] {
		return true
	}

	// Check prefix matches
	if am.mode >= AllowlistPrefix {
		for _, prefix := range am.prefixCommands {
			if strings.HasPrefix(baseCmd, prefix) {
				return true
			}
		}
	}

	// Check pattern matches
	if am.mode >= AllowlistPattern {
		for _, pattern := range am.patterns {
			if pattern.MatchString(command) {
				return true
			}
		}
	}

	return false
}

// BlocklistManager manages blocked commands
type BlocklistManager struct {
	exactCommands map[string]bool
	patterns      []*regexp.Regexp
}

// NewBlocklistManager creates a new blocklist manager
func NewBlocklistManager(exact map[string]bool, patterns []string) *BlocklistManager {
	bm := &BlocklistManager{
		exactCommands: make(map[string]bool),
	}

	// Copy exact commands
	for cmd := range exact {
		bm.exactCommands[cmd] = true
	}

	// Compile patterns
	for _, pattern := range patterns {
		if re, err := regexp.Compile(pattern); err == nil {
			bm.patterns = append(bm.patterns, re)
		}
	}

	return bm
}

// IsBlocked checks if a command is blocked
func (bm *BlocklistManager) IsBlocked(command string) bool {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}
	baseCmd := filepath.Base(parts[0])

	// Check exact matches
	if bm.exactCommands[baseCmd] {
		return true
	}

	// Check pattern matches
	for _, pattern := range bm.patterns {
		if pattern.MatchString(command) {
			return true
		}
	}

	return false
}

// SecurityManager manages command security
type SecurityManager struct {
	allowlist *AllowlistManager
	blocklist *BlocklistManager
}

// NewSecurityManager creates a new security manager
func NewSecurityManager(config *SecurityConfig) *SecurityManager {
	return &SecurityManager{
		allowlist: NewAllowlistManager(config.AllowlistMode, config.Allowlist, config.AllowlistPrefixes, config.AllowlistPatterns),
		blocklist: NewBlocklistManager(config.Blocklist, config.BlocklistPatterns),
	}
}

// ValidateCommand validates a command against security policies
func (sm *SecurityManager) ValidateCommand(cmd *Command) error {
	// Check blocklist first (fastest rejection)
	if sm.blocklist.IsBlocked(cmd.Command) {
		return &SecurityError{
			Type:    "blocked_command",
			Message: "command is blocked",
			Command: cmd.Command,
		}
	}

	// Check allowlist
	if !sm.allowlist.IsAllowed(cmd.Command) {
		return &SecurityError{
			Type:    "not_allowed",
			Message: "command is not in allowlist",
			Command: cmd.Command,
		}
	}

	// Check for dangerous patterns
	if sm.containsDangerousPatterns(cmd.Command) {
		return &SecurityError{
			Type:    "dangerous_pattern",
			Message: "command contains dangerous patterns",
			Command: cmd.Command,
		}
	}

	// Check arguments
	for _, arg := range cmd.Args {
		if sm.containsDangerousPatterns(arg) {
			return &SecurityError{
				Type:    "dangerous_argument",
				Message: "argument contains dangerous patterns",
				Command: cmd.Command,
			}
		}
	}

	// Validate working directory
	if cmd.WorkDir != "" {
		if !isValidPath(cmd.WorkDir) {
			return &SecurityError{
				Type:    "invalid_workdir",
				Message: "working directory path is invalid",
				Command: cmd.Command,
			}
		}
	}

	return nil
}

// containsDangerousPatterns checks for dangerous command patterns
func (sm *SecurityManager) containsDangerousPatterns(s string) bool {
	dangerousPatterns := []string{
		"rm -rf /",
		"rm -fr /",
		":(){ :|:& };:", // Fork bomb
		"> /dev/sda",
		"> /dev/sd",
		"mkfs",
		"dd if=/dev/zero",
		"dd if=/dev/random",
		"chmod -R 777 /",
		"chown -R",
		"> /dev/null; wget",
		"> /dev/null; curl",
	}

	lowerS := strings.ToLower(s)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerS, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// SanitizePath sanitizes a file path to prevent directory traversal
func SanitizePath(path string) string {
	// Clean the path
	cleaned := filepath.Clean(path)

	// Remove any .. components
	cleaned = strings.ReplaceAll(cleaned, "..", "")

	return cleaned
}

// SanitizeEnv sanitizes environment variables to prevent injection
func SanitizeEnv(env map[string]string) map[string]string {
	sanitized := make(map[string]string)
	for k, v := range env {
		// Only allow alphanumeric keys with underscores
		if isValidEnvKey(k) {
			// Remove potentially dangerous characters from values
			sanitized[k] = sanitizeEnvValue(v)
		}
	}
	return sanitized
}

// isValidEnvKey checks if an environment variable key is valid
func isValidEnvKey(key string) bool {
	if len(key) == 0 {
		return false
	}

	// First character must be letter or underscore
	if !((key[0] >= 'A' && key[0] <= 'Z') || (key[0] >= 'a' && key[0] <= 'z') || key[0] == '_') {
		return false
	}

	// Remaining characters must be alphanumeric or underscore
	for i := 1; i < len(key); i++ {
		c := key[i]
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}

	return true
}

// sanitizeEnvValue sanitizes an environment variable value
func sanitizeEnvValue(value string) string {
	// Remove null bytes
	value = strings.ReplaceAll(value, "\x00", "")

	// Remove command substitution attempts
	value = strings.ReplaceAll(value, "$(", "")
	value = strings.ReplaceAll(value, "`", "")

	return value
}

// isValidPath checks if a path is valid and safe
func isValidPath(path string) bool {
	// Path cannot be empty
	if path == "" {
		return false
	}

	// Path cannot contain null bytes
	if strings.Contains(path, "\x00") {
		return false
	}

	// Path cannot contain command substitution
	if strings.Contains(path, "$(") || strings.Contains(path, "`") {
		return false
	}

	// Clean and check for directory traversal
	cleaned := filepath.Clean(path)
	if strings.Contains(cleaned, "..") {
		return false
	}

	return true
}

// DefaultAllowlist returns a default allowlist configuration
func DefaultAllowlist() map[string]bool {
	return map[string]bool{
		"ls":      true,
		"cat":     true,
		"grep":    true,
		"find":    true,
		"git":     true,
		"npm":     true,
		"go":      true,
		"python":  true,
		"python3": true,
		"node":    true,
		"make":    true,
		"cargo":   true,
		"docker":  true,
		"kubectl": true,
		"echo":    true,
		"printf":  true,
		"pwd":     true,
		"which":   true,
		"whoami":  true,
		"date":    true,
		"head":    true,
		"tail":    true,
		"wc":      true,
		"sort":    true,
		"uniq":    true,
		"cut":     true,
		"sed":     true,
		"awk":     true,
		"tr":      true,
		"tee":     true,
		"xargs":   true,
		"env":     true,
		"export":  true,
		"cd":      true,
		"mkdir":   true,
		"touch":   true,
		"cp":      true,
		"mv":      true,
		"diff":    true,
		"patch":   true,
		"tar":     true,
		"gzip":    true,
		"gunzip":  true,
		"zip":     true,
		"unzip":   true,
		"curl":    true,
		"wget":    true,
		"ssh":     true,
		"scp":     true,
		"rsync":   true,
		"sleep":   true,
		"exit":    true,
		"test":    true,
		"true":    true,
		"false":   true,
		"for":     true,
		"while":   true,
		"if":      true,
		"sh":      true,
		"bash":    true,
	}
}

// DefaultBlocklist returns a default blocklist configuration
func DefaultBlocklist() map[string]bool {
	return map[string]bool{
		"rm":       true,
		"rmdir":    true,
		"dd":       true,
		"mkfs":     true,
		"fdisk":    true,
		"parted":   true,
		"shutdown": true,
		"reboot":   true,
		"halt":     true,
		"poweroff": true,
		"init":     true,
		"kill":     true,
		"killall":  true,
		"pkill":    true,
		"format":   true,
	}
}

// DefaultBlocklistPatterns returns default blocklist regex patterns
func DefaultBlocklistPatterns() []string {
	return []string{
		`rm\s+-r[f]?\s+/`,
		`>\s*/dev/sd[a-z]`,
		`:\(\)\s*\{`,                 // Fork bomb pattern
		`chmod\s+-R\s+777\s+/`,       // Dangerous permission change
		`chown\s+-R\s+.*\s+/`,        // Dangerous ownership change
		`dd\s+if=/dev/(zero|random)`, // Disk wiping
		`mkfs\.`,                     // Filesystem creation
	}
}
