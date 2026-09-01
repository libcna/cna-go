package content

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	graphics "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Graphics"
	"github.com/openeggbert/cna-go/internal/interop"
	"github.com/openeggbert/cna-go/internal/servicebridge"
)

// ---------------------------------------------------------------------------
// Foundation 63 — ContentManager, the root of the Content subsystem.
// ---------------------------------------------------------------------------

// ContentManager is Microsoft.Xna.Framework.Content.ContentManager.
//
//	.class public ContentManager extends System.Object implements IDisposable
//	  .ctor(IServiceProvider)
//	  .ctor(IServiceProvider, String)
//
// # Creation is DEFERRED, and CNA is why
//
// The reference's constructors store the service provider and a root directory
// and reach nothing: the graphics device is resolved lazily, at load time.
// CNA's `cna_content_manager_create` needs a callback-scoped device handle,
// which exists only inside a lifecycle callback.
//
// A projection that created eagerly would refuse a constructor the reference
// accepts anywhere. So the native manager is created at the FIRST operation
// that needs it -- which is inside a callback by construction, because every
// such operation reaches the device -- and the constructors do what the
// reference's do: store and return.
//
// # Ownership
//
//	OWNED, generation-checked, destroyed with cna_content_manager_destroy.
//
// An asset a load hands back is INDEPENDENTLY owned: CNA states that it
// "remains valid across content-manager unload or destruction and must be
// destroyed before the parent game", which is the reference's rule too --
// Unload disposes what the manager still holds, and a caller that kept a
// reference keeps a live object.
type ContentManager struct {
	// services is the IServiceProvider the constructor was given. The reference
	// stores it and resolves the device out of it at load time; so does this.
	services any
	// rootDirectory is the value until the native manager exists, and the
	// value the native manager is created with.
	rootDirectory string
	// resource is the native manager, created lazily. It is nil until the first
	// operation that needs one.
	resource *interop.Resource
	disposed bool
}

// errContentArgumentNull projects System.ArgumentNullException, which both
// constructors throw for a null service provider.
var errContentArgumentNull = errors.New("content argument is nil")

// errContentManagerDisposed projects the ObjectDisposedException every member
// throws after Dispose.
var errContentManagerDisposed = errors.New("ContentManager is disposed")

// NewContentManagerByIServiceProvider is
// ContentManager::.ctor(IServiceProvider):
//
//	if (serviceProvider == null) throw new ArgumentNullException("serviceProvider");
//	this.serviceProvider = serviceProvider;
//
// The root directory it leaves is String.Empty, which the two-argument
// constructor is the way to change.
func NewContentManagerByIServiceProvider(serviceProvider any) (*ContentManager, error) {
	return NewContentManagerByIServiceProviderAndString(serviceProvider, "")
}

// NewContentManagerByIServiceProviderAndString is
// ContentManager::.ctor(IServiceProvider, String), which adds one guard and one
// store:
//
//	if (rootDirectory == null) throw new ArgumentNullException("rootDirectory");
//
// Go has no null string, so that guard has no counterpart and the empty string
// is a valid root -- which CNA says too: "an empty root is valid".
func NewContentManagerByIServiceProviderAndString(serviceProvider any, rootDirectory string) (*ContentManager, error) {
	if serviceProvider == nil {
		return nil, fmt.Errorf("%w: serviceProvider", errContentArgumentNull)
	}
	return &ContentManager{services: serviceProvider, rootDirectory: rootDirectory}, nil
}

// ServiceProvider is ContentManager::get_ServiceProvider, one `ldfld` over what
// the constructor stored. `System.IServiceProvider` is a BCL interface, which
// the settled rule projects to no Go interface, so the position is `any` and
// the value is whatever the consumer supplied.
func (m *ContentManager) ServiceProvider() (any, error) {
	if m == nil {
		return nil, errContentManagerDisposed
	}
	return m.services, nil
}

// RootDirectory is ContentManager::get_RootDirectory. Once the native manager
// exists it is CNA's copy that answers, because CNA is what resolves an asset
// path and a value that disagreed with it would be a lie about where a load
// looks.
func (m *ContentManager) RootDirectory() (string, error) {
	if m == nil || m.disposed {
		return "", errContentManagerDisposed
	}
	if m.resource == nil {
		return m.rootDirectory, nil
	}
	return m.resource.ContentRootDirectory()
}

// SetRootDirectory is ContentManager::set_RootDirectory.
//
// CNA does NOT unload the existing cache when the root changes, and neither
// does the reference: "Existing cache entries are not unloaded." Both leave
// Unload as the way to invalidate them.
func (m *ContentManager) SetRootDirectory(value string) error {
	if m == nil || m.disposed {
		return errContentManagerDisposed
	}
	m.rootDirectory = value
	if m.resource == nil {
		return nil
	}
	return m.resource.SetContentRootDirectory(value)
}

