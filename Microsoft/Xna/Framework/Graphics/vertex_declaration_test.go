package graphics

import (
	"errors"
	"strings"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// element is the corpus-local shorthand. Every test below builds declarations
// out of it, so a reader can see the layout rather than four constructor
// arguments.
func element(offset int32, format VertexElementFormat, usage VertexElementUsage, usageIndex int32) VertexElement {
	return NewVertexElement(offset, format, usage, usageIndex)
}

// positionColour is the smallest declaration that is legal in the reference:
// a Vector3 position at 0 and a Color at 12, stride 16.
func positionColour() []VertexElement {
	return []VertexElement{
		element(0, VertexElementFormatVector3, VertexElementUsagePosition, 0),
		element(12, VertexElementFormatColor, VertexElementUsageColor, 0),
	}
}

// TestTheComputedStrideIsTheLargestEndOffset pins
// VertexElementValidator::GetVertexStride, which is a MAXIMUM rather than a
// sum. A declaration with a gap strides over the gap, and a projection that
// summed the sizes would produce 16 for the gapped layout below instead of 32.
func TestTheComputedStrideIsTheLargestEndOffset(t *testing.T) {
	declaration, err := NewVertexDeclarationBySliceOfVertexElement(positionColour())
	if err != nil {
		t.Fatalf("NewVertexDeclarationBySliceOfVertexElement: %v", err)
	}
	if got := declaration.VertexStride(); got != 16 {
		t.Fatalf("computed stride = %d, want 12 + 4", got)
	}

	gapped := []VertexElement{
		element(0, VertexElementFormatVector3, VertexElementUsagePosition, 0),
		element(28, VertexElementFormatSingle, VertexElementUsageFog, 0),
	}
	declaration, err = NewVertexDeclarationBySliceOfVertexElement(gapped)
	if err != nil {
		t.Fatalf("gapped declaration: %v", err)
	}
	if got := declaration.VertexStride(); got != 32 {
		t.Fatalf("gapped stride = %d, want the largest END offset, 28 + 4", got)
	}
}

// TestBothConstructorsRefuseAnEmptyElementArray pins the branch a null and an
// empty array SHARE in the reference: both `brtrue` tests fall through to the
// same ArgumentNullException("elements", NullNotAllowed) throw.
func TestBothConstructorsRefuseAnEmptyElementArray(t *testing.T) {
	for name, build := range map[string]func() (*VertexDeclaration, error){
		"explicit stride": func() (*VertexDeclaration, error) {
			return NewVertexDeclarationByInt32AndSliceOfVertexElement(16, nil)
		},
		"computed stride": func() (*VertexDeclaration, error) {
			return NewVertexDeclarationBySliceOfVertexElement([]VertexElement{})
		},
	} {
		declaration, err := build()
		if err == nil {
			t.Fatalf("%s: an empty element array was accepted", name)
		}
		if declaration != nil {
			t.Fatalf("%s: a declaration was returned alongside the error", name)
		}
		if !strings.Contains(err.Error(), nullNotAllowed) {
			t.Fatalf("%s: %v, want the reference's NullNotAllowed message", name, err)
		}
	}
}

// TestTheElementArrayIsClonedInAndOut pins both `Array.Clone()` calls. A
// projection that stored the caller's slice would let a caller invalidate a
// declaration the validator already accepted, and one that returned the stored
// slice would let them do it afterwards.
func TestTheElementArrayIsClonedInAndOut(t *testing.T) {
	source := positionColour()
	declaration, err := NewVertexDeclarationByInt32AndSliceOfVertexElement(16, source)
	if err != nil {
		t.Fatalf("NewVertexDeclarationByInt32AndSliceOfVertexElement: %v", err)
	}
	// Mutating the array the caller passed must not reach the declaration.
	source[0].SetOffset(4)
	if got := declaration.GetVertexElements()[0].Offset(); got != 0 {
		t.Fatalf("element offset = %d after mutating the caller's array; the constructor clones", got)
	}
	// Mutating the array the getter returned must not reach it either.
	returned := declaration.GetVertexElements()
	returned[1].SetUsageIndex(7)
	if got := declaration.GetVertexElements()[1].UsageIndex(); got != 0 {
		t.Fatalf("usage index = %d after mutating a returned array; GetVertexElements clones", got)
	}
	if &returned[0] == &declaration.GetVertexElements()[0] {
		t.Fatal("two GetVertexElements calls shared an array")
	}
}

// TestTheValidatorRejectsEachDefectWithTheReferenceMessage is the milestone's
// centre. Each case is one check in VertexElementValidator::Validate, in the
// order the reference performs them, with the FrameworkResources sentence it
// throws.
func TestTheValidatorRejectsEachDefectWithTheReferenceMessage(t *testing.T) {
	cases := []struct {
		name     string
		stride   int32
		elements []VertexElement
		message  string
	}{
		{
			name:     "a non-positive stride",
			stride:   0,
			elements: positionColour(),
			message:  "vertexStride",
		},
		{
			name:     "a stride that is not a multiple of four",
			stride:   18,
			elements: positionColour(),
			message:  vertexElementOffsetNotMultipleFour,
		},
		{
			name:   "a usage outside the declared range",
			stride: 16,
			elements: []VertexElement{
				element(0, VertexElementFormatVector3, VertexElementUsage(13), 0),
			},
			message: "Usage 13 is out of range.",
		},
		{
			name:   "an element that does not fit the stride",
			stride: 8,
			elements: []VertexElement{
				element(0, VertexElementFormatVector3, VertexElementUsagePosition, 0),
			},
			message: "Element Position0 does not fit within the specified vertex stride.",
		},
		{
			name:   "an element offset that is not a multiple of four",
			stride: 16,
			elements: []VertexElement{
				element(2, VertexElementFormatSingle, VertexElementUsagePosition, 0),
			},
			message: vertexElementOffsetNotMultipleFour,
		},
		{
			name:   "two elements with the same usage and index",
			stride: 16,
			elements: []VertexElement{
				element(0, VertexElementFormatSingle, VertexElementUsagePosition, 0),
				element(4, VertexElementFormatSingle, VertexElementUsagePosition, 0),
			},
			message: "Duplicate element Position0.",
		},
		{
			name:   "two elements claiming the same byte",
			stride: 16,
			elements: []VertexElement{
				element(0, VertexElementFormatVector2, VertexElementUsagePosition, 0),
				element(4, VertexElementFormatSingle, VertexElementUsageFog, 0),
			},
			message: "Elements Position0 and Fog0 are overlapping.",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			declaration, err := NewVertexDeclarationByInt32AndSliceOfVertexElement(testCase.stride, testCase.elements)
			if err == nil {
				t.Fatal("the validator accepted the declaration")
			}
			if declaration != nil {
				t.Fatal("a declaration was returned alongside the error")
			}
			if !strings.Contains(err.Error(), testCase.message) {
				t.Fatalf("%v, want the reference's %q", err, testCase.message)
			}
		})
	}
}

