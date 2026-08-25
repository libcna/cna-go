package main

import (
	"encoding/json"
	"go/types"
)

type contract struct {
	SchemaVersion int            `json:"schemaVersion"`
	Profile       string         `json:"profile"`
	Types         []contractType `json:"types"`
}

type contractType struct {
	Name              string             `json:"name"`
	Kind              string             `json:"kind"`
	Flags             bool               `json:"flags"`
	Sealed            bool               `json:"sealed"`
	UnderlyingType    *string            `json:"underlyingType"`
	BaseType          *string            `json:"baseType"`
	Interfaces        []string           `json:"interfaces"`
	DirectInterfaces  []string           `json:"directInterfaces"`
	GenericParameters []genericParameter `json:"genericParameters"`
	Members           []contractMember   `json:"members"`
}

type genericParameter struct {
	Name               string   `json:"name"`
	Position           int      `json:"position"`
	SpecialConstraints []string `json:"specialConstraints"`
	TypeConstraints    []string `json:"typeConstraints"`
}

type contractMember struct {
	Kind              string              `json:"kind"`
	Name              string              `json:"name"`
	Static            bool                `json:"static"`
	Access            string              `json:"access"`
	ReturnType        *string             `json:"returnType"`
	Type              *string             `json:"type"`
	Constant          bool                `json:"constant"`
	Value             json.RawMessage     `json:"value"`
	Get               bool                `json:"get"`
	Set               bool                `json:"set"`
	GetAccess         *string             `json:"getAccess"`
	SetAccess         *string             `json:"setAccess"`
	GenericParameters []genericParameter  `json:"genericParameters"`
	Parameters        []contractParameter `json:"parameters"`
}

type contractParameter struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Ref      bool   `json:"ref"`
	Out      bool   `json:"out"`
	In       bool   `json:"in"`
	Optional bool   `json:"optional"`
}

type expectedSurface struct {
	Types              map[symbolKey]*expectedType
	Members            map[symbolKey]*expectedMember
	InterfaceWitnesses map[symbolKey]*expectedInterfaceWitness
	MappingIssues      []diagnostic
	ReferenceTypes     int
	ReferenceMembers   int
	ExpectedGoTypes    int
	ExpectedGoMembers  int
}

type expectedType struct {
	Key              symbolKey
	XNA              string
	GoName           string
	PackagePath      string
	Kind             string
	Flags            bool
	BaseType         string
	Interfaces       []string
	AllInterfaces    []string
	GenericParameter []string
	MappedInterfaces []mappedInterface
	Members          []symbolKey
	SourceMembers    int
}

type mappedInterface struct {
	XNA           string
	GoName        string
	TypeArguments []string
}

type expectedInterfaceWitness struct {
	Key             symbolKey
	Owner           string
	SourceInterface string
	InterfaceMember string
	GoName          string
	Parameters      []string
	Results         []string
	Reason          string
}

type expectedMember struct {
	Key            symbolKey
	XNA            string
	Owner          string
	SourceKind     string
	GoKind         string
	GoName         string
	PackagePath    string
	Receiver       string
	Parameters     []string
	Results        []string
	EnumValue      *string
	FlagsOwner     bool
	OverloadMapped bool
	ErrorAdded     bool
}

type actualSurface struct {
	Types       map[symbolKey]*actualType
	Members     map[symbolKey]*actualMember
	TypeErrors  []string
	Unmeasured  []string
	PackageDirs map[string]string
	Packages    map[string]*types.Package
}

type actualType struct {
	Key                symbolKey
	Kind               string
	Underlying         string
	FlagsMarker        bool
	TypeParameters     []string
	ExportedEmbeddings []string
}

type actualMember struct {
	Key        symbolKey
	Kind       string
	Parameters []string
	Results    []string
	Value      *string
	Position   string
}

type symbolKey struct {
	Package  string
	Receiver string
	Name     string
}

func (k symbolKey) String() string {
	if k.Receiver != "" {
		return k.Package + ":" + k.Receiver + "." + k.Name
	}
	return k.Package + ":" + k.Name
}

type diagnostic struct {
	Category string `json:"category"`
	XNA      string `json:"xna,omitempty"`
	Go       string `json:"go,omitempty"`
	Message  string `json:"message"`
}

