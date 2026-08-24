package main

import (
	"fmt"
	"sort"
	"strings"
)

var adapterTypes = map[string]bool{
	"EventSubscription": true,
	"GameCallbacks":     true,
	"TimeSpan":          true,
}

var adapterFunctions = map[string]bool{
	"NewGame":           true,
	"TimeSpanFromTicks": true,
}

func verify(expected *expectedSurface, actual *actualSurface, allowlistEntries int, mode string, contractHash, mappingHash string) report {
	result := report{
		SchemaVersion: 1,
		Profile:       "XNA 4.0 Windows runtime",
		Mode:          mode,
		Summary:       make(map[string]int),
		Metadata: reportMetadata{
			ContractSHA256:  contractHash,
			MappingSHA256:   mappingHash,
			Extractor:       "Go compiler exports + go/parser + go/ast + go/types",
			TypeCheckErrors: len(actual.TypeErrors),
		},
	}
	for _, category := range diagnosticCategories {
		result.Summary[category] = 0
	}
	result.Summary["REFERENCE_TYPES"] = expected.ReferenceTypes
	result.Summary["REFERENCE_MEMBERS"] = expected.ReferenceMembers
	result.Summary["EXPECTED_GO_TYPES"] = expected.ExpectedGoTypes
	result.Summary["EXPECTED_GO_MEMBERS"] = expected.ExpectedGoMembers
	result.Summary["ALLOWLIST_ENTRIES"] = allowlistEntries
	if allowlistEntries > 0 {
		addDiagnostic(&result, diagnostic{Category: "ALLOWLIST_ENTRIES", Message: fmt.Sprintf("mapping allowlist has %d entries", allowlistEntries)})
	}
	for _, source := range actual.Unmeasured {
		addDiagnostic(&result, diagnostic{Category: "UNMEASURED_STRUCTURAL_CATEGORY", Go: source, Message: "source requested an unmeasured structural category"})
	}

	typeDiagnostics := make(map[string]int)
	missingMembers := make(map[string][]string)
	presentTypes := 0
	presentMembers := 0

	for _, et := range sortedExpectedTypes(expected) {
		at, ok := actual.Types[et.Key]
		if !ok {
			addDiagnostic(&result, diagnostic{Category: "MISSING_TYPE", XNA: et.XNA, Go: et.Key.String(), Message: "mapped Go type is absent"})
			typeDiagnostics[et.XNA]++
			result.MissingTypes = append(result.MissingTypes, et.XNA)
			continue
		}
		presentTypes++
		if !typeKindMatches(et, at) {
			addDiagnostic(&result, diagnostic{Category: "TYPE_KIND_MISMATCH", XNA: et.XNA, Go: et.Key.String(), Message: fmt.Sprintf("expected %s projection, found %s (%s)", et.Kind, at.Kind, at.Underlying)})
			typeDiagnostics[et.XNA]++
		}
		if len(et.GenericParameter) != len(at.TypeParameters) || !equalStrings(et.GenericParameter, at.TypeParameters) {
			addDiagnostic(&result, diagnostic{Category: "GENERIC_MAPPING_MISMATCH", XNA: et.XNA, Go: et.Key.String(), Message: fmt.Sprintf("expected type parameters %v, found %v", et.GenericParameter, at.TypeParameters)})
			typeDiagnostics[et.XNA]++
		}
		if et.Flags != at.FlagsMarker {
			addDiagnostic(&result, diagnostic{Category: "FLAGS_MAPPING_MISMATCH", XNA: et.XNA, Go: et.Key.String(), Message: fmt.Sprintf("expected xna:flags=%t, found %t", et.Flags, at.FlagsMarker)})
			typeDiagnostics[et.XNA]++
		}
		if et.BaseType != "" && strings.HasPrefix(et.BaseType, "Microsoft.Xna.Framework") && len(at.ExportedEmbeddings) > 0 && et.Kind == "class" {
			addDiagnostic(&result, diagnostic{Category: "BASE_MAPPING_MISMATCH", XNA: et.XNA, Go: et.Key.String(), Message: "CLR base type was projected as exported Go embedding"})
			typeDiagnostics[et.XNA]++
		}
		if et.Kind == "interface" && at.Kind != "interface" {
			addDiagnostic(&result, diagnostic{Category: "INTERFACE_MAPPING_MISMATCH", XNA: et.XNA, Go: et.Key.String(), Message: "XNA interface is not a Go interface"})
			typeDiagnostics[et.XNA]++
		}
		for _, memberKey := range et.Members {
			em := expected.Members[memberKey]
			am, memberOK := actual.Members[memberKey]
			if !memberOK {
				addDiagnostic(&result, diagnostic{Category: "MISSING_MEMBER", XNA: em.XNA, Go: memberKey.String(), Message: "mapped Go member is absent"})
				typeDiagnostics[et.XNA]++
				missingMembers[et.XNA] = append(missingMembers[et.XNA], em.XNA+" -> "+memberKey.String())
				addMissingSpecialization(&result, expected, actual, em)
				continue
			}
			presentMembers++
			before := len(result.Diagnostics)
			compareMember(&result, em, am)
			typeDiagnostics[et.XNA] += len(result.Diagnostics) - before
		}
	}

	for key, at := range actual.Types {
		if _, ok := expected.Types[key]; ok || isAdapterType(key, at) {
			continue
		}
		addDiagnostic(&result, diagnostic{Category: "UNEXPECTED_TYPE", Go: key.String(), Message: "exported type does not map to the selected XNA profile or a declared language adapter"})
	}
	for key, am := range actual.Members {
		if _, ok := expected.Members[key]; ok || isAdapterMember(key) {
			continue
		}
		addDiagnostic(&result, diagnostic{Category: "UNEXPECTED_MEMBER", Go: key.String(), Message: "exported member does not map to the selected XNA profile or a declared language adapter"})
		_ = am
	}

	measureLeaks(&result, actual)
	for _, et := range sortedExpectedTypes(expected) {
		if _, missing := contains(result.MissingTypes, et.XNA); missing {
			continue
		}
		if typeDiagnostics[et.XNA] == 0 {
			result.CompleteTypes = append(result.CompleteTypes, et.XNA)
		} else {
			sort.Strings(missingMembers[et.XNA])
			result.PartialTypes = append(result.PartialTypes, typeStatus{XNA: et.XNA, MissingMembers: missingMembers[et.XNA], Diagnostics: typeDiagnostics[et.XNA]})
		}
	}
	sort.Strings(result.CompleteTypes)
	sort.Strings(result.MissingTypes)
	sort.Slice(result.PartialTypes, func(i, j int) bool { return result.PartialTypes[i].XNA < result.PartialTypes[j].XNA })

	result.Summary["TARGET_TYPES"] = presentTypes
	result.Summary["TARGET_MEMBERS"] = presentMembers
	result.Summary["COMPLETE_TYPES"] = len(result.CompleteTypes)
	result.Summary["PARTIAL_TYPES"] = len(result.PartialTypes)
	result.Summary["MISSING_TYPES"] = len(result.MissingTypes)
	result.Summary["TOTAL_DIAGNOSTICS"] = len(result.Diagnostics)
	return result
}

