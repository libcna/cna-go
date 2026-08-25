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

	// Foundation 26. Microsoft.Xna.Framework.Game.dll shows
	// GameComponentCollection as a sealed Collection<IGameComponent> subclass
	// whose whole implementation is managed list work plus two delegate
	// fields: the constructor is one `call base..ctor`, the four overrides
	// index, insert, remove and clear the inherited List<IGameComponent> and
	// raise two events, and the two raise helpers are a null test and an
	// Invoke. Its inherited base, System.Collections.ObjectModel.Collection`1,
	// is equally managed in the pinned mscorlib. Nothing reaches a device, an
	// allocation, or native code.
	"Microsoft.Xna.Framework.GameComponentCollection": true,

	// Foundation 27. Microsoft.Xna.Framework.dll shows VisualizationData as
	// four fields and two one-ldfld getters: the constructor allocates two
	// 256-element Single arrays with `newarr` and wraps each in a
	// ReadOnlyCollection`1<float32>, validating nothing. It reaches no device,
	// no allocation beyond those arrays, and no native code, and it starts no
	// playback.
	"Microsoft.Xna.Framework.Media.VisualizationData": true,
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
	// ever names, MAPPED when a derived XNA type may be projected today and
	// the base contributes no inherited surface, COMPOSED when the base is a
	// supported BCL family whose public members the derived type re-exposes
	// through a private adapter, and DEFERRED when the relationship is still
	// an open public-API decision and no derived type may be projected yet.
	//
	// COMPOSED is the one status under which a base contributes projected Go
	// identities. They are counted as BCL-inherited projections and never as
	// XNA-declared reference members, so no member is ever counted twice and
	// the pinned XNA-declared member count is unaffected.
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
		Adapter:   "collectionBase[T]",
		Status:    "COMPOSED",
		Rationale: "modelled by the private collectionBase[T] adapter; a derived XNA class holds it in an unexported field and re-exposes the eleven inherited public members through measured forwarding, with no exported embedding",
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
	// GameComponentCollection mixes both provenance classes in one
	// fallibility table, which is correct: fallibility is a property of a
	// projected operation, and an inherited operation's failures come from the
	// pinned Collection<T>/List<T> IL plus whatever this subclass's hooks add.
	//
	// The four XNA-declared overrides all fail: InsertItem rejects a
	// duplicate, RemoveItem reads an element that may be out of range,
	// SetItem throws unconditionally, and ClearItems propagates a handler
	// failure. The constructor is one `call base..ctor` and cannot.
	//
	// Of the eleven inherited members, Count, Contains, IndexOf and
	// GetEnumerator only read. Add, Clear, Insert, Remove and RemoveAt each
	// reach a hook that can fail, and Insert and RemoveAt validate their index
	// first. CopyTo carries Array.Copy's three argument failures.
	//
	// property|Item marks both accessors, and both genuinely fail: the getter
	// forwards to List<T>'s bounds check, and the setter validates the index
	// and then reaches a SetItem that never succeeds.
	"Microsoft.Xna.Framework.GameComponentCollection": {
		"method|InsertItem": true,
		"method|RemoveItem": true,
		"method|SetItem":    true,
		"method|ClearItems": true,
		"property|Item":     true,
		"method|Add":        true,
		"method|Clear":      true,
		"method|CopyTo":     true,
		"method|Insert":     true,
		"method|Remove":     true,
		"method|RemoveAt":   true,
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
		inherited := mapInheritedBaseMembers(s, byIdentity, owner, *t)
		owner.BCLInheritedCLRMembers = inheritedCLRMemberCount(*t)
		owner.BCLInheritedProjections = len(inherited)
		s.BCLInheritedCLRMembers += owner.BCLInheritedCLRMembers
		s.BCLInheritedProjections += owner.BCLInheritedProjections
		allMembers = append(allMembers, inherited...)
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
	// System.Collections.ObjectModel.ReadOnlyCollection<T> appears at
	// signature positions in the pinned contract -- VisualizationData returns
	// two of them and Microphone.All returns one -- so it needs a public Go
	// spelling on exactly the footing System.TimeSpan and
	// System.EventHandler<T> already have, or the members that return it
	// cannot be projected at all.
	//
	// It is a CLR class, so it keeps reference semantics and projects as a
	// pointer. The element type is mapped exactly and never degraded, for the
	// same reason the event handler's argument is not.
	//
	// This is the SIGNATURE-POSITION projection only. Whether an XNA class may
	// DERIVE from ReadOnlyCollection<T> is a separate question, still deferred
	// in bclBaseRelationships.
	if inner, ok := genericTypeArgument(raw, "System.Collections.ObjectModel.ReadOnlyCollection`1["); ok {
		return "*" + frameworkQualified(owner, "ReadOnlyCollection") + "[" + mapType(s, byIdentity, owner, inner) + "]"
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

// ---------------------------------------------------------------------------
// The general BCL base-class composition projection.
// ---------------------------------------------------------------------------

// bclBaseAdapter is one supported BCL base-class family.
//
// A CLR class that inherits a supported BCL collection base is projected as a
// concrete Go reference type that CONTAINS a private generic adapter for that
// base and re-exposes the base's public surface through measured forwarding
// members. The adapter is implementation machinery: it is not an XNA type, not
// an exported field, not a public base-class object, not an embedded public
// API, and not a native handle.
//
// Exported embedding is refused rather than merely discouraged. Writing
//
//	type GameComponentCollection struct { Collection[IGameComponent] }
//
// would publish a Go type the XNA contract never declares, promote a whole
// method set the derived type never declared, imply a Go subtype relationship
// CLR inheritance does not create, and let a consumer type-assert to support
// machinery. verifyBCLBaseProjection rejects it.
//
// The registry is deliberately not a general .NET runtime. Only the families a
// pinned XNA public type actually inherits are represented, and a family is
// only usable once its exact behavior has been read from the pinned BCL.
type bclBaseAdapter struct {
	// GoAdapter is the unexported Go type family that models the base. It is
	// never exported and never named by any projected signature.
	GoAdapter string
	// AdapterField is the unexported field name a consumer must hold it in.
	AdapterField string
	// GenericArity is how many type arguments the CLR base takes.
	GenericArity int
	// BehaviorLevel is SUPPORTED when every inherited public member below has
	// its exact pinned behavior established, and PARTIAL when some inherited
	// member is present in the inventory but not yet faithful.
	BehaviorLevel string
	// Authority is the exact BCL binary the behavior was read from.
	Authority string
	// AuthoritySHA256 pins that binary.
	AuthoritySHA256 string
	// Members is the exact public member inventory the base contributes to
	// every derived type. Constructors are excluded because the CLR does not
	// inherit them; protected and explicitly implemented members are excluded
	// because they are not public surface.
	Members []bclInheritedMember
	// Excluded records the base members deliberately left unprojected, with
	// the reason, so an exclusion is measured rather than silent.
	Excluded  []bclExcludedMember
	Rationale string
}

// bclInheritedMember is one public CLR member of a BCL base, expressed as the
// same contractMember shape a declared XNA member uses so it runs through the
// identical naming, overload, direction, and fallibility machinery. Type
// strings use the CLR generic-argument tokens !0 and !1, which are substituted
// with the derived type's actual base arguments before mapping.
type bclInheritedMember struct {
	Member    contractMember
	Rationale string
}

// bclExcludedMember records one public-looking base member that is not
// projected, and why.
type bclExcludedMember struct {
	CLRMember string
	Reason    string
}

func bclMethod(name string, returnType string, parameters ...contractParameter) contractMember {
	member := contractMember{Kind: "method", Name: name, Access: "public", Parameters: parameters}
	if returnType != "" {
		member.ReturnType = &returnType
	} else {
		void := "System.Void"
		member.ReturnType = &void
	}
	return member
}

func bclProperty(name string, propertyType string, get, set bool, parameters ...contractParameter) contractMember {
	access := "public"
	member := contractMember{Kind: "property", Name: name, Access: access, Type: &propertyType, Get: get, Set: set, Parameters: parameters}
	if get {
		member.GetAccess = &access
	}
	if set {
		member.SetAccess = &access
	}
	return member
}

func bclParameter(name, clrType string) contractParameter {
	return contractParameter{Name: name, Type: clrType}
}

// bclBaseAdapters declares every supported BCL base-class family.
//
// A base listed here has status COMPOSED in bclBaseRelationships, which is what
// admits its derived XNA types for projection.
var bclBaseAdapters = map[string]bclBaseAdapter{
	// System.Collections.ObjectModel.Collection<T>.
	//
	// Read from mscorlib 4.0.30319.1, the .NET Framework 4.0 RTM binary every
	// pinned XNA assembly binds against with `.assembly extern mscorlib
	// 4.0.0.0` and public key token b77a5c561934e089.
	//
	// The class stores its elements in a private IList<T> field, `items`. Its
	// parameterless constructor -- the only one any pinned XNA subclass calls
	// -- assigns `new List<T>()`, so the store is always List<T> and its
	// IsReadOnly is always false; the `items.IsReadOnly` guard that opens six
	// of the public mutators is therefore statically dead for every consumer
	// in the profile and is not projected as a failure mode.
	//
	// Its public surface is exactly the eleven members below. Everything else
	// the class declares is one of three things that are not public surface:
	// two constructors, which the CLR does not inherit; the protected Items
	// property and the four protected virtual hooks; and fourteen private
	// explicit implementations of IList, ICollection, IEnumerable and
	// ICollection<T>.IsReadOnly, which the settled BCL-interface rule already
	// excludes.
	"System.Collections.ObjectModel.Collection`1": {
		GoAdapter:       "collectionBase[T]",
		AdapterField:    "base",
		GenericArity:    1,
		BehaviorLevel:   "SUPPORTED",
		Authority:       "mscorlib.dll 4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0",
		AuthoritySHA256: "5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63",
		Rationale:       "a mutable BCL collection base projected as private composition plus measured forwarding; every public mutator routes through the equivalent overridable hook so a subclass override runs",
		Members: []bclInheritedMember{
			{Member: bclProperty("Count", "System.Int32", true, false),
				Rationale: "get_Count is `items.Count`; it cannot fail"},
			{Member: bclProperty("Item", "!0", true, true, bclParameter("index", "System.Int32")),
				Rationale: "get_Item is one `items[index]` whose bounds check is List<T>'s unsigned `(uint)index >= (uint)_size`; set_Item validates `index < 0 || index >= Count` itself and only then reaches the SetItem hook"},
			{Member: bclMethod("Add", "System.Void", bclParameter("item", "!0")),
				Rationale: "Add is `InsertItem(Count, item)` and validates nothing of its own"},
			{Member: bclMethod("Clear", "System.Void"),
				Rationale: "Clear is `ClearItems()`"},
			{Member: bclMethod("Contains", "System.Boolean", bclParameter("item", "!0")),
				Rationale: "Contains is `items.Contains(item)`, a forward linear scan over EqualityComparer<T>.Default that stops at the first match"},
			{Member: bclMethod("CopyTo", "System.Void", bclParameter("array", "!0[]"), bclParameter("index", "System.Int32")),
				Rationale: "CopyTo is `items.CopyTo(array, index)` and then Array.Copy, whose three failures are a null destination, a negative index, and a destination too small for Count elements"},
			{Member: bclMethod("GetEnumerator", "System.Collections.Generic.IEnumerator`1[!0]"),
				Rationale: "GetEnumerator returns `items.GetEnumerator()` boxed as IEnumerator<T>, so the observed enumerator is List<T>.Enumerator and its version invalidation is List<T>'s"},
			{Member: bclMethod("IndexOf", "System.Int32", bclParameter("item", "!0")),
				Rationale: "IndexOf is `items.IndexOf(item)`, ending in Array.IndexOf<T> over the same default comparer"},
			{Member: bclMethod("Insert", "System.Void", bclParameter("index", "System.Int32"), bclParameter("item", "!0")),
				Rationale: "Insert guards `index < 0 || index > Count`, which admits Count itself, and then reaches the InsertItem hook"},
			{Member: bclMethod("Remove", "System.Boolean", bclParameter("item", "!0")),
				Rationale: "Remove returns false without touching the collection when IndexOf finds nothing, and otherwise reaches the RemoveItem hook and returns true"},
			{Member: bclMethod("RemoveAt", "System.Void", bclParameter("index", "System.Int32")),
				Rationale: "RemoveAt guards `index < 0 || index >= Count` and then reaches the RemoveItem hook"},
		},
		Excluded: []bclExcludedMember{
			{CLRMember: ".ctor()", Reason: "the CLR does not inherit constructors; a derived type projects its own"},
			{CLRMember: ".ctor(IList`1<T>)", Reason: "the CLR does not inherit constructors, and no pinned XNA subclass calls this overload"},
			{CLRMember: "Items", Reason: "`family`, so it is not public surface; exposing it would hand out the backing store"},
			{CLRMember: "InsertItem", Reason: "`family` virtual hook, projected as the private collectionOverrides interface rather than as public Go API"},
			{CLRMember: "RemoveItem", Reason: "`family` virtual hook, projected as the private collectionOverrides interface"},
			{CLRMember: "SetItem", Reason: "`family` virtual hook, projected as the private collectionOverrides interface"},
			{CLRMember: "ClearItems", Reason: "`family` virtual hook, projected as the private collectionOverrides interface"},
			{CLRMember: "ICollection<T>.IsReadOnly", Reason: "private explicit implementation; the settled BCL-interface rule projects nothing for it"},
			{CLRMember: "IEnumerable.GetEnumerator", Reason: "private explicit implementation, and the generic GetEnumerator already carries enumeration"},
			{CLRMember: "ICollection.IsSynchronized", Reason: "private explicit implementation"},
			{CLRMember: "ICollection.SyncRoot", Reason: "private explicit implementation, and CNA-Go projects no CLR sync root"},
			{CLRMember: "ICollection.CopyTo", Reason: "private explicit implementation; the generic CopyTo carries the operation"},
			{CLRMember: "IList.Item", Reason: "private explicit implementation of the non-generic indexer"},
			{CLRMember: "IList.IsReadOnly", Reason: "private explicit implementation"},
			{CLRMember: "IList.IsFixedSize", Reason: "private explicit implementation"},
			{CLRMember: "IList.Add", Reason: "private explicit implementation"},
			{CLRMember: "IList.Contains", Reason: "private explicit implementation"},
			{CLRMember: "IList.IndexOf", Reason: "private explicit implementation"},
			{CLRMember: "IList.Insert", Reason: "private explicit implementation"},
			{CLRMember: "IList.Remove", Reason: "private explicit implementation"},
		},
	},
}

// bclBaseArguments splits the CLR generic arguments off a base identity, so
// `System.Collections.ObjectModel.Collection`1[Microsoft.Xna.Framework.IGameComponent]`
// yields exactly one argument. It splits at top-level commas so a nested
// generic argument survives intact.
func bclBaseArguments(raw string) []string {
	open := strings.Index(raw, "[")
	if open < 0 || !strings.HasSuffix(raw, "]") {
		return nil
	}
	inner := raw[open+1 : len(raw)-1]
	var arguments []string
	depth, start := 0, 0
	for i := 0; i < len(inner); i++ {
		switch inner[i] {
		case '[':
			depth++
		case ']':
			depth--
		case ',':
			if depth == 0 {
				arguments = append(arguments, strings.TrimSpace(inner[start:i]))
				start = i + 1
			}
		}
	}
	arguments = append(arguments, strings.TrimSpace(inner[start:]))
	return arguments
}

// substituteBaseArguments replaces the CLR generic-argument tokens !0 and !1 in
// one inherited member's type strings with the derived type's actual base
// arguments. Longer tokens are substituted first so !10 could never be read as
// !1 followed by a literal 0.
func substituteBaseArguments(raw string, arguments []string) string {
	for i := len(arguments) - 1; i >= 0; i-- {
		raw = strings.ReplaceAll(raw, "!"+strconv.Itoa(i), arguments[i])
	}
	return raw
}

// instantiateInheritedMember produces the contractMember one BCL base
// contributes to one derived XNA type, with the base's generic arguments
// substituted.
func instantiateInheritedMember(member contractMember, arguments []string) contractMember {
	instance := member
	if member.ReturnType != nil {
		substituted := substituteBaseArguments(*member.ReturnType, arguments)
		instance.ReturnType = &substituted
	}
	if member.Type != nil {
		substituted := substituteBaseArguments(*member.Type, arguments)
		instance.Type = &substituted
	}
	instance.Parameters = make([]contractParameter, len(member.Parameters))
	for i, parameter := range member.Parameters {
		instance.Parameters[i] = parameter
		instance.Parameters[i].Type = substituteBaseArguments(parameter.Type, arguments)
	}
	return instance
}

// mapInheritedBaseMembers projects the public members a supported BCL base
// contributes to one derived XNA type.
//
// The inherited members run through the same mapMember machinery a declared
// XNA member does, so they obey the identical naming, overload, parameter
// direction, and per-operation fallibility rules, and they take part in the
// same collision resolution. That is deliberate: GameComponentCollection
// declares a protected SetItem override AND inherits a public Item setter,
// which the settled rules both spell SetItem, and the established collision
// rule must resolve that rather than a bespoke exception.
//
// Provenance is recorded on every member it produces. An inherited member is
// never counted as an XNA-declared reference member, so REFERENCE_MEMBERS
// keeps naming exactly what the Microsoft assemblies declare while
// EXPECTED_GO_MEMBERS grows by the surface that is now representable.
func mapInheritedBaseMembers(s *expectedSurface, byIdentity map[string]*contractType, owner *expectedType, t contractType) []*expectedMember {
	if t.BaseType == nil {
		return nil
	}
	raw := *t.BaseType
	identity := baseIdentityWithoutArguments(raw)
	relationship, declared := bclBaseRelationships[identity]
	if !declared || relationship.Status != "COMPOSED" {
		return nil
	}
	adapter, present := bclBaseAdapters[identity]
	if !present {
		return nil
	}
	arguments := bclBaseArguments(raw)
	if len(arguments) != adapter.GenericArity {
		s.MappingIssues = append(s.MappingIssues, diagnostic{
			Category: "BASE_MAPPING_MISMATCH", XNA: t.Name, Go: owner.Key.String(),
			Message: fmt.Sprintf("CLR base %q takes %d generic arguments, the derived type supplies %d", identity, adapter.GenericArity, len(arguments)),
		})
		return nil
	}

	// The inherited members form their own overload namespace. None of the
	// supported families declares two public members with one CLR name, so
	// every inherited member is a unique-name projection; the assertion below
	// keeps that a measured fact rather than an assumption.
	inheritedGroups := make(map[string]int)
	synthetic := contractType{Name: t.Name, Kind: t.Kind, Sealed: t.Sealed}
	for _, entry := range adapter.Members {
		instance := instantiateInheritedMember(entry.Member, arguments)
		synthetic.Members = append(synthetic.Members, instance)
		if instance.Kind == "method" {
			inheritedGroups[instance.Name]++
		}
	}
	for name, count := range inheritedGroups {
		if count > 1 {
			s.MappingIssues = append(s.MappingIssues, diagnostic{
				Category: "BASE_MAPPING_MISMATCH", XNA: t.Name, Go: owner.Key.String(),
				Message: fmt.Sprintf("inherited BCL member %q from %q is overloaded, which the inherited projection does not model", name, identity),
			})
		}
	}

	var mapped []*expectedMember
	groups := overloadGroups(synthetic)
	for _, instance := range synthetic.Members {
		for _, member := range mapMember(s, byIdentity, owner, synthetic, instance, groups) {
			member.BCLBase = identity
			member.BCLMember = instance.Name
			member.XNA = identity + "::" + strings.TrimPrefix(member.XNA, t.Name+"::")
			mapped = append(mapped, member)
		}
	}
	return mapped
}

// inheritedCLRMemberCount is how many public CLR members one derived type's
// supported BCL base contributes, before any of them is projected. It is the
// BCL-INHERITED provenance class of the identity accounting, kept separate
// from the XNA-declared reference members the Microsoft assemblies declare.
func inheritedCLRMemberCount(t contractType) int {
	if t.BaseType == nil {
		return 0
	}
	identity := baseIdentityWithoutArguments(*t.BaseType)
	if relationship, declared := bclBaseRelationships[identity]; !declared || relationship.Status != "COMPOSED" {
		return 0
	}
	return len(bclBaseAdapters[identity].Members)
}

// goAdapterArgument is the Go spelling of one CLR generic argument as it
// appears in a private BCL adapter field type. The adapter is declared in the
// same package as the consumers that hold it, so an XNA argument needs no
// package qualifier.
func goAdapterArgument(clr string) string {
	if mapped, ok := bclTypes[clr]; ok {
		return mapped
	}
	return flattenedBaseName(clr)
}

// bclSignatureAdapters declares every BCL type the pinned XNA contract carries
// at a PUBLIC SIGNATURE POSITION and that therefore needs a public Go
// spelling, together with the exact public CLR member inventory that spelling
// must reproduce.
//
// This is a different role from bclBaseAdapters and the two must not be
// confused. A base adapter is PRIVATE machinery a derived type composes and
// forwards; a signature adapter is a PUBLIC type because a projected member
// returns one, exactly as System.TimeSpan and System.EventHandler<T> are
// public because projected members carry them.
//
// One CLR type can legitimately appear in both roles. ReadOnlyCollection<T> is
// a signature adapter here, because VisualizationData returns two of them,
// while remaining DEFERRED as a base in bclBaseRelationships: whether an XNA
// class may derive from it is a separate question, and no consumer needs the
// answer yet.
//
// The registry exists so the adapter's surface is measured rather than merely
// permitted. Without it an adapter type is a hole in the unexpected-member
// scan, since every exported member on an adapter receiver is admitted.
var bclSignatureAdapters = map[string]bclBaseAdapter{
	// System.Collections.ObjectModel.ReadOnlyCollection<T>.
	//
	// It holds one private `IList<T> list` field and forwards every read to
	// it. It stores the list rather than copying it, so read-only means no
	// public mutation THROUGH THIS SURFACE, not immutable storage: the owner
	// keeps writing and every change is visible through the view.
	//
	// Its public surface is exactly the six members below. The one
	// constructor is not inherited by anything; `Items` is `family`; and every
	// mutator -- ICollection<T>.Add/Clear/Remove, IList<T>.Insert/RemoveAt,
	// IList<T>.set_Item and the whole non-generic IList set -- is a private
	// explicit implementation, so the settled BCL-interface rule already
	// excludes them and read-only needed no new decision.
	"System.Collections.ObjectModel.ReadOnlyCollection`1": {
		GoAdapter:       "ReadOnlyCollection[T]",
		GenericArity:    1,
		BehaviorLevel:   "SUPPORTED",
		Authority:       "mscorlib.dll 4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0",
		AuthoritySHA256: "5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63",
		Rationale:       "a read-only BCL view the pinned contract returns from projected members; it forwards every read to the list it was given, so enumeration semantics and mutation visibility are the underlying list's rather than the view's",
		Members: []bclInheritedMember{
			{Member: bclProperty("Count", "System.Int32", true, false),
				Rationale: "get_Count is one forwarded `list.Count`"},
			{Member: bclProperty("Item", "!0", true, false, bclParameter("index", "System.Int32")),
				Rationale: "get_Item is one forwarded `list[index]`; the SETTER is a private explicit implementation, which is what makes the view read-only"},
			{Member: bclMethod("Contains", "System.Boolean", bclParameter("value", "!0")),
				Rationale: "Contains is one forwarded `list.Contains(value)` over EqualityComparer<T>.Default"},
			{Member: bclMethod("CopyTo", "System.Void", bclParameter("array", "!0[]"), bclParameter("index", "System.Int32")),
				Rationale: "CopyTo is one forwarded `list.CopyTo(array, index)`, which for an array-backed list carries Array.Copy's three failures"},
			{Member: bclMethod("GetEnumerator", "System.Collections.Generic.IEnumerator`1[!0]"),
				Rationale: "GetEnumerator forwards to the underlying list's enumerator, so an array-backed view is NOT version-checked while a List<T>-backed one is"},
			{Member: bclMethod("IndexOf", "System.Int32", bclParameter("value", "!0")),
				Rationale: "IndexOf is one forwarded `list.IndexOf(value)`"},
		},
		Excluded: []bclExcludedMember{
			{CLRMember: ".ctor(IList`1<T>)", Reason: "the adapter's own constructor is the projection of it; no XNA member is projected from it"},
			{CLRMember: "Items", Reason: "`family`; exposing it would hand out the wrapped list"},
			{CLRMember: "ICollection<T>.IsReadOnly", Reason: "private explicit implementation; the settled BCL-interface rule projects nothing for it"},
			{CLRMember: "IList<T>.Item", Reason: "private explicit implementation whose setter throws NotSupportedException; the generic getter is the projected indexer"},
			{CLRMember: "ICollection<T>.Add", Reason: "private explicit mutator; read-only means it is not public surface"},
			{CLRMember: "ICollection<T>.Clear", Reason: "private explicit mutator"},
			{CLRMember: "ICollection<T>.Remove", Reason: "private explicit mutator"},
			{CLRMember: "IList<T>.Insert", Reason: "private explicit mutator"},
			{CLRMember: "IList<T>.RemoveAt", Reason: "private explicit mutator"},
			{CLRMember: "IEnumerable.GetEnumerator", Reason: "private explicit implementation; the generic GetEnumerator carries enumeration"},
			{CLRMember: "ICollection.IsSynchronized", Reason: "private explicit implementation"},
			{CLRMember: "ICollection.SyncRoot", Reason: "private explicit implementation, and CNA-Go projects no CLR sync root"},
			{CLRMember: "ICollection.CopyTo", Reason: "private explicit implementation"},
			{CLRMember: "IList.IsFixedSize", Reason: "private explicit implementation"},
			{CLRMember: "IList.IsReadOnly", Reason: "private explicit implementation"},
			{CLRMember: "IList.Item", Reason: "private explicit implementation"},
			{CLRMember: "IList.Add", Reason: "private explicit mutator"},
			{CLRMember: "IList.Clear", Reason: "private explicit mutator"},
			{CLRMember: "IList.Contains", Reason: "private explicit implementation"},
			{CLRMember: "IList.IndexOf", Reason: "private explicit implementation"},
			{CLRMember: "IList.Insert", Reason: "private explicit mutator"},
			{CLRMember: "IList.Remove", Reason: "private explicit mutator"},
			{CLRMember: "IList.RemoveAt", Reason: "private explicit mutator"},
		},
	},
}

// bclSignatureAdapterConstructors are the exported functions that build a
// signature adapter, declared per adapter so a stray exported constructor is
// still an unexpected member.
var bclSignatureAdapterConstructors = map[string]string{
	"NewReadOnlyCollectionOverSingles": "System.Collections.ObjectModel.ReadOnlyCollection`1",
}

// bclSignatureAdapterGoName is the Go type name of one signature adapter,
// without its type parameters.
func bclSignatureAdapterGoName(adapter bclBaseAdapter) string {
	if open := strings.Index(adapter.GoAdapter, "["); open >= 0 {
		return adapter.GoAdapter[:open]
	}
	return adapter.GoAdapter
}