type report struct {
	SchemaVersion                int                           `json:"schemaVersion"`
	Profile                      string                        `json:"profile"`
	Mode                         string                        `json:"mode"`
	Summary                      map[string]int                `json:"summary"`
	Diagnostics                  []diagnostic                  `json:"diagnostics"`
	CompleteTypes                []string                      `json:"completeTypes"`
	PartialTypes                 []typeStatus                  `json:"partialTypes"`
	MissingTypes                 []string                      `json:"missingTypes"`
	InterfaceWitnessProjections  []interfaceWitnessProjection  `json:"interfaceWitnessProjections,omitempty"`
	PackedInterfaceConformance   []packedInterfaceConformance  `json:"packedInterfaceConformance,omitempty"`
	PackedVectorTypeMeasurements []packedVectorTypeMeasurement `json:"packedVectorTypeMeasurements,omitempty"`
	VertexElementClosure         vertexElementClosure          `json:"vertexElementClosure"`
	PlayerIndexKeyboardClosure   playerIndexKeyboardClosure    `json:"playerIndexKeyboardClosure"`
	DisplayOrientationClosure    displayOrientationClosure     `json:"displayOrientationGraphicsManagerClosure"`
	BufferUsageClosure           bufferUsageClosure            `json:"bufferUsageClosure"`
	ClearOptionsClosure          clearOptionsClosure           `json:"clearOptionsClosure"`
	SurfaceFormatClosure         surfaceFormatClosure          `json:"surfaceFormatClosure"`
	DepthFormatClosure           depthFormatClosure            `json:"depthFormatClosure"`
	GraphicsProfileClosure       graphicsProfileClosure        `json:"graphicsProfileClosure"`
	Metadata                     reportMetadata                `json:"metadata"`
}

type graphicsProfileClosure struct {
	SourceTypes          int                    `json:"sourceTypes"`
	SourceIdentities     int                    `json:"sourceIdentities"`
	ExpectedGoIdentities int                    `json:"expectedGoIdentities"`
	TargetTypes          int                    `json:"targetTypes"`
	TargetGoIdentities   int                    `json:"targetGoIdentities"`
	LocalDiagnostics     int                    `json:"localDiagnostics"`
	ExpectedKind         string                 `json:"expectedKind"`
	ActualKind           string                 `json:"actualKind"`
	UnderlyingType       string                 `json:"underlyingType"`
	Flags                bool                   `json:"flags"`
	ValueStorageExcluded bool                   `json:"valueStorageExcluded"`
	Values               []enumValueMeasurement `json:"values"`
	Status               string                 `json:"status"`
}

type depthFormatClosure struct {
	SourceTypes          int                    `json:"sourceTypes"`
	SourceIdentities     int                    `json:"sourceIdentities"`
	ExpectedGoIdentities int                    `json:"expectedGoIdentities"`
	TargetTypes          int                    `json:"targetTypes"`
	TargetGoIdentities   int                    `json:"targetGoIdentities"`
	LocalDiagnostics     int                    `json:"localDiagnostics"`
	ExpectedKind         string                 `json:"expectedKind"`
	ActualKind           string                 `json:"actualKind"`
	UnderlyingType       string                 `json:"underlyingType"`
	Flags                bool                   `json:"flags"`
	ValueStorageExcluded bool                   `json:"valueStorageExcluded"`
	Values               []enumValueMeasurement `json:"values"`
	Status               string                 `json:"status"`
}

type surfaceFormatClosure struct {
	SourceTypes          int                    `json:"sourceTypes"`
	SourceIdentities     int                    `json:"sourceIdentities"`
	ExpectedGoIdentities int                    `json:"expectedGoIdentities"`
	TargetTypes          int                    `json:"targetTypes"`
	TargetGoIdentities   int                    `json:"targetGoIdentities"`
	LocalDiagnostics     int                    `json:"localDiagnostics"`
	ExpectedKind         string                 `json:"expectedKind"`
	ActualKind           string                 `json:"actualKind"`
	UnderlyingType       string                 `json:"underlyingType"`
	Flags                bool                   `json:"flags"`
	ValueStorageExcluded bool                   `json:"valueStorageExcluded"`
	Values               []enumValueMeasurement `json:"values"`
	Status               string                 `json:"status"`
}

type enumValueMeasurement struct {
	Name          string `json:"name"`
	ExpectedValue string `json:"expectedValue"`
	ActualValue   string `json:"actualValue"`
	Status        string `json:"status"`
}

