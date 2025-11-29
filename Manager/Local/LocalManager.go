package Local

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/neerajchowdary889/GoRoutinesManager/Manager/Interface"
	"github.com/neerajchowdary889/GoRoutinesManager/types"
	"github.com/neerajchowdary889/GoRoutinesManager/types/Errors"
)

type LocalManagerStruct struct {
	AppName   string
	LocalName string
}

func NewLocalManager(appName, localName string) Interface.LocalGoroutineManagerInterface {
	return &LocalManagerStruct{
		AppName:   appName,
		LocalName: localName,
	}
}

func (LM *LocalManagerStruct) CreateLocal(localName string) (*types.LocalManager, error) {
	// First get the app manager
	appManager, err := types.GetAppManager(LM.AppName)
	if err != nil {
		return nil, err
	}

	// Directly call the CreateLocal method of the app manager
	// CreateLocal function will handle the checking and creation of the local manager
	localManager, err := appManager.CreateLocal(localName)
	switch err {
	case Errors.ErrLocalManagerNotFound:
		return nil, fmt.Errorf("%w: %s", Errors.ErrLocalManagerNotFound, localName)
	case Errors.WrngLocalManagerAlreadyExists:
		// Return the existing local manager and also return error as nil
		return localManager, nil
	default:
		// Fill the structs
		localManager.SetLocalContext().
			SetLocalMutex().
			SetLocalWaitGroup()
	}
	return localManager, nil
}

// Shutdowner
func (LM *LocalManagerStruct) Shutdown(safe bool) error {
	//TODO
	return nil
}

// FunctionShutdowner
func (LM *LocalManagerStruct) ShutdownFunction(functionName string, timeout time.Duration) error {
	localManager, err := types.GetLocalManager(LM.AppName, LM.LocalName)
	if err != nil {
		return err
	}

	// Get all routines
	routines, err := LM.GetAllGoroutines()
	if err != nil {
		return err
	}

	// Cancel all routines with this function name
	for _, routine := range routines {
		if routine.GetFunctionName() == functionName {
			cancel := routine.GetCancel()
			if cancel != nil {
				cancel()
			}
		}
	}

	// Wait for completion with timeout
	completed := LM.WaitForFunctionWithTimeout(functionName, timeout)
	if !completed {
		return fmt.Errorf("shutdown timeout for function: %s", functionName)
	}

	// Clean up the wait group
	localManager.RemoveFunctionWg(functionName)

	return nil
}

// GoroutineSpawner
// Go spawns a new goroutine, tracks it in the LocalManager, and returns the routine ID.
// The goroutine is spawned with a context derived from the LocalManager's parent context.
// The done channel is closed when the goroutine completes.
func (LM *LocalManagerStruct) Go(functionName string, workerFunc func(ctx context.Context) error) error {
	return LM.spawnGoroutine(functionName, workerFunc, false)
}

// GoWithWaitGroup spawns a goroutine and automatically manages the function wait group.
// This is a convenience method that calls wg.Add(1) before spawning and wg.Done() after completion.
// Use this when you want automatic wait group management.
// For manual control, use Go() and manage the wait group yourself via NewFunctionWaitGroup().
func (LM *LocalManagerStruct) GoWithWaitGroup(functionName string, workerFunc func(ctx context.Context) error) error {
	return LM.spawnGoroutine(functionName, workerFunc, true)
}

// spawnGoroutine is the internal implementation for spawning goroutines.
// If useWaitGroup is true, it automatically manages Add/Done for the function wait group.
func (LM *LocalManagerStruct) spawnGoroutine(functionName string, workerFunc func(ctx context.Context) error, useWaitGroup bool) error {
	// Get the types.LocalManager instance
	localManager, err := types.GetLocalManager(LM.AppName, LM.LocalName)
	if err != nil {
		return err
	}

	var wg *sync.WaitGroup
	if useWaitGroup {
		// Get or create function wait group
		wg, err = LM.NewFunctionWaitGroup(context.Background(), functionName)
		if err != nil {
			return err
		}
		// Increment wait group BEFORE spawning goroutine
		wg.Add(1)
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
			if useWaitGroup && wg != nil {
				// Decrement wait group when routine completes
				wg.Done()
			}
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
func (LM *LocalManagerStruct) GetAllGoroutines() ([]*types.Routine, error) {
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

func (LM *LocalManagerStruct) GetGoroutineCount() int {
	localManager, err := types.GetLocalManager(LM.AppName, LM.LocalName)
	if err != nil {
		return 0
	}
	return localManager.GetRoutineCount()
}

// FunctionWaitGroupCreator
func (LM *LocalManagerStruct) NewFunctionWaitGroup(ctx context.Context, functionName string) (*sync.WaitGroup, error) {
	localManager, err := types.GetLocalManager(LM.AppName, LM.LocalName)
	if err != nil {
		return nil, err
	}

	// Check if wait group already exists
	wg, err := localManager.GetFunctionWg(functionName)
	if err == nil {
		// Already exists, return it
		return wg, nil
	}

	// Create new wait group
	localManager.AddFunctionWg(functionName)

	// Retrieve and return the newly created wait group
	return localManager.GetFunctionWg(functionName)
}

// Routine management methods - these operate on individual routines by ID

// CancelRoutine cancels a routine's context by its ID.
// Returns an error if the routine is not found.
func (LM *LocalManagerStruct) CancelRoutine(routineID string) error {
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
func (LM *LocalManagerStruct) WaitForRoutine(routineID string, timeout time.Duration) bool {
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
func (LM *LocalManagerStruct) IsRoutineDone(routineID string) bool {
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
func (LM *LocalManagerStruct) GetRoutineContext(routineID string) context.Context {
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
func (LM *LocalManagerStruct) GetRoutineStartedAt(routineID string) int64 {
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
func (LM *LocalManagerStruct) GetRoutineUptime(routineID string) time.Duration {
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
func (LM *LocalManagerStruct) IsRoutineContextCancelled(routineID string) bool {
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
func (LM *LocalManagerStruct) GetRoutine(routineID string) (*types.Routine, error) {
	localManager, err := types.GetLocalManager(LM.AppName, LM.LocalName)
	if err != nil {
		return nil, err
	}
	return localManager.GetRoutine(routineID)
}

// WaitForFunction waits for all goroutines of a specific function to complete.
func (LM *LocalManagerStruct) WaitForFunction(functionName string) error {
	localManager, err := types.GetLocalManager(LM.AppName, LM.LocalName)
	if err != nil {
		return err
	}

	wg, err := localManager.GetFunctionWg(functionName)
	if err != nil {
		return err // No wait group for this function
	}

	wg.Wait()
	return nil
}

// WaitForFunctionWithTimeout waits for all goroutines of a function with a timeout.
// Returns true if all completed, false if timeout occurred.
func (LM *LocalManagerStruct) WaitForFunctionWithTimeout(functionName string, timeout time.Duration) bool {
	done := make(chan struct{})

	go func() {
		LM.WaitForFunction(functionName)
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// GetFunctionGoroutineCount returns the number of goroutines for a specific function.
func (LM *LocalManagerStruct) GetFunctionGoroutineCount(functionName string) int {
	routines, err := LM.GetAllGoroutines()
	if err != nil {
		return 0
	}

	count := 0
	for _, routine := range routines {
		if routine.GetFunctionName() == functionName {
			count++
		}
	}
	return count
}
