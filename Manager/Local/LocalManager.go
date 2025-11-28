package Local

import (
	"context"
	"sync"
	"time"

	"github.com/neerajchowdary889/GoRoutinesManager/Manager/Interface"
	"github.com/neerajchowdary889/GoRoutinesManager/types"
	"github.com/neerajchowdary889/GoRoutinesManager/types/Errors"
)

type LocalManager struct {
	AppName   string
	LocalName string
}

func NewLocalManager(appName, localName string) Interface.LocalGoroutineManagerInterface {
	return &LocalManager{
		AppName:   appName,
		LocalName: localName,
	}
}

func (LM *LocalManager) CreateLocal(localName string) (*types.LocalManager, error) {
	// First get the app manager
	appManager, err := types.GetAppManager(LM.AppName)
	if err != nil {
		return nil, err
	}
	if !types.IsIntilized().Local(appManager.GetAppName(), LM.LocalName) {
		localManager := types.NewLocalManager(LM.LocalName, appManager.GetAppName()).SetLocalContext().SetLocalMutex().SetLocalWaitGroup()
		if localManager == nil {
			return nil, Errors.ErrLocalManagerNotFound
		}
		appManager.AddLocalManager(LM.LocalName, localManager)
		return localManager, nil
	}
	return appManager.GetLocalManager(LM.LocalName)
}

// Shutdowner
func (LM *LocalManager) Shutdown(safe bool) error {
	//TODO
	return nil
}

// FunctionShutdowner
func (LM *LocalManager) ShutdownFunction(functionName string, timeout time.Duration) error {
	//TODO
	return nil
}

// GoroutineSpawner
// Go spawns a new goroutine, tracks it in the LocalManager, and returns the routine ID.
// The goroutine is spawned with a context derived from the LocalManager's parent context.
// The done channel is closed when the goroutine completes.
func (LM *LocalManager) Go(functionName string, workerFunc func(ctx context.Context) error) error {
	// Get the types.LocalManager instance
	localManager, err := types.GetLocalManager(LM.AppName, LM.LocalName)
	if err != nil {
		return err
	}
	
	// Create a child context with cancel for this routine
	routineCtx, cancel := localManager.SpawnChild()

	// Create the done channel (bidirectional, buffered size 1)
	// This allows non-blocking close even if nothing is reading
	doneChan := make(chan struct{}, 1)

	// Create a new Routine instance
	routine := types.NewGoRoutine(functionName).
		SetContext(routineCtx).
		SetCancel(cancel).
		SetDone(doneChan) // Override the channel created in NewGoRoutine

	// Add routine to LocalManager for tracking
	localManager.AddRoutine(routine)

	// Spawn the goroutine
	go func() {
		defer func() {
			// Close the done channel when routine completes
			// The done channel is buffered (size 1) so this won't block
			close(doneChan)
		}()

		// Execute the worker function with the routine's context
		_ = workerFunc(routineCtx)
	}()

	return nil
}

// GoroutineLister
func (LM *LocalManager) GetAllGoroutines() ([]*types.Routine, error) {
	localManager, err := types.GetLocalManager(LM.AppName, LM.LocalName)
	if err != nil {
		return nil, err
	}

	routines := localManager.GetRoutines()
	// Convert map to slice
	result := make([]*types.Routine, 0, len(routines))
	for _, routine := range routines {
		result = append(result, routine)
	}
	return result, nil
}

func (LM *LocalManager) GetGoroutineCount() int {
	localManager, err := types.GetLocalManager(LM.AppName, LM.LocalName)
	if err != nil {
		return 0
	}
	return localManager.GetRoutineCount()
}

// FunctionWaitGroupCreator
func (LM *LocalManager) NewFunctionWaitGroup(ctx context.Context, functionName string) (*sync.WaitGroup, error) {
	//TODO
	return nil, nil
}

// Routine management methods - these operate on individual routines by ID

