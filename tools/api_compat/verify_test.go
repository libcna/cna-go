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
			if strings.HasPrefix(fixture.Mutation, "f19rh_") {
				expected, actual = rawHandleMutationCase(t, fixture.Mutation)
				result := verify(expected, actual, 0, "leak-only", "contract", "mapping")
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
	defer func() {
		pureManagedTypes = savedTypes
		classifiedInterfaces = savedInterfaces
		managedFallibleMembers = savedFallible
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
)

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
	if events != 49 || accessors != 98 {
		t.Fatalf("profile events = %d producing %d accessors, want 49 and 98", events, accessors)
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
	return expected, actual, &copiedType
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
		if row.AddsProjectedSurface {
			t.Fatalf("%s claims to add projected surface; a CLR base contributes no Go member identity", row.CLRBase)
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
