package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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
	// The expected Go surface is pinned by its THREE provenance classes rather
	// than by one total. 3243 is the projection of the 2,964 XNA-declared
	// reference members and must never move; the BCL-inherited and
	// XNA-inherited projections are the surfaces the two composition
	// projections make representable, and each is pinned separately so a
	// change in any one class is attributed rather than absorbed.
	declared := surface.ExpectedGoMembers - surface.BCLInheritedProjections - surface.XNAInheritedProjections
	if surface.ExpectedGoTypes != 257 || declared != 3243 {
		t.Fatalf("XNA-declared mapped counts = %d/%d", surface.ExpectedGoTypes, declared)
	}
	if surface.BCLInheritedCLRMembers != 11 || surface.BCLInheritedProjections != 12 {
		t.Fatalf("BCL inherited counts = %d CLR members/%d projections", surface.BCLInheritedCLRMembers, surface.BCLInheritedProjections)
	}
	if surface.XNAInheritedCLRMembers != 14 || surface.XNAInheritedProjections != 24 {
		t.Fatalf("XNA inherited counts = %d CLR members/%d projections", surface.XNAInheritedCLRMembers, surface.XNAInheritedProjections)
	}
	if surface.ExpectedGoMembers != 3279 {
		t.Fatalf("mapped counts = %d/%d", surface.ExpectedGoTypes, surface.ExpectedGoMembers)
	}
	// Every expected Go member has exactly one provenance class, so the three
	// partitions are disjoint and exhaust the surface.
	declaredMembers, bclInherited, xnaInherited := 0, 0, 0
	for _, member := range surface.Members {
		switch {
		case member.BCLBase != "" && member.XNABase != "":
			t.Fatalf("member %s carries two provenance classes", member.Key.String())
		case member.BCLBase != "":
			bclInherited++
			if member.BCLMember == "" {
				t.Fatalf("inherited member %s names no CLR base member", member.Key.String())
			}
		case member.XNABase != "":
			xnaInherited++
			if member.XNABaseMember == "" {
				t.Fatalf("XNA-inherited member %s names no CLR base member", member.Key.String())
			}
		default:
			declaredMembers++
		}
	}
	if declaredMembers != declared || bclInherited != surface.BCLInheritedProjections || xnaInherited != surface.XNAInheritedProjections {
		t.Fatalf("provenance partition = %d declared/%d BCL-inherited/%d XNA-inherited, want %d/%d/%d",
			declaredMembers, bclInherited, xnaInherited, declared, surface.BCLInheritedProjections, surface.XNAInheritedProjections)
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

func TestDisplayOrientationGraphicsManagerMappedContract(t *testing.T) {
	reference := loadPinnedContract(t)
	surface, err := buildExpected(reference)
	if err != nil {
		t.Fatal(err)
	}

	display := surface.typeForXNA(displayOrientationIdentity)
	if display == nil || display.Kind != "enum" || !display.Flags || display.SourceMembers != 5 || len(display.Members) != 4 {
		t.Fatalf("DisplayOrientation projection = %+v", display)
	}
	for name, value := range map[string]string{"Default": "0", "LandscapeLeft": "1", "LandscapeRight": "2", "Portrait": "4"} {
		member := surface.Members[symbolKey{Package: display.PackagePath, Name: "DisplayOrientation" + name}]
		if member == nil || member.GoKind != "const" || member.EnumValue == nil || *member.EnumValue != value || !equalStrings(member.Results, []string{"DisplayOrientation"}) {
			t.Fatalf("DisplayOrientation%s projection = %+v", name, member)
		}
	}

	manager := surface.typeForXNA(graphicsManagerIdentity)
	getter := surface.Members[symbolKey{Package: manager.PackagePath, Receiver: manager.GoName, Name: supportedOrientationsName}]
	setter := surface.Members[symbolKey{Package: manager.PackagePath, Receiver: manager.GoName, Name: "Set" + supportedOrientationsName}]
	if getter == nil || setter == nil || getter.SourceKind != "property" || setter.SourceKind != "property" ||
		!equalStrings(getter.Results, []string{"DisplayOrientation"}) || len(getter.Parameters) != 0 || getter.ErrorAdded ||
		!equalStrings(setter.Parameters, []string{"DisplayOrientation"}) || len(setter.Results) != 0 || setter.ErrorAdded {
		t.Fatalf("SupportedOrientations projection = getter %+v setter %+v", getter, setter)
	}
}

func TestBufferUsageMappedContract(t *testing.T) {
	reference := loadPinnedContract(t)
	surface, err := buildExpected(reference)
	if err != nil {
		t.Fatal(err)
	}

	usage := surface.typeForXNA(bufferUsageIdentity)
	if usage == nil || usage.Kind != "enum" || !usage.Flags || usage.SourceMembers != 3 || len(usage.Members) != 2 {
		t.Fatalf("BufferUsage projection = %+v", usage)
	}
	for name, value := range map[string]string{"None": "0", "WriteOnly": "1"} {
		member := surface.Members[symbolKey{Package: usage.PackagePath, Name: "BufferUsage" + name}]
		if member == nil || member.GoKind != "const" || member.EnumValue == nil || *member.EnumValue != value || !equalStrings(member.Results, []string{"BufferUsage"}) {
			t.Fatalf("BufferUsage%s projection = %+v", name, member)
		}
	}
	if surface.Members[symbolKey{Package: usage.PackagePath, Name: "BufferUsageValue__"}] != nil ||
		surface.Members[symbolKey{Package: usage.PackagePath, Name: "BufferUsagevalue__"}] != nil {
		t.Fatal("enum value__ storage was projected")
	}
}

func TestClearOptionsMappedContract(t *testing.T) {
	reference := loadPinnedContract(t)
	surface, err := buildExpected(reference)
	if err != nil {
		t.Fatal(err)
	}

	options := surface.typeForXNA(clearOptionsIdentity)
	if options == nil || options.Kind != "enum" || !options.Flags || options.SourceMembers != 4 || len(options.Members) != 3 {
		t.Fatalf("ClearOptions projection = %+v", options)
	}
	for name, value := range map[string]string{"Target": "1", "DepthBuffer": "2", "Stencil": "4"} {
		member := surface.Members[symbolKey{Package: options.PackagePath, Name: "ClearOptions" + name}]
		if member == nil || member.GoKind != "const" || member.EnumValue == nil || *member.EnumValue != value || !equalStrings(member.Results, []string{"ClearOptions"}) {
			t.Fatalf("ClearOptions%s projection = %+v", name, member)
		}
	}
	for _, name := range []string{"Value__", "value__", "None", "Default", "All"} {
		if surface.Members[symbolKey{Package: options.PackagePath, Name: "ClearOptions" + name}] != nil {
			t.Fatalf("undeclared ClearOptions%s was projected", name)
		}
	}
	if enumHasNamedZero(surface, options) {
		t.Fatal("ClearOptions unexpectedly has a source-declared zero literal")
	}
}

func TestSurfaceFormatMappedContract(t *testing.T) {
	reference := loadPinnedContract(t)
	surface, err := buildExpected(reference)
	if err != nil {
		t.Fatal(err)
	}

	format := surface.typeForXNA(surfaceFormatIdentity)
	if format == nil || format.Kind != "enum" || format.Flags || format.SourceMembers != 21 || len(format.Members) != 20 || len(format.Interfaces) != 0 {
		t.Fatalf("SurfaceFormat projection = %+v", format)
	}
	for _, wanted := range surfaceFormatValues {
		member := surface.Members[symbolKey{Package: format.PackagePath, Name: "SurfaceFormat" + wanted.Name}]
		if member == nil || member.GoKind != "const" || member.EnumValue == nil || *member.EnumValue != wanted.Value || !equalStrings(member.Results, []string{"SurfaceFormat"}) {
			t.Fatalf("SurfaceFormat%s projection = %+v", wanted.Name, member)
		}
	}
	for _, name := range []string{"Value__", "value__"} {
		if surface.Members[symbolKey{Package: format.PackagePath, Name: "SurfaceFormat" + name}] != nil {
			t.Fatalf("enum storage SurfaceFormat%s was projected", name)
		}
	}
	if !enumHasNamedZero(surface, format) {
		t.Fatal("SurfaceFormat Color=0 was not measured as the source-declared zero literal")
	}
}

func TestButtonStateMappedContract(t *testing.T) {
	reference := loadPinnedContract(t)
	surface, err := buildExpected(reference)
	if err != nil {
		t.Fatal(err)
	}

	state := surface.typeForXNA(buttonStateIdentity)
	if state == nil || state.Kind != "enum" || state.Flags || state.SourceMembers != 3 || len(state.Members) != 2 || len(state.Interfaces) != 0 {
		t.Fatalf("ButtonState projection = %+v", state)
	}
	if state.PackagePath != modulePath+"/Microsoft/Xna/Framework/Input" {
		t.Fatalf("ButtonState package = %q", state.PackagePath)
	}
	for _, wanted := range buttonStateValues {
		member := surface.Members[symbolKey{Package: state.PackagePath, Name: "ButtonState" + wanted.Name}]
		if member == nil || member.GoKind != "const" || member.EnumValue == nil || *member.EnumValue != wanted.Value || !equalStrings(member.Results, []string{"ButtonState"}) {
			t.Fatalf("ButtonState%s projection = %+v", wanted.Name, member)
		}
	}
	for _, name := range []string{"Value__", "value__"} {
		if surface.Members[symbolKey{Package: state.PackagePath, Name: "ButtonState" + name}] != nil {
			t.Fatalf("enum storage ButtonState%s was projected", name)
		}
	}
	if !enumHasNamedZero(surface, state) {
		t.Fatal("ButtonState Released=0 was not measured as the source-declared zero literal")
	}
}

func TestGraphicsProfileMappedContract(t *testing.T) {
	reference := loadPinnedContract(t)
	surface, err := buildExpected(reference)
	if err != nil {
		t.Fatal(err)
	}

	profile := surface.typeForXNA(graphicsProfileIdentity)
	if profile == nil || profile.Kind != "enum" || profile.Flags || profile.SourceMembers != 3 || len(profile.Members) != 2 || len(profile.Interfaces) != 0 {
		t.Fatalf("GraphicsProfile projection = %+v", profile)
	}
	for _, wanted := range graphicsProfileValues {
		member := surface.Members[symbolKey{Package: profile.PackagePath, Name: "GraphicsProfile" + wanted.Name}]
		if member == nil || member.GoKind != "const" || member.EnumValue == nil || *member.EnumValue != wanted.Value || !equalStrings(member.Results, []string{"GraphicsProfile"}) {
			t.Fatalf("GraphicsProfile%s projection = %+v", wanted.Name, member)
		}
	}
	for _, name := range []string{"Value__", "value__"} {
		if surface.Members[symbolKey{Package: profile.PackagePath, Name: "GraphicsProfile" + name}] != nil {
			t.Fatalf("enum storage GraphicsProfile%s was projected", name)
		}
	}
	if !enumHasNamedZero(surface, profile) {
		t.Fatal("GraphicsProfile Reach=0 was not measured as the source-declared zero literal")
	}
}

func TestDepthFormatMappedContract(t *testing.T) {
	reference := loadPinnedContract(t)
	surface, err := buildExpected(reference)
	if err != nil {
		t.Fatal(err)
	}

	format := surface.typeForXNA(depthFormatIdentity)
	if format == nil || format.Kind != "enum" || format.Flags || format.SourceMembers != 5 || len(format.Members) != 4 || len(format.Interfaces) != 0 {
		t.Fatalf("DepthFormat projection = %+v", format)
	}
	for _, wanted := range depthFormatValues {
		member := surface.Members[symbolKey{Package: format.PackagePath, Name: "DepthFormat" + wanted.Name}]
		if member == nil || member.GoKind != "const" || member.EnumValue == nil || *member.EnumValue != wanted.Value || !equalStrings(member.Results, []string{"DepthFormat"}) {
			t.Fatalf("DepthFormat%s projection = %+v", wanted.Name, member)
		}
	}
	for _, name := range []string{"Value__", "value__"} {
		if surface.Members[symbolKey{Package: format.PackagePath, Name: "DepthFormat" + name}] != nil {
			t.Fatalf("enum storage DepthFormat%s was projected", name)
		}
	}
	if !enumHasNamedZero(surface, format) {
		t.Fatal("DepthFormat None=0 was not measured as the source-declared zero literal")
	}
}

func TestFlagsEnumWithoutNamedZeroIsValidGenerically(t *testing.T) {
	int32Type := "System.Int32"
	enumType := "Microsoft.Xna.Framework.Graphics.ProbeFlagsNoZero"
	reference := contract{
		SchemaVersion: 2,
		Profile:       "XNA 4.0 Windows runtime",
		Types: []contractType{{
			Name: enumType, Kind: "enum", Flags: true, Sealed: true,
			UnderlyingType: &int32Type,
			Members: []contractMember{
				{Kind: "field", Name: "value__", Type: &int32Type},
				{Kind: "field", Name: "First", Type: &enumType, Static: true, Constant: true, Value: json.RawMessage(`"1"`)},
				{Kind: "field", Name: "Second", Type: &enumType, Static: true, Constant: true, Value: json.RawMessage(`"2"`)},
				{Kind: "field", Name: "Third", Type: &enumType, Static: true, Constant: true, Value: json.RawMessage(`"4"`)},
			},
		}},
	}
	expected, err := buildExpected(reference)
	if err != nil {
		t.Fatal(err)
	}
	owner := expected.typeForXNA(enumType)
	if owner == nil || owner.SourceMembers != 4 || len(owner.Members) != 3 || enumHasNamedZero(expected, owner) {
		t.Fatalf("generic no-zero flags projection = %+v", owner)
	}
	actual := &actualSurface{
		Types: map[symbolKey]*actualType{
			owner.Key: {Key: owner.Key, Kind: "named", Underlying: "int32", FlagsMarker: true},
		},
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: make(map[string]string),
		Packages:    make(map[string]*types.Package),
	}
	for _, key := range owner.Members {
		member := expected.Members[key]
		value := *member.EnumValue
		actual.Members[key] = &actualMember{Key: key, Kind: "const", Results: []string{owner.GoName}, Value: &value}
	}
	result := verify(expected, actual, 0, "report", "contract", "mapping")
	if result.Summary["TOTAL_DIAGNOSTICS"] != 0 || result.Summary["FLAGS_MAPPING_MISMATCH"] != 0 || result.Summary["ENUM_VALUE_MISMATCH"] != 0 {
		t.Fatalf("generic no-zero flags enum failed verification: %v", result.Summary)
	}
}

func TestFlagsDirectiveRequiresExactMarker(t *testing.T) {
	exact := &ast.CommentGroup{List: []*ast.Comment{{Text: "// xna:flags"}}}
	if !hasDirectiveNamed("xna:flags", exact) {
		t.Fatal("exact xna:flags marker was not detected")
	}
	for _, text := range []string{"// xna:flags=false", "// not-xna:flags", "// comment mentioning xna:flags"} {
		group := &ast.CommentGroup{List: []*ast.Comment{{Text: text}}}
		if hasDirectiveNamed("xna:flags", group) {
			t.Fatalf("non-exact flags marker %q was accepted", text)
		}
	}
}

func TestBufferUsageCurrentSurfaceAndLocalClosure(t *testing.T) {
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
	closure := result.BufferUsageClosure
	if closure.Status != "PASS" || closure.SourceTypes != 1 || closure.SourceIdentities != 3 || closure.ExpectedGoIdentities != 2 ||
		closure.TargetTypes != 1 || closure.TargetGoIdentities != 2 || closure.LocalDiagnostics != 0 || closure.ExpectedKind != "enum" ||
		closure.ActualKind != "named" || closure.UnderlyingType != "int32" || !closure.Flags || closure.NoneValue != "0" ||
		closure.WriteOnlyValue != "1" || !closure.ValueStorageExcluded {
		t.Fatalf("BufferUsage closure = %+v", closure)
	}
}

func TestClearOptionsCurrentSurfaceAndLocalClosure(t *testing.T) {
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
	closure := result.ClearOptionsClosure
	if closure.Status != "PASS" || closure.SourceTypes != 1 || closure.SourceIdentities != 4 || closure.ExpectedGoIdentities != 3 ||
		closure.TargetTypes != 1 || closure.TargetGoIdentities != 3 || closure.LocalDiagnostics != 0 || closure.ExpectedKind != "enum" ||
		closure.ActualKind != "named" || closure.UnderlyingType != "int32" || !closure.Flags || closure.TargetValue != "1" ||
		closure.DepthBufferValue != "2" || closure.StencilValue != "4" || !closure.ValueStorageExcluded || closure.NamedZeroMember ||
		closure.ClearOptionsNonePresent || closure.ClearOptionsDefaultPresent || closure.ClearOptionsAllPresent {
		t.Fatalf("ClearOptions closure = %+v", closure)
	}
}

func TestSurfaceFormatCurrentSurfaceAndLocalClosure(t *testing.T) {
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
	closure := result.SurfaceFormatClosure
	if closure.Status != "PASS" || closure.SourceTypes != 1 || closure.SourceIdentities != 21 || closure.ExpectedGoIdentities != 20 ||
		closure.TargetTypes != 1 || closure.TargetGoIdentities != 20 || closure.LocalDiagnostics != 0 || closure.ExpectedKind != "enum" ||
		closure.ActualKind != "named" || closure.UnderlyingType != "int32" || closure.Flags || !closure.ValueStorageExcluded || len(closure.Values) != 20 {
		t.Fatalf("SurfaceFormat closure = %+v", closure)
	}
	for _, row := range closure.Values {
		if row.Status != "PASS" || row.ActualValue != row.ExpectedValue {
			t.Fatalf("SurfaceFormat value row = %+v", row)
		}
	}
}

func TestButtonStateCurrentSurfaceAndLocalClosure(t *testing.T) {
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
	closure := result.ButtonStateClosure
	if closure.Status != "PASS" || closure.SourceTypes != 1 || closure.SourceIdentities != 3 || closure.ExpectedGoIdentities != 2 ||
		closure.TargetTypes != 1 || closure.TargetGoIdentities != 2 || closure.LocalDiagnostics != 0 || closure.ExpectedKind != "enum" ||
		closure.ActualKind != "named" || closure.UnderlyingType != "int32" || closure.Flags || !closure.ValueStorageExcluded || len(closure.Values) != 2 {
		t.Fatalf("ButtonState closure = %+v", closure)
	}
	for _, row := range closure.Values {
		if row.Status != "PASS" || row.ActualValue != row.ExpectedValue {
			t.Fatalf("ButtonState value row = %+v", row)
		}
	}
}

func TestGraphicsProfileCurrentSurfaceAndLocalClosure(t *testing.T) {
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
	closure := result.GraphicsProfileClosure
	if closure.Status != "PASS" || closure.SourceTypes != 1 || closure.SourceIdentities != 3 || closure.ExpectedGoIdentities != 2 ||
		closure.TargetTypes != 1 || closure.TargetGoIdentities != 2 || closure.LocalDiagnostics != 0 || closure.ExpectedKind != "enum" ||
		closure.ActualKind != "named" || closure.UnderlyingType != "int32" || closure.Flags || !closure.ValueStorageExcluded || len(closure.Values) != 2 {
		t.Fatalf("GraphicsProfile closure = %+v", closure)
	}
	for _, row := range closure.Values {
		if row.Status != "PASS" || row.ActualValue != row.ExpectedValue {
			t.Fatalf("GraphicsProfile value row = %+v", row)
		}
	}
}

func TestDepthFormatCurrentSurfaceAndLocalClosure(t *testing.T) {
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
	closure := result.DepthFormatClosure
	if closure.Status != "PASS" || closure.SourceTypes != 1 || closure.SourceIdentities != 5 || closure.ExpectedGoIdentities != 4 ||
		closure.TargetTypes != 1 || closure.TargetGoIdentities != 4 || closure.LocalDiagnostics != 0 || closure.ExpectedKind != "enum" ||
		closure.ActualKind != "named" || closure.UnderlyingType != "int32" || closure.Flags || !closure.ValueStorageExcluded || len(closure.Values) != 4 {
		t.Fatalf("DepthFormat closure = %+v", closure)
	}
	for _, row := range closure.Values {
		if row.Status != "PASS" || row.ActualValue != row.ExpectedValue {
			t.Fatalf("DepthFormat value row = %+v", row)
		}
	}
}

func TestDisplayOrientationGraphicsManagerCurrentSurfaceAndSelectedClosure(t *testing.T) {
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
	closure := result.DisplayOrientationClosure
	if closure.Status != "PASS" || closure.SourceTypes != 2 || closure.SourceIdentities != 6 || closure.MappedGoIdentities != 6 ||
		closure.TargetTypes != 2 || closure.TargetGoIdentities != 6 || closure.DisplayOrientationLocalDiagnostics != 0 ||
		closure.SupportedPropertyLocalDiagnostics != 0 || closure.GraphicsManagerRemainingMissing != 40 || len(closure.SliceMeasurements) != 2 {
		t.Fatalf("DisplayOrientation/GDM closure = %+v", closure)
	}
	for _, row := range closure.SliceMeasurements {
		if row.LocalDiagnostics != 0 || row.TargetGoMembers != row.ExpectedGoMembers {
			t.Fatalf("DisplayOrientation/GDM slice row = %+v", row)
		}
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

// loadPinnedSurfaces builds both halves of a real verification run: the
// expected surface from the pinned contract and the actual one from the live
// checkout. It is what a measurement about the CURRENT state of the binding
// has to be run against, as opposed to a synthetic fixture.
func loadPinnedSurfaces(t *testing.T) (*expectedSurface, *actualSurface) {
	t.Helper()
	expected, err := buildExpected(loadPinnedContract(t))
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
		t.Fatalf("Go type-check admission produced %d errors; first: %s", len(actual.TypeErrors), actual.TypeErrors[0])
	}
	return expected, actual
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
			if strings.HasPrefix(fixture.Mutation, "f19rh_") {
				expected, actual = rawHandleMutationCase(t, fixture.Mutation)
				result := verify(expected, actual, 0, "leak-only", "contract", "mapping")
				if result.Summary[fixture.Category] == 0 {
					t.Fatalf("mutation %q did not trigger %s; summary=%v", fixture.Mutation, fixture.Category, result.Summary)
				}
				return
			}
			if strings.HasPrefix(fixture.Mutation, "f24bcli_") {
				expected, actual = bclInterfaceMutationCase(t, fixture.Mutation)
				result := verify(expected, actual, 0, "report", "contract", "mapping")
				if result.Summary[fixture.Category] == 0 {
					t.Fatalf("mutation %q did not trigger %s; summary=%v", fixture.Mutation, fixture.Category, result.Summary)
				}
				return
			}
			if strings.HasPrefix(fixture.Mutation, "f22ev_") {
				expected, actual = eventProjectionMutationCase(t, fixture.Mutation)
				result := verify(expected, actual, 0, "report", "contract", "mapping")
				if result.Summary[fixture.Category] == 0 {
					t.Fatalf("mutation %q did not trigger %s; summary=%v", fixture.Mutation, fixture.Category, result.Summary)
				}
				return
			}
			if strings.HasPrefix(fixture.Mutation, "f33xna_") {
				result := xnaBaseMutationCase(t, fixture.Mutation)
				if result.Summary[fixture.Category] == 0 {
					t.Fatalf("mutation %q did not trigger %s; summary=%v", fixture.Mutation, fixture.Category, result.Summary)
				}
				return
			}
			if strings.HasPrefix(fixture.Mutation, "f31base_") {
				result := gameBaseCallMutationCase(t, fixture.Mutation)
				if result.Summary[fixture.Category] == 0 {
					t.Fatalf("mutation %q did not trigger %s; summary=%v", fixture.Mutation, fixture.Category, result.Summary)
				}
				return
			}
			if strings.HasPrefix(fixture.Mutation, "f36signal_") {
				result := gameSignalMutationCase(t, fixture.Mutation)
				if result.Summary[fixture.Category] == 0 {
					t.Fatalf("mutation %q did not trigger %s; summary=%v", fixture.Mutation, fixture.Category, result.Summary)
				}
				return
			}
			if strings.HasPrefix(fixture.Mutation, "f38hook_") {
				result := gameFrameHookOverrideMutationCase(t, fixture.Mutation)
				if result.Summary[fixture.Category] == 0 {
					t.Fatalf("mutation %q did not trigger %s; summary=%v", fixture.Mutation, fixture.Category, result.Summary)
				}
				return
			}
			if strings.HasPrefix(fixture.Mutation, "f27sig_") {
				expected, actual = bclSignatureAdapterMutationCase(t, fixture.Mutation)
				result := report{Summary: make(map[string]int)}
				measureBCLSignatureAdapters(&result, expected, actual)
				if result.Summary[fixture.Category] == 0 {
					t.Fatalf("mutation %q did not trigger %s; summary=%v", fixture.Mutation, fixture.Category, result.Summary)
				}
				return
			}
			if strings.HasPrefix(fixture.Mutation, "f26bcl_") {
				expected, actual = bclCompositionMutationCase(t, fixture.Mutation)
				result := verify(expected, actual, 0, "report", "contract", "mapping")
				if result.Summary[fixture.Category] == 0 {
					t.Fatalf("mutation %q did not trigger %s; summary=%v", fixture.Mutation, fixture.Category, result.Summary)
				}
				return
			}
			if strings.HasPrefix(fixture.Mutation, "f22base_") {
				expected, actual = baseProjectionMutationCase(t, fixture.Mutation)
				result := verify(expected, actual, 0, "report", "contract", "mapping")
				if result.Summary[fixture.Category] == 0 {
					t.Fatalf("mutation %q did not trigger %s; summary=%v", fixture.Mutation, fixture.Category, result.Summary)
				}
				return
			}
			if strings.HasPrefix(fixture.Mutation, "f18cls_") {
				result := interfaceClassificationMutationCase(t, fixture.Mutation)
				if result.Summary[fixture.Category] == 0 {
					t.Fatalf("mutation %q did not trigger %s; summary=%v", fixture.Mutation, fixture.Category, result.Summary)
				}
				return
			}
			if strings.HasPrefix(fixture.Mutation, "f18if_") {
				expected, actual = managedInterfaceMutationCase(t, fixture.Mutation)
				result := verify(expected, actual, 0, "report", "contract", "mapping")
				if result.Summary[fixture.Category] == 0 {
					t.Fatalf("mutation %q did not trigger %s; summary=%v", fixture.Mutation, fixture.Category, result.Summary)
				}
				return
			}
			if strings.HasPrefix(fixture.Mutation, "f17cls_") {
				result := classificationMutationCase(t, fixture.Mutation)
				if result.Summary[fixture.Category] == 0 {
					t.Fatalf("mutation %q did not trigger %s; summary=%v", fixture.Mutation, fixture.Category, result.Summary)
				}
				return
			}
			if strings.HasPrefix(fixture.Mutation, "f17mc_") {
				expected, actual = managedClassMutationCase(t, fixture.Mutation)
			} else if strings.HasPrefix(fixture.Mutation, "f15vs_") {
				expected, actual = valueStructMutationCase(t, fixture.Mutation)
			} else if strings.HasPrefix(fixture.Mutation, "f14_") {
				expected, actual = foundation14EnumMutationCase(t, fixture.Mutation)
			} else if strings.HasPrefix(fixture.Mutation, "button_state_") {
				expected, actual = buttonStateMutationCase(t, fixture.Mutation)
			} else if strings.HasPrefix(fixture.Mutation, "graphics_profile_") {
				expected, actual = graphicsProfileMutationCase(t, fixture.Mutation)
			} else if strings.HasPrefix(fixture.Mutation, "depth_format_") {
				expected, actual = depthFormatMutationCase(t, fixture.Mutation)
			} else if strings.HasPrefix(fixture.Mutation, "surface_format_") {
				expected, actual = surfaceFormatMutationCase(t, fixture.Mutation)
			} else if strings.HasPrefix(fixture.Mutation, "clear_options_") {
				expected, actual = clearOptionsMutationCase(t, fixture.Mutation)
			} else if strings.HasPrefix(fixture.Mutation, "buffer_usage_") {
				expected, actual = bufferUsageMutationCase(t, fixture.Mutation)
			} else if strings.HasPrefix(fixture.Mutation, "display_orientation_") || strings.HasPrefix(fixture.Mutation, "graphics_manager_orientation_") {
				expected, actual = displayOrientationMutationCase(t, fixture.Mutation)
			} else if strings.HasPrefix(fixture.Mutation, "player_index_") || strings.HasPrefix(fixture.Mutation, "keyboard_player_index_") {
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

func buttonStateMutationCase(t *testing.T, mutation string) (*expectedSurface, *actualSurface) {
	t.Helper()
	full, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	fullType := full.typeForXNA(buttonStateIdentity)
	copiedType := *fullType
	copiedType.Members = append([]symbolKey(nil), fullType.Members...)
	expected := &expectedSurface{
		Types:              map[symbolKey]*expectedType{copiedType.Key: &copiedType},
		Members:            make(map[symbolKey]*expectedMember),
		InterfaceWitnesses: make(map[symbolKey]*expectedInterfaceWitness),
		ReferenceTypes:     1,
		ReferenceMembers:   3,
		ExpectedGoTypes:    1,
		ExpectedGoMembers:  2,
	}
	actual := &actualSurface{
		Types: map[symbolKey]*actualType{
			copiedType.Key: {Key: copiedType.Key, Kind: "named", Underlying: "int32"},
		},
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: make(map[string]string),
		Packages:    make(map[string]*types.Package),
	}
	for _, memberKey := range copiedType.Members {
		fullMember := full.Members[memberKey]
		copiedMember := *fullMember
		copiedMember.Parameters = append([]string(nil), fullMember.Parameters...)
		copiedMember.Results = append([]string(nil), fullMember.Results...)
		expected.Members[memberKey] = &copiedMember
		value := *copiedMember.EnumValue
		actual.Members[memberKey] = &actualMember{Key: memberKey, Kind: "const", Results: []string{"ButtonState"}, Value: &value}
	}

	const inputPackage = modulePath + "/Microsoft/Xna/Framework/Input"
	const graphicsPackage = modulePath + "/Microsoft/Xna/Framework/Graphics"
	typeKey := symbolKey{Package: inputPackage, Name: "ButtonState"}
	constant := func(name string) symbolKey {
		return symbolKey{Package: inputPackage, Name: "ButtonState" + name}
	}
	setWrongValue := func(name, value string) { actual.Members[constant(name)].Value = &value }
	rename := func(from, to string) {
		original := constant(from)
		renamed := constant(to)
		member := *actual.Members[original]
		delete(actual.Members, original)
		member.Key = renamed
		actual.Members[renamed] = &member
	}
	switch mutation {
	case "button_state_missing":
		delete(actual.Types, typeKey)
	case "button_state_wrong_package":
		movedType := *actual.Types[typeKey]
		delete(actual.Types, typeKey)
		movedType.Key.Package = graphicsPackage
		actual.Types[movedType.Key] = &movedType
		for _, wanted := range buttonStateValues {
			key := constant(wanted.Name)
			movedMember := *actual.Members[key]
			delete(actual.Members, key)
			movedMember.Key.Package = graphicsPackage
			actual.Members[movedMember.Key] = &movedMember
		}
	case "button_state_wrong_kind":
		actual.Types[typeKey].Kind = "struct"
	case "button_state_wrong_underlying_type":
		actual.Types[typeKey].Underlying = "uint32"
	case "button_state_accidentally_flags":
		actual.Types[typeKey].FlagsMarker = true
	case "button_state_wrong_released_value":
		setWrongValue("Released", "1")
	case "button_state_wrong_pressed_value":
		setWrongValue("Pressed", "2")
	case "button_state_missing_pressed":
		delete(actual.Members, constant("Pressed"))
	case "button_state_value_storage_projected":
		key := constant("Value__")
		value := "0"
		actual.Members[key] = &actualMember{Key: key, Kind: "const", Results: []string{"int32"}, Value: &value}
	case "button_state_extra_constant":
		key := constant("None")
		value := "2"
		actual.Members[key] = &actualMember{Key: key, Kind: "const", Results: []string{"ButtonState"}, Value: &value}
	case "button_state_renamed_released":
		rename("Released", "Release")
	case "button_state_renamed_pressed":
		rename("Pressed", "Press")
	case "button_state_exported_helper":
		key := symbolKey{Package: inputPackage, Receiver: "ButtonState", Name: "IsPressed"}
		actual.Members[key] = &actualMember{Key: key, Kind: "method", Results: []string{"bool"}}
	default:
		t.Fatalf("unknown ButtonState mutation %q", mutation)
	}
	return expected, actual
}

func graphicsProfileMutationCase(t *testing.T, mutation string) (*expectedSurface, *actualSurface) {
	t.Helper()
	full, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	fullType := full.typeForXNA(graphicsProfileIdentity)
	copiedType := *fullType
	copiedType.Members = append([]symbolKey(nil), fullType.Members...)
	expected := &expectedSurface{
		Types:              map[symbolKey]*expectedType{copiedType.Key: &copiedType},
		Members:            make(map[symbolKey]*expectedMember),
		InterfaceWitnesses: make(map[symbolKey]*expectedInterfaceWitness),
		ReferenceTypes:     1,
		ReferenceMembers:   3,
		ExpectedGoTypes:    1,
		ExpectedGoMembers:  2,
	}
	actual := &actualSurface{
		Types: map[symbolKey]*actualType{
			copiedType.Key: {Key: copiedType.Key, Kind: "named", Underlying: "int32"},
		},
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: make(map[string]string),
		Packages:    make(map[string]*types.Package),
	}
	for _, memberKey := range copiedType.Members {
		fullMember := full.Members[memberKey]
		copiedMember := *fullMember
		copiedMember.Parameters = append([]string(nil), fullMember.Parameters...)
		copiedMember.Results = append([]string(nil), fullMember.Results...)
		expected.Members[memberKey] = &copiedMember
		value := *copiedMember.EnumValue
		actual.Members[memberKey] = &actualMember{Key: memberKey, Kind: "const", Results: []string{"GraphicsProfile"}, Value: &value}
	}

	const graphicsPackage = modulePath + "/Microsoft/Xna/Framework/Graphics"
	const frameworkPackage = modulePath + "/Microsoft/Xna/Framework"
	typeKey := symbolKey{Package: graphicsPackage, Name: "GraphicsProfile"}
	constant := func(name string) symbolKey {
		return symbolKey{Package: graphicsPackage, Name: "GraphicsProfile" + name}
	}
	setWrongValue := func(name, value string) { actual.Members[constant(name)].Value = &value }
	switch mutation {
	case "graphics_profile_missing":
		delete(actual.Types, typeKey)
	case "graphics_profile_wrong_package":
		movedType := *actual.Types[typeKey]
		delete(actual.Types, typeKey)
		movedType.Key.Package = frameworkPackage
		actual.Types[movedType.Key] = &movedType
		for _, wanted := range graphicsProfileValues {
			key := constant(wanted.Name)
			movedMember := *actual.Members[key]
			delete(actual.Members, key)
			movedMember.Key.Package = frameworkPackage
			actual.Members[movedMember.Key] = &movedMember
		}
	case "graphics_profile_wrong_kind":
		actual.Types[typeKey].Kind = "struct"
	case "graphics_profile_wrong_underlying_type":
		actual.Types[typeKey].Underlying = "uint32"
	case "graphics_profile_accidentally_flags":
		actual.Types[typeKey].FlagsMarker = true
	case "graphics_profile_wrong_reach_value":
		setWrongValue("Reach", "1")
	case "graphics_profile_wrong_hidef_value":
		setWrongValue("HiDef", "2")
	case "graphics_profile_missing_hidef":
		delete(actual.Members, constant("HiDef"))
	case "graphics_profile_value_storage_projected":
		key := constant("Value__")
		value := "0"
		actual.Members[key] = &actualMember{Key: key, Kind: "const", Results: []string{"int32"}, Value: &value}
	case "graphics_profile_extra_constant":
		key := constant("Default")
		value := "2"
		actual.Members[key] = &actualMember{Key: key, Kind: "const", Results: []string{"GraphicsProfile"}, Value: &value}
	case "graphics_profile_renamed_hidef":
		original := constant("HiDef")
		renamed := constant("Hidef")
		member := *actual.Members[original]
		delete(actual.Members, original)
		member.Key = renamed
		actual.Members[renamed] = &member
	case "graphics_profile_exported_helper":
		key := symbolKey{Package: graphicsPackage, Receiver: "GraphicsProfile", Name: "String"}
		actual.Members[key] = &actualMember{Key: key, Kind: "method", Results: []string{"string"}}
	default:
		t.Fatalf("unknown GraphicsProfile mutation %q", mutation)
	}
	return expected, actual
}

func depthFormatMutationCase(t *testing.T, mutation string) (*expectedSurface, *actualSurface) {
	t.Helper()
	full, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	fullType := full.typeForXNA(depthFormatIdentity)
	copiedType := *fullType
	copiedType.Members = append([]symbolKey(nil), fullType.Members...)
	expected := &expectedSurface{
		Types:              map[symbolKey]*expectedType{copiedType.Key: &copiedType},
		Members:            make(map[symbolKey]*expectedMember),
		InterfaceWitnesses: make(map[symbolKey]*expectedInterfaceWitness),
		ReferenceTypes:     1,
		ReferenceMembers:   5,
		ExpectedGoTypes:    1,
		ExpectedGoMembers:  4,
	}
	actual := &actualSurface{
		Types: map[symbolKey]*actualType{
			copiedType.Key: {Key: copiedType.Key, Kind: "named", Underlying: "int32"},
		},
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: make(map[string]string),
		Packages:    make(map[string]*types.Package),
	}
	for _, memberKey := range copiedType.Members {
		fullMember := full.Members[memberKey]
		copiedMember := *fullMember
		copiedMember.Parameters = append([]string(nil), fullMember.Parameters...)
		copiedMember.Results = append([]string(nil), fullMember.Results...)
		expected.Members[memberKey] = &copiedMember
		value := *copiedMember.EnumValue
		actual.Members[memberKey] = &actualMember{Key: memberKey, Kind: "const", Results: []string{"DepthFormat"}, Value: &value}
	}

	const graphicsPackage = modulePath + "/Microsoft/Xna/Framework/Graphics"
	const frameworkPackage = modulePath + "/Microsoft/Xna/Framework"
	typeKey := symbolKey{Package: graphicsPackage, Name: "DepthFormat"}
	constant := func(name string) symbolKey { return symbolKey{Package: graphicsPackage, Name: "DepthFormat" + name} }
	setWrongValue := func(name, value string) { actual.Members[constant(name)].Value = &value }
	switch mutation {
	case "depth_format_missing":
		delete(actual.Types, typeKey)
	case "depth_format_wrong_package":
		movedType := *actual.Types[typeKey]
		delete(actual.Types, typeKey)
		movedType.Key.Package = frameworkPackage
		actual.Types[movedType.Key] = &movedType
		for _, wanted := range depthFormatValues {
			key := constant(wanted.Name)
			movedMember := *actual.Members[key]
			delete(actual.Members, key)
			movedMember.Key.Package = frameworkPackage
			actual.Members[movedMember.Key] = &movedMember
		}
	case "depth_format_wrong_kind":
		actual.Types[typeKey].Kind = "struct"
	case "depth_format_wrong_underlying_type":
		actual.Types[typeKey].Underlying = "uint32"
	case "depth_format_accidentally_flags":
		actual.Types[typeKey].FlagsMarker = true
	case "depth_format_wrong_none_value":
		setWrongValue("None", "1")
	case "depth_format_wrong_depth16_value":
		setWrongValue("Depth16", "2")
	case "depth_format_wrong_depth24_value":
		setWrongValue("Depth24", "3")
	case "depth_format_wrong_depth24_stencil8_value":
		setWrongValue("Depth24Stencil8", "4")
	case "depth_format_missing_depth24":
		delete(actual.Members, constant("Depth24"))
	case "depth_format_missing_depth24_stencil8":
		delete(actual.Members, constant("Depth24Stencil8"))
	case "depth_format_value_storage_projected":
		key := constant("Value__")
		value := "0"
		actual.Members[key] = &actualMember{Key: key, Kind: "const", Results: []string{"int32"}, Value: &value}
	case "depth_format_extra_constant":
		key := constant("Depth32")
		value := "4"
		actual.Members[key] = &actualMember{Key: key, Kind: "const", Results: []string{"DepthFormat"}, Value: &value}
	case "depth_format_exported_helper":
		key := symbolKey{Package: graphicsPackage, Receiver: "DepthFormat", Name: "String"}
		actual.Members[key] = &actualMember{Key: key, Kind: "method", Results: []string{"string"}}
	case "depth_format_renamed_depth24_stencil8":
		original := constant("Depth24Stencil8")
		renamed := constant("Depth24Stencil08")
		member := *actual.Members[original]
		delete(actual.Members, original)
		member.Key = renamed
		actual.Members[renamed] = &member
	default:
		t.Fatalf("unknown DepthFormat mutation %q", mutation)
	}
	return expected, actual
}

func surfaceFormatMutationCase(t *testing.T, mutation string) (*expectedSurface, *actualSurface) {
	t.Helper()
	full, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	fullType := full.typeForXNA(surfaceFormatIdentity)
	copiedType := *fullType
	copiedType.Members = append([]symbolKey(nil), fullType.Members...)
	expected := &expectedSurface{
		Types:              map[symbolKey]*expectedType{copiedType.Key: &copiedType},
		Members:            make(map[symbolKey]*expectedMember),
		InterfaceWitnesses: make(map[symbolKey]*expectedInterfaceWitness),
		ReferenceTypes:     1,
		ReferenceMembers:   21,
		ExpectedGoTypes:    1,
		ExpectedGoMembers:  20,
	}
	actual := &actualSurface{
		Types: map[symbolKey]*actualType{
			copiedType.Key: {Key: copiedType.Key, Kind: "named", Underlying: "int32"},
		},
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: make(map[string]string),
		Packages:    make(map[string]*types.Package),
	}
	for _, memberKey := range copiedType.Members {
		fullMember := full.Members[memberKey]
		copiedMember := *fullMember
		copiedMember.Parameters = append([]string(nil), fullMember.Parameters...)
		copiedMember.Results = append([]string(nil), fullMember.Results...)
		expected.Members[memberKey] = &copiedMember
		value := *copiedMember.EnumValue
		actual.Members[memberKey] = &actualMember{Key: memberKey, Kind: "const", Results: []string{"SurfaceFormat"}, Value: &value}
	}

	const graphicsPackage = modulePath + "/Microsoft/Xna/Framework/Graphics"
	const frameworkPackage = modulePath + "/Microsoft/Xna/Framework"
	typeKey := symbolKey{Package: graphicsPackage, Name: "SurfaceFormat"}
	constant := func(name string) symbolKey { return symbolKey{Package: graphicsPackage, Name: "SurfaceFormat" + name} }
	setWrongValue := func(name, value string) { actual.Members[constant(name)].Value = &value }
	switch mutation {
	case "surface_format_missing":
		delete(actual.Types, typeKey)
	case "surface_format_wrong_package":
		movedType := *actual.Types[typeKey]
		delete(actual.Types, typeKey)
		movedType.Key.Package = frameworkPackage
		actual.Types[movedType.Key] = &movedType
		for _, wanted := range surfaceFormatValues {
			key := constant(wanted.Name)
			movedMember := *actual.Members[key]
			delete(actual.Members, key)
			movedMember.Key.Package = frameworkPackage
			actual.Members[movedMember.Key] = &movedMember
		}
	case "surface_format_wrong_kind":
		actual.Types[typeKey].Kind = "struct"
	case "surface_format_wrong_underlying_type":
		actual.Types[typeKey].Underlying = "uint32"
	case "surface_format_accidentally_flags":
		actual.Types[typeKey].FlagsMarker = true
	case "surface_format_wrong_color_value":
		setWrongValue("Color", "1")
	case "surface_format_wrong_bgr565_value":
		setWrongValue("Bgr565", "2")
	case "surface_format_wrong_dxt1_value":
		setWrongValue("Dxt1", "5")
	case "surface_format_wrong_rgba1010102_value":
		setWrongValue("Rgba1010102", "10")
	case "surface_format_wrong_alpha8_value":
		setWrongValue("Alpha8", "13")
	case "surface_format_wrong_half_vector4_value":
		setWrongValue("HalfVector4", "19")
	case "surface_format_wrong_hdr_blendable_value":
		setWrongValue("HdrBlendable", "20")
	case "surface_format_missing_dxt3":
		delete(actual.Members, constant("Dxt3"))
	case "surface_format_missing_hdr_blendable":
		delete(actual.Members, constant("HdrBlendable"))
	case "surface_format_value_storage_projected":
		key := constant("Value__")
		value := "0"
		actual.Members[key] = &actualMember{Key: key, Kind: "const", Results: []string{"int32"}, Value: &value}
	case "surface_format_extra_constant":
		key := constant("Unknown")
		value := "20"
		actual.Members[key] = &actualMember{Key: key, Kind: "const", Results: []string{"SurfaceFormat"}, Value: &value}
	case "surface_format_exported_helper":
		key := symbolKey{Package: graphicsPackage, Receiver: "SurfaceFormat", Name: "String"}
		actual.Members[key] = &actualMember{Key: key, Kind: "method", Results: []string{"string"}}
	case "surface_format_renamed_bgr565":
		original := constant("Bgr565")
		renamed := constant("BGR565")
		member := *actual.Members[original]
		delete(actual.Members, original)
		member.Key = renamed
		actual.Members[renamed] = &member
	default:
		t.Fatalf("unknown SurfaceFormat mutation %q", mutation)
	}
	return expected, actual
}

func clearOptionsMutationCase(t *testing.T, mutation string) (*expectedSurface, *actualSurface) {
	t.Helper()
	full, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	fullType := full.typeForXNA(clearOptionsIdentity)
	copiedType := *fullType
	copiedType.Members = append([]symbolKey(nil), fullType.Members...)
	expected := &expectedSurface{
		Types:              map[symbolKey]*expectedType{copiedType.Key: &copiedType},
		Members:            make(map[symbolKey]*expectedMember),
		InterfaceWitnesses: make(map[symbolKey]*expectedInterfaceWitness),
		ReferenceTypes:     1,
		ReferenceMembers:   4,
		ExpectedGoTypes:    1,
		ExpectedGoMembers:  3,
	}
	actual := &actualSurface{
		Types: map[symbolKey]*actualType{
			copiedType.Key: {Key: copiedType.Key, Kind: "named", Underlying: "int32", FlagsMarker: true},
		},
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: make(map[string]string),
		Packages:    make(map[string]*types.Package),
	}
	for _, memberKey := range copiedType.Members {
		fullMember := full.Members[memberKey]
		copiedMember := *fullMember
		copiedMember.Parameters = append([]string(nil), fullMember.Parameters...)
		copiedMember.Results = append([]string(nil), fullMember.Results...)
		expected.Members[memberKey] = &copiedMember
		value := *copiedMember.EnumValue
		actual.Members[memberKey] = &actualMember{Key: memberKey, Kind: "const", Results: []string{"ClearOptions"}, Value: &value}
	}

	const graphicsPackage = modulePath + "/Microsoft/Xna/Framework/Graphics"
	const frameworkPackage = modulePath + "/Microsoft/Xna/Framework"
	typeKey := symbolKey{Package: graphicsPackage, Name: "ClearOptions"}
	constant := func(name string) symbolKey { return symbolKey{Package: graphicsPackage, Name: "ClearOptions" + name} }
	switch mutation {
	case "clear_options_missing":
		delete(actual.Types, typeKey)
	case "clear_options_wrong_package":
		movedType := *actual.Types[typeKey]
		delete(actual.Types, typeKey)
		movedType.Key.Package = frameworkPackage
		actual.Types[movedType.Key] = &movedType
		for _, name := range []string{"Target", "DepthBuffer", "Stencil"} {
			key := constant(name)
			movedMember := *actual.Members[key]
			delete(actual.Members, key)
			movedMember.Key.Package = frameworkPackage
			actual.Members[movedMember.Key] = &movedMember
		}
	case "clear_options_wrong_kind":
		actual.Types[typeKey].Kind = "struct"
	case "clear_options_wrong_underlying_type":
		actual.Types[typeKey].Underlying = "uint32"
	case "clear_options_missing_flags_marker", "clear_options_flags_false":
		actual.Types[typeKey].FlagsMarker = false
	case "clear_options_wrong_target_value":
		wrong := "2"
		actual.Members[constant("Target")].Value = &wrong
	case "clear_options_wrong_depth_buffer_value":
		wrong := "3"
		actual.Members[constant("DepthBuffer")].Value = &wrong
	case "clear_options_wrong_stencil_value":
		wrong := "8"
		actual.Members[constant("Stencil")].Value = &wrong
	case "clear_options_missing_stencil":
		delete(actual.Members, constant("Stencil"))
	case "clear_options_value_storage_projected":
		key := constant("Value__")
		value := "0"
		actual.Members[key] = &actualMember{Key: key, Kind: "const", Results: []string{"int32"}, Value: &value}
	case "clear_options_invented_none":
		key := constant("None")
		value := "0"
		actual.Members[key] = &actualMember{Key: key, Kind: "const", Results: []string{"ClearOptions"}, Value: &value}
	case "clear_options_invented_all":
		key := constant("All")
		value := "7"
		actual.Members[key] = &actualMember{Key: key, Kind: "const", Results: []string{"ClearOptions"}, Value: &value}
	case "clear_options_exported_helper":
		key := symbolKey{Package: graphicsPackage, Receiver: "ClearOptions", Name: "String"}
		actual.Members[key] = &actualMember{Key: key, Kind: "method", Results: []string{"string"}}
	default:
		t.Fatalf("unknown ClearOptions mutation %q", mutation)
	}
	return expected, actual
}

func bufferUsageMutationCase(t *testing.T, mutation string) (*expectedSurface, *actualSurface) {
	t.Helper()
	full, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	fullType := full.typeForXNA(bufferUsageIdentity)
	copiedType := *fullType
	copiedType.Members = append([]symbolKey(nil), fullType.Members...)
	expected := &expectedSurface{
		Types:              map[symbolKey]*expectedType{copiedType.Key: &copiedType},
		Members:            make(map[symbolKey]*expectedMember),
		InterfaceWitnesses: make(map[symbolKey]*expectedInterfaceWitness),
		ReferenceTypes:     1,
		ReferenceMembers:   3,
		ExpectedGoTypes:    1,
		ExpectedGoMembers:  2,
	}
	actual := &actualSurface{
		Types: map[symbolKey]*actualType{
			copiedType.Key: {Key: copiedType.Key, Kind: "named", Underlying: "int32", FlagsMarker: true},
		},
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: make(map[string]string),
		Packages:    make(map[string]*types.Package),
	}
	for _, memberKey := range copiedType.Members {
		fullMember := full.Members[memberKey]
		copiedMember := *fullMember
		copiedMember.Parameters = append([]string(nil), fullMember.Parameters...)
		copiedMember.Results = append([]string(nil), fullMember.Results...)
		expected.Members[memberKey] = &copiedMember
		value := *copiedMember.EnumValue
		actual.Members[memberKey] = &actualMember{Key: memberKey, Kind: "const", Results: []string{"BufferUsage"}, Value: &value}
	}

	const graphicsPackage = modulePath + "/Microsoft/Xna/Framework/Graphics"
	const frameworkPackage = modulePath + "/Microsoft/Xna/Framework"
	typeKey := symbolKey{Package: graphicsPackage, Name: "BufferUsage"}
	constant := func(name string) symbolKey { return symbolKey{Package: graphicsPackage, Name: "BufferUsage" + name} }
	switch mutation {
	case "buffer_usage_missing":
		delete(actual.Types, typeKey)
	case "buffer_usage_wrong_package":
		movedType := *actual.Types[typeKey]
		delete(actual.Types, typeKey)
		movedType.Key.Package = frameworkPackage
		actual.Types[movedType.Key] = &movedType
		for _, name := range []string{"None", "WriteOnly"} {
			key := constant(name)
			movedMember := *actual.Members[key]
			delete(actual.Members, key)
			movedMember.Key.Package = frameworkPackage
			actual.Members[movedMember.Key] = &movedMember
		}
	case "buffer_usage_wrong_kind":
		actual.Types[typeKey].Kind = "struct"
	case "buffer_usage_wrong_underlying_type":
		actual.Types[typeKey].Underlying = "uint32"
	case "buffer_usage_missing_flags_marker", "buffer_usage_flags_false":
		actual.Types[typeKey].FlagsMarker = false
	case "buffer_usage_wrong_none_value":
		wrong := "1"
		actual.Members[constant("None")].Value = &wrong
	case "buffer_usage_wrong_write_only_value":
		wrong := "2"
		actual.Members[constant("WriteOnly")].Value = &wrong
	case "buffer_usage_missing_write_only":
		delete(actual.Members, constant("WriteOnly"))
	case "buffer_usage_value_storage_projected":
		key := constant("Value__")
		value := "0"
		actual.Members[key] = &actualMember{Key: key, Kind: "const", Results: []string{"int32"}, Value: &value}
	case "buffer_usage_extra_constant":
		key := constant("Discard")
		value := "2"
		actual.Members[key] = &actualMember{Key: key, Kind: "const", Results: []string{"BufferUsage"}, Value: &value}
	case "buffer_usage_exported_helper":
		key := symbolKey{Package: graphicsPackage, Receiver: "BufferUsage", Name: "String"}
		actual.Members[key] = &actualMember{Key: key, Kind: "method", Results: []string{"string"}}
	default:
		t.Fatalf("unknown BufferUsage mutation %q", mutation)
	}
	return expected, actual
}

func displayOrientationMutationCase(t *testing.T, mutation string) (*expectedSurface, *actualSurface) {
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
		ReferenceMembers:   6,
		ExpectedGoTypes:    2,
		ExpectedGoMembers:  6,
	}
	actual := &actualSurface{
		Types:       make(map[symbolKey]*actualType),
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: make(map[string]string),
		Packages:    make(map[string]*types.Package),
	}

	copyType := func(owner *expectedType, sourceMembers int, include func(*expectedMember) bool) {
		copiedType := *owner
		copiedType.SourceMembers = sourceMembers
		copiedType.Members = nil
		for _, memberKey := range owner.Members {
			fullMember := full.Members[memberKey]
			if !include(fullMember) {
				continue
			}
			copiedMember := *fullMember
			copiedMember.Parameters = append([]string(nil), fullMember.Parameters...)
			copiedMember.Results = append([]string(nil), fullMember.Results...)
			expected.Members[memberKey] = &copiedMember
			copiedType.Members = append(copiedType.Members, memberKey)
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
		expected.Types[copiedType.Key] = &copiedType
		actualKind, underlying := "struct", "struct{}"
		if copiedType.Kind == "enum" {
			actualKind, underlying = "named", "int32"
		}
		actual.Types[copiedType.Key] = &actualType{Key: copiedType.Key, Kind: actualKind, Underlying: underlying, FlagsMarker: copiedType.Flags}
	}

	display := full.typeForXNA(displayOrientationIdentity)
	copyType(display, 5, func(*expectedMember) bool { return true })
	manager := full.typeForXNA(graphicsManagerIdentity)
	copyType(manager, 1, func(member *expectedMember) bool {
		return member.SourceKind == "property" && strings.Contains(member.XNA, "::"+supportedOrientationsName+"(")
	})

	const frameworkPackage = modulePath + "/Microsoft/Xna/Framework"
	const graphicsPackage = frameworkPackage + "/Graphics"
	displayType := symbolKey{Package: frameworkPackage, Name: "DisplayOrientation"}
	displayConstant := func(name string) symbolKey {
		return symbolKey{Package: frameworkPackage, Name: "DisplayOrientation" + name}
	}
	getter := symbolKey{Package: frameworkPackage, Receiver: "GraphicsDeviceManager", Name: supportedOrientationsName}
	setter := symbolKey{Package: frameworkPackage, Receiver: "GraphicsDeviceManager", Name: "Set" + supportedOrientationsName}

	switch mutation {
	case "display_orientation_wrong_kind":
		actual.Types[displayType].Kind = "struct"
	case "display_orientation_wrong_underlying_type":
		actual.Types[displayType].Underlying = "uint32"
	case "display_orientation_missing_flags":
		actual.Types[displayType].FlagsMarker = false
	case "display_orientation_wrong_default_value":
		wrong := "1"
		actual.Members[displayConstant("Default")].Value = &wrong
	case "display_orientation_wrong_portrait_value":
		wrong := "8"
		actual.Members[displayConstant("Portrait")].Value = &wrong
	case "display_orientation_missing_landscape_right":
		delete(actual.Members, displayConstant("LandscapeRight"))
	case "graphics_manager_orientation_missing_getter":
		delete(actual.Members, getter)
	case "graphics_manager_orientation_missing_setter", "graphics_manager_orientation_read_only":
		delete(actual.Members, setter)
	case "graphics_manager_orientation_getter_wrong_type":
		actual.Members[getter].Results = []string{"int32"}
	case "graphics_manager_orientation_setter_wrong_type":
		actual.Members[setter].Parameters = []string{"int32"}
	case "graphics_manager_orientation_setter_returns_error":
		actual.Members[setter].Results = []string{"error"}
	case "graphics_manager_orientation_static":
		member := actual.Members[getter]
		delete(actual.Members, getter)
		wrong := symbolKey{Package: frameworkPackage, Name: "GraphicsDeviceManagerSupportedOrientations"}
		member.Key = wrong
		actual.Members[wrong] = member
	case "graphics_manager_orientation_moved_to_graphics":
		for _, key := range []symbolKey{getter, setter} {
			member := actual.Members[key]
			delete(actual.Members, key)
			wrong := key
			wrong.Package = graphicsPackage
			member.Key = wrong
			actual.Members[wrong] = member
		}
	case "graphics_manager_orientation_public_dirty":
		wrong := symbolKey{Package: frameworkPackage, Receiver: "GraphicsDeviceManager", Name: "Dirty"}
		actual.Members[wrong] = &actualMember{Key: wrong, Kind: "method", Results: []string{"bool"}}
	default:
		t.Fatalf("unknown DisplayOrientation/GDM mutation %q", mutation)
	}
	return expected, actual
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

// foundation14EnumByIdentity returns the pinned Foundation-14 batch entry for
// an XNA identity.
func foundation14EnumByIdentity(t *testing.T, identity string) foundation14Enum {
	t.Helper()
	for _, pinned := range allBatchEnums() {
		if pinned.Identity == identity {
			return pinned
		}
	}
	t.Fatalf("%s is not a pinned batch enum", identity)
	return foundation14Enum{}
}

// TestBatchEnumMappedContracts admits every enum in the Foundation-14
// pure-managed batch against the pinned XNA 4.0 Windows contract. The verifier
// table and the contract must agree on kind, flags, underlying storage, the
// exact literal names, and the exact raw values, and the synthetic value__
// storage field must never reach the Go projection.
func TestBatchEnumMappedContracts(t *testing.T) {
	reference := loadPinnedContract(t)
	surface, err := buildExpected(reference)
	if err != nil {
		t.Fatal(err)
	}
	if len(foundation14Enums) != 25 {
		t.Fatalf("Foundation-14 batch size = %d, want 25", len(foundation14Enums))
	}
	if len(foundation15Enums) != 5 {
		t.Fatalf("Foundation-15 batch size = %d, want 5", len(foundation15Enums))
	}
	contractTypes := make(map[string]contractType, len(reference.Types))
	for _, declared := range reference.Types {
		contractTypes[declared.Name] = declared
	}
	batch := allBatchEnums()
	seen := make(map[string]bool, len(batch))
	identities := 0
	for _, pinned := range batch {
		if seen[pinned.Identity] {
			t.Fatalf("%s appears twice in the batch table", pinned.Identity)
		}
		seen[pinned.Identity] = true

		declared, ok := contractTypes[pinned.Identity]
		if !ok {
			t.Fatalf("%s is not in the pinned contract", pinned.Identity)
		}
		if declared.Kind != "enum" || declared.Flags != pinned.Flags ||
			valueOrEmpty(declared.UnderlyingType) != "System.Int32" ||
			valueOrEmpty(declared.BaseType) != "System.Enum" || len(declared.DirectInterfaces) != 0 {
			t.Fatalf("%s pinned shape = kind %q flags %t underlying %q base %q interfaces %v",
				pinned.Identity, declared.Kind, declared.Flags, valueOrEmpty(declared.UnderlyingType),
				valueOrEmpty(declared.BaseType), declared.DirectInterfaces)
		}

		// The contract's own literal table must match the verifier table
		// exactly, in both directions.
		contractValues := make(map[string]string)
		storage := 0
		for _, member := range declared.Members {
			if member.Kind != "field" {
				t.Fatalf("%s declares a non-field member %q", pinned.Identity, member.Name)
			}
			if member.Name == "value__" {
				storage++
				continue
			}
			contractValues[member.Name] = normalizeInteger(strings.Trim(string(member.Value), "\""))
		}
		if storage != 1 {
			t.Fatalf("%s declares %d value__ storage fields, want 1", pinned.Identity, storage)
		}
		if len(contractValues) != len(pinned.Values) {
			t.Fatalf("%s literal count = %d, want %d", pinned.Identity, len(contractValues), len(pinned.Values))
		}
		for _, wanted := range pinned.Values {
			got, ok := contractValues[wanted.Name]
			if !ok {
				t.Fatalf("%s.%s is not declared by the pinned contract", pinned.Identity, wanted.Name)
			}
			if got != normalizeInteger(wanted.Value) {
				t.Fatalf("%s.%s pinned value = %s, want %s", pinned.Identity, wanted.Name, got, wanted.Value)
			}
			delete(contractValues, wanted.Name)
		}
		if len(contractValues) != 0 {
			t.Fatalf("%s has unmapped pinned literals %v", pinned.Identity, contractValues)
		}

		mapped := surface.typeForXNA(pinned.Identity)
		if mapped == nil || mapped.Kind != "enum" || mapped.Flags != pinned.Flags ||
			mapped.SourceMembers != len(pinned.Values)+1 || len(mapped.Members) != len(pinned.Values) ||
			len(mapped.Interfaces) != 0 {
			t.Fatalf("%s projection = %+v", pinned.Identity, mapped)
		}
		namespace := pinned.Identity[:strings.LastIndex(pinned.Identity, ".")]
		if mapped.PackagePath != packagePathForNamespace(namespace) {
			t.Fatalf("%s package = %q", pinned.Identity, mapped.PackagePath)
		}
		if mapped.GoName != pinned.Identity[strings.LastIndex(pinned.Identity, ".")+1:] {
			t.Fatalf("%s Go name = %q", pinned.Identity, mapped.GoName)
		}
		for _, wanted := range pinned.Values {
			member := surface.Members[symbolKey{Package: mapped.PackagePath, Name: mapped.GoName + wanted.Name}]
			if member == nil || member.GoKind != "const" || member.EnumValue == nil ||
				normalizeInteger(*member.EnumValue) != normalizeInteger(wanted.Value) ||
				!equalStrings(member.Results, []string{mapped.GoName}) {
				t.Fatalf("%s%s projection = %+v", mapped.GoName, wanted.Name, member)
			}
		}
		for _, name := range []string{"Value__", "value__"} {
			if surface.Members[symbolKey{Package: mapped.PackagePath, Name: mapped.GoName + name}] != nil {
				t.Fatalf("enum storage %s%s was projected", mapped.GoName, name)
			}
		}
		identities += len(pinned.Values)
	}
	if identities != 167 {
		t.Fatalf("batch mapped identities = %d, want 167 (121 Foundation-14 + 46 Foundation-15)", identities)
	}
}

// foundation14EnumSurfaces builds an isolated correct expected/actual surface
// pair for one Foundation-14 batch enum, so a mutation applied afterwards is
// the only defect the verifier can see.
func foundation14EnumSurfaces(t *testing.T, identity string) (*expectedSurface, *actualSurface, *expectedType, foundation14Enum) {
	t.Helper()
	pinned := foundation14EnumByIdentity(t, identity)
	full, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	fullType := full.typeForXNA(identity)
	if fullType == nil {
		t.Fatalf("%s is not in the pinned contract", identity)
	}
	copiedType := *fullType
	copiedType.Members = append([]symbolKey(nil), fullType.Members...)
	expected := &expectedSurface{
		Types:              map[symbolKey]*expectedType{copiedType.Key: &copiedType},
		Members:            make(map[symbolKey]*expectedMember),
		InterfaceWitnesses: make(map[symbolKey]*expectedInterfaceWitness),
		ReferenceTypes:     1,
		ReferenceMembers:   len(pinned.Values) + 1,
		ExpectedGoTypes:    1,
		ExpectedGoMembers:  len(pinned.Values),
	}
	actual := &actualSurface{
		Types: map[symbolKey]*actualType{
			copiedType.Key: {Key: copiedType.Key, Kind: "named", Underlying: "int32", FlagsMarker: pinned.Flags},
		},
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: make(map[string]string),
		Packages:    make(map[string]*types.Package),
	}
	for _, memberKey := range copiedType.Members {
		fullMember := full.Members[memberKey]
		copiedMember := *fullMember
		copiedMember.Parameters = append([]string(nil), fullMember.Parameters...)
		copiedMember.Results = append([]string(nil), fullMember.Results...)
		expected.Members[memberKey] = &copiedMember
		value := *copiedMember.EnumValue
		actual.Members[memberKey] = &actualMember{
			Key: memberKey, Kind: "const", Results: []string{copiedType.GoName}, Value: &value,
		}
	}
	return expected, actual, &copiedType, pinned
}

// foundation14EnumDefects are the structural defects every Foundation-14 batch
// enum is negatively fixtured against. Each one is a way an enum projection
// could silently drift from the pinned contract.
var foundation14EnumDefects = []struct {
	Name     string
	Category string
	// FlagsOnly restricts a defect to flags enums, OrdinaryOnly to
	// non-flags enums; both false means the defect applies to every enum.
	FlagsOnly    bool
	OrdinaryOnly bool
	Apply        func(expected *expectedSurface, actual *actualSurface, owner *expectedType, pinned foundation14Enum)
}{
	{Name: "missing_type", Category: "MISSING_TYPE", Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType, _ foundation14Enum) {
		delete(actual.Types, owner.Key)
	}},
	{Name: "wrong_package", Category: "MISSING_TYPE", Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType, pinned foundation14Enum) {
		const elsewhere = modulePath + "/Microsoft/Xna/Framework"
		moved := *actual.Types[owner.Key]
		delete(actual.Types, owner.Key)
		moved.Key.Package = elsewhere
		actual.Types[moved.Key] = &moved
		for _, literal := range pinned.Values {
			key := symbolKey{Package: owner.PackagePath, Name: owner.GoName + literal.Name}
			member := *actual.Members[key]
			delete(actual.Members, key)
			member.Key.Package = elsewhere
			actual.Members[member.Key] = &member
		}
	}},
	{Name: "wrong_kind", Category: "TYPE_KIND_MISMATCH", Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType, _ foundation14Enum) {
		actual.Types[owner.Key].Kind = "struct"
	}},
	{Name: "wrong_underlying_type", Category: "TYPE_KIND_MISMATCH", Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType, _ foundation14Enum) {
		actual.Types[owner.Key].Underlying = "uint32"
	}},
	{Name: "accidentally_flags", Category: "FLAGS_MAPPING_MISMATCH", OrdinaryOnly: true, Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType, _ foundation14Enum) {
		actual.Types[owner.Key].FlagsMarker = true
	}},
	{Name: "flags_directive_dropped", Category: "FLAGS_MAPPING_MISMATCH", FlagsOnly: true, Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType, _ foundation14Enum) {
		actual.Types[owner.Key].FlagsMarker = false
	}},
	{Name: "wrong_first_value", Category: "ENUM_VALUE_MISMATCH", Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType, pinned foundation14Enum) {
		key := symbolKey{Package: owner.PackagePath, Name: owner.GoName + pinned.Values[0].Name}
		drifted := "9999"
		actual.Members[key].Value = &drifted
	}},
	{Name: "wrong_last_value", Category: "ENUM_VALUE_MISMATCH", Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType, pinned foundation14Enum) {
		last := pinned.Values[len(pinned.Values)-1]
		key := symbolKey{Package: owner.PackagePath, Name: owner.GoName + last.Name}
		drifted := "-7"
		actual.Members[key].Value = &drifted
	}},
	{Name: "iota_renumbering", Category: "ENUM_VALUE_MISMATCH", Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType, pinned foundation14Enum) {
		// An `iota` block renumbers every literal 0, 1, 2, ... in source
		// order, which is exactly why the enum policy forbids iota.
		for index, literal := range pinned.Values {
			key := symbolKey{Package: owner.PackagePath, Name: owner.GoName + literal.Name}
			renumbered := strconv.Itoa(index)
			actual.Members[key].Value = &renumbered
		}
	}},
	{Name: "missing_last_literal", Category: "MISSING_MEMBER", Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType, pinned foundation14Enum) {
		last := pinned.Values[len(pinned.Values)-1]
		delete(actual.Members, symbolKey{Package: owner.PackagePath, Name: owner.GoName + last.Name})
	}},
	{Name: "renamed_literal", Category: "MISSING_MEMBER", Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType, pinned foundation14Enum) {
		key := symbolKey{Package: owner.PackagePath, Name: owner.GoName + pinned.Values[0].Name}
		member := *actual.Members[key]
		delete(actual.Members, key)
		member.Key = symbolKey{Package: owner.PackagePath, Name: owner.GoName + pinned.Values[0].Name + "Renamed"}
		actual.Members[member.Key] = &member
	}},
	{Name: "value_storage_projected", Category: "UNEXPECTED_MEMBER", Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType, _ foundation14Enum) {
		key := symbolKey{Package: owner.PackagePath, Name: owner.GoName + "Value__"}
		value := "0"
		actual.Members[key] = &actualMember{Key: key, Kind: "const", Results: []string{"int32"}, Value: &value}
	}},
	{Name: "invented_constant", Category: "UNEXPECTED_MEMBER", Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType, _ foundation14Enum) {
		// None/Default/All must never be invented for an enum whose pinned
		// contract does not declare them.
		key := symbolKey{Package: owner.PackagePath, Name: owner.GoName + "AllInvented"}
		value := "255"
		actual.Members[key] = &actualMember{Key: key, Kind: "const", Results: []string{owner.GoName}, Value: &value}
	}},
	{Name: "exported_helper", Category: "UNEXPECTED_MEMBER", Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType, _ foundation14Enum) {
		key := symbolKey{Package: owner.PackagePath, Receiver: owner.GoName, Name: "String"}
		actual.Members[key] = &actualMember{Key: key, Kind: "method", Results: []string{"string"}}
	}},
}

// foundation14EnumMutationCase applies one named Foundation-14 defect to one
// batch enum. Mutation ids have the form
// f14_<defect>__<Namespace-qualified identity>.
func foundation14EnumMutationCase(t *testing.T, mutation string) (*expectedSurface, *actualSurface) {
	t.Helper()
	trimmed := strings.TrimPrefix(mutation, "f14_")
	parts := strings.SplitN(trimmed, "__", 2)
	if len(parts) != 2 {
		t.Fatalf("malformed Foundation-14 mutation %q", mutation)
	}
	defectName, identity := parts[0], parts[1]
	expected, actual, owner, pinned := foundation14EnumSurfaces(t, identity)
	for _, defect := range foundation14EnumDefects {
		if defect.Name != defectName {
			continue
		}
		defect.Apply(expected, actual, owner, pinned)
		return expected, actual
	}
	t.Fatalf("unknown Foundation-14 defect %q", defectName)
	return nil, nil
}

// TestBatchEnumDefectsRejectedForEveryType is the exhaustive negative
// fixture for the batch: every applicable structural defect is applied to
// every one of the 25 completed enums, and each one must raise its category.
// A clean baseline is asserted first so a defect cannot pass by accident.
func TestBatchEnumDefectsRejectedForEveryType(t *testing.T) {
	cases := 0
	for _, pinned := range allBatchEnums() {
		pinned := pinned
		t.Run(pinned.Identity, func(t *testing.T) {
			baselineExpected, baselineActual, _, _ := foundation14EnumSurfaces(t, pinned.Identity)
			baseline := verify(baselineExpected, baselineActual, 0, "report", "contract", "mapping")
			if baseline.Summary["TOTAL_DIAGNOSTICS"] != 0 {
				t.Fatalf("unmutated %s baseline is not clean: %v", pinned.Identity, baseline.Diagnostics)
			}
			if len(baseline.Foundation14EnumClosures) != len(foundation14Enums) ||
				len(baseline.Foundation15EnumClosures) != len(foundation15Enums) {
				t.Fatalf("closure counts = %d/%d", len(baseline.Foundation14EnumClosures), len(baseline.Foundation15EnumClosures))
			}
			for _, defect := range foundation14EnumDefects {
				if defect.FlagsOnly && !pinned.Flags {
					continue
				}
				if defect.OrdinaryOnly && pinned.Flags {
					continue
				}
				if defect.Name == "iota_renumbering" && enumAlreadySequentialFromZero(pinned) {
					// Renumbering is invisible for an enum whose pinned
					// values already are 0, 1, 2, ... in source order.
					continue
				}
				defect := defect
				t.Run(defect.Name, func(t *testing.T) {
					expected, actual, owner, entry := foundation14EnumSurfaces(t, pinned.Identity)
					defect.Apply(expected, actual, owner, entry)
					result := verify(expected, actual, 0, "report", "contract", "mapping")
					if result.Summary[defect.Category] == 0 {
						t.Fatalf("defect %q on %s did not raise %s; summary=%v",
							defect.Name, pinned.Identity, defect.Category, result.Summary)
					}
					closures := append(append([]enumClosure(nil), result.Foundation14EnumClosures...), result.Foundation15EnumClosures...)
					for _, closure := range closures {
						if closure.XNA == pinned.Identity && closure.Status != "FAIL" {
							t.Fatalf("defect %q on %s left the closure measurement at %q",
								defect.Name, pinned.Identity, closure.Status)
						}
					}
				})
				cases++
			}
		})
	}
	if cases < 360 {
		t.Fatalf("batch negative fixture count = %d, want at least 360", cases)
	}
}

// enumAlreadySequentialFromZero reports whether a pinned enum's literals are
// already 0, 1, 2, ... in source order.
func enumAlreadySequentialFromZero(pinned foundation14Enum) bool {
	for index, literal := range pinned.Values {
		if normalizeInteger(literal.Value) != strconv.Itoa(index) {
			return false
		}
	}
	return true
}

// valueStructSurfaces builds an isolated correct expected/actual surface pair
// for one Foundation-15 value struct, so a mutation applied afterwards is the
// only defect the verifier can see.
func valueStructSurfaces(t *testing.T, identity string) (*expectedSurface, *actualSurface, *expectedType) {
	t.Helper()
	full, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	fullType := full.typeForXNA(identity)
	if fullType == nil {
		t.Fatalf("%s is not in the pinned contract", identity)
	}
	copiedType := *fullType
	copiedType.Members = append([]symbolKey(nil), fullType.Members...)
	expected := &expectedSurface{
		Types:              map[symbolKey]*expectedType{copiedType.Key: &copiedType},
		Members:            make(map[symbolKey]*expectedMember),
		InterfaceWitnesses: make(map[symbolKey]*expectedInterfaceWitness),
		ReferenceTypes:     1,
		ReferenceMembers:   copiedType.SourceMembers,
		ExpectedGoTypes:    1,
		ExpectedGoMembers:  len(copiedType.Members),
	}
	actual := &actualSurface{
		Types: map[symbolKey]*actualType{
			copiedType.Key: {Key: copiedType.Key, Kind: "struct"},
		},
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: make(map[string]string),
		Packages:    make(map[string]*types.Package),
	}
	for _, memberKey := range copiedType.Members {
		fullMember := full.Members[memberKey]
		copiedMember := *fullMember
		copiedMember.Parameters = append([]string(nil), fullMember.Parameters...)
		copiedMember.Results = append([]string(nil), fullMember.Results...)
		expected.Members[memberKey] = &copiedMember
		actual.Members[memberKey] = &actualMember{
			Key:        memberKey,
			Kind:       copiedMember.GoKind,
			Parameters: append([]string(nil), copiedMember.Parameters...),
			Results:    append([]string(nil), copiedMember.Results...),
		}
	}
	return expected, actual, &copiedType
}

// valueStructDefects are the structural defects every Foundation-15 value
// struct is negatively fixtured against.
var valueStructDefects = []struct {
	Name     string
	Category string
	Apply    func(expected *expectedSurface, actual *actualSurface, owner *expectedType)
}{
	{Name: "missing_type", Category: "MISSING_TYPE", Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType) {
		delete(actual.Types, owner.Key)
	}},
	{Name: "wrong_package", Category: "MISSING_TYPE", Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType) {
		const elsewhere = modulePath + "/Microsoft/Xna/Framework/Graphics"
		moved := *actual.Types[owner.Key]
		delete(actual.Types, owner.Key)
		moved.Key.Package = elsewhere
		actual.Types[moved.Key] = &moved
	}},
	{Name: "projected_as_class", Category: "TYPE_KIND_MISMATCH", Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType) {
		// A reference-class projection would silently change copy semantics
		// for a System.ValueType.
		actual.Types[owner.Key].Kind = "named"
	}},
	{Name: "missing_last_member", Category: "MISSING_MEMBER", Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType) {
		delete(actual.Members, owner.Members[len(owner.Members)-1])
	}},
	{Name: "missing_first_member", Category: "MISSING_MEMBER", Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType) {
		delete(actual.Members, owner.Members[0])
	}},
	{Name: "synthetic_error_result", Category: "RETURN_MAPPING_MISMATCH", Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType) {
		// The central semantic claim of the family: infallible managed value
		// work must never gain a synthetic Go error result.
		key := firstResultBearingMember(owner)
		member := actual.Members[key]
		member.Results = append(append([]string(nil), member.Results...), "error")
	}},
	{Name: "wrong_result_type", Category: "RETURN_MAPPING_MISMATCH", Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType) {
		actual.Members[firstResultBearingMember(owner)].Results = []string{"complex128"}
	}},
	{Name: "wrong_constructor_parameters", Category: "PARAMETER_MAPPING_MISMATCH", Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType) {
		// Mutate whichever constructor the type projects; overloaded
		// constructors carry a By<Type> suffix.
		for _, key := range owner.Members {
			if key.Receiver == "" && strings.HasPrefix(key.Name, "New"+owner.GoName) {
				actual.Members[key].Parameters = []string{"int32"}
				return
			}
		}
	}},
	{Name: "renamed_last_member", Category: "MISSING_MEMBER", Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType) {
		key := owner.Members[len(owner.Members)-1]
		member := *actual.Members[key]
		delete(actual.Members, key)
		member.Key = symbolKey{Package: key.Package, Receiver: key.Receiver, Name: key.Name + "Renamed"}
		actual.Members[member.Key] = &member
	}},
	{Name: "unexpected_mutator", Category: "UNEXPECTED_MEMBER", Apply: func(_ *expectedSurface, actual *actualSurface, owner *expectedType) {
		// These values are immutable in the reference; an invented setter
		// would be new public surface.
		key := symbolKey{Package: owner.PackagePath, Receiver: owner.GoName, Name: "SetInvented"}
		actual.Members[key] = &actualMember{Key: key, Kind: "method", Parameters: []string{"int32"}}
	}},
}

// firstResultBearingMember returns the first member of a value struct that
// produces a result, in the type's declared member order.
func firstResultBearingMember(owner *expectedType) symbolKey {
	for _, key := range owner.Members {
		return key
	}
	return symbolKey{}
}

// TestFoundation15ValueStructDefectsRejectedForEveryType applies every
// structural defect to every value struct in the cluster, asserting a clean
// baseline first so no defect can pass by accident.
func TestFoundation15ValueStructDefectsRejectedForEveryType(t *testing.T) {
	if len(allValueStructs()) != 8 {
		t.Fatalf("value-struct cluster size = %d, want 8", len(allValueStructs()))
	}
	cases := 0
	for _, identity := range allValueStructs() {
		identity := identity
		t.Run(identity, func(t *testing.T) {
			baseExpected, baseActual, _ := valueStructSurfaces(t, identity)
			baseline := verify(baseExpected, baseActual, 0, "report", "contract", "mapping")
			if baseline.Summary["TOTAL_DIAGNOSTICS"] != 0 {
				t.Fatalf("unmutated %s baseline is not clean: %v", identity, baseline.Diagnostics)
			}
			for _, defect := range valueStructDefects {
				defect := defect
				t.Run(defect.Name, func(t *testing.T) {
					expected, actual, owner := valueStructSurfaces(t, identity)
					defect.Apply(expected, actual, owner)
					result := verify(expected, actual, 0, "report", "contract", "mapping")
					if result.Summary[defect.Category] == 0 {
						t.Fatalf("defect %q on %s did not raise %s; summary=%v",
							defect.Name, identity, defect.Category, result.Summary)
					}
					closures := append(append([]valueStructClosure(nil), result.Foundation15ValueStructs...), result.Foundation16ValueStructs...)
					for _, closure := range closures {
						if closure.XNA == identity && closure.Status != "FAIL" {
							t.Fatalf("defect %q on %s left the closure measurement at %q",
								defect.Name, identity, closure.Status)
						}
					}
				})
				cases++
			}
		})
	}
	if cases != len(allValueStructs())*len(valueStructDefects) {
		t.Fatalf("value-struct negative fixture count = %d", cases)
	}
}

// TestFoundation15ValueStructsAreInfallibleManagedValues asserts, against the
// pinned contract, that the whole cluster projects as System.ValueType structs
// with no synthetic Go error result on any member.
func TestFoundation15ValueStructsAreInfallibleManagedValues(t *testing.T) {
	surface, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	identities := 0
	for _, identity := range allValueStructs() {
		owner := surface.typeForXNA(identity)
		if owner == nil || owner.Kind != "struct" || owner.BaseType != "System.ValueType" ||
			len(owner.Members) != owner.SourceMembers {
			t.Fatalf("%s projection = %+v", identity, owner)
		}
		namespace := identity[:strings.LastIndex(identity, ".")]
		if owner.PackagePath != packagePathForNamespace(namespace) {
			t.Fatalf("%s package = %q", identity, owner.PackagePath)
		}
		// TouchLocation is the one cluster member with a declared direct
		// interface; it is the value-typed System.IEquatable`1 of itself,
		// which the established managed-interface policy already covers.
		for _, declared := range owner.Interfaces {
			if !strings.HasPrefix(declared, "System.IEquatable`1[") {
				t.Fatalf("%s declares unexpected interface %q", identity, declared)
			}
		}
		for _, key := range owner.Members {
			member := surface.Members[key]
			for _, result := range member.Results {
				if result == "error" {
					t.Fatalf("%s.%s carries a synthetic error result", identity, key.Name)
				}
			}
			if member.ErrorAdded {
				t.Fatalf("%s.%s was marked as gaining an error result", identity, key.Name)
			}
		}
		identities += len(owner.Members)
	}
	if identities != 91 {
		t.Fatalf("value-struct cluster identities = %d, want 91", identities)
	}
}

// valueStructMutationCase applies one named Foundation-15 value-struct defect.
// Mutation ids have the form f15vs_<defect>__<identity>.
func valueStructMutationCase(t *testing.T, mutation string) (*expectedSurface, *actualSurface) {
	t.Helper()
	parts := strings.SplitN(strings.TrimPrefix(mutation, "f15vs_"), "__", 2)
	if len(parts) != 2 {
		t.Fatalf("malformed value-struct mutation %q", mutation)
	}
	expected, actual, owner := valueStructSurfaces(t, parts[1])
	for _, defect := range valueStructDefects {
		if defect.Name != parts[0] {
			continue
		}
		defect.Apply(expected, actual, owner)
		return expected, actual
	}
	t.Fatalf("unknown value-struct defect %q", parts[0])
	return nil, nil
}

// managedClassSurfaces builds an isolated correct expected/actual surface pair
// for one pure-managed CLR class, so a defect applied afterwards is the only
// thing the verifier can see. The actual side is generated from the expected
// side, which makes an unmutated baseline clean by construction.
func managedClassSurfaces(t *testing.T, identity string) (*expectedSurface, *actualSurface, *expectedType) {
	t.Helper()
	full, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	return isolateManagedClass(t, full, identity)
}

func isolateManagedClass(t *testing.T, full *expectedSurface, identity string) (*expectedSurface, *actualSurface, *expectedType) {
	t.Helper()
	fullType := full.typeForXNA(identity)
	if fullType == nil {
		t.Fatalf("%s is not in the pinned contract", identity)
	}
	copiedType := *fullType
	copiedType.Members = append([]symbolKey(nil), fullType.Members...)
	expected := &expectedSurface{
		Types:              map[symbolKey]*expectedType{copiedType.Key: &copiedType},
		Members:            make(map[symbolKey]*expectedMember),
		InterfaceWitnesses: make(map[symbolKey]*expectedInterfaceWitness),
		ReferenceTypes:     1,
		ReferenceMembers:   copiedType.SourceMembers,
		ExpectedGoTypes:    1,
		ExpectedGoMembers:  len(copiedType.Members),
	}
	actual := &actualSurface{
		Types: map[symbolKey]*actualType{
			copiedType.Key: {Key: copiedType.Key, Kind: "struct"},
		},
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: make(map[string]string),
		Packages:    make(map[string]*types.Package),
	}
	for _, memberKey := range copiedType.Members {
		fullMember := full.Members[memberKey]
		copiedMember := *fullMember
		copiedMember.Parameters = append([]string(nil), fullMember.Parameters...)
		copiedMember.Results = append([]string(nil), fullMember.Results...)
		expected.Members[memberKey] = &copiedMember
		actual.Members[memberKey] = &actualMember{
			Key:        memberKey,
			Kind:       copiedMember.GoKind,
			Parameters: append([]string(nil), copiedMember.Parameters...),
			Results:    append([]string(nil), copiedMember.Results...),
		}
	}
	addIteratorAdapter(actual, &copiedType)
	return expected, actual, &copiedType
}

// addIteratorAdapter gives an isolated surface the module-wide Iterator<T>
// language adapter when the owner declares a BCL collection interface. The
// adapter is a real part of the module, not part of the type under test, so
// omitting it would make a correct baseline look broken.
func addIteratorAdapter(actual *actualSurface, owner *expectedType) {
	if !containsInterfacePrefix(owner.AllInterfaces, "System.Collections.Generic.ICollection`1[") {
		return
	}
	const frameworkPackage = modulePath + "/Microsoft/Xna/Framework"
	iteratorKey := symbolKey{Package: frameworkPackage, Name: "Iterator"}
	actual.Types[iteratorKey] = &actualType{Key: iteratorKey, Kind: "interface", TypeParameters: []string{"T"}}
	nextKey := symbolKey{Package: frameworkPackage, Receiver: "Iterator", Name: "Next"}
	actual.Members[nextKey] = &actualMember{Key: nextKey, Kind: "method", Results: []string{"T", "bool", "error"}}
}

// accessorKey returns the projected member key of one accessor of one property
// on a pure-managed class, so a defect can target exactly one accessor.
func accessorKey(t *testing.T, expected *expectedSurface, owner *expectedType, property, accessor string) symbolKey {
	t.Helper()
	for _, key := range owner.Members {
		member := expected.Members[key]
		if member.Accessor != accessor {
			continue
		}
		if !strings.HasPrefix(member.XNA, owner.XNA+"::"+property+"(") {
			continue
		}
		return key
	}
	t.Fatalf("%s has no projected %s accessor for %s", owner.XNA, accessor, property)
	return symbolKey{}
}

// anyAccessorKey returns the projected key of the first accessor of the given
// kind, in declared order. Every class in the cluster has at least one of each.
func anyAccessorKey(t *testing.T, expected *expectedSurface, owner *expectedType, accessor string, fallible bool) symbolKey {
	t.Helper()
	for _, key := range owner.Members {
		if member := expected.Members[key]; member.Accessor == accessor && member.ErrorAdded == fallible {
			return key
		}
	}
	t.Fatalf("%s has no %s accessor with fallible=%t", owner.XNA, accessor, fallible)
	return symbolKey{}
}

// managedClassDefects are the target-side structural and fallibility defects
// every pure-managed CLR class is negatively fixtured against. The fallibility
// entries cover all four accessor cases the per-operation rule creates:
// an invented error on a getter or on an infallible setter, and a dropped
// error on the one setter that genuinely has one.
var managedClassDefects = []struct {
	Name     string
	Category string
	Requires func(expected *expectedSurface, owner *expectedType) bool
	Apply    func(t *testing.T, expected *expectedSurface, actual *actualSurface, owner *expectedType)
}{
	{Name: "missing_type", Category: "MISSING_TYPE", Apply: func(_ *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		delete(actual.Types, owner.Key)
	}},
	{Name: "wrong_package", Category: "MISSING_TYPE", Apply: func(_ *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		moved := *actual.Types[owner.Key]
		delete(actual.Types, owner.Key)
		moved.Key.Package = somewhereElse(owner.PackagePath)
		actual.Types[moved.Key] = &moved
	}},
	{Name: "projected_as_named_type", Category: "TYPE_KIND_MISMATCH", Apply: func(_ *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		actual.Types[owner.Key].Kind = "named"
	}},
	{Name: "wrong_constructor_semantics", Category: "RETURN_MAPPING_MISMATCH", Requires: typeHasConstructor, Apply: func(t *testing.T, expected *expectedSurface, actual *actualSurface, owner *expectedType) {
		// A CLR class keeps reference semantics and a CLR struct keeps value
		// semantics. Swapping either one silently changes whether two
		// variables share mutations, so the defect is the opposite of
		// whichever projection is correct for this owner.
		key := constructorKey(t, expected, owner)
		swapped := owner.GoName
		if owner.Kind == "struct" {
			swapped = "*" + owner.GoName
		}
		results := append([]string(nil), actual.Members[key].Results...)
		results[0] = swapped
		actual.Members[key].Results = results
	}},
	{Name: "missing_first_member", Category: "MISSING_MEMBER", Apply: func(_ *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		delete(actual.Members, owner.Members[0])
	}},
	{Name: "missing_last_member", Category: "MISSING_MEMBER", Apply: func(_ *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		delete(actual.Members, owner.Members[len(owner.Members)-1])
	}},
	{Name: "renamed_last_member", Category: "MISSING_MEMBER", Apply: func(_ *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		key := owner.Members[len(owner.Members)-1]
		member := *actual.Members[key]
		delete(actual.Members, key)
		member.Key = symbolKey{Package: key.Package, Receiver: key.Receiver, Name: key.Name + "Renamed"}
		actual.Members[member.Key] = &member
	}},
	{Name: "unexpected_member", Category: "UNEXPECTED_MEMBER", Apply: func(_ *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		key := symbolKey{Package: owner.PackagePath, Receiver: owner.GoName, Name: "SetInvented"}
		actual.Members[key] = &actualMember{Key: key, Kind: "method", Parameters: []string{"int32"}}
	}},
	{Name: "wrong_setter_parameter", Category: "PARAMETER_MAPPING_MISMATCH", Requires: typeHasSetter, Apply: func(t *testing.T, expected *expectedSurface, actual *actualSurface, owner *expectedType) {
		// complex128 is chosen because no XNA member maps to it, so this can
		// never coincide with the correct parameter type of any accessor.
		actual.Members[anySetterKey(t, expected, owner)].Parameters = []string{"complex128"}
	}},
	{Name: "artificial_getter_error", Category: "ERROR_MAPPING_MISMATCH", Requires: typeHasInfallibleGetter, Apply: func(t *testing.T, expected *expectedSurface, actual *actualSurface, owner *expectedType) {
		// Every getter in the cluster is one ldfld. None may gain an error.
		key := anyAccessorKey(t, expected, owner, "get", false)
		member := actual.Members[key]
		member.Results = append(append([]string(nil), member.Results...), "error")
	}},
	{Name: "artificial_setter_error", Category: "ERROR_MAPPING_MISMATCH", Requires: typeHasInfallibleSetter, Apply: func(t *testing.T, expected *expectedSurface, actual *actualSurface, owner *expectedType) {
		// A setter that validates nothing must not gain an error just
		// because a sibling setter on the same type validates.
		actual.Members[anyAccessorKey(t, expected, owner, "set", false)].Results = []string{"error"}
	}},
	{Name: "artificial_constructor_error", Category: "ERROR_MAPPING_MISMATCH", Requires: typeHasInfallibleConstructor, Apply: func(t *testing.T, expected *expectedSurface, actual *actualSurface, owner *expectedType) {
		key := constructorKey(t, expected, owner)
		member := actual.Members[key]
		member.Results = append(append([]string(nil), member.Results...), "error")
	}},
	{Name: "native_facade_projection", Category: "ERROR_MAPPING_MISMATCH", Requires: typeHasInfallibleMember, Apply: func(_ *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		// The whole-type defect: a pure-managed class written as if it were a
		// native-backed facade, so every projected operation carries an error.
		for _, key := range owner.Members {
			member := actual.Members[key]
			if len(member.Results) > 0 && member.Results[len(member.Results)-1] == "error" {
				continue
			}
			member.Results = append(append([]string(nil), member.Results...), "error")
		}
	}},
	{Name: "dropped_error", Category: "ERROR_MAPPING_MISMATCH", Requires: typeHasFallibleMember, Apply: func(t *testing.T, expected *expectedSurface, actual *actualSurface, owner *expectedType) {
		// The mirror of the artificial-error defects: an operation the
		// reference proves can throw must keep somewhere to report it.
		for _, key := range owner.Members {
			if !expected.Members[key].ErrorAdded {
				continue
			}
			member := actual.Members[key]
			member.Results = member.Results[:len(member.Results)-1]
			return
		}
		t.Fatalf("%s projects no fallible operation", owner.XNA)
	}},
}

func typeHasConstructor(expected *expectedSurface, owner *expectedType) bool {
	for _, key := range owner.Members {
		if expected.Members[key].SourceKind == "constructor" {
			return true
		}
	}
	return false
}

func typeHasInfallibleConstructor(expected *expectedSurface, owner *expectedType) bool {
	for _, key := range owner.Members {
		if member := expected.Members[key]; member.SourceKind == "constructor" && !member.ErrorAdded {
			return true
		}
	}
	return false
}

func typeHasSetter(expected *expectedSurface, owner *expectedType) bool {
	for _, key := range owner.Members {
		if expected.Members[key].Accessor == "set" {
			return true
		}
	}
	return false
}

func typeHasInfallibleSetter(expected *expectedSurface, owner *expectedType) bool {
	for _, key := range owner.Members {
		if member := expected.Members[key]; member.Accessor == "set" && !member.ErrorAdded {
			return true
		}
	}
	return false
}

func typeHasInfallibleGetter(expected *expectedSurface, owner *expectedType) bool {
	for _, key := range owner.Members {
		if member := expected.Members[key]; member.Accessor == "get" && !member.ErrorAdded {
			return true
		}
	}
	return false
}

func typeHasInfallibleMember(expected *expectedSurface, owner *expectedType) bool {
	for _, key := range owner.Members {
		if !expected.Members[key].ErrorAdded {
			return true
		}
	}
	return false
}

func typeHasFallibleMember(expected *expectedSurface, owner *expectedType) bool {
	for _, key := range owner.Members {
		if expected.Members[key].ErrorAdded {
			return true
		}
	}
	return false
}

// anySetterKey returns the projected key of the first property setter in
// declared order, whatever its fallibility.
func anySetterKey(t *testing.T, expected *expectedSurface, owner *expectedType) symbolKey {
	t.Helper()
	for _, key := range owner.Members {
		if expected.Members[key].Accessor == "set" {
			return key
		}
	}
	t.Fatalf("%s projects no property setter", owner.XNA)
	return symbolKey{}
}

// constructorKey returns the projected constructor of a pure-managed class.
func constructorKey(t *testing.T, expected *expectedSurface, owner *expectedType) symbolKey {
	t.Helper()
	for _, key := range owner.Members {
		if expected.Members[key].SourceKind == "constructor" {
			return key
		}
	}
	t.Fatalf("%s projects no constructor", owner.XNA)
	return symbolKey{}
}

// TestFoundation17ManagedClassDefectsRejectedForEveryType applies every
// target-side defect to every pure-managed CLR class, asserting a clean
// baseline first so no defect can pass by accident.
func TestFoundation17ManagedClassDefectsRejectedForEveryType(t *testing.T) {
	if len(allManagedClasses()) != 9 {
		t.Fatalf("pure-managed type cluster size = %d, want 9", len(allManagedClasses()))
	}
	cases, skipped := 0, 0
	for _, identity := range allManagedClasses() {
		identity := identity
		t.Run(identity, func(t *testing.T) {
			baseExpected, baseActual, baseOwner := managedClassSurfaces(t, identity)
			baseline := verify(baseExpected, baseActual, 0, "report", "contract", "mapping")
			if baseline.Summary["TOTAL_DIAGNOSTICS"] != 0 {
				t.Fatalf("unmutated %s baseline is not clean: %v", identity, baseline.Diagnostics)
			}
			for _, closure := range allManagedClassClosures(baseline) {
				if closure.XNA == identity && closure.Status != "PASS" {
					t.Fatalf("unmutated %s closure = %q", identity, closure.Status)
				}
			}
			for _, defect := range managedClassDefects {
				defect := defect
				if defect.Requires != nil && !defect.Requires(baseExpected, baseOwner) {
					skipped++
					continue
				}
				t.Run(defect.Name, func(t *testing.T) {
					expected, actual, owner := managedClassSurfaces(t, identity)
					defect.Apply(t, expected, actual, owner)
					result := verify(expected, actual, 0, "report", "contract", "mapping")
					if result.Summary[defect.Category] == 0 {
						t.Fatalf("defect %q on %s did not raise %s; summary=%v",
							defect.Name, identity, defect.Category, result.Summary)
					}
					for _, closure := range allManagedClassClosures(result) {
						if closure.XNA == identity && closure.Status != "FAIL" {
							t.Fatalf("defect %q on %s left the closure measurement at %q",
								defect.Name, identity, closure.Status)
						}
					}
				})
				cases++
			}
		})
	}
	// The cluster deliberately spans an all-infallible class, a class with a
	// single fallible setter, a large all-infallible descriptor, a fallible
	// value struct, and a bare cursor, so not every defect is expressible on
	// every member. Each skip is counted rather than silently dropped:
	//
	//   AudioListener               1  dropped_error: nothing can fail
	//   AudioEmitter                0  its one fallible setter expresses all 14
	//   PresentationParameters      1  dropped_error: nothing can fail
	//   TouchCollection             2  artificial_setter_error and
	//                                  artificial_constructor_error: its only
	//                                  setter and only constructor are both
	//                                  already fallible
	//   TouchCollection+Enumerator  5  no constructor, no setter, and no
	//                                  infallible getter
	//   GameServiceContainer        3  no property accessors at all
	//   Foundation 23, the three System.EventArgs carriers:
	//   GameComponentCollectionEventArgs
	//                               4  every accessor is an infallible getter:
	//                                  no setter, and its one constructor and
	//                                  its one getter cannot fail
	//   ResourceCreatedEventArgs    5  one infallible getter, no setter and no
	//                                  public constructor at all
	//   ResourceDestroyedEventArgs  4  two infallible getters, no setter and no
	//                                  public constructor
	if cases+skipped != len(allManagedClasses())*len(managedClassDefects) {
		t.Fatalf("pure-managed type fixture accounting = %d applied + %d skipped", cases, skipped)
	}
	if cases != 101 || skipped != 25 {
		t.Fatalf("pure-managed type negative fixtures = %d applied, %d skipped", cases, skipped)
	}
}

// withClassification runs fn with the pure-managed classification tables
// temporarily mutated, then restores them exactly. It lets the negative
// fixtures below attack the classification rule itself rather than only the Go
// target, which is where the two directions of the class rule live.
func withClassification(t *testing.T, mutate func(), fn func()) {
	t.Helper()
	savedTypes := make(map[string]bool, len(pureManagedTypes))
	for key, value := range pureManagedTypes {
		savedTypes[key] = value
	}
	savedInterfaces := make(map[string]bool, len(classifiedInterfaces))
	for key, value := range classifiedInterfaces {
		savedInterfaces[key] = value
	}
	savedFallible := make(map[string]map[string]bool, len(managedFallibleMembers))
	for owner, keys := range managedFallibleMembers {
		copied := make(map[string]bool, len(keys))
		for key, value := range keys {
			copied[key] = value
		}
		savedFallible[owner] = copied
	}
	// managedStoredMembers is the mirror-image table: it LOWERS fallibility on
	// a native-backed facade where managedFallibleMembers RAISES it on a
	// pure-managed owner. A defect that mutates one must be restorable exactly
	// like a defect that mutates the other.
	savedStored := make(map[string]map[string]bool, len(managedStoredMembers))
	for owner, keys := range managedStoredMembers {
		copied := make(map[string]bool, len(keys))
		for key, value := range keys {
			copied[key] = value
		}
		savedStored[owner] = copied
	}
	defer func() {
		pureManagedTypes = savedTypes
		classifiedInterfaces = savedInterfaces
		managedFallibleMembers = savedFallible
		managedStoredMembers = savedStored
	}()
	mutate()
	fn()
}

// classificationDefectResult verifies the correct Go projection of identity
// against an expected surface rebuilt under a mutated classification, which is
// exactly what a wrong classification decision would produce.
func classificationDefectResult(t *testing.T, identity string, mutate func()) report {
	t.Helper()
	_, actual, _ := managedClassSurfaces(t, identity)
	var result report
	withClassification(t, mutate, func() {
		full, err := buildExpected(loadPinnedContract(t))
		if err != nil {
			t.Fatal(err)
		}
		mutated, _, _ := isolateManagedClass(t, full, identity)
		result = verify(mutated, actual, 0, "report", "contract", "mapping")
	})
	return result
}

const (
	audioListenerIdentity = "Microsoft.Xna.Framework.Audio.AudioListener"
	audioEmitterIdentity  = "Microsoft.Xna.Framework.Audio.AudioEmitter"
	texture2DIdentity     = "Microsoft.Xna.Framework.Graphics.Texture2D"
	gameIdentity          = "Microsoft.Xna.Framework.Game"
)

// TestProjectedFallibilityFlagAlwaysMatchesTheProjectedResults is a structural
// invariant over the WHOLE expected surface rather than a targeted mutation.
//
// Every expected member carries both a boolean saying whether the projection
// added a Go error and a result list that either ends in "error" or does not.
// Those two must never disagree: the flag is what the verifier compares against
// the real Go signature, so a member whose flag says "fallible" while its
// results say otherwise reports a mismatch against correct Go code, and a
// member with the opposite skew lets a wrong signature through.
//
// The invariant is not hypothetical. The property branch rebuilds an accessor's
// results from scratch but used to inherit the flag from the whole-member
// classification, which made the accessor-level decision one-directional: it
// could raise fallibility on a pure-managed owner but never lower it on a
// native-backed one. Game::Components is the first get-only stored property on
// a fallible-by-default owner, and it is what exposed the skew.
func TestProjectedFallibilityFlagAlwaysMatchesTheProjectedResults(t *testing.T) {
	expected, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for key, member := range expected.Members {
		endsWithError := len(member.Results) > 0 && member.Results[len(member.Results)-1] == "error"
		if member.ErrorAdded != endsWithError {
			t.Fatalf("%s (%s): ErrorAdded=%t but results %v", key.String(), member.XNA,
				member.ErrorAdded, member.Results)
		}
		checked++
	}
	if checked < 3000 {
		t.Fatalf("only %d expected members checked; the invariant must cover the whole surface", checked)
	}
}

// TestClassClassificationDefectsAreRejected attacks the two general rules this
// milestone introduced -- pure-managed class classification and per-operation
// fallibility -- in both directions, using the real classification tables.
func TestClassClassificationDefectsAreRejected(t *testing.T) {
	cases := []struct {
		name     string
		identity string
		// wantMessage pins the exact accessor and direction the diagnostic
		// must name, so a defect cannot pass by raising the right category for
		// the wrong reason.
		wantMessage string
	}{
		// CLR `class` alone must never make a type fallible. Dropping the
		// classification is exactly the pre-Foundation-17 behavior.
		{"managed_class_demoted_to_native_facade", audioListenerIdentity,
			"property getter expected fallible, projected infallible"},
		{"managed_class_with_validating_setter_demoted", audioEmitterIdentity,
			"property getter expected fallible, projected infallible"},
		// The opposite direction: a genuinely native-backed class must not be
		// admitted as pure managed, which would strip the error result from
		// every operation that crosses the native boundary.
		{"native_backed_class_admitted_as_pure_managed", texture2DIdentity,
			"expected infallible, projected fallible"},
		// Accessor-level fallibility must not silently widen to the whole
		// property: DopplerScale's getter is one ldfld and cannot fail.
		{"accessor_fallibility_widened_to_whole_property", audioEmitterIdentity,
			"property getter expected fallible, projected infallible"},
		// And must not silently narrow to the wrong accessor.
		{"accessor_fallibility_moved_to_the_getter", audioEmitterIdentity,
			"property setter expected infallible, projected fallible"},
		// Dropping the accessor-level entry loses the one genuine throw.
		{"accessor_fallibility_dropped", audioEmitterIdentity,
			"property setter expected infallible, projected fallible"},
		// Foundation 30, the mirror image on a native-backed facade. Game keeps
		// the runtime default because the native CNA host owns its loop, so
		// losing the stored-member entry puts a synthetic error back on two
		// getters that are one `ldfld` each.
		{"stored_getter_on_native_facade_demoted", gameIdentity,
			"property getter expected fallible, projected infallible"},
		// The entry is per projected operation. Classifying only Components
		// leaves Services fallible, which is the same defect as classifying a
		// whole type instead of reading each member's IL.
		{"stored_getter_classified_for_one_member_only", gameIdentity,
			"property getter expected fallible, projected infallible"},
	}
	if len(cases) != len(classificationDefects) {
		t.Fatalf("classification defect coverage = %d of %d", len(cases), len(classificationDefects))
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			mutate, ok := classificationDefects[testCase.name]
			if !ok {
				t.Fatalf("classification defect %q is not in the shared table", testCase.name)
			}
			result := classificationDefectResult(t, testCase.identity, mutate)
			if result.Summary["ERROR_MAPPING_MISMATCH"] == 0 {
				t.Fatalf("classification defect %q did not raise ERROR_MAPPING_MISMATCH; summary=%v",
					testCase.name, result.Summary)
			}
			found := false
			for _, item := range result.Diagnostics {
				if item.Category == "ERROR_MAPPING_MISMATCH" && strings.Contains(item.Message, testCase.wantMessage) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("classification defect %q raised ERROR_MAPPING_MISMATCH but no diagnostic said %q; diagnostics=%v",
					testCase.name, testCase.wantMessage, result.Diagnostics)
			}
		})
	}
}

// TestFoundation17ManagedClassMappedContracts pins the exact projected contract
// of both audio descriptors against the reference contract, including the one
// asymmetry that motivated per-operation fallibility.
func TestFoundation17ManagedClassMappedContracts(t *testing.T) {
	surface, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	const audioPackage = modulePath + "/Microsoft/Xna/Framework/Audio"
	positional := []string{"Position", "Velocity", "Forward", "Up"}

	for _, owner := range []struct {
		identity   string
		goName     string
		source     int
		identities int
	}{
		{audioListenerIdentity, "AudioListener", 5, 9},
		{audioEmitterIdentity, "AudioEmitter", 6, 11},
	} {
		owner := owner
		t.Run(owner.identity, func(t *testing.T) {
			et := surface.typeForXNA(owner.identity)
			if et == nil {
				t.Fatalf("%s is absent from the pinned contract", owner.identity)
			}
			if et.Kind != "class" || et.BaseType != "System.Object" {
				t.Fatalf("%s = kind %q base %q", owner.identity, et.Kind, et.BaseType)
			}
			if len(et.Interfaces) != 0 {
				t.Fatalf("%s declares interfaces %v", owner.identity, et.Interfaces)
			}
			if et.PackagePath != audioPackage || et.GoName != owner.goName {
				t.Fatalf("%s = %s.%s", owner.identity, et.PackagePath, et.GoName)
			}
			if et.SourceMembers != owner.source || len(et.Members) != owner.identities {
				t.Fatalf("%s = %d source members, %d Go identities",
					owner.identity, et.SourceMembers, len(et.Members))
			}
			if !pureManagedTypes[owner.identity] {
				t.Fatalf("%s is not classified pure managed", owner.identity)
			}

			// Reference semantics: the constructor yields a pointer.
			constructor := surface.Members[symbolKey{Package: audioPackage, Name: "New" + owner.goName}]
			if constructor == nil || len(constructor.Parameters) != 0 ||
				!equalStrings(constructor.Results, []string{"*" + owner.goName}) {
				t.Fatalf("%s constructor = %+v", owner.identity, constructor)
			}

			// The four positional properties are infallible on both accessors
			// in both types.
			for _, name := range positional {
				getter := surface.Members[symbolKey{Package: audioPackage, Receiver: owner.goName, Name: name}]
				if getter == nil || getter.Accessor != "get" || getter.ErrorAdded ||
					len(getter.Parameters) != 0 || !equalStrings(getter.Results, []string{"framework.Vector3"}) {
					t.Fatalf("%s.%s getter = %+v", owner.identity, name, getter)
				}
				setter := surface.Members[symbolKey{Package: audioPackage, Receiver: owner.goName, Name: "Set" + name}]
				if setter == nil || setter.Accessor != "set" || setter.ErrorAdded ||
					!equalStrings(setter.Parameters, []string{"framework.Vector3"}) || len(setter.Results) != 0 {
					t.Fatalf("%s.Set%s = %+v", owner.identity, name, setter)
				}
			}
		})
	}

	// DopplerScale is the whole point of accessor-level fallibility: one
	// property whose getter cannot fail and whose setter can.
	getter := surface.Members[symbolKey{Package: audioPackage, Receiver: "AudioEmitter", Name: "DopplerScale"}]
	if getter == nil || getter.ErrorAdded || !equalStrings(getter.Results, []string{"float32"}) {
		t.Fatalf("AudioEmitter.DopplerScale getter = %+v", getter)
	}
	setter := surface.Members[symbolKey{Package: audioPackage, Receiver: "AudioEmitter", Name: "SetDopplerScale"}]
	if setter == nil || !setter.ErrorAdded ||
		!equalStrings(setter.Parameters, []string{"float32"}) || !equalStrings(setter.Results, []string{"error"}) {
		t.Fatalf("AudioEmitter.SetDopplerScale = %+v", setter)
	}
	if getterKey := accessorKey(t, surface, surface.typeForXNA(audioEmitterIdentity), "DopplerScale", "get"); getterKey.Name != "DopplerScale" {
		t.Fatalf("DopplerScale getter key = %+v", getterKey)
	}
	if setterKey := accessorKey(t, surface, surface.typeForXNA(audioEmitterIdentity), "DopplerScale", "set"); setterKey.Name != "SetDopplerScale" {
		t.Fatalf("DopplerScale setter key = %+v", setterKey)
	}
}

// TestFallibilityKeysAreAccessorSpecific pins the general key scheme itself,
// independently of any one type, so a future validating setter is expressible
// without touching verifier logic.
func TestFallibilityKeysAreAccessorSpecific(t *testing.T) {
	property := contractMember{Kind: "property", Name: "Sample"}
	if got := fallibilityKeys(property, "get"); !equalStrings(got, []string{"property-get|Sample", "property|Sample"}) {
		t.Fatalf("getter keys = %v", got)
	}
	if got := fallibilityKeys(property, "set"); !equalStrings(got, []string{"property-set|Sample", "property|Sample"}) {
		t.Fatalf("setter keys = %v", got)
	}
	if got := fallibilityKeys(property, ""); !equalStrings(got, []string{"property|Sample"}) {
		t.Fatalf("whole-property keys = %v", got)
	}
	for _, kind := range []string{"constructor", "method", "field", "event"} {
		member := contractMember{Kind: kind, Name: "Sample"}
		if got := fallibilityKeys(member, ""); !equalStrings(got, []string{kind + "|Sample"}) {
			t.Fatalf("%s keys = %v", kind, got)
		}
	}

	// A pure-managed owner with an accessor-level entry marks exactly one
	// accessor; the same entry spelled as a whole property marks both.
	owner := contractType{Name: "Synthetic.Owner", Kind: "class"}
	withClassification(t, func() {
		pureManagedTypes[owner.Name] = true
		managedFallibleMembers[owner.Name] = map[string]bool{"property-set|Sample": true}
	}, func() {
		if isFallible(owner, property, "get") || !isFallible(owner, property, "set") {
			t.Fatal("accessor-level entry did not isolate the setter")
		}
	})
	withClassification(t, func() {
		pureManagedTypes[owner.Name] = true
		managedFallibleMembers[owner.Name] = map[string]bool{"property|Sample": true}
	}, func() {
		if !isFallible(owner, property, "get") || !isFallible(owner, property, "set") {
			t.Fatal("whole-property entry did not mark both accessors")
		}
	})
	// Classification alone, with no entry, never adds an error.
	withClassification(t, func() {
		pureManagedTypes[owner.Name] = true
		delete(managedFallibleMembers, owner.Name)
	}, func() {
		if isFallible(owner, property, "get") || isFallible(owner, property, "set") {
			t.Fatal("pure-managed classification invented an error result")
		}
	})
	// Without the classification, CLR `class` alone makes both accessors
	// fallible; that is the native-facade default this milestone narrowed.
	withClassification(t, func() {
		delete(pureManagedTypes, owner.Name)
		delete(managedFallibleMembers, owner.Name)
	}, func() {
		if !isFallible(owner, property, "get") || !isFallible(owner, property, "set") {
			t.Fatal("native facade default did not make the class fallible")
		}
	})
}

// managedClassMutationCase applies one named Foundation-17 target-side
// managed-class defect. Mutation ids have the form f17mc_<defect>__<identity>.
func managedClassMutationCase(t *testing.T, mutation string) (*expectedSurface, *actualSurface) {
	t.Helper()
	parts := strings.SplitN(strings.TrimPrefix(mutation, "f17mc_"), "__", 2)
	if len(parts) != 2 {
		t.Fatalf("malformed managed-class mutation %q", mutation)
	}
	expected, actual, owner := managedClassSurfaces(t, parts[1])
	for _, defect := range managedClassDefects {
		if defect.Name != parts[0] {
			continue
		}
		if defect.Requires != nil && !defect.Requires(expected, owner) {
			t.Fatalf("managed-type defect %q is not expressible on %s", parts[0], parts[1])
		}
		defect.Apply(t, expected, actual, owner)
		return expected, actual
	}
	t.Fatalf("unknown managed-class defect %q", parts[0])
	return nil, nil
}

// classificationMutationCase applies one named Foundation-17 classification
// defect. These mutate the classification tables rather than the Go target, so
// they return a finished report rather than a surface pair. Mutation ids have
// the form f17cls_<defect>__<identity>.
func classificationMutationCase(t *testing.T, mutation string) report {
	t.Helper()
	parts := strings.SplitN(strings.TrimPrefix(mutation, "f17cls_"), "__", 2)
	if len(parts) != 2 {
		t.Fatalf("malformed classification mutation %q", mutation)
	}
	identity := parts[1]
	mutate, ok := classificationDefects[parts[0]]
	if !ok {
		t.Fatalf("unknown classification defect %q", parts[0])
	}
	return classificationDefectResult(t, identity, mutate)
}

// classificationDefects is the shared table behind both the named
// classification test and the mutation inventory, so the two cannot drift.
var classificationDefects = map[string]func(){
	"managed_class_demoted_to_native_facade": func() {
		delete(pureManagedTypes, audioListenerIdentity)
	},
	"managed_class_with_validating_setter_demoted": func() {
		delete(pureManagedTypes, audioEmitterIdentity)
	},
	"native_backed_class_admitted_as_pure_managed": func() {
		pureManagedTypes[texture2DIdentity] = true
	},
	"accessor_fallibility_widened_to_whole_property": func() {
		managedFallibleMembers[audioEmitterIdentity] = map[string]bool{"property|DopplerScale": true}
	},
	"accessor_fallibility_moved_to_the_getter": func() {
		managedFallibleMembers[audioEmitterIdentity] = map[string]bool{"property-get|DopplerScale": true}
	},
	"accessor_fallibility_dropped": func() {
		delete(managedFallibleMembers, audioEmitterIdentity)
	},
	// Foundation 30. The mirror image on a native-backed facade: Game is
	// fallible by default because the native CNA runtime owns its host, but
	// get_Components and get_Services are one `ldfld` each. Dropping the
	// managedStoredMembers entry re-adds the synthetic error their IL cannot
	// produce.
	"stored_getter_on_native_facade_demoted": func() {
		delete(managedStoredMembers, gameIdentity)
	},
	// The entry is per projected operation, never per type: keeping only
	// Components leaves Services fallible, which is exactly the defect of
	// classifying a whole type instead of reading each member's IL.
	"stored_getter_classified_for_one_member_only": func() {
		managedStoredMembers[gameIdentity] = map[string]bool{"property-get|Components": true}
	},
}

// managedInterfaceSurfaces builds an isolated correct expected/actual surface
// pair for one projected CLR interface.
func managedInterfaceSurfaces(t *testing.T, identity string) (*expectedSurface, *actualSurface, *expectedType) {
	t.Helper()
	full, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	return isolateManagedInterface(t, full, identity)
}

func isolateManagedInterface(t *testing.T, full *expectedSurface, identity string) (*expectedSurface, *actualSurface, *expectedType) {
	t.Helper()
	fullType := full.typeForXNA(identity)
	if fullType == nil {
		t.Fatalf("%s is not in the pinned contract", identity)
	}
	copiedType := *fullType
	copiedType.Members = append([]symbolKey(nil), fullType.Members...)
	expected := &expectedSurface{
		Types:              map[symbolKey]*expectedType{copiedType.Key: &copiedType},
		Members:            make(map[symbolKey]*expectedMember),
		InterfaceWitnesses: make(map[symbolKey]*expectedInterfaceWitness),
		ReferenceTypes:     1,
		ReferenceMembers:   copiedType.SourceMembers,
		ExpectedGoTypes:    1,
		ExpectedGoMembers:  len(copiedType.Members),
	}
	actual := &actualSurface{
		Types: map[symbolKey]*actualType{
			copiedType.Key: {Key: copiedType.Key, Kind: "interface"},
		},
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: make(map[string]string),
		Packages:    make(map[string]*types.Package),
	}
	for _, memberKey := range copiedType.Members {
		fullMember := full.Members[memberKey]
		copiedMember := *fullMember
		copiedMember.Parameters = append([]string(nil), fullMember.Parameters...)
		copiedMember.Results = append([]string(nil), fullMember.Results...)
		expected.Members[memberKey] = &copiedMember
		actual.Members[memberKey] = &actualMember{
			Key:        memberKey,
			Kind:       copiedMember.GoKind,
			Parameters: append([]string(nil), copiedMember.Parameters...),
			Results:    append([]string(nil), copiedMember.Results...),
		}
	}
	return expected, actual, &copiedType
}

// firstMemberWithFallibility returns the first projected operation whose
// expected fallibility matches want, in declared order.
func firstMemberWithFallibility(t *testing.T, expected *expectedSurface, owner *expectedType, want bool) symbolKey {
	t.Helper()
	for _, key := range owner.Members {
		if expected.Members[key].ErrorAdded == want {
			return key
		}
	}
	t.Fatalf("%s has no operation with fallible=%t", owner.XNA, want)
	return symbolKey{}
}

// managedInterfaceDefects are the target-side defects every projected
// interface is negatively fixtured against. The fallibility entries cover both
// directions of the per-operation rule on an interface owner.
//
// Requires, when set, reports whether a contract has the shape a defect needs.
// The cluster deliberately spans a uniformly infallible contract, a uniformly
// fallible one, and a mixed one, so not every defect is expressible on every
// member; a skipped case is counted rather than silently dropped.
var managedInterfaceDefects = []struct {
	Name     string
	Category string
	Requires func(expected *expectedSurface, owner *expectedType) bool
	Apply    func(t *testing.T, expected *expectedSurface, actual *actualSurface, owner *expectedType)
}{
	{Name: "missing_type", Category: "MISSING_TYPE", Apply: func(_ *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		delete(actual.Types, owner.Key)
	}},
	{Name: "wrong_package", Category: "MISSING_TYPE", Apply: func(_ *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		moved := *actual.Types[owner.Key]
		delete(actual.Types, owner.Key)
		moved.Key.Package = somewhereElse(owner.PackagePath)
		actual.Types[moved.Key] = &moved
	}},
	{Name: "projected_as_struct", Category: "TYPE_KIND_MISMATCH", Apply: func(_ *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		// A concrete struct cannot stand in for a contract with no
		// implementation in scope.
		actual.Types[owner.Key].Kind = "struct"
	}},
	{Name: "missing_first_member", Category: "MISSING_MEMBER", Apply: func(_ *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		delete(actual.Members, owner.Members[0])
	}},
	{Name: "missing_last_member", Category: "MISSING_MEMBER", Apply: func(_ *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		delete(actual.Members, owner.Members[len(owner.Members)-1])
	}},
	{Name: "renamed_last_member", Category: "MISSING_MEMBER", Apply: func(_ *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		key := owner.Members[len(owner.Members)-1]
		member := *actual.Members[key]
		delete(actual.Members, key)
		member.Key = symbolKey{Package: key.Package, Receiver: key.Receiver, Name: key.Name + "Renamed"}
		actual.Members[member.Key] = &member
	}},
	{Name: "unexpected_member", Category: "UNEXPECTED_MEMBER", Apply: func(_ *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		key := symbolKey{Package: owner.PackagePath, Receiver: owner.GoName, Name: "Invented"}
		actual.Members[key] = &actualMember{Key: key, Kind: "method"}
	}},
	{Name: "wrong_parameter", Category: "PARAMETER_MAPPING_MISMATCH", Requires: interfaceHasParameterizedOperation, Apply: func(t *testing.T, expected *expectedSurface, actual *actualSurface, owner *expectedType) {
		for _, key := range owner.Members {
			if len(expected.Members[key].Parameters) == 1 {
				actual.Members[key].Parameters = []string{"complex128"}
				return
			}
		}
		t.Fatalf("%s projects no single-parameter operation", owner.XNA)
	}},
	{Name: "artificial_error", Category: "ERROR_MAPPING_MISMATCH", Requires: interfaceHasInfallibleOperation, Apply: func(t *testing.T, expected *expectedSurface, actual *actualSurface, owner *expectedType) {
		// Interface ownership alone must never add an error result.
		key := firstMemberWithFallibility(t, expected, owner, false)
		member := actual.Members[key]
		member.Results = append(append([]string(nil), member.Results...), "error")
	}},
	{Name: "dropped_error", Category: "ERROR_MAPPING_MISMATCH", Requires: interfaceHasFallibleOperation, Apply: func(t *testing.T, expected *expectedSurface, actual *actualSurface, owner *expectedType) {
		// An operation that measurably crosses a runtime boundary must keep
		// somewhere to report failure.
		key := firstMemberWithFallibility(t, expected, owner, true)
		member := actual.Members[key]
		member.Results = member.Results[:len(member.Results)-1]
	}},
	{Name: "error_replaces_source_result", Category: "RETURN_MAPPING_MISMATCH", Requires: interfaceHasFallibleValueOperation, Apply: func(t *testing.T, expected *expectedSurface, actual *actualSurface, owner *expectedType) {
		// BeginDraw's Boolean and FogColor's Vector3 are source results and
		// must stay channels of their own rather than collapsing into the
		// error.
		for _, key := range owner.Members {
			if member := expected.Members[key]; member.ErrorAdded && len(member.Results) > 1 {
				actual.Members[key].Results = []string{"error"}
				return
			}
		}
		t.Fatalf("%s projects no fallible value-producing operation", owner.XNA)
	}},
}

func interfaceHasParameterizedOperation(expected *expectedSurface, owner *expectedType) bool {
	for _, key := range owner.Members {
		if len(expected.Members[key].Parameters) == 1 {
			return true
		}
	}
	return false
}

func interfaceHasInfallibleOperation(expected *expectedSurface, owner *expectedType) bool {
	for _, key := range owner.Members {
		if !expected.Members[key].ErrorAdded {
			return true
		}
	}
	return false
}

func interfaceHasFallibleOperation(expected *expectedSurface, owner *expectedType) bool {
	for _, key := range owner.Members {
		if expected.Members[key].ErrorAdded {
			return true
		}
	}
	return false
}

func interfaceHasFallibleValueOperation(expected *expectedSurface, owner *expectedType) bool {
	for _, key := range owner.Members {
		if member := expected.Members[key]; member.ErrorAdded && len(member.Results) > 1 {
			return true
		}
	}
	return false
}

// TestFoundation18InterfaceDefectsRejectedForEveryType applies every
// target-side defect to every projected interface, asserting a clean baseline
// first so no defect can pass by accident.
func TestFoundation18InterfaceDefectsRejectedForEveryType(t *testing.T) {
	if len(allManagedInterfaces()) != 6 {
		t.Fatalf("interface cluster size = %d, want 6", len(allManagedInterfaces()))
	}
	cases, skipped := 0, 0
	for _, pinned := range allManagedInterfaces() {
		pinned := pinned
		t.Run(pinned.XNA, func(t *testing.T) {
			baseExpected, baseActual, baseOwner := managedInterfaceSurfaces(t, pinned.XNA)
			baseline := verify(baseExpected, baseActual, 0, "report", "contract", "mapping")
			if baseline.Summary["TOTAL_DIAGNOSTICS"] != 0 {
				t.Fatalf("unmutated %s baseline is not clean: %v", pinned.XNA, baseline.Diagnostics)
			}
			for _, closure := range baseline.Foundation18Interfaces {
				if closure.XNA == pinned.XNA && closure.Status != "PASS" {
					t.Fatalf("unmutated %s closure = %q", pinned.XNA, closure.Status)
				}
			}
			for _, defect := range managedInterfaceDefects {
				defect := defect
				if defect.Requires != nil && !defect.Requires(baseExpected, baseOwner) {
					skipped++
					continue
				}
				t.Run(defect.Name, func(t *testing.T) {
					expected, actual, owner := managedInterfaceSurfaces(t, pinned.XNA)
					defect.Apply(t, expected, actual, owner)
					result := verify(expected, actual, 0, "report", "contract", "mapping")
					if result.Summary[defect.Category] == 0 {
						t.Fatalf("defect %q on %s did not raise %s; summary=%v",
							defect.Name, pinned.XNA, defect.Category, result.Summary)
					}
					for _, closure := range result.Foundation18Interfaces {
						if closure.XNA == pinned.XNA && closure.Status != "FAIL" {
							t.Fatalf("defect %q on %s left the closure measurement at %q",
								defect.Name, pinned.XNA, closure.Status)
						}
					}
				})
				cases++
			}
		})
	}
	// The seven shape-dependent skips, by contract:
	//   IEffectMatrices        2  no fallible operation exists, so
	//                             dropped_error and error_replaces_source_result
	//                             have nothing to attack
	//   IEffectFog             0  the mixed contract expresses every defect
	//   IGameComponent         3  takes no parameters, has no infallible
	//                             operation, and its one fallible operation
	//                             produces no value alongside the error
	//   IGraphicsDeviceManager 2  takes no parameters and has no infallible
	//                             operation; BeginDraw does produce a value,
	//                             so error_replaces_source_result applies
	//   IUpdateable            0  its event accessors take a parameter, carry
	//   IDrawable              0  an error, and return a value alongside it,
	//                             while its properties and its Update or Draw
	//                             are infallible, so every defect applies
	if cases+skipped != len(allManagedInterfaces())*len(managedInterfaceDefects) {
		t.Fatalf("interface fixture accounting = %d applied + %d skipped", cases, skipped)
	}
	if cases != 59 || skipped != 7 {
		t.Fatalf("interface negative fixtures = %d applied, %d skipped", cases, skipped)
	}
}

// TestFoundation18InterfaceMappedContracts pins the exact projected contract of
// each interface, including which operations are fallible and why.
func TestFoundation18InterfaceMappedContracts(t *testing.T) {
	surface, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	const graphicsPackage = modulePath + "/Microsoft/Xna/Framework/Graphics"
	const frameworkPackage = modulePath + "/Microsoft/Xna/Framework"

	for _, pinned := range allManagedInterfaces() {
		pinned := pinned
		t.Run(pinned.XNA, func(t *testing.T) {
			owner := surface.typeForXNA(pinned.XNA)
			if owner == nil || owner.Kind != "interface" {
				t.Fatalf("%s = %+v", pinned.XNA, owner)
			}
			if len(owner.Interfaces) != 0 {
				t.Fatalf("%s declares base interfaces %v", pinned.XNA, owner.Interfaces)
			}
			if classifiedInterfaces[pinned.XNA] != pinned.Classified {
				t.Fatalf("%s classification = %t, want %t",
					pinned.XNA, classifiedInterfaces[pinned.XNA], pinned.Classified)
			}
			fallible := make(map[string]bool, len(pinned.FallibleOperations)+len(pinned.EventAccessors))
			for _, name := range pinned.FallibleOperations {
				fallible[name] = true
			}
			// An event accessor's error comes from the settled event accessor
			// projection, not from the contract's boundary, so it is expected
			// to be fallible without counting as a boundary operation.
			for _, name := range pinned.EventAccessors {
				fallible[name] = true
			}
			seen := 0
			for _, key := range owner.Members {
				member := surface.Members[key]
				if member.GoKind != "method" || key.Receiver != owner.GoName {
					t.Fatalf("%s.%s = kind %q receiver %q", pinned.XNA, key.Name, member.GoKind, key.Receiver)
				}
				if member.ErrorAdded != fallible[key.Name] {
					t.Fatalf("%s.%s fallible = %t, want %t",
						pinned.XNA, key.Name, member.ErrorAdded, fallible[key.Name])
				}
				if fallible[key.Name] {
					seen++
				}
			}
			if seen != len(pinned.FallibleOperations)+len(pinned.EventAccessors) {
				t.Fatalf("%s matched %d of %d fallible operations and %d event accessors",
					pinned.XNA, seen, len(pinned.FallibleOperations), len(pinned.EventAccessors))
			}
		})
	}

	// The exact signatures that make the mixed contract legible.
	for name, wanted := range map[string]struct {
		parameters []string
		results    []string
	}{
		"World":         {nil, []string{"framework.Matrix"}},
		"SetWorld":      {[]string{"framework.Matrix"}, nil},
		"FogEnabled":    {nil, []string{"bool"}},
		"SetFogEnabled": {[]string{"bool"}, nil},
		"FogStart":      {nil, []string{"float32"}},
		"FogEnd":        {nil, []string{"float32"}},
		"FogColor":      {nil, []string{"framework.Vector3", "error"}},
		"SetFogColor":   {[]string{"framework.Vector3"}, []string{"error"}},
	} {
		receiver := "IEffectFog"
		if strings.Contains(name, "World") {
			receiver = "IEffectMatrices"
		}
		member := surface.Members[symbolKey{Package: graphicsPackage, Receiver: receiver, Name: name}]
		if member == nil || !equalStrings(member.Parameters, wanted.parameters) ||
			!equalStrings(member.Results, wanted.results) {
			t.Fatalf("%s.%s = %+v", receiver, name, member)
		}
	}

	// BeginDraw keeps its source Boolean and its error as separate channels.
	beginDraw := surface.Members[symbolKey{Package: frameworkPackage, Receiver: "IGraphicsDeviceManager", Name: "BeginDraw"}]
	if beginDraw == nil || !equalStrings(beginDraw.Results, []string{"bool", "error"}) {
		t.Fatalf("IGraphicsDeviceManager.BeginDraw = %+v", beginDraw)
	}
}

// TestInterfaceClassificationDefectsAreRejected attacks the interface
// classification rule in both directions using the real classification tables.
func TestInterfaceClassificationDefectsAreRejected(t *testing.T) {
	cases := []struct {
		name        string
		identity    string
		wantMessage string
	}{
		// Dropping the classification restores the interface-kind default,
		// which would make six measurably managed accessors fallible.
		{"pure_managed_interface_demoted_to_runtime", effectMatricesIdentity,
			"property getter expected fallible, projected infallible"},
		// A runtime-boundary contract must not be admitted as classified with
		// no fallible operation recorded.
		{"runtime_interface_admitted_as_pure_managed", graphicsDeviceManagerInterfaceIdentity,
			"method expected infallible, projected fallible"},
		// Losing the FogColor entry drops the one measured D3DX boundary.
		{"interface_runtime_operation_dropped", effectFogIdentity,
			"property getter expected infallible, projected fallible"},
		// Widening it to every fog operation would invent six errors.
		{"interface_runtime_operation_widened", effectFogIdentity,
			"property getter expected fallible, projected infallible"},
	}
	if len(cases) != len(interfaceClassificationDefects) {
		t.Fatalf("interface classification coverage = %d of %d", len(cases), len(interfaceClassificationDefects))
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			mutate, ok := interfaceClassificationDefects[testCase.name]
			if !ok {
				t.Fatalf("interface classification defect %q is not in the shared table", testCase.name)
			}
			_, actual, _ := managedInterfaceSurfaces(t, testCase.identity)
			var result report
			withClassification(t, mutate, func() {
				full, err := buildExpected(loadPinnedContract(t))
				if err != nil {
					t.Fatal(err)
				}
				mutated, _, _ := isolateManagedInterface(t, full, testCase.identity)
				result = verify(mutated, actual, 0, "report", "contract", "mapping")
			})
			if result.Summary["ERROR_MAPPING_MISMATCH"] == 0 {
				t.Fatalf("interface classification defect %q did not raise ERROR_MAPPING_MISMATCH; summary=%v",
					testCase.name, result.Summary)
			}
			found := false
			for _, item := range result.Diagnostics {
				if item.Category == "ERROR_MAPPING_MISMATCH" && strings.Contains(item.Message, testCase.wantMessage) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("interface classification defect %q raised no diagnostic saying %q; diagnostics=%v",
					testCase.name, testCase.wantMessage, result.Diagnostics)
			}
		})
	}
}

const (
	effectMatricesIdentity                 = "Microsoft.Xna.Framework.Graphics.IEffectMatrices"
	effectFogIdentity                      = "Microsoft.Xna.Framework.Graphics.IEffectFog"
	graphicsDeviceManagerInterfaceIdentity = "Microsoft.Xna.Framework.IGraphicsDeviceManager"
)

// managedInterfaceMutationCase applies one named Foundation-18 target-side
// interface defect. Mutation ids have the form f18if_<defect>__<identity>.
func managedInterfaceMutationCase(t *testing.T, mutation string) (*expectedSurface, *actualSurface) {
	t.Helper()
	parts := strings.SplitN(strings.TrimPrefix(mutation, "f18if_"), "__", 2)
	if len(parts) != 2 {
		t.Fatalf("malformed interface mutation %q", mutation)
	}
	expected, actual, owner := managedInterfaceSurfaces(t, parts[1])
	for _, defect := range managedInterfaceDefects {
		if defect.Name != parts[0] {
			continue
		}
		if defect.Requires != nil && !defect.Requires(expected, owner) {
			t.Fatalf("interface defect %q is not expressible on %s", parts[0], parts[1])
		}
		defect.Apply(t, expected, actual, owner)
		return expected, actual
	}
	t.Fatalf("unknown interface defect %q", parts[0])
	return nil, nil
}

// interfaceClassificationMutationCase applies one named Foundation-18
// interface classification defect. Mutation ids have the form
// f18cls_<defect>__<identity>.
func interfaceClassificationMutationCase(t *testing.T, mutation string) report {
	t.Helper()
	parts := strings.SplitN(strings.TrimPrefix(mutation, "f18cls_"), "__", 2)
	if len(parts) != 2 {
		t.Fatalf("malformed interface classification mutation %q", mutation)
	}
	mutate, ok := interfaceClassificationDefects[parts[0]]
	if !ok {
		t.Fatalf("unknown interface classification defect %q", parts[0])
	}
	_, actual, _ := managedInterfaceSurfaces(t, parts[1])
	var result report
	withClassification(t, mutate, func() {
		full, err := buildExpected(loadPinnedContract(t))
		if err != nil {
			t.Fatal(err)
		}
		mutated, _, _ := isolateManagedInterface(t, full, parts[1])
		result = verify(mutated, actual, 0, "report", "contract", "mapping")
	})
	return result
}

// interfaceClassificationDefects is the shared table behind both the named
// interface classification test and the mutation inventory.
var interfaceClassificationDefects = map[string]func(){
	"pure_managed_interface_demoted_to_runtime": func() {
		delete(classifiedInterfaces, effectMatricesIdentity)
	},
	"runtime_interface_admitted_as_pure_managed": func() {
		classifiedInterfaces[graphicsDeviceManagerInterfaceIdentity] = true
	},
	"interface_runtime_operation_dropped": func() {
		delete(managedFallibleMembers, effectFogIdentity)
	},
	"interface_runtime_operation_widened": func() {
		managedFallibleMembers[effectFogIdentity] = map[string]bool{
			"property|FogColor": true, "property|FogEnabled": true,
			"property|FogStart": true, "property|FogEnd": true,
		}
	},
}

// allManagedClassClosures flattens the per-milestone managed-class closure
// slices so a shared defect matrix can assert on whichever one carries the
// type under test.
func allManagedClassClosures(result report) []managedTypeClosure {
	all := append([]managedTypeClosure(nil), result.Foundation17ManagedClasses...)
	all = append(all, result.Foundation19ManagedClasses...)
	all = append(all, result.Foundation20ValueContracts...)
	return append(all, result.Foundation21ManagedClasses...)
}

// somewhereElse returns a mapped package path that is never the given one, so
// a wrong-package defect relocates the type for every owner regardless of
// which package it actually lives in.
func somewhereElse(packagePath string) string {
	const media = modulePath + "/Microsoft/Xna/Framework/Media"
	const graphics = modulePath + "/Microsoft/Xna/Framework/Graphics"
	if packagePath == media {
		return graphics
	}
	return media
}

const presentationParametersIdentity = "Microsoft.Xna.Framework.Graphics.PresentationParameters"

// TestIntPtrProjectsToPointerSizedWord pins the general language projection and
// its one admitted consumer. Every System.IntPtr in the pinned profile is
// declared here, so the rule is stated once and measured against all of them.
func TestIntPtrProjectsToPointerSizedWord(t *testing.T) {
	if bclTypes["System.IntPtr"] != "uintptr" {
		t.Fatalf("System.IntPtr maps to %q", bclTypes["System.IntPtr"])
	}
	surface, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}

	// Every position in the whole profile that carries a pointer-sized word,
	// with the source member that put it there.
	declared := make(map[string]int)
	for _, member := range surface.Members {
		for _, text := range append(append([]string(nil), member.Parameters...), member.Results...) {
			if pointerSizedWord.MatchString(text) {
				declared[member.XNA]++
			}
		}
	}
	// Six CLR members declare System.IntPtr. GameWindow.Handle and
	// GraphicsAdapter.MonitorHandle are read-only properties, and the
	// three-parameter GraphicsDevice.Present overload takes one IntPtr
	// window-handle override, so those contribute one position each. The three read/write properties --
	// PresentationParameters.DeviceWindowHandle and the two static
	// WindowHandle properties on Mouse and TouchPanel -- contribute a getter
	// result and a setter parameter, so two each.
	want := map[string]int{
		"Microsoft.Xna.Framework.GameWindow::Handle()":                      1,
		"Microsoft.Xna.Framework.Graphics.GraphicsAdapter::MonitorHandle()": 1,
		"Microsoft.Xna.Framework.Graphics.GraphicsDevice::Present(System.Nullable`1[Microsoft.Xna.Framework.Rectangle],System.Nullable`1[Microsoft.Xna.Framework.Rectangle],System.IntPtr)": 1,
		"Microsoft.Xna.Framework.Graphics.PresentationParameters::DeviceWindowHandle()":                                                                                                     2,
		"Microsoft.Xna.Framework.Input.Mouse::WindowHandle()":                                                                                                                               2,
		"Microsoft.Xna.Framework.Input.Touch.TouchPanel::WindowHandle()":                                                                                                                    2,
	}
	if len(declared) != len(want) {
		t.Fatalf("pointer-sized word positions on %d members, want %d: %v", len(declared), len(want), declared)
	}
	for identity, count := range want {
		if declared[identity] != count {
			t.Fatalf("%s carries %d pointer-sized positions, want %d", identity, declared[identity], count)
		}
	}

	// The one implemented consumer, spelled out.
	const graphicsPackage = modulePath + "/Microsoft/Xna/Framework/Graphics"
	getter := surface.Members[symbolKey{Package: graphicsPackage, Receiver: "PresentationParameters", Name: "DeviceWindowHandle"}]
	if getter == nil || getter.ErrorAdded || !equalStrings(getter.Results, []string{"uintptr"}) {
		t.Fatalf("DeviceWindowHandle getter = %+v", getter)
	}
	setter := surface.Members[symbolKey{Package: graphicsPackage, Receiver: "PresentationParameters", Name: "SetDeviceWindowHandle"}]
	if setter == nil || setter.ErrorAdded || !equalStrings(setter.Parameters, []string{"uintptr"}) || len(setter.Results) != 0 {
		t.Fatalf("SetDeviceWindowHandle = %+v", setter)
	}
}

// rawHandleFixture is one positive or negative raw-handle case applied to an
// isolated PresentationParameters surface.
type rawHandleFixture struct {
	Name  string
	Leaks bool
	Apply func(t *testing.T, expected *expectedSurface, actual *actualSurface, owner *expectedType)
}

// rawHandleFixtures exercise both sides of the narrowed rule: the admitted
// IntPtr projection must not be flagged, and every other route to a
// pointer-sized word or a native identity in public surface must be.
var rawHandleFixtures = []rawHandleFixture{
	{Name: "admitted_intptr_getter_and_setter", Leaks: false, Apply: func(_ *testing.T, _ *expectedSurface, _ *actualSurface, _ *expectedType) {
		// The unmutated surface already projects both DeviceWindowHandle
		// accessors as uintptr. This is the positive fixture.
	}},
	{Name: "uintptr_result_where_source_declares_int32", Leaks: true, Apply: func(t *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		actual.Members[namedMember(t, owner, "BackBufferWidth")].Results = []string{"uintptr"}
	}},
	{Name: "uintptr_parameter_where_source_declares_int32", Leaks: true, Apply: func(t *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		actual.Members[namedMember(t, owner, "SetBackBufferWidth")].Parameters = []string{"uintptr"}
	}},
	{Name: "uintptr_drifted_from_parameter_to_result", Leaks: true, Apply: func(t *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		// The setter's admitted position is parameter 0, not result 0.
		member := actual.Members[namedMember(t, owner, "SetDeviceWindowHandle")]
		member.Parameters = nil
		member.Results = []string{"uintptr"}
	}},
	{Name: "uintptr_drifted_to_an_unadmitted_index", Leaks: true, Apply: func(t *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		member := actual.Members[namedMember(t, owner, "SetDeviceWindowHandle")]
		member.Parameters = []string{"int32", "uintptr"}
	}},
	{Name: "uintptr_slice_result", Leaks: true, Apply: func(t *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		actual.Members[namedMember(t, owner, "BackBufferWidth")].Results = []string{"[]uintptr"}
	}},
	{Name: "uintptr_pointer_result", Leaks: true, Apply: func(t *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		actual.Members[namedMember(t, owner, "BackBufferWidth")].Results = []string{"*uintptr"}
	}},
	{Name: "uintptr_on_an_invented_member", Leaks: true, Apply: func(_ *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		key := symbolKey{Package: owner.PackagePath, Receiver: owner.GoName, Name: "NativeWindow"}
		actual.Members[key] = &actualMember{Key: key, Kind: "method", Results: []string{"uintptr"}}
	}},
	{Name: "exported_named_type_over_uintptr", Leaks: true, Apply: func(_ *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		key := symbolKey{Package: owner.PackagePath, Name: "WindowToken"}
		actual.Types[key] = &actualType{Key: key, Kind: "named", Underlying: "uintptr"}
	}},
	{Name: "unsafe_pointer_result", Leaks: false, Apply: func(t *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		// unsafe.Pointer is a PUBLIC_NATIVE_FFI_LEAK, not a RAW_HANDLE_LEAK;
		// asserting that keeps the two categories from collapsing.
		actual.Members[namedMember(t, owner, "BackBufferWidth")].Results = []string{"unsafe.Pointer"}
	}},
	{Name: "cna_prefixed_member_name", Leaks: true, Apply: func(_ *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		key := symbolKey{Package: owner.PackagePath, Receiver: owner.GoName, Name: "CnaSwapChain"}
		actual.Members[key] = &actualMember{Key: key, Kind: "method", Results: []string{"int32"}}
	}},
	{Name: "native_handle_type_name", Leaks: true, Apply: func(_ *testing.T, _ *expectedSurface, actual *actualSurface, owner *expectedType) {
		key := symbolKey{Package: owner.PackagePath, Name: "SwapChainNativeHandle"}
		actual.Types[key] = &actualType{Key: key, Kind: "struct"}
	}},
}

// namedMember returns the projected key of one member of a class by Go name.
func namedMember(t *testing.T, owner *expectedType, name string) symbolKey {
	t.Helper()
	for _, key := range owner.Members {
		if key.Name == name {
			return key
		}
	}
	t.Fatalf("%s projects no member named %q", owner.XNA, name)
	return symbolKey{}
}

// TestRawHandleLeakDistinguishesTheIntPtrProjection runs every fixture and
// asserts the exact verdict, so neither direction of the narrowed rule can
// regress: the admitted projection must stay clean and everything else must
// still be caught.
func TestRawHandleLeakDistinguishesTheIntPtrProjection(t *testing.T) {
	positives, negatives := 0, 0
	for _, fixture := range rawHandleFixtures {
		fixture := fixture
		t.Run(fixture.Name, func(t *testing.T) {
			expected, actual, owner := managedClassSurfaces(t, presentationParametersIdentity)
			fixture.Apply(t, expected, actual, owner)
			result := verify(expected, actual, 0, "leak-only", "contract", "mapping")
			got := result.Summary["RAW_HANDLE_LEAK"] > 0
			if got != fixture.Leaks {
				t.Fatalf("raw-handle fixture %q reported leak=%t, want %t; diagnostics=%v",
					fixture.Name, got, fixture.Leaks, result.Diagnostics)
			}
			if fixture.Name == "unsafe_pointer_result" && result.Summary["PUBLIC_NATIVE_FFI_LEAK"] == 0 {
				t.Fatal("unsafe.Pointer did not raise PUBLIC_NATIVE_FFI_LEAK")
			}
		})
		if fixture.Leaks {
			negatives++
		} else {
			positives++
		}
	}
	if positives != 2 || negatives != 10 {
		t.Fatalf("raw-handle fixtures = %d admitted, %d rejected", positives, negatives)
	}
}

// rawHandleMutationCase applies one named Foundation-19 raw-handle fixture.
// Mutation ids have the form f19rh_<fixture>.
func rawHandleMutationCase(t *testing.T, mutation string) (*expectedSurface, *actualSurface) {
	t.Helper()
	name := strings.TrimPrefix(mutation, "f19rh_")
	for _, fixture := range rawHandleFixtures {
		if fixture.Name != name {
			continue
		}
		expected, actual, owner := managedClassSurfaces(t, presentationParametersIdentity)
		fixture.Apply(t, expected, actual, owner)
		return expected, actual
	}
	t.Fatalf("unknown raw-handle fixture %q", name)
	return nil, nil
}

// isolateEventOwner builds an isolated, initially correct expected/actual pair
// for one event-bearing XNA type, so an event defect can be injected on the
// target side and nothing else can account for the resulting diagnostic.
func isolateEventOwner(t *testing.T, identity string) (*expectedSurface, *actualSurface, *expectedType) {
	t.Helper()
	full, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	fullType := full.typeForXNA(identity)
	if fullType == nil {
		t.Fatalf("%s is not in the pinned contract", identity)
	}
	copiedType := *fullType
	copiedType.Members = nil
	kind := "struct"
	if fullType.Kind == "interface" {
		kind = "interface"
	}
	expected := &expectedSurface{
		Types:              map[symbolKey]*expectedType{copiedType.Key: &copiedType},
		Members:            make(map[symbolKey]*expectedMember),
		InterfaceWitnesses: make(map[symbolKey]*expectedInterfaceWitness),
		ReferenceTypes:     1,
		ExpectedGoTypes:    1,
	}
	actual := &actualSurface{
		Types:       map[symbolKey]*actualType{copiedType.Key: {Key: copiedType.Key, Kind: kind}},
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: make(map[string]string),
		Packages:    make(map[string]*types.Package),
	}
	// Only the event accessors are carried across, so every diagnostic the
	// fixtures produce is attributable to the event projection alone.
	for _, memberKey := range fullType.Members {
		fullMember := full.Members[memberKey]
		if fullMember.SourceKind != "event" {
			continue
		}
		copiedMember := *fullMember
		copiedMember.Parameters = append([]string(nil), fullMember.Parameters...)
		copiedMember.Results = append([]string(nil), fullMember.Results...)
		copiedType.Members = append(copiedType.Members, memberKey)
		expected.Members[memberKey] = &copiedMember
		actual.Members[memberKey] = &actualMember{
			Key:        memberKey,
			Kind:       copiedMember.GoKind,
			Parameters: append([]string(nil), copiedMember.Parameters...),
			Results:    append([]string(nil), copiedMember.Results...),
		}
	}
	if len(copiedType.Members) == 0 {
		t.Fatalf("%s declares no event", identity)
	}
	expected.ReferenceMembers = len(copiedType.Members) / 2
	expected.ExpectedGoMembers = len(copiedType.Members)
	return expected, actual, &copiedType
}

func eventAccessorKeys(t *testing.T, expected *expectedSurface, owner *expectedType) (add, remove symbolKey) {
	t.Helper()
	for _, key := range owner.Members {
		if strings.HasPrefix(key.Name, "Add") && add == (symbolKey{}) {
			add = key
		}
		if strings.HasPrefix(key.Name, "Remove") && remove == (symbolKey{}) {
			remove = key
		}
	}
	if add == (symbolKey{}) || remove == (symbolKey{}) {
		t.Fatalf("%s has no add/remove accessor pair", owner.XNA)
	}
	return add, remove
}

// TestEventProjectionIsMeasuredExactly pins the settled event mapping on both
// sides of the package-qualification rule and proves the generic argument is
// carried exactly rather than degraded.
func TestEventProjectionIsMeasuredExactly(t *testing.T) {
	surface, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	const frameworkPackage = modulePath + "/Microsoft/Xna/Framework"
	const graphicsPackage = modulePath + "/Microsoft/Xna/Framework/Graphics"

	for _, want := range []struct {
		pkg        string
		receiver   string
		name       string
		parameters []string
		results    []string
	}{
		// Same package: the adapters are unqualified.
		{frameworkPackage, "IUpdateable", "AddEnabledChangedHandler",
			[]string{"EventHandler[*EventArgs]"}, []string{"EventSubscription", "error"}},
		{frameworkPackage, "IUpdateable", "RemoveEnabledChangedHandler",
			[]string{"EventSubscription"}, []string{"error"}},
		// Descendant package: every adapter takes the framework qualification.
		{graphicsPackage, "DynamicVertexBuffer", "AddContentLostHandler",
			[]string{"framework.EventHandler[*framework.EventArgs]"}, []string{"framework.EventSubscription", "error"}},
		{graphicsPackage, "DynamicVertexBuffer", "RemoveContentLostHandler",
			[]string{"framework.EventSubscription"}, []string{"error"}},
		// A non-EventArgs generic argument is carried exactly, and an XNA args
		// type in the owner's own package is not over-qualified.
		{graphicsPackage, "GraphicsDevice", "AddResourceCreatedHandler",
			[]string{"framework.EventHandler[*ResourceCreatedEventArgs]"}, []string{"framework.EventSubscription", "error"}},
		{frameworkPackage, "GraphicsDeviceManager", "AddPreparingDeviceSettingsHandler",
			[]string{"EventHandler[*PreparingDeviceSettingsEventArgs]"}, []string{"EventSubscription", "error"}},
	} {
		want := want
		t.Run(want.receiver+"."+want.name, func(t *testing.T) {
			member := surface.Members[symbolKey{Package: want.pkg, Receiver: want.receiver, Name: want.name}]
			if member == nil {
				t.Fatalf("%s.%s is not projected", want.receiver, want.name)
			}
			if member.SourceKind != "event" {
				t.Fatalf("source kind = %q", member.SourceKind)
			}
			if !equalStrings(member.Parameters, want.parameters) {
				t.Fatalf("parameters = %v, want %v", member.Parameters, want.parameters)
			}
			if !equalStrings(member.Results, want.results) {
				t.Fatalf("results = %v, want %v", member.Results, want.results)
			}
			if !member.ErrorAdded {
				t.Fatal("event accessor lost its error channel")
			}
		})
	}

	// Every CLR event in the profile becomes exactly two accessors and no event
	// anywhere degrades its handler to `any`.
	events, accessors := 0, 0
	for _, et := range sortedExpectedTypes(surface) {
		for _, key := range et.Members {
			member := surface.Members[key]
			if member.SourceKind != "event" {
				continue
			}
			accessors++
			// A static event projects as a package function whose name carries
			// the declaring type, so the accessor stem is not at index 0.
			registration := strings.HasPrefix(key.Name, "Add")
			if key.Receiver == "" {
				registration = strings.HasPrefix(strings.TrimPrefix(key.Name, et.GoName), "Add")
			}
			if registration {
				events++
				if len(member.Parameters) != 1 || strings.Contains(member.Parameters[0], "any") {
					t.Fatalf("%s handler = %v", member.XNA, member.Parameters)
				}
				if !strings.Contains(member.Parameters[0], "EventHandler[") {
					t.Fatalf("%s handler = %v, want the EventHandler adapter", member.XNA, member.Parameters)
				}
			}
		}
	}
	// 49 XNA-DECLARED events producing 98 accessors, plus the six an
	// XNA-inherited projection adds: DrawableGameComponent and
	// GamerServicesComponent each inherit GameComponent's three events, and
	// each inherited event projects the same two accessors a declared one does.
	// The declared count is what must never move; the inherited part is pinned
	// beside it so a change in either is attributed.
	declaredEvents, declaredAccessors := 0, 0
	inheritedEvents, inheritedAccessors := 0, 0
	for _, et := range surface.Types {
		for _, key := range et.Members {
			member := surface.Members[key]
			if member.SourceKind != "event" {
				continue
			}
			registration := strings.HasPrefix(key.Name, "Add")
			if key.Receiver == "" {
				registration = strings.HasPrefix(strings.TrimPrefix(key.Name, et.GoName), "Add")
			}
			if member.XNABase != "" {
				inheritedAccessors++
				if registration {
					inheritedEvents++
				}
				continue
			}
			declaredAccessors++
			if registration {
				declaredEvents++
			}
		}
	}
	if declaredEvents != 49 || declaredAccessors != 98 {
		t.Fatalf("XNA-declared events = %d producing %d accessors, want 49 and 98", declaredEvents, declaredAccessors)
	}
	if inheritedEvents != 6 || inheritedAccessors != 12 {
		t.Fatalf("XNA-inherited events = %d producing %d accessors, want 6 and 12", inheritedEvents, inheritedAccessors)
	}
	if events != declaredEvents+inheritedEvents || accessors != declaredAccessors+inheritedAccessors {
		t.Fatalf("event partition = %d/%d, walk found %d/%d", events, accessors,
			declaredEvents+inheritedEvents, declaredAccessors+inheritedAccessors)
	}
}

// eventProjectionDefects are the target-side defects the event mapping is
// negatively fixtured against. Each one is a way a binding could plausibly
// weaken the event projection.
var eventProjectionDefects = []struct {
	Name     string
	Category string
	Apply    func(t *testing.T, actual *actualSurface, add, remove symbolKey, qualifier string)
}{
	{"handler-degraded-to-any", "EVENT_MAPPING_MISMATCH", func(t *testing.T, actual *actualSurface, add, remove symbolKey, q string) {
		actual.Members[add].Parameters = []string{q + "EventHandler[any]"}
	}},
	{"handler-erased-to-any", "EVENT_MAPPING_MISMATCH", func(t *testing.T, actual *actualSurface, add, remove symbolKey, q string) {
		actual.Members[add].Parameters = []string{"any"}
	}},
	{"wrong-generic-argument", "PARAMETER_MAPPING_MISMATCH", func(t *testing.T, actual *actualSurface, add, remove symbolKey, q string) {
		actual.Members[add].Parameters = []string{q + "EventHandler[*" + q + "GameTime]"}
	}},
	{"event-args-projected-by-value", "PARAMETER_MAPPING_MISMATCH", func(t *testing.T, actual *actualSurface, add, remove symbolKey, q string) {
		actual.Members[add].Parameters = []string{q + "EventHandler[" + q + "EventArgs]"}
	}},
	{"handler-is-a-raw-func", "EVENT_MAPPING_MISMATCH", func(t *testing.T, actual *actualSurface, add, remove symbolKey, q string) {
		actual.Members[add].Parameters = []string{"func(sender any, args *" + q + "EventArgs) error"}
	}},
	{"handler-is-a-channel", "EVENT_MAPPING_MISMATCH", func(t *testing.T, actual *actualSurface, add, remove symbolKey, q string) {
		actual.Members[add].Parameters = []string{"chan *" + q + "EventArgs"}
	}},
	{"handler-is-a-raw-callback-word", "EVENT_MAPPING_MISMATCH", func(t *testing.T, actual *actualSurface, add, remove symbolKey, q string) {
		actual.Members[add].Parameters = []string{"unsafe.Pointer"}
	}},
	{"subscription-token-dropped", "EVENT_MAPPING_MISMATCH", func(t *testing.T, actual *actualSurface, add, remove symbolKey, q string) {
		actual.Members[add].Results = []string{"error"}
	}},
	{"subscription-token-is-a-native-handle", "EVENT_MAPPING_MISMATCH", func(t *testing.T, actual *actualSurface, add, remove symbolKey, q string) {
		actual.Members[add].Results = []string{"uintptr", "error"}
	}},
	{"removal-takes-the-handler", "EVENT_MAPPING_MISMATCH", func(t *testing.T, actual *actualSurface, add, remove symbolKey, q string) {
		actual.Members[remove].Parameters = []string{q + "EventHandler[*" + q + "EventArgs]"}
	}},
	{"removal-takes-a-native-handle", "EVENT_MAPPING_MISMATCH", func(t *testing.T, actual *actualSurface, add, remove symbolKey, q string) {
		actual.Members[remove].Parameters = []string{"uintptr"}
	}},
	{"registration-error-channel-dropped", "ERROR_MAPPING_MISMATCH", func(t *testing.T, actual *actualSurface, add, remove symbolKey, q string) {
		actual.Members[add].Results = []string{q + "EventSubscription"}
	}},
	{"removal-error-channel-dropped", "ERROR_MAPPING_MISMATCH", func(t *testing.T, actual *actualSurface, add, remove symbolKey, q string) {
		actual.Members[remove].Results = nil
	}},
	{"registration-accessor-missing", "MISSING_MEMBER", func(t *testing.T, actual *actualSurface, add, remove symbolKey, q string) {
		delete(actual.Members, add)
	}},
	{"removal-accessor-missing", "MISSING_MEMBER", func(t *testing.T, actual *actualSurface, add, remove symbolKey, q string) {
		delete(actual.Members, remove)
	}},
	{"clr-accessor-name-leaked", "EVENT_MAPPING_MISMATCH", func(t *testing.T, actual *actualSurface, add, remove symbolKey, q string) {
		leaked := symbolKey{Package: add.Package, Receiver: add.Receiver, Name: strings.TrimSuffix(add.Name, "Handler")}
		actual.Members[leaked] = &actualMember{Key: leaked, Kind: "method"}
	}},
	{"accessor-projected-as-a-field", "EVENT_MAPPING_MISMATCH", func(t *testing.T, actual *actualSurface, add, remove symbolKey, q string) {
		actual.Members[add].Kind = "field"
	}},
}

// TestEventProjectionDefectsAreRejected runs every event defect against an
// owner in the framework package and an owner in a descendant package, so the
// package-qualification half of the rule is attacked as hard as the shape half.
func TestEventProjectionDefectsAreRejected(t *testing.T) {
	owners := []struct {
		identity  string
		qualifier string
	}{
		{"Microsoft.Xna.Framework.IUpdateable", ""},
		{"Microsoft.Xna.Framework.Graphics.DynamicVertexBuffer", "framework."},
	}
	cases := 0
	for _, owner := range owners {
		owner := owner
		t.Run(owner.identity, func(t *testing.T) {
			baseExpected, baseActual, baseOwner := isolateEventOwner(t, owner.identity)
			baseline := verify(baseExpected, baseActual, 0, "report", "contract", "mapping")
			if baseline.Summary["TOTAL_DIAGNOSTICS"] != 0 {
				t.Fatalf("unmutated %s baseline is not clean: %v", owner.identity, baseline.Diagnostics)
			}
			_ = baseOwner
			for _, defect := range eventProjectionDefects {
				defect := defect
				t.Run(defect.Name, func(t *testing.T) {
					expected, actual, isolated := isolateEventOwner(t, owner.identity)
					add, remove := eventAccessorKeys(t, expected, isolated)
					defect.Apply(t, actual, add, remove, owner.qualifier)
					result := verify(expected, actual, 0, "report", "contract", "mapping")
					if result.Summary[defect.Category] == 0 {
						t.Fatalf("defect %q on %s did not raise %s; summary=%v",
							defect.Name, owner.identity, defect.Category, result.Summary)
					}
				})
				cases++
			}
		})
	}
	if cases != len(owners)*len(eventProjectionDefects) {
		t.Fatalf("event fixture accounting = %d", cases)
	}
	if cases != 34 {
		t.Fatalf("event negative fixtures = %d, want 34", cases)
	}
}

// TestEventAdapterSurfaceIsDeclaredLanguageSupport proves the four event
// adapters are measured as language support: they are registered adapters, they
// live in the framework package, and none of them is an XNA identity.
func TestEventAdapterSurfaceIsDeclaredLanguageSupport(t *testing.T) {
	for _, name := range []string{"EventArgs", "EventHandler", "EventSource", "EventSubscription"} {
		if !adapterTypes[name] {
			t.Fatalf("%s is not a declared language adapter", name)
		}
	}
	if !adapterFunctions["EventArgsEmpty"] {
		t.Fatal("EventArgsEmpty is not a declared adapter function")
	}
	surface, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	// No adapter may collide with a projected XNA identity.
	for key := range surface.Types {
		if adapterTypes[key.Name] && key.Name != "TimeSpan" && key.Name != "GameCallbacks" {
			t.Fatalf("language adapter %s collides with a projected XNA type", key.Name)
		}
	}
	const frameworkPackage = modulePath + "/Microsoft/Xna/Framework"
	for _, name := range []string{"EventArgs", "EventHandler", "EventSource", "EventSubscription"} {
		if !isAdapterType(symbolKey{Package: frameworkPackage, Name: name}, &actualType{}) {
			t.Fatalf("%s is not admitted as a framework-package adapter type", name)
		}
		if isAdapterType(symbolKey{Package: frameworkPackage + "/Graphics", Name: name}, &actualType{}) {
			t.Fatalf("%s was admitted outside the framework package", name)
		}
	}
}

// TestBCLBaseRelationshipsAreExhaustive proves the base table covers every
// non-XNA CLR base in the pinned profile. That exhaustiveness is what makes the
// relationship measured rather than silently dropped: a base nobody has decided
// about cannot exist without failing here.
func TestBCLBaseRelationshipsAreExhaustive(t *testing.T) {
	reference := loadPinnedContract(t)
	seen := make(map[string]int)
	for _, declared := range reference.Types {
		base := valueOrEmpty(declared.BaseType)
		if base == "" || strings.HasPrefix(base, "Microsoft.Xna.Framework") {
			continue
		}
		identity := baseIdentityWithoutArguments(base)
		seen[identity]++
		if _, ok := bclBaseRelationships[identity]; !ok {
			t.Fatalf("%s derives from undeclared BCL base %q", declared.Name, identity)
		}
	}
	for identity := range bclBaseRelationships {
		if seen[identity] == 0 {
			t.Fatalf("declared BCL base %q has no derived type in the profile", identity)
		}
	}
	// The three universal CLR roots plus the nine special bases the profile
	// actually uses.
	if len(bclBaseRelationships) != 12 {
		t.Fatalf("declared BCL base relationships = %d, want 12", len(bclBaseRelationships))
	}
	if bclBaseRelationships["System.EventArgs"].Status != "MAPPED" ||
		bclBaseRelationships["System.EventArgs"].Adapter != "EventArgs" {
		t.Fatalf("System.EventArgs = %+v", bclBaseRelationships["System.EventArgs"])
	}
	for _, deferred := range []string{"System.Exception", "System.Attribute", "System.Runtime.InteropServices.ExternalException"} {
		if bclBaseRelationships[deferred].Status != "DEFERRED" {
			t.Fatalf("%s = %+v, want DEFERRED", deferred, bclBaseRelationships[deferred])
		}
	}
}

// baseProjectionFixture isolates one derived type and projects it correctly, so
// a base defect can be injected with nothing else to account for it.
func baseProjectionFixture(t *testing.T, identity string) (*expectedSurface, *actualSurface, *expectedType) {
	t.Helper()
	full, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	fullType := full.typeForXNA(identity)
	if fullType == nil {
		t.Fatalf("%s is not in the pinned contract", identity)
	}
	copiedType := *fullType
	copiedType.Members = nil
	expected := &expectedSurface{
		Types:              map[symbolKey]*expectedType{copiedType.Key: &copiedType},
		Members:            make(map[symbolKey]*expectedMember),
		InterfaceWitnesses: make(map[symbolKey]*expectedInterfaceWitness),
		ReferenceTypes:     1,
		ExpectedGoTypes:    1,
	}
	actual := &actualSurface{
		Types:       map[symbolKey]*actualType{copiedType.Key: {Key: copiedType.Key, Kind: "struct"}},
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: make(map[string]string),
		Packages:    make(map[string]*types.Package),
	}
	seedSignatureAdapters(actual)
	return expected, actual, &copiedType
}

// seedSignatureAdapters gives an isolated fixture the BCL signature adapters
// the framework package must always carry, so a fixture represents a VALID
// surface and a mutation's diagnostic is caused by the mutation rather than by
// the fixture being incomplete.
func seedSignatureAdapters(actual *actualSurface) {
	frameworkPackage := modulePath + "/Microsoft/Xna/Framework"
	// Naming the package is what turns the adapter measurement on, so a
	// seeded fixture exercises it rather than skipping it.
	if actual.PackageDirs == nil {
		actual.PackageDirs = make(map[string]string)
	}
	actual.PackageDirs[frameworkPackage] = "Microsoft/Xna/Framework"
	for _, adapter := range bclSignatureAdapters {
		goName := bclSignatureAdapterGoName(adapter)
		key := symbolKey{Package: frameworkPackage, Name: goName}
		if _, present := actual.Types[key]; !present {
			actual.Types[key] = &actualType{Key: key, Kind: "struct", TypeParameters: []string{"T"}}
		}
		for _, entry := range adapter.Members {
			memberKey := symbolKey{Package: frameworkPackage, Receiver: goName, Name: entry.Member.Name}
			if _, present := actual.Members[memberKey]; !present {
				actual.Members[memberKey] = &actualMember{Key: memberKey, Kind: "method"}
			}
		}
	}
}

// TestBCLBaseProjectionDefectsAreRejected attacks the base decision in every
// direction the rule forbids.
func TestBCLBaseProjectionDefectsAreRejected(t *testing.T) {
	// A MAPPED base: the derived type may be projected, but not by faking CLR
	// inheritance and not as something other than a reference struct.
	const mapped = "Microsoft.Xna.Framework.GameComponentCollectionEventArgs"
	// A DEFERRED base: no derived type may be projected at all yet.
	const deferred = "Microsoft.Xna.Framework.Graphics.DeviceLostException"

	for _, defect := range []struct {
		name     string
		identity string
		apply    func(expected *expectedSurface, actual *actualSurface, owner *expectedType)
	}{
		{"exported-embedding-fakes-inheritance", mapped, func(e *expectedSurface, a *actualSurface, owner *expectedType) {
			a.Types[owner.Key].ExportedEmbeddings = []string{"EventArgs"}
		}},
		{"framework-adapter-embedded-by-qualified-name", mapped, func(e *expectedSurface, a *actualSurface, owner *expectedType) {
			a.Types[owner.Key].ExportedEmbeddings = []string{"framework.EventArgs"}
		}},
		{"derived-class-projected-as-an-interface", mapped, func(e *expectedSurface, a *actualSurface, owner *expectedType) {
			a.Types[owner.Key].Kind = "interface"
		}},
		{"deferred-base-projected-anyway", deferred, func(e *expectedSurface, a *actualSurface, owner *expectedType) {
			// The fixture already projects it; that alone is the defect.
		}},
		{"undeclared-bcl-base", mapped, func(e *expectedSurface, a *actualSurface, owner *expectedType) {
			owner.BaseType = "System.Something.Undecided"
		}},
	} {
		defect := defect
		t.Run(defect.name, func(t *testing.T) {
			expected, actual, owner := baseProjectionFixture(t, defect.identity)
			baseline := verify(expected, actual, 0, "report", "contract", "mapping")
			if defect.name != "deferred-base-projected-anyway" && baseline.Summary["BASE_MAPPING_MISMATCH"] != 0 {
				t.Fatalf("unmutated %s baseline already fails: %v", defect.identity, baseline.Diagnostics)
			}
			expected, actual, owner = baseProjectionFixture(t, defect.identity)
			defect.apply(expected, actual, owner)
			result := verify(expected, actual, 0, "report", "contract", "mapping")
			if result.Summary["BASE_MAPPING_MISMATCH"] == 0 {
				t.Fatalf("defect %q did not raise BASE_MAPPING_MISMATCH; summary=%v", defect.name, result.Summary)
			}
		})
	}
}

// TestBCLBaseRelationshipMeasurementIsReported proves the relationship table
// reaches the report with a verdict per base rather than only when something
// goes wrong.
func TestBCLBaseRelationshipMeasurementIsReported(t *testing.T) {
	expected, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	actual := &actualSurface{
		Types:       make(map[symbolKey]*actualType),
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: make(map[string]string),
		Packages:    make(map[string]*types.Package),
	}
	measurements := measureBCLBaseRelationships(expected, actual)
	if len(measurements) != len(bclBaseRelationships) {
		t.Fatalf("measured %d relationships, want %d", len(measurements), len(bclBaseRelationships))
	}
	derived := 0
	for _, row := range measurements {
		if row.Verdict != "PASS" {
			t.Fatalf("%s verdict = %q with nothing projected", row.CLRBase, row.Verdict)
		}
		// COMPOSED is the one status under which a base contributes Go
		// member identities, and it must be the only one. Every other
		// relationship keeps the original invariant: a CLR base contributes
		// no Go member identity of its own.
		if row.AddsProjectedSurface != (row.Status == "COMPOSED") {
			t.Fatalf("%s is %s and reports addsProjectedSurface=%v; only a COMPOSED base contributes Go member identities",
				row.CLRBase, row.Status, row.AddsProjectedSurface)
		}
		if row.Status == "COMPOSED" {
			if _, hasAdapter := bclBaseAdapters[row.CLRBase]; !hasAdapter {
				t.Fatalf("%s is COMPOSED but declares no base adapter", row.CLRBase)
			}
		}
		derived += row.DerivedTypes
	}
	// Every non-XNA-based type in the profile is accounted for by exactly one
	// relationship row.
	want := 0
	for _, declared := range loadPinnedContract(t).Types {
		base := valueOrEmpty(declared.BaseType)
		if base != "" && !strings.HasPrefix(base, "Microsoft.Xna.Framework") {
			want++
		}
	}
	if derived != want {
		t.Fatalf("relationship rows cover %d derived types, want %d", derived, want)
	}
}

// eventProjectionMutationCase applies one named event defect. Mutation ids have
// the form f22ev_<defect>__<identity>.
func eventProjectionMutationCase(t *testing.T, mutation string) (*expectedSurface, *actualSurface) {
	t.Helper()
	parts := strings.SplitN(strings.TrimPrefix(mutation, "f22ev_"), "__", 2)
	if len(parts) != 2 {
		t.Fatalf("malformed event mutation %q", mutation)
	}
	expected, actual, owner := isolateEventOwner(t, parts[1])
	add, remove := eventAccessorKeys(t, expected, owner)
	qualifier := ""
	if owner.PackagePath != modulePath+"/Microsoft/Xna/Framework" {
		qualifier = "framework."
	}
	for _, defect := range eventProjectionDefects {
		if defect.Name != parts[0] {
			continue
		}
		defect.Apply(t, actual, add, remove, qualifier)
		return expected, actual
	}
	t.Fatalf("unknown event defect %q", parts[0])
	return nil, nil
}

// baseProjectionMutationCase applies one named BCL base defect. Mutation ids
// have the form f22base_<defect>__<identity>.
func baseProjectionMutationCase(t *testing.T, mutation string) (*expectedSurface, *actualSurface) {
	t.Helper()
	parts := strings.SplitN(strings.TrimPrefix(mutation, "f22base_"), "__", 2)
	if len(parts) != 2 {
		t.Fatalf("malformed base mutation %q", mutation)
	}
	expected, actual, owner := baseProjectionFixture(t, parts[1])
	switch parts[0] {
	case "exported_embedding":
		actual.Types[owner.Key].ExportedEmbeddings = []string{"EventArgs"}
	case "qualified_embedding":
		actual.Types[owner.Key].ExportedEmbeddings = []string{"framework.EventArgs"}
	case "projected_as_interface":
		actual.Types[owner.Key].Kind = "interface"
	case "deferred_base_projected":
		// Projecting the type at all is the defect.
	case "undeclared_base":
		owner.BaseType = "System.Something.Undecided"
	default:
		t.Fatalf("unknown base defect %q", parts[0])
	}
	return expected, actual
}

// isolateTypeSurface builds an isolated, initially correct expected/actual pair
// for one XNA type with every projected member present.
func isolateTypeSurface(t *testing.T, identity string) (*expectedSurface, *actualSurface, *expectedType) {
	t.Helper()
	full, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	fullType := full.typeForXNA(identity)
	if fullType == nil {
		t.Fatalf("%s is not in the pinned contract", identity)
	}
	copiedType := *fullType
	copiedType.Members = append([]symbolKey(nil), fullType.Members...)
	kind := "struct"
	if fullType.Kind == "interface" {
		kind = "interface"
	}
	expected := &expectedSurface{
		Types:              map[symbolKey]*expectedType{copiedType.Key: &copiedType},
		Members:            make(map[symbolKey]*expectedMember),
		InterfaceWitnesses: make(map[symbolKey]*expectedInterfaceWitness),
		ReferenceTypes:     1,
		ReferenceMembers:   copiedType.SourceMembers,
		ExpectedGoTypes:    1,
		ExpectedGoMembers:  len(copiedType.Members),
	}
	actual := &actualSurface{
		Types:       map[symbolKey]*actualType{copiedType.Key: {Key: copiedType.Key, Kind: kind}},
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: make(map[string]string),
		Packages:    make(map[string]*types.Package),
	}
	for _, memberKey := range copiedType.Members {
		fullMember := full.Members[memberKey]
		copiedMember := *fullMember
		copiedMember.Parameters = append([]string(nil), fullMember.Parameters...)
		copiedMember.Results = append([]string(nil), fullMember.Results...)
		expected.Members[memberKey] = &copiedMember
		actual.Members[memberKey] = &actualMember{
			Key:        memberKey,
			Kind:       copiedMember.GoKind,
			Parameters: append([]string(nil), copiedMember.Parameters...),
			Results:    append([]string(nil), copiedMember.Results...),
			Value:      copiedMember.EnumValue,
		}
	}
	seedSignatureAdapters(actual)
	return expected, actual, &copiedType
}

func disposeMemberKey(t *testing.T, owner *expectedType) symbolKey {
	t.Helper()
	for _, key := range owner.Members {
		if key.Name == "Dispose" {
			return key
		}
	}
	t.Fatalf("%s declares no projected Dispose", owner.XNA)
	return symbolKey{}
}

// TestBCLInterfaceRelationshipsAreExhaustive proves the interface table covers
// every non-XNA direct interface in the profile, and that each entry claims
// exactly one thing: it contributes no projected Go surface.
func TestBCLInterfaceRelationshipsAreExhaustive(t *testing.T) {
	reference := loadPinnedContract(t)
	seen := make(map[string]int)
	for _, declared := range reference.Types {
		for _, raw := range declared.DirectInterfaces {
			identity := baseIdentityWithoutArguments(raw)
			if strings.HasPrefix(identity, "Microsoft.Xna.Framework") {
				continue
			}
			seen[identity]++
			if _, ok := bclInterfaceRelationships[identity]; !ok {
				t.Fatalf("%s declares undeclared BCL interface %q", declared.Name, identity)
			}
		}
	}
	for identity := range bclInterfaceRelationships {
		if seen[identity] == 0 {
			t.Fatalf("declared BCL interface %q has no declaring type in the profile", identity)
		}
		if bclInterfaceRelationships[identity].Status != "MAPPED_NO_SURFACE" {
			t.Fatalf("%s status = %q", identity, bclInterfaceRelationships[identity].Status)
		}
	}
	if len(bclInterfaceRelationships) != 8 {
		t.Fatalf("declared BCL interface relationships = %d, want 8", len(bclInterfaceRelationships))
	}
	if seen["System.IDisposable"] != 29 {
		t.Fatalf("System.IDisposable declaring types = %d, want 29", seen["System.IDisposable"])
	}

	// The same exhaustiveness applies to XNA-namespaced interfaces that are not
	// public contract types: they are assembly-visible, so they must be
	// declared internal rather than silently skipped.
	surface, err := buildExpected(reference)
	if err != nil {
		t.Fatal(err)
	}
	internal := make(map[string]int)
	for _, declared := range reference.Types {
		for _, raw := range declared.DirectInterfaces {
			identity := baseIdentityWithoutArguments(raw)
			if !strings.HasPrefix(identity, "Microsoft.Xna.Framework") {
				continue
			}
			if surface.typeForXNA(identity) != nil {
				continue
			}
			internal[identity]++
			if _, ok := internalXNAInterfaces[identity]; !ok {
				t.Fatalf("%s declares undeclared internal XNA interface %q", declared.Name, identity)
			}
		}
	}
	for identity, relationship := range internalXNAInterfaces {
		if internal[identity] == 0 {
			t.Fatalf("declared internal XNA interface %q has no declaring type", identity)
		}
		if relationship.Status != "INTERNAL_NO_SURFACE" {
			t.Fatalf("%s status = %q", identity, relationship.Status)
		}
		// An internal interface must not also be a public contract type.
		if surface.typeForXNA(identity) != nil {
			t.Fatalf("%s is declared internal but is a public contract type", identity)
		}
		// None of its members may be projected anywhere.
		for _, member := range relationship.Members {
			for key := range surface.Members {
				if key.Name == member && strings.HasPrefix(key.Package, modulePath+"/Microsoft/Xna/Framework/Graphics") &&
					key.Receiver != "" && surface.Members[key].SourceKind == "method" {
					t.Fatalf("internal interface member %q is projected as %s", member, key.String())
				}
			}
		}
	}
	if len(internalXNAInterfaces) != 2 {
		t.Fatalf("declared internal XNA interfaces = %d, want 2", len(internalXNAInterfaces))
	}
	if internal["Microsoft.Xna.Framework.Graphics.IGraphicsResource"] != 7 ||
		internal["Microsoft.Xna.Framework.Graphics.IDynamicGraphicsResource"] != 4 {
		t.Fatalf("internal interface declaring types = %v", internal)
	}
}

// TestIDisposableAddsNoProjectedSurface is the arithmetic behind the rule: the
// interface's own member never becomes a Go identity, and the only Dispose
// identities anywhere in the profile come from types that declare Dispose in
// their own public contract.
func TestIDisposableAddsNoProjectedSurface(t *testing.T) {
	reference := loadPinnedContract(t)
	surface, err := buildExpected(reference)
	if err != nil {
		t.Fatal(err)
	}
	declaresDispose := make(map[string]bool)
	implementsIDisposable := make(map[string]bool)
	for _, declared := range reference.Types {
		for _, raw := range declared.DirectInterfaces {
			if baseIdentityWithoutArguments(raw) == "System.IDisposable" {
				implementsIDisposable[declared.Name] = true
			}
		}
		for _, member := range declared.Members {
			if strings.HasPrefix(member.Name, "Dispose") && member.Kind == "method" {
				declaresDispose[declared.Name] = true
			}
		}
	}

	// Every projected Dispose identity traces to a declared XNA member -- on
	// the owner itself, or, for an XNA-inherited projection, on the base it is
	// attributed to. The second case is not an exception to the rule: it is the
	// same rule read through the provenance the member carries.
	for key, member := range surface.Members {
		if !strings.HasPrefix(key.Name, "Dispose") {
			continue
		}
		if member.XNABase != "" {
			if !declaresDispose[member.XNABase] {
				t.Fatalf("%s projects %s as inherited from %s, which declares no Dispose member",
					member.Owner, key.Name, member.XNABase)
			}
			continue
		}
		if !declaresDispose[member.Owner] {
			t.Fatalf("%s projects %s but declares no Dispose member", member.Owner, key.Name)
		}
	}

	// The profile's proof case: GraphicsDeviceManager implements IDisposable
	// but implements it explicitly, so its Dispose() is not public surface and
	// nothing may be projected for it.
	manager := surface.typeForXNA("Microsoft.Xna.Framework.GraphicsDeviceManager")
	if manager == nil || !implementsIDisposable["Microsoft.Xna.Framework.GraphicsDeviceManager"] {
		t.Fatal("GraphicsDeviceManager is not the expected IDisposable implementor")
	}
	// Its public contract declares exactly one Dispose, the protected
	// Dispose(bool), so exactly one Dispose identity is projected and it takes
	// the Boolean. A parameterless Dispose() here would be the interface's
	// member leaking into public surface it does not occupy.
	disposeIdentities := 0
	for _, key := range manager.Members {
		if !strings.HasPrefix(key.Name, "Dispose") {
			continue
		}
		disposeIdentities++
		member := surface.Members[key]
		if !equalStrings(member.Parameters, []string{"bool"}) {
			t.Fatalf("GraphicsDeviceManager.%s takes %v; its CLR Dispose() is an explicit interface implementation and is not public surface",
				key.Name, member.Parameters)
		}
	}
	if disposeIdentities != 1 {
		t.Fatalf("GraphicsDeviceManager projects %d Dispose identities, want exactly the declared Dispose(bool)", disposeIdentities)
	}

	// No language adapter, and no projected type anywhere, is a disposal shape
	// invented from the interface.
	for name := range inventedDisposalNames {
		if adapterTypes[name] {
			t.Fatalf("%q is registered as a language adapter; System.IDisposable adds no Go type", name)
		}
		for key := range surface.Types {
			if key.Name == name {
				t.Fatalf("%q is a projected XNA type name, so it cannot also be a forbidden disposal shape", name)
			}
		}
		for key := range surface.Members {
			if key.Name == name {
				t.Fatalf("%q is a projected XNA member, so it cannot also be a forbidden disposal shape", name)
			}
		}
	}
}

// bclInterfaceDefects are the target-side defects the no-surface rule is
// negatively fixtured against.
var bclInterfaceDefects = []struct {
	Name     string
	Category string
	Apply    func(t *testing.T, expected *expectedSurface, actual *actualSurface, owner *expectedType)
}{
	{"invented-close-alias", "INTERFACE_MAPPING_MISMATCH", func(t *testing.T, e *expectedSurface, a *actualSurface, owner *expectedType) {
		key := symbolKey{Package: owner.Key.Package, Receiver: owner.GoName, Name: "Close"}
		a.Members[key] = &actualMember{Key: key, Kind: "method", Results: []string{"error"}}
	}},
	{"invented-disposable-interface-embedding", "INTERFACE_MAPPING_MISMATCH", func(t *testing.T, e *expectedSurface, a *actualSurface, owner *expectedType) {
		a.Types[owner.Key].ExportedEmbeddings = []string{"Disposable"}
	}},
	{"invented-qualified-disposable-embedding", "INTERFACE_MAPPING_MISMATCH", func(t *testing.T, e *expectedSurface, a *actualSurface, owner *expectedType) {
		a.Types[owner.Key].ExportedEmbeddings = []string{"framework.IDisposable"}
	}},
	{"invented-finalizer-surface", "INTERFACE_MAPPING_MISMATCH", func(t *testing.T, e *expectedSurface, a *actualSurface, owner *expectedType) {
		key := symbolKey{Package: owner.Key.Package, Receiver: owner.GoName, Name: "SetFinalizer"}
		a.Members[key] = &actualMember{Key: key, Kind: "method"}
	}},
	{"invented-ownership-wrapper", "INTERFACE_MAPPING_MISMATCH", func(t *testing.T, e *expectedSurface, a *actualSurface, owner *expectedType) {
		key := symbolKey{Package: owner.Key.Package, Receiver: owner.GoName, Name: "DisposeAll"}
		a.Members[key] = &actualMember{Key: key, Kind: "method"}
	}},
	{"undeclared-bcl-interface", "INTERFACE_MAPPING_MISMATCH", func(t *testing.T, e *expectedSurface, a *actualSurface, owner *expectedType) {
		owner.Interfaces = append(owner.Interfaces, "System.Something.Undecided")
	}},
	{"undeclared-internal-xna-interface", "INTERFACE_MAPPING_MISMATCH", func(t *testing.T, e *expectedSurface, a *actualSurface, owner *expectedType) {
		owner.Interfaces = append(owner.Interfaces, "Microsoft.Xna.Framework.Graphics.IUndecidedInternal")
	}},
	{"declared-dispose-dropped", "MISSING_MEMBER", func(t *testing.T, e *expectedSurface, a *actualSurface, owner *expectedType) {
		delete(a.Members, disposeMemberKey(t, owner))
	}},
	{"declared-dispose-fallibility-changed", "ERROR_MAPPING_MISMATCH", func(t *testing.T, e *expectedSurface, a *actualSurface, owner *expectedType) {
		key := disposeMemberKey(t, owner)
		member := a.Members[key]
		if len(member.Results) > 0 && member.Results[len(member.Results)-1] == "error" {
			member.Results = member.Results[:len(member.Results)-1]
			return
		}
		member.Results = append(member.Results, "error")
	}},
}

// TestBCLInterfaceRelationshipDefectsAreRejected attacks the no-surface rule on
// a type that declares Dispose publicly.
func TestBCLInterfaceRelationshipDefectsAreRejected(t *testing.T) {
	const identity = "Microsoft.Xna.Framework.Audio.SoundEffect"
	baseExpected, baseActual, _ := isolateTypeSurface(t, identity)
	baseline := verify(baseExpected, baseActual, 0, "report", "contract", "mapping")
	if baseline.Summary["TOTAL_DIAGNOSTICS"] != 0 {
		t.Fatalf("unmutated %s baseline is not clean: %v", identity, baseline.Diagnostics)
	}
	for _, defect := range bclInterfaceDefects {
		defect := defect
		t.Run(defect.Name, func(t *testing.T) {
			expected, actual, owner := isolateTypeSurface(t, identity)
			defect.Apply(t, expected, actual, owner)
			result := verify(expected, actual, 0, "report", "contract", "mapping")
			if result.Summary[defect.Category] == 0 {
				t.Fatalf("defect %q did not raise %s; summary=%v", defect.Name, defect.Category, result.Summary)
			}
		})
	}
}

// TestSyntheticDisposeIsRejected is the defect System.IDisposable would cause if
// it were treated as adding surface: a Dispose on a type whose XNA contract
// declares none.
func TestSyntheticDisposeIsRejected(t *testing.T) {
	// Curve implements no IDisposable and declares no Dispose.
	const identity = "Microsoft.Xna.Framework.Curve"
	expected, actual, owner := isolateTypeSurface(t, identity)
	baseline := verify(expected, actual, 0, "report", "contract", "mapping")
	if baseline.Summary["INTERFACE_MAPPING_MISMATCH"] != 0 {
		t.Fatalf("unmutated %s baseline is not clean: %v", identity, baseline.Diagnostics)
	}
	for _, key := range owner.Members {
		if strings.HasPrefix(key.Name, "Dispose") {
			t.Fatalf("%s already declares %s", identity, key.Name)
		}
	}
	expected, actual, owner = isolateTypeSurface(t, identity)
	key := symbolKey{Package: owner.Key.Package, Receiver: owner.GoName, Name: "Dispose"}
	actual.Members[key] = &actualMember{Key: key, Kind: "method", Results: []string{"error"}}
	result := verify(expected, actual, 0, "report", "contract", "mapping")
	if result.Summary["INTERFACE_MAPPING_MISMATCH"] == 0 {
		t.Fatalf("a synthesized Dispose was not rejected; summary=%v", result.Summary)
	}
}

// TestBCLInterfaceMeasurementIsReported proves every declared interface reaches
// the report with the no-surface arithmetic.
func TestBCLInterfaceMeasurementIsReported(t *testing.T) {
	expected, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	actual := &actualSurface{
		Types:       make(map[symbolKey]*actualType),
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: make(map[string]string),
		Packages:    make(map[string]*types.Package),
	}
	measurements := measureBCLInterfaceRelationships(expected, actual)
	want := len(bclInterfaceRelationships) + len(internalXNAInterfaces)
	if len(measurements) != want {
		t.Fatalf("measured %d interfaces, want %d", len(measurements), want)
	}
	for _, row := range measurements {
		if row.Verdict != "PASS" || row.ProjectedMembers != 0 {
			t.Fatalf("%s = %+v", row.CLRInterface, row)
		}
		if row.DeclaringTypes == 0 || row.CLRMembers == 0 {
			t.Fatalf("%s declares nothing: %+v", row.CLRInterface, row)
		}
	}
}

// bclInterfaceMutationCase applies one named BCL interface defect. Mutation ids
// have the form f24bcli_<defect>__<identity>.
func bclInterfaceMutationCase(t *testing.T, mutation string) (*expectedSurface, *actualSurface) {
	t.Helper()
	parts := strings.SplitN(strings.TrimPrefix(mutation, "f24bcli_"), "__", 2)
	if len(parts) != 2 {
		t.Fatalf("malformed BCL interface mutation %q", mutation)
	}
	expected, actual, owner := isolateTypeSurface(t, parts[1])
	if parts[0] == "synthetic_dispose" {
		key := symbolKey{Package: owner.Key.Package, Receiver: owner.GoName, Name: "Dispose"}
		actual.Members[key] = &actualMember{Key: key, Kind: "method", Results: []string{"error"}}
		return expected, actual
	}
	for _, defect := range bclInterfaceDefects {
		if defect.Name != parts[0] {
			continue
		}
		defect.Apply(t, expected, actual, owner)
		return expected, actual
	}
	t.Fatalf("unknown BCL interface defect %q", parts[0])
	return nil, nil
}

// bclCompositionFixture builds an isolated, initially correct expected/actual
// pair for one XNA type whose CLR base is a supported BCL family, including
// the private adapter field the composition projection requires and the
// Iterator<T> adapter the inherited GetEnumerator returns.
func bclCompositionFixture(t *testing.T, identity string) (*expectedSurface, *actualSurface, *expectedType) {
	t.Helper()
	expected, actual, owner := isolateTypeSurface(t, identity)
	adapter, composed := composedBaseAdapter(owner)
	if !composed {
		t.Fatalf("%s does not inherit a supported BCL base", identity)
	}
	actual.Types[owner.Key].Fields = []actualField{
		{Name: adapter.AdapterField, Type: adapterFieldType(adapter, owner.BaseType)},
		{Name: "componentAdded", Type: "EventSource[*GameComponentCollectionEventArgs]"},
	}
	iteratorKey := symbolKey{Package: modulePath + "/Microsoft/Xna/Framework", Name: "Iterator"}
	actual.Types[iteratorKey] = &actualType{Key: iteratorKey, Kind: "interface", TypeParameters: []string{"T"}}
	nextKey := symbolKey{Package: iteratorKey.Package, Receiver: "Iterator", Name: "Next"}
	actual.Members[nextKey] = &actualMember{Key: nextKey, Kind: "method", Results: []string{"T", "bool", "error"}}
	return expected, actual, owner
}

// inheritedMemberKey is the projected Go key of one inherited BCL member on
// one consumer, found by its CLR provenance rather than by its Go spelling, so
// a mutation names the CLR member it is attacking.
func inheritedMemberKey(t *testing.T, expected *expectedSurface, owner *expectedType, clrMember, accessor string) symbolKey {
	t.Helper()
	for _, key := range owner.Members {
		member := expected.Members[key]
		if member.BCLBase != "" && member.BCLMember == clrMember && member.Accessor == accessor {
			return key
		}
	}
	t.Fatalf("%s projects no inherited %s (accessor %q)", owner.XNA, clrMember, accessor)
	return symbolKey{}
}

// declaredMemberKey is the projected Go key of one XNA-DECLARED member, used to
// prove a declared member and an inherited one never share a provenance class.
func declaredMemberKey(t *testing.T, expected *expectedSurface, owner *expectedType, clrMember string) symbolKey {
	t.Helper()
	for _, key := range owner.Members {
		member := expected.Members[key]
		if member.BCLBase != "" {
			continue
		}
		name := member.XNA[strings.Index(member.XNA, "::")+2:]
		if open := strings.Index(name, "("); open >= 0 {
			name = name[:open]
		}
		if name == clrMember {
			return key
		}
	}
	t.Fatalf("%s declares no %s", owner.XNA, clrMember)
	return symbolKey{}
}

// bclCompositionMutationCase attacks the BCL base-class composition projection
// in every direction the architecture forbids.
func bclCompositionMutationCase(t *testing.T, mutation string) (*expectedSurface, *actualSurface) {
	t.Helper()
	parts := strings.SplitN(strings.TrimPrefix(mutation, "f26bcl_"), "__", 2)
	if len(parts) != 2 {
		t.Fatalf("malformed composition mutation %q", mutation)
	}
	expected, actual, owner := bclCompositionFixture(t, parts[1])
	adapter, _ := composedBaseAdapter(owner)
	switch parts[0] {
	case "base_silently_dropped":
		// A base nobody declared must never pass unnoticed.
		owner.BaseType = "System.Collections.ObjectModel.SomeOtherCollection`1[Microsoft.Xna.Framework.IGameComponent]"
	case "exported_embedding":
		// The refused shape: `type X struct { Collection[T] }`.
		actual.Types[owner.Key].ExportedEmbeddings = []string{"Collection[IGameComponent]"}
	case "unexported_embedding":
		// Embedding the private adapter still promotes a method set CNA-Go
		// never measured, so it is refused too.
		actual.Types[owner.Key].Fields = append(actual.Types[owner.Key].Fields,
			actualField{Name: "collectionBase", Type: adapterFieldType(adapter, owner.BaseType), Embedded: true})
	case "raw_slice_projection":
		// `type GameComponentCollection []IGameComponent` is not a struct.
		actual.Types[owner.Key].Kind = "other"
		actual.Types[owner.Key].Underlying = "[]IGameComponent"
	case "exported_raw_slice_field":
		actual.Types[owner.Key].Fields = append(actual.Types[owner.Key].Fields,
			actualField{Name: "Items", Type: "[]IGameComponent", Exported: true})
	case "exported_raw_map_field":
		actual.Types[owner.Key].Fields = append(actual.Types[owner.Key].Fields,
			actualField{Name: "Entries", Type: "map[string]string", Exported: true})
	case "missing_adapter_field":
		actual.Types[owner.Key].Fields = nil
	case "exported_adapter_field":
		actual.Types[owner.Key].Fields[0].Exported = true
	case "wrong_adapter_generic":
		actual.Types[owner.Key].Fields[0].Type = "collectionBase[IUpdateable]"
	case "wrong_adapter_family":
		actual.Types[owner.Key].Fields[0].Type = "[]IGameComponent"
	case "inherited_member_missing":
		delete(actual.Members, inheritedMemberKey(t, expected, owner, "Add", ""))
	case "inherited_indexer_setter_missing":
		delete(actual.Members, inheritedMemberKey(t, expected, owner, "Item", "set"))
	case "extra_inherited_member":
		key := symbolKey{Package: owner.Key.Package, Receiver: owner.GoName, Name: "AddRange"}
		actual.Members[key] = &actualMember{Key: key, Kind: "method", Parameters: []string{"[]IGameComponent"}, Results: []string{"error"}}
	case "excluded_member_projected":
		// IsReadOnly is a private explicit implementation on Collection<T>,
		// so promoting it invents surface the CLR does not expose.
		key := symbolKey{Package: owner.Key.Package, Receiver: owner.GoName, Name: "IsReadOnly"}
		actual.Members[key] = &actualMember{Key: key, Kind: "method", Results: []string{"bool"}}
	case "wrong_count_projection":
		actual.Members[inheritedMemberKey(t, expected, owner, "Count", "get")].Results = []string{"int"}
	case "wrong_indexer_projection":
		actual.Members[inheritedMemberKey(t, expected, owner, "Item", "get")].Results = []string{"IGameComponent"}
	case "wrong_enumerator_type":
		actual.Members[inheritedMemberKey(t, expected, owner, "GetEnumerator", "")].Results = []string{"[]IGameComponent"}
	case "enumerator_adapter_absent":
		delete(actual.Types, symbolKey{Package: modulePath + "/Microsoft/Xna/Framework", Name: "Iterator"})
	case "infallible_mutator":
		// Add reaches a hook that rejects duplicates, so dropping its error
		// result hides a real failure mode.
		actual.Members[inheritedMemberKey(t, expected, owner, "Add", "")].Results = nil
	case "infallible_indexer_setter":
		actual.Members[inheritedMemberKey(t, expected, owner, "Item", "set")].Results = nil
	case "fallible_reader":
		actual.Members[inheritedMemberKey(t, expected, owner, "IndexOf", "")].Results = []string{"int32", "error"}
	case "declared_override_missing":
		delete(actual.Members, declaredMemberKey(t, expected, owner, "InsertItem"))
	default:
		t.Fatalf("unknown composition defect %q", parts[0])
	}
	return expected, actual
}

// TestBCLBaseCompositionIsMeasuredNotAssumed proves the registry describes the
// pinned reality rather than restating the implementation.
func TestBCLBaseCompositionIsMeasuredNotAssumed(t *testing.T) {
	surface, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	// Every COMPOSED base must have an adapter, and every adapter must belong
	// to a COMPOSED base: neither half can drift.
	for identity, relationship := range bclBaseRelationships {
		_, hasAdapter := bclBaseAdapters[identity]
		if (relationship.Status == "COMPOSED") != hasAdapter {
			t.Fatalf("%s has status %q but adapter presence %v", identity, relationship.Status, hasAdapter)
		}
	}
	for identity, adapter := range bclBaseAdapters {
		if adapter.GoAdapter == "" || adapter.AdapterField == "" || adapter.Authority == "" || adapter.AuthoritySHA256 == "" {
			t.Fatalf("%s adapter is under-specified: %+v", identity, adapter)
		}
		if strings.ToUpper(adapter.GoAdapter[:1]) == adapter.GoAdapter[:1] {
			t.Fatalf("%s adapter %q must be unexported", identity, adapter.GoAdapter)
		}
		if strings.ToUpper(adapter.AdapterField[:1]) == adapter.AdapterField[:1] {
			t.Fatalf("%s adapter field %q must be unexported", identity, adapter.AdapterField)
		}
		if len(adapter.Members) == 0 || len(adapter.Excluded) == 0 {
			t.Fatalf("%s must inventory both its projected and its excluded members", identity)
		}
		seen := make(map[string]bool)
		for _, entry := range adapter.Members {
			if entry.Rationale == "" {
				t.Fatalf("%s::%s has no recorded rationale", identity, entry.Member.Name)
			}
			key := entry.Member.Kind + "|" + entry.Member.Name
			if seen[key] {
				t.Fatalf("%s inventories %s twice", identity, key)
			}
			seen[key] = true
		}
		for _, excluded := range adapter.Excluded {
			if excluded.Reason == "" {
				t.Fatalf("%s excludes %s with no reason", identity, excluded.CLRMember)
			}
		}
	}

	owner := surface.typeForXNA("Microsoft.Xna.Framework.GameComponentCollection")
	if owner.SourceMembers != 7 || owner.BCLInheritedCLRMembers != 11 || owner.BCLInheritedProjections != 12 {
		t.Fatalf("GameComponentCollection accounting = %d declared/%d inherited CLR/%d inherited projections",
			owner.SourceMembers, owner.BCLInheritedCLRMembers, owner.BCLInheritedProjections)
	}
	if len(owner.Members) != 21 {
		t.Fatalf("GameComponentCollection projects %d Go members, want 21", len(owner.Members))
	}
	// The collision between the declared protected SetItem override and the
	// inherited Item setter is resolved by the settled rule, not by a bespoke
	// exception, so both colliders carry their source-kind suffix.
	setter := surface.Members[inheritedMemberKey(t, surface, owner, "Item", "set")]
	if setter.GoName != "SetItemProperty" {
		t.Fatalf("inherited indexer setter = %q, want SetItemProperty", setter.GoName)
	}
	override := surface.Members[declaredMemberKey(t, surface, owner, "SetItem")]
	if override.GoName != "SetItemMethod" {
		t.Fatalf("declared SetItem override = %q, want SetItemMethod", override.GoName)
	}
	if setter.BCLBase != "System.Collections.ObjectModel.Collection`1" || override.BCLBase != "" {
		t.Fatalf("provenance = setter %q, override %q", setter.BCLBase, override.BCLBase)
	}
	// The inherited getter keeps the plain name because nothing collides with
	// it, and it carries the bounds failure List<T> owns.
	getter := surface.Members[inheritedMemberKey(t, surface, owner, "Item", "get")]
	if getter.GoName != "Item" || !equalStrings(getter.Results, []string{"IGameComponent", "error"}) {
		t.Fatalf("inherited indexer getter = %q %v", getter.GoName, getter.Results)
	}
	// The read-only members gain no error result.
	for _, clrMember := range []string{"Count", "Contains", "IndexOf", "GetEnumerator"} {
		accessor := ""
		if clrMember == "Count" {
			accessor = "get"
		}
		member := surface.Members[inheritedMemberKey(t, surface, owner, clrMember, accessor)]
		for _, result := range member.Results {
			if result == "error" {
				t.Fatalf("inherited %s must not gain an error result: %v", clrMember, member.Results)
			}
		}
	}
	// Every mutator does.
	for _, clrMember := range []string{"Add", "Clear", "CopyTo", "Insert", "Remove", "RemoveAt"} {
		member := surface.Members[inheritedMemberKey(t, surface, owner, clrMember, "")]
		if len(member.Results) == 0 || member.Results[len(member.Results)-1] != "error" {
			t.Fatalf("inherited %s must be fallible: %v", clrMember, member.Results)
		}
	}
}

// TestBCLBaseCompositionDefectsAreRejected runs the composition negative
// controls through the real verifier.
func TestBCLBaseCompositionDefectsAreRejected(t *testing.T) {
	cases := []struct {
		mutation string
		category string
	}{
		{"base_silently_dropped", "BASE_MAPPING_MISMATCH"},
		{"exported_embedding", "BASE_MAPPING_MISMATCH"},
		{"unexported_embedding", "BASE_MAPPING_MISMATCH"},
		{"raw_slice_projection", "BASE_MAPPING_MISMATCH"},
		{"exported_raw_slice_field", "BASE_MAPPING_MISMATCH"},
		{"exported_raw_map_field", "BASE_MAPPING_MISMATCH"},
		{"missing_adapter_field", "BASE_MAPPING_MISMATCH"},
		{"exported_adapter_field", "BASE_MAPPING_MISMATCH"},
		{"wrong_adapter_generic", "BASE_MAPPING_MISMATCH"},
		{"wrong_adapter_family", "BASE_MAPPING_MISMATCH"},
		{"excluded_member_projected", "BASE_MAPPING_MISMATCH"},
		{"inherited_member_missing", "MISSING_MEMBER"},
		{"inherited_indexer_setter_missing", "MISSING_MEMBER"},
		{"declared_override_missing", "MISSING_MEMBER"},
		{"extra_inherited_member", "UNEXPECTED_MEMBER"},
		{"wrong_count_projection", "RETURN_MAPPING_MISMATCH"},
		{"wrong_indexer_projection", "RETURN_MAPPING_MISMATCH"},
		{"wrong_enumerator_type", "RETURN_MAPPING_MISMATCH"},
		{"enumerator_adapter_absent", "INTERFACE_MAPPING_MISMATCH"},
		{"infallible_mutator", "ERROR_MAPPING_MISMATCH"},
		{"infallible_indexer_setter", "ERROR_MAPPING_MISMATCH"},
		{"fallible_reader", "ERROR_MAPPING_MISMATCH"},
	}
	for _, testCase := range cases {
		t.Run(testCase.mutation, func(t *testing.T) {
			mutation := "f26bcl_" + testCase.mutation + "__Microsoft.Xna.Framework.GameComponentCollection"
			expected, actual := bclCompositionMutationCase(t, mutation)
			result := verify(expected, actual, 0, "report", "contract", "mapping")
			if result.Summary[testCase.category] == 0 {
				t.Fatalf("mutation %q did not trigger %s; summary=%v", mutation, testCase.category, result.Summary)
			}
		})
	}
}

// TestBCLCompositionFixtureIsCleanBeforeMutation is the control the negative
// controls depend on: the unmutated fixture must produce no diagnostic, so a
// mutation's diagnostic is caused by the mutation.
func TestBCLCompositionFixtureIsCleanBeforeMutation(t *testing.T) {
	expected, actual, _ := bclCompositionFixture(t, "Microsoft.Xna.Framework.GameComponentCollection")
	result := verify(expected, actual, 0, "report", "contract", "mapping")
	if result.Summary["TOTAL_DIAGNOSTICS"] != 0 {
		t.Fatalf("clean composition fixture produced %d diagnostics: %v", result.Summary["TOTAL_DIAGNOSTICS"], result.Diagnostics)
	}
}

// bclSignatureAdapterFixture builds an isolated, initially correct pair for the
// BCL signature adapters: the framework package with each declared adapter type
// and its exact public member set present, and nothing else.
func bclSignatureAdapterFixture(t *testing.T) (*expectedSurface, *actualSurface) {
	t.Helper()
	frameworkPackage := modulePath + "/Microsoft/Xna/Framework"
	expected := &expectedSurface{
		Types:              make(map[symbolKey]*expectedType),
		Members:            make(map[symbolKey]*expectedMember),
		InterfaceWitnesses: make(map[symbolKey]*expectedInterfaceWitness),
	}
	actual := &actualSurface{
		Types:       make(map[symbolKey]*actualType),
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: map[string]string{frameworkPackage: "Microsoft/Xna/Framework"},
		Packages:    make(map[string]*types.Package),
	}
	seedSignatureAdapters(actual)
	return expected, actual
}

func bclSignatureAdapterMutationCase(t *testing.T, mutation string) (*expectedSurface, *actualSurface) {
	t.Helper()
	expected, actual := bclSignatureAdapterFixture(t)
	frameworkPackage := modulePath + "/Microsoft/Xna/Framework"
	adapter := bclSignatureAdapters["System.Collections.ObjectModel.ReadOnlyCollection`1"]
	goName := bclSignatureAdapterGoName(adapter)
	switch strings.TrimPrefix(mutation, "f27sig_") {
	case "adapter_type_absent":
		delete(actual.Types, symbolKey{Package: frameworkPackage, Name: goName})
	case "public_member_missing":
		delete(actual.Members, symbolKey{Package: frameworkPackage, Receiver: goName, Name: "IndexOf"})
	case "enumerator_missing":
		delete(actual.Members, symbolKey{Package: frameworkPackage, Receiver: goName, Name: "GetEnumerator"})
	case "extra_public_member":
		key := symbolKey{Package: frameworkPackage, Receiver: goName, Name: "Reverse"}
		actual.Members[key] = &actualMember{Key: key, Kind: "method"}
	case "read_only_made_mutable_add":
		// Promoting an explicitly implemented mutator would make a read-only
		// view writable, which is the whole point of the type.
		key := symbolKey{Package: frameworkPackage, Receiver: goName, Name: "Add"}
		actual.Members[key] = &actualMember{Key: key, Kind: "method"}
	case "read_only_made_mutable_setter":
		key := symbolKey{Package: frameworkPackage, Receiver: goName, Name: "SetItem"}
		actual.Members[key] = &actualMember{Key: key, Kind: "method"}
	case "read_only_made_mutable_clear":
		key := symbolKey{Package: frameworkPackage, Receiver: goName, Name: "Clear"}
		actual.Members[key] = &actualMember{Key: key, Kind: "method"}
	case "excluded_member_promoted":
		key := symbolKey{Package: frameworkPackage, Receiver: goName, Name: "Items"}
		actual.Members[key] = &actualMember{Key: key, Kind: "method"}
	case "sync_root_promoted":
		key := symbolKey{Package: frameworkPackage, Receiver: goName, Name: "SyncRoot"}
		actual.Members[key] = &actualMember{Key: key, Kind: "method"}
	default:
		t.Fatalf("unknown signature adapter defect %q", mutation)
	}
	return expected, actual
}

// TestBCLSignatureAdapterFixtureIsCleanBeforeMutation is the control the
// signature-adapter negative controls depend on.
func TestBCLSignatureAdapterFixtureIsCleanBeforeMutation(t *testing.T) {
	expected, actual := bclSignatureAdapterFixture(t)
	result := report{Summary: make(map[string]int)}
	measureBCLSignatureAdapters(&result, expected, actual)
	if result.Summary["LANGUAGE_MAPPING_MISMATCH"] != 0 {
		t.Fatalf("clean signature adapter fixture produced %d diagnostics: %v",
			result.Summary["LANGUAGE_MAPPING_MISMATCH"], result.Diagnostics)
	}
}

// TestBCLSignatureAdapterDefectsAreRejected attacks the signature-adapter
// projection, including every way a read-only view could be made mutable.
func TestBCLSignatureAdapterDefectsAreRejected(t *testing.T) {
	mutations := []string{
		"adapter_type_absent",
		"public_member_missing",
		"enumerator_missing",
		"extra_public_member",
		"read_only_made_mutable_add",
		"read_only_made_mutable_setter",
		"read_only_made_mutable_clear",
		"excluded_member_promoted",
		"sync_root_promoted",
	}
	for _, mutation := range mutations {
		t.Run(mutation, func(t *testing.T) {
			expected, actual := bclSignatureAdapterMutationCase(t, "f27sig_"+mutation)
			result := report{Summary: make(map[string]int)}
			measureBCLSignatureAdapters(&result, expected, actual)
			if result.Summary["LANGUAGE_MAPPING_MISMATCH"] == 0 {
				t.Fatalf("mutation %q did not trigger LANGUAGE_MAPPING_MISMATCH", mutation)
			}
		})
	}
}

// TestBCLSignatureAdaptersAreMeasuredNotAssumed proves the signature-adapter
// registry describes the pinned reality, and that the two adapter roles stay
// distinct.
func TestBCLSignatureAdaptersAreMeasuredNotAssumed(t *testing.T) {
	for identity, adapter := range bclSignatureAdapters {
		if adapter.Authority == "" || adapter.AuthoritySHA256 == "" || adapter.Rationale == "" {
			t.Fatalf("%s signature adapter is under-specified", identity)
		}
		if len(adapter.Members) == 0 || len(adapter.Excluded) == 0 {
			t.Fatalf("%s must inventory both its projected and its excluded members", identity)
		}
		goName := bclSignatureAdapterGoName(adapter)
		if strings.ToUpper(goName[:1]) != goName[:1] {
			t.Fatalf("%s adapter %q must be exported: a signature adapter is a public type", identity, goName)
		}
		if !adapterTypes[goName] {
			t.Fatalf("%s adapter %q is not admitted as a language adapter", identity, goName)
		}
		for _, entry := range adapter.Members {
			if entry.Rationale == "" {
				t.Fatalf("%s::%s has no recorded rationale", identity, entry.Member.Name)
			}
			if entry.Member.Kind == "property" && entry.Member.Set {
				t.Fatalf("%s::%s declares a public setter, which no read-only view has", identity, entry.Member.Name)
			}
		}
		for _, excluded := range adapter.Excluded {
			if excluded.Reason == "" {
				t.Fatalf("%s excludes %s with no reason", identity, excluded.CLRMember)
			}
		}
		// The two adapter roles are distinct. ReadOnlyCollection<T> is a
		// signature adapter AND a deferred base, and neither fact implies the
		// other.
		if relationship, declared := bclBaseRelationships[identity]; declared && relationship.Status == "COMPOSED" {
			if _, alsoBase := bclBaseAdapters[identity]; !alsoBase {
				t.Fatalf("%s is COMPOSED as a base but declares no base adapter", identity)
			}
		}
	}
	// The read-only base relationship is still deferred, so no XNA type may
	// derive from it yet even though the signature projection exists.
	relationship := bclBaseRelationships["System.Collections.ObjectModel.ReadOnlyCollection`1"]
	if relationship.Status != "DEFERRED" {
		t.Fatalf("ReadOnlyCollection base status = %q, want DEFERRED", relationship.Status)
	}
	if _, isBaseAdapter := bclBaseAdapters["System.Collections.ObjectModel.ReadOnlyCollection`1"]; isBaseAdapter {
		t.Fatal("a DEFERRED base must not declare a base adapter")
	}
}

// TestEveryDeferredBaseNamesItsBlockers turns "DEFERRED" from a status word
// into a measured claim.
//
// A base nobody has decided about is already a diagnostic. This adds the other
// half: a base somebody has explicitly deferred must say what blocks it, named
// to the exact inherited member or the exact architecture decision, so the
// frontier is an inventory rather than a shrug.
func TestEveryDeferredBaseNamesItsBlockers(t *testing.T) {
	for identity, relationship := range bclBaseRelationships {
		switch relationship.Status {
		case "IMPLIED", "MAPPED", "COMPOSED":
			if len(relationship.Blockers) != 0 {
				t.Fatalf("%s is %s but records blockers", identity, relationship.Status)
			}
			continue
		case "DEFERRED":
		default:
			t.Fatalf("%s has unknown status %q", identity, relationship.Status)
		}
		if len(relationship.Blockers) == 0 {
			t.Fatalf("%s is DEFERRED but names nothing that blocks it", identity)
		}
		for _, blocker := range relationship.Blockers {
			if blocker.Needs == "" || blocker.Detail == "" {
				t.Fatalf("%s has an under-specified blocker: %+v", identity, blocker)
			}
			switch blocker.Kind {
			case "SUBSYSTEM", "ARCHITECTURE":
			default:
				t.Fatalf("%s blocker has unknown kind %q", identity, blocker.Kind)
			}
		}
	}
}

// TestDeferredBaseWithoutBlockersIsRejected is the negative control for the
// claim above, run through the real measurement rather than the table.
func TestDeferredBaseWithoutBlockersIsRejected(t *testing.T) {
	identity := "System.Exception"
	original := bclBaseRelationships[identity]
	t.Cleanup(func() { bclBaseRelationships[identity] = original })

	stripped := original
	stripped.Blockers = nil
	bclBaseRelationships[identity] = stripped

	expected, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	actual := &actualSurface{
		Types: make(map[symbolKey]*actualType), Members: make(map[symbolKey]*actualMember),
		PackageDirs: make(map[string]string), Packages: make(map[string]*types.Package),
	}
	measurements := measureBCLBaseRelationships(expected, actual)
	for _, row := range measurements {
		if row.CLRBase == identity && row.Verdict != "FAIL" {
			t.Fatalf("a DEFERRED base with no blockers must FAIL, got %q", row.Verdict)
		}
	}
}

// TestSystemExceptionAuditIsRecorded pins the specific findings of the
// directed System.Exception audit, so a later edit cannot quietly soften them.
func TestSystemExceptionAuditIsRecorded(t *testing.T) {
	relationship := bclBaseRelationships["System.Exception"]
	needs := make(map[string]string, len(relationship.Blockers))
	architecture := 0
	for _, blocker := range relationship.Blockers {
		if blocker.Kind == "ARCHITECTURE" {
			architecture++
			continue
		}
		needs[blocker.CLRMember] = blocker.Needs
	}
	if architecture < 2 {
		t.Fatalf("the audit found two architecture obstacles, the table records %d", architecture)
	}
	for member, subsystem := range map[string]string{
		"Data":          "System.Collections.IDictionary",
		"TargetSite":    "System.Reflection.MethodBase",
		"GetObjectData": "System.Runtime.Serialization",
	} {
		if needs[member] != subsystem {
			t.Fatalf("System.Exception::%s must be recorded as needing %s, got %q", member, subsystem, needs[member])
		}
	}
	// The eight derived types declare constructors and nothing else, which is
	// why the whole question is about inherited surface.
	reference := loadPinnedContract(t)
	derived := 0
	for _, declared := range reference.Types {
		if declared.BaseType == nil {
			continue
		}
		base := baseIdentityWithoutArguments(*declared.BaseType)
		if base != "System.Exception" && base != "System.Runtime.InteropServices.ExternalException" {
			continue
		}
		derived++
		for _, member := range declared.Members {
			if member.Kind != "constructor" {
				t.Fatalf("%s declares a non-constructor member %s, which the audit did not account for", declared.Name, member.Name)
			}
		}
	}
	if derived != 8 {
		t.Fatalf("the audit covered 8 derived types, the contract has %d", derived)
	}
}

// ---------------------------------------------------------------------------
// Foundation 31 — negative controls for the Game base-call language adapters.
// ---------------------------------------------------------------------------

// gameBaseCallFixture builds a surface pair that exercises the base-call family
// and nothing else about it is clean: the expected surface is the whole pinned
// contract, so Game is present and its five callback members exist, while the
// actual surface carries only the framework package and the five adapters.
//
// Every defect below is measured as a DELTA against this fixture's own
// baseline, so the fixture's unrelated MISSING_TYPE noise cannot make a defect
// look detected.
func gameBaseCallFixture(t *testing.T) (*expectedSurface, *actualSurface) {
	t.Helper()
	expected, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	actual := &actualSurface{
		Types:       make(map[symbolKey]*actualType),
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: map[string]string{modulePath + "/Microsoft/Xna/Framework": "Microsoft/Xna/Framework"},
		Packages:    make(map[string]*types.Package),
	}
	// The framework package always carries the BCL signature adapters too, so
	// they are seeded as well: an incomplete fixture would report their
	// absence and drown the defect this test is about. The same holds for the
	// members the Foundation-36 registries name -- Game's event accessors, its
	// raise sites and its four frame hooks all live in this package.
	seedSignatureAdapters(actual)
	seedGameBaseCallAdapters(actual)
	seedGameSignalMembers(t, expected, actual)
	seedGameCallbacksMembers(actual)
	actual.Packages[modulePath+"/Microsoft/Xna/Framework"] = frameHookCapabilityPackage()
	return expected, actual
}

// seedGameCallbacksMembers puts the five mandatory override members on the
// fixture's actual surface, so the contract's size is a measured five before
// any mutation touches it.
func seedGameCallbacksMembers(actual *actualSurface) {
	frameworkPackage := modulePath + "/Microsoft/Xna/Framework"
	shapes := map[string]struct {
		parameters []string
		results    []string
	}{
		"Initialize":    {[]string{"*Game"}, []string{"error"}},
		"LoadContent":   {[]string{"*Game"}, []string{"error"}},
		"Update":        {[]string{"*Game", "GameTime"}, []string{"error"}},
		"Draw":          {[]string{"*Game", "GameTime"}, []string{"error"}},
		"UnloadContent": {[]string{"*Game"}, []string{"error"}},
	}
	for name, shape := range shapes {
		key := symbolKey{Package: frameworkPackage, Receiver: "GameCallbacks", Name: name}
		actual.Members[key] = &actualMember{Key: key, Kind: "method", Parameters: shape.parameters, Results: shape.results}
	}
	actual.Types[symbolKey{Package: frameworkPackage, Name: "GameCallbacks"}] = &actualType{
		Key: symbolKey{Package: frameworkPackage, Name: "GameCallbacks"}, Kind: "interface",
	}
	actual.Types[symbolKey{Package: frameworkPackage, Name: "Game"}] = &actualType{
		Key: symbolKey{Package: frameworkPackage, Name: "Game"}, Kind: "struct",
		Fields: []actualField{
			{Name: "callbacks", Type: "GameCallbacks"},
			{Name: "beginDrawOverride", Type: "gameBeginDrawOverride"},
		},
	}
}

// frameHookCapabilityPackage builds the compiler evidence the capability
// measurement reads: a framework package carrying Game and the four unexported
// single-method override contracts, exactly as the real package declares them.
//
// It is constructed rather than parsed so a defect can rebuild it with one
// capability deliberately wrong -- bundled, exported, unexported-method, or
// absent -- and prove the measurement catches each.
func frameHookCapabilityPackage() *types.Package {
	return frameHookCapabilityPackageWith(nil)
}

// frameHookCapabilityPackageWith rebuilds the fixture package with one
// capability replaced by a deliberately wrong shape.
func frameHookCapabilityPackageWith(mutate func(pkg *types.Package, game *types.Named)) *types.Package {
	path := modulePath + "/Microsoft/Xna/Framework"
	pkg := types.NewPackage(path, "framework")
	gameName := types.NewTypeName(token.NoPos, pkg, "Game", nil)
	game := types.NewNamed(gameName, types.NewStruct(nil, nil), nil)
	pkg.Scope().Insert(gameName)
	// The mutation runs FIRST because a package scope keeps the first
	// declaration of a name: whatever a defect declares here is what the four
	// canonical declarations below then fail to replace, which is exactly the
	// substitution the defect is trying to make.
	if mutate != nil {
		mutate(pkg, game)
	}
	declareFrameHookCapability(pkg, game, "gameBeginRunOverride", "BeginRun", errorInterfaceType())
	declareFrameHookCapability(pkg, game, "gameEndRunOverride", "EndRun", errorInterfaceType())
	declareFrameHookCapability(pkg, game, "gameBeginDrawOverride", "BeginDraw", types.Typ[types.Bool], errorInterfaceType())
	declareFrameHookCapability(pkg, game, "gameEndDrawOverride", "EndDraw", errorInterfaceType())
	pkg.MarkComplete()
	return pkg
}

func errorInterfaceType() types.Type { return types.Universe.Lookup("error").Type() }

// declareFrameHookCapability inserts one single-method interface taking *Game
// and returning the given results, replacing any earlier declaration of the
// same name so a defect can overwrite one.
func declareFrameHookCapability(pkg *types.Package, game *types.Named, name, method string, results ...types.Type) {
	signature := frameHookCapabilitySignature(pkg, game, results...)
	function := types.NewFunc(token.NoPos, pkg, method, signature)
	insertFrameHookInterface(pkg, name, function)
}

func frameHookCapabilitySignature(pkg *types.Package, game *types.Named, results ...types.Type) *types.Signature {
	parameters := types.NewTuple(types.NewVar(token.NoPos, pkg, "game", types.NewPointer(game)))
	resultVars := make([]*types.Var, 0, len(results))
	for _, result := range results {
		resultVars = append(resultVars, types.NewVar(token.NoPos, pkg, "", result))
	}
	return types.NewSignatureType(nil, nil, nil, parameters, types.NewTuple(resultVars...), false)
}

func insertFrameHookInterface(pkg *types.Package, name string, methods ...*types.Func) {
	contract := types.NewInterfaceType(methods, nil)
	contract.Complete()
	typeName := types.NewTypeName(token.NoPos, pkg, name, nil)
	types.NewNamed(typeName, contract, nil)
	// Insert keeps the first declaration of a name, which is what lets a defect
	// declare a wrong-shaped capability before the canonical one is offered.
	pkg.Scope().Insert(typeName)
}

// seedGameBaseCallAdapters puts the exact declared adapters into an actual
// surface, so a fixture starts VALID and a mutation's diagnostic is caused by
// the mutation.
func seedGameBaseCallAdapters(actual *actualSurface) {
	frameworkPackage := modulePath + "/Microsoft/Xna/Framework"
	for _, adapter := range gameBaseCallAdapters {
		key := symbolKey{Package: frameworkPackage, Name: adapter.GoFunction}
		actual.Members[key] = &actualMember{
			Key: key, Kind: "func",
			Parameters: append([]string(nil), adapter.Parameters...),
			Results:    append([]string(nil), adapter.Results...),
		}
	}
}

// withGameBaseCallAdapters runs fn with the registry mutated and restores it.
func withGameBaseCallAdapters(t *testing.T, mutate func(), fn func()) {
	t.Helper()
	saved := make(map[string]gameBaseCallAdapter, len(gameBaseCallAdapters))
	for name, adapter := range gameBaseCallAdapters {
		saved[name] = adapter
	}
	defer func() { gameBaseCallAdapters = saved }()
	mutate()
	fn()
}

// gameBaseCallDefects is the shared table behind both the named test and the
// mutation inventory, so the two cannot drift. Each entry mutates either the
// registry or the actual surface and must raise LANGUAGE_MAPPING_MISMATCH.
var gameBaseCallDefects = map[string]func(actual *actualSurface){
	// The helper simply is not there, so a callback cannot reach its base.
	"adapter_absent_from_the_package": func(actual *actualSurface) {
		delete(actual.Members, symbolKey{Package: modulePath + "/Microsoft/Xna/Framework", Name: "GameBaseUpdate"})
	},
	// A helper projected as a method on Game would put a name Microsoft never
	// declared onto Game's projected member surface.
	"adapter_projected_as_a_method_on_game": func(actual *actualSurface) {
		key := symbolKey{Package: modulePath + "/Microsoft/Xna/Framework", Name: "GameBaseUpdate"}
		actual.Members[key].Kind = "method"
	},
	// Dropping the *Game parameter would make the helper operate on state it
	// was not handed, which is not what a CLR base call does.
	"adapter_signature_drops_the_game_parameter": func(actual *actualSurface) {
		key := symbolKey{Package: modulePath + "/Microsoft/Xna/Framework", Name: "GameBaseUpdate"}
		actual.Members[key].Parameters = []string{"GameTime"}
	},
	// A result the registry does not declare.
	"adapter_signature_gains_a_result": func(actual *actualSurface) {
		key := symbolKey{Package: modulePath + "/Microsoft/Xna/Framework", Name: "GameBaseDraw"}
		actual.Members[key].Results = []string{"bool", "error"}
	},
	// An exported base-call helper nobody declared and therefore nobody
	// measured.
	"arbitrary_extra_base_helper": func(actual *actualSurface) {
		key := symbolKey{Package: modulePath + "/Microsoft/Xna/Framework", Name: "GameBaseTick"}
		actual.Members[key] = &actualMember{Key: key, Kind: "func", Parameters: []string{"*Game"}, Results: []string{"error"}}
	},
	// A supported virtual left with no way to reach its base.
	"adapter_missing_for_a_supported_virtual": func(actual *actualSurface) {
		delete(gameBaseCallAdapters, "Draw")
		delete(actual.Members, symbolKey{Package: modulePath + "/Microsoft/Xna/Framework", Name: "GameBaseDraw"})
	},
	// An adapter for a member GameCallbacks does not project. Game::Run is
	// public and non-virtual, so it has no base a CLR override could call.
	"adapter_declared_for_a_member_that_is_not_a_supported_virtual": func(actual *actualSurface) {
		gameBaseCallAdapters["Run"] = gameBaseCallAdapter{
			CLRMember: "Microsoft.Xna.Framework.Game::Run", GoFunction: "GameBaseRun",
			Parameters: []string{"*Game"}, Results: []string{"error"},
			Fallibility:   []gameBaseCallFallibility{{Kind: "GUARD", Reason: "invented"}},
			ReferenceBody: []string{"invented"},
		}
		key := symbolKey{Package: modulePath + "/Microsoft/Xna/Framework", Name: "GameBaseRun"}
		actual.Members[key] = &actualMember{Key: key, Kind: "func", Parameters: []string{"*Game"}, Results: []string{"error"}}
	},
	// An error result nobody justified is a synthetic failure mode.
	"error_result_with_no_recorded_reason": func(actual *actualSurface) {
		adapter := gameBaseCallAdapters["Update"]
		adapter.Fallibility = nil
		gameBaseCallAdapters["Update"] = adapter
	},
	// A fallibility reason with no error result claims something the
	// signature does not.
	"recorded_reason_with_no_error_result": func(actual *actualSurface) {
		adapter := gameBaseCallAdapters["Draw"]
		adapter.Results = nil
		gameBaseCallAdapters["Draw"] = adapter
		key := symbolKey{Package: modulePath + "/Microsoft/Xna/Framework", Name: "GameBaseDraw"}
		actual.Members[key].Results = nil
	},
	// A deferred reference step that records nothing is exactly the defect the
	// deferred-BCL-base rule already forbids.
	"deferred_step_records_no_reason": func(actual *actualSurface) {
		adapter := gameBaseCallAdapters["Update"]
		adapter.Deferred = []gameBaseCallDeferral{{Step: "FrameworkDispatcher.Update()", Class: "SUBSYSTEM"}}
		gameBaseCallAdapters["Update"] = adapter
	},
	// An unclassified deferral hides which kind of gap it is.
	"deferred_step_unclassified": func(actual *actualSurface) {
		adapter := gameBaseCallAdapters["Update"]
		adapter.Deferred = []gameBaseCallDeferral{{Step: "FrameworkDispatcher.Update()", Class: "LATER", Reason: "not now"}}
		gameBaseCallAdapters["Update"] = adapter
	},
	// A deferral that IS observable is a stop condition, not a deferral: it
	// means the projection changed what a consumer can see.
	"deferred_step_marked_observable": func(actual *actualSurface) {
		adapter := gameBaseCallAdapters["Initialize"]
		deferred := append([]gameBaseCallDeferral(nil), adapter.Deferred...)
		deferred[0].Observable = true
		adapter.Deferred = deferred
		gameBaseCallAdapters["Initialize"] = adapter
	},
	// Without a recorded reference body there is nothing to check the
	// projection against.
	"adapter_records_no_reference_body": func(actual *actualSurface) {
		adapter := gameBaseCallAdapters["LoadContent"]
		adapter.ReferenceBody = nil
		gameBaseCallAdapters["LoadContent"] = adapter
	},
}

// TestGameBaseCallDefectsAreRejected attacks every rule the base-call family
// rests on. The family is language support, so the danger is not a wrong XNA
// signature but an INVENTED one: a helper nobody declared, a helper for a
// member that has no base, a silent deferral, or a synthetic error.
func TestGameBaseCallDefectsAreRejected(t *testing.T) {
	baselineExpected, baselineActual := gameBaseCallFixture(t)
	baseline := verify(baselineExpected, baselineActual, 0, "report", "contract", "mapping")
	if baseline.Summary["LANGUAGE_MAPPING_MISMATCH"] != 0 {
		t.Fatalf("the unmutated base-call fixture is not clean: %d LANGUAGE_MAPPING_MISMATCH", baseline.Summary["LANGUAGE_MAPPING_MISMATCH"])
	}
	if baseline.Summary["GAME_BASE_CALL_ADAPTERS"] != len(gameBaseCallAdapters) {
		t.Fatalf("fixture measured %d adapters, registry declares %d",
			baseline.Summary["GAME_BASE_CALL_ADAPTERS"], len(gameBaseCallAdapters))
	}

	names := make([]string, 0, len(gameBaseCallDefects))
	for name := range gameBaseCallDefects {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		defect := gameBaseCallDefects[name]
		t.Run(name, func(t *testing.T) {
			expected, actual := gameBaseCallFixture(t)
			var result report
			withGameBaseCallAdapters(t, func() { defect(actual) }, func() {
				result = verify(expected, actual, 0, "report", "contract", "mapping")
			})
			if result.Summary["LANGUAGE_MAPPING_MISMATCH"] == 0 {
				t.Fatalf("base-call defect %q raised no LANGUAGE_MAPPING_MISMATCH", name)
			}
		})
	}
}

// TestGameBaseCallAdaptersAreNotXNAIdentities is the accounting claim: the
// family is language support and must not inflate any identity counter.
//
// It compares the pinned expected surface against itself with the registry
// present, which is the only state the binding ever runs in, and asserts that
// none of the five function names appears anywhere in the expected surface as
// a projected XNA member or type.
func TestGameBaseCallAdaptersAreNotXNAIdentities(t *testing.T) {
	expected, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	frameworkPackage := modulePath + "/Microsoft/Xna/Framework"
	for _, adapter := range gameBaseCallAdapters {
		key := symbolKey{Package: frameworkPackage, Name: adapter.GoFunction}
		if member, present := expected.Members[key]; present {
			t.Fatalf("base-call adapter %s is a projected XNA member (%s); it is language support and must never be one",
				adapter.GoFunction, member.XNA)
		}
		if _, present := expected.Types[key]; present {
			t.Fatalf("base-call adapter %s is a projected XNA type", adapter.GoFunction)
		}
		// Nor may it be a member of Game or of GameCallbacks under any name.
		for memberKey := range expected.Members {
			if memberKey.Package == frameworkPackage && memberKey.Name == adapter.GoFunction {
				t.Fatalf("base-call adapter %s appears as a projected member on receiver %q",
					adapter.GoFunction, memberKey.Receiver)
			}
		}
	}
	if expected.ReferenceMembers != 2964 {
		t.Fatalf("REFERENCE_MEMBERS moved to %d; the base-call family must not touch it", expected.ReferenceMembers)
	}
	// The base-call family adds no identity. EXPECTED_GO_MEMBERS is admitted by
	// its parts rather than by one total, so what is pinned here is the
	// XNA-DECLARED part: 3243, which never moves. The two inherited provenance
	// classes are pinned by their own registries.
	declared := expected.ExpectedGoMembers - expected.BCLInheritedProjections - expected.XNAInheritedProjections
	if declared != 3243 {
		t.Fatalf("XNA-declared member projections moved to %d; the base-call family must not touch them", declared)
	}
}

// TestEveryGameCallbacksMemberHasExactlyOneBaseCallAdapter is claim (3) as a
// direct assertion rather than as a defect: the registry is closed, and it is
// closed around exactly the five protected virtuals GameCallbacks projects.
func TestEveryGameCallbacksMemberHasExactlyOneBaseCallAdapter(t *testing.T) {
	expected, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	frameworkPackage := modulePath + "/Microsoft/Xna/Framework"
	callbacks := make(map[string]bool)
	for _, member := range expected.Members {
		if member.PackagePath == frameworkPackage && member.Receiver == "GameCallbacks" {
			callbacks[member.GoName] = true
			if member.SourceAccess != "protected" {
				t.Fatalf("GameCallbacks projects %s, which the contract declares %q; the adapter family assumes protected virtuals",
					member.GoName, member.SourceAccess)
			}
		}
	}
	if len(callbacks) != 5 {
		t.Fatalf("expected 5 GameCallbacks members, found %d: %v", len(callbacks), callbacks)
	}
	for name := range callbacks {
		if _, declared := gameBaseCallAdapters[name]; !declared {
			t.Fatalf("GameCallbacks member %s has no base-call adapter", name)
		}
	}
	for name := range gameBaseCallAdapters {
		if !callbacks[name] {
			t.Fatalf("base-call adapter %s corresponds to no GameCallbacks member", name)
		}
	}
}

// gameBaseCallMutationCase applies one named Foundation-31 base-call defect
// from the shared table, so the mutation inventory and the named test cannot
// drift. Mutation ids have the form f31base_<defect>.
func gameBaseCallMutationCase(t *testing.T, mutation string) report {
	t.Helper()
	name := strings.TrimPrefix(mutation, "f31base_")
	defect, ok := gameBaseCallDefects[name]
	if !ok {
		t.Fatalf("unknown base-call defect %q", name)
	}
	expected, actual := gameBaseCallFixture(t)
	var result report
	withGameBaseCallAdapters(t, func() { defect(actual) }, func() {
		result = verify(expected, actual, 0, "report", "contract", "mapping")
	})
	return result
}

// TestMappingRulesDeclareTheSameBaseCallAdaptersAsTheRegistry keeps the
// documented rules file and the executable registry from drifting.
//
// mapping-rules.json is hashed into every report, so it is the record of what
// the binding claims its rules are; gameBaseCallAdapters is what the verifier
// actually enforces. A family documented in one and not the other would let a
// reader of the report believe something the tool never checks.
func TestMappingRulesDeclareTheSameBaseCallAdaptersAsTheRegistry(t *testing.T) {
	data, err := os.ReadFile("mapping-rules.json")
	if err != nil {
		t.Fatal(err)
	}
	var rules struct {
		GameBaseCallAdapters struct {
			Adapters map[string]struct {
				CLRMember  string `json:"clrMember"`
				GoFunction string `json:"goFunction"`
				Signature  string `json:"signature"`
			} `json:"adapters"`
		} `json:"gameBaseCallAdapters"`
	}
	if err := json.Unmarshal(data, &rules); err != nil {
		t.Fatal(err)
	}
	documented := rules.GameBaseCallAdapters.Adapters
	if len(documented) != len(gameBaseCallAdapters) {
		t.Fatalf("mapping-rules.json documents %d base-call adapters, the registry declares %d",
			len(documented), len(gameBaseCallAdapters))
	}
	for name, adapter := range gameBaseCallAdapters {
		entry, present := documented[name]
		if !present {
			t.Fatalf("base-call adapter %q is in the registry but not in mapping-rules.json", name)
		}
		if entry.CLRMember != adapter.CLRMember {
			t.Fatalf("base-call adapter %q: rules say CLR member %q, registry says %q", name, entry.CLRMember, adapter.CLRMember)
		}
		if entry.GoFunction != adapter.GoFunction {
			t.Fatalf("base-call adapter %q: rules say Go function %q, registry says %q", name, entry.GoFunction, adapter.GoFunction)
		}
		// The documented signature must spell the registry's exact parameters
		// and results, so a widened signature cannot hide behind prose.
		wanted := "func " + adapter.GoFunction + "(game *Game"
		for _, parameter := range adapter.Parameters[1:] {
			wanted += ", " + strings.ToLower(parameter[:1]) + parameter[1:] + " " + parameter
		}
		wanted += ")"
		if len(adapter.Results) > 0 {
			wanted += " " + strings.Join(adapter.Results, ", ")
		}
		if entry.Signature != wanted {
			t.Fatalf("base-call adapter %q: rules document signature %q, registry implies %q", name, entry.Signature, wanted)
		}
	}
}

// TestEveryBaseCallDefectHasAMutationFixture keeps the shared defect table and
// the mutation inventory from drifting.
func TestEveryBaseCallDefectHasAMutationFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/mutations.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []mutationFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	inventoried := make(map[string]bool)
	for _, fixture := range fixtures {
		if strings.HasPrefix(fixture.Mutation, "f31base_") {
			inventoried[strings.TrimPrefix(fixture.Mutation, "f31base_")] = true
		}
	}
	for name := range gameBaseCallDefects {
		if !inventoried[name] {
			t.Fatalf("base-call defect %q has no mutation fixture", name)
		}
	}
	for name := range inventoried {
		if _, declared := gameBaseCallDefects[name]; !declared {
			t.Fatalf("mutation fixture f31base_%s names no defect in the shared table", name)
		}
	}
}

// ---------------------------------------------------------------------------
// Foundation 32 — negative controls for compiler-checked interface conformance.
// ---------------------------------------------------------------------------

// conformanceFixture builds a synthetic package containing one interface and
// one class, so the conformance rule can be attacked directly.
//
// The evidence the rule consumes is go/types, not the mapping tables, so a
// defect has to be expressed as real compiler evidence. Constructing it here is
// what lets the control assert the rule rather than assert the binding.
func conformanceFixture(t *testing.T, shape string) (*expectedSurface, *actualSurface) {
	t.Helper()
	const (
		packagePath   = "example.test/pkg"
		clrOwner      = "Probe.Namespace.Probe"
		clrInterface  = "Probe.Namespace.IProbe"
		goOwner       = "Probe"
		goInterface   = "IProbe"
		contractMethd = "Run"
	)
	pkg := types.NewPackage(packagePath, "pkg")

	// The contract: Run() error.
	errorType := types.Universe.Lookup("error").Type()
	contractSignature := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(), types.NewTuple(types.NewVar(token.NoPos, pkg, "", errorType)), false)
	contractMethod := types.NewFunc(token.NoPos, pkg, contractMethd, contractSignature)
	contract := types.NewInterfaceType([]*types.Func{contractMethod}, nil)
	contractNamed := types.NewNamed(types.NewTypeName(token.NoPos, pkg, goInterface, nil), contract, nil)
	pkg.Scope().Insert(contractNamed.Obj())

	ownerNamed := types.NewNamed(types.NewTypeName(token.NoPos, pkg, goOwner, nil), types.NewStruct(nil, nil), nil)
	pkg.Scope().Insert(ownerNamed.Obj())

	receiver := types.NewVar(token.NoPos, pkg, "p", types.NewPointer(ownerNamed))
	switch shape {
	case "conforming":
		ownerNamed.AddMethod(types.NewFunc(token.NoPos, pkg, contractMethd,
			types.NewSignatureType(receiver, nil, nil,
				types.NewTuple(), types.NewTuple(types.NewVar(token.NoPos, pkg, "", errorType)), false)))
	case "wrong-signature":
		// Run() with no result: the member exists and its name matches, which
		// is exactly what a structural member comparison would accept.
		ownerNamed.AddMethod(types.NewFunc(token.NoPos, pkg, contractMethd,
			types.NewSignatureType(receiver, nil, nil, types.NewTuple(), types.NewTuple(), false)))
	case "value-receiver-only":
		// Declared on the VALUE method set. CNA-Go projects a CLR class as a
		// pointer facade, and *Probe still carries a value method, so this one
		// conforms; it is here as the control that the rule tests the pointer
		// method set rather than accidentally testing the value one.
		valueReceiver := types.NewVar(token.NoPos, pkg, "p", ownerNamed)
		ownerNamed.AddMethod(types.NewFunc(token.NoPos, pkg, contractMethd,
			types.NewSignatureType(valueReceiver, nil, nil,
				types.NewTuple(), types.NewTuple(types.NewVar(token.NoPos, pkg, "", errorType)), false)))
	case "absent":
		// No method at all.
	default:
		t.Fatalf("unknown fixture shape %q", shape)
	}
	pkg.MarkComplete()

	ownerKey := symbolKey{Package: packagePath, Name: goOwner}
	interfaceKey := symbolKey{Package: packagePath, Name: goInterface}
	expected := &expectedSurface{
		Types: map[symbolKey]*expectedType{
			ownerKey: {Key: ownerKey, XNA: clrOwner, GoName: goOwner, PackagePath: packagePath, Kind: "class",
				MappedInterfaces: []mappedInterface{{XNA: clrInterface, GoName: goInterface}}},
			interfaceKey: {Key: interfaceKey, XNA: clrInterface, GoName: goInterface, PackagePath: packagePath, Kind: "interface"},
		},
		Members:            make(map[symbolKey]*expectedMember),
		InterfaceWitnesses: make(map[symbolKey]*expectedInterfaceWitness),
	}
	actual := &actualSurface{
		Types:       make(map[symbolKey]*actualType),
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: map[string]string{packagePath: "pkg"},
		Packages:    map[string]*types.Package{packagePath: pkg},
	}
	return expected, actual
}

// TestDeclaredInterfaceConformanceDefectsAreRejected attacks the rule in the
// two directions a structural member comparison cannot catch.
func TestDeclaredInterfaceConformanceDefectsAreRejected(t *testing.T) {
	const clrOwner = "Probe.Namespace.Probe"
	for _, testCase := range []struct {
		shape       string
		wantVerdict string
		wantMessage string
	}{
		{"conforming", "PASS", ""},
		{"value-receiver-only", "PASS", ""},
		{"absent", "FAIL", "Run is absent"},
		{"wrong-signature", "FAIL", "Run has the wrong signature"},
	} {
		t.Run(testCase.shape, func(t *testing.T) {
			expected, actual := conformanceFixture(t, testCase.shape)
			result := report{Summary: make(map[string]int)}
			measurements := measureDeclaredInterfaceConformance(&result, expected, actual, []string{clrOwner})
			if len(measurements) != 1 {
				t.Fatalf("expected one measurement, got %d", len(measurements))
			}
			if measurements[0].Verdict != testCase.wantVerdict {
				t.Fatalf("verdict %q, want %q", measurements[0].Verdict, testCase.wantVerdict)
			}
			if testCase.wantVerdict == "PASS" {
				if result.Summary["INTERFACE_MAPPING_MISMATCH"] != 0 {
					t.Fatalf("a conforming class produced %d diagnostics", result.Summary["INTERFACE_MAPPING_MISMATCH"])
				}
				return
			}
			if result.Summary["INTERFACE_MAPPING_MISMATCH"] == 0 {
				t.Fatalf("shape %q raised no INTERFACE_MAPPING_MISMATCH", testCase.shape)
			}
			found := false
			for _, item := range result.Diagnostics {
				if item.Category == "INTERFACE_MAPPING_MISMATCH" && strings.Contains(item.Message, testCase.wantMessage) {
					found = true
				}
			}
			if !found {
				t.Fatalf("no diagnostic named %q; got %v", testCase.wantMessage, result.Diagnostics)
			}
		})
	}
}

// TestAPartialTypeIsNotHeldToItsDeclaredInterfaces pins the completeness gate.
// GraphicsDeviceManager declares two contracts and satisfies neither, because
// 20 of its members are missing; that gap is already MISSING_MEMBER and must
// not be reported a second time as a conformance failure.
func TestAPartialTypeIsNotHeldToItsDeclaredInterfaces(t *testing.T) {
	expected, actual := conformanceFixture(t, "absent")
	result := report{Summary: make(map[string]int)}
	// The same fixture, with the owner absent from the complete list.
	measurements := measureDeclaredInterfaceConformance(&result, expected, actual, nil)
	if len(measurements) != 0 {
		t.Fatalf("a partial type was measured for conformance: %v", measurements)
	}
	if result.Summary["INTERFACE_MAPPING_MISMATCH"] != 0 {
		t.Fatal("a partial type produced a conformance diagnostic")
	}
}

// TestGameComponentSatisfiesBothDeclaredContracts asserts the live claim by
// name, so the measurement cannot silently stop covering it.
func TestGameComponentSatisfiesBothDeclaredContracts(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	actual, err := extractActual(root)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	result := verify(expected, actual, 0, "report", "contract", "mapping")
	wanted := map[string]bool{
		"Microsoft.Xna.Framework.IGameComponent": false,
		"Microsoft.Xna.Framework.IUpdateable":    false,
	}
	for _, measurement := range result.DeclaredInterfaceConformance {
		if measurement.Owner != "Microsoft.Xna.Framework.GameComponent" {
			continue
		}
		if _, tracked := wanted[measurement.CLRInterface]; !tracked {
			t.Fatalf("unexpected conformance claim %q", measurement.CLRInterface)
		}
		if measurement.Verdict != "PASS" || !measurement.PointerSatisfies {
			t.Fatalf("%s does not satisfy %s", measurement.GoOwner, measurement.GoInterface)
		}
		wanted[measurement.CLRInterface] = true
	}
	for identity, measured := range wanted {
		if !measured {
			t.Fatalf("GameComponent's declared %s was never measured for conformance", identity)
		}
	}
}

// ---------------------------------------------------------------------------
// Foundation 33 — negative controls for the XNA base frontier.
// ---------------------------------------------------------------------------

// withXNABaseRelationships runs fn with the registry mutated and restores it.
func withXNABaseRelationships(t *testing.T, mutate func(), fn func()) {
	t.Helper()
	saved := make(map[string]xnaBaseRelationship, len(xnaBaseRelationships))
	for base, relationship := range xnaBaseRelationships {
		saved[base] = relationship
	}
	defer func() { xnaBaseRelationships = saved }()
	mutate()
	fn()
}

// deferredBaseFixtureName is the base the DEFERRED-only defects below mutate.
// It has to be a relationship that is still DEFERRED: GameComponent became the
// first COMPOSED one in Foundation 41, and a COMPOSED relationship is not held
// to the blocker rules, which is the whole point of the status.
const deferredBaseFixtureName = "Microsoft.Xna.Framework.Graphics.GraphicsResource"

const gameComponentBase = "Microsoft.Xna.Framework.GameComponent"

// xnaBaseDefects is the shared table behind both the named test and the
// mutation inventory. Each entry either mutates the registry or supplies a
// wrong completeness claim, and must raise BASE_MAPPING_MISMATCH.
//
// The signature carries the complete-type list because one defect -- the
// substantive one -- is expressed there rather than in the registry.
var xnaBaseDefects = map[string]func(complete *[]string){
	// The silence this whole frontier exists to end: a class inherited by
	// another class in the profile, with nothing recorded about it.
	"unrecorded_xna_base": func(*[]string) {
		delete(xnaBaseRelationships, gameComponentBase)
	},
	// Foundation 29's rule, on the second frontier: a deferral that records
	// nothing says nothing.
	"deferred_base_records_no_blocker": func(*[]string) {
		relationship := xnaBaseRelationships[deferredBaseFixtureName]
		relationship.Blockers = nil
		xnaBaseRelationships[deferredBaseFixtureName] = relationship
	},
	"blocker_class_is_unrecorded": func(*[]string) {
		relationship := xnaBaseRelationships[deferredBaseFixtureName]
		relationship.Blockers = []xnaBaseBlocker{{Class: "LATER", Detail: "not now"}}
		xnaBaseRelationships[deferredBaseFixtureName] = relationship
	},
	"blocker_records_no_detail": func(*[]string) {
		relationship := xnaBaseRelationships[deferredBaseFixtureName]
		relationship.Blockers = []xnaBaseBlocker{{Class: "ARCHITECTURE"}}
		xnaBaseRelationships[deferredBaseFixtureName] = relationship
	},
	"status_is_neither_composed_nor_deferred": func(*[]string) {
		relationship := xnaBaseRelationships[deferredBaseFixtureName]
		relationship.Status = "SOON"
		xnaBaseRelationships[deferredBaseFixtureName] = relationship
	},
	// A registry entry for a base nothing actually derives from is a stale
	// claim, and would let a real relationship hide behind a plausible one.
	"stale_relationship_with_no_derived_type": func(*[]string) {
		xnaBaseRelationships["Microsoft.Xna.Framework.Vector3"] = xnaBaseRelationship{
			Status: "DEFERRED", Blockers: []xnaBaseBlocker{{Class: "ARCHITECTURE", Detail: "invented"}},
		}
	},
	// The substantive rule. Texture2D inherits nine public members from
	// Texture and GraphicsResource that CNA-Go does not project, so calling it
	// COMPLETE asserts something false.
	"derived_type_of_a_deferred_base_reported_complete": func(complete *[]string) {
		*complete = append(*complete, "Microsoft.Xna.Framework.Graphics.Texture2D")
	},
}

// TestXNABaseFrontierDefectsAreRejected attacks every rule the second base
// frontier rests on.
func TestXNABaseFrontierDefectsAreRejected(t *testing.T) {
	expected, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	baseline := report{Summary: make(map[string]int)}
	baselineProjections := measureXNABaseRelationships(&baseline, expected, nil)
	if baseline.Summary["BASE_MAPPING_MISMATCH"] != 0 {
		t.Fatalf("the unmutated XNA base frontier is not clean: %v", baseline.Diagnostics)
	}
	if len(baselineProjections) != len(xnaBaseRelationships) {
		t.Fatalf("measured %d relationships, registry declares %d",
			len(baselineProjections), len(xnaBaseRelationships))
	}
	for _, projection := range baselineProjections {
		if projection.Verdict != "PASS" {
			t.Fatalf("%s did not pass in the baseline", projection.CLRBase)
		}
		if len(projection.Derived) == 0 {
			t.Fatalf("%s has no derived type", projection.CLRBase)
		}
	}

	names := make([]string, 0, len(xnaBaseDefects))
	for name := range xnaBaseDefects {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		defect := xnaBaseDefects[name]
		t.Run(name, func(t *testing.T) {
			var complete []string
			result := report{Summary: make(map[string]int)}
			withXNABaseRelationships(t, func() { defect(&complete) }, func() {
				measureXNABaseRelationships(&result, expected, complete)
			})
			if result.Summary["BASE_MAPPING_MISMATCH"] == 0 {
				t.Fatalf("XNA base defect %q raised no BASE_MAPPING_MISMATCH", name)
			}
		})
	}
}

// TestEveryXNABaseLinkInTheContractIsRecorded is the closure claim stated
// directly: the registry covers every XNA-to-XNA inheritance the pinned profile
// declares, so a new one cannot appear unnoticed.
func TestEveryXNABaseLinkInTheContractIsRecorded(t *testing.T) {
	contract := loadPinnedContract(t)
	byName := make(map[string]bool, len(contract.Types))
	for _, t := range contract.Types {
		byName[t.Name] = true
	}
	links := make(map[string][]string)
	for _, entry := range contract.Types {
		if entry.BaseType == nil || !byName[*entry.BaseType] {
			continue
		}
		links[*entry.BaseType] = append(links[*entry.BaseType], entry.Name)
	}
	if len(links) != len(xnaBaseRelationships) {
		t.Fatalf("the contract declares %d XNA base links, the registry records %d", len(links), len(xnaBaseRelationships))
	}
	derivedTotal := 0
	for base, derived := range links {
		derivedTotal += len(derived)
		if _, recorded := xnaBaseRelationships[base]; !recorded {
			t.Fatalf("XNA base %q is inherited by %v and is not recorded", base, derived)
		}
	}
	if derivedTotal != 41 {
		t.Fatalf("the contract declares %d XNA-derived types, expected 41", derivedTotal)
	}
}

// TestNoCompleteTypeInheritsFromADeferredXNABase states the substantive
// invariant against the LIVE binding rather than a fixture.
func TestNoCompleteTypeInheritsFromADeferredXNABase(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	actual, err := extractActual(root)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	result := verify(expected, actual, 0, "report", "contract", "mapping")
	complete := make(map[string]bool, len(result.CompleteTypes))
	for _, identity := range result.CompleteTypes {
		complete[identity] = true
	}
	for _, projection := range result.XNABaseRelationships {
		if projection.Status != "DEFERRED" {
			continue
		}
		for _, identity := range projection.Derived {
			if complete[identity] {
				t.Fatalf("%s is COMPLETE but inherits %d unprojected public members from the deferred base %s",
					identity, projection.InheritedPublicMembers, projection.CLRBase)
			}
		}
	}
	if result.Summary["XNA_BASE_RELATIONSHIPS"] != 12 {
		t.Fatalf("XNA_BASE_RELATIONSHIPS=%d", result.Summary["XNA_BASE_RELATIONSHIPS"])
	}
	if result.Summary["XNA_BASE_DERIVED_TYPES"] != 41 {
		t.Fatalf("XNA_BASE_DERIVED_TYPES=%d", result.Summary["XNA_BASE_DERIVED_TYPES"])
	}
}

// xnaBaseMutationCase applies one named Foundation-33 XNA base defect from the
// shared table. Mutation ids have the form f33xna_<defect>.
func xnaBaseMutationCase(t *testing.T, mutation string) report {
	t.Helper()
	name := strings.TrimPrefix(mutation, "f33xna_")
	defect, ok := xnaBaseDefects[name]
	if !ok {
		t.Fatalf("unknown XNA base defect %q", name)
	}
	expected, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	var complete []string
	result := report{Summary: make(map[string]int)}
	withXNABaseRelationships(t, func() { defect(&complete) }, func() {
		measureXNABaseRelationships(&result, expected, complete)
	})
	return result
}

// TestEveryXNABaseDefectHasAMutationFixture keeps the shared defect table and
// the mutation inventory from drifting.
func TestEveryXNABaseDefectHasAMutationFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/mutations.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []mutationFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	inventoried := make(map[string]bool)
	for _, fixture := range fixtures {
		if strings.HasPrefix(fixture.Mutation, "f33xna_") {
			inventoried[strings.TrimPrefix(fixture.Mutation, "f33xna_")] = true
		}
	}
	for name := range xnaBaseDefects {
		if !inventoried[name] {
			t.Fatalf("XNA base defect %q has no mutation fixture", name)
		}
	}
	for name := range inventoried {
		if _, declared := xnaBaseDefects[name]; !declared {
			t.Fatalf("mutation fixture f33xna_%s names no defect in the shared table", name)
		}
	}
}

// TestMappingRulesDeclareTheSameXNABasesAsTheRegistry keeps the documented
// rules file and the executable registry from drifting, the same way the
// base-call family's guard does.
func TestMappingRulesDeclareTheSameXNABasesAsTheRegistry(t *testing.T) {
	data, err := os.ReadFile("mapping-rules.json")
	if err != nil {
		t.Fatal(err)
	}
	var rules struct {
		XNABaseRelationships struct {
			BlockerClasses map[string]string `json:"blockerClasses"`
			Statuses       map[string]string `json:"statuses"`
		} `json:"xnaBaseRelationships"`
	}
	if err := json.Unmarshal(data, &rules); err != nil {
		t.Fatal(err)
	}
	if len(rules.XNABaseRelationships.BlockerClasses) != len(xnaBaseBlockerClasses) {
		t.Fatalf("mapping-rules.json documents %d blocker classes, the registry admits %d",
			len(rules.XNABaseRelationships.BlockerClasses), len(xnaBaseBlockerClasses))
	}
	for class := range xnaBaseBlockerClasses {
		if _, documented := rules.XNABaseRelationships.BlockerClasses[class]; !documented {
			t.Fatalf("blocker class %q is admitted by the registry and not documented", class)
		}
	}
	if len(rules.XNABaseRelationships.Statuses) != 2 {
		t.Fatalf("expected two documented statuses, got %d", len(rules.XNABaseRelationships.Statuses))
	}
	// Every status the registry actually uses must be documented.
	for base, relationship := range xnaBaseRelationships {
		if _, documented := rules.XNABaseRelationships.Statuses[relationship.Status]; !documented {
			t.Fatalf("%s carries undocumented status %q", base, relationship.Status)
		}
	}
}

// ---------------------------------------------------------------------------
// Foundation 36 — negative controls for the native game-signal bridge and the
// frame-hook frontier.
// ---------------------------------------------------------------------------

// gameSignalFixture builds a surface pair that exercises the two Foundation-36
// registries and nothing else. The expected surface is the whole pinned
// contract, so Game, its four events, its three raise sites and its four frame
// hooks are all present; the actual surface carries only the framework package
// with exactly the members those registries name.
//
// Every defect below is measured as a delta against this fixture's own
// baseline, so the fixture's unrelated MISSING_TYPE noise cannot make a defect
// look detected.
func gameSignalFixture(t *testing.T) (*expectedSurface, *actualSurface) {
	t.Helper()
	expected, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	actual := &actualSurface{
		Types:       make(map[symbolKey]*actualType),
		Members:     make(map[symbolKey]*actualMember),
		PackageDirs: map[string]string{modulePath + "/Microsoft/Xna/Framework": "Microsoft/Xna/Framework"},
		Packages:    make(map[string]*types.Package),
	}
	seedSignatureAdapters(actual)
	seedGameBaseCallAdapters(actual)
	seedGameSignalMembers(t, expected, actual)
	seedGameCallbacksMembers(actual)
	actual.Packages[modulePath+"/Microsoft/Xna/Framework"] = frameHookCapabilityPackage()
	return expected, actual
}

// seedGameSignalMembers copies the projected members the two registries name
// out of the expected surface and into the actual one, so the fixture starts
// VALID and every diagnostic below is caused by its own mutation.
func seedGameSignalMembers(t *testing.T, expected *expectedSurface, actual *actualSurface) {
	t.Helper()
	gameType := expected.typeForXNA("Microsoft.Xna.Framework.Game")
	if gameType == nil {
		t.Fatal("the pinned contract does not declare Game")
	}
	wanted := make(map[string]bool)
	for name, signal := range gameNativeSignals {
		wanted["Add"+name+"Handler"] = true
		wanted["Remove"+name+"Handler"] = true
		if signal.RaiseSite != "" {
			wanted[signal.RaiseSite] = true
		}
		if signal.ManagedRaiseSite != "" {
			wanted[signal.ManagedRaiseSite] = true
		}
	}
	for name := range gameFrameHooks {
		wanted[name] = true
	}
	for _, key := range gameType.Members {
		member := expected.Members[key]
		if member == nil || !wanted[member.GoName] {
			continue
		}
		actual.Members[key] = &actualMember{
			Key: key, Kind: member.GoKind,
			Parameters: append([]string(nil), member.Parameters...),
			Results:    append([]string(nil), member.Results...),
		}
	}
}

// withGameNativeSignals runs fn with the signal registry mutated, then restores
// it.
func withGameNativeSignals(t *testing.T, mutate func(), fn func()) {
	t.Helper()
	saved := make(map[string]gameNativeSignal, len(gameNativeSignals))
	for name, signal := range gameNativeSignals {
		saved[name] = signal
	}
	defer func() { gameNativeSignals = saved }()
	mutate()
	fn()
}

// withGameFrameHooks runs fn with the frame-hook registry mutated, then
// restores it.
func withGameFrameHooks(t *testing.T, mutate func(), fn func()) {
	t.Helper()
	saved := make(map[string]gameFrameHook, len(gameFrameHooks))
	for name, hook := range gameFrameHooks {
		saved[name] = hook
	}
	defer func() { gameFrameHooks = saved }()
	mutate()
	fn()
}

func frameworkGameKey(name string) symbolKey {
	return symbolKey{Package: modulePath + "/Microsoft/Xna/Framework", Receiver: "Game", Name: name}
}

// gameSignalDefects is the shared table behind both the named test and the
// mutation inventory. Each entry breaks exactly one rule the native-signal
// bridge or the frame-hook frontier rests on, and each must raise
// LANGUAGE_MAPPING_MISMATCH.
var gameSignalDefects = map[string]func(expected *expectedSurface, actual *actualSurface){
	// ---- the native signal bridge --------------------------------------
	// An event Game declares with no bound signal is an accessor pair nothing
	// can ever fire.
	"signal_absent_for_a_declared_event": func(_ *expectedSurface, _ *actualSurface) {
		delete(gameNativeSignals, "Exiting")
	},
	// A signal for an event Game does not declare is an invention.
	"signal_declared_for_an_event_game_does_not_declare": func(_ *expectedSurface, _ *actualSurface) {
		gameNativeSignals["Suspended"] = gameNativeSignal{
			CNAConstant: "CNA_GAME_EVENT_SUSPENDED", CNAIdentity: 9,
			CLREvent: "Microsoft.Xna.Framework.Game::Suspended", Sender: "GAME",
			ReferencePath: []string{"invented"}, RuntimeEvidence: "VERIFIED_NATIVE",
		}
	},
	// Two events sharing one native identity would deliver one signal to both.
	"duplicate_cna_identity": func(_ *expectedSurface, _ *actualSurface) {
		signal := gameNativeSignals["Exiting"]
		signal.CNAIdentity = gameNativeSignals["Activated"].CNAIdentity
		gameNativeSignals["Exiting"] = signal
	},
	// A signal that names no canonical constant is unbound in fact.
	"signal_names_no_cna_constant": func(_ *expectedSurface, _ *actualSurface) {
		signal := gameNativeSignals["Disposed"]
		signal.CNAConstant = ""
		gameNativeSignals["Disposed"] = signal
	},
	// A raise site the reference does not declare.
	"raise_site_is_not_a_projected_method": func(_ *expectedSurface, _ *actualSurface) {
		signal := gameNativeSignals["Activated"]
		signal.RaiseSite = "OnActivating"
		gameNativeSignals["Activated"] = signal
	},
	// Inventing an OnDisposed. The reference has none: Dispose(bool) invokes
	// the delegate field directly.
	"raise_site_invented_for_an_event_that_has_none": func(_ *expectedSurface, _ *actualSurface) {
		signal := gameNativeSignals["Disposed"]
		signal.RaiseSite = "OnDisposed"
		gameNativeSignals["Disposed"] = signal
	},
	// Dropping a raise site the pinned contract does declare.
	"raise_site_omitted_for_an_event_that_has_one": func(_ *expectedSurface, _ *actualSurface) {
		signal := gameNativeSignals["Exiting"]
		signal.RaiseSite = ""
		gameNativeSignals["Exiting"] = signal
	},
	// A raise site whose projected shape is not (any, *EventArgs) error is not
	// the CLR raise site's shape.
	"raise_site_signature_is_not_the_clr_shape": func(expected *expectedSurface, _ *actualSurface) {
		expected.Members[frameworkGameKey("OnExiting")].Parameters = []string{"*EventArgs"}
	},
	// The raise site is declared but the package does not ship it.
	"raise_site_absent_from_the_package": func(_ *expectedSurface, actual *actualSurface) {
		delete(actual.Members, frameworkGameKey("OnDeactivated"))
	},
	// One of the two accessors is missing, so the signal has nowhere to arrive.
	"accessor_absent_from_the_package": func(_ *expectedSurface, actual *actualSurface) {
		delete(actual.Members, frameworkGameKey("AddActivatedHandler"))
	},
	// A sender that is neither `this` nor null. Every raise in this family
	// pushes one of the two.
	"sender_is_neither_game_nor_null": func(_ *expectedSurface, _ *actualSurface) {
		signal := gameNativeSignals["Activated"]
		signal.Sender = "HOST"
		gameNativeSignals["Activated"] = signal
	},
	// Without a recorded reference path there is nothing to check against.
	"signal_records_no_reference_path": func(_ *expectedSurface, _ *actualSurface) {
		signal := gameNativeSignals["Disposed"]
		signal.ReferencePath = nil
		gameNativeSignals["Disposed"] = signal
	},
	// An unrecognised evidence class hides whether the signal was ever seen.
	"runtime_evidence_unclassified": func(_ *expectedSurface, _ *actualSurface) {
		signal := gameNativeSignals["Exiting"]
		signal.RuntimeEvidence = "PROBABLY_FINE"
		gameNativeSignals["Exiting"] = signal
	},
	// A signal the environment cannot deliver must say why. An unexplained one
	// is an unproved claim wearing a label.
	"unverified_signal_records_no_reason": func(_ *expectedSurface, _ *actualSurface) {
		signal := gameNativeSignals["Deactivated"]
		signal.EvidenceReason = ""
		gameNativeSignals["Deactivated"] = signal
	},
	// And a verified one must not: an excuse next to a verification is a sign
	// the label and the evidence disagree.
	"verified_signal_records_an_excuse": func(_ *expectedSurface, _ *actualSurface) {
		signal := gameNativeSignals["Activated"]
		signal.EvidenceReason = "not really"
		gameNativeSignals["Activated"] = signal
	},
	// ---- Foundation 39: the authoritative raise path --------------------
	// The rule Foundation 34 satisfied and Foundation 39 replaced. Binding a
	// native signal to an event whose reference raise path is MANAGED puts the
	// raise at a moment the reference has no raise at.
	"native_signal_drives_a_managed_raise_path": func(_ *expectedSurface, _ *actualSurface) {
		signal := gameNativeSignals["Disposed"]
		signal.NativeSignalRole = "PUBLIC_EVENT_RAISE"
		gameNativeSignals["Disposed"] = signal
	},
	// And the mirror: an event the host really does raise, demoted to a signal
	// that raises nothing, would leave the projected accessors unfireable.
	"host_raised_event_demoted_to_lifecycle_only": func(_ *expectedSurface, _ *actualSurface) {
		signal := gameNativeSignals["Exiting"]
		signal.NativeSignalRole = "LIFECYCLE_ONLY"
		gameNativeSignals["Exiting"] = signal
	},
	"raise_path_unclassified": func(_ *expectedSurface, _ *actualSurface) {
		signal := gameNativeSignals["Activated"]
		signal.RaisePath = "SOMEWHERE"
		gameNativeSignals["Activated"] = signal
	},
	"native_signal_role_unclassified": func(_ *expectedSurface, _ *actualSurface) {
		signal := gameNativeSignals["Activated"]
		signal.NativeSignalRole = "MAYBE"
		gameNativeSignals["Activated"] = signal
	},
	// A managed raise path that names no member is a claim with nothing behind
	// it: the event would have no reachable raise site at all.
	"managed_raise_path_names_no_member": func(_ *expectedSurface, _ *actualSurface) {
		signal := gameNativeSignals["Disposed"]
		signal.ManagedRaiseSite = ""
		gameNativeSignals["Disposed"] = signal
	},
	"managed_raise_site_is_not_projected": func(_ *expectedSurface, _ *actualSurface) {
		signal := gameNativeSignals["Disposed"]
		signal.ManagedRaiseSite = "DisposeSomehow"
		gameNativeSignals["Disposed"] = signal
	},
	// The reference raises this event from a protected virtual. A managed raise
	// site that is a public member would be a different contract.
	"managed_raise_site_is_not_protected": func(expected *expectedSurface, _ *actualSurface) {
		expected.Members[frameworkGameKey("DisposeByBoolean")].SourceAccess = "public"
	},
	"managed_raise_site_absent_from_the_package": func(_ *expectedSurface, actual *actualSurface) {
		delete(actual.Members, frameworkGameKey("DisposeByBoolean"))
	},
	// A host-raised event that also claims a managed raise site claims two.
	"host_raised_event_claims_a_managed_raise_site": func(_ *expectedSurface, _ *actualSurface) {
		signal := gameNativeSignals["Activated"]
		signal.ManagedRaiseSite = "DisposeByBoolean"
		gameNativeSignals["Activated"] = signal
	},
	// Without the native moment there is nothing to compare the reference's
	// raise site against, which is the comparison the whole rule rests on.
	"signal_records_no_native_moment": func(_ *expectedSurface, _ *actualSurface) {
		signal := gameNativeSignals["Disposed"]
		signal.NativeSignalMoment = ""
		gameNativeSignals["Disposed"] = signal
	},

	// A raise path that exists on Game but no signal declares is a raise
	// nothing measures.
	"raise_site_exists_that_no_signal_declares": func(expected *expectedSurface, _ *actualSurface) {
		gameType := expected.typeForXNA("Microsoft.Xna.Framework.Game")
		key := frameworkGameKey("OnResumed")
		expected.Members[key] = &expectedMember{
			Key: key, XNA: "Microsoft.Xna.Framework.Game::OnResumed(System.Object,System.EventArgs)",
			Owner: "Microsoft.Xna.Framework.Game", SourceKind: "method", SourceAccess: "protected",
			GoKind: "method", GoName: "OnResumed", Receiver: "Game",
			PackagePath: modulePath + "/Microsoft/Xna/Framework",
			Parameters:  []string{"any", "*EventArgs"}, Results: []string{"error"},
		}
		gameType.Members = append(gameType.Members, key)
	},

	// ---- the frame-hook frontier ---------------------------------------
	// The hook is declared but the package does not ship it.
	"frame_hook_absent_from_the_package": func(_ *expectedSurface, actual *actualSurface) {
		delete(actual.Members, frameworkGameKey("BeginDraw"))
	},
	// A hook for a member Game does not project.
	"frame_hook_declared_for_a_member_that_is_not_projected": func(_ *expectedSurface, _ *actualSurface) {
		gameFrameHooks["BeginFrame"] = gameFrameHook{
			CLRMember: "Microsoft.Xna.Framework.Game::BeginFrame", GoName: "BeginFrame",
			Results: []string{"error"}, NativeHook: "CNA_GameFrameHooks::begin_frame",
			ReasonUninstalled: "invented", NativeOrdering: "invented",
			ReferenceBody: []string{"invented"},
		}
	},
	// A hook whose declared signature is not the projected one.
	"frame_hook_signature_mismatch": func(_ *expectedSurface, _ *actualSurface) {
		hook := gameFrameHooks["BeginDraw"]
		hook.Results = []string{"error"}
		gameFrameHooks["BeginDraw"] = hook
	},
	// A GameBase... helper for a member that is not a GameCallbacks override.
	// The base body is reachable as a method on Game and needs no helper.
	"frame_hook_gains_a_base_call_helper": func(_ *expectedSurface, actual *actualSurface) {
		key := symbolKey{Package: modulePath + "/Microsoft/Xna/Framework", Name: "GameBaseBeginRun"}
		actual.Members[key] = &actualMember{Key: key, Kind: "func", Parameters: []string{"*Game"}, Results: []string{"error"}}
	},
	// A hook declared for one of the five mandatory override members would put
	// it in two contracts at once.
	"frame_hook_declared_for_a_gamecallbacks_member": func(_ *expectedSurface, _ *actualSurface) {
		gameFrameHooks["Update"] = gameFrameHook{
			CLRMember: "Microsoft.Xna.Framework.Game::Update", GoName: "Update",
			Parameters: []string{"GameTime"}, Results: []string{"error"},
			NativeHook: "CNA_GameFrameHooks::update", ReasonUninstalled: "invented",
			NativeOrdering: "invented", ReferenceBody: []string{"invented"},
		}
	},
	// An uninstalled native hook with no reason is a silence, not a decision.
	"frame_hook_uninstalled_without_a_reason": func(_ *expectedSurface, _ *actualSurface) {
		hook := gameFrameHooks["EndRun"]
		hook.Installation = "NEVER"
		hook.ReasonUninstalled, hook.Capability = "", ""
		gameFrameHooks["EndRun"] = hook
	},
	// And an installable one that still carries the excuse for not installing
	// it means the record and the code disagree.
	"frame_hook_installed_but_records_a_reason": func(_ *expectedSurface, _ *actualSurface) {
		hook := gameFrameHooks["EndDraw"]
		hook.ReasonUninstalled = "left out"
		gameFrameHooks["EndDraw"] = hook
	},
	// A hook that names no canonical CNA hook records no position at all.
	"frame_hook_names_no_native_hook": func(_ *expectedSurface, _ *actualSurface) {
		hook := gameFrameHooks["BeginRun"]
		hook.NativeHook = ""
		gameFrameHooks["BeginRun"] = hook
	},
	// The correspondence between the native position and the reference call
	// site is the whole claim; unrecorded, it is an assertion.
	"frame_hook_records_no_native_ordering": func(_ *expectedSurface, _ *actualSurface) {
		hook := gameFrameHooks["BeginDraw"]
		hook.NativeOrdering = ""
		gameFrameHooks["BeginDraw"] = hook
	},
	// Without a reference body there is nothing to check the projection against.
	"frame_hook_records_no_reference_body": func(_ *expectedSurface, _ *actualSurface) {
		hook := gameFrameHooks["EndDraw"]
		hook.ReferenceBody = nil
		gameFrameHooks["EndDraw"] = hook
	},
	// The same three deferral rules the base-call family already carries.
	"frame_hook_deferral_unclassified": func(_ *expectedSurface, _ *actualSurface) {
		hook := gameFrameHooks["BeginDraw"]
		hook.Deferred = []gameBaseCallDeferral{{Step: "Logger.BeginLogEvent", Class: "LATER", Reason: "not now"}}
		gameFrameHooks["BeginDraw"] = hook
	},
	"frame_hook_deferral_records_no_reason": func(_ *expectedSurface, _ *actualSurface) {
		hook := gameFrameHooks["BeginDraw"]
		hook.Deferred = []gameBaseCallDeferral{{Step: "Logger.BeginLogEvent", Class: "UNOBSERVABLE"}}
		gameFrameHooks["BeginDraw"] = hook
	},
	"frame_hook_deferral_marked_observable": func(_ *expectedSurface, _ *actualSurface) {
		hook := gameFrameHooks["EndDraw"]
		deferred := append([]gameBaseCallDeferral(nil), hook.Deferred...)
		deferred[0].Observable = true
		hook.Deferred = deferred
		gameFrameHooks["EndDraw"] = hook
	},
	// A hook projected onto anything but Game would move a member Microsoft
	// declared on Game somewhere else.
	"frame_hook_projected_onto_another_receiver": func(expected *expectedSurface, _ *actualSurface) {
		expected.Members[frameworkGameKey("EndRun")].Receiver = "GameCallbacks"
	},
	// And a hook that is not a protected virtual has no base body to project.
	"frame_hook_names_a_public_member": func(expected *expectedSurface, _ *actualSurface) {
		expected.Members[frameworkGameKey("BeginRun")].SourceAccess = "public"
	},
}

// ---------------------------------------------------------------------------
// Foundation 38 — negative controls for the optional per-hook structural
// frame-hook override mechanism.
// ---------------------------------------------------------------------------

// gameFrameHookOverrideDefects is the shared table behind the named test and
// the mutation inventory. Each entry breaks exactly one rule the override
// mechanism rests on, and each must raise LANGUAGE_MAPPING_MISMATCH.
//
// The danger here is not a wrong XNA signature. It is a mechanism that LOOKS
// like the decided one and is not: a hook installed whether or not the consumer
// asked for it, a bundled contract that forces no-op overrides, an exported
// capability that publishes a new public framework identity, a registration
// call that reintroduces mutable per-Game callback state, or a sixth member on
// GameCallbacks that breaks every existing external implementation.
var gameFrameHookOverrideDefects = map[string]func(expected *expectedSurface, actual *actualSurface){
	// ---- installation --------------------------------------------------
	// There is no third installation class. An unconditionally installed hook
	// runs a base body at a frame position CNA-Go picked, which is exactly the
	// automatic base behaviour Foundation 31 refused.
	"hook_installed_unconditionally": func(_ *expectedSurface, _ *actualSurface) {
		hook := gameFrameHooks["BeginRun"]
		hook.Installation = "ALWAYS"
		gameFrameHooks["BeginRun"] = hook
	},
	"installation_class_unrecognised": func(_ *expectedSurface, _ *actualSurface) {
		hook := gameFrameHooks["EndRun"]
		hook.Installation = "SOMETIMES"
		gameFrameHooks["EndRun"] = hook
	},
	// A hook behind an override that names nothing to opt in with.
	"override_hook_names_no_capability": func(_ *expectedSurface, _ *actualSurface) {
		hook := gameFrameHooks["BeginDraw"]
		hook.Capability = ""
		gameFrameHooks["BeginDraw"] = hook
	},
	// And a hook that is never installed but still claims a capability, which
	// would be an opt-in that opts into nothing.
	"never_installed_hook_names_a_capability": func(_ *expectedSurface, _ *actualSurface) {
		hook := gameFrameHooks["EndDraw"]
		hook.Installation = "NEVER"
		hook.ReasonUninstalled = "not today"
		gameFrameHooks["EndDraw"] = hook
	},

	// ---- base invocation -----------------------------------------------
	// CNA-Go never runs a base body on the consumer's behalf, at any position.
	"base_invoked_automatically": func(_ *expectedSurface, _ *actualSurface) {
		hook := gameFrameHooks["BeginRun"]
		hook.BaseInvocation = "AUTOMATIC"
		gameFrameHooks["BeginRun"] = hook
	},
	"base_invocation_records_no_evidence": func(_ *expectedSurface, _ *actualSurface) {
		hook := gameFrameHooks["EndDraw"]
		hook.BaseInvocationEvidence = ""
		gameFrameHooks["EndDraw"] = hook
	},

	// ---- the capability identities -------------------------------------
	// An exported capability publishes a new public framework contract, which
	// is the whole thing a structural interface exists to avoid.
	"capability_is_exported": func(_ *expectedSurface, _ *actualSurface) {
		hook := gameFrameHooks["BeginDraw"]
		hook.Capability = "GameBeginDrawOverride"
		gameFrameHooks["BeginDraw"] = hook
	},
	// An exported twin beside the private one is the same publication by
	// another route.
	"exported_capability_type_exists_beside_it": func(_ *expectedSurface, actual *actualSurface) {
		key := symbolKey{Package: modulePath + "/Microsoft/Xna/Framework", Name: "GameBeginDrawOverride"}
		actual.Types[key] = &actualType{Key: key, Kind: "interface"}
	},
	// Two hooks sharing one identity cannot express two independent overrides.
	"two_hooks_share_one_capability": func(_ *expectedSurface, _ *actualSurface) {
		hook := gameFrameHooks["EndRun"]
		hook.Capability = gameFrameHooks["BeginRun"].Capability
		gameFrameHooks["EndRun"] = hook
	},
	// A capability the package does not declare cannot be satisfied at all.
	"capability_does_not_exist": func(_ *expectedSurface, _ *actualSurface) {
		hook := gameFrameHooks["EndRun"]
		hook.Capability = "gameNoSuchOverride"
		gameFrameHooks["EndRun"] = hook
	},
	// THE bundled-contract control. One interface carrying more than one hook
	// forces a consumer who overrode one virtual to supply no-ops for the
	// others, and a no-op override still installs a hook and takes the base's
	// place at that frame position.
	"capability_bundles_more_than_one_hook": func(_ *expectedSurface, actual *actualSurface) {
		actual.Packages[modulePath+"/Microsoft/Xna/Framework"] = frameHookCapabilityPackageWith(
			func(pkg *types.Package, game *types.Named) {
				insertFrameHookInterface(pkg, "gameBeginRunOverride",
					types.NewFunc(token.NoPos, pkg, "BeginRun", frameHookCapabilitySignature(pkg, game, errorInterfaceType())),
					types.NewFunc(token.NoPos, pkg, "EndRun", frameHookCapabilitySignature(pkg, game, errorInterfaceType())),
				)
			})
	},
	// A capability whose one method is unexported can never be satisfied from
	// another package, so no external consumer could opt in.
	"capability_method_is_unexported": func(_ *expectedSurface, actual *actualSurface) {
		actual.Packages[modulePath+"/Microsoft/Xna/Framework"] = frameHookCapabilityPackageWith(
			func(pkg *types.Package, game *types.Named) {
				insertFrameHookInterface(pkg, "gameEndDrawOverride",
					types.NewFunc(token.NoPos, pkg, "endDraw", frameHookCapabilitySignature(pkg, game, errorInterfaceType())),
				)
			})
	},
	// A capability that is not an interface cannot be satisfied structurally.
	"capability_is_not_an_interface": func(_ *expectedSurface, actual *actualSurface) {
		actual.Packages[modulePath+"/Microsoft/Xna/Framework"] = frameHookCapabilityPackageWith(
			func(pkg *types.Package, _ *types.Named) {
				name := types.NewTypeName(token.NoPos, pkg, "gameEndRunOverride", nil)
				types.NewNamed(name, types.NewStruct(nil, nil), nil)
				pkg.Scope().Insert(name)
			})
	},
	// The declared method name is what a consumer has to write; a mismatch
	// means the registry documents an opt-in nobody can perform.
	"capability_method_name_is_not_the_hook": func(_ *expectedSurface, _ *actualSurface) {
		hook := gameFrameHooks["BeginDraw"]
		hook.CapabilityMethod = "Begin"
		gameFrameHooks["BeginDraw"] = hook
	},
	// The override must receive the owning Game; without it there is nothing
	// to call the base on.
	"capability_does_not_take_the_owning_game": func(_ *expectedSurface, _ *actualSurface) {
		hook := gameFrameHooks["EndDraw"]
		hook.CapabilityParameters = nil
		gameFrameHooks["EndDraw"] = hook
	},
	// THE value-channel control. An override whose results differ from the base
	// body it replaces has lost or invented a channel; for BeginDraw that means
	// collapsing the drawing decision into the error.
	"capability_results_differ_from_the_base_body": func(_ *expectedSurface, _ *actualSurface) {
		hook := gameFrameHooks["BeginDraw"]
		hook.CapabilityResults = []string{"error"}
		gameFrameHooks["BeginDraw"] = hook
	},
	// And the same disagreement seen from the compiler side rather than the
	// registry side.
	"capability_compiler_shape_disagrees_with_the_registry": func(_ *expectedSurface, actual *actualSurface) {
		actual.Packages[modulePath+"/Microsoft/Xna/Framework"] = frameHookCapabilityPackageWith(
			func(pkg *types.Package, game *types.Named) {
				declareFrameHookCapability(pkg, game, "gameBeginDrawOverride", "BeginDraw", errorInterfaceType())
			})
	},

	// ---- no registration surface, no sixth mandatory member --------------
	// A registration function is mutable per-Game callback state by another
	// name, and it makes the override set something that can change under a
	// running frame loop.
	"public_registration_function_exists": func(_ *expectedSurface, actual *actualSurface) {
		key := symbolKey{Package: modulePath + "/Microsoft/Xna/Framework", Name: "SetGameBeginDrawOverride"}
		actual.Members[key] = &actualMember{Key: key, Kind: "func", Parameters: []string{"*Game", "any"}, Results: []string{"error"}}
	},
	"public_registration_method_exists": func(_ *expectedSurface, actual *actualSurface) {
		key := symbolKey{Package: modulePath + "/Microsoft/Xna/Framework", Receiver: "Game", Name: "RegisterBeginDrawOverride"}
		actual.Members[key] = &actualMember{Key: key, Kind: "method", Parameters: []string{"any"}, Results: []string{"error"}}
	},
	// A sixth mandatory member breaks every external GameCallbacks
	// implementation that already exists, which is the reason the overrides are
	// optional structural capabilities in the first place.
	"gamecallbacks_gains_a_sixth_member": func(_ *expectedSurface, actual *actualSurface) {
		key := symbolKey{Package: modulePath + "/Microsoft/Xna/Framework", Receiver: "GameCallbacks", Name: "BeginDraw"}
		actual.Members[key] = &actualMember{Key: key, Kind: "method", Parameters: []string{"*Game"}, Results: []string{"bool", "error"}}
	},
	"gamecallbacks_loses_a_member": func(_ *expectedSurface, actual *actualSurface) {
		delete(actual.Members, symbolKey{Package: modulePath + "/Microsoft/Xna/Framework", Receiver: "GameCallbacks", Name: "Update"})
	},
	// An embedded contract would make GameCallbacks' member count a fiction: a
	// capability could arrive through the mandatory interface after all.
	"gamecallbacks_embeds_another_contract": func(_ *expectedSurface, actual *actualSurface) {
		key := symbolKey{Package: modulePath + "/Microsoft/Xna/Framework", Name: "GameCallbacks"}
		actual.Types[key] = &actualType{Key: key, Kind: "interface", ExportedEmbeddings: []string{"GameFrameHookCallbacks"}}
	},
	// THE anonymous-callback-field control. An exported or embedded field on
	// Game is a callback slot anything could be written into after
	// construction, which is precisely the mutable state this design has none
	// of.
	"game_carries_an_exported_callback_field": func(_ *expectedSurface, actual *actualSurface) {
		key := symbolKey{Package: modulePath + "/Microsoft/Xna/Framework", Name: "Game"}
		actual.Types[key] = &actualType{Key: key, Kind: "struct", Fields: []actualField{
			{Name: "Overrides", Type: "any", Exported: true},
		}}
	},
	"game_embeds_a_callback_field": func(_ *expectedSurface, actual *actualSurface) {
		key := symbolKey{Package: modulePath + "/Microsoft/Xna/Framework", Name: "Game"}
		actual.Types[key] = &actualType{Key: key, Kind: "struct", Fields: []actualField{
			{Name: "gameBeginDrawOverride", Type: "gameBeginDrawOverride", Embedded: true},
		}}
	},
}

// TestGameFrameHookOverrideDefectsAreRejected attacks every rule the optional
// per-hook override mechanism rests on.
func TestGameFrameHookOverrideDefectsAreRejected(t *testing.T) {
	baselineExpected, baselineActual := gameSignalFixture(t)
	baseline := verify(baselineExpected, baselineActual, 0, "report", "contract", "mapping")
	if baseline.Summary["LANGUAGE_MAPPING_MISMATCH"] != 0 {
		t.Fatalf("the unmutated fixture is not clean: %d LANGUAGE_MAPPING_MISMATCH", baseline.Summary["LANGUAGE_MAPPING_MISMATCH"])
	}
	if baseline.Summary["GAME_FRAME_HOOKS_INSTALLED_ON_OVERRIDE"] != len(gameFrameHooks) {
		t.Fatalf("fixture measured %d hooks installed behind an override, registry declares %d",
			baseline.Summary["GAME_FRAME_HOOKS_INSTALLED_ON_OVERRIDE"], len(gameFrameHooks))
	}
	if baseline.Summary["GAME_FRAME_HOOK_OVERRIDE_CAPABILITIES"] != len(gameFrameHooks) {
		t.Fatalf("fixture measured %d capabilities, want one per hook", baseline.Summary["GAME_FRAME_HOOK_OVERRIDE_CAPABILITIES"])
	}
	if baseline.Summary["GAME_FRAME_HOOKS_NEVER_INSTALLED"] != 0 {
		t.Fatalf("fixture measured %d hooks that are never installed", baseline.Summary["GAME_FRAME_HOOKS_NEVER_INSTALLED"])
	}
	if baseline.Summary["GAME_CALLBACKS_MEMBERS"] != 5 {
		t.Fatalf("fixture measured GameCallbacks with %d members, want exactly 5", baseline.Summary["GAME_CALLBACKS_MEMBERS"])
	}

	names := make([]string, 0, len(gameFrameHookOverrideDefects))
	for name := range gameFrameHookOverrideDefects {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		defect := gameFrameHookOverrideDefects[name]
		t.Run(name, func(t *testing.T) {
			result := gameFrameHookOverrideMutation(t, name, defect)
			if result.Summary["LANGUAGE_MAPPING_MISMATCH"] == 0 {
				t.Fatalf("override defect %q raised no LANGUAGE_MAPPING_MISMATCH", name)
			}
		})
	}
}

func gameFrameHookOverrideMutation(t *testing.T, name string, defect func(*expectedSurface, *actualSurface)) report {
	t.Helper()
	expected, actual := gameSignalFixture(t)
	var result report
	withGameFrameHooks(t, func() { defect(expected, actual) }, func() {
		result = verify(expected, actual, 0, "report", "contract", "mapping")
	})
	return result
}

// gameFrameHookOverrideMutationCase applies one Foundation-38 defect to a fresh
// fixture, so the shared table drives both the named test and the inventory.
func gameFrameHookOverrideMutationCase(t *testing.T, mutation string) report {
	t.Helper()
	name := strings.TrimPrefix(mutation, "f38hook_")
	defect, ok := gameFrameHookOverrideDefects[name]
	if !ok {
		t.Fatalf("unknown frame-hook override defect %q", name)
	}
	return gameFrameHookOverrideMutation(t, name, defect)
}

// TestEveryFrameHookOverrideDefectHasAMutationFixture keeps the shared table
// and the mutation inventory from drifting.
func TestEveryFrameHookOverrideDefectHasAMutationFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/mutations.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []mutationFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	inventoried := make(map[string]bool)
	for _, fixture := range fixtures {
		if strings.HasPrefix(fixture.Mutation, "f38hook_") {
			inventoried[strings.TrimPrefix(fixture.Mutation, "f38hook_")] = true
		}
	}
	for name := range gameFrameHookOverrideDefects {
		if !inventoried[name] {
			t.Fatalf("frame-hook override defect %q has no mutation fixture", name)
		}
	}
	for name := range inventoried {
		if _, declared := gameFrameHookOverrideDefects[name]; !declared {
			t.Fatalf("mutation fixture f38hook_%s names no defect in the shared table", name)
		}
	}
}

// TestTheOverrideMechanismAddsNoXNAIdentity is the accounting claim. The four
// capabilities are Go language support: they are unexported, they are not
// members of any projected type, and they must move no identity counter.
func TestTheOverrideMechanismAddsNoXNAIdentity(t *testing.T) {
	expected, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	for name, hook := range gameFrameHooks {
		if hook.Capability == "" {
			continue
		}
		for key, member := range expected.Members {
			if member.GoName == hook.Capability || key.Receiver == hook.Capability {
				t.Fatalf("capability %s for hook %s appears in the expected XNA surface as %s", hook.Capability, name, key.String())
			}
		}
		if expected.typeForXNA("Microsoft.Xna.Framework."+hook.Capability) != nil {
			t.Fatalf("capability %s is an XNA identity", hook.Capability)
		}
	}
}

// TestGameSignalDefectsAreRejected attacks every rule the native game-signal
// bridge and the frame-hook frontier rest on.
//
// The danger both registries defend against is the same one the base-call
// family has: not a wrong XNA signature, but an INVENTED claim -- a raise site
// the reference never declared, an event bound to nothing, a native hook
// silently left out, or a runtime evidence label with no evidence behind it.
func TestGameSignalDefectsAreRejected(t *testing.T) {
	baselineExpected, baselineActual := gameSignalFixture(t)
	baseline := verify(baselineExpected, baselineActual, 0, "report", "contract", "mapping")
	if baseline.Summary["LANGUAGE_MAPPING_MISMATCH"] != 0 {
		t.Fatalf("the unmutated signal fixture is not clean: %d LANGUAGE_MAPPING_MISMATCH", baseline.Summary["LANGUAGE_MAPPING_MISMATCH"])
	}
	if baseline.Summary["GAME_NATIVE_SIGNALS"] != len(gameNativeSignals) {
		t.Fatalf("fixture measured %d signals, registry declares %d",
			baseline.Summary["GAME_NATIVE_SIGNALS"], len(gameNativeSignals))
	}
	if baseline.Summary["GAME_FRAME_HOOKS"] != len(gameFrameHooks) {
		t.Fatalf("fixture measured %d frame hooks, registry declares %d",
			baseline.Summary["GAME_FRAME_HOOKS"], len(gameFrameHooks))
	}
	if baseline.Summary["GAME_FRAME_HOOKS_NEVER_INSTALLED"] != 0 {
		t.Fatalf("fixture measured %d frame hooks that are never installed; all four are installed behind an override",
			baseline.Summary["GAME_FRAME_HOOKS_NEVER_INSTALLED"])
	}

	names := make([]string, 0, len(gameSignalDefects))
	for name := range gameSignalDefects {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		defect := gameSignalDefects[name]
		t.Run(name, func(t *testing.T) {
			expected, actual := gameSignalFixture(t)
			var result report
			withGameNativeSignals(t, func() {}, func() {
				withGameFrameHooks(t, func() { defect(expected, actual) }, func() {
					result = verify(expected, actual, 0, "report", "contract", "mapping")
				})
			})
			if result.Summary["LANGUAGE_MAPPING_MISMATCH"] == 0 {
				t.Fatalf("signal defect %q raised no LANGUAGE_MAPPING_MISMATCH", name)
			}
		})
	}
}

// gameSignalMutationCase applies one Foundation-36 defect to a fresh fixture
// and returns the resulting report, so the shared defect table drives both the
// named test and the mutation inventory.
func gameSignalMutationCase(t *testing.T, mutation string) report {
	t.Helper()
	name := strings.TrimPrefix(mutation, "f36signal_")
	defect, ok := gameSignalDefects[name]
	if !ok {
		t.Fatalf("unknown game-signal defect %q", name)
	}
	expected, actual := gameSignalFixture(t)
	var result report
	withGameNativeSignals(t, func() {}, func() {
		withGameFrameHooks(t, func() { defect(expected, actual) }, func() {
			result = verify(expected, actual, 0, "report", "contract", "mapping")
		})
	})
	return result
}

// TestEveryGameSignalDefectHasAMutationFixture keeps the shared defect table
// and the mutation inventory from drifting.
func TestEveryGameSignalDefectHasAMutationFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/mutations.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []mutationFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	inventoried := make(map[string]bool)
	for _, fixture := range fixtures {
		if strings.HasPrefix(fixture.Mutation, "f36signal_") {
			inventoried[strings.TrimPrefix(fixture.Mutation, "f36signal_")] = true
		}
	}
	for name := range gameSignalDefects {
		if !inventoried[name] {
			t.Fatalf("game-signal defect %q has no mutation fixture", name)
		}
	}
	for name := range inventoried {
		if _, declared := gameSignalDefects[name]; !declared {
			t.Fatalf("mutation fixture f36signal_%s names no defect in the shared table", name)
		}
	}
}

// TestGameNativeSignalsAndFrameHooksAreNotXNAIdentities is the accounting
// claim: both registries measure an existing projection and must not inflate
// any identity counter. The members they name are already counted as Game's
// own; the registries only prove the binding around them.
func TestGameNativeSignalsAndFrameHooksAreNotXNAIdentities(t *testing.T) {
	expected, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	referenceMembers := expected.ReferenceMembers
	expectedMembers := len(expected.Members)
	gameType := expected.typeForXNA("Microsoft.Xna.Framework.Game")
	if gameType == nil {
		t.Fatal("the pinned contract does not declare Game")
	}
	gameMembers := len(gameType.Members)

	_, actual := gameSignalFixture(t)
	result := verify(expected, actual, 0, "report", "contract", "mapping")
	if result.Summary["GAME_NATIVE_SIGNALS"] != len(gameNativeSignals) {
		t.Fatalf("measured %d signals, registry declares %d", result.Summary["GAME_NATIVE_SIGNALS"], len(gameNativeSignals))
	}
	if expected.ReferenceMembers != referenceMembers {
		t.Fatalf("REFERENCE_MEMBERS moved from %d to %d", referenceMembers, expected.ReferenceMembers)
	}
	if len(expected.Members) != expectedMembers {
		t.Fatalf("EXPECTED_GO_MEMBERS moved from %d to %d", expectedMembers, len(expected.Members))
	}
	if len(gameType.Members) != gameMembers {
		t.Fatalf("Game's projected member set moved from %d to %d", gameMembers, len(gameType.Members))
	}
}

// TestMappingRulesDeclareTheSameGameSignalsAsTheRegistry keeps the documented
// rules file and the executable registry from drifting.
//
// mapping-rules.json is hashed into every report, so it is the record of what
// the binding claims its rules are; gameNativeSignals is what the verifier
// actually enforces. A signal documented in one and not the other would let a
// binding claim a raise path nobody checks, or check one nobody published.
func TestMappingRulesDeclareTheSameGameSignalsAsTheRegistry(t *testing.T) {
	data, err := os.ReadFile("mapping-rules.json")
	if err != nil {
		t.Fatal(err)
	}
	var rules struct {
		GameNativeSignals struct {
			Senders          map[string]string `json:"senders"`
			RuntimeEvidence  map[string]string `json:"runtimeEvidence"`
			RaisePath        map[string]string `json:"raisePath"`
			NativeSignalRole map[string]string `json:"nativeSignalRole"`
			Signals          map[string]struct {
				CNAConstant      string `json:"cnaConstant"`
				CNAIdentity      int    `json:"cnaIdentity"`
				CLREvent         string `json:"clrEvent"`
				RaiseSite        string `json:"raiseSite"`
				Sender           string `json:"sender"`
				EdgeTriggered    bool   `json:"edgeTriggered"`
				RuntimeEvidence  string `json:"runtimeEvidence"`
				RaisePath        string `json:"raisePath"`
				NativeSignalRole string `json:"nativeSignalRole"`
				ManagedRaiseSite string `json:"managedRaiseSite"`
			} `json:"signals"`
		} `json:"gameNativeSignals"`
		GameFrameHooks struct {
			Installation   map[string]string `json:"installation"`
			BaseInvocation map[string]string `json:"baseInvocation"`
			Hooks          map[string]struct {
				CLRMember           string `json:"clrMember"`
				Signature           string `json:"signature"`
				NativeHook          string `json:"nativeHook"`
				Installation        string `json:"installation"`
				BaseInvocation      string `json:"baseInvocation"`
				Capability          string `json:"capability"`
				CapabilitySignature string `json:"capabilitySignature"`
			} `json:"hooks"`
		} `json:"gameFrameHooks"`
	}
	if err := json.Unmarshal(data, &rules); err != nil {
		t.Fatal(err)
	}

	documented := rules.GameNativeSignals.Signals
	if len(documented) != len(gameNativeSignals) {
		t.Fatalf("mapping-rules.json documents %d native signals, the registry declares %d",
			len(documented), len(gameNativeSignals))
	}
	for name, signal := range gameNativeSignals {
		entry, present := documented[name]
		if !present {
			t.Fatalf("native signal %q is in the registry but not in mapping-rules.json", name)
		}
		if entry.CNAConstant != signal.CNAConstant || entry.CNAIdentity != signal.CNAIdentity {
			t.Fatalf("native signal %q: rules say %s=%d, registry says %s=%d",
				name, entry.CNAConstant, entry.CNAIdentity, signal.CNAConstant, signal.CNAIdentity)
		}
		if entry.CLREvent != signal.CLREvent {
			t.Fatalf("native signal %q: rules say CLR event %q, registry says %q", name, entry.CLREvent, signal.CLREvent)
		}
		if entry.RaiseSite != signal.RaiseSite {
			t.Fatalf("native signal %q: rules say raise site %q, registry says %q", name, entry.RaiseSite, signal.RaiseSite)
		}
		if entry.Sender != signal.Sender {
			t.Fatalf("native signal %q: rules say sender %q, registry says %q", name, entry.Sender, signal.Sender)
		}
		if entry.EdgeTriggered != signal.EdgeTriggered {
			t.Fatalf("native signal %q: rules say edgeTriggered=%t, registry says %t", name, entry.EdgeTriggered, signal.EdgeTriggered)
		}
		if entry.RuntimeEvidence != signal.RuntimeEvidence {
			t.Fatalf("native signal %q: rules say evidence %q, registry says %q", name, entry.RuntimeEvidence, signal.RuntimeEvidence)
		}
		if entry.RaisePath != signal.RaisePath {
			t.Fatalf("native signal %q: rules say raise path %q, registry says %q", name, entry.RaisePath, signal.RaisePath)
		}
		if entry.NativeSignalRole != signal.NativeSignalRole {
			t.Fatalf("native signal %q: rules say signal role %q, registry says %q", name, entry.NativeSignalRole, signal.NativeSignalRole)
		}
		if entry.ManagedRaiseSite != signal.ManagedRaiseSite {
			t.Fatalf("native signal %q: rules say managed raise site %q, registry says %q", name, entry.ManagedRaiseSite, signal.ManagedRaiseSite)
		}
	}
	// The two Foundation-39 vocabularies are closed and must be documented
	// exactly as the verifier admits them.
	for class := range gameEventRaisePaths {
		if _, documented := rules.GameNativeSignals.RaisePath[class]; !documented {
			t.Fatalf("raise path %q is admitted by the verifier but not documented", class)
		}
	}
	if len(rules.GameNativeSignals.RaisePath) != len(gameEventRaisePaths) {
		t.Fatalf("mapping-rules.json documents %d raise paths, the verifier admits %d",
			len(rules.GameNativeSignals.RaisePath), len(gameEventRaisePaths))
	}
	for class := range gameNativeSignalRoles {
		if _, documented := rules.GameNativeSignals.NativeSignalRole[class]; !documented {
			t.Fatalf("native signal role %q is admitted by the verifier but not documented", class)
		}
	}
	if len(rules.GameNativeSignals.NativeSignalRole) != len(gameNativeSignalRoles) {
		t.Fatalf("mapping-rules.json documents %d signal roles, the verifier admits %d",
			len(rules.GameNativeSignals.NativeSignalRole), len(gameNativeSignalRoles))
	}
	// The two closed vocabularies must be documented exactly as the verifier
	// enforces them, so neither can gain a class in prose alone.
	for sender := range gameNativeSignalSenders {
		if _, documented := rules.GameNativeSignals.Senders[sender]; !documented {
			t.Fatalf("sender %q is admitted by the verifier but not documented", sender)
		}
	}
	if len(rules.GameNativeSignals.Senders) != len(gameNativeSignalSenders) {
		t.Fatalf("mapping-rules.json documents %d senders, the verifier admits %d",
			len(rules.GameNativeSignals.Senders), len(gameNativeSignalSenders))
	}
	for evidence := range gameNativeSignalEvidence {
		if _, documented := rules.GameNativeSignals.RuntimeEvidence[evidence]; !documented {
			t.Fatalf("runtime evidence class %q is admitted by the verifier but not documented", evidence)
		}
	}
	if len(rules.GameNativeSignals.RuntimeEvidence) != len(gameNativeSignalEvidence) {
		t.Fatalf("mapping-rules.json documents %d evidence classes, the verifier admits %d",
			len(rules.GameNativeSignals.RuntimeEvidence), len(gameNativeSignalEvidence))
	}

	// The installation vocabulary is closed and must be documented exactly as
	// the verifier admits it, so a class cannot be added in prose alone.
	for class := range gameFrameHookInstallations {
		if _, documented := rules.GameFrameHooks.Installation[class]; !documented {
			t.Fatalf("installation class %q is admitted by the verifier but not documented", class)
		}
	}
	if len(rules.GameFrameHooks.Installation) != len(gameFrameHookInstallations) {
		t.Fatalf("mapping-rules.json documents %d installation classes, the verifier admits %d",
			len(rules.GameFrameHooks.Installation), len(gameFrameHookInstallations))
	}
	for class := range gameFrameHookBaseInvocations {
		if _, documented := rules.GameFrameHooks.BaseInvocation[class]; !documented {
			t.Fatalf("base-invocation class %q is admitted by the verifier but not documented", class)
		}
	}
	if len(rules.GameFrameHooks.BaseInvocation) != len(gameFrameHookBaseInvocations) {
		t.Fatalf("mapping-rules.json documents %d base-invocation classes, the verifier admits %d",
			len(rules.GameFrameHooks.BaseInvocation), len(gameFrameHookBaseInvocations))
	}

	documentedHooks := rules.GameFrameHooks.Hooks
	if len(documentedHooks) != len(gameFrameHooks) {
		t.Fatalf("mapping-rules.json documents %d frame hooks, the registry declares %d",
			len(documentedHooks), len(gameFrameHooks))
	}
	for name, hook := range gameFrameHooks {
		entry, present := documentedHooks[name]
		if !present {
			t.Fatalf("frame hook %q is in the registry but not in mapping-rules.json", name)
		}
		if entry.CLRMember != hook.CLRMember {
			t.Fatalf("frame hook %q: rules say CLR member %q, registry says %q", name, entry.CLRMember, hook.CLRMember)
		}
		if entry.NativeHook != hook.NativeHook {
			t.Fatalf("frame hook %q: rules say native hook %q, registry says %q", name, entry.NativeHook, hook.NativeHook)
		}
		if entry.Installation != hook.Installation {
			t.Fatalf("frame hook %q: rules say installation=%q, registry says %q", name, entry.Installation, hook.Installation)
		}
		if entry.BaseInvocation != hook.BaseInvocation {
			t.Fatalf("frame hook %q: rules say baseInvocation=%q, registry says %q", name, entry.BaseInvocation, hook.BaseInvocation)
		}
		if entry.Capability != hook.Capability {
			t.Fatalf("frame hook %q: rules say capability %q, registry says %q", name, entry.Capability, hook.Capability)
		}
		// The documented capability signature must spell the registry's exact
		// method, so an override contract cannot widen or lose a channel in
		// prose. BeginDraw's Boolean is the one that matters most.
		wantedCapability := ""
		if hook.Capability != "" {
			wantedCapability = hook.CapabilityMethod + "(" + strings.Join(hook.CapabilityParameters, ", ") + ")"
			switch len(hook.CapabilityResults) {
			case 0:
			case 1:
				wantedCapability += " " + hook.CapabilityResults[0]
			default:
				wantedCapability += " (" + strings.Join(hook.CapabilityResults, ", ") + ")"
			}
		}
		if entry.CapabilitySignature != wantedCapability {
			t.Fatalf("frame hook %q: rules document capability signature %q, registry implies %q",
				name, entry.CapabilitySignature, wantedCapability)
		}
		// The documented signature must spell the registry's exact results, so
		// a widened one cannot hide behind prose. In particular BeginDraw's
		// Boolean must stay a separate channel from its error.
		wanted := "func (g *Game) " + hook.GoName + "()"
		switch len(hook.Results) {
		case 0:
		case 1:
			wanted += " " + hook.Results[0]
		default:
			wanted += " (" + strings.Join(hook.Results, ", ") + ")"
		}
		if entry.Signature != wanted {
			t.Fatalf("frame hook %q: rules document signature %q, registry implies %q", name, entry.Signature, wanted)
		}
	}
}

// ---------------------------------------------------------------------------
// Foundation 40 — the base-typed public signature inventory.
// ---------------------------------------------------------------------------

// TestXNABaseSubstitutabilityIsDerivedFromTheContract proves the inventory is a
// measurement and not a table: the relationships it walks come from the pinned
// contract's own baseType fields, and they must agree exactly with the
// hand-written registry.
func TestXNABaseSubstitutabilityIsDerivedFromTheContract(t *testing.T) {
	expected, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(expected.XNABaseDerivedByBase) != len(xnaBaseRelationships) {
		t.Fatalf("the contract declares %d XNA base relationships, the registry records %d",
			len(expected.XNABaseDerivedByBase), len(xnaBaseRelationships))
	}
	derivedTotal := 0
	for base, derived := range expected.XNABaseDerivedByBase {
		if _, declared := xnaBaseRelationships[base]; !declared {
			t.Fatalf("the contract declares classes deriving from %s and the registry does not record it", base)
		}
		derivedTotal += len(derived)
	}
	if derivedTotal != 41 {
		t.Fatalf("the contract declares %d XNA-derived types, the pinned count is 41", derivedTotal)
	}
	// Every recorded position must name a real base and a real carrier, and
	// must carry the CLR type it was read from.
	for _, row := range expected.XNABaseSubstitutability {
		if _, isBase := expected.XNABaseDerivedByBase[row.Base]; !isBase {
			t.Fatalf("position on %s.%s names %q, which no class derives from", row.Carrier, row.Member, row.Base)
		}
		if expected.typeForXNA(row.Carrier) == nil {
			t.Fatalf("position names carrier %q, which the contract does not declare", row.Carrier)
		}
		if row.CLRType == "" || row.Position == "" {
			t.Fatalf("position on %s.%s records no CLR type or no position", row.Carrier, row.Member)
		}
	}
}

// TestTheThreeFamiliesWithNoSubstitutabilityRequirement is the measurement the
// XNA inheritance architecture turns on.
//
// GameComponent, GraphicsResource and MathTypeConverter carry 25 of the 41
// derived types between them, and NOT ONE public signature in the whole profile
// names any of the three. No consumer can ever be handed a signature that
// requires a DrawableGameComponent to stand in for a GameComponent, because no
// such signature exists. Private named composition with explicit forwarding is
// therefore not a compromise for these families -- it is exactly sufficient,
// and no public reference abstraction can be justified by anything in the
// contract.
func TestTheThreeFamiliesWithNoSubstitutabilityRequirement(t *testing.T) {
	expected, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	positions := make(map[string]int)
	for _, row := range expected.XNABaseSubstitutability {
		positions[row.Base]++
	}
	free := map[string]int{
		"Microsoft.Xna.Framework.GameComponent":             2,
		"Microsoft.Xna.Framework.Graphics.GraphicsResource": 11,
		"Microsoft.Xna.Framework.Design.MathTypeConverter":  12,
	}
	covered := 0
	for base, wantDerived := range free {
		if got := positions[base]; got != 0 {
			t.Fatalf("%s is named in %d public positions; the architecture rests on it being named in none", base, got)
		}
		if got := len(expected.XNABaseDerivedByBase[base]); got != wantDerived {
			t.Fatalf("%s has %d derived types, want %d", base, got, wantDerived)
		}
		covered += wantDerived
	}
	if covered != 25 {
		t.Fatalf("the three requirement-free families cover %d derived types, want 25", covered)
	}
}

// TestNoBaseFamilyHasALiveSubstitutabilityRequirementYet records the state the
// whole profile is in, so the day it changes is the day this test says so.
//
// A LIVE requirement needs both ends: a projected carrier naming the base, and
// a projected derived type. Texture2D is the closest -- 17 positions, nine of
// them on SpriteBatch, which CNA-Go projects -- and it is still LATENT only
// because its one derived type, RenderTarget2D, is not projected. Projecting
// RenderTarget2D while SpriteBatch.Draw takes a Texture2D is exactly what would
// make it live.
func TestNoBaseFamilyHasALiveSubstitutabilityRequirementYet(t *testing.T) {
	expected, actual := loadPinnedSurfaces(t)
	result := verify(expected, actual, 0, "report", "contract", "mapping")
	if got := result.Summary["XNA_BASE_SUBSTITUTABILITY_LIVE"]; got != 0 {
		var live []string
		for _, measurement := range result.XNABaseSubstitutability {
			if measurement.Requirement == "LIVE" {
				live = append(live, measurement.Base)
			}
		}
		t.Fatalf("%d base families have a live substitutability requirement: %v. Private composition is no longer "+
			"sufficient for them, and the public reference abstraction that has been deferred must now be decided", got, live)
	}
	if got := result.Summary["XNA_BASE_SUBSTITUTABILITY_NONE"]; got != 3 {
		t.Fatalf("%d families have no substitutability requirement, want 3", got)
	}
	if got := result.Summary["XNA_BASE_SUBSTITUTABILITY_LATENT"]; got != 9 {
		t.Fatalf("%d families have a latent requirement, want 9", got)
	}
	if got := result.Summary["XNA_BASE_TYPED_SIGNATURE_POSITIONS"]; got != 51 {
		t.Fatalf("%d base-typed public signature positions, want 51", got)
	}
	// Every family carries exactly one classified requirement.
	for _, measurement := range result.XNABaseSubstitutability {
		if _, classified := xnaBaseSubstitutabilityRequirements[measurement.Requirement]; !classified {
			t.Fatalf("%s carries unclassified requirement %q", measurement.Base, measurement.Requirement)
		}
		if measurement.Positions != len(measurement.Rows) {
			t.Fatalf("%s reports %d positions and lists %d", measurement.Base, measurement.Positions, len(measurement.Rows))
		}
	}
}

// TestCLRTypeIdentitiesFindsEveryPositionAValueCouldFlowThrough pins the scanner
// the inventory rests on. A base named behind an array, a by-reference marker or
// a generic argument is still a position a derived value has to flow through,
// and missing any of those would silently shrink the requirement.
func TestCLRTypeIdentitiesFindsEveryPositionAValueCouldFlowThrough(t *testing.T) {
	for raw, want := range map[string][]string{
		"Microsoft.Xna.Framework.Graphics.Texture2D":   {"Microsoft.Xna.Framework.Graphics.Texture2D"},
		"Microsoft.Xna.Framework.Graphics.Texture2D[]": {"Microsoft.Xna.Framework.Graphics.Texture2D"},
		"Microsoft.Xna.Framework.Graphics.Texture2D&":  {"Microsoft.Xna.Framework.Graphics.Texture2D"},
		"System.Collections.Generic.List`1[Microsoft.Xna.Framework.GameComponent]": {
			"System.Collections.Generic.List`1", "Microsoft.Xna.Framework.GameComponent",
		},
		"System.Func`2[Microsoft.Xna.Framework.GameComponent,Microsoft.Xna.Framework.Graphics.Effect]": {
			"System.Func`2", "Microsoft.Xna.Framework.GameComponent", "Microsoft.Xna.Framework.Graphics.Effect",
		},
		"": nil,
	} {
		got := clrTypeIdentities(raw)
		if len(got) != len(want) {
			t.Fatalf("%q yielded %v, want %v", raw, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%q yielded %v, want %v", raw, got, want)
			}
		}
	}
}

// TestXNABaseSubstitutabilityDefectsAreRejected attacks the cross-check between
// the derived inventory and the hand-written registry, which is what stops
// either from going stale alone.
func TestXNABaseSubstitutabilityDefectsAreRejected(t *testing.T) {
	for name, mutate := range map[string]func(){
		"registry_forgets_a_relationship_the_contract_declares": func() {
			delete(xnaBaseRelationships, "Microsoft.Xna.Framework.GameComponent")
		},
		"registry_invents_a_relationship_the_contract_does_not_declare": func() {
			xnaBaseRelationships["Microsoft.Xna.Framework.GameTime"] = xnaBaseRelationship{
				Status: "DEFERRED", Blockers: []xnaBaseBlocker{xnaBaseComposition},
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			expected, actual := loadPinnedSurfaces(t)
			saved := make(map[string]xnaBaseRelationship, len(xnaBaseRelationships))
			for key, value := range xnaBaseRelationships {
				saved[key] = value
			}
			defer func() { xnaBaseRelationships = saved }()
			mutate()
			result := verify(expected, actual, 0, "report", "contract", "mapping")
			if result.Summary["BASE_MAPPING_MISMATCH"] == 0 {
				t.Fatalf("defect %q raised no BASE_MAPPING_MISMATCH", name)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Foundation 41 — the XNA-to-XNA composition rule.
// ---------------------------------------------------------------------------

// xnaCompositionFixture projects DrawableGameComponent into the actual surface
// exactly as the composition rule requires -- a private named *GameComponent
// field and nothing else -- so every defect below is caused by its own
// mutation.
//
// It is synthetic on purpose. CNA-Go does not project DrawableGameComponent
// yet, and its remaining blockers are about device runtime rather than
// inheritance, so the rule has to be proved against a type that satisfies it
// rather than against one that does not exist.
func xnaCompositionFixture(t *testing.T) (*expectedSurface, *actualSurface) {
	t.Helper()
	expected, actual := loadPinnedSurfaces(t)
	derived := expected.typeForXNA("Microsoft.Xna.Framework.DrawableGameComponent")
	if derived == nil {
		t.Fatal("the pinned contract does not declare DrawableGameComponent")
	}
	actual.Types[derived.Key] = &actualType{
		Key: derived.Key, Kind: "struct",
		Fields: []actualField{
			{Name: "base", Type: "*GameComponent"},
			{Name: "visible", Type: "bool"},
			{Name: "drawOrder", Type: "int32"},
		},
	}
	return expected, actual
}

// TestGameComponentIsTheFirstComposedRelationship records the state the
// architecture reached, and why this family and not another.
func TestGameComponentIsTheFirstComposedRelationship(t *testing.T) {
	expected, actual := loadPinnedSurfaces(t)
	result := verify(expected, actual, 0, "report", "contract", "mapping")
	if got := result.Summary["XNA_COMPOSED_BASE_RELATIONSHIPS"]; got != 1 {
		t.Fatalf("%d COMPOSED XNA base relationships, want exactly 1", got)
	}
	if got := result.Summary["XNA_COMPOSED_DERIVED_TYPES"]; got != 2 {
		t.Fatalf("the composed relationship covers %d derived types, want 2", got)
	}
	// The family was chosen because Foundation 40 measured that nothing names
	// it. That is the whole justification, so it is asserted here too.
	for _, measurement := range result.XNABaseSubstitutability {
		if measurement.Base != "Microsoft.Xna.Framework.GameComponent" {
			continue
		}
		if measurement.Requirement != "NONE" {
			t.Fatalf("GameComponent's substitutability requirement is %q; the composition rule was adopted for this family "+
				"because it is NONE, so the justification and the measurement must agree", measurement.Requirement)
		}
	}
	if got := result.Summary["XNA_INHERITED_PUBLIC_MEMBERS"]; got != 14 {
		t.Fatalf("%d inherited public CLR members, want 14", got)
	}
	if got := result.Summary["XNA_INHERITED_MEMBER_PROJECTIONS"]; got != 24 {
		t.Fatalf("%d inherited Go projections, want 24", got)
	}
	if got := result.Summary["XNA_INHERITED_ATTRIBUTED_MEMBERS"]; got != 24 {
		t.Fatalf("%d attributed inherited members, want every one of the 24", got)
	}
}

// TestEveryMemberHasExactlyOneProvenanceClass is the accounting claim the third
// provenance class exists to keep true: XNA_DECLARED, BCL_INHERITED and
// XNA_INHERITED partition the expected surface, and REFERENCE_MEMBERS still
// names exactly what Microsoft declares.
func TestEveryMemberHasExactlyOneProvenanceClass(t *testing.T) {
	expected, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	declared, bclInherited, xnaInherited := 0, 0, 0
	for key, member := range expected.Members {
		switch {
		case member.BCLBase != "" && member.XNABase != "":
			t.Fatalf("%s carries both a BCL base and an XNA base", key.String())
		case member.BCLBase != "":
			bclInherited++
		case member.XNABase != "":
			xnaInherited++
		default:
			declared++
		}
	}
	if declared+bclInherited+xnaInherited != expected.ExpectedGoMembers {
		t.Fatalf("the three provenance classes cover %d members, the surface has %d",
			declared+bclInherited+xnaInherited, expected.ExpectedGoMembers)
	}
	if declared != 3243 {
		t.Fatalf("%d XNA-declared member projections, the pinned count is 3243", declared)
	}
	if bclInherited != expected.BCLInheritedProjections || xnaInherited != expected.XNAInheritedProjections {
		t.Fatalf("provenance walk found %d BCL-inherited and %d XNA-inherited, the surface reports %d and %d",
			bclInherited, xnaInherited, expected.BCLInheritedProjections, expected.XNAInheritedProjections)
	}
	if expected.ReferenceMembers != 2964 {
		t.Fatalf("REFERENCE_MEMBERS moved to %d; the third provenance class must not touch what Microsoft declares", expected.ReferenceMembers)
	}
}

// TestAnOverriddenMemberIsNotAlsoInherited holds the exclusion rule. A derived
// class that redeclares an inherited member is overriding it, so the projected
// member is the DERIVED one and counting it twice would both inflate the
// accounting and claim a forwarding that must not exist.
func TestAnOverriddenMemberIsNotAlsoInherited(t *testing.T) {
	expected, err := buildExpected(loadPinnedContract(t))
	if err != nil {
		t.Fatal(err)
	}
	derived := expected.typeForXNA("Microsoft.Xna.Framework.DrawableGameComponent")
	if derived == nil {
		t.Fatal("the pinned contract does not declare DrawableGameComponent")
	}
	seen := make(map[string]string)
	for _, key := range derived.Members {
		member := expected.Members[key]
		provenance := "XNA_DECLARED"
		if member.XNABase != "" {
			provenance = "XNA_INHERITED"
		}
		if previous, duplicate := seen[member.GoName]; duplicate {
			t.Fatalf("DrawableGameComponent projects %s twice: %s and %s", member.GoName, previous, provenance)
		}
		seen[member.GoName] = provenance
	}
	// Initialize is declared by DrawableGameComponent itself -- it is the
	// override that resolves IGraphicsDeviceService -- so it must be
	// XNA_DECLARED and must NOT also arrive as inherited.
	if got := seen["Initialize"]; got != "XNA_DECLARED" {
		t.Fatalf("DrawableGameComponent::Initialize has provenance %q, want XNA_DECLARED: the derived class declares it", got)
	}
	// Update is not redeclared, so it is inherited and forwarded.
	if got := seen["Update"]; got != "XNA_INHERITED" {
		t.Fatalf("DrawableGameComponent::Update has provenance %q, want XNA_INHERITED", got)
	}
}

// TestXNACompositionDefectsAreRejected attacks every rule the composition
// projection rests on, against a derived type that otherwise satisfies it.
func TestXNACompositionDefectsAreRejected(t *testing.T) {
	derivedKey := func(expected *expectedSurface) symbolKey {
		return expected.typeForXNA("Microsoft.Xna.Framework.DrawableGameComponent").Key
	}
	for name, defect := range map[string]func(*expectedSurface, *actualSurface){
		// THE embedding control. Go embedding promotes the base's whole method
		// set, so a member the derived class overrides would silently keep the
		// base's body wherever the derived one was not redeclared exactly.
		"derived_type_embeds_its_base": func(expected *expectedSurface, actual *actualSurface) {
			actual.Types[derivedKey(expected)].Fields = []actualField{
				{Name: "GameComponent", Type: "*GameComponent", Embedded: true, Exported: true},
			}
		},
		"derived_type_embeds_its_base_unexported": func(expected *expectedSurface, actual *actualSurface) {
			actual.Types[derivedKey(expected)].Fields = []actualField{
				{Name: "gameComponent", Type: "*GameComponent", Embedded: true},
			}
		},
		// An exported base field hands a consumer the base object and lets them
		// mutate it behind the derived type's back.
		"base_is_held_in_an_exported_field": func(expected *expectedSurface, actual *actualSurface) {
			actual.Types[derivedKey(expected)].Fields = []actualField{
				{Name: "Base", Type: "*GameComponent", Exported: true},
			}
		},
		"derived_type_holds_no_base_at_all": func(expected *expectedSurface, actual *actualSurface) {
			actual.Types[derivedKey(expected)].Fields = []actualField{{Name: "visible", Type: "bool"}}
		},
		// A public accessor for the base object. The contract declares none and
		// Foundation 40 measured that no signature in the profile needs one.
		"derived_type_exposes_a_base_accessor": func(expected *expectedSurface, actual *actualSurface) {
			derived := expected.typeForXNA("Microsoft.Xna.Framework.DrawableGameComponent")
			key := symbolKey{Package: derived.PackagePath, Receiver: derived.GoName, Name: "Base"}
			actual.Members[key] = &actualMember{Key: key, Kind: "method", Results: []string{"*GameComponent"}}
		},
		"derived_type_exposes_an_upcast": func(expected *expectedSurface, actual *actualSurface) {
			derived := expected.typeForXNA("Microsoft.Xna.Framework.DrawableGameComponent")
			key := symbolKey{Package: derived.PackagePath, Receiver: derived.GoName, Name: "AsGameComponent"}
			actual.Members[key] = &actualMember{Key: key, Kind: "method", Results: []string{"*GameComponent"}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			expected, actual := xnaCompositionFixture(t)
			before := verify(expected, actual, 0, "report", "contract", "mapping").Summary["BASE_MAPPING_MISMATCH"]
			expected, actual = xnaCompositionFixture(t)
			defect(expected, actual)
			after := verify(expected, actual, 0, "report", "contract", "mapping").Summary["BASE_MAPPING_MISMATCH"]
			if after <= before {
				t.Fatalf("composition defect %q raised no new BASE_MAPPING_MISMATCH (%d -> %d)", name, before, after)
			}
		})
	}
}

// TestComposedRelationshipDefectsAreRejected attacks the registry side of the
// same rule.
func TestComposedRelationshipDefectsAreRejected(t *testing.T) {
	for name, mutate := range map[string]func(){
		"composed_relationship_for_a_type_the_contract_does_not_declare": func() {
			xnaBaseRelationships["Microsoft.Xna.Framework.NotAThing"] = xnaBaseRelationship{Status: "COMPOSED"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			expected, actual := loadPinnedSurfaces(t)
			before := verify(expected, actual, 0, "report", "contract", "mapping").Summary["BASE_MAPPING_MISMATCH"]
			var after int
			withXNABaseRelationships(t, mutate, func() {
				expected, actual := loadPinnedSurfaces(t)
				after = verify(expected, actual, 0, "report", "contract", "mapping").Summary["BASE_MAPPING_MISMATCH"]
			})
			if after <= before {
				t.Fatalf("defect %q raised no new BASE_MAPPING_MISMATCH (%d -> %d)", name, before, after)
			}
		})
	}
}
