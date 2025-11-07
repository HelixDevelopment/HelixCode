package builder

import (
	"fmt"
	"strings"

	"dev.helix.code/internal/focus"
	"dev.helix.code/internal/session"
)

// SessionSource provides context from session manager
type SessionSource struct {
	manager *session.Manager
}

// NewSessionSource creates a new session source
func NewSessionSource(manager *session.Manager) *SessionSource {
	return &SessionSource{
		manager: manager,
	}
}

// GetContext returns context from active session
func (s *SessionSource) GetContext() ([]*ContextItem, error) {
	active := s.manager.GetActive()
	if active == nil {
		return []*ContextItem{}, nil
	}

	content := fmt.Sprintf("Session: %s\nMode: %s\nStatus: %s\nDuration: %v\n",
		active.Name,
		active.Mode,
		active.Status,
		active.Duration,
	)

	// Add description if present
	if active.Description != "" {
		content += fmt.Sprintf("Description: %s\n", active.Description)
	}

	// Add metadata
	if len(active.Metadata) > 0 {
		content += "\nMetadata:\n"
		for k, v := range active.Metadata {
			content += fmt.Sprintf("  %s: %s\n", k, v)
		}
	}

	// Add tags
	if len(active.Tags) > 0 {
		content += "\nTags: " + strings.Join(active.Tags, ", ") + "\n"
	}

	return []*ContextItem{
		{
			Type:     SourceSession,
			Priority: PriorityHigh,
			Title:    "Current Session",
			Content:  content,
			Metadata: active.Metadata,
			Size:     len(content),
		},
	}, nil
}

// Type returns the source type
func (s *SessionSource) Type() SourceType {
	return SourceSession
}

// FocusSource provides context from focus chain
type FocusSource struct {
	manager  *focus.Manager
	maxItems int
}

// NewFocusSource creates a new focus source
func NewFocusSource(manager *focus.Manager, maxItems int) *FocusSource {
	if maxItems <= 0 {
		maxItems = 10
	}

	return &FocusSource{
		manager:  manager,
		maxItems: maxItems,
	}
}

// GetContext returns context from active focus chain
func (f *FocusSource) GetContext() ([]*ContextItem, error) {
	chain, err := f.manager.GetActiveChain()
	if err != nil {
		return []*ContextItem{}, nil // No active chain
	}

	recent := chain.GetRecent(f.maxItems)
	if len(recent) == 0 {
		return []*ContextItem{}, nil
	}

	content := fmt.Sprintf("Focus Chain: %s\nRecent focuses (%d):\n", chain.Name, len(recent))

	for i, focus := range recent {
		content += fmt.Sprintf("\n%d. %s", i+1, focus.Target)

		if focus.Type != "" {
			content += fmt.Sprintf(" (%s)", focus.Type)
		}

		content += "\n"

		// Add priority if not normal
		if focus.Priority != 5 {
			content += fmt.Sprintf("   Priority: %d\n", focus.Priority)
		}

		// Add tags if present
		if len(focus.Tags) > 0 {
			content += fmt.Sprintf("   Tags: %s\n", strings.Join(focus.Tags, ", "))
		}

		// Add metadata if present
		if len(focus.Metadata) > 0 {
			content += "   Metadata:\n"
			for k, v := range focus.Metadata {
				content += fmt.Sprintf("     %s: %s\n", k, v)
			}
		}
	}

	return []*ContextItem{
		{
			Type:     SourceFocus,
			Priority: PriorityHigh,
			Title:    "Recent Focus",
			Content:  content,
			Metadata: make(map[string]string),
			Size:     len(content),
		},
	}, nil
}

// Type returns the source type
func (f *FocusSource) Type() SourceType {
	return SourceFocus
}

// ProjectSource provides project context
type ProjectSource struct {
	projectName string
	description string
	metadata    map[string]string
}

// NewProjectSource creates a new project source
func NewProjectSource(projectName, description string, metadata map[string]string) *ProjectSource {
	if metadata == nil {
		metadata = make(map[string]string)
	}

	return &ProjectSource{
		projectName: projectName,
		description: description,
		metadata:    metadata,
	}
}