// TestTheStrideFitBoundIsExclusiveAtExactlyOneByteOver pins the `>` in
//
//	if (offset < 0 || offset + size > vertexStride) throw ...
//
// at the only input that can tell it from `>= vertexStride + 1`. Offsets and
// sizes are otherwise multiples of four, so no legal layout ever lands one byte
// over -- but the offset check comes AFTER this one, so an element at offset 1
// reaches here unfiltered. A projection whose bound was one byte loose would
// let it through and report the WRONG failure: "must be multiples of four"
// instead of "does not fit within the specified vertex stride".
func TestTheStrideFitBoundIsExclusiveAtExactlyOneByteOver(t *testing.T) {
	_, err := NewVertexDeclarationByInt32AndSliceOfVertexElement(4, []VertexElement{
		element(1, VertexElementFormatSingle, VertexElementUsagePosition, 0),
	})
	if err == nil {
		t.Fatal("an element ending one byte past the stride was accepted")
	}
	if !strings.Contains(err.Error(), "does not fit within the specified vertex stride.") {
		t.Fatalf("%v, want the stride failure: the fit check runs before the offset check", err)
	}
}

// TestTheValidatorReportsTheEARLIEROverlappingElementFirst pins the occupancy
// map's payload. The reference fills an `int[stride]` with the INDEX of the
// element that claimed each byte, which is the only reason the message can name
// both elements -- and it names the one already there first.
func TestTheValidatorReportsTheEARLIEROverlappingElementFirst(t *testing.T) {
	_, err := NewVertexDeclarationByInt32AndSliceOfVertexElement(16, []VertexElement{
		element(0, VertexElementFormatVector4, VertexElementUsageNormal, 3),
		element(8, VertexElementFormatSingle, VertexElementUsageTangent, 1),
	})
	if err == nil {
		t.Fatal("overlapping elements were accepted")
	}
	if !strings.Contains(err.Error(), "Elements Normal3 and Tangent1 are overlapping.") {
		t.Fatalf("%v, want the earlier element named first", err)
	}
}

