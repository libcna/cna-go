package media

import (
	"github.com/openeggbert/cna-go/internal/interop"
)

// The picture half of the media library.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// Picture and PictureAlbum are the same shape as the five music entities
// Foundation 95 projected: an owned handle, IDisposable, IEquatable<T>, a name
// and a type name. What differs is what each describes.

// Picture is Microsoft.Xna.Framework.Media.Picture.
type Picture struct{ mediaObject }

// newPicture wraps a handle CNA produced. It is unexported because the contract
// declares no constructor.
func newPicture(handle uint64) *Picture { return &Picture{mediaObject{handle: handle}} }

// Dispose is Picture::Dispose(), which latches first and then releases the native
// object -- the ordering Foundation 95 established for this family.
func (p *Picture) Dispose() error {
	if p == nil || p.disposed {
		return nil
	}
	p.disposed = true
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil
	}
	if err := runtime.MediaPictureDispose(p.handle); err != nil {
		return err
	}
	return runtime.MediaPictureDestroy(p.handle)
}

// Finalize is Picture::Finalize, the protected finalizer. Nothing calls it.
func (p *Picture) Finalize() error { return p.Dispose() }

// IsDisposed is Picture::get_IsDisposed, read from the managed latch.
func (p *Picture) IsDisposed() bool { return p == nil || p.disposed }

// Name is Picture::get_Name.
func (p *Picture) Name() (string, error) {
	if err := p.usable(); err != nil {
		return "", err
	}
	return runtimeFor().MediaPictureName(p.handle)
}

// ToString is Picture::ToString(), which the reference implements as the name.
func (p *Picture) ToString() (string, error) {
	if err := p.usable(); err != nil {
		return "", err
	}
	return runtimeFor().MediaPictureTypeName(p.handle)
}

// GetHashCode is Picture::GetHashCode.
func (p *Picture) GetHashCode() (int32, error) {
	if err := p.usable(); err != nil {
		return 0, err
	}
	return runtimeFor().MediaPictureHashCode(p.handle)
}

// EqualsByObject is Picture::Equals(Object), false for anything that is not a
// Picture -- including nil.
func (p *Picture) EqualsByObject(obj any) (bool, error) {
	other, ok := obj.(*Picture)
	if !ok {
		return false, nil
	}
	return p.EqualsByPicture(other)
}

// EqualsByPicture is Picture::Equals(Picture), the IEquatable<T> member.
func (p *Picture) EqualsByPicture(other *Picture) (bool, error) {
	if err := p.usable(); err != nil {
		return false, err
	}
	if other == nil {
		return false, nil
	}
	if err := other.usable(); err != nil {
		return false, err
	}
	return runtimeFor().MediaPictureEquals(p.handle, other.handle)
}

// PictureOperatorEqualityByPictureAndPicture is Picture::op_Equality. Two nils are EQUAL
// and one nil is not, which the reference answers before the native call.
func PictureOperatorEqualityByPictureAndPicture(left, right *Picture) (bool, error) {
	if left == nil || right == nil {
		return left == right, nil
	}
	return left.EqualsByPicture(right)
}

// PictureOperatorInequalityByPictureAndPicture is Picture::op_Inequality.
func PictureOperatorInequalityByPictureAndPicture(left, right *Picture) (bool, error) {
	equal, err := PictureOperatorEqualityByPictureAndPicture(left, right)
	return !equal, err
}

// PictureAlbum is Microsoft.Xna.Framework.Media.PictureAlbum.
type PictureAlbum struct{ mediaObject }

// newPictureAlbum wraps a handle CNA produced. It is unexported because the contract
// declares no constructor.
func newPictureAlbum(handle uint64) *PictureAlbum { return &PictureAlbum{mediaObject{handle: handle}} }

// Dispose is PictureAlbum::Dispose(), which latches first and then releases the native
// object -- the ordering Foundation 95 established for this family.
func (p *PictureAlbum) Dispose() error {
	if p == nil || p.disposed {
		return nil
	}
	p.disposed = true
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil
	}
	if err := runtime.MediaPictureAlbumDispose(p.handle); err != nil {
		return err
	}
	return runtime.MediaPictureAlbumDestroy(p.handle)
}

// Finalize is PictureAlbum::Finalize, the protected finalizer. Nothing calls it.
func (p *PictureAlbum) Finalize() error { return p.Dispose() }

// IsDisposed is PictureAlbum::get_IsDisposed, read from the managed latch.
func (p *PictureAlbum) IsDisposed() bool { return p == nil || p.disposed }

// Name is PictureAlbum::get_Name.
func (p *PictureAlbum) Name() (string, error) {
	if err := p.usable(); err != nil {
		return "", err
	}
	return runtimeFor().MediaPictureAlbumName(p.handle)
}

// ToString is PictureAlbum::ToString(), which the reference implements as the name.
func (p *PictureAlbum) ToString() (string, error) {
	if err := p.usable(); err != nil {
		return "", err
	}
	return runtimeFor().MediaPictureAlbumTypeName(p.handle)
}

// GetHashCode is PictureAlbum::GetHashCode.
func (p *PictureAlbum) GetHashCode() (int32, error) {
	if err := p.usable(); err != nil {
		return 0, err
	}
	return runtimeFor().MediaPictureAlbumHashCode(p.handle)
}

// EqualsByObject is PictureAlbum::Equals(Object), false for anything that is not a
// PictureAlbum -- including nil.
func (p *PictureAlbum) EqualsByObject(obj any) (bool, error) {
	other, ok := obj.(*PictureAlbum)
	if !ok {
		return false, nil
	}
	return p.EqualsByPictureAlbum(other)
}

// EqualsByPictureAlbum is PictureAlbum::Equals(PictureAlbum), the IEquatable<T> member.
func (p *PictureAlbum) EqualsByPictureAlbum(other *PictureAlbum) (bool, error) {
	if err := p.usable(); err != nil {
		return false, err
	}
	if other == nil {
		return false, nil
	}
	if err := other.usable(); err != nil {
		return false, err
	}
	return runtimeFor().MediaPictureAlbumEquals(p.handle, other.handle)
}

// PictureAlbumOperatorEqualityByPictureAlbumAndPictureAlbum is PictureAlbum::op_Equality. Two nils are EQUAL
// and one nil is not, which the reference answers before the native call.
func PictureAlbumOperatorEqualityByPictureAlbumAndPictureAlbum(left, right *PictureAlbum) (bool, error) {
	if left == nil || right == nil {
		return left == right, nil
	}
	return left.EqualsByPictureAlbum(right)
}

// PictureAlbumOperatorInequalityByPictureAlbumAndPictureAlbum is PictureAlbum::op_Inequality.
func PictureAlbumOperatorInequalityByPictureAlbumAndPictureAlbum(left, right *PictureAlbum) (bool, error) {
	equal, err := PictureAlbumOperatorEqualityByPictureAlbumAndPictureAlbum(left, right)
	return !equal, err
}
