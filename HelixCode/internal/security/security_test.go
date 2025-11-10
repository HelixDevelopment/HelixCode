package security

import (
	"sync"
	"testing"
	"time"
)

// ========================================
// Constructor Tests
// ========================================

func TestNewSecurityManager(t *testing.T) {
	sm := NewSecurityManager()
	if sm == nil {
		t.Fatal("Expected security manager, got nil")
	}
	if sm.logger == nil {
		t.Error("Expected logger to be initialized")
	}
	if sm.scanResults == nil {
		t.Error("Expected scanResults map to be initialized")
	}
}

// ========================================
// Global Security Manager Tests
// ========================================

func TestInitGlobalSecurityManager(t *testing.T) {
	// Reset global state for testing
	globalSecurityManager = nil
	securityOnce = sync.Once{}

	err := InitGlobalSecurityManager()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if globalSecurityManager == nil {
		t.Fatal("Expected global security manager to be initialized")
	}
}

func TestGetGlobalSecurityManager(t *testing.T) {
	// Ensure it's initialized
	InitGlobalSecurityManager()

	sm := GetGlobalSecurityManager()
	if sm == nil {
		t.Fatal("Expected global security manager, got nil")
	}

	// Should return the same instance
	sm2 := GetGlobalSecurityManager()
	if sm != sm2 {
		t.Error("Expected same instance (singleton pattern)")
	}
}

func TestInitGlobalSecurityManager_OnlyOnce(t *testing.T) {
	// Reset global state
	globalSecurityManager = nil
	securityOnce = sync.Once{}

	// Call multiple times
	InitGlobalSecurityManager()
	first := globalSecurityManager

	InitGlobalSecurityManager()
	second := globalSecurityManager

	if first != second {
		t.Error("Expected singleton to be initialized only once")
	}
}

// ========================================
// ScanFeature Tests
// ========================================

func TestSecurityManager_ScanFeature(t *testing.T) {
	sm := NewSecurityManager()

	result, err := sm.ScanFeature("test-feature")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected scan result, got nil")
	}

	if result.FeatureName != "test-feature" {
		t.Errorf("Expected feature name 'test-feature', got %s", result.FeatureName)
	}

	if !result.Success {
		t.Error("Expected scan to succeed")
	}

	if !result.CanProceed {
		t.Error("Expected CanProceed to be true")
	}

	if result.SecurityScore != 95 {
		t.Errorf("Expected security score 95, got %d", result.SecurityScore)
	}

	if result.Issues == nil {
		t.Error("Expected issues slice to be initialized")
	}

	if len(result.Recommendations) == 0 {
		t.Error("Expected at least one recommendation")
	}

	if result.ScanTime < 0 {
		t.Error("Expected positive scan time")
	}

	if result.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestSecurityManager_ScanFeature_StoresResult(t *testing.T) {
	sm := NewSecurityManager()

	result, err := sm.ScanFeature("stored-feature")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that result was stored
	sm.mutex.RLock()
	storedResult, exists := sm.scanResults["stored-feature"]
	sm.mutex.RUnlock()

	if !exists {
		t.Fatal("Expected scan result to be stored")
	}

	if storedResult != result {
		t.Error("Expected stored result to match returned result")
	}
}

func TestSecurityManager_ScanFeature_MultipleScan(t *testing.T) {
	sm := NewSecurityManager()

	// Scan multiple features
	features := []string{"feature1", "feature2", "feature3"}
	for _, feature := range features {
		result, err := sm.ScanFeature(feature)
		if err != nil {
			t.Errorf("Failed to scan %s: %v", feature, err)
		}
		if result.FeatureName != feature {
			t.Errorf("Expected feature name %s, got %s", feature, result.FeatureName)
		}
	}

	// Verify all results stored
	sm.mutex.RLock()
	storedCount := len(sm.scanResults)
	sm.mutex.RUnlock()

	if storedCount != len(features) {
		t.Errorf("Expected %d stored results, got %d", len(features), storedCount)
	}
}

// ========================================
// Security Score Tests
// ========================================

