package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// NewUsageAnalytics creates a new usage analytics system
func NewUsageAnalytics(baseDir string) *UsageAnalytics {
	analyticsDir := filepath.Join(baseDir, "analytics")
	os.MkdirAll(analyticsDir, 0755)

	analytics := &UsageAnalytics{
		modelUsageStats:    make(map[string]*ModelUsageStats),
		taskPatterns:       make(map[string]*TaskPattern),
		userPreferences:    make(map[string]*UserPreferences),
		performanceHistory: make(map[string]*PerformanceHistory),
	}

	// Load existing data
	analytics.loadAnalyticsData(analyticsDir)

	return analytics
}

// RecordModelUsage records usage statistics for a model
func (a *UsageAnalytics) RecordModelUsage(ctx context.Context, modelID, provider, userID string, metrics *UsageMetrics) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Get or create model usage stats
	stats, exists := a.modelUsageStats[modelID]
	if !exists {
		stats = &ModelUsageStats{
			ModelID:           modelID,
			TotalRequests:     0,
			AverageLatency:    0.0,
			SuccessRate:       1.0,
			UserSatisfaction:  5.0,
			PreferredBy:       []string{},
			CommonTasks:       []string{},
			PerformanceIssues: []string{},
			LastUsed:          time.Now(),
			UsageTrend:        "stable",
		}
		a.modelUsageStats[modelID] = stats
	}

	// Update statistics
	stats.TotalRequests++
	stats.LastUsed = time.Now()

	if metrics.LatencyMs > 0 {
		// Update average latency
		stats.AverageLatency = (stats.AverageLatency + metrics.LatencyMs) / 2.0
	}

	if metrics.Success {
		stats.SuccessRate = (stats.SuccessRate + 1.0) / 2.0
	} else {
		stats.SuccessRate = (stats.SuccessRate + 0.0) / 2.0
	}

	// Update user satisfaction
	if metrics.UserRating > 0 {
		stats.UserSatisfaction = (stats.UserSatisfaction + metrics.UserRating) / 2.0
	}

	// Add to preferred by if not already present
	if !contains(stats.PreferredBy, userID) {
		stats.PreferredBy = append(stats.PreferredBy, userID)
	}

	// Add task to common tasks
	if metrics.TaskType != "" && !contains(stats.CommonTasks, metrics.TaskType) {
		stats.CommonTasks = append(stats.CommonTasks, metrics.TaskType)
	}

	// Update usage trend
	a.updateUsageTrend(modelID, stats)

	// Record performance data point
	a.recordPerformanceDataPoint(modelID, provider, metrics)

	// Save analytics data
	return a.saveAnalyticsData()
}

// UsageMetrics represents metrics from a single usage event
type UsageMetrics struct {
	Timestamp      time.Time `json:"timestamp"`
	LatencyMs      float64   `json:"latency_ms"`
	Success        bool      `json:"success"`
	UserRating     float64   `json:"user_rating"`
	TaskType       string    `json:"task_type"`
	InputTokens    int64     `json:"input_tokens"`
	OutputTokens   int64     `json:"output_tokens"`
	MemoryUsage    int64     `json:"memory_usage_mb"`
	GPUUtilization float64   `json:"gpu_utilization"`
	CPUUtilization float64   `json:"cpu_utilization"`
	Provider       string    `json:"provider"`
	ModelVersion   string    `json:"model_version"`
	ErrorType      string    `json:"error_type,omitempty"`
	ErrorCode      string    `json:"error_code,omitempty"`
}

// updateUsageTrend updates the usage trend for a model
func (a *UsageAnalytics) updateUsageTrend(modelID string, stats *ModelUsageStats) {
	// Simplified trend analysis
	if stats.TotalRequests < 10 {
		stats.UsageTrend = "stable"
		return
	}

	// In a real implementation, this would analyze usage over time
	// For now, use simple heuristics
	if stats.UserSatisfaction > 4.0 && stats.SuccessRate > 0.9 {
		stats.UsageTrend = "increasing"
	} else if stats.UserSatisfaction < 3.0 || stats.SuccessRate < 0.7 {
		stats.UsageTrend = "decreasing"
	} else {
		stats.UsageTrend = "stable"
	}
}

