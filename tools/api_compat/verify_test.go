package main

import (
	"encoding/json"
	"os"
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
			expected, actual := mutationCase(fixture.Mutation)
			result := verify(expected, actual, 0, "report", "contract", "mapping")
			if result.Summary[fixture.Category] == 0 {
				t.Fatalf("mutation %q did not trigger %s; summary=%v", fixture.Mutation, fixture.Category, result.Summary)
			}
		})
	}
}

func mutationCase(mutation string) (*expectedSurface, *actualSurface) {
	const pkg = modulePath + "/Microsoft/Xna/Framework"
	typeKey := symbolKey{Package: pkg, Name: "Probe"}
	memberKey := symbolKey{Package: pkg, Receiver: "Probe", Name: "Act"}
	et := &expectedType{Key: typeKey, XNA: "Microsoft.Xna.Framework.Probe", GoName: "Probe", PackagePath: pkg, Kind: "struct", Members: []symbolKey{memberKey}}
	em := &expectedMember{Key: memberKey, XNA: "Microsoft.Xna.Framework.Probe::Act(System.Int32)", Owner: et.XNA, SourceKind: "method", GoKind: "method", GoName: "Act", PackagePath: pkg, Receiver: "Probe", Parameters: []string{"int32"}, Results: []string{"bool"}}
	expected := &expectedSurface{Types: map[symbolKey]*expectedType{typeKey: et}, Members: map[symbolKey]*expectedMember{memberKey: em}, ReferenceTypes: 1, ReferenceMembers: 1, ExpectedGoTypes: 1, ExpectedGoMembers: 1}
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
	}
	return expected, actual
}
