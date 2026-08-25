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
	// BCLInheritedCLRMembers is how many public CLR members the supported BCL
	// bases contribute across the profile, counted once per derived type.
	BCLInheritedCLRMembers int
	// BCLInheritedProjections is how many Go identities those members project
	// to. It differs from BCLInheritedCLRMembers because a CLR property with
	// both accessors projects two Go members.
	BCLInheritedProjections int
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
	// BCLInheritedCLRMembers is how many public CLR members this type's
	// supported BCL base contributes, and BCLInheritedProjections is how many
	// Go identities they become. Both are zero for a type with no COMPOSED
	// base, which is every type in the profile but the collection consumers.
	BCLInheritedCLRMembers  int
	BCLInheritedProjections int
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
	// Accessor is "get" or "set" when this member is one projected accessor
	// of a CLR property and empty for every other member kind. Fallibility is
	// decided per accessor, so the two accessors of one property are separate
	// expected members that can disagree about their error result.
	Accessor string
	// BCLBase is the CLR identity of the supported BCL base class this member
	// is inherited from, and is empty for a member the XNA assembly declares
	// itself. It is the member's provenance class: every expected Go member
	// has exactly one, so no member is counted both as XNA-declared and as
	// BCL-inherited.
	BCLBase string
	// BCLMember is the exact CLR member name on that base. Together with
	// BCLBase and Key it is the full attribution the inherited projection
	// promises: exact BCL base, exact CLR member, exact projected Go member.
	BCLMember string
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
	// Fields is every struct field the type declares, exported or not. The
	// unexported ones are measured because the BCL base-class composition
	// projection makes a private field load-bearing: a COMPOSED base must be
	// held in an unexported field of a declared name and adapter type, and
	// that claim cannot be checked from the public surface alone.
	Fields []actualField
}

type actualField struct {
	Name     string
	Type     string
	Exported bool
	Embedded bool
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
	ButtonStateClosure           buttonStateClosure            `json:"buttonStateClosure"`
	Foundation14EnumClosures     []enumClosure                 `json:"foundation14EnumClosures"`
	Foundation15EnumClosures     []enumClosure                 `json:"foundation15EnumClosures"`
	Foundation15ValueStructs     []valueStructClosure          `json:"foundation15ValueStructClosures"`
	Foundation16ValueStructs     []valueStructClosure          `json:"foundation16ValueStructClosures"`
	Foundation17ManagedClasses   []managedTypeClosure          `json:"foundation17ManagedClassClosures"`
	Foundation18Interfaces       []managedInterfaceClosure     `json:"foundation18InterfaceClosures"`
	Foundation19ManagedClasses   []managedTypeClosure          `json:"foundation19ManagedClassClosures"`
	Foundation20ValueContracts   []managedTypeClosure          `json:"foundation20ValueContractClosures"`
	Foundation21ManagedClasses   []managedTypeClosure          `json:"foundation21ManagedClassClosures"`
	Foundation23Interfaces       []managedInterfaceClosure     `json:"foundation23InterfaceClosures"`
	Foundation23ManagedClasses   []managedTypeClosure          `json:"foundation23ManagedClassClosures"`
	BCLBaseRelationships         []bclBaseProjection           `json:"bclBaseRelationships"`
	BCLBaseAdapters              []bclBaseAdapterMeasurement   `json:"bclBaseAdapters"`
	BCLInterfaceRelationships    []bclInterfaceProjection      `json:"bclInterfaceRelationships"`
	Metadata                     reportMetadata                `json:"metadata"`
}

// bclBaseProjection measures one non-XNA CLR base type across every XNA type
// that derives from it.
//
// Go has no CLR inheritance and CNA-Go deliberately refuses to fake it with
// exported embedding, so a derived type is projected as its own reference type
// and the base survives only as this measured relationship. Recording it is
// what keeps a dropped base from being invisible: every non-XNA base in the
// pinned contract appears here with a status, so a base nobody has decided
// about is a diagnostic rather than a silent omission.
type bclBaseProjection struct {
	// CLRBase is the CLR base identity with any generic arguments removed.
	CLRBase string `json:"clrBase"`
	// Adapter is the framework-package language adapter that models the base,
	// empty when the relationship contributes no adapter.
	Adapter string `json:"adapter,omitempty"`
	// Status is IMPLIED, MAPPED, or DEFERRED.
	Status string `json:"status"`
	// AddsProjectedSurface records the invariant every relationship in this
	// table shares: a base contributes no Go member identities of its own.
	AddsProjectedSurface bool `json:"addsProjectedSurface"`
	// DerivedTypes is every XNA type in the profile with this CLR base.
	DerivedTypes int `json:"derivedTypes"`
	// ProjectedTypes is how many of them are present in the Go surface.
	ProjectedTypes int `json:"projectedTypes"`
	// ExportedEmbeddings counts derived Go types that faked the inheritance
	// with an exported embedded field. It must stay zero.
	ExportedEmbeddings int    `json:"exportedEmbeddings"`
	Rationale          string `json:"rationale"`
	Verdict            string `json:"verdict"`
}

