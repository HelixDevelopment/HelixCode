package llm

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// LoadBalancer provides intelligent load balancing across providers
type LoadBalancer struct {
	manager      *AutoLLMManager
	strategies    map[string]LoadBalancingStrategy
	currentStrategy string
	isRunning     bool
	mutex         sync.RWMutex
	stats         *LoadBalancingStats
	requestCount  int64
}

// LoadBalancingStrategy defines load balancing algorithm
type LoadBalancingStrategy interface {
	SelectProvider(providers []*AutoProvider) *AutoProvider
	GetName() string
}

// LoadBalancingStats tracks load balancing statistics
type LoadBalancingStats struct {
	TotalRequests      int64                    `json:"total_requests"`
	ProviderCounts     map[string]int64           `json:"provider_counts"`
	ResponseTimes      map[string]float64         `json:"response_times"`
	ErrorRates         map[string]float64         `json:"error_rates"`
	LastUpdated       time.Time                  `json:"last_updated"`
	Strategy          string                    `json:"strategy"`
	OptimalProvider   string                    `json:"optimal_provider"`
}

// Alert represents system alert
type Alert struct {
	Type      string    `json:"type"`
	Provider  string    `json:"provider"`
	Message   string    `json:"message"`
	Severity  string    `json:"severity"`
	Timestamp time.Time `json:"timestamp"`
}

// AlertSystem manages system alerts
type AlertSystem struct {
	alerts []Alert
	mutex   sync.RWMutex
}

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(manager *AutoLLMManager) *LoadBalancer {
	strategies := make(map[string]LoadBalancingStrategy)
	
	// Initialize load balancing strategies
	strategies["round_robin"] = &RoundRobinStrategy{}
	strategies["least_connections"] = &LeastConnectionsStrategy{}
	strategies["response_time"] = &ResponseTimeStrategy{}
	strategies["weighted"] = &WeightedStrategy{}
	strategies["performance_based"] = &PerformanceBasedStrategy{}
	
	return &LoadBalancer{
		manager:        manager,
		strategies:     strategies,
		currentStrategy: "performance_based",
		stats:          &LoadBalancingStats{
			ProviderCounts: make(map[string]int64),
			ResponseTimes:  make(map[string]float64),
			ErrorRates:     make(map[string]float64),
			Strategy:       "performance_based",
		},
	}
}

// Start begins load balancing operations
func (lb *LoadBalancer) Start(ctx context.Context) error {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	if lb.isRunning {
		return nil
	}

	log.Println("⚖️ Starting intelligent load balancing...")
	
	// Start statistics collection
	go lb.collectStats(ctx)
	
	lb.isRunning = true
	log.Println("✅ Load balancer started")
	
	return nil
}

// SelectOptimalProvider selects the best provider for a request
func (lb *LoadBalancer) SelectOptimalProvider(ctx context.Context) *AutoProvider {
	providers := lb.getHealthyProviders()
	if len(providers) == 0 {
		return nil
	}

	// Get current strategy
	strategy := lb.strategies[lb.currentStrategy]
	if strategy == nil {
		strategy = lb.strategies["performance_based"]
	}

	// Select provider using strategy
	selected := strategy.SelectProvider(providers)
	
	// Update statistics
	lb.updateStats(selected)
	
	log.Printf("⚖️ Selected provider: %s (strategy: %s)", selected.Name, lb.currentStrategy)
	
	return selected
}

// getHealthyProviders returns list of healthy providers
func (lb *LoadBalancer) getHealthyProviders() []*AutoProvider {
	status := lb.manager.GetStatus()
	var healthy []*AutoProvider
	
	for _, provider := range status {
		if provider.Status == "running" && provider.Health.IsHealthy {
			healthy = append(healthy, provider)
		}
	}
	
	return healthy
}

// updateStats updates load balancing statistics
func (lb *LoadBalancer) updateStats(provider *AutoProvider) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	
	lb.stats.TotalRequests++
	lb.stats.ProviderCounts[provider.Name]++
	
	// Update last updated
	lb.stats.LastUpdated = time.Now()
}

// collectStats collects performance statistics
func (lb *LoadBalancer) collectStats(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			lb.collectPerformanceStats()
		}
	}
}

