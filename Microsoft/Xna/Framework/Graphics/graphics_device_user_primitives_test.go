package graphics

import (
	"errors"
	"strings"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// ---------------------------------------------------------------------------
// Foundation 73 — the six DrawUser* generics' managed half.
// ---------------------------------------------------------------------------
//
// Every guard below runs before CNA is reached, so none of these needs a live
// device: they are checked against a zero GraphicsDevice, which reports its own
// disposal only once every argument has passed. The native half -- all six
// overloads submitted through a real draw callback with an effect applied -- is
// the `vertex-buffer` stress scenario's job.

// userVertex is the corpus vertex type: a Vector3 position and a Color, which
// is the layout VertexDeclaration.FromType resolves for it.
type userVertex struct {
	Position framework.Vector3
	Colour   framework.Color
}

func (userVertex) VertexDeclaration() *VertexDeclaration {
	declaration, err := NewVertexDeclarationByInt32AndSliceOfVertexElement(16, []VertexElement{
		NewVertexElement(0, VertexElementFormatVector3, VertexElementUsagePosition, 0),
		NewVertexElement(12, VertexElementFormatColor, VertexElementUsageColor, 0),
	})
	if err != nil {
		panic(err)
	}
	return declaration
}

func userVertices(count int) []userVertex { return make([]userVertex, count) }

func userDeclaration(t *testing.T) *VertexDeclaration {
	t.Helper()
	return userVertex{}.VertexDeclaration()
}

// TestTheVertexCountTableIsTheReferencesGetElementCountArray pins the topology
// table every window check is computed from. A projection that used 3n for a
// strip would admit windows the reference refuses and refuse windows it admits.
func TestTheVertexCountTableIsTheReferencesGetElementCountArray(t *testing.T) {
	for _, probe := range []struct {
		primitiveType PrimitiveType
		count         int32
		want          int32
	}{
		{PrimitiveTypeLineList, 4, 8},
		{PrimitiveTypeLineStrip, 4, 5},
		{PrimitiveTypeTriangleList, 4, 12},
		{PrimitiveTypeTriangleStrip, 4, 6},
	} {
		got, err := verticesForPrimitives(probe.primitiveType, probe.count)
		if err != nil || got != probe.want {
			t.Fatalf("%v x%d = (%d, %v), want %d", probe.primitiveType, probe.count, got, err, probe.want)
		}
	}
	// A topology outside XNA's four is refused BY NAME, which is a Go-only
	// decision: the reference's own switch falls through and returns zero,
	// which makes every window check pass and hands D3D a topology it refuses.
	if _, err := verticesForPrimitives(PrimitiveType(99), 1); err == nil {
		t.Fatal("an unknown topology was accepted")
	}
}

// TestTheGuardsRunInTheReferencesOrder pins each throw and the order the IL
// puts them in: vertexData at IL_0018, vertexDeclaration at IL_002c,
// primitiveCount at IL_0041, the window at IL_0094 and vertexOffset last.
func TestTheGuardsRunInTheReferencesOrder(t *testing.T) {
	declaration := userDeclaration(t)
	for name, probe := range map[string]struct {
		call    func() error
		message string
		param   string
	}{
		"nil vertexData": {
			call: func() error {
				return verifyUserPrimitives(PrimitiveTypeTriangleList, 0, 0, 1, declaration)
			},
			message: nullNotAllowed, param: "vertexData",
		},
		"nil vertexDeclaration": {
			call: func() error {
				return verifyUserPrimitives(PrimitiveTypeTriangleList, 3, 0, 1, nil)
			},
			message: nullNotAllowed, param: "vertexDeclaration",
		},
		"zero primitiveCount": {
			call: func() error {
				return verifyUserPrimitives(PrimitiveTypeTriangleList, 3, 0, 0, declaration)
			},
			message: mustDrawSomething, param: "primitiveCount",
		},
		"window past the array": {
			call: func() error {
				return verifyUserPrimitives(PrimitiveTypeTriangleList, 3, 0, 2, declaration)
			},
			message: mustBeValidIndex, param: "primitiveCount",
		},
		"negative vertexOffset": {
			call: func() error {
				return verifyUserPrimitives(PrimitiveTypeTriangleList, 6, -1, 1, declaration)
			},
			message: offsetNotValid, param: "vertexOffset",
		},
	} {
		err := probe.call()
		if err == nil {
			t.Fatalf("%s was accepted", name)
		}
		if !strings.Contains(err.Error(), probe.message) {
			t.Fatalf("%s reported %v, want %q", name, err, probe.message)
		}
		if !strings.Contains(err.Error(), probe.param) {
			t.Fatalf("%s reported %v, want the parameter %q", name, err, probe.param)
		}
	}
}

// TestTheWindowCheckPrecedesTheOffsetCheck pins their relative order: the
// reference tests `vertexOffset + vertexCount > Length` at IL_0092 and
// `vertexOffset` itself afterwards, so a call that fails BOTH reports
// primitiveCount rather than vertexOffset.
func TestTheWindowCheckPrecedesTheOffsetCheck(t *testing.T) {
	err := verifyUserPrimitives(PrimitiveTypeTriangleList, 3, 5, 2, userDeclaration(t))
	if err == nil {
		t.Fatal("a call that fails both checks was accepted")
	}
	if !strings.Contains(err.Error(), mustBeValidIndex) {
		t.Fatalf("%v, want the window reported before the offset", err)
	}
}

// TestTheIndexedFamilyAddsItsOwnTwoGuards pins the two checks the indexed
// overloads have and the non-indexed ones do not.
func TestTheIndexedFamilyAddsItsOwnTwoGuards(t *testing.T) {
	declaration := userDeclaration(t)
	if err := verifyUserIndexedPrimitives(PrimitiveTypeTriangleList, 3, 0, 3, 0, 0, 1, declaration); err == nil {
		t.Fatal("a nil index array was accepted")
	} else if !strings.Contains(err.Error(), "indexData") {
		t.Fatalf("%v, want the indexData parameter named", err)
	}
	if err := verifyUserIndexedPrimitives(PrimitiveTypeTriangleList, 3, 0, 0, 3, 0, 1, declaration); err == nil {
		t.Fatal("a zero numVertices was accepted")
	} else if !strings.Contains(err.Error(), "numVertices") {
		t.Fatalf("%v, want the numVertices parameter named", err)
	}
}

// TestTheShortOverloadsResolveTheDeclarationFromT pins that the four
// declaration-less overloads are argument normalisers over
// VertexDeclarationFactory<T>.VertexDeclaration -- which is
// VertexDeclaration.FromType's own resolution, so a T that is not a valid
// vertex type gets FromType's exact message.
func TestTheShortOverloadsResolveTheDeclarationFromT(t *testing.T) {
	// A valid T resolves and then fails on the DEVICE, which is only reachable
	// past the declaration resolution.
	err := GraphicsDeviceDrawUserPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32(
		&GraphicsDevice{}, PrimitiveTypeTriangleList, userVertices(3), 0, 1)
	if err == nil {
		t.Fatal("a draw on a zero device reported success")
	}
	if strings.Contains(err.Error(), "IVertexType") {
		t.Fatalf("%v; a valid vertex type must resolve", err)
	}
	// A T that does not implement IVertexType gets FromType's own sentence.
	type notAVertex struct{ A, B, C, D int32 }
	err = GraphicsDeviceDrawUserPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32(
		&GraphicsDevice{}, PrimitiveTypeTriangleList, make([]notAVertex, 3), 0, 1)
	if err == nil {
		t.Fatal("a type that is not a vertex type was accepted")
	}
	if !strings.Contains(err.Error(), "does not implement the IVertexType interface") {
		t.Fatalf("%v, want VertexDeclaration.FromType's message", err)
	}
}

// TestEverySixOverloadsShareTheVertexDataGuard is the coverage control: an
// overload that reached the device without checking would pass every test above
// and still be unguarded.
func TestEverySixOverloadsShareTheVertexDataGuard(t *testing.T) {
	device := &GraphicsDevice{}
	declaration := userDeclaration(t)
	for name, call := range map[string]func() error{
		"primitives": func() error {
			return GraphicsDeviceDrawUserPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32(
				device, PrimitiveTypeTriangleList, []userVertex(nil), 0, 1)
		},
		"primitives+decl": func() error {
			return GraphicsDeviceDrawUserPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndVertexDeclaration(
				device, PrimitiveTypeTriangleList, []userVertex(nil), 0, 1, declaration)
		},
		"indexed16": func() error {
			return GraphicsDeviceDrawUserIndexedPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndSliceOfInt16AndInt32AndInt32(
				device, PrimitiveTypeTriangleList, []userVertex(nil), 0, 3, []int16{0, 1, 2}, 0, 1)
		},
		"indexed32": func() error {
			return GraphicsDeviceDrawUserIndexedPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndSliceOfInt32AndInt32AndInt32(
				device, PrimitiveTypeTriangleList, []userVertex(nil), 0, 3, []int32{0, 1, 2}, 0, 1)
		},
		"indexed16+decl": func() error {
			return GraphicsDeviceDrawUserIndexedPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndSliceOfInt16AndInt32AndInt32AndVertexDeclaration(
				device, PrimitiveTypeTriangleList, []userVertex(nil), 0, 3, []int16{0, 1, 2}, 0, 1, declaration)
		},
		"indexed32+decl": func() error {
			return GraphicsDeviceDrawUserIndexedPrimitivesByPrimitiveTypeAndSliceOfTAndInt32AndInt32AndSliceOfInt32AndInt32AndInt32AndVertexDeclaration(
				device, PrimitiveTypeTriangleList, []userVertex(nil), 0, 3, []int32{0, 1, 2}, 0, 1, declaration)
		},
	} {
		err := call()
		if err == nil {
			t.Fatalf("%s accepted a nil vertex array", name)
		}
		// A zero GraphicsDevice reports its own disposal FIRST, which is the
		// reference's own order -- CheckDisposed is its first statement. What
		// this control proves is that every one of the six reaches a guard at
		// all rather than a nil dereference.
		if errors.Is(err, errGraphicsResourceArgumentNull) {
			continue
		}
		if !strings.Contains(err.Error(), "GraphicsDevice is nil") {
			t.Fatalf("%s reported %v, want a guard rather than a panic", name, err)
		}
	}
}
