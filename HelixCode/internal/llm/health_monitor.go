package llm

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// HealthMonitor provides automated health monitoring for all providers
type HealthMonitor struct {
	manager       *AutoLLMManager
	checkInterval time.Duration
	isRunning     bool
	mutex         sync.RWMutex
	stopChan      chan bool
	client        *http.Client
	alertSystem   *AlertSystem
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(manager *AutoLLMManager) *HealthMonitor {
	return &HealthMonitor{
		manager:       manager,
		checkInterval: 30 * time.Second,
		stopChan:      make(chan bool),
		client:        &http.Client{Timeout: 5 * time.Second},
		alertSystem:   NewAlertSystem(),
	}
}

// Start begins automated health monitoring
func (hm *HealthMonitor) Start(ctx context.Context) error {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	if hm.isRunning {
		return nil
	}

	log.Println("🏥 Starting automated health monitoring...")

	ticker := time.NewTicker(hm.checkInterval)
	defer ticker.Stop()

	hm.isRunning = true

	for {
		select {
		case <-ctx.Done():
			log.Println("🏥 Health monitor stopped")
			hm.isRunning = false
			return nil
		case <-hm.stopChan:
			log.Println("🏥 Health monitor stopped")
			hm.isRunning = false
			return nil
		case <-ticker.C:
			hm.performHealthChecks()
		}
	}
}

// performHealthChecks checks health of all providers
func (hm *HealthMonitor) performHealthChecks() {
	providers := hm.manager.GetStatus()

	for name, provider := range providers {
		if provider.Status != "running" {
			continue
		}

		isHealthy, responseTime, err := hm.checkProviderHealth(provider)

		hm.updateProviderHealth(provider, isHealthy, responseTime, err)

		if !isHealthy {
			hm.handleUnhealthyProvider(name, provider, err)
		}
	}
}

// checkProviderHealth performs health check on a single provider
func (hm *HealthMonitor) checkProviderHealth(provider *AutoProvider) (bool, int, error) {
	start := time.Now()

	resp, err := hm.client.Get(provider.HealthURL)
	responseTime := int(time.Since(start).Milliseconds())

	if err != nil {
		return false, responseTime, fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200, responseTime, nil
}

// updateProviderHealth updates provider health status
func (hm *HealthMonitor) updateProviderHealth(provider *AutoProvider, isHealthy bool, responseTime int, err error) {
	provider.Health.LastCheck = time.Now()
	provider.Health.ResponseTime = responseTime
	provider.Health.IsHealthy = isHealthy

	if isHealthy {
		provider.Health.Status = "healthy"
		provider.Health.Error = ""
		provider.RetryCount = 0
	} else {
		provider.Health.Status = "unhealthy"
		if err != nil {
			provider.Health.Error = err.Error()
		}
		provider.RetryCount++
	}
}

// handleUnhealthyProvider handles unhealthy provider scenarios
func (hm *HealthMonitor) handleUnhealthyProvider(name string, provider *AutoProvider, err error) {
	log.Printf("🚨 Provider %s is unhealthy: %v", name, err)

	// Send alert
	hm.alertSystem.SendAlert(&Alert{
		Type:      "health_failure",
		Provider:  name,
		Message:   fmt.Sprintf("Provider %s health check failed: %v", name, err),
		Severity:  "warning",
		Timestamp: time.Now(),
	})

	// Trigger auto-recovery if within retry limits
	if provider.RetryCount <= hm.manager.config.Health.MaxRetries {
		go hm.triggerAutoRecovery(name, provider)
	} else {
		// Max retries exceeded, send critical alert
		hm.alertSystem.SendAlert(&Alert{
			Type:      "max_retries_exceeded",
			Provider:  name,
			Message:   fmt.Sprintf("Provider %s exceeded max recovery attempts", name),
			Severity:  "critical",
			Timestamp: time.Now(),
		})
	}
}

// triggerAutoRecovery triggers automatic recovery for a provider
func (hm *HealthMonitor) triggerAutoRecovery(name string, provider *AutoProvider) {
	log.Printf("🔄 Triggering auto-recovery for %s (attempt %d)", name, provider.RetryCount)

	// Wait before recovery attempt
	time.Sleep(time.Duration(hm.manager.config.Health.RetryDelay) * time.Second)

	// Attempt recovery
	if err := hm.manager.autoRecoverProvider(provider); err != nil {
		log.Printf("❌ Auto-recovery failed for %s: %v", name, err)

		// Send recovery failure alert
		hm.alertSystem.SendAlert(&Alert{
			Type:      "recovery_failed",
			Provider:  name,
			Message:   fmt.Sprintf("Auto-recovery failed for %s: %v", name, err),
			Severity:  "error",
			Timestamp: time.Now(),
		})
	} else {
		log.Printf("✅ Auto-recovery successful for %s", name)

		// Send recovery success alert
		hm.alertSystem.SendAlert(&Alert{
			Type:      "recovery_successful",
			Provider:  name,
			Message:   fmt.Sprintf("Auto-recovery successful for %s", name),
			Severity:  "info",
			Timestamp: time.Now(),
		})
	}
}

// Stop stops the health monitor
func (hm *HealthMonitor) Stop() {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	if hm.isRunning {
		close(hm.stopChan)
	}
}

// SetInterval updates the health check interval
func (hm *HealthMonitor) SetInterval(interval time.Duration) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	hm.checkInterval = interval
	log.Printf("🏥 Health check interval updated to %v", interval)
}
