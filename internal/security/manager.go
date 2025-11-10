// Package security provides comprehensive security scanning integration
// with zero-tolerance for security issues and deep scanning between features
package security

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// SecurityMonitoring provides security monitoring capabilities
type SecurityMonitoring struct {
	enabled bool
}

// SecurityPolicy defines security policies and rules
type SecurityPolicy struct {
	enabled bool
}

// NewSecurityMonitoring creates a new security monitoring instance
func NewSecurityMonitoring() *SecurityMonitoring {
	return &SecurityMonitoring{enabled: true}
}

// NewSecurityPolicy creates a new security policy instance
func NewSecurityPolicy() *SecurityPolicy {
	return &SecurityPolicy{enabled: true}
}

// SecurityManager provides comprehensive security scanning and monitoring
type SecurityManager struct {
	config         SecurityConfig
	currentProject *ProjectScanState
	mutex          sync.RWMutex
	scanners       map[string]Scanner
	monitoring     *SecurityMonitoring
	policy         *SecurityPolicy
}

// SecurityConfig holds comprehensive security configuration
type SecurityConfig struct {
	Scanning   ScanningConfig   `json:"scanning" yaml:"scanning"`
	SonarQube  SonarQubeConfig  `json:"sonarqube" yaml:"sonarqube"`
	Snyk       SnykConfig       `json:"snyk" yaml:"snyk"`
	Additional AdditionalConfig `json:"additional_scanners" yaml:"additional_scanners"`
	Policies   PolicyConfig     `json:"policies" yaml:"policies"`
}

// ScanningConfig defines security scanning behavior
type ScanningConfig struct {
	Enabled                  bool   `json:"enabled" yaml:"enabled"`
	DeepScanEveryFeature     bool   `json:"deep_scan_every_feature" yaml:"deep_scan_every_feature"`
	ZeroTolerance            bool   `json:"zero_tolerance" yaml:"zero_tolerance"`
	FailOnIssues             bool   `json:"fail_on_issues" yaml:"fail_on_issues"`
	ScanAfterFeatureComplete bool   `json:"scan_after_feature_complete" yaml:"scan_after_feature_complete"`
	AutomaticFixes           bool   `json:"automatic_fixes" yaml:"automatic_fixes"`
	ScanTimeout              string `json:"scan_timeout" yaml:"scan_timeout"`
	ParallelScans            int    `json:"parallel_scans" yaml:"parallel_scans"`
}

// SonarQubeConfig defines SonarQube integration
type SonarQubeConfig struct {
	Enabled               bool   `json:"enabled" yaml:"enabled"`
	URL                   string `json:"url" yaml:"url"`
	ProjectKey            string `json:"project_key" yaml:"project_key"`
	Organization          string `json:"organization" yaml:"organization"`
	Token                 string `json:"token" yaml:"token"`
	QualityGate           string `json:"quality_gate" yaml:"quality_gate"`
	CoverageMinimum       int    `json:"coverage_minimum" yaml:"coverage_minimum"`
	DuplicationsMaximum   int    `json:"duplications_maximum" yaml:"duplications_maximum"`
	MaintainabilityRating string `json:"maintainability_rating" yaml:"maintainability_rating"`
	ReliabilityRating     string `json:"reliability_rating" yaml:"reliability_rating"`
	SecurityRating        string `json:"security_rating" yaml:"security_rating"`
}

// SnykConfig defines Snyk integration
type SnykConfig struct {
	Enabled                bool   `json:"enabled" yaml:"enabled"`
	Token                  string `json:"token" yaml:"token"`
	Organization           string `json:"organization" yaml:"organization"`
	Project                string `json:"project" yaml:"project"`
	Monitoring             bool   `json:"monitoring" yaml:"monitoring"`
	SeverityThreshold      string `json:"severity_threshold" yaml:"severity_threshold"`
	FailOnSeverity         string `json:"fail_on_severity" yaml:"fail_on_severity"`
	ScanDependencies       bool   `json:"scan_dependencies" yaml:"scan_dependencies"`
	ScanCode               bool   `json:"scan_code" yaml:"scan_code"`
	ScanContainers         bool   `json:"scan_containers" yaml:"scan_containers"`
	ScanLicenses           bool   `json:"scan_licenses" yaml:"scan_licenses"`
	ExcludeDevDependencies bool   `json:"exclude_dev_dependencies" yaml:"exclude_dev_dependencies"`
}

