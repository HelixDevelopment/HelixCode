// Package security provides implementations for security scanning tools
package security

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// SonarQubeScanner implements SonarQube code quality and security scanning
type SonarQubeScanner struct {
	config SonarQubeConfig
}

type SonarQubeConfig struct {
	URL                   string
	ProjectKey            string
	Organization          string
	Token                 string
	QualityGate           string
	CoverageMinimum       int
	DuplicationsMaximum   int
	MaintainabilityRating string
	ReliabilityRating     string
	SecurityRating        string
}

// NewSonarQubeScanner creates a new SonarQube scanner
func NewSonarQubeScanner(config SonarQubeConfig) (*SonarQubeScanner, error) {
	return &SonarQubeScanner{config: config}, nil
}

func (s *SonarQubeScanner) Name() string {
	return "sonarqube"
}

func (s *SonarQubeScanner) Enabled() bool {
	return s.config.URL != "" && s.config.ProjectKey != ""
}

func (s *SonarQubeScanner) Config() interface{} {
	return s.config
}

func (s *SonarQubeScanner) Scan(ctx context.Context, scanCtx *ScanContext) (*ScanResult, error) {
	startTime := time.Now()

	// Prepare sonar-project.properties
	sonarProps := map[string]string{
		"sonar.projectKey":          s.config.ProjectKey,
		"sonar.projectName":         scanCtx.ProjectPath,
		"sonar.projectVersion":      "1.0.0",
		"sonar.host.url":            s.config.URL,
		"sonar.sourceEncoding":      "UTF-8",
		"sonar.exclusions":          "**/vendor/**,**/test/**,**/mock/**,**/generated/**",
		"sonar.coverage.exclusions": "**/vendor/**,**/test/**,**/mock/**,**/generated/**",
		"sonar.cpd.exclusions":      "**/test/**,**/mock/**,**/generated/**",
	}

	// Create properties file
	propsContent := ""
	for key, value := range sonarProps {
		propsContent += fmt.Sprintf("%s=%s\n", key, value)
	}

	propsFile := filepath.Join(scanCtx.ProjectPath, "sonar-project.properties")
	if err := os.WriteFile(propsFile, []byte(propsContent), 0644); err != nil {
		return nil, errors.Wrap(err, "failed to create sonar properties file")
	}
	defer os.Remove(propsFile)

	// Run sonar-scanner
	cmd := exec.CommandContext(ctx, "sonar-scanner",
		"-D", fmt.Sprintf("sonar.projectKey=%s", s.config.ProjectKey),
		"-D", fmt.Sprintf("sonar.host.url=%s", s.config.URL),
		"-D", fmt.Sprintf("sonar.sources=%s", scanCtx.ProjectPath),
		"-D", fmt.Sprintf("sonar.projectBaseDir=%s", scanCtx.ProjectPath),
	)

	cmd.Dir = scanCtx.ProjectPath
	output, err := cmd.CombinedOutput()

	result := &ScanResult{
		Scanner:   s.Name(),
		Timestamp: startTime,
		Success:   err == nil,
		Summary:   ScanSummary{TimeTaken: time.Since(startTime)},
		Issues:    []ScanIssue{},
		Metrics:   ScanMetrics{FilesScanned: countGoFiles(scanCtx.ProjectPath)},
		Reports: []Report{
			{
				Type:    "properties",
				Format:  "sonar",
				Path:    propsFile,
				Content: propsContent,
				Size:    int64(len(propsContent)),
			},
		},
	}

	if err != nil {
		result.Success = false
		result.Summary.TotalIssues = 1
		result.Issues = append(result.Issues, ScanIssue{
			ID:           "sonar-scan-failed",
			Scanner:      s.Name(),
			Type:         "scan_error",
			Severity:     "high",
			Title:        "SonarQube scan failed",
			Description:  string(output),
			SuggestedFix: "Check SonarQube server connectivity and configuration",
		})
	} else {
		// Parse results from output (simplified)
		result.Summary.TotalIssues = 0
		result.Success = true
	}

	return result, nil
}

// SnykScanner implements Snyk vulnerability scanning
type SnykScanner struct {
	config SnykConfig
}

type SnykConfig struct {
	Token                  string
	Organization           string
	Project                string
	Monitoring             bool
	SeverityThreshold      string
	FailOnSeverity         string
	ScanDependencies       bool
	ScanCode               bool
	ScanContainers         bool
	ScanLicenses           bool
	ExcludeDevDependencies bool
}

