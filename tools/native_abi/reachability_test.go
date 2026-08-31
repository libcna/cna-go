package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A route that is bound and never called is not evidence of anything. It costs
// a dlsym, a trampoline, a _Static_assert and a line in the ABI report, and it
// makes BOUND_FUNCTIONS report a boundary wider than the one CNA-Go actually
// exercises.
//
// Foundation 49 produced exactly that. cna_graphics_device_manager_dispose was
// bound in Foundation 48; after a SIGSEGV during run teardown its single call
// site was removed, and the typedef, the X-macro entry, the trampoline and
// interop.ManagerDispose all stayed behind. Nothing failed, because nothing
// measured the chain from a bound route to a caller outside the interop
// package.
//
// This test measures that chain, following the source rather than a naming
// convention:
//
//	route        cna_foo               abi_manifest.h X-macro entry
//	trampoline   the C function whose body calls api.cna_foo(...)
//	cgo call     the Go function whose body calls C.<that trampoline>(...)
//	reachability that Go function must be reachable, inside package interop,
//	             from something the rest of the module actually names
//
// The roots of that reachability are what the module names from outside:
// interop.X selectors, exported methods, and package init. A function reachable
// from none of them is bound and dead.
//
// It is proved falsifiable by the defect that motivated it: with
// cna_graphics_device_manager_dispose still bound, this test named it and
// failed with "bound and never reached".
func TestEveryBoundRouteIsReachedFromOutsideTheInteropPackage(t *testing.T) {
	root := repositoryRoot(t)

	routes := manifestRoutes(t, root)
	if len(routes) == 0 {
		t.Fatal("no routes parsed from the manifest")
	}

	bridgeSource := readRepoFile(t, root, "internal/interop/bridge.c")
	cgoSource := readRepoFile(t, root, "internal/interop/native_linux.go")
	reachable := reachableInteropFunctions(t, root)

	for _, route := range routes {
		trampoline, ok := enclosingCFunction(bridgeSource, "api."+route+"(")
		if !ok {
			t.Errorf("%s: bridge.c never calls the resolved pointer", route)
			continue
		}
		wrapper, ok := enclosingGoFunction(cgoSource, "C."+trampoline+"(")
		if !ok {
			t.Errorf("%s: native_linux.go never calls C.%s", route, trampoline)
			continue
		}
		if !reachable[wrapper] {
			t.Errorf("%s: bound and never reached -- %s calls C.%s and nothing outside internal/interop reaches %s",
				route, wrapper, trampoline, wrapper)
		}
	}
}

func readRepoFile(t *testing.T, root, relative string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, relative))
	if err != nil {
		t.Fatalf("read %s: %v", relative, err)
	}
	return string(data)
}

func manifestRoutes(t *testing.T, root string) []string {
	t.Helper()
	manifest := readRepoFile(t, root, "internal/interop/abi_manifest.h")
	pattern := regexp.MustCompile(`(?m)^\s*X\((cna_\w+)\)`)
	var routes []string
	for _, match := range pattern.FindAllStringSubmatch(manifest, -1) {
		routes = append(routes, match[1])
	}
	return routes
}

var cFunctionStart = regexp.MustCompile(`(?m)^[A-Za-z_][\w ]*[\s\*](\w+)\(`)

// enclosingCFunction reports the name of the C function whose body contains
// needle. Definitions in bridge.c start at column zero, which is what makes the
// last match before needle the enclosing one.
func enclosingCFunction(source, needle string) (string, bool) {
	index := strings.Index(source, needle)
	if index < 0 {
		return "", false
	}
	starts := cFunctionStart.FindAllStringSubmatchIndex(source[:index], -1)
	if len(starts) == 0 {
		return "", false
	}
	last := starts[len(starts)-1]
	return source[last[2]:last[3]], true
}

var goFunctionStart = regexp.MustCompile(`(?m)^func (?:\([^)]*\) )?(\w+)\(`)

