package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type mutationFixture struct {
	ID       string `json:"id"`
	Mutation string `json:"mutation"`
	Category string `json:"category"`
}

func TestPinnedContractAndMappedCounts(t *testing.T) {
	data, err := os.ReadFile("reference/xna40-windows-runtime-contract.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := sha256Hex(data); got != "7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc" {
		t.Fatalf("contract hash = %s", got)
	}
	var reference contract
	if err := json.Unmarshal(data, &reference); err != nil {
		t.Fatal(err)
	}
	surface, err := buildExpected(reference)
	if err != nil {
		t.Fatal(err)
	}
	if surface.ReferenceTypes != 257 || surface.ReferenceMembers != 2964 {
		t.Fatalf("reference counts = %d/%d", surface.ReferenceTypes, surface.ReferenceMembers)
	}
	if surface.ExpectedGoTypes != 257 || surface.ExpectedGoMembers != 3243 {
		t.Fatalf("mapped counts = %d/%d", surface.ExpectedGoTypes, surface.ExpectedGoMembers)
	}
}

func TestNullableMappingKeepsInputReturnOutAndErrorDistinct(t *testing.T) {
	const nullableSingle = "System.Nullable`1[System.Single]"
	owner := &expectedType{PackagePath: modulePath + "/Microsoft/Xna/Framework"}
	surface := &expectedSurface{}

	if got := mapType(surface, nil, owner, nullableSingle); got != "*float32" {
		t.Fatalf("nullable input = %q, want *float32", got)
	}
	if got := typeShape(nullableSingle); got != "NullableOfSingle" {
		t.Fatalf("nullable source shape = %q, want NullableOfSingle", got)
	}
	if got := mapReturn(surface, nil, owner, stringPointer(nullableSingle)); !equalStrings(got, []string{"float32", "bool"}) {
		t.Fatalf("nullable return = %v, want [float32 bool]", got)
	}
	inputs, outputs, directed := mapParameters(surface, nil, owner, []contractParameter{
		{Name: "value", Type: nullableSingle},
		{Name: "result", Type: nullableSingle + "&", Out: true},
	})
	if !equalStrings(inputs, []string{"*float32"}) || !equalStrings(outputs, []string{"float32", "bool"}) || !directed {
		t.Fatalf("nullable parameters = inputs %v outputs %v directed %t", inputs, outputs, directed)
	}
	withError := append(append([]string(nil), mapReturn(surface, nil, owner, stringPointer(nullableSingle))...), "error")
	if !equalStrings(withError, []string{"float32", "bool", "error"}) {
		t.Fatalf("nullable/error result = %v", withError)
	}
}

func TestOwnerGenericParameterSubstitution(t *testing.T) {
	owner := &expectedType{
		XNA:              "Example.Pair`2",
		PackagePath:      modulePath + "/Microsoft/Xna/Framework",
		GenericParameter: []string{"TFirst", "TSecond"},
	}
	surface := &expectedSurface{}

	if got := mapType(surface, nil, owner, "!0"); got != "TFirst" {
		t.Fatalf("!0 = %q, want TFirst", got)
	}
	if got := mapType(surface, nil, owner, "!1"); got != "TSecond" {
		t.Fatalf("!1 = %q, want TSecond", got)
	}
	if got := mapType(surface, nil, owner, "!0[]"); got != "[]TFirst" {
		t.Fatalf("!0[] = %q, want []TFirst", got)
	}
	if got := mapType(surface, nil, owner, "System.Nullable`1[!1]"); got != "*TSecond" {
		t.Fatalf("Nullable<!1> = %q, want *TSecond", got)
	}
	if got := mapType(surface, nil, owner, "System.Collections.Generic.IEnumerator`1[!0]"); got != "Iterator[TFirst]" {
		t.Fatalf("IEnumerator<!0> = %q, want Iterator[TFirst]", got)
	}
	if _, matched, err := mapOwnerGenericParameter(owner, "!!0"); matched || err != nil {
		t.Fatalf("method token !!0 was treated as an owner token: matched=%t err=%v", matched, err)
	}

	for _, invalid := range []string{"!", "!x", "!-1", "!2"} {
		before := len(surface.MappingIssues)
		if got := mapType(surface, nil, owner, invalid); got != "any" {
			t.Fatalf("invalid %s = %q, want any", invalid, got)
		}
		if len(surface.MappingIssues) != before+1 || surface.MappingIssues[len(surface.MappingIssues)-1].Category != "GENERIC_MAPPING_MISMATCH" {
			t.Fatalf("invalid %s did not add GENERIC_MAPPING_MISMATCH: %+v", invalid, surface.MappingIssues)
		}
	}
}

