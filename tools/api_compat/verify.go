package main

import (
	"fmt"
	"go/types"
	"sort"
	"strings"
)

const packedVectorNamespace = "Microsoft.Xna.Framework.Graphics.PackedVector."

var vertexElementClosureTypes = []string{
	"Microsoft.Xna.Framework.Graphics.VertexElement",
	"Microsoft.Xna.Framework.Graphics.VertexElementFormat",
	"Microsoft.Xna.Framework.Graphics.VertexElementUsage",
}

var playerIndexKeyboardClosureTypes = []string{
	"Microsoft.Xna.Framework.PlayerIndex",
	"Microsoft.Xna.Framework.Input.Keyboard",
}

var adapterTypes = map[string]bool{
	"EventSubscription": true,
	"GameCallbacks":     true,
	"Iterator":          true,
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
	typeDiagnostics := make(map[string]int)
	missingMembers := make(map[string][]string)
	result.Summary["REFERENCE_TYPES"] = expected.ReferenceTypes
	result.Summary["REFERENCE_MEMBERS"] = expected.ReferenceMembers
	result.Summary["EXPECTED_GO_TYPES"] = expected.ExpectedGoTypes
	result.Summary["EXPECTED_GO_MEMBERS"] = expected.ExpectedGoMembers
	result.Summary["INTERFACE_WITNESS_PROJECTIONS"] = len(expected.InterfaceWitnesses)
	result.Summary["ALLOWLIST_ENTRIES"] = allowlistEntries
	if allowlistEntries > 0 {
		addDiagnostic(&result, diagnostic{Category: "ALLOWLIST_ENTRIES", Message: fmt.Sprintf("mapping allowlist has %d entries", allowlistEntries)})
	}
	for _, source := range actual.Unmeasured {
		addDiagnostic(&result, diagnostic{Category: "UNMEASURED_STRUCTURAL_CATEGORY", Go: source, Message: "source requested an unmeasured structural category"})
	}
	for _, issue := range expected.MappingIssues {
		addDiagnostic(&result, issue)
		if issue.XNA != "" {
			typeDiagnostics[issue.XNA]++
		}
	}

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
		before := len(result.Diagnostics)
		measureCollectionInterfaceProjection(&result, expected, actual, et)
		measureDirectInterfaceInheritance(&result, actual, et, at)
		measureInterfaceWitnesses(&result, expected, actual, et)
		if measurement, measured := measurePackedInterfaceConformance(&result, actual, et); measured {
			result.PackedInterfaceConformance = append(result.PackedInterfaceConformance, measurement)
		}
		typeDiagnostics[et.XNA] += len(result.Diagnostics) - before
	}

	for key, at := range actual.Types {
		if _, ok := expected.Types[key]; ok || isAdapterType(key, at) {
			continue
		}
		addDiagnostic(&result, diagnostic{Category: "UNEXPECTED_TYPE", Go: key.String(), Message: "exported type does not map to the selected XNA profile or a declared language adapter"})
	}
	for key, am := range actual.Members {
		if _, ok := expected.Members[key]; ok || expected.InterfaceWitnesses[key] != nil || isAdapterMember(key) {
			continue
		}
		addDiagnostic(&result, diagnostic{Category: "UNEXPECTED_MEMBER", Go: key.String(), Message: "exported member does not map to the selected XNA profile or a declared language adapter"})
		if owner := expectedTypeForActualMember(expected, key); owner != nil {
			typeDiagnostics[owner.XNA]++
		}
		_ = am
	}

	measureLeaks(&result, actual)
	for _, et := range sortedExpectedTypes(expected) {
		if !strings.HasPrefix(et.XNA, packedVectorNamespace) {
			continue
		}
		measurement := packedVectorTypeMeasurement{
			XNA:               et.XNA,
			GoName:            et.GoName,
			SourceMembers:     et.SourceMembers,
			ExpectedGoMembers: len(et.Members),
			LocalDiagnostics:  typeDiagnostics[et.XNA],
		}
		if at := actual.Types[et.Key]; at != nil {
			measurement.TypeKind = at.Kind
			for _, key := range et.Members {
				if actual.Members[key] != nil {
					measurement.TargetGoMembers++
				}
			}
		} else {
			measurement.TypeKind = "missing"
		}
		if mapped, ok := directPackedInterface(et); ok {
			measurement.TPacked = firstOrEmpty(mapped.TypeArguments)
			measurement.DirectInterfaceStatus = "FAIL"
			for _, conformance := range result.PackedInterfaceConformance {
				if conformance.Owner == et.XNA {
					measurement.DirectInterfaceStatus = conformance.Status
					break
				}
			}
		}
		result.PackedVectorTypeMeasurements = append(result.PackedVectorTypeMeasurements, measurement)
	}
	result.VertexElementClosure = measureVertexElementClosure(expected, actual, typeDiagnostics)
	result.PlayerIndexKeyboardClosure = measurePlayerIndexKeyboardClosure(expected, actual, typeDiagnostics)
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
	result.Summary["INTERFACE_WITNESS_PROJECTIONS"] = len(result.InterfaceWitnessProjections)
	for _, witness := range result.InterfaceWitnessProjections {
		switch witness.Member {
		case "PackFromVector4":
			result.Summary["PACKFROMVECTOR4_WITNESS_PROJECTIONS"]++
		case "ToVector4":
			result.Summary["TOVECTOR4_WITNESS_PROJECTIONS"]++
		}
	}
	result.Summary["TOTAL_DIAGNOSTICS"] = len(result.Diagnostics)
	return result
}

