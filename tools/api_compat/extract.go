package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func extractActual(root string) (*actualSurface, error) {
	exportFiles, err := compiledExportFiles(root)
	if err != nil {
		return nil, err
	}
	surface := &actualSurface{
		Types:       make(map[symbolKey]*actualType),
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: make(map[string]string),
		Packages:    make(map[string]*types.Package),
	}
	frameworkRoot := filepath.Join(root, "Microsoft", "Xna", "Framework")
	err = filepath.WalkDir(frameworkRoot, func(dir string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			return nil
		}
		files, err := goFiles(dir)
		if err != nil {
			return err
		}
		if len(files) == 0 {
			return nil
		}
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			return err
		}
		packagePath := modulePath + "/" + filepath.ToSlash(rel)
		surface.PackageDirs[packagePath] = dir
		if err := extractPackage(surface, packagePath, dir, files, exportFiles); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return surface, nil
}

// compiledExportFiles admits the exact source set through the Go compiler and
// returns compiler-produced export data for every dependency. The AST/types
// extractor therefore measures a surface that is known to compile, and local
// module imports are real packages rather than permissive stubs.
func compiledExportFiles(root string) (map[string]string, error) {
	goBinary := filepath.Join(runtime.GOROOT(), "bin", "go")
	command := exec.Command(goBinary, "-C", root, "list", "-deps", "-export", "-json", "./Microsoft/Xna/Framework/...")
	output, err := command.Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("Go compiler admission failed: %s", strings.TrimSpace(string(exit.Stderr)))
		}
		return nil, fmt.Errorf("run Go compiler admission: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(output))
	exports := make(map[string]string)
	for {
		var item struct {
			ImportPath string
			Export     string
		}
		if err := decoder.Decode(&item); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decode Go compiler export inventory: %w", err)
		}
		if item.ImportPath != "" && item.Export != "" {
			exports[item.ImportPath] = item.Export
		}
	}
	return exports, nil
}

func goFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	ctx := build.Default
	ctx.CgoEnabled = true
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		matched, matchErr := ctx.MatchFile(dir, name)
		if matchErr != nil {
			return nil, matchErr
		}
		if matched {
			files = append(files, filepath.Join(dir, name))
		}
	}
	return files, nil
}

func extractPackage(surface *actualSurface, packagePath, dir string, filenames []string, exportFiles map[string]string) error {
	fset := token.NewFileSet()
	files := make([]*ast.File, 0, len(filenames))
	aliases := make(map[string]string)
	for _, filename := range filenames {
		source, readErr := os.ReadFile(filename)
		if readErr != nil {
			return readErr
		}
		if bytes.Contains(source, []byte("xna:unmeasured")) {
			surface.Unmeasured = append(surface.Unmeasured, filename)
		}
		file, err := parser.ParseFile(fset, filename, source, parser.ParseComments|parser.AllErrors)
		if err != nil {
			return fmt.Errorf("parse %s: %w", filename, err)
		}
		files = append(files, file)
		for _, spec := range file.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				continue
			}
			alias := filepath.Base(importPath)
			if spec.Name != nil && spec.Name.Name != "_" && spec.Name.Name != "." {
				alias = spec.Name.Name
			}
			aliases[alias] = canonicalPackageQualifier(importPath)
		}
	}

	info := &types.Info{
		Defs: make(map[*ast.Ident]types.Object),
	}
	conf := types.Config{
		Importer: compilerExportImporter(fset, exportFiles),
		Error: func(err error) {
			surface.TypeErrors = append(surface.TypeErrors, packagePath+": "+err.Error())
		},
	}
	checkedPackage, _ := conf.Check(packagePath, fset, files, info)
	if checkedPackage != nil {
		surface.Packages[packagePath] = checkedPackage
	}

	for _, file := range files {
		for _, decl := range file.Decls {
			switch declaration := decl.(type) {
			case *ast.GenDecl:
				extractGeneralDeclaration(surface, packagePath, fset, aliases, info, declaration)
			case *ast.FuncDecl:
				extractFunction(surface, packagePath, fset, aliases, declaration)
			}
		}
	}
	return nil
}

func compilerExportImporter(fset *token.FileSet, exports map[string]string) types.Importer {
	lookup := func(importPath string) (io.ReadCloser, error) {
		filename, ok := exports[importPath]
		if !ok {
			return nil, fmt.Errorf("compiler export data is absent for %s", importPath)
		}
		return os.Open(filename)
	}
	return importer.ForCompiler(fset, "gc", lookup)
}

