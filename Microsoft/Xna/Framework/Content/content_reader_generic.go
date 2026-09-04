package content

import "fmt"

// ContentReader's ten GENERIC members.
//
// Go cannot declare a method with its own type parameters, so the settled rule
// projects a CLR generic method as a package-level FUNCTION whose first
// argument is the receiver. That is the same rule ContentManagerLoad already
// obeys.
//
// # None of them can dispatch, and the reason is measured
//
// Every one of these ends in a type reader: the reference looks up the reader
// for T, or is handed one, and calls its Read. CNA-Go can do the first half --
// `cna_content_reader_read_object_tag` reads the tag that says which reader the
// asset names -- and cannot do the second, because the readers themselves are
// instantiated from a type NAME by reflection the CLR has and Go does not.
//
// So each refuses, and the refusal says which type had no reader. That is the
// same limitation ContentTypeReader records, surfacing where a caller meets it.

// ContentReaderReadObjectByNone is ContentReader::ReadObject<T>().
func ContentReaderReadObjectByNone[T any](reader *ContentReader) (T, error) {
	return contentReaderNoTypeReader[T](reader, "ReadObject")
}

// ContentReaderReadObjectByT is ContentReader::ReadObject<T>(T), which fills an
// existing instance when its reader says it can.
func ContentReaderReadObjectByT[T any](reader *ContentReader, existingInstance T) (T, error) {
	return contentReaderNoTypeReader[T](reader, "ReadObject")
}

// ContentReaderReadObjectByContentTypeReader is
// ContentReader::ReadObject<T>(ContentTypeReader), which uses the reader it is
// handed instead of looking one up.
func ContentReaderReadObjectByContentTypeReader[T any](reader *ContentReader, typeReader ContentTypeReaderReference) (T, error) {
	return contentReaderWithTypeReader[T](reader, typeReader, nil)
}

// ContentReaderReadObjectByContentTypeReaderAndT is the four-argument form.
func ContentReaderReadObjectByContentTypeReaderAndT[T any](reader *ContentReader, typeReader ContentTypeReaderReference, existingInstance T) (T, error) {
	return contentReaderWithTypeReader[T](reader, typeReader, existingInstance)
}

// ContentReaderReadRawObjectByNone is ContentReader::ReadRawObject<T>().
//
// The difference from ReadObject is the TAG: ReadObject reads a type-reader
// index first and ReadRawObject does not, because the caller already knows what
// is there. Both end in the same dispatch.
func ContentReaderReadRawObjectByNone[T any](reader *ContentReader) (T, error) {
	return contentReaderNoTypeReader[T](reader, "ReadRawObject")
}

// ContentReaderReadRawObjectByT is ContentReader::ReadRawObject<T>(T).
func ContentReaderReadRawObjectByT[T any](reader *ContentReader, existingInstance T) (T, error) {
	return contentReaderNoTypeReader[T](reader, "ReadRawObject")
}

// ContentReaderReadRawObjectByContentTypeReader is
// ContentReader::ReadRawObject<T>(ContentTypeReader).
func ContentReaderReadRawObjectByContentTypeReader[T any](reader *ContentReader, typeReader ContentTypeReaderReference) (T, error) {
	return contentReaderWithTypeReader[T](reader, typeReader, nil)
}

// ContentReaderReadRawObjectByContentTypeReaderAndT is the four-argument form.
func ContentReaderReadRawObjectByContentTypeReaderAndT[T any](reader *ContentReader, typeReader ContentTypeReaderReference, existingInstance T) (T, error) {
	return contentReaderWithTypeReader[T](reader, typeReader, existingInstance)
}

// ContentReaderReadSharedResource is
// ContentReader::ReadSharedResource<T>(Action<T>).
//
// A shared resource is written once and referenced many times, so the reference
// does not hand the value back: it records the caller's FIXUP and calls it once
// the resource has been read, which may be after this member returns. The
// Action<T> projects to a Go func for the reason every other CLR delegate does.
func ContentReaderReadSharedResource[T any](reader *ContentReader, fixup func(T)) error {
	if err := reader.usable(); err != nil {
		return err
	}
	if fixup == nil {
		return contentArgumentNullError("fixup")
	}
	return fmt.Errorf("%w: shared resources need a type reader, and this binding instantiates none", errContentLoad)
}

// ContentReaderReadExternalReference is
// ContentReader::ReadExternalReference<T>(), which loads a DIFFERENT asset
// through the same content manager and is why a reader created standalone
// cannot serve one.
func ContentReaderReadExternalReference[T any](reader *ContentReader) (T, error) {
	var zero T
	if err := reader.usable(); err != nil {
		return zero, err
	}
	if func() bool { m, err := reader.ContentManager(); return err != nil || m == nil }() {
		return zero, fmt.Errorf("%w: an external reference needs a content manager and this reader has none", errContentLoad)
	}
	return contentReaderNoTypeReader[T](reader, "ReadExternalReference")
}

// contentAssetName answers the asset name for a message, and answers empty
// rather than failing: a refusal that could not name its asset is still a
// refusal, and swallowing a second error inside the first would hide it.
func contentAssetName(reader *ContentReader) string {
	name, err := reader.AssetName()
	if err != nil {
		return ""
	}
	return name
}

// contentReaderNoTypeReader is the shared refusal. It names the TYPE, because
// which reader was missing is the only useful thing a caller can act on.
func contentReaderNoTypeReader[T any](reader *ContentReader, member string) (T, error) {
	var zero T
	if err := reader.usable(); err != nil {
		return zero, err
	}
	// The tag is still read, because the reference reads it before it
	// dispatches and skipping it would leave the stream at the wrong position
	// for whatever a caller does next.
	if member == "ReadObject" {
		if _, err := reader.readObjectTag(); err != nil {
			return zero, err
		}
	}
	return zero, fmt.Errorf("%w: %s found no content type reader for %T", errContentLoad, member, zero)
}

// contentReaderWithTypeReader is the pair that is HANDED a reader rather than
// looking one up. It still cannot complete, because the reader it is handed can
// only be one this module produced and this module produces none.
func contentReaderWithTypeReader[T any](reader *ContentReader, typeReader ContentTypeReaderReference, existingInstance any) (T, error) {
	var zero T
	if err := reader.usable(); err != nil {
		return zero, err
	}
	if typeReader == nil {
		return zero, contentArgumentNullError("typeReader")
	}
	value, err := typeReader.Read(reader, existingInstance)
	if err != nil {
		return zero, err
	}
	typed, ok := value.(T)
	if !ok {
		return zero, fmt.Errorf("%w: %s", errContentLoad,
			fmt.Sprintf(badXnbWrongType, contentAssetName(reader), value, zero))
	}
	return typed, nil
}