// enclosingGoFunction reports the name of the Go func or method whose body
// contains needle.
func enclosingGoFunction(source, needle string) (string, bool) {
	index := strings.Index(source, needle)
	if index < 0 {
		return "", false
	}
	starts := goFunctionStart.FindAllStringSubmatchIndex(source[:index], -1)
	if len(starts) == 0 {
		return "", false
	}
	last := starts[len(starts)-1]
	return source[last[2]:last[3]], true
}

func interopGoFiles(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, "internal/interop"))
	if err != nil {
		t.Fatalf("read internal/interop: %v", err)
	}
	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		files = append(files, filepath.Join(root, "internal/interop", name))
	}
	return files
}

// reachableInteropFunctions reports every function and method in package
// interop that the rest of the module can reach.
//
// The roots are the three ways it can: an interop.X selector written outside
// the package, an exported method (whose receiver type crosses the boundary as
// a value), and package init. Everything those reach transitively is reachable;
// nothing else is.
func reachableInteropFunctions(t *testing.T, root string) map[string]bool {
	t.Helper()

	references := map[string]map[string]bool{}
	roots := map[string]bool{}
	exportedSelectors := interopSelectorsOutsideInterop(t, root)

	set := token.NewFileSet()
	for _, path := range interopGoFiles(t, root) {
		file, err := parser.ParseFile(set, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			name := function.Name.Name
			switch {
			case name == "init":
				roots[name] = true
			case function.Recv != nil && function.Name.IsExported():
				// A method on a type the module holds; the module reaches it
				// through the value, not through an interop.X selector.
				roots[name] = true
			case function.Recv == nil && exportedSelectors[name]:
				roots[name] = true
			}
			if references[name] == nil {
				references[name] = map[string]bool{}
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				if identifier, ok := node.(*ast.Ident); ok {
					references[name][identifier.Name] = true
				}
				if selector, ok := node.(*ast.SelectorExpr); ok {
					references[name][selector.Sel.Name] = true
				}
				return true
			})
		}
	}

	reachable := map[string]bool{}
	var visit func(string)
	visit = func(name string) {
		if reachable[name] {
			return
		}
		reachable[name] = true
		for referenced := range references[name] {
			if _, declared := references[referenced]; declared {
				visit(referenced)
			}
		}
	}
	for name := range roots {
		visit(name)
	}
	return reachable
}

// interopSelectorsOutsideInterop reports every interop.X the module names from
// outside the package.
//
// It reads the parsed syntax tree rather than the file text, and the difference
// is not cosmetic: the first version of this scan was a regexp over raw source,
// and the doc comment above -- which names interop.ManagerDispose as the defect
// being described -- was enough to make the dead route look reached. A mention
// in a comment is not a call.
func interopSelectorsOutsideInterop(t *testing.T, root string) map[string]bool {
	t.Helper()
	used := map[string]bool{}
	set := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == ".git" || strings.HasPrefix(name, "build") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.Contains(filepath.ToSlash(path), "/internal/interop/") {
			return nil
		}
		file, err := parser.ParseFile(set, path, nil, 0)
		if err != nil {
			// A testdata module that is not part of this build still counts as
			// a consumer, so a parse failure is reported rather than skipped.
			t.Fatalf("parse %s: %v", path, err)
		}

		local := ""
		for _, imported := range file.Imports {
			if imported.Path == nil || !strings.HasSuffix(strings.Trim(imported.Path.Value, `"`), "/internal/interop") {
				continue
			}
			local = "interop"
			if imported.Name != nil {
				local = imported.Name.Name
			}
		}
		if local == "" || local == "_" {
			return nil
		}

		ast.Inspect(file, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			identifier, ok := selector.X.(*ast.Ident)
			if !ok || identifier.Name != local {
				return true
			}
			used[selector.Sel.Name] = true
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return used
}