type clearOptionsClosure struct {
	SourceTypes                int    `json:"sourceTypes"`
	SourceIdentities           int    `json:"sourceIdentities"`
	ExpectedGoIdentities       int    `json:"expectedGoIdentities"`
	TargetTypes                int    `json:"targetTypes"`
	TargetGoIdentities         int    `json:"targetGoIdentities"`
	LocalDiagnostics           int    `json:"localDiagnostics"`
	ExpectedKind               string `json:"expectedKind"`
	ActualKind                 string `json:"actualKind"`
	UnderlyingType             string `json:"underlyingType"`
	Flags                      bool   `json:"flags"`
	TargetValue                string `json:"targetValue"`
	DepthBufferValue           string `json:"depthBufferValue"`
	StencilValue               string `json:"stencilValue"`
	ValueStorageExcluded       bool   `json:"valueStorageExcluded"`
	NamedZeroMember            bool   `json:"namedZeroMember"`
	ClearOptionsNonePresent    bool   `json:"clearOptionsNonePresent"`
	ClearOptionsDefaultPresent bool   `json:"clearOptionsDefaultPresent"`
	ClearOptionsAllPresent     bool   `json:"clearOptionsAllPresent"`
	Status                     string `json:"status"`
}

type bufferUsageClosure struct {
	SourceTypes          int    `json:"sourceTypes"`
	SourceIdentities     int    `json:"sourceIdentities"`
	ExpectedGoIdentities int    `json:"expectedGoIdentities"`
	TargetTypes          int    `json:"targetTypes"`
	TargetGoIdentities   int    `json:"targetGoIdentities"`
	LocalDiagnostics     int    `json:"localDiagnostics"`
	ExpectedKind         string `json:"expectedKind"`
	ActualKind           string `json:"actualKind"`
	UnderlyingType       string `json:"underlyingType"`
	Flags                bool   `json:"flags"`
	NoneValue            string `json:"noneValue"`
	WriteOnlyValue       string `json:"writeOnlyValue"`
	ValueStorageExcluded bool   `json:"valueStorageExcluded"`
	Status               string `json:"status"`
}

type displayOrientationClosure struct {
	SourceTypes                        int                                  `json:"sourceTypes"`
	SourceIdentities                   int                                  `json:"sourceIdentities"`
	MappedGoIdentities                 int                                  `json:"mappedGoIdentities"`
	TargetTypes                        int                                  `json:"targetTypes"`
	TargetGoIdentities                 int                                  `json:"targetGoIdentities"`
	DisplayOrientationLocalDiagnostics int                                  `json:"displayOrientationLocalDiagnostics"`
	SupportedPropertyLocalDiagnostics  int                                  `json:"supportedOrientationsLocalDiagnostics"`
	GraphicsManagerRemainingMissing    int                                  `json:"graphicsDeviceManagerRemainingMissingMembers"`
	Status                             string                               `json:"status"`
	SliceMeasurements                  []displayOrientationSliceMeasurement `json:"sliceMeasurements"`
}

type displayOrientationSliceMeasurement struct {
	XNA               string `json:"xna"`
	GoName            string `json:"goName"`
	Scope             string `json:"scope"`
	SourceMembers     int    `json:"sourceMembers"`
	ExpectedGoMembers int    `json:"expectedGoMembers"`
	TargetGoMembers   int    `json:"targetGoMembers"`
	LocalDiagnostics  int    `json:"localDiagnostics"`
	ExpectedKind      string `json:"expectedKind"`
	ActualKind        string `json:"actualKind"`
	ActualUnderlying  string `json:"actualUnderlying,omitempty"`
}

type playerIndexKeyboardClosure struct {
	SourceTypes        int                          `json:"sourceTypes"`
	SourceIdentities   int                          `json:"sourceIdentities"`
	MappedGoIdentities int                          `json:"mappedGoIdentities"`
	TargetTypes        int                          `json:"targetTypes"`
	TargetGoIdentities int                          `json:"targetGoIdentities"`
	LocalDiagnostics   int                          `json:"localDiagnostics"`
	Status             string                       `json:"status"`
	TypeMeasurements   []playerIndexTypeMeasurement `json:"typeMeasurements"`
}

type playerIndexTypeMeasurement struct {
	XNA               string `json:"xna"`
	GoName            string `json:"goName"`
	SourceMembers     int    `json:"sourceMembers"`
	ExpectedGoMembers int    `json:"expectedGoMembers"`
	TargetGoMembers   int    `json:"targetGoMembers"`
	LocalDiagnostics  int    `json:"localDiagnostics"`
	ExpectedKind      string `json:"expectedKind"`
	ActualKind        string `json:"actualKind"`
	ActualUnderlying  string `json:"actualUnderlying,omitempty"`
}