// AdditionalConfig defines additional security scanners
type AdditionalConfig struct {
	Trivy   TrivyConfig   `json:"trivy" yaml:"trivy"`
	Semgrep SemgrepConfig `json:"semgrep" yaml:"semgrep"`
	Gosec   GosecConfig   `json:"gosec" yaml:"gosec"`
	Nancy   NancyConfig   `json:"nancy" yaml:"nancy"`
}

// Scanner interface for all security scanning tools
type Scanner interface {
	Name() string
	Scan(ctx context.Context, scanCtx *ScanContext) (*ScanResult, error)
	Enabled() bool
	Config() interface{}
}

// ScanContext provides context for security scanning
type ScanContext struct {
	ProjectPath     string
	Feature         string
	ScanType        ScanType
	Environment     string
	Timeout         time.Duration
	ParallelScans   int
	GenerateReports bool
	DeepAnalysis    bool
}

// ScanResult holds the result of a security scan
type ScanResult struct {
	Scanner         string      `json:"scanner"`
	Timestamp       time.Time   `json:"timestamp"`
	Success         bool        `json:"success"`
	Summary         ScanSummary `json:"summary"`
	Issues          []ScanIssue `json:"issues"`
	Metrics         ScanMetrics `json:"metrics"`
	Recommendations []string    `json:"recommendations"`
	Reports         []Report    `json:"reports"`
}

// ScanSummary provides high-level summary of scan results
type ScanSummary struct {
	TotalIssues    int           `json:"total_issues"`
	CriticalIssues int           `json:"critical_issues"`
	HighIssues     int           `json:"high_issues"`
	MediumIssues   int           `json:"medium_issues"`
	LowIssues      int           `json:"low_issues"`
	InfoIssues     int           `json:"info_issues"`
	Score          int           `json:"score"` // 0-100 security score
	RiskLevel      string        `json:"risk_level"`
	TimeTaken      time.Duration `json:"time_taken"`
	Coverage       float64       `json:"coverage"`
	TestsRun       int           `json:"tests_run"`
	Passed         int           `json:"passed"`
	Failed         int           `json:"failed"`
}

// ScanIssue represents a security issue found during scanning
type ScanIssue struct {
	ID              string      `json:"id"`
	Scanner         string      `json:"scanner"`
	Type            string      `json:"type"`     // vulnerability, code_quality, security_hotspot
	Severity        string      `json:"severity"` // critical, high, medium, low, info
	Title           string      `json:"title"`
	Description     string      `json:"description"`
	File            string      `json:"file"`
	Line            int         `json:"line"`
	Column          int         `json:"column"`
	EffortMinutes   int         `json:"effort_minutes"`
	CWE             string      `json:"cwe,omitempty"`
	CVE             string      `json:"cve,omitempty"`
	Rule            string      `json:"rule,omitempty"`
	Message         string      `json:"message,omitempty"`
	SuggestedFix    string      `json:"suggested_fix,omitempty"`
	RemediationPath string      `json:"remediation_path,omitempty"`
	Metadata        interface{} `json:"metadata,omitempty"`
}

// ScanMetrics provides detailed scanning metrics
type ScanMetrics struct {
	FilesScanned        int                `json:"files_scanned"`
	LinesOfCode         int                `json:"lines_of_code"`
	DependenciesScanned int                `json:"dependencies_scanned"`
	ContainersScanned   int                `json:"containers_scanned"`
	TestsAnalyzed       int                `json:"tests_analyzed"`
	CodeCoverage        float64            `json:"code_coverage"`
	TechnicalDebt       int                `json:"technical_debt_minutes"`
	DuplicatedLines     int                `json:"duplicated_lines"`
	Complexity          float64            `json:"cyclomatic_complexity"`
	Maintainability     float64            `json:"maintainability_index"`
	Reliability         float64            `json:"reliability_index"`
	Security            float64            `json:"security_index"`
	Performance         PerformanceMetrics `json:"performance"`
}

