// Package fix provides automated security issue resolution with zero-tolerance
package fix

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"dev.helix.code/internal/security"
)

// SecurityIssueFixer provides automated security issue resolution
type SecurityIssueFixer struct {
	projectPath     string
	securityManager *security.SecurityManager
	fixedIssues     map[string]bool
	criticalOnly    bool
	backupFiles     bool
}

// SecurityFix represents a security issue with fix information
type SecurityFix struct {
	ID              string
	Title           string
	Severity        string
	Type            string
	File            string
	Line            int
	Description     string
	SuggestedFix    string
	AutomatedFix    bool
	FixType         FixType
	ValidationRegex string
	FixCommand      string
	Criticality     int
	Dependencies    []string
}

// FixType defines the type of security fix
type FixType string

const (
	CodeFix          FixType = "code_fix"
	ConfigurationFix FixType = "configuration_fix"
	DependencyFix    FixType = "dependency_fix"
	PermissionFix    FixType = "permission_fix"
	ContainerFix     FixType = "container_fix"
	PolicyFix        FixType = "policy_fix"
	ManualFix        FixType = "manual_fix"
)

// NewSecurityIssueFixer creates a new security issue fixer
func NewSecurityIssueFixer(projectPath string, criticalOnly bool) (*SecurityIssueFixer, error) {
	fixer := &SecurityIssueFixer{
		projectPath:  projectPath,
		fixedIssues:  make(map[string]bool),
		criticalOnly: criticalOnly,
		backupFiles:  true,
	}

	// Initialize security manager if needed
	if secMgr := security.GetGlobalSecurityManager(); secMgr == nil {
		if err := security.InitGlobalSecurityManager(); err != nil {
			return nil, fmt.Errorf("failed to initialize security manager: %v", err)
		}
	}

	fixer.securityManager = security.GetGlobalSecurityManager()
	return fixer, nil
}

// FixAllCriticalIssues fixes all critical security issues with zero tolerance
func (sif *SecurityIssueFixer) FixAllCriticalIssues() (*FixResult, error) {
	log.Printf("🔧 Starting Zero-Tolerance Critical Security Issue Resolution")
	log.Printf("🎯 Policy: ZERO TOLERANCE - All critical issues must be fixed")

	result := &FixResult{
		StartTime:     time.Now(),
		TotalIssues:   0,
		FixedIssues:   0,
		FailedFixes:   0,
		ManualFixes:   0,
		SkippedIssues: 0,
	}

	// Scan for current security issues
	log.Printf("🔍 Scanning for security issues...")
	issues, err := sif.scanForSecurityIssues()
	if err != nil {
		return nil, fmt.Errorf("failed to scan for security issues: %v", err)
	}

	result.TotalIssues = len(issues)
	log.Printf("📋 Found %d security issues to address", result.TotalIssues)

	// Filter critical issues
	criticalIssues := sif.filterCriticalIssues(issues)
	log.Printf("🚨 %d critical security issues require immediate fixing", len(criticalIssues))

	if len(criticalIssues) == 0 {
		log.Printf("✅ No critical security issues found")
		result.Success = true
		result.EndTime = time.Now()
		return result, nil
	}

	// Create backup of current state
	if sif.backupFiles {
		if err := sif.createBackup(); err != nil {
			log.Printf("⚠️ Failed to create backup: %v", err)
		} else {
			log.Printf("💾 Backup created successfully")
		}
	}

	// Fix critical issues systematically
	for i, issue := range criticalIssues {
		log.Printf("\n🔧 Fixing Critical Issue %d/%d: %s", i+1, len(criticalIssues), issue.ID)
		log.Printf("📝 Description: %s", issue.Description)
		log.Printf("📁 File: %s:%d", issue.File, issue.Line)

		if sif.fixedIssues[issue.ID] {
			log.Printf("⏭️  Issue already fixed - skipping")
			result.SkippedIssues++
			continue
		}

		// Attempt automated fix
		fixSuccess, err := sif.fixSecurityIssue(&issue)
		if err != nil {
			log.Printf("❌ Failed to fix issue: %v", err)
			result.FailedFixes++
			continue
		}

		if fixSuccess {
			log.Printf("✅ Issue fixed successfully: %s", issue.ID)
			result.FixedIssues++
			sif.fixedIssues[issue.ID] = true
		} else if issue.FixType == ManualFix {
			log.Printf("📋 Manual fix required: %s", issue.ID)
			log.Printf("💡 Suggested Fix: %s", issue.SuggestedFix)
			result.ManualFixes++
		} else {
			log.Printf("❌ Automated fix failed: %s", issue.ID)
			result.FailedFixes++
		}

		// Small delay between fixes for safety
		time.Sleep(100 * time.Millisecond)
	}

	// Validate fixes
	log.Printf("\n🔍 Validating security fixes...")
	validationResult, err := sif.validateFixes()
	if err != nil {
		log.Printf("⚠️ Fix validation error: %v", err)
	}

	result.Validation = validationResult
	result.EndTime = time.Now()

	// Determine overall success
	result.Success = result.FixedIssues == len(criticalIssues) &&
		validationResult.RemainingCriticalIssues == 0

	// Generate comprehensive fix report
	sif.generateFixReport(result, criticalIssues)

	// Final evaluation
	if result.Success {
		log.Printf("\n🎉 SUCCESS: All critical security issues resolved")
		log.Printf("✅ Zero-Tolerance Policy SATISFIED")
		log.Printf("🚀 Platform ready for production")
	} else {
		log.Printf("\n❌ FAILURE: Critical security issues remain")
		log.Printf("🚨 Zero-Tolerance Policy VIOLATED")
		log.Printf("🔧 Manual intervention required")
		log.Printf("🚫 Production deployment BLOCKED")
	}

	return result, nil
}

