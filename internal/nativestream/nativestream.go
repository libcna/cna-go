// SPDX-License-Identifier: MS-PL

// Package nativestream carries the one thing a projected Stream has that a
// projected member in another package needs: the CNA handle behind it.
//
// # Why not a method on the stream
//
// `StorageContainer.CreateFile` and its three `OpenFile` overloads return
// `System.IO.Stream`, which the settled rule projects as the standard-library
// Go interface whose role it is -- io.ReadWriteSeeker. The concrete type behind
// that interface is unexported, exactly as the CLR's FileStream is the BCL's
// business rather than XNA's.
//
// `ContentReader` then has to build a native reader OVER that stream:
// `cna_content_reader_create` takes a `CNA_StorageStreamHandle`, and the only
// routes in the ABI that produce one are Storage's. So the content package
// needs the handle, the storage package owns it, and the two are different
// packages.
//
// Three ways out were weighed and two were rejected:
//
//   - an EXPORTED accessor on the stream, or an exported interface a consumer
//     could name. The pinned contract declares no way to get a native handle
//     out of a System.IO.Stream, so this would be public surface XNA does not
//     have. Rejected.
//   - duplicating the handle in the content package, which would give one
//     stream two owners and two lifetimes. Rejected.
//   - this: an `internal` package both can import and no consumer of the module
//     can. It is the same shape internal/bclhash and internal/dispatcher
//     already have, and for the same reason -- Go cannot share an unexported
//     symbol across packages.
//
// # It is a lookup, not an owner
//
// Registering a stream does NOT make this package responsible for closing it.
// The stream's owner still closes it, and Forget is what an owner calls when it
// does, so a handle cannot outlive the object it belongs to.
package nativestream

import "sync"

var (
	mutex   sync.Mutex
	handles = map[any]uint64{}
)

// Register records the CNA handle behind one projected stream. The key is the
// stream object itself, so two streams over the same file stay distinct.
func Register(stream any, handle uint64) {
	if stream == nil {
		return
	}
	mutex.Lock()
	defer mutex.Unlock()
	handles[stream] = handle
}

// Forget drops a stream's entry. Its owner calls this when it closes the
// stream, so a handle never outlives what it names.
func Forget(stream any) {
	if stream == nil {
		return
	}
	mutex.Lock()
	defer mutex.Unlock()
	delete(handles, stream)
}

// HandleOf answers the CNA handle behind a projected stream, and whether there
// is one. A stream this module did not create -- a consumer's own io.Reader
// handed to a member that takes one -- has no handle, and answering false is
// how a caller learns that rather than by receiving a zero that looks valid.
func HandleOf(stream any) (uint64, bool) {
	if stream == nil {
		return 0, false
	}
	mutex.Lock()
	defer mutex.Unlock()
	handle, present := handles[stream]
	return handle, present
}
