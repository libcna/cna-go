package graphics

import (
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

func vector3(x, y, z float32) framework.Vector3 {
	return framework.NewVector3BySingleAndSingleAndSingle(x, y, z)
}

func vector2(x, y float32) framework.Vector2 {
	return framework.NewVector2BySingleAndSingle(x, y)
}

// TestVertexDeclarationsAreTheReferencesElementTables is the claim the whole
// family rests on: the four static declarations carry the exact offsets,
// formats, usages and strides the reference's four static constructors build.
//
// The offsets are NOT derived from the Go struct layout. Go's layout is not the
// CLR's marshalled one, and a test that computed them would be asserting a
// coincidence rather than reproducing a table.
func TestVertexDeclarationsAreTheReferencesElementTables(t *testing.T) {
	cases := []struct {
		name        string
		declaration *VertexDeclaration
		stride      int32
		elements    []VertexElement
	}{
		{
			name:        "VertexPositionColor",
			declaration: VertexPositionColorVertexDeclaration(),
			stride:      16,
			elements: []VertexElement{
				NewVertexElement(0, VertexElementFormatVector3, VertexElementUsagePosition, 0),
				NewVertexElement(12, VertexElementFormatColor, VertexElementUsageColor, 0),
			},
		},
		{
			name:        "VertexPositionTexture",
			declaration: VertexPositionTextureVertexDeclaration(),
			stride:      20,
			elements: []VertexElement{
				NewVertexElement(0, VertexElementFormatVector3, VertexElementUsagePosition, 0),
				NewVertexElement(12, VertexElementFormatVector2, VertexElementUsageTextureCoordinate, 0),
			},
		},
		{
			name:        "VertexPositionColorTexture",
			declaration: VertexPositionColorTextureVertexDeclaration(),
			stride:      24,
			elements: []VertexElement{
				NewVertexElement(0, VertexElementFormatVector3, VertexElementUsagePosition, 0),
				NewVertexElement(12, VertexElementFormatColor, VertexElementUsageColor, 0),
				NewVertexElement(16, VertexElementFormatVector2, VertexElementUsageTextureCoordinate, 0),
			},
		},
		{
			name:        "VertexPositionNormalTexture",
			declaration: VertexPositionNormalTextureVertexDeclaration(),
			stride:      32,
			elements: []VertexElement{
				NewVertexElement(0, VertexElementFormatVector3, VertexElementUsagePosition, 0),
				NewVertexElement(12, VertexElementFormatVector3, VertexElementUsageNormal, 0),
				NewVertexElement(24, VertexElementFormatVector2, VertexElementUsageTextureCoordinate, 0),
			},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.declaration == nil {
				t.Fatal("the static declaration is nil, so its element table was refused")
			}
			if got := testCase.declaration.VertexStride(); got != testCase.stride {
				t.Fatalf("VertexStride = %d, want %d", got, testCase.stride)
			}
			elements := testCase.declaration.GetVertexElements()
			if len(elements) != len(testCase.elements) {
				t.Fatalf("%d elements, want %d", len(elements), len(testCase.elements))
			}
			for i, want := range testCase.elements {
				got := elements[i]
				if got.Offset() != want.Offset() || got.VertexElementFormat() != want.VertexElementFormat() ||
					got.VertexElementUsage() != want.VertexElementUsage() || got.UsageIndex() != want.UsageIndex() {
					t.Fatalf("element %d = %s, want %s", i, got.ToString(), want.ToString())
				}
			}
			// The `.cctor` names the declaration after itself.
			if got := testCase.declaration.Name(); got != testCase.name+".VertexDeclaration" {
				t.Fatalf("Name = %q", got)
			}
		})
	}
}

// TestVertexDeclarationsAreTheSameObjectEveryCall is the static-readonly-field
// claim: the CLR `.cctor` runs once and every read of the field answers the same
// object, and the two readers -- the static field and the IVertexType witness --
// answer the SAME one.
func TestVertexDeclarationsAreTheSameObjectEveryCall(t *testing.T) {
	if VertexPositionColorVertexDeclaration() != VertexPositionColorVertexDeclaration() {
		t.Fatal("the static declaration was rebuilt")
	}
	if (VertexPositionColor{}).VertexDeclaration() != VertexPositionColorVertexDeclaration() {
		t.Fatal("the IVertexType witness answered a different declaration from the static field")
	}
	if (VertexPositionTexture{}).VertexDeclaration() != VertexPositionTextureVertexDeclaration() ||
		(VertexPositionColorTexture{}).VertexDeclaration() != VertexPositionColorTextureVertexDeclaration() ||
		(VertexPositionNormalTexture{}).VertexDeclaration() != VertexPositionNormalTextureVertexDeclaration() {
		t.Fatal("a witness and its static field disagree")
	}
	// Four distinct declarations, not one shared object.
	seen := map[*VertexDeclaration]bool{
		VertexPositionColorVertexDeclaration():         true,
		VertexPositionTextureVertexDeclaration():       true,
		VertexPositionColorTextureVertexDeclaration():  true,
		VertexPositionNormalTextureVertexDeclaration(): true,
	}
	if len(seen) != 4 {
		t.Fatalf("the four types share %d declarations", len(seen))
	}
}