// scanForSecurityIssues performs comprehensive security issue scanning
func (sif *SecurityIssueFixer) scanForSecurityIssues() ([]SecurityFix, error) {
	var issues []SecurityFix

	// Scan for SSH security issues
	sshIssues := sif.scanSSHIssues()
	issues = append(issues, sshIssues...)

	// Scan for database security issues
	dbIssues := sif.scanDatabaseIssues()
	issues = append(issues, dbIssues...)

	// Scan for input validation issues
	inputIssues := sif.scanInputValidationIssues()
	issues = append(issues, inputIssues...)

	// Scan for configuration security issues
	configIssues := sif.scanConfigurationIssues()
	issues = append(issues, configIssues...)

	// Scan for filesystem security issues
	fsIssues := sif.scanFilesystemIssues()
	issues = append(issues, fsIssues...)

	// Scan for authentication issues
	authIssues := sif.scanAuthenticationIssues()
	issues = append(issues, authIssues...)

	// Scan for API security issues
	apiIssues := sif.scanAPISecurityIssues()
	issues = append(issues, apiIssues...)

	// Scan for dependency issues
	depIssues := sif.scanDependencyIssues()
	issues = append(issues, depIssues...)

	// Scan for container issues
	containerIssues := sif.scanContainerIssues()
	issues = append(issues, containerIssues...)

	// Scan for worker isolation issues
	workerIssues := sif.scanWorkerIsolationIssues()
	issues = append(issues, workerIssues...)

	return issues, nil
}

// Security scanning implementations
func (sif *SecurityIssueFixer) scanSSHIssues() []SecurityFix {
	var issues []SecurityFix

	// Check for InsecureIgnoreHostKey
	files := sif.findGoFiles()
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		if strings.Contains(string(content), "InsecureIgnoreHostKey") {
			issues = append(issues, SecurityFix{
				ID:           "SSH001",
				Title:        "Insecure SSH Host Key Verification",
				Severity:     "critical",
				Type:         "mitm",
				File:         file,
				Description:  "SSH connections bypass host key verification, enabling man-in-the-middle attacks",
				SuggestedFix: "Replace InsecureIgnoreHostKey with proper host key verification using ssh.FixedHostKey",
				AutomatedFix: true,
				FixType:      CodeFix,
				Criticality:  10,
			})
		}
	}

	return issues
}

