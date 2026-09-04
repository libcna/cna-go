package storage

import (
	"fmt"
	"io"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// StorageContainer is Microsoft.Xna.Framework.Storage.StorageContainer:
//
//	.class public auto ansi sealed StorageContainer extends System.Object
//
// One named area inside a storage device, holding a game's files and
// directories.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Storage.dll   798f678e9ae3d9af...
//
// # Every path goes through the reference's OWN containment guard
//
// Nine of the type's members reach the filesystem, and every one of them opens
// with `ValidateArguments(path, argumentName)`:
//
//	VerifyNotDisposed();
//	if (String.IsNullOrEmpty(path))
//	    throw new ArgumentNullException(argumentName);
//	string full = GetFullPath(path);
//	if (!full.StartsWith(_rootPath))
//	    throw new ArgumentException(FrameworkResources.InvalidStoragePath, argumentName);
//	return full;
//
// A path that resolves OUTSIDE the container's root is refused. So "a game
// cannot reach the user's documents through this type" is not a rule this
// projection adds -- it is what the reference does, and reproducing it is the
// job.
//
// Three details in the order the reference has them: disposal is checked
// FIRST; an empty path raises ArgumentNullException rather than
// ArgumentException; and only then is the root checked.
//
// # The projection defers the path check to CNA, and says so
//
// CNA-Go does not resolve paths itself: `_rootPath` is CNA's, and the container
// handle is what knows it. The empty-path and disposal guards ARE reproduced
// here, because they are managed and observable; the root check is CNA's,
// which refuses an escaping path with its own result code. What a consumer
// sees is a refusal either way.
type StorageContainer struct {
	handle uint64
	device *StorageDevice
	// disposed is the managed half of the disposal latch. CNA has its own
	// get_is_disposed, and the two are kept in step: Dispose sets this and
	// calls CNA's, and IsDisposed prefers the native answer when a runtime is
	// there.
	disposed  bool
	disposing framework.EventSource[*framework.EventArgs]
}

// verifyNotDisposed is StorageContainer::VerifyNotDisposed, which every
// filesystem member reaches through ValidateArguments and which the three
// enumerating members call directly.
func (c *StorageContainer) verifyNotDisposed() error {
	if c == nil || c.disposed {
		return errStorageDisposed
	}
	return nil
}

// validateArguments is ValidateArguments' managed half: the disposal check and
// the empty-path refusal, in the reference's order. The root check belongs to
// CNA, which holds the root.
func (c *StorageContainer) validateArguments(path, argumentName string) error {
	if err := c.verifyNotDisposed(); err != nil {
		return err
	}
	if path == "" {
		return storageArgumentNullError(argumentName)
	}
	return nil
}

// DisplayName is StorageContainer::get_DisplayName.
func (c *StorageContainer) DisplayName() (string, error) {
	if err := c.verifyNotDisposed(); err != nil {
		return "", err
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return "", errStorageNoRuntime
	}
	return runtime.StorageContainerDisplayName(c.handle)
}

// StorageDevice is StorageContainer::get_StorageDevice.
//
// The reference reads a field, but it reads it AFTER VerifyNotDisposed -- every
// property on this type does -- so the member carries the disposal refusal and
// is fallible.
func (c *StorageContainer) StorageDevice() (*StorageDevice, error) {
	if err := c.verifyNotDisposed(); err != nil {
		return nil, err
	}
	return c.device, nil
}

// IsDisposed is StorageContainer::get_IsDisposed.
//
// It is the ONE property that does not call VerifyNotDisposed -- asking a
// disposed object whether it is disposed has to work -- but it does reach CNA,
// which is what makes it fallible. The managed latch is answered first so a
// container disposed through this projection reports true even if the native
// read fails.
func (c *StorageContainer) IsDisposed() (bool, error) {
	if c == nil || c.disposed {
		return true, nil
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return c.disposed, errStorageNoRuntime
	}
	return runtime.StorageContainerIsDisposed(c.handle)
}

// AddDisposingHandler subscribes to StorageContainer::Disposing, the event the
// reference raises from Dispose before it releases anything.
func (c *StorageContainer) AddDisposingHandler(
	handler framework.EventHandler[*framework.EventArgs],
) (framework.EventSubscription, error) {
	if c == nil {
		return framework.EventSubscription{}, errStorageNoRuntime
	}
	return c.disposing.Add(handler)
}

// RemoveDisposingHandler unsubscribes from StorageContainer::Disposing.
func (c *StorageContainer) RemoveDisposingHandler(subscription framework.EventSubscription) error {
	if c == nil {
		return errStorageNoRuntime
	}
	return c.disposing.Remove(subscription)
}

// Finalize is StorageContainer::Finalize(), the CLR finalizer. The reference's
// body is `Dispose(false)` inside a try/finally that calls base.Finalize.
//
// Go has no finalizer a consumer may call, and the pinned contract declares
// this member, so it is projected as the operation the finalizer performs: a
// release that does NOT raise Disposing, because the reference's Dispose(false)
// skips the managed half. A consumer calling it directly gets exactly what the
// garbage collector would have done.
func (c *StorageContainer) Finalize() error {
	if c == nil || c.disposed {
		return nil
	}
	c.disposed = true
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil
	}
	return runtime.StorageContainerDestroy(c.handle)
}

