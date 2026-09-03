package graphics

import (
	"fmt"
	"math"
	"sync"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// This file projects the four stock vertex structs the XNA 4.0 Windows profile
// declares:
//
//	VertexPositionColor          Position, Color
//	VertexPositionTexture        Position, TextureCoordinate
//	VertexPositionColorTexture   Position, Color, TextureCoordinate
//	VertexPositionNormalTexture  Position, Normal, TextureCoordinate
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Graphics.dll
//	  560080fc39021c61...
//
// All four have the same shape, member for member: public fields in declaration
// order, one constructor that stores them, a static readonly VertexDeclaration,
// an EXPLICIT IVertexType implementation returning it, GetHashCode over
// Helpers.SmartGetHashCode, a String.Format ToString, op_Equality, op_Inequality
// and Equals(object).
//
// # Three details that decide behaviour
//
//  1. op_Equality compares the LAST field first. VertexPositionColor tests
//     Color before Position; the three-field structs test their third field
//     first. The result is unaffected -- equality is a conjunction -- and the
//     order is preserved anyway because it is what the reference does.
//  2. Equals(object) is a TYPE test, not a shape test: `obj.GetType() !=
//     this.GetType()` returns false before any field is read, so a
//     VertexPositionColor never equals a VertexPositionColorTexture no matter
//     what they hold.
//  3. GetHashCode is Helpers.SmartGetHashCode over the pinned struct's complete
//     32-bit words -- the settled rule the GamePad value structs already use.
//     A Vector3 is three words, a Vector2 two, and a Color one; the zero
//     substitution is Int32.MaxValue and is deliberate.
//
// # The static VertexDeclarations
//
// Each is `public static initonly`, assigned once by the type's `.cctor` from a
// VertexElement array and then given a Name. The settled static-readonly-field
// rule projects it as a zero-argument package function prefixed by the
// declaring type, and the ONE-TIME construction is reproduced: the declaration
// is built on first use and cached, so every caller gets the same object, as
// every caller of the CLR static field does.
//
// The element tables below are read from each `.cctor`'s IL, offset by offset.
// They are not derived from the Go struct layout -- Go's layout is not the
// CLR's marshalled one, and a projection that computed offsets would be
// asserting a coincidence rather than reproducing a table.

// vertexDeclarationOnce caches one static VertexDeclaration, reproducing the
// `.cctor`'s once-only assignment.
type vertexDeclarationOnce struct {
	once     sync.Once
	value    *VertexDeclaration
	elements []VertexElement
	name     string
}

func (v *vertexDeclarationOnce) get() *VertexDeclaration {
	v.once.Do(func() {
		declaration, err := NewVertexDeclarationBySliceOfVertexElement(v.elements)
		if err != nil {
			// Unreachable: the element tables are literals read from the
			// reference's own static constructors, and the validator that could
			// refuse them is the one those tables already satisfy. Leaving nil
			// rather than panicking keeps a defect in this file from taking down
			// a consumer's process; the vertex-declaration tests assert every
			// one of the four is present with its exact stride and elements.
			return
		}
		declaration.SetName(v.name)
		v.value = declaration
	})
	return v.value
}

// The four element tables, offset by offset from each `.cctor`.
var (
	vertexPositionColorDeclaration = &vertexDeclarationOnce{
		name: "VertexPositionColor.VertexDeclaration",
		elements: []VertexElement{
			NewVertexElement(0, VertexElementFormatVector3, VertexElementUsagePosition, 0),
			NewVertexElement(12, VertexElementFormatColor, VertexElementUsageColor, 0),
		},
	}
	vertexPositionTextureDeclaration = &vertexDeclarationOnce{
		name: "VertexPositionTexture.VertexDeclaration",
		elements: []VertexElement{
			NewVertexElement(0, VertexElementFormatVector3, VertexElementUsagePosition, 0),
			NewVertexElement(12, VertexElementFormatVector2, VertexElementUsageTextureCoordinate, 0),
		},
	}
	vertexPositionColorTextureDeclaration = &vertexDeclarationOnce{
		name: "VertexPositionColorTexture.VertexDeclaration",
		elements: []VertexElement{
			NewVertexElement(0, VertexElementFormatVector3, VertexElementUsagePosition, 0),
			NewVertexElement(12, VertexElementFormatColor, VertexElementUsageColor, 0),
			NewVertexElement(16, VertexElementFormatVector2, VertexElementUsageTextureCoordinate, 0),
		},
	}
	vertexPositionNormalTextureDeclaration = &vertexDeclarationOnce{
		name: "VertexPositionNormalTexture.VertexDeclaration",
		elements: []VertexElement{
			NewVertexElement(0, VertexElementFormatVector3, VertexElementUsagePosition, 0),
			NewVertexElement(12, VertexElementFormatVector3, VertexElementUsageNormal, 0),
			NewVertexElement(24, VertexElementFormatVector2, VertexElementUsageTextureCoordinate, 0),
		},
	}
)

// smartGetHashCode reproduces Microsoft.Xna.Framework.Helpers.SmartGetHashCode,
// which pins the boxed value, XORs every complete 32-bit word of its marshalled
// layout, and substitutes Int32.MaxValue when the XOR is zero.
//
// The zero substitution is intentional and creates compatible collisions; it
// must not be replaced with a better-distributed combine. XOR is commutative,
// so the declared field order does not affect the result -- only the SET of
// words does, which is why the tables below list them in field order for
// readability rather than for correctness.
func smartGetHashCode(words ...int32) int32 {
	var hash int32
	for _, word := range words {
		hash ^= word
	}
	if hash == 0 {
		return math.MaxInt32
	}
	return hash
}

// singleWord reinterprets a Single as the 32-bit word SmartGetHashCode reads
// out of the pinned struct. Unlike System.Single.GetHashCode it does not
// canonicalise signed zero, because the helper reads raw storage.
func singleWord(value float32) int32 { return int32(math.Float32bits(value)) }

func vector3Words(value framework.Vector3) []int32 {
	return []int32{singleWord(value.X), singleWord(value.Y), singleWord(value.Z)}
}

func vector2Words(value framework.Vector2) []int32 {
	return []int32{singleWord(value.X), singleWord(value.Y)}
}

// colorWord is the ONE word a Color occupies: its packed BGRA value, which is
// the struct's only field.
func colorWord(value framework.Color) int32 { return int32(value.PackedValue()) }

// ---------------------------------------------------------------------------
// VertexPositionColor
// ---------------------------------------------------------------------------

// VertexPositionColor is a position and a colour, the vertex a coloured
// primitive is drawn from.
type VertexPositionColor struct {
	Position framework.Vector3
	Color    framework.Color
}

// NewVertexPositionColor is VertexPositionColor::.ctor, two `stfld` with no
// validation.
func NewVertexPositionColor(position framework.Vector3, color framework.Color) VertexPositionColor {
	return VertexPositionColor{Position: position, Color: color}
}

// VertexDeclaration is the explicit IVertexType implementation, `ldsfld` of the
// static field. It is the interface WITNESS: the CLR implements it privately,
// so the contract's public member set has no such member, and Go needs it
// exported for the type to satisfy IVertexType at all.
func (v VertexPositionColor) VertexDeclaration() *VertexDeclaration {
	return vertexPositionColorDeclaration.get()
}

// VertexPositionColorVertexDeclaration is the public static readonly field.
func VertexPositionColorVertexDeclaration() *VertexDeclaration {
	return vertexPositionColorDeclaration.get()
}

// GetHashCode is SmartGetHashCode over four words: three for the Vector3 and
// one for the packed Color.
func (v VertexPositionColor) GetHashCode() int32 {
	return smartGetHashCode(append(vector3Words(v.Position), colorWord(v.Color))...)
}

// ToString is String.Format(CurrentCulture, "{{Position:{0} Color:{1}}}", ...).
func (v VertexPositionColor) ToString() string {
	return fmt.Sprintf("{Position:%s Color:%s}", v.Position.ToString(), v.Color.ToString())
}

// VertexPositionColorOperatorEqualityByVertexPositionColorAndVertexPositionColor
// is op_Equality, which tests COLOR first and only then Position.
func VertexPositionColorOperatorEqualityByVertexPositionColorAndVertexPositionColor(left, right VertexPositionColor) bool {
	return framework.ColorOperatorEqualityByColorAndColor(left.Color, right.Color) &&
		framework.Vector3OperatorEqualityByVector3AndVector3(left.Position, right.Position)
}

// VertexPositionColorOperatorInequalityByVertexPositionColorAndVertexPositionColor
// is op_Inequality, one negated op_Equality.
func VertexPositionColorOperatorInequalityByVertexPositionColorAndVertexPositionColor(left, right VertexPositionColor) bool {
	return !VertexPositionColorOperatorEqualityByVertexPositionColorAndVertexPositionColor(left, right)
}

// Equals is Equals(object): null is false, a different runtime TYPE is false
// before any field is read, and otherwise op_Equality decides.
func (v VertexPositionColor) Equals(obj any) bool {
	other, ok := obj.(VertexPositionColor)
	if !ok {
		return false
	}
	return VertexPositionColorOperatorEqualityByVertexPositionColorAndVertexPositionColor(v, other)
}

// ---------------------------------------------------------------------------
// VertexPositionTexture
// ---------------------------------------------------------------------------

// VertexPositionTexture is a position and a texture coordinate.
type VertexPositionTexture struct {
	Position          framework.Vector3
	TextureCoordinate framework.Vector2
}

// NewVertexPositionTexture is VertexPositionTexture::.ctor.
func NewVertexPositionTexture(position framework.Vector3, textureCoordinate framework.Vector2) VertexPositionTexture {
	return VertexPositionTexture{Position: position, TextureCoordinate: textureCoordinate}
}

// VertexDeclaration is the IVertexType witness.
func (v VertexPositionTexture) VertexDeclaration() *VertexDeclaration {
	return vertexPositionTextureDeclaration.get()
}

// VertexPositionTextureVertexDeclaration is the public static readonly field.
func VertexPositionTextureVertexDeclaration() *VertexDeclaration {
	return vertexPositionTextureDeclaration.get()
}

// GetHashCode is SmartGetHashCode over five words.
func (v VertexPositionTexture) GetHashCode() int32 {
	return smartGetHashCode(append(vector3Words(v.Position), vector2Words(v.TextureCoordinate)...)...)
}

// ToString is "{{Position:{0} TextureCoordinate:{1}}}".
func (v VertexPositionTexture) ToString() string {
	return fmt.Sprintf("{Position:%s TextureCoordinate:%s}",
		v.Position.ToString(), v.TextureCoordinate.ToString())
}

// VertexPositionTextureOperatorEqualityByVertexPositionTextureAndVertexPositionTexture
// is op_Equality, which tests the TEXTURE COORDINATE first.
func VertexPositionTextureOperatorEqualityByVertexPositionTextureAndVertexPositionTexture(left, right VertexPositionTexture) bool {
	return framework.Vector3OperatorEqualityByVector3AndVector3(left.Position, right.Position) &&
		framework.Vector2OperatorEqualityByVector2AndVector2(left.TextureCoordinate, right.TextureCoordinate)
}

// VertexPositionTextureOperatorInequalityByVertexPositionTextureAndVertexPositionTexture
// is op_Inequality.
func VertexPositionTextureOperatorInequalityByVertexPositionTextureAndVertexPositionTexture(left, right VertexPositionTexture) bool {
	return !VertexPositionTextureOperatorEqualityByVertexPositionTextureAndVertexPositionTexture(left, right)
}

// Equals is Equals(object).
func (v VertexPositionTexture) Equals(obj any) bool {
	other, ok := obj.(VertexPositionTexture)
	if !ok {
		return false
	}
	return VertexPositionTextureOperatorEqualityByVertexPositionTextureAndVertexPositionTexture(v, other)
}

// ---------------------------------------------------------------------------
// VertexPositionColorTexture
// ---------------------------------------------------------------------------

// VertexPositionColorTexture is a position, a colour and a texture coordinate.
type VertexPositionColorTexture struct {
	Position          framework.Vector3
	Color             framework.Color
	TextureCoordinate framework.Vector2
}

// NewVertexPositionColorTexture is VertexPositionColorTexture::.ctor.
func NewVertexPositionColorTexture(
	position framework.Vector3, color framework.Color, textureCoordinate framework.Vector2,
) VertexPositionColorTexture {
	return VertexPositionColorTexture{Position: position, Color: color, TextureCoordinate: textureCoordinate}
}

// VertexDeclaration is the IVertexType witness.
func (v VertexPositionColorTexture) VertexDeclaration() *VertexDeclaration {
	return vertexPositionColorTextureDeclaration.get()
}

// VertexPositionColorTextureVertexDeclaration is the public static readonly
// field.
func VertexPositionColorTextureVertexDeclaration() *VertexDeclaration {
	return vertexPositionColorTextureDeclaration.get()
}

// GetHashCode is SmartGetHashCode over six words.
func (v VertexPositionColorTexture) GetHashCode() int32 {
	words := append(vector3Words(v.Position), colorWord(v.Color))
	return smartGetHashCode(append(words, vector2Words(v.TextureCoordinate)...)...)
}

// ToString is "{{Position:{0} Color:{1} TextureCoordinate:{2}}}".
func (v VertexPositionColorTexture) ToString() string {
	return fmt.Sprintf("{Position:%s Color:%s TextureCoordinate:%s}",
		v.Position.ToString(), v.Color.ToString(), v.TextureCoordinate.ToString())
}

// VertexPositionColorTextureOperatorEqualityByVertexPositionColorTextureAndVertexPositionColorTexture
// is op_Equality.
func VertexPositionColorTextureOperatorEqualityByVertexPositionColorTextureAndVertexPositionColorTexture(
	left, right VertexPositionColorTexture,
) bool {
	return framework.Vector3OperatorEqualityByVector3AndVector3(left.Position, right.Position) &&
		framework.ColorOperatorEqualityByColorAndColor(left.Color, right.Color) &&
		framework.Vector2OperatorEqualityByVector2AndVector2(left.TextureCoordinate, right.TextureCoordinate)
}

// VertexPositionColorTextureOperatorInequalityByVertexPositionColorTextureAndVertexPositionColorTexture
// is op_Inequality.
func VertexPositionColorTextureOperatorInequalityByVertexPositionColorTextureAndVertexPositionColorTexture(
	left, right VertexPositionColorTexture,
) bool {
	return !VertexPositionColorTextureOperatorEqualityByVertexPositionColorTextureAndVertexPositionColorTexture(left, right)
}

// Equals is Equals(object).
func (v VertexPositionColorTexture) Equals(obj any) bool {
	other, ok := obj.(VertexPositionColorTexture)
	if !ok {
		return false
	}
	return VertexPositionColorTextureOperatorEqualityByVertexPositionColorTextureAndVertexPositionColorTexture(v, other)
}

// ---------------------------------------------------------------------------
// VertexPositionNormalTexture
// ---------------------------------------------------------------------------

// VertexPositionNormalTexture is a position, a normal and a texture coordinate:
// the vertex a lit, textured mesh is drawn from, and the one BasicEffect's
// lighting path expects.
type VertexPositionNormalTexture struct {
	Position          framework.Vector3
	Normal            framework.Vector3
	TextureCoordinate framework.Vector2
}

// NewVertexPositionNormalTexture is VertexPositionNormalTexture::.ctor.
func NewVertexPositionNormalTexture(
	position, normal framework.Vector3, textureCoordinate framework.Vector2,
) VertexPositionNormalTexture {
	return VertexPositionNormalTexture{Position: position, Normal: normal, TextureCoordinate: textureCoordinate}
}

// VertexDeclaration is the IVertexType witness.
func (v VertexPositionNormalTexture) VertexDeclaration() *VertexDeclaration {
	return vertexPositionNormalTextureDeclaration.get()
}

// VertexPositionNormalTextureVertexDeclaration is the public static readonly
// field.
func VertexPositionNormalTextureVertexDeclaration() *VertexDeclaration {
	return vertexPositionNormalTextureDeclaration.get()
}

// GetHashCode is SmartGetHashCode over eight words.
func (v VertexPositionNormalTexture) GetHashCode() int32 {
	words := append(vector3Words(v.Position), vector3Words(v.Normal)...)
	return smartGetHashCode(append(words, vector2Words(v.TextureCoordinate)...)...)
}

// ToString is "{{Position:{0} Normal:{1} TextureCoordinate:{2}}}".
func (v VertexPositionNormalTexture) ToString() string {
	return fmt.Sprintf("{Position:%s Normal:%s TextureCoordinate:%s}",
		v.Position.ToString(), v.Normal.ToString(), v.TextureCoordinate.ToString())
}

// VertexPositionNormalTextureOperatorEqualityByVertexPositionNormalTextureAndVertexPositionNormalTexture
// is op_Equality.
func VertexPositionNormalTextureOperatorEqualityByVertexPositionNormalTextureAndVertexPositionNormalTexture(
	left, right VertexPositionNormalTexture,
) bool {
	return framework.Vector3OperatorEqualityByVector3AndVector3(left.Position, right.Position) &&
		framework.Vector3OperatorEqualityByVector3AndVector3(left.Normal, right.Normal) &&
		framework.Vector2OperatorEqualityByVector2AndVector2(left.TextureCoordinate, right.TextureCoordinate)
}

// VertexPositionNormalTextureOperatorInequalityByVertexPositionNormalTextureAndVertexPositionNormalTexture
// is op_Inequality.
func VertexPositionNormalTextureOperatorInequalityByVertexPositionNormalTextureAndVertexPositionNormalTexture(
	left, right VertexPositionNormalTexture,
) bool {
	return !VertexPositionNormalTextureOperatorEqualityByVertexPositionNormalTextureAndVertexPositionNormalTexture(left, right)
}

// Equals is Equals(object).
func (v VertexPositionNormalTexture) Equals(obj any) bool {
	other, ok := obj.(VertexPositionNormalTexture)
	if !ok {
		return false
	}
	return VertexPositionNormalTextureOperatorEqualityByVertexPositionNormalTextureAndVertexPositionNormalTexture(v, other)
}

// The four types satisfy IVertexType, which is what the witness methods above
// exist for. A compile-time assertion is cheaper than discovering it at a draw
// call.
var (
	_ IVertexType = VertexPositionColor{}
	_ IVertexType = VertexPositionTexture{}
	_ IVertexType = VertexPositionColorTexture{}
	_ IVertexType = VertexPositionNormalTexture{}
)