func measurePlayerIndexKeyboardClosure(expected *expectedSurface, actual *actualSurface, typeDiagnostics map[string]int) playerIndexKeyboardClosure {
	measurement := playerIndexKeyboardClosure{
		SourceTypes: len(playerIndexKeyboardClosureTypes),
		Status:      "FAIL",
	}
	for _, identity := range playerIndexKeyboardClosureTypes {
		owner := expected.typeForXNA(identity)
		if owner == nil {
			continue
		}
		row := playerIndexTypeMeasurement{
			XNA:               owner.XNA,
			GoName:            owner.GoName,
			SourceMembers:     owner.SourceMembers,
			ExpectedGoMembers: len(owner.Members),
			LocalDiagnostics:  typeDiagnostics[owner.XNA],
			ExpectedKind:      owner.Kind,
			ActualKind:        "missing",
		}
		measurement.SourceIdentities += row.SourceMembers
		measurement.MappedGoIdentities += row.ExpectedGoMembers
		measurement.LocalDiagnostics += row.LocalDiagnostics
		if target := actual.Types[owner.Key]; target != nil {
			measurement.TargetTypes++
			row.ActualKind = target.Kind
			row.ActualUnderlying = target.Underlying
			for _, memberKey := range owner.Members {
				if actual.Members[memberKey] != nil {
					row.TargetGoMembers++
				}
			}
			measurement.TargetGoIdentities += row.TargetGoMembers
		}
		measurement.TypeMeasurements = append(measurement.TypeMeasurements, row)
	}
	if measurement.SourceTypes == 2 && measurement.SourceIdentities == 7 && measurement.MappedGoIdentities == 6 &&
		measurement.TargetTypes == 2 && measurement.TargetGoIdentities == 6 && measurement.LocalDiagnostics == 0 {
		measurement.Status = "PASS"
	}
	return measurement
}

func expectedTypeForActualMember(expected *expectedSurface, key symbolKey) *expectedType {
	if key.Receiver == "" {
		return nil
	}
	return expected.Types[symbolKey{Package: key.Package, Name: key.Receiver}]
}

