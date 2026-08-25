package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
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
			if strings.HasPrefix(fixture.Mutation, "f15vs_") {
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