func typeKindMatches(expected *expectedType, actual *actualType) bool {
	switch expected.Kind {
	case "struct", "class":
		return actual.Kind == "struct"
	case "interface":
		return actual.Kind == "interface"
	case "enum":
		return actual.Kind == "named" && actual.Underlying == "int32"
	default:
		return true
	}
}

func compareMember(result *report, expected *expectedMember, actual *actualMember) {
	if expected.GoKind != actual.Kind {
		category := categoryForMember(expected)
		addDiagnostic(result, diagnostic{Category: category, XNA: expected.XNA, Go: actual.Key.String(), Message: fmt.Sprintf("expected Go %s, found %s", expected.GoKind, actual.Kind)})
	}
	if !equalStrings(expected.Parameters, actual.Parameters) {
		addDiagnostic(result, diagnostic{Category: "PARAMETER_MAPPING_MISMATCH", XNA: expected.XNA, Go: actual.Key.String(), Message: fmt.Sprintf("expected parameters %v, found %v", expected.Parameters, actual.Parameters)})
		addDiagnostic(result, diagnostic{Category: "METHOD_SIGNATURE_MAPPING_MISMATCH", XNA: expected.XNA, Go: actual.Key.String(), Message: "mapped parameter signature differs"})
		if hasRefOut(expected.XNA) {
			addDiagnostic(result, diagnostic{Category: "REF_OUT_MAPPING_MISMATCH", XNA: expected.XNA, Go: actual.Key.String(), Message: "ref/out parameter projection differs"})
		}
	}
	if !equalStrings(expected.Results, actual.Results) {
		addDiagnostic(result, diagnostic{Category: "RETURN_MAPPING_MISMATCH", XNA: expected.XNA, Go: actual.Key.String(), Message: fmt.Sprintf("expected results %v, found %v", expected.Results, actual.Results)})
		addDiagnostic(result, diagnostic{Category: "METHOD_SIGNATURE_MAPPING_MISMATCH", XNA: expected.XNA, Go: actual.Key.String(), Message: "mapped result signature differs"})
	}
	expectedError := expected.ErrorAdded
	actualError := len(actual.Results) > 0 && actual.Results[len(actual.Results)-1] == "error"
	if expectedError != actualError {
		addDiagnostic(result, diagnostic{Category: "ERROR_MAPPING_MISMATCH", XNA: expected.XNA, Go: actual.Key.String(), Message: fmt.Sprintf("expected language-added error=%t, found %t", expectedError, actualError)})
	}
	if expected.EnumValue != nil {
		if actual.Value == nil || normalizeInteger(*actual.Value) != normalizeInteger(*expected.EnumValue) {
			found := "<none>"
			if actual.Value != nil {
				found = *actual.Value
			}
			addDiagnostic(result, diagnostic{Category: "ENUM_VALUE_MISMATCH", XNA: expected.XNA, Go: actual.Key.String(), Message: fmt.Sprintf("expected raw enum value %s, found %s", *expected.EnumValue, found)})
		}
	}
}