// recordPerformanceDataPoint records a performance data point
func (a *UsageAnalytics) recordPerformanceDataPoint(modelID, provider string, metrics *UsageMetrics) {
	// Get or create performance history
	history, exists := a.performanceHistory[modelID]
	if !exists {
		history = &PerformanceHistory{
			ModelID:             modelID,
			Provider:            provider,
			TimeSeries:          []PerformanceDataPoint{},
			AverageMetrics:      nil,
			OptimizationHistory: []OptimizationRecord{},
		}
		a.performanceHistory[modelID] = history
	}

	// Create performance data point
	dataPoint := PerformanceDataPoint{
		Timestamp:       metrics.Timestamp,
		TokensPerSecond: float64(metrics.InputTokens+metrics.OutputTokens) / metrics.LatencyMs * 1000.0,
		MemoryUsage:     metrics.MemoryUsage,
		Latency:         int64(metrics.LatencyMs),
		SuccessRate:     1.0,
		UserRating:      metrics.UserRating,
		HardwareConfig: map[string]interface{}{
			"provider": provider,
			"gpu_util": metrics.GPUUtilization,
			"cpu_util": metrics.CPUUtilization,
		},
	}

	if !metrics.Success {
		dataPoint.SuccessRate = 0.0
	}

	// Add to time series
	history.TimeSeries = append(history.TimeSeries, dataPoint)

	// Keep only last 1000 data points to prevent unbounded growth
	if len(history.TimeSeries) > 1000 {
		history.TimeSeries = history.TimeSeries[1:]
	}

	// Update average metrics
	a.updateAverageMetrics(history)
}

// updateAverageMetrics updates average performance metrics
func (a *UsageAnalytics) updateAverageMetrics(history *PerformanceHistory) {
	if len(history.TimeSeries) == 0 {
		return
	}

	totalTPS := 0.0
	totalMemory := int64(0)
	totalLatency := int64(0)
	totalSuccess := 0.0
	totalRating := 0.0
	count := len(history.TimeSeries)

	for _, point := range history.TimeSeries {
		totalTPS += point.TokensPerSecond
		totalMemory += point.MemoryUsage
		totalLatency += point.Latency
		totalSuccess += point.SuccessRate
		totalRating += point.UserRating
	}

	history.AverageMetrics = &PerformanceEstimate{
		TokensPerSecond: totalTPS / float64(count),
		MemoryUsage:     totalMemory / int64(count),
		Latency:         totalLatency / int64(count),
		QualityScore:    totalRating / float64(count),
		Throughput:      int64(totalTPS / float64(count) * 60),
		CostPerMillion:  0.0, // Local models have minimal cost
	}
}

// RecordTaskPattern records a task pattern
func (a *UsageAnalytics) RecordTaskPattern(ctx context.Context, taskType, modelID string, complexity float64, metrics *UsageMetrics) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Get or create task pattern
	pattern, exists := a.taskPatterns[taskType]
	if !exists {
		pattern = &TaskPattern{
			TaskType:                taskType,
			CommonModels:            []string{},
			AverageComplexity:       complexity,
			PeakHours:               []string{},
			PerformanceRequirements: make(map[string]float64),
			RecommendedModelSizes:   []string{"7B", "13B"},
		}
		a.taskPatterns[taskType] = pattern
	}

	// Update pattern
	// Update average complexity
	pattern.AverageComplexity = (pattern.AverageComplexity + complexity) / 2.0

	// Add model if not already present
	if !contains(pattern.CommonModels, modelID) {
		pattern.CommonModels = append(pattern.CommonModels, modelID)
	}

	// Update performance requirements
	if metrics.LatencyMs > 0 {
		current, exists := pattern.PerformanceRequirements["latency"]
		if !exists {
			pattern.PerformanceRequirements["latency"] = metrics.LatencyMs
		} else {
			pattern.PerformanceRequirements["latency"] = (current + metrics.LatencyMs) / 2.0
		}
	}

	// Update peak hours (simplified)
	hour := metrics.Timestamp.Hour()
	hourStr := fmt.Sprintf("%02d:00", hour)
	if !contains(pattern.PeakHours, hourStr) {
		pattern.PeakHours = append(pattern.PeakHours, hourStr)
		// Keep only peak hours
		if len(pattern.PeakHours) > 10 {
			pattern.PeakHours = pattern.PeakHours[1:]
		}
	}

	return a.saveAnalyticsData()
}

