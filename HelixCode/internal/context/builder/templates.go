package builder

// GetDefaultTemplates returns the default context templates
func GetDefaultTemplates() []*Template {
	return []*Template{
		GetCodingTemplate(),
		GetDebuggingTemplate(),
		GetPlanningTemplate(),
		GetReviewTemplate(),
		GetRefactoringTemplate(),
	}
}

// GetCodingTemplate returns template for coding sessions
func GetCodingTemplate() *Template {
	return &Template{
		Name:        "coding",
		Description: "Context for coding and implementation",
		Sections: []*TemplateSection{
			{
				Title:    "Current Session",
				Types:    []SourceType{SourceSession},
				Priority: PriorityHigh,
				MaxItems: 1,
			},
			{
				Title:    "Recent Focus",
				Types:    []SourceType{SourceFocus},
				Priority: PriorityHigh,
				MaxItems: 1,
			},
			{
				Title:    "Project Information",
				Types:    []SourceType{SourceProject},
				Priority: PriorityNormal,
				MaxItems: 1,
			},
			{
				Title:    "Relevant Files",
				Types:    []SourceType{SourceFile},
				Priority: PriorityHigh,
				MaxItems: 5,
			},
			{
				Title:    "Recent Errors",
				Types:    []SourceType{SourceError},
				Priority: PriorityCritical,
				MaxItems: 3,
			},
		},
	}
}

// GetDebuggingTemplate returns template for debugging sessions
func GetDebuggingTemplate() *Template {
	return &Template{
		Name:        "debugging",
		Description: "Context for debugging and troubleshooting",
		Sections: []*TemplateSection{
			{
				Title:    "Recent Errors",
				Types:    []SourceType{SourceError},
				Priority: PriorityCritical,
				MaxItems: 5,
			},
			{
				Title:    "Current Session",
				Types:    []SourceType{SourceSession},
				Priority: PriorityHigh,
				MaxItems: 1,
			},
			{
				Title:    "Recent Focus",
				Types:    []SourceType{SourceFocus},
				Priority: PriorityHigh,
				MaxItems: 1,
			},
			{
				Title:    "Related Files",
				Types:    []SourceType{SourceFile},
				Priority: PriorityHigh,
				MaxItems: 3,
			},
			{
				Title:    "Logs",
				Types:    []SourceType{SourceLog},
				Priority: PriorityNormal,
				MaxItems: 3,
			},
		},
	}
}

// GetPlanningTemplate returns template for planning sessions
func GetPlanningTemplate() *Template {
	return &Template{
		Name:        "planning",
		Description: "Context for planning and design",
		Sections: []*TemplateSection{
			{
				Title:    "Current Session",
				Types:    []SourceType{SourceSession},
				Priority: PriorityHigh,
				MaxItems: 1,
			},
			{
				Title:    "Project Information",
				Types:    []SourceType{SourceProject},
				Priority: PriorityHigh,
				MaxItems: 1,
			},
			{
				Title:    "Recent Focus",
				Types:    []SourceType{SourceFocus},
				Priority: PriorityNormal,
				MaxItems: 1,
			},
			{
				Title:    "Design Documents",
				Types:    []SourceType{SourceFile},
				Priority: PriorityNormal,
				MaxItems: 5,
			},
		},
	}
}

// GetReviewTemplate returns template for code review
func GetReviewTemplate() *Template {
	return &Template{
		Name:        "review",
		Description: "Context for code review",
		Sections: []*TemplateSection{
			{
				Title:    "Current Session",
				Types:    []SourceType{SourceSession},
				Priority: PriorityNormal,
				MaxItems: 1,
			},
			{
				Title:    "Files Under Review",
				Types:    []SourceType{SourceFile},
				Priority: PriorityHigh,
				MaxItems: 10,
			},
			{
				Title:    "Git Changes",
				Types:    []SourceType{SourceGit},
				Priority: PriorityHigh,
				MaxItems: 1,
			},
			{
				Title:    "Project Standards",
				Types:    []SourceType{SourceProject},
				Priority: PriorityNormal,
				MaxItems: 1,
			},
		},
	}
}

// GetRefactoringTemplate returns template for refactoring
func GetRefactoringTemplate() *Template {
	return &Template{
		Name:        "refactoring",
		Description: "Context for refactoring code",
		Sections: []*TemplateSection{
			{
				Title:    "Current Session",
				Types:    []SourceType{SourceSession},
				Priority: PriorityHigh,
				MaxItems: 1,
			},
			{
				Title:    "Code To Refactor",
				Types:    []SourceType{SourceFile},
				Priority: PriorityCritical,
				MaxItems: 3,
			},
			{
				Title:    "Recent Focus",
				Types:    []SourceType{SourceFocus},
				Priority: PriorityNormal,
				MaxItems: 1,
			},
			{
				Title:    "Related Files",
				Types:    []SourceType{SourceFile},
				Priority: PriorityNormal,
				MaxItems: 5,
			},
			{
				Title:    "Project Standards",
				Types:    []SourceType{SourceProject},
				Priority: PriorityNormal,
				MaxItems: 1,
			},
		},
	}
}

// RegisterDefaultTemplates registers all default templates in the builder
func RegisterDefaultTemplates(builder *Builder) {
	templates := GetDefaultTemplates()
	for _, template := range templates {
		builder.RegisterTemplate(template)
	}
}