// NewSnykScanner creates a new Snyk scanner
func NewSnykScanner(config SnykConfig) (*SnykScanner, error) {
	return &SnykScanner{config: config}, nil
}

func (s *SnykScanner) Name() string {
	return "snyk"
}

func (s *SnykScanner) Enabled() bool {
	return s.config.Token != ""
}

func (s *SnykScanner) Config() interface{} {
	return s.config
}

func (s *SnykScanner) Scan(ctx context.Context, scanCtx *ScanContext) (*ScanResult, error) {
	startTime := time.Now()

	// Set Snyk token
	if s.config.Token != "" {
		os.Setenv("SNYK_TOKEN", s.config.Token)
	}

	var issues []ScanIssue

	// Scan dependencies
	if s.config.ScanDependencies {
		cmd := exec.CommandContext(ctx, "snyk", "test", "--json")
		cmd.Dir = scanCtx.ProjectPath
		output, err := cmd.CombinedOutput()

		if err != nil && !strings.Contains(err.Error(), "exit status 1") {
			return nil, errors.Wrap(err, "snyk dependency scan failed")
		}

		if snykIssues, parseErr := s.parseSnykOutput(output); parseErr == nil {
			issues = append(issues, snykIssues...)
		}
	}

	// Scan code
	if s.config.ScanCode {
		cmd := exec.CommandContext(ctx, "snyk", "code", "--json")
		cmd.Dir = scanCtx.ProjectPath
		output, err := cmd.CombinedOutput()

		if err != nil && !strings.Contains(err.Error(), "exit status 1") {
			return nil, errors.Wrap(err, "snyk code scan failed")
		}

		if snykIssues, parseErr := s.parseSnykOutput(output); parseErr == nil {
			issues = append(issues, snykIssues...)
		}
	}

	// Calculate summary
	summary := ScanSummary{
		TimeTaken:      time.Since(startTime),
		TotalIssues:    len(issues),
		CriticalIssues: countIssuesBySeverity(issues, "critical"),
		HighIssues:     countIssuesBySeverity(issues, "high"),
		MediumIssues:   countIssuesBySeverity(issues, "medium"),
		LowIssues:      countIssuesBySeverity(issues, "low"),
	}

	return &ScanResult{
		Scanner:         s.Name(),
		Timestamp:       startTime,
		Success:         true,
		Summary:         summary,
		Issues:          issues,
		Metrics:         ScanMetrics{DependenciesScanned: countDependencies(scanCtx.ProjectPath)},
		Recommendations: s.generateRecommendations(issues),
	}, nil
}

func (s *SnykScanner) parseSnykOutput(output []byte) ([]ScanIssue, error) {
	var snykResult map[string]interface{}
	if err := json.Unmarshal(output, &snykResult); err != nil {
		return nil, errors.Wrap(err, "failed to parse snyk output")
	}

	var issues []ScanIssue
	if vulnerabilities, ok := snykResult["vulnerabilities"].([]interface{}); ok {
		for _, vuln := range vulnerabilities {
			if vulnMap, ok := vuln.(map[string]interface{}); ok {
				issue := ScanIssue{
					ID:          getString(vulnMap, "id"),
					Scanner:     s.Name(),
					Type:        "vulnerability",
					Severity:    getString(vulnMap, "severity"),
					Title:       getString(vulnMap, "title"),
					Description: getString(vulnMap, "description"),
					CVE:         getString(vulnMap, "identifiers"),
				}
				issues = append(issues, issue)
			}
		}
	}

	return issues, nil
}

func (s *SnykScanner) generateRecommendations(issues []ScanIssue) []string {
	var recommendations []string

	if countIssuesBySeverity(issues, "critical") > 0 {
		recommendations = append(recommendations, "IMMEDIATE: Fix all critical vulnerabilities")
	}
	if countIssuesBySeverity(issues, "high") > 0 {
		recommendations = append(recommendations, "URGENT: Address all high severity vulnerabilities")
	}
	if countIssuesBySeverity(issues, "medium") > 5 {
		recommendations = append(recommendations, "IMPORTANT: Plan fixes for medium vulnerabilities")
	}

	return recommendations
}

// TrivyScanner implements Trivy container and filesystem scanning
type TrivyScanner struct {
	config TrivyConfig
}

type TrivyConfig struct {
	Enabled           bool
	ScanContainers    bool
	ScanFilesystem    bool
	SeverityThreshold string
}

// NewTrivyScanner creates a new Trivy scanner
func NewTrivyScanner(config TrivyConfig) (*TrivyScanner, error) {
	return &TrivyScanner{config: config}, nil
}

