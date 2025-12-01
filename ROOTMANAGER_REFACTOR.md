# RootManager Refactoring: Concrete Implementation Guide

## Overview

This document provides a concrete refactoring from the singleton pattern to the **RootManager + optional Default** pattern, which is semantically better suited for a goroutine supervision library.

**Key Principle:** The root of the supervision tree (RootManager) replaces the global singleton, while maintaining backward compatibility through an optional default instance.

---

## Architecture Comparison

### Before (Singleton):
```
types.Global (package-level var)
  └── AppManagers map[string]*AppManager
      └── LocalManagers map[string]*LocalManager
          └── Routines map[string]*Routine
```

### After (RootManager):
```
Root (explicit instance)
  └── apps map[string]*AppManager
      └── locals map[string]*LocalManager
          └── routines map[string]*Routine

Optional: DefaultRoot() → returns default Root instance
```

---

## Step 1: Create Root Type (Replaces GlobalManager)

### New File: `types/root.go`

```go
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
type Root struct {
	mu          sync.RWMutex
	apps        map[string]*AppManager
	ctx         context.Context
	cancel      context.CancelFunc
	wg          *sync.WaitGroup
	metadata    *Metadata
	
	// Options
	shutdownTimeout time.Duration
	maxRoutines     int
	metricsEnabled  bool
	metricsURL      string
}

// RootOption configures a Root instance.
type RootOption func(*Root)

// WithShutdownTimeout sets the default shutdown timeout.
func WithShutdownTimeout(timeout time.Duration) RootOption {
	return func(r *Root) {
		r.shutdownTimeout = timeout
	}
}

// WithMaxRoutines sets the maximum number of routines allowed.
func WithMaxRoutines(max int) RootOption {
	return func(r *Root) {
		r.maxRoutines = max
	}
}

// WithMetrics enables metrics collection.
func WithMetrics(enabled bool, url string) RootOption {
	return func(r *Root) {
		r.metricsEnabled = enabled
		r.metricsURL = url
	}
}

// NewRoot creates a new root manager instance.
// This is the explicit, testable way to create a supervision tree.
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
	r.ctx, r.cancel = Context.GetGlobalContext().Init(), func() {
		Context.GetGlobalContext().Shutdown()
	}
	
	return r
}

// App returns or creates an AppManager for the given app name.
func (r *Root) App(name string) *AppManager {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if app, ok := r.apps[name]; ok {
		return app
	}
	
	app := &AppManager{
		AppName:       name,
		root:          r,
		LocalManagers: make(map[string]*LocalManager),
		wg:            &sync.WaitGroup{},
	}
	
	// Set app context
	app.SetAppContext()
	
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

// Shutdown shuts down all app managers.
func (r *Root) Shutdown(safe bool) error {
	r.mu.Lock()
	apps := make([]*AppManager, 0, len(r.apps))
	for _, app := range r.apps {
		apps = append(apps, app)
	}
	r.mu.Unlock()
	
	if safe {
		// Safe shutdown: wait for all apps
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
	// Implementation similar to GlobalManager.UpdateGlobalMetadata
	// ... (see existing implementation)
	return r.metadata, nil
}
```

---

## Step 2: Update AppManager to Reference Root

### Modified: `types/BuilderAppManager.go`

```go
package types

import (
	"context"
	"fmt"
	"sync"

	"github.com/neerajchowdary889/GoRoutinesManager/Context"
	"github.com/neerajchowdary889/GoRoutinesManager/types/Errors"
)

const (
	Prefix_AppManager = "AppManager."
)

// NewAppManager creates a new AppManager attached to a root.
// OLD: func NewAppManager(appName string) *AppManager
// NEW: AppManager is created via root.App(name)
func newAppManager(root *Root, appName string) *AppManager {
	appMgr := &AppManager{
		AppName:       appName,
		root:          root,  // ✅ Reference to root instead of global
		LocalManagers: make(map[string]*LocalManager),
		wg:            &sync.WaitGroup{},
	}
	appMgr.SetAppContext()
	return appMgr
}

// AppManager now has a reference to root
type AppManager struct {
	AppMu         *sync.RWMutex
	AppName       string
	root          *Root  // ✅ Reference to root
	LocalManagers map[string]*LocalManager
	Ctx           context.Context
	Cancel        context.CancelFunc
	Wg            *sync.WaitGroup
	ParentCtx     context.Context
}

// Local returns or creates a LocalManager for the given local name.
func (AM *AppManager) Local(localName string) *LocalManager {
	AM.LockAppWriteMutex()
	defer AM.UnlockAppWriteMutex()
	
	if local, ok := AM.LocalManagers[localName]; ok {
		return local
	}
	
	local := newLocalManager(AM.root, AM.AppName, localName)
	local.SetParentContext(AM.ParentCtx)
	AM.LocalManagers[localName] = local
	return local
}

// GetLocal returns an existing LocalManager or an error.
func (AM *AppManager) GetLocal(localName string) (*LocalManager, error) {
	AM.LockAppReadMutex()
	defer AM.UnlockAppReadMutex()
	
	if _, ok := AM.LocalManagers[localName]; !ok {
		return nil, fmt.Errorf("%w: %s", Errors.ErrLocalManagerNotFound, localName)
	}
	return AM.LocalManagers[localName], nil
}

// ... rest of AppManager methods remain similar, but remove references to Global
```

