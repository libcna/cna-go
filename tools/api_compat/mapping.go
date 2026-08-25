package main

import (
	"fmt"
	"hash/fnv"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const modulePath = "github.com/openeggbert/cna-go"

var bclTypes = map[string]string{
	"System.Boolean":   "bool",
	"System.Byte":      "uint8",
	"System.SByte":     "int8",
	"System.Int16":     "int16",
	"System.UInt16":    "uint16",
	"System.Int32":     "int32",
	"System.UInt32":    "uint32",
	"System.Int64":     "int64",
	"System.UInt64":    "uint64",
	"System.Single":    "float32",
	"System.Double":    "float64",
	"System.Char":      "uint16",
	"System.String":    "string",
	"System.Object":    "any",
	"System.IntPtr":    "uintptr",
	"System.TimeSpan":  "TimeSpan",
	"System.Type":      "reflect.Type",
	"System.IO.Stream": "io.Reader",
}

var operatorNames = map[string]string{
	"op_Addition":      "Addition",
	"op_Subtraction":   "Subtraction",
	"op_Multiply":      "Multiply",
	"op_Division":      "Division",
	"op_UnaryNegation": "UnaryNegation",
	"op_Equality":      "Equality",
	"op_Inequality":    "Inequality",
	"op_Implicit":      "Implicit",
	"op_Explicit":      "Explicit",
}

// pureManagedTypes is the explicit pure-managed CLR type classification. It is
// the general class/struct-classification boundary of the binding, not a list
// of value structs: CLR `class` alone is never evidence of native backing.
//
// A CLR type is admitted here only when authoritative Microsoft XNA IL proves
// that its selected public behavior is backed entirely by managed fields and
// deterministic managed code, and therefore owns no CNA native object and
// needs no FFI, no native allocation, no renderer/device query, no native
// destruction, no callback registration, no thread-affinity lifecycle, and no
// external hardware state.
//
// Admission does not change CLR reference semantics: an admitted `class` still
// projects as a Go pointer facade, so two variables that reference the same
// instance observe the same mutations. It only removes the synthetic native
// runtime `error` that a native-backed facade would carry.
//
// Genuinely native-backed classes -- Game, GraphicsDeviceManager,
// GraphicsDevice, SpriteBatch, Texture2D -- are deliberately absent and keep
// their fallible native facade behavior.
var pureManagedTypes = map[string]bool{
	"Microsoft.Xna.Framework.MathHelper":         true,
	"Microsoft.Xna.Framework.Vector2":            true,
	"Microsoft.Xna.Framework.Vector3":            true,
	"Microsoft.Xna.Framework.Vector4":            true,
	"Microsoft.Xna.Framework.Quaternion":         true,
	"Microsoft.Xna.Framework.Matrix":             true,
	"Microsoft.Xna.Framework.Color":              true,
	"Microsoft.Xna.Framework.Point":              true,
	"Microsoft.Xna.Framework.Rectangle":          true,
	"Microsoft.Xna.Framework.Ray":                true,
	"Microsoft.Xna.Framework.Plane":              true,
	"Microsoft.Xna.Framework.BoundingBox":        true,
	"Microsoft.Xna.Framework.BoundingSphere":     true,
	"Microsoft.Xna.Framework.BoundingFrustum":    true,
	"Microsoft.Xna.Framework.GameTime":           true,
	"Microsoft.Xna.Framework.Curve":              true,
	"Microsoft.Xna.Framework.CurveKey":           true,
	"Microsoft.Xna.Framework.CurveKeyCollection": true,
	"Microsoft.Xna.Framework.CurveContinuity":    true,
	"Microsoft.Xna.Framework.CurveLoopType":      true,
	"Microsoft.Xna.Framework.CurveTangent":       true,

	// Foundation 17. Microsoft.Xna.Framework.dll IL
	// (sha256 38e7093f52d7474bbc6256906519781a1210d7da50a1c667b52716fcf49ca130)
	// shows both audio positional descriptors as plain managed field storage
	// over an assembly-private XACT_LISTENER_DATA/XACT_EMITTER_DATA value:
	// every public accessor is one ldfld/stfld plus the managed, side-effect
	// free UnsafeNativeStructures::FlipHandedness. No public member reaches
	// XACT, a device, or any native allocation.
	"Microsoft.Xna.Framework.Audio.AudioListener": true,
	"Microsoft.Xna.Framework.Audio.AudioEmitter":  true,

	// Foundation 19. Microsoft.Xna.Framework.Graphics.dll IL
	// (sha256 560080fc39021c611ca9d076dcebed312faf6d7d1413c2dc523683ea635e9f55)
	// shows PresentationParameters as a descriptor over one assembly-visible
	// nested `Settings` value struct: every accessor is one ldflda plus one
	// ldfld or stfld, Bounds is computed from two stored extents, and Clone
	// copies the whole value struct. It stores a platform window handle but
	// never creates, resets, presents, enumerates, or looks anything up.
	"Microsoft.Xna.Framework.Graphics.PresentationParameters": true,

	// Foundation 20. Microsoft.Xna.Framework.Input.Touch.dll IL
	// (sha256 b0585224c18022c3661057ae79544644c10f33f1dc529678364f3d6b25151c25)
	// shows TouchCollection as a fixed eight-slot inline value struct over
	// TouchLocation with no array allocation and no device access, and its
	// nested Enumerator as a two-field cursor over a copy of it. Both are
	// System.ValueType rather than class, but the classification is what
	// makes their per-operation fallibility expressible: several of their
	// members throw from authoritative managed argument validation.
	"Microsoft.Xna.Framework.Input.Touch.TouchCollection":            true,
	"Microsoft.Xna.Framework.Input.Touch.TouchCollection+Enumerator": true,

	// Foundation 23. The three dependency-complete System.EventArgs carriers.
	// Microsoft.Xna.Framework.Game.dll shows GameComponentCollectionEventArgs
	// as one private IGameComponent field: the public constructor is
	// `call EventArgs::.ctor` plus one stfld with no validation, and the getter
	// is one ldfld. Microsoft.Xna.Framework.Graphics.dll shows both resource
	// carriers the same way over `assembly` fields, with `assembly`
	// constructors that are not part of the public contract. None of the three
	// reaches a device, an allocation, or any native code.
	"Microsoft.Xna.Framework.GameComponentCollectionEventArgs":    true,
	"Microsoft.Xna.Framework.Graphics.ResourceCreatedEventArgs":   true,
	"Microsoft.Xna.Framework.Graphics.ResourceDestroyedEventArgs": true,

	// Foundation 21. Microsoft.Xna.Framework.Game.dll IL
	// (sha256 b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0)
	// shows GameServiceContainer as one private Dictionary<Type, object> with
	// three methods that validate their arguments and then add, remove, or
	// look up an entry. It creates no device, starts no game, and reaches no
	// native code.
	"Microsoft.Xna.Framework.GameServiceContainer": true,
}

// bclBaseRelationship is how one non-XNA CLR base type is projected.
//
// Go has no CLR inheritance, and CNA-Go's established policy is that an
// exported embedded field must never be used to simulate one: embedding would
// promote the base's members into the derived type's Go method set, inventing
// surface the XNA contract never declared and making the derived type
// assignable in ways CLR inheritance does not imply. A CLR base therefore
// survives as this measured relationship rather than as Go structure, and the
// invariant every entry shares is that the base contributes no projected Go
// member identity of its own.
type bclBaseRelationship struct {
	// Adapter is the framework-package language adapter that models the base,
	// empty when the relationship needs none.
	Adapter string
	// Status is IMPLIED for the three universal CLR roots that no projection
	// ever names, MAPPED when a derived XNA type may be projected today, and
	// DEFERRED when the relationship is still an open public-API decision and
	// no derived type may be projected yet.
	Status string
	// Rationale records why the relationship is where it is.
	Rationale string
}

// bclBaseRelationships declares every non-XNA CLR base type that appears in
// the pinned profile. The table is exhaustive by construction: a base absent
// from it raises BASE_MAPPING_MISMATCH, so a base type can never be dropped
// silently, and DEFERRED means a derived type must stay unprojected rather
// than be projected under a base nobody has decided about.
var bclBaseRelationships = map[string]bclBaseRelationship{
	"System.Object": {
		Status:    "IMPLIED",
		Rationale: "the universal CLR root; a Go struct or interface names no base and inherits nothing",
	},
	"System.ValueType": {
		Status:    "IMPLIED",
		Rationale: "the CLR value root; the struct projection already carries value semantics",
	},
	"System.Enum": {
		Status:    "IMPLIED",
		Rationale: "the CLR enum root; the named-int32 projection already carries it",
	},
	"System.EventArgs": {
		Adapter:   "EventArgs",
		Status:    "MAPPED",
		Rationale: "modelled by the framework EventArgs language adapter; a derived XNA class keeps CLR reference semantics and projects as its own pointer type with no exported embedding",
	},
	"System.Exception": {
		Status:    "DEFERRED",
		Rationale: "CLR exception types as Go types is a separate public-API decision; CNA-Go reports failure through error results and has no exception hierarchy",
	},
	"System.Runtime.InteropServices.ExternalException": {
		Status:    "DEFERRED",
		Rationale: "derives from System.Exception and inherits the same open decision",
	},
	"System.Attribute": {
		Status:    "DEFERRED",
		Rationale: "Go has no attribute metadata; the content serializer attributes need a separate mapping",
	},
	"System.IO.BinaryReader": {
		Status:    "DEFERRED",
		Rationale: "requires a stream-reader base whose seek and encoding behavior is a separate mapping",
	},
	"System.ComponentModel.ExpandableObjectConverter": {
		Status:    "DEFERRED",
		Rationale: "part of the System.ComponentModel TypeConverter subsystem, which is a separate mapping",
	},
	"System.Collections.ObjectModel.Collection`1": {
		Status:    "DEFERRED",
		Rationale: "a mutable BCL collection base whose projected surface is a separate mapping",
	},
	"System.Collections.ObjectModel.ReadOnlyCollection`1": {
		Status:    "DEFERRED",
		Rationale: "a read-only BCL collection base whose projected surface is a separate mapping",
	},
	"System.Collections.Generic.Dictionary`2": {
		Status:    "DEFERRED",
		Rationale: "a BCL dictionary base whose projected surface is a separate mapping",
	},
}

// bclInterfaceRelationship is how one non-XNA CLR interface a projected XNA
// type declares is accounted for.
//
// The general rule every entry shares is that the interface itself contributes
// **no projected Go surface**. That is not an assumption: in the pinned profile
// each of these interfaces is satisfied one of two ways, and neither produces a
// new Go identity.
//
//   - The XNA type already declares the interface's members publicly, so the
//     concrete method set is the whole projection. IEquatable<T>'s Equals(T),
//     IComparable<T>'s CompareTo(T), IServiceProvider's GetService(Type), and
//     the collection interfaces all work this way.
//   - The type implements the interface explicitly, so the CLR member is
//     `private ... .override` and is not public surface at all. GraphicsDeviceManager
//     does this for System.IDisposable: its Dispose() is
//     `.method private hidebysig newslot virtual final instance void
//     System.IDisposable.Dispose()` with `.override [mscorlib]System.IDisposable::Dispose`,
//     so the public contract carries only Dispose(bool), and projecting a
//     Dispose() for it would invent surface the reference does not expose.
//
// Declaring an interface here therefore does not make any type safe or
// complete. It records that the BCL name is accounted for, so the dependency
// frontier no longer treats it as unmapped, and so a new BCL interface cannot
// arrive unnoticed.
type bclInterfaceRelationship struct {
	// Status is MAPPED_NO_SURFACE for every entry the profile currently needs.
	Status string
	// Members are the interface's CLR members, recorded so the no-surface claim
	// names what it is claiming about.
	Members []string
	// Rationale is why the interface adds nothing.
	Rationale string
}

// bclInterfaceRelationships declares every non-XNA interface any type in the
// pinned profile lists as a direct interface. It is exhaustive by construction:
// an undeclared one raises INTERFACE_MAPPING_MISMATCH.
var bclInterfaceRelationships = map[string]bclInterfaceRelationship{
	"System.IDisposable": {
		Status:  "MAPPED_NO_SURFACE",
		Members: []string{"Dispose"},
		Rationale: "adds no Go surface of its own: no Disposable interface, no Close alias, no io.Closer adaptation, and no finalizer. " +
			"Twenty-eight of the twenty-nine declaring types already declare Dispose publicly and it maps as an ordinary member; " +
			"GraphicsDeviceManager implements it explicitly, so its Dispose() is not public surface and nothing is projected for it. " +
			"Ownership and lifetime remain a per-type question that this relationship does not answer.",
	},
	"System.IEquatable`1": {
		Status:    "MAPPED_NO_SURFACE",
		Members:   []string{"Equals"},
		Rationale: "every declaring XNA type already declares the strongly typed Equals publicly, so the concrete method set is the whole projection",
	},
	"System.IComparable`1": {
		Status:    "MAPPED_NO_SURFACE",
		Members:   []string{"CompareTo"},
		Rationale: "CurveKey already declares CompareTo publicly",
	},
	"System.IServiceProvider": {
		Status:    "MAPPED_NO_SURFACE",
		Members:   []string{"GetService"},
		Rationale: "GameServiceContainer already declares GetService publicly; settled in Foundation 21",
	},
	"System.Collections.Generic.IEnumerable`1": {
		Status:    "MAPPED_NO_SURFACE",
		Members:   []string{"GetEnumerator"},
		Rationale: "projected by the declared collection rule as GetEnumerator, returning either the collection's own public enumerator type or the Iterator<T> adapter",
	},
	"System.Collections.Generic.IEnumerator`1": {
		Status:    "MAPPED_NO_SURFACE",
		Members:   []string{"Current", "MoveNext", "Reset", "Dispose"},
		Rationale: "projected by the declared collection rule as the Iterator<T> adapter or the collection's own enumerator members",
	},
	"System.Collections.Generic.ICollection`1": {
		Status:    "MAPPED_NO_SURFACE",
		Members:   []string{"Count", "IsReadOnly", "Add", "Clear", "Contains", "CopyTo", "Remove"},
		Rationale: "a concrete Go method set on the XNA collection, never a fake BCL package; settled in Foundation 20",
	},
	"System.Collections.Generic.IList`1": {
		Status:    "MAPPED_NO_SURFACE",
		Members:   []string{"Item", "IndexOf", "Insert", "RemoveAt"},
		Rationale: "its indexer and index methods are already public members of the declaring XNA collection; settled in Foundation 20",
	},
}

// internalXNAInterfaces are XNA-namespaced interfaces that public XNA types
// declare but that are not public surface themselves.
//
// Both are declared `.class interface private` in
// Microsoft.Xna.Framework.Graphics.dll, so they are assembly-visible only and
// correctly absent from the 257-type public contract. An internal interface
// cannot contribute public surface by definition, so the no-surface rule
// applies to them for a stronger reason than it applies to a BCL interface:
// there is no public member to project in the first place.
//
// They are declared rather than skipped so the dependency frontier does not
// count them as unmapped names and so a newly internal-implemented interface
// cannot appear unnoticed. Their members are recorded to name what the
// no-surface claim is about.
var internalXNAInterfaces = map[string]bclInterfaceRelationship{
	"Microsoft.Xna.Framework.Graphics.IGraphicsResource": {
		Status:  "INTERNAL_NO_SURFACE",
		Members: []string{"ReleaseNativeObject", "SaveDataForRecreation", "RecreateAndPopulateObject"},
		Rationale: "assembly-visible device-loss plumbing declared by seven public graphics types; " +
			"none of its three members appears in any public contract, and projecting one would expose CNA-internal recreation machinery",
	},
	"Microsoft.Xna.Framework.Graphics.IDynamicGraphicsResource": {
		Status:  "INTERNAL_NO_SURFACE",
		Members: []string{"ContentLost", "IsContentLost", "SetContentLost"},
		Rationale: "assembly-visible content-loss plumbing declared by four public graphics types; " +
			"the public ContentLost event those types expose is their own declared member, not this interface's",
	},
}

// inventedDisposalNames are Go identities a binding might synthesize from
// System.IDisposable and that the no-surface rule forbids. None of them is an
// XNA identity anywhere in the profile.
var inventedDisposalNames = map[string]string{
	"Close":        "Close alias invented from System.IDisposable",
	"Closer":       "io.Closer adaptation invented from System.IDisposable",
	"Disposable":   "Disposable interface invented from System.IDisposable",
	"IDisposable":  "IDisposable interface invented from System.IDisposable",
	"DisposeAll":   "ownership wrapper invented from System.IDisposable",
	"SetFinalizer": "finalizer surface invented from System.IDisposable",
}

// baseIdentityWithoutArguments strips generic arguments so a constructed base
// such as ReadOnlyCollection`1[ModelBone] is looked up by its open identity.
func baseIdentityWithoutArguments(raw string) string {
	if bracket := strings.Index(raw, "["); bracket >= 0 {
		return raw[:bracket]
	}
	return raw
}

// classifiedInterfaces is the explicit, reusable policy boundary for
// structural interfaces whose fallibility is decided per projected operation
// from authoritative evidence rather than by the interface-kind default.
//
// Interface ownership alone must never add a synthetic Go error result. An
// interface listed here starts from "no operation is fallible" and gains an
// error only where managedFallibleMembers records one, using the same
// accessor-level keys as any other owner. An interface that is *not* listed
// here keeps the native/runtime default in which every operation is fallible,
// which is correct for a contract whose whole purpose is to cross a qualified
// runtime boundary.
//
// The evidence for an entry is the reference implementor IL in the assembly
// that declares the interface, not a guess about what an unknown implementor
// might do. Where every shipped implementor agrees, that agreement is the
// contract's measured behavior.
var classifiedInterfaces = map[string]bool{
	"Microsoft.Xna.Framework.Graphics.PackedVector.IPackedVector":   true,
	"Microsoft.Xna.Framework.Graphics.PackedVector.IPackedVector`1": true,

	// Foundation 18. In Microsoft.Xna.Framework.Graphics.dll
	// (sha256 560080fc39021c611ca9d076dcebed312faf6d7d1413c2dc523683ea635e9f55)
	// all five shipped implementors -- AlphaTestEffect, BasicEffect,
	// DualTextureEffect, EnvironmentMapEffect, and SkinnedEffect -- back
	// World, View, and Projection with a managed field read/write plus a
	// managed dirty-flag OR, on both accessors, with no device access.
	"Microsoft.Xna.Framework.Graphics.IEffectMatrices": true,
	// The same five implementors back FogEnabled, FogStart, and FogEnd the
	// same managed way, but route FogColor through EffectParameter, which
	// calls unmanaged D3DX and throws on a failed HRESULT. That single
	// operation is therefore fallible and the other six are not; see
	// managedFallibleMembers.
	"Microsoft.Xna.Framework.Graphics.IEffectFog": true,

	// Foundation 23. Microsoft.Xna.Framework.Game.dll declares both contracts
	// and ships exactly one implementor of each: GameComponent for IUpdateable
	// and DrawableGameComponent for IDrawable. Every selected operation in that
	// IL is managed field work --
	//
	//	get_Enabled / get_UpdateOrder / get_Visible / get_DrawOrder
	//	                       one ldfld, `ret`
	//	Update(GameTime) / Draw(GameTime)
	//	                       a bare `ret`, code size 1
	//
	// -- so neither contract crosses a runtime boundary and neither gains a
	// synthetic error. DrawableGameComponent does reach the graphics runtime,
	// but through get_GraphicsDevice and Initialize, which belong to
	// IGameComponent and to the class itself, not to these two contracts. That
	// is why IGameComponent stays unclassified and fallible while these are
	// classified and infallible: the boundary is read per contract from its own
	// implementor IL.
	//
	// Their event accessors still carry an error, from the settled event
	// accessor projection rather than from this classification.
	"Microsoft.Xna.Framework.IUpdateable": true,
	"Microsoft.Xna.Framework.IDrawable":   true,
}

// managedFallibleMembers records, per pure-managed owner, exactly which
// projected operations carry a Go error result. Fallibility is a property of a
// single projected operation, never of a whole type: a CLR property may throw
// from its setter while its getter is one ldfld that cannot fail.
//
// Keys are produced by fallibilityKeys and are, from most to least specific:
//
//	constructor|.ctor          one constructor (overloads share the CLR name)
//	method|<Name>              one ordinary or static method
//	field|<Name>               one field projection
//	property-get|<Name>        that property's getter only
//	property-set|<Name>        that property's setter only
//	property|<Name>            both accessors of that property
//
// property|<Name> stays supported because some XNA properties genuinely throw
// from both accessors -- CurveKeyCollection's indexer validates its index on
// read and on write. Marking a property whose IL only validates on assignment
// with property|<Name> is a defect, not a shorthand: it would add an error
// result to a getter that cannot fail. The verifier measures both accessors
// independently so that substitution is rejected.
var managedFallibleMembers = map[string]map[string]bool{
	"Microsoft.Xna.Framework.Curve": {
		"method|ComputeTangent": true,
	},
	"Microsoft.Xna.Framework.CurveKey": {
		"method|CompareTo": true,
	},
	"Microsoft.Xna.Framework.CurveKeyCollection": {
		"method|Add":      true,
		"method|CopyTo":   true,
		"method|RemoveAt": true,
		"property|Item":   true,
	},
	// AudioEmitter::set_DopplerScale is the first measured accessor-level
	// case. Its IL guards the store with `ldarg.1; ldc.r4 0.0; bge.un.s`,
	// throwing System.ArgumentOutOfRangeException only when the branch is
	// not taken. get_DopplerScale is one ldfld and cannot fail.
	"Microsoft.Xna.Framework.Audio.AudioEmitter": {
		"property-set|DopplerScale": true,
	},
	// IEffectFog::FogColor is the first measured runtime-boundary operation
	// on an otherwise managed interface. Every shipped implementor reads and
	// writes it through EffectParameter::GetValueVector3/SetValue, which end
	// in `calli unmanaged stdcall` into ID3DXBaseEffect and throw
	// GraphicsHelpers::GetExceptionFromResult on a negative HRESULT. Both
	// accessors cross the boundary, so both are fallible -- unlike
	// AudioEmitter::DopplerScale, where only one accessor is.
	"Microsoft.Xna.Framework.Graphics.IEffectFog": {
		"property-get|FogColor": true,
		"property-set|FogColor": true,
	},
	// TouchCollection is a read-only view: every IList<T> mutator is an
	// unconditional `newobj NotSupportedException; throw`, and the indexer
	// setter is one too. The indexer getter and the constructor validate
	// their arguments, and CopyTo validates three things. Its remaining
	// members -- IsConnected, Count, IsReadOnly, Contains, IndexOf, FindById,
	// and GetEnumerator -- index only inside the measured range and cannot
	// fail.
	"Microsoft.Xna.Framework.Input.Touch.TouchCollection": {
		"constructor|.ctor": true,
		"property-get|Item": true,
		"property-set|Item": true,
		"method|Add":        true,
		"method|Clear":      true,
		"method|Insert":     true,
		"method|Remove":     true,
		"method|RemoveAt":   true,
		"method|CopyTo":     true,
	},
	// Enumerator::get_Current forwards to TouchCollection::get_Item, so it
	// throws before the first MoveNext and after the cursor is exhausted.
	// MoveNext is pure arithmetic and Dispose is a bare `ret`.
	"Microsoft.Xna.Framework.Input.Touch.TouchCollection+Enumerator": {
		"property-get|Current": true,
	},
	// All three GameServiceContainer methods validate and throw; the
	// constructor only allocates the backing dictionary and cannot fail.
	"Microsoft.Xna.Framework.GameServiceContainer": {
		"method|AddService":    true,
		"method|RemoveService": true,
		"method|GetService":    true,
	},
}

// fallibilityKeys returns the managedFallibleMembers keys that can mark one
// projected operation fallible, most specific first. accessor is "get" or
// "set" for a projected property accessor and empty for every other member
// kind, so an accessor-level key wins over the whole-property key while the
// whole-property key still marks both accessors.
func fallibilityKeys(m contractMember, accessor string) []string {
	if m.Kind == "property" && accessor != "" {
		return []string{"property-" + accessor + "|" + m.Name, "property|" + m.Name}
	}
	return []string{m.Kind + "|" + m.Name}
}

// managedStoredMembers identifies members on otherwise native-backed class
// facades whose reference implementation is only managed field access. These
// members must not gain a synthetic runtime error result.
var managedStoredMembers = map[string]map[string]bool{
	"Microsoft.Xna.Framework.GraphicsDeviceManager": {
		"property|SupportedOrientations": true,
	},
}

func buildExpected(c contract) (*expectedSurface, error) {
	s := &expectedSurface{
		Types:              make(map[symbolKey]*expectedType),
		Members:            make(map[symbolKey]*expectedMember),
		InterfaceWitnesses: make(map[symbolKey]*expectedInterfaceWitness),
		ReferenceTypes:     len(c.Types),
	}
	byIdentity := make(map[string]*contractType, len(c.Types))
	nameCollisions := make(map[string]int)
	for i := range c.Types {
		t := &c.Types[i]
		byIdentity[t.Name] = t
		nameCollisions[namespaceOf(t.Name)+"|"+flattenedBaseName(t.Name)]++
		s.ReferenceMembers += len(t.Members)
	}

	for i := range c.Types {
		t := &c.Types[i]
		goName := mappedTypeName(*t, nameCollisions)
		pkg := packagePathForNamespace(namespaceOf(t.Name))
		key := symbolKey{Package: pkg, Name: goName}
		if _, exists := s.Types[key]; exists {
			return nil, fmt.Errorf("mapped type collision at %s", key.String())
		}
		genericNames := make([]string, len(t.GenericParameters))
		for j, gp := range t.GenericParameters {
			genericNames[j] = gp.Name
		}
		s.Types[key] = &expectedType{
			Key: key, XNA: t.Name, GoName: goName, PackagePath: pkg,
			Kind: t.Kind, Flags: t.Flags, BaseType: valueOrEmpty(t.BaseType),
			Interfaces:       append([]string(nil), t.DirectInterfaces...),
			AllInterfaces:    append([]string(nil), t.Interfaces...),
			GenericParameter: genericNames,
			SourceMembers:    len(t.Members),
		}
	}

	var allMembers []*expectedMember
	for i := range c.Types {
		t := &c.Types[i]
		owner := s.typeForXNA(t.Name)
		groups := overloadGroups(*t)
		for j := range t.Members {
			m := &t.Members[j]
			mapped := mapMember(s, byIdentity, owner, *t, *m, groups)
			allMembers = append(allMembers, mapped...)
		}
	}
	resolveMemberCollisions(allMembers)
	for _, em := range allMembers {
		if _, exists := s.Members[em.Key]; exists {
			return nil, fmt.Errorf("unresolved mapped member collision at %s from %s", em.Key.String(), em.XNA)
		}
		s.Members[em.Key] = em
		owner := s.typeForXNA(em.Owner)
		owner.Members = append(owner.Members, em.Key)
	}
	buildMappedInterfacesAndWitnesses(s, byIdentity)
	s.ExpectedGoTypes = len(s.Types)
	s.ExpectedGoMembers = len(s.Members)
	return s, nil
}

func resolveMemberCollisions(members []*expectedMember) {
	groups := make(map[symbolKey][]*expectedMember)
	for _, member := range members {
		groups[member.Key] = append(groups[member.Key], member)
	}
	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		for _, member := range group {
			member.GoName += collisionKindSuffix(member)
			member.Key.Name = member.GoName
		}
	}
	groups = make(map[symbolKey][]*expectedMember)
	for _, member := range members {
		groups[member.Key] = append(groups[member.Key], member)
	}
	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		for _, member := range group {
			member.GoName += "Signature" + signatureDigest(member.XNA)
			member.Key.Name = member.GoName
		}
	}
}