func addMissingSpecialization(result *report, expected *expectedSurface, actual *actualSurface, member *expectedMember) {
	if member.OverloadMapped {
		prefix := strings.Split(member.GoName, "By")[0]
		for key := range actual.Members {
			_, isMappedMember := expected.Members[key]
			if !isMappedMember && key.Package == member.PackagePath && key.Receiver == member.Receiver && (key.Name == prefix || strings.HasPrefix(key.Name, prefix+"By")) {
				addDiagnostic(result, diagnostic{Category: "OVERLOAD_MAPPING_MISMATCH", XNA: member.XNA, Go: key.String(), Message: "overload group contains a non-matching mapped name"})
				break
			}
		}
	}
	if strings.Contains(member.XNA, "::op_") {
		owner := expected.typeForXNA(member.Owner)
		operatorPrefix := ""
		if owner != nil {
			operatorPrefix = owner.GoName + "Operator"
		}
		for key := range actual.Members {
			_, isMappedMember := expected.Members[key]
			if !isMappedMember && key.Package == member.PackagePath && key.Receiver == "" && strings.HasPrefix(key.Name, operatorPrefix) {
				addDiagnostic(result, diagnostic{Category: "OPERATOR_MAPPING_MISMATCH", XNA: member.XNA, Go: key.String(), Message: "operator group contains a non-matching mapped name"})
				break
			}
		}
	}
	if member.SourceKind == "event" {
		prefix := strings.TrimSuffix(member.GoName, "Handler")
		for key := range actual.Members {
			_, isMappedMember := expected.Members[key]
			if !isMappedMember && key.Package == member.PackagePath && key.Receiver == member.Receiver && strings.HasPrefix(key.Name, prefix) {
				addDiagnostic(result, diagnostic{Category: "EVENT_MAPPING_MISMATCH", XNA: member.XNA, Go: key.String(), Message: "event group contains a non-matching add/remove projection"})
				break
			}
		}
	}
}

