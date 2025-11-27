package App

import(
	"sync"

	"github.com/neerajchowdary889/GoRoutinesManager/Context"

	"github.com/neerajchowdary889/GoRoutinesManager/Manager/Interface"
	"github.com/neerajchowdary889/GoRoutinesManager/types"
	"github.com/neerajchowdary889/GoRoutinesManager/Manager/Global"

)

type AppManager struct {}

func NewAppManager() Interface.AppGoroutineManagerInterface {
	return &AppManager{}
}

func (a *AppManager) CreateApp(appName string) (*types.AppManager, error) {

    // First check if the app manager is already initialized
	if !types.IsIntilized().App(appName) {
		// If Global Manager is Not Intilized, then we need to initialize it
		globalManager := Global.NewGlobalManager()
		err := globalManager.Init()
		if err != nil {
			return nil, err
		}
	}

	if types.IsIntilized().App(appName) {
		return types.GetAppManager(appName)
	}

	ctx := Context.GetAppContext(appName).Get()
	Done := func() {
		Context.GetAppContext(appName).Done(ctx)
	}
	
	wg, err := a.NewWaitGroup()
	if err != nil {
		return nil, err
	}
	AppMu := &sync.RWMutex{}

	app := &types.AppManager{
		AppName: appName,
		AppMu: AppMu,
		LocalManagers: make(map[string]*types.LocalManager),
		Ctx: ctx,
		Cancel: Done,
		Wg: wg,
	}
	types.SetAppManager(appName, app)
	return app, nil
}

func (a *AppManager) Shutdown(safe bool) error {
	return nil
}

func (a *AppManager) NewWaitGroup() (*sync.WaitGroup, error) {
	wg := &sync.WaitGroup{}
	return wg, nil
}

func (a *AppManager) CreateLocal(localName string) (*types.LocalManager, error) {
	return nil, nil
}

func (a *AppManager) GetAllLocalManagers() ([]*types.LocalManager, error) {
	return nil, nil
}