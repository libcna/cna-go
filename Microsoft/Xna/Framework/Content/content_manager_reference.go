package content

// ContentManagerReference is the exported interface every
// ContentManager-typed position takes.
//
// # Why the position widens
//
// `ContentManager::Load<T>` and `ReadAsset<T>` are generic METHODS, which Go
// cannot declare as methods, so the settled rule projects them as package
// functions whose first argument is the receiver. That argument is a
// ContentManager-typed position, and ResourceContentManager is a projected
// derived type -- which is exactly what makes the substitutable-base
// requirement LIVE, the same way Texture2D's and Effect's went live.
//
// So a consumer holding a ResourceContentManager can call
// ContentManagerLoad with it, which in C# is the ordinary thing to do and in Go
// needs this interface to be possible at all.
//
// # It carries an unexported method
//
// `contentManager()` keeps the interface satisfiable only inside this module,
// which is the settled shape: a consumer can NAME the interface and pass values
// to it, and cannot invent a third implementation. It also hands back the base
// half, which is what the package functions actually operate on.
type ContentManagerReference interface {
	// ServiceProvider is ContentManager::get_ServiceProvider.
	ServiceProvider() (any, error)
	// RootDirectory is ContentManager::get_RootDirectory.
	RootDirectory() (string, error)
	// SetRootDirectory is ContentManager::set_RootDirectory.
	SetRootDirectory(value string) error
	// Unload is ContentManager::Unload.
	Unload() error
	// contentManager answers the base half. It is unexported, so only this
	// module can satisfy the interface.
	contentManager() *ContentManager
}

// contentManager satisfies ContentManagerReference for the base itself: a
// ContentManager nothing composes IS its own base half.
func (m *ContentManager) contentManager() *ContentManager { return m }
