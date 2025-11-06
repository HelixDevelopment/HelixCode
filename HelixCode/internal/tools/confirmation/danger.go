package confirmation

import (
	"strings"
)

// DangerPattern defines a dangerous pattern
type DangerPattern struct {
	Name        string
	Description string
	Risk        RiskLevel
	Reversible  bool
	Match       func(ConfirmationRequest) bool
}

// DangerAssessment contains risk assessment
type DangerAssessment struct {
	Risk       RiskLevel
	Dangers    []string
	Reversible bool
}

// DangerDetector identifies dangerous operations
type DangerDetector struct {
	patterns []DangerPattern
}

// NewDangerDetector creates a new danger detector with default patterns
func NewDangerDetector() *DangerDetector {
	return &DangerDetector{
		patterns: defaultDangerPatterns(),
	}
}

// Detect checks if operation is dangerous
func (dd *DangerDetector) Detect(req ConfirmationRequest) *DangerAssessment {
	assessment := &DangerAssessment{
		Risk:       RiskLow,
		Dangers:    []string{},
		Reversible: true,
	}

	for _, pattern := range dd.patterns {
		if pattern.Match(req) {
			assessment.Risk = maxRisk(assessment.Risk, pattern.Risk)
			assessment.Dangers = append(assessment.Dangers, pattern.Description)
			if !pattern.Reversible {
				assessment.Reversible = false
			}
		}
	}

	return assessment
}

// AddPattern adds a custom danger pattern
func (dd *DangerDetector) AddPattern(pattern DangerPattern) {
	dd.patterns = append(dd.patterns, pattern)
}