func collisionKindSuffix(member *expectedMember) string {
	if strings.Contains(member.XNA, "::op_") {
		return "Operator"
	}
	switch member.SourceKind {
	case "constructor":
		return "Constructor"
	case "field":
		return "Field"
	case "property":
		return "Property"
	case "event":
		return "Event"
	default:
		return "Method"
	}
}

func signatureDigest(identity string) string {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(identity))
	return fmt.Sprintf("%08X", hash.Sum32())
}

func (s *expectedSurface) typeForXNA(identity string) *expectedType {
	for _, t := range s.Types {
		if t.XNA == identity {
			return t
		}
	}
	return nil
}

func overloadGroups(t contractType) map[string]int {
	result := make(map[string]int)
	constructors := 0
	for _, m := range t.Members {
		if m.Kind == "constructor" {
			constructors++
			continue
		}
		if m.Kind == "method" {
			result[fmt.Sprintf("%t|%s", m.Static, m.Name)]++
		}
	}
	result["constructors"] = constructors
	return result
}

func mapMember(s *expectedSurface, byIdentity map[string]*contractType, owner *expectedType, t contractType, m contractMember, groups map[string]int) []*expectedMember {
	xna := memberIdentity(t.Name, m)
	base := &expectedMember{XNA: xna, Owner: t.Name, SourceKind: m.Kind, PackagePath: owner.PackagePath, Receiver: owner.GoName}
	parameters, outResults, hasDirection := mapParameters(s, byIdentity, owner, m.Parameters)
	base.Parameters = parameters
	base.Results = mapReturn(s, byIdentity, owner, m.ReturnType)
	base.Results = append(base.Results, outResults...)
	if isFallible(t, m, "") {
		base.Results = append(base.Results, "error")
		base.ErrorAdded = true
	}
	shape := parameterShape(m.Parameters)

	switch m.Kind {
	case "constructor":
		name := "New" + owner.GoName
		if groups["constructors"] > 1 {
			name += "By" + shape
			base.OverloadMapped = true
		}
		base.GoName, base.GoKind, base.Receiver = name, "func", ""
		result := owner.GoName
		if t.Kind == "class" && t.Name != "Microsoft.Xna.Framework.GameTime" {
			result = "*" + result
		}
		base.Results = append([]string{result}, base.Results...)
		if t.Name == "Microsoft.Xna.Framework.Game" {
			base.Parameters = []string{"GameCallbacks"}
		}
	case "field":
		if t.Kind == "enum" && m.Name == "value__" {
			return nil
		}
		mappedType := mapType(s, byIdentity, owner, valueOrEmpty(m.Type))
		if t.Kind == "enum" {
			base.GoName, base.GoKind, base.Receiver = owner.GoName+m.Name, "const", ""
			base.Results = []string{owner.GoName}
			if len(m.Value) != 0 && string(m.Value) != "null" {
				v := strings.Trim(string(m.Value), "\"")
				base.EnumValue = &v
			}
		} else if m.Static {
			base.GoName, base.Receiver = owner.GoName+m.Name, ""
			if m.Constant {
				base.GoKind = "const"
				base.Results = []string{mappedType}
			} else {
				base.GoKind = "func"
				base.Parameters = nil
				base.Results = []string{mappedType}
			}
		} else {
			base.GoName, base.GoKind = m.Name, "field"
			base.Parameters = nil
			base.Results = []string{mappedType}
		}
	case "property":
		mappedType := mapType(s, byIdentity, owner, valueOrEmpty(m.Type))
		var result []*expectedMember
		if m.Get {
			get := cloneMember(base)
			get.GoKind = chooseMemberKind(m.Static)
			get.GoName = m.Name
			if m.Static {
				get.GoName = owner.GoName + m.Name
				get.Receiver = ""
			}
			get.Parameters = mapIndexerParameters(s, byIdentity, owner, m.Parameters)
			get.Results = mapResultType(s, byIdentity, owner, valueOrEmpty(m.Type))
			get.Accessor = "get"
			if isFallible(t, m, "get") {
				get.Results = append(get.Results, "error")
				get.ErrorAdded = true
			}
			result = append(result, get)
		}
		if m.Set {
			set := cloneMember(base)
			set.GoKind = chooseMemberKind(m.Static)
			set.GoName = "Set" + m.Name
			if m.Static {
				set.GoName = "Set" + owner.GoName + m.Name
				set.Receiver = ""
			}
			set.Parameters = append(mapIndexerParameters(s, byIdentity, owner, m.Parameters), mappedType)
			set.Results = nil
			set.Accessor = "set"
			if isFallible(t, m, "set") {
				set.Results = []string{"error"}
				set.ErrorAdded = true
			}
			result = append(result, set)
		}
		if target := descendantPropertyType(s, m); target != nil && target.PackagePath != owner.PackagePath && strings.HasPrefix(target.PackagePath, owner.PackagePath+"/") {
			for _, item := range result {
				item.PackagePath = target.PackagePath
				item.Receiver = ""
				item.GoKind = "func"
				if strings.HasPrefix(item.GoName, "Set") {
					item.GoName = "Set" + owner.GoName + m.Name
					item.Parameters = []string{"*framework." + owner.GoName, mapType(s, byIdentity, target, valueOrEmpty(m.Type))}
				} else {
					item.GoName = owner.GoName + m.Name
					item.Parameters = []string{"*framework." + owner.GoName}
					item.Results = []string{mapType(s, byIdentity, target, valueOrEmpty(m.Type))}
					if isFallible(t, m, "get") {
						item.Results = append(item.Results, "error")
					}
				}
			}
		}
		for _, item := range result {
			item.Key = symbolKey{Package: item.PackagePath, Receiver: item.Receiver, Name: item.GoName}
		}
		return result
	case "method":
		if op, ok := operatorNames[m.Name]; ok {
			base.GoName = owner.GoName + "Operator" + op + "By" + shape
			base.GoKind, base.Receiver, base.OverloadMapped = "func", "", true
		} else {
			base.GoName = m.Name
			base.GoKind = chooseMemberKind(m.Static)
			if m.Static {
				base.GoName = owner.GoName + m.Name
				base.Receiver = ""
			}
			if groups[fmt.Sprintf("%t|%s", m.Static, m.Name)] > 1 {
				base.GoName += "By" + shape
				base.OverloadMapped = true
			}
		}
		if hasDirection {
			base.OverloadMapped = base.OverloadMapped || groups[fmt.Sprintf("%t|%s", m.Static, m.Name)] > 1
		}
	case "event":
		handler := mapType(s, byIdentity, owner, valueOrEmpty(m.Type))
		add := cloneMember(base)
		remove := cloneMember(base)
		add.GoName, remove.GoName = "Add"+m.Name+"Handler", "Remove"+m.Name+"Handler"
		add.GoKind, remove.GoKind = chooseMemberKind(m.Static), chooseMemberKind(m.Static)
		if m.Static {
			add.GoName, remove.GoName = owner.GoName+add.GoName, owner.GoName+remove.GoName
			add.Receiver, remove.Receiver = "", ""
		}
		// The event accessor projection is settled: one CLR event becomes
		// exactly two Go accessors, Add returns the subscription token, and
		// Remove consumes one. Both carry an error because the accessor pair
		// is the projection of CLR add_/remove_ semantics, not because the
		// declaring type happens to be fallible, so ErrorAdded is pinned here
		// rather than inherited from the owner's classification.
		subscription := frameworkQualified(owner, "EventSubscription")
		add.Parameters, add.Results = []string{handler}, []string{subscription, "error"}
		remove.Parameters, remove.Results = []string{subscription}, []string{"error"}
		add.ErrorAdded, remove.ErrorAdded = true, true
		add.Key = symbolKey{Package: add.PackagePath, Receiver: add.Receiver, Name: add.GoName}
		remove.Key = symbolKey{Package: remove.PackagePath, Receiver: remove.Receiver, Name: remove.GoName}
		return []*expectedMember{add, remove}
	default:
		return nil
	}
	if t.Name == "Microsoft.Xna.Framework.Game" && m.Kind == "method" && isGameLifecycleOverride(m.Name) {
		base.Receiver = "GameCallbacks"
		base.GoKind = "method"
		base.Parameters = append([]string{"*Game"}, base.Parameters...)
	}

	base.Key = symbolKey{Package: base.PackagePath, Receiver: base.Receiver, Name: base.GoName}
	return []*expectedMember{base}
}