// CancelRoutine cancels a routine's context by its ID.
// Returns an error if the routine is not found.
func (LM *LocalManager) CancelRoutine(routineID string) error {
	localManager, err := types.GetLocalManager(LM.AppName, LM.LocalName)
	if err != nil {
		return err
	}

	routine, err := localManager.GetRoutine(routineID)
	if err != nil {
		return err
	}

	cancel := routine.GetCancel()
	if cancel != nil {
		cancel()
	}
	return nil
}

// WaitForRoutine blocks until the routine's done channel is signaled or the timeout expires.
// Returns true if the routine completed, false if timeout occurred or routine not found.
func (LM *LocalManager) WaitForRoutine(routineID string, timeout time.Duration) bool {
	localManager, err := types.GetLocalManager(LM.AppName, LM.LocalName)
	if err != nil {
		return false
	}

	routine, err := localManager.GetRoutine(routineID)
	if err != nil {
		return false
	}

	doneChan := routine.DoneChan()
	if doneChan == nil {
		return false
	}

	select {
	case <-doneChan:
		return true
	case <-time.After(timeout):
		return false
	}
}

// IsRoutineDone checks if a routine's done channel has been signaled.
// Returns false if routine is not found or done channel is nil.
func (LM *LocalManager) IsRoutineDone(routineID string) bool {
	localManager, err := types.GetLocalManager(LM.AppName, LM.LocalName)
	if err != nil {
		return false
	}

	routine, err := localManager.GetRoutine(routineID)
	if err != nil {
		return false
	}

	doneChan := routine.DoneChan()
	if doneChan == nil {
		return false
	}

	select {
	case <-doneChan:
		return true
	default:
		return false
	}
}

// GetRoutineContext returns the context associated with a routine by ID.
// Returns context.Background() if routine is not found or context is nil.
func (LM *LocalManager) GetRoutineContext(routineID string) context.Context {
	localManager, err := types.GetLocalManager(LM.AppName, LM.LocalName)
	if err != nil {
		return context.Background()
	}

	routine, err := localManager.GetRoutine(routineID)
	if err != nil {
		return context.Background()
	}

	ctx := routine.GetContext()
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// GetRoutineStartedAt returns the timestamp when a routine was started.
// Returns 0 if routine is not found.
func (LM *LocalManager) GetRoutineStartedAt(routineID string) int64 {
	localManager, err := types.GetLocalManager(LM.AppName, LM.LocalName)
	if err != nil {
		return 0
	}

	routine, err := localManager.GetRoutine(routineID)
	if err != nil {
		return 0
	}

	return routine.GetStartedAt()
}

// GetRoutineUptime returns the duration a routine has been running.
// Returns 0 if routine is not found or not started.
func (LM *LocalManager) GetRoutineUptime(routineID string) time.Duration {
	localManager, err := types.GetLocalManager(LM.AppName, LM.LocalName)
	if err != nil {
		return 0
	}

	routine, err := localManager.GetRoutine(routineID)
	if err != nil {
		return 0
	}

	startedAt := routine.GetStartedAt()
	if startedAt == 0 {
		return 0
	}

	now := time.Now().UnixNano()
	return time.Duration(now - startedAt)
}

// IsRoutineContextCancelled checks if a routine's context has been cancelled.
// Returns false if routine is not found or context is nil.
func (LM *LocalManager) IsRoutineContextCancelled(routineID string) bool {
	localManager, err := types.GetLocalManager(LM.AppName, LM.LocalName)
	if err != nil {
		return false
	}

	routine, err := localManager.GetRoutine(routineID)
	if err != nil {
		return false
	}

	ctx := routine.GetContext()
	if ctx == nil {
		return false
	}

	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// GetRoutine returns a routine by its ID.
// Returns an error if the routine is not found.
func (LM *LocalManager) GetRoutine(routineID string) (*types.Routine, error) {
	localManager, err := types.GetLocalManager(LM.AppName, LM.LocalName)
	if err != nil {
		return nil, err
	}
	return localManager.GetRoutine(routineID)
}