func TestPackedVectorMappedContract(t *testing.T) {
	reference := loadPinnedContract(t)
	surface, err := buildExpected(reference)
	if err != nil {
		t.Fatal(err)
	}

	const prefix = "Microsoft.Xna.Framework.Graphics.PackedVector."
	want := map[string]struct {
		goName       string
		source       int
		mapped       int
		tPacked      string
		witnessCount int
	}{
		prefix + "IPackedVector":    {"IPackedVector", 2, 2, "", 0},
		prefix + "IPackedVector`1":  {"IPackedVectorOfTPacked", 1, 2, "TPacked", 0},
		prefix + "Alpha8":           {"Alpha8", 9, 10, "uint8", 2},
		prefix + "Bgr565":           {"Bgr565", 10, 11, "uint16", 2},
		prefix + "Bgra4444":         {"Bgra4444", 10, 11, "uint16", 1},
		prefix + "Bgra5551":         {"Bgra5551", 10, 11, "uint16", 1},
		prefix + "Byte4":            {"Byte4", 10, 11, "uint32", 1},
		prefix + "HalfSingle":       {"HalfSingle", 9, 10, "uint16", 2},
		prefix + "HalfVector2":      {"HalfVector2", 10, 11, "uint32", 2},
		prefix + "HalfVector4":      {"HalfVector4", 10, 11, "uint64", 1},
		prefix + "NormalizedByte2":  {"NormalizedByte2", 10, 11, "uint16", 2},
		prefix + "NormalizedByte4":  {"NormalizedByte4", 10, 11, "uint32", 1},
		prefix + "NormalizedShort2": {"NormalizedShort2", 10, 11, "uint32", 2},
		prefix + "NormalizedShort4": {"NormalizedShort4", 10, 11, "uint64", 1},
		prefix + "Rg32":             {"Rg32", 10, 11, "uint32", 2},
		prefix + "Rgba1010102":      {"Rgba1010102", 10, 11, "uint32", 1},
		prefix + "Rgba64":           {"Rgba64", 10, 11, "uint64", 1},
		prefix + "Short2":           {"Short2", 10, 11, "uint32", 2},
		prefix + "Short4":           {"Short4", 10, 11, "uint64", 1},
	}

	sourceTotal := 0
	mappedTotal := 0
	for identity, expected := range want {
		mapped := surface.typeForXNA(identity)
		if mapped == nil {
			t.Fatalf("%s was not mapped", identity)
		}
		if mapped.GoName != expected.goName || mapped.SourceMembers != expected.source || len(mapped.Members) != expected.mapped {
			t.Fatalf("%s = name %s source %d mapped %d, want %+v", identity, mapped.GoName, mapped.SourceMembers, len(mapped.Members), expected)
		}
		if expected.tPacked != "" && identity != prefix+"IPackedVector`1" {
			packedInterface, ok := directPackedInterface(mapped)
			if !ok || !equalStrings(packedInterface.TypeArguments, []string{expected.tPacked}) {
				t.Fatalf("%s packed interface = %+v, want %s", identity, packedInterface, expected.tPacked)
			}
		}
		witnesses := 0
		for _, witness := range surface.InterfaceWitnesses {
			if witness.Owner == identity {
				witnesses++
			}
		}
		if witnesses != expected.witnessCount {
			t.Fatalf("%s witnesses = %d, want %d", identity, witnesses, expected.witnessCount)
		}
		sourceTotal += expected.source
		mappedTotal += expected.mapped
	}
	if sourceTotal != 171 || mappedTotal != 189 || len(surface.InterfaceWitnesses) != 25 {
		t.Fatalf("PackedVector totals = source %d mapped %d witnesses %d", sourceTotal, mappedTotal, len(surface.InterfaceWitnesses))
	}

	base := surface.typeForXNA(prefix + "IPackedVector")
	toVector4 := surface.Members[symbolKey{Package: base.PackagePath, Receiver: base.GoName, Name: "ToVector4"}]
	packFromVector4 := surface.Members[symbolKey{Package: base.PackagePath, Receiver: base.GoName, Name: "PackFromVector4"}]
	if toVector4 == nil || packFromVector4 == nil || toVector4.ErrorAdded || packFromVector4.ErrorAdded || !equalStrings(toVector4.Results, []string{"framework.Vector4"}) || len(packFromVector4.Results) != 0 {
		t.Fatalf("managed base interface signatures = ToVector4 %+v PackFromVector4 %+v", toVector4, packFromVector4)
	}
	generic := surface.typeForXNA(prefix + "IPackedVector`1")
	if !equalStrings(generic.GenericParameter, []string{"TPacked"}) || len(generic.MappedInterfaces) != 1 || generic.MappedInterfaces[0].GoName != "IPackedVector" {
		t.Fatalf("generic packed interface identity/inheritance = %+v", generic)
	}
	getter := surface.Members[symbolKey{Package: generic.PackagePath, Receiver: generic.GoName, Name: "PackedValue"}]
	setter := surface.Members[symbolKey{Package: generic.PackagePath, Receiver: generic.GoName, Name: "SetPackedValue"}]
	if getter == nil || setter == nil || getter.ErrorAdded || setter.ErrorAdded || !equalStrings(getter.Results, []string{"TPacked"}) || !equalStrings(setter.Parameters, []string{"TPacked"}) || len(setter.Results) != 0 {
		t.Fatalf("generic PackedValue projection = getter %+v setter %+v", getter, setter)
	}
}

