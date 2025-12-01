# RootManager Implementation: Concrete Before/After

## Summary

This document shows the **concrete implementation** of the RootManager pattern, demonstrating the before/after transformation from singleton to RootManager.

**Key Insight:** RootManager is semantically better than Container because:
- It's the **root of the supervision tree** (domain-specific)
- It **directly replaces GlobalManager** (no extra indirection)
- It includes **DefaultRoot()** for convenience (built-in, not afterthought)

---

## Before: Singleton Pattern

### `types/types.go` (Current)
```go
// Singleton pattern to not repeat the same managers again
var (
	Global *GlobalManager  // ❌ Global state
)

func GetGlobalManager() (*GlobalManager, error) {
	if Global == nil {
		return nil, Errors.ErrGlobalManagerNotFound
	}
	return Global, nil
}
```

### `types/SIngleton.go` (Current)
```go
func SetGlobalManager(global *GlobalManager) {
	Once.Do(func() {
		Global = global  // ❌ Sets global variable
	})
}

func GetAppManager(appName string) (*AppManager, error) {
	if !IsIntilized().App(appName) {
		return nil, Errors.ErrAppManagerNotFound
	}
	return Global.GetAppManager(appName)  // ❌ Accesses global
}
```

### Usage (Current)
```go
// ❌ Implicit global dependency
globalMgr := Global.NewGlobalManager()
globalMgr.Init()  // Uses types.Global internally

appMgr := App.NewAppManager("payments")
app, _ := appMgr.CreateApp()  // Uses types.Global internally
```

---

## After: RootManager Pattern

### `types/root.go` (New)
```go
// Root is the root of the supervision tree.
// It replaces the singleton GlobalManager pattern.
type Root struct {
	mu          sync.RWMutex
	apps        map[string]*AppManager  // ✅ Owns apps directly
	ctx         context.Context
	cancel      context.CancelFunc
	wg          *sync.WaitGroup
	metadata    *Metadata
	shutdownTimeout time.Duration
	maxRoutines     int
}

// NewRoot creates a new root instance (explicit, testable)
func NewRoot(opts ...RootOption) *Root {
	r := &Root{
		apps:            make(map[string]*AppManager),
		wg:              &sync.WaitGroup{},
		shutdownTimeout: 10 * time.Second,
	}
	// ... initialization
	return r
}

// App returns or creates an AppManager
func (r *Root) App(name string) *AppManager {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if app, ok := r.apps[name]; ok {
		return app
	}
	
	app := newAppManager(r, name)  // ✅ Passes root reference
	r.apps[name] = app
	return app
}
```

### `types/default.go` (New)
```go
var (
	defaultRootOnce sync.Once
	defaultRoot     *Root
)

// DefaultRoot provides singleton-like convenience
func DefaultRoot() *Root {
	defaultRootOnce.Do(func() {
		defaultRoot = NewRoot()  // ✅ Creates instance, not global var
	})
	return defaultRoot
}
```

### Usage (New - Explicit)
```go
// ✅ Explicit root - testable, no global state
root := types.NewRoot(
	types.WithShutdownTimeout(5*time.Second),
	types.WithMaxRoutines(100),
)

app := root.App("payments")        // ✅ Direct access
local := app.Local("settlement")   // ✅ Chainable API
local.Go("worker", fn)
```

### Usage (New - Convenience)
```go
// ✅ Convenience API - uses DefaultRoot() internally
app := types.DefaultRoot().App("payments")
local := app.Local("settlement")
local.Go("worker", fn)
```

---

## Key Differences

### 1. **Ownership Model**

**Before (Singleton):**
```
types.Global (package-level var)
  └── AppManagers map[string]*AppManager
```

**After (RootManager):**
```
Root (explicit instance)
  └── apps map[string]*AppManager
```

### 2. **Creation**

**Before:**
```go
// ❌ Implicit global initialization
Global := types.NewGlobalManager()
types.SetGlobalManager(Global)
```

**After:**
```go
// ✅ Explicit instance creation
root := types.NewRoot()
// OR
root := types.DefaultRoot()  // Convenience
```

### 3. **Access Pattern**

**Before:**
```go
// ❌ Accesses global through package functions
app, _ := types.GetAppManager("payments")
```

**After:**
```go
// ✅ Direct method call on root instance
app := root.App("payments")
```

### 4. **Testing**

**Before:**
```go
func TestSomething(t *testing.T) {
	// ❌ Must reset global state
	types.Global = nil
	// ❌ Can't run tests in parallel
	// Test code...
}
```

**After:**
```go
func TestSomething(t *testing.T) {
	// ✅ Fresh instance per test
	root := types.NewRoot()
	// ✅ Can run in parallel
	t.Parallel()
	// Test code...
}
```

---

## Migration Strategy

### Phase 1: Add Root Type (Non-Breaking)

1. ✅ Add `types/root.go` with Root type
2. ✅ Add `types/default.go` with DefaultRoot()
3. ✅ Keep existing singleton code working
4. ⚠️ Mark singleton as deprecated

### Phase 2: Update AppManager/LocalManager

**Current:**
```go
type AppManager struct {
	AppName       string
	LocalManagers map[string]*LocalManager
	// ... no reference to root
}
```

