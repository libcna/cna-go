package content

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The managed half of ContentManager is testable without a native runtime,
// because the reference's own constructors, RootDirectory and Dispose reach
// nothing: they store, read and unload. Every test below pins a body that is
// measured from IL, and each says which one.

// TestConstructorRefusesANilServiceProvider pins the ONE guard both
// constructors share:
//
//	if (serviceProvider == null) throw new ArgumentNullException("serviceProvider");
func TestConstructorRefusesANilServiceProvider(t *testing.T) {
	if _, err := NewContentManagerByIServiceProvider(nil); err == nil {
		t.Fatal("NewContentManagerByIServiceProvider(nil) reported no error")
	} else if !errors.Is(err, errContentArgumentNull) {
		t.Fatalf("NewContentManagerByIServiceProvider(nil) = %v, want the argument-null refusal", err)
	}
	if _, err := NewContentManagerByIServiceProviderAndString(nil, "Content"); err == nil {
		t.Fatal("the two-argument constructor accepted a nil service provider")
	}
}

// TestConstructorStoresWhatItWasGiven pins both `stfld`s and the getter that
// reads one of them. The reference's one-argument constructor leaves the root
// directory String.Empty; Go's empty string is the same value.
func TestConstructorStoresWhatItWasGiven(t *testing.T) {
	services := struct{ name string }{name: "provider"}
	manager, err := NewContentManagerByIServiceProvider(services)
	if err != nil {
		t.Fatalf("NewContentManagerByIServiceProvider: %v", err)
	}
	stored, err := manager.ServiceProvider()
	if err != nil {
		t.Fatalf("ServiceProvider: %v", err)
	}
	if stored != any(services) {
		t.Fatalf("ServiceProvider = %v, want the provider the constructor was given", stored)
	}
	root, err := manager.RootDirectory()
	if err != nil {
		t.Fatalf("RootDirectory: %v", err)
	}
	if root != "" {
		t.Fatalf("RootDirectory = %q, want the empty root the one-argument constructor leaves", root)
	}

	withRoot, err := NewContentManagerByIServiceProviderAndString(services, "Assets")
	if err != nil {
		t.Fatalf("NewContentManagerByIServiceProviderAndString: %v", err)
	}
	if root, err = withRoot.RootDirectory(); err != nil || root != "Assets" {
		t.Fatalf("RootDirectory = %q, %v, want \"Assets\" and no error", root, err)
	}
}

// TestSetRootDirectoryDoesNotUnload pins the sentence CNA and the reference
// agree on: "Existing cache entries are not unloaded." With no native manager
// the setter is a store, and the getter answers it.
func TestSetRootDirectoryDoesNotUnload(t *testing.T) {
	manager, err := NewContentManagerByIServiceProvider(struct{}{})
	if err != nil {
		t.Fatalf("NewContentManagerByIServiceProvider: %v", err)
	}
	if err := manager.SetRootDirectory("first"); err != nil {
		t.Fatalf("SetRootDirectory: %v", err)
	}
	if err := manager.SetRootDirectory("second"); err != nil {
		t.Fatalf("SetRootDirectory: %v", err)
	}
	root, err := manager.RootDirectory()
	if err != nil {
		t.Fatalf("RootDirectory: %v", err)
	}
	if root != "second" {
		t.Fatalf("RootDirectory = %q, want the last value assigned", root)
	}
}

// TestUnloadOnAManagerThatNeverLoadedIsANoOp pins the deferred-creation rule
// from the observable side: a manager with no native half unloads nothing and
// reports no failure, exactly as the reference's Unload over an empty cache
// does.
func TestUnloadOnAManagerThatNeverLoadedIsANoOp(t *testing.T) {
	manager, err := NewContentManagerByIServiceProvider(struct{}{})
	if err != nil {
		t.Fatalf("NewContentManagerByIServiceProvider: %v", err)
	}
	if manager.resource != nil {
		t.Fatal("the constructor created a native content manager, which defers to the first load")
	}
	if err := manager.Unload(); err != nil {
		t.Fatalf("Unload on a manager that never loaded: %v", err)
	}
}

