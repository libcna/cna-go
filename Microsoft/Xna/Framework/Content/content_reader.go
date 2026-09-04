package content

import (
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// ContentReader is Microsoft.Xna.Framework.Content.ContentReader:
//
//	.class public auto ansi sealed beforefieldinit ContentReader
//	       extends [mscorlib]System.IO.BinaryReader
//
// The reader a content type reader is handed: it knows the asset being loaded
// and how to pull XNA's value types out of the compiled stream.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # It has NO public constructor, exactly like the Model family
//
// The pinned contract declares none. A ContentReader reaches a consumer only
// from the content pipeline, which is why the projection's factory is
// unexported: what a consumer receives is a reader over an asset, never one it
// built.
//
// # Its base is DEFERRED, and half of why is now known to be wrong
//
// `System.IO.BinaryReader` has been a deferred BCL base since Foundation 29,
// partly because "the reader's inherited surface depends on seeking". Foundation
// 91 measured that: ContentReader's whole IL contains ZERO `Stream::Seek` calls,
// and the one apparent hit was the substring of `get_CanSeek` inside a
// `private static` helper whose check is skipped when the stream cannot seek.
//
// `Read7BitEncodedInt` is the half that is real, and it is `protected` -- not
// public surface either way. So what the deferral costs today is the inherited
// READ members, which a type reader uses constantly. That remains open, and the
// contract's own eighteen declared members are what this projects.
type ContentReader struct {
	// base is the private System.IO.BinaryReader adapter, held by the settled
	// composition rule: an unexported field, never embedded and never returned.
	base   binaryReaderBase
	handle uint64
	// contentManager is the `contentManager` FIELD get_ContentManager reads.
	// CNA has a get_content_manager route and it is deliberately unbound: the
	// reference reads a field, so asking CNA would be a second answer to a
	// question the projection already holds -- the same judgement the graphics
	// resource name routes got.
	contentManager *ContentManager
	// disposed is the managed latch. Destroying the native reader twice is not
	// something CNA promises to tolerate.
	disposed bool
}

// newContentReader builds a reader over a storage stream. It is unexported
// because the reference's constructor is `assembly` and the contract declares
// none.
//
// The stream must be one this module produced -- a StorageContainer file --
// because CNA's reader takes a CNA_StorageStreamHandle and only Storage makes
// one. A consumer's own io.Reader has no handle, and answering that plainly is
// better than handing CNA a zero that looks valid.
func newContentReader(streamHandle uint64, manager *ContentManager, managerHandle uint64, assetName string, version int32, platform uint8) (*ContentReader, error) {
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil, errContentNoRuntime
	}
	handle, err := runtime.ContentReaderCreate(interop.ContentReaderCreateValues{
		ContentManager: managerHandle,
		Stream:         streamHandle,
		AssetName:      assetName,
		Version:        version,
		Platform:       platform,
	})
	if err != nil {
		return nil, err
	}
	reader := &ContentReader{handle: handle, contentManager: manager}
	reader.base.owner = reader
	return reader, nil
}

// errContentNoRuntime is the Go-only refusal a member answers with no runtime
// loaded. The reference reads a stream directly.
var errContentNoRuntime = errContentReaderNoRuntime()

func errContentReaderNoRuntime() error {
	return fmt.Errorf("%w: this member needs a loaded runtime", errContentLoad)
}

// AssetName is ContentReader::get_AssetName, the logical name of the asset
// being read. It is what every ContentLoadException this family raises names.
//
// The reference reads a field; this reaches CNA, because the name is copied
// into the native reader at creation and the projection keeps one copy rather
// than two that could disagree. Reaching CNA is what makes it fallible.
func (r *ContentReader) AssetName() (string, error) {
	if err := r.usable(); err != nil {
		return "", err
	}
	runtime, _ := interop.CurrentRuntime()
	return runtime.ContentReaderAssetName(r.handle)
}

// ContentManager is ContentReader::get_ContentManager, one `ldfld`: the manager
// that started the load, or nil for a reader created standalone.
//
// It carries an error only because a closed reader must refuse -- the field
// itself cannot fail, and CNA is never asked.
func (r *ContentReader) ContentManager() (*ContentManager, error) {
	if r == nil || r.disposed {
		return nil, fmt.Errorf("%w: the content reader is closed", errContentLoad)
	}
	return r.contentManager, nil
}

// ReadVector2 is ContentReader::ReadVector2().
func (r *ContentReader) ReadVector2() (framework.Vector2, error) {
	values, err := r.readFloats(2)
	if err != nil {
		return framework.Vector2{}, err
	}
	return framework.NewVector2BySingleAndSingle(values[0], values[1]), nil
}