// bclBaseAdapterMeasurement measures one supported BCL base-class family: the
// registry entry, the private Go adapter that models it, the exact inherited
// public member inventory, and every concrete XNA consumer.
//
// It is the whole identity accounting for the composition projection. Each
// inherited projection is attributable to an exact BCL base, an exact CLR
// member, and an exact projected Go member, and none of them is counted as an
// XNA-declared reference member.
type bclBaseAdapterMeasurement struct {
	CLRBase string `json:"clrBase"`
	// GoAdapter is the unexported adapter family. It is never exported and
	// never named by a projected signature.
	GoAdapter string `json:"goAdapter"`
	// AdapterField is the unexported field a consumer holds it in.
	AdapterField  string `json:"adapterField"`
	BehaviorLevel string `json:"behaviorLevel"`
	// Authority is the exact BCL binary the behavior was read from, pinned by
	// AuthoritySHA256.
	Authority       string `json:"authority"`
	AuthoritySHA256 string `json:"authoritySha256"`
	// InheritedCLRMembers is the size of the public member inventory, and
	// InheritedProjections is how many Go identities one consumer gains.
	InheritedCLRMembers  int `json:"inheritedClrMembers"`
	InheritedProjections int `json:"inheritedProjectionsPerConsumer"`
	// ExcludedMembers is how many base members are deliberately unprojected.
	ExcludedMembers int `json:"excludedMembers"`
	// Consumers is every XNA type that inherits this base.
	Consumers []bclBaseAdapterConsumer `json:"consumers"`
	// Inventory is the exact per-member attribution.
	Inventory  []bclInheritedMemberMeasurement `json:"inheritedMemberInventory"`
	Exclusions []bclInheritedExclusion         `json:"exclusions"`
	Rationale  string                          `json:"rationale"`
	Verdict    string                          `json:"verdict"`
}

type bclBaseAdapterConsumer struct {
	XNA string `json:"xna"`
	Go  string `json:"go"`
	// BaseArguments is the CLR generic arguments this consumer supplies.
	BaseArguments []string `json:"baseArguments"`
	// Projected is whether the consumer is present in the Go surface.
	Projected bool `json:"projected"`
	// AdapterFieldType is the exact private field type the consumer must
	// declare, and AdapterFieldPresent whether it does.
	AdapterFieldType    string `json:"adapterFieldType"`
	AdapterFieldPresent bool   `json:"adapterFieldPresent"`
	// ExportedEmbeddings must stay zero: the base is never Go embedding.
	ExportedEmbeddings int `json:"exportedEmbeddings"`
	// DeclaredMembers and InheritedMembers are this consumer's two provenance
	// classes, and their projections never overlap.
	DeclaredMembers      int    `json:"xnaDeclaredMembers"`
	DeclaredProjections  int    `json:"xnaDeclaredProjections"`
	InheritedProjections int    `json:"bclInheritedProjections"`
	Verdict              string `json:"verdict"`
}

// bclInheritedMemberMeasurement is one row of the attribution table: exact BCL
// base, exact CLR member, exact projected Go member.
type bclInheritedMemberMeasurement struct {
	CLRBase   string `json:"clrBase"`
	CLRMember string `json:"clrMember"`
	CLRKind   string `json:"clrKind"`
	Consumer  string `json:"consumer"`
	GoMember  string `json:"goMember"`
	GoResults string `json:"goResults"`
	Accessor  string `json:"accessor,omitempty"`
	Present   bool   `json:"present"`
	Rationale string `json:"rationale"`
}

type bclInheritedExclusion struct {
	CLRMember string `json:"clrMember"`
	Reason    string `json:"reason"`
}

