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
	if hm.isRunning {
		hm.mutex.Unlock()
		return nil
	}
	hm.isRunning = true
	// Recreate stop channel in case it was closed previously
	hm.stopChan = make(chan bool)
	hm.mutex.Unlock()

	log.Println("🏥 Starting automated health monitoring...")

	ticker := time.NewTicker(hm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			hm.mutex.Lock()
			hm.isRunning = false
			hm.mutex.Unlock()
			log.Println("🏥 Health monitor stopped")
			return nil
		case <-hm.stopChan:
			hm.mutex.Lock()
			hm.isRunning = false
			hm.mutex.Unlock()
			log.Println("🏥 Health monitor stopped")
			return nil
		case <-ticker.C:
			hm.performHealthChecks()
		}
	}
}

// performHealthChecks checks health of all providers.
//
// HXC-205: this iterates the manager's LIVE records, deliberately NOT
// GetStatus(). GetStatus now hands back a deep copy the caller owns, so a
// verdict written into it would be silently discarded and health monitoring
// would become a no-op that still looked green. Before the fix this code
// depended on GetStatus returning a SHALLOW copy — the verdict reached manager
// state only by accident, through the aliased Health pointer, while
// RetryCount (a value field on the copy) was written to the throwaway and lost
// on every cycle. Both halves now go through the manager's accessors, which
// take its lock.
func (hm *HealthMonitor) performHealthChecks() {
	for _, entry := range hm.manager.providerEntries() {
		name, provider := entry.name, entry.provider

		if hm.manager.statusOf(provider) != "running" {
			continue
		}

		// Probe with no lock held: an HTTP call with a 5s timeout.
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

// updateProviderHealth records a health verdict in manager state.
//
// HXC-205: the five Health fields and RetryCount were written here with no
// lock, on a record shared with AutoLLMManager.autoHealthCheck, which writes
// the same words from the background-task goroutine whenever AutoMonitor is
// enabled. They are now applied in a single critical section, so a reader
// observes one whole verdict or the other and never a mixture of the two —
// which is how a provider came to be reported healthy when the probe had found
// it stopped.
//
// The record is mutated in place, so this stays correct both for providers the
// manager owns and for detached records handed in directly.
func (hm *HealthMonitor) updateProviderHealth(provider *AutoProvider, isHealthy bool, responseTime int, err error) {
	hm.manager.applyHealthResult(provider, isHealthy, responseTime, err)
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

	// Trigger auto-recovery if within retry limits. The counter is read back
	// under the manager's lock (HXC-205) rather than off the record directly.
	if hm.manager.retryCountOf(provider) <= hm.manager.config.Health.MaxRetries {
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
	log.Printf("🔄 Triggering auto-recovery for %s (attempt %d)", name, hm.manager.retryCountOf(provider))

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
	running := hm.isRunning
	if running {
		hm.isRunning = false
	}
	hm.mutex.Unlock()

	if running {
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
