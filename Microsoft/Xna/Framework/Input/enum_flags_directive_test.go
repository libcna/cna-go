package input

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// flagsDirectiveAt reports whether the named type declaration inside file
// carries the `xna:flags` directive. It measures the real declaration site
// rather than an extractor model field, so an enum whose flags projection
// silently changes in source is rejected by the owning package's own tests.
func flagsDirectiveAt(t *testing.T, file, typeName string) bool {
	t.Helper()
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, file, nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var doc []*ast.CommentGroup
	found := false
	for _, decl := range parsed.Decls {
		general, ok := decl.(*ast.GenDecl)
		if !ok || general.Tok != token.TYPE {
			continue
		}
		for _, spec := range general.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != typeName {
				continue
			}
			found = true
			doc = append(doc, typeSpec.Doc, general.Doc)
		}
	}
	if !found {
		t.Fatalf("%s type declaration was not found in %s", typeName, file)
	}
	return hasFlagsDirective(doc)
}

// TestFlagsDirectiveDetector is the negative fixture for the detector itself:
// a detector that accepted anything would make every ordinary-enum flags
// assertion in this package vacuous.
func TestFlagsDirectiveDetector(t *testing.T) {
	if hasFlagsDirective(nil) {
		t.Fatal("the detector reported a directive for no comment groups")
	}
	if hasFlagsDirective([]*ast.CommentGroup{nil}) {
		t.Fatal("the detector reported a directive for a nil comment group")
	}
	exact := []*ast.CommentGroup{{List: []*ast.Comment{{Text: "// xna:flags"}}}}
	if !hasFlagsDirective(exact) {
		t.Fatal("the detector missed an exact xna:flags directive")
	}
	for _, text := range []string{"// xna:flags=false", "// not-xna:flags", "// comment mentioning xna:flags", "//xna:flags"} {
		near := []*ast.CommentGroup{{List: []*ast.Comment{{Text: text}}}}
		if hasFlagsDirective(near) {
			t.Fatalf("the detector accepted %q as an xna:flags directive", text)
		}
	}
}