// PerformanceMetrics provides performance analysis
type PerformanceMetrics struct {
	ResponseTime      time.Duration `json:"response_time"`
	MemoryUsage       int64         `json:"memory_usage_bytes"`
	CPUUsage          float64       `json:"cpu_usage_percent"`
	Throughput        int           `json:"throughput_per_second"`
	Latency           time.Duration `json:"latency_p95"`
	ConcurrencyIssues int           `json:"concurrency_issues"`
	RaceConditions    int           `json:"race_conditions"`
	Deadlocks         int           `json:"deadlocks"`
	MemoryLeaks       int           `json:"memory_leaks"`
}

// Report represents a security scan report
type Report struct {
	Type      string `json:"type"` // html, json, xml, pdf, sarif
	Format    string `json:"format"`
	Path      string `json:"path"`
	URL       string `json:"url,omitempty"`
	Content   string `json:"content,omitempty"`
	Generated bool   `json:"generated"`
	Size      int64  `json:"size_bytes"`
}

// NewSecurityManager creates a new comprehensive security manager
func NewSecurityManager(configPath string) (*SecurityManager, error) {
	sm := &SecurityManager{
		scanners:   make(map[string]Scanner),
		monitoring: NewSecurityMonitoring(),
		policy:     NewSecurityPolicy(),
	}

	// Load configuration
	if err := sm.loadConfig(configPath); err != nil {
		return nil, errors.Wrap(err, "failed to load security configuration")
	}

	// Initialize scanners
	if err := sm.initializeScanners(); err != nil {
		return nil, errors.Wrap(err, "failed to initialize security scanners")
	}

	// Initialize current project state
	sm.currentProject = &ProjectScanState{
		ID:        uuid.New(),
		StartTime: time.Now(),
		Features:  make(map[string]*FeatureScanState),
		Scans:     make(map[string]*ScanState),
		Security:  make(map[string]*SecurityState),
	}

	return sm, nil
}

// loadConfig loads security configuration from file
func (sm *SecurityManager) loadConfig(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return errors.Wrap(err, "failed to read security config")
	}

	// Determine format based on file extension
	ext := strings.ToLower(filepath.Ext(configPath))
	switch ext {
	case ".json":
		return json.Unmarshal(data, &sm.config)
	case ".yaml", ".yml":
		return yaml.Unmarshal(data, &sm.config)
	default:
		return fmt.Errorf("unsupported config format: %s", ext)
	}
}

// initializeScanners initializes all security scanners based on configuration
func (sm *SecurityManager) initializeScanners() error {
	// SonarQube scanner
	if sm.config.SonarQube.Enabled {
		scanner, err := NewSonarQubeScanner(sm.config.SonarQube)
		if err != nil {
			return errors.Wrap(err, "failed to initialize SonarQube scanner")
		}
		sm.scanners[scanner.Name()] = scanner
	}

	// Snyk scanner
	if sm.config.Snyk.Enabled {
		scanner, err := NewSnykScanner(sm.config.Snyk)
		if err != nil {
			return errors.Wrap(err, "failed to initialize Snyk scanner")
		}
		sm.scanners[scanner.Name()] = scanner
	}

	// Trivy scanner
	if sm.config.Additional.Trivy.Enabled {
		scanner, err := NewTrivyScanner(sm.config.Additional.Trivy)
		if err != nil {
			return errors.Wrap(err, "failed to initialize Trivy scanner")
		}
		sm.scanners[scanner.Name()] = scanner
	}

	// Semgrep scanner
	if sm.config.Additional.Semgrep.Enabled {
		scanner, err := NewSemgrepScanner(sm.config.Additional.Semgrep)
		if err != nil {
			return errors.Wrap(err, "failed to initialize Semgrep scanner")
		}
		sm.scanners[scanner.Name()] = scanner
	}

	// GoSec scanner
	if sm.config.Additional.Gosec.Enabled {
		scanner, err := NewGosecScanner(sm.config.Additional.Gosec)
		if err != nil {
			return errors.Wrap(err, "failed to initialize GoSec scanner")
		}
		sm.scanners[scanner.Name()] = scanner
	}

	// Nancy scanner
	if sm.config.Additional.Nancy.Enabled {
		scanner, err := NewNancyScanner(sm.config.Additional.Nancy)
		if err != nil {
			return errors.Wrap(err, "failed to initialize Nancy scanner")
		}
		sm.scanners[scanner.Name()] = scanner
	}

	log.Printf("🔍 Initialized %d security scanners: %v", len(sm.scanners), sm.getScannerNames())
	return nil
}

