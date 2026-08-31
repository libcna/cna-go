package main

import (
	"fmt"
	"go/types"
	"regexp"
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

const (
	displayOrientationIdentity = "Microsoft.Xna.Framework.DisplayOrientation"
	graphicsManagerIdentity    = "Microsoft.Xna.Framework.GraphicsDeviceManager"
	supportedOrientationsName  = "SupportedOrientations"
	bufferUsageIdentity        = "Microsoft.Xna.Framework.Graphics.BufferUsage"
	clearOptionsIdentity       = "Microsoft.Xna.Framework.Graphics.ClearOptions"
	surfaceFormatIdentity      = "Microsoft.Xna.Framework.Graphics.SurfaceFormat"
	depthFormatIdentity        = "Microsoft.Xna.Framework.Graphics.DepthFormat"
	graphicsProfileIdentity    = "Microsoft.Xna.Framework.Graphics.GraphicsProfile"
	buttonStateIdentity        = "Microsoft.Xna.Framework.Input.ButtonState"
)

var buttonStateValues = []struct {
	Name  string
	Value string
}{
	{"Released", "0"},
	{"Pressed", "1"},
}

var graphicsProfileValues = []struct {
	Name  string
	Value string
}{
	{"Reach", "0"},
	{"HiDef", "1"},
}

var depthFormatValues = []struct {
	Name  string
	Value string
}{
	{"None", "0"},
	{"Depth16", "1"},
	{"Depth24", "2"},
	{"Depth24Stencil8", "3"},
}

var surfaceFormatValues = []struct {
	Name  string
	Value string
}{
	{"Color", "0"},
	{"Bgr565", "1"},
	{"Bgra5551", "2"},
	{"Bgra4444", "3"},
	{"Dxt1", "4"},
	{"Dxt3", "5"},
	{"Dxt5", "6"},
	{"NormalizedByte2", "7"},
	{"NormalizedByte4", "8"},
	{"Rgba1010102", "9"},
	{"Rg32", "10"},
	{"Rgba64", "11"},
	{"Alpha8", "12"},
	{"Single", "13"},
	{"Vector2", "14"},
	{"Vector4", "15"},
	{"HalfSingle", "16"},
	{"HalfVector2", "17"},
	{"HalfVector4", "18"},
	{"HdrBlendable", "19"},
}

// adapterTypes are the CNA-Go language-support types. They are declared in
// mapping-rules.json under languageAdapters, live in the framework package, and
// are measured as adapters rather than counted as XNA type identities.
var adapterTypes = map[string]bool{
	"EventArgs":         true,
	"EventHandler":      true,
	"EventSource":       true,
	"EventSubscription": true,
	"GameCallbacks":     true,
	"Iterator":          true,
	"TimeSpan":          true,
	// ReadOnlyCollection is a MEASURED adapter: bclSignatureAdapters pins its
	// exact public member set, so admitting it here does not admit whatever
	// members it happens to declare.
	"ReadOnlyCollection": true,
}

var adapterFunctions = map[string]bool{
	"EventArgsEmpty":    true,
	"NewGame":           true,
	"TimeSpanFromTicks": true,
}

func init() {
	for name := range bclSignatureAdapterConstructors {
		adapterFunctions[name] = true
	}
	// The Game base-call helpers are admitted FROM the measured registry
	// rather than by hand, so a helper can only be admitted by being declared
	// -- and being declared is what subjects it to measureGameBaseCallAdapters.
	// There is no allowlist here and no way to add one.
	for _, adapter := range gameBaseCallAdapters {
		adapterFunctions[adapter.GoFunction] = true
	}
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
	// Both base-call counters are seeded so a run that measures nothing still
	// reports 0 rather than omitting the key, which is what keeps a silently
	// unmeasured family from reading like a clean one.
	result.Summary["GAME_BASE_CALL_ADAPTERS"] = 0
	result.Summary["GAME_BASE_CALL_DEFERRED_STEPS"] = 0
	result.Summary["DECLARED_INTERFACE_CONFORMANCE"] = 0
	result.Summary["XNA_BASE_RELATIONSHIPS"] = 0
	result.Summary["XNA_BASE_DERIVED_TYPES"] = 0
	result.Summary["XNA_DEFERRED_BASE_BLOCKERS"] = 0
	result.Summary["XNA_INHERITED_PUBLIC_MEMBERS_UNPROJECTED"] = 0
	// Foundation 36's counters are seeded for the same reason: a native signal
	// family that measured nothing must report zeros rather than omit the keys.
	result.Summary["GAME_NATIVE_SIGNALS"] = 0
	result.Summary["GAME_NATIVE_SIGNAL_RAISE_SITES"] = 0
	result.Summary["GAME_NATIVE_SIGNALS_RUNTIME_DEFERRED"] = 0
	result.Summary["GAME_NATIVE_SIGNALS_LIFECYCLE_ONLY"] = 0
	result.Summary["GAME_MANAGED_EVENT_RAISE_SITES"] = 0
	result.Summary["XNA_BASE_TYPED_SIGNATURE_POSITIONS"] = 0
	result.Summary["XNA_BASE_SUBSTITUTABILITY_NONE"] = 0
	result.Summary["XNA_BASE_SUBSTITUTABILITY_LATENT"] = 0
	result.Summary["XNA_BASE_SUBSTITUTABILITY_LIVE"] = 0
	result.Summary["XNA_COMPOSED_BASE_RELATIONSHIPS"] = 0
	result.Summary["XNA_COMPOSED_DERIVED_TYPES"] = 0
	result.Summary["XNA_COMPOSED_DERIVED_TYPES_PROJECTED"] = 0
	result.Summary["XNA_INHERITED_ATTRIBUTED_MEMBERS"] = 0
	result.Summary["GAME_FRAME_HOOKS"] = 0
	result.Summary["GAME_FRAME_HOOKS_NEVER_INSTALLED"] = 0
	result.Summary["GAME_FRAME_HOOKS_INSTALLED_ON_OVERRIDE"] = 0
	result.Summary["GAME_FRAME_HOOK_OVERRIDE_CAPABILITIES"] = 0
	result.Summary["GAME_CALLBACKS_MEMBERS"] = 0
	result.Summary["GAME_FRAME_HOOK_DEFERRED_STEPS"] = 0
	typeDiagnostics := make(map[string]int)
	missingMembers := make(map[string][]string)
	result.Summary["REFERENCE_TYPES"] = expected.ReferenceTypes
	// REFERENCE_MEMBERS keeps naming exactly what the Microsoft XNA
	// assemblies declare. It is deliberately NOT inflated with inherited
	// mscorlib members: pretending a BCL member was declared in the XNA
	// metadata would falsify the pinned contract. The two names below are
	// aliases that make the three provenance classes explicit in the report.
	result.Summary["REFERENCE_MEMBERS"] = expected.ReferenceMembers
	result.Summary["REFERENCE_XNA_MEMBERS"] = expected.ReferenceMembers
	result.Summary["BCL_INHERITED_PUBLIC_MEMBERS"] = expected.BCLInheritedCLRMembers
	result.Summary["BCL_INHERITED_MEMBER_PROJECTIONS"] = expected.BCLInheritedProjections
	result.Summary["XNA_INHERITED_PUBLIC_MEMBERS"] = expected.XNAInheritedCLRMembers
	result.Summary["XNA_INHERITED_MEMBER_PROJECTIONS"] = expected.XNAInheritedProjections
	result.Summary["EXPECTED_GO_TYPES"] = expected.ExpectedGoTypes
	result.Summary["EXPECTED_GO_MEMBERS"] = expected.ExpectedGoMembers
	result.Summary["INTERFACE_WITNESS_PROJECTIONS"] = len(expected.InterfaceWitnesses)
	result.BCLBaseAdapters = measureBCLBaseAdapters(expected, actual)
	result.BCLSignatureAdapters = measureBCLSignatureAdapters(&result, expected, actual)
	result.Summary["BCL_SIGNATURE_ADAPTERS"] = len(result.BCLSignatureAdapters)
	for _, adapter := range result.BCLSignatureAdapters {
		result.Summary["BCL_SIGNATURE_ADAPTER_CARRIERS"] += adapter.SignaturePositions
	}
	result.GameBaseCallAdapters = measureGameBaseCallAdapters(&result, expected, actual)
	result.Summary["GAME_BASE_CALL_ADAPTERS"] = len(result.GameBaseCallAdapters)
	result.GameNativeSignals = measureGameNativeSignals(&result, expected, actual)
	result.Summary["GAME_NATIVE_SIGNALS"] = len(result.GameNativeSignals)
	result.GameFrameHooks = measureGameFrameHooks(&result, expected, actual)
	result.XNABaseSubstitutability = measureXNABaseSubstitutability(&result, expected, actual)
	result.XNAComposition = measureXNAComposition(&result, expected, actual)
	result.Summary["GAME_FRAME_HOOKS"] = len(result.GameFrameHooks)
	result.Summary["BCL_BASE_ADAPTERS"] = len(result.BCLBaseAdapters)
	for _, adapter := range result.BCLBaseAdapters {
		result.Summary["BCL_BASE_ADAPTER_CONSUMERS"] += len(adapter.Consumers)
		if adapter.Verdict != "PASS" {
			addDiagnostic(&result, diagnostic{
				Category: "BASE_MAPPING_MISMATCH", XNA: adapter.CLRBase,
				Message: "BCL base adapter measurement failed",
			})
		}
	}
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
		verifyBCLBaseProjection(&result, typeDiagnostics, et, at)
		verifyBCLInterfaceRelationships(&result, expected, actual, typeDiagnostics, et)
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
		verifyEventGroupLeaks(&result, expected, actual, et)
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

	measureLeaks(&result, expected, actual)
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
	result.DisplayOrientationClosure = measureDisplayOrientationClosure(expected, actual, typeDiagnostics)
	result.BufferUsageClosure = measureBufferUsageClosure(expected, actual, typeDiagnostics)
	result.ClearOptionsClosure = measureClearOptionsClosure(expected, actual, typeDiagnostics)
	result.SurfaceFormatClosure = measureSurfaceFormatClosure(expected, actual, typeDiagnostics)
	result.DepthFormatClosure = measureDepthFormatClosure(expected, actual, typeDiagnostics)
	result.GraphicsProfileClosure = measureGraphicsProfileClosure(expected, actual, typeDiagnostics)
	result.ButtonStateClosure = measureButtonStateClosure(expected, actual, typeDiagnostics)
	result.Foundation14EnumClosures = measureFoundation14EnumClosures(expected, actual, typeDiagnostics)
	result.Foundation15EnumClosures = measureBatchEnumClosures(expected, actual, typeDiagnostics, foundation15Enums)
	result.Foundation15ValueStructs = measureValueStructClosures(expected, actual, typeDiagnostics, foundation15ValueStructs)
	result.Foundation16ValueStructs = measureValueStructClosures(expected, actual, typeDiagnostics, foundation16ValueStructs)
	result.Foundation17ManagedClasses = measureManagedClassClosures(expected, actual, typeDiagnostics, foundation17ManagedClasses)
	result.Foundation18Interfaces = measureManagedInterfaceClosures(expected, actual, typeDiagnostics, foundation18Interfaces)
	result.Foundation19ManagedClasses = measureManagedClassClosures(expected, actual, typeDiagnostics, foundation19ManagedClasses)
	result.Foundation20ValueContracts = measureManagedClassClosures(expected, actual, typeDiagnostics, foundation20ValueContracts)
	result.Foundation21ManagedClasses = measureManagedClassClosures(expected, actual, typeDiagnostics, foundation21ManagedClasses)
	result.Foundation23Interfaces = measureManagedInterfaceClosures(expected, actual, typeDiagnostics, foundation23Interfaces)
	result.Foundation23ManagedClasses = measureManagedClassClosures(expected, actual, typeDiagnostics, foundation23ManagedClasses)
	result.BCLBaseRelationships = measureBCLBaseRelationships(expected, actual)
	for _, relationship := range result.BCLBaseRelationships {
		result.Summary["BCL_DEFERRED_BASE_BLOCKERS"] += len(relationship.Blockers)
		if relationship.Verdict != "PASS" {
			addDiagnostic(&result, diagnostic{
				Category: "BASE_MAPPING_MISMATCH", XNA: relationship.CLRBase,
				Message: "BCL base relationship measurement failed: a deferred base must name what blocks it",
			})
		}
	}
	result.BCLInterfaceRelationships = measureBCLInterfaceRelationships(expected, actual)
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

	result.XNABaseRelationships = measureXNABaseRelationships(&result, expected, result.CompleteTypes)
	result.Summary["XNA_BASE_RELATIONSHIPS"] = len(result.XNABaseRelationships)
	result.DeclaredInterfaceConformance = measureDeclaredInterfaceConformance(&result, expected, actual, result.CompleteTypes)
	result.Summary["DECLARED_INTERFACE_CONFORMANCE"] = len(result.DeclaredInterfaceConformance)

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

// enumLiteral is one pinned XNA enum literal: its exact source name and its
// exact source raw value.
type enumLiteral struct {
	Name  string
	Value string
}

// foundation14Enum pins one Foundation-14 ordinary or flags enum against the
// XNA 4.0 Windows contract. The table is authoritative for the verifier: a
// literal that is renamed, revalued, dropped, or invented in Go source is
// rejected because it no longer matches this table and the pinned contract at
// the same time.
type foundation14Enum struct {
	Identity string
	Flags    bool
	Values   []enumLiteral
}

// foundation14Enums is the complete Foundation-14 pure-managed batch: 25 enums
// carrying 121 mapped Go identities. Every entry is an ordinary or flags enum
// whose only public-signature dependency is System.Int32, so none of them adds
// a runtime, native, or capability route.
var foundation14Enums = []foundation14Enum{
	{
		Identity: "Microsoft.Xna.Framework.Graphics.RenderTargetUsage",
		Flags:    false,
		Values: []enumLiteral{
			{"DiscardContents", "0"},
			{"PreserveContents", "1"},
			{"PlatformContents", "2"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Graphics.CubeMapFace",
		Flags:    false,
		Values: []enumLiteral{
			{"PositiveX", "0"},
			{"NegativeX", "1"},
			{"PositiveY", "2"},
			{"NegativeY", "3"},
			{"PositiveZ", "4"},
			{"NegativeZ", "5"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Audio.AudioChannels",
		Flags:    false,
		Values: []enumLiteral{
			{"Mono", "1"},
			{"Stereo", "2"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Audio.AudioStopOptions",
		Flags:    false,
		Values: []enumLiteral{
			{"AsAuthored", "0"},
			{"Immediate", "1"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Graphics.IndexElementSize",
		Flags:    false,
		Values: []enumLiteral{
			{"SixteenBits", "0"},
			{"ThirtyTwoBits", "1"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Graphics.SetDataOptions",
		Flags:    true,
		Values: []enumLiteral{
			{"None", "0"},
			{"Discard", "1"},
			{"NoOverwrite", "2"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Media.MediaState",
		Flags:    false,
		Values: []enumLiteral{
			{"Stopped", "0"},
			{"Playing", "1"},
			{"Paused", "2"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Graphics.EffectParameterClass",
		Flags:    false,
		Values: []enumLiteral{
			{"Scalar", "0"},
			{"Vector", "1"},
			{"Matrix", "2"},
			{"Object", "3"},
			{"Struct", "4"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Graphics.CompareFunction",
		Flags:    false,
		Values: []enumLiteral{
			{"Always", "0"},
			{"Never", "1"},
			{"Less", "2"},
			{"LessEqual", "3"},
			{"Equal", "4"},
			{"GreaterEqual", "5"},
			{"Greater", "6"},
			{"NotEqual", "7"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Graphics.EffectParameterType",
		Flags:    false,
		Values: []enumLiteral{
			{"Void", "0"},
			{"Bool", "1"},
			{"Int32", "2"},
			{"Single", "3"},
			{"String", "4"},
			{"Texture", "5"},
			{"Texture1D", "6"},
			{"Texture2D", "7"},
			{"Texture3D", "8"},
			{"TextureCube", "9"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Input.Touch.GestureType",
		Flags:    true,
		Values: []enumLiteral{
			{"None", "0"},
			{"Tap", "1"},
			{"DoubleTap", "2"},
			{"Hold", "4"},
			{"HorizontalDrag", "8"},
			{"VerticalDrag", "16"},
			{"FreeDrag", "32"},
			{"Pinch", "64"},
			{"Flick", "128"},
			{"DragComplete", "256"},
			{"PinchComplete", "512"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Input.Buttons",
		Flags:    true,
		Values: []enumLiteral{
			{"DPadUp", "1"},
			{"DPadDown", "2"},
			{"DPadLeft", "4"},
			{"DPadRight", "8"},
			{"Start", "16"},
			{"Back", "32"},
			{"LeftStick", "64"},
			{"RightStick", "128"},
			{"LeftShoulder", "256"},
			{"RightShoulder", "512"},
			{"BigButton", "2048"},
			{"A", "4096"},
			{"B", "8192"},
			{"X", "16384"},
			{"Y", "32768"},
			{"RightThumbstickUp", "16777216"},
			{"RightThumbstickDown", "33554432"},
			{"RightThumbstickRight", "67108864"},
			{"RightThumbstickLeft", "134217728"},
			{"LeftThumbstickUp", "268435456"},
			{"LeftThumbstickDown", "536870912"},
			{"LeftThumbstickRight", "1073741824"},
			{"LeftThumbstickLeft", "2097152"},
			{"LeftTrigger", "8388608"},
			{"RightTrigger", "4194304"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Audio.MicrophoneState",
		Flags:    false,
		Values: []enumLiteral{
			{"Started", "0"},
			{"Stopped", "1"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Graphics.FillMode",
		Flags:    false,
		Values: []enumLiteral{
			{"Solid", "0"},
			{"WireFrame", "1"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Media.MediaSourceType",
		Flags:    false,
		Values: []enumLiteral{
			{"LocalDevice", "0"},
			{"WindowsMediaConnect", "4"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Audio.SoundState",
		Flags:    false,
		Values: []enumLiteral{
			{"Playing", "0"},
			{"Paused", "1"},
			{"Stopped", "2"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Graphics.CullMode",
		Flags:    false,
		Values: []enumLiteral{
			{"None", "0"},
			{"CullClockwiseFace", "1"},
			{"CullCounterClockwiseFace", "2"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Graphics.GraphicsDeviceStatus",
		Flags:    false,
		Values: []enumLiteral{
			{"Normal", "0"},
			{"Lost", "1"},
			{"NotReset", "2"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Graphics.TextureAddressMode",
		Flags:    false,
		Values: []enumLiteral{
			{"Wrap", "0"},
			{"Clamp", "1"},
			{"Mirror", "2"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Input.GamePadDeadZone",
		Flags:    false,
		Values: []enumLiteral{
			{"None", "0"},
			{"IndependentAxes", "1"},
			{"Circular", "2"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Media.VideoSoundtrackType",
		Flags:    false,
		Values: []enumLiteral{
			{"Music", "0"},
			{"Dialog", "1"},
			{"MusicAndDialog", "2"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Graphics.PresentInterval",
		Flags:    false,
		Values: []enumLiteral{
			{"Default", "0"},
			{"One", "1"},
			{"Two", "2"},
			{"Immediate", "3"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Graphics.PrimitiveType",
		Flags:    false,
		Values: []enumLiteral{
			{"TriangleList", "0"},
			{"TriangleStrip", "1"},
			{"LineList", "2"},
			{"LineStrip", "3"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Input.Touch.TouchLocationState",
		Flags:    false,
		Values: []enumLiteral{
			{"Invalid", "0"},
			{"Released", "1"},
			{"Pressed", "2"},
			{"Moved", "3"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Graphics.BlendFunction",
		Flags:    false,
		Values: []enumLiteral{
			{"Add", "0"},
			{"Subtract", "1"},
			{"ReverseSubtract", "2"},
			{"Min", "3"},
			{"Max", "4"},
		},
	},
}

// foundation15Enums is the Foundation-15 pure-managed batch B enum cluster:
// the last five ordinary and flags enums whose only public-signature
// dependency is System.Int32. Completing them closes the safe pure-managed
// leaf-enum category entirely.
var foundation15Enums = []foundation14Enum{
	{
		Identity: "Microsoft.Xna.Framework.Graphics.ColorWriteChannels",
		Flags:    true,
		Values: []enumLiteral{
			{"None", "0"},
			{"Red", "1"},
			{"Green", "2"},
			{"Blue", "4"},
			{"Alpha", "8"},
			{"All", "15"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Graphics.StencilOperation",
		Flags:    false,
		Values: []enumLiteral{
			{"Keep", "0"},
			{"Zero", "1"},
			{"Replace", "2"},
			{"Increment", "3"},
			{"Decrement", "4"},
			{"IncrementSaturation", "5"},
			{"DecrementSaturation", "6"},
			{"Invert", "7"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Graphics.TextureFilter",
		Flags:    false,
		Values: []enumLiteral{
			{"Linear", "0"},
			{"Point", "1"},
			{"Anisotropic", "2"},
			{"LinearMipPoint", "3"},
			{"PointMipLinear", "4"},
			{"MinLinearMagPointMipLinear", "5"},
			{"MinLinearMagPointMipPoint", "6"},
			{"MinPointMagLinearMipLinear", "7"},
			{"MinPointMagLinearMipPoint", "8"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Input.GamePadType",
		Flags:    false,
		Values: []enumLiteral{
			{"Unknown", "0"},
			{"GamePad", "1"},
			{"Wheel", "2"},
			{"ArcadeStick", "3"},
			{"FlightStick", "4"},
			{"DancePad", "5"},
			{"Guitar", "6"},
			{"AlternateGuitar", "7"},
			{"DrumKit", "8"},
			{"BigButtonPad", "768"},
		},
	},
	{
		Identity: "Microsoft.Xna.Framework.Graphics.Blend",
		Flags:    false,
		Values: []enumLiteral{
			{"One", "0"},
			{"Zero", "1"},
			{"SourceColor", "2"},
			{"InverseSourceColor", "3"},
			{"SourceAlpha", "4"},
			{"InverseSourceAlpha", "5"},
			{"DestinationColor", "6"},
			{"InverseDestinationColor", "7"},
			{"DestinationAlpha", "8"},
			{"InverseDestinationAlpha", "9"},
			{"BlendFactor", "10"},
			{"InverseBlendFactor", "11"},
			{"SourceAlphaSaturation", "12"},
		},
	},
}

// allBatchEnums is every pinned enum measured by the shared table-driven
// closure category, across milestones.
func allBatchEnums() []foundation14Enum {
	all := make([]foundation14Enum, 0, len(foundation14Enums)+len(foundation15Enums))
	all = append(all, foundation14Enums...)
	all = append(all, foundation15Enums...)
	return all
}

func measureFoundation14EnumClosures(expected *expectedSurface, actual *actualSurface, typeDiagnostics map[string]int) []enumClosure {
	return measureBatchEnumClosures(expected, actual, typeDiagnostics, foundation14Enums)
}

func measureBatchEnumClosures(expected *expectedSurface, actual *actualSurface, typeDiagnostics map[string]int, batch []foundation14Enum) []enumClosure {
	measurements := make([]enumClosure, 0, len(batch))
	for _, pinned := range batch {
		measurements = append(measurements, measureEnumClosure(expected, actual, typeDiagnostics, pinned))
	}
	return measurements
}

// measureEnumClosure applies the established enum-storage rule to one pinned
// enum: the synthetic value__ storage field is never projected, every named
// literal must exist with its exact raw value, the Go type must be a named
// int32, and the xna:flags marker must match the pinned flags bit exactly.
func measureEnumClosure(expected *expectedSurface, actual *actualSurface, typeDiagnostics map[string]int, pinned foundation14Enum) enumClosure {
	measurement := enumClosure{
		XNA:                  pinned.Identity,
		SourceTypes:          1,
		ExpectedKind:         "enum",
		ActualKind:           "missing",
		UnderlyingType:       "missing",
		ExpectedFlags:        pinned.Flags,
		ValueStorageExcluded: true,
		Status:               "FAIL",
	}
	owner := expected.typeForXNA(pinned.Identity)
	if owner == nil {
		return measurement
	}
	measurement.GoName = owner.GoName
	measurement.PackagePath = owner.PackagePath
	measurement.SourceIdentities = owner.SourceMembers
	measurement.ExpectedGoIdentities = len(owner.Members)
	measurement.LocalDiagnostics = typeDiagnostics[owner.XNA]
	if target := actual.Types[owner.Key]; target != nil {
		measurement.TargetTypes = 1
		measurement.ActualKind = target.Kind
		measurement.UnderlyingType = target.Underlying
		measurement.Flags = target.FlagsMarker
	}

	valuesPass := true
	for _, wanted := range pinned.Values {
		row := enumValueMeasurement{
			Name:          wanted.Name,
			ExpectedValue: wanted.Value,
			ActualValue:   "missing",
			Status:        "FAIL",
		}
		key := symbolKey{Package: owner.PackagePath, Name: owner.GoName + wanted.Name}
		expectedMember := expected.Members[key]
		actualMember := actual.Members[key]
		if actualMember != nil {
			measurement.TargetGoIdentities++
			if actualMember.Value != nil {
				row.ActualValue = normalizeInteger(*actualMember.Value)
			}
		}
		if expectedMember != nil && expectedMember.EnumValue != nil &&
			normalizeInteger(*expectedMember.EnumValue) == wanted.Value && row.ActualValue == wanted.Value {
			row.Status = "PASS"
		} else {
			valuesPass = false
		}
		measurement.Values = append(measurement.Values, row)
	}
	for key := range actual.Members {
		if key.Package != owner.PackagePath || !strings.HasPrefix(key.Name, owner.GoName) {
			continue
		}
		if strings.EqualFold(strings.TrimPrefix(key.Name, owner.GoName), "value__") {
			measurement.ValueStorageExcluded = false
		}
	}
	if measurement.SourceIdentities == len(pinned.Values)+1 && measurement.ExpectedGoIdentities == len(pinned.Values) &&
		measurement.TargetTypes == 1 && measurement.TargetGoIdentities == len(pinned.Values) && measurement.LocalDiagnostics == 0 &&
		measurement.ActualKind == "named" && measurement.UnderlyingType == "int32" && measurement.Flags == pinned.Flags &&
		measurement.ValueStorageExcluded && len(measurement.Values) == len(pinned.Values) && valuesPass {
		measurement.Status = "PASS"
	}
	return measurement
}

func measureButtonStateClosure(expected *expectedSurface, actual *actualSurface, typeDiagnostics map[string]int) buttonStateClosure {
	measurement := buttonStateClosure{
		SourceTypes:          1,
		ExpectedKind:         "enum",
		ActualKind:           "missing",
		UnderlyingType:       "missing",
		ValueStorageExcluded: true,
		Status:               "FAIL",
	}
	owner := expected.typeForXNA(buttonStateIdentity)
	if owner == nil {
		return measurement
	}
	measurement.SourceIdentities = owner.SourceMembers
	measurement.ExpectedGoIdentities = len(owner.Members)
	measurement.LocalDiagnostics = typeDiagnostics[owner.XNA]
	if target := actual.Types[owner.Key]; target != nil {
		measurement.TargetTypes = 1
		measurement.ActualKind = target.Kind
		measurement.UnderlyingType = target.Underlying
		measurement.Flags = target.FlagsMarker
	}

	valuesPass := true
	for _, wanted := range buttonStateValues {
		row := enumValueMeasurement{
			Name:          wanted.Name,
			ExpectedValue: wanted.Value,
			ActualValue:   "missing",
			Status:        "FAIL",
		}
		key := symbolKey{Package: owner.PackagePath, Name: owner.GoName + wanted.Name}
		expectedMember := expected.Members[key]
		actualMember := actual.Members[key]
		if actualMember != nil {
			measurement.TargetGoIdentities++
			if actualMember.Value != nil {
				row.ActualValue = normalizeInteger(*actualMember.Value)
			}
		}
		if expectedMember != nil && expectedMember.EnumValue != nil &&
			normalizeInteger(*expectedMember.EnumValue) == wanted.Value && row.ActualValue == wanted.Value {
			row.Status = "PASS"
		} else {
			valuesPass = false
		}
		measurement.Values = append(measurement.Values, row)
	}
	for key := range actual.Members {
		if key.Package != owner.PackagePath || !strings.HasPrefix(key.Name, owner.GoName) {
			continue
		}
		if strings.EqualFold(strings.TrimPrefix(key.Name, owner.GoName), "value__") {
			measurement.ValueStorageExcluded = false
		}
	}
	if measurement.SourceIdentities == 3 && measurement.ExpectedGoIdentities == 2 &&
		measurement.TargetTypes == 1 && measurement.TargetGoIdentities == 2 && measurement.LocalDiagnostics == 0 &&
		measurement.ActualKind == "named" && measurement.UnderlyingType == "int32" && !measurement.Flags &&
		measurement.ValueStorageExcluded && len(measurement.Values) == 2 && valuesPass {
		measurement.Status = "PASS"
	}
	return measurement
}

func measureGraphicsProfileClosure(expected *expectedSurface, actual *actualSurface, typeDiagnostics map[string]int) graphicsProfileClosure {
	measurement := graphicsProfileClosure{
		SourceTypes:          1,
		ExpectedKind:         "enum",
		ActualKind:           "missing",
		UnderlyingType:       "missing",
		ValueStorageExcluded: true,
		Status:               "FAIL",
	}
	owner := expected.typeForXNA(graphicsProfileIdentity)
	if owner == nil {
		return measurement
	}
	measurement.SourceIdentities = owner.SourceMembers
	measurement.ExpectedGoIdentities = len(owner.Members)
	measurement.LocalDiagnostics = typeDiagnostics[owner.XNA]
	if target := actual.Types[owner.Key]; target != nil {
		measurement.TargetTypes = 1
		measurement.ActualKind = target.Kind
		measurement.UnderlyingType = target.Underlying
		measurement.Flags = target.FlagsMarker
	}

	valuesPass := true
	for _, wanted := range graphicsProfileValues {
		row := enumValueMeasurement{
			Name:          wanted.Name,
			ExpectedValue: wanted.Value,
			ActualValue:   "missing",
			Status:        "FAIL",
		}
		key := symbolKey{Package: owner.PackagePath, Name: owner.GoName + wanted.Name}
		expectedMember := expected.Members[key]
		actualMember := actual.Members[key]
		if actualMember != nil {
			measurement.TargetGoIdentities++
			if actualMember.Value != nil {
				row.ActualValue = normalizeInteger(*actualMember.Value)
			}
		}
		if expectedMember != nil && expectedMember.EnumValue != nil &&
			normalizeInteger(*expectedMember.EnumValue) == wanted.Value && row.ActualValue == wanted.Value {
			row.Status = "PASS"
		} else {
			valuesPass = false
		}
		measurement.Values = append(measurement.Values, row)
	}
	for key := range actual.Members {
		if key.Package != owner.PackagePath || !strings.HasPrefix(key.Name, owner.GoName) {
			continue
		}
		if strings.EqualFold(strings.TrimPrefix(key.Name, owner.GoName), "value__") {
			measurement.ValueStorageExcluded = false
		}
	}
	if measurement.SourceIdentities == 3 && measurement.ExpectedGoIdentities == 2 &&
		measurement.TargetTypes == 1 && measurement.TargetGoIdentities == 2 && measurement.LocalDiagnostics == 0 &&
		measurement.ActualKind == "named" && measurement.UnderlyingType == "int32" && !measurement.Flags &&
		measurement.ValueStorageExcluded && len(measurement.Values) == 2 && valuesPass {
		measurement.Status = "PASS"
	}
	return measurement
}

func measureDepthFormatClosure(expected *expectedSurface, actual *actualSurface, typeDiagnostics map[string]int) depthFormatClosure {
	measurement := depthFormatClosure{
		SourceTypes:          1,
		ExpectedKind:         "enum",
		ActualKind:           "missing",
		UnderlyingType:       "missing",
		ValueStorageExcluded: true,
		Status:               "FAIL",
	}
	owner := expected.typeForXNA(depthFormatIdentity)
	if owner == nil {
		return measurement
	}
	measurement.SourceIdentities = owner.SourceMembers
	measurement.ExpectedGoIdentities = len(owner.Members)
	measurement.LocalDiagnostics = typeDiagnostics[owner.XNA]
	if target := actual.Types[owner.Key]; target != nil {
		measurement.TargetTypes = 1
		measurement.ActualKind = target.Kind
		measurement.UnderlyingType = target.Underlying
		measurement.Flags = target.FlagsMarker
	}

	valuesPass := true
	for _, wanted := range depthFormatValues {
		row := enumValueMeasurement{
			Name:          wanted.Name,
			ExpectedValue: wanted.Value,
			ActualValue:   "missing",
			Status:        "FAIL",
		}
		key := symbolKey{Package: owner.PackagePath, Name: owner.GoName + wanted.Name}
		expectedMember := expected.Members[key]
		actualMember := actual.Members[key]
		if actualMember != nil {
			measurement.TargetGoIdentities++
			if actualMember.Value != nil {
				row.ActualValue = normalizeInteger(*actualMember.Value)
			}
		}
		if expectedMember != nil && expectedMember.EnumValue != nil &&
			normalizeInteger(*expectedMember.EnumValue) == wanted.Value && row.ActualValue == wanted.Value {
			row.Status = "PASS"
		} else {
			valuesPass = false
		}
		measurement.Values = append(measurement.Values, row)
	}
	for key := range actual.Members {
		if key.Package != owner.PackagePath || !strings.HasPrefix(key.Name, owner.GoName) {
			continue
		}
		if strings.EqualFold(strings.TrimPrefix(key.Name, owner.GoName), "value__") {
			measurement.ValueStorageExcluded = false
		}
	}
	if measurement.SourceIdentities == 5 && measurement.ExpectedGoIdentities == 4 &&
		measurement.TargetTypes == 1 && measurement.TargetGoIdentities == 4 && measurement.LocalDiagnostics == 0 &&
		measurement.ActualKind == "named" && measurement.UnderlyingType == "int32" && !measurement.Flags &&
		measurement.ValueStorageExcluded && len(measurement.Values) == 4 && valuesPass {
		measurement.Status = "PASS"
	}
	return measurement
}

func measureSurfaceFormatClosure(expected *expectedSurface, actual *actualSurface, typeDiagnostics map[string]int) surfaceFormatClosure {
	measurement := surfaceFormatClosure{
		SourceTypes:          1,
		ExpectedKind:         "enum",
		ActualKind:           "missing",
		UnderlyingType:       "missing",
		ValueStorageExcluded: true,
		Status:               "FAIL",
	}
	owner := expected.typeForXNA(surfaceFormatIdentity)
	if owner == nil {
		return measurement
	}
	measurement.SourceIdentities = owner.SourceMembers
	measurement.ExpectedGoIdentities = len(owner.Members)
	measurement.LocalDiagnostics = typeDiagnostics[owner.XNA]
	if target := actual.Types[owner.Key]; target != nil {
		measurement.TargetTypes = 1
		measurement.ActualKind = target.Kind
		measurement.UnderlyingType = target.Underlying
		measurement.Flags = target.FlagsMarker
	}

	valuesPass := true
	for _, wanted := range surfaceFormatValues {
		row := enumValueMeasurement{
			Name:          wanted.Name,
			ExpectedValue: wanted.Value,
			ActualValue:   "missing",
			Status:        "FAIL",
		}
		key := symbolKey{Package: owner.PackagePath, Name: owner.GoName + wanted.Name}
		expectedMember := expected.Members[key]
		actualMember := actual.Members[key]
		if actualMember != nil {
			measurement.TargetGoIdentities++
			if actualMember.Value != nil {
				row.ActualValue = normalizeInteger(*actualMember.Value)
			}
		}
		if expectedMember != nil && expectedMember.EnumValue != nil &&
			normalizeInteger(*expectedMember.EnumValue) == wanted.Value && row.ActualValue == wanted.Value {
			row.Status = "PASS"
		} else {
			valuesPass = false
		}
		measurement.Values = append(measurement.Values, row)
	}
	for key := range actual.Members {
		if key.Package != owner.PackagePath || !strings.HasPrefix(key.Name, owner.GoName) {
			continue
		}
		if strings.EqualFold(strings.TrimPrefix(key.Name, owner.GoName), "value__") {
			measurement.ValueStorageExcluded = false
		}
	}
	if measurement.SourceIdentities == 21 && measurement.ExpectedGoIdentities == 20 &&
		measurement.TargetTypes == 1 && measurement.TargetGoIdentities == 20 && measurement.LocalDiagnostics == 0 &&
		measurement.ActualKind == "named" && measurement.UnderlyingType == "int32" && !measurement.Flags &&
		measurement.ValueStorageExcluded && len(measurement.Values) == 20 && valuesPass {
		measurement.Status = "PASS"
	}
	return measurement
}

func measureClearOptionsClosure(expected *expectedSurface, actual *actualSurface, typeDiagnostics map[string]int) clearOptionsClosure {
	measurement := clearOptionsClosure{
		SourceTypes:          1,
		ExpectedKind:         "enum",
		ActualKind:           "missing",
		UnderlyingType:       "missing",
		TargetValue:          "missing",
		DepthBufferValue:     "missing",
		StencilValue:         "missing",
		ValueStorageExcluded: true,
		Status:               "FAIL",
	}
	owner := expected.typeForXNA(clearOptionsIdentity)
	if owner == nil {
		return measurement
	}
	measurement.SourceIdentities = owner.SourceMembers
	measurement.ExpectedGoIdentities = len(owner.Members)
	measurement.LocalDiagnostics = typeDiagnostics[owner.XNA]
	measurement.NamedZeroMember = enumHasNamedZero(expected, owner)
	if target := actual.Types[owner.Key]; target != nil {
		measurement.TargetTypes = 1
		measurement.ActualKind = target.Kind
		measurement.UnderlyingType = target.Underlying
		measurement.Flags = target.FlagsMarker
	}
	for _, key := range owner.Members {
		member := expected.Members[key]
		actualMember := actual.Members[key]
		if actualMember == nil {
			continue
		}
		measurement.TargetGoIdentities++
		value := "missing"
		if actualMember.Value != nil {
			value = normalizeInteger(*actualMember.Value)
		}
		switch member.GoName {
		case "ClearOptionsTarget":
			measurement.TargetValue = value
		case "ClearOptionsDepthBuffer":
			measurement.DepthBufferValue = value
		case "ClearOptionsStencil":
			measurement.StencilValue = value
		}
	}
	for key := range actual.Members {
		if key.Package != owner.PackagePath || !strings.HasPrefix(key.Name, owner.GoName) {
			continue
		}
		suffix := strings.TrimPrefix(key.Name, owner.GoName)
		switch {
		case strings.EqualFold(suffix, "value__"):
			measurement.ValueStorageExcluded = false
		case suffix == "None":
			measurement.ClearOptionsNonePresent = true
		case suffix == "Default":
			measurement.ClearOptionsDefaultPresent = true
		case suffix == "All":
			measurement.ClearOptionsAllPresent = true
		}
	}
	if measurement.SourceIdentities == 4 && measurement.ExpectedGoIdentities == 3 &&
		measurement.TargetTypes == 1 && measurement.TargetGoIdentities == 3 && measurement.LocalDiagnostics == 0 &&
		measurement.ActualKind == "named" && measurement.UnderlyingType == "int32" && measurement.Flags &&
		measurement.TargetValue == "1" && measurement.DepthBufferValue == "2" && measurement.StencilValue == "4" &&
		measurement.ValueStorageExcluded && !measurement.NamedZeroMember && !measurement.ClearOptionsNonePresent &&
		!measurement.ClearOptionsDefaultPresent && !measurement.ClearOptionsAllPresent {
		measurement.Status = "PASS"
	}
	return measurement
}

// enumHasNamedZero measures the source-declared enum literals. The Go zero
// value is always representable and does not require a synthetic CLR name.
func enumHasNamedZero(expected *expectedSurface, owner *expectedType) bool {
	if owner == nil || owner.Kind != "enum" {
		return false
	}
	for _, key := range owner.Members {
		member := expected.Members[key]
		if member != nil && member.EnumValue != nil && normalizeInteger(*member.EnumValue) == "0" {
			return true
		}
	}
	return false
}

func measureBufferUsageClosure(expected *expectedSurface, actual *actualSurface, typeDiagnostics map[string]int) bufferUsageClosure {
	measurement := bufferUsageClosure{
		SourceTypes:          1,
		ExpectedKind:         "enum",
		ActualKind:           "missing",
		UnderlyingType:       "missing",
		NoneValue:            "missing",
		WriteOnlyValue:       "missing",
		ValueStorageExcluded: true,
		Status:               "FAIL",
	}
	owner := expected.typeForXNA(bufferUsageIdentity)
	if owner == nil {
		return measurement
	}
	measurement.SourceIdentities = owner.SourceMembers
	measurement.ExpectedGoIdentities = len(owner.Members)
	measurement.LocalDiagnostics = typeDiagnostics[owner.XNA]
	if target := actual.Types[owner.Key]; target != nil {
		measurement.TargetTypes = 1
		measurement.ActualKind = target.Kind
		measurement.UnderlyingType = target.Underlying
		measurement.Flags = target.FlagsMarker
	}
	for _, key := range owner.Members {
		member := expected.Members[key]
		actualMember := actual.Members[key]
		if actualMember == nil {
			continue
		}
		measurement.TargetGoIdentities++
		value := "missing"
		if actualMember.Value != nil {
			value = normalizeInteger(*actualMember.Value)
		}
		switch member.GoName {
		case "BufferUsageNone":
			measurement.NoneValue = value
		case "BufferUsageWriteOnly":
			measurement.WriteOnlyValue = value
		}
	}
	for key := range actual.Members {
		if key.Package != owner.PackagePath || !strings.HasPrefix(key.Name, owner.GoName) {
			continue
		}
		if strings.EqualFold(strings.TrimPrefix(key.Name, owner.GoName), "value__") {
			measurement.ValueStorageExcluded = false
		}
	}
	if measurement.SourceIdentities == 3 && measurement.ExpectedGoIdentities == 2 &&
		measurement.TargetTypes == 1 && measurement.TargetGoIdentities == 2 && measurement.LocalDiagnostics == 0 &&
		measurement.ActualKind == "named" && measurement.UnderlyingType == "int32" && measurement.Flags &&
		measurement.NoneValue == "0" && measurement.WriteOnlyValue == "1" && measurement.ValueStorageExcluded {
		measurement.Status = "PASS"
	}
	return measurement
}

// graphicsManagerRemainingMissing is the exact number of GraphicsDeviceManager
// members the binding still does not project, and it is asserted rather than
// bounded on purpose: this closure measurement exists to say how much of the
// type is REACHED, and a range would let the number drift in either direction
// without anyone noticing.
//
// It was 40 from Foundation 13 until Foundation 48 projected the type's nine
// configuration properties, its two static defaults, ApplyChanges and
// ToggleFullScreen. The six that remain need GraphicsDeviceInformation, which
// needs GraphicsAdapter.
const graphicsManagerRemainingMissing = 20

func measureDisplayOrientationClosure(expected *expectedSurface, actual *actualSurface, typeDiagnostics map[string]int) displayOrientationClosure {
	measurement := displayOrientationClosure{SourceTypes: 2, Status: "FAIL"}

	display := expected.typeForXNA(displayOrientationIdentity)
	if display != nil {
		row := displayOrientationSliceMeasurement{
			XNA:               display.XNA,
			GoName:            display.GoName,
			Scope:             "complete type",
			SourceMembers:     display.SourceMembers,
			ExpectedGoMembers: len(display.Members),
			LocalDiagnostics:  typeDiagnostics[display.XNA],
			ExpectedKind:      display.Kind,
			ActualKind:        "missing",
		}
		measurement.SourceIdentities += row.SourceMembers
		measurement.MappedGoIdentities += row.ExpectedGoMembers
		measurement.DisplayOrientationLocalDiagnostics = row.LocalDiagnostics
		if target := actual.Types[display.Key]; target != nil {
			measurement.TargetTypes++
			row.ActualKind = target.Kind
			row.ActualUnderlying = target.Underlying
			for _, key := range display.Members {
				if actual.Members[key] != nil {
					row.TargetGoMembers++
				}
			}
			measurement.TargetGoIdentities += row.TargetGoMembers
		}
		measurement.SliceMeasurements = append(measurement.SliceMeasurements, row)
	}

	manager := expected.typeForXNA(graphicsManagerIdentity)
	if manager != nil {
		row := displayOrientationSliceMeasurement{
			XNA:               manager.XNA + "." + supportedOrientationsName,
			GoName:            manager.GoName + "." + supportedOrientationsName,
			Scope:             "selected property getter and setter",
			SourceMembers:     1,
			ExpectedGoMembers: 2,
			ExpectedKind:      "property",
			ActualKind:        "missing",
		}
		measurement.SourceIdentities += row.SourceMembers
		measurement.MappedGoIdentities += row.ExpectedGoMembers
		if target := actual.Types[manager.Key]; target != nil {
			measurement.TargetTypes++
			row.ActualKind = "property"
		}
		for _, key := range manager.Members {
			member := expected.Members[key]
			if member == nil {
				continue
			}
			if member.SourceKind == "property" && strings.Contains(member.XNA, "::"+supportedOrientationsName+"(") {
				actualMember := actual.Members[key]
				if actualMember == nil {
					row.LocalDiagnostics++
					continue
				}
				row.TargetGoMembers++
				local := report{Summary: make(map[string]int)}
				compareMember(&local, member, actualMember)
				row.LocalDiagnostics += len(local.Diagnostics)
				continue
			}
			if actual.Members[key] == nil {
				measurement.GraphicsManagerRemainingMissing++
			}
		}
		measurement.TargetGoIdentities += row.TargetGoMembers
		measurement.SupportedPropertyLocalDiagnostics = row.LocalDiagnostics
		measurement.SliceMeasurements = append(measurement.SliceMeasurements, row)
	}

	if measurement.SourceIdentities == 6 && measurement.MappedGoIdentities == 6 &&
		measurement.TargetTypes == 2 && measurement.TargetGoIdentities == 6 &&
		measurement.DisplayOrientationLocalDiagnostics == 0 && measurement.SupportedPropertyLocalDiagnostics == 0 &&
		measurement.GraphicsManagerRemainingMissing == graphicsManagerRemainingMissing {
		measurement.Status = "PASS"
	}
	return measurement
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
	if key.Receiver != "" {
		return expected.Types[symbolKey{Package: key.Package, Name: key.Receiver}]
	}
	var best *expectedType
	for _, candidate := range expected.Types {
		if candidate.PackagePath != key.Package || !strings.HasPrefix(key.Name, candidate.GoName) {
			continue
		}
		if best == nil || len(candidate.GoName) > len(best.GoName) {
			best = candidate
		}
	}
	return best
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

	// The required set depends on how the type acquires ICollection<T>.
	//
	// A type that DECLARES the interface, such as TouchCollection or
	// CurveKeyCollection, declares its members publicly in the XNA metadata,
	// so all eight are public surface and all eight must be projected.
	//
	// A type that INHERITS it from a supported BCL base acquires whatever
	// that base makes public, which is not the same set. Collection<T>
	// implements ICollection<T>.IsReadOnly EXPLICITLY --
	//
	//	.method private hidebysig newslot specialname virtual final
	//	        instance bool 'System.Collections.Generic.ICollection<T>.get_IsReadOnly'()
	//
	// -- so it is not public surface at all, and `new Collection<int>()
	// .IsReadOnly` does not compile in C# either. Requiring it here would
	// force CNA-Go to invent a public member the CLR does not expose, which
	// is exactly the failure the no-surface rule exists to prevent. The
	// required set for such a type is therefore the adapter's measured public
	// inventory, and every member the adapter records as excluded is checked
	// to be ABSENT rather than present.
	required := []string{"Add", "Clear", "Contains", "CopyTo", "Remove", "Count", "IsReadOnly", "GetEnumerator"}
	if adapter, composed := composedBaseAdapter(owner); composed {
		required = nil
		for _, entry := range adapter.Members {
			required = append(required, entry.Member.Name)
		}
		verifyExcludedBaseMembersAbsent(result, expected, actual, owner, adapter)
	}
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
		addDiagnostic(result, diagnostic{Category: "ERROR_MAPPING_MISMATCH", XNA: expected.XNA, Go: actual.Key.String(), Message: errorMappingMessage(expected, expectedError, actualError)})
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
	if expected.SourceKind == "event" {
		verifyEventAccessorProjection(result, expected, actual)
	}
}

// eventHandlerSpelling matches the projected System.EventHandler<T> adapter and
// captures the package qualifier and the mapped generic argument, so a defect
// can be named as the event defect it is rather than as an anonymous parameter
// difference.
var eventHandlerSpelling = regexp.MustCompile(`^(framework\.)?EventHandler\[(.+)\]$`)

// eventSubscriptionSpelling matches the projected subscription token.
var eventSubscriptionSpelling = regexp.MustCompile(`^(framework\.)?EventSubscription$`)

// verifyEventAccessorProjection checks one projected event accessor against the
// settled event mapping directly, independently of whether it happens to equal
// the expected signature string.
//
// The rule it enforces is the whole event decision: one CLR event becomes
// exactly two Go accessors; Add takes the typed EventHandler adapter and returns
// the opaque token plus an error; Remove takes the token and returns an error.
// Checking it structurally is what makes a degraded handler -- `any`, a bare
// func, a channel, a raw callback word -- a named event defect instead of a
// generic signature difference.
func verifyEventAccessorProjection(result *report, expected *expectedMember, actual *actualMember) {
	fail := func(format string, args ...any) {
		addDiagnostic(result, diagnostic{
			Category: "EVENT_MAPPING_MISMATCH", XNA: expected.XNA, Go: actual.Key.String(),
			Message: fmt.Sprintf(format, args...),
		})
	}
	qualifier := ""
	if expected.PackagePath != modulePath+"/Microsoft/Xna/Framework" {
		qualifier = "framework."
	}
	removal := strings.HasPrefix(actual.Key.Name, "Remove")
	if removal {
		if len(actual.Parameters) != 1 || !eventSubscriptionSpelling.MatchString(actual.Parameters[0]) {
			fail("event removal must take exactly one %sEventSubscription, found %v", qualifier, actual.Parameters)
		} else if actual.Parameters[0] != qualifier+"EventSubscription" {
			fail("event removal token is spelled %q, want %q for package %s", actual.Parameters[0], qualifier+"EventSubscription", expected.PackagePath)
		}
		if !equalStrings(actual.Results, []string{"error"}) {
			fail("event removal must return exactly one error, found %v", actual.Results)
		}
		return
	}
	if len(actual.Parameters) != 1 {
		fail("event registration must take exactly one handler, found %v", actual.Parameters)
		return
	}
	match := eventHandlerSpelling.FindStringSubmatch(actual.Parameters[0])
	if match == nil {
		fail("event registration handler is %q, want the %sEventHandler[T] adapter", actual.Parameters[0], qualifier)
		return
	}
	if match[1] != qualifier {
		fail("event registration handler is spelled %q, want the %sEventHandler qualification for package %s", actual.Parameters[0], qualifier, expected.PackagePath)
	}
	if match[2] == "any" {
		fail("event registration handler degraded its generic argument to any")
	}
	if len(actual.Results) != 2 || !eventSubscriptionSpelling.MatchString(actual.Results[0]) || actual.Results[1] != "error" {
		fail("event registration must return (%sEventSubscription, error), found %v", qualifier, actual.Results)
		return
	}
	if actual.Results[0] != qualifier+"EventSubscription" {
		fail("event registration token is spelled %q, want %q for package %s", actual.Results[0], qualifier+"EventSubscription", expected.PackagePath)
	}
}

// errorMappingMessage names the exact fallibility defect. Because fallibility
// is decided per projected operation, the two accessors of one CLR property
// are separate members that can disagree, so the message distinguishes all
// four accessor cases -- getter expected fallible but projected infallible and
// the reverse, and the same pair for the setter -- from the ordinary
// constructor/method case.
func errorMappingMessage(expected *expectedMember, expectedError, actualError bool) string {
	subject := "operation"
	switch expected.Accessor {
	case "get":
		subject = "property getter"
	case "set":
		subject = "property setter"
	default:
		switch expected.SourceKind {
		case "constructor":
			subject = "constructor"
		case "method":
			subject = "method"
		case "field":
			subject = "field projection"
		case "event":
			subject = "event accessor"
		}
	}
	direction := "expected infallible, projected fallible"
	if expectedError {
		direction = "expected fallible, projected infallible"
	}
	return fmt.Sprintf("%s %s: expected language-added error=%t, found %t", subject, direction, expectedError, actualError)
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

// pointerSizedWord matches a whole-word `uintptr` anywhere inside a mapped
// type expression, so `[]uintptr`, `*uintptr`, and `func(uintptr)` are seen as
// readily as a bare `uintptr`, while an identifier such as `uintptrish` is not.
var pointerSizedWord = regexp.MustCompile(`\buintptr\b`)

func measureLeaks(result *report, expected *expectedSurface, actual *actualSurface) {
	for key, t := range actual.Types {
		if t.Kind != "struct" && t.Kind != "interface" {
			inspectLeakText(result, key.String(), t.Underlying)
			// No XNA type projects to a bare pointer-sized word. A named
			// exported type over uintptr would be a handle type in all but
			// name.
			if pointerSizedWord.MatchString(t.Underlying) {
				addDiagnostic(result, diagnostic{Category: "RAW_HANDLE_LEAK", Go: key.String(), Message: "exported type is defined over a pointer-sized machine word"})
			}
		}
		if strings.Contains(strings.ToLower(key.Name), "nativehandle") || strings.Contains(strings.ToLower(key.Name), "rawhandle") || strings.HasPrefix(strings.ToLower(key.Name), "cna") {
			addDiagnostic(result, diagnostic{Category: "RAW_HANDLE_LEAK", Go: key.String(), Message: "exported type name exposes native-handle/FFI identity"})
		}
	}
	for key, member := range actual.Members {
		for _, value := range append(append([]string(nil), member.Parameters...), member.Results...) {
			inspectLeakText(result, key.String(), value)
		}
		measurePointerSizedWordPositions(result, expected, key, member)
		lower := strings.ToLower(key.Name)
		if strings.Contains(lower, "nativehandle") || strings.Contains(lower, "rawhandle") || strings.HasPrefix(lower, "cna") {
			addDiagnostic(result, diagnostic{Category: "RAW_HANDLE_LEAK", Go: key.String(), Message: "exported member name exposes native-handle/FFI identity"})
		}
	}
}

// measurePointerSizedWordPositions enforces the one narrow exception to the
// raw-handle rule.
//
// A public Go `uintptr` is an allowed *language projection* only where the
// authoritative XNA metadata declares System.IntPtr at that exact public
// signature position. Because System.IntPtr is the only source type that maps
// to uintptr, the expected surface already carries uintptr at precisely the
// admitted positions, and positional agreement with it is the whole test.
//
// Everything else still leaks: a uintptr the source never declared, a uintptr
// that has drifted from a parameter to a result or between indices, and any
// uintptr on a member the reference profile does not declare at all -- which
// includes an invented member and a language adapter, none of which carries a
// pointer-sized word.
//
// The admission is about the *bit value* the XNA contract carries at that
// position. It does not make the value dereferenceable, does not make it a CNA
// or SDL handle, and does not admit unsafe.Pointer or an implementation-only
// native pointer anywhere; those remain caught by inspectLeakText and by the
// name rules above.
func measurePointerSizedWordPositions(result *report, expected *expectedSurface, key symbolKey, member *actualMember) {
	var admittedParameters, admittedResults []string
	if expected != nil {
		if em := expected.Members[key]; em != nil {
			admittedParameters, admittedResults = em.Parameters, em.Results
		} else if witness := expected.InterfaceWitnesses[key]; witness != nil {
			admittedParameters, admittedResults = witness.Parameters, witness.Results
		}
	}
	leak := func(role string, index int, text string) {
		addDiagnostic(result, diagnostic{
			Category: "RAW_HANDLE_LEAK",
			Go:       key.String(),
			Message: fmt.Sprintf("exported %s %d is %q, but the reference profile declares no System.IntPtr at that position",
				role, index, text),
		})
	}
	admits := func(admitted []string, index int) bool {
		return index < len(admitted) && pointerSizedWord.MatchString(admitted[index])
	}
	for index, text := range member.Parameters {
		if pointerSizedWord.MatchString(text) && !admits(admittedParameters, index) {
			leak("parameter", index, text)
		}
	}
	for index, text := range member.Results {
		if pointerSizedWord.MatchString(text) && !admits(admittedResults, index) {
			leak("result", index, text)
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

// foundation15ValueStructs is the Foundation-15 pure managed value-struct
// cluster. Every entry is a System.ValueType whose reference implementation is
// deterministic managed value work read from the retained XNA assemblies, so
// none of them may gain a synthetic error result, a native route, or a device
// dependency.
var foundation15ValueStructs = []string{
	"Microsoft.Xna.Framework.Input.GamePadThumbSticks",
	"Microsoft.Xna.Framework.Input.GamePadTriggers",
	"Microsoft.Xna.Framework.Input.GamePadDPad",
	"Microsoft.Xna.Framework.Input.GamePadButtons",
	"Microsoft.Xna.Framework.Input.MouseState",
	"Microsoft.Xna.Framework.Input.Touch.GestureSample",
	"Microsoft.Xna.Framework.Input.Touch.TouchLocation",
}

// foundation16ValueStructs is the Foundation-16 closure. GamePadState is the
// value struct that cluster B unlocked: its constructors take exactly the
// Foundation-15 game pad values, and its IsButtonDown reproduces the internal
// XInput packing and dead-zone rules as pure managed arithmetic.
var foundation16ValueStructs = []string{
	"Microsoft.Xna.Framework.Input.GamePadState",
}

// allValueStructs is every pinned value struct measured by the shared
// table-driven closure category, across milestones.
func allValueStructs() []string {
	all := make([]string, 0, len(foundation15ValueStructs)+len(foundation16ValueStructs))
	all = append(all, foundation15ValueStructs...)
	all = append(all, foundation16ValueStructs...)
	return all
}

func measureValueStructClosures(expected *expectedSurface, actual *actualSurface, typeDiagnostics map[string]int, batch []string) []valueStructClosure {
	measurements := make([]valueStructClosure, 0, len(batch))
	for _, identity := range batch {
		measurements = append(measurements, measureValueStructClosure(expected, actual, typeDiagnostics, identity))
	}
	return measurements
}

func measureValueStructClosure(expected *expectedSurface, actual *actualSurface, typeDiagnostics map[string]int, identity string) valueStructClosure {
	measurement := valueStructClosure{
		XNA:          identity,
		SourceTypes:  1,
		ExpectedKind: "struct",
		ActualKind:   "missing",
		Status:       "FAIL",
	}
	owner := expected.typeForXNA(identity)
	if owner == nil {
		return measurement
	}
	measurement.GoName = owner.GoName
	measurement.PackagePath = owner.PackagePath
	measurement.SourceIdentities = owner.SourceMembers
	measurement.ExpectedGoIdentities = len(owner.Members)
	measurement.LocalDiagnostics = typeDiagnostics[owner.XNA]
	measurement.BaseType = owner.BaseType
	if target := actual.Types[owner.Key]; target != nil {
		measurement.TargetTypes = 1
		measurement.ActualKind = target.Kind
	}

	membersPass := true
	for _, key := range owner.Members {
		expectedMember := expected.Members[key]
		row := valueStructMember{
			GoKind:   expectedMember.GoKind,
			Receiver: key.Receiver,
			Name:     key.Name,
			Status:   "FAIL",
		}
		row.ExpectedResults = append([]string(nil), expectedMember.Results...)
		for _, result := range expectedMember.Results {
			if result == "error" {
				measurement.ErrorResults++
			}
		}
		if actualMember := actual.Members[key]; actualMember != nil {
			measurement.TargetGoIdentities++
			row.ActualResults = append([]string(nil), actualMember.Results...)
			for _, result := range actualMember.Results {
				if result == "error" {
					measurement.ErrorResults++
				}
			}
			if equalStrings(row.ExpectedResults, row.ActualResults) {
				row.Status = "PASS"
			}
		}
		if row.Status != "PASS" {
			membersPass = false
		}
		measurement.Members = append(measurement.Members, row)
	}
	if measurement.TargetTypes == 1 && measurement.ActualKind == "struct" &&
		measurement.BaseType == "System.ValueType" && measurement.LocalDiagnostics == 0 &&
		measurement.TargetGoIdentities == measurement.ExpectedGoIdentities &&
		measurement.ExpectedGoIdentities == measurement.SourceIdentities &&
		measurement.ErrorResults == 0 && membersPass {
		measurement.Status = "PASS"
	}
	return measurement
}

// foundation17ManagedClasses is the Foundation-17 pure-managed CLR class
// cluster: the first types admitted under the general class-classification
// rule rather than because they are value types.
//
// Both are declared `class` in Microsoft.Xna.Framework.dll
// (sha256 38e7093f52d7474bbc6256906519781a1210d7da50a1c667b52716fcf49ca130) and
// both keep CLR reference semantics, but neither owns a native object: every
// public accessor is one ldfld/stfld over an assembly-private XACT descriptor
// value plus the managed UnsafeNativeStructures::FlipHandedness.
var foundation17ManagedClasses = []string{
	"Microsoft.Xna.Framework.Audio.AudioListener",
	"Microsoft.Xna.Framework.Audio.AudioEmitter",
}

// foundation19ManagedClasses is the Foundation-19 pure-managed CLR class
// closure. PresentationParameters is the first admitted class whose public
// surface carries a System.IntPtr, and the first descriptor for a runtime the
// binding does not have: it stores a platform window handle without creating,
// resetting, presenting, enumerating, or looking anything up.
var foundation19ManagedClasses = []string{
	"Microsoft.Xna.Framework.Graphics.PresentationParameters",
}

// foundation21ManagedClasses is the Foundation-21 closure. GameServiceContainer
// is a pure managed registry whose only unsatisfied dependency was the declared
// System.IServiceProvider, an interface whose single member the class already
// declares publicly and which therefore adds no projected surface.
var foundation21ManagedClasses = []string{
	"Microsoft.Xna.Framework.GameServiceContainer",
}

// foundation23ManagedClasses is the Foundation-23 closure: the three
// System.EventArgs carriers the base relationship unblocked. Each is one or two
// managed fields behind get-only accessors, and each demonstrates the base rule
// -- a CLR base survives as a measured relationship, never as Go embedding.
//
// GameComponentCollectionEventArgs projects its public constructor; the two
// graphics carriers declare theirs `assembly`, so neither gets one.
var foundation23ManagedClasses = []string{
	"Microsoft.Xna.Framework.GameComponentCollectionEventArgs",
	"Microsoft.Xna.Framework.Graphics.ResourceCreatedEventArgs",
	"Microsoft.Xna.Framework.Graphics.ResourceDestroyedEventArgs",
}

// allManagedClasses is every pinned pure-managed CLR class measured by the
// shared table-driven closure category, across milestones.
func allManagedClasses() []string {
	all := make([]string, 0, len(foundation17ManagedClasses)+len(foundation19ManagedClasses)+
		len(foundation20ValueContracts)+len(foundation21ManagedClasses)+len(foundation23ManagedClasses))
	all = append(all, foundation17ManagedClasses...)
	all = append(all, foundation19ManagedClasses...)
	all = append(all, foundation20ValueContracts...)
	all = append(all, foundation21ManagedClasses...)
	all = append(all, foundation23ManagedClasses...)
	return all
}

func measureManagedClassClosures(expected *expectedSurface, actual *actualSurface, typeDiagnostics map[string]int, batch []string) []managedTypeClosure {
	measurements := make([]managedTypeClosure, 0, len(batch))
	for _, identity := range batch {
		measurements = append(measurements, measureManagedClassClosure(expected, actual, typeDiagnostics, identity))
	}
	return measurements
}

func measureManagedClassClosure(expected *expectedSurface, actual *actualSurface, typeDiagnostics map[string]int, identity string) managedTypeClosure {
	measurement := managedTypeClosure{
		XNA:         identity,
		SourceTypes: 1,
		ActualKind:  "missing",
		PureManaged: pureManagedTypes[identity],
		Status:      "FAIL",
	}
	owner := expected.typeForXNA(identity)
	if owner == nil {
		return measurement
	}
	// The category spans both CLR kinds. A class keeps reference semantics and
	// projects a pointer constructor; a struct keeps value semantics and
	// projects a value constructor. Getting that backwards is the defect the
	// ReferenceProjection check exists to catch, so the expected shape is
	// derived from the pinned metadata rather than assumed.
	measurement.SourceKind = owner.Kind
	measurement.ExpectedKind = owner.Kind
	measurement.ValueSemantics = owner.Kind == "struct"
	measurement.GoName = owner.GoName
	measurement.PackagePath = owner.PackagePath
	measurement.SourceIdentities = owner.SourceMembers
	measurement.ExpectedGoIdentities = len(owner.Members)
	measurement.LocalDiagnostics = typeDiagnostics[owner.XNA]
	measurement.BaseType = owner.BaseType
	if target := actual.Types[owner.Key]; target != nil {
		measurement.TargetTypes = 1
		measurement.ActualKind = target.Kind
	}

	membersPass := true
	getters := make(map[string]bool)
	setters := make(map[string]bool)
	for _, key := range owner.Members {
		em := expected.Members[key]
		row := managedTypeMember{
			XNA:              em.XNA,
			SourceKind:       em.SourceKind,
			Accessor:         em.Accessor,
			GoKind:           em.GoKind,
			Receiver:         key.Receiver,
			Name:             key.Name,
			ExpectedFallible: em.ErrorAdded,
			ExpectedResults:  append([]string(nil), em.Results...),
			Status:           "FAIL",
		}
		switch em.Accessor {
		case "get":
			getters[em.XNA] = true
			if em.ErrorAdded {
				measurement.FallibleGetters++
			}
		case "set":
			setters[em.XNA] = true
			if em.ErrorAdded {
				measurement.FallibleSetters++
			}
		default:
			if em.ErrorAdded {
				measurement.FallibleOperations++
			}
		}
		if em.ErrorAdded {
			measurement.ErrorResults++
		}
		if am := actual.Members[key]; am != nil {
			measurement.TargetGoIdentities++
			row.ActualResults = append([]string(nil), am.Results...)
			row.ActualFallible = len(am.Results) > 0 && am.Results[len(am.Results)-1] == "error"
			if equalStrings(row.ExpectedResults, row.ActualResults) &&
				equalStrings(em.Parameters, am.Parameters) &&
				row.ExpectedFallible == row.ActualFallible {
				row.Status = "PASS"
			}
		}
		if row.Status != "PASS" {
			membersPass = false
		}
		measurement.Members = append(measurement.Members, row)
	}
	for xna := range getters {
		if setters[xna] {
			measurement.AccessorPairs++
		}
	}

	// A CLR class keeps reference semantics whether or not it is pure managed,
	// so its constructor must produce a Go pointer; a CLR struct keeps value
	// semantics and must not. Either way, projecting the wrong one silently
	// changes whether two variables share mutations.
	wantedProjection := owner.GoName
	wantedBase := "System.ValueType"
	if owner.Kind == "class" {
		wantedProjection = "*" + owner.GoName
		wantedBase = "System.Object"
	}
	// A CLR class need not derive from System.Object directly. When it derives
	// from a BCL base instead, that base must be a declared relationship whose
	// status permits projection, and the relationship -- not Go structure -- is
	// what carries it. Hardcoding System.Object here would silently drop the
	// base, which is the defect the relationship table exists to prevent.
	basePass := measurement.BaseType == wantedBase
	measurement.BaseRelationship = "DIRECT"
	if owner.Kind == "class" && !basePass {
		relationship, declared := bclBaseRelationships[baseIdentityWithoutArguments(measurement.BaseType)]
		if declared && relationship.Status == "MAPPED" {
			basePass = true
			measurement.BaseRelationship = relationship.Adapter
		} else {
			measurement.BaseRelationship = "UNDECIDED"
		}
	}
	for _, key := range owner.Members {
		em := expected.Members[key]
		if em.SourceKind != "constructor" || len(em.Results) == 0 {
			continue
		}
		measurement.ReferenceProjection = em.Results[0]
		break
	}
	projectionPass := measurement.ReferenceProjection == wantedProjection
	if measurement.ReferenceProjection == "" {
		// A type with no public constructor cannot state its semantics
		// through one; the type-kind check carries the whole claim there.
		projectionPass = true
	}

	if measurement.TargetTypes == 1 && measurement.ActualKind == "struct" &&
		measurement.PureManaged && basePass &&
		measurement.LocalDiagnostics == 0 &&
		measurement.TargetGoIdentities == measurement.ExpectedGoIdentities &&
		projectionPass && membersPass {
		measurement.Status = "PASS"
	}
	return measurement
}

// foundation20ValueContracts is the Foundation-20 closure. It is the first
// cluster that is simultaneously a CLR value type and fallible: TouchCollection
// is a System.ValueType, so it keeps copy semantics, yet nine of its sixteen
// projected operations carry an error because the reference validates or
// unconditionally throws.
var foundation20ValueContracts = []string{
	"Microsoft.Xna.Framework.Input.Touch.TouchCollection",
	"Microsoft.Xna.Framework.Input.Touch.TouchCollection+Enumerator",
}

// foundation18Interface pins one projected CLR interface contract together
// with the boundary classification its reference implementor IL proves and the
// exact set of operations that classification makes fallible.
type foundation18Interface struct {
	XNA        string
	Boundary   string
	Classified bool
	// FallibleOperations are the projected Go names that must carry an error
	// result because this contract's boundary makes them fallible. Every other
	// projected name must not, apart from EventAccessors.
	FallibleOperations []string
	// EventAccessors are projected Go names that carry an error because of the
	// settled event accessor projection rather than because of the contract's
	// boundary. Keeping them in their own list is what stops an infallible
	// contract's event accessors from reading as evidence of a runtime
	// boundary that its implementor IL does not show.
	EventAccessors []string
}

// foundation18Interfaces is the Foundation-18 interface cluster. The two
// boundaries are measured from the assembly that declares each interface, not
// inferred from the interface kind.
var foundation18Interfaces = []foundation18Interface{
	{
		XNA:                "Microsoft.Xna.Framework.Graphics.IEffectMatrices",
		Boundary:           "PURE_MANAGED",
		Classified:         true,
		FallibleOperations: nil,
	},
	{
		XNA:        "Microsoft.Xna.Framework.Graphics.IEffectFog",
		Boundary:   "MIXED_MANAGED_AND_RUNTIME",
		Classified: true,
		// Only FogColor reaches EffectParameter and therefore D3DX.
		FallibleOperations: []string{"FogColor", "SetFogColor"},
	},
	{
		XNA:                "Microsoft.Xna.Framework.IGameComponent",
		Boundary:           "RUNTIME",
		Classified:         false,
		FallibleOperations: []string{"Initialize"},
	},
	{
		XNA:                "Microsoft.Xna.Framework.IGraphicsDeviceManager",
		Boundary:           "RUNTIME",
		Classified:         false,
		FallibleOperations: []string{"CreateDevice", "BeginDraw", "EndDraw"},
	},
}

// foundation23Interfaces is the Foundation-23 interface cluster: the two
// event-bearing component contracts the event mapping unblocked.
//
// Both are PURE_MANAGED on their own implementor IL. Microsoft.Xna.Framework.Game.dll
// declares both and ships one implementor of each, GameComponent and
// DrawableGameComponent, whose selected operations are one ldfld per property
// getter and a bare `ret` for Update and Draw. Their four event accessors each
// carry an error from the settled event accessor projection, which is why they
// are listed as EventAccessors rather than as FallibleOperations.
var foundation23Interfaces = []foundation18Interface{
	{
		XNA:                "Microsoft.Xna.Framework.IUpdateable",
		Boundary:           "PURE_MANAGED",
		Classified:         true,
		FallibleOperations: nil,
		EventAccessors: []string{
			"AddEnabledChangedHandler", "RemoveEnabledChangedHandler",
			"AddUpdateOrderChangedHandler", "RemoveUpdateOrderChangedHandler",
		},
	},
	{
		XNA:                "Microsoft.Xna.Framework.IDrawable",
		Boundary:           "PURE_MANAGED",
		Classified:         true,
		FallibleOperations: nil,
		EventAccessors: []string{
			"AddVisibleChangedHandler", "RemoveVisibleChangedHandler",
			"AddDrawOrderChangedHandler", "RemoveDrawOrderChangedHandler",
		},
	},
}

// allManagedInterfaces is every pinned interface measured by the shared
// table-driven closure category, across milestones.
func allManagedInterfaces() []foundation18Interface {
	all := make([]foundation18Interface, 0, len(foundation18Interfaces)+len(foundation23Interfaces))
	all = append(all, foundation18Interfaces...)
	all = append(all, foundation23Interfaces...)
	return all
}

func measureManagedInterfaceClosures(expected *expectedSurface, actual *actualSurface, typeDiagnostics map[string]int, batch []foundation18Interface) []managedInterfaceClosure {
	measurements := make([]managedInterfaceClosure, 0, len(batch))
	for _, pinned := range batch {
		measurements = append(measurements, measureManagedInterfaceClosure(expected, actual, typeDiagnostics, pinned))
	}
	return measurements
}

func measureManagedInterfaceClosure(expected *expectedSurface, actual *actualSurface, typeDiagnostics map[string]int, pinned foundation18Interface) managedInterfaceClosure {
	measurement := managedInterfaceClosure{
		XNA:          pinned.XNA,
		SourceTypes:  1,
		ExpectedKind: "interface",
		ActualKind:   "missing",
		Classified:   classifiedInterfaces[pinned.XNA],
		Boundary:     pinned.Boundary,
		Status:       "FAIL",
	}
	owner := expected.typeForXNA(pinned.XNA)
	if owner == nil {
		return measurement
	}
	measurement.GoName = owner.GoName
	measurement.PackagePath = owner.PackagePath
	measurement.SourceIdentities = owner.SourceMembers
	measurement.ExpectedGoIdentities = len(owner.Members)
	measurement.LocalDiagnostics = typeDiagnostics[owner.XNA]
	if target := actual.Types[owner.Key]; target != nil {
		measurement.TargetTypes = 1
		measurement.ActualKind = target.Kind
	}

	fallible := make(map[string]bool, len(pinned.FallibleOperations)+len(pinned.EventAccessors))
	for _, name := range pinned.FallibleOperations {
		fallible[name] = true
	}
	eventAccessor := make(map[string]bool, len(pinned.EventAccessors))
	for _, name := range pinned.EventAccessors {
		fallible[name] = true
		eventAccessor[name] = true
	}
	membersPass := true
	getters := make(map[string]bool)
	setters := make(map[string]bool)
	for _, key := range owner.Members {
		em := expected.Members[key]
		row := managedInterfaceMember{
			XNA:              em.XNA,
			SourceKind:       em.SourceKind,
			Accessor:         em.Accessor,
			Name:             key.Name,
			ExpectedFallible: em.ErrorAdded,
			ExpectedResults:  append([]string(nil), em.Results...),
			Status:           "FAIL",
		}
		// The pinned table, not the mapping tables, decides which operations
		// may be fallible, so a classification edit alone cannot move the
		// boundary without failing here.
		if em.ErrorAdded != fallible[key.Name] {
			membersPass = false
			measurement.Members = append(measurement.Members, row)
			continue
		}
		switch em.Accessor {
		case "get":
			getters[em.XNA] = true
			if em.ErrorAdded {
				measurement.FallibleGetters++
			}
		case "set":
			setters[em.XNA] = true
			if em.ErrorAdded {
				measurement.FallibleSetters++
			}
		default:
			if em.ErrorAdded && !eventAccessor[key.Name] {
				measurement.FallibleOperations++
			}
		}
		if eventAccessor[key.Name] {
			measurement.EventAccessors++
		}
		if em.ErrorAdded {
			measurement.ErrorResults++
		}
		if am := actual.Members[key]; am != nil {
			measurement.TargetGoIdentities++
			row.ActualResults = append([]string(nil), am.Results...)
			row.ActualFallible = len(am.Results) > 0 && am.Results[len(am.Results)-1] == "error"
			if am.Kind == "method" && equalStrings(row.ExpectedResults, row.ActualResults) &&
				equalStrings(em.Parameters, am.Parameters) &&
				row.ExpectedFallible == row.ActualFallible {
				row.Status = "PASS"
			}
		}
		if row.Status != "PASS" {
			membersPass = false
		}
		measurement.Members = append(measurement.Members, row)
	}
	for xna := range getters {
		if setters[xna] {
			measurement.AccessorPairs++
		}
	}
	if measurement.TargetTypes == 1 && measurement.ActualKind == "interface" &&
		measurement.Classified == pinned.Classified &&
		measurement.LocalDiagnostics == 0 &&
		measurement.TargetGoIdentities == measurement.ExpectedGoIdentities &&
		measurement.ErrorResults == len(pinned.FallibleOperations)+len(pinned.EventAccessors) &&
		measurement.EventAccessors == len(pinned.EventAccessors) && membersPass {
		measurement.Status = "PASS"
	}
	return measurement
}

// verifyBCLBaseProjection measures one XNA type's non-XNA CLR base.
//
// The three claims it enforces are the whole content of the base decision:
//
//   - a non-XNA base must be a declared relationship, so no base is dropped in
//     silence and no new BCL base can arrive unnoticed;
//   - a DEFERRED base must have no projected derived type, so a type can never
//     be shipped under a base relationship nobody has decided about;
//   - a MAPPED base must not be faked with exported Go embedding, because
//     embedding would promote members the XNA contract never declared.
//
// actual is nil when the derived type is not projected yet, which is the normal
// state for a DEFERRED base.
func verifyBCLBaseProjection(result *report, typeDiagnostics map[string]int, et *expectedType, actual *actualType) {
	if et.BaseType == "" || strings.HasPrefix(et.BaseType, "Microsoft.Xna.Framework") {
		return
	}
	identity := baseIdentityWithoutArguments(et.BaseType)
	relationship, declared := bclBaseRelationships[identity]
	if !declared {
		addDiagnostic(result, diagnostic{
			Category: "BASE_MAPPING_MISMATCH", XNA: et.XNA, Go: et.Key.String(),
			Message: fmt.Sprintf("CLR base %q is not a declared BCL base relationship", identity),
		})
		typeDiagnostics[et.XNA]++
		return
	}
	if actual == nil {
		return
	}
	if relationship.Status == "DEFERRED" {
		addDiagnostic(result, diagnostic{
			Category: "BASE_MAPPING_MISMATCH", XNA: et.XNA, Go: et.Key.String(),
			Message: fmt.Sprintf("CLR base %q is a deferred BCL base relationship, so the derived type must not be projected yet", identity),
		})
		typeDiagnostics[et.XNA]++
		return
	}
	if relationship.Status != "MAPPED" && relationship.Status != "COMPOSED" {
		return
	}
	if len(actual.ExportedEmbeddings) > 0 {
		addDiagnostic(result, diagnostic{
			Category: "BASE_MAPPING_MISMATCH", XNA: et.XNA, Go: et.Key.String(),
			Message: fmt.Sprintf("CLR base %q was faked with exported Go embedding %v", identity, actual.ExportedEmbeddings),
		})
		typeDiagnostics[et.XNA]++
	}
	if relationship.Adapter != "" && actual.Kind != "struct" {
		addDiagnostic(result, diagnostic{
			Category: "BASE_MAPPING_MISMATCH", XNA: et.XNA, Go: et.Key.String(),
			Message: fmt.Sprintf("CLR class deriving from %q must project as a Go struct reference type, found %s", identity, actual.Kind),
		})
		typeDiagnostics[et.XNA]++
	}
	if relationship.Status == "COMPOSED" {
		verifyBCLBaseComposition(result, typeDiagnostics, et, actual, identity)
	}
}

// verifyBCLBaseComposition enforces the composition model on one derived type
// whose CLR base is a supported BCL family.
//
// The claims are the whole content of the architecture:
//
//   - the concrete Go type must be a struct reference type, so a Dictionary
//     consumer can never be `type LaunchParameters map[string]string` and a
//     Collection consumer can never be a bare slice;
//   - it must hold the adapter in an UNEXPORTED field of the declared name and
//     the declared adapter family, so the base state is real and private;
//   - it must expose no exported field whose type is the raw backing store,
//     so the storage can never be reached or replaced from outside;
//   - it must not embed the adapter, exported or not, because promotion would
//     publish forwarding CNA-Go never measured.
func verifyBCLBaseComposition(result *report, typeDiagnostics map[string]int, et *expectedType, actual *actualType, identity string) {
	adapter, present := bclBaseAdapters[identity]
	if !present {
		addDiagnostic(result, diagnostic{
			Category: "BASE_MAPPING_MISMATCH", XNA: et.XNA, Go: et.Key.String(),
			Message: fmt.Sprintf("CLR base %q is COMPOSED but declares no BCL base adapter", identity),
		})
		typeDiagnostics[et.XNA]++
		return
	}
	wanted := adapterFieldType(adapter, et.BaseType)
	found := false
	for _, field := range actual.Fields {
		if field.Embedded {
			addDiagnostic(result, diagnostic{
				Category: "BASE_MAPPING_MISMATCH", XNA: et.XNA, Go: et.Key.String(),
				Message: fmt.Sprintf("BCL base adapter for %q must be a named private field, found embedded %s", identity, field.Type),
			})
			typeDiagnostics[et.XNA]++
			continue
		}
		if field.Name != adapter.AdapterField {
			continue
		}
		found = true
		if field.Exported {
			addDiagnostic(result, diagnostic{
				Category: "BASE_MAPPING_MISMATCH", XNA: et.XNA, Go: et.Key.String(),
				Message: fmt.Sprintf("BCL base adapter field %q for %q must be unexported", field.Name, identity),
			})
			typeDiagnostics[et.XNA]++
		}
		if field.Type != wanted {
			addDiagnostic(result, diagnostic{
				Category: "BASE_MAPPING_MISMATCH", XNA: et.XNA, Go: et.Key.String(),
				Message: fmt.Sprintf("BCL base adapter field %q for %q must have type %s, found %s", field.Name, identity, wanted, field.Type),
			})
			typeDiagnostics[et.XNA]++
		}
	}
	if !found {
		addDiagnostic(result, diagnostic{
			Category: "BASE_MAPPING_MISMATCH", XNA: et.XNA, Go: et.Key.String(),
			Message: fmt.Sprintf("CLR base %q must be held in the private adapter field %q of type %s", identity, adapter.AdapterField, wanted),
		})
		typeDiagnostics[et.XNA]++
	}
	for _, field := range actual.Fields {
		if !field.Exported {
			continue
		}
		if strings.HasPrefix(field.Type, "[]") || strings.HasPrefix(field.Type, "map[") {
			addDiagnostic(result, diagnostic{
				Category: "BASE_MAPPING_MISMATCH", XNA: et.XNA, Go: et.Key.String(),
				Message: fmt.Sprintf("BCL base %q consumer exposes raw backing storage as exported field %s %s", identity, field.Name, field.Type),
			})
			typeDiagnostics[et.XNA]++
		}
	}
}

// adapterFieldType is the exact Go spelling the private adapter field must
// have on one derived type: the adapter family with the CLR base's own generic
// arguments substituted for its parameters.
func adapterFieldType(adapter bclBaseAdapter, baseType string) string {
	arguments := bclBaseArguments(baseType)
	open := strings.Index(adapter.GoAdapter, "[")
	if open < 0 || len(arguments) != adapter.GenericArity {
		return adapter.GoAdapter
	}
	mapped := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		mapped = append(mapped, goAdapterArgument(argument))
	}
	return adapter.GoAdapter[:open] + "[" + strings.Join(mapped, ", ") + "]"
}

// measureBCLBaseRelationships summarises every declared non-XNA base across the
// profile so the report carries the whole relationship table, not only the
// entries that happen to have a projected derived type.
func measureBCLBaseRelationships(expected *expectedSurface, actual *actualSurface) []bclBaseProjection {
	derived := make(map[string]*bclBaseProjection, len(bclBaseRelationships))
	for identity, relationship := range bclBaseRelationships {
		row := &bclBaseProjection{
			CLRBase: identity, Adapter: relationship.Adapter, Status: relationship.Status,
			AddsProjectedSurface: relationship.Status == "COMPOSED",
			Rationale:            relationship.Rationale, Verdict: "PASS",
		}
		for _, blocker := range relationship.Blockers {
			row.Blockers = append(row.Blockers, bclBaseBlockerMeasurement{
				Kind: blocker.Kind, CLRMember: blocker.CLRMember, Needs: blocker.Needs, Detail: blocker.Detail,
			})
		}
		derived[identity] = row
	}
	for _, et := range sortedExpectedTypes(expected) {
		if et.BaseType == "" || strings.HasPrefix(et.BaseType, "Microsoft.Xna.Framework") {
			continue
		}
		row := derived[baseIdentityWithoutArguments(et.BaseType)]
		if row == nil {
			continue
		}
		row.DerivedTypes++
		at := actual.Types[et.Key]
		if at == nil {
			continue
		}
		row.ProjectedTypes++
		if len(at.ExportedEmbeddings) > 0 {
			row.ExportedEmbeddings++
		}
	}
	measurements := make([]bclBaseProjection, 0, len(derived))
	for _, row := range derived {
		if row.ExportedEmbeddings != 0 || (row.Status == "DEFERRED" && row.ProjectedTypes != 0) {
			row.Verdict = "FAIL"
		}
		// A DEFERRED base must name why. Deferring with no recorded blocker
		// would be an unmeasured structural category wearing a status word.
		if row.Status == "DEFERRED" && len(row.Blockers) == 0 {
			row.Verdict = "FAIL"
		}
		measurements = append(measurements, *row)
	}
	sort.Slice(measurements, func(i, j int) bool { return measurements[i].CLRBase < measurements[j].CLRBase })
	return measurements
}

// verifyEventGroupLeaks rejects any exported member that occupies an event's
// name space without being one of the event's two projected accessors.
//
// The settled projection is exactly two accessors per CLR event, so anything
// else on the same receiver that spells the event is a leak: the CLR accessor
// names add_X, remove_X and raise_X, the CLR raise helper OnX, and the bare
// event name X, which is how an event would look if it had been projected as a
// property, a closure field, or a channel instead.
//
// The scan runs for every projected event, not only for events whose accessors
// are missing, so a leak alongside a correct projection is still an event
// defect rather than an anonymous unexpected symbol.
func verifyEventGroupLeaks(result *report, expected *expectedSurface, actual *actualSurface, et *expectedType) {
	for _, memberKey := range et.Members {
		member := expected.Members[memberKey]
		if member == nil || member.SourceKind != "event" || !strings.HasPrefix(memberKey.Name, "Add") {
			continue
		}
		// The XNA event name, recovered from the projected accessor name so the
		// static declaring-type prefix is handled the same way.
		// A static event projects as a package function whose name carries the
		// declaring type, so the receiver is empty and the type prefix is part
		// of the accessor name.
		base := strings.TrimSuffix(memberKey.Name, "Handler")
		if memberKey.Receiver == "" {
			base = strings.TrimPrefix(base, et.GoName)
		}
		base = strings.TrimPrefix(base, "Add")
		if base == "" {
			continue
		}
		forbidden := map[string]string{
			base:             "event projected as a bare member instead of an add/remove accessor pair",
			"add_" + base:    "CLR add_ accessor leaked as an XNA identity",
			"remove_" + base: "CLR remove_ accessor leaked as an XNA identity",
			"raise_" + base:  "CLR raise_ accessor leaked as an XNA identity",
			"On" + base:      "CLR protected raise helper leaked as an XNA identity",
		}
		for key := range actual.Members {
			if key.Package != memberKey.Package || key.Receiver != memberKey.Receiver {
				continue
			}
			if _, mapped := expected.Members[key]; mapped {
				continue
			}
			reason, forbiddenName := forbidden[key.Name]
			if !forbiddenName {
				// Anything else sharing the accessor prefix is a stray member of
				// the same event group.
				if !strings.HasPrefix(key.Name, strings.TrimSuffix(memberKey.Name, "Handler")) {
					continue
				}
				reason = "event group contains a non-matching add/remove projection"
			}
			addDiagnostic(result, diagnostic{
				Category: "EVENT_MAPPING_MISMATCH", XNA: member.XNA, Go: key.String(), Message: reason,
			})
		}
	}
}

// verifyBCLInterfaceRelationships measures the non-XNA interfaces one XNA type
// declares.
//
// The claim is that such an interface contributes no projected Go surface, so
// the check is not "is it mapped" but "did anything appear because of it". It
// enforces three things:
//
//   - every non-XNA direct interface is a declared relationship, so a new BCL
//     interface cannot arrive unnoticed;
//   - none of the invented disposal shapes exists on the projected type -- no
//     Close alias, no Disposable interface, no io.Closer adaptation, no
//     finalizer surface, no ownership wrapper;
//   - no Dispose member exists on a type whose XNA contract declares none,
//     which is the exact defect System.IDisposable would cause if it were
//     treated as adding surface. GraphicsDeviceManager is the profile's real
//     instance: it implements IDisposable explicitly, so its Dispose() is not
//     public surface.
//
// A Dispose the XNA contract does declare is an ordinary member and is measured
// by the ordinary member comparison, with its own fallibility, exactly like any
// other method.
func verifyBCLInterfaceRelationships(result *report, expected *expectedSurface, actual *actualSurface, typeDiagnostics map[string]int, et *expectedType) {
	declaresDispose := false
	for _, memberKey := range et.Members {
		if strings.HasPrefix(memberKey.Name, "Dispose") {
			declaresDispose = true
			break
		}
	}
	for _, raw := range et.Interfaces {
		identity := baseIdentityWithoutArguments(raw)
		if expected.typeForXNA(identity) != nil {
			// A public XNA interface is measured by the ordinary interface
			// machinery, not here.
			continue
		}
		if strings.HasPrefix(identity, "Microsoft.Xna.Framework") {
			if _, declared := internalXNAInterfaces[identity]; !declared {
				addDiagnostic(result, diagnostic{
					Category: "INTERFACE_MAPPING_MISMATCH", XNA: et.XNA, Go: et.Key.String(),
					Message: fmt.Sprintf("XNA interface %q is neither a public contract type nor a declared internal interface", identity),
				})
				typeDiagnostics[et.XNA]++
			}
			continue
		}
		if _, declared := bclInterfaceRelationships[identity]; !declared {
			addDiagnostic(result, diagnostic{
				Category: "INTERFACE_MAPPING_MISMATCH", XNA: et.XNA, Go: et.Key.String(),
				Message: fmt.Sprintf("CLR interface %q is not a declared BCL interface relationship", identity),
			})
			typeDiagnostics[et.XNA]++
		}
	}
	target := actual.Types[et.Key]
	if target == nil {
		return
	}
	for _, embedded := range target.ExportedEmbeddings {
		if reason, invented := inventedDisposalNames[strings.TrimPrefix(embedded, "framework.")]; invented {
			addDiagnostic(result, diagnostic{
				Category: "INTERFACE_MAPPING_MISMATCH", XNA: et.XNA, Go: et.Key.String(),
				Message: reason + " was embedded in the projected type",
			})
			typeDiagnostics[et.XNA]++
		}
	}
	for key := range actual.Members {
		if key.Package != et.Key.Package || key.Receiver != et.GoName {
			continue
		}
		if _, mapped := expected.Members[key]; mapped {
			continue
		}
		if reason, invented := inventedDisposalNames[key.Name]; invented {
			addDiagnostic(result, diagnostic{
				Category: "INTERFACE_MAPPING_MISMATCH", XNA: et.XNA, Go: key.String(), Message: reason,
			})
			typeDiagnostics[et.XNA]++
			continue
		}
		if strings.HasPrefix(key.Name, "Dispose") && !declaresDispose {
			addDiagnostic(result, diagnostic{
				Category: "INTERFACE_MAPPING_MISMATCH", XNA: et.XNA, Go: key.String(),
				Message: "Dispose was synthesized on a type whose XNA contract declares none; System.IDisposable adds no projected surface",
			})
			typeDiagnostics[et.XNA]++
		}
	}
}

// measureBCLInterfaceRelationships summarises the declared no-surface
// interfaces across the whole profile, so the report carries the claim and its
// arithmetic rather than only the failures.
func measureBCLInterfaceRelationships(expected *expectedSurface, actual *actualSurface) []bclInterfaceProjection {
	rows := make(map[string]*bclInterfaceProjection, len(bclInterfaceRelationships)+len(internalXNAInterfaces))
	for _, table := range []map[string]bclInterfaceRelationship{bclInterfaceRelationships, internalXNAInterfaces} {
		for identity, relationship := range table {
			rows[identity] = &bclInterfaceProjection{
				CLRInterface: identity, Status: relationship.Status,
				CLRMembers: len(relationship.Members), ProjectedMembers: 0,
				Rationale: relationship.Rationale, Verdict: "PASS",
			}
		}
	}
	for _, et := range sortedExpectedTypes(expected) {
		for _, raw := range et.Interfaces {
			identity := baseIdentityWithoutArguments(raw)
			row := rows[identity]
			if row == nil {
				continue
			}
			row.DeclaringTypes++
			if actual.Types[et.Key] != nil {
				row.ProjectedTypes++
			}
		}
	}
	measurements := make([]bclInterfaceProjection, 0, len(rows))
	for _, row := range rows {
		if row.ProjectedMembers != 0 {
			row.Verdict = "FAIL"
		}
		measurements = append(measurements, *row)
	}
	sort.Slice(measurements, func(i, j int) bool { return measurements[i].CLRInterface < measurements[j].CLRInterface })
	return measurements
}

// measureBCLBaseAdapters produces the whole BCL base-class adapter registry
// measurement: every supported family, its private Go adapter, its exact
// inherited public member inventory, its deliberate exclusions, and every
// concrete XNA consumer with that consumer's two provenance classes.
//
// The registry is measured whether or not a consumer is projected yet, so a
// family that is declared but unused is visible rather than absent, and a
// consumer that regresses is a FAIL rather than a silently missing row.
func measureBCLBaseAdapters(expected *expectedSurface, actual *actualSurface) []bclBaseAdapterMeasurement {
	measurements := make([]bclBaseAdapterMeasurement, 0, len(bclBaseAdapters))
	for identity, adapter := range bclBaseAdapters {
		measurement := bclBaseAdapterMeasurement{
			CLRBase: identity, GoAdapter: adapter.GoAdapter, AdapterField: adapter.AdapterField,
			BehaviorLevel: adapter.BehaviorLevel, Authority: adapter.Authority,
			AuthoritySHA256: adapter.AuthoritySHA256, InheritedCLRMembers: len(adapter.Members),
			ExcludedMembers: len(adapter.Excluded), Rationale: adapter.Rationale, Verdict: "PASS",
		}
		for _, excluded := range adapter.Excluded {
			measurement.Exclusions = append(measurement.Exclusions, bclInheritedExclusion{CLRMember: excluded.CLRMember, Reason: excluded.Reason})
		}
		for _, et := range sortedExpectedTypes(expected) {
			if baseIdentityWithoutArguments(et.BaseType) != identity {
				continue
			}
			at := actual.Types[et.Key]
			consumer := bclBaseAdapterConsumer{
				XNA: et.XNA, Go: et.Key.String(), BaseArguments: bclBaseArguments(et.BaseType),
				Projected: at != nil, AdapterFieldType: adapterFieldType(adapter, et.BaseType),
				DeclaredMembers: et.SourceMembers, InheritedProjections: et.BCLInheritedProjections,
				DeclaredProjections: len(et.Members) - et.BCLInheritedProjections, Verdict: "PASS",
			}
			if at != nil {
				consumer.ExportedEmbeddings = len(at.ExportedEmbeddings)
				for _, field := range at.Fields {
					if field.Name == adapter.AdapterField && !field.Exported && !field.Embedded && field.Type == consumer.AdapterFieldType {
						consumer.AdapterFieldPresent = true
					}
				}
				if !consumer.AdapterFieldPresent || consumer.ExportedEmbeddings != 0 {
					consumer.Verdict = "FAIL"
					measurement.Verdict = "FAIL"
				}
			}
			measurement.Consumers = append(measurement.Consumers, consumer)
			measurement.Inventory = append(measurement.Inventory, inheritedMemberInventory(expected, actual, adapter, et)...)
			if measurement.InheritedProjections == 0 {
				measurement.InheritedProjections = et.BCLInheritedProjections
			}
		}
		if adapter.BehaviorLevel != "SUPPORTED" && adapter.BehaviorLevel != "PARTIAL" {
			measurement.Verdict = "FAIL"
		}
		measurements = append(measurements, measurement)
	}
	sort.Slice(measurements, func(i, j int) bool { return measurements[i].CLRBase < measurements[j].CLRBase })
	return measurements
}

// inheritedMemberInventory is the per-member attribution table for one
// consumer: for every public CLR member of the base, the exact projected Go
// member and whether the Go surface actually carries it.
func inheritedMemberInventory(expected *expectedSurface, actual *actualSurface, adapter bclBaseAdapter, et *expectedType) []bclInheritedMemberMeasurement {
	rationale := make(map[string]string, len(adapter.Members))
	for _, entry := range adapter.Members {
		rationale[entry.Member.Name] = entry.Rationale
	}
	var rows []bclInheritedMemberMeasurement
	for _, key := range et.Members {
		member := expected.Members[key]
		if member == nil || member.BCLBase == "" {
			continue
		}
		row := bclInheritedMemberMeasurement{
			CLRBase: member.BCLBase, CLRMember: member.BCLMember, CLRKind: member.SourceKind,
			Consumer: et.XNA, GoMember: member.Key.String(), GoResults: strings.Join(member.Results, ","),
			Accessor: member.Accessor, Present: actual.Members[member.Key] != nil,
			Rationale: rationale[member.BCLMember],
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CLRMember != rows[j].CLRMember {
			return rows[i].CLRMember < rows[j].CLRMember
		}
		return rows[i].Accessor < rows[j].Accessor
	})
	return rows
}

// composedBaseAdapter returns the supported BCL base adapter one XNA type
// inherits, when it has one.
func composedBaseAdapter(owner *expectedType) (bclBaseAdapter, bool) {
	if owner.BaseType == "" {
		return bclBaseAdapter{}, false
	}
	identity := baseIdentityWithoutArguments(owner.BaseType)
	if relationship, declared := bclBaseRelationships[identity]; !declared || relationship.Status != "COMPOSED" {
		return bclBaseAdapter{}, false
	}
	adapter, present := bclBaseAdapters[identity]
	return adapter, present
}

// verifyExcludedBaseMembersAbsent is the negative half of the inherited
// projection claim: a base member the registry records as deliberately
// unprojected must not appear on the Go type.
//
// It is what stops the composition projection from quietly growing surface the
// CLR does not have -- an IsReadOnly or a SyncRoot promoted from an explicit
// implementation, or an Items accessor that would hand out the backing store.
//
// An explicitly implemented member is recorded under its CLR interface, as
// `ICollection<T>.IsReadOnly`. Go has no explicit interface implementation, so
// the only way such a member could reach the Go surface is under its bare
// name, and that bare name is what is checked. When the bare name is also a
// projected inherited member -- `IList.Add` against the generic Add -- the
// projected one owns it and there is nothing to reject. Constructors are
// skipped outright: the CLR does not inherit them and no Go method spells one.
func verifyExcludedBaseMembersAbsent(result *report, expected *expectedSurface, actual *actualSurface, owner *expectedType, adapter bclBaseAdapter) {
	projected := make(map[string]bool, len(adapter.Members))
	for _, entry := range adapter.Members {
		projected[entry.Member.Name] = true
	}
	// A name the derived XNA type declares ITSELF is out of scope. The claim
	// is about provenance, not about the spelling: GameComponentCollection
	// declares its own InsertItem, RemoveItem, SetItem and ClearItems
	// overrides in the pinned metadata, and CNA-Go projects a declared
	// protected member like any other, so those Go members exist because the
	// XNA assembly declares them and not because the base leaked them.
	declared := make(map[string]bool)
	for _, key := range owner.Members {
		member := expected.Members[key]
		if member == nil || member.BCLBase != "" {
			continue
		}
		if separator := strings.Index(member.XNA, "::"); separator >= 0 {
			name := member.XNA[separator+2:]
			if open := strings.Index(name, "("); open >= 0 {
				name = name[:open]
			}
			declared[name] = true
		}
	}
	for _, excluded := range adapter.Excluded {
		name := excluded.CLRMember
		if strings.Contains(name, "(") {
			continue
		}
		if dot := strings.LastIndex(name, "."); dot >= 0 {
			name = name[dot+1:]
		}
		if projected[name] || declared[name] {
			continue
		}
		key := symbolKey{Package: owner.Key.Package, Receiver: owner.GoName, Name: name}
		if _, present := actual.Members[key]; present {
			addDiagnostic(result, diagnostic{
				Category: "BASE_MAPPING_MISMATCH", XNA: owner.XNA, Go: key.String(),
				Message: fmt.Sprintf("BCL base member %q is excluded from the inherited projection (%s) but the Go type declares it", name, excluded.Reason),
			})
		}
	}
}

// measureBCLSignatureAdapters pins the exported surface of every BCL signature
// adapter to its exact public CLR member inventory, and records which
// projected XNA members carry the type.
//
// Three claims are enforced:
//
//   - every inventoried CLR member has its projected Go member present;
//   - the adapter declares no OTHER exported member, so an adapter type is not
//     a hole in the unexpected-member scan;
//   - no member the registry records as excluded appears under its bare name,
//     so a private explicit implementation is never promoted to public Go
//     surface -- which for a read-only view is what keeps it read-only.
func measureBCLSignatureAdapters(result *report, expected *expectedSurface, actual *actualSurface) []bclSignatureAdapterMeasurement {
	frameworkPackage := modulePath + "/Microsoft/Xna/Framework"
	// The adapters are a claim about the real framework package, so the
	// measurement runs only when that package was actually extracted. An
	// isolated per-type fixture models one XNA type and no package at all;
	// demanding the whole adapter surface of it would make every such fixture
	// dirty and would say nothing about the binding.
	if actual.PackageDirs[frameworkPackage] == "" {
		return nil
	}
	measurements := make([]bclSignatureAdapterMeasurement, 0, len(bclSignatureAdapters))
	for identity, adapter := range bclSignatureAdapters {
		goName := bclSignatureAdapterGoName(adapter)
		measurement := bclSignatureAdapterMeasurement{
			CLRType: identity, GoAdapter: adapter.GoAdapter, BehaviorLevel: adapter.BehaviorLevel,
			Authority: adapter.Authority, AuthoritySHA256: adapter.AuthoritySHA256,
			CLRMembers: len(adapter.Members), ExcludedMembers: len(adapter.Excluded),
			Rationale: adapter.Rationale, Verdict: "PASS",
		}
		for _, excluded := range adapter.Excluded {
			measurement.Exclusions = append(measurement.Exclusions, bclInheritedExclusion{CLRMember: excluded.CLRMember, Reason: excluded.Reason})
		}
		if _, present := actual.Types[symbolKey{Package: frameworkPackage, Name: goName}]; !present {
			addDiagnostic(result, diagnostic{
				Category: "LANGUAGE_MAPPING_MISMATCH", XNA: identity, Go: frameworkPackage + ":" + goName,
				Message: "declared BCL signature adapter is absent from the framework package",
			})
			measurement.Verdict = "FAIL"
		}

		// Every inventoried CLR member maps to one exported Go member. A
		// property with both accessors would map to two; none of these has a
		// public setter, which is the read-only claim itself.
		wanted := make(map[string]bool)
		for _, entry := range adapter.Members {
			goMember := entry.Member.Name
			if entry.Member.Kind == "property" && entry.Member.Set {
				addDiagnostic(result, diagnostic{
					Category: "LANGUAGE_MAPPING_MISMATCH", XNA: identity, Go: goName,
					Message: fmt.Sprintf("signature adapter inventory declares a public setter for %q, which the read-only projection does not model", entry.Member.Name),
				})
				measurement.Verdict = "FAIL"
			}
			wanted[goMember] = true
			key := symbolKey{Package: frameworkPackage, Receiver: goName, Name: goMember}
			_, present := actual.Members[key]
			accessor := ""
			if entry.Member.Kind == "property" {
				accessor = "get"
			}
			row := bclInheritedMemberMeasurement{
				CLRType: identity, CLRBase: identity, CLRMember: entry.Member.Name, CLRKind: entry.Member.Kind,
				Consumer: goName, GoMember: key.String(), Accessor: accessor,
				Present: present, Rationale: entry.Rationale,
			}
			if member := actual.Members[key]; member != nil {
				row.GoResults = strings.Join(member.Results, ",")
			}
			if !present {
				addDiagnostic(result, diagnostic{
					Category: "LANGUAGE_MAPPING_MISMATCH", XNA: identity + "::" + entry.Member.Name, Go: key.String(),
					Message: "BCL signature adapter is missing a public member of the pinned CLR type",
				})
				measurement.Verdict = "FAIL"
			}
			measurement.Inventory = append(measurement.Inventory, row)
			measurement.GoMembers++
		}

		// Nothing else may be exported on the adapter.
		for key := range actual.Members {
			if key.Package != frameworkPackage || key.Receiver != goName || wanted[key.Name] {
				continue
			}
			addDiagnostic(result, diagnostic{
				Category: "LANGUAGE_MAPPING_MISMATCH", XNA: identity, Go: key.String(),
				Message: "BCL signature adapter exports a member the pinned CLR type does not declare publicly",
			})
			measurement.Verdict = "FAIL"
		}
		// A member the registry excludes must not appear under its bare name.
		for _, excluded := range adapter.Excluded {
			name := excluded.CLRMember
			if strings.Contains(name, "(") {
				continue
			}
			if dot := strings.LastIndex(name, "."); dot >= 0 {
				name = name[dot+1:]
			}
			if wanted[name] {
				continue
			}
			if _, present := actual.Members[symbolKey{Package: frameworkPackage, Receiver: goName, Name: name}]; present {
				addDiagnostic(result, diagnostic{
					Category: "LANGUAGE_MAPPING_MISMATCH", XNA: identity, Go: goName + "." + name,
					Message: fmt.Sprintf("BCL signature adapter promotes %q, which is excluded (%s)", excluded.CLRMember, excluded.Reason),
				})
				measurement.Verdict = "FAIL"
			}
		}

		// Which projected XNA members carry the type.
		needle := "*" + goName + "["
		qualified := "*framework." + goName + "["
		for _, key := range sortedMemberKeys(expected) {
			member := expected.Members[key]
			for _, result := range append(append([]string(nil), member.Results...), member.Parameters...) {
				if strings.HasPrefix(result, needle) || strings.HasPrefix(result, qualified) {
					measurement.SignaturePositions++
					measurement.Carriers = append(measurement.Carriers, member.XNA)
					break
				}
			}
		}
		sort.Strings(measurement.Carriers)
		measurements = append(measurements, measurement)
	}
	sort.Slice(measurements, func(i, j int) bool { return measurements[i].CLRType < measurements[j].CLRType })
	return measurements
}

// sortedActualMemberKeys and sortedActualTypeKeys return the actual surface's
// keys in a stable order, so a diagnostic driven by scanning the whole package
// is reported deterministically.
func sortedActualMemberKeys(members map[symbolKey]*actualMember) []symbolKey {
	keys := make([]symbolKey, 0, len(members))
	for key := range members {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })
	return keys
}

func sortedActualTypeKeys(values map[symbolKey]*actualType) []symbolKey {
	keys := make([]symbolKey, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })
	return keys
}

// sortedMemberKeys returns every expected member key in a stable order.
func sortedMemberKeys(expected *expectedSurface) []symbolKey {
	keys := make([]symbolKey, 0, len(expected.Members))
	for key := range expected.Members {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })
	return keys
}

// ---------------------------------------------------------------------------
// Foundation 31 — measuring the Game base-call language adapters.
// ---------------------------------------------------------------------------

// gameBaseCallDeferralClasses are the only classes a deferred reference step
// may carry. An unclassified deferral is a verifier failure, exactly as an
// unrecorded blocker on a deferred BCL base is.
var gameBaseCallDeferralClasses = map[string]bool{
	// The step needs a .NET or CNA subsystem CNA-Go does not have.
	"SUBSYSTEM": true,
	// A cross-cutting decision blocks the step; no single member carries it.
	"ARCHITECTURE": true,
	// The step has no effect any projected surface can observe.
	"UNOBSERVABLE": true,
}

// measureGameBaseCallAdapters proves that CNA-Go's base-call family is exactly
// the set of protected virtuals GameCallbacks projects -- no more and no fewer
// -- and that each adapter's Go signature, fallibility and deferrals are the
// declared ones.
//
// The four claims it exists to defend:
//
//  1. Every adapter corresponds to a REAL protected XNA virtual. The CLR
//     member is looked up in the expected surface built from the pinned
//     contract, and its declared accessibility must be `protected`.
//  2. No arbitrary extra base helper exists. Every exported package-level
//     framework function whose name begins with GameBase must be a declared
//     adapter, so a helper nobody measured cannot be added.
//  3. No supported virtual is missing its helper. Every GameCallbacks member
//     must have exactly one adapter, so an override cannot be left unable to
//     reach its base.
//  4. Nothing here is XNA identity. The measurement adds no expected member
//     and no reference member; the adapters live in adapterFunctions, which
//     is populated FROM this registry rather than by hand.
func measureGameBaseCallAdapters(result *report, expected *expectedSurface, actual *actualSurface) []gameBaseCallMeasurement {
	frameworkPackage := modulePath + "/Microsoft/Xna/Framework"
	// Like the BCL adapter measurements, this is a claim about the real
	// framework package. An isolated per-type fixture models one XNA type and
	// no package at all, so demanding the family of it would say nothing.
	//
	// The family is equally a claim about Game: every adapter must correspond
	// to a protected virtual GameCallbacks projects, and an expected surface
	// that does not carry Game cannot answer that. Both conditions therefore
	// gate the measurement, so a fixture is measured on what it actually
	// models rather than failed for what it does not.
	if actual.PackageDirs[frameworkPackage] == "" || expected.typeForXNA("Microsoft.Xna.Framework.Game") == nil {
		return nil
	}

	// The callback members the mapper redirects, taken from the expected
	// surface rather than from a second hand-written list, so the two cannot
	// drift.
	callbackMembers := make(map[string]*expectedMember)
	for _, member := range expected.Members {
		if member.PackagePath == frameworkPackage && member.Receiver == "GameCallbacks" {
			callbackMembers[member.GoName] = member
		}
	}

	names := make([]string, 0, len(gameBaseCallAdapters))
	for name := range gameBaseCallAdapters {
		names = append(names, name)
	}
	sort.Strings(names)

	declaredFunctions := make(map[string]bool, len(gameBaseCallAdapters))
	measurements := make([]gameBaseCallMeasurement, 0, len(gameBaseCallAdapters))
	for _, name := range names {
		adapter := gameBaseCallAdapters[name]
		declaredFunctions[adapter.GoFunction] = true
		measurement := gameBaseCallMeasurement{
			CLRMember: adapter.CLRMember, CallbackMember: name,
			GoFunction:    frameworkPackage + ":" + adapter.GoFunction,
			Parameters:    append([]string(nil), adapter.Parameters...),
			Results:       append([]string(nil), adapter.Results...),
			ReferenceBody: append([]string(nil), adapter.ReferenceBody...),
			Verdict:       "PASS",
		}
		fail := func(message string) {
			addDiagnostic(result, diagnostic{
				Category: "LANGUAGE_MAPPING_MISMATCH", XNA: adapter.CLRMember,
				Go: measurement.GoFunction, Message: message,
			})
			measurement.Verdict = "FAIL"
		}

		// (1) the adapter names a real protected virtual GameCallbacks projects.
		callback, projected := callbackMembers[name]
		if !projected {
			fail("declared base-call adapter does not correspond to a GameCallbacks member")
		} else {
			measurement.CLRAccess = callback.SourceAccess
			if callback.SourceAccess != "protected" {
				fail(fmt.Sprintf("base-call adapter names a %q member; only a protected virtual has a base a CLR override can call", callback.SourceAccess))
			}
			if wanted := callback.XNA; !strings.HasPrefix(wanted, adapter.CLRMember+"(") && wanted != adapter.CLRMember {
				fail(fmt.Sprintf("base-call adapter declares CLR member %q but the projected callback member is %q", adapter.CLRMember, wanted))
			}
		}

		// The Go function must exist with the exact declared signature.
		key := symbolKey{Package: frameworkPackage, Name: adapter.GoFunction}
		member, present := actual.Members[key]
		switch {
		case !present:
			fail("declared base-call adapter is absent from the framework package")
		default:
			if member.Kind != "func" {
				fail(fmt.Sprintf("base-call adapter is a Go %s; it must be a package-level func so Game's projected member surface gains no name Microsoft never declared", member.Kind))
			}
			if !equalStrings(adapter.Parameters, member.Parameters) {
				fail(fmt.Sprintf("base-call adapter expects parameters %v, found %v", adapter.Parameters, member.Parameters))
			}
			if !equalStrings(adapter.Results, member.Results) {
				fail(fmt.Sprintf("base-call adapter expects results %v, found %v", adapter.Results, member.Results))
			}
			if len(adapter.Parameters) == 0 || adapter.Parameters[0] != "*Game" {
				fail("a base-call adapter's first parameter must be *Game, which is the `this` a CLR base call passes implicitly")
			}
		}

		// Fallibility must be justified rather than assumed, in both
		// directions: an error result with no recorded reason is a synthetic
		// failure mode, and a recorded reason with no error result is a claim
		// the signature does not make.
		endsWithError := len(adapter.Results) > 0 && adapter.Results[len(adapter.Results)-1] == "error"
		if endsWithError != (len(adapter.Fallibility) > 0) {
			fail(fmt.Sprintf("base-call adapter results %v and %d recorded fallibility reasons disagree",
				adapter.Results, len(adapter.Fallibility)))
		}
		for _, reason := range adapter.Fallibility {
			measurement.Fallibility = append(measurement.Fallibility,
				gameBaseCallFallibilityRow{Kind: reason.Kind, Reason: reason.Reason})
			if reason.Kind != "GUARD" && reason.Kind != "REFERENCE" {
				fail(fmt.Sprintf("base-call adapter fallibility kind %q is not GUARD or REFERENCE", reason.Kind))
			}
			if strings.TrimSpace(reason.Reason) == "" {
				fail(fmt.Sprintf("base-call adapter fallibility %q records no reason", reason.Kind))
			}
		}

		if len(adapter.ReferenceBody) == 0 {
			fail("base-call adapter records no reference body, so there is nothing to check the projection against")
		}

		// Every unreproduced reference step must be recorded, classified, and
		// unobservable. A deferral that IS observable from the managed surface
		// is a stop condition rather than a deferral, so it is rejected here.
		for _, deferral := range adapter.Deferred {
			measurement.Deferred = append(measurement.Deferred, gameBaseCallDeferralRow{
				Step: deferral.Step, Class: deferral.Class,
				Reason: deferral.Reason, Observable: deferral.Observable,
			})
			result.Summary["GAME_BASE_CALL_DEFERRED_STEPS"]++
			if !gameBaseCallDeferralClasses[deferral.Class] {
				fail(fmt.Sprintf("deferred reference step %q carries unclassified class %q", deferral.Step, deferral.Class))
			}
			if strings.TrimSpace(deferral.Reason) == "" {
				fail(fmt.Sprintf("deferred reference step %q records no reason", deferral.Step))
			}
			if deferral.Observable {
				fail(fmt.Sprintf("deferred reference step %q is observable from the managed component surface, which makes it a stop condition rather than a deferral", deferral.Step))
			}
		}
		measurements = append(measurements, measurement)
	}

	// (3) no supported virtual may be left without a helper.
	for goName, member := range callbackMembers {
		if _, declared := gameBaseCallAdapters[goName]; !declared {
			addDiagnostic(result, diagnostic{
				Category: "LANGUAGE_MAPPING_MISMATCH", XNA: member.XNA,
				Go:      frameworkPackage + ":GameCallbacks." + goName,
				Message: "GameCallbacks projects a protected virtual with no base-call adapter, so an override cannot reach its base",
			})
		}
	}

	// (2) and no arbitrary extra helper may exist.
	for key := range actual.Members {
		if key.Package != frameworkPackage || key.Receiver != "" || !strings.HasPrefix(key.Name, "GameBase") {
			continue
		}
		if declaredFunctions[key.Name] {
			continue
		}
		addDiagnostic(result, diagnostic{
			Category: "LANGUAGE_MAPPING_MISMATCH", Go: key.String(),
			Message: "exported base-call helper is not a declared Game base-call adapter",
		})
	}
	return measurements
}

// ---------------------------------------------------------------------------
// Foundation 32 — declared CLR interface conformance, checked by the compiler.
// ---------------------------------------------------------------------------

// measureDeclaredInterfaceConformance proves that a COMPLETE projected class
// actually satisfies the Go projection of every XNA interface its CLR metadata
// declares.
//
// Until this milestone the only conformance the verifier checked by compiler
// evidence was the PackedVector family's, through interface witnesses. Every
// other declaration was checked structurally -- member for member -- which is
// necessary but not sufficient: a receiver-kind mistake, a type-parameter
// mistake, or a member that matches the interface's shape without satisfying it
// would pass the member comparison and still leave the class unusable where the
// contract says it belongs.
//
// The rule is:
//
//	a projected class that CLR metadata says implements a projected XNA
//	interface must satisfy that interface's Go projection, on the POINTER
//	method set, and the compiler must say so
//
// # Why only complete types
//
// A partial type is missing members by definition, so requiring it to satisfy a
// contract would report the same gap twice: once as MISSING_MEMBER and again as
// a conformance failure. The claim is meaningful exactly when the type's whole
// surface is present, which is what completeness means. GraphicsDeviceManager
// is the live example: it declares IGraphicsDeviceService and
// IGraphicsDeviceManager and satisfies neither, because 20 of its members are
// still missing, and that is already fully reported.
//
// # Why the pointer method set
//
// CNA-Go projects every CLR class as a Go pointer facade, so the class's
// identity is *T and it is *T that must implement. A value type is a different
// projection with its own settled rules and is not covered here.
func measureDeclaredInterfaceConformance(result *report, expected *expectedSurface, actual *actualSurface, complete []string) []declaredInterfaceConformance {
	completeSet := make(map[string]bool, len(complete))
	for _, identity := range complete {
		completeSet[identity] = true
	}
	measurements := make([]declaredInterfaceConformance, 0)
	for _, owner := range sortedExpectedTypes(expected) {
		if owner.Kind != "class" || !completeSet[owner.XNA] || len(owner.MappedInterfaces) == 0 {
			continue
		}
		// The PackedVector family is measured by its own generic-aware
		// conformance check and by interface witnesses, so it is not measured
		// twice under a rule that cannot express its type argument.
		if strings.HasPrefix(owner.XNA, packedVectorNamespace) {
			continue
		}
		pkg := actual.Packages[owner.PackagePath]
		ownerObject := lookupNamed(pkg, owner.GoName)
		for _, mapped := range owner.MappedInterfaces {
			identity, arguments := splitConstructedType(mapped.XNA)
			mappedType := expected.typeForXNA(identity)
			// Only an interface CNA-Go actually projects as a Go interface can
			// be satisfied. A CLR interface with no Go projection is carried by
			// the settled BCL-interface relationship instead.
			if mappedType == nil || mappedType.Kind != "interface" {
				continue
			}
			if _, missing := contains(result.MissingTypes, identity); missing {
				continue
			}
			// A generic CLR interface would need its type arguments
			// instantiated, which only the packed family has and which is
			// measured there. Nothing else in the profile declares one.
			if len(arguments) > 0 {
				continue
			}
			measurement := declaredInterfaceConformance{
				Owner: owner.XNA, GoOwner: "*" + owner.GoName,
				CLRInterface: mapped.XNA, GoInterface: mapped.GoName, Verdict: "FAIL",
			}
			interfaceObject := lookupNamed(actual.Packages[mappedType.PackagePath], mappedType.GoName)
			switch {
			case pkg == nil || ownerObject == nil:
				addDiagnostic(result, diagnostic{
					Category: "INTERFACE_MAPPING_MISMATCH", XNA: owner.XNA, Go: owner.Key.String(),
					Message: "compiler evidence for the declaring class is absent, so declared interface conformance cannot be checked",
				})
			case interfaceObject == nil:
				addDiagnostic(result, diagnostic{
					Category: "INTERFACE_MAPPING_MISMATCH", XNA: identity, Go: mappedType.Key.String(),
					Message: "compiler evidence for the mapped interface is absent, so declared interface conformance cannot be checked",
				})
			default:
				contract, ok := interfaceObject.Underlying().(*types.Interface)
				if !ok {
					addDiagnostic(result, diagnostic{
						Category: "INTERFACE_MAPPING_MISMATCH", XNA: identity, Go: mappedType.Key.String(),
						Message: "mapped XNA interface is not a Go interface",
					})
					break
				}
				contract.Complete()
				measurement.PointerSatisfies = types.Implements(types.NewPointer(ownerObject), contract)
				if measurement.PointerSatisfies {
					measurement.Verdict = "PASS"
					break
				}
				addDiagnostic(result, diagnostic{
					Category: "INTERFACE_MAPPING_MISMATCH", XNA: owner.XNA, Go: owner.Key.String(),
					Message: fmt.Sprintf("CLR metadata declares %s on this class, but *%s does not satisfy the Go projection %s: %s",
						mapped.XNA, owner.GoName, mapped.GoName,
						missingInterfaceMethod(types.NewPointer(ownerObject), contract)),
				})
			}
			measurements = append(measurements, measurement)
		}
	}
	return measurements
}

// lookupNamed resolves one exported Go type name to its *types.Named, or nil.
func lookupNamed(pkg *types.Package, name string) *types.Named {
	if pkg == nil {
		return nil
	}
	object := pkg.Scope().Lookup(name)
	if object == nil {
		return nil
	}
	named, ok := object.Type().(*types.Named)
	if !ok {
		return nil
	}
	return named
}

// missingInterfaceMethod names the first contract method the type does not
// satisfy, so a conformance failure says WHICH member is wrong rather than only
// that something is.
func missingInterfaceMethod(target types.Type, contract *types.Interface) string {
	method, wrongType := types.MissingMethod(target, contract, true)
	if method == nil {
		return "no missing method was reported"
	}
	if wrongType {
		return fmt.Sprintf("%s has the wrong signature", method.Name())
	}
	return fmt.Sprintf("%s is absent", method.Name())
}

// measureXNABaseRelationships records every CLR class in the pinned profile
// that another class in the same profile inherits from, with a status and,
// where the status is DEFERRED, what blocks it.
//
// The rule it enforces is the Foundation-29 discipline applied to a second
// frontier:
//
//	every XNA-to-XNA base link the contract declares must be recorded, a
//	deferred base must name what blocks it, and no derived type of a deferred
//	base may be COMPLETE
//
// The last clause is the substantive one. A derived type of a deferred base is
// missing its inherited public surface, so calling it complete would assert
// something false: Texture2D inherits nine members from Texture and
// GraphicsResource that CNA-Go does not project, and SpriteBatch inherits seven
// from GraphicsResource. Both are legitimately PARTIAL today, and this makes
// that a checked fact rather than a coincidence.
func measureXNABaseRelationships(result *report, expected *expectedSurface, complete []string) []xnaBaseProjection {
	completeSet := make(map[string]bool, len(complete))
	for _, identity := range complete {
		completeSet[identity] = true
	}
	derived := make(map[string][]string)
	for _, owner := range sortedExpectedTypes(expected) {
		if owner.BaseType == "" {
			continue
		}
		base := expected.typeForXNA(owner.BaseType)
		if base == nil {
			// The base is not in the pinned profile, so it is a BCL base and
			// belongs to the other frontier.
			continue
		}
		derived[owner.BaseType] = append(derived[owner.BaseType], owner.XNA)
	}

	// The claim is made only about bases the expected surface actually models.
	// An isolated per-type fixture carries one XNA type and no inheritance, so
	// demanding the whole frontier of it would fail every fixture for what it
	// does not model; a full surface carries all twelve and is measured in
	// full, including the stale-entry check, because the base type IS present
	// there.
	bases := make([]string, 0, len(derived))
	for base := range derived {
		bases = append(bases, base)
	}
	for base := range xnaBaseRelationships {
		if _, live := derived[base]; live {
			continue
		}
		if expected.typeForXNA(base) == nil {
			continue
		}
		bases = append(bases, base)
	}
	sort.Strings(bases)

	projections := make([]xnaBaseProjection, 0, len(bases))
	for _, base := range bases {
		relationship, recorded := xnaBaseRelationships[base]
		baseType := expected.typeForXNA(base)
		projection := xnaBaseProjection{
			CLRBase: base, Status: relationship.Status,
			Derived: append([]string(nil), derived[base]...), Verdict: "PASS",
		}
		sort.Strings(projection.Derived)
		if baseType != nil {
			projection.InheritedPublicMembers = baseType.PublicCLRMembers
		}
		fail := func(message string) {
			addDiagnostic(result, diagnostic{
				Category: "BASE_MAPPING_MISMATCH", XNA: base, Message: message,
			})
			projection.Verdict = "FAIL"
		}
		if !recorded {
			fail("an XNA class is inherited by another XNA class in the profile and the relationship is not recorded")
			projections = append(projections, projection)
			continue
		}
		if len(projection.Derived) == 0 {
			fail("a recorded XNA base relationship has no derived type in the pinned profile")
		}
		switch relationship.Status {
		case "COMPOSED":
			// Reserved: no XNA base is composed yet. When one is, its derived
			// types must re-expose its public surface, and the accounting rule
			// that admits them belongs here.
		case "DEFERRED":
			if len(relationship.Blockers) == 0 {
				fail("a DEFERRED XNA base records no blocker, so nothing says why it is deferred")
			}
			for _, blocker := range relationship.Blockers {
				projection.Blockers = append(projection.Blockers,
					xnaBaseBlockerRow{Class: blocker.Class, Detail: blocker.Detail})
				result.Summary["XNA_DEFERRED_BASE_BLOCKERS"]++
				if _, known := xnaBaseBlockerClasses[blocker.Class]; !known {
					fail(fmt.Sprintf("XNA base blocker class %q is not one of the recorded classes", blocker.Class))
				}
				if strings.TrimSpace(blocker.Detail) == "" {
					fail(fmt.Sprintf("XNA base blocker %q records no detail", blocker.Class))
				}
			}
			for _, identity := range projection.Derived {
				if completeSet[identity] {
					fail(fmt.Sprintf("%s derives from a DEFERRED XNA base and is reported COMPLETE, but it cannot be: the %d public members it inherits are not projected",
						identity, projection.InheritedPublicMembers))
				}
			}
			result.Summary["XNA_INHERITED_PUBLIC_MEMBERS_UNPROJECTED"] += projection.InheritedPublicMembers * len(projection.Derived)
		default:
			fail(fmt.Sprintf("XNA base status %q is neither COMPOSED nor DEFERRED", relationship.Status))
		}
		result.Summary["XNA_BASE_DERIVED_TYPES"] += len(projection.Derived)
		projections = append(projections, projection)
	}
	return projections
}

// ---------------------------------------------------------------------------
// Foundation 36 — the native game-signal bridge, and the frame-hook frontier.
// ---------------------------------------------------------------------------

// measureGameNativeSignals proves that every CLR event Game declares is bound
// to exactly one canonical CNA signal, through exactly the raise path the
// reference uses.
//
// Five claims it exists to defend:
//
//  1. Every declared signal names an event Game really declares, and every
//     event Game declares has a signal. A projected event with no raise path
//     is an accessor pair nothing can ever fire; a signal for an event that
//     does not exist is an invention.
//  2. Each event's two projected accessors are the ones the registry names, so
//     an accessor cannot be renamed out from under the binding.
//  3. A declared raise site is a REAL protected virtual of Game with the exact
//     (any, *EventArgs) error shape, and an ABSENT raise site means the
//     reference genuinely declares none. Disposed is the only one with none,
//     and inventing an OnDisposed would be caught here rather than shipped.
//  4. Every On... member Game projects is a declared raise site, so a raise
//     path cannot be added without being measured.
//  5. The runtime evidence is honest: an event the qualification environment
//     cannot deliver is NOT_RUN_ENVIRONMENT with a reason, and a verified one
//     carries no excuse.
//  6. Foundation 39: the event's AUTHORITATIVE raise path is recorded, and the
//     bound signal's role agrees with it. A native signal may implement the
//     raise path only when the semantics align; when the reference raises the
//     event from managed code CNA-Go projects, the signal is LIFECYCLE_ONLY and
//     the raise is a named projected member. Binding a signal to an event whose
//     reference raise site is not the host's -- which is the divergence
//     Foundation 34 shipped for Disposed -- is now a diagnostic.
func measureGameNativeSignals(result *report, expected *expectedSurface, actual *actualSurface) []gameNativeSignalMeasurement {
	frameworkPackage := modulePath + "/Microsoft/Xna/Framework"
	gameType := expected.typeForXNA("Microsoft.Xna.Framework.Game")
	if actual.PackageDirs[frameworkPackage] == "" || gameType == nil {
		return nil
	}

	// Game's events and its On... members, taken from the expected surface so
	// the registry is checked against the pinned contract rather than against a
	// second hand-written list.
	eventAccessors := make(map[string][]*expectedMember)
	raiseSites := make(map[string]*expectedMember)
	gameMembersByGoName := make(map[string]*expectedMember)
	for _, key := range gameType.Members {
		member := expected.Members[key]
		if member == nil {
			continue
		}
		gameMembersByGoName[member.GoName] = member
		switch {
		case member.SourceKind == "event":
			name := clrMemberName(member.XNA)
			eventAccessors[name] = append(eventAccessors[name], member)
		case member.SourceKind == "method" && strings.HasPrefix(member.GoName, "On"):
			raiseSites[member.GoName] = member
		}
	}

	names := make([]string, 0, len(gameNativeSignals))
	for name := range gameNativeSignals {
		names = append(names, name)
	}
	sort.Strings(names)

	declaredRaiseSites := make(map[string]bool, len(gameNativeSignals))
	identities := make(map[int]string, len(gameNativeSignals))
	measurements := make([]gameNativeSignalMeasurement, 0, len(gameNativeSignals))
	for _, name := range names {
		signal := gameNativeSignals[name]
		measurement := gameNativeSignalMeasurement{
			CNAConstant: signal.CNAConstant, CNAIdentity: signal.CNAIdentity,
			CLREvent: signal.CLREvent, RaiseSite: signal.RaiseSite,
			RaisePath: signal.RaisePath, NativeSignalRole: signal.NativeSignalRole,
			ManagedRaiseSite:   signal.ManagedRaiseSite,
			NativeSignalMoment: signal.NativeSignalMoment,
			Sender:             signal.Sender, EdgeTriggered: signal.EdgeTriggered,
			ReferencePath:   append([]string(nil), signal.ReferencePath...),
			RuntimeEvidence: signal.RuntimeEvidence, EvidenceReason: signal.EvidenceReason,
			Verdict: "PASS",
		}
		fail := func(message string) {
			addDiagnostic(result, diagnostic{
				Category: "LANGUAGE_MAPPING_MISMATCH", XNA: signal.CLREvent,
				Go: frameworkPackage + ":Game." + name, Message: message,
			})
			measurement.Verdict = "FAIL"
		}

		if other, duplicate := identities[signal.CNAIdentity]; duplicate {
			fail(fmt.Sprintf("CNA identity %d is already bound to %q; two events cannot share one signal", signal.CNAIdentity, other))
		}
		identities[signal.CNAIdentity] = name
		if signal.CNAConstant == "" {
			fail("declared signal names no CNA_GAME_EVENT_* constant")
		}

		// (1) and (2): the event exists and projects to the named accessor pair.
		accessors := eventAccessors[name]
		switch {
		case len(accessors) == 0:
			fail("declared signal names an event Game does not declare")
		case len(accessors) != 2:
			fail(fmt.Sprintf("event projects %d accessors; the settled event mapping projects exactly two", len(accessors)))
		default:
			for _, accessor := range accessors {
				switch {
				case strings.HasPrefix(accessor.GoName, "Add"):
					measurement.AddAccessor = accessor.GoName
				case strings.HasPrefix(accessor.GoName, "Remove"):
					measurement.RemoveAccessor = accessor.GoName
				}
				if _, present := actual.Members[accessor.Key]; !present {
					fail(fmt.Sprintf("projected accessor %s is absent, so the bound signal has nowhere to arrive", accessor.Key))
				}
			}
			if measurement.AddAccessor != "Add"+name+"Handler" || measurement.RemoveAccessor != "Remove"+name+"Handler" {
				fail(fmt.Sprintf("event projects accessors %q and %q, not the settled Add%sHandler/Remove%sHandler pair",
					measurement.AddAccessor, measurement.RemoveAccessor, name, name))
			}
		}

		// (3): the raise site is real, protected and exactly shaped -- or the
		// reference really declares none.
		if signal.RaiseSite == "" {
			if site, declared := raiseSites["On"+name]; declared {
				fail(fmt.Sprintf("registry records no raise site but the pinned contract declares %s", site.XNA))
			}
		} else {
			declaredRaiseSites[signal.RaiseSite] = true
			site, declared := raiseSites[signal.RaiseSite]
			if !declared {
				fail(fmt.Sprintf("declared raise site %s is not a projected method of Game", signal.RaiseSite))
			} else {
				measurement.RaiseSiteAccess = site.SourceAccess
				if site.SourceAccess != "protected" {
					fail(fmt.Sprintf("raise site %s is %q; the reference routes a raise through a protected virtual", signal.RaiseSite, site.SourceAccess))
				}
				if want := []string{"any", "*EventArgs"}; !equalStrings(want, site.Parameters) {
					fail(fmt.Sprintf("raise site %s takes %v, want %v", signal.RaiseSite, site.Parameters, want))
				}
				if want := []string{"error"}; !equalStrings(want, site.Results) {
					fail(fmt.Sprintf("raise site %s returns %v, want %v", signal.RaiseSite, site.Results, want))
				}
				if _, present := actual.Members[site.Key]; !present {
					fail(fmt.Sprintf("declared raise site %s is absent from the framework package", site.Key))
				}
			}
		}

		if !gameNativeSignalSenders[signal.Sender] {
			fail(fmt.Sprintf("sender %q is neither GAME nor NULL; every raise in this family pushes `this` or null", signal.Sender))
		}
		if len(signal.ReferencePath) == 0 {
			fail("declared signal records no reference raise path, so there is nothing to check the projection against")
		}

		// (5): the runtime evidence is a recorded class, and an unverified
		// signal must say why rather than passing quietly.
		if !gameNativeSignalEvidence[signal.RuntimeEvidence] {
			fail(fmt.Sprintf("runtime evidence %q is not VERIFIED_NATIVE or NOT_RUN_ENVIRONMENT", signal.RuntimeEvidence))
		}
		switch signal.RuntimeEvidence {
		case "NOT_RUN_ENVIRONMENT":
			result.Summary["GAME_NATIVE_SIGNALS_RUNTIME_DEFERRED"]++
			if strings.TrimSpace(signal.EvidenceReason) == "" {
				fail("a signal the environment cannot deliver must record why; an unexplained one is an unproved claim")
			}
		case "VERIFIED_NATIVE":
			if strings.TrimSpace(signal.EvidenceReason) != "" {
				fail("a verified signal records an evidence reason, which only an unverified one may carry")
			}
		}
		if signal.RaiseSite != "" {
			result.Summary["GAME_NATIVE_SIGNAL_RAISE_SITES"]++
		}

		// (6): the authoritative raise path, and the bound signal's role.
		if _, classified := gameEventRaisePaths[signal.RaisePath]; !classified {
			fail(fmt.Sprintf("raise path %q is neither NATIVE_HOST_SIGNAL nor MANAGED", signal.RaisePath))
		}
		if _, classified := gameNativeSignalRoles[signal.NativeSignalRole]; !classified {
			fail(fmt.Sprintf("native signal role %q is neither PUBLIC_EVENT_RAISE nor LIFECYCLE_ONLY", signal.NativeSignalRole))
		}
		if wanted, paired := gameEventRaisePathRoles[signal.RaisePath]; paired && wanted != signal.NativeSignalRole {
			fail(fmt.Sprintf("raise path %s pairs with signal role %s, not %s: a native signal may implement the raise path only when the semantics align, and it may not raise an event the reference raises somewhere else",
				signal.RaisePath, wanted, signal.NativeSignalRole))
		}
		if strings.TrimSpace(signal.NativeSignalMoment) == "" {
			fail("declared signal records no native moment, so there is nothing to compare the reference's raise site against")
		}
		switch signal.NativeSignalRole {
		case "LIFECYCLE_ONLY":
			result.Summary["GAME_NATIVE_SIGNALS_LIFECYCLE_ONLY"]++
		}
		if signal.RaisePath == "MANAGED" {
			result.Summary["GAME_MANAGED_EVENT_RAISE_SITES"]++
			switch {
			case signal.ManagedRaiseSite == "":
				fail("an event whose reference raise path is managed must name the projected member that raises it")
			default:
				member, projected := gameMembersByGoName[signal.ManagedRaiseSite]
				switch {
				case !projected:
					fail(fmt.Sprintf("managed raise site %s is not a projected member of Game", signal.ManagedRaiseSite))
				case member.SourceAccess != "protected":
					fail(fmt.Sprintf("managed raise site %s is a %q member; the reference raises this event from a protected virtual", signal.ManagedRaiseSite, member.SourceAccess))
				default:
					if _, present := actual.Members[member.Key]; !present {
						fail(fmt.Sprintf("declared managed raise site %s is absent from the framework package", signal.ManagedRaiseSite))
					}
				}
			}
		} else if signal.ManagedRaiseSite != "" {
			fail(fmt.Sprintf("a %s event names managed raise site %s; only a MANAGED raise path has one", signal.RaisePath, signal.ManagedRaiseSite))
		}
		measurements = append(measurements, measurement)
	}

	// (1), the other direction: no event Game declares may be left unbound.
	for name, accessors := range eventAccessors {
		if _, declared := gameNativeSignals[name]; declared {
			continue
		}
		addDiagnostic(result, diagnostic{
			Category: "LANGUAGE_MAPPING_MISMATCH", XNA: accessors[0].XNA,
			Go:      frameworkPackage + ":Game." + accessors[0].GoName,
			Message: "Game declares an event with no bound native signal, so the projected accessors have no raise path",
		})
	}

	// (4): every On... member Game projects must be a declared raise site.
	for goName, site := range raiseSites {
		if declaredRaiseSites[goName] {
			continue
		}
		addDiagnostic(result, diagnostic{
			Category: "LANGUAGE_MAPPING_MISMATCH", XNA: site.XNA,
			Go:      site.Key.String(),
			Message: "Game projects a raise site that no native signal declares, so a raise path exists that nothing measures",
		})
	}
	return measurements
}

// clrMemberName returns the member half of a "Namespace.Type::Member(...)"
// identity, without its parameter list.
func clrMemberName(identity string) string {
	name := identity
	if index := strings.LastIndex(name, "::"); index >= 0 {
		name = name[index+2:]
	}
	if index := strings.Index(name, "("); index >= 0 {
		name = name[:index]
	}
	return name
}

// measureGameFrameHooks proves that Game's four frame-boundary protected
// virtuals are projected as methods on Game, that each records the canonical
// CNA hook sitting at the same frame position, and that CNA-Go's decision not
// to install those hooks is a recorded decision with a reason.
//
// Four claims:
//
//  1. Each declared hook names a REAL protected virtual of Game, read from the
//     pinned contract.
//  2. None of them is a GameCallbacks member. The mandatory override contract
//     keeps exactly its five members, and a frame hook that joined it would
//     break every existing external implementation.
//  3. None of them has a GameBase... helper. The base-call registry is keyed by
//     the GameCallbacks members, so GameBaseBeginRun and friends are exactly
//     the invented helpers that registry's closure rule exists to stop.
//  4. An uninstalled native hook records why, and every unreproduced reference
//     step is classified and unobservable -- the same rule the base-call
//     adapters follow, because it is the same kind of claim.
//  5. A hook installed ON_OVERRIDE names a real, UNEXPORTED, single-method Go
//     interface whose one method has the measured shape, no two hooks share
//     one, and the exported spelling of it does not exist. That is what makes
//     "installed iff the consumer supplies an override" a measured fact rather
//     than a sentence: the capability is the only way to opt in, and it
//     publishes no new public contract.
//  6. Nothing offers a public registration surface. No exported member of the
//     framework package may end in "Override", so SetBeginDrawOverride,
//     RegisterEndRunOverride and every other spelling of mutable per-Game
//     callback state is refused by name.
func measureGameFrameHooks(result *report, expected *expectedSurface, actual *actualSurface) []gameFrameHookMeasurement {
	frameworkPackage := modulePath + "/Microsoft/Xna/Framework"
	gameType := expected.typeForXNA("Microsoft.Xna.Framework.Game")
	if actual.PackageDirs[frameworkPackage] == "" || gameType == nil {
		return nil
	}

	gameMembers := make(map[string]*expectedMember)
	callbackMembers := make(map[string]bool)
	for _, key := range gameType.Members {
		if member := expected.Members[key]; member != nil {
			gameMembers[member.GoName] = member
		}
	}
	for _, member := range expected.Members {
		if member.PackagePath == frameworkPackage && member.Receiver == "GameCallbacks" {
			callbackMembers[member.GoName] = true
		}
	}

	names := make([]string, 0, len(gameFrameHooks))
	for name := range gameFrameHooks {
		names = append(names, name)
	}
	sort.Strings(names)

	// The capability identities are measured against compiler evidence, so a
	// declared interface that does not exist -- or exists with the wrong shape,
	// or exported -- is a diagnostic rather than a comment.
	frameworkTypes := actual.Packages[frameworkPackage]
	capabilityOwners := make(map[string]string, len(gameFrameHooks))

	measurements := make([]gameFrameHookMeasurement, 0, len(gameFrameHooks))
	for _, name := range names {
		hook := gameFrameHooks[name]
		measurement := gameFrameHookMeasurement{
			CLRMember: hook.CLRMember,
			GoMember:  frameworkPackage + ":Game." + hook.GoName,
			// A nil parameter list and an empty one are the same claim; the
			// report spells both as an empty list so the JSON is stable.
			Parameters:           append([]string{}, hook.Parameters...),
			Results:              append([]string{}, hook.Results...),
			NativeHook:           hook.NativeHook,
			Installation:         hook.Installation,
			ReasonUninstalled:    hook.ReasonUninstalled,
			Capability:           hook.Capability,
			CapabilityMethod:     hook.CapabilityMethod,
			CapabilityParameters: append([]string{}, hook.CapabilityParameters...),
			CapabilityResults:    append([]string{}, hook.CapabilityResults...),
			NativeOrdering:       hook.NativeOrdering,
			ReferenceBody:        append([]string(nil), hook.ReferenceBody...),
			Verdict:              "PASS",
		}
		fail := func(message string) {
			addDiagnostic(result, diagnostic{
				Category: "LANGUAGE_MAPPING_MISMATCH", XNA: hook.CLRMember,
				Go: measurement.GoMember, Message: message,
			})
			measurement.Verdict = "FAIL"
		}

		// (1) a real protected virtual of Game.
		member, projected := gameMembers[hook.GoName]
		switch {
		case !projected:
			fail("declared frame hook is not a projected member of Game")
		default:
			measurement.CLRAccess = member.SourceAccess
			if member.SourceAccess != "protected" {
				fail(fmt.Sprintf("frame hook names a %q member; each of these four is a protected virtual", member.SourceAccess))
			}
			if wanted := member.XNA; !strings.HasPrefix(wanted, hook.CLRMember+"(") && wanted != hook.CLRMember {
				fail(fmt.Sprintf("frame hook declares CLR member %q but the projected member is %q", hook.CLRMember, wanted))
			}
			if member.Receiver != "Game" {
				fail(fmt.Sprintf("frame hook projects onto receiver %q; it must be a method on Game, where Microsoft declared it", member.Receiver))
			}
			if !equalStrings(hook.Parameters, member.Parameters) {
				fail(fmt.Sprintf("frame hook expects parameters %v, found %v", hook.Parameters, member.Parameters))
			}
			if !equalStrings(hook.Results, member.Results) {
				fail(fmt.Sprintf("frame hook expects results %v, found %v", hook.Results, member.Results))
			}
			if _, present := actual.Members[member.Key]; !present {
				fail("declared frame hook is absent from the framework package")
			}
		}

		// (2) it is not part of the mandatory override contract.
		if callbackMembers[hook.GoName] {
			fail("frame hook is also a GameCallbacks member; the mandatory override contract must keep exactly its five members")
		}

		// (3) and it has no base-call helper.
		helper := "GameBase" + hook.GoName
		if _, declared := gameBaseCallAdapters[hook.GoName]; declared {
			fail("frame hook has a declared base-call adapter; that registry is keyed by the GameCallbacks members")
		}
		if _, present := actual.Members[symbolKey{Package: frameworkPackage, Name: helper}]; present {
			fail(fmt.Sprintf("%s exists; a frame hook's base body is reachable as a method on Game and needs no helper", helper))
		}

		// (4) the install decision and every deferral are recorded.
		if hook.NativeHook == "" {
			fail("declared frame hook names no canonical CNA hook, so the position it claims to correspond to is unrecorded")
		}
		if strings.TrimSpace(hook.NativeOrdering) == "" {
			fail("declared frame hook records no measured native ordering")
		}
		if _, classified := gameFrameHookInstallations[hook.Installation]; !classified {
			fail(fmt.Sprintf("installation class %q is neither NEVER nor ON_OVERRIDE; there is no third class, because an unconditionally installed hook runs a base body at a frame position CNA-Go picked", hook.Installation))
		}
		if _, classified := gameFrameHookBaseInvocations[hook.BaseInvocation]; !classified {
			fail(fmt.Sprintf("base invocation %q is not EXPLICIT_ONLY; CNA-Go never runs a base body on the consumer's behalf", hook.BaseInvocation))
		}
		if strings.TrimSpace(hook.BaseInvocationEvidence) == "" {
			fail("a hook claiming the base runs only where the override calls it must record how that is known")
		}
		switch hook.Installation {
		case "NEVER":
			result.Summary["GAME_FRAME_HOOKS_NEVER_INSTALLED"]++
			if strings.TrimSpace(hook.ReasonUninstalled) == "" {
				fail("an uninstalled native hook must record why; an unexplained one is a silence rather than a decision")
			}
		case "ON_OVERRIDE":
			result.Summary["GAME_FRAME_HOOKS_INSTALLED_ON_OVERRIDE"]++
			if strings.TrimSpace(hook.ReasonUninstalled) != "" {
				fail("a hook installed behind an override records a reason for not installing it; it is installed whenever the consumer opts in")
			}
		}

		// (5) the capability behind an ON_OVERRIDE hook.
		switch {
		case hook.Installation != "ON_OVERRIDE":
			if hook.Capability != "" {
				fail(fmt.Sprintf("a %s hook names capability %q; only ON_OVERRIDE is opted into", hook.Installation, hook.Capability))
			}
		case hook.Capability == "":
			fail("a hook installed behind an override names no capability, so there is nothing a consumer can declare to opt in")
		default:
			result.Summary["GAME_FRAME_HOOK_OVERRIDE_CAPABILITIES"]++
			if owner, duplicate := capabilityOwners[hook.Capability]; duplicate {
				fail(fmt.Sprintf("capability %s is already the capability of %s; four independent overrides need four identities", hook.Capability, owner))
			}
			capabilityOwners[hook.Capability] = hook.GoName
			if exportedGoIdentifier(hook.Capability) {
				fail(fmt.Sprintf("capability %s is exported; the mechanism publishes no new public framework contract, and a structural interface needs no exported name", hook.Capability))
			}
			if hook.CapabilityMethod != hook.GoName {
				fail(fmt.Sprintf("capability %s declares method %q; a consumer overrides %s by declaring a method of that name", hook.Capability, hook.CapabilityMethod, hook.GoName))
			}
			if !equalStrings(hook.CapabilityParameters, gameFrameHookCapabilityParameters) {
				fail(fmt.Sprintf("capability %s takes %v; every optional override takes the owning Game, which is the `this` a CLR base call passes implicitly", hook.Capability, hook.CapabilityParameters))
			}
			if !equalStrings(hook.CapabilityResults, hook.Results) {
				fail(fmt.Sprintf("capability %s returns %v but the base body it replaces returns %v; the override and the base must share their channels exactly", hook.Capability, hook.CapabilityResults, hook.Results))
			}
			measureFrameHookCapabilityShape(fail, frameworkTypes, hook)
			// The exported spelling must not exist beside it. An exported twin
			// would be the public capability contract this design refuses.
			exported := strings.ToUpper(hook.Capability[:1]) + hook.Capability[1:]
			if _, present := actual.Types[symbolKey{Package: frameworkPackage, Name: exported}]; present {
				fail(fmt.Sprintf("%s exists beside the private capability %s; a capability interface must not be exported", exported, hook.Capability))
			}
		}
		if len(hook.ReferenceBody) == 0 {
			fail("declared frame hook records no reference body, so there is nothing to check the projection against")
		}
		for _, deferral := range hook.Deferred {
			measurement.Deferred = append(measurement.Deferred, gameBaseCallDeferralRow{
				Step: deferral.Step, Class: deferral.Class,
				Reason: deferral.Reason, Observable: deferral.Observable,
			})
			result.Summary["GAME_FRAME_HOOK_DEFERRED_STEPS"]++
			if !gameBaseCallDeferralClasses[deferral.Class] {
				fail(fmt.Sprintf("deferred reference step %q carries unclassified class %q", deferral.Step, deferral.Class))
			}
			if strings.TrimSpace(deferral.Reason) == "" {
				fail(fmt.Sprintf("deferred reference step %q records no reason", deferral.Step))
			}
			if deferral.Observable {
				fail(fmt.Sprintf("deferred reference step %q is observable, which makes it a stop condition rather than a deferral", deferral.Step))
			}
		}
		measurements = append(measurements, measurement)
	}

	measureGameCallbacksShape(result, actual, frameworkPackage)

	// (6) no public registration surface anywhere in the package. A consumer
	// opts in by declaring a method and by nothing else, so there is no
	// installation act to observe and no mutable per-Game callback state to
	// hold. Refusing the whole name shape, rather than a list of spellings,
	// is what makes that a closed claim.
	for _, key := range sortedActualMemberKeys(actual.Members) {
		if key.Package != frameworkPackage {
			continue
		}
		if !strings.HasSuffix(key.Name, "Override") && !strings.HasSuffix(key.Name, "Overrides") {
			continue
		}
		addDiagnostic(result, diagnostic{
			Category: "LANGUAGE_MAPPING_MISMATCH", XNA: "Microsoft.Xna.Framework.Game",
			Go:      key.String(),
			Message: "the framework package exports a member naming an override; the optional frame-hook capabilities are discovered once at construction and have no registration, replacement or removal API",
		})
	}
	for _, key := range sortedActualTypeKeys(actual.Types) {
		if key.Package != frameworkPackage {
			continue
		}
		if !strings.HasSuffix(key.Name, "Override") && !strings.HasSuffix(key.Name, "Overrides") {
			continue
		}
		addDiagnostic(result, diagnostic{
			Category: "LANGUAGE_MAPPING_MISMATCH", XNA: "Microsoft.Xna.Framework.Game",
			Go:      key.String(),
			Message: "the framework package exports a type naming an override; the four capabilities are unexported structural interfaces and publish no public contract",
		})
	}
	return measurements
}

// gameCallbacksMembers is the mandatory override contract, spelled out. It is
// closed and it is exactly five: the optional frame-hook overrides deliberately
// did NOT join it, because a sixth member would break every external
// implementation written against the five.
var gameCallbacksMembers = []string{"Draw", "Initialize", "LoadContent", "UnloadContent", "Update"}

// measureGameCallbacksShape holds the promise the override mechanism was shaped
// around, from the actual surface rather than from the mapping: GameCallbacks
// still declares exactly the five members, and Game still stores its callback
// state privately rather than in an exported or embedded field where anything
// could be substituted into it.
func measureGameCallbacksShape(result *report, actual *actualSurface, frameworkPackage string) {
	found := make([]string, 0, len(gameCallbacksMembers))
	for _, key := range sortedActualMemberKeys(actual.Members) {
		if key.Package == frameworkPackage && key.Receiver == "GameCallbacks" {
			found = append(found, key.Name)
		}
	}
	result.Summary["GAME_CALLBACKS_MEMBERS"] = len(found)
	if !equalStrings(found, gameCallbacksMembers) {
		addDiagnostic(result, diagnostic{
			Category: "LANGUAGE_MAPPING_MISMATCH", XNA: "Microsoft.Xna.Framework.Game",
			Go:      frameworkPackage + ":GameCallbacks",
			Message: fmt.Sprintf("GameCallbacks declares %v; the mandatory override contract is exactly %v, and the optional frame-hook overrides are structural capabilities precisely so it stays that way", found, gameCallbacksMembers),
		})
	}
	// GameCallbacks must embed nothing. An embedded `any`, or an embedded
	// second contract, would let a capability arrive through the mandatory
	// interface and would make its member count a fiction.
	if contract := actual.Types[symbolKey{Package: frameworkPackage, Name: "GameCallbacks"}]; contract != nil && len(contract.ExportedEmbeddings) != 0 {
		addDiagnostic(result, diagnostic{
			Category: "LANGUAGE_MAPPING_MISMATCH", XNA: "Microsoft.Xna.Framework.Game",
			Go:      frameworkPackage + ":GameCallbacks",
			Message: fmt.Sprintf("GameCallbacks embeds %v; the mandatory override contract declares its members directly so its size is a measured fact", contract.ExportedEmbeddings),
		})
	}
	// Game's captured capabilities are private fields. An exported or embedded
	// one would be an arbitrary callback slot a consumer could write into after
	// construction, which is the mutable registration state this design has
	// none of.
	game := actual.Types[symbolKey{Package: frameworkPackage, Name: "Game"}]
	if game == nil {
		return
	}
	for _, field := range game.Fields {
		if !field.Exported && !field.Embedded {
			continue
		}
		addDiagnostic(result, diagnostic{
			Category: "LANGUAGE_MAPPING_MISMATCH", XNA: "Microsoft.Xna.Framework.Game",
			Go:      frameworkPackage + ":Game." + field.Name,
			Message: "Game carries an exported or embedded field; the callback object and the four captured frame-hook capabilities are private state, decided once at construction and writable by nothing afterwards",
		})
	}
}

// measureFrameHookCapabilityShape proves the declared capability really is a
// one-method Go interface of the measured shape, using compiler evidence rather
// than the registry's own word for it.
//
// One method is the whole point. A bundled contract carrying all four would
// force a consumer who overrides one virtual to supply no-ops for the other
// three, and a no-op override is not the same thing as no override at all: it
// installs a native hook and takes the base's place at that frame position.
func measureFrameHookCapabilityShape(fail func(string), pkg *types.Package, hook gameFrameHook) {
	if pkg == nil {
		fail(fmt.Sprintf("compiler evidence for the framework package is absent, so capability %s cannot be measured", hook.Capability))
		return
	}
	named := lookupNamed(pkg, hook.Capability)
	if named == nil {
		fail(fmt.Sprintf("capability %s is declared by the registry and does not exist in the framework package", hook.Capability))
		return
	}
	contract, ok := named.Underlying().(*types.Interface)
	if !ok {
		fail(fmt.Sprintf("capability %s is not a Go interface, so nothing can satisfy it structurally", hook.Capability))
		return
	}
	contract.Complete()
	if contract.NumMethods() != 1 {
		fail(fmt.Sprintf("capability %s declares %d methods; each optional override is exactly one, so a consumer never supplies a no-op for a virtual they did not override", hook.Capability, contract.NumMethods()))
		return
	}
	method := contract.Method(0)
	if method.Name() != hook.CapabilityMethod {
		fail(fmt.Sprintf("capability %s declares %q, want %q", hook.Capability, method.Name(), hook.CapabilityMethod))
		return
	}
	if !method.Exported() {
		fail(fmt.Sprintf("capability %s declares an unexported method; an external consumer in another package could never satisfy it", hook.Capability))
		return
	}
	signature, ok := method.Type().(*types.Signature)
	if !ok {
		return
	}
	parameters := make([]string, 0, signature.Params().Len())
	for i := 0; i < signature.Params().Len(); i++ {
		parameters = append(parameters, frameHookCapabilityTypeName(signature.Params().At(i).Type()))
	}
	results := make([]string, 0, signature.Results().Len())
	for i := 0; i < signature.Results().Len(); i++ {
		results = append(results, frameHookCapabilityTypeName(signature.Results().At(i).Type()))
	}
	if !equalStrings(parameters, hook.CapabilityParameters) {
		fail(fmt.Sprintf("capability %s.%s takes %v, the registry declares %v", hook.Capability, method.Name(), parameters, hook.CapabilityParameters))
	}
	if !equalStrings(results, hook.CapabilityResults) {
		fail(fmt.Sprintf("capability %s.%s returns %v, the registry declares %v", hook.Capability, method.Name(), results, hook.CapabilityResults))
	}
}

// frameHookCapabilityTypeName spells a capability method's parameter or result
// the way the registry does: unqualified inside the declaring package, so
// *Game rather than *framework.Game.
func frameHookCapabilityTypeName(value types.Type) string {
	switch typed := value.(type) {
	case *types.Pointer:
		return "*" + frameHookCapabilityTypeName(typed.Elem())
	case *types.Named:
		return typed.Obj().Name()
	default:
		return value.String()
	}
}

// exportedGoIdentifier reports whether a Go identifier is exported, which for
// the capability names is the difference between a private mechanism and a new
// public framework contract.
func exportedGoIdentifier(name string) bool {
	if name == "" {
		return false
	}
	return strings.ToUpper(name[:1]) == name[:1]
}

// ---------------------------------------------------------------------------
// Foundation 40 — classifying the base-typed public signature inventory.
// ---------------------------------------------------------------------------

// xnaBaseSubstitutabilityRequirements are the only classes a base family's
// substitutability requirement may carry.
var xnaBaseSubstitutabilityRequirements = map[string]string{
	// No public signature in the whole profile names this base. Private
	// composition with explicit forwarding is not a compromise for such a
	// family -- it is exactly sufficient, and no reference abstraction can be
	// justified by anything in the contract.
	"NONE": "no public signature in the profile names this base, so no derived value can ever be required to stand in for one",
	// Positions exist, but nothing can flow through one today: either no
	// carrier of a position is projected, or no derived type is.
	"LATENT": "public positions name this base, but no projected carrier meets a projected derived type, so nothing can flow through one yet",
	// A projected carrier names the base AND a derived type is projected, so a
	// consumer can hold a derived value and a signature that must accept it.
	// This is the only state that forces a public reference abstraction.
	"LIVE": "a projected carrier names this base and a derived type is projected, so a derived value must be acceptable where the base is named",
}

// measureXNABaseSubstitutability turns the mechanical inventory into a per-family
// verdict, and cross-checks the contract-derived relationships against the
// registry so neither can drift alone.
//
// It is the measurement Foundation 40 exists for. The XNA-to-XNA inheritance
// architecture has to answer one question before it can choose a composition
// rule -- does any public XNA signature require a derived value to be accepted
// where its base is named -- and the honest way to answer it is to count, from
// the pinned contract, rather than to reason about what a framework "probably"
// does.
func measureXNABaseSubstitutability(result *report, expected *expectedSurface, actual *actualSurface) []xnaBaseSubstitutabilityMeasurement {
	// "Projected" is read straight from the actual surface rather than from the
	// missing-type list, so the measurement does not depend on where in verify
	// it happens to run.
	projected := make(map[string]bool, len(expected.Types))
	for _, et := range expected.Types {
		if _, present := actual.Types[et.Key]; present {
			projected[et.XNA] = true
		}
	}

	rows := make(map[string][]xnaBaseSubstitutabilityRow, len(expected.XNABaseDerivedByBase))
	for _, row := range expected.XNABaseSubstitutability {
		rows[row.Base] = append(rows[row.Base], row)
		result.Summary["XNA_BASE_TYPED_SIGNATURE_POSITIONS"]++
	}

	bases := make([]string, 0, len(expected.XNABaseDerivedByBase))
	for base := range expected.XNABaseDerivedByBase {
		bases = append(bases, base)
	}
	sort.Strings(bases)

	measurements := make([]xnaBaseSubstitutabilityMeasurement, 0, len(bases))
	for _, base := range bases {
		derived := expected.XNABaseDerivedByBase[base]
		measurement := xnaBaseSubstitutabilityMeasurement{
			Base: base, DerivedTypes: len(derived),
			Positions: len(rows[base]), Rows: append([]xnaBaseSubstitutabilityRow(nil), rows[base]...),
			Verdict: "PASS",
		}
		if measurement.Rows == nil {
			measurement.Rows = []xnaBaseSubstitutabilityRow{}
		}
		for _, name := range derived {
			if projected[name] {
				measurement.ProjectedDerivedTypes++
			}
		}
		for _, row := range rows[base] {
			if projected[row.Carrier] {
				measurement.PositionsOnProjectedCarriers++
			}
		}
		switch {
		case measurement.Positions == 0:
			measurement.Requirement = "NONE"
		case measurement.PositionsOnProjectedCarriers > 0 && measurement.ProjectedDerivedTypes > 0:
			measurement.Requirement = "LIVE"
		default:
			measurement.Requirement = "LATENT"
		}
		result.Summary["XNA_BASE_SUBSTITUTABILITY_"+measurement.Requirement]++

		// The registry and the contract must agree about which relationships
		// exist. The registry is hand-written and the inventory is derived, so
		// comparing them is what stops one from quietly going stale.
		if _, declared := xnaBaseRelationships[base]; !declared {
			addDiagnostic(result, diagnostic{
				Category: "BASE_MAPPING_MISMATCH", XNA: base, Go: "",
				Message: "the contract declares XNA classes deriving from this type and the XNA base registry does not record it",
			})
			measurement.Verdict = "FAIL"
		}
		measurements = append(measurements, measurement)
	}
	// The reverse direction, guarded on the base type actually being in the
	// surface under measurement. A synthetic fixture that declares only a few
	// types is not evidence that the registry invented a relationship; only a
	// surface that HAS the base and no class deriving from it is.
	registryBases := make([]string, 0, len(xnaBaseRelationships))
	for base := range xnaBaseRelationships {
		registryBases = append(registryBases, base)
	}
	sort.Strings(registryBases)
	for _, base := range registryBases {
		if _, present := expected.XNABaseDerivedByBase[base]; present {
			continue
		}
		if expected.typeForXNA(base) == nil {
			continue
		}
		addDiagnostic(result, diagnostic{
			Category: "BASE_MAPPING_MISMATCH", XNA: base, Go: "",
			Message: "the XNA base registry records a relationship the pinned contract does not declare",
		})
	}
	return measurements
}

// ---------------------------------------------------------------------------
// Foundation 41 — measuring the XNA-to-XNA composition rule.
// ---------------------------------------------------------------------------

// xnaCompositionForbiddenAccessors are the public member names a derived type
// must NOT gain just because it composes a base. The pinned contract declares
// none of them on any derived class, so every one would be invented surface --
// and each is a way to hand a consumer the base object and let them mutate it
// behind the derived type's back.
var xnaCompositionForbiddenAccessors = []string{"Base", "Parent", "BaseObject", "Inner", "Super"}

// measureXNAComposition proves the composition rule on every COMPOSED XNA base
// relationship: the base is private named state on the derived type, never Go
// embedding, and the derived type publishes no accessor for it.
//
// Four claims:
//
//  1. A COMPOSED relationship names a base the contract really declares, and
//     really has derived classes.
//  2. Every derived type CNA-Go projects holds the base as a PRIVATE NAMED
//     field of pointer type. Embedding is refused: it promotes the base's whole
//     method set, so a member the derived class overrides would silently keep
//     the base's body wherever the derived one was not redeclared with exactly
//     the right shape.
//  3. No derived type exposes Base, Parent or any other accessor for the base
//     object. The contract declares none, and Foundation 40 measured that no
//     public signature in the profile needs one.
//  4. Every inherited public member is attributed: it carries the exact XNA
//     base and the exact CLR member it came from, and it is never also counted
//     as XNA-declared or BCL-inherited.
func measureXNAComposition(result *report, expected *expectedSurface, actual *actualSurface) []xnaCompositionMeasurement {
	// A surface with no XNA-to-XNA relationships at all is a synthetic fixture
	// measuring something else, not a profile whose composed bases are missing.
	// Guarding on that keeps this measurement out of every other fixture while
	// still catching a COMPOSED entry that names nothing in the real contract.
	if len(expected.XNABaseDerivedByBase) == 0 {
		return nil
	}
	bases := make([]string, 0, len(xnaBaseRelationships))
	for base := range xnaBaseRelationships {
		bases = append(bases, base)
	}
	sort.Strings(bases)

	measurements := make([]xnaCompositionMeasurement, 0)
	for _, base := range bases {
		relationship := xnaBaseRelationships[base]
		if relationship.Status != "COMPOSED" {
			continue
		}
		result.Summary["XNA_COMPOSED_BASE_RELATIONSHIPS"]++
		baseType := expected.typeForXNA(base)
		measurement := xnaCompositionMeasurement{CLRBase: base, Verdict: "PASS"}
		if baseType == nil {
			addDiagnostic(result, diagnostic{
				Category: "BASE_MAPPING_MISMATCH", XNA: base,
				Message: "a COMPOSED XNA base relationship names a type the pinned contract does not declare",
			})
			measurements = append(measurements, measurement)
			continue
		}
		measurement.GoBase = baseType.GoName

		for _, derivedName := range expected.XNABaseDerivedByBase[base] {
			derived := expected.typeForXNA(derivedName)
			if derived == nil {
				continue
			}
			result.Summary["XNA_COMPOSED_DERIVED_TYPES"]++
			row := xnaCompositionRow{
				Derived: derivedName, GoDerived: derived.GoName,
				InheritedCLRMembers:  derived.XNAInheritedCLRMembers,
				InheritedProjections: derived.XNAInheritedProjections,
				Composition:          "NOT_PROJECTED",
			}
			at, projected := actual.Types[derived.Key]
			if !projected {
				measurement.Rows = append(measurement.Rows, row)
				continue
			}
			result.Summary["XNA_COMPOSED_DERIVED_TYPES_PROJECTED"]++
			fail := func(message string) {
				addDiagnostic(result, diagnostic{
					Category: "BASE_MAPPING_MISMATCH", XNA: derivedName,
					Go: derived.Key.String(), Message: message,
				})
				measurement.Verdict = "FAIL"
				row.Composition = "FAIL"
			}

			// (2) private named composition, never embedding.
			wanted := "*" + baseType.GoName
			held := false
			for _, field := range at.Fields {
				if field.Embedded {
					fail(fmt.Sprintf("%s embeds %q; an XNA base is private named state, because embedding promotes the base's whole method set and a member the derived class overrides would silently keep the base's body", derived.GoName, field.Type))
					continue
				}
				if field.Type != wanted {
					continue
				}
				if field.Exported {
					fail(fmt.Sprintf("%s holds its base in the EXPORTED field %s; the base object is private implementation state and the contract declares no accessor for it", derived.GoName, field.Name))
					continue
				}
				held = true
				row.BaseField = field.Name
			}
			if !held && row.Composition != "FAIL" {
				fail(fmt.Sprintf("%s holds no private %s field; a COMPOSED XNA base is projected as private named composition", derived.GoName, wanted))
			}

			// (3) no accessor for the base object.
			for _, forbidden := range xnaCompositionForbiddenAccessors {
				key := symbolKey{Package: derived.PackagePath, Receiver: derived.GoName, Name: forbidden}
				if _, present := actual.Members[key]; present {
					fail(fmt.Sprintf("%s exposes %s; the pinned contract declares no accessor for the base object and no public signature in the profile needs one", derived.GoName, forbidden))
				}
			}
			key := symbolKey{Package: derived.PackagePath, Receiver: derived.GoName, Name: "As" + baseType.GoName}
			if _, present := actual.Members[key]; present {
				fail(fmt.Sprintf("%s exposes As%s; the contract declares no such conversion", derived.GoName, baseType.GoName))
			}
			if row.Composition != "FAIL" {
				row.Composition = "PRIVATE_NAMED"
			}
			measurement.Rows = append(measurement.Rows, row)
		}
		measurements = append(measurements, measurement)
	}

	// (4) every member carries exactly one provenance class.
	for _, key := range sortedMemberKeys(expected) {
		member := expected.Members[key]
		if member.XNABase == "" {
			continue
		}
		result.Summary["XNA_INHERITED_ATTRIBUTED_MEMBERS"]++
		switch {
		case member.BCLBase != "":
			addDiagnostic(result, diagnostic{
				Category: "BASE_MAPPING_MISMATCH", XNA: member.XNA, Go: key.String(),
				Message: "member carries both an XNA base and a BCL base; every projected member has exactly one provenance class",
			})
		case member.XNABaseMember == "":
			addDiagnostic(result, diagnostic{
				Category: "BASE_MAPPING_MISMATCH", XNA: member.XNA, Go: key.String(),
				Message: "an XNA-inherited member names no CLR member on its base, so the attribution the provenance class promises is incomplete",
			})
		case expected.typeForXNA(member.XNABase) == nil:
			addDiagnostic(result, diagnostic{
				Category: "BASE_MAPPING_MISMATCH", XNA: member.XNA, Go: key.String(),
				Message: "an XNA-inherited member names a base the pinned contract does not declare",
			})
		}
	}
	return measurements
}