// TestTheUsageCheckRunsBeforeTheStrideCheck pins the ORDER, which is not
// cosmetic: an element that is both badly-used and outside the stride must
// report the usage, because a caller who fixed only the second failure would
// still be refused.
func TestTheUsageCheckRunsBeforeTheStrideCheck(t *testing.T) {
	_, err := NewVertexDeclarationByInt32AndSliceOfVertexElement(4, []VertexElement{
		element(0, VertexElementFormatVector4, VertexElementUsage(-1), 0),
	})
	if err == nil {
		t.Fatal("an element that is both badly-used and oversized was accepted")
	}
	if !strings.Contains(err.Error(), "is out of range.") {
		t.Fatalf("%v, want the usage failure the reference reports first", err)
	}
}

// TestAnUnknownFormatOccupiesNoBytes pins GetTypeSize's `ldc.i4.0` default. It
// is not a failure: the element takes no space, so it neither overflows the
// stride nor overlaps anything. Reproducing it as a rejection would refuse a
// declaration the reference accepts.
func TestAnUnknownFormatOccupiesNoBytes(t *testing.T) {
	declaration, err := NewVertexDeclarationByInt32AndSliceOfVertexElement(4, []VertexElement{
		element(0, VertexElementFormat(99), VertexElementUsagePosition, 0),
		element(0, VertexElementFormatSingle, VertexElementUsageFog, 0),
	})
	if err != nil {
		t.Fatalf("a zero-size element was refused: %v", err)
	}
	if got := declaration.VertexStride(); got != 4 {
		t.Fatalf("stride = %d, want the explicit 4", got)
	}
	computed, err := NewVertexDeclarationBySliceOfVertexElement([]VertexElement{
		element(0, VertexElementFormat(99), VertexElementUsagePosition, 0),
		element(0, VertexElementFormatSingle, VertexElementUsageFog, 0),
	})
	if err != nil {
		t.Fatalf("computing a stride over a zero-size element: %v", err)
	}
	if got := computed.VertexStride(); got != 4 {
		t.Fatalf("computed stride = %d; a zero-size element contributes its offset only", got)
	}
}

// TestAComputedStrideIsStillValidated pins that the second constructor is not a
// shortcut: the stride it computes goes through the same checks, so a layout
// whose largest end offset is not a multiple of four is refused.
func TestAComputedStrideIsStillValidated(t *testing.T) {
	_, err := NewVertexDeclarationBySliceOfVertexElement([]VertexElement{
		element(0, VertexElementFormatVector3, VertexElementUsagePosition, 0),
		element(12, VertexElementFormat(99), VertexElementUsageFog, 0),
	})
	// The computed stride is 12, which IS a multiple of four, so this must be
	// accepted -- the negative case is the offset check below.
	if err != nil {
		t.Fatalf("a valid computed stride was refused: %v", err)
	}
	_, err = NewVertexDeclarationBySliceOfVertexElement([]VertexElement{
		element(2, VertexElementFormatSingle, VertexElementUsagePosition, 0),
	})
	if err == nil {
		t.Fatal("the computed-stride constructor skipped the offset check")
	}
	if !strings.Contains(err.Error(), vertexElementOffsetNotMultipleFour) {
		t.Fatalf("%v, want the multiple-of-four refusal", err)
	}
}

