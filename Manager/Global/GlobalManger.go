package Global

import (
	"sync"

	"github.com/neerajchowdary889/GoRoutinesManager/Context"

	"github.com/neerajchowdary889/GoRoutinesManager/Manager/Interface"
	"github.com/neerajchowdary889/GoRoutinesManager/types"
)

type GlobalManager struct {}

func NewGlobalManager() Interface.GlobalGoroutineManagerInterface {
	return &GlobalManager{}
}

func (g *GlobalManager) Init() error {
	if types.IsIntilized().Global() {
		return nil
	}

	ctx := Context.GetGlobalContext().Get()
	Done := func() {
		Context.GetGlobalContext().Done(ctx)
	}

	wg, err := g.NewWaitGroup()
	if err != nil {
		return err
	}

	// Create global mutex
	globalMutex := &sync.RWMutex{}

	types.SetGlobalManager(&types.GlobalManager{
		GlobalMu: globalMutex,
		AppManagers: make(map[string]*types.AppManager),
		Ctx: ctx,
		Cancel: Done,
		Wg: wg,
	})

	return nil
}

func (g *GlobalManager) Shutdown(safe bool) error {
	return nil
}

func (g *GlobalManager) NewWaitGroup() (*sync.WaitGroup, error) {
	wg := &sync.WaitGroup{}
	return wg, nil
}


func (g *GlobalManager) GetAllAppManagers() ([]*types.AppManager, error) {
	return nil, nil
}

func (g *GlobalManager) GetAppManagerCount() int {
	return 0
}

func (g *GlobalManager) GetAllLocalManagers() ([]*types.LocalManager, error) {
	return nil, nil
}

func (g *GlobalManager) GetLocalManagerCount() int {
	return 0
}

func (g *GlobalManager) GetAllGoroutines() ([]*types.Routine, error) {
	return nil, nil
}

func (g *GlobalManager) GetGoroutineCount() int {
	return 0
}