// SetUserPreferences sets user preferences
func (a *UsageAnalytics) SetUserPreferences(ctx context.Context, prefs *UserPreferences) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.userPreferences[prefs.UserID] = prefs
	return a.saveAnalyticsData()
}

// GetUserPreferences gets user preferences
func (a *UsageAnalytics) GetUserPreferences(ctx context.Context, userID string) (*UserPreferences, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if prefs, exists := a.userPreferences[userID]; exists {
		return prefs, nil
	}

	// Return default preferences
	return &UserPreferences{
		UserID:              userID,
		PreferredProviders:  []string{"vllm", "llamacpp"},
		QualityPreference:   "balanced",
		BudgetConstraints:   make(map[string]float64),
		TaskFrequencies:     make(map[string]int),
		HardwareConstraints: make(map[string]bool),
		PrivacyRequirements: make(map[string]bool),
	}, nil
}

// GetModelUsageStats gets usage statistics for a model
func (a *UsageAnalytics) GetModelUsageStats(ctx context.Context, modelID string) (*ModelUsageStats, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if stats, exists := a.modelUsageStats[modelID]; exists {
		return stats, nil
	}

	return nil, fmt.Errorf("model %s not found", modelID)
}

// GetTopModelsByUsage gets top models by usage
func (a *UsageAnalytics) GetTopModelsByUsage(ctx context.Context, limit int) ([]*ModelUsageStats, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var models []*ModelUsageStats
	for _, stats := range a.modelUsageStats {
		models = append(models, stats)
	}

	// Sort by total requests
	for i := 0; i < len(models)-1; i++ {
		for j := i + 1; j < len(models); j++ {
			if models[i].TotalRequests < models[j].TotalRequests {
				models[i], models[j] = models[j], models[i]
			}
		}
	}

	if limit > 0 && len(models) > limit {
		models = models[:limit]
	}

	return models, nil
}

// GetTaskPatterns gets task patterns
func (a *UsageAnalytics) GetTaskPatterns(ctx context.Context) (map[string]*TaskPattern, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Return a copy to prevent concurrent modification
	copy := make(map[string]*TaskPattern)
	for k, v := range a.taskPatterns {
		copy[k] = v
	}

	return copy, nil
}

// GetPerformanceHistory gets performance history for a model
func (a *UsageAnalytics) GetPerformanceHistory(ctx context.Context, modelID string) (*PerformanceHistory, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if history, exists := a.performanceHistory[modelID]; exists {
		return history, nil
	}

	return nil, fmt.Errorf("performance history for model %s not found", modelID)
}

// RecordOptimization records an optimization attempt
func (a *UsageAnalytics) RecordOptimization(ctx context.Context, modelID, provider, optimizationType string, before, after *PerformanceEstimate, success bool, method string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Get performance history
	history, exists := a.performanceHistory[modelID]
	if !exists {
		history = &PerformanceHistory{
			ModelID:             modelID,
			Provider:            provider,
			TimeSeries:          []PerformanceDataPoint{},
			AverageMetrics:      nil,
			OptimizationHistory: []OptimizationRecord{},
		}
		a.performanceHistory[modelID] = history
	}

	// Calculate improvement
	improvement := 0.0
	if before != nil && after != nil && before.TokensPerSecond > 0 {
		improvement = ((after.TokensPerSecond - before.TokensPerSecond) / before.TokensPerSecond) * 100.0
	}

	// Create optimization record
	record := OptimizationRecord{
		Timestamp:        time.Now(),
		OptimizationType: optimizationType,
		BeforeMetrics:    before,
		AfterMetrics:     after,
		Improvement:      improvement,
		Success:          success,
		Method:           method,
	}

	// Add to history
	history.OptimizationHistory = append(history.OptimizationHistory, record)

	// Keep only last 100 records
	if len(history.OptimizationHistory) > 100 {
		history.OptimizationHistory = history.OptimizationHistory[1:]
	}

	return a.saveAnalyticsData()
}