func measureVertexElementClosure(expected *expectedSurface, actual *actualSurface, typeDiagnostics map[string]int) vertexElementClosure {
	measurement := vertexElementClosure{
		SourceTypes:        len(vertexElementClosureTypes),
		WritableProperties: 4,
		ProjectedAccessors: 8,
		Status:             "FAIL",
	}
	for _, identity := range vertexElementClosureTypes {
		owner := expected.typeForXNA(identity)
		if owner == nil {
			continue
		}
		row := vertexElementTypeMeasurement{
			XNA:               owner.XNA,
			GoName:            owner.GoName,
			SourceMembers:     owner.SourceMembers,
			ExpectedGoMembers: len(owner.Members),
			LocalDiagnostics:  typeDiagnostics[owner.XNA],
			ExpectedKind:      owner.Kind,
			ActualKind:        "missing",
		}
		measurement.SourceIdentities += row.SourceMembers
		measurement.MappedGoIdentities += row.ExpectedGoMembers
		measurement.LocalDiagnostics += row.LocalDiagnostics
		if target := actual.Types[owner.Key]; target != nil {
			measurement.TargetTypes++
			row.ActualKind = target.Kind
			row.ActualUnderlying = target.Underlying
			for _, memberKey := range owner.Members {
				if actual.Members[memberKey] != nil {
					row.TargetGoMembers++
				}
			}
			measurement.TargetGoIdentities += row.TargetGoMembers
		}
		measurement.TypeMeasurements = append(measurement.TypeMeasurements, row)
	}
	if measurement.SourceTypes == 3 && measurement.SourceIdentities == 37 && measurement.MappedGoIdentities == 39 &&
		measurement.TargetTypes == 3 && measurement.TargetGoIdentities == 39 && measurement.LocalDiagnostics == 0 {
		measurement.Status = "PASS"
	}
	return measurement
}

func measureDirectInterfaceInheritance(result *report, actual *actualSurface, owner *expectedType, target *actualType) {
	if owner.Kind != "interface" || len(owner.MappedInterfaces) == 0 {
		return
	}
	wanted := make([]string, len(owner.MappedInterfaces))
	for i, mapped := range owner.MappedInterfaces {
		wanted[i] = mappedInterfaceDisplay(mapped)
	}
	if !equalUnorderedStrings(wanted, target.ExportedEmbeddings) {
		addDiagnostic(result, diagnostic{
			Category: "INTERFACE_MAPPING_MISMATCH",
			XNA:      owner.XNA,
			Go:       owner.Key.String(),
			Message:  fmt.Sprintf("expected direct interface embeddings %v, found %v", wanted, target.ExportedEmbeddings),
		})
	}
}

func measureInterfaceWitnesses(result *report, expected *expectedSurface, actual *actualSurface, owner *expectedType) {
	var witnesses []*expectedInterfaceWitness
	for _, witness := range expected.InterfaceWitnesses {
		if witness.Owner == owner.XNA {
			witnesses = append(witnesses, witness)
		}
	}
	sort.Slice(witnesses, func(i, j int) bool { return witnesses[i].Key.Name < witnesses[j].Key.Name })
	for _, witness := range witnesses {
		row := interfaceWitnessProjection{
			Owner:           witness.Owner,
			Member:          witness.GoName,
			SourceInterface: witness.SourceInterface,
			InterfaceMember: witness.InterfaceMember,
			Reason:          witness.Reason,
			Signature:       witnessSignature(witness),
			Status:          "PASS",
		}
		actualMember := actual.Members[witness.Key]
		if actualMember == nil {
			row.Status = "MISSING"
			addDiagnostic(result, diagnostic{
				Category: "INTERFACE_MAPPING_MISMATCH",
				XNA:      witness.Owner,
				Go:       witness.Key.String(),
				Message:  "required explicit-interface witness method is absent",
			})
			result.InterfaceWitnessProjections = append(result.InterfaceWitnessProjections, row)
			continue
		}
		if actualMember.Kind != "method" || !equalStrings(witness.Parameters, actualMember.Parameters) || !equalStrings(witness.Results, actualMember.Results) {
			row.Status = "SIGNATURE_MISMATCH"
			addDiagnostic(result, diagnostic{
				Category: "INTERFACE_MAPPING_MISMATCH",
				XNA:      witness.Owner,
				Go:       witness.Key.String(),
				Message:  fmt.Sprintf("witness expected parameters/results %v/%v, found %v/%v", witness.Parameters, witness.Results, actualMember.Parameters, actualMember.Results),
			})
			expectedMember := &expectedMember{
				Key: witness.Key, XNA: witness.InterfaceMember, Owner: witness.Owner,
				SourceKind: "method", GoKind: "method", GoName: witness.GoName,
				PackagePath: witness.Key.Package, Receiver: witness.Key.Receiver,
				Parameters: witness.Parameters, Results: witness.Results,
			}
			compareMember(result, expectedMember, actualMember)
		}
		result.InterfaceWitnessProjections = append(result.InterfaceWitnessProjections, row)
	}
}