func descendantPropertyType(surface *expectedSurface, member contractMember) *expectedType {
	if member.Type == nil {
		return nil
	}
	identity := strings.TrimSuffix(*member.Type, "&")
	if bracket := strings.Index(identity, "["); bracket >= 0 {
		identity = identity[:bracket]
	}
	return surface.typeForXNA(identity)
}

func isGameLifecycleOverride(name string) bool {
	switch name {
	case "Initialize", "LoadContent", "Update", "Draw", "UnloadContent":
		return true
	default:
		return false
	}
}

func cloneMember(in *expectedMember) *expectedMember {
	out := *in
	out.Parameters = append([]string(nil), in.Parameters...)
	out.Results = append([]string(nil), in.Results...)
	return &out
}

func chooseMemberKind(static bool) string {
	if static {
		return "func"
	}
	return "method"
}

func mapParameters(s *expectedSurface, byIdentity map[string]*contractType, owner *expectedType, params []contractParameter) ([]string, []string, bool) {
	var inputs, outputs []string
	hasDirection := false
	for _, p := range params {
		mapped := mapType(s, byIdentity, owner, p.Type)
		if p.Out {
			outputs = append(outputs, mapResultType(s, byIdentity, owner, p.Type)...)
			hasDirection = true
			continue
		}
		if p.Ref {
			mapped = "*" + strings.TrimPrefix(mapped, "*")
			hasDirection = true
		}
		inputs = append(inputs, mapped)
	}
	return inputs, outputs, hasDirection
}