// GenerateUsageReport generates a comprehensive usage report
func (a *UsageAnalytics) GenerateUsageReport(ctx context.Context, timeRange TimeRange) (*UsageReport, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	report := &UsageReport{
		GeneratedAt:         time.Now(),
		TimeRange:           timeRange,
		Summary:             &UsageSummary{},
		TopModels:           []*ModelUsageStats{},
		TaskAnalysis:        make(map[string]*TaskAnalysis),
		PerformanceAnalysis: &PerformanceAnalysis{},
		UserAnalysis:        &UserAnalysis{},
		Recommendations:     []string{},
	}

	// Calculate summary
	report.Summary = a.calculateUsageSummary()

	// Get top models
	topModels, _ := a.GetTopModelsByUsage(ctx, 10)
	report.TopModels = topModels

	// Analyze tasks
	report.TaskAnalysis = a.analyzeTasks()

	// Analyze performance
	report.PerformanceAnalysis = a.analyzePerformance()

	// Analyze users
	report.UserAnalysis = a.analyzeUsers()

	// Generate recommendations
	report.Recommendations = a.generateRecommendations(report)

	return report, nil
}

// UsageReport represents a comprehensive usage report
type UsageReport struct {
	GeneratedAt         time.Time                `json:"generated_at"`
	TimeRange           TimeRange                `json:"time_range"`
	Summary             *UsageSummary            `json:"summary"`
	TopModels           []*ModelUsageStats       `json:"top_models"`
	TaskAnalysis        map[string]*TaskAnalysis `json:"task_analysis"`
	PerformanceAnalysis *PerformanceAnalysis     `json:"performance_analysis"`
	UserAnalysis        *UserAnalysis            `json:"user_analysis"`
	Recommendations     []string                 `json:"recommendations"`
}

// TimeRange represents a time range for reports
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// UsageSummary provides high-level usage statistics
type UsageSummary struct {
	TotalModels         int                `json:"total_models"`
	TotalRequests       int64              `json:"total_requests"`
	AverageLatency      float64            `json:"average_latency_ms"`
	OverallSuccessRate  float64            `json:"overall_success_rate"`
	AverageSatisfaction float64            `json:"average_satisfaction"`
	MostUsedProviders   []string           `json:"most_used_providers"`
	TrendingModels      []string           `json:"trending_models"`
	PerformanceTrends   map[string]float64 `json:"performance_trends"`
}

// TaskAnalysis provides analysis of task patterns
type TaskAnalysis struct {
	TaskType          string   `json:"task_type"`
	Frequency         int64    `json:"frequency"`
	AverageComplexity float64  `json:"average_complexity"`
	PreferredModels   []string `json:"preferred_models"`
	AverageLatency    float64  `json:"average_latency_ms"`
	SuccessRate       float64  `json:"success_rate"`
	PeakHours         []string `json:"peak_hours"`
	Trends            []string `json:"trends"`
}

// PerformanceAnalysis provides performance insights
type PerformanceAnalysis struct {
	AverageTPS         float64             `json:"average_tps"`
	OptimalProviders   map[string]float64  `json:"optimal_providers"`
	BottleneckAnalysis *BottleneckAnalysis `json:"bottleneck_analysis"`
	OptimizationImpact *OptimizationImpact `json:"optimization_impact"`
	Recommendations    []string            `json:"recommendations"`
}

// BottleneckAnalysis identifies performance bottlenecks
type BottleneckAnalysis struct {
	MemoryBottleneck  bool     `json:"memory_bottleneck"`
	GPUBottleneck     bool     `json:"gpu_bottleneck"`
	CPUBottleneck     bool     `json:"cpu_bottleneck"`
	NetworkBottleneck bool     `json:"network_bottleneck"`
	AffectedModels    []string `json:"affected_models"`
	Recommendations   []string `json:"recommendations"`
}

// OptimizationImpact shows the impact of optimizations
type OptimizationImpact struct {
	TotalOptimizations int64              `json:"total_optimizations"`
	AverageImprovement float64            `json:"average_improvement"`
	SuccessfulRate     float64            `json:"successful_rate"`
	MostEffectiveTypes []string           `json:"most_effective_types"`
	ImpactByModel      map[string]float64 `json:"impact_by_model"`
}

// UserAnalysis provides user behavior insights
type UserAnalysis struct {
	TotalUsers             int64               `json:"total_users"`
	AverageRequestsPerUser float64             `json:"average_requests_per_user"`
	UserSegments           map[string]int64    `json:"user_segments"`
	PreferredProviders     map[string][]string `json:"preferred_providers"`
	UserRetention          *UserRetention      `json:"user_retention"`
	BehavioralTrends       []string            `json:"behavioral_trends"`
}

