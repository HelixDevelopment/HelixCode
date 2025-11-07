package mentions

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// GitMentionHandler handles @git-changes and @[commit-hash] mentions
type GitMentionHandler struct {
	workspaceRoot string
}

// NewGitMentionHandler creates a new git mention handler
func NewGitMentionHandler(workspaceRoot string) *GitMentionHandler {
	return &GitMentionHandler{
		workspaceRoot: workspaceRoot,
	}
}

// Type returns the mention type
func (h *GitMentionHandler) Type() MentionType {
	return MentionTypeGitChanges
}

// CanHandle checks if this handler can handle the mention
func (h *GitMentionHandler) CanHandle(mention string) bool {
	return strings.HasPrefix(mention, "@git-changes") ||
		(strings.HasPrefix(mention, "@[") && len(mention) > 8)
}

// Resolve resolves the git mention
func (h *GitMentionHandler) Resolve(ctx context.Context, target string, options map[string]string) (*MentionContext, error) {
	var content string
	var mentionType MentionType
	var targetDesc string

	if target == "" || strings.HasPrefix(target, "git-changes") {
		// Get uncommitted changes
		mentionType = MentionTypeGitChanges
		targetDesc = "uncommitted changes"

		// Get staged changes
		stagedCmd := exec.CommandContext(ctx, "git", "-C", h.workspaceRoot, "diff", "--cached")
		staged, _ := stagedCmd.Output()

		// Get unstaged changes
		unstagedCmd := exec.CommandContext(ctx, "git", "-C", h.workspaceRoot, "diff")
		unstaged, _ := unstagedCmd.Output()

		content = fmt.Sprintf("=== Staged Changes ===\n%s\n\n=== Unstaged Changes ===\n%s",
			string(staged), string(unstaged))
	} else {
		// Get specific commit
		mentionType = MentionTypeCommit
		targetDesc = target

		// Get commit diff
		cmd := exec.CommandContext(ctx, "git", "-C", h.workspaceRoot, "show", target)
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to get commit %s: %w", target, err)
		}
		content = string(output)
	}

	tokenCount := len(content) / 4

	return &MentionContext{
		Type:       mentionType,
		Target:     targetDesc,
		Content:    content,
		TokenCount: tokenCount,
		Metadata: map[string]interface{}{
			"workspace": h.workspaceRoot,
		},
		ResolvedAt: time.Now(),
	}, nil
}