// bclInterfaceProjection measures one non-XNA CLR interface that XNA types
// declare.
//
// The measured claim is ProjectedMembers == 0: the interface itself contributes
// no Go member identity. Its CLR members reach the Go surface only when the XNA
// type declares them publicly in its own right, in which case they are ordinary
// members of that type, or not at all, when the type implements the interface
// explicitly.
type bclInterfaceProjection struct {
	CLRInterface string `json:"clrInterface"`
	Status       string `json:"status"`
	// CLRMembers is how many members the interface declares.
	CLRMembers int `json:"clrMembers"`
	// ProjectedMembers is how many Go identities the interface itself
	// contributes. It must stay zero.
	ProjectedMembers int `json:"projectedMembers"`
	// DeclaringTypes is every XNA type in the profile that lists it directly.
	DeclaringTypes int `json:"declaringTypes"`
	// ProjectedTypes is how many of those are present in the Go surface.
	ProjectedTypes int    `json:"projectedTypes"`
	Rationale      string `json:"rationale"`
	Verdict        string `json:"verdict"`
}

// enumClosure is the reusable ordinary/flags enum closure measurement. The
// Foundation-14 pure-managed batch completed 25 enums that differ only in
// their pinned literal table, so they are measured by one table-driven
// category rather than 25 near-identical bespoke closures.
type enumClosure struct {
	XNA                  string                 `json:"xna"`
	GoName               string                 `json:"goName"`
	PackagePath          string                 `json:"packagePath"`
	SourceTypes          int                    `json:"sourceTypes"`
	SourceIdentities     int                    `json:"sourceIdentities"`
	ExpectedGoIdentities int                    `json:"expectedGoIdentities"`
	TargetTypes          int                    `json:"targetTypes"`
	TargetGoIdentities   int                    `json:"targetGoIdentities"`
	LocalDiagnostics     int                    `json:"localDiagnostics"`
	ExpectedKind         string                 `json:"expectedKind"`
	ActualKind           string                 `json:"actualKind"`
	UnderlyingType       string                 `json:"underlyingType"`
	ExpectedFlags        bool                   `json:"expectedFlags"`
	Flags                bool                   `json:"flags"`
	ValueStorageExcluded bool                   `json:"valueStorageExcluded"`
	Values               []enumValueMeasurement `json:"values"`
	Status               string                 `json:"status"`
}

// valueStructClosure measures one pure managed XNA value struct. Beyond the
// identity arithmetic it records the central semantic claim of the family:
// every member is infallible managed value work, so no member may carry a
// synthetic Go error result.
type valueStructClosure struct {
	XNA                  string              `json:"xna"`
	GoName               string              `json:"goName"`
	PackagePath          string              `json:"packagePath"`
	SourceTypes          int                 `json:"sourceTypes"`
	SourceIdentities     int                 `json:"sourceIdentities"`
	ExpectedGoIdentities int                 `json:"expectedGoIdentities"`
	TargetTypes          int                 `json:"targetTypes"`
	TargetGoIdentities   int                 `json:"targetGoIdentities"`
	LocalDiagnostics     int                 `json:"localDiagnostics"`
	ExpectedKind         string              `json:"expectedKind"`
	ActualKind           string              `json:"actualKind"`
	BaseType             string              `json:"baseType"`
	ErrorResults         int                 `json:"errorResults"`
	Members              []valueStructMember `json:"members"`
	Status               string              `json:"status"`
}

// managedInterfaceClosure measures one projected CLR interface contract. The
// claim it records is that interface kind alone decides nothing: each
// operation's fallibility comes from the reference implementor IL, so one
// contract may legitimately mix infallible managed accessors with operations
// that cross a qualified runtime boundary.
type managedInterfaceClosure struct {
	XNA                  string                   `json:"xna"`
	GoName               string                   `json:"goName"`
	PackagePath          string                   `json:"packagePath"`
	SourceTypes          int                      `json:"sourceTypes"`
	SourceIdentities     int                      `json:"sourceIdentities"`
	ExpectedGoIdentities int                      `json:"expectedGoIdentities"`
	TargetTypes          int                      `json:"targetTypes"`
	TargetGoIdentities   int                      `json:"targetGoIdentities"`
	LocalDiagnostics     int                      `json:"localDiagnostics"`
	ExpectedKind         string                   `json:"expectedKind"`
	ActualKind           string                   `json:"actualKind"`
	Classified           bool                     `json:"classified"`
	Boundary             string                   `json:"boundary"`
	AccessorPairs        int                      `json:"accessorPairs"`
	FallibleGetters      int                      `json:"fallibleGetters"`
	FallibleSetters      int                      `json:"fallibleSetters"`
	FallibleOperations   int                      `json:"fallibleOperations"`
	EventAccessors       int                      `json:"eventAccessors"`
	ErrorResults         int                      `json:"errorResults"`
	Members              []managedInterfaceMember `json:"members"`
	Status               string                   `json:"status"`
}