// TestTheInheritedGraphicsResourceSurfaceIsPresent pins the nine members a
// consumer holding a VertexDeclaration really has in C#, and the two answers
// that are specific to this type: the device is NIL because `_parent` is
// assigned only by the internal Bind, and ToString falls back to THIS type's
// runtime CLR name through the composed base's `this`.
func TestTheInheritedGraphicsResourceSurfaceIsPresent(t *testing.T) {
	declaration, err := NewVertexDeclarationBySliceOfVertexElement(positionColour())
	if err != nil {
		t.Fatalf("NewVertexDeclarationBySliceOfVertexElement: %v", err)
	}
	if declaration.GraphicsDevice() != nil {
		t.Fatal("a constructed declaration reports a GraphicsDevice; the reference's _parent is set only by the internal Bind")
	}
	if declaration.IsDisposed() {
		t.Fatal("a fresh declaration reports disposed")
	}
	if got := declaration.ToString(); got != "Microsoft.Xna.Framework.Graphics.VertexDeclaration" {
		t.Fatalf("ToString = %q; the CLR `this` must reach the outermost object", got)
	}
	declaration.SetName("layout")
	declaration.SetTag(42)
	if declaration.Name() != "layout" || declaration.Tag() != 42 {
		t.Fatalf("Name/Tag = %q/%v", declaration.Name(), declaration.Tag())
	}
	if got := declaration.ToString(); got != "layout" {
		t.Fatalf("ToString = %q, want the Name once it has one", got)
	}
}

// TestDisposalIsIdempotentAndRaisesDisposingOnce pins the inherited disposal:
// VertexDeclaration's own override adds nothing but Unbind, whose one branch is
// unreachable from public surface, so the base's flag and event are the whole
// observable behaviour.
func TestDisposalIsIdempotentAndRaisesDisposingOnce(t *testing.T) {
	declaration, err := NewVertexDeclarationBySliceOfVertexElement(positionColour())
	if err != nil {
		t.Fatalf("NewVertexDeclarationBySliceOfVertexElement: %v", err)
	}
	raises := 0
	var sender any
	var observedDisposed bool
	if _, err := declaration.AddDisposingHandler(func(s any, _ *framework.EventArgs) error {
		raises++
		sender = s
		observedDisposed = declaration.IsDisposed()
		return nil
	}); err != nil {
		t.Fatalf("AddDisposingHandler: %v", err)
	}
	if err := declaration.DisposeByNone(); err != nil {
		t.Fatalf("DisposeByNone: %v", err)
	}
	if err := declaration.DisposeByNone(); err != nil {
		t.Fatalf("a second DisposeByNone: %v", err)
	}
	if raises != 1 {
		t.Fatalf("Disposing raised %d times, want once: GraphicsResource's isDisposed flag makes it idempotent", raises)
	}
	if sender != any(declaration) {
		t.Fatal("the Disposing sender is not the declaration; the composed base must raise with the CLR `this`")
	}
	if !observedDisposed {
		t.Fatal("the handler observed IsDisposed == false; the flag is set before the event is raised")
	}
	// The two members with no disposal check still answer, because the
	// reference's getters have none either.
	if declaration.VertexStride() != 16 || len(declaration.GetVertexElements()) != 2 {
		t.Fatal("a disposed declaration stopped answering; neither getter checks disposal")
	}
}

// TestAZeroDeclarationIsRefusedRatherThanPanicking covers the Go-only guard.
func TestAZeroDeclarationIsRefusedRatherThanPanicking(t *testing.T) {
	var absent *VertexDeclaration
	if absent.VertexStride() != 0 || absent.GetVertexElements() != nil {
		t.Fatal("a nil declaration answered with values")
	}
	if !absent.IsDisposed() {
		t.Fatal("a nil declaration reports not disposed")
	}
	if err := absent.DisposeByNone(); !errors.Is(err, errVertexDeclarationNil) {
		t.Fatalf("DisposeByNone on nil = %v, want the uninitialized refusal", err)
	}
	if _, err := absent.AddDisposingHandler(nil); !errors.Is(err, errVertexDeclarationNil) {
		t.Fatalf("AddDisposingHandler on nil = %v", err)
	}
}
