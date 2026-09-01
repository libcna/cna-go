package graphics

import (
	"errors"
	"unsafe"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 72 — EffectParameter and EffectParameterCollection.
// ---------------------------------------------------------------------------

// EffectParameter is Microsoft.Xna.Framework.Graphics.EffectParameter, the
// widest type in the cluster: 49 public members over one D3DX parameter handle.
//
// # Nine field reads, and forty members that reach a runtime
//
//	get_Name           get_Semantic       get_RowCount     get_ColumnCount
//	get_ParameterClass get_ParameterType  get_Elements     get_StructureMembers
//	get_Annotations
//
// are `ldfld` in the reference over state its constructor filled. Everything
// else -- eighteen SetValue overloads, two SetValueTranspose, and twenty-two
// GetValue* -- reaches D3DX and is fallible.
//
// # The InvalidCastException guard is reproduced MANAGED-SIDE
//
// Every value getter begins with the same shape:
//
//	if (_paramClass != <the class this overload reads> &&
//	    pElementCollection.Count == 0)
//	    throw new InvalidCastException();
//
// a parameterless InvalidCastException -- all 49 throw sites in the type are
// `newobj InvalidCastException::.ctor()` with no message. The guard is
// reproduced here rather than left to CNA because the Foundation 72 probe
// measured CNA ACCEPTING a mismatched write: setting a Single on
// AlphaTestEffect's DiffuseColor, which is a four-column Vector parameter,
// returned success and read back zero. A projection that forwarded would turn
// the reference's refusal into a silent no-op.
type EffectParameter struct {
	view             *interop.EffectView
	name             string
	semantic         string
	rowCount         int32
	columnCount      int32
	parameterClass   EffectParameterClass
	parameterType    EffectParameterType
	elements         *EffectParameterCollection
	structureMembers *EffectParameterCollection
	annotations      *EffectAnnotationCollection
}

// newEffectParameter reads the metadata once and materialises the three nested
// collections, which is what the reference's constructor does.
//
// The recursion terminates on CNA's own data: a leaf parameter reports zero
// elements and zero structure members, which the probe confirmed for every one
// of AlphaTestEffect's six.
func newEffectParameter(view *interop.EffectView) (*EffectParameter, error) {
	metadata, err := view.ParameterMetadata()
	if err != nil {
		return nil, err
	}
	name, err := view.String(interop.EffectStringParameterName)
	if err != nil {
		return nil, err
	}
	semantic, err := view.String(interop.EffectStringParameterSemantic)
	if err != nil {
		return nil, err
	}
	parameter := &EffectParameter{
		view:           view,
		name:           name,
		semantic:       semantic,
		rowCount:       metadata.RowCount,
		columnCount:    metadata.ColumnCount,
		parameterClass: EffectParameterClass(metadata.ParameterClass),
		parameterType:  EffectParameterType(metadata.ParameterType),
	}
	elementView, err := view.Elements()
	if err != nil {
		return nil, err
	}
	if parameter.elements, err = newEffectParameterCollection(elementView); err != nil {
		return nil, err
	}
	memberView, err := view.StructureMembers()
	if err != nil {
		return nil, err
	}
	if parameter.structureMembers, err = newEffectParameterCollection(memberView); err != nil {
		return nil, err
	}
	annotationView, err := view.ParameterAnnotations()
	if err != nil {
		return nil, err
	}
	if parameter.annotations, err = newEffectAnnotationCollection(annotationView); err != nil {
		return nil, err
	}
	return parameter, nil
}

// The nine infallible readers.

// Name is EffectParameter::get_Name, one `ldfld`.
func (p *EffectParameter) Name() string {
	if p == nil {
		return ""
	}
	return p.name
}

// Semantic is EffectParameter::get_Semantic, one `ldfld`.
func (p *EffectParameter) Semantic() string {
	if p == nil {
		return ""
	}
	return p.semantic
}

// RowCount is EffectParameter::get_RowCount, one `ldfld`.
func (p *EffectParameter) RowCount() int32 {
	if p == nil {
		return 0
	}
	return p.rowCount
}

// ColumnCount is EffectParameter::get_ColumnCount, one `ldfld`.
func (p *EffectParameter) ColumnCount() int32 {
	if p == nil {
		return 0
	}
	return p.columnCount
}

// ParameterClass is EffectParameter::get_ParameterClass, one `ldfld`.
func (p *EffectParameter) ParameterClass() EffectParameterClass {
	if p == nil {
		return 0
	}
	return p.parameterClass
}

// ParameterType is EffectParameter::get_ParameterType, one `ldfld`.
func (p *EffectParameter) ParameterType() EffectParameterType {
	if p == nil {
		return 0
	}
	return p.parameterType
}

// Elements is EffectParameter::get_Elements, one `ldfld` over the collection
// the constructor built -- so it answers the SAME object every call.
func (p *EffectParameter) Elements() *EffectParameterCollection {
	if p == nil {
		return nil
	}
	return p.elements
}

// StructureMembers is EffectParameter::get_StructureMembers, the same shape.
func (p *EffectParameter) StructureMembers() *EffectParameterCollection {
	if p == nil {
		return nil
	}
	return p.structureMembers
}

// Annotations is EffectParameter::get_Annotations, the same shape.
func (p *EffectParameter) Annotations() *EffectAnnotationCollection {
	if p == nil {
		return nil
	}
	return p.annotations
}

// castGuard is the shared prologue of every value member:
//
//	if (_paramClass != expected && pElementCollection.Count == 0)
//	    throw new InvalidCastException();
//
// The second half is what makes an ARRAY parameter readable through a scalar
// overload: a parameter with elements is admitted whatever its own class is,
// which is how the reference lets `float[] v = p.GetValueSingleArray(n)` work
// on a Vector parameter.
func (p *EffectParameter) castGuard(expected EffectParameterClass) error {
	if p == nil {
		return errEffectNil
	}
	if p.parameterClass != expected && p.elements.Count() == 0 {
		return errInvalidCast
	}
	return nil
}

// The scalar getters. Each names the class its reference body compares against.

// GetValueBoolean is EffectParameter::GetValueBoolean.
func (p *EffectParameter) GetValueBoolean() (bool, error) {
	if err := p.castGuard(EffectParameterClassScalar); err != nil {
		return false, err
	}
	var value int32
	if err := p.view.ParameterValue(interop.EffectValueBoolean, unsafe.Pointer(&value)); err != nil {
		return false, err
	}
	return value != 0, nil
}

// GetValueInt32 is EffectParameter::GetValueInt32.
func (p *EffectParameter) GetValueInt32() (int32, error) {
	if err := p.castGuard(EffectParameterClassScalar); err != nil {
		return 0, err
	}
	var value int32
	err := p.view.ParameterValue(interop.EffectValueInt32, unsafe.Pointer(&value))
	return value, err
}

// GetValueSingle is EffectParameter::GetValueSingle.
func (p *EffectParameter) GetValueSingle() (float32, error) {
	if err := p.castGuard(EffectParameterClassScalar); err != nil {
		return 0, err
	}
	var value float32
	err := p.view.ParameterValue(interop.EffectValueSingle, unsafe.Pointer(&value))
	return value, err
}

// GetValueVector2 is EffectParameter::GetValueVector2.
func (p *EffectParameter) GetValueVector2() (framework.Vector2, error) {
	if err := p.castGuard(EffectParameterClassVector); err != nil {
		return framework.Vector2{}, err
	}
	var value framework.Vector2
	err := p.view.ParameterValue(interop.EffectValueVector2, unsafe.Pointer(&value))
	return value, err
}

// GetValueVector3 is EffectParameter::GetValueVector3.
func (p *EffectParameter) GetValueVector3() (framework.Vector3, error) {
	if err := p.castGuard(EffectParameterClassVector); err != nil {
		return framework.Vector3{}, err
	}
	var value framework.Vector3
	err := p.view.ParameterValue(interop.EffectValueVector3, unsafe.Pointer(&value))
	return value, err
}

// GetValueVector4 is EffectParameter::GetValueVector4.
func (p *EffectParameter) GetValueVector4() (framework.Vector4, error) {
	if err := p.castGuard(EffectParameterClassVector); err != nil {
		return framework.Vector4{}, err
	}
	var value framework.Vector4
	err := p.view.ParameterValue(interop.EffectValueVector4, unsafe.Pointer(&value))
	return value, err
}

// GetValueQuaternion is EffectParameter::GetValueQuaternion.
func (p *EffectParameter) GetValueQuaternion() (framework.Quaternion, error) {
	if err := p.castGuard(EffectParameterClassVector); err != nil {
		return framework.Quaternion{}, err
	}
	var value framework.Quaternion
	err := p.view.ParameterValue(interop.EffectValueQuaternion, unsafe.Pointer(&value))
	return value, err
}

// GetValueMatrix is EffectParameter::GetValueMatrix.
func (p *EffectParameter) GetValueMatrix() (framework.Matrix, error) {
	return p.matrixValue(interop.EffectValueMatrix)
}

// GetValueMatrixTranspose is EffectParameter::GetValueMatrixTranspose, which
// reaches D3DX's TRANSPOSED getter rather than transposing the result here --
// exactly as CNA's CNA_EFFECT_VALUE_MATRIX_TRANSPOSE identity does.
func (p *EffectParameter) GetValueMatrixTranspose() (framework.Matrix, error) {
	return p.matrixValue(interop.EffectValueMatrixTranspose)
}

func (p *EffectParameter) matrixValue(valueType uint32) (framework.Matrix, error) {
	if err := p.castGuard(EffectParameterClassMatrix); err != nil {
		return framework.Matrix{}, err
	}
	var values [16]float32
	if err := p.view.ParameterValue(valueType, unsafe.Pointer(&values[0])); err != nil {
		return framework.Matrix{}, err
	}
	return matrixFromRowMajor(values), nil
}

// GetValueString is EffectParameter::GetValueString.
func (p *EffectParameter) GetValueString() (string, error) {
	if p == nil {
		return "", errEffectNil
	}
	if p.parameterClass != EffectParameterClassObject {
		return "", errInvalidCast
	}
	return p.view.String(interop.EffectStringParameterValue)
}

// The three texture getters. Each names a DIFFERENT CNA texture identity, and
// the reference declares three separate members rather than one generic --
// which is what makes CNA_EffectTextureType a tagged identity rather than a
// property of the handle.

// GetValueTexture2D is EffectParameter::GetValueTexture2D.
func (p *EffectParameter) GetValueTexture2D() (*Texture2D, error) {
	if p == nil {
		return nil, errEffectNil
	}
	if p.parameterClass != EffectParameterClassObject {
		return nil, errInvalidCast
	}
	return nil, errEffectTextureIdentity
}

// GetValueTexture3D is EffectParameter::GetValueTexture3D.
func (p *EffectParameter) GetValueTexture3D() (*Texture3D, error) {
	if p == nil {
		return nil, errEffectNil
	}
	if p.parameterClass != EffectParameterClassObject {
		return nil, errInvalidCast
	}
	return nil, errEffectTextureIdentity
}

// GetValueTextureCube is EffectParameter::GetValueTextureCube.
func (p *EffectParameter) GetValueTextureCube() (*TextureCube, error) {
	if p == nil {
		return nil, errEffectNil
	}
	if p.parameterClass != EffectParameterClassObject {
		return nil, errInvalidCast
	}
	return nil, errEffectTextureIdentity
}

// errEffectTextureIdentity records the one divergence in this type, in its own
// words rather than as a borrowed XNA message.
//
// EffectParameter::GetValueTexture2D returns the SAME Texture2D object the
// setter was given: the reference stores the managed reference alongside the
// D3DX handle. CNA's C ABI stores only the handle --
// cna_effect_parameter_get_value_texture reports "the retained handle or
// invalid handle for null" -- so there is no object identity to give back, and
// rebuilding a facade over the handle would hand the consumer a DIFFERENT
// object with the same native half. That is worse than refusing:
// `if (p.GetValueTexture2D() == myTexture)` would silently become false.
//
// The three getters therefore refuse WITHOUT reaching CNA, and
// cna_effect_parameter_get_value_texture stays unbound, recorded in the
// native-ABI registry under REPRESENTATION. Calling it and discarding the
// answer would bind a route to produce a value nothing can use.
var errEffectTextureIdentity = errors.New(
	"CNA reports an effect texture parameter as a handle and not as an object, so the Texture the setter was given cannot be returned as the same object")

// The array getters. Each allocates the array the reference allocates and
// refuses a non-positive count with the reference's own parameterless
// ArgumentOutOfRangeException.

// GetValueBooleanArray is EffectParameter::GetValueBooleanArray(Int32).
func (p *EffectParameter) GetValueBooleanArray(count int32) ([]bool, error) {
	raw, err := p.int32Array(EffectParameterClassScalar, interop.EffectValueBoolean, count)
	if err != nil {
		return nil, err
	}
	values := make([]bool, len(raw))
	for index := range raw {
		values[index] = raw[index] != 0
	}
	return values, nil
}

// GetValueInt32Array is EffectParameter::GetValueInt32Array(Int32).
func (p *EffectParameter) GetValueInt32Array(count int32) ([]int32, error) {
	return p.int32Array(EffectParameterClassScalar, interop.EffectValueInt32, count)
}

func (p *EffectParameter) int32Array(expected EffectParameterClass, valueType uint32, count int32) ([]int32, error) {
	if err := p.arrayGuard(expected, count); err != nil {
		return nil, err
	}
	values := make([]int32, count)
	if _, err := p.view.ParameterValues(valueType, uint64(count), unsafe.Pointer(&values[0]), uint64(count)); err != nil {
		return nil, err
	}
	return values, nil
}

// GetValueSingleArray is EffectParameter::GetValueSingleArray(Int32).
func (p *EffectParameter) GetValueSingleArray(count int32) ([]float32, error) {
	if err := p.arrayGuard(EffectParameterClassScalar, count); err != nil {
		return nil, err
	}
	values := make([]float32, count)
	if _, err := p.view.ParameterValues(interop.EffectValueSingle, uint64(count), unsafe.Pointer(&values[0]), uint64(count)); err != nil {
		return nil, err
	}
	return values, nil
}

// GetValueVector2Array is EffectParameter::GetValueVector2Array(Int32).
func (p *EffectParameter) GetValueVector2Array(count int32) ([]framework.Vector2, error) {
	if err := p.arrayGuard(EffectParameterClassVector, count); err != nil {
		return nil, err
	}
	values := make([]framework.Vector2, count)
	if _, err := p.view.ParameterValues(interop.EffectValueVector2, uint64(count), unsafe.Pointer(&values[0]), uint64(count)); err != nil {
		return nil, err
	}
	return values, nil
}

// GetValueVector3Array is EffectParameter::GetValueVector3Array(Int32).
func (p *EffectParameter) GetValueVector3Array(count int32) ([]framework.Vector3, error) {
	if err := p.arrayGuard(EffectParameterClassVector, count); err != nil {
		return nil, err
	}
	values := make([]framework.Vector3, count)
	if _, err := p.view.ParameterValues(interop.EffectValueVector3, uint64(count), unsafe.Pointer(&values[0]), uint64(count)); err != nil {
		return nil, err
	}
	return values, nil
}

// GetValueVector4Array is EffectParameter::GetValueVector4Array(Int32).
func (p *EffectParameter) GetValueVector4Array(count int32) ([]framework.Vector4, error) {
	if err := p.arrayGuard(EffectParameterClassVector, count); err != nil {
		return nil, err
	}
	values := make([]framework.Vector4, count)
	if _, err := p.view.ParameterValues(interop.EffectValueVector4, uint64(count), unsafe.Pointer(&values[0]), uint64(count)); err != nil {
		return nil, err
	}
	return values, nil
}

// GetValueQuaternionArray is EffectParameter::GetValueQuaternionArray(Int32).
func (p *EffectParameter) GetValueQuaternionArray(count int32) ([]framework.Quaternion, error) {
	if err := p.arrayGuard(EffectParameterClassVector, count); err != nil {
		return nil, err
	}
	values := make([]framework.Quaternion, count)
	if _, err := p.view.ParameterValues(interop.EffectValueQuaternion, uint64(count), unsafe.Pointer(&values[0]), uint64(count)); err != nil {
		return nil, err
	}
	return values, nil
}

// GetValueMatrixArray is EffectParameter::GetValueMatrixArray(Int32).
func (p *EffectParameter) GetValueMatrixArray(count int32) ([]framework.Matrix, error) {
	return p.matrixArray(interop.EffectValueMatrix, count)
}

// GetValueMatrixTransposeArray is
// EffectParameter::GetValueMatrixTransposeArray(Int32).
func (p *EffectParameter) GetValueMatrixTransposeArray(count int32) ([]framework.Matrix, error) {
	return p.matrixArray(interop.EffectValueMatrixTranspose, count)
}

func (p *EffectParameter) matrixArray(valueType uint32, count int32) ([]framework.Matrix, error) {
	if err := p.arrayGuard(EffectParameterClassMatrix, count); err != nil {
		return nil, err
	}
	raw := make([]float32, int(count)*16)
	if _, err := p.view.ParameterValues(valueType, uint64(count), unsafe.Pointer(&raw[0]), uint64(count)); err != nil {
		return nil, err
	}
	values := make([]framework.Matrix, count)
	for index := range values {
		var block [16]float32
		copy(block[:], raw[index*16:(index+1)*16])
		values[index] = matrixFromRowMajor(block)
	}
	return values, nil
}

// arrayGuard is the array getters' shared prologue:
//
//	if (count <= 0) throw new ArgumentOutOfRangeException();
//
// a PARAMETERLESS ArgumentOutOfRangeException -- the reference names no
// parameter and loads no resource string -- followed by the same cast guard the
// scalar getters apply.
func (p *EffectParameter) arrayGuard(expected EffectParameterClass, count int32) error {
	if p == nil {
		return errEffectNil
	}
	if count <= 0 {
		return errArgumentOutOfRange
	}
	return p.castGuard(expected)
}

// The eighteen setters. Every one is `SetValue(value)` in the reference with
// the same cast guard the matching getter has, so the guard is applied here for
// the reason it is applied there: CNA accepts a mismatched write silently.

// SetValueByBoolean is EffectParameter::SetValue(Boolean).
func (p *EffectParameter) SetValueByBoolean(value bool) error {
	if err := p.castGuard(EffectParameterClassScalar); err != nil {
		return err
	}
	stored := int32(0)
	if value {
		stored = 1
	}
	return p.view.SetParameterValue(interop.EffectValueBoolean, unsafe.Pointer(&stored))
}

// SetValueBySliceOfBoolean is EffectParameter::SetValue(Boolean[]).
func (p *EffectParameter) SetValueBySliceOfBoolean(value []bool) error {
	if err := p.castGuard(EffectParameterClassScalar); err != nil {
		return err
	}
	stored := make([]int32, len(value))
	for index := range value {
		if value[index] {
			stored[index] = 1
		}
	}
	return p.setValues(interop.EffectValueBoolean, unsafe.Pointer(unsafe.SliceData(stored)), len(stored))
}

// SetValueByInt32 is EffectParameter::SetValue(Int32).
func (p *EffectParameter) SetValueByInt32(value int32) error {
	if err := p.castGuard(EffectParameterClassScalar); err != nil {
		return err
	}
	return p.view.SetParameterValue(interop.EffectValueInt32, unsafe.Pointer(&value))
}

// SetValueBySliceOfInt32 is EffectParameter::SetValue(Int32[]).
func (p *EffectParameter) SetValueBySliceOfInt32(value []int32) error {
	if err := p.castGuard(EffectParameterClassScalar); err != nil {
		return err
	}
	return p.setValues(interop.EffectValueInt32, unsafe.Pointer(unsafe.SliceData(value)), len(value))
}

// SetValueBySingle is EffectParameter::SetValue(Single).
func (p *EffectParameter) SetValueBySingle(value float32) error {
	if err := p.castGuard(EffectParameterClassScalar); err != nil {
		return err
	}
	return p.view.SetParameterValue(interop.EffectValueSingle, unsafe.Pointer(&value))
}

// SetValueBySliceOfSingle is EffectParameter::SetValue(Single[]).
func (p *EffectParameter) SetValueBySliceOfSingle(value []float32) error {
	if err := p.castGuard(EffectParameterClassScalar); err != nil {
		return err
	}
	return p.setValues(interop.EffectValueSingle, unsafe.Pointer(unsafe.SliceData(value)), len(value))
}

// SetValueByVector2 is EffectParameter::SetValue(Vector2).
func (p *EffectParameter) SetValueByVector2(value framework.Vector2) error {
	if err := p.castGuard(EffectParameterClassVector); err != nil {
		return err
	}
	return p.view.SetParameterValue(interop.EffectValueVector2, unsafe.Pointer(&value))
}

// SetValueBySliceOfVector2 is EffectParameter::SetValue(Vector2[]).
func (p *EffectParameter) SetValueBySliceOfVector2(value []framework.Vector2) error {
	if err := p.castGuard(EffectParameterClassVector); err != nil {
		return err
	}
	return p.setValues(interop.EffectValueVector2, unsafe.Pointer(unsafe.SliceData(value)), len(value))
}

// SetValueByVector3 is EffectParameter::SetValue(Vector3).
func (p *EffectParameter) SetValueByVector3(value framework.Vector3) error {
	if err := p.castGuard(EffectParameterClassVector); err != nil {
		return err
	}
	return p.view.SetParameterValue(interop.EffectValueVector3, unsafe.Pointer(&value))
}

// SetValueBySliceOfVector3 is EffectParameter::SetValue(Vector3[]).
func (p *EffectParameter) SetValueBySliceOfVector3(value []framework.Vector3) error {
	if err := p.castGuard(EffectParameterClassVector); err != nil {
		return err
	}
	return p.setValues(interop.EffectValueVector3, unsafe.Pointer(unsafe.SliceData(value)), len(value))
}

// SetValueByVector4 is EffectParameter::SetValue(Vector4).
func (p *EffectParameter) SetValueByVector4(value framework.Vector4) error {
	if err := p.castGuard(EffectParameterClassVector); err != nil {
		return err
	}
	return p.view.SetParameterValue(interop.EffectValueVector4, unsafe.Pointer(&value))
}

// SetValueBySliceOfVector4 is EffectParameter::SetValue(Vector4[]).
func (p *EffectParameter) SetValueBySliceOfVector4(value []framework.Vector4) error {
	if err := p.castGuard(EffectParameterClassVector); err != nil {
		return err
	}
	return p.setValues(interop.EffectValueVector4, unsafe.Pointer(unsafe.SliceData(value)), len(value))
}

// SetValueByQuaternion is EffectParameter::SetValue(Quaternion).
func (p *EffectParameter) SetValueByQuaternion(value framework.Quaternion) error {
	if err := p.castGuard(EffectParameterClassVector); err != nil {
		return err
	}
	return p.view.SetParameterValue(interop.EffectValueQuaternion, unsafe.Pointer(&value))
}

// SetValueBySliceOfQuaternion is EffectParameter::SetValue(Quaternion[]).
func (p *EffectParameter) SetValueBySliceOfQuaternion(value []framework.Quaternion) error {
	if err := p.castGuard(EffectParameterClassVector); err != nil {
		return err
	}
	return p.setValues(interop.EffectValueQuaternion, unsafe.Pointer(unsafe.SliceData(value)), len(value))
}

// SetValueByMatrix is EffectParameter::SetValue(Matrix).
func (p *EffectParameter) SetValueByMatrix(value framework.Matrix) error {
	return p.setMatrix(interop.EffectValueMatrix, value)
}

// SetValueBySliceOfMatrix is EffectParameter::SetValue(Matrix[]).
func (p *EffectParameter) SetValueBySliceOfMatrix(value []framework.Matrix) error {
	return p.setMatrixArray(interop.EffectValueMatrix, value)
}

// SetValueTransposeByMatrix is EffectParameter::SetValueTranspose(Matrix),
// which reaches D3DX's transposed SETTER rather than transposing here.
func (p *EffectParameter) SetValueTransposeByMatrix(value framework.Matrix) error {
	return p.setMatrix(interop.EffectValueMatrixTranspose, value)
}

// SetValueTransposeBySliceOfMatrix is
// EffectParameter::SetValueTranspose(Matrix[]).
func (p *EffectParameter) SetValueTransposeBySliceOfMatrix(value []framework.Matrix) error {
	return p.setMatrixArray(interop.EffectValueMatrixTranspose, value)
}

func (p *EffectParameter) setMatrix(valueType uint32, value framework.Matrix) error {
	if err := p.castGuard(EffectParameterClassMatrix); err != nil {
		return err
	}
	block := matrixToRowMajor(value)
	return p.view.SetParameterValue(valueType, unsafe.Pointer(&block[0]))
}

func (p *EffectParameter) setMatrixArray(valueType uint32, value []framework.Matrix) error {
	if err := p.castGuard(EffectParameterClassMatrix); err != nil {
		return err
	}
	raw := make([]float32, len(value)*16)
	for index := range value {
		block := matrixToRowMajor(value[index])
		copy(raw[index*16:], block[:])
	}
	return p.setValues(valueType, unsafe.Pointer(unsafe.SliceData(raw)), len(value))
}

// SetValueByString is EffectParameter::SetValue(String).
func (p *EffectParameter) SetValueByString(value string) error {
	if p == nil {
		return errEffectNil
	}
	if p.parameterClass != EffectParameterClassObject {
		return errInvalidCast
	}
	return p.view.SetParameterValueString(value)
}

// SetValueByTexture is EffectParameter::SetValue(Texture), whose parameter is
// the BASE class -- so it takes the settled reference interface and a
// Texture2D, TextureCube or Texture3D all flow into it.
func (p *EffectParameter) SetValueByTexture(value TextureReference) error {
	if p == nil {
		return errEffectNil
	}
	if p.parameterClass != EffectParameterClassObject {
		return errInvalidCast
	}
	texture := resolveTexture(value)
	if texture == nil {
		return p.view.SetParameterValueTexture(interop.EffectTextureBase, nil)
	}
	return p.view.SetParameterValueTexture(interop.EffectTextureBase, texture.nativeResource())
}

func (p *EffectParameter) setValues(valueType uint32, values unsafe.Pointer, count int) error {
	if count == 0 {
		return p.view.SetParameterValues(valueType, nil, 0)
	}
	return p.view.SetParameterValues(valueType, values, uint64(count))
}

// matrixToRowMajor is matrixFromRowMajor's inverse, written out for the same
// reason.
func matrixToRowMajor(value framework.Matrix) [16]float32 {
	return [16]float32{
		value.M11, value.M12, value.M13, value.M14,
		value.M21, value.M22, value.M23, value.M24,
		value.M31, value.M32, value.M33, value.M34,
		value.M41, value.M42, value.M43, value.M44,
	}
}

// dispose releases every view this parameter's graph owns.
func (p *EffectParameter) dispose() error {
	if p == nil {
		return nil
	}
	return errors.Join(
		p.elements.dispose(),
		p.structureMembers.dispose(),
		p.annotations.dispose(),
		p.view.Dispose(),
	)
}

// EffectParameterCollection is
// Microsoft.Xna.Framework.Graphics.EffectParameterCollection.
//
// Its two indexers answer NULL out of range and for an unknown name, exactly as
// EffectAnnotationCollection's do -- and its third member disagrees with them
// about string comparison, which is the reference's own inconsistency and is
// preserved:
//
//	get_Item(string)          String.op_Equality               ordinal, case SENSITIVE
//	GetParameterBySemantic    String.Compare(a, b, Ordinal-
//	                          IgnoreCase)                      case INSENSITIVE
type EffectParameterCollection struct {
	view       *interop.EffectView
	parameters []*EffectParameter
}

func newEffectParameterCollection(view *interop.EffectView) (*EffectParameterCollection, error) {
	count, err := view.Count(interop.EffectCollectionParameter)
	if err != nil {
		return nil, err
	}
	collection := &EffectParameterCollection{view: view}
	for index := uint64(0); index < count; index++ {
		element, err := view.At(interop.EffectCollectionParameter, index, interop.EffectViewParameter)
		if err != nil {
			return nil, err
		}
		parameter, err := newEffectParameter(element)
		if err != nil {
			return nil, err
		}
		collection.parameters = append(collection.parameters, parameter)
	}
	return collection, nil
}

// Count is EffectParameterCollection::get_Count.
func (c *EffectParameterCollection) Count() int32 {
	if c == nil {
		return 0
	}
	return int32(len(c.parameters))
}

// ItemPropertySignatureC421CB56 is EffectParameterCollection::get_Item(Int32).
func (c *EffectParameterCollection) ItemPropertySignatureC421CB56(index int32) *EffectParameter {
	if c == nil || index < 0 || int(index) >= len(c.parameters) {
		return nil
	}
	return c.parameters[index]
}

// ItemPropertySignatureD47463F7 is EffectParameterCollection::get_Item(String), an ORDINAL
// case-sensitive scan.
func (c *EffectParameterCollection) ItemPropertySignatureD47463F7(name string) *EffectParameter {
	if c == nil {
		return nil
	}
	for _, parameter := range c.parameters {
		if parameter.name == name {
			return parameter
		}
	}
	return nil
}

// GetParameterBySemantic is
// EffectParameterCollection::GetParameterBySemantic(String), which compares
// with `String.Compare(a, b, StringComparison.OrdinalIgnoreCase)` -- the
// literal 5 at IL_0024 -- and is therefore case-INSENSITIVE where the string
// indexer three members above it is not.
func (c *EffectParameterCollection) GetParameterBySemantic(semantic string) *EffectParameter {
	if c == nil {
		return nil
	}
	for _, parameter := range c.parameters {
		if equalsOrdinalIgnoreCase(parameter.semantic, semantic) {
			return parameter
		}
	}
	return nil
}

// equalsOrdinalIgnoreCase is System.String.Compare(a, b, OrdinalIgnoreCase) == 0.
//
// It is spelled out rather than taken from strings.EqualFold, which folds by
// UNICODE case rules: OrdinalIgnoreCase folds only ASCII A-Z, so 'K' and the
// Kelvin sign U+212A are equal under EqualFold and are NOT equal under the
// comparison the reference performs.
func equalsOrdinalIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for index := 0; index < len(a); index++ {
		x, y := a[index], b[index]
		if 'a' <= x && x <= 'z' {
			x -= 'a' - 'A'
		}
		if 'a' <= y && y <= 'z' {
			y -= 'a' - 'A'
		}
		if x != y {
			return false
		}
	}
	return true
}

// GetEnumerator is EffectParameterCollection::GetEnumerator.
func (c *EffectParameterCollection) GetEnumerator() framework.Iterator[*EffectParameter] {
	if c == nil {
		return &effectIterator[*EffectParameter]{}
	}
	return &effectIterator[*EffectParameter]{items: c.parameters}
}

func (c *EffectParameterCollection) dispose() error {
	if c == nil {
		return nil
	}
	var failures []error
	for _, parameter := range c.parameters {
		failures = append(failures, parameter.dispose())
	}
	failures = append(failures, c.view.Dispose())
	return errors.Join(failures...)
}