func TestVertexElementMappedContract(t *testing.T) {
	reference := loadPinnedContract(t)
	surface, err := buildExpected(reference)
	if err != nil {
		t.Fatal(err)
	}

	const prefix = "Microsoft.Xna.Framework.Graphics."
	want := map[string]struct {
		source int
		mapped int
		kind   string
	}{
		prefix + "VertexElement":       {10, 14, "struct"},
		prefix + "VertexElementFormat": {13, 12, "enum"},
		prefix + "VertexElementUsage":  {14, 13, "enum"},
	}
	sourceTotal, mappedTotal := 0, 0
	for identity, expected := range want {
		mapped := surface.typeForXNA(identity)
		if mapped == nil {
			t.Fatalf("%s was not mapped", identity)
		}
		if mapped.SourceMembers != expected.source || len(mapped.Members) != expected.mapped || mapped.Kind != expected.kind {
			t.Fatalf("%s = source %d mapped %d kind %s, want %+v", identity, mapped.SourceMembers, len(mapped.Members), mapped.Kind, expected)
		}
		sourceTotal += mapped.SourceMembers
		mappedTotal += len(mapped.Members)
	}
	if sourceTotal != 37 || mappedTotal != 39 {
		t.Fatalf("vertex closure totals = source %d mapped %d", sourceTotal, mappedTotal)
	}

	vertex := surface.typeForXNA(prefix + "VertexElement")
	constructor := surface.Members[symbolKey{Package: vertex.PackagePath, Name: "NewVertexElement"}]
	if constructor == nil || !equalStrings(constructor.Parameters, []string{"int32", "VertexElementFormat", "VertexElementUsage", "int32"}) ||
		!equalStrings(constructor.Results, []string{"VertexElement"}) || constructor.ErrorAdded {
		t.Fatalf("constructor projection = %+v", constructor)
	}
	for _, property := range []struct {
		name       string
		mappedType string
	}{
		{"Offset", "int32"},
		{"VertexElementFormat", "VertexElementFormat"},
		{"VertexElementUsage", "VertexElementUsage"},
		{"UsageIndex", "int32"},
	} {
		getter := surface.Members[symbolKey{Package: vertex.PackagePath, Receiver: vertex.GoName, Name: property.name}]
		setter := surface.Members[symbolKey{Package: vertex.PackagePath, Receiver: vertex.GoName, Name: "Set" + property.name}]
		if getter == nil || setter == nil || !equalStrings(getter.Results, []string{property.mappedType}) ||
			!equalStrings(setter.Parameters, []string{property.mappedType}) || len(setter.Results) != 0 || getter.ErrorAdded || setter.ErrorAdded {
			t.Fatalf("%s projection = getter %+v setter %+v", property.name, getter, setter)
		}
	}
	if surface.Members[symbolKey{Package: vertex.PackagePath, Receiver: vertex.GoName, Name: "EqualsByVertexElement"}] != nil {
		t.Fatal("invented typed Equals(VertexElement) projection")
	}
	equalsObject := surface.Members[symbolKey{Package: vertex.PackagePath, Receiver: vertex.GoName, Name: "Equals"}]
	if equalsObject == nil || !equalStrings(equalsObject.Parameters, []string{"any"}) || !equalStrings(equalsObject.Results, []string{"bool"}) || equalsObject.ErrorAdded {
		t.Fatalf("unique Equals(Object) projection = %+v", equalsObject)
	}
}

func TestPlayerIndexKeyboardMappedContract(t *testing.T) {
	reference := loadPinnedContract(t)
	surface, err := buildExpected(reference)
	if err != nil {
		t.Fatal(err)
	}

	playerIndex := surface.typeForXNA("Microsoft.Xna.Framework.PlayerIndex")
	if playerIndex == nil || playerIndex.Kind != "enum" || playerIndex.Flags || playerIndex.SourceMembers != 5 || len(playerIndex.Members) != 4 {
		t.Fatalf("PlayerIndex projection = %+v", playerIndex)
	}
	for name, value := range map[string]string{"One": "0", "Two": "1", "Three": "2", "Four": "3"} {
		member := surface.Members[symbolKey{Package: playerIndex.PackagePath, Name: "PlayerIndex" + name}]
		if member == nil || member.GoKind != "const" || member.EnumValue == nil || *member.EnumValue != value || !equalStrings(member.Results, []string{"PlayerIndex"}) {
			t.Fatalf("PlayerIndex%s projection = %+v", name, member)
		}
	}

	keyboard := surface.typeForXNA("Microsoft.Xna.Framework.Input.Keyboard")
	if keyboard == nil || keyboard.Kind != "class" || keyboard.SourceMembers != 2 || len(keyboard.Members) != 2 {
		t.Fatalf("Keyboard projection = %+v", keyboard)
	}
	none := surface.Members[symbolKey{Package: keyboard.PackagePath, Name: "KeyboardGetStateByNone"}]
	byPlayerIndex := surface.Members[symbolKey{Package: keyboard.PackagePath, Name: "KeyboardGetStateByPlayerIndex"}]
	if none == nil || !none.OverloadMapped || len(none.Parameters) != 0 || !equalStrings(none.Results, []string{"KeyboardState", "error"}) || !none.ErrorAdded {
		t.Fatalf("Keyboard.GetState() projection = %+v", none)
	}
	if byPlayerIndex == nil || !byPlayerIndex.OverloadMapped || !equalStrings(byPlayerIndex.Parameters, []string{"framework.PlayerIndex"}) ||
		!equalStrings(byPlayerIndex.Results, []string{"KeyboardState", "error"}) || !byPlayerIndex.ErrorAdded {
		t.Fatalf("Keyboard.GetState(PlayerIndex) projection = %+v", byPlayerIndex)
	}
}

