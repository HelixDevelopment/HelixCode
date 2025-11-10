// Package testing provides comprehensive test integration with security scanning
package testing

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"dev.helix.code/internal/security"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// SecurityTestRunner integrates security scanning into testing workflow
type SecurityTestRunner struct {
	securityManager *security.SecurityManager
	testResults     map[string]*SecurityTestResult
	mutex           sync.RWMutex
	config          SecurityTestConfig
}

// SecurityTestConfig defines security testing behavior
type SecurityTestConfig struct {
	ScanBeforeTests     bool     `json:"scan_before_tests"`
	ScanAfterTests      bool     `json:"scan_after_tests"`
	ScanOnEachTest      bool     `json:"scan_on_each_test"`
	ScanOnTestFailure   bool     `json:"scan_on_test_failure"`
	DeepScanEnabled     bool     `json:"deep_scan_enabled"`
	FailTestOnIssues    bool     `json:"fail_test_on_issues"`
	SecurityGatePass    bool     `json:"security_gate_pass"`
	FeatureScanRequired bool     `json:"feature_scan_required"`
	ExcludedPaths       []string `json:"excluded_paths"`
	RequiredScanners    []string `json:"required_scanners"`
	ScoreThreshold      int      `json:"score_threshold"`
}

// SecurityTestResult holds results of security testing
type SecurityTestResult struct {
	ID              uuid.UUID                   `json:"id"`
	TestName        string                      `json:"test_name"`
	TestType        SecurityTestType            `json:"test_type"`
	StartTime       time.Time                   `json:"start_time"`
	EndTime         time.Time                   `json:"end_time"`
	Duration        time.Duration               `json:"duration"`
	SecurityScan    *security.FeatureScanResult `json:"security_scan"`
	TestPassed      bool                        `json:"test_passed"`
	SecurityPassed  bool                        `json:"security_passed"`
	CanProceed      bool                        `json:"can_proceed"`
	IssuesFound     int                         `json:"issues_found"`
	CriticalIssues  int                         `json:"critical_issues"`
	SecurityScore   int                         `json:"security_score"`
	Recommendations []string                    `json:"recommendations"`
}

// SecurityTestType defines types of security tests
type SecurityTestType string

const (
	UnitTestSecurity        SecurityTestType = "unit_test_security"
	IntegrationTestSecurity SecurityTestType = "integration_test_security"
	E2ETestSecurity         SecurityTestType = "e2e_test_security"
	PerformanceTestSecurity SecurityTestType = "performance_test_security"
	FeatureSecurityTest     SecurityTestType = "feature_security_test"
	BuildSecurityTest       SecurityTestType = "build_security_test"
	DeploymentSecurityTest  SecurityTestType = "deployment_security_test"
)

// NewSecurityTestRunner creates a new security test runner
func NewSecurityTestRunner(config SecurityTestConfig) (*SecurityTestRunner, error) {
	// Initialize security manager
	secManager := security.GetGlobalSecurityManager()
	if secManager == nil {
		if err := security.InitGlobalSecurityManager(); err != nil {
			return nil, errors.Wrap(err, "failed to initialize security manager")
		}
		secManager = security.GetGlobalSecurityManager()
	}

	return &SecurityTestRunner{
		securityManager: secManager,
		testResults:     make(map[string]*SecurityTestResult),
		config:          config,
	}, nil
}

