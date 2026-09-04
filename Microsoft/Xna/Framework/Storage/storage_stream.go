package storage

import (
	"errors"
	"io"

	"github.com/openeggbert/cna-go/internal/interop"
)

// storageStream is the Go projection of the System.IO.Stream a container's
// CreateFile and OpenFile hand back.
//
// It is UNEXPORTED on purpose. The pinned contract names `System.IO.Stream` at
// those return positions, and the settled rule gives a BCL type at a signature
// position the standard-library Go type whose ROLE it is -- here
// io.ReadWriteSeeker, because a save file is read, written and sought. What a
// consumer receives is that interface; the concrete type behind it is CNA-Go's
// business, exactly as the CLR's FileStream is the BCL's.
//
// # It is a live handle, not a buffer
//
// Every method is one CNA call. Nothing is cached, because a second reader of
// the same file would then see a stale length -- and CNA is where the file
// actually is.
type storageStream struct {
	handle uint64
	closed bool
}

// The three CNA_SeekOrigin values, which are System.IO.SeekOrigin's and
// io.Seek's alike: Begin 0, Current 1, End 2. All three agree, so the whence a
// Go caller passes goes straight through.
const (
	seekOriginBegin   = 0
	seekOriginCurrent = 1
	seekOriginEnd     = 2
)

// errStorageStreamClosed is what a closed stream answers. io.Reader's contract
// asks for an error rather than a silent zero read.
var errStorageStreamClosed = errors.New("the storage stream is closed")

func newStorageStream(handle uint64) *storageStream {
	return &storageStream{handle: handle}
}

// Read is io.Reader over cna_storage_stream_read.
//
// The io.Reader contract requires io.EOF at the end of the file, and CNA
// reports the end as a zero-length read with no error -- so the translation is
// this projection's, and it is the one every Go consumer depends on.
func (s *storageStream) Read(destination []byte) (int, error) {
	if s == nil || s.closed {
		return 0, errStorageStreamClosed
	}
	if len(destination) == 0 {
		return 0, nil
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return 0, errStorageNoRuntime
	}
	read, err := runtime.StorageStreamRead(s.handle, destination)
	if err != nil {
		return read, err
	}
	if read == 0 {
		return 0, io.EOF
	}
	return read, nil
}

// Write is io.Writer over cna_storage_stream_write.
//
// CNA either writes every byte or fails, so a short write cannot be reported;
// io.Writer requires an error whenever n < len(p), and the only n this can
// return is len(p) or a failure.
func (s *storageStream) Write(data []byte) (int, error) {
	if s == nil || s.closed {
		return 0, errStorageStreamClosed
	}
	if len(data) == 0 {
		return 0, nil
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return 0, errStorageNoRuntime
	}
	if err := runtime.StorageStreamWrite(s.handle, data); err != nil {
		return 0, err
	}
	return len(data), nil
}

// Seek is io.Seeker over cna_storage_stream_seek. The three whence values are
// the same integers on both sides, which is why none is translated.
func (s *storageStream) Seek(offset int64, whence int) (int64, error) {
	if s == nil || s.closed {
		return 0, errStorageStreamClosed
	}
	switch whence {
	case io.SeekStart, io.SeekCurrent, io.SeekEnd:
	default:
		return 0, storageOutOfRangeError("whence")
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return 0, errStorageNoRuntime
	}
	return runtime.StorageStreamSeek(s.handle, offset, uint32(whence))
}

// Close is io.Closer over cna_storage_stream_close, which CNA documents as
// idempotent -- so a second Close is a no-op here too rather than a refusal.
func (s *storageStream) Close() error {
	if s == nil || s.closed {
		return nil
	}
	s.closed = true
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil
	}
	return runtime.StorageStreamClose(s.handle)
}

// The remaining stream routes, reached by a consumer through the io interfaces
// above or not at all. They are kept as methods rather than dropped because
// each has a call site: Length backs the seek-to-end a Go caller writes as
// Seek(0, io.SeekEnd), and the three capability reads are what a stress slice
// asserts against the mode it opened with.

func (s *storageStream) length() (int64, error) {
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return 0, errStorageNoRuntime
	}
	return runtime.StorageStreamLength(s.handle)
}

func (s *storageStream) position() (int64, error) {
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return 0, errStorageNoRuntime
	}
	return runtime.StorageStreamPosition(s.handle)
}

func (s *storageStream) setLength(length int64) error {
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return errStorageNoRuntime
	}
	return runtime.StorageStreamSetLength(s.handle, length)
}

func (s *storageStream) flush() error {
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return errStorageNoRuntime
	}
	return runtime.StorageStreamFlush(s.handle)
}

func (s *storageStream) canRead() (bool, error) {
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return false, errStorageNoRuntime
	}
	return runtime.StorageStreamCanRead(s.handle)
}

func (s *storageStream) canWrite() (bool, error) {
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return false, errStorageNoRuntime
	}
	return runtime.StorageStreamCanWrite(s.handle)
}

func (s *storageStream) canSeek() (bool, error) {
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return false, errStorageNoRuntime
	}
	return runtime.StorageStreamCanSeek(s.handle)
}
