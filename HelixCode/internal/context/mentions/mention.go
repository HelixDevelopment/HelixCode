package mentions

import (
	"context"
	"time"
)

// MentionType represents the type of mention
type MentionType string

const (
	MentionTypeFile       MentionType = "file"
	MentionTypeFolder     MentionType = "folder"
	MentionTypeURL        MentionType = "url"
	MentionTypeGitChanges MentionType = "git-changes"
	MentionTypeCommit     MentionType = "commit"
	MentionTypeTerminal   MentionType = "terminal"
	MentionTypeProblems   MentionType = "problems"
)

// MentionContext contains the resolved context for a mention
type MentionContext struct {
	Type       MentionType            `json:"type"`
	Target     string                 `json:"target"`
	Content    string                 `json:"content"`
	TokenCount int                    `json:"token_count"`
	Metadata   map[string]interface{} `json:"metadata"`
	ResolvedAt time.Time              `json:"resolved_at"`
}

// MentionHandler interface for handling different mention types
type MentionHandler interface {
	Type() MentionType
	CanHandle(mention string) bool
	Resolve(ctx context.Context, mention string, options map[string]string) (*MentionContext, error)
}

// MentionResult contains the final processed content
type MentionResult struct {
	OriginalText  string           `json:"original_text"`
	ProcessedText string           `json:"processed_text"`
	Contexts      []MentionContext `json:"contexts"`
	TotalTokens   int              `json:"total_tokens"`
}