// UserRetention shows user retention metrics
type UserRetention struct {
	DailyRetention   float64 `json:"daily_retention"`
	WeeklyRetention  float64 `json:"weekly_retention"`
	MonthlyRetention float64 `json:"monthly_retention"`
	ChurnRate        float64 `json:"churn_rate"`
}

// Helper methods for analytics

func (a *UsageAnalytics) calculateUsageSummary() *UsageSummary {
	summary := &UsageSummary{
		TotalModels:         len(a.modelUsageStats),
		TotalRequests:       0,
		AverageLatency:      0.0,
		OverallSuccessRate:  0.0,
		AverageSatisfaction: 0.0,
		MostUsedProviders:   []string{},
		TrendingModels:      []string{},
		PerformanceTrends:   make(map[string]float64),
	}

	// Calculate aggregates
	totalLatency := 0.0
	totalSuccess := 0.0
	totalSatisfaction := 0.0
	count := 0

	for _, stats := range a.modelUsageStats {
		summary.TotalRequests += stats.TotalRequests
		totalLatency += stats.AverageLatency
		totalSuccess += stats.SuccessRate
		totalSatisfaction += stats.UserSatisfaction
		count++

		// Track trending models
		if stats.UsageTrend == "increasing" {
			summary.TrendingModels = append(summary.TrendingModels, stats.ModelID)
		}
	}

	if count > 0 {
		summary.AverageLatency = totalLatency / float64(count)
		summary.OverallSuccessRate = totalSuccess / float64(count)
		summary.AverageSatisfaction = totalSatisfaction / float64(count)
	}

	// Most used providers would be calculated from performance history
	summary.MostUsedProviders = []string{"vllm", "llamacpp", "ollama"} // Placeholder

	return summary
}

func (a *UsageAnalytics) analyzeTasks() map[string]*TaskAnalysis {
	analysis := make(map[string]*TaskAnalysis)

	for taskType, pattern := range a.taskPatterns {
		analysis[taskType] = &TaskAnalysis{
			TaskType:          taskType,
			Frequency:         int64(len(pattern.CommonModels)), // Simplified
			AverageComplexity: pattern.AverageComplexity,
			PreferredModels:   pattern.CommonModels,
			AverageLatency:    pattern.PerformanceRequirements["latency"],
			SuccessRate:       0.9, // Placeholder
			PeakHours:         pattern.PeakHours,
			Trends:            []string{"stable"}, // Placeholder
		}
	}

	return analysis
}

func (a *UsageAnalytics) analyzePerformance() *PerformanceAnalysis {
	analysis := &PerformanceAnalysis{
		AverageTPS:       0.0,
		OptimalProviders: make(map[string]float64),
		BottleneckAnalysis: &BottleneckAnalysis{
			MemoryBottleneck:  false,
			GPUBottleneck:     false,
			CPUBottleneck:     false,
			NetworkBottleneck: false,
			AffectedModels:    []string{},
			Recommendations:   []string{},
		},
		OptimizationImpact: &OptimizationImpact{
			TotalOptimizations: 0,
			AverageImprovement: 0.0,
			SuccessfulRate:     0.0,
			MostEffectiveTypes: []string{},
			ImpactByModel:      make(map[string]float64),
		},
		Recommendations: []string{},
	}

	// Calculate average TPS
	totalTPS := 0.0
	count := 0

	for _, history := range a.performanceHistory {
		if history.AverageMetrics != nil {
			totalTPS += history.AverageMetrics.TokensPerSecond
			count++

			// Track optimizations
			analysis.OptimizationImpact.TotalOptimizations += int64(len(history.OptimizationHistory))

			for _, record := range history.OptimizationHistory {
				analysis.OptimizationImpact.AverageImprovement += record.Improvement
				if record.Success {
					analysis.OptimizationImpact.SuccessfulRate += 1.0
				}
			}
		}
	}

	if count > 0 {
		analysis.AverageTPS = totalTPS / float64(count)
	}

	if analysis.OptimizationImpact.TotalOptimizations > 0 {
		analysis.OptimizationImpact.AverageImprovement /= float64(analysis.OptimizationImpact.TotalOptimizations)
		analysis.OptimizationImpact.SuccessfulRate /= float64(analysis.OptimizationImpact.TotalOptimizations)
	}

	// Optimal providers (simplified)
	analysis.OptimalProviders = map[string]float64{
		"vllm":     25.0,
		"llamacpp": 15.0,
		"ollama":   12.0,
		"localai":  10.0,
	}

	return analysis
}