// ReadVector3 is ContentReader::ReadVector3().
func (r *ContentReader) ReadVector3() (framework.Vector3, error) {
	values, err := r.readFloats(3)
	if err != nil {
		return framework.Vector3{}, err
	}
	return framework.NewVector3BySingleAndSingleAndSingle(values[0], values[1], values[2]), nil
}

// ReadVector4 is ContentReader::ReadVector4().
func (r *ContentReader) ReadVector4() (framework.Vector4, error) {
	values, err := r.readFloats(4)
	if err != nil {
		return framework.Vector4{}, err
	}
	return framework.NewVector4BySingleAndSingleAndSingleAndSingle(
		values[0], values[1], values[2], values[3]), nil
}

// ReadQuaternion is ContentReader::ReadQuaternion().
func (r *ContentReader) ReadQuaternion() (framework.Quaternion, error) {
	values, err := r.readFloats(5)
	if err != nil {
		return framework.Quaternion{}, err
	}
	return framework.NewQuaternionBySingleAndSingleAndSingleAndSingle(
		values[0], values[1], values[2], values[3]), nil
}

// ReadMatrix is ContentReader::ReadMatrix(), which reads sixteen floats in the
// reference's own row-major order.
func (r *ContentReader) ReadMatrix() (framework.Matrix, error) {
	values, err := r.readFloats(16)
	if err != nil {
		return framework.Matrix{}, err
	}
	return framework.NewMatrix(
		values[0], values[1], values[2], values[3],
		values[4], values[5], values[6], values[7],
		values[8], values[9], values[10], values[11],
		values[12], values[13], values[14], values[15]), nil
}

// ReadColor is ContentReader::ReadColor().
//
// CNA hands back four separate channel bytes rather than a packed word, so the
// projection builds the Color from them the way XNA's own constructor does.
func (r *ContentReader) ReadColor() (framework.Color, error) {
	if err := r.usable(); err != nil {
		return framework.Color{}, err
	}
	runtime, _ := interop.CurrentRuntime()
	channels, err := runtime.ContentReaderReadColor(r.handle)
	if err != nil {
		return framework.Color{}, err
	}
	return framework.NewColorByInt32AndInt32AndInt32AndInt32(
		int32(channels[0]), int32(channels[1]), int32(channels[2]), int32(channels[3])), nil
}

// readFloats is the shared body of the five float-valued reads.
func (r *ContentReader) readFloats(kind int) ([]float32, error) {
	if err := r.usable(); err != nil {
		return nil, err
	}
	runtime, _ := interop.CurrentRuntime()
	return runtime.ContentReaderReadFloats(r.handle, kind)
}

// readExact reads exactly count bytes, which is what CNA's read_bytes_exact
// promises: a short read is a failure rather than a partial answer.
func (r *ContentReader) readExact(count int32, readerName string) ([]byte, error) {
	if err := r.usable(); err != nil {
		return nil, err
	}
	runtime, _ := interop.CurrentRuntime()
	buffer := make([]byte, count)
	read, err := runtime.ContentReaderReadBytesExact(r.handle, count, readerName, buffer)
	if err != nil {
		return nil, err
	}
	if read != int(count) {
		name, _ := runtime.ContentReaderAssetName(r.handle)
		return nil, fmt.Errorf("%w: %s", errContentLoad, fmt.Sprintf(badXnbSize, name))
	}
	return buffer, nil
}

// usable is the guard every read shares.
func (r *ContentReader) usable() error {
	if r == nil || r.disposed {
		return fmt.Errorf("%w: the content reader is closed", errContentLoad)
	}
	if _, ok := interop.CurrentRuntime(); !ok {
		return errContentNoRuntime
	}
	return nil
}

// readObjectTag reads the type-reader index ReadObject consumes before it
// dispatches. It is unexported because the reference reads it from inside
// ReadObject rather than exposing it.
func (r *ContentReader) readObjectTag() (bool, error) {
	if err := r.usable(); err != nil {
		return false, err
	}
	runtime, _ := interop.CurrentRuntime()
	return runtime.ContentReaderReadObjectTag(r.handle)
}

// initializeTypeReaders walks the asset's type manifest. It is unexported
// because the reference does it from its `assembly` construction path, not from
// anything the contract declares.
func (r *ContentReader) initializeTypeReaders() error {
	if err := r.usable(); err != nil {
		return err
	}
	runtime, _ := interop.CurrentRuntime()
	return runtime.ContentReaderInitializeTypeReaders(r.handle)
}

// close releases the native reader. CNA's reader closes the stream it borrowed,
// which is why the stream's own Close stays idempotent.
func (r *ContentReader) close() error {
	if r == nil || r.disposed {
		return nil
	}
	r.disposed = true
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil
	}
	return runtime.ContentReaderDestroy(r.handle)
}
