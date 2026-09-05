package media

import (
	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// The two picture collections, the same six members over the same shape as the
// five music collections Foundation 95 projected.

// PictureCollection is Microsoft.Xna.Framework.Media.PictureCollection.
type PictureCollection struct{ mediaObject }

// newPictureCollection wraps a handle CNA produced.
func newPictureCollection(handle uint64) *PictureCollection {
	return &PictureCollection{mediaObject{handle: handle}}
}

// Dispose is PictureCollection::Dispose().
func (p *PictureCollection) Dispose() error {
	if p == nil || p.disposed {
		return nil
	}
	p.disposed = true
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil
	}
	if err := runtime.MediaPictureCollectionDispose(p.handle); err != nil {
		return err
	}
	return runtime.MediaPictureCollectionDestroy(p.handle)
}

// Finalize is PictureCollection::Finalize, the protected finalizer.
func (p *PictureCollection) Finalize() error { return p.Dispose() }

// IsDisposed is PictureCollection::get_IsDisposed, read from the managed latch.
func (p *PictureCollection) IsDisposed() bool { return p == nil || p.disposed }

// Count is PictureCollection::get_Count.
func (p *PictureCollection) Count() (int32, error) {
	if err := p.usable(); err != nil {
		return 0, err
	}
	return runtimeFor().MediaPictureCollectionCount(p.handle)
}

// Item is PictureCollection::get_Item(Int32), the indexer.
func (p *PictureCollection) Item(index int32) (*Picture, error) {
	if err := p.usable(); err != nil {
		return nil, err
	}
	count, err := runtimeFor().MediaPictureCollectionCount(p.handle)
	if err != nil {
		return nil, err
	}
	if err := checkCollectionIndex(index, count); err != nil {
		return nil, err
	}
	handle, err := runtimeFor().MediaPictureCollectionAt(p.handle, index)
	if err != nil {
		return nil, err
	}
	return newPicture(handle), nil
}

// GetEnumerator is PictureCollection::GetEnumerator(), the IEnumerable<Picture> member.
func (p *PictureCollection) GetEnumerator() (framework.Iterator[*Picture], error) {
	if err := p.usable(); err != nil {
		return nil, err
	}
	count, err := runtimeFor().MediaPictureCollectionCount(p.handle)
	if err != nil {
		return nil, err
	}
	return &pictureCollectionIterator{owner: p, count: count, index: -1}, nil
}

// pictureCollectionIterator walks by index, reading the count ONCE when the walk
// starts -- the reference's own shape.
type pictureCollectionIterator struct {
	owner *PictureCollection
	count int32
	index int32
}

// Next is the Iterator adapter's one member.
func (i *pictureCollectionIterator) Next() (*Picture, bool, error) {
	if i.index+1 >= i.count {
		return nil, false, nil
	}
	i.index++
	item, err := i.owner.Item(i.index)
	if err != nil {
		return nil, false, err
	}
	return item, true, nil
}

// PictureAlbumCollection is Microsoft.Xna.Framework.Media.PictureAlbumCollection.
type PictureAlbumCollection struct{ mediaObject }

// newPictureAlbumCollection wraps a handle CNA produced.
func newPictureAlbumCollection(handle uint64) *PictureAlbumCollection {
	return &PictureAlbumCollection{mediaObject{handle: handle}}
}

// Dispose is PictureAlbumCollection::Dispose().
func (p *PictureAlbumCollection) Dispose() error {
	if p == nil || p.disposed {
		return nil
	}
	p.disposed = true
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil
	}
	if err := runtime.MediaPictureAlbumCollectionDispose(p.handle); err != nil {
		return err
	}
	return runtime.MediaPictureAlbumCollectionDestroy(p.handle)
}

// Finalize is PictureAlbumCollection::Finalize, the protected finalizer.
func (p *PictureAlbumCollection) Finalize() error { return p.Dispose() }

// IsDisposed is PictureAlbumCollection::get_IsDisposed, read from the managed latch.
func (p *PictureAlbumCollection) IsDisposed() bool { return p == nil || p.disposed }

// Count is PictureAlbumCollection::get_Count.
func (p *PictureAlbumCollection) Count() (int32, error) {
	if err := p.usable(); err != nil {
		return 0, err
	}
	return runtimeFor().MediaPictureAlbumCollectionCount(p.handle)
}

// Item is PictureAlbumCollection::get_Item(Int32), the indexer.
func (p *PictureAlbumCollection) Item(index int32) (*PictureAlbum, error) {
	if err := p.usable(); err != nil {
		return nil, err
	}
	count, err := runtimeFor().MediaPictureAlbumCollectionCount(p.handle)
	if err != nil {
		return nil, err
	}
	if err := checkCollectionIndex(index, count); err != nil {
		return nil, err
	}
	handle, err := runtimeFor().MediaPictureAlbumCollectionAt(p.handle, index)
	if err != nil {
		return nil, err
	}
	return newPictureAlbum(handle), nil
}

// GetEnumerator is PictureAlbumCollection::GetEnumerator(), the IEnumerable<PictureAlbum> member.
func (p *PictureAlbumCollection) GetEnumerator() (framework.Iterator[*PictureAlbum], error) {
	if err := p.usable(); err != nil {
		return nil, err
	}
	count, err := runtimeFor().MediaPictureAlbumCollectionCount(p.handle)
	if err != nil {
		return nil, err
	}
	return &pictureAlbumCollectionIterator{owner: p, count: count, index: -1}, nil
}

// pictureAlbumCollectionIterator walks by index, reading the count ONCE when the walk
// starts -- the reference's own shape.
type pictureAlbumCollectionIterator struct {
	owner *PictureAlbumCollection
	count int32
	index int32
}

// Next is the Iterator adapter's one member.
func (i *pictureAlbumCollectionIterator) Next() (*PictureAlbum, bool, error) {
	if i.index+1 >= i.count {
		return nil, false, nil
	}
	i.index++
	item, err := i.owner.Item(i.index)
	if err != nil {
		return nil, false, err
	}
	return item, true, nil
}