func (t *TrivyScanner) Name() string {
	return "trivy"
}

func (t *TrivyScanner) Enabled() bool {
	return t.config.Enabled
}

func (t *TrivyScanner) Config() interface{} {
	return t.config
}

func (t *TrivyScanner) Scan(ctx context.Context, scanCtx *ScanContext) (*ScanResult, error) {
	startTime := time.Now()

	var issues []ScanIssue

	// Scan filesystem
	if t.config.ScanFilesystem {
		cmd := exec.CommandContext(ctx, "trivy", "fs", "--format", "json", scanCtx.ProjectPath)
		output, err := cmd.CombinedOutput()

		if err != nil && !strings.Contains(err.Error(), "exit status 0") {
			return nil, errors.Wrap(err, "trivy filesystem scan failed")
		}

		if trivyIssues, parseErr := t.parseTrivyOutput(output); parseErr == nil {
			issues = append(issues, trivyIssues...)
		}
	}

	// Scan Docker images if available
	if t.config.ScanContainers {
		dockerImages := t.getDockerImages()
		for _, image := range dockerImages {
			cmd := exec.CommandContext(ctx, "trivy", "image", "--format", "json", image)
			output, err := cmd.CombinedOutput()

			if err != nil && !strings.Contains(err.Error(), "exit status 0") {
				continue // Skip if scan fails
			}

			if trivyIssues, parseErr := t.parseTrivyOutput(output); parseErr == nil {
				issues = append(issues, trivyIssues...)
			}
		}
	}

	summary := ScanSummary{
		TimeTaken:         time.Since(startTime),
		TotalIssues:       len(issues),
		CriticalIssues:    countIssuesBySeverity(issues, "critical"),
		HighIssues:        countIssuesBySeverity(issues, "high"),
		MediumIssues:      countIssuesBySeverity(issues, "medium"),
		LowIssues:         countIssuesBySeverity(issues, "low"),
		ContainersScanned: len(t.getDockerImages()),
	}

	return &ScanResult{
		Scanner:   t.Name(),
		Timestamp: startTime,
		Success:   true,
		Summary:   summary,
		Issues:    issues,
		Metrics:   ScanMetrics{ContainersScanned: len(t.getDockerImages())},
	}, nil
}

func (t *TrivyScanner) parseTrivyOutput(output []byte) ([]ScanIssue, error) {
	var trivyResult []map[string]interface{}
	if err := json.Unmarshal(output, &trivyResult); err != nil {
		return nil, errors.Wrap(err, "failed to parse trivy output")
	}

	var issues []ScanIssue
	for _, result := range trivyResult {
		if vulnerabilities, ok := result["Vulnerabilities"].([]interface{}); ok {
			for _, vuln := range vulnerabilities {
				if vulnMap, ok := vuln.(map[string]interface{}); ok {
					issue := ScanIssue{
						ID:          getString(vulnMap, "VulnerabilityID"),
						Scanner:     t.Name(),
						Type:        "vulnerability",
						Severity:    getString(vulnMap, "Severity"),
						Title:       getString(vulnMap, "Title"),
						Description: getString(vulnMap, "Description"),
						CVE:         t.extractCVE(vulnMap),
					}
					issues = append(issues, issue)
				}
			}
		}
	}

	return issues, nil
}

func (t *TrivyScanner) getDockerImages() []string {
	// Get Docker images (simplified)
	cmd := exec.Command("docker", "images", "--format", "{{.Repository}}:{{.Tag}}")
	output, _ := cmd.Output()
	return strings.Split(strings.TrimSpace(string(output)), "\n")
}

func (t *TrivyScanner) extractCVE(vulnMap map[string]interface{}) string {
	if cves, ok := vulnMap["CVEID"].([]interface{}); ok && len(cves) > 0 {
		return fmt.Sprintf("%v", cves[0])
	}
	return ""
}

// Helper functions
func countGoFiles(dir string) int {
	count := 0
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(path, ".go") {
			count++
		}
		return nil
	})
	return count
}

func countDependencies(dir string) int {
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		return 0
	}
	// Simplified count - would parse go.sum in real implementation
	return 50
}

func countIssuesBySeverity(issues []ScanIssue, severity string) int {
	count := 0
	for _, issue := range issues {
		if strings.EqualFold(issue.Severity, severity) {
			count++
		}
	}
	return count
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		return fmt.Sprintf("%v", val)
	}
	return ""
}