// Unload is ContentManager::Unload, which disposes every asset the manager
// still holds and empties its cache. A manager that has never loaded anything
// has no native half yet and unloads nothing.
func (m *ContentManager) Unload() error {
	if m == nil || m.disposed {
		return errContentManagerDisposed
	}
	if m.resource == nil {
		return nil
	}
	return m.resource.UnloadContent()
}

// OpenStream is ContentManager::OpenStream(String), the protected member that
// opens the `.xnb` file an asset name resolves to:
//
//	return TitleContainer.OpenStream(Path.Combine(RootDirectory, assetName) + ".xnb");
//
// CNA resolves the path -- `cna_content_manager_copy_asset_path` reports the
// root joined with the name, whether or not a file exists there -- and the
// stream itself is Go's, because a `System.IO.Stream` at a public position is
// an io.Reader and no native stream object crosses this ABI. CNA says the same
// from its side: "No native service-provider, path or stream object crosses the
// C boundary."
//
// The `.xnb` suffix is the reference's and is appended here, because CNA's path
// resolution stops at the name.
func (m *ContentManager) OpenStream(assetName string) (io.Reader, error) {
	if m == nil || m.disposed {
		return nil, errContentManagerDisposed
	}
	if assetName == "" {
		return nil, fmt.Errorf("%w: assetName", errContentArgumentNull)
	}
	resource, err := m.native()
	if err != nil {
		return nil, err
	}
	path, err := resource.ContentAssetPath(assetName)
	if err != nil {
		return nil, err
	}
	// The file is returned through an io.Reader, so the failure must be
	// returned as a NIL one. `return os.Open(...)` would hand back a non-nil
	// interface holding a nil *os.File on every failure -- the classic Go typed
	// -nil -- and a caller checking the reader rather than the error would
	// dereference it.
	file, err := os.Open(path + ".xnb")
	if err != nil {
		return nil, err
	}
	return file, nil
}

// DisposeByNone is ContentManager::Dispose(), the sealed IDisposable member:
//
//	Dispose(true);
//	GC.SuppressFinalize(this);
func (m *ContentManager) DisposeByNone() error {
	return m.DisposeByBoolean(true)
}

// DisposeByBoolean is ContentManager::Dispose(bool):
//
//	if (disposing) Unload();
//
// The reference's whole body. It is idempotent here because a second call finds
// the native manager already destroyed, and the reference is idempotent for the
// same kind of reason: Unload on an empty cache does nothing.
func (m *ContentManager) DisposeByBoolean(disposing bool) error {
	if m == nil {
		return errContentManagerDisposed
	}
	if !disposing || m.disposed {
		return nil
	}
	m.disposed = true
	if m.resource == nil {
		return nil
	}
	resource := m.resource
	m.resource = nil
	if err := resource.UnloadContent(); err != nil {
		_ = resource.Dispose()
		return err
	}
	return resource.Dispose()
}

// native creates the CNA manager on first use. See the type comment for why it
// is deferred: CNA needs a callback-scoped device and the reference's
// constructor needs none.
func (m *ContentManager) native() (*interop.Resource, error) {
	if m.resource != nil {
		return m.resource, nil
	}
	device, err := m.graphicsDevice()
	if err != nil {
		return nil, err
	}
	created, err := servicebridge.CreateContentManagerResource(device, m.rootDirectory)
	if err != nil {
		return nil, err
	}
	resource, typed := created.(*interop.Resource)
	if !typed || resource == nil {
		return nil, errors.New("the content manager factory produced no native manager")
	}
	m.resource = resource
	return resource, nil
}

// graphicsDeviceServiceType is the CLR type token the reference's
// `ldtoken IGraphicsDeviceService; GetService` pair resolves with, which is
// what the reference's own ContentManager resolves the device through.
var graphicsDeviceServiceType = reflect.TypeOf((*graphics.IGraphicsDeviceService)(nil)).Elem()

func (m *ContentManager) graphicsDevice() (*graphics.GraphicsDevice, error) {
	container, ok := m.services.(*framework.GameServiceContainer)
	if !ok || container == nil {
		return nil, errors.New("the content manager's service provider is not a GameServiceContainer")
	}
	provider, err := container.GetService(graphicsDeviceServiceType)
	if err != nil {
		return nil, err
	}
	service, typed := provider.(graphics.IGraphicsDeviceService)
	if !typed {
		return nil, errors.New("the content manager's service provider publishes no IGraphicsDeviceService")
	}
	device := service.GraphicsDevice()
	if device == nil {
		return nil, errors.New("the graphics device service publishes no device yet")
	}
	return device, nil
}