func mapIndexerParameters(s *expectedSurface, byIdentity map[string]*contractType, owner *expectedType, params []contractParameter) []string {
	inputs, _, _ := mapParameters(s, byIdentity, owner, params)
	return inputs
}

func mapReturn(s *expectedSurface, byIdentity map[string]*contractType, owner *expectedType, raw *string) []string {
	if raw == nil || *raw == "System.Void" {
		return nil
	}
	return mapResultType(s, byIdentity, owner, *raw)
}

func mapResultType(s *expectedSurface, byIdentity map[string]*contractType, owner *expectedType, raw string) []string {
	if inner, ok := nullableInner(raw); ok {
		return []string{strings.TrimPrefix(mapType(s, byIdentity, owner, inner), "*"), "bool"}
	}
	return []string{mapType(s, byIdentity, owner, raw)}
}

func mapType(s *expectedSurface, byIdentity map[string]*contractType, owner *expectedType, raw string) string {
	raw = strings.TrimSuffix(raw, "&")
	if inner, ok := nullableInner(raw); ok {
		return "*" + strings.TrimPrefix(mapType(s, byIdentity, owner, inner), "*")
	}
	if strings.HasSuffix(raw, "[]") {
		return "[]" + mapType(s, byIdentity, owner, strings.TrimSuffix(raw, "[]"))
	}
	if mapped, matched, err := mapOwnerGenericParameter(owner, raw); matched {
		if err != nil {
			addGenericMappingIssue(s, owner, raw, err)
			return "any"
		}
		return mapped
	}
	if inner, ok := genericTypeArgument(raw, "System.Collections.Generic.IEnumerator`1["); ok {
		return frameworkQualified(owner, "Iterator") + "[" + mapType(s, byIdentity, owner, inner) + "]"
	}
	// System.EventHandler<T> is the delegate every one of the 49 public CLR
	// events in the profile declares. The generic argument is mapped exactly:
	// degrading it to `any` would erase the args identity the handler is
	// given, which is the whole reason the CLR event is generic.
	if inner, ok := genericTypeArgument(raw, "System.EventHandler`1["); ok {
		return frameworkQualified(owner, "EventHandler") + "[" + mapType(s, byIdentity, owner, inner) + "]"
	}
	// System.EventArgs is a CLR class, so it keeps CLR reference semantics and
	// projects as a pointer to the framework-package language adapter.
	if raw == "System.EventArgs" {
		return "*" + frameworkQualified(owner, "EventArgs")
	}
	if mapped, ok := bclTypes[raw]; ok {
		// TimeSpan is the one primitive BCL entry that maps to a CNA-Go type
		// rather than a Go builtin or standard-library type, so it obeys the
		// same package-qualification rule as every other framework-package
		// value. mapping-rules.json already declares it as framework.TimeSpan.
		if raw == "System.TimeSpan" && owner.PackagePath != modulePath+"/Microsoft/Xna/Framework" {
			return "framework." + mapped
		}
		return mapped
	}
	if raw == "System.Void" || raw == "" {
		return ""
	}
	if !strings.Contains(raw, ".") && !strings.Contains(raw, "[") {
		return raw
	}
	identity := raw
	if bracket := strings.Index(identity, "["); bracket >= 0 {
		identity = identity[:bracket]
	}
	identity = strings.TrimSuffix(identity, "&")
	ct, ok := byIdentity[identity]
	if !ok {
		if strings.HasPrefix(raw, "System.Collections.Generic.IEnumerable`1") || strings.HasPrefix(raw, "System.Collections.Generic.IList`1") || strings.HasPrefix(raw, "System.Collections.Generic.ICollection`1") {
			return "any"
		}
		return "any"
	}
	et := s.typeForXNA(ct.Name)
	name := et.GoName
	if et.PackagePath != owner.PackagePath {
		name = strings.ToLower(path.Base(et.PackagePath)) + "." + name
	}
	if ct.Kind == "class" && ct.Name != "Microsoft.Xna.Framework.GameTime" {
		name = "*" + name
	}
	return name
}