func categoryForMember(member *expectedMember) string {
	switch member.SourceKind {
	case "field":
		return "FIELD_MAPPING_MISMATCH"
	case "property":
		return "PROPERTY_MAPPING_MISMATCH"
	case "event":
		return "EVENT_MAPPING_MISMATCH"
	case "method":
		if strings.Contains(member.XNA, "::op_") {
			return "OPERATOR_MAPPING_MISMATCH"
		}
		return "METHOD_SIGNATURE_MAPPING_MISMATCH"
	default:
		return "LANGUAGE_MAPPING_MISMATCH"
	}
}

func measureLeaks(result *report, actual *actualSurface) {
	for key, t := range actual.Types {
		if t.Kind != "struct" && t.Kind != "interface" {
			inspectLeakText(result, key.String(), t.Underlying)
		}
		if strings.Contains(strings.ToLower(key.Name), "nativehandle") || strings.Contains(strings.ToLower(key.Name), "rawhandle") || strings.HasPrefix(strings.ToLower(key.Name), "cna") {
			addDiagnostic(result, diagnostic{Category: "RAW_HANDLE_LEAK", Go: key.String(), Message: "exported type name exposes native-handle/FFI identity"})
		}
	}
	for key, member := range actual.Members {
		for _, value := range append(append([]string(nil), member.Parameters...), member.Results...) {
			inspectLeakText(result, key.String(), value)
		}
		lower := strings.ToLower(key.Name)
		if strings.Contains(lower, "nativehandle") || strings.Contains(lower, "rawhandle") || strings.HasPrefix(lower, "cna") {
			addDiagnostic(result, diagnostic{Category: "RAW_HANDLE_LEAK", Go: key.String(), Message: "exported member name exposes native-handle/FFI identity"})
		}
	}
}

func inspectLeakText(result *report, goIdentity, text string) {
	if strings.Contains(text, "internal/") || strings.Contains(text, "interop.") {
		addDiagnostic(result, diagnostic{Category: "INTERNAL_TYPE_LEAK", Go: goIdentity, Message: "exported signature references an internal type"})
	}
	if strings.Contains(text, "unsafe.Pointer") || strings.Contains(text, "C.") {
		addDiagnostic(result, diagnostic{Category: "PUBLIC_NATIVE_FFI_LEAK", Go: goIdentity, Message: "exported signature references unsafe/C FFI state"})
	}
}

func isAdapterType(key symbolKey, actual *actualType) bool {
	return key.Package == modulePath+"/Microsoft/Xna/Framework" && adapterTypes[key.Name] && actual != nil
}

func isAdapterMember(key symbolKey) bool {
	if key.Package != modulePath+"/Microsoft/Xna/Framework" {
		return false
	}
	if adapterTypes[key.Receiver] {
		return true
	}
	return key.Receiver == "" && adapterFunctions[key.Name]
}

func addDiagnostic(result *report, item diagnostic) {
	result.Diagnostics = append(result.Diagnostics, item)
	result.Summary[item.Category]++
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func hasRefOut(identity string) bool {
	return strings.Contains(identity, "ref ") || strings.Contains(identity, "out ") || strings.Contains(identity, "in ")
}

func normalizeInteger(value string) string {
	return strings.TrimSpace(strings.Trim(value, "\""))
}

func contains(values []string, wanted string) (int, bool) {
	for i, value := range values {
		if value == wanted {
			return i, true
		}
	}
	return -1, false
}
