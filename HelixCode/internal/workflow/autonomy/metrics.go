package autonomy

import (
	"sync/atomic"
	"time"
)

// Metrics tracks autonomy system performance
type Metrics struct {
	PermissionChecks   atomic.Int64
	PermissionsGranted atomic.Int64
	PermissionsDenied  atomic.Int64
	ActionsExecuted    atomic.Int64
	ActionsFailed      atomic.Int64
	ActionsConfirmed   atomic.Int64
	ActionsRejected    atomic.Int64
	AutoRetries        atomic.Int64
	ModeChanges        atomic.Int64
	Escalations        atomic.Int64

	AverageCheckTime   atomic.Int64 // microseconds
	AverageExecuteTime atomic.Int64 // milliseconds
}

// NewMetrics creates a new metrics tracker
func NewMetrics() *Metrics {
	return &Metrics{}
}

// RecordPermissionCheck records a permission check
func (m *Metrics) RecordPermissionCheck(duration time.Duration, granted bool) {
	m.PermissionChecks.Add(1)
	if granted {
		m.PermissionsGranted.Add(1)
	} else {
		m.PermissionsDenied.Add(1)
	}

	// Update average check time using exponential moving average
	current := m.AverageCheckTime.Load()
	newAvg := (current*9 + duration.Microseconds()) / 10
	m.AverageCheckTime.Store(newAvg)
}

// RecordExecution records an action execution
func (m *Metrics) RecordExecution(result *ActionResult) {
	m.ActionsExecuted.Add(1)

	if result.Success {
		if result.Confirmed {
			m.ActionsConfirmed.Add(1)
		}
	} else {
		m.ActionsFailed.Add(1)
		if !result.Confirmed {
			m.ActionsRejected.Add(1)
		}
	}

	if result.Retries > 0 {
		m.AutoRetries.Add(int64(result.Retries))
	}

	if result.Escalated {
		m.Escalations.Add(1)
	}

	// Update average execution time
	current := m.AverageExecuteTime.Load()
	newAvg := (current*9 + result.Duration.Milliseconds()) / 10
	m.AverageExecuteTime.Store(newAvg)
}

// RecordModeChange records a mode change
func (m *Metrics) RecordModeChange() {
	m.ModeChanges.Add(1)
}

// RecordEscalation records an escalation
func (m *Metrics) RecordEscalation() {
	m.Escalations.Add(1)
}

// GetStats returns current statistics
func (m *Metrics) GetStats() MetricsStats {
	return MetricsStats{
		PermissionChecks:   m.PermissionChecks.Load(),
		PermissionsGranted: m.PermissionsGranted.Load(),
		PermissionsDenied:  m.PermissionsDenied.Load(),
		ActionsExecuted:    m.ActionsExecuted.Load(),
		ActionsFailed:      m.ActionsFailed.Load(),
		ActionsConfirmed:   m.ActionsConfirmed.Load(),
		ActionsRejected:    m.ActionsRejected.Load(),
		AutoRetries:        m.AutoRetries.Load(),
		ModeChanges:        m.ModeChanges.Load(),
		Escalations:        m.Escalations.Load(),
		AverageCheckTime:   time.Duration(m.AverageCheckTime.Load()) * time.Microsecond,
		AverageExecuteTime: time.Duration(m.AverageExecuteTime.Load()) * time.Millisecond,
	}
}

// MetricsStats represents a snapshot of metrics
type MetricsStats struct {
	PermissionChecks   int64
	PermissionsGranted int64
	PermissionsDenied  int64
	ActionsExecuted    int64
	ActionsFailed      int64
	ActionsConfirmed   int64
	ActionsRejected    int64
	AutoRetries        int64
	ModeChanges        int64
	Escalations        int64
	AverageCheckTime   time.Duration
	AverageExecuteTime time.Duration
}

// SuccessRate returns the action success rate (0-1)
func (s MetricsStats) SuccessRate() float64 {
	if s.ActionsExecuted == 0 {
		return 0
	}
	successful := s.ActionsExecuted - s.ActionsFailed
	return float64(successful) / float64(s.ActionsExecuted)
}

// ConfirmationRate returns the confirmation rate (0-1)
func (s MetricsStats) ConfirmationRate() float64 {
	confirmed := s.ActionsConfirmed + s.ActionsRejected
	if confirmed == 0 {
		return 0
	}
	return float64(s.ActionsConfirmed) / float64(confirmed)
}

// ApprovalRate returns the permission approval rate (0-1)
func (s MetricsStats) ApprovalRate() float64 {
	if s.PermissionChecks == 0 {
		return 0
	}
	return float64(s.PermissionsGranted) / float64(s.PermissionChecks)
}

// Reset resets all metrics to zero
func (m *Metrics) Reset() {
	m.PermissionChecks.Store(0)
	m.PermissionsGranted.Store(0)
	m.PermissionsDenied.Store(0)
	m.ActionsExecuted.Store(0)
	m.ActionsFailed.Store(0)
	m.ActionsConfirmed.Store(0)
	m.ActionsRejected.Store(0)
	m.AutoRetries.Store(0)
	m.ModeChanges.Store(0)
	m.Escalations.Store(0)
	m.AverageCheckTime.Store(0)
	m.AverageExecuteTime.Store(0)
}