type vertexElementClosure struct {
	SourceTypes        int                            `json:"sourceTypes"`
	SourceIdentities   int                            `json:"sourceIdentities"`
	MappedGoIdentities int                            `json:"mappedGoIdentities"`
	TargetTypes        int                            `json:"targetTypes"`
	TargetGoIdentities int                            `json:"targetGoIdentities"`
	WritableProperties int                            `json:"writableProperties"`
	ProjectedAccessors int                            `json:"projectedAccessors"`
	LocalDiagnostics   int                            `json:"localDiagnostics"`
	Status             string                         `json:"status"`
	TypeMeasurements   []vertexElementTypeMeasurement `json:"typeMeasurements"`
}

type vertexElementTypeMeasurement struct {
	XNA               string `json:"xna"`
	GoName            string `json:"goName"`
	SourceMembers     int    `json:"sourceMembers"`
	ExpectedGoMembers int    `json:"expectedGoMembers"`
	TargetGoMembers   int    `json:"targetGoMembers"`
	LocalDiagnostics  int    `json:"localDiagnostics"`
	ExpectedKind      string `json:"expectedKind"`
	ActualKind        string `json:"actualKind"`
	ActualUnderlying  string `json:"actualUnderlying,omitempty"`
}

type interfaceWitnessProjection struct {
	Owner           string `json:"owner"`
	Member          string `json:"member"`
	SourceInterface string `json:"sourceInterface"`
	InterfaceMember string `json:"interfaceMember"`
	Reason          string `json:"reason"`
	Signature       string `json:"signature"`
	Status          string `json:"status"`
}

type packedInterfaceConformance struct {
	Owner                     string `json:"owner"`
	Interface                 string `json:"interface"`
	TPacked                   string `json:"tPacked"`
	PointerMethodSetSatisfies bool   `json:"pointerMethodSetSatisfies"`
	ValueMethodSetSatisfies   bool   `json:"valueMethodSetSatisfies"`
	TransitiveBaseSatisfies   bool   `json:"transitiveBaseSatisfies"`
	Status                    string `json:"status"`
}

type packedVectorTypeMeasurement struct {
	XNA                   string `json:"xna"`
	GoName                string `json:"goName"`
	SourceMembers         int    `json:"sourceMembers"`
	ExpectedGoMembers     int    `json:"expectedGoMembers"`
	TargetGoMembers       int    `json:"targetGoMembers"`
	LocalDiagnostics      int    `json:"localDiagnostics"`
	TypeKind              string `json:"typeKind"`
	TPacked               string `json:"tPacked,omitempty"`
	DirectInterfaceStatus string `json:"directPackedInterfaceStatus,omitempty"`
}

type typeStatus struct {
	XNA            string   `json:"xna"`
	MissingMembers []string `json:"missingMembers"`
	Diagnostics    int      `json:"diagnostics"`
}

type reportMetadata struct {
	ContractSHA256  string `json:"contractSha256"`
	MappingSHA256   string `json:"mappingSha256"`
	Extractor       string `json:"extractor"`
	TypeCheckErrors int    `json:"typeCheckErrors"`
}

var diagnosticCategories = []string{
	"MISSING_TYPE",
	"MISSING_MEMBER",
	"UNEXPECTED_TYPE",
	"UNEXPECTED_MEMBER",
	"TYPE_KIND_MISMATCH",
	"BASE_MAPPING_MISMATCH",
	"INTERFACE_MAPPING_MISMATCH",
	"FIELD_MAPPING_MISMATCH",
	"PROPERTY_MAPPING_MISMATCH",
	"METHOD_SIGNATURE_MAPPING_MISMATCH",
	"PARAMETER_MAPPING_MISMATCH",
	"RETURN_MAPPING_MISMATCH",
	"ERROR_MAPPING_MISMATCH",
	"OVERLOAD_MAPPING_MISMATCH",
	"GENERIC_MAPPING_MISMATCH",
	"ENUM_VALUE_MISMATCH",
	"FLAGS_MAPPING_MISMATCH",
	"EVENT_MAPPING_MISMATCH",
	"OPERATOR_MAPPING_MISMATCH",
	"REF_OUT_MAPPING_MISMATCH",
	"LANGUAGE_MAPPING_MISMATCH",
	"INTERNAL_TYPE_LEAK",
	"RAW_HANDLE_LEAK",
	"PUBLIC_NATIVE_FFI_LEAK",
	"ALLOWLIST_ENTRIES",
	"UNMEASURED_STRUCTURAL_CATEGORY",
}