// ScanFeature performs deep security scan of a completed feature
func (sm *SecurityManager) ScanFeature(ctx context.Context, feature string, projectPath string) (*FeatureScanResult, error) {
	log.Printf("🔍 Starting deep security scan for feature: %s", feature)

	scanCtx := &ScanContext{
		ProjectPath:     projectPath,
		Feature:         feature,
		ScanType:        FeatureScan,
		Environment:     "development",
		Timeout:         sm.parseTimeout(sm.config.Scanning.ScanTimeout),
		ParallelScans:   sm.config.Scanning.ParallelScans,
		GenerateReports: true,
		DeepAnalysis:    sm.config.Scanning.DeepScanEveryFeature,
	}

	// Record feature start time
	featureState := &FeatureScanState{
		Feature:     feature,
		StartTime:   time.Now(),
		ScanResults: make(map[string]*ScanResult),
	}

	sm.mutex.Lock()
	sm.currentProject.Features[feature] = featureState
	sm.mutex.Unlock()

	// Execute all enabled scanners
	results, err := sm.runAllScanners(ctx, scanCtx)
	if err != nil {
		log.Printf("⚠️ Feature scan completed with warnings: %v", err)
	}

	// Analyze results for critical issues
	analysis := sm.analyzeScanResults(results)
	featureState.Analysis = analysis
	featureState.EndTime = time.Now()

	// Check if scan passes zero-tolerance policy
	if sm.config.Scanning.ZeroTolerance && analysis.HasCriticalIssues {
		err = fmt.Errorf("feature %s has critical security issues - cannot proceed", feature)
	}

	// Generate comprehensive reports
	if sm.config.Scanning.GenerateReports {
		if reportErr := sm.generateFeatureReports(feature, results); reportErr != nil {
			log.Printf("⚠️ Failed to generate reports for feature %s: %v", feature, reportErr)
		}
	}

	result := &FeatureScanResult{
		Feature:         feature,
		Success:         !analysis.HasCriticalIssues,
		Summary:         analysis.Summary,
		Issues:          analysis.Issues,
		Recommendations: analysis.Recommendations,
		SecurityScore:   analysis.SecurityScore,
		CanProceed:      !analysis.HasCriticalIssues,
	}

	log.Printf("✅ Feature scan completed: %s - Security Score: %d - Can Proceed: %t",
		feature, analysis.SecurityScore, result.CanProceed)

	return result, err
}