func extractGeneralDeclaration(surface *actualSurface, packagePath string, fset *token.FileSet, aliases map[string]string, info *types.Info, decl *ast.GenDecl) {
	for _, raw := range decl.Specs {
		switch spec := raw.(type) {
		case *ast.TypeSpec:
			if !spec.Name.IsExported() {
				continue
			}
			key := symbolKey{Package: packagePath, Name: spec.Name.Name}
			actual := &actualType{Key: key, Kind: typeKind(spec.Type), Underlying: normalizeExpr(spec.Type, aliases)}
			if hasDirectiveNamed("xna:flags", spec.Doc, decl.Doc) {
				actual.FlagsMarker = true
			}
			if spec.TypeParams != nil {
				for _, field := range spec.TypeParams.List {
					for _, name := range field.Names {
						actual.TypeParameters = append(actual.TypeParameters, name.Name)
					}
				}
			}
			if structType, ok := spec.Type.(*ast.StructType); ok {
				for _, field := range structType.Fields.List {
					if len(field.Names) == 0 {
						text := normalizeExpr(field.Type, aliases)
						if ast.IsExported(strings.TrimPrefix(text, "*")) {
							actual.ExportedEmbeddings = append(actual.ExportedEmbeddings, text)
						}
						continue
					}
					for _, name := range field.Names {
						if !name.IsExported() {
							continue
						}
						memberKey := symbolKey{Package: packagePath, Receiver: spec.Name.Name, Name: name.Name}
						surface.Members[memberKey] = &actualMember{Key: memberKey, Kind: "field", Results: []string{normalizeExpr(field.Type, aliases)}, Position: fset.Position(name.Pos()).String()}
					}
				}
			}
			if interfaceType, ok := spec.Type.(*ast.InterfaceType); ok {
				for _, field := range interfaceType.Methods.List {
					if len(field.Names) == 0 {
						if normalizeExpr(field.Type, aliases) != "" {
							actual.ExportedEmbeddings = append(actual.ExportedEmbeddings, normalizeExpr(field.Type, aliases))
						}
						continue
					}
					function, ok := field.Type.(*ast.FuncType)
					if !ok {
						continue
					}
					for _, name := range field.Names {
						if !name.IsExported() {
							continue
						}
						memberKey := symbolKey{Package: packagePath, Receiver: spec.Name.Name, Name: name.Name}
						surface.Members[memberKey] = &actualMember{Key: memberKey, Kind: "method", Parameters: fieldTypes(function.Params, aliases), Results: fieldTypes(function.Results, aliases), Position: fset.Position(name.Pos()).String()}
					}
				}
			}
			surface.Types[key] = actual
		case *ast.ValueSpec:
			for i, name := range spec.Names {
				if !name.IsExported() {
					continue
				}
				kind := "var"
				if decl.Tok == token.CONST {
					kind = "const"
				}
				key := symbolKey{Package: packagePath, Name: name.Name}
				member := &actualMember{Key: key, Kind: kind, Position: fset.Position(name.Pos()).String()}
				if spec.Type != nil {
					member.Results = []string{normalizeExpr(spec.Type, aliases)}
				}
				if object, ok := info.Defs[name].(*types.Const); ok {
					v := object.Val().ExactString()
					member.Value = &v
				} else if i < len(spec.Values) {
					v := normalizeExpr(spec.Values[i], aliases)
					member.Value = &v
				}
				surface.Members[key] = member
			}
		}
	}
}

func extractFunction(surface *actualSurface, packagePath string, fset *token.FileSet, aliases map[string]string, decl *ast.FuncDecl) {
	if !decl.Name.IsExported() {
		return
	}
	receiver := ""
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		receiver = receiverName(decl.Recv.List[0].Type)
		if receiver == "" || !ast.IsExported(receiver) {
			return
		}
	}
	key := symbolKey{Package: packagePath, Receiver: receiver, Name: decl.Name.Name}
	kind := "func"
	if receiver != "" {
		kind = "method"
	}
	surface.Members[key] = &actualMember{
		Key: key, Kind: kind,
		Parameters: fieldTypes(decl.Type.Params, aliases),
		Results:    fieldTypes(decl.Type.Results, aliases),
		Position:   fset.Position(decl.Name.Pos()).String(),
	}
}

func typeKind(expr ast.Expr) string {
	switch expr.(type) {
	case *ast.StructType:
		return "struct"
	case *ast.InterfaceType:
		return "interface"
	case *ast.FuncType:
		return "delegate"
	case *ast.Ident, *ast.SelectorExpr:
		return "named"
	default:
		return "other"
	}
}

func receiverName(expr ast.Expr) string {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if index, ok := expr.(*ast.IndexExpr); ok {
		expr = index.X
	}
	if index, ok := expr.(*ast.IndexListExpr); ok {
		expr = index.X
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}

func fieldTypes(fields *ast.FieldList, aliases map[string]string) []string {
	if fields == nil {
		return nil
	}
	var result []string
	for _, field := range fields.List {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		text := normalizeExpr(field.Type, aliases)
		for i := 0; i < count; i++ {
			result = append(result, text)
		}
	}
	return result
}

func normalizeExpr(expr ast.Expr, aliases map[string]string) string {
	if expr == nil {
		return ""
	}
	var buffer bytes.Buffer
	if err := format.Node(&buffer, token.NewFileSet(), expr); err != nil {
		return ""
	}
	text := buffer.String()
	for alias, canonical := range aliases {
		text = strings.ReplaceAll(text, alias+".", canonical+".")
	}
	return strings.ReplaceAll(text, "interface{}", "any")
}

func canonicalPackageQualifier(importPath string) string {
	if importPath == modulePath+"/Microsoft/Xna/Framework" {
		return "framework"
	}
	return strings.ToLower(filepath.Base(importPath))
}

func hasDirectiveNamed(name string, groups ...*ast.CommentGroup) bool {
	for _, group := range groups {
		if group == nil {
			continue
		}
		for _, comment := range group.List {
			if strings.Contains(strings.TrimSpace(strings.TrimPrefix(comment.Text, "//")), name) {
				return true
			}
		}
	}
	return false
}