// frameworkQualified spells a framework-package name the way a source file in
// owner's package must spell it. Every CNA-Go language adapter lives in the
// framework package, so each one takes this qualification.
func frameworkQualified(owner *expectedType, name string) string {
	if owner != nil && owner.PackagePath == modulePath+"/Microsoft/Xna/Framework" {
		return name
	}
	return "framework." + name
}

func nullableInner(raw string) (string, bool) {
	raw = strings.TrimSuffix(raw, "&")
	if !strings.HasPrefix(raw, "System.Nullable`1[") || !strings.HasSuffix(raw, "]") {
		return "", false
	}
	return strings.TrimSuffix(strings.TrimPrefix(raw, "System.Nullable`1["), "]"), true
}

func genericTypeArgument(raw, prefix string) (string, bool) {
	if !strings.HasPrefix(raw, prefix) || !strings.HasSuffix(raw, "]") {
		return "", false
	}
	return strings.TrimSuffix(strings.TrimPrefix(raw, prefix), "]"), true
}

// isFallible decides whether one projected operation gains a Go error result.
// accessor is "get" or "set" when the operation is a projected property
// accessor and empty otherwise, so the two accessors of one CLR property are
// classified independently.
func isFallible(t contractType, m contractMember, accessor string) bool {
	keys := fallibilityKeys(m, accessor)
	if pureManagedTypes[t.Name] || classifiedInterfaces[t.Name] || t.Kind == "enum" {
		for _, key := range keys {
			if managedFallibleMembers[t.Name][key] {
				return true
			}
		}
		return false
	}
	for _, key := range keys {
		if managedStoredMembers[t.Name][key] {
			return false
		}
	}
	if m.Kind == "field" || m.Name == "ToString" || m.Name == "GetHashCode" || strings.HasPrefix(m.Name, "op_") {
		return false
	}
	return t.Kind == "class" || t.Kind == "interface"
}