func TestPlayerIndexKeyboardCurrentSurfaceAndLocalClosure(t *testing.T) {
	reference := loadPinnedContract(t)
	expected, err := buildExpected(reference)
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	actual, err := extractActual(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(actual.TypeErrors) != 0 {
		t.Fatalf("type errors: %v", actual.TypeErrors)
	}
	result := verify(expected, actual, 0, "report", "contract", "mapping")
	closure := result.PlayerIndexKeyboardClosure
	if closure.Status != "PASS" || closure.SourceTypes != 2 || closure.SourceIdentities != 7 || closure.MappedGoIdentities != 6 ||
		closure.TargetTypes != 2 || closure.TargetGoIdentities != 6 || closure.LocalDiagnostics != 0 || len(closure.TypeMeasurements) != 2 {
		t.Fatalf("PlayerIndex/Keyboard closure = %+v", closure)
	}
	for _, row := range closure.TypeMeasurements {
		if row.LocalDiagnostics != 0 || row.TargetGoMembers != row.ExpectedGoMembers {
			t.Fatalf("PlayerIndex/Keyboard type row = %+v", row)
		}
	}
}

func TestVertexElementCurrentSurfaceAndLocalClosure(t *testing.T) {
	reference := loadPinnedContract(t)
	expected, err := buildExpected(reference)
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	actual, err := extractActual(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(actual.TypeErrors) != 0 {
		t.Fatalf("type errors: %v", actual.TypeErrors)
	}
	result := verify(expected, actual, 0, "report", "contract", "mapping")
	closure := result.VertexElementClosure
	if closure.Status != "PASS" || closure.SourceTypes != 3 || closure.SourceIdentities != 37 || closure.MappedGoIdentities != 39 ||
		closure.TargetTypes != 3 || closure.TargetGoIdentities != 39 || closure.WritableProperties != 4 || closure.ProjectedAccessors != 8 ||
		closure.LocalDiagnostics != 0 || len(closure.TypeMeasurements) != 3 {
		t.Fatalf("vertex closure = %+v", closure)
	}
	for _, row := range closure.TypeMeasurements {
		if row.LocalDiagnostics != 0 || row.TargetGoMembers != row.ExpectedGoMembers {
			t.Fatalf("vertex type row = %+v", row)
		}
	}
}

func loadPinnedContract(t *testing.T) contract {
	t.Helper()
	data, err := os.ReadFile("reference/xna40-windows-runtime-contract.json")
	if err != nil {
		t.Fatal(err)
	}
	var reference contract
	if err := json.Unmarshal(data, &reference); err != nil {
		t.Fatal(err)
	}
	return reference
}

func TestPackedVectorCurrentSurfaceAndConformance(t *testing.T) {
	reference := loadPinnedContract(t)
	expected, err := buildExpected(reference)
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	actual, err := extractActual(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(actual.TypeErrors) != 0 {
		t.Fatalf("type errors: %v", actual.TypeErrors)
	}
	result := verify(expected, actual, 0, "report", "contract", "mapping")
	if result.Summary["INTERFACE_WITNESS_PROJECTIONS"] != 25 || result.Summary["PACKFROMVECTOR4_WITNESS_PROJECTIONS"] != 17 || result.Summary["TOVECTOR4_WITNESS_PROJECTIONS"] != 8 {
		t.Fatalf("witness counters = %v", result.Summary)
	}
	if len(result.PackedInterfaceConformance) != 17 || len(result.PackedVectorTypeMeasurements) != 19 {
		t.Fatalf("packed measurements = conformance %d types %d", len(result.PackedInterfaceConformance), len(result.PackedVectorTypeMeasurements))
	}
	for _, conformance := range result.PackedInterfaceConformance {
		if conformance.Status != "PASS" || !conformance.PointerMethodSetSatisfies || conformance.ValueMethodSetSatisfies || !conformance.TransitiveBaseSatisfies {
			t.Fatalf("conformance failed: %+v", conformance)
		}
	}
	for _, measurement := range result.PackedVectorTypeMeasurements {
		if measurement.LocalDiagnostics != 0 || measurement.TargetGoMembers != measurement.ExpectedGoMembers {
			t.Fatalf("local PackedVector surface failed: %+v", measurement)
		}
	}
	for _, category := range diagnosticCategories[2:] {
		if result.Summary[category] != 0 {
			t.Fatalf("%s = %d", category, result.Summary[category])
		}
	}
}

func TestPackedPointerMethodSetPolicyRejectsValueSatisfaction(t *testing.T) {
	const pkgPath = modulePath + "/Microsoft/Xna/Framework/Graphics/PackedVector"
	const source = `package packedvector
type Vector4 struct{}
type IPackedVector interface { ToVector4() Vector4; PackFromVector4(Vector4) }
type IPackedVectorOfTPacked[TPacked any] interface { IPackedVector; PackedValue() TPacked; SetPackedValue(TPacked) }
type Alpha8 struct{}
func (Alpha8) ToVector4() Vector4 { return Vector4{} }
func (Alpha8) PackFromVector4(Vector4) {}
func (Alpha8) PackedValue() uint8 { return 0 }
func (Alpha8) SetPackedValue(uint8) {}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "mutation.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := new(types.Config).Check(pkgPath, fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	owner := &expectedType{
		Key:         symbolKey{Package: pkgPath, Name: "Alpha8"},
		XNA:         packedVectorNamespace + "Alpha8",
		GoName:      "Alpha8",
		PackagePath: pkgPath,
		MappedInterfaces: []mappedInterface{{
			XNA:           packedVectorNamespace + "IPackedVector`1[System.Byte]",
			GoName:        "IPackedVectorOfTPacked",
			TypeArguments: []string{"uint8"},
		}},
	}
	result := report{Summary: map[string]int{}}
	measurement, measured := measurePackedInterfaceConformance(&result, &actualSurface{Packages: map[string]*types.Package{pkgPath: pkg}}, owner)
	if !measured || measurement.Status != "FAIL" || !measurement.ValueMethodSetSatisfies || result.Summary["INTERFACE_MAPPING_MISMATCH"] == 0 {
		t.Fatalf("value receiver mutation was accepted: measurement=%+v summary=%v", measurement, result.Summary)
	}
}