// TestDisposeIsIdempotentAndClosesEveryMember pins Dispose(bool)'s body and the
// ObjectDisposedException every member throws afterwards.
func TestDisposeIsIdempotentAndClosesEveryMember(t *testing.T) {
	manager, err := NewContentManagerByIServiceProvider(struct{}{})
	if err != nil {
		t.Fatalf("NewContentManagerByIServiceProvider: %v", err)
	}
	if err := manager.DisposeByNone(); err != nil {
		t.Fatalf("DisposeByNone: %v", err)
	}
	if err := manager.DisposeByNone(); err != nil {
		t.Fatalf("a second DisposeByNone: %v", err)
	}
	// Dispose(false) is the finalizer path, whose body is `if (disposing)`
	// taken false: it does nothing at all, including on a live manager.
	live, err := NewContentManagerByIServiceProvider(struct{}{})
	if err != nil {
		t.Fatalf("NewContentManagerByIServiceProvider: %v", err)
	}
	if err := live.DisposeByBoolean(false); err != nil {
		t.Fatalf("DisposeByBoolean(false): %v", err)
	}
	if live.disposed {
		t.Fatal("DisposeByBoolean(false) disposed the manager; the reference's body is `if (disposing) Unload()`")
	}

	if _, err := manager.RootDirectory(); !errors.Is(err, errContentManagerDisposed) {
		t.Fatalf("RootDirectory after Dispose = %v, want the disposed refusal", err)
	}
	if err := manager.SetRootDirectory("x"); !errors.Is(err, errContentManagerDisposed) {
		t.Fatalf("SetRootDirectory after Dispose = %v, want the disposed refusal", err)
	}
	if err := manager.Unload(); !errors.Is(err, errContentManagerDisposed) {
		t.Fatalf("Unload after Dispose = %v, want the disposed refusal", err)
	}
	if _, err := manager.OpenStream("a"); !errors.Is(err, errContentManagerDisposed) {
		t.Fatalf("OpenStream after Dispose = %v, want the disposed refusal", err)
	}
	if _, err := ContentManagerLoad[*struct{}](manager, "a"); !errors.Is(err, errContentManagerDisposed) {
		t.Fatalf("Load after Dispose = %v, want the disposed refusal", err)
	}
}

// TestLoadRefusesATypeOutsideTheClosedSetBeforeReachingTheDevice is the
// milestone's honesty test. The refusal must name the Go type, and it must
// happen without a graphics device -- so a consumer learns that CNA-Go cannot
// load a SpriteFont, rather than that no device is registered.
func TestLoadRefusesATypeOutsideTheClosedSetBeforeReachingTheDevice(t *testing.T) {
	manager, err := NewContentManagerByIServiceProvider(struct{}{})
	if err != nil {
		t.Fatalf("NewContentManagerByIServiceProvider: %v", err)
	}
	type spriteFontPlaceholder struct{}
	_, err = ContentManagerLoad[*spriteFontPlaceholder](manager, "font")
	if err == nil {
		t.Fatal("Load of an unprojected asset type reported no error")
	}
	if !errors.Is(err, errContentUnsupportedAsset) {
		t.Fatalf("Load = %v, want the unsupported-asset refusal", err)
	}
	if !strings.Contains(err.Error(), "spriteFontPlaceholder") {
		t.Fatalf("Load = %v, want the refusal to name the Go type it cannot load", err)
	}
	if manager.resource != nil {
		t.Fatal("the refusal created a native content manager, which it must not: it never reaches one")
	}
}

// TestLoadRefusesAnEmptyAssetName pins the reference's
// `if (assetName == null) throw new ArgumentNullException("assetName")`. Go has
// no null string, so the empty string is what stands in for it, and the message
// carries the reference's parameter name.
func TestLoadRefusesAnEmptyAssetName(t *testing.T) {
	manager, err := NewContentManagerByIServiceProvider(struct{}{})
	if err != nil {
		t.Fatalf("NewContentManagerByIServiceProvider: %v", err)
	}
	if _, err := ContentManagerLoad[*struct{}](manager, ""); !errors.Is(err, errContentArgumentNull) {
		t.Fatalf("Load(\"\") = %v, want the argument-null refusal", err)
	}
	if _, err := manager.OpenStream(""); !errors.Is(err, errContentArgumentNull) {
		t.Fatalf("OpenStream(\"\") = %v, want the argument-null refusal", err)
	}
}

// TestReadAssetAcceptsAndIgnoresTheRecordAction pins the recorded divergence:
// the action has no counterpart because CNA's manager owns the relationship it
// exists for. It must not change the answer.
func TestReadAssetAcceptsAndIgnoresTheRecordAction(t *testing.T) {
	manager, err := NewContentManagerByIServiceProvider(struct{}{})
	if err != nil {
		t.Fatalf("NewContentManagerByIServiceProvider: %v", err)
	}
	calls := 0
	record := func(any) { calls++ }
	if _, err := ContentManagerReadAsset[*struct{}](manager, "asset", record); !errors.Is(err, errContentUnsupportedAsset) {
		t.Fatalf("ReadAsset = %v, want the unsupported-asset refusal Load gives", err)
	}
	if calls != 0 {
		t.Fatalf("the record action was invoked %d times; the projection accepts it and uses it nowhere", calls)
	}
}

// TestOpenStreamRefusesWithoutADeviceRatherThanGuessingAPath is the boundary
// between the two halves. OpenStream needs CNA to resolve the path, so it needs
// the native manager, so it needs the device -- and with no
// IGraphicsDeviceService registered it must say so rather than joining the path
// itself and opening a file the reference would not have opened.
func TestOpenStreamRefusesWithoutADeviceRatherThanGuessingAPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "asset.xnb"), []byte("not an asset"), 0o600); err != nil {
		t.Fatalf("writing the decoy: %v", err)
	}
	manager, err := NewContentManagerByIServiceProviderAndString(struct{}{}, dir)
	if err != nil {
		t.Fatalf("NewContentManagerByIServiceProviderAndString: %v", err)
	}
	reader, err := manager.OpenStream("asset")
	if err == nil {
		t.Fatal("OpenStream opened a stream with no graphics device service; CNA resolves the path, not Go")
	}
	if reader != nil {
		t.Fatal("OpenStream returned a reader alongside its error")
	}
}
