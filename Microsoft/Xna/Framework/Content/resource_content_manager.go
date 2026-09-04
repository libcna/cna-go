package content

import (
	"bytes"
	"fmt"
	"io"
)

// ResourceContentManager is
// Microsoft.Xna.Framework.Content.ResourceContentManager:
//
//	.class public auto ansi beforefieldinit ResourceContentManager
//	       extends Microsoft.Xna.Framework.Content.ContentManager
//
// A content manager that reads its assets out of an assembly's embedded
// resources rather than off disk.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # System.Resources.ResourceManager, measured rather than deferred
//
// This type was blocked for eight milestones on "System.Resources.ResourceManager
// is a BCL subsystem outside the seven pinned assemblies and has no CNA
// counterpart". The subsystem is real; the block was not, and measuring it says
// why.
//
// The pinned contract names ResourceManager at exactly ONE signature position,
// this constructor, and ResourceContentManager uses exactly ONE member of it:
//
//	callvirt instance object System.Resources.ResourceManager::GetObject(string)
//
// That is a name-to-object lookup and nothing else. The settled rule for a BCL
// type at a signature position is the standard-library Go type whose ROLE it is,
// chosen from what the profile's positions measurably do -- the rule that made a
// read-only Stream position an io.Reader. Here that role is a func, and a
// consumer supplies the lookup its own resources need.
//
// This does NOT claim a ResourceManager is a function. It claims that the one
// thing the profile does with one is call GetObject, which is what the
// projection reproduces.
type ResourceContentManager struct {
	base *ContentManager
	// resourceManager is the private field the constructor stores and
	// OpenStream reads.
	resourceManager func(string) any
}

// The two FrameworkResources messages this type throws, verified byte for byte.
const (
	openResourceNotFound  = "Error loading \"%s\". Resource not found."
	openResourceNotBinary = "Error loading \"%s\". Not a binary resource."
)

// NewResourceContentManager is
// ResourceContentManager::.ctor(IServiceProvider, ResourceManager), measured:
//
//	base..ctor(serviceProvider);
//	if (resourceManager == null)
//	    throw new ArgumentNullException("resourceManager");
//	this.resourceManager = resourceManager;
//
// The BASE constructor runs FIRST, so a null service provider is refused by
// ContentManager's own guard before the resource manager is even looked at.
func NewResourceContentManager(serviceProvider any, resourceManager func(string) any) (*ResourceContentManager, error) {
	base, err := NewContentManagerByIServiceProvider(serviceProvider)
	if err != nil {
		return nil, err
	}
	if resourceManager == nil {
		return nil, contentArgumentNullError("resourceManager")
	}
	manager := &ResourceContentManager{base: base, resourceManager: resourceManager}
	// Install the CLR `this`, so the base's three ObjectDisposedException sites
	// name ResourceContentManager rather than ContentManager.
	base.bindDerived(manager)
	return manager, nil
}

// clrTypeName is what `this.ToString()` answers on a ResourceContentManager.
func (m *ResourceContentManager) clrTypeName() string {
	return "Microsoft.Xna.Framework.Content.ResourceContentManager"
}

// OpenStream is ResourceContentManager::OpenStream(String), measured in full:
//
//	object resource = resourceManager.GetObject(assetName);
//	if (resource == null)
//	    throw new ContentLoadException(Format(OpenResourceNotFound, assetName));
//	byte[] bytes = resource as byte[];
//	if (bytes == null)
//	    throw new ContentLoadException(Format(OpenResourceNotBinary, assetName));
//	return new MemoryStream(bytes, false);
//
// Two refusals with two DIFFERENT messages: a resource that is absent and one
// that is present but not binary. Collapsing them would lose the distinction a
// consumer needs -- the first is a missing asset, the second a misbuilt one.
//
// The stream is over the bytes the lookup answered, so it is read-only and
// needs no CNA handle: nothing here reaches the native content reader.
func (m *ResourceContentManager) OpenStream(assetName string) (io.Reader, error) {
	if m == nil || m.resourceManager == nil {
		return nil, contentArgumentNullError("resourceManager")
	}
	resource := m.resourceManager(assetName)
	if resource == nil {
		return nil, fmt.Errorf("%w: %s", errContentLoad,
			fmt.Sprintf(openResourceNotFound, assetName))
	}
	data, ok := resource.([]byte)
	if !ok {
		return nil, fmt.Errorf("%w: %s", errContentLoad,
			fmt.Sprintf(openResourceNotBinary, assetName))
	}
	// `new MemoryStream(bytes, false)` -- the second argument is writable, and
	// it is FALSE, so the stream the reference hands back cannot be written.
	return bytes.NewReader(data), nil
}

// The inherited ContentManager surface.
//
// Go has no inheritance, so the base is held in an unexported field and every
// inherited public member is a measured forwarding method. The adapter is never
// embedded, never exported and never returned: no caller can reach the base
// half or replace it.
//
// Load and ReadAsset are absent here for a reason that is not composition. They
// are GENERIC methods, and Go cannot declare a method with its own type
// parameters, so the settled rule projects them as package FUNCTIONS taking the
// receiver first -- ContentManagerLoad and ContentManagerReadAsset. A
// ResourceContentManager reaches them through its base, which is what
// contentManager() answers with.

// ServiceProvider is the inherited ContentManager::get_ServiceProvider.
func (m *ResourceContentManager) ServiceProvider() (any, error) {
	if m == nil {
		return nil, contentArgumentNullError("resourceManager")
	}
	return m.base.ServiceProvider()
}

// RootDirectory is the inherited ContentManager::get_RootDirectory.
func (m *ResourceContentManager) RootDirectory() (string, error) {
	if m == nil {
		return "", contentArgumentNullError("resourceManager")
	}
	return m.base.RootDirectory()
}

// SetRootDirectory is the inherited ContentManager::set_RootDirectory.
func (m *ResourceContentManager) SetRootDirectory(value string) error {
	if m == nil {
		return contentArgumentNullError("resourceManager")
	}
	return m.base.SetRootDirectory(value)
}

// Unload is the inherited ContentManager::Unload. It is one of the base's three
// IDENTITY SITES, so a disposed ResourceContentManager's refusal names
// ResourceContentManager -- which is the whole reason the CLR `this` is bound.
func (m *ResourceContentManager) Unload() error {
	if m == nil {
		return contentArgumentNullError("resourceManager")
	}
	return m.base.Unload()
}

// Dispose is the inherited ContentManager::Dispose().
func (m *ResourceContentManager) Dispose() error {
	if m == nil {
		return nil
	}
	return m.base.DisposeByNone()
}

// contentManager answers the base half, so the package's generic functions --
// ContentManagerLoad and ContentManagerReadAsset -- can be reached with a
// ResourceContentManager. It is unexported: the contract declares no accessor
// for the base object, and exporting one would hand out the composition.
func (m *ResourceContentManager) contentManager() *ContentManager {
	if m == nil {
		return nil
	}
	return m.base
}
