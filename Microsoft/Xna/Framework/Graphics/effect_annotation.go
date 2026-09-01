package graphics

import (
	"errors"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 72 — EffectAnnotation and EffectAnnotationCollection.
// ---------------------------------------------------------------------------
//
// # The one architectural fact the whole cluster rests on
//
// CNA hands out a FRESH owned view handle from every accessor. The Foundation
// 72 probe measured two `cna_effect_get_parameters` calls answering two
// different handles, and two `get_at(0)` calls answering two more. The
// reference's accessors are `ldfld` over collections its CONSTRUCTOR built, so
// they answer the same managed object forever.
//
// So every projected object in this cluster is built ONCE, eagerly, when its
// owner is built -- which is exactly what the reference does -- and holds the
// one view handle it was given. Nothing here calls a CNA accessor twice for one
// logical object, and no member allocates a handle.
//
// # Ownership
//
//	the effect      OWNED, cna_effect_destroy
//	every view      OWNED, its own cna_effect_*_destroy
//
// A view OUTLIVES its effect: the probe destroyed an effect with views alive
// and the views still answered `get_count`. So there is no destruction order
// between an effect and its graph, and views register under the effect's own
// parent rather than under the effect.

// errEffectNil is the Go-only guard for a zero value in this cluster.
var errEffectNil = errors.New("effect object is nil or uninitialized")

// errInvalidCast projects System.InvalidCastException, which EffectParameter's
// value getters throw with NO message: every one of the 49 throw sites in the
// reference is `newobj InvalidCastException::.ctor()`.
var errInvalidCast = errors.New("the effect value cannot be read as this type")

// EffectAnnotation is Microsoft.Xna.Framework.Graphics.EffectAnnotation:
//
//	.class public auto ansi sealed beforefieldinit EffectAnnotation
//	       extends [mscorlib]System.Object
//
// It extends System.Object, is sealed, and has no public constructor: its
// `.ctor` is `assembly`, so a consumer only ever reaches one through a
// collection.
//
// # Six field reads and eight value getters
//
// Name, Semantic, RowCount, ColumnCount, ParameterClass and ParameterType are
// `ldfld` over fields the constructor filled from D3DX reflection. The eight
// GetValue* methods build a TEMPORARY EffectParameter over the same handle and
// delegate to it, inside a try/finally, so they carry that parameter's
// InvalidCastException.
type EffectAnnotation struct {
	view           *interop.EffectView
	name           string
	semantic       string
	rowCount       int32
	columnCount    int32
	parameterClass EffectParameterClass
	parameterType  EffectParameterType
}

// newEffectAnnotation reads the immutable metadata once, which is what the
// reference's constructor does.
func newEffectAnnotation(view *interop.EffectView) (*EffectAnnotation, error) {
	metadata, err := view.AnnotationMetadata()
	if err != nil {
		return nil, err
	}
	name, err := view.String(interop.EffectStringAnnotationName)
	if err != nil {
		return nil, err
	}
	semantic, err := view.String(interop.EffectStringAnnotationSemantic)
	if err != nil {
		return nil, err
	}
	return &EffectAnnotation{
		view:           view,
		name:           name,
		semantic:       semantic,
		rowCount:       metadata.RowCount,
		columnCount:    metadata.ColumnCount,
		parameterClass: EffectParameterClass(metadata.ParameterClass),
		parameterType:  EffectParameterType(metadata.ParameterType),
	}, nil
}

// Name is EffectAnnotation::get_Name, one `ldfld`.
func (a *EffectAnnotation) Name() string {
	if a == nil {
		return ""
	}
	return a.name
}

// Semantic is EffectAnnotation::get_Semantic, one `ldfld`.
func (a *EffectAnnotation) Semantic() string {
	if a == nil {
		return ""
	}
	return a.semantic
}

// RowCount is EffectAnnotation::get_RowCount, one `ldfld`.
func (a *EffectAnnotation) RowCount() int32 {
	if a == nil {
		return 0
	}
	return a.rowCount
}

// ColumnCount is EffectAnnotation::get_ColumnCount, one `ldfld`.
func (a *EffectAnnotation) ColumnCount() int32 {
	if a == nil {
		return 0
	}
	return a.columnCount
}

// ParameterClass is EffectAnnotation::get_ParameterClass, one `ldfld`.
func (a *EffectAnnotation) ParameterClass() EffectParameterClass {
	if a == nil {
		return 0
	}
	return a.parameterClass
}

// ParameterType is EffectAnnotation::get_ParameterType, one `ldfld`.
func (a *EffectAnnotation) ParameterType() EffectParameterType {
	if a == nil {
		return 0
	}
	return a.parameterType
}

// GetValueBoolean is EffectAnnotation::GetValueBoolean, which builds a
// temporary EffectParameter over the annotation's own handle and calls that
// type's GetValueBoolean -- so it carries the same InvalidCastException.
func (a *EffectAnnotation) GetValueBoolean() (bool, error) {
	if a == nil {
		return false, errEffectNil
	}
	return a.view.AnnotationBoolean()
}

// GetValueInt32 is EffectAnnotation::GetValueInt32.
func (a *EffectAnnotation) GetValueInt32() (int32, error) {
	if a == nil {
		return 0, errEffectNil
	}
	return a.view.AnnotationInt32()
}

// GetValueSingle is EffectAnnotation::GetValueSingle.
func (a *EffectAnnotation) GetValueSingle() (float32, error) {
	if a == nil {
		return 0, errEffectNil
	}
	return a.view.AnnotationSingle()
}

// GetValueString is EffectAnnotation::GetValueString.
func (a *EffectAnnotation) GetValueString() (string, error) {
	if a == nil {
		return "", errEffectNil
	}
	return a.view.String(interop.EffectStringAnnotationValue)
}

// GetValueVector2 is EffectAnnotation::GetValueVector2.
func (a *EffectAnnotation) GetValueVector2() (framework.Vector2, error) {
	if a == nil {
		return framework.Vector2{}, errEffectNil
	}
	values, err := a.view.AnnotationVector(2)
	if err != nil {
		return framework.Vector2{}, err
	}
	return framework.NewVector2BySingleAndSingle(values[0], values[1]), nil
}

// GetValueVector3 is EffectAnnotation::GetValueVector3.
func (a *EffectAnnotation) GetValueVector3() (framework.Vector3, error) {
	if a == nil {
		return framework.Vector3{}, errEffectNil
	}
	values, err := a.view.AnnotationVector(3)
	if err != nil {
		return framework.Vector3{}, err
	}
	return framework.NewVector3BySingleAndSingleAndSingle(values[0], values[1], values[2]), nil
}

// GetValueVector4 is EffectAnnotation::GetValueVector4.
func (a *EffectAnnotation) GetValueVector4() (framework.Vector4, error) {
	if a == nil {
		return framework.Vector4{}, errEffectNil
	}
	values, err := a.view.AnnotationVector(4)
	if err != nil {
		return framework.Vector4{}, err
	}
	return framework.NewVector4BySingleAndSingleAndSingleAndSingle(values[0], values[1], values[2], values[3]), nil
}

// GetValueMatrix is EffectAnnotation::GetValueMatrix. CNA_Matrix is sixteen
// floats in ROW-MAJOR order, which is the order Matrix's own fields take.
func (a *EffectAnnotation) GetValueMatrix() (framework.Matrix, error) {
	if a == nil {
		return framework.Matrix{}, errEffectNil
	}
	values, err := a.view.AnnotationMatrix()
	if err != nil {
		return framework.Matrix{}, err
	}
	return matrixFromRowMajor(values), nil
}

// matrixFromRowMajor is the one place CNA_Matrix's element order is stated.
// Both sides are row-major, so the mapping is positional -- and it is written
// out rather than memcpy'd so a change on either side is a compile error rather
// than a silently transposed transform.
func matrixFromRowMajor(values [16]float32) framework.Matrix {
	return framework.NewMatrix(
		values[0], values[1], values[2], values[3],
		values[4], values[5], values[6], values[7],
		values[8], values[9], values[10], values[11],
		values[12], values[13], values[14], values[15])
}

// EffectAnnotationCollection is
// Microsoft.Xna.Framework.Graphics.EffectAnnotationCollection.
//
// # Both indexers answer NULL rather than throwing
//
//	get_Item(int32)   index < 0 || index >= Count  ->  ldnull; ret
//	get_Item(string)  no element whose _name matches -> ldnull; ret
//
// That is the reference's own behaviour and it is not what a reader would
// guess: an out-of-range index on a BCL List<T> throws, and this collection
// checks the range itself first and answers null instead.
//
// The string indexer compares with `String.op_Equality`, which is ORDINAL and
// case-SENSITIVE -- unlike EffectParameterCollection::GetParameterBySemantic,
// which compares with StringComparison.OrdinalIgnoreCase. The two live three
// hundred lines apart in the same assembly and disagree, so each is projected
// as it is.
type EffectAnnotationCollection struct {
	view        *interop.EffectView
	annotations []*EffectAnnotation
}

// newEffectAnnotationCollection materialises every element once, which is what
// the reference's constructor does: it walks the native annotation count and
// builds a List<EffectAnnotation> before anyone can read it.
func newEffectAnnotationCollection(view *interop.EffectView) (*EffectAnnotationCollection, error) {
	count, err := view.Count(interop.EffectCollectionAnnotation)
	if err != nil {
		return nil, err
	}
	collection := &EffectAnnotationCollection{view: view}
	for index := uint64(0); index < count; index++ {
		element, err := view.At(interop.EffectCollectionAnnotation, index, interop.EffectViewAnnotation)
		if err != nil {
			return nil, err
		}
		annotation, err := newEffectAnnotation(element)
		if err != nil {
			return nil, err
		}
		collection.annotations = append(collection.annotations, annotation)
	}
	return collection, nil
}

// Count is EffectAnnotationCollection::get_Count, one forwarded List<T>.Count.
func (c *EffectAnnotationCollection) Count() int32 {
	if c == nil {
		return 0
	}
	return int32(len(c.annotations))
}

// ItemPropertySignature771818B0 is EffectAnnotationCollection::get_Item(Int32).
// The hashed name is the settled overload-collision rule: the type declares TWO
// indexers with one CLR name, so neither can take it. An index outside
// the collection answers nil, which is the reference's own `ldnull`.
func (c *EffectAnnotationCollection) ItemPropertySignature771818B0(index int32) *EffectAnnotation {
	if c == nil || index < 0 || int(index) >= len(c.annotations) {
		return nil
	}
	return c.annotations[index]
}

// ItemPropertySignatureFA6B0951 is EffectAnnotationCollection::get_Item(String),
// a linear scan
// with ORDINAL equality that answers nil when nothing matches.
func (c *EffectAnnotationCollection) ItemPropertySignatureFA6B0951(name string) *EffectAnnotation {
	if c == nil {
		return nil
	}
	for _, annotation := range c.annotations {
		if annotation.name == name {
			return annotation
		}
	}
	return nil
}

// GetEnumerator is EffectAnnotationCollection::GetEnumerator, which returns
// List<EffectAnnotation>.Enumerator -- projected as the settled Iterator[T].
func (c *EffectAnnotationCollection) GetEnumerator() framework.Iterator[*EffectAnnotation] {
	if c == nil {
		return &effectIterator[*EffectAnnotation]{}
	}
	return &effectIterator[*EffectAnnotation]{items: c.annotations}
}

// dispose releases every view this collection owns. It is unexported: the
// reference's collections are not IDisposable and a consumer never releases
// one; the effect's own Dispose walks its graph.
func (c *EffectAnnotationCollection) dispose() error {
	if c == nil {
		return nil
	}
	var failures []error
	for _, annotation := range c.annotations {
		failures = append(failures, annotation.view.Dispose())
	}
	failures = append(failures, c.view.Dispose())
	return errors.Join(failures...)
}

// effectIterator is the Go language adapter for the List<T>.Enumerator all
// four collections in this cluster hand out. One generic type serves all four
// because their enumerators differ only in element type, and none of the four
// lists is ever mutated after its collection is built -- the reference builds
// them in a constructor and CNA reports them once -- so there is no version
// check to reproduce and no enumeration failure to report.
type effectIterator[T any] struct {
	items []T
	at    int
}

func (i *effectIterator[T]) Next() (T, bool, error) {
	var zero T
	if i == nil || i.at >= len(i.items) {
		return zero, false, nil
	}
	item := i.items[i.at]
	i.at++
	return item, true, nil
}