func mapOwnerGenericParameter(owner *expectedType, raw string) (string, bool, error) {
	if !strings.HasPrefix(raw, "!") || strings.HasPrefix(raw, "!!") {
		return "", false, nil
	}
	indexText := strings.TrimPrefix(raw, "!")
	if indexText == "" {
		return "", true, fmt.Errorf("generic parameter token has no index")
	}
	index, err := strconv.Atoi(indexText)
	if err != nil || index < 0 {
		return "", true, fmt.Errorf("generic parameter token %q has an invalid index", raw)
	}
	if owner == nil || index >= len(owner.GenericParameter) {
		return "", true, fmt.Errorf("generic parameter token %q has no declared owner parameter", raw)
	}
	return owner.GenericParameter[index], true, nil
}

func addGenericMappingIssue(surface *expectedSurface, owner *expectedType, raw string, err error) {
	if surface == nil {
		return
	}
	xna := ""
	goIdentity := ""
	if owner != nil {
		xna = owner.XNA
		goIdentity = owner.Key.String()
	}
	message := fmt.Sprintf("cannot substitute CLR generic parameter %s: %v", raw, err)
	for _, issue := range surface.MappingIssues {
		if issue.Category == "GENERIC_MAPPING_MISMATCH" && issue.XNA == xna && issue.Message == message {
			return
		}
	}
	surface.MappingIssues = append(surface.MappingIssues, diagnostic{
		Category: "GENERIC_MAPPING_MISMATCH",
		XNA:      xna,
		Go:       goIdentity,
		Message:  message,
	})
}