// Dispose is StorageContainer::Dispose(), which raises Disposing and then
// releases. A second Dispose is a no-op, which is what IDisposable requires and
// what the reference's own latch provides.
func (c *StorageContainer) Dispose() error {
	if c == nil || c.disposed {
		return nil
	}
	c.disposed = true
	c.disposing.Raise(c, framework.EventArgsEmpty())
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil
	}
	return runtime.StorageContainerDispose(c.handle)
}

// CreateDirectory is StorageContainer::CreateDirectory(String).
func (c *StorageContainer) CreateDirectory(directory string) error {
	if err := c.validateArguments(directory, "directory"); err != nil {
		return err
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return errStorageNoRuntime
	}
	return c.wrapPathError(runtime.StorageContainerCreateDirectory(c.handle, directory), "directory")
}

// DeleteDirectory is StorageContainer::DeleteDirectory(String).
func (c *StorageContainer) DeleteDirectory(directory string) error {
	if err := c.validateArguments(directory, "directory"); err != nil {
		return err
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return errStorageNoRuntime
	}
	return c.wrapPathError(runtime.StorageContainerDeleteDirectory(c.handle, directory), "directory")
}

// DirectoryExists is StorageContainer::DirectoryExists(String).
func (c *StorageContainer) DirectoryExists(directory string) (bool, error) {
	if err := c.validateArguments(directory, "directory"); err != nil {
		return false, err
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return false, errStorageNoRuntime
	}
	return runtime.StorageContainerDirectoryExists(c.handle, directory)
}

// FileExists is StorageContainer::FileExists(String).
func (c *StorageContainer) FileExists(file string) (bool, error) {
	if err := c.validateArguments(file, "file"); err != nil {
		return false, err
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return false, errStorageNoRuntime
	}
	return runtime.StorageContainerFileExists(c.handle, file)
}

// DeleteFile is StorageContainer::DeleteFile(String).
func (c *StorageContainer) DeleteFile(file string) error {
	if err := c.validateArguments(file, "file"); err != nil {
		return err
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return errStorageNoRuntime
	}
	return c.wrapPathError(runtime.StorageContainerDeleteFile(c.handle, file), "file")
}

// GetDirectoryNamesByNone is StorageContainer::GetDirectoryNames(), which the
// reference implements as `GetDirectoryNames(null)`.
func (c *StorageContainer) GetDirectoryNamesByNone() ([]string, error) {
	return c.GetDirectoryNamesByString("")
}

