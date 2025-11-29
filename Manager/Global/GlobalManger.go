package Global

import (
	AppHelper "github.com/neerajchowdary889/GoRoutinesManager/Helper/App"
	LocalHelper "github.com/neerajchowdary889/GoRoutinesManager/Helper/Local"
	"github.com/neerajchowdary889/GoRoutinesManager/Manager/Interface"
	"github.com/neerajchowdary889/GoRoutinesManager/types"
)

type GlobalManager struct{}

func NewGlobalManager() Interface.GlobalGoroutineManagerInterface {
	return &GlobalManager{}
}

func (GM *GlobalManager) Init() error {
	if types.IsIntilized().Global() {
		return nil
	}

	Global := types.NewGlobalManager().SetGlobalMutex().SetGlobalWaitGroup().SetGlobalContext()
	types.SetGlobalManager(Global)

	return nil
}

func (GM *GlobalManager) Shutdown(safe bool) error {
	globalMgr, err := types.GetGlobalManager()
	if err != nil {
		return err
	}

	// Get all app managers
	appManagers, err := GM.GetAllAppManagers()
	if err != nil {
		return err
	}

	if safe {
		// Safe shutdown: wait for all app managers to complete gracefully
		// Use the GlobalManager's wait group
		if globalMgr.Wg != nil {
			// Add all app managers to the wait group
			for _, appMgr := range appManagers {
				globalMgr.Wg.Add(1)
				go func(am *types.AppManager) {
					defer globalMgr.Wg.Done()
					// Safe shutdown: wait for app manager's wait group
					if am.Wg != nil {
						am.Wg.Wait()
					}
				}(appMgr)
			}
			// Wait for all app managers to shutdown
			globalMgr.Wg.Wait()
		}
	} else {
		// Unsafe shutdown: cancel all app manager contexts forcefully
		for _, appMgr := range appManagers {
			// Cancel all local manager contexts
			for _, localMgr := range appMgr.GetLocalManagers() {
				// Cancel all routine contexts
				for _, routine := range localMgr.GetRoutines() {
					cancel := routine.GetCancel()
					if cancel != nil {
						cancel()
					}
				}

				// Cancel local manager context
				if localMgr.Cancel != nil {
					localMgr.Cancel()
				}
			}

			// Cancel the app manager's context
			if appMgr.Cancel != nil {
				appMgr.Cancel()
			}
		}

		// Cancel the global manager's context
		if globalMgr.Cancel != nil {
			globalMgr.Cancel()
		}
	}

	return nil
}

func (GM *GlobalManager) GetAllAppManagers() ([]*types.AppManager, error) {
	Global, error := types.GetGlobalManager()
	if error != nil {
		return nil, error
	}

	mapValue := Global.GetAppManagers()
	helper := AppHelper.NewAppHelper()
	return helper.MapToSlice(mapValue), nil
}

func (GM *GlobalManager) GetAppManagerCount() int {
	Global, error := types.GetGlobalManager()
	if error != nil {
		return 0
	}
	return Global.GetAppManagerCount()
}

func (GM *GlobalManager) GetAllLocalManagers() ([]*types.LocalManager, error) {
	// Get all app managers first
	appManagers, error := GM.GetAllAppManagers()
	if error != nil {
		return nil, error
	}

	// Get all local managers from each app manager
	var localManagers []*types.LocalManager
	for _, appManager := range appManagers {
		// Convert map to slice
		LocalManagerSlice := LocalHelper.NewLocalHelper().MapToSlice(appManager.LocalManagers)
		localManagers = append(localManagers, LocalManagerSlice...)
	}

	return localManagers, nil
}

func (GM *GlobalManager) GetLocalManagerCount() int {
	// get all the local managers first
	// Dont use GetAllLocalManagers() as it will create a new slice - memory usage would be O(n)
	// and it will be a performance issue
	App, error := GM.GetAllAppManagers()
	if error != nil {
		return 0
	}
	i := 0
	for _, Apps := range App {
		i += Apps.GetLocalManagerCount()
	}
	return i
}

func (GM *GlobalManager) GetAllGoroutines() ([]*types.Routine, error) {
	// Get all app managers first
	appManagers, error := GM.GetAllAppManagers()
	if error != nil {
		return nil, error
	}

	// Get all goroutines from each app manager - would run on O(n*m)
	var goroutines []*types.Routine
	for _, appManager := range appManagers {
		LocalManagers := appManager.GetLocalManagers()
		for _, localManager := range LocalManagers {
			Routines := localManager.GetRoutines()
			goroutines = append(goroutines, LocalHelper.NewLocalHelper().RoutinesMapToSlice(Routines)...)
		}
	}
	return goroutines, nil
}

func (GM *GlobalManager) GetGoroutineCount() int {
	// get all the goroutines first
	// Dont use GetAllGoroutines() as it will create a new slice - memory usage would be O(n)
	// and it will be a performance issue
	App, error := GM.GetAllAppManagers()
	if error != nil {
		return 0
	}
	i := 0
	for _, Apps := range App {
		LocalManagers := Apps.GetLocalManagers()
		for _, LocalManager := range LocalManagers {
			i += LocalManager.GetRoutineCount()
		}
	}
	return i
}