func buildMappedInterfacesAndWitnesses(surface *expectedSurface, byIdentity map[string]*contractType) {
	for _, owner := range sortedExpectedTypes(surface) {
		contractOwner := byIdentity[owner.XNA]
		if contractOwner == nil {
			continue
		}
		for _, raw := range contractOwner.DirectInterfaces {
			identity, arguments := splitConstructedType(raw)
			interfaceType := byIdentity[identity]
			if interfaceType == nil || interfaceType.Kind != "interface" {
				continue
			}
			mapped := mappedInterface{XNA: raw}
			mappedType := surface.typeForXNA(identity)
			mapped.GoName = qualifiedTypeName(owner, mappedType)
			for _, argument := range arguments {
				mapped.TypeArguments = append(mapped.TypeArguments, mapType(surface, byIdentity, owner, argument))
			}
			owner.MappedInterfaces = append(owner.MappedInterfaces, mapped)
			if contractOwner.Kind == "struct" && strings.HasPrefix(owner.XNA, packedVectorNamespace) {
				collectInterfaceWitnesses(surface, byIdentity, owner, interfaceType, mapped.TypeArguments, map[string]bool{})
			}
		}
	}
}

func collectInterfaceWitnesses(surface *expectedSurface, byIdentity map[string]*contractType, owner *expectedType, interfaceType *contractType, arguments []string, visited map[string]bool) {
	visitKey := interfaceType.Name + "[" + strings.Join(arguments, ",") + "]"
	if visited[visitKey] {
		return
	}
	visited[visitKey] = true

	substitutions := make(map[string]string)
	for i, parameter := range interfaceType.GenericParameters {
		if i < len(arguments) {
			substitutions[parameter.Name] = arguments[i]
		}
	}
	mappedInterfaceType := surface.typeForXNA(interfaceType.Name)
	if mappedInterfaceType != nil {
		for _, memberKey := range mappedInterfaceType.Members {
			member := surface.Members[memberKey]
			parameters := substituteMappedTypes(member.Parameters, substitutions)
			results := substituteMappedTypes(member.Results, substitutions)
			concreteKey := symbolKey{Package: owner.PackagePath, Receiver: owner.GoName, Name: member.GoName}
			if concrete := surface.Members[concreteKey]; concrete != nil && equalStrings(concrete.Parameters, parameters) && equalStrings(concrete.Results, results) {
				continue
			}
			if _, exists := surface.InterfaceWitnesses[concreteKey]; exists {
				continue
			}
			surface.InterfaceWitnesses[concreteKey] = &expectedInterfaceWitness{
				Key:             concreteKey,
				Owner:           owner.XNA,
				SourceInterface: interfaceType.Name,
				InterfaceMember: member.XNA,
				GoName:          member.GoName,
				Parameters:      parameters,
				Results:         results,
				Reason:          "exported Go method required to witness an explicit CLR interface implementation",
			}
		}
	}

	for _, rawBase := range interfaceType.DirectInterfaces {
		baseIdentity, rawArguments := splitConstructedType(rawBase)
		baseType := byIdentity[baseIdentity]
		if baseType == nil || baseType.Kind != "interface" {
			continue
		}
		baseArguments := make([]string, 0, len(rawArguments))
		for _, rawArgument := range rawArguments {
			if mapped, ok := substitutions[strings.TrimPrefix(rawArgument, "!")]; ok {
				baseArguments = append(baseArguments, mapped)
				continue
			}
			baseArguments = append(baseArguments, rawArgument)
		}
		collectInterfaceWitnesses(surface, byIdentity, owner, baseType, baseArguments, visited)
	}
}

