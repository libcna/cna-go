package content

import (
	"fmt"
	"reflect"
)

// ContentTypeReaderOfT is Microsoft.Xna.Framework.Content.ContentTypeReader`1:
//
//	.class public abstract auto ansi beforefieldinit ContentTypeReader`1<T>
//	       extends Microsoft.Xna.Framework.Content.ContentTypeReader
//
// The typed base almost every real reader derives from. It supplies the target
// type from its own type argument and does the cast the untyped Read would
// otherwise leave to the reader.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # The Go name spells out the type parameter
//
// `ContentTypeReader`1` and the non-generic `ContentTypeReader` are two
// different CLR types with the same source name, and Go has no way to overload
// a type name. The settled rule -- the one that produced
// IPackedVectorOfTPacked for IPackedVector`1 -- appends `Of` and the parameter
// name, so this is ContentTypeReaderOfT.
//
// # Its Read is the type check, and it is the only behaviour here
//
//	object Read(ContentReader input, object existingInstance)
//	{
//	    T typed;
//	    if (existingInstance == null) typed = default(T);
//	    else {
//	        if (!(existingInstance is T))
//	            throw CreateContentLoadException(BadXnbWrongType, null,
//	                                             input.AssetName,
//	                                             typeof(T), existingInstance.GetType());
//	        typed = (T)existingInstance;
//	    }
//	    return Read(input, typed);
//	}
//
// So a null existing instance becomes `default(T)` -- the zero value, not a
// refusal -- and a WRONG one is the error. The message names both types, which
// is why the projection carries the reflect.Type of each rather than a string.
type ContentTypeReaderOfT[T any] struct {
	base *ContentTypeReader
}

// NewContentTypeReaderOfT is ContentTypeReader`1::.ctor(), whose whole body is
//
//	base..ctor(typeof(T))
//
// so the target type comes from the type argument and a consumer never names
// it twice.
func NewContentTypeReaderOfT[T any]() *ContentTypeReaderOfT[T] {
	// `typeof(T)`. A Go type parameter has no token, so the type is taken from
	// the zero value -- which is exactly what typeof(T) names.
	var zero T
	return &ContentTypeReaderOfT[T]{base: NewContentTypeReader(reflect.TypeOf(&zero).Elem())}
}

// TargetType is the inherited ContentTypeReader::get_TargetType.
func (r *ContentTypeReaderOfT[T]) TargetType() reflect.Type { return r.base.TargetType() }

// TypeVersion is the inherited ContentTypeReader::get_TypeVersion.
func (r *ContentTypeReaderOfT[T]) TypeVersion() int32 { return r.base.TypeVersion() }

// CanDeserializeIntoExistingObject is the inherited getter.
func (r *ContentTypeReaderOfT[T]) CanDeserializeIntoExistingObject() bool {
	return r.base.CanDeserializeIntoExistingObject()
}

// initialize is ContentTypeReader`1::Initialize, which the reference overrides
// only to forward to the base -- `ldarg.0; ldarg.1; call base::Initialize`.
//
// It is unexported because the pinned contract does not carry it on the GENERIC
// type: the inherited one is what a caller reaches, and an exported override
// that only forwards would be a second spelling of one member.
func (r *ContentTypeReaderOfT[T]) initialize(manager *ContentTypeReaderManager) error {
	return r.base.Initialize(manager)
}

// ReadByContentReaderAndObject is ContentTypeReader`1::Read(ContentReader,
// Object), the override that casts and then calls the typed member.
func (r *ContentTypeReaderOfT[T]) ReadByContentReaderAndObject(input *ContentReader, existingInstance any) (any, error) {
	var typed T
	if existingInstance != nil {
		converted, ok := existingInstance.(T)
		if !ok {
			return nil, r.wrongTypeError(input, existingInstance)
		}
		typed = converted
	}
	return r.base.Read(input, typed)
}

// ReadByContentReaderAnd0 is ContentTypeReader`1's typed abstract member.
// The `0` in the name is the CLR parameter shape -- `!0` is the type parameter,
// which is what the overload rule spells rather than the Go letter T. Like
// the untyped one it has no body to run -- `abstract` means the reference has
// none either -- so it refuses and names the type it was built for.
func (r *ContentTypeReaderOfT[T]) ReadByContentReaderAnd0(input *ContentReader, existingInstance T) (T, error) {
	var zero T
	_, err := r.base.Read(input, nil)
	return zero, err
}

// wrongTypeError builds the ContentLoadException the cast failure raises. Its
// message names the asset and BOTH types, in the reference's own order: what
// the file contains, then what the load asked for.
func (r *ContentTypeReaderOfT[T]) wrongTypeError(input *ContentReader, existingInstance any) error {
	assetName := contentAssetName(input)
	return fmt.Errorf("%w: %s", errContentLoad,
		fmt.Sprintf(badXnbWrongType, assetName,
			reflect.TypeOf(existingInstance), r.TargetType()))
}