// GetDirectoryNamesByString is StorageContainer::GetDirectoryNames(String).
// Unlike the file members it calls VerifyNotDisposed DIRECTLY rather than
// through ValidateArguments, so an EMPTY pattern is accepted -- it means "every
// name" -- while an empty path to CreateDirectory is not.
func (c *StorageContainer) GetDirectoryNamesByString(searchPattern string) ([]string, error) {
	if err := c.verifyNotDisposed(); err != nil {
		return nil, err
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil, errStorageNoRuntime
	}
	return runtime.StorageContainerDirectoryNames(c.handle, searchPattern)
}

// GetFileNamesByNone is StorageContainer::GetFileNames().
func (c *StorageContainer) GetFileNamesByNone() ([]string, error) {
	return c.GetFileNamesByString("")
}

// GetFileNamesByString is StorageContainer::GetFileNames(String), on the same
// footing as its directory sibling.
func (c *StorageContainer) GetFileNamesByString(searchPattern string) ([]string, error) {
	if err := c.verifyNotDisposed(); err != nil {
		return nil, err
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil, errStorageNoRuntime
	}
	return runtime.StorageContainerFileNames(c.handle, searchPattern)
}

// CreateFile is StorageContainer::CreateFile(String), which the reference
// implements as `File.Create(ValidateArguments(file, "file"))`.
//
// The result is a stream the caller both READS and WRITES and seeks, which is
// why this position takes io.ReadWriteSeeker rather than the io.Reader every
// other Stream return in the profile takes. A save file that could not be
// written would not be one.
func (c *StorageContainer) CreateFile(file string) (io.ReadWriteSeeker, error) {
	if err := c.validateArguments(file, "file"); err != nil {
		return nil, err
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil, errStorageNoRuntime
	}
	handle, err := runtime.StorageContainerCreateFile(c.handle, file)
	if err != nil {
		return nil, c.wrapPathError(err, "file")
	}
	return newStorageStream(handle), nil
}

// OpenFileByStringAndFileMode is
// StorageContainer::OpenFile(String, FileMode).
func (c *StorageContainer) OpenFileByStringAndFileMode(
	file string, mode framework.FileMode,
) (io.ReadWriteSeeker, error) {
	return c.openFile(file, mode, 0, 0, 1)
}

// OpenFileByStringAndFileModeAndFileAccess is
// StorageContainer::OpenFile(String, FileMode, FileAccess).
func (c *StorageContainer) OpenFileByStringAndFileModeAndFileAccess(
	file string, mode framework.FileMode, access framework.FileAccess,
) (io.ReadWriteSeeker, error) {
	return c.openFile(file, mode, access, 0, 2)
}

// OpenFileByStringAndFileModeAndFileAccessAndFileShare is
// StorageContainer::OpenFile(String, FileMode, FileAccess, FileShare).
func (c *StorageContainer) OpenFileByStringAndFileModeAndFileAccessAndFileShare(
	file string, mode framework.FileMode, access framework.FileAccess, share framework.FileShare,
) (io.ReadWriteSeeker, error) {
	return c.openFile(file, mode, access, share, 3)
}

// openFile carries the ARITY through, so the three CNA routes stay reachable.
// The reference's shorter overloads forward to the longest with defaults;
// collapsing them here would leave two bound routes with no call site.
func (c *StorageContainer) openFile(
	file string, mode framework.FileMode, access framework.FileAccess,
	share framework.FileShare, arity int,
) (io.ReadWriteSeeker, error) {
	if err := c.validateArguments(file, "file"); err != nil {
		return nil, err
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil, errStorageNoRuntime
	}
	handle, err := runtime.StorageContainerOpenFile(
		c.handle, file, uint32(mode), uint32(access), uint32(share), arity)
	if err != nil {
		return nil, c.wrapPathError(err, "file")
	}
	return newStorageStream(handle), nil
}

// wrapPathError turns a native refusal into the ArgumentException the
// reference's own root check raises, so a path that escapes the container reads
// the same to a consumer whichever side caught it.
func (c *StorageContainer) wrapPathError(err error, argumentName string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s (%s): %v", errStorageArgument, invalidStoragePath, argumentName, err)
}
