package types

import (
	"sync"
	"time"
)

// This file is to set the metadata for the project
func (GM *GlobalManager) NewMetadata() *Metadata {
	// If metadata already exists, return it
	if GM.GetMetadata() != nil {
		return GM.GetMetadata()
	}

	md := &Metadata{
		mu:              &sync.RWMutex{},
		Metrics:         false,
		ShutdownTimeout: 10 * time.Second,
		UpdateInterval:  UpdateInterval,
	}
	GM.SetMetadata(md)
	return md
}

// Right now our codebase not respecting the max routines limit, will set it
func (MD *Metadata) SetMaxRoutines(maxroutines int) *Metadata {
	// Lock and update
	MD.mu.Lock()
	defer MD.mu.Unlock()
	MD.MaxRoutines = maxroutines
	return MD
}

func (MD *Metadata) SetShutdownTimeout(timeout time.Duration) *Metadata {
	// Lock and update
	MD.mu.Lock()
	defer MD.mu.Unlock()
	MD.ShutdownTimeout = timeout
	// Set to global variable
	ShutdownTimeout = timeout
	return MD
}

func (MD *Metadata) SetMetrics(metrics bool, URL string, interval time.Duration) *Metadata {
	// Lock and update
	MD.mu.Lock()
	defer MD.mu.Unlock()
	MD.Metrics = metrics
	MD.MetricsURL = URL
	MD.UpdateInterval = interval
	// Set to global variable (similar to ShutdownTimeout)
	// The collector will observe this change via types.UpdateInterval
	if interval > 0 {
		UpdateInterval = interval
	}
	return MD
}

func (MD *Metadata) GetMetadata() *Metadata {
	// Lock and update
	MD.mu.RLock()
	defer MD.mu.RUnlock()
	return MD
}

func (MD *Metadata) UpdateIntervalTime(time time.Duration) time.Duration {
	// Lock and update
	MD.mu.Lock()
	defer MD.mu.Unlock()
	// Set to global variable
	if time > 0 {
		UpdateInterval = time
	}
	return MD.UpdateInterval
}

// ✅ ADD these getter methods
func (MD *Metadata) GetMetrics() bool {
    MD.mu.RLock()
    defer MD.mu.RUnlock()
    return MD.Metrics
}

func (MD *Metadata) GetMaxRoutines() int {
    MD.mu.RLock()
    defer MD.mu.RUnlock()
    return MD.MaxRoutines
}

func (MD *Metadata) GetUpdateInterval() time.Duration {
    MD.mu.RLock()
    defer MD.mu.RUnlock()
    return MD.UpdateInterval
}

func (MD *Metadata) GetShutdownTimeout() time.Duration {
    MD.mu.RLock()
    defer MD.mu.RUnlock()
    return MD.ShutdownTimeout
}

func (MD *Metadata) GetMetricsURL() string {
    MD.mu.RLock()
    defer MD.mu.RUnlock()
    return MD.MetricsURL
}