// collectPerformanceStats collects performance statistics from providers
func (lb *LoadBalancer) collectPerformanceStats() {
	status := lb.manager.GetStatus()
	
	for name, provider := range status {
		lb.mutex.Lock()
		
		// Update response times
		if provider.Health.ResponseTime > 0 {
			if _, exists := lb.stats.ResponseTimes[name]; !exists {
				lb.stats.ResponseTimes[name] = 0
			}
			// Weighted average (10% new, 90% old)
			lb.stats.ResponseTimes[name] = 0.1*float64(provider.Health.ResponseTime) + 0.9*lb.stats.ResponseTimes[name]
		}
		
		// Update error rates
		if provider.Metrics != nil {
			if _, exists := lb.stats.ErrorRates[name]; !exists {
				lb.stats.ErrorRates[name] = 0
			}
			// Weighted average
			lb.stats.ErrorRates[name] = 0.1*provider.Metrics.ErrorRate + 0.9*lb.stats.ErrorRates[name]
		}
		
		lb.mutex.Unlock()
	}
	
	// Determine optimal provider
	lb.determineOptimalProvider()
}

// determineOptimalProvider determines the best performing provider
func (lb *LoadBalancer) determineOptimalProvider() {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	
	status := lb.manager.GetStatus()
	
	var bestProvider string
	var bestScore float64 = -1
	
	for name, provider := range status {
		if provider.Status != "running" || !provider.Health.IsHealthy {
			continue
		}
		
		// Calculate performance score
		score := lb.calculatePerformanceScore(name, provider)
		
		if score > bestScore {
			bestScore = score
			bestProvider = name
		}
	}
	
	lb.stats.OptimalProvider = bestProvider
}

// calculatePerformanceScore calculates performance score for a provider
func (lb *LoadBalancer) calculatePerformanceScore(name string, provider *AutoProvider) float64 {
	// Factors:
	// - Response time (lower is better)
	// - Error rate (lower is better)
	// - Throughput (higher is better)
	
	responseTimeScore := 100.0
	if rt, exists := lb.stats.ResponseTimes[name]; exists && rt > 0 {
		responseTimeScore = 10000.0 / rt // Inverse of response time
	}
	
	errorRateScore := 100.0
	if er, exists := lb.stats.ErrorRates[name]; exists {
		errorRateScore = (100.0 - er) * 10 // Penalty for errors
	}
	
	throughputScore := 100.0
	if provider.Metrics != nil && provider.Metrics.TokensPerSecond > 0 {
		throughputScore = provider.Metrics.TokensPerSecond / 5 // Normalize to 0-100
	}
	
	// Weighted score
	return responseTimeScore*0.4 + errorRateScore*0.3 + throughputScore*0.3
}

// GetStats returns load balancing statistics
func (lb *LoadBalancer) GetStats() *LoadBalancingStats {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()
	
	// Return copy of stats
	stats := *lb.stats
	stats.ProviderCounts = make(map[string]int64)
	stats.ResponseTimes = make(map[string]float64)
	stats.ErrorRates = make(map[string]float64)
	
	for k, v := range lb.stats.ProviderCounts {
		stats.ProviderCounts[k] = v
	}
	for k, v := range lb.stats.ResponseTimes {
		stats.ResponseTimes[k] = v
	}
	for k, v := range lb.stats.ErrorRates {
		stats.ErrorRates[k] = v
	}
	
	return &stats
}

// SetStrategy changes the load balancing strategy
func (lb *LoadBalancer) SetStrategy(strategyName string) error {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	
	if _, exists := lb.strategies[strategyName]; !exists {
		return fmt.Errorf("unknown load balancing strategy: %s", strategyName)
	}
	
	oldStrategy := lb.currentStrategy
	lb.currentStrategy = strategyName
	lb.stats.Strategy = strategyName
	
	log.Printf("⚖️ Load balancing strategy changed: %s -> %s", oldStrategy, strategyName)
	
	return nil
}

// Stop stops the load balancer
func (lb *LoadBalancer) Stop() {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	
	lb.isRunning = false
	log.Println("⏹️ Load balancer stopped")
}

// Load Balancing Strategies Implementation

// RoundRobinStrategy selects providers in round-robin order
type RoundRobinStrategy struct {
	current int
	mutex    sync.Mutex
}

func (rr *RoundRobinStrategy) SelectProvider(providers []*AutoProvider) *AutoProvider {
	rr.mutex.Lock()
	defer rr.mutex.Unlock()
	
	if len(providers) == 0 {
		return nil
	}
	
	provider := providers[rr.current]
	rr.current = (rr.current + 1) % len(providers)
	
	return provider
}

func (rr *RoundRobinStrategy) GetName() string {
	return "round_robin"
}

// LeastConnectionsStrategy selects provider with fewest active connections
type LeastConnectionsStrategy struct{}

