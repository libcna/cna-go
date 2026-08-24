package main

import "testing"

func TestValidateRejectsUnknownStatusAndMissingRequiredRows(t *testing.T) {
	value := inventory{SchemaVersion: 1, Profile: "p", QualifiedPlatform: "q"}
	if err := validate(value); err == nil {
		t.Fatal("missing required rows were accepted")
	}
	value.Capabilities = []capability{{ID: "x", Capability: "x", Status: "PASS", Evidence: "e", Notes: "n"}}
	if err := validate(value); err == nil {
		t.Fatal("unknown status was accepted")
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	value := inventory{SchemaVersion: 1, Profile: "p", QualifiedPlatform: "q", Capabilities: []capability{
		{ID: "b", Capability: "b", Status: "VERIFIED_NATIVE", Evidence: "e", Notes: "n"},
		{ID: "a", Capability: "a", Status: "BACKEND_BLOCKED", Evidence: "e", Notes: "n"},
	}}
	first := string(render(value))
	second := string(render(value))
	if first != second {
		t.Fatal("render output changed without input changes")
	}
}