func TestSecurityManager_GetSecurityScore(t *testing.T) {
	sm := NewSecurityManager()

	// Initial score should be 0
	score := sm.GetSecurityScore()
	if score != 0 {
		t.Errorf("Expected initial security score 0, got %d", score)
	}

	// Update score
	sm.UpdateSecurityMetrics(0, 0, 85)

	// Get updated score
	score = sm.GetSecurityScore()
	if score != 85 {
		t.Errorf("Expected security score 85, got %d", score)
	}
}

// ========================================
// Critical Issues Tests
// ========================================

func TestSecurityManager_GetCriticalIssues(t *testing.T) {
	sm := NewSecurityManager()

	// Initial count should be 0
	count := sm.GetCriticalIssues()
	if count != 0 {
		t.Errorf("Expected initial critical issues 0, got %d", count)
	}

	// Update critical issues
	sm.UpdateSecurityMetrics(5, 0, 70)

	// Get updated count
	count = sm.GetCriticalIssues()
	if count != 5 {
		t.Errorf("Expected critical issues 5, got %d", count)
	}
}

// ========================================
// High Issues Tests
// ========================================

func TestSecurityManager_GetHighIssues(t *testing.T) {
	sm := NewSecurityManager()

	// Initial count should be 0
	count := sm.GetHighIssues()
	if count != 0 {
		t.Errorf("Expected initial high issues 0, got %d", count)
	}

	// Update high issues
	sm.UpdateSecurityMetrics(0, 10, 80)

	// Get updated count
	count = sm.GetHighIssues()
	if count != 10 {
		t.Errorf("Expected high issues 10, got %d", count)
	}
}

// ========================================
// Zero Tolerance Tests
// ========================================

func TestSecurityManager_ValidateZeroTolerance(t *testing.T) {
	sm := NewSecurityManager()

	t.Run("no critical issues - passes", func(t *testing.T) {
		sm.UpdateSecurityMetrics(0, 5, 90)
		if !sm.ValidateZeroTolerance() {
			t.Error("Expected zero-tolerance to pass with 0 critical issues")
		}
	})

	t.Run("has critical issues - fails", func(t *testing.T) {
		sm.UpdateSecurityMetrics(1, 0, 80)
		if sm.ValidateZeroTolerance() {
			t.Error("Expected zero-tolerance to fail with critical issues")
		}
	})

	t.Run("multiple critical issues - fails", func(t *testing.T) {
		sm.UpdateSecurityMetrics(10, 5, 50)
		if sm.ValidateZeroTolerance() {
			t.Error("Expected zero-tolerance to fail with multiple critical issues")
		}
	})
}

// ========================================
// UpdateSecurityMetrics Tests
// ========================================

func TestSecurityManager_UpdateSecurityMetrics(t *testing.T) {
	sm := NewSecurityManager()

	sm.UpdateSecurityMetrics(3, 7, 75)

	// Verify all values updated
	if sm.GetCriticalIssues() != 3 {
		t.Errorf("Expected critical issues 3, got %d", sm.GetCriticalIssues())
	}

	if sm.GetHighIssues() != 7 {
		t.Errorf("Expected high issues 7, got %d", sm.GetHighIssues())
	}

	if sm.GetSecurityScore() != 75 {
		t.Errorf("Expected security score 75, got %d", sm.GetSecurityScore())
	}
}

func TestSecurityManager_UpdateSecurityMetrics_Multiple(t *testing.T) {
	sm := NewSecurityManager()

	// Update multiple times
	updates := []struct {
		critical int
		high     int
		score    int
	}{
		{5, 10, 60},
		{2, 8, 75},
		{0, 3, 95},
	}

	for _, update := range updates {
		sm.UpdateSecurityMetrics(update.critical, update.high, update.score)

		if sm.GetCriticalIssues() != update.critical {
			t.Errorf("Expected critical issues %d, got %d", update.critical, sm.GetCriticalIssues())
		}
		if sm.GetHighIssues() != update.high {
			t.Errorf("Expected high issues %d, got %d", update.high, sm.GetHighIssues())
		}
		if sm.GetSecurityScore() != update.score {
			t.Errorf("Expected security score %d, got %d", update.score, sm.GetSecurityScore())
		}
	}
}