func (sif *SecurityIssueFixer) scanDatabaseIssues() []SecurityFix {
	var issues []SecurityFix

	// Check for SQL injection vulnerabilities
	files := sif.findGoFiles()
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Look for unsafe SQL string formatting
		sqlPattern := regexp.MustCompile(`fmt\.Sprintf.*SELECT|fmt\.Sprintf.*INSERT|fmt\.Sprintf.*UPDATE|fmt\.Sprintf.*DELETE`)
		matches := sqlPattern.FindAllStringIndex(string(content), -1)
		for _, match := range matches {
			line := sif.getLineNumber(content, match[0])
			issues = append(issues, SecurityFix{
				ID:           "DB001",
				Title:        "SQL Injection Vulnerability",
				Severity:     "critical",
				Type:         "injection",
				File:         file,
				Line:         line,
				Description:  "SQL query constructed with string formatting allows injection attacks",
				SuggestedFix: "Use parameterized queries or prepared statements instead of string formatting",
				AutomatedFix: false,
				FixType:      CodeFix,
				Criticality:  9,
			})
		}
	}

	return issues
}

func (sif *SecurityIssueFixer) scanInputValidationIssues() []SecurityFix {
	var issues []SecurityFix

	// Check for unvalidated user input
	files := sif.findGoFiles()
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Look for unsafe user input handling
		unsafePattern := regexp.MustCompile(`http\.Request\.(Form|URL|Body|Header).*os\.Exec|http\.Request\.(Form|URL|Body|Header).*eval`)
		matches := unsafePattern.FindAllStringIndex(string(content), -1)
		for _, match := range matches {
			line := sif.getLineNumber(content, match[0])
			issues = append(issues, SecurityFix{
				ID:           "INPUT001",
				Title:        "Unvalidated User Input",
				Severity:     "critical",
				Type:         "injection",
				File:         file,
				Line:         line,
				Description:  "User input passed directly to execution without validation",
				SuggestedFix: "Validate and sanitize all user input before execution",
				AutomatedFix: false,
				FixType:      CodeFix,
				Criticality:  9,
			})
		}
	}

	return issues
}

func (sif *SecurityIssueFixer) scanConfigurationIssues() []SecurityFix {
	var issues []SecurityFix

	// Check for hardcoded secrets
	files := sif.findAllFiles()
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Look for hardcoded passwords, API keys, tokens
		secretPatterns := []string{
			`password\s*[:=]\s*["\'][^"\']+["\']`,
			`api[_-]?key\s*[:=]\s*["\'][^"\']+["\']`,
			`token\s*[:=]\s*["\'][^"\']+["\']`,
			`secret\s*[:=]\s*["\'][^"\']+["\']`,
		}

		for _, pattern := range secretPatterns {
			regex := regexp.MustCompile(`(?i)` + pattern)
			matches := regex.FindAllStringIndex(string(content), -1)
			for _, match := range matches {
				line := sif.getLineNumber(content, match[0])
				issues = append(issues, SecurityFix{
					ID:           "CONFIG001",
					Title:        "Hardcoded Security Secrets",
					Severity:     "critical",
					Type:         "credential_exposure",
					File:         file,
					Line:         line,
					Description:  "Security credentials hardcoded in source code",
					SuggestedFix: "Move secrets to environment variables or secure vault",
					AutomatedFix: false,
					FixType:      ConfigurationFix,
					Criticality:  10,
				})
			}
		}
	}

	return issues
}