---

## Step 3: Update LocalManager to Reference Root

### Modified: `types/BuilderLocalManager.go`

```go
package types

import (
	"context"
	"fmt"
	"sync"

	"github.com/neerajchowdary889/GoRoutinesManager/Context"
	"github.com/neerajchowdary889/GoRoutinesManager/types/Errors"
)

const (
	Prefix_LocalManager = "LocalManager."
)

// newLocalManager creates a new LocalManager attached to a root.
func newLocalManager(root *Root, appName, localName string) *LocalManager {
	LocalManager := &LocalManager{
		LocalName:   localName,
		root:        root,  // ✅ Reference to root
		Routines:    make(map[string]*Routine),
		FunctionWgs: make(map[string]*sync.WaitGroup),
		Wg:          &sync.WaitGroup{},
	}
	return LocalManager
}

// LocalManager now has a reference to root
type LocalManager struct {
	LocalMu     *sync.RWMutex
	LocalName   string
	root        *Root  // ✅ Reference to root
	Routines    map[string]*Routine
	Ctx         context.Context
	Cancel      context.CancelFunc
	Wg          *sync.WaitGroup
	FunctionWgs map[string]*sync.WaitGroup
	ParentCtx   context.Context
}

// ... rest of LocalManager methods remain similar, but remove references to Global
```

---

## Step 4: Provide Default Root (Optional Convenience)

### New File: `types/default.go`

```go
package types

import (
	"sync"
)

var (
	defaultRootOnce sync.Once
	defaultRoot     *Root
)

// DefaultRoot returns the default root instance.
// This provides a singleton-like convenience for simple applications,
// but is optional - you can always create your own Root with NewRoot().
func DefaultRoot() *Root {
	defaultRootOnce.Do(func() {
		defaultRoot = NewRoot()
	})
	return defaultRoot
}

// ResetDefaultRoot resets the default root (useful for testing).
// This is exported for testing purposes only.
func ResetDefaultRoot() {
	defaultRootOnce = sync.Once{}
	defaultRoot = nil
}
```

---

## Step 5: Update Public API (Backward Compatible)

### New File: `grm.go` (Package-level convenience functions)

```go
package grm

import (
	"context"
	"github.com/neerajchowdary889/GoRoutinesManager/types"
)

// App is a convenience function that uses the default root.
// For explicit control, use root.App(name) instead.
func App(name string) *types.AppManager {
	return types.DefaultRoot().App(name)
}

// Local is a convenience function that uses the default root.
// For explicit control, use root.App(appName).Local(localName) instead.
func Local(appName, localName string) *types.LocalManager {
	return types.DefaultRoot().App(appName).Local(localName)
}

// NewRoot creates a new root manager instance.
// Use this for explicit control, testing, or multiple supervision trees.
func NewRoot(opts ...types.RootOption) *types.Root {
	return types.NewRoot(opts...)
}

// DefaultRoot returns the default root instance.
func DefaultRoot() *types.Root {
	return types.DefaultRoot()
}
```

---

## Step 6: Migration Helpers (Backward Compatibility)

### New File: `types/migration.go`

```go
package types

// LegacyGlobalManager provides backward compatibility with old singleton API.
// This wraps the default root to match the old GlobalManager interface.
type LegacyGlobalManager struct {
	root *Root
}

// NewLegacyGlobalManager creates a legacy wrapper around the default root.
// This allows old code to continue working during migration.
func NewLegacyGlobalManager() *LegacyGlobalManager {
	return &LegacyGlobalManager{
		root: DefaultRoot(),
	}
}

// Init initializes the root (no-op if already initialized).
func (lgm *LegacyGlobalManager) Init() error {
	// Root is already initialized when created
	return nil
}

// Shutdown shuts down the root.
func (lgm *LegacyGlobalManager) Shutdown(safe bool) error {
	return lgm.root.Shutdown(safe)
}

// GetAllAppManagers returns all app managers.
func (lgm *LegacyGlobalManager) GetAllAppManagers() ([]*AppManager, error) {
	return lgm.root.GetAllApps(), nil
}

// GetAppManagerCount returns the number of app managers.
func (lgm *LegacyGlobalManager) GetAppManagerCount() int {
	return lgm.root.GetAppAppCount()
}

// ... implement other GlobalManager interface methods
```

---

## Step 7: Update Manager Implementations

### Modified: `Manager/Global/GlobalManager.go`