func TestPackedGenericConformanceRejectsWrongTPackedAndMissingMutation(t *testing.T) {
	const pkgPath = modulePath + "/Microsoft/Xna/Framework/Graphics/PackedVector"
	fixtures := []struct {
		name       string
		sourceTail string
		tPacked    string
	}{
		{
			name: "wrong TPacked",
			sourceTail: `
func (Alpha8) ToVector4() Vector4 { return Vector4{} }
func (*Alpha8) PackFromVector4(Vector4) {}
func (Alpha8) PackedValue() uint8 { return 0 }
func (*Alpha8) SetPackedValue(uint8) {}
`,
			tPacked: "uint16",
		},
		{
			name: "missing PackFromVector4",
			sourceTail: `
func (Alpha8) ToVector4() Vector4 { return Vector4{} }
func (Alpha8) PackedValue() uint8 { return 0 }
func (*Alpha8) SetPackedValue(uint8) {}
`,
			tPacked: "uint8",
		},
		{
			name: "wrong PackedValue setter type",
			sourceTail: `
func (Alpha8) ToVector4() Vector4 { return Vector4{} }
func (*Alpha8) PackFromVector4(Vector4) {}
func (Alpha8) PackedValue() uint8 { return 0 }
func (*Alpha8) SetPackedValue(uint16) {}
`,
			tPacked: "uint8",
		},
	}
	const sourceHead = `package packedvector
type Vector4 struct{}
type IPackedVector interface { ToVector4() Vector4; PackFromVector4(Vector4) }
type IPackedVectorOfTPacked[TPacked any] interface { IPackedVector; PackedValue() TPacked; SetPackedValue(TPacked) }
type Alpha8 struct{}
`
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "mutation.go", sourceHead+fixture.sourceTail, 0)
			if err != nil {
				t.Fatal(err)
			}
			pkg, err := new(types.Config).Check(pkgPath, fset, []*ast.File{file}, nil)
			if err != nil {
				t.Fatal(err)
			}
			owner := &expectedType{
				Key:         symbolKey{Package: pkgPath, Name: "Alpha8"},
				XNA:         packedVectorNamespace + "Alpha8",
				GoName:      "Alpha8",
				PackagePath: pkgPath,
				MappedInterfaces: []mappedInterface{{
					XNA:           packedVectorNamespace + "IPackedVector`1[System.Byte]",
					GoName:        "IPackedVectorOfTPacked",
					TypeArguments: []string{fixture.tPacked},
				}},
			}
			result := report{Summary: map[string]int{}}
			measurement, measured := measurePackedInterfaceConformance(&result, &actualSurface{Packages: map[string]*types.Package{pkgPath: pkg}}, owner)
			if !measured || measurement.Status != "FAIL" || result.Summary["INTERFACE_MAPPING_MISMATCH"] == 0 {
				t.Fatalf("mutation was accepted: measurement=%+v summary=%v", measurement, result.Summary)
			}
		})
	}
}

func TestCurveFamilyMappedContract(t *testing.T) {
	data, err := os.ReadFile("reference/xna40-windows-runtime-contract.json")
	if err != nil {
		t.Fatal(err)
	}
	var reference contract
	if err := json.Unmarshal(data, &reference); err != nil {
		t.Fatal(err)
	}
	surface, err := buildExpected(reference)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{
		"Microsoft.Xna.Framework.Curve":              13,
		"Microsoft.Xna.Framework.CurveKey":           19,
		"Microsoft.Xna.Framework.CurveKeyCollection": 14,
		"Microsoft.Xna.Framework.CurveContinuity":    2,
		"Microsoft.Xna.Framework.CurveLoopType":      5,
		"Microsoft.Xna.Framework.CurveTangent":       3,
	}
	for identity, members := range want {
		mapped := surface.typeForXNA(identity)
		if mapped == nil {
			t.Fatalf("%s was not mapped", identity)
		}
		if len(mapped.Members) != members {
			t.Fatalf("%s mapped members = %d, want %d", identity, len(mapped.Members), members)
		}
	}
	collection := surface.typeForXNA("Microsoft.Xna.Framework.CurveKeyCollection")
	if !containsInterfacePrefix(collection.Interfaces, "System.Collections.Generic.ICollection`1[") ||
		!containsInterfacePrefix(collection.AllInterfaces, "System.Collections.Generic.IEnumerable`1[") {
		t.Fatalf("collection interfaces = direct %v all %v", collection.Interfaces, collection.AllInterfaces)
	}
	getEnumerator := surface.Members[symbolKey{Package: collection.PackagePath, Receiver: collection.GoName, Name: "GetEnumerator"}]
	if getEnumerator == nil || !equalStrings(getEnumerator.Results, []string{"Iterator[*CurveKey]"}) {
		t.Fatalf("GetEnumerator results = %v", getEnumerator)
	}
	item := surface.Members[symbolKey{Package: collection.PackagePath, Receiver: collection.GoName, Name: "Item"}]
	setItem := surface.Members[symbolKey{Package: collection.PackagePath, Receiver: collection.GoName, Name: "SetItem"}]
	if item == nil || setItem == nil || !equalStrings(item.Results, []string{"*CurveKey", "error"}) || !equalStrings(setItem.Results, []string{"error"}) {
		t.Fatalf("indexer projection = getter %v setter %v", item, setItem)
	}
}

func stringPointer(value string) *string { return &value }

func TestMutationFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/mutations.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []mutationFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	if len(fixtures) < 20 {
		t.Fatalf("only %d mutation fixtures", len(fixtures))
	}
	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(fixture.ID, func(t *testing.T) {
			var expected *expectedSurface
			var actual *actualSurface
			if strings.HasPrefix(fixture.Mutation, "player_index_") || strings.HasPrefix(fixture.Mutation, "keyboard_player_index_") {
				expected, actual = playerIndexKeyboardMutationCase(t, fixture.Mutation)
			} else if strings.HasPrefix(fixture.Mutation, "vertex_") {
				expected, actual = vertexElementMutationCase(t, fixture.Mutation)
			} else {
				expected, actual = mutationCase(fixture.Mutation)
			}
			result := verify(expected, actual, 0, "report", "contract", "mapping")
			if result.Summary[fixture.Category] == 0 {
				t.Fatalf("mutation %q did not trigger %s; summary=%v", fixture.Mutation, fixture.Category, result.Summary)
			}
		})
	}
}

