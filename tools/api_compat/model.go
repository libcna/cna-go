package main

import "encoding/json"

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
	Types             map[symbolKey]*expectedType
	Members           map[symbolKey]*expectedMember
	ReferenceTypes    int
	ReferenceMembers  int
	ExpectedGoTypes   int
	ExpectedGoMembers int
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
	GenericParameter []string
	Members          []symbolKey
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
	SchemaVersion int            `json:"schemaVersion"`
	Profile       string         `json:"profile"`
	Mode          string         `json:"mode"`
	Summary       map[string]int `json:"summary"`
	Diagnostics   []diagnostic   `json:"diagnostics"`
	CompleteTypes []string       `json:"completeTypes"`
	PartialTypes  []typeStatus   `json:"partialTypes"`
	MissingTypes  []string       `json:"missingTypes"`
	Metadata      reportMetadata `json:"metadata"`
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
