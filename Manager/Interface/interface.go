package Interface

import (
	"context"
	"sync"
	"time"

	"github.com/neerajchowdary889/GoRoutinesManager/types"
)

// Initializer initializes the manager
type LifeCycle interface {
	Init() error
	Shutdown(safe bool) error
}


// GoroutineSpawner spawns and tracks goroutines
type GoroutineSpawner interface {
	Go(functionName string, workerFunc func(ctx context.Context) error) error
}

// FunctionShutdowner handles shutdown of specific functions
type FunctionShutdowner interface {
	ShutdownFunction(functionName string, timeout time.Duration) error
}

// GoroutineLister lists all tracked goroutines
type GoroutineLister interface {
	GetAllGoroutines() ([]*types.Routine, error)
	GetGoroutineCount() int
	GetFunctionCount(functionName string) int
}

// LocalManagerLister lists all local managers
type LocalManagerLister interface {
	GetAllLocalManagers() ([]*types.LocalGoroutineManager, error)
	GetLocalManagerCount() int
}

// AppManagerLister lists all app managers
type AppManagerLister interface {
	GetAllAppManagers() ([]*types.AppGoroutineManager, error)
	GetAppManagerCount() int
}

// AppManagerCreator creates new app managers
type AppManagerCreator interface {
	CreateApp(appName string) (*types.AppGoroutineManager, error)
}

// LocalManagerCreator creates new local managers
type LocalManagerCreator interface {
	CreateLocal(localName string) (*types.LocalGoroutineManager, error)
}

type WaitGroupCreator interface {
	NewWaitGroup(ctx context.Context) (*sync.WaitGroup, error)
}

type FunctionWaitGroupCreator interface {
	NewFunctionWaitGroup(ctx context.Context, functionName string) (*sync.WaitGroup, error)
}

// ----------------------
// Composed interfaces
// ----------------------

// GlobalGoroutineManagerInterface defines the complete interface for global manager
type GlobalGoroutineManagerInterface interface {
	LifeCycle

	AppManagerCreator
	AppManagerLister

	LocalManagerLister

	GoroutineLister

	WaitGroupCreator
}

// AppGoroutineManagerInterface defines the complete interface for app manager
type AppGoroutineManagerInterface interface {
	LifeCycle

	LocalManagerCreator
	LocalManagerLister

	GoroutineLister
	WaitGroupCreator
}

// LocalGoroutineManagerInterface defines the complete interface for local manager
type LocalGoroutineManagerInterface interface {
	LifeCycle
	FunctionShutdowner

	GoroutineSpawner
	GoroutineLister

	WaitGroupCreator
	FunctionWaitGroupCreator
}
