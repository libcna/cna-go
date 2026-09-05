package media

import (
	"github.com/openeggbert/cna-go/internal/interop"
)

// MediaQueue is Microsoft.Xna.Framework.Media.MediaQueue: what the media
// player is playing, in order.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # It is the one media type with NO disposal
//
// Every other handle-owning type in this namespace declares Dispose, Finalize
// and IsDisposed. This one declares none of the three -- four members and all
// of them reads.
//
// CNA agrees: cna_media_player_get_queue answers a BORROWED handle, a view of
// the process-wide queue rather than an object the caller owns. So the
// projection holds it with no disposal latch, which is what makes the type's
// shape and its contract say the same thing.
type MediaQueue struct {
	handle uint64
}

// Count is MediaQueue::get_Count.
func (q *MediaQueue) Count() (int32, error) {
	runtime, err := q.usableQueue()
	if err != nil {
		return 0, err
	}
	return runtime.MediaQueueCount(q.handle)
}

// ActiveSongIndex is MediaQueue::get_ActiveSongIndex.
//
// An EMPTY queue answers -1, which is the reference's own "nothing is playing"
// rather than a failure -- and is why the value is not bounds-checked here.
func (q *MediaQueue) ActiveSongIndex() (int32, error) {
	runtime, err := q.usableQueue()
	if err != nil {
		return 0, err
	}
	return runtime.MediaQueueActiveSongIndex(q.handle)
}

// SetActiveSongIndex is MediaQueue::set_ActiveSongIndex, which MOVES the
// player to another entry rather than merely recording a number.
//
// The index is bounds-checked before it is sent, because the reference's own
// setter indexes its list and an out-of-range value would be an
// ArgumentOutOfRangeException there.
func (q *MediaQueue) SetActiveSongIndex(value int32) error {
	runtime, err := q.usableQueue()
	if err != nil {
		return err
	}
	count, err := runtime.MediaQueueCount(q.handle)
	if err != nil {
		return err
	}
	if err := checkCollectionIndex(value, count); err != nil {
		return err
	}
	return runtime.MediaQueueSetActiveSongIndex(q.handle, value)
}

// ActiveSong is MediaQueue::get_ActiveSong. A queue with nothing playing has
// none, which CNA reports as an availability flag and the reference answers
// null for.
func (q *MediaQueue) ActiveSong() (*Song, error) {
	runtime, err := q.usableQueue()
	if err != nil {
		return nil, err
	}
	handle, available, err := runtime.MediaQueueActiveSong(q.handle)
	if err != nil || !available {
		return nil, err
	}
	return newSong(handle), nil
}

// Item is MediaQueue::get_Item(Int32), the indexer.
func (q *MediaQueue) Item(index int32) (*Song, error) {
	runtime, err := q.usableQueue()
	if err != nil {
		return nil, err
	}
	count, err := runtime.MediaQueueCount(q.handle)
	if err != nil {
		return nil, err
	}
	if err := checkCollectionIndex(index, count); err != nil {
		return nil, err
	}
	handle, err := runtime.MediaQueueAt(q.handle, index)
	if err != nil {
		return nil, err
	}
	return newSong(handle), nil
}

// usableQueue is this type's whole guard. There is no disposal latch to check,
// because the contract declares none and the handle is borrowed.
func (q *MediaQueue) usableQueue() (*interop.Runtime, error) {
	if q == nil {
		return nil, mediaArgumentNullError("queue")
	}
	return mediaPlayerRuntime()
}
