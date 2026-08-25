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
		measurement.GraphicsManagerRemainingMissing == 40 {
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