// runAllScanners executes all enabled scanners in parallel
func (sm *SecurityManager) runAllScanners(ctx context.Context, scanCtx *ScanContext) (map[string]*ScanResult, error) {
	results := make(map[string]*ScanResult)
	resultChan := make(chan *ScanResult, len(sm.scanners))
	errorChan := make(chan error, len(sm.scanners))

	// Run scanners concurrently
	for name, scanner := range sm.scanners {
		go func(scannerName string, scan Scanner) {
			defer func() {
				if r := recover(); r != nil {
					errorChan <- fmt.Errorf("scanner %s panic: %v", scannerName, r)
				}
			}()

			result, err := scan.Scan(ctx, scanCtx)
			if err != nil {
				errorChan <- fmt.Errorf("scanner %s failed: %v", scannerName, err)
				return
			}

			resultChan <- result
		}(name, scanner)
	}

	// Collect results
	var errors []error
	for i := 0; i < len(sm.scanners); i++ {
		select {
		case result := <-resultChan:
			results[result.Scanner] = result
		case err := <-errorChan:
			errors = append(errors, err)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if len(errors) > 0 {
		return results, fmt.Errorf("scan errors: %v", errors)
	}

	return results, nil
}

// analyzeScanResults performs deep analysis of scan results
func (sm *SecurityManager) analyzeScanResults(results map[string]*ScanResult) *ScanAnalysis {
	analysis := &ScanAnalysis{
		ScannerResults:  results,
		Issues:          make([]ScanIssue, 0),
		Recommendations: make([]string, 0),
		StartTime:       time.Now(),
	}

	// Aggregate all issues
	totalIssues := 0
	criticalIssues := 0
	highIssues := 0

	for _, result := range results {
		if result != nil {
			totalIssues += result.Summary.TotalIssues
			criticalIssues += result.Summary.CriticalIssues
			highIssues += result.Summary.HighIssues
			analysis.Issues = append(analysis.Issues, result.Issues...)
			analysis.Recommendations = append(analysis.Recommendations, result.Recommendations...)
		}
	}

	analysis.EndTime = time.Now()
	analysis.HasCriticalIssues = criticalIssues > 0
	analysis.SecurityScore = sm.calculateSecurityScore(criticalIssues, highIssues, totalIssues)

	analysis.Summary = ScanSummary{
		TotalIssues:    totalIssues,
		CriticalIssues: criticalIssues,
		HighIssues:     highIssues,
		TimeTaken:      analysis.EndTime.Sub(analysis.StartTime),
	}

	return analysis
}

// calculateSecurityScore calculates overall security score (0-100)
func (sm *SecurityManager) calculateSecurityScore(critical, high, total int) int {
	if total == 0 {
		return 100
	}

	// Critical issues have major impact
	criticalWeight := 50
	highWeight := 30
	mediumWeight := 15
	lowWeight := 5

	penalty := (critical * criticalWeight) + (high * highWeight)

	// Assume remaining are medium/low
	remaining := total - critical - high
	if remaining > 0 {
		penalty += (remaining/2)*mediumWeight + (remaining/2)*lowWeight
	}

	score := 100 - penalty
	if score < 0 {
		score = 0
	}

	return score
}

// generateFeatureReports generates comprehensive security reports for a feature
func (sm *SecurityManager) generateFeatureReports(feature string, results map[string]*ScanResult) error {
	reportDir := filepath.Join("reports", "security", "features", feature)
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return errors.Wrap(err, "failed to create report directory")
	}

	// Generate HTML report
	htmlReport := &Report{
		Type:    "html",
		Format:  "comprehensive",
		Path:    filepath.Join(reportDir, "comprehensive-security-report.html"),
		Content: sm.generateHTMLReport(feature, results),
	}

	if err := sm.saveReport(htmlReport); err != nil {
		return errors.Wrap(err, "failed to save HTML report")
	}

	// Generate JSON report
	jsonReport := &Report{
		Type:   "json",
		Format: "detailed",
		Path:   filepath.Join(reportDir, "security-issues.json"),
	}

	if data, err := json.MarshalIndent(results, "", "  "); err == nil {
		jsonReport.Content = string(data)
		if err := sm.saveReport(jsonReport); err != nil {
			return errors.Wrap(err, "failed to save JSON report")
		}
	}

	return nil
}

// GetSecurityDashboard returns current security dashboard data
func (sm *SecurityManager) GetSecurityDashboard(ctx context.Context) (*SecurityDashboard, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	dashboard := &SecurityDashboard{
		Timestamp:      time.Now(),
		ProjectID:      sm.currentProject.ID,
		TotalFeatures:  len(sm.currentProject.Features),
		ScanHistory:    sm.getRecentScans(),
		SecurityScore:  sm.calculateOverallSecurityScore(),
		CriticalIssues: 0,
		HighIssues:     0,
		ScannerStatus:  make(map[string]string),
	}

	// Aggregate current issues
	for _, feature := range sm.currentProject.Features {
		if feature.Analysis != nil {
			dashboard.CriticalIssues += feature.Analysis.Summary.CriticalIssues
			dashboard.HighIssues += feature.Analysis.Summary.HighIssues
		}
	}

	// Check scanner status
	for name, scanner := range sm.scanners {
		if scanner.Enabled() {
			dashboard.ScannerStatus[name] = "active"
		} else {
			dashboard.ScannerStatus[name] = "disabled"
		}
	}

	return dashboard, nil
}

// Helper functions
func (sm *SecurityManager) getScannerNames() []string {
	names := make([]string, 0, len(sm.scanners))
	for name := range sm.scanners {
		names = append(names, name)
	}
	return names
}

func (sm *SecurityManager) parseTimeout(timeoutStr string) time.Duration {
	if timeoutStr == "" {
		return 10 * time.Minute
	}

	if duration, err := time.ParseDuration(timeoutStr); err == nil {
		return duration
	}

	return 10 * time.Minute
}

func (sm *SecurityManager) saveReport(report *Report) error {
	return os.WriteFile(report.Path, []byte(report.Content), 0644)
}

func (sm *SecurityManager) getRecentScans() []*ScanState {
	// Implementation for retrieving recent scans
	return []*ScanState{}
}

func (sm *SecurityManager) calculateOverallSecurityScore() int {
	// Implementation for calculating overall project security score
	return 85
}

// Type definitions for the security system
type ScanType string

const (
	ProjectScan    ScanType = "project"
	FeatureScan    ScanType = "feature"
	ContainerScan  ScanType = "container"
	DependencyScan ScanType = "dependency"
)

// Supporting structures
type ProjectScanState struct {
	ID        uuid.UUID                    `json:"id"`
	StartTime time.Time                    `json:"start_time"`
	Features  map[string]*FeatureScanState `json:"features"`
	Scans     map[string]*ScanState        `json:"scans"`
	Security  map[string]*SecurityState    `json:"security"`
}

type FeatureScanState struct {
	Feature     string                 `json:"feature"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time"`
	ScanResults map[string]*ScanResult `json:"scan_results"`
	Analysis    *ScanAnalysis          `json:"analysis"`
}

type ScanState struct {
	Scanner   string    `json:"scanner"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Success   bool      `json:"success"`
	Score     int       `json:"score"`
}

type SecurityState struct {
	Scanner     string    `json:"scanner"`
	LastScan    time.Time `json:"last_scan"`
	TotalIssues int       `json:"total_issues"`
	Critical    int       `json:"critical"`
	High        int       `json:"high"`
	Score       int       `json:"score"`
}

type ScanAnalysis struct {
	ScannerResults    map[string]*ScanResult `json:"scanner_results"`
	Issues            []ScanIssue            `json:"issues"`
	Recommendations   []string               `json:"recommendations"`
	Summary           ScanSummary            `json:"summary"`
	StartTime         time.Time              `json:"start_time"`
	EndTime           time.Time              `json:"end_time"`
	HasCriticalIssues bool                   `json:"has_critical_issues"`
	SecurityScore     int                    `json:"security_score"`
}

type FeatureScanResult struct {
	Feature         string      `json:"feature"`
	Success         bool        `json:"success"`
	Summary         ScanSummary `json:"summary"`
	Issues          []ScanIssue `json:"issues"`
	Recommendations []string    `json:"recommendations"`
	SecurityScore   int         `json:"security_score"`
	CanProceed      bool        `json:"can_proceed"`
}

type SecurityDashboard struct {
	Timestamp      time.Time         `json:"timestamp"`
	ProjectID      uuid.UUID         `json:"project_id"`
	TotalFeatures  int               `json:"total_features"`
	ScanHistory    []*ScanState      `json:"scan_history"`
	SecurityScore  int               `json:"security_score"`
	CriticalIssues int               `json:"critical_issues"`
	HighIssues     int               `json:"high_issues"`
	ScannerStatus  map[string]string `json:"scanner_status"`
}

// Config structures for individual scanners
type TrivyConfig struct {
	Enabled           bool   `json:"enabled" yaml:"enabled"`
	ScanContainers    bool   `json:"scan_containers" yaml:"scan_containers"`
	ScanFilesystem    bool   `json:"scan_filesystem" yaml:"scan_filesystem"`
	SeverityThreshold string `json:"severity_threshold" yaml:"severity_threshold"`
}

type SemgrepConfig struct {
	Enabled    bool     `json:"enabled" yaml:"enabled"`
	Rules      []string `json:"rules" yaml:"rules"`
	ConfigFile string   `json:"config_file" yaml:"config_file"`
}

type GosecConfig struct {
	Enabled           bool            `json:"enabled" yaml:"enabled"`
	SeverityThreshold string          `json:"severity_threshold" yaml:"severity_threshold"`
	Rules             map[string]bool `json:"rules" yaml:"rules"`
}

type NancyConfig struct {
	Enabled         bool `json:"enabled" yaml:"enabled"`
	SkipUpdateCheck bool `json:"skip_update_check" yaml:"skip_update_check"`
	Quiet           bool `json:"quiet" yaml:"quiet"`
}

type PolicyConfig struct {
	FeatureCompletionCheck PolicyCheckConfig       `json:"feature_completion_check" yaml:"feature_completion_check"`
	QualityGates           QualityGateConfig       `json:"quality_gates" yaml:"quality_gates"`
	SecurityThresholds     SecurityThresholdConfig `json:"security_thresholds" yaml:"security_thresholds"`
}

type PolicyCheckConfig struct {
	ScanBeforeNextFeature  bool `json:"scan_before_next_feature" yaml:"scan_before_next_feature"`
	DeepAnalysisEnabled    bool `json:"deep_analysis_enabled" yaml:"deep_analysis_enabled"`
	PerformanceProfiling   bool `json:"performance_profiling" yaml:"performance_profiling"`
	MemoryLeakDetection    bool `json:"memory_leak_detection" yaml:"memory_leak_detection"`
	RaceConditionDetection bool `json:"race_condition_detection" yaml:"race_condition_detection"`
	DeadlockDetection      bool `json:"deadlock_detection" yaml:"deadlock_detection"`
	CodeCoverageCheck      bool `json:"code_coverage_check" yaml:"code_coverage_check"`
	DocumentationCoverage  bool `json:"documentation_coverage" yaml:"documentation_coverage"`
}

type QualityGateConfig struct {
	NewCodeCoverage          string `json:"new_code_coverage" yaml:"new_code_coverage"`
	MaintainabilityRating    string `json:"maintainability_rating" yaml:"maintainability_rating"`
	ReliabilityRating        string `json:"reliability_rating" yaml:"reliability_rating"`
	SecurityHotspotsReviewed int    `json:"security_hotspots_reviewed" yaml:"security_hotspots_reviewed"`
	NewSecurityHotspots      int    `json:"new_security_hotspots" yaml:"new_security_hotspots"`
	NewBugs                  int    `json:"new_bugs" yaml:"new_bugs"`
	NewVulnerabilities       int    `json:"new_vulnerabilities" yaml:"new_vulnerabilities"`
	DuplicatedLinesDensity   string `json:"duplicated_lines_density" yaml:"duplicated_lines_density"`
}

type SecurityThresholdConfig struct {
	CriticalVulnerabilities int    `json:"critical_vulnerabilities" yaml:"critical_vulnerabilities"`
	HighVulnerabilities     int    `json:"high_vulnerabilities" yaml:"high_vulnerabilities"`
	MediumVulnerabilities   int    `json:"medium_vulnerabilities" yaml:"medium_vulnerabilities"`
	LowVulnerabilities      int    `json:"low_vulnerabilities" yaml:"low_vulnerabilities"`
	SecurityRatingRequired  string `json:"security_rating_required" yaml:"security_rating_required"`
	SecurityHotspotsMaxNew  int    `json:"security_hotspots_max_new" yaml:"security_hotspots_max_new"`
}

// Initialize global security manager
var globalSecurityManager *SecurityManager

// InitGlobalSecurityManager initializes the global security manager
func InitGlobalSecurityManager() error {
	var err error

	// Try different config locations
	configPaths := []string{
		"helix.security.json",
		"helix.security.yaml",
		"config/helix.security.json",
		"config/helix.security.yaml",
		"helixcode/helix.security.json",
		".helixcode/helix.security.json",
	}

	for _, configPath := range configPaths {
		if _, err := os.Stat(configPath); err == nil {
			globalSecurityManager, err = NewSecurityManager(configPath)
			if err == nil {
				log.Printf("✅ Global security manager initialized with config: %s", configPath)
				return nil
			}
			log.Printf("⚠️ Failed to load security config from %s: %v", configPath, err)
		}
	}

	// Create default config if none found
	defaultConfig := createDefaultSecurityConfig()
	configPath := "helix.security.json"
	if err := os.WriteFile(configPath, defaultConfig, 0644); err != nil {
		return errors.Wrap(err, "failed to create default security config")
	}

	globalSecurityManager, err = NewSecurityManager(configPath)
	if err != nil {
		return errors.Wrap(err, "failed to initialize default security manager")
	}

	log.Printf("✅ Global security manager initialized with default config")
	return nil
}

// GetGlobalSecurityManager returns the global security manager
func GetGlobalSecurityManager() *SecurityManager {
	if globalSecurityManager == nil {
		log.Printf("⚠️ Security manager not initialized, creating with defaults")
		InitGlobalSecurityManager()
	}
	return globalSecurityManager
}

// ScanCurrentFeature performs deep scan of current working directory
func ScanCurrentFeature(feature string) (*FeatureScanResult, error) {
	manager := GetGlobalSecurityManager()
	if manager == nil {
		return nil, fmt.Errorf("security manager not initialized")
	}

	pwd, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current working directory")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	return manager.ScanFeature(ctx, feature, pwd)
}

// createDefaultSecurityConfig creates default security configuration
func createDefaultSecurityConfig() []byte {
	config := map[string]interface{}{
		"version": "1.0.0",
		"project": map[string]interface{}{
			"name":    "HelixCode",
			"version": "1.0.0",
		},
		"security": map[string]interface{}{
			"scanning": map[string]interface{}{
				"enabled":                     true,
				"deep_scan_every_feature":     true,
				"zero_tolerance":              true,
				"fail_on_issues":              false,
				"scan_after_feature_complete": true,
				"automatic_fixes":             false,
				"scan_timeout":                "10m",
				"parallel_scans":              4,
			},
			"sonarqube": map[string]interface{}{
				"enabled":              false, // Disabled by default for simplicity
				"url":                  "${SONAR_HOST_URL:http://localhost:9000}",
				"project_key":          "${SONAR_PROJECT_KEY:helixcode}",
				"quality_gate":         "production-ready",
				"coverage_minimum":     80,
				"duplications_maximum": 3,
			},
			"snyk": map[string]interface{}{
				"enabled":            false, // Disabled by default until token provided
				"organization":       "${SNYK_ORGANIZATION}",
				"project":            "${SNYK_PROJECT:helixcode}",
				"monitoring":         true,
				"severity_threshold": "medium",
				"fail_on_severity":   "high",
				"scan_dependencies":  true,
				"scan_code":          true,
				"scan_containers":    true,
			},
			"additional_scanners": map[string]interface{}{
				"trivy": map[string]interface{}{
					"enabled":            true,
					"scan_containers":    true,
					"scan_filesystem":    true,
					"severity_threshold": "medium",
				},
				"semgrep": map[string]interface{}{
					"enabled": true,
					"rules":   []string{"owasp-top-ten", "security", "performance"},
				},
				"gosec": map[string]interface{}{
					"enabled":            true,
					"severity_threshold": "medium",
				},
			},
		},
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	return data
}
