package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"dev.helix.code/internal/llm"
)

// Advanced discovery and analytics command implementations

func runDiscover(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	// Initialize model discovery engine
	discoveryEngine := llm.NewModelDiscoveryEngine(getLocalLLMBaseDir())
	
	fmt.Println("🔍 Discovering models...")
	fmt.Printf("Source: %s\n", discoverSource)
	if discoverFilter != "" {
		fmt.Printf("Filter: %s\n", discoverFilter)
	}
	fmt.Println()
	
	// Get all available models (simplified for discover)
	// In a real implementation, this would fetch from external sources
	models := []*llm.ModelInfo{
		{
			ID:           "llama-3-8b-instruct",
			Name:         "Llama 3 8B Instruct",
			Format:       llm.FormatGGUF,
			Size:         4700000000,
			ContextSize:  8192,
			Provider:     llm.ProviderLocalLLM,
			Capabilities: []llm.ModelCapability{llm.CapabilityCodeGeneration, llm.CapabilityReasoning},
		},
		{
			ID:           "mistral-7b-instruct",
			Name:         "Mistral 7B Instruct",
			Format:       llm.FormatGGUF,
			Size:         4100000000,
			ContextSize:  32768,
			Provider:     llm.ProviderLocalLLM,
			Capabilities: []llm.ModelCapability{llm.CapabilityCodeGeneration, llm.CapabilityAnalysis},
		},
		{
			ID:           "codellama-7b-instruct",
			Name:         "CodeLlama 7B Instruct",
			Format:       llm.FormatGGUF,
			Size:         3800000000,
			ContextSize:  16384,
			Provider:     llm.ProviderLocalLLM,
			Capabilities: []llm.ModelCapability{llm.CapabilityCodeGeneration, llm.CapabilityDebugging},
		},
	}
	
	// Apply filter if specified
	if discoverFilter != "" {
		filter := strings.ToLower(discoverFilter)
		filtered := []*llm.ModelInfo{}
		for _, model := range models {
			if strings.Contains(strings.ToLower(model.ID), filter) ||
			   strings.Contains(strings.ToLower(model.Name), filter) {
				filtered = append(filtered, model)
			}
		}
		models = filtered
	}
	
	if len(models) == 0 {
		fmt.Printf("❌ No models found matching filter: %s\n", discoverFilter)
		return nil
	}
	
	// Display models in table format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSIZE\tFORMAT\tCONTEXT\tCAPABILITIES")
	fmt.Fprintln(w, "--\t----\t----\t------\t-------\t------------")
	
	for _, model := range models {
		capabilities := make([]string, len(model.Capabilities))
		for i, cap := range model.Capabilities {
			capabilities[i] = strings.TrimPrefix(strings.ToLower(string(cap)), "capability_")
		}
		
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\n",
			model.ID,
			model.Name,
			formatBytes(model.Size),
			model.Format,
			model.ContextSize,
			strings.Join(capabilities, ","))
	}
	
	w.Flush()
	
	fmt.Printf("\n📊 Found %d models\n", len(models))
	fmt.Println("💡 Use 'helix local-llm recommend' to get personalized recommendations")
	
	return nil
}

