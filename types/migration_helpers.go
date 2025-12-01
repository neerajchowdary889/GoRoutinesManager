package types

import (
	"github.com/neerajchowdary889/GoRoutinesManager/types/Errors"
)

// LegacyGlobalManager provides backward compatibility with old singleton API.
// This wraps the default root to match the old GlobalManager interface.
// Use this during migration to maintain backward compatibility.
//
// Deprecated: Use Root directly instead. This will be removed in v2.0.
type LegacyGlobalManager struct {
	root *Root
}

// NewLegacyGlobalManager creates a legacy wrapper around the default root.
// This allows old code to continue working during migration.
//
// Deprecated: Use types.NewRoot() or types.DefaultRoot() instead.
func NewLegacyGlobalManager() *LegacyGlobalManager {
	return &LegacyGlobalManager{
		root: DefaultRoot(),
	}
}

// Init initializes the root (no-op if already initialized).
func (lgm *LegacyGlobalManager) Init() error {
	// Root is already initialized when created
	return nil
}

// Shutdown shuts down the root.
func (lgm *LegacyGlobalManager) Shutdown(safe bool) error {
	return lgm.root.Shutdown(safe)
}

// GetAllAppManagers returns all app managers.
func (lgm *LegacyGlobalManager) GetAllAppManagers() ([]*AppManager, error) {
	return lgm.root.GetAllApps(), nil
}

// GetAppManagerCount returns the number of app managers.
func (lgm *LegacyGlobalManager) GetAppManagerCount() int {
	return lgm.root.GetAppCount()
}

// GetAppManager returns a specific app manager.
func (lgm *LegacyGlobalManager) GetAppManager(appName string) (*AppManager, error) {
	return lgm.root.GetApp(appName)
}

// GetMetadata returns the root metadata.
func (lgm *LegacyGlobalManager) GetMetadata() (*Metadata, error) {
	return lgm.root.Metadata(), nil
}

// UpdateMetadata updates the root metadata.
func (lgm *LegacyGlobalManager) UpdateMetadata(flag string, value interface{}) (*Metadata, error) {
	return lgm.root.UpdateMetadata(flag, value)
}

// GetGlobalManager returns the legacy global manager (wraps default root).
// This maintains backward compatibility with code that uses types.GetGlobalManager().
//
// Deprecated: Use types.DefaultRoot() or types.NewRoot() instead.
func GetGlobalManager() (*LegacyGlobalManager, error) {
	return NewLegacyGlobalManager(), nil
}

// SetGlobalManager is a no-op in the new API (for backward compatibility only).
// The root is managed internally.
//
// Deprecated: Use types.NewRoot() or types.DefaultRoot() instead.
func SetGlobalManager(_ *GlobalManager) {
	// No-op: Root is managed internally
}

// GetAppManager returns an app manager from the default root.
// This maintains backward compatibility.
//
// Deprecated: Use root.App(name) or types.DefaultRoot().App(name) instead.
func GetAppManager(appName string) (*AppManager, error) {
	root := DefaultRoot()
	return root.GetApp(appName)
}

// SetAppManager adds an app manager to the default root.
// This maintains backward compatibility.
//
// Deprecated: Use root.App(name) instead.
func SetAppManager(appName string, app *AppManager) {
	root := DefaultRoot()
	root.mu.Lock()
	defer root.mu.Unlock()
	root.apps[appName] = app
}

// GetLocalManager returns a local manager from the default root.
// This maintains backward compatibility.
//
// Deprecated: Use root.App(appName).Local(localName) instead.
func GetLocalManager(appName, localName string) (*LocalManager, error) {
	root := DefaultRoot()
	app, err := root.GetApp(appName)
	if err != nil {
		return nil, err
	}
	return app.GetLocalManager(localName)
}

// SetLocalManager adds a local manager to the default root.
// This maintains backward compatibility.
//
// Deprecated: Use root.App(appName).Local(localName) instead.
func SetLocalManager(appName, localName string, local *LocalManager) {
	root := DefaultRoot()
	app := root.App(appName)
	app.AddLocalManager(localName, local)
}