**Updated:**
```go
type AppManager struct {
	AppName       string
	root          *Root  // ✅ Reference to root
	LocalManagers map[string]*LocalManager
	// ...
}

// newAppManager creates AppManager attached to root
func newAppManager(root *Root, appName string) *AppManager {
	return &AppManager{
		AppName: appName,
		root:    root,  // ✅ Stores root reference
		// ...
	}
}
```

### Phase 3: Update Manager Implementations

**Current:**
```go
func (GM *GlobalManager) Init() error {
	Global := types.NewGlobalManager()  // ❌ Uses global
	types.SetGlobalManager(Global)
	return nil
}
```

**Updated:**
```go
type GlobalManager struct {
	root *types.Root  // ✅ Holds root reference
}

func NewGlobalManager(root *types.Root) *GlobalManager {
	if root == nil {
		root = types.DefaultRoot()  // Fallback
	}
	return &GlobalManager{root: root}
}

func (GM *GlobalManager) Init() error {
	// Root is already initialized when created
	return nil
}
```

### Phase 4: Update Public API

**Add convenience functions:**
```go
// grm.go (package-level)
package grm

// App is convenience function using default root
func App(name string) *types.AppManager {
	return types.DefaultRoot().App(name)
}

// Local is convenience function using default root
func Local(appName, localName string) *types.LocalManager {
	return types.DefaultRoot().App(appName).Local(localName)
}
```

---

## API Comparison

### RootManager Style (Recommended)
```go
// Explicit (best for tests, complex apps)
root := grm.NewRoot()
app := root.App("payments")
local := app.Local("settlement")

// Convenience (simple apps)
app := grm.App("payments")
local := grm.Local("payments", "settlement")
```

### Container Style (Alternative)
```go
// More verbose, extra indirection
container := grm.NewContainer()
globalMgr := grm.NewGlobalManager(container)
app := globalMgr.App("payments")
local := app.Local("settlement")
```

**Why RootManager is better:**
- ✅ Direct: `root.App()` vs `container.GetGlobal().App()`
- ✅ Semantic: "Root" fits supervision tree model
- ✅ Simpler: No extra Container layer
- ✅ Built-in default: DefaultRoot() included

---

## Complete Example

### Before (Singleton)
```go
package main

import (
	"context"
	"github.com/neerajchowdary889/GoRoutinesManager/Manager/Global"
	"github.com/neerajchowdary889/GoRoutinesManager/Manager/App"
	"github.com/neerajchowdary889/GoRoutinesManager/Manager/Local"
)

func main() {
	// ❌ Implicit global initialization
	globalMgr := Global.NewGlobalManager()
	globalMgr.Init()
	
	appMgr := App.NewAppManager("payments")
	app, _ := appMgr.CreateApp()
	
	localMgr := Local.NewLocalManager("payments", "settlement")
	local, _ := localMgr.CreateLocal("settlement")
	
	localMgr.Go("worker", func(ctx context.Context) error {
		// ...
		return nil
	})
	
	globalMgr.Shutdown(true)
}
```

### After (RootManager)
```go
package main

import (
	"context"
	"time"
	"github.com/neerajchowdary889/GoRoutinesManager/types"
)

func main() {
	// ✅ Explicit root with options
	root := types.NewRoot(
		types.WithShutdownTimeout(5*time.Second),
		types.WithMaxRoutines(100),
	)
	
	// ✅ Direct, chainable API
	app := root.App("payments")
	local := app.Local("settlement")
	
	local.Go("worker", func(ctx context.Context) error {
		// ...
		return nil
	})
	
	// ✅ Shutdown root
	root.Shutdown(true)
}
```

### After (Convenience API)
```go
package main

import (
	"context"
	"github.com/neerajchowdary889/GoRoutinesManager/grm"
)

func main() {
	// ✅ Convenience API (uses DefaultRoot internally)
	local := grm.Local("payments", "settlement")
	
	local.Go("worker", func(ctx context.Context) error {
		// ...
		return nil
	})
	
	// ✅ Shutdown default root
	grm.DefaultRoot().Shutdown(true)
}
```

---

## Benefits Summary

| Aspect | Singleton | RootManager |
|--------|-----------|-------------|
| **Testability** | ❌ Hard (global state) | ✅ Easy (fresh instances) |
| **Parallel Tests** | ❌ No | ✅ Yes |
| **Multiple Instances** | ❌ No | ✅ Yes |
| **Explicit Dependencies** | ❌ No | ✅ Yes |
| **Semantic Clarity** | ⚠️ Generic | ✅ Domain-specific |
| **Migration Path** | N/A | ✅ Gradual (backward compat) |

---

## Next Steps

1. **Review the RootManager implementation** in `types/root.go`
2. **Update AppManager/LocalManager** to reference Root
3. **Add DefaultRoot()** for convenience
4. **Update Manager implementations** to accept Root
5. **Add migration helpers** for backward compatibility
6. **Update tests** to use Root
7. **Update documentation** with new API

The RootManager pattern provides the best balance of:
- ✅ **Explicit control** (for tests and complex apps)
- ✅ **Convenience** (DefaultRoot for simple apps)
- ✅ **Semantic clarity** (fits supervision tree model)
- ✅ **Backward compatibility** (migration helpers)

---

*This implementation follows Go best practices and provides a clean migration path from singleton to RootManager.*