func qualifiedTypeName(owner, target *expectedType) string {
	if target == nil {
		return "any"
	}
	if target.PackagePath == owner.PackagePath {
		return target.GoName
	}
	return strings.ToLower(path.Base(target.PackagePath)) + "." + target.GoName
}

func substituteMappedTypes(values []string, substitutions map[string]string) []string {
	result := make([]string, len(values))
	for i, value := range values {
		if replacement, ok := substitutions[value]; ok {
			result[i] = replacement
		} else {
			result[i] = value
		}
	}
	return result
}

func splitConstructedType(raw string) (string, []string) {
	open := strings.Index(raw, "[")
	if open < 0 || !strings.HasSuffix(raw, "]") {
		return raw, nil
	}
	identity := raw[:open]
	contents := raw[open+1 : len(raw)-1]
	var arguments []string
	start, depth := 0, 0
	for i, character := range contents {
		switch character {
		case '[':
			depth++
		case ']':
			depth--
		case ',':
			if depth == 0 {
				arguments = append(arguments, strings.TrimSpace(contents[start:i]))
				start = i + 1
			}
		}
	}
	arguments = append(arguments, strings.TrimSpace(contents[start:]))
	return identity, arguments
}

func parameterShape(params []contractParameter) string {
	if len(params) == 0 {
		return "None"
	}
	parts := make([]string, 0, len(params))
	for _, p := range params {
		prefix := ""
		if p.Out {
			prefix = "Out"
		} else if p.Ref {
			prefix = "Ref"
		} else if p.In {
			prefix = "In"
		}
		parts = append(parts, prefix+typeShape(p.Type))
	}
	return strings.Join(parts, "And")
}

var nonIdentifier = regexp.MustCompile(`[^A-Za-z0-9]+`)

func typeShape(raw string) string {
	raw = strings.TrimSuffix(raw, "&")
	if strings.HasPrefix(raw, "System.Nullable`1[") && strings.HasSuffix(raw, "]") {
		inner := strings.TrimSuffix(strings.TrimPrefix(raw, "System.Nullable`1["), "]")
		return "NullableOf" + typeShape(inner)
	}
	array := strings.HasSuffix(raw, "[]")
	raw = strings.TrimSuffix(raw, "[]")
	if dot := strings.LastIndex(raw, "."); dot >= 0 {
		raw = raw[dot+1:]
	}
	raw = strings.ReplaceAll(raw, "+", "")
	raw = strings.ReplaceAll(raw, "`", "Of")
	raw = nonIdentifier.ReplaceAllString(raw, "")
	if array {
		raw = "SliceOf" + raw
	}
	return exportIdentifier(raw)
}

func memberIdentity(owner string, m contractMember) string {
	params := make([]string, len(m.Parameters))
	for i, p := range m.Parameters {
		direction := ""
		if p.Out {
			direction = "out "
		} else if p.Ref {
			direction = "ref "
		} else if p.In {
			direction = "in "
		}
		params[i] = direction + p.Type
	}
	return fmt.Sprintf("%s::%s(%s)", owner, m.Name, strings.Join(params, ","))
}

func namespaceOf(identity string) string {
	top := identity
	if plus := strings.Index(top, "+"); plus >= 0 {
		top = top[:plus]
	}
	dot := strings.LastIndex(top, ".")
	if dot < 0 {
		return ""
	}
	return top[:dot]
}

func flattenedBaseName(identity string) string {
	name := identity[strings.LastIndex(identity, ".")+1:]
	parts := strings.Split(name, "+")
	for i := range parts {
		if tick := strings.Index(parts[i], "`"); tick >= 0 {
			parts[i] = parts[i][:tick]
		}
	}
	return strings.Join(parts, "")
}

func mappedTypeName(t contractType, collisions map[string]int) string {
	base := flattenedBaseName(t.Name)
	if len(t.GenericParameters) > 0 && collisions[namespaceOf(t.Name)+"|"+base] > 1 {
		base += "Of"
		for _, p := range t.GenericParameters {
			base += exportIdentifier(p.Name)
		}
	}
	return exportIdentifier(base)
}

func exportIdentifier(value string) string {
	if value == "" {
		return value
	}
	runes := []rune(value)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func packagePathForNamespace(namespace string) string {
	return modulePath + "/" + strings.ReplaceAll(namespace, ".", "/")
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func sortedExpectedTypes(s *expectedSurface) []*expectedType {
	result := make([]*expectedType, 0, len(s.Types))
	for _, t := range s.Types {
		result = append(result, t)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].XNA < result[j].XNA })
	return result
}