func (a *UsageAnalytics) analyzeUsers() *UserAnalysis {
	analysis := &UserAnalysis{
		TotalUsers:             int64(len(a.userPreferences)),
		AverageRequestsPerUser: 0.0,
		UserSegments:           make(map[string]int64),
		PreferredProviders:     make(map[string][]string),
		UserRetention: &UserRetention{
			DailyRetention:   0.85,
			WeeklyRetention:  0.70,
			MonthlyRetention: 0.50,
			ChurnRate:        0.05,
		},
		BehavioralTrends: []string{"increasing usage", "preference for speed"},
	}

	// Analyze user preferences
	for userID, prefs := range a.userPreferences {
		for _, provider := range prefs.PreferredProviders {
			if analysis.PreferredProviders[provider] == nil {
				analysis.PreferredProviders[provider] = []string{}
			}
			analysis.PreferredProviders[provider] = append(analysis.PreferredProviders[provider], userID)
		}

		// Segment users (simplified)
		if prefs.QualityPreference == "fast" {
			analysis.UserSegments["performance_focused"]++
		} else if prefs.QualityPreference == "quality" {
			analysis.UserSegments["quality_focused"]++
		} else {
			analysis.UserSegments["balanced"]++
		}
	}

	return analysis
}

func (a *UsageAnalytics) generateRecommendations(report *UsageReport) []string {
	recommendations := []string{}

	// Model recommendations
	// TODO: Implement trending models analysis
	// if len(report.TrendingModels) > 0 {
	// 	recommendations = append(recommendations, fmt.Sprintf("Consider allocating more resources to trending models: %s", strings.Join(report.TrendingModels, ", ")))
	// }

	// Performance recommendations
	if report.PerformanceAnalysis.AverageTPS < 10.0 {
		recommendations = append(recommendations, "Consider optimizing providers for better throughput")
	}

	// User recommendations
	if report.UserAnalysis.UserRetention.MonthlyRetention < 0.4 {
		recommendations = append(recommendations, "Focus on improving user experience to increase retention")
	}

	return recommendations
}

// File I/O methods

func (a *UsageAnalytics) loadAnalyticsData(dir string) error {
	// Load model usage stats
	if data, err := os.ReadFile(filepath.Join(dir, "model_usage_stats.json")); err == nil {
		json.Unmarshal(data, &a.modelUsageStats)
	}

	// Load task patterns
	if data, err := os.ReadFile(filepath.Join(dir, "task_patterns.json")); err == nil {
		json.Unmarshal(data, &a.taskPatterns)
	}

	// Load user preferences
	if data, err := os.ReadFile(filepath.Join(dir, "user_preferences.json")); err == nil {
		json.Unmarshal(data, &a.userPreferences)
	}

	// Load performance history
	if data, err := os.ReadFile(filepath.Join(dir, "performance_history.json")); err == nil {
		json.Unmarshal(data, &a.performanceHistory)
	}

	return nil
}

func (a *UsageAnalytics) saveAnalyticsData() error {
	dir := filepath.Join(filepath.Dir(a.modelUsageStats[""].LastUsed.String()), "analytics")
	os.MkdirAll(dir, 0755)

	// Save model usage stats
	if data, err := json.MarshalIndent(a.modelUsageStats, "", "  "); err == nil {
		os.WriteFile(filepath.Join(dir, "model_usage_stats.json"), data, 0644)
	}

	// Save task patterns
	if data, err := json.MarshalIndent(a.taskPatterns, "", "  "); err == nil {
		os.WriteFile(filepath.Join(dir, "task_patterns.json"), data, 0644)
	}

	// Save user preferences
	if data, err := json.MarshalIndent(a.userPreferences, "", "  "); err == nil {
		os.WriteFile(filepath.Join(dir, "user_preferences.json"), data, 0644)
	}

	// Save performance history
	if data, err := json.MarshalIndent(a.performanceHistory, "", "  "); err == nil {
		os.WriteFile(filepath.Join(dir, "performance_history.json"), data, 0644)
	}

	return nil
}

// Helper function

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
