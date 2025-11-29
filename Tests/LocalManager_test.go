package Tests

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/neerajchowdary889/GoRoutinesManager/Manager/App"
	"github.com/neerajchowdary889/GoRoutinesManager/Manager/Local"
)

func TestLocalManager_CreateLocal(t *testing.T) {
	resetGlobalState()

	// Setup: create app first
	appMgr := App.NewAppManager("test-app")
	_, err := appMgr.CreateApp()
	if err != nil {
		t.Fatalf("CreateApp() failed: %v", err)
	}

	// Create local manager
	localMgr := Local.NewLocalManager("test-app", "test-local")
	local, err := localMgr.CreateLocal("test-local")
	if err != nil {
		t.Fatalf("CreateLocal() failed: %v", err)
	}

	if local == nil {
		t.Fatal("CreateLocal() returned nil")
	}
}

func TestLocalManager_CreateLocal_Idempotent(t *testing.T) {
	resetGlobalState()

	appMgr := App.NewAppManager("test-app")
	_, err := appMgr.CreateApp()
	if err != nil {
		t.Fatalf("CreateApp() failed: %v", err)
	}

	localMgr := Local.NewLocalManager("test-app", "test-local")

	// Create first time
	local1, err := localMgr.CreateLocal("test-local")
	if err != nil {
		t.Fatalf("First CreateLocal() failed: %v", err)
	}

	// Create second time (should return existing)
	local2, err := localMgr.CreateLocal("test-local")
	if err != nil {
		t.Fatalf("Second CreateLocal() failed: %v", err)
	}

	// Should be the same instance
	if local1 != local2 {
		t.Error("CreateLocal() should return same instance on subsequent calls")
	}
}

func TestLocalManager_Go_SpawnGoroutine(t *testing.T) {
	resetGlobalState()

	// Setup
	appMgr := App.NewAppManager("test-app")
	_, err := appMgr.CreateApp()
	if err != nil {
		t.Fatalf("CreateApp() failed: %v", err)
	}

	localMgr := Local.NewLocalManager("test-app", "test-local")
	_, err = localMgr.CreateLocal("test-local")
	if err != nil {
		t.Fatalf("CreateLocal() failed: %v", err)
	}

	// Spawn a goroutine
	executed := atomic.Bool{}
	err = localMgr.Go("testFunc", func(ctx context.Context) error {
		executed.Store(true)
		return nil
	})

	if err != nil {
		t.Fatalf("Go() failed: %v", err)
	}

	// Wait a bit for goroutine to execute
	time.Sleep(100 * time.Millisecond)

	if !executed.Load() {
		t.Error("Goroutine was not executed")
	}

	// Verify routine was tracked
	count := localMgr.GetGoroutineCount()
	if count != 1 {
		t.Errorf("Expected 1 tracked goroutine, got %d", count)
	}
}

func TestLocalManager_Go_MultipleGoroutines(t *testing.T) {
	resetGlobalState()

	appMgr := App.NewAppManager("test-app")
	_, err := appMgr.CreateApp()
	if err != nil {
		t.Fatalf("CreateApp() failed: %v", err)
	}

	localMgr := Local.NewLocalManager("test-app", "test-local")
	_, err = localMgr.CreateLocal("test-local")
	if err != nil {
		t.Fatalf("CreateLocal() failed: %v", err)
	}

	// Spawn multiple goroutines
	for i := 0; i < 5; i++ {
		err = localMgr.Go("testFunc", func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		})
		if err != nil {
			t.Fatalf("Go() failed: %v", err)
		}
	}

	// Verify all were tracked
	count := localMgr.GetGoroutineCount()
	if count != 5 {
		t.Errorf("Expected 5 tracked goroutines, got %d", count)
	}
}

func TestLocalManager_GetAllGoroutines(t *testing.T) {
	resetGlobalState()

	appMgr := App.NewAppManager("test-app")
	_, err := appMgr.CreateApp()
	if err != nil {
		t.Fatalf("CreateApp() failed: %v", err)
	}

	localMgr := Local.NewLocalManager("test-app", "test-local")
	_, err = localMgr.CreateLocal("test-local")
	if err != nil {
		t.Fatalf("CreateLocal() failed: %v", err)
	}

	// Initially empty
	routines, err := localMgr.GetAllGoroutines()
	if err != nil {
		t.Fatalf("GetAllGoroutines() failed: %v", err)
	}
	if len(routines) != 0 {
		t.Errorf("Expected 0 goroutines, got %d", len(routines))
	}

	// Spawn some goroutines
	for i := 0; i < 3; i++ {
		localMgr.Go("testFunc", func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})
	}

	// Get all
	routines, err = localMgr.GetAllGoroutines()
	if err != nil {
		t.Fatalf("GetAllGoroutines() failed: %v", err)
	}
	if len(routines) != 3 {
		t.Errorf("Expected 3 goroutines, got %d", len(routines))
	}

	// Verify each routine has required fields
	for _, routine := range routines {
		if routine.GetID() == "" {
			t.Error("Routine should have an ID")
		}
		if routine.GetFunctionName() == "" {
			t.Error("Routine should have a function name")
		}
		if routine.GetStartedAt() == 0 {
			t.Error("Routine should have a start timestamp")
		}
	}
}

