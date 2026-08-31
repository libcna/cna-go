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
	// PublicCLRMembers is how many PUBLIC CLR members this type declares:
	// what a derived type inherits into its own public surface. Constructors
	// are excluded because they are not inherited. It is carried so the XNA
	// base frontier can state the size of a deferred base's contribution
	// instead of only naming it.
	PublicCLRMembers int
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
	// SourceAccess is the CLR accessibility the pinned contract declares for
	// this member -- "public", "protected", and so on. It is carried so a
	// claim about accessibility can be MEASURED against the contract rather
	// than asserted in prose: the Game base-call adapters, for instance, may
	// only exist for a member the contract declares protected.
	SourceAccess string
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
	SchemaVersion                int                              `json:"schemaVersion"`
	Profile                      string                           `json:"profile"`
	Mode                         string                           `json:"mode"`
	Summary                      map[string]int                   `json:"summary"`
	Diagnostics                  []diagnostic                     `json:"diagnostics"`
	CompleteTypes                []string                         `json:"completeTypes"`
	PartialTypes                 []typeStatus                     `json:"partialTypes"`
	MissingTypes                 []string                         `json:"missingTypes"`
	InterfaceWitnessProjections  []interfaceWitnessProjection     `json:"interfaceWitnessProjections,omitempty"`
	PackedInterfaceConformance   []packedInterfaceConformance     `json:"packedInterfaceConformance,omitempty"`
	PackedVectorTypeMeasurements []packedVectorTypeMeasurement    `json:"packedVectorTypeMeasurements,omitempty"`
	VertexElementClosure         vertexElementClosure             `json:"vertexElementClosure"`
	PlayerIndexKeyboardClosure   playerIndexKeyboardClosure       `json:"playerIndexKeyboardClosure"`
	DisplayOrientationClosure    displayOrientationClosure        `json:"displayOrientationGraphicsManagerClosure"`
	BufferUsageClosure           bufferUsageClosure               `json:"bufferUsageClosure"`
	ClearOptionsClosure          clearOptionsClosure              `json:"clearOptionsClosure"`
	SurfaceFormatClosure         surfaceFormatClosure             `json:"surfaceFormatClosure"`
	DepthFormatClosure           depthFormatClosure               `json:"depthFormatClosure"`
	GraphicsProfileClosure       graphicsProfileClosure           `json:"graphicsProfileClosure"`
	ButtonStateClosure           buttonStateClosure               `json:"buttonStateClosure"`
	Foundation14EnumClosures     []enumClosure                    `json:"foundation14EnumClosures"`
	Foundation15EnumClosures     []enumClosure                    `json:"foundation15EnumClosures"`
	Foundation15ValueStructs     []valueStructClosure             `json:"foundation15ValueStructClosures"`
	Foundation16ValueStructs     []valueStructClosure             `json:"foundation16ValueStructClosures"`
	Foundation17ManagedClasses   []managedTypeClosure             `json:"foundation17ManagedClassClosures"`
	Foundation18Interfaces       []managedInterfaceClosure        `json:"foundation18InterfaceClosures"`
	Foundation19ManagedClasses   []managedTypeClosure             `json:"foundation19ManagedClassClosures"`
	Foundation20ValueContracts   []managedTypeClosure             `json:"foundation20ValueContractClosures"`
	Foundation21ManagedClasses   []managedTypeClosure             `json:"foundation21ManagedClassClosures"`
	Foundation23Interfaces       []managedInterfaceClosure        `json:"foundation23InterfaceClosures"`
	Foundation23ManagedClasses   []managedTypeClosure             `json:"foundation23ManagedClassClosures"`
	BCLBaseRelationships         []bclBaseProjection              `json:"bclBaseRelationships"`
	BCLBaseAdapters              []bclBaseAdapterMeasurement      `json:"bclBaseAdapters"`
	BCLSignatureAdapters         []bclSignatureAdapterMeasurement `json:"bclSignatureAdapters"`
	BCLInterfaceRelationships    []bclInterfaceProjection         `json:"bclInterfaceRelationships"`
	GameBaseCallAdapters         []gameBaseCallMeasurement        `json:"gameBaseCallAdapters"`
	DeclaredInterfaceConformance []declaredInterfaceConformance   `json:"declaredInterfaceConformance"`
	XNABaseRelationships         []xnaBaseProjection              `json:"xnaBaseRelationships"`
	GameNativeSignals            []gameNativeSignalMeasurement    `json:"gameNativeSignals"`
	GameFrameHooks               []gameFrameHookMeasurement       `json:"gameFrameHooks"`
	Metadata                     reportMetadata                   `json:"metadata"`
}

