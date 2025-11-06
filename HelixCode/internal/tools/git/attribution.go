package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// AttributionManager handles commit attribution
type AttributionManager struct {
	config *AttributionConfig
}

// AttributionConfig configures attribution
type AttributionConfig struct {
	EnableCoAuthors   bool   `yaml:"co_authors"`
	ClaudeAttribution bool   `yaml:"claude_attribution"`
	ClaudeName        string `yaml:"claude_name"`
	ClaudeEmail       string `yaml:"claude_email"`
	AutoSignOff       bool   `yaml:"auto_sign_off"`
}

// Attribution represents a commit attribution
type Attribution struct {
	Type  AttributionType
	Name  string
	Email string
}

// AttributionType specifies attribution type
type AttributionType int

const (
	AttributionCoAuthor AttributionType = iota
	AttributionSignedOff
	AttributionReviewed
	AttributionTested
)

// NewAttributionManager creates a new attribution manager
func NewAttributionManager() *AttributionManager {
	return &AttributionManager{
		config: &AttributionConfig{
			EnableCoAuthors:   true,
			ClaudeAttribution: true,
			ClaudeName:        "Claude",
			ClaudeEmail:       "noreply@anthropic.com",
			AutoSignOff:       false,
		},
	}
}

// AddAttribution adds attribution to commit message
func (am *AttributionManager) AddAttribution(message string, attrs []Attribution) string {
	if !am.config.EnableCoAuthors {
		return message
	}

	var footer strings.Builder

	// Extract existing footer
	parts := strings.Split(message, "\n\n")
	body := message
	existingFooter := ""
	if len(parts) > 1 {
		body = strings.Join(parts[:len(parts)-1], "\n\n")
		existingFooter = parts[len(parts)-1]
	}

	// Add attributions
	for _, attr := range attrs {
		switch attr.Type {
		case AttributionCoAuthor:
			footer.WriteString(fmt.Sprintf("Co-authored-by: %s <%s>\n", attr.Name, attr.Email))
		case AttributionSignedOff:
			footer.WriteString(fmt.Sprintf("Signed-off-by: %s <%s>\n", attr.Name, attr.Email))
		case AttributionReviewed:
			footer.WriteString(fmt.Sprintf("Reviewed-by: %s <%s>\n", attr.Name, attr.Email))
		case AttributionTested:
			footer.WriteString(fmt.Sprintf("Tested-by: %s <%s>\n", attr.Name, attr.Email))
		}
	}

	// Combine
	result := body
	if existingFooter != "" && !strings.Contains(footer.String(), existingFooter) {
		result += "\n\n" + existingFooter
	}
	if footer.Len() > 0 {
		result += "\n\n" + strings.TrimSpace(footer.String())
	}

	return result
}

// GetClaudeAttribution returns Claude attribution
func (am *AttributionManager) GetClaudeAttribution() Attribution {
	return Attribution{
		Type:  AttributionCoAuthor,
		Name:  am.config.ClaudeName,
		Email: am.config.ClaudeEmail,
	}
}

// GetDefaultAttributions returns default attributions including Claude
func (am *AttributionManager) GetDefaultAttributions() []Attribution {
	attrs := []Attribution{}

	if am.config.ClaudeAttribution {
		attrs = append(attrs, am.GetClaudeAttribution())
	}

	return attrs
}

// AmendDetector detects when it's safe to amend
type AmendDetector struct {
	repoPath string
}

// NewAmendDetector creates a new amend detector
func NewAmendDetector(repoPath string) *AmendDetector {
	return &AmendDetector{
		repoPath: repoPath,
	}
}

// CanAmend checks if it's safe to amend the last commit
func (ad *AmendDetector) CanAmend(ctx context.Context) (bool, string) {
	// Check if commit is pushed
	if pushed, err := ad.isCommitPushed(ctx); err != nil {
		return false, fmt.Sprintf("error checking push status: %v", err)
	} else if pushed {
		return false, "commit already pushed"
	}

	// Check authorship
	if foreign, err := ad.isForeignCommit(ctx); err != nil {
		return false, fmt.Sprintf("error checking authorship: %v", err)
	} else if foreign {
		return false, "not authored by current user"
	}

	// Check if on main/master
	if protected, err := ad.isProtectedBranch(ctx); err != nil {
		return false, fmt.Sprintf("error checking branch: %v", err)
	} else if protected {
		return false, "on protected branch"
	}

	return true, ""
}

// isCommitPushed checks if the last commit is pushed to remote
func (ad *AmendDetector) isCommitPushed(ctx context.Context) (bool, error) {
	// Get HEAD commit hash
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = ad.repoPath
	headOutput, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git rev-parse HEAD failed: %w", err)
	}
	headHash := strings.TrimSpace(string(headOutput))

	// Get remote HEAD hash
	cmd = exec.CommandContext(ctx, "git", "rev-parse", "@{u}")
	cmd.Dir = ad.repoPath
	remoteOutput, err := cmd.Output()
	if err != nil {
		// No upstream configured
		return false, nil
	}
	remoteHash := strings.TrimSpace(string(remoteOutput))

	// Check if HEAD is in remote history
	cmd = exec.CommandContext(ctx, "git", "merge-base", "--is-ancestor", headHash, remoteHash)
	cmd.Dir = ad.repoPath
	if err := cmd.Run(); err == nil {
		// HEAD is ancestor of remote, so it's pushed
		return true, nil
	}

	// Check if remote is ancestor of HEAD (HEAD is ahead)
	cmd = exec.CommandContext(ctx, "git", "merge-base", "--is-ancestor", remoteHash, headHash)
	cmd.Dir = ad.repoPath
	if err := cmd.Run(); err == nil {
		// Remote is ancestor of HEAD, so HEAD is not pushed
		return false, nil
	}

	// Default to safe (don't amend if uncertain)
	return true, nil
}