func playerIndexKeyboardMutationCase(t *testing.T, mutation string) (*expectedSurface, *actualSurface) {
	t.Helper()
	full, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	expected := &expectedSurface{
		Types:              make(map[symbolKey]*expectedType),
		Members:            make(map[symbolKey]*expectedMember),
		InterfaceWitnesses: make(map[symbolKey]*expectedInterfaceWitness),
		ReferenceTypes:     2,
		ReferenceMembers:   7,
		ExpectedGoTypes:    2,
		ExpectedGoMembers:  6,
	}
	actual := &actualSurface{
		Types:       make(map[symbolKey]*actualType),
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: make(map[string]string),
		Packages:    make(map[string]*types.Package),
	}
	for _, identity := range playerIndexKeyboardClosureTypes {
		fullType := full.typeForXNA(identity)
		copiedType := *fullType
		copiedType.Members = append([]symbolKey(nil), fullType.Members...)
		expected.Types[copiedType.Key] = &copiedType
		actualKind, underlying := "struct", "struct{}"
		if copiedType.Kind == "enum" {
			actualKind, underlying = "named", "int32"
		}
		actual.Types[copiedType.Key] = &actualType{Key: copiedType.Key, Kind: actualKind, Underlying: underlying, FlagsMarker: copiedType.Flags}
		for _, memberKey := range copiedType.Members {
			fullMember := full.Members[memberKey]
			copiedMember := *fullMember
			copiedMember.Parameters = append([]string(nil), fullMember.Parameters...)
			copiedMember.Results = append([]string(nil), fullMember.Results...)
			expected.Members[memberKey] = &copiedMember
			actualMember := &actualMember{
				Key:        memberKey,
				Kind:       copiedMember.GoKind,
				Parameters: append([]string(nil), copiedMember.Parameters...),
				Results:    append([]string(nil), copiedMember.Results...),
			}
			if copiedMember.EnumValue != nil {
				value := *copiedMember.EnumValue
				actualMember.Value = &value
			}
			actual.Members[memberKey] = actualMember
		}
	}

	const frameworkPackage = modulePath + "/Microsoft/Xna/Framework"
	const inputPackage = frameworkPackage + "/Input"
	playerType := symbolKey{Package: frameworkPackage, Name: "PlayerIndex"}
	playerConstant := func(name string) symbolKey { return symbolKey{Package: frameworkPackage, Name: "PlayerIndex" + name} }
	keyboardFunction := func(name string) symbolKey { return symbolKey{Package: inputPackage, Name: name} }
	overload := keyboardFunction("KeyboardGetStateByPlayerIndex")
	switch mutation {
	case "player_index_wrong_kind":
		actual.Types[playerType].Kind = "struct"
	case "player_index_wrong_underlying_type":
		actual.Types[playerType].Underlying = "uint32"
	case "player_index_accidentally_flags":
		actual.Types[playerType].FlagsMarker = true
	case "player_index_wrong_one_value":
		wrong := "1"
		actual.Members[playerConstant("One")].Value = &wrong
	case "player_index_wrong_four_value":
		wrong := "4"
		actual.Members[playerConstant("Four")].Value = &wrong
	case "player_index_missing_four":
		delete(actual.Members, playerConstant("Four"))
	case "keyboard_player_index_missing_overload":
		delete(actual.Members, overload)
	case "keyboard_player_index_parameter_int32":
		actual.Members[overload].Parameters = []string{"int32"}
	case "keyboard_player_index_wrong_return":
		actual.Members[overload].Results = []string{"int32", "error"}
	case "keyboard_player_index_missing_error":
		actual.Members[overload].Results = []string{"KeyboardState"}
	case "keyboard_player_index_wrong_overload_name":
		delete(actual.Members, overload)
		wrong := keyboardFunction("KeyboardGetStateByInt32")
		actual.Members[wrong] = &actualMember{Key: wrong, Kind: "func", Parameters: []string{"framework.PlayerIndex"}, Results: []string{"KeyboardState", "error"}}
	default:
		t.Fatalf("unknown PlayerIndex/Keyboard mutation %q", mutation)
	}
	return expected, actual
}