func measurePackedInterfaceConformance(result *report, actual *actualSurface, owner *expectedType) (packedInterfaceConformance, bool) {
	mapped, ok := directPackedInterface(owner)
	if !ok {
		return packedInterfaceConformance{}, false
	}
	measurement := packedInterfaceConformance{
		Owner:     owner.XNA,
		Interface: mappedInterfaceDisplay(mapped),
		TPacked:   firstOrEmpty(mapped.TypeArguments),
		Status:    "FAIL",
	}
	pkg := actual.Packages[owner.PackagePath]
	if pkg == nil {
		addPackedConformanceDiagnostic(result, owner, "compiler package evidence is absent")
		return measurement, true
	}
	ownerObject := pkg.Scope().Lookup(owner.GoName)
	interfaceObject := pkg.Scope().Lookup("IPackedVectorOfTPacked")
	baseObject := pkg.Scope().Lookup("IPackedVector")
	if ownerObject == nil || interfaceObject == nil || baseObject == nil {
		addPackedConformanceDiagnostic(result, owner, "packed owner or mapped interface identity is absent from compiler scope")
		return measurement, true
	}
	ownerNamed, ownerOK := ownerObject.Type().(*types.Named)
	interfaceNamed, interfaceOK := interfaceObject.Type().(*types.Named)
	baseNamed, baseOK := baseObject.Type().(*types.Named)
	typeArgument, argumentOK := mappedBasicType(measurement.TPacked)
	if !ownerOK || !interfaceOK || !baseOK || !argumentOK {
		addPackedConformanceDiagnostic(result, owner, "packed owner/interface/type argument could not be represented by go/types")
		return measurement, true
	}
	instantiated, err := types.Instantiate(nil, interfaceNamed, []types.Type{typeArgument}, true)
	if err != nil {
		addPackedConformanceDiagnostic(result, owner, "generic packed interface instantiation failed: "+err.Error())
		return measurement, true
	}
	packedInterface, ok := instantiated.Underlying().(*types.Interface)
	if !ok {
		addPackedConformanceDiagnostic(result, owner, "mapped generic packed identity is not a Go interface")
		return measurement, true
	}
	baseInterface, ok := baseNamed.Underlying().(*types.Interface)
	if !ok {
		addPackedConformanceDiagnostic(result, owner, "mapped packed base identity is not a Go interface")
		return measurement, true
	}
	packedInterface.Complete()
	baseInterface.Complete()
	pointer := types.NewPointer(ownerNamed)
	measurement.PointerMethodSetSatisfies = types.Implements(pointer, packedInterface)
	measurement.ValueMethodSetSatisfies = types.Implements(ownerNamed, packedInterface)
	measurement.TransitiveBaseSatisfies = types.Implements(pointer, baseInterface)
	if measurement.PointerMethodSetSatisfies && !measurement.ValueMethodSetSatisfies && measurement.TransitiveBaseSatisfies {
		measurement.Status = "PASS"
		return measurement, true
	}
	addPackedConformanceDiagnostic(result, owner, fmt.Sprintf(
		"expected *%s to satisfy %s and IPackedVector while value %s does not; pointer=%t value=%t transitive=%t",
		owner.GoName, measurement.Interface, owner.GoName,
		measurement.PointerMethodSetSatisfies, measurement.ValueMethodSetSatisfies, measurement.TransitiveBaseSatisfies,
	))
	return measurement, true
}