// defaultDangerPatterns returns default danger detection patterns
func defaultDangerPatterns() []DangerPattern {
	return []DangerPattern{
		{
			Name:        "delete_operation",
			Description: "Deleting files or data",
			Risk:        RiskHigh,
			Reversible:  false,
			Match: func(req ConfirmationRequest) bool {
				return req.Operation.Type == OpDelete
			},
		},
		{
			Name:        "system_files",
			Description: "Operating on system files",
			Risk:        RiskCritical,
			Reversible:  false,
			Match: func(req ConfirmationRequest) bool {
				systemPaths := []string{"/etc", "/sys", "/bin", "/usr", "/sbin", "/boot"}
				for _, path := range systemPaths {
					if strings.HasPrefix(req.Operation.Target, path) {
						return true
					}
				}
				return false
			},
		},
		{
			Name:        "rm_rf_command",
			Description: "Recursive force delete (rm -rf)",
			Risk:        RiskCritical,
			Reversible:  false,
			Match: func(req ConfirmationRequest) bool {
				if req.ToolName == "bash" {
					if cmd, ok := req.Parameters["command"].(string); ok {
						return strings.Contains(cmd, "rm -rf") || strings.Contains(cmd, "rm -fr")
					}
				}
				return false
			},
		},
		{
			Name:        "git_force_push",
			Description: "Force pushing to git remote",
			Risk:        RiskHigh,
			Reversible:  false,
			Match: func(req ConfirmationRequest) bool {
				if req.ToolName == "git" {
					if cmd, ok := req.Parameters["command"].(string); ok {
						return strings.Contains(cmd, "push") && (strings.Contains(cmd, "--force") || strings.Contains(cmd, "-f"))
					}
				}
				return false
			},
		},
		{
			Name:        "main_branch_operation",
			Description: "Operating on main/master branch",
			Risk:        RiskMedium,
			Reversible:  true,
			Match: func(req ConfirmationRequest) bool {
				if branch, ok := req.Parameters["branch"].(string); ok {
					return branch == "main" || branch == "master"
				}
				return false
			},
		},
		{
			Name:        "network_request",
			Description: "Making network requests",
			Risk:        RiskMedium,
			Reversible:  true,
			Match: func(req ConfirmationRequest) bool {
				return req.Operation.Type == OpNetwork
			},
		},
		{
			Name:        "sudo_command",
			Description: "Executing with elevated privileges (sudo)",
			Risk:        RiskHigh,
			Reversible:  false,
			Match: func(req ConfirmationRequest) bool {
				if req.ToolName == "bash" {
					if cmd, ok := req.Parameters["command"].(string); ok {
						return strings.HasPrefix(strings.TrimSpace(cmd), "sudo ")
					}
				}
				return false
			},
		},
		{
			Name:        "chmod_777",
			Description: "Setting dangerous file permissions (chmod 777)",
			Risk:        RiskHigh,
			Reversible:  true,
			Match: func(req ConfirmationRequest) bool {
				if req.ToolName == "bash" {
					if cmd, ok := req.Parameters["command"].(string); ok {
						return strings.Contains(cmd, "chmod 777") || strings.Contains(cmd, "chmod -R 777")
					}
				}
				return false
			},
		},
		{
			Name:        "drop_table",
			Description: "Dropping database tables",
			Risk:        RiskCritical,
			Reversible:  false,
			Match: func(req ConfirmationRequest) bool {
				if cmd, ok := req.Parameters["command"].(string); ok {
					cmdLower := strings.ToLower(cmd)
					return strings.Contains(cmdLower, "drop table") || strings.Contains(cmdLower, "drop database")
				}
				return false
			},
		},
		{
			Name:        "truncate_table",
			Description: "Truncating database tables",
			Risk:        RiskHigh,
			Reversible:  false,
			Match: func(req ConfirmationRequest) bool {
				if cmd, ok := req.Parameters["command"].(string); ok {
					cmdLower := strings.ToLower(cmd)
					return strings.Contains(cmdLower, "truncate table")
				}
				return false
			},
		},
		{
			Name:        "npm_publish",
			Description: "Publishing to npm registry",
			Risk:        RiskHigh,
			Reversible:  false,
			Match: func(req ConfirmationRequest) bool {
				if req.ToolName == "bash" {
					if cmd, ok := req.Parameters["command"].(string); ok {
						return strings.Contains(cmd, "npm publish")
					}
				}
				return false
			},
		},
		{
			Name:        "pip_upload",
			Description: "Uploading to PyPI",
			Risk:        RiskHigh,
			Reversible:  false,
			Match: func(req ConfirmationRequest) bool {
				if req.ToolName == "bash" {
					if cmd, ok := req.Parameters["command"].(string); ok {
						return strings.Contains(cmd, "twine upload") || strings.Contains(cmd, "python setup.py upload")
					}
				}
				return false
			},
		},
		{
			Name:        "docker_system_prune",
			Description: "Removing all unused Docker resources",
			Risk:        RiskMedium,
			Reversible:  false,
			Match: func(req ConfirmationRequest) bool {
				if req.ToolName == "bash" {
					if cmd, ok := req.Parameters["command"].(string); ok {
						return strings.Contains(cmd, "docker system prune")
					}
				}
				return false
			},
		},
		{
			Name:        "format_disk",
			Description: "Formatting disk or partition",
			Risk:        RiskCritical,
			Reversible:  false,
			Match: func(req ConfirmationRequest) bool {
				if req.ToolName == "bash" {
					if cmd, ok := req.Parameters["command"].(string); ok {
						return strings.Contains(cmd, "mkfs") || strings.Contains(cmd, "format")
					}
				}
				return false
			},
		},
		{
			Name:        "kill_process",
			Description: "Terminating processes",
			Risk:        RiskMedium,
			Reversible:  false,
			Match: func(req ConfirmationRequest) bool {
				if req.ToolName == "bash" {
					if cmd, ok := req.Parameters["command"].(string); ok {
						return strings.Contains(cmd, "kill -9") || strings.Contains(cmd, "pkill")
					}
				}
				return false
			},
		},
	}
}

// maxRisk returns the higher risk level
func maxRisk(a, b RiskLevel) RiskLevel {
	if a > b {
		return a
	}
	return b
}
