package event

import "sync"

var (
	globalBus      *EventBus
	globalBusOnce  sync.Once
	globalBusMutex sync.RWMutex
)

// GetGlobalBus returns the global event bus instance
func GetGlobalBus() *EventBus {
	globalBusOnce.Do(func() {
		globalBus = NewEventBus(true) // Async by default for better performance
	})
	return globalBus
}

// SetGlobalBus sets the global event bus (useful for testing)
func SetGlobalBus(bus *EventBus) {
	globalBusMutex.Lock()
	defer globalBusMutex.Unlock()
	globalBus = bus
}

// ResetGlobalBus resets the global bus (useful for testing)
func ResetGlobalBus() {
	globalBusMutex.Lock()
	defer globalBusMutex.Unlock()
	globalBus = nil
	globalBusOnce = sync.Once{}
}