// GetContext returns project context
func (p *ProjectSource) GetContext() ([]*ContextItem, error) {
	content := fmt.Sprintf("Project: %s\n", p.projectName)

	if p.description != "" {
		content += fmt.Sprintf("Description: %s\n", p.description)
	}

	if len(p.metadata) > 0 {
		content += "\nProject Information:\n"
		for k, v := range p.metadata {
			content += fmt.Sprintf("  %s: %s\n", k, v)
		}
	}

	return []*ContextItem{
		{
			Type:     SourceProject,
			Priority: PriorityNormal,
			Title:    "Project Information",
			Content:  content,
			Metadata: p.metadata,
			Size:     len(content),
		},
	}, nil
}

// Type returns the source type
func (p *ProjectSource) Type() SourceType {
	return SourceProject
}

// FileSource provides file content context
type FileSource struct {
	filePath string
	content  string
	priority Priority
}

// NewFileSource creates a new file source
func NewFileSource(filePath, content string, priority Priority) *FileSource {
	return &FileSource{
		filePath: filePath,
		content:  content,
		priority: priority,
	}
}

// GetContext returns file context
func (f *FileSource) GetContext() ([]*ContextItem, error) {
	content := fmt.Sprintf("File: %s\n\n```\n%s\n```\n", f.filePath, f.content)

	return []*ContextItem{
		{
			Type:     SourceFile,
			Priority: f.priority,
			Title:    f.filePath,
			Content:  content,
			Metadata: map[string]string{"file_path": f.filePath},
			Size:     len(content),
		},
	}, nil
}

// Type returns the source type
func (f *FileSource) Type() SourceType {
	return SourceFile
}

// ErrorSource provides error context
type ErrorSource struct {
	errors []ErrorInfo
}

// ErrorInfo represents an error with context
type ErrorInfo struct {
	Message   string
	File      string
	Line      int
	Timestamp string
}

// NewErrorSource creates a new error source
func NewErrorSource() *ErrorSource {
	return &ErrorSource{
		errors: make([]ErrorInfo, 0),
	}
}

// AddError adds an error to the source
func (e *ErrorSource) AddError(message, file string, line int, timestamp string) {
	e.errors = append(e.errors, ErrorInfo{
		Message:   message,
		File:      file,
		Line:      line,
		Timestamp: timestamp,
	})
}

// GetContext returns error context
func (e *ErrorSource) GetContext() ([]*ContextItem, error) {
	if len(e.errors) == 0 {
		return []*ContextItem{}, nil
	}

	content := fmt.Sprintf("Recent Errors (%d):\n\n", len(e.errors))

	for i, err := range e.errors {
		content += fmt.Sprintf("%d. %s\n", i+1, err.Message)

		if err.File != "" {
			content += fmt.Sprintf("   File: %s", err.File)
			if err.Line > 0 {
				content += fmt.Sprintf(":%d", err.Line)
			}
			content += "\n"
		}

		if err.Timestamp != "" {
			content += fmt.Sprintf("   Time: %s\n", err.Timestamp)
		}

		content += "\n"
	}

	return []*ContextItem{
		{
			Type:     SourceError,
			Priority: PriorityCritical,
			Title:    "Recent Errors",
			Content:  content,
			Metadata: make(map[string]string),
			Size:     len(content),
		},
	}, nil
}

// Type returns the source type
func (e *ErrorSource) Type() SourceType {
	return SourceError
}

// CustomSource provides custom context
type CustomSource struct {
	sourceType SourceType
	getFunc    func() ([]*ContextItem, error)
}

// NewCustomSource creates a new custom source
func NewCustomSource(sourceType SourceType, getFunc func() ([]*ContextItem, error)) *CustomSource {
	return &CustomSource{
		sourceType: sourceType,
		getFunc:    getFunc,
	}
}

// GetContext returns custom context
func (c *CustomSource) GetContext() ([]*ContextItem, error) {
	return c.getFunc()
}

// Type returns the source type
func (c *CustomSource) Type() SourceType {
	return c.sourceType
}