func vertexElementMutationCase(t *testing.T, mutation string) (*expectedSurface, *actualSurface) {
	t.Helper()
	full, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	expected := &expectedSurface{
		Types:              make(map[symbolKey]*expectedType),
		Members:            make(map[symbolKey]*expectedMember),
		InterfaceWitnesses: make(map[symbolKey]*expectedInterfaceWitness),
		ReferenceTypes:     3,
		ReferenceMembers:   37,
		ExpectedGoTypes:    3,
		ExpectedGoMembers:  39,
	}
	actual := &actualSurface{
		Types:       make(map[symbolKey]*actualType),
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: make(map[string]string),
		Packages:    make(map[string]*types.Package),
	}
	for _, identity := range vertexElementClosureTypes {
		fullType := full.typeForXNA(identity)
		copiedType := *fullType
		copiedType.Members = append([]symbolKey(nil), fullType.Members...)
		expected.Types[copiedType.Key] = &copiedType
		actualKind, underlying := "struct", "struct{}"
		if copiedType.Kind == "enum" {
			actualKind, underlying = "named", "int32"
		}
		actual.Types[copiedType.Key] = &actualType{Key: copiedType.Key, Kind: actualKind, Underlying: underlying, FlagsMarker: copiedType.Flags}
		for _, memberKey := range copiedType.Members {
			fullMember := full.Members[memberKey]
			copiedMember := *fullMember
			copiedMember.Parameters = append([]string(nil), fullMember.Parameters...)
			copiedMember.Results = append([]string(nil), fullMember.Results...)
			expected.Members[memberKey] = &copiedMember
			actualMember := &actualMember{
				Key:        memberKey,
				Kind:       copiedMember.GoKind,
				Parameters: append([]string(nil), copiedMember.Parameters...),
				Results:    append([]string(nil), copiedMember.Results...),
			}
			if copiedMember.EnumValue != nil {
				value := *copiedMember.EnumValue
				actualMember.Value = &value
			}
			actual.Members[memberKey] = actualMember
		}
	}

	const pkg = modulePath + "/Microsoft/Xna/Framework/Graphics"
	vertexTypeKey := symbolKey{Package: pkg, Name: "VertexElement"}
	member := func(receiver, name string) symbolKey { return symbolKey{Package: pkg, Receiver: receiver, Name: name} }
	function := func(name string) symbolKey { return symbolKey{Package: pkg, Name: name} }
	switch mutation {
	case "vertex_wrong_kind":
		actual.Types[vertexTypeKey].Kind = "interface"
	case "vertex_offset_exposed_field":
		actual.Members[member("VertexElement", "Offset")].Kind = "field"
	case "vertex_missing_offset_setter":
		delete(actual.Members, member("VertexElement", "SetOffset"))
	case "vertex_wrong_offset_type":
		actual.Members[member("VertexElement", "Offset")].Results = []string{"uint32"}
	case "vertex_missing_format_setter":
		delete(actual.Members, member("VertexElement", "SetVertexElementFormat"))
	case "vertex_wrong_format_property_type":
		actual.Members[member("VertexElement", "SetVertexElementFormat")].Parameters = []string{"VertexElementUsage"}
	case "vertex_missing_usage_index_setter":
		delete(actual.Members, member("VertexElement", "SetUsageIndex"))
	case "vertex_constructor_parameter_order":
		actual.Members[function("NewVertexElement")].Parameters = []string{"int32", "VertexElementUsage", "VertexElementFormat", "int32"}
	case "vertex_constructor_wrong_enum_type":
		actual.Members[function("NewVertexElement")].Parameters = []string{"int32", "VertexElementUsage", "VertexElementUsage", "int32"}
	case "vertex_unexpected_typed_equals":
		key := member("VertexElement", "EqualsByVertexElement")
		actual.Members[key] = &actualMember{Key: key, Kind: "method", Parameters: []string{"VertexElement"}, Results: []string{"bool"}}
	case "vertex_missing_equality_operator":
		delete(actual.Members, function("VertexElementOperatorEqualityByVertexElementAndVertexElement"))
	case "vertex_missing_inequality_operator":
		delete(actual.Members, function("VertexElementOperatorInequalityByVertexElementAndVertexElement"))
	case "vertex_format_wrong_enum_value":
		wrong := "12"
		actual.Members[function("VertexElementFormatHalfVector4")].Value = &wrong
	case "vertex_usage_wrong_enum_value":
		wrong := "13"
		actual.Members[function("VertexElementUsageTessellateFactor")].Value = &wrong
	case "vertex_enum_accidentally_flags":
		actual.Types[symbolKey{Package: pkg, Name: "VertexElementFormat"}].FlagsMarker = true
	default:
		t.Fatalf("unknown vertex mutation %q", mutation)
	}
	return expected, actual
}