// isForeignCommit checks if last commit is by another author
func (ad *AmendDetector) isForeignCommit(ctx context.Context) (bool, error) {
	// Get commit author email
	cmd := exec.CommandContext(ctx, "git", "log", "-1", "--format=%ae")
	cmd.Dir = ad.repoPath
	authorOutput, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git log failed: %w", err)
	}
	commitEmail := strings.TrimSpace(string(authorOutput))

	// Get current user email
	cmd = exec.CommandContext(ctx, "git", "config", "user.email")
	cmd.Dir = ad.repoPath
	userOutput, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git config user.email failed: %w", err)
	}
	currentEmail := strings.TrimSpace(string(userOutput))

	return commitEmail != currentEmail, nil
}

// isProtectedBranch checks if on main/master
func (ad *AmendDetector) isProtectedBranch(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = ad.repoPath
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git rev-parse failed: %w", err)
	}

	branchName := strings.TrimSpace(string(output))
	protected := []string{"main", "master", "develop", "production"}

	for _, p := range protected {
		if branchName == p {
			return true, nil
		}
	}

	return false, nil
}

// GetLastCommitInfo retrieves information about the last commit
func (ad *AmendDetector) GetLastCommitInfo(ctx context.Context) (*CommitInfo, error) {
	// Get commit hash
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = ad.repoPath
	hashOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git rev-parse HEAD failed: %w", err)
	}

	// Get commit message
	cmd = exec.CommandContext(ctx, "git", "log", "-1", "--format=%B")
	cmd.Dir = ad.repoPath
	msgOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	// Get author
	cmd = exec.CommandContext(ctx, "git", "log", "-1", "--format=%an")
	cmd.Dir = ad.repoPath
	authorNameOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log author name failed: %w", err)
	}

	cmd = exec.CommandContext(ctx, "git", "log", "-1", "--format=%ae")
	cmd.Dir = ad.repoPath
	authorEmailOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log author email failed: %w", err)
	}

	// Get timestamp
	cmd = exec.CommandContext(ctx, "git", "log", "-1", "--format=%aI")
	cmd.Dir = ad.repoPath
	timestampOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log timestamp failed: %w", err)
	}

	return &CommitInfo{
		Hash:    strings.TrimSpace(string(hashOutput)),
		Message: strings.TrimSpace(string(msgOutput)),
		Author: Person{
			Name:  strings.TrimSpace(string(authorNameOutput)),
			Email: strings.TrimSpace(string(authorEmailOutput)),
		},
		Timestamp: strings.TrimSpace(string(timestampOutput)),
	}, nil
}

// CommitInfo contains information about a commit
type CommitInfo struct {
	Hash      string
	Message   string
	Author    Person
	Timestamp string
}

// FormatAttribution formats an attribution for display
func FormatAttribution(attr Attribution) string {
	switch attr.Type {
	case AttributionCoAuthor:
		return fmt.Sprintf("Co-authored-by: %s <%s>", attr.Name, attr.Email)
	case AttributionSignedOff:
		return fmt.Sprintf("Signed-off-by: %s <%s>", attr.Name, attr.Email)
	case AttributionReviewed:
		return fmt.Sprintf("Reviewed-by: %s <%s>", attr.Name, attr.Email)
	case AttributionTested:
		return fmt.Sprintf("Tested-by: %s <%s>", attr.Name, attr.Email)
	default:
		return fmt.Sprintf("%s <%s>", attr.Name, attr.Email)
	}
}

// ParseAttribution parses an attribution line
func ParseAttribution(line string) (*Attribution, error) {
	line = strings.TrimSpace(line)

	// Try to match different attribution types
	patterns := map[AttributionType]string{
		AttributionCoAuthor:  "Co-authored-by:",
		AttributionSignedOff: "Signed-off-by:",
		AttributionReviewed:  "Reviewed-by:",
		AttributionTested:    "Tested-by:",
	}

	for attrType, prefix := range patterns {
		if strings.HasPrefix(line, prefix) {
			rest := strings.TrimSpace(strings.TrimPrefix(line, prefix))
			// Parse "Name <email>"
			if idx := strings.Index(rest, "<"); idx > 0 {
				name := strings.TrimSpace(rest[:idx])
				email := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(rest[idx:], "<"), ">"))
				return &Attribution{
					Type:  attrType,
					Name:  name,
					Email: email,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("invalid attribution line: %s", line)
}

// ExtractAttributions extracts all attributions from a commit message
func ExtractAttributions(message string) []Attribution {
	var attrs []Attribution
	lines := strings.Split(message, "\n")

	for _, line := range lines {
		if attr, err := ParseAttribution(line); err == nil {
			attrs = append(attrs, *attr)
		}
	}

	return attrs
}