// RunTestWithSecurity wraps test execution with comprehensive security scanning
func (str *SecurityTestRunner) RunTestWithSecurity(ctx context.Context, testName string, testFunc func() error, projectPath string) (*SecurityTestResult, error) {
	log.Printf("🔍 Running security-enabled test: %s", testName)

	result := &SecurityTestResult{
		ID:        uuid.New(),
		TestName:  testName,
		TestType:  determineTestType(testName),
		StartTime: time.Now(),
	}

	// Pre-test security scan if configured
	if str.config.ScanBeforeTests {
		if preScan, err := str.runSecurityScan(ctx, testName+"_pre", projectPath); err != nil {
			log.Printf("⚠️ Pre-test security scan failed: %v", err)
		} else {
			result.IssuesFound += len(preScan.Issues)
			result.CriticalIssues += countCriticalIssues(preScan.Issues)
		}
	}

	// Execute the actual test
	var testErr error
	testPassed := true
	testStartTime := time.Now()

	defer func() {
		if r := recover(); r != nil {
			testErr = fmt.Errorf("test panic: %v", r)
			testPassed = false
			log.Printf("💥 Test %s panicked: %v", testName, r)
		}
	}()

	// Run the test function
	if testFunc != nil {
		testErr = testFunc()
		testPassed = testErr == nil
	}

	testDuration := time.Since(testStartTime)

	// Post-test security scan if configured
	var postScan *security.FeatureScanResult
	if str.config.ScanAfterTests || (str.config.ScanOnTestFailure && !testPassed) || str.config.ScanOnEachTest {
		var scanErr error
		postScan, scanErr = str.runSecurityScan(ctx, testName+"_post", projectPath)
		if scanErr != nil {
			log.Printf("⚠️ Post-test security scan failed: %v", scanErr)
		}
	}

	// Combine results
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.TestPassed = testPassed
	result.SecurityScan = postScan

	if postScan != nil {
		result.SecurityPassed = postScan.Success
		result.CanProceed = postScan.CanProceed
		result.IssuesFound += len(postScan.Issues)
		result.CriticalIssues += countCriticalIssues(postScan.Issues)
		result.SecurityScore = postScan.SecurityScore
		result.Recommendations = postScan.Recommendations
	}

	// Store result
	str.mutex.Lock()
	str.testResults[testName] = result
	str.mutex.Unlock()

	// Determine overall success (for clarity)
	_ = testPassed && result.SecurityPassed

	// Generate comprehensive report
	str.generateTestReport(result, testDuration)

	// Check if we can proceed to next step
	if !result.CanProceed && str.config.FeatureScanRequired {
		return result, fmt.Errorf("security gate failed for test %s - critical issues found", testName)
	}

	log.Printf("✅ Security-enabled test completed: %s - Test: %t - Security: %t - Issues: %d",
		testName, testPassed, result.SecurityPassed, result.IssuesFound)

	return result, nil
}

// RunFeatureWithSecurity wraps feature development with deep security scanning
func (str *SecurityTestRunner) RunFeatureWithSecurity(ctx context.Context, featureName string, featureFunc func() error, projectPath string) (*SecurityTestResult, error) {
	log.Printf("🔍 Running security-enabled feature development: %s", featureName)

	result := &SecurityTestResult{
		ID:        uuid.New(),
		TestName:  featureName,
		TestType:  FeatureSecurityTest,
		StartTime: time.Now(),
	}

	// Pre-feature deep security scan
	if preScan, err := str.runSecurityScan(ctx, featureName+"_pre_feature", projectPath); err != nil {
		return nil, errors.Wrap(err, "pre-feature security scan failed")
	} else {
		result.IssuesFound += len(preScan.Issues)
		result.CriticalIssues += countCriticalIssues(preScan.Issues)

		// Check if pre-existing issues prevent feature development
		if str.config.DeepScanEnabled && result.CriticalIssues > 0 {
			return nil, fmt.Errorf("critical security issues exist - cannot proceed with feature %s", featureName)
		}
	}

	// Execute the feature development
	var featureErr error
	featureStartTime := time.Now()

	defer func() {
		if r := recover(); r != nil {
			featureErr = fmt.Errorf("feature development panic: %v", r)
			log.Printf("💥 Feature %s development panicked: %v", featureName, r)
		}
	}()

	if featureFunc != nil {
		featureErr = featureFunc()
	}

	featureDuration := time.Since(featureStartTime)

	// Post-feature comprehensive security scan
	postScan, scanErr := str.runSecurityScan(ctx, featureName+"_post_feature", projectPath)
	if scanErr != nil {
		return nil, errors.Wrap(scanErr, "post-feature security scan failed")
	}

	// Analyze new security issues introduced by feature
	newIssues := len(postScan.Issues) - result.IssuesFound
	newCritical := postScan.SecurityScore - result.SecurityScore

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.TestPassed = featureErr == nil
	result.SecurityScan = postScan
	result.SecurityPassed = postScan.Success
	result.CanProceed = postScan.CanProceed
	result.IssuesFound = len(postScan.Issues)
	result.CriticalIssues = countCriticalIssues(postScan.Issues)
	result.SecurityScore = postScan.SecurityScore
	result.Recommendations = postScan.Recommendations

	// Add specific recommendations based on feature changes
	if newIssues > 0 {
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Feature %s introduced %d new security issues", featureName, newIssues))
	}
	if newCritical > 0 {
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Feature %s reduced security score by %d points", featureName, -newCritical))
	}

	// Store result
	str.mutex.Lock()
	str.testResults[featureName] = result
	str.mutex.Unlock()

	// Generate comprehensive feature report
	str.generateFeatureReport(result, featureDuration, newIssues)

	// Security gate check
	if str.config.FeatureScanRequired && !postScan.CanProceed {
		return result, fmt.Errorf("security gate failed for feature %s - feature introduced critical issues", featureName)
	}

	// Security score check
	if str.config.ScoreThreshold > 0 && postScan.SecurityScore < str.config.ScoreThreshold {
		return result, fmt.Errorf("security score too low for feature %s: %d < %d",
			featureName, postScan.SecurityScore, str.config.ScoreThreshold)
	}

	log.Printf("✅ Security-enabled feature completed: %s - Success: %t - Security Score: %d - New Issues: %d",
		featureName, result.TestPassed, postScan.SecurityScore, newIssues)

	return result, nil
}