// ========================================
// Concurrency Tests
// ========================================

func TestSecurityManager_Concurrency_ScanFeature(t *testing.T) {
	sm := NewSecurityManager()
	const numGoroutines = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			featureName := "concurrent-feature-" + string(rune('0'+id))
			_, err := sm.ScanFeature(featureName)
			if err != nil {
				t.Errorf("Goroutine %d failed: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all scans were stored
	sm.mutex.RLock()
	count := len(sm.scanResults)
	sm.mutex.RUnlock()

	if count != numGoroutines {
		t.Errorf("Expected %d scan results, got %d", numGoroutines, count)
	}
}

func TestSecurityManager_Concurrency_UpdateAndRead(t *testing.T) {
	sm := NewSecurityManager()
	const numGoroutines = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // readers and writers

	// Writers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			sm.UpdateSecurityMetrics(id, id*2, id*10)
		}(i)
	}

	// Readers
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			sm.GetCriticalIssues()
			sm.GetHighIssues()
			sm.GetSecurityScore()
			sm.ValidateZeroTolerance()
		}()
	}

	wg.Wait()
	// If we reach here without deadlock, the test passes
}

// ========================================
// FeatureScanResult Tests
// ========================================

func TestFeatureScanResult_Fields(t *testing.T) {
	now := time.Now()
	scanTime := 100 * time.Millisecond

	result := &FeatureScanResult{
		FeatureName:     "test-feature",
		Success:         true,
		CanProceed:      true,
		SecurityScore:   90,
		Issues:          []interface{}{"issue1", "issue2"},
		Recommendations: []string{"rec1", "rec2"},
		ScanTime:        scanTime,
		Timestamp:       now,
	}

	if result.FeatureName != "test-feature" {
		t.Error("FeatureName field mismatch")
	}
	if !result.Success {
		t.Error("Success field mismatch")
	}
	if !result.CanProceed {
		t.Error("CanProceed field mismatch")
	}
	if result.SecurityScore != 90 {
		t.Error("SecurityScore field mismatch")
	}
	if len(result.Issues) != 2 {
		t.Error("Issues field mismatch")
	}
	if len(result.Recommendations) != 2 {
		t.Error("Recommendations field mismatch")
	}
	if result.ScanTime != scanTime {
		t.Error("ScanTime field mismatch")
	}
	if !result.Timestamp.Equal(now) {
		t.Error("Timestamp field mismatch")
	}
}

// ========================================
// Edge Cases
// ========================================

func TestSecurityManager_ScanFeature_EmptyName(t *testing.T) {
	sm := NewSecurityManager()

	result, err := sm.ScanFeature("")
	if err != nil {
		t.Errorf("Expected no error for empty feature name, got %v", err)
	}
	if result.FeatureName != "" {
		t.Error("Expected empty feature name to be preserved")
	}
}

func TestSecurityManager_UpdateSecurityMetrics_NegativeValues(t *testing.T) {
	sm := NewSecurityManager()

	// Test with negative values (edge case)
	sm.UpdateSecurityMetrics(-1, -1, -10)

	if sm.GetCriticalIssues() != -1 {
		t.Error("Expected negative critical issues to be stored as-is")
	}
	if sm.GetHighIssues() != -1 {
		t.Error("Expected negative high issues to be stored as-is")
	}
	if sm.GetSecurityScore() != -10 {
		t.Error("Expected negative score to be stored as-is")
	}
}

func TestSecurityManager_UpdateSecurityMetrics_ZeroValues(t *testing.T) {
	sm := NewSecurityManager()

	// Set non-zero values first
	sm.UpdateSecurityMetrics(5, 10, 80)

	// Reset to zero
	sm.UpdateSecurityMetrics(0, 0, 0)

	if sm.GetCriticalIssues() != 0 {
		t.Error("Expected critical issues to be reset to 0")
	}
	if sm.GetHighIssues() != 0 {
		t.Error("Expected high issues to be reset to 0")
	}
	if sm.GetSecurityScore() != 0 {
		t.Error("Expected security score to be reset to 0")
	}
}