func runRecommend(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	// Initialize model discovery engine
	discoveryEngine := llm.NewModelDiscoveryEngine(getLocalLLMBaseDir())
	
	// Build recommendation request
	req := &llm.RecommendationRequest{
		TaskTypes:         recommendTaskTypes,
		QualityPreference:  recommendQualityPreference,
		PrivacyLevel:      recommendPrivacyLevel,
		MaxRecommendations: 10,
	}
	
	if recommendMaxMemory > 0 {
		req.Constraints = map[string]interface{}{
			"max_memory_mb": float64(recommendMaxMemory),
		}
	}
	
	if recommendBudgetLimit > 0 {
		req.BudgetLimit = recommendBudgetLimit
	}
	
	if len(recommendProviders) > 0 {
		req.IncludeProviders = recommendProviders
	}
	
	fmt.Println("🧠 Getting personalized model recommendations...")
	if len(req.TaskTypes) > 0 {
		fmt.Printf("Tasks: %s\n", strings.Join(req.TaskTypes, ", "))
	}
	fmt.Printf("Quality: %s\n", req.QualityPreference)
	fmt.Printf("Privacy: %s\n", req.PrivacyLevel)
	if recommendMaxMemory > 0 {
		fmt.Printf("Max Memory: %d MB\n", recommendMaxMemory)
	}
	if recommendBudgetLimit > 0 {
		fmt.Printf("Budget: $%.4f per 1M tokens\n", recommendBudgetLimit)
	}
	fmt.Println()
	
	// Get recommendations
	resp, err := discoveryEngine.GetRecommendations(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get recommendations: %w", err)
	}
	
	if len(resp.Recommendations) == 0 {
		fmt.Println("❌ No suitable models found for your requirements")
		fmt.Println("💡 Try adjusting your constraints or preferences")
		return nil
	}
	
	// Display recommendations
	fmt.Printf("🎯 Found %d recommendations (search time: %v)\n\n", 
		len(resp.Recommendations), resp.SearchTime)
	
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "RANK\tMODEL\tSCORE\tPERFORMANCE\tHARDWARE\tREASONS")
	fmt.Fprintln(w, "----\t-----\t-----\t-----------\t--------\t-------")
	
	for i, rec := range resp.Recommendations {
		var reasons []string
		if len(rec.Reasons) > 2 {
			reasons = rec.Reasons[:2]
		} else {
			reasons = rec.Reasons
		}
		
		var performance, hardware string
		if rec.EstimatedPerformance != nil {
			performance = fmt.Sprintf("%.1f TPS", rec.EstimatedPerformance.TokensPerSecond)
		}
		if rec.HardwareFit != nil {
			hardware = fmt.Sprintf("%.1f%%", rec.HardwareFit.OverallFit*100)
		}
		
		fmt.Fprintf(w, "%d\t%s\t%.2f\t%s\t%s\t%s\n",
			i+1,
			rec.Model.Name,
			rec.RecommendationScore,
			performance,
			hardware,
			strings.Join(reasons, "; "))
	}
	
	w.Flush()
	
	// Show insights if available
	if resp.Insights != nil {
		fmt.Println("\n💡 Insights:")
		for _, insight := range resp.Insights.MarketTrends {
			fmt.Printf("  • %s\n", insight)
		}
		for _, reasoning := range resp.Insights.RecommendationReasoning {
			fmt.Printf("  • %s\n", reasoning)
		}
	}
	
	fmt.Printf("\n📊 Relevance Score: %.2f\n", resp.RelevanceScore)
	fmt.Println("💡 Use 'helix local-llm download-all <model-id>' to download and share model")
	
	return nil
}

func runAnalytics(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	// Initialize usage analytics
	analytics := llm.NewUsageAnalytics(getLocalLLMBaseDir())
	
	// Parse time range
	timeRange := parseTimeRange(analyticsTimeRange)
	
	fmt.Printf("📊 Generating analytics report for %v\n\n", timeRange)
	
	// Generate usage report
	report, err := analytics.GenerateUsageReport(ctx, timeRange)
	if err != nil {
		return fmt.Errorf("failed to generate analytics: %w", err)
	}
	
	// Display summary
	fmt.Println("📈 Executive Summary:")
	fmt.Printf("  Total Models: %d\n", report.Summary.TotalModels)
	fmt.Printf("  Total Requests: %d\n", report.Summary.TotalRequests)
	fmt.Printf("  Average Latency: %.1f ms\n", report.Summary.AverageLatency)
	fmt.Printf("  Success Rate: %.1f%%\n", report.Summary.OverallSuccessRate*100)
	fmt.Printf("  User Satisfaction: %.1f/5.0\n", report.Summary.AverageSatisfaction)
	fmt.Println()
	
	// Display top models
	fmt.Println("🏆 Top Models:")
	for i, model := range report.TopModels[:min(5, len(report.TopModels))] {
		fmt.Printf("  %d. %s (%d requests, %.1f%% satisfaction)\n",
			i+1, model.ModelID, model.TotalRequests, model.UserSatisfaction)
	}
	fmt.Println()
	
	// Display performance analysis
	fmt.Println("⚡ Performance Analysis:")
	fmt.Printf("  Average TPS: %.1f\n", report.PerformanceAnalysis.AverageTPS)
	for provider, tps := range report.PerformanceAnalysis.OptimalProviders {
		fmt.Printf("  %s: %.1f TPS\n", provider, tps)
	}
	fmt.Println()
	
	// Display recommendations
	if len(report.Recommendations) > 0 {
		fmt.Println("💡 Recommendations:")
		for _, rec := range report.Recommendations {
			fmt.Printf("  • %s\n", rec)
		}
	}
	
	return nil
}

