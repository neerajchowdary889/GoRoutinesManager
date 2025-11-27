package types

import (
	"context"
	"sync"
)

func NewAppManager(appName string) *AppManager {
	if IsIntilized().App(appName) {
		value, err := Global.GetAppManager(appName)
		if err != nil {
			return nil
		}
		return value
	}
	return &AppManager{
		AppName:       appName,
		LocalManagers: make(map[string]*LocalManager),
	}
}

// Lock APIs

// LockAppReadMutex locks the app read mutex for the app manager - This is used to read the app manager's data
func (AM *AppManager) LockAppReadMutex() {
	if AM.AppMu == nil {
		AM.SetAppMutex(&sync.RWMutex{})
	}
	AM.AppMu.RLock()
}

// UnlockAppReadMutex unlocks the app read mutex for the app manager - This is used to read the app manager's data
func (AM *AppManager) UnlockAppReadMutex() {
	if AM.AppMu == nil {
		AM.SetAppMutex(&sync.RWMutex{})
	}
	AM.AppMu.RUnlock()
}

// Set APIs

// SetAppName sets the name of the app for the app manager
func (AM *AppManager) SetAppName(appName string) AppManager {
	AM.AppName = appName
	return *AM
}

func (AM *AppManager) SetAppMutex(mu *sync.RWMutex) AppManager {
	AM.AppMu = mu
	return *AM
}

func (AM *AppManager) SetAppContext(ctx context.Context, cancel context.CancelFunc) AppManager {
	AM.Ctx = ctx
	AM.Cancel = cancel
	return *AM
}

func (AM *AppManager) SetAppWaitGroup(wg *sync.WaitGroup) AppManager {
	AM.Wg = wg
	return *AM
}

// AddLocalManager adds a new local manager to the app manager
func (AM *AppManager) AddLocalManager(localName string, local *LocalManager) AppManager {
	if IsIntilized().Local(AM.AppName, localName) {
		return *AM
	}
	AM.LockAppWriteMutex()
	defer AM.UnlockAppWriteMutex()
	AM.LocalManagers[localName] = local
	return *AM
}