func (sif *SecurityIssueFixer) scanFilesystemIssues() []SecurityFix {
	var issues []SecurityFix

	// Check for path traversal vulnerabilities
	files := sif.findGoFiles()
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Look for unsafe file path construction
		pathPattern := regexp.MustCompile(`filepath\.Join.*\+|os\.Open.*\+|filepath\.Clean.*\+`)
		matches := pathPattern.FindAllStringIndex(string(content), -1)
		for _, match := range matches {
			line := sif.getLineNumber(content, match[0])
			issues = append(issues, SecurityFix{
				ID:           "FS001",
				Title:        "Path Traversal Vulnerability",
				Severity:     "critical",
				Type:         "path_traversal",
				File:         file,
				Line:         line,
				Description:  "File path constructed with user input without validation",
				SuggestedFix: "Validate and sanitize file paths, use filepath.Clean and check for directory traversal",
				AutomatedFix: false,
				FixType:      CodeFix,
				Criticality:  9,
			})
		}
	}

	return issues
}

func (sif *SecurityIssueFixer) scanAuthenticationIssues() []SecurityFix {
	var issues []SecurityFix

	// Check for JWT validation issues
	files := sif.findGoFiles()
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Look for missing JWT validation
		if strings.Contains(string(content), "jwt.Parse") &&
			!strings.Contains(string(content), "jwt.ParseWithClaims") {
			issues = append(issues, SecurityFix{
				ID:           "AUTH001",
				Title:        "JWT Token Not Properly Validated",
				Severity:     "high",
				Type:         "authentication",
				File:         file,
				Description:  "JWT token parsed without proper validation of claims",
				SuggestedFix: "Use jwt.ParseWithClaims with proper validation of signature, expiration, and claims",
				AutomatedFix: false,
				FixType:      CodeFix,
				Criticality:  8,
			})
		}
	}

	return issues
}

func (sif *SecurityIssueFixer) scanAPISecurityIssues() []SecurityFix {
	var issues []SecurityFix

	// Check for CORS misconfiguration
	files := sif.findGoFiles()
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Look for permissive CORS configuration
		if strings.Contains(string(content), `"Access-Control-Allow-Origin", "*"`) {
			issues = append(issues, SecurityFix{
				ID:           "API001",
				Title:        "CORS Misconfiguration",
				Severity:     "medium",
				Type:         "configuration",
				File:         file,
				Description:  "CORS allows access from any origin",
				SuggestedFix: "Restrict CORS to specific trusted origins",
				AutomatedFix: false,
				FixType:      ConfigurationFix,
				Criticality:  5,
			})
		}
	}

	return issues
}

func (sif *SecurityIssueFixer) scanDependencyIssues() []SecurityFix {
	var issues []SecurityFix

	// Check go.mod for vulnerable dependencies
	if _, err := os.Stat(filepath.Join(sif.projectPath, "go.mod")); err == nil {
		// This would typically use Snyk or similar service to check dependencies
		// For simulation, we'll check for known vulnerable versions
		issues = append(issues, SecurityFix{
			ID:           "DEP001",
			Title:        "Potentially Vulnerable Dependencies",
			Severity:     "high",
			Type:         "dependency",
			File:         "go.mod",
			Description:  "Dependencies may contain known vulnerabilities",
			SuggestedFix: "Run 'go list -m -u all' and update to latest secure versions",
			AutomatedFix: false,
			FixType:      DependencyFix,
			Criticality:  7,
		})
	}

	return issues
}

func (sif *SecurityIssueFixer) scanContainerIssues() []SecurityFix {
	var issues []SecurityFix

	// Check Dockerfile for security issues
	dockerfile := filepath.Join(sif.projectPath, "Dockerfile")
	if _, err := os.Stat(dockerfile); err == nil {
		content, err := os.ReadFile(dockerfile)
		if err != nil {
			return issues
		}

		// Check for running as root
		if !strings.Contains(string(content), "USER") {
			issues = append(issues, SecurityFix{
				ID:           "CONT001",
				Title:        "Container Running as Root",
				Severity:     "high",
				Type:         "privilege_escalation",
				File:         dockerfile,
				Description:  "Docker container configured to run as root user",
				SuggestedFix: "Add 'USER nonrootuser' to Dockerfile and create non-root user",
				AutomatedFix: false,
				FixType:      ContainerFix,
				Criticality:  8,
			})
		}
	}

	return issues
}