func addPackedConformanceDiagnostic(result *report, owner *expectedType, message string) {
	addDiagnostic(result, diagnostic{
		Category: "INTERFACE_MAPPING_MISMATCH",
		XNA:      owner.XNA,
		Go:       owner.Key.String(),
		Message:  message,
	})
}

func directPackedInterface(owner *expectedType) (mappedInterface, bool) {
	if !strings.HasPrefix(owner.XNA, packedVectorNamespace) {
		return mappedInterface{}, false
	}
	for _, mapped := range owner.MappedInterfaces {
		identity, _ := splitConstructedType(mapped.XNA)
		if identity == "Microsoft.Xna.Framework.Graphics.PackedVector.IPackedVector`1" {
			return mapped, true
		}
	}
	return mappedInterface{}, false
}

func mappedBasicType(name string) (types.Type, bool) {
	switch name {
	case "uint8":
		return types.Typ[types.Uint8], true
	case "uint16":
		return types.Typ[types.Uint16], true
	case "uint32":
		return types.Typ[types.Uint32], true
	case "uint64":
		return types.Typ[types.Uint64], true
	default:
		return nil, false
	}
}

func mappedInterfaceDisplay(mapped mappedInterface) string {
	if len(mapped.TypeArguments) == 0 {
		return mapped.GoName
	}
	return mapped.GoName + "[" + strings.Join(mapped.TypeArguments, ",") + "]"
}

func witnessSignature(witness *expectedInterfaceWitness) string {
	results := strings.Join(witness.Results, ",")
	if len(witness.Results) > 1 {
		results = "(" + results + ")"
	}
	if results != "" {
		results = " " + results
	}
	return witness.GoName + "(" + strings.Join(witness.Parameters, ",") + ")" + results
}

func equalUnorderedStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := append([]string(nil), left...)
	rightCopy := append([]string(nil), right...)
	sort.Strings(leftCopy)
	sort.Strings(rightCopy)
	return equalStrings(leftCopy, rightCopy)
}

func firstOrEmpty(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func measureCollectionInterfaceProjection(result *report, expected *expectedSurface, actual *actualSurface, owner *expectedType) {
	if !containsInterfacePrefix(owner.AllInterfaces, "System.Collections.Generic.ICollection`1[") {
		return
	}

	required := []string{"Add", "Clear", "Contains", "CopyTo", "Remove", "Count", "IsReadOnly", "GetEnumerator"}
	for _, name := range required {
		found := false
		for _, key := range owner.Members {
			member := expected.Members[key]
			if member.GoName == name {
				_, found = actual.Members[key]
				break
			}
		}
		if !found {
			addDiagnostic(result, diagnostic{
				Category: "INTERFACE_MAPPING_MISMATCH",
				XNA:      owner.XNA,
				Go:       owner.Key.String(),
				Message:  "ICollection<T> projection is missing " + name,
			})
			return
		}
	}

	iteratorKey := symbolKey{Package: modulePath + "/Microsoft/Xna/Framework", Name: "Iterator"}
	iterator, ok := actual.Types[iteratorKey]
	if !ok || iterator.Kind != "interface" || !equalStrings(iterator.TypeParameters, []string{"T"}) {
		addDiagnostic(result, diagnostic{
			Category: "INTERFACE_MAPPING_MISMATCH",
			XNA:      owner.XNA,
			Go:       iteratorKey.String(),
			Message:  "IEnumerator<T> must use the measured generic Iterator<T> adapter",
		})
		return
	}
	nextKey := symbolKey{Package: iteratorKey.Package, Receiver: "Iterator", Name: "Next"}
	next, ok := actual.Members[nextKey]
	if !ok || len(next.Parameters) != 0 || !equalStrings(next.Results, []string{"T", "bool", "error"}) {
		addDiagnostic(result, diagnostic{
			Category: "INTERFACE_MAPPING_MISMATCH",
			XNA:      owner.XNA,
			Go:       nextKey.String(),
			Message:  "Iterator<T>.Next must return (T, bool, error)",
		})
	}
}

func containsInterfacePrefix(interfaces []string, prefix string) bool {
	for _, value := range interfaces {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
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
