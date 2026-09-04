package content

import (
	"fmt"
	"reflect"
)

// ContentTypeReader is Microsoft.Xna.Framework.Content.ContentTypeReader:
//
//	.class public abstract auto ansi beforefieldinit ContentTypeReader
//	       extends [mscorlib]System.Object
//
// The base every content type reader derives from: it announces which CLR type
// it deserializes, and the pipeline asks it to read one.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # Its substitutability is LIVE, which decided the shape
//
// `ContentReader::ReadObject(ContentTypeReader)` is a base-typed PARAMETER
// position and `ContentTypeReader<T>` is a projected derived type, so the
// settled substitutable-base rule applies exactly as it does to Texture2D,
// Texture, Effect and TextureCube: the position takes an exported INTERFACE
// with an unexported method, nameable by a consumer and satisfiable only inside
// this module.
//
// The first design here was different and wrong. It projected the type the way
// Game is projected -- a host struct plus a consumer-supplied callbacks
// interface -- on the assumption that writing a custom type reader is the
// point. The verifier rejected it, and the verifier is right about the
// consequence: in THIS binding the type-reader dispatch is CNA's
// (`cna_content_reader_initialize_type_readers` walks the manifest), so the
// readers that exist are the ones CNA knows. A consumer cannot add one, and
// saying so plainly is better than exporting a hook nothing would call.
//
// That is a real limitation and it is recorded rather than hidden: XNA lets a
// game ship a reader for its own content, and this binding does not, because
// the reader is instantiated from a type NAME by reflection the CLR has and Go
// does not.
type ContentTypeReader struct {
	// targetType is the private `targetType` field get_TargetType answers.
	targetType reflect.Type
	// targetIsValueType is the `assembly initonly` TargetIsValueType field. The
	// constructor sets it from targetType.IsValueType and ONLY when the type is
	// non-null, so a reader built over a nil type leaves it false rather than
	// failing.
	targetIsValueType bool
	// typeVersion and canDeserializeIntoExistingObject are what the two public
	// virtuals answer. Both base bodies are `ldc.i4.0; ret`, so both default to
	// the zero value and a CNA-supplied reader may say otherwise.
	typeVersion                      int32
	canDeserializeIntoExistingObject bool
}

// ContentTypeReaderReference is the exported interface every
// ContentTypeReader-typed position takes.
//
// It carries an unexported method, so only this module can satisfy it -- which
// is the settled substitutable-base shape and, here, also the honest one: the
// readers that exist are the ones CNA's manifest walk produced.
type ContentTypeReaderReference interface {
	// TargetType is ContentTypeReader::get_TargetType.
	TargetType() reflect.Type
	// TypeVersion is ContentTypeReader::get_TypeVersion.
	TypeVersion() int32
	// CanDeserializeIntoExistingObject is the getter of the same name.
	CanDeserializeIntoExistingObject() bool
	// Initialize is ContentTypeReader::Initialize.
	Initialize(manager *ContentTypeReaderManager) error
	// Read is ContentTypeReader::Read.
	Read(input *ContentReader, existingInstance any) (any, error)
	// contentTypeReader keeps the interface unsatisfiable from outside this
	// module, which is what makes the substitutable-base rule safe.
	contentTypeReader() *ContentTypeReader
}

// contentArgumentNullError wraps the package's existing
// System.ArgumentNullException sentinel with a parameter name.
func contentArgumentNullError(parameter string) error {
	return fmt.Errorf("%w: %s", errContentArgumentNull, parameter)
}

// NewContentTypeReader is ContentTypeReader::.ctor(Type), which the reference
// declares `family`. Its body is exactly:
//
//	this.targetType = targetType;
//	if (targetType != null)
//	    this.TargetIsValueType = targetType.IsValueType;
//
// A NULL target type is accepted and leaves TargetIsValueType false. That is
// measured, not assumed: the `op_Inequality` test is why the assignment is
// guarded at all.
func NewContentTypeReader(targetType reflect.Type) *ContentTypeReader {
	reader := &ContentTypeReader{targetType: targetType}
	if targetType != nil {
		reader.targetIsValueType = isValueType(targetType)
	}
	return reader
}

// isValueType projects System.Type::get_IsValueType onto Go's type system.
//
// A CLR value type is one that is not a reference: a struct, an enum or a
// primitive. Go's reference kinds are the pointer-shaped ones plus the
// interface, so everything else stands for a CLR value type. The project's own
// mapping is what makes the correspondence exact -- a CLR class projects to a
// Go POINTER and a CLR struct to a Go value.
func isValueType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map,
		reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return false
	default:
		return true
	}
}

// TargetType is ContentTypeReader::get_TargetType, one `ldfld`.
func (r *ContentTypeReader) TargetType() reflect.Type {
	if r == nil {
		return nil
	}
	return r.targetType
}

// TypeVersion is ContentTypeReader::get_TypeVersion, a public virtual whose
// base body is `ldc.i4.0; ret`.
func (r *ContentTypeReader) TypeVersion() int32 {
	if r == nil {
		return 0
	}
	return r.typeVersion
}

// CanDeserializeIntoExistingObject is the getter of the same name, whose base
// body is also `ldc.i4.0; ret` -- so the default is FALSE, and a reader that
// can fill an existing instance is the exception.
func (r *ContentTypeReader) CanDeserializeIntoExistingObject() bool {
	if r == nil {
		return false
	}
	return r.canDeserializeIntoExistingObject
}

// Initialize is ContentTypeReader::Initialize(ContentTypeReaderManager), whose
// base body is ONE `ret` -- it does nothing at all. A reader that needs to
// resolve the readers for its nested types is where the reference overrides it,
// and CNA performs that resolution during its own manifest walk.
func (r *ContentTypeReader) Initialize(manager *ContentTypeReaderManager) error {
	return nil
}

// Read is ContentTypeReader::Read(ContentReader, object), which the reference
// declares abstract.
//
// A bare ContentTypeReader has no body to run: `abstract` means the reference
// has none either, and CNA-Go cannot instantiate one from a type name. So this
// refuses rather than inventing an answer, and the refusal names the type the
// reader was built for.
func (r *ContentTypeReader) Read(input *ContentReader, existingInstance any) (any, error) {
	target := "an unnamed type"
	if r != nil && r.targetType != nil {
		target = r.targetType.String()
	}
	return nil, fmt.Errorf("%w: no content type reader is available for %s", errContentLoad, target)
}

// contentTypeReader satisfies ContentTypeReaderReference.
func (r *ContentTypeReader) contentTypeReader() *ContentTypeReader { return r }

// targetIsValue answers the `assembly` TargetIsValueType field for the rest of
// the package. It is not projected: the field is `assembly` in the reference,
// so no consumer can read it there either.
func (r *ContentTypeReader) targetIsValue() bool { return r != nil && r.targetIsValueType }