func mutationCase(mutation string) (*expectedSurface, *actualSurface) {
	const pkg = modulePath + "/Microsoft/Xna/Framework"
	typeKey := symbolKey{Package: pkg, Name: "Probe"}
	memberKey := symbolKey{Package: pkg, Receiver: "Probe", Name: "Act"}
	et := &expectedType{Key: typeKey, XNA: "Microsoft.Xna.Framework.Probe", GoName: "Probe", PackagePath: pkg, Kind: "struct", Members: []symbolKey{memberKey}}
	em := &expectedMember{Key: memberKey, XNA: "Microsoft.Xna.Framework.Probe::Act(System.Int32)", Owner: et.XNA, SourceKind: "method", GoKind: "method", GoName: "Act", PackagePath: pkg, Receiver: "Probe", Parameters: []string{"int32"}, Results: []string{"bool"}}
	expected := &expectedSurface{Types: map[symbolKey]*expectedType{typeKey: et}, Members: map[symbolKey]*expectedMember{memberKey: em}, InterfaceWitnesses: make(map[symbolKey]*expectedInterfaceWitness), ReferenceTypes: 1, ReferenceMembers: 1, ExpectedGoTypes: 1, ExpectedGoMembers: 1}
	at := &actualType{Key: typeKey, Kind: "struct", Underlying: "struct{}"}
	am := &actualMember{Key: memberKey, Kind: "method", Parameters: []string{"int32"}, Results: []string{"bool"}}
	actual := &actualSurface{Types: map[symbolKey]*actualType{typeKey: at}, Members: map[symbolKey]*actualMember{memberKey: am}, PackageDirs: map[string]string{}}

	switch mutation {
	case "missing_type":
		delete(actual.Types, typeKey)
	case "missing_method":
		delete(actual.Members, memberKey)
	case "wrong_package":
		delete(actual.Types, typeKey)
		wrong := symbolKey{Package: pkg + "/Wrong", Name: "Probe"}
		at.Key = wrong
		actual.Types[wrong] = at
	case "wrong_kind":
		at.Kind = "interface"
	case "wrong_field":
		em.SourceKind, em.GoKind, am.Kind = "field", "field", "method"
	case "wrong_property":
		em.SourceKind, em.GoKind, am.Kind = "property", "method", "field"
	case "wrong_constructor":
		em.SourceKind, em.GoKind, em.Receiver, em.GoName = "constructor", "func", "", "NewProbe"
		delete(expected.Members, memberKey)
		em.Key = symbolKey{Package: pkg, Name: "NewProbe"}
		et.Members = []symbolKey{em.Key}
		expected.Members[em.Key] = em
	case "wrong_overload":
		em.OverloadMapped, em.GoName = true, "ActByInt32"
		delete(expected.Members, memberKey)
		em.Key = symbolKey{Package: pkg, Receiver: "Probe", Name: em.GoName}
		et.Members = []symbolKey{em.Key}
		expected.Members[em.Key] = em
	case "wrong_parameter":
		am.Parameters = []string{"uint32"}
	case "wrong_result":
		am.Results = []string{"int32"}
	case "wrong_error":
		em.ErrorAdded = true
		em.Results = []string{"bool", "error"}
	case "wrong_enum":
		value := "1"
		wrong := "2"
		em.SourceKind, em.GoKind, em.EnumValue, am.Kind, am.Value = "field", "const", &value, "const", &wrong
	case "wrong_flags":
		et.Kind, et.Flags, at.Kind, at.Underlying = "enum", true, "named", "int32"
	case "wrong_static_prefix":
		em.Receiver, em.GoKind, em.GoName = "", "func", "ProbeAct"
		delete(expected.Members, memberKey)
		em.Key = symbolKey{Package: pkg, Name: "ProbeAct"}
		et.Members = []symbolKey{em.Key}
		expected.Members[em.Key] = em
	case "wrong_operator":
		em.XNA, em.OverloadMapped, em.Receiver, em.GoKind, em.GoName = "Microsoft.Xna.Framework.Probe::op_Addition(Probe,Probe)", true, "", "func", "ProbeOperatorAdditionByProbeAndProbe"
		delete(expected.Members, memberKey)
		em.Key = symbolKey{Package: pkg, Name: em.GoName}
		et.Members = []symbolKey{em.Key}
		expected.Members[em.Key] = em
		delete(actual.Members, memberKey)
		wrong := symbolKey{Package: pkg, Name: "ProbeOperatorAddition"}
		actual.Members[wrong] = &actualMember{Key: wrong, Kind: "func"}
	case "wrong_ref_out":
		em.XNA = "Microsoft.Xna.Framework.Probe::Act(out System.Int32)"
		am.Parameters = []string{"*int32"}
	case "wrong_nested":
		delete(actual.Types, typeKey)
		wrong := symbolKey{Package: pkg, Name: "Inner"}
		actual.Types[wrong] = &actualType{Key: wrong, Kind: "struct"}
	case "wrong_generic":
		et.GenericParameter = []string{"T"}
	case "wrong_collection_interface":
		et.Kind = "class"
		et.AllInterfaces = []string{"System.Collections.Generic.ICollection`1[Microsoft.Xna.Framework.Probe]"}
	case "wrong_event":
		em.SourceKind, em.GoKind, em.GoName = "event", "method", "AddChangedHandler"
		delete(expected.Members, memberKey)
		em.Key = symbolKey{Package: pkg, Receiver: "Probe", Name: em.GoName}
		et.Members = []symbolKey{em.Key}
		expected.Members[em.Key] = em
		delete(actual.Members, memberKey)
		wrong := symbolKey{Package: pkg, Receiver: "Probe", Name: "AddChanged"}
		actual.Members[wrong] = &actualMember{Key: wrong, Kind: "method"}
	case "unexpected":
		wrong := symbolKey{Package: pkg, Name: "Invented"}
		actual.Members[wrong] = &actualMember{Key: wrong, Kind: "func"}
	case "pointer_leak":
		am.Parameters = []string{"unsafe.Pointer"}
	case "handle_leak":
		wrong := symbolKey{Package: pkg, Name: "NativeHandle"}
		actual.Types[wrong] = &actualType{Key: wrong, Kind: "named", Underlying: "uintptr"}
	case "internal_leak":
		am.Results = []string{"interop.GameRef"}
	case "unmeasured":
		actual.Unmeasured = []string{"probe.go"}
	case "missing_packed_witness", "wrong_packed_witness":
		witnessKey := symbolKey{Package: pkg, Receiver: "Probe", Name: "PackFromVector4"}
		expected.InterfaceWitnesses[witnessKey] = &expectedInterfaceWitness{
			Key: witnessKey, Owner: et.XNA, SourceInterface: packedVectorNamespace + "IPackedVector",
			InterfaceMember: packedVectorNamespace + "IPackedVector::PackFromVector4(Microsoft.Xna.Framework.Vector4)",
			GoName:          "PackFromVector4", Parameters: []string{"Vector4"},
		}
		if mutation == "wrong_packed_witness" {
			actual.Members[witnessKey] = &actualMember{Key: witnessKey, Kind: "method", Parameters: []string{"Vector3"}}
		}
	case "wrong_packed_tovector_result":
		witnessKey := symbolKey{Package: pkg, Receiver: "Probe", Name: "ToVector4"}
		expected.InterfaceWitnesses[witnessKey] = &expectedInterfaceWitness{
			Key: witnessKey, Owner: et.XNA, SourceInterface: packedVectorNamespace + "IPackedVector",
			InterfaceMember: packedVectorNamespace + "IPackedVector::ToVector4()",
			GoName:          "ToVector4", Results: []string{"Vector4"},
		}
		actual.Members[witnessKey] = &actualMember{Key: witnessKey, Kind: "method", Results: []string{"Vector3"}}
	case "bogus_packed_witness":
		bogus := symbolKey{Package: pkg, Receiver: "Probe", Name: "InventedWitness"}
		actual.Members[bogus] = &actualMember{Key: bogus, Kind: "method"}
	case "wrong_packed_setter":
		em.GoName = "SetPackedValue"
		em.Parameters = []string{"uint8"}
		am.Parameters = []string{"uint16"}
	case "missing_packed_inheritance":
		et.Kind = "interface"
		at.Kind = "interface"
		et.MappedInterfaces = []mappedInterface{{XNA: packedVectorNamespace + "IPackedVector", GoName: "IPackedVector"}}
	case "wrong_packed_generic_name":
		delete(expected.Types, typeKey)
		delete(actual.Types, typeKey)
		expectedKey := symbolKey{Package: pkg, Name: "IPackedVectorOfTPacked"}
		et.Key, et.GoName, et.Kind, et.GenericParameter = expectedKey, expectedKey.Name, "interface", []string{"TPacked"}
		expected.Types[expectedKey] = et
		wrongKey := symbolKey{Package: pkg, Name: "IPackedVectorGeneric"}
		at.Key, at.Kind, at.TypeParameters = wrongKey, "interface", []string{"TPacked"}
		actual.Types[wrongKey] = at
	}
	return expected, actual
}
