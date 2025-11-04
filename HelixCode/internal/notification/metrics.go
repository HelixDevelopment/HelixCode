package notification

import (
	"sync"
	"time"
)

// Metrics tracks notification system metrics
type Metrics struct {
	// Notification counts
	TotalSent       int64
	TotalFailed     int64
	TotalRetries    int64
	TotalQueued     int64
	TotalRateLimited int64

	// Channel-specific metrics
	ChannelMetrics map[string]*ChannelMetrics

	// Response time metrics
	AverageResponseTime time.Duration
	MinResponseTime     time.Duration
	MaxResponseTime     time.Duration
	TotalResponseTime   time.Duration
	ResponseTimeCount   int64

	// Event metrics
	EventsProcessed int64
	EventsIgnored   int64

	mutex sync.RWMutex
}

// ChannelMetrics tracks metrics for a specific channel
type ChannelMetrics struct {
	Sent         int64
	Failed       int64
	Retries      int64
	RateLimited  int64
	AvgTime      time.Duration
	TotalTime    time.Duration
	TimeCount    int64
	mutex        sync.Mutex
}

// NewMetrics creates a new metrics tracker
func NewMetrics() *Metrics {
	return &Metrics{
		ChannelMetrics: make(map[string]*ChannelMetrics),
		MinResponseTime: time.Hour, // Start with high value
	}
}

// RecordSent records a successful send
func (m *Metrics) RecordSent(channel string, duration time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.TotalSent++
	m.updateResponseTime(duration)

	if _, ok := m.ChannelMetrics[channel]; !ok {
		m.ChannelMetrics[channel] = &ChannelMetrics{}
	}

	cm := m.ChannelMetrics[channel]
	cm.mutex.Lock()
	cm.Sent++
	cm.TotalTime += duration
	cm.TimeCount++
	if cm.TimeCount > 0 {
		cm.AvgTime = time.Duration(int64(cm.TotalTime) / cm.TimeCount)
	}
	cm.mutex.Unlock()
}

// RecordFailed records a failed send
func (m *Metrics) RecordFailed(channel string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.TotalFailed++

	if _, ok := m.ChannelMetrics[channel]; !ok {
		m.ChannelMetrics[channel] = &ChannelMetrics{}
	}

	cm := m.ChannelMetrics[channel]
	cm.mutex.Lock()
	cm.Failed++
	cm.mutex.Unlock()
}

// RecordRetry records a retry attempt
func (m *Metrics) RecordRetry(channel string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.TotalRetries++

	if _, ok := m.ChannelMetrics[channel]; !ok {
		m.ChannelMetrics[channel] = &ChannelMetrics{}
	}

	cm := m.ChannelMetrics[channel]
	cm.mutex.Lock()
	cm.Retries++
	cm.mutex.Unlock()
}

// RecordQueued records a queued notification
func (m *Metrics) RecordQueued() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.TotalQueued++
}

// RecordRateLimited records a rate-limited notification
func (m *Metrics) RecordRateLimited(channel string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.TotalRateLimited++

	if _, ok := m.ChannelMetrics[channel]; !ok {
		m.ChannelMetrics[channel] = &ChannelMetrics{}
	}

	cm := m.ChannelMetrics[channel]
	cm.mutex.Lock()
	cm.RateLimited++
	cm.mutex.Unlock()
}

// RecordEventProcessed records a processed event
func (m *Metrics) RecordEventProcessed() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.EventsProcessed++
}

// RecordEventIgnored records an ignored event
func (m *Metrics) RecordEventIgnored() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.EventsIgnored++
}

// updateResponseTime updates response time metrics (must be called with lock held)
func (m *Metrics) updateResponseTime(duration time.Duration) {
	m.TotalResponseTime += duration
	m.ResponseTimeCount++

	if m.ResponseTimeCount > 0 {
		m.AverageResponseTime = time.Duration(int64(m.TotalResponseTime) / m.ResponseTimeCount)
	}

	if duration < m.MinResponseTime {
		m.MinResponseTime = duration
	}

	if duration > m.MaxResponseTime {
		m.MaxResponseTime = duration
	}
}

// GetMetrics returns a copy of current metrics
func (m *Metrics) GetMetrics() Metrics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Create a copy
	copy := Metrics{
		TotalSent:           m.TotalSent,
		TotalFailed:         m.TotalFailed,
		TotalRetries:        m.TotalRetries,
		TotalQueued:         m.TotalQueued,
		TotalRateLimited:    m.TotalRateLimited,
		AverageResponseTime: m.AverageResponseTime,
		MinResponseTime:     m.MinResponseTime,
		MaxResponseTime:     m.MaxResponseTime,
		TotalResponseTime:   m.TotalResponseTime,
		ResponseTimeCount:   m.ResponseTimeCount,
		EventsProcessed:     m.EventsProcessed,
		EventsIgnored:       m.EventsIgnored,
		ChannelMetrics:      make(map[string]*ChannelMetrics),
	}

	for channel, cm := range m.ChannelMetrics {
		cm.mutex.Lock()
		copy.ChannelMetrics[channel] = &ChannelMetrics{
			Sent:        cm.Sent,
			Failed:      cm.Failed,
			Retries:     cm.Retries,
			RateLimited: cm.RateLimited,
			AvgTime:     cm.AvgTime,
			TotalTime:   cm.TotalTime,
			TimeCount:   cm.TimeCount,
		}
		cm.mutex.Unlock()
	}

	return copy
}

// Reset resets all metrics
func (m *Metrics) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.TotalSent = 0
	m.TotalFailed = 0
	m.TotalRetries = 0
	m.TotalQueued = 0
	m.TotalRateLimited = 0
	m.AverageResponseTime = 0
	m.MinResponseTime = time.Hour
	m.MaxResponseTime = 0
	m.TotalResponseTime = 0
	m.ResponseTimeCount = 0
	m.EventsProcessed = 0
	m.EventsIgnored = 0
	m.ChannelMetrics = make(map[string]*ChannelMetrics)
}

// GetSuccessRate returns the success rate as a percentage
func (m *Metrics) GetSuccessRate() float64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	total := m.TotalSent + m.TotalFailed
	if total == 0 {
		return 100.0
	}

	return float64(m.TotalSent) / float64(total) * 100.0
}

// GetChannelSuccessRate returns the success rate for a specific channel
func (m *Metrics) GetChannelSuccessRate(channel string) float64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	cm, ok := m.ChannelMetrics[channel]
	if !ok {
		return 100.0
	}

	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	total := cm.Sent + cm.Failed
	if total == 0 {
		return 100.0
	}

	return float64(cm.Sent) / float64(total) * 100.0
}
