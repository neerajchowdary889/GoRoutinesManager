package types

import (
	"context"
	"sync"
)

// GlobalGoroutineManager manages all app-level goroutine managers
type GlobalGoroutineManager struct {
	GlobalMu    sync.RWMutex
	AppManagers map[string]*AppGoroutineManager
	Ctx         context.Context
	Cancel      context.CancelFunc
	Wg          sync.WaitGroup
}

// AppGoroutineManager manages local-level goroutine managers for a specific app/module
type AppGoroutineManager struct {
	LocalMu       sync.RWMutex
	AppName       string
	LocalManagers map[string]*LocalGoroutineManager
	Ctx           context.Context
	Cancel        context.CancelFunc
	Wg            sync.WaitGroup
	ParentCtx     context.Context
}

// LocalGoroutineManager manages goroutines for a specific file/module within an app
type LocalGoroutineManager struct {
	LocalMu     sync.RWMutex
	LocalName   string
	Routines    map[string]*Routine
	Ctx         context.Context
	Cancel      context.CancelFunc
	Wg          sync.WaitGroup
	FunctionWgs map[string]*sync.WaitGroup // Per function name for selective shutdown
	ParentCtx   context.Context
}

// Routine represents a tracked goroutine
type Routine struct {
	ID           string
	FunctionName string
	Ctx          context.Context
	Done         chan struct{}
	StartedAt    int64 // Unix timestamp or monotonic time
}