// TestVertexStructHashCodesMatchTheSmartHelper pins values computed from
// Helpers.SmartGetHashCode's rule rather than from this implementation: XOR
// every complete 32-bit word of the marshalled struct, and substitute
// Int32.MaxValue for a zero result.
func TestVertexStructHashCodesMatchTheSmartHelper(t *testing.T) {
	red := framework.NewColorByInt32AndInt32AndInt32(255, 0, 0)
	if got := NewVertexPositionColor(vector3(1, 2, 3), red).GetHashCode(); got != -1061158657 {
		t.Fatalf("VertexPositionColor hash = %d", got)
	}
	if got := NewVertexPositionTexture(vector3(1, 2, 3), vector2(0.25, 0.5)).GetHashCode(); got != 1044381696 {
		t.Fatalf("VertexPositionTexture hash = %d", got)
	}
	mixed := framework.NewColorByInt32AndInt32AndInt32AndInt32(10, 20, 30, 40)
	if got := NewVertexPositionColorTexture(vector3(1, 2, 3), mixed, vector2(0.25, 0.5)).GetHashCode(); got != 375264266 {
		t.Fatalf("VertexPositionColorTexture hash = %d", got)
	}
	if got := NewVertexPositionNormalTexture(vector3(1, 2, 3), vector3(0, 1, 0), vector2(0.25, 0.5)).GetHashCode(); got != 29360128 {
		t.Fatalf("VertexPositionNormalTexture hash = %d", got)
	}
	// The zero substitution. An all-zero VertexPositionColor XORs to zero, and
	// the helper answers Int32.MaxValue rather than zero.
	if got := (VertexPositionColor{}).GetHashCode(); got != 2147483647 {
		t.Fatalf("the zero vertex hashed to %d, want the Int32.MaxValue substitution", got)
	}
}

func TestVertexStructEqualityAndToString(t *testing.T) {
	red := framework.NewColorByInt32AndInt32AndInt32(255, 0, 0)
	blue := framework.NewColorByInt32AndInt32AndInt32(0, 0, 255)
	left := NewVertexPositionColor(vector3(1, 2, 3), red)
	same := NewVertexPositionColor(vector3(1, 2, 3), red)
	otherColour := NewVertexPositionColor(vector3(1, 2, 3), blue)
	otherPosition := NewVertexPositionColor(vector3(9, 2, 3), red)

	if !VertexPositionColorOperatorEqualityByVertexPositionColorAndVertexPositionColor(left, same) {
		t.Fatal("two identical vertices are not equal")
	}
	if VertexPositionColorOperatorEqualityByVertexPositionColorAndVertexPositionColor(left, otherColour) {
		t.Fatal("a colour change did not break equality")
	}
	if VertexPositionColorOperatorEqualityByVertexPositionColorAndVertexPositionColor(left, otherPosition) {
		t.Fatal("a position change did not break equality")
	}
	if !VertexPositionColorOperatorInequalityByVertexPositionColorAndVertexPositionColor(left, otherColour) {
		t.Fatal("op_Inequality is not the negation of op_Equality")
	}
	if !left.Equals(same) || left.Equals(otherColour) {
		t.Fatal("Equals disagrees with op_Equality")
	}
	// Equals is a TYPE test before it is a field test: a different vertex type
	// with the same position is never equal.
	if left.Equals(NewVertexPositionTexture(vector3(1, 2, 3), vector2(0, 0))) {
		t.Fatal("Equals matched a different vertex type")
	}
	if left.Equals(nil) || left.Equals("not a vertex") {
		t.Fatal("Equals matched a non-vertex")
	}

	want := "{Position:" + vector3(1, 2, 3).ToString() + " Color:" + red.ToString() + "}"
	if got := left.ToString(); got != want {
		t.Fatalf("ToString = %q, want %q", got, want)
	}
	texture := NewVertexPositionNormalTexture(vector3(1, 2, 3), vector3(0, 1, 0), vector2(0.25, 0.5))
	wantNormal := "{Position:" + vector3(1, 2, 3).ToString() +
		" Normal:" + vector3(0, 1, 0).ToString() +
		" TextureCoordinate:" + vector2(0.25, 0.5).ToString() + "}"
	if got := texture.ToString(); got != wantNormal {
		t.Fatalf("ToString = %q, want %q", got, wantNormal)
	}
}

// TestVertexStructsResolveThroughFromType is what makes them usable at a draw
// call: VertexDeclaration.FromType finds the declaration by reflecting over the
// type's IVertexType implementation, and every one of the four must resolve.
func TestVertexStructsResolveThroughFromType(t *testing.T) {
	cases := map[string]struct {
		instance IVertexType
		want     *VertexDeclaration
	}{
		"VertexPositionColor":         {VertexPositionColor{}, VertexPositionColorVertexDeclaration()},
		"VertexPositionTexture":       {VertexPositionTexture{}, VertexPositionTextureVertexDeclaration()},
		"VertexPositionColorTexture":  {VertexPositionColorTexture{}, VertexPositionColorTextureVertexDeclaration()},
		"VertexPositionNormalTexture": {VertexPositionNormalTexture{}, VertexPositionNormalTextureVertexDeclaration()},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			if testCase.instance.VertexDeclaration() != testCase.want {
				t.Fatal("the interface value did not answer the type's own declaration")
			}
		})
	}
}