func runReport(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	// Initialize usage analytics
	analytics := llm.NewUsageAnalytics(getLocalLLMBaseDir())
	
	// Parse time range
	timeRange := parseTimeRange(analyticsTimeRange)
	
	fmt.Printf("📋 Generating comprehensive report (%s format) for %v\n\n", 
		reportFormat, timeRange)
	
	// Generate usage report
	report, err := analytics.GenerateUsageReport(ctx, timeRange)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}
	
	// Output based on format
	switch reportFormat {
	case "json":
		if data, err := json.MarshalIndent(report, "", "  "); err == nil {
			fmt.Println(string(data))
		}
	case "csv":
		outputCSVReport(report)
	default: // table
		outputTableReport(report)
	}
	
	return nil
}

func runInsights(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	// Initialize model discovery engine and analytics
	discoveryEngine := llm.NewModelDiscoveryEngine(getLocalLLMBaseDir())
	analytics := llm.NewUsageAnalytics(getLocalLLMBaseDir())
	
	fmt.Println("🧠 Generating AI-powered insights...")
	fmt.Printf("Insights Type: %s\n\n", insightsType)
	
	// Generate usage report for insights
	timeRange := parseTimeRange("7d")
	report, err := analytics.GenerateUsageReport(ctx, timeRange)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}
	
	// Generate insights based on type
	switch insightsType {
	case "performance":
		generatePerformanceInsights(report)
	case "usage":
		generateUsageInsights(report)
	case "models":
		generateModelInsights(report)
	default: // all
		generatePerformanceInsights(report)
		generateUsageInsights(report)
		generateModelInsights(report)
	}
	
	return nil
}

// Helper functions for advanced commands

func parseTimeRange(timeRange string) llm.TimeRange {
	var start, end time.Time
	now := time.Now()
	
	switch timeRange {
	case "1d":
		start = now.Add(-24 * time.Hour)
		end = now
	case "7d":
		start = now.Add(-7 * 24 * time.Hour)
		end = now
	case "30d":
		start = now.Add(-30 * 24 * time.Hour)
		end = now
	case "all":
		start = time.Time{} // Zero time
		end = now
	default:
		start = now.Add(-7 * 24 * time.Hour)
		end = now
	}
	
	return llm.TimeRange{
		Start: start,
		End:   end,
	}
}

func outputTableReport(report *llm.UsageReport) {
	// Executive summary table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "METRIC\tVALUE")
	fmt.Fprintln(w, "------\t-----")
	fmt.Fprintf(w, "Total Models\t%d\n", report.Summary.TotalModels)
	fmt.Fprintf(w, "Total Requests\t%d\n", report.Summary.TotalRequests)
	fmt.Fprintf(w, "Average Latency\t%.1f ms\n", report.Summary.AverageLatency)
	fmt.Fprintf(w, "Success Rate\t%.1f%%\n", report.Summary.OverallSuccessRate*100)
	fmt.Fprintf(w, "User Satisfaction\t%.1f/5.0\n", report.Summary.AverageSatisfaction)
	w.Flush()
}

func outputCSVReport(report *llm.UsageReport) {
	// Simple CSV output for executive summary
	fmt.Println("metric,value")
	fmt.Printf("total_models,%d\n", report.Summary.TotalModels)
	fmt.Printf("total_requests,%d\n", report.Summary.TotalRequests)
	fmt.Printf("average_latency,%.1f\n", report.Summary.AverageLatency)
	fmt.Printf("success_rate,%.3f\n", report.Summary.OverallSuccessRate)
	fmt.Printf("user_satisfaction,%.1f\n", report.Summary.AverageSatisfaction)
}