// runSecurityScan executes comprehensive security scan
func (str *SecurityTestRunner) runSecurityScan(ctx context.Context, scanName, projectPath string) (*security.FeatureScanResult, error) {
	if str.securityManager == nil {
		return nil, fmt.Errorf("security manager not initialized")
	}

	// Create scan context with timeout
	scanCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	return str.securityManager.ScanFeature(scanCtx, scanName, projectPath)
}

// GetSecurityTestDashboard returns comprehensive security testing dashboard
func (str *SecurityTestRunner) GetSecurityTestDashboard(ctx context.Context) (*SecurityTestDashboard, error) {
	str.mutex.RLock()
	defer str.mutex.RUnlock()

	dashboard := &SecurityTestDashboard{
		Timestamp:       time.Now(),
		TotalTests:      len(str.testResults),
		PassedTests:     0,
		FailedTests:     0,
		SecurityPassed:  0,
		SecurityFailed:  0,
		TotalIssues:     0,
		CriticalIssues:  0,
		AverageScore:    0,
		RecentTests:     make([]*SecurityTestResult, 0),
		TestTypes:       make(map[SecurityTestType]int),
		Recommendations: make([]string, 0),
	}

	var totalScore int

	for _, result := range str.testResults {
		if result.TestPassed {
			dashboard.PassedTests++
		} else {
			dashboard.FailedTests++
		}

		if result.SecurityPassed {
			dashboard.SecurityPassed++
		} else {
			dashboard.SecurityFailed++
		}

		dashboard.TotalIssues += result.IssuesFound
		dashboard.CriticalIssues += result.CriticalIssues
		totalScore += result.SecurityScore

		if _, exists := dashboard.TestTypes[result.TestType]; !exists {
			dashboard.TestTypes[result.TestType] = 0
		}
		dashboard.TestTypes[result.TestType]++

		// Add to recent tests (last 24 hours)
		if time.Since(result.EndTime) < 24*time.Hour {
			dashboard.RecentTests = append(dashboard.RecentTests, result)
		}
	}

	if len(str.testResults) > 0 {
		dashboard.AverageScore = totalScore / len(str.testResults)
	}

	// Generate recommendations
	dashboard.Recommendations = str.generateDashboardRecommendations()

	return dashboard, nil
}

// Helper functions

func (str *SecurityTestRunner) generateTestReport(result *SecurityTestResult, testDuration time.Duration) {
	report := fmt.Sprintf(`
========================================
SECURITY-ENABLED TEST REPORT
========================================

Test Name: %s
Test Type: %s
Duration: %v
Test Passed: %t
Security Passed: %t
Can Proceed: %t

Security Summary:
- Total Issues: %d
- Critical Issues: %d
- Security Score: %d

Test Duration: %v
Issues Found During Test: %d

Recommendations:
%s
========================================
`, result.TestName, result.TestType, result.Duration, result.TestPassed,
		result.SecurityPassed, result.CanProceed, result.IssuesFound,
		result.CriticalIssues, result.SecurityScore, testDuration,
		result.IssuesFound, strings.Join(result.Recommendations, "\n- "))

	// Save report to file
	reportDir := "reports/security/tests"
	os.MkdirAll(reportDir, 0755)

	reportFile := filepath.Join(reportDir, result.TestName+"_security_report.txt")
	os.WriteFile(reportFile, []byte(report), 0644)
}

func (str *SecurityTestRunner) generateFeatureReport(result *SecurityTestResult, featureDuration time.Duration, newIssues int) {
	report := fmt.Sprintf(`
========================================
SECURITY-ENABLED FEATURE REPORT
========================================

Feature Name: %s
Feature Duration: %v
Feature Success: %t
Security Passed: %t
Can Proceed: %t

Security Summary:
- Total Issues: %d
- Critical Issues: %d
- New Security Issues: %d
- Security Score: %d

Development Metrics:
- Feature Development Time: %v
- Security Scan Time: %v
- Total Time: %v

Security Analysis:
%s
========================================
`, result.TestName, featureDuration, result.TestPassed, result.SecurityPassed,
		result.CanProceed, result.IssuesFound, result.CriticalIssues, newIssues,
		result.SecurityScore, featureDuration, result.Duration-featureDuration,
		result.Duration, strings.Join(result.Recommendations, "\n- "))

	// Save report to file
	reportDir := "reports/security/features"
	os.MkdirAll(reportDir, 0755)

	reportFile := filepath.Join(reportDir, result.TestName+"_feature_security_report.txt")
	os.WriteFile(reportFile, []byte(report), 0644)
}