func (lc *LeastConnectionsStrategy) SelectProvider(providers []*AutoProvider) *AutoProvider {
	if len(providers) == 0 {
		return nil
	}
	
	var bestProvider *AutoProvider
	minConnections := int(^uint(0) >> 1) // Max int
	
	for _, provider := range providers {
		connections := provider.Metrics.ActiveRequests
		if connections < minConnections {
			minConnections = connections
			bestProvider = provider
		}
	}
	
	return bestProvider
}

func (lc *LeastConnectionsStrategy) GetName() string {
	return "least_connections"
}

// ResponseTimeStrategy selects provider with lowest response time
type ResponseTimeStrategy struct{}

func (rt *ResponseTimeStrategy) SelectProvider(providers []*AutoProvider) *AutoProvider {
	if len(providers) == 0 {
		return nil
	}
	
	var bestProvider *AutoProvider
	minResponseTime := int(^uint(0) >> 1) // Max int
	
	for _, provider := range providers {
		responseTime := provider.Health.ResponseTime
		if responseTime > 0 && responseTime < minResponseTime {
			minResponseTime = responseTime
			bestProvider = provider
		}
	}
	
	return bestProvider
}

func (rt *ResponseTimeStrategy) GetName() string {
	return "response_time"
}

// WeightedStrategy selects providers based on assigned weights
type WeightedStrategy struct{}

func (ws *WeightedStrategy) SelectProvider(providers []*AutoProvider) *AutoProvider {
	if len(providers) == 0 {
		return nil
	}
	
	// For simplicity, use random weighted selection
	// In production, this would use actual performance-based weights
	rand.Seed(time.Now().UnixNano())
	index := rand.Intn(len(providers))
	
	return providers[index]
}

func (ws *WeightedStrategy) GetName() string {
	return "weighted"
}

// PerformanceBasedStrategy selects provider with best overall performance
type PerformanceBasedStrategy struct{}

func (pb *PerformanceBasedStrategy) SelectProvider(providers []*AutoProvider) *AutoProvider {
	if len(providers) == 0 {
		return nil
	}
	
	// Score providers based on performance metrics
	type ProviderScore struct {
		*AutoProvider
		score float64
	}
	
	var scoredProviders []ProviderScore
	
	for _, provider := range providers {
		score := pb.calculateScore(provider)
		scoredProviders = append(scoredProviders, ProviderScore{provider, score})
	}
	
	// Sort by score (descending)
	sort.Slice(scoredProviders, func(i, j int) bool {
		return scoredProviders[i].score > scoredProviders[j].score
	})
	
	if len(scoredProviders) > 0 {
		return scoredProviders[0].AutoProvider
	}
	
	return nil
}

func (pb *PerformanceBasedStrategy) calculateScore(provider *AutoProvider) float64 {
	// Factors with weights:
	// - Response time: 40%
	// - Throughput: 30%
	// - Error rate: 20%
	// - Availability: 10%
	
	responseTimeWeight := 0.4
	throughputWeight := 0.3
	errorRateWeight := 0.2
	availabilityWeight := 0.1
	
	responseTimeScore := 100.0
	if provider.Health.ResponseTime > 0 {
		responseTimeScore = 1000.0 / float64(provider.Health.ResponseTime)
	}
	
	throughputScore := provider.Metrics.TokensPerSecond / 10 // Normalize
	
	errorRateScore := (100.0 - provider.Metrics.ErrorRate) * 2
	
	availabilityScore := 100.0
	if provider.Health.IsHealthy {
		availabilityScore = 100.0
	} else {
		availabilityScore = 0.0
	}
	
	return responseTimeWeight*responseTimeScore + 
		   throughputWeight*throughputScore + 
		   errorRateWeight*errorRateScore + 
		   availabilityWeight*availabilityScore
}

func (pb *PerformanceBasedStrategy) GetName() string {
	return "performance_based"
}

// AlertSystem Implementation

// NewAlertSystem creates a new alert system
func NewAlertSystem() *AlertSystem {
	return &AlertSystem{
		alerts: make([]Alert, 0),
	}
}

// SendAlert sends an alert
func (as *AlertSystem) SendAlert(alert *Alert) {
	as.mutex.Lock()
	defer as.mutex.Unlock()
	
	as.alerts = append(as.alerts, *alert)
	
	// Log alert
	log.Printf("🚨 ALERT [%s]: %s", alert.Severity, alert.Message)
	
	// Keep only last 1000 alerts
	if len(as.alerts) > 1000 {
		as.alerts = as.alerts[1:]
	}
}

// GetAlerts returns recent alerts
func (as *AlertSystem) GetAlerts() []Alert {
	as.mutex.RLock()
	defer as.mutex.RUnlock()
	
	alerts := make([]Alert, len(as.alerts))
	copy(alerts, as.alerts)
	
	return alerts
}