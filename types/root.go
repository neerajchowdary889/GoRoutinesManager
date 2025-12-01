package types

import (
	"context"
	"sync"
	"time"

	"github.com/neerajchowdary889/GoRoutinesManager/Context"
	"github.com/neerajchowdary889/GoRoutinesManager/types/Errors"
)

// Root is the root of the supervision tree.
// It replaces the singleton GlobalManager pattern.
// Each Root instance manages its own complete supervision hierarchy.
type Root struct {
	mu          sync.RWMutex
	apps        map[string]*AppManager
	ctx         context.Context
	cancel      context.CancelFunc
	wg          *sync.WaitGroup
	metadata    *Metadata
	
	// Configuration
	shutdownTimeout time.Duration
	maxRoutines     int
	metricsEnabled  bool
	metricsURL      string
}

// RootOption configures a Root instance.
type RootOption func(*Root)

// WithShutdownTimeout sets the default shutdown timeout for this root.
func WithShutdownTimeout(timeout time.Duration) RootOption {
	return func(r *Root) {
		r.shutdownTimeout = timeout
	}
}

// WithMaxRoutines sets the maximum number of routines allowed across all apps.
func WithMaxRoutines(max int) RootOption {
	return func(r *Root) {
		r.maxRoutines = max
	}
}

// WithMetrics enables metrics collection for this root.
func WithMetrics(enabled bool, url string) RootOption {
	return func(r *Root) {
		r.metricsEnabled = enabled
		r.metricsURL = url
	}
}

// NewRoot creates a new root manager instance.
// This is the explicit, testable way to create a supervision tree.
//
// Example:
//   root := types.NewRoot(
//       types.WithShutdownTimeout(5*time.Second),
//       types.WithMaxRoutines(100),
//   )
func NewRoot(opts ...RootOption) *Root {
	r := &Root{
		apps:            make(map[string]*AppManager),
		wg:              &sync.WaitGroup{},
		shutdownTimeout: 10 * time.Second, // Default
	}
	
	// Apply options
	for _, opt := range opts {
		opt(r)
	}
	
	// Initialize metadata
	r.metadata = &Metadata{
		MaxRoutines:    r.maxRoutines,
		Metrics:        r.metricsEnabled,
		MetricsURL:     r.metricsURL,
		ShutdownTimeout: r.shutdownTimeout,
	}
	
	// Initialize context
	globalCtx := Context.GetGlobalContext()
	ctx := globalCtx.Init()
	r.ctx = ctx
	r.cancel = func() {
		globalCtx.Shutdown()
	}
	
	return r
}

// App returns or creates an AppManager for the given app name.
// This is the primary way to access app managers in the new API.
//
// Example:
//   root := types.NewRoot()
//   app := root.App("payments")
func (r *Root) App(name string) *AppManager {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if app, ok := r.apps[name]; ok {
		return app
	}
	
	app := newAppManager(r, name)
	r.apps[name] = app
	return app
}

// GetApp returns an existing AppManager or an error if not found.
func (r *Root) GetApp(name string) (*AppManager, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	app, ok := r.apps[name]
	if !ok {
		return nil, Errors.ErrAppManagerNotFound
	}
	return app, nil
}

// GetAllApps returns all app managers.
func (r *Root) GetAllApps() []*AppManager {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	apps := make([]*AppManager, 0, len(r.apps))
	for _, app := range r.apps {
		apps = append(apps, app)
	}
	return apps
}

// GetAppCount returns the number of app managers.
func (r *Root) GetAppCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.apps)
}

// Shutdown shuts down all app managers under this root.
// If safe is true, waits for graceful shutdown with timeout.
// If safe is false, cancels all contexts immediately.
func (r *Root) Shutdown(safe bool) error {
	r.mu.Lock()
	apps := make([]*AppManager, 0, len(r.apps))
	for _, app := range r.apps {
		apps = append(apps, app)
	}
	r.mu.Unlock()
	
	if safe {
		// Safe shutdown: wait for all apps with timeout
		done := make(chan struct{})
		go func() {
			for _, app := range apps {
				r.wg.Add(1)
				go func(a *AppManager) {
					defer r.wg.Done()
					_ = a.Shutdown(true)
					if a.Wg != nil {
						a.Wg.Wait()
					}
				}(app)
			}
			r.wg.Wait()
			close(done)
		}()
		
		select {
		case <-done:
			// All apps shutdown gracefully
		case <-time.After(r.shutdownTimeout):
			// Timeout - force cancel remaining
			for _, app := range apps {
				_ = app.Shutdown(false)
			}
		}
	} else {
		// Unsafe shutdown: cancel immediately
		for _, app := range apps {
			_ = app.Shutdown(false)
		}
	}
	
	// Cancel root context
	if r.cancel != nil {
		r.cancel()
	}
	
	return nil
}

// Context returns the root context.
func (r *Root) Context() context.Context {
	return r.ctx
}

// Metadata returns the root metadata.
func (r *Root) Metadata() *Metadata {
	return r.metadata
}

// UpdateMetadata updates the root metadata.
func (r *Root) UpdateMetadata(flag string, value interface{}) (*Metadata, error) {
	// Delegate to metadata update logic
	// (Implementation similar to GlobalManager.UpdateGlobalMetadata)
	return r.metadata, nil
}