func (str *SecurityTestRunner) generateDashboardRecommendations() []string {
	var recommendations []string

	str.mutex.RLock()
	defer str.mutex.RUnlock()

	criticalCount := 0
	totalIssues := 0

	for _, result := range str.testResults {
		criticalCount += result.CriticalIssues
		totalIssues += result.IssuesFound
	}

	if criticalCount > 0 {
		recommendations = append(recommendations, fmt.Sprintf("URGENT: %d critical security issues found across all tests", criticalCount))
	}

	if totalIssues > 50 {
		recommendations = append(recommendations, fmt.Sprintf("IMPORTANT: %d total security issues found - consider security sprint", totalIssues))
	}

	if len(str.testResults) > 0 {
		failedCount := 0
		for _, result := range str.testResults {
			if !result.SecurityPassed {
				failedCount++
			}
		}

		if failedCount > len(str.testResults)/2 {
			recommendations = append(recommendations, "CRITICAL: More than 50% of tests have security issues - must address immediately")
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "✅ Security testing looks good - continue monitoring")
	}

	return recommendations
}

// Supporting types
type SecurityTestDashboard struct {
	Timestamp       time.Time                `json:"timestamp"`
	TotalTests      int                      `json:"total_tests"`
	PassedTests     int                      `json:"passed_tests"`
	FailedTests     int                      `json:"failed_tests"`
	SecurityPassed  int                      `json:"security_passed"`
	SecurityFailed  int                      `json:"security_failed"`
	TotalIssues     int                      `json:"total_issues"`
	CriticalIssues  int                      `json:"critical_issues"`
	AverageScore    int                      `json:"average_score"`
	RecentTests     []*SecurityTestResult    `json:"recent_tests"`
	TestTypes       map[SecurityTestType]int `json:"test_types"`
	Recommendations []string                 `json:"recommendations"`
}

// Helper functions
func determineTestType(testName string) SecurityTestType {
	testName = strings.ToLower(testName)

	if strings.Contains(testName, "unit") {
		return UnitTestSecurity
	} else if strings.Contains(testName, "integration") {
		return IntegrationTestSecurity
	} else if strings.Contains(testName, "e2e") || strings.Contains(testName, "end") {
		return E2ETestSecurity
	} else if strings.Contains(testName, "performance") || strings.Contains(testName, "perf") {
		return PerformanceTestSecurity
	} else if strings.Contains(testName, "feature") {
		return FeatureSecurityTest
	} else if strings.Contains(testName, "build") {
		return BuildSecurityTest
	} else if strings.Contains(testName, "deploy") {
		return DeploymentSecurityTest
	}

	// Default to unit test
	return UnitTestSecurity
}

func countCriticalIssues(issues []security.ScanIssue) int {
	count := 0
	for _, issue := range issues {
		if strings.EqualFold(issue.Severity, "critical") {
			count++
		}
	}
	return count
}

// Global test runner
var globalTestRunner *SecurityTestRunner

// InitGlobalSecurityTestRunner initializes global security test runner
func InitGlobalSecurityTestRunner(config SecurityTestConfig) error {
	var err error
	globalTestRunner, err = NewSecurityTestRunner(config)
	if err != nil {
		return errors.Wrap(err, "failed to initialize global security test runner")
	}

	log.Printf("✅ Global security test runner initialized")
	return nil
}

// GetGlobalSecurityTestRunner returns the global security test runner
func GetGlobalSecurityTestRunner() *SecurityTestRunner {
	if globalTestRunner == nil {
		// Initialize with default config
		defaultConfig := SecurityTestConfig{
			ScanBeforeTests:     false,
			ScanAfterTests:      true,
			ScanOnEachTest:      false,
			ScanOnTestFailure:   true,
			DeepScanEnabled:     false,
			FailTestOnIssues:    false,
			SecurityGatePass:    true,
			FeatureScanRequired: false,
			ExcludedPaths:       []string{"vendor", "test", "mock"},
			RequiredScanners:    []string{"gosec", "trivy"},
			ScoreThreshold:      70,
		}
		InitGlobalSecurityTestRunner(defaultConfig)
	}
	return globalTestRunner
}

// RunTest wraps test function with security scanning (convenience function)
func RunTest(ctx context.Context, testName string, testFunc func() error, projectPath string) (*SecurityTestResult, error) {
	runner := GetGlobalSecurityTestRunner()
	return runner.RunTestWithSecurity(ctx, testName, testFunc, projectPath)
}

// RunFeature wraps feature development with security scanning (convenience function)
func RunFeature(ctx context.Context, featureName string, featureFunc func() error, projectPath string) (*SecurityTestResult, error) {
	runner := GetGlobalSecurityTestRunner()
	return runner.RunFeatureWithSecurity(ctx, featureName, featureFunc, projectPath)
}