// gameNativeSignalMeasurement records one canonical CNA game signal bound to one
// CLR event Game declares: the projected accessor pair, the reference raise
// path, the sender that path pushes, and the runtime evidence class.
//
// Like the base-call adapters it adds no XNA identity. The accessors it names
// are already counted as the event's two projected members; this measurement
// only proves the binding around them is the one the reference describes.
type gameNativeSignalMeasurement struct {
	// CNAConstant and CNAIdentity name the canonical signal. The C-side chain
	// that proves those values is measured by tools/native_abi, not here.
	CNAConstant string `json:"cnaConstant"`
	CNAIdentity int    `json:"cnaIdentity"`
	// CLREvent is the event Game declares; AddAccessor and RemoveAccessor are
	// the two Go members the settled event mapping projects it to.
	CLREvent       string `json:"clrEvent"`
	AddAccessor    string `json:"addAccessor"`
	RemoveAccessor string `json:"removeAccessor"`
	// RaiseSite is the projected protected virtual the raise routes through,
	// and is empty for the one event whose reference has none. RaiseSiteAccess
	// is read from the pinned contract so a claim of "protected" is measured.
	RaiseSite       string `json:"raiseSite"`
	RaiseSiteAccess string `json:"raiseSiteAccess"`
	// RaisePath is NATIVE_HOST_SIGNAL or MANAGED, NativeSignalRole is
	// PUBLIC_EVENT_RAISE or LIFECYCLE_ONLY, and ManagedRaiseSite names the
	// projected member that raises a MANAGED event. NativeSignalMoment is what
	// the CNA signal actually means, which is how a LIFECYCLE_ONLY role states
	// why the semantics do not align.
	RaisePath          string   `json:"raisePath"`
	NativeSignalRole   string   `json:"nativeSignalRole"`
	ManagedRaiseSite   string   `json:"managedRaiseSite,omitempty"`
	NativeSignalMoment string   `json:"nativeSignalMoment"`
	Sender             string   `json:"sender"`
	EdgeTriggered      bool     `json:"edgeTriggered"`
	ReferencePath      []string `json:"referencePath"`
	RuntimeEvidence    string   `json:"runtimeEvidence"`
	EvidenceReason     string   `json:"evidenceReason,omitempty"`
	Verdict            string   `json:"verdict"`
}

// gameFrameHookMeasurement records one of Game's frame-boundary protected
// virtuals, the canonical CNA hook at the same position, and the measured fact
// that CNA-Go does not install it.
type gameFrameHookMeasurement struct {
	CLRMember string `json:"clrMember"`
	CLRAccess string `json:"clrAccess"`
	// GoMember is the fully qualified projected method. It is a method on Game
	// rather than a GameCallbacks member or a GameBase... helper, and the
	// verifier proves all three.
	GoMember   string   `json:"goMember"`
	Parameters []string `json:"parameters"`
	Results    []string `json:"results"`
	// NativeHook is the canonical hook at the same frame position, Installation
	// records WHEN CNA-Go installs it -- NEVER, ON_OVERRIDE or ALWAYS -- and
	// ReasonUninstalled is carried only by a hook that is never installed.
	NativeHook        string `json:"nativeHook"`
	Installation      string `json:"installation"`
	ReasonUninstalled string `json:"reasonUninstalled,omitempty"`
	// Capability is the unexported single-method Go interface an external
	// callback object satisfies structurally to override this hook, and the
	// three fields after it are the exact method it has to declare.
	Capability           string                    `json:"capability,omitempty"`
	CapabilityMethod     string                    `json:"capabilityMethod,omitempty"`
	CapabilityParameters []string                  `json:"capabilityParameters,omitempty"`
	CapabilityResults    []string                  `json:"capabilityResults,omitempty"`
	NativeOrdering       string                    `json:"nativeOrdering"`
	ReferenceBody        []string                  `json:"referenceBody"`
	Deferred             []gameBaseCallDeferralRow `json:"deferredSteps"`
	Verdict              string                    `json:"verdict"`
}

// gameBaseCallMeasurement records one Game base-call language adapter: the
// protected virtual it runs the base body of, the exact Go function that does
// it, and every reference step the projection does not reproduce.
//
// It is a LANGUAGE-SUPPORT measurement, deliberately kept out of the XNA
// identity accounting. The adapters add nothing to REFERENCE_MEMBERS, nothing
// to EXPECTED_GO_MEMBERS, and nothing to any type's projected member set: they
// are the Go spelling of a CLR `base.X(...)` call site, which is syntax rather
// than surface.
type gameBaseCallMeasurement struct {
	// CLRMember is the protected virtual whose base body the function runs.
	CLRMember string `json:"clrMember"`
	// CLRAccess and CLRVirtual are read from the pinned contract, so an
	// adapter cannot be declared for a member that is not a protected virtual.
	CLRAccess string `json:"clrAccess"`
	// CallbackMember is the GameCallbacks member that projects the override.
	// Every adapter has exactly one and every callback member has exactly one
	// adapter.
	CallbackMember string `json:"callbackMember"`
	// GoFunction is the fully qualified package-level Go function.
	GoFunction string   `json:"goFunction"`
	Parameters []string `json:"parameters"`
	Results    []string `json:"results"`
	// Fallibility names every reason the function can report an error.
	Fallibility []gameBaseCallFallibilityRow `json:"fallibility"`
	// ReferenceBody is the reference base body, in order.
	ReferenceBody []string `json:"referenceBody"`
	// Deferred is every reference step not reproduced, each with a class and a
	// reason. A deferral that is observable from the managed component surface
	// is rejected rather than recorded.
	Deferred []gameBaseCallDeferralRow `json:"deferredSteps"`
	Verdict  string                    `json:"verdict"`
}