func TestLocalManager_GetGoroutineCount(t *testing.T) {
	resetGlobalState()

	appMgr := App.NewAppManager("test-app")
	_, err := appMgr.CreateApp()
	if err != nil {
		t.Fatalf("CreateApp() failed: %v", err)
	}

	localMgr := Local.NewLocalManager("test-app", "test-local")
	_, err = localMgr.CreateLocal("test-local")
	if err != nil {
		t.Fatalf("CreateLocal() failed: %v", err)
	}

	// Initially 0
	count := localMgr.GetGoroutineCount()
	if count != 0 {
		t.Errorf("Expected 0 goroutines, got %d", count)
	}

	// Add some
	localMgr.Go("func1", func(ctx context.Context) error { return nil })
	localMgr.Go("func2", func(ctx context.Context) error { return nil })

	count = localMgr.GetGoroutineCount()
	if count != 2 {
		t.Errorf("Expected 2 goroutines, got %d", count)
	}
}

func TestLocalManager_ContextPropagation(t *testing.T) {
	resetGlobalState()

	appMgr := App.NewAppManager("test-app")
	_, err := appMgr.CreateApp()
	if err != nil {
		t.Fatalf("CreateApp() failed: %v", err)
	}

	localMgr := Local.NewLocalManager("test-app", "test-local")
	_, err = localMgr.CreateLocal("test-local")
	if err != nil {
		t.Fatalf("CreateLocal() failed: %v", err)
	}

	// Spawn a goroutine that checks context
	contextReceived := atomic.Bool{}
	err = localMgr.Go("testFunc", func(ctx context.Context) error {
		if ctx != nil {
			contextReceived.Store(true)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Go() failed: %v", err)
	}

	// Wait for execution
	time.Sleep(100 * time.Millisecond)

	if !contextReceived.Load() {
		t.Error("Goroutine should receive a valid context")
	}
}

func TestLocalManager_GoroutineCompletion(t *testing.T) {
	resetGlobalState()

	appMgr := App.NewAppManager("test-app")
	_, err := appMgr.CreateApp()
	if err != nil {
		t.Fatalf("CreateApp() failed: %v", err)
	}

	localMgr := Local.NewLocalManager("test-app", "test-local")
	_, err = localMgr.CreateLocal("test-local")
	if err != nil {
		t.Fatalf("CreateLocal() failed: %v", err)
	}

	// Spawn a short-lived goroutine
	completed := atomic.Bool{}
	err = localMgr.Go("testFunc", func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		completed.Store(true)
		return nil
	})
	if err != nil {
		t.Fatalf("Go() failed: %v", err)
	}

	// Wait for completion
	time.Sleep(150 * time.Millisecond)

	if !completed.Load() {
		t.Error("Goroutine should have completed")
	}
}

func TestLocalManager_MultipleLocalManagers(t *testing.T) {
	resetGlobalState()

	appMgr := App.NewAppManager("test-app")
	_, err := appMgr.CreateApp()
	if err != nil {
		t.Fatalf("CreateApp() failed: %v", err)
	}

	// Create multiple local managers
	local1Mgr := Local.NewLocalManager("test-app", "local1")
	local2Mgr := Local.NewLocalManager("test-app", "local2")

	_, err = local1Mgr.CreateLocal("local1")
	if err != nil {
		t.Fatalf("CreateLocal(local1) failed: %v", err)
	}

	_, err = local2Mgr.CreateLocal("local2")
	if err != nil {
		t.Fatalf("CreateLocal(local2) failed: %v", err)
	}

	// Spawn goroutines in each
	local1Mgr.Go("func1", func(ctx context.Context) error { return nil })
	local1Mgr.Go("func2", func(ctx context.Context) error { return nil })
	local2Mgr.Go("func3", func(ctx context.Context) error { return nil })

	// Verify counts are independent
	count1 := local1Mgr.GetGoroutineCount()
	if count1 != 2 {
		t.Errorf("Local1 should have 2 goroutines, got %d", count1)
	}

	count2 := local2Mgr.GetGoroutineCount()
	if count2 != 1 {
		t.Errorf("Local2 should have 1 goroutine, got %d", count2)
	}
}

func TestLocalManager_ErrorHandling(t *testing.T) {
	resetGlobalState()

	appMgr := App.NewAppManager("test-app")
	_, err := appMgr.CreateApp()
	if err != nil {
		t.Fatalf("CreateApp() failed: %v", err)
	}

	localMgr := Local.NewLocalManager("test-app", "test-local")
	_, err = localMgr.CreateLocal("test-local")
	if err != nil {
		t.Fatalf("CreateLocal() failed: %v", err)
	}

	// Spawn a goroutine that returns an error
	errorReturned := atomic.Bool{}
	err = localMgr.Go("testFunc", func(ctx context.Context) error {
		errorReturned.Store(true)
		return context.Canceled
	})
	if err != nil {
		t.Fatalf("Go() failed: %v", err)
	}

	// Wait for execution
	time.Sleep(100 * time.Millisecond)

	if !errorReturned.Load() {
		t.Error("Goroutine should have executed and returned error")
	}
}