```go
package Global

import (
	"github.com/neerajchowdary889/GoRoutinesManager/Manager/Interface"
	"github.com/neerajchowdary889/GoRoutinesManager/types"
)

type GlobalManager struct {
	root *types.Root  // ✅ Holds reference to root
}

// NewGlobalManager creates a GlobalManager that wraps a root.
// For new code, use root.App() directly instead.
func NewGlobalManager(root *types.Root) Interface.GlobalGoroutineManagerInterface {
	if root == nil {
		root = types.DefaultRoot()  // Fallback to default
	}
	return &GlobalManager{
		root: root,
	}
}

func (GM *GlobalManager) Init() error {
	// Root is already initialized when created
	return nil
}

func (GM *GlobalManager) Shutdown(safe bool) error {
	return GM.root.Shutdown(safe)
}

func (GM *GlobalManager) GetAllAppManagers() ([]*types.AppManager, error) {
	return GM.root.GetAllApps(), nil
}

// ... rest of methods delegate to root
```

---

## Usage Examples

### Explicit Root (Recommended for New Code)

```go
package main

import (
	"context"
	"time"
	"github.com/neerajchowdary889/GoRoutinesManager/types"
)

func main() {
	// ✅ Explicit root - testable, no global state
	root := types.NewRoot(
		types.WithShutdownTimeout(5*time.Second),
		types.WithMaxRoutines(100),
		types.WithMetrics(true, "http://localhost:9090"),
	)
	
	// Create app manager
	app := root.App("payments")
	
	// Create local manager
	local := app.Local("settlement")
	
	// Spawn goroutine
	local.Go("worker", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(1 * time.Second):
				// Do work
			}
		}
	})
	
	// Shutdown
	root.Shutdown(true)
}
```

### Default Root (Convenience for Simple Apps)

```go
package main

import (
	"context"
	"github.com/neerajchowdary889/GoRoutinesManager/grm"
)

func main() {
	// ✅ Convenience API - uses default root internally
	local := grm.Local("payments", "settlement")
	
	local.Go("worker", func(ctx context.Context) error {
		// ...
		return nil
	})
	
	// Shutdown default root
	grm.DefaultRoot().Shutdown(true)
}
```

### Testing (Isolated Instances)

```go
func TestSomething(t *testing.T) {
	// ✅ Each test gets fresh root - no global state
	root := types.NewRoot()
	
	app := root.App("test-app")
	local := app.Local("test-local")
	
	// Test code...
	
	// Cleanup
	root.Shutdown(true)
}

func TestParallel(t *testing.T) {
	t.Parallel()  // ✅ Now safe - no shared global state
	
	root := types.NewRoot()
	// ... test code
}
```

---

## Migration Path

### Phase 1: Add Root Type (Non-Breaking)

1. Add `types/root.go` with Root type
2. Add `types/default.go` with DefaultRoot()
3. Keep existing singleton code working
4. Mark singleton as deprecated

### Phase 2: Update Internal Code

1. Update AppManager and LocalManager to reference Root
2. Update Manager implementations to use Root
3. Keep legacy wrappers for backward compatibility

### Phase 3: Update Public API

1. Add convenience functions in `grm.go`
2. Update examples to use Root
3. Update documentation

### Phase 4: Remove Singleton (Breaking Change)

1. Remove `types.Global` variable
2. Remove singleton functions
3. Require Root in all constructors

---

## Benefits Summary

### ✅ Testability
- Each test gets a fresh Root instance
- No global state to reset
- Tests can run in parallel

### ✅ Flexibility
- Multiple Root instances in same process
- Different configurations per Root
- Explicit dependencies

### ✅ Backward Compatibility
- DefaultRoot() provides singleton-like convenience
- Legacy wrappers during migration
- Gradual migration path

### ✅ Semantic Clarity
- RootManager fits the supervision tree model
- Clear hierarchy: Root → App → Local → Routine
- No "container" abstraction needed

---

## Comparison: RootManager vs Container

| Aspect | RootManager | Container |
|--------|-------------|-----------|
| **Semantic Fit** | ✅ Perfect (supervision tree) | ⚠️ Generic (service registry) |
| **Naming** | ✅ Root/Tree/Supervisor | ⚠️ Container/Registry |
| **Default Instance** | ✅ Built-in (DefaultRoot) | ❌ Not included |
| **Migration** | ✅ Direct (Global → Root) | ⚠️ Extra layer (Global → Container → Global) |
| **API Clarity** | ✅ `root.App().Local()` | ⚠️ `container.GetGlobal().App()` |

**Recommendation:** Use **RootManager** pattern for this codebase. It's semantically better, includes default instance support, and provides a cleaner migration path.

---

## Next Steps

1. **Implement Root type** (`types/root.go`)
2. **Update AppManager/LocalManager** to reference Root
3. **Add DefaultRoot()** for convenience
4. **Update Manager implementations** to use Root
5. **Add migration helpers** for backward compatibility
6. **Update tests** to use Root instead of singleton
7. **Update documentation** with new API

---

*End of Refactoring Guide*