// managedInterfaceMember is one projected operation of a CLR interface.
type managedInterfaceMember struct {
	XNA              string   `json:"xna"`
	SourceKind       string   `json:"sourceKind"`
	Accessor         string   `json:"accessor,omitempty"`
	Name             string   `json:"name"`
	ExpectedFallible bool     `json:"expectedFallible"`
	ActualFallible   bool     `json:"actualFallible"`
	ExpectedResults  []string `json:"expectedResults"`
	ActualResults    []string `json:"actualResults"`
	Status           string   `json:"status"`
}

// managedTypeClosure measures one pure-managed CLR class: a type whose CLR
// kind is `class`, so it keeps reference semantics and projects as a Go
// pointer facade, but whose authoritative IL proves the selected public
// behavior is entirely managed. Beyond the identity arithmetic it records the
// two claims that separate this family from a native-backed facade and from a
// value struct:
//
//   - reference semantics: the constructor returns *T, not T, so two variables
//     referencing one instance observe the same mutations;
//   - per-operation fallibility: an error result belongs to a single projected
//     operation, so one property's setter may carry an error while its own
//     getter does not.
type managedTypeClosure struct {
	XNA                  string `json:"xna"`
	SourceKind           string `json:"sourceKind"`
	ValueSemantics       bool   `json:"valueSemantics"`
	GoName               string `json:"goName"`
	PackagePath          string `json:"packagePath"`
	SourceTypes          int    `json:"sourceTypes"`
	SourceIdentities     int    `json:"sourceIdentities"`
	ExpectedGoIdentities int    `json:"expectedGoIdentities"`
	TargetTypes          int    `json:"targetTypes"`
	TargetGoIdentities   int    `json:"targetGoIdentities"`
	LocalDiagnostics     int    `json:"localDiagnostics"`
	ExpectedKind         string `json:"expectedKind"`
	ActualKind           string `json:"actualKind"`
	BaseType             string `json:"baseType"`
	// BaseRelationship is DIRECT when the CLR base is the kind's own root, the
	// adapter name when a declared MAPPED BCL base carries it, and UNDECIDED
	// when the base is not a relationship that permits projection.
	BaseRelationship    string              `json:"baseRelationship"`
	PureManaged         bool                `json:"pureManaged"`
	ReferenceProjection string              `json:"referenceProjection"`
	AccessorPairs       int                 `json:"accessorPairs"`
	FallibleGetters     int                 `json:"fallibleGetters"`
	FallibleSetters     int                 `json:"fallibleSetters"`
	FallibleOperations  int                 `json:"fallibleOperations"`
	ErrorResults        int                 `json:"errorResults"`
	Members             []managedTypeMember `json:"members"`
	Status              string              `json:"status"`
}

// managedTypeMember is one projected operation of a pure-managed CLR class.
// A CLR property contributes two of these rows, one per accessor, each with
// its own fallibility verdict.
type managedTypeMember struct {
	XNA              string   `json:"xna"`
	SourceKind       string   `json:"sourceKind"`
	Accessor         string   `json:"accessor,omitempty"`
	GoKind           string   `json:"goKind"`
	Receiver         string   `json:"receiver,omitempty"`
	Name             string   `json:"name"`
	ExpectedFallible bool     `json:"expectedFallible"`
	ActualFallible   bool     `json:"actualFallible"`
	ExpectedResults  []string `json:"expectedResults"`
	ActualResults    []string `json:"actualResults"`
	Status           string   `json:"status"`
}

type valueStructMember struct {
	GoKind          string   `json:"goKind"`
	Receiver        string   `json:"receiver"`
	Name            string   `json:"name"`
	ExpectedResults []string `json:"expectedResults"`
	ActualResults   []string `json:"actualResults"`
	Status          string   `json:"status"`
}

type buttonStateClosure struct {
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