// xnaBaseProjection measures one CLR class in the pinned profile that another
// class in the same profile inherits from.
//
// It is the second base frontier. bclBaseProjection covers a base OUTSIDE the
// contract; this covers one INSIDE it, whose public surface a derived type
// inherits without the contract redeclaring it. Recording every one with a
// status is what keeps an unprojected inherited surface from being invisible.
type xnaBaseProjection struct {
	CLRBase string `json:"clrBase"`
	// Status is COMPOSED or DEFERRED.
	Status string `json:"status"`
	// Derived is every type in the profile that inherits from it.
	Derived []string `json:"derived"`
	// InheritedPublicMembers is how many public CLR members the base declares,
	// which is what each derived type inherits and, while DEFERRED, does not
	// get.
	InheritedPublicMembers int                 `json:"inheritedPublicMembers"`
	Blockers               []xnaBaseBlockerRow `json:"blockers,omitempty"`
	Verdict                string              `json:"verdict"`
}

type xnaBaseBlockerRow struct {
	Class  string `json:"class"`
	Detail string `json:"detail"`
}

// declaredInterfaceConformance records one compiler-checked claim: a complete
// projected class satisfies the Go projection of an XNA interface its CLR
// metadata declares.
type declaredInterfaceConformance struct {
	Owner        string `json:"owner"`
	GoOwner      string `json:"goOwner"`
	CLRInterface string `json:"clrInterface"`
	GoInterface  string `json:"goInterface"`
	// PointerSatisfies is go/types' answer for the pointer method set, which
	// is the method set of the facade CNA-Go projects a CLR class as.
	PointerSatisfies bool   `json:"pointerSatisfies"`
	Verdict          string `json:"verdict"`
}

type gameBaseCallFallibilityRow struct {
	Kind   string `json:"kind"`
	Reason string `json:"reason"`
}

type gameBaseCallDeferralRow struct {
	Step       string `json:"step"`
	Class      string `json:"class"`
	Reason     string `json:"reason"`
	Observable bool   `json:"observable"`
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
	ExportedEmbeddings int `json:"exportedEmbeddings"`
	// Blockers is why a DEFERRED base is deferred, named to the exact
	// inherited member or the exact architecture decision, so "deferred" is a
	// measured claim rather than a word.
	Blockers  []bclBaseBlockerMeasurement `json:"blockers,omitempty"`
	Rationale string                      `json:"rationale"`
	Verdict   string                      `json:"verdict"`
}

type bclBaseBlockerMeasurement struct {
	Kind      string `json:"kind"`
	CLRMember string `json:"clrMember,omitempty"`
	Needs     string `json:"needs"`
	Detail    string `json:"detail"`
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

// bclSignatureAdapterMeasurement measures one BCL type the pinned contract
// carries at a public signature position, and pins the exported Go surface of
// its adapter to the exact public CLR member inventory.
//
// Without it an adapter type would be a hole in the unexpected-member scan,
// because every exported member on an adapter receiver is admitted.
type bclSignatureAdapterMeasurement struct {
	CLRType         string `json:"clrType"`
	GoAdapter       string `json:"goAdapter"`
	BehaviorLevel   string `json:"behaviorLevel"`
	Authority       string `json:"authority"`
	AuthoritySHA256 string `json:"authoritySha256"`
	// CLRMembers is the public member inventory and GoMembers the exported Go
	// members that must reproduce it, one for one.
	CLRMembers int `json:"clrPublicMembers"`
	GoMembers  int `json:"goMembers"`
	// ExcludedMembers is how many CLR members are deliberately unprojected.
	ExcludedMembers int `json:"excludedMembers"`
	// SignaturePositions is how many projected XNA members carry this type.
	SignaturePositions int `json:"signaturePositions"`
	// Carriers is the XNA members that carry it.
	Carriers   []string                        `json:"carriers"`
	Inventory  []bclInheritedMemberMeasurement `json:"memberInventory"`
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
	// CLRType is set for a signature adapter and CLRBase for a base adapter,
	// so one row always names which role it belongs to.
	CLRType   string `json:"clrType,omitempty"`
	CLRBase   string `json:"clrBase,omitempty"`
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
