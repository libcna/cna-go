package input

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"testing"
)

func TestButtonStateCompleteRawValueTable(t *testing.T) {
	values := []struct {
		name  string
		value ButtonState
		want  int32
	}{
		{"Released", ButtonStateReleased, 0},
		{"Pressed", ButtonStatePressed, 1},
	}
	for _, item := range values {
		if got := int32(item.value); got != item.want {
			t.Errorf("ButtonState%s = %d, want %d", item.name, got, item.want)
		}
	}
	if got := reflect.TypeOf(ButtonStateReleased).Kind(); got != reflect.Int32 {
		t.Fatalf("ButtonState underlying kind = %s, want int32", got)
	}
}

func TestButtonStateZeroAndArbitraryRawValues(t *testing.T) {
	var zero ButtonState
	if zero != ButtonStateReleased {
		t.Fatalf("zero ButtonState = %d, want Released (%d)", zero, ButtonStateReleased)
	}
	for _, raw := range []int32{2, 12345, -1} {
		if got := int32(ButtonState(raw)); got != raw {
			t.Fatalf("ButtonState(%d) = %d", raw, got)
		}
	}
}

// TestButtonStateSourceCarriesNoFlagsDirective measures the real declaration
// site rather than an extractor model field: the ordinary-enum projection is
// only correct while no `xna:flags` directive is attached to the type.
func TestButtonStateSourceCarriesNoFlagsDirective(t *testing.T) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "button_state.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var doc []*ast.CommentGroup
	found := false
	for _, decl := range file.Decls {
		general, ok := decl.(*ast.GenDecl)
		if !ok || general.Tok != token.TYPE {
			continue
		}
		for _, spec := range general.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != "ButtonState" {
				continue
			}
			found = true
			doc = append(doc, typeSpec.Doc, general.Doc)
		}
	}
	if !found {
		t.Fatal("ButtonState type declaration was not found in button_state.go")
	}
	if hasFlagsDirective(doc) {
		t.Fatal("ButtonState carries an xna:flags directive")
	}
	mutated := append(doc, &ast.CommentGroup{List: []*ast.Comment{{Text: "// xna:flags"}}})
	if !hasFlagsDirective(mutated) {
		t.Fatal("the flags-directive detector failed to reject a mutated ButtonState declaration")
	}
}

func hasFlagsDirective(groups []*ast.CommentGroup) bool {
	for _, group := range groups {
		if group == nil {
			continue
		}
		for _, comment := range group.List {
			if comment.Text == "// xna:flags" {
				return true
			}
		}
	}
	return false
}