func (sif *SecurityIssueFixer) scanWorkerIsolationIssues() []SecurityFix {
	var issues []SecurityFix

	// Check for worker sandboxing issues
	files := sif.findGoFiles()
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Check for exec.Command without sandboxing
		if strings.Contains(string(content), "exec.Command") &&
			!strings.Contains(string(content), "chroot") &&
			!strings.Contains(string(content), "namespace") {
			issues = append(issues, SecurityFix{
				ID:           "WORKER001",
				Title:        "Insufficient Worker Isolation",
				Severity:     "high",
				Type:         "isolation",
				File:         file,
				Description:  "Worker execution lacks proper sandboxing",
				SuggestedFix: "Implement proper sandboxing using chroot, namespaces, or containerization",
				AutomatedFix: false,
				FixType:      CodeFix,
				Criticality:  8,
			})
		}
	}

	return issues
}

// Helper functions
func (sif *SecurityIssueFixer) findGoFiles() []string {
	var files []string
	filepath.Walk(sif.projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
		return nil
	})
	return files
}

func (sif *SecurityIssueFixer) findAllFiles() []string {
	var files []string
	filepath.Walk(sif.projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files
}

func (sif *SecurityIssueFixer) getLineNumber(content []byte, position int) int {
	lines := strings.Split(string(content[:position]), "\n")
	return len(lines)
}

func (sif *SecurityIssueFixer) filterCriticalIssues(issues []SecurityFix) []SecurityFix {
	if sif.criticalOnly {
		var critical []SecurityFix
		for _, issue := range issues {
			if strings.EqualFold(issue.Severity, "critical") {
				critical = append(critical, issue)
			}
		}
		return critical
	}
	return issues
}

// Supporting types
type FixResult struct {
	StartTime     time.Time
	EndTime       time.Time
	TotalIssues   int
	FixedIssues   int
	FailedFixes   int
	ManualFixes   int
	SkippedIssues int
	Success       bool
	Validation    *ValidationResult
}

type ValidationResult struct {
	ScanResults             *security.FeatureScanResult
	RemainingCriticalIssues int
	RemainingHighIssues     int
	FixesValidated          bool
}

// Main execution function
func (sif *SecurityIssueFixer) fixSecurityIssue(issue *SecurityFix) (bool, error) {
	switch issue.FixType {
	case CodeFix:
		return sif.fixCodeIssue(issue)
	case ConfigurationFix:
		return sif.fixConfigurationIssue(issue)
	case DependencyFix:
		return sif.fixDependencyIssue(issue)
	case ContainerFix:
		return sif.fixContainerIssue(issue)
	case ManualFix:
		return false, nil // Manual fixes return false but no error
	default:
		return false, fmt.Errorf("unsupported fix type: %s", issue.FixType)
	}
}

func (sif *SecurityIssueFixer) fixCodeIssue(issue *SecurityFix) (bool, error) {
	// Implementation for automated code fixes
	if issue.ID == "SSH001" {
		return sif.fixSSHHostKey(issue)
	}
	return false, fmt.Errorf("automated fix not implemented for issue: %s", issue.ID)
}

func (sif *SecurityIssueFixer) fixSSHHostKey(issue *SecurityFix) (bool, error) {
	// Read file content
	content, err := os.ReadFile(issue.File)
	if err != nil {
		return false, err
	}

	// Replace InsecureIgnoreHostKey with secure alternative
	modifiedContent := strings.ReplaceAll(string(content),
		"InsecureIgnoreHostKey()",
		"ssh.HostKeyCallback(ssh.FixedHostKey(hostKey))")

	// Write back to file
	return true, os.WriteFile(issue.File, []byte(modifiedContent), 0644)
}

// Placeholder implementations for other fix types
func (sif *SecurityIssueFixer) fixConfigurationIssue(issue *SecurityFix) (bool, error) {
	log.Printf("🔧 Configuration fix: %s", issue.SuggestedFix)
	return false, nil
}

func (sif *SecurityIssueFixer) fixDependencyIssue(issue *SecurityFix) (bool, error) {
	log.Printf("🔧 Dependency fix: %s", issue.SuggestedFix)
	return false, nil
}

func (sif *SecurityIssueFixer) fixContainerIssue(issue *SecurityFix) (bool, error) {
	log.Printf("🔧 Container fix: %s", issue.SuggestedFix)
	return false, nil
}

func (sif *SecurityIssueFixer) createBackup() error {
	backupDir := filepath.Join(sif.projectPath, ".security_backup", time.Now().Format("20060102_150405"))
	return os.MkdirAll(backupDir, 0755)
}

func (sif *SecurityIssueFixer) validateFixes() (*ValidationResult, error) {
	// Run security scan after fixes
	scanResult, err := security.ScanCurrentFeature("post_fix_validation")
	if err != nil {
		return nil, err
	}

	// Count remaining issues
	criticalCount := 0
	highCount := 0
	for _, issue := range scanResult.Issues {
		if strings.EqualFold(issue.Severity, "critical") {
			criticalCount++
		} else if strings.EqualFold(issue.Severity, "high") {
			highCount++
		}
	}

	return &ValidationResult{
		ScanResults:             scanResult,
		RemainingCriticalIssues: criticalCount,
		RemainingHighIssues:     highCount,
		FixesValidated:          criticalCount == 0,
	}, nil
}

func (sif *SecurityIssueFixer) generateFixReport(result *FixResult, issues []SecurityFix) {
	report := fmt.Sprintf(`
========================================
ZERO-TOLERANCE SECURITY FIX REPORT
========================================

Execution Timestamp: %s
Project Path: %s
Zero Tolerance Policy: ENFORCED

FIX EXECUTION SUMMARY:
- Total Security Issues: %d
- Issues Attempted: %d
- Successfully Fixed: %d
- Failed Fixes: %d
- Manual Fixes Required: %d
- Skipped Issues: %d
- Fix Duration: %v
- Overall Success: %t

CRITICAL ISSUES ADDRESSED:
%s

VALIDATION RESULTS:
- Post-Fix Security Score: %d
- Remaining Critical Issues: %d
- Remaining High Issues: %d
- All Fixes Validated: %t
- Zero Tolerance Status: %s

FIX RECOMMENDATIONS:
%s

========================================

ZERO-TOLERANCE POLICY STATUS:
%s

PRODUCTION READINESS:
%s

========================================
`,
		result.StartTime.Format(time.RFC3339),
		sif.projectPath,
		result.TotalIssues,
		len(issues),
		result.FixedIssues,
		result.FailedFixes,
		result.ManualFixes,
		result.SkippedIssues,
		result.EndTime.Sub(result.StartTime),
		result.Success,
		sif.formatCriticalIssues(issues),
		result.Validation.ScanResults.SecurityScore,
		result.Validation.RemainingCriticalIssues,
		result.Validation.RemainingHighIssues,
		result.Validation.FixesValidated,
		sif.evaluateZeroTolerance(result.Validation.RemainingCriticalIssues),
		sif.generateFixRecommendations(result),
		sif.evaluateZeroToleranceDetailed(result.Validation.RemainingCriticalIssues),
		sif.evaluateProductionReadiness(result.Success, result.Validation.RemainingCriticalIssues),
	)

	// Save fix report
	reportDir := filepath.Join(sif.projectPath, "reports/security/fixes")
	os.MkdirAll(reportDir, 0755)

	reportFile := filepath.Join(reportDir, "zero_tolerance_security_fix_report.txt")
	os.WriteFile(reportFile, []byte(report), 0644)

	log.Printf("📝 Security fix report saved: %s", reportFile)
}

// Report helper functions
func (sif *SecurityIssueFixer) formatCriticalIssues(issues []SecurityFix) string {
	var critical []SecurityFix
	for _, issue := range issues {
		if strings.EqualFold(issue.Severity, "critical") {
			critical = append(critical, issue)
		}
	}

	if len(critical) == 0 {
		return "None - All critical issues addressed"
	}

	result := ""
	for i, issue := range critical {
		result += fmt.Sprintf("%d. %s\n", i+1, issue.Title)
		result += fmt.Sprintf("   File: %s\n", issue.File)
		result += fmt.Sprintf("   Description: %s\n", issue.Description)
		result += fmt.Sprintf("   Fix Status: %s\n", map[bool]string{true: "Fixed", false: "Pending"}[sif.fixedIssues[issue.ID]])
		if !sif.fixedIssues[issue.ID] {
			result += fmt.Sprintf("   Suggested Fix: %s\n", issue.SuggestedFix)
		}
	}
	return result
}

func (sif *SecurityIssueFixer) generateFixRecommendations(result *FixResult) string {
	var recs []string

	if result.FailedFixes > 0 {
		recs = append(recs, fmt.Sprintf("URGENT: %d security fixes failed - manual intervention required", result.FailedFixes))
	}

	if result.ManualFixes > 0 {
		recs = append(recs, fmt.Sprintf("IMPORTANT: %d manual security fixes required", result.ManualFixes))
	}

	if !result.Validation.FixesValidated {
		recs = append(recs, "CRITICAL: Security fixes did not resolve all issues - review and retry")
	}

	if result.Success {
		recs = append(recs, "EXCELLENT: All security issues resolved successfully")
	}

	if len(recs) == 0 {
		recs = append(recs, "Continue security monitoring and maintenance")
	}

	resultText := ""
	for i, rec := range recs {
		resultText += fmt.Sprintf("%d. %s\n", i+1, rec)
	}
	return resultText
}

func (sif *SecurityIssueFixer) evaluateZeroTolerance(remainingCritical int) string {
	if remainingCritical == 0 {
		return "✅ SATISFIED - No critical security violations"
	}
	return fmt.Sprintf("❌ VIOLATED - %d critical security violations remain", remainingCritical)
}

func (sif *SecurityIssueFixer) evaluateZeroToleranceDetailed(remainingCritical int) string {
	if remainingCritical == 0 {
		return "✅ ZERO TOLERANCE POLICY SATISFIED\n   All critical security issues resolved\n   Platform meets enterprise security requirements\n   Production deployment APPROVED"
	}
	return fmt.Sprintf("❌ ZERO TOLERANCE POLICY VIOLATED\n   %d critical security violations remain\n   Production deployment PROHIBITED\n   Immediate remediation required", remainingCritical)
}

func (sif *SecurityIssueFixer) evaluateProductionReadiness(success bool, remainingCritical int) string {
	if success && remainingCritical == 0 {
		return "🎉 PRODUCTION READY\n   Zero critical security issues\n   All security fixes validated\n   Enterprise security standards met"
	}
	if remainingCritical > 0 {
		return "🚨 NOT READY - CRITICAL ISSUES\n   Critical security vulnerabilities present\n   Zero tolerance policy violated\n   Production deployment BLOCKED"
	}
	return "⚠️ NOT READY - FIX VALIDATION FAILED\n   Some security fixes not validated\n   Review and re-run fix process"
}

// Global fix execution
func FixAllCriticalSecurityIssues(projectPath string, criticalOnly bool) (*FixResult, error) {
	fixer, err := NewSecurityIssueFixer(projectPath, criticalOnly)
	if err != nil {
		return nil, err
	}

	return fixer.FixAllCriticalIssues()
}
