package contextlock

import (
	"context"
	"sync"
)

// CRWMutex is a context-aware reader-writer mutex that allows multiple
// concurrent readers or a single writer, with context-based cancellation support.
type CRWMutex struct {
	mu            sync.Mutex
	readers       int
	writer        bool
	readerSignal  *sync.Cond
	writerSignal  *sync.Cond
}

// NewCRWMutex creates a new context-aware reader-writer mutex
func NewCRWMutex() *CRWMutex {
	m := &CRWMutex{}
	m.readerSignal = sync.NewCond(&m.mu)
	m.writerSignal = sync.NewCond(&m.mu)
	return m
}

// RLock acquires a read lock. Multiple readers can hold the lock simultaneously.
// Returns an error if the context is cancelled before acquiring the lock.
func (m *CRWMutex) RLock(ctx context.Context) error {
	// Check context before attempting to lock
	if err := ctx.Err(); err != nil {
		return err
	}

	acquired := make(chan struct{})
	go func() {
		m.mu.Lock()
		// Wait while there's a writer
		for m.writer {
			m.readerSignal.Wait()
		}
		m.readers++
		m.mu.Unlock()
		close(acquired)
	}()

	select {
	case <-acquired:
		return nil
	case <-ctx.Done():
		// Try to clean up if we haven't acquired yet
		// This is best-effort; if we already acquired, RUnlock must be called
		return ctx.Err()
	}
}

// RUnlock releases a read lock
func (m *CRWMutex) RUnlock() {
	m.mu.Lock()
	m.readers--
	if m.readers == 0 {
		// Signal waiting writer that all readers are done
		m.writerSignal.Signal()
	}
	m.mu.Unlock()
}

// Lock acquires a write lock. Only one writer can hold the lock at a time,
// and no readers can be active. Returns an error if the context is cancelled.
func (m *CRWMutex) Lock(ctx context.Context) error {
	// Check context before attempting to lock
	if err := ctx.Err(); err != nil {
		return err
	}

	acquired := make(chan struct{})
	go func() {
		m.mu.Lock()
		// Wait while there's another writer or any readers
		for m.writer || m.readers > 0 {
			m.writerSignal.Wait()
		}
		m.writer = true
		m.mu.Unlock()
		close(acquired)
	}()

	select {
	case <-acquired:
		return nil
	case <-ctx.Done():
		// Try to clean up if we haven't acquired yet
		return ctx.Err()
	}
}

// Unlock releases a write lock
func (m *CRWMutex) Unlock() {
	m.mu.Lock()
	m.writer = false
	// Signal all waiting readers
	m.readerSignal.Broadcast()
	// Signal one waiting writer
	m.writerSignal.Signal()
	m.mu.Unlock()
}

// TryRLock attempts to acquire a read lock without blocking.
// Returns true if successful, false otherwise.
func (m *CRWMutex) TryRLock() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.writer {
		return false
	}
	m.readers++
	return true
}

// TryLock attempts to acquire a write lock without blocking.
// Returns true if successful, false otherwise.
func (m *CRWMutex) TryLock() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.writer || m.readers > 0 {
		return false
	}
	m.writer = true
	return true
}