func generatePerformanceInsights(report *llm.UsageReport) {
	fmt.Println("⚡ Performance Insights:")
	
	// Bottleneck analysis
	if report.PerformanceAnalysis.BottleneckAnalysis != nil {
		bottlenecks := []string{}
		if report.PerformanceAnalysis.BottleneckAnalysis.MemoryBottleneck {
			bottlenecks = append(bottlenecks, "Memory")
		}
		if report.PerformanceAnalysis.BottleneckAnalysis.GPUBottleneck {
			bottlenecks = append(bottlenecks, "GPU")
		}
		if report.PerformanceAnalysis.BottleneckAnalysis.CPUBottleneck {
			bottlenecks = append(bottlenecks, "CPU")
		}
		
		if len(bottlenecks) > 0 {
			fmt.Printf("  🎯 Identified Bottlenecks: %s\n", strings.Join(bottlenecks, ", "))
			for _, rec := range report.PerformanceAnalysis.BottleneckAnalysis.Recommendations {
				fmt.Printf("  💡 Recommendation: %s\n", rec)
			}
		}
	}
	
	// Optimization impact
	if report.PerformanceAnalysis.OptimizationImpact != nil {
		fmt.Printf("  📈 Optimization Success Rate: %.1f%%\n", 
			report.PerformanceAnalysis.OptimizationImpact.SuccessfulRate*100)
		fmt.Printf("  🚀 Average Performance Improvement: %.1f%%\n", 
			report.PerformanceAnalysis.OptimizationImpact.AverageImprovement)
	}
	
	fmt.Println()
}

func generateUsageInsights(report *llm.UsageReport) {
	fmt.Println("📊 Usage Insights:")
	
	// User segments
	if report.UserAnalysis.UserSegments != nil {
		fmt.Println("  👥 User Segments:")
		for segment, count := range report.UserAnalysis.UserSegments {
			fmt.Printf("    • %s: %d users\n", segment, count)
		}
	}
	
	// Behavioral trends
	if len(report.UserAnalysis.BehavioralTrends) > 0 {
		fmt.Println("  📈 Behavioral Trends:")
		for _, trend := range report.UserAnalysis.BehavioralTrends {
			fmt.Printf("    • %s\n", trend)
		}
	}
	
	// Retention insights
	if report.UserAnalysis.UserRetention != nil {
		fmt.Printf("  🔄 User Retention: Daily %.1f%%, Weekly %.1f%%, Monthly %.1f%%\n",
			report.UserAnalysis.UserRetention.DailyRetention*100,
			report.UserAnalysis.UserRetention.WeeklyRetention*100,
			report.UserAnalysis.UserRetention.MonthlyRetention*100)
		fmt.Printf("  📉 Churn Rate: %.1f%%\n", report.UserAnalysis.UserRetention.ChurnRate*100)
	}
	
	fmt.Println()
}

func generateModelInsights(report *llm.UsageReport) {
	fmt.Println("🏆 Model Insights:")
	
	// Trending models
	if len(report.Summary.TrendingModels) > 0 {
		fmt.Println("  📈 Trending Models:")
		for _, model := range report.Summary.TrendingModels {
			fmt.Printf("    • %s\n", model)
		}
	}
	
	// Top performing models
	if len(report.TopModels) > 0 {
		fmt.Println("  🏆 Top Performing Models:")
		for i, model := range report.TopModels[:min(3, len(report.TopModels))] {
			fmt.Printf("    %d. %s (%.1f%% satisfaction)\n", i+1, model.ModelID, model.UserSatisfaction)
		}
	}
	
	// Task analysis
	if report.TaskAnalysis != nil {
		fmt.Println("  🎯 Task Performance:")
		for task, analysis := range report.TaskAnalysis {
			fmt.Printf("    • %s: %.1f%% success, %.1f ms average latency\n", 
				task, analysis.SuccessRate*100, analysis.AverageLatency)
		}
	}
	
	fmt.Println()
}