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
	// Foundation 69. System.Text.StringBuilder, on exactly the measurement
	// System.IO.Stream's entry rests on.
	//
	// The pinned contract carries StringBuilder at FOUR public signature
	// positions -- SpriteFont::MeasureString and SpriteBatch's three DrawString
	// shapes -- and every one of them is an input the reference only READS.
	// SpriteFont/StringProxy is the proof: it stores the builder and calls
	// get_Length and get_Chars, and nothing anywhere else in the profile
	// touches one.
	//
	// So the position takes the standard-library Go type whose ROLE it is,
	// exactly as a read-only Stream position takes io.Reader, rather than a
	// reimplemented BCL class. `strings.Builder` is Go's mutable text
	// accumulator, `Len` and `String` are the two reads StringProxy makes, and
	// nil is the reference's null.
	//
	// The overload identity survives: `string` and `*strings.Builder` are
	// different Go types, so the CLR's two-member overload set stays two Go
	// members, which is what the overload set means.
	"System.Text.StringBuilder": "*strings.Builder",
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

	// Foundation 52. Microsoft.Xna.Framework.Graphics.dll IL shows DisplayMode
	// as three private fields and six members over them, with no constructor in
	// the public contract at all. Width, Height and Format are one ldfld each;
	// AspectRatio is 38 bytes of arithmetic over two of those fields;
	// TitleSafeArea calls Viewport::GetTitleSafeArea, which is ten bytes
	// building a Rectangle from its four arguments; ToString is a String.Format
	// over the four. Nothing reaches a device, an adapter or a handle.
	//
	// The type is native-SOURCED and pure managed, and those are different
	// facts. Every DisplayMode a consumer can hold was reported by a member
	// that asked CNA -- GraphicsDevice.DisplayMode today -- and that member
	// carries the error. Once the value exists it is three numbers, and a
	// getter over them has no failure mode to report.
	"Microsoft.Xna.Framework.Graphics.DisplayMode": true,

	// Foundation 67. VertexBufferBinding is three private fields and eight
	// members over them: three constructors that validate and store, one
	// op_Implicit that calls the shortest of them, three one-`ldfld` getters,
	// and nothing else. Its constructors READ VertexBuffer::_vertexCount --
	// a managed field on a native-backed type -- which is a managed read, not
	// a device query, so the binding owns no native object and needs no FFI.
	//
	// This is the same native-SOURCED / pure-managed split DisplayMode has,
	// and it is why the entry is admissible: the BUFFER carries the native
	// error, and a struct describing where in it to start does not.
	"Microsoft.Xna.Framework.Graphics.VertexBufferBinding": true,

	// Foundation 73. RenderTargetBinding is the same shape one family over: two
	// private fields, two constructors that validate and store, one
	// op_Implicit over the shorter of them, and two one-`ldfld` getters. It
	// owns no native object -- the TARGET does -- so the same native-SOURCED /
	// pure-managed split admits it.
	"Microsoft.Xna.Framework.Graphics.RenderTargetBinding": true,

	// Foundation 32. Microsoft.Xna.Framework.Game.dll IL
	// (sha256 b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0)
	// shows GameComponent as managed field work throughout. The constructor is
	// 21 bytes of stfld, both getters and get_Game are one ldfld each, both
	// setters are compare-store-announce, Initialize and Update are a bare
	// `ret` of code size 1, Finalize's one branch returns immediately, and the
	// two On... methods are a null test and a delegate Invoke. Nothing in the
	// class reaches a device, a host, a window, or a CNA handle.
	//
	// Its one non-local statement is Dispose(bool)'s
	// `get_Game().get_Components().Remove(this)`, which is a managed
	// collection call -- and the reason the type could not be projected until
	// Game exposed Components.
	"Microsoft.Xna.Framework.GameComponent": true,

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

	// Foundation 46. DrawableGameComponent is pure managed in the same sense
	// GameComponent is, and the distinction is worth stating because the type
	// is the profile's bridge to the graphics runtime and looks native-backed
	// from the outside.
	//
	// It is not. Everything its IL does is managed: three bool/int32 fields,
	// two delegate fields, a GetService lookup out of a managed dictionary,
	// four Delegate.Combine subscriptions, and four method bodies that are a
	// single `ret` each. The one member that reaches a device --
	// get_GraphicsDevice -- does not touch one either: it null-checks a field
	// and forwards to the SERVICE's property, and the service is a consumer's
	// own object. The device it hands back is native; obtaining it is not.
	//
	// Its fallible members are named individually below, exactly as
	// GameComponent's are.
	"Microsoft.Xna.Framework.DrawableGameComponent": true,

	// Foundation 78. The eight XNA exception types. Every one of them declares
	// ONLY constructors, and each constructor is a `base..ctor(...)` and
	// nothing else. Nothing in any of them reaches a device, a runtime or a
	// throw site: the whole family is field stores and string formatting.
	"Microsoft.Xna.Framework.Audio.InstancePlayLimitException":           true,
	"Microsoft.Xna.Framework.Audio.NoAudioHardwareException":             true,
	"Microsoft.Xna.Framework.Audio.NoMicrophoneConnectedException":       true,
	"Microsoft.Xna.Framework.Content.ContentLoadException":               true,
	"Microsoft.Xna.Framework.Graphics.DeviceLostException":               true,
	"Microsoft.Xna.Framework.Graphics.DeviceNotResetException":           true,
	"Microsoft.Xna.Framework.Graphics.NoSuitableGraphicsDeviceException": true,
	"Microsoft.Xna.Framework.Storage.StorageDeviceNotConnectedException": true,

	// Foundation 77. The four stock vertex structs are public fields, a
	// storing constructor, a static readonly VertexDeclaration built from a
	// literal element table, a SmartGetHashCode, a String.Format ToString and
	// three equality members. Nothing in any of them reaches a device: even the
	// static declaration is managed validation over a literal array.
	"Microsoft.Xna.Framework.Graphics.VertexPositionColor":         true,
	"Microsoft.Xna.Framework.Graphics.VertexPositionColorTexture":  true,
	"Microsoft.Xna.Framework.Graphics.VertexPositionNormalTexture": true,
	"Microsoft.Xna.Framework.Graphics.VertexPositionTexture":       true,

	// Foundation 75. GraphicsDeviceInformation is three private fields, three
	// one-`ldfld` getters, three one-`stfld` setters, an equality chain, an XOR
	// hash and a Clone. Nothing in its own IL reaches a device; the constructor
	// reads GraphicsAdapter.DefaultAdapter, which is another type's member and
	// is why exactly two of its projected operations are fallible.
	"Microsoft.Xna.Framework.GraphicsDeviceInformation": true,

	// Foundation 75. PreparingDeviceSettingsEventArgs is one private field, a
	// constructor that stores it with no validation, and a one-`ldfld` getter.
	"Microsoft.Xna.Framework.PreparingDeviceSettingsEventArgs": true,

	// Foundation 74. LaunchParameters declares one constructor, whose body is
	// the base Dictionary constructor and a string parse of
	// Environment.GetCommandLineArgs(). It creates no device, starts no game,
	// and reaches no native code; its base is equally managed in the pinned
	// mscorlib.
	"Microsoft.Xna.Framework.LaunchParameters": true,
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
	// Blockers is why a DEFERRED base is deferred, named to the exact
	// inherited member or the exact architecture decision. A DEFERRED base
	// with no blocker is a defect: "deferred" must be a measured claim about
	// something specific, not a word that postpones the question.
	Blockers []bclBaseBlocker
}

// bclBaseBlocker is one reason a base cannot be projected yet.
//
// SUBSYSTEM means one inherited member's type belongs to a .NET subsystem
// CNA-Go has not mapped, and names both the member and the subsystem.
// ARCHITECTURE means the obstacle is a cross-cutting public-API decision
// rather than a missing type, and the CLRMember is empty because no single
// member carries it.
type bclBaseBlocker struct {
	Kind      string
	CLRMember string
	Needs     string
	Detail    string
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
		Adapter:   "bclexception.State",
		Status:    "COMPOSED",
		Rationale: "modelled by the private internal/bclexception.State adapter; the eight derived XNA exception types hold it in an unexported field and re-expose the eight projected inherited members through measured forwarding. The CLR exception OBJECT and Go's operation-error channel are different contracts, and this base settles the first without touching the second",
	},
	"System.Runtime.InteropServices.ExternalException": {
		Adapter:   "bclexception.State",
		Status:    "COMPOSED",
		Rationale: "the same private adapter as System.Exception, with the ErrorCode member and the ToString override its three XNA subclasses inherit",
	},
	"System.Attribute": {
		Status:    "DEFERRED",
		Rationale: "Go has no attribute metadata; the content serializer attributes need a separate mapping",
		Blockers: []bclBaseBlocker{
			{Kind: "ARCHITECTURE", Needs: "an attribute application model",
				Detail: "Go has no attribute metadata, so a projected attribute type could be constructed and read back but never applied to anything; the five ContentSerializer* types would be inert data objects whose whole purpose, annotating content-pipeline members, is unrepresentable"},
			{Kind: "SUBSYSTEM", CLRMember: "GetCustomAttribute", Needs: "System.Reflection",
				Detail: "the inherited static lookups take Assembly, MemberInfo, Module and ParameterInfo, none of which CNA-Go maps; whether inherited STATICS are part of a derived type's projected surface is itself undecided, having never arisen before"},
			{Kind: "SUBSYSTEM", CLRMember: "TypeId", Needs: "a decision on System.Object-typed members",
				Detail: "the default implementation returns GetType(), so projecting it as any would hand back a reflect.Type through an untyped result"},
		},
	},
	"System.IO.BinaryReader": {
		Status:    "DEFERRED",
		Rationale: "requires a stream-reader base whose seek and encoding behavior is a separate mapping",
		Blockers: []bclBaseBlocker{
			{Kind: "SUBSYSTEM", CLRMember: "BaseStream", Needs: "System.IO.Stream as a seekable stream",
				Detail: "CNA-Go maps System.IO.Stream to io.Reader, which carries neither seeking nor the encoding-aware Read7BitEncodedInt behavior the reader's inherited surface depends on"},
		},
	},
	"System.ComponentModel.ExpandableObjectConverter": {
		Status:    "DEFERRED",
		Rationale: "part of the System.ComponentModel TypeConverter subsystem, which is a separate mapping",
		Blockers: []bclBaseBlocker{
			{Kind: "SUBSYSTEM", CLRMember: "CanConvertFrom", Needs: "System.ComponentModel.ITypeDescriptorContext",
				Detail: "the descriptor-context contract, which is the single largest blocker in the profile at thirteen types"},
			{Kind: "SUBSYSTEM", CLRMember: "ConvertFrom", Needs: "System.Globalization.CultureInfo",
				Detail: "culture-aware conversion, which twelve types depend on"},
			{Kind: "SUBSYSTEM", CLRMember: "GetProperties", Needs: "System.Collections.IDictionary",
				Detail: "shared with System.Exception::Data"},
		},
	},
	"System.Collections.ObjectModel.Collection`1": {
		Adapter:   "collectionBase[T]",
		Status:    "COMPOSED",
		Rationale: "modelled by the private collectionBase[T] adapter; a derived XNA class holds it in an unexported field and re-exposes the eleven inherited public members through measured forwarding, with no exported embedding",
	},
	"System.Collections.ObjectModel.ReadOnlyCollection`1": {
		Status:    "DEFERRED",
		Rationale: "DEFERRED AS A BASE ONLY; it is SUPPORTED as a signature adapter, and the two roles are independent",
		Blockers: []bclBaseBlocker{
			{Kind: "ARCHITECTURE", CLRMember: "GetEnumerator", Needs: "a rule for a derived member that HIDES an inherited one",
				Detail: "all four base consumers declare a GetEnumerator returning their own nested Enumerator, which hides the inherited one. The settled collision rule would resolve that into two hashed names, neither of which is GetEnumerator. The principled answer -- a hidden inherited member is unreachable because CNA-Go projects no base type to cast to -- is available but untested, because every consumer is blocked on its element type anyway"},
			{Kind: "SUBSYSTEM", Needs: "ModelBone, Effect, ModelMesh, ModelMeshPart",
				Detail: "each of the four consumers needs its element type, and all four element types are content-pipeline blocked"},
		},
	},
	// Foundation 74. Five of the six blockers Foundation 29 recorded here were
	// missing Go SPELLINGS, and a spelling is something this milestone can
	// supply: KeyCollection, ValueCollection, KeyValuePair and
	// IEqualityComparer<T> are now signature adapters, and the nested
	// Enumerator projects through the settled List<T>.Enumerator rule.
	// OnDeserialization needed no subsystem at all -- its only non-empty path
	// is unreachable without the `family` serialization constructor, which the
	// CLR does not inherit.
	//
	// The sixth, GetObjectData, is not a spelling problem and does not become
	// one. It stays measured, as the adapter's one
	// BCL_PROJECTION_BLOCKED_EXTERNAL exclusion.
	"System.Collections.Generic.Dictionary`2": {
		Adapter:   "dictionaryBase[TKey, TValue]",
		Status:    "COMPOSED",
		Rationale: "modelled by the private dictionaryBase[TKey, TValue] adapter, which reproduces the reference's buckets/entries/free-list/version structure because enumeration order is public surface; a derived XNA class holds it in an unexported field and re-exposes thirteen of the fourteen inherited public members through measured forwarding, with no exported embedding and emphatically not as a Go map",
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
	// Foundation 79. The third of the three interfaces the stock effects
	// share, and the least uniform. In the same assembly, the three light
	// accessors and both LightingEnabled accessors are one `ldfld` or one
	// `stfld` plus a dirty-flag OR in every shipped implementor, and
	// AmbientLightColor is the same in the three that declare it -- BasicEffect,
	// SkinnedEffect and EnvironmentMapEffect. EnableDefaultLighting is the one
	// that is not: it routes through EffectHelpers::EnableDefaultLighting,
	// twelve DirectionalLight setters, each `callvirt EffectParameter::SetValue`
	// ending in `calli unmanaged stdcall`. See managedFallibleMembers.
	"Microsoft.Xna.Framework.Graphics.IEffectLights": true,

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

	// Foundation 28. IGraphicsDeviceService publishes a device; it does not
	// reach one. Microsoft.Xna.Framework.Game.dll ships exactly one
	// implementor, GraphicsDeviceManager, whose get_GraphicsDevice is
	// `ldarg.0; ldfld device; ret` -- a stored reference handed over, null
	// before a device exists. Nothing in the contract creates, resets, queries
	// or disposes anything.
	//
	// This is the sharpest per-contract split in the profile: the SAME class
	// implements IGraphicsDeviceManager, which stays unclassified and fallible
	// because its CreateDevice, BeginDraw and EndDraw genuinely cross into the
	// runtime. The boundary is read per contract, never per class.
	"Microsoft.Xna.Framework.Graphics.IGraphicsDeviceService": true,

	// Foundation 66. IVertexType publishes a layout; it does not build one.
	// Microsoft.Xna.Framework.Graphics.dll ships FIVE implementors --
	// VertexPositionColor, VertexPositionTexture,
	// VertexPositionColorTexture, VertexPositionNormalTexture and
	// VertexPositionNormalTextureBumpTexture -- and every one of them is the
	// same six bytes:
	//
	//	.override IVertexType::get_VertexDeclaration
	//	  ldsfld VertexDeclaration; ret
	//
	// a static field read of a declaration the type's own `.cctor` built once.
	// No device, no validation, no throw site.
	"Microsoft.Xna.Framework.Graphics.IVertexType": true,
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
	// Foundation 67. VertexBufferBinding is a pure value struct and therefore
	// starts infallible, and all four of its constructing members genuinely
	// throw:
	//
	//	.ctor(VertexBuffer, int32, int32)
	//	  vertexBuffer == null            ArgumentNullException(NullNotAllowed)
	//	  vertexOffset < 0 || >= count    ArgumentOutOfRangeException
	//	  instanceFrequency < 0           ArgumentOutOfRangeException
	//
	// The two shorter constructors drop checks and keep the null one, and
	// op_Implicit is `newobj .ctor(VertexBuffer); ret` -- so it carries exactly
	// that constructor's failure and nothing else. The three GETTERS are field
	// reads and are deliberately absent from this entry.
	"Microsoft.Xna.Framework.Graphics.VertexBufferBinding": {
		"constructor|.ctor":  true,
		"method|op_Implicit": true,
	},
	// Foundation 73. RenderTargetBinding's two constructors and its
	// op_Implicit, on the same evidence:
	//
	//	.ctor(RenderTarget2D)             renderTarget == null  ArgumentNullException
	//	.ctor(RenderTargetCube, face)     renderTarget == null  ArgumentNullException
	//	                                  face out of range     ArgumentOutOfRangeException
	//
	// and op_Implicit is `newobj .ctor(RenderTarget2D); ret`, so it carries
	// exactly that constructor's failure. The two GETTERS are field reads and
	// are deliberately absent.
	"Microsoft.Xna.Framework.Graphics.RenderTargetBinding": {
		"constructor|.ctor":  true,
		"method|op_Implicit": true,
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

	// Foundation 32. GameComponent is pure managed, so it starts from "nothing
	// is fallible" and each entry below names its own evidence. Six members are
	// deliberately absent: the constructor validates nothing, Update and
	// Finalize are observably no-ops, and the three getters are one ldfld each.
	"Microsoft.Xna.Framework.GameComponent": {
		// A bare `ret`, so the class itself never fails -- but IGameComponent
		// declares an error result, on the evidence of its OTHER implementor:
		// DrawableGameComponent.Initialize resolves IGraphicsDeviceService out
		// of Game.Services and throws when it is absent. Go requires an exact
		// signature to satisfy an interface, so the member carries the
		// contract's channel and never uses it. This is the first member in
		// the profile whose fallibility comes from a contract it implements
		// rather than from its own body, and it is recorded as such.
		"method|Initialize": true,
		// Dispose() and Dispose(bool) share one CLR name and one entry.
		// Dispose(bool) runs Game.Components.Remove, which is fallible on the
		// settled collection projection, and then raises Disposed, whose
		// consumer handlers may fail. Dispose() is `Dispose(true)` plus a
		// GC.SuppressFinalize that has nothing to suppress.
		"method|Dispose": true,
		// Both raise sites invoke consumer handlers.
		"method|OnEnabledChanged":     true,
		"method|OnUpdateOrderChanged": true,
		// Only the SETTERS announce. Each is compare, store, then a virtual
		// call to its On... method, so it inherits exactly that method's
		// fallibility -- and a suppressed assignment announces nothing and
		// cannot fail at all.
		"property-set|Enabled":     true,
		"property-set|UpdateOrder": true,
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
	// IEffectLights::EnableDefaultLighting, the same measurement one level out:
	// the operation itself does no runtime work, and every one of the twelve
	// DirectionalLight writes it makes does.
	"Microsoft.Xna.Framework.Graphics.IEffectLights": {
		"method|EnableDefaultLighting": true,
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
	// Foundation 46. DrawableGameComponent starts from "nothing is fallible"
	// and each entry names its own evidence. Six of its members are
	// deliberately absent: the constructor stores two fields and calls the
	// base, Draw, LoadContent and UnloadContent are each a bare `ret`, and the
	// two getters are one ldfld each.
	//
	// The last two entries are INHERITED members. Fallibility is classified
	// against the owner a member is projected on, so a derived type must name
	// the inherited members that fail as well as its own; leaving them out
	// would silently make GameComponent::set_Enabled infallible on
	// DrawableGameComponent and fallible on GameComponent, for the same body.
	"Microsoft.Xna.Framework.DrawableGameComponent": {
		// Resolves IGraphicsDeviceService out of Game.Services and throws
		// InvalidOperationException(MissingGraphicsDeviceService) when it is
		// absent; also calls the base Initialize, which carries
		// IGameComponent's channel.
		"method|Initialize": true,
		// Dispose(bool) removes four consumer-supplied event registrations and
		// then calls GameComponent::Dispose(bool), which removes the component
		// from Game.Components and raises Disposed.
		"method|Dispose": true,
		// Both raise sites invoke consumer handlers.
		"method|OnVisibleChanged":   true,
		"method|OnDrawOrderChanged": true,
		// get_GraphicsDevice throws InvalidOperationException when the service
		// field is still null. Its message is NOT what the resource key
		// suggests: the key is PropertyCannotBeCalledBeforeInitialize and the
		// string reads "The GraphicsDevice property cannot be used before
		// Initialize has been called." There is no setter to classify.
		"property-get|GraphicsDevice": true,
		// Only the SETTERS announce: each is compare, store, then a virtual
		// call to its On... method, so it inherits exactly that method's
		// fallibility. A suppressed assignment announces nothing.
		"property-set|Visible":   true,
		"property-set|DrawOrder": true,
		// The two inherited setters, on GameComponent's own evidence.
		"property-set|Enabled":     true,
		"property-set|UpdateOrder": true,
	},
	// Foundation 75. GraphicsDeviceInformation is pure managed, so it starts
	// from "nothing is fallible", and exactly three of its ten projected
	// operations earn an error.
	//
	// The constructor's second statement is
	// `adapter = GraphicsAdapter.DefaultAdapter`, and CNA-Go's projection of
	// that static enumerates CNA's adapters -- so a program with no live native
	// device cannot answer it, and the reference's own getter is equally free to
	// throw. Clone inherits that failure because its first instruction is
	// `newobj GraphicsDeviceInformation::.ctor()`, whose adapter it then throws
	// away; a Clone that skipped the call would be a different method.
	//
	// set_Adapter is the third, and its guard is a REFERENCE BUG that is
	// reproduced rather than corrected: it tests `this.adapter`, the field it is
	// about to overwrite, and not the `value` it was given. The other five
	// accessors are one `ldfld` or one `stfld` each and are deliberately absent,
	// as are Equals and GetHashCode.
	"Microsoft.Xna.Framework.GraphicsDeviceInformation": {
		"constructor|.ctor":    true,
		"method|Clone":         true,
		"property-set|Adapter": true,
	},

	// Foundation 74. LaunchParameters is pure managed, so it starts from
	// "nothing is fallible", and exactly two of its fifteen projected
	// operations earn an error.
	//
	// The null-key ArgumentNullException that opens Insert, FindEntry and
	// Remove is NOT among them: the base's key type is System.String, which
	// projects to Go string, so `key == null` is unrepresentable and the guard
	// is statically dead -- the same shape as Collection<T>'s dead
	// items.IsReadOnly guard. ContainsKey, Remove, TryGetValue and the indexer
	// SETTER therefore have no reachable failure at all, and set_Item in
	// particular adds a missing key rather than refusing.
	"Microsoft.Xna.Framework.LaunchParameters": {
		// get_Item is the one inherited read that fails, with
		// KeyNotFoundException. Its SETTER is deliberately absent.
		"property-get|Item": true,
		// Add is Insert(..., add: true), whose reachable failure is the
		// ArgumentException a duplicate key raises.
		"method|Add": true,
	},
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
// facades whose reference implementation reaches no runtime boundary. These
// members must not gain a synthetic runtime error result.
//
// Three body shapes qualify, and each entry below names which one it is:
//
//	a managed field read      ldarg.0; ldfld <field>; ret
//	a compile-time constant   ldc.i4.<n>; ret
//	an empty body             ret
//
// Foundation 45 added the last two. They are the same judgement as the first --
// the reference reaches nothing, so there is no failure to report -- and the
// evidence is the same kind: the whole method, in the implementor the selected
// profile ships. A member is admitted here on its BODY, never on the intuition
// that it "should not fail".
var managedStoredMembers = map[string]map[string]bool{
	// GraphicsDeviceManager's nine configuration properties. Every one of the
	// reference's getters is a single `ldfld` -- no validation, no device, no
	// throw site -- and every setter is a store plus `isDeviceDirty = true`,
	// with the two dimension setters also validating their argument.
	//
	// So the GETTERS are listed and the SETTERS deliberately are not. In the
	// reference the value reaches the device later, at ChangeDevice; in CNA-Go
	// the object that runs ChangeDevice is CNA's manager, so a setter has to
	// push and that call can genuinely be refused. Classifying a setter
	// infallible would mean swallowing that.
	//
	// Foundation 48 moved SupportedOrientations from `property|` to
	// `property-get|` for exactly that reason: it used to store a value that
	// reached nothing, so both accessors were correctly infallible; it now
	// pushes like its eight neighbours.
	"Microsoft.Xna.Framework.GraphicsDeviceManager": {
		"property-get|SupportedOrientations":          true,
		"property-get|GraphicsProfile":                true,
		"property-get|PreferredBackBufferFormat":      true,
		"property-get|PreferredBackBufferWidth":       true,
		"property-get|PreferredBackBufferHeight":      true,
		"property-get|PreferredDepthStencilFormat":    true,
		"property-get|IsFullScreen":                   true,
		"property-get|SynchronizeWithVerticalRetrace": true,
		"property-get|PreferMultiSampling":            true,
	},
	// Foundation 30. Game is a hybrid: the native CNA runtime owns the host,
	// the frame loop and the device, so Game stays a native-backed facade and
	// its members are fallible by default. These two are not. In
	// Microsoft.Xna.Framework.Game.dll each getter is the whole method:
	//
	//	get_Components
	//	  ldarg.0; ldfld GameComponentCollection Game::gameComponents; ret
	//	get_Services
	//	  ldarg.0; ldfld GameServiceContainer Game::gameServices; ret
	//
	// Both fields are assigned once during construction -- gameServices from a
	// field initializer, gameComponents right after the base constructor -- and
	// never reassigned anywhere in the class. The getters allocate nothing,
	// validate nothing, reach no host, no window and no device, and have no
	// throw site, so a synthetic error result would be an invented failure
	// mode. Both are get-only in the reference, so there is no setter to
	// classify.
	//
	// Foundation 42 adds four more, on the same evidence and with one extra
	// consequence recorded. Each getter is the whole method:
	//
	//	get_InactiveSleepTime  ldarg.0; ldfld TimeSpan Game::inactiveSleepTime; ret
	//	get_TargetElapsedTime  ldarg.0; ldfld TimeSpan Game::targetElapsedTime; ret
	//	get_IsFixedTimeStep    ldarg.0; ldfld bool     Game::isFixedTimeStep;   ret
	//	get_IsMouseVisible     ldarg.0; ldfld bool     Game::isMouseVisible;    ret
	//
	// Seven bytes each, no validation, no host, no window, no device, no throw
	// site. They are field reads and a synthetic error would be an invented
	// failure mode.
	//
	// Their SETTERS are deliberately NOT listed. In the reference the managed
	// loop reads these fields every frame; in CNA-Go the loop is native, so a
	// setter has to reach it, and that call can genuinely be refused -- from
	// the wrong thread, or with an argument CNA rejects. Classifying a setter
	// infallible would mean swallowing that.
	//
	// Foundation 45 adds Window on the same evidence. Its body is
	//
	//	get_Window
	//	  ldarg.0; ldfld GameHost Game::host; brfalse RET_NULL
	//	  ldarg.0; ldfld GameHost Game::host
	//	  callvirt GameHost::get_Window; ret
	//	  RET_NULL: ldnull; ret
	//
	// which is a null check plus one virtual call to WindowsGameHost::get_Window,
	// and THAT is `ldarg.0; ldfld WindowsGameWindow; ret`. The host field is
	// assigned once, by EnsureHost() from inside the constructor, so for a
	// constructed Game the null branch is unreachable and the whole member is
	// two field reads. It reaches no window, no device and no platform.
	//
	// Foundation 50 adds IsActive, and it is the one entry in this table whose
	// reference body is NOT a field read: get_IsActive is 30 bytes that consult
	// GamerServicesDispatcher.IsInitialized and Guide.IsVisible before reading
	// the field. It is listed anyway, on measured evidence rather than on
	// resemblance. Both statics live in
	// Microsoft.Xna.Framework.GamerServices.dll, IsInitialized is
	// `packetBuffer != null`, and packetBuffer has exactly one stsfld in that
	// whole assembly -- inside GamerServicesDispatcher.Initialize, a method on
	// a type CNA-Go projects no part of. The branch is unreachable for every
	// expressible CNA-Go program, so what remains is `ldfld isActive`: no
	// validation, no host, no window, no device, no throw site.
	//
	// Foundation 63 adds Content, the plainest entry in the table:
	//
	//	get_Content
	//	  ldarg.0; ldfld ContentManager Game::content; ret
	//
	// seven bytes, one field read. The field is assigned in the constructor and
	// by set_Content, and NOTHING about reading it reaches the content
	// pipeline: the ContentManager it hands back defers its own native manager
	// to the first load. Its SETTER is not listed and stays fallible -- it
	// throws ArgumentNullException for a null value, which is a real reference
	// failure, not a synthetic one.
	"Microsoft.Xna.Framework.Game": {
		"property-get|Content":           true,
		"property-get|Components":        true,
		"property-get|Services":          true,
		"property-get|InactiveSleepTime": true,
		"property-get|TargetElapsedTime": true,
		"property-get|IsFixedTimeStep":   true,
		"property-get|IsMouseVisible":    true,
		"property-get|Window":            true,
		"property-get|IsActive":          true,
		// Foundation 76. Game::ShowMissingRequirementMessage is
		// `host == null ? false : host.ShowMissingRequirementMessage(e)`, and
		// both of WindowsGameHost's dialog branches are selected by an `isinst`
		// against an exception type no consumer can yet construct. The
		// reachable body is the base host's `ldc.i4.0; ret` -- a compile-time
		// constant, which is one of the three shapes this table admits.
		"method|ShowMissingRequirementMessage": true,
		// Foundation 74. Game::get_LaunchParameters is
		// `ldarg.0; ldfld launchParameters; ret` -- one field read of an object
		// the constructor allocated, reaching no host and no runtime.
		"property-get|LaunchParameters": true,
	},
	// Foundation 45. GameWindow is native-backed -- most of its members reach
	// the platform window -- so it starts fallible and these three name their
	// own evidence. Each is the WHOLE method:
	//
	//	GameWindow::get_Title
	//	  ldarg.0; ldfld string GameWindow::title; ret
	//
	// a field read on the ABSTRACT BASE, which every implementor shares. Its
	// setter is deliberately absent from this table: set_Title calls the
	// abstract SetTitle, which reaches the platform window.
	//
	//	WindowsGameWindow::get_CurrentOrientation
	//	  ldc.i4.0; ret
	//
	// a constant. In the selected Windows runtime profile the reference never
	// asks the platform for an orientation; it answers
	// DisplayOrientation.Default unconditionally. A projection that could fail
	// here would be reporting a failure the reference cannot have, and one that
	// queried CNA could disagree with the reference while being "right".
	//
	//	WindowsGameWindow::SetSupportedOrientations
	//	  ret
	//
	// an empty body. This is the member that looked like GameWindow's one open
	// question, and the answer is that the reference does nothing: it is not a
	// CNA gap, and CNA's only orientation setter belongs to
	// GraphicsDeviceManager rather than to the window.
	"Microsoft.Xna.Framework.GameWindow": {
		"property-get|Title":              true,
		"property-get|CurrentOrientation": true,
		"method|SetSupportedOrientations": true,
	},
	// Foundation 56. The graphics base chain. GraphicsResource is native-backed
	// -- it holds the resource handle and its Dispose destroys it -- so it
	// starts fallible and these six name their own evidence.
	//
	//	get_GraphicsDevice  ldarg.0; ldfld _parent; ret
	//	get_IsDisposed      ldarg.0; ldfld isDisposed; ret
	//
	// Two field reads, seven bytes each, no validation and no throw site.
	//
	// Name and Tag are listed with BOTH accessors, which is unusual here and is
	// measured rather than assumed. Their bodies are conditional:
	//
	//	get_Name  if (_internalHandle != 0)
	//	              return _parent.Resources.GetCachedName(_internalHandle);
	//	          return _localName;
	//
	// and the branch that looks like it reaches something does not.
	// DeviceResourceManager::GetCachedName is 79 bytes of
	// `Dictionary<ulong, ResourceData>` under a `Monitor`, answering
	// `String.Empty` for an absent key; SetCachedName is its store. Both are
	// managed, allocate nothing, reach no D3D call and have no throw site. The
	// cache is per-resource storage reached indirectly, so a setter here pushes
	// nothing to a device and cannot be refused -- which is exactly why the
	// GraphicsDeviceManager setters above are NOT listed and these are.
	//
	// CNA has counterpart routes -- cna_graphics_resource_set_name, copy_name,
	// get_tag and set_tag -- and Foundation 56 wrongly said it did not.
	// Foundation 57 measured them and left them unbound, per route, in
	// tools/native_abi's deliberatelyUnboundRoutes: they refuse a SpriteBatch
	// handle, which is a GraphicsResource in this contract, and set_name
	// validates UTF-8 where set_Name validates nothing. Neither changes this
	// classification, which is read off the REFERENCE's body.
	"Microsoft.Xna.Framework.Graphics.GraphicsResource": {
		"property-get|GraphicsDevice": true,
		"property-get|IsDisposed":     true,
		"property-get|Name":           true,
		"property-set|Name":           true,
		"property-get|Tag":            true,
		"property-set|Tag":            true,
	},
	// Texture's whole public surface, and both members are the same shape:
	//
	//	get_Format      ldarg.0; ldfld _format; ret
	//	get_LevelCount  ldarg.0; ldfld _levelCount; ret
	//
	// Both fields are filled once by Texture::InitializeDescription during
	// construction and never written again. Neither getter checks disposal --
	// the reference answers after Dispose and so does CNA-Go.
	"Microsoft.Xna.Framework.Graphics.Texture": {
		"property-get|Format":     true,
		"property-get|LevelCount": true,
	},
	// Foundation 64. VertexDeclaration's two declared members. Neither reaches
	// anything, and the type as a whole reaches nothing native:
	//
	//	get_VertexStride   ldarg.0; ldfld _vertexStride; ret
	//	GetVertexElements  ldarg.0; ldfld _elements; Array.Clone(); castclass; ret
	//
	// The second is listed even though it is not a bare field read, because the
	// rule is about reaching a runtime boundary rather than about instruction
	// count: `Array.Clone` is a managed allocation with no throw site a caller
	// can provoke. The CONSTRUCTORS are of course fallible and are not listed;
	// VertexElementValidator is their whole failure surface.
	"Microsoft.Xna.Framework.Graphics.VertexDeclaration": {
		"property-get|VertexStride": true,
		"method|GetVertexElements":  true,
	},
	// Foundation 65. IndexBuffer's three properties. Two are field reads and
	// the third is a comparison over one:
	//
	//	get_IndexCount        ldarg.0; ldfld _indexCount; ret
	//	get_IndexElementSize  _indexSize == 2 ? SixteenBits : ThirtyTwoBits
	//	get_BufferUsage       ConvertDxBufferUsageToXna(_usage) -- a switch
	//
	// None reaches D3D and none checks disposal, so all three answer after
	// Dispose in the reference and here. The two CONVERSIONS are the reason
	// CNA-Go records what CNA applied at construction rather than asking CNA
	// per call: a getter that asked would be fallible, and the reference's is
	// not.
	"Microsoft.Xna.Framework.Graphics.IndexBuffer": {
		"property-get|IndexCount":       true,
		"property-get|IndexElementSize": true,
		"property-get|BufferUsage":      true,
	},
	// Foundation 66. VertexBuffer's three, on the same evidence as
	// IndexBuffer's. get_VertexDeclaration is the plainest: it hands back the
	// CALLER'S object, stored by the constructor, so it is one `ldfld` over a
	// reference the buffer does not own.
	"Microsoft.Xna.Framework.Graphics.VertexBuffer": {
		"property-get|VertexCount":       true,
		"property-get|VertexDeclaration": true,
		"property-get|BufferUsage":       true,
	},
	// Foundation 67. GraphicsDevice's two BOUND-BUFFER readers, and they are
	// the first entries this native-backed type gets that are not about a value
	// it asked CNA for:
	//
	//	get_Indices       ldarg.0; ldfld _currentIB; ret
	//	GetVertexBuffers  new VertexBufferBinding[currentVertexBufferCount];
	//	                  Array.Copy(currentVertexBuffers, copy, count); ret
	//
	// Both answer from managed fields the SETTERS maintain. That is not a
	// convenience: CNA hands back a handle, and a handle cannot be turned into
	// the Go object a consumer is holding without a registry that would retain
	// every buffer for the life of the process. The reference keeps the same
	// two fields for the same reason -- it must give back the object the caller
	// bound, not an equivalent one.
	//
	// The SETTERS are not listed and stay fallible: binding is what actually
	// changes the device, and it reaches CNA.
	"Microsoft.Xna.Framework.Graphics.GraphicsDevice": {
		"property-get|Indices":    true,
		"method|GetVertexBuffers": true,
		// Foundation 73. GetRenderTargets is GetVertexBuffers' twin:
		//
		//	if (currentRenderTargetCount == 0) return emptyRenderTargetBindings;
		//	copy = new RenderTargetBinding[count]; Array.Copy(...); return copy;
		//
		// a managed count and a managed array copy, with no device in it. The
		// SETTERS stay fallible: binding is what reaches CNA.
		"method|GetRenderTargets": true,
	},
	// Foundation 68. GraphicsAdapter's eleven readers. Every one is one `ldfld`
	// in the reference over a value D3D9 enumeration filled once, and CNA-Go
	// takes the same snapshot once -- at enumeration, in readAdapter -- so a
	// getter here reaches nothing either.
	//
	// The three members that ASK the adapter again are deliberately absent:
	// IsProfileSupported and the two Query... members reach CNA per call and
	// carry its error, which is where this projection genuinely differs from
	// the reference's cached capability bits.
	"Microsoft.Xna.Framework.Graphics.GraphicsAdapter": {
		"property-get|Description":           true,
		"property-get|DeviceName":            true,
		"property-get|VendorId":              true,
		"property-get|DeviceId":              true,
		"property-get|SubSystemId":           true,
		"property-get|Revision":              true,
		"property-get|IsDefaultAdapter":      true,
		"property-get|IsWideScreen":          true,
		"property-get|CurrentDisplayMode":    true,
		"property-get|SupportedDisplayModes": true,
		"property-get|MonitorHandle":         true,
	},
	// Foundation 72. The Effect cluster's thirty-six readers.
	//
	// Every one is a field read over state the reference's CONSTRUCTOR filled
	// from D3DX reflection, and the whole graph is built before anyone can read
	// it -- which is exactly what CNA-Go does, once, at construction. So none
	// of them reaches a runtime in either implementation.
	//
	// The four collections' members are the same managed list walk
	// DisplayModeCollection's already are: get_Count is one forwarded
	// List<T>.Count, both indexers are bounds-checked reads that answer NULL
	// rather than throwing, GetEnumerator forwards to the list's own, and
	// GetParameterBySemantic is a linear scan with String.Compare.
	//
	// Deliberately ABSENT and therefore fallible: every SetValue, every
	// GetValue*, EffectPass::Apply, Effect::Clone, Effect::OnApply and
	// set_CurrentTechnique. Each of those reaches D3DX in the reference and
	// CNA here, and each carries a real refusal -- InvalidCastException from a
	// value member, InvalidOperationException from Apply and the technique
	// setter, ObjectDisposedException from both.
	"Microsoft.Xna.Framework.Graphics.Effect": {
		"property-get|Parameters":       true,
		"property-get|Techniques":       true,
		"property-get|CurrentTechnique": true,
	},
	"Microsoft.Xna.Framework.Graphics.EffectTechnique": {
		"property-get|Name":        true,
		"property-get|Passes":      true,
		"property-get|Annotations": true,
	},
	"Microsoft.Xna.Framework.Graphics.EffectPass": {
		"property-get|Name":        true,
		"property-get|Annotations": true,
	},
	"Microsoft.Xna.Framework.Graphics.EffectParameter": {
		"property-get|Name":             true,
		"property-get|Semantic":         true,
		"property-get|RowCount":         true,
		"property-get|ColumnCount":      true,
		"property-get|ParameterClass":   true,
		"property-get|ParameterType":    true,
		"property-get|Elements":         true,
		"property-get|StructureMembers": true,
		"property-get|Annotations":      true,
	},
	"Microsoft.Xna.Framework.Graphics.EffectAnnotation": {
		"property-get|Name":           true,
		"property-get|Semantic":       true,
		"property-get|RowCount":       true,
		"property-get|ColumnCount":    true,
		"property-get|ParameterClass": true,
		"property-get|ParameterType":  true,
	},
	"Microsoft.Xna.Framework.Graphics.EffectTechniqueCollection": {
		"property-get|Count":   true,
		"property-get|Item":    true,
		"method|GetEnumerator": true,
	},
	"Microsoft.Xna.Framework.Graphics.EffectPassCollection": {
		"property-get|Count":   true,
		"property-get|Item":    true,
		"method|GetEnumerator": true,
	},
	"Microsoft.Xna.Framework.Graphics.EffectParameterCollection": {
		"property-get|Count":            true,
		"property-get|Item":             true,
		"method|GetEnumerator":          true,
		"method|GetParameterBySemantic": true,
	},
	"Microsoft.Xna.Framework.Graphics.EffectAnnotationCollection": {
		"property-get|Count":   true,
		"property-get|Item":    true,
		"method|GetEnumerator": true,
	},
	// Foundation 71. TextureCube's one and Texture3D's three. Every one is
	//
	//	get_Size    ldarg.0; ldfld _size;   ret
	//	get_Width   ldarg.0; ldfld _width;  ret
	//	get_Height  ldarg.0; ldfld _height; ret
	//	get_Depth   ldarg.0; ldfld _depth;  ret
	//
	// seven bytes each over a field the constructor's InitializeDescription
	// filled once from the CREATED surface. None checks disposal, so all four
	// answer after Dispose -- which is exactly the evidence Texture2D's Width
	// and Height already rest on, on the same base and the same helper.
	"Microsoft.Xna.Framework.Graphics.TextureCube": {
		"property-get|Size": true,
	},
	"Microsoft.Xna.Framework.Graphics.Texture3D": {
		"property-get|Width":  true,
		"property-get|Height": true,
		"property-get|Depth":  true,
	},
	// Foundation 69. SpriteFont's four readers.
	//
	//	get_LineSpacing       ldarg.0; ldfld lineSpacing; ret
	//	get_Spacing           ldarg.0; ldfld spacing; ret
	//	get_DefaultCharacter  ldarg.0; ldfld defaultCharacter; ret
	//	get_Characters        the lazily built ReadOnlyCollection<char>, cached
	//
	// The first three are seven bytes each over a managed field. The fourth
	// builds a view over another managed field on first call and stores it, so
	// the second call is one more `ldfld` -- and neither branch reaches a
	// runtime.
	//
	// The three SETTERS are deliberately absent and stay fallible, for the
	// reason Game's `stfld` setters are: each has to reach CNA as well, because
	// cna_sprite_batch_draw_string lays text out from the NATIVE font's values.
	// A managed-only store would let a drawn string disagree with a measured
	// one.
	//
	// MeasureString is absent too, and for a DIFFERENT reason -- it is not a
	// stored read at all. Its body throws a real ArgumentException for a
	// character the font has no glyph for and no default character to
	// substitute, so its error channel carries the reference's own refusal.
	"Microsoft.Xna.Framework.Graphics.SpriteFont": {
		"property-get|LineSpacing":      true,
		"property-get|Spacing":          true,
		"property-get|DefaultCharacter": true,
		"property-get|Characters":       true,
	},
	// DisplayModeCollection's two members are a managed list walk and a
	// managed filter over it. Its constructor is `assembly`, so every
	// collection a consumer can hold was built from a snapshot that is never
	// mutated afterwards -- there is no version check to fail and no
	// enumeration error to report.
	"Microsoft.Xna.Framework.Graphics.DisplayModeCollection": {
		"property-get|Item":    true,
		"method|GetEnumerator": true,
	},
	// Foundation 79. BasicEffect's managed state, which is nearly all of it.
	//
	// Every property listed below is one `ldfld` to read and one `stfld` plus a
	// dirty-flag `or` to write, seven bytes and twenty-two or twenty-three
	// respectively, with five of the setters carrying a `beq` early return
	// instead. None reaches a device, a parameter or a throw site: the push
	// happens later, in OnApply, which is fallible and is deliberately absent
	// from this list.
	//
	// The FOUR that are absent are absent because the reference really does
	// reach the effect for them -- SpecularColor and SpecularPower through
	// specularColorParam/specularPowerParam, FogColor through fogColorParam and
	// Texture through textureParam, each `callvirt EffectParameter::SetValue`
	// or a GetValue counterpart ending in `calli unmanaged stdcall`. So is
	// EnableDefaultLighting, for the same reason one level out, and so are both
	// constructors, whose tails call SpecularColor and SpecularPower.
	//
	// The three DirectionalLight properties are get-only in the contract and
	// their getters are `ldfld` of a field CacheEffectParameters filled once.
	"Microsoft.Xna.Framework.Graphics.BasicEffect": {
		"property|World":                  true,
		"property|View":                   true,
		"property|Projection":             true,
		"property|DiffuseColor":           true,
		"property|EmissiveColor":          true,
		"property|Alpha":                  true,
		"property|LightingEnabled":        true,
		"property|PreferPerPixelLighting": true,
		"property|AmbientLightColor":      true,
		"property|FogEnabled":             true,
		"property|FogStart":               true,
		"property|FogEnd":                 true,
		"property|TextureEnabled":         true,
		"property|VertexColorEnabled":     true,
		"property-get|DirectionalLight0":  true,
		"property-get|DirectionalLight1":  true,
		"property-get|DirectionalLight2":  true,
	},
	// Foundation 80. AlphaTestEffect's managed state, which is fourteen of its
	// sixteen members. The two that are absent are FogColor and Texture, the
	// only two whose reference bodies reach an EffectParameter -- 12 bytes to
	// read and 13 to write, against 7 and 22-or-23 for every other accessor.
	//
	// AlphaFunction and ReferenceAlpha are listed and are worth naming: both
	// raise the AlphaTest flag, which no other stock effect uses, and NEITHER
	// validates. An undeclared CompareFunction and a reference alpha outside
	// 0..255 are stored and reported back, so there is no refusal to carry.
	"Microsoft.Xna.Framework.Graphics.AlphaTestEffect": {
		"property|World":              true,
		"property|View":               true,
		"property|Projection":         true,
		"property|DiffuseColor":       true,
		"property|Alpha":              true,
		"property|FogEnabled":         true,
		"property|FogStart":           true,
		"property|FogEnd":             true,
		"property|VertexColorEnabled": true,
		"property|AlphaFunction":      true,
		"property|ReferenceAlpha":     true,
	},
	// DualTextureEffect's ten, on the same measurement. THREE of its thirteen
	// cross rather than two, because it has a second texture layer, and all
	// three are absent from this list.
	"Microsoft.Xna.Framework.Graphics.DualTextureEffect": {
		"property|World":              true,
		"property|View":               true,
		"property|Projection":         true,
		"property|DiffuseColor":       true,
		"property|Alpha":              true,
		"property|FogEnabled":         true,
		"property|FogStart":           true,
		"property|FogEnd":             true,
		"property|VertexColorEnabled": true,
	},
	// Foundation 81. EnvironmentMapEffect's fourteen managed members. The six
	// that are absent all reach an EffectParameter: FogColor, Texture,
	// EnvironmentMap, EnvironmentMapAmount, EnvironmentMapSpecular and
	// FresnelFactor -- the most of any stock effect.
	//
	// LightingEnabled is absent for a different reason: it is not a declared
	// member of this type at all. Both accessors are EXPLICIT implementations
	// of IEffectLights, so they are interface witnesses and take the
	// interface's own measured fallibility.
	"Microsoft.Xna.Framework.Graphics.EnvironmentMapEffect": {
		"property|World":                 true,
		"property|View":                  true,
		"property|Projection":            true,
		"property|DiffuseColor":          true,
		"property|EmissiveColor":         true,
		"property|Alpha":                 true,
		"property|AmbientLightColor":     true,
		"property|FogEnabled":            true,
		"property|FogStart":              true,
		"property|FogEnd":                true,
		"property-get|DirectionalLight0": true,
		"property-get|DirectionalLight1": true,
		"property-get|DirectionalLight2": true,
	},
	// SkinnedEffect's fifteen. Four of its members reach a parameter --
	// SpecularColor, SpecularPower, FogColor and Texture -- and three more are
	// absent for reasons of their own:
	//
	//	WeightsPerVertex   the SETTER validates {1,2,4} and throws
	//	                   ArgumentOutOfRangeException, so only the getter is
	//	                   listed and the accessor-level key says so
	//	SetBoneTransforms  writes bonesParam
	//	GetBoneTransforms  reads it, and validates its count twice
	"Microsoft.Xna.Framework.Graphics.SkinnedEffect": {
		"property|World":                  true,
		"property|View":                   true,
		"property|Projection":             true,
		"property|DiffuseColor":           true,
		"property|EmissiveColor":          true,
		"property|Alpha":                  true,
		"property|PreferPerPixelLighting": true,
		"property|AmbientLightColor":      true,
		"property|FogEnabled":             true,
		"property|FogStart":               true,
		"property|FogEnd":                 true,
		"property-get|DirectionalLight0":  true,
		"property-get|DirectionalLight1":  true,
		"property-get|DirectionalLight2":  true,
		"property-get|WeightsPerVertex":   true,
	},
	// DirectionalLight is the mirror image, and it is the reason the entries
	// below are accessor-level rather than whole-property.
	//
	//	get_Enabled        ldarg.0; ldfld enabled;             ret
	//	get_Direction      ldarg.0; ldfld cachedDirection;     ret
	//	get_DiffuseColor   ldarg.0; ldfld cachedDiffuseColor;  ret
	//	get_SpecularColor  ldarg.0; ldfld cachedSpecularColor; ret
	//
	// All four getters are seven bytes over a cache. All four SETTERS write
	// through an EffectParameter -- set_Enabled writes two of them -- so all
	// four are fallible and none is listed. The CONSTRUCTOR is not listed
	// either: its no-clone arm calls three of those setters.
	"Microsoft.Xna.Framework.Graphics.DirectionalLight": {
		"property-get|Enabled":       true,
		"property-get|Direction":     true,
		"property-get|DiffuseColor":  true,
		"property-get|SpecularColor": true,
	},
	// Texture2D's three geometry members, on the same evidence, and correcting
	// a claim CNA-Go made without it. Their bodies are:
	//
	//	get_Width   ldarg.0; ldfld _width; ret
	//	get_Height  ldarg.0; ldfld _height; ret
	//	get_Bounds  newobj Rectangle::.ctor(0, 0, _width, _height)
	//
	// `_width` and `_height` are stored by Texture2D::InitializeDescription
	// from the CREATED surface's D3DSURFACE_DESC, once, at construction. No
	// getter consults the texture again and none checks disposal.
	//
	// CNA-Go used to project all three fallible, on a comment claiming they
	// "read a disposed-checked native texture". They do not, in either runtime:
	// CNA-Go caches CNA's reported description at construction exactly as the
	// reference caches D3D's, and both then answer from a managed field.
	"Microsoft.Xna.Framework.Graphics.Texture2D": {
		"property-get|Width":  true,
		"property-get|Height": true,
		"property-get|Bounds": true,
	},
	// Foundation 59. BlendState's twelve value getters. Every one is
	//
	//	get_ColorSourceBlend  ldarg.0; ldfld cachedColorSourceBlend; ret
	//
	// seven bytes over a managed field, with no device and no throw site. Their
	// SETTERS are deliberately absent: each begins with ThrowIfBound(), which
	// raises a real InvalidOperationException once the state object is bound,
	// so a setter's error channel carries the reference's own refusal rather
	// than an invented one.
	//
	// The CONSTRUCTOR is here too, which no other entry in this table is. Its
	// whole body is `SetDefaults(); isBound = false;` under a try/fault that
	// disposes a half-built object -- and SetDefaults is thirteen stores each
	// preceded by a ThrowIfBound that cannot fire, because isBound is still
	// false on an object nothing has seen. Nothing in it can fail, and Go has no
	// half-built object for the fault handler to clean up.
	"Microsoft.Xna.Framework.Graphics.BlendState": {
		"constructor|.ctor":                  true,
		"property-get|ColorSourceBlend":      true,
		"property-get|ColorDestinationBlend": true,
		"property-get|ColorBlendFunction":    true,
		"property-get|AlphaSourceBlend":      true,
		"property-get|AlphaDestinationBlend": true,
		"property-get|AlphaBlendFunction":    true,
		"property-get|ColorWriteChannels":    true,
		"property-get|ColorWriteChannels1":   true,
		"property-get|ColorWriteChannels2":   true,
		"property-get|ColorWriteChannels3":   true,
		"property-get|BlendFactor":           true,
		"property-get|MultiSampleMask":       true,
	},
	// The other three state objects, on identical evidence: every getter is one
	// `ldfld`, every constructor is `SetDefaults(); isBound = false`, and every
	// SETTER is absent from this table because ThrowIfBound is a real refusal.
	"Microsoft.Xna.Framework.Graphics.DepthStencilState": {
		"constructor|.ctor":                                   true,
		"property-get|DepthBufferEnable":                      true,
		"property-get|DepthBufferWriteEnable":                 true,
		"property-get|DepthBufferFunction":                    true,
		"property-get|StencilEnable":                          true,
		"property-get|StencilFunction":                        true,
		"property-get|StencilPass":                            true,
		"property-get|StencilFail":                            true,
		"property-get|StencilDepthBufferFail":                 true,
		"property-get|TwoSidedStencilMode":                    true,
		"property-get|CounterClockwiseStencilFunction":        true,
		"property-get|CounterClockwiseStencilPass":            true,
		"property-get|CounterClockwiseStencilFail":            true,
		"property-get|CounterClockwiseStencilDepthBufferFail": true,
		"property-get|StencilMask":                            true,
		"property-get|StencilWriteMask":                       true,
		"property-get|ReferenceStencil":                       true,
	},
	"Microsoft.Xna.Framework.Graphics.RasterizerState": {
		"constructor|.ctor":                 true,
		"property-get|CullMode":             true,
		"property-get|FillMode":             true,
		"property-get|ScissorTestEnable":    true,
		"property-get|MultiSampleAntiAlias": true,
		"property-get|DepthBias":            true,
		"property-get|SlopeScaleDepthBias":  true,
	},
	"Microsoft.Xna.Framework.Graphics.SamplerState": {
		"constructor|.ctor":                    true,
		"property-get|Filter":                  true,
		"property-get|AddressU":                true,
		"property-get|AddressV":                true,
		"property-get|AddressW":                true,
		"property-get|MaxAnisotropy":           true,
		"property-get|MaxMipLevel":             true,
		"property-get|MipMapLevelOfDetailBias": true,
	},
	// Foundation 58. RenderTarget2D's three description properties are each two
	// field reads through the helper the constructor built:
	//
	//	get_DepthStencilFormat  ldfld helper; ldfld RenderTargetHelper::depthFormat
	//	get_MultiSampleCount    ldfld helper; ldfld RenderTargetHelper::multiSampleCount
	//	get_RenderTargetUsage   ldfld helper; ldfld RenderTargetHelper::usage
	//
	// Twelve bytes each, no validation, no device and no throw site. The values
	// are what GraphicsAdapter::QueryFormat SELECTED, stored once at
	// construction, so a synthetic error would be an invented failure mode.
	//
	// IsContentLost is deliberately NOT here. Its body is
	//
	//	if (!_contentLost) _contentLost = _parent.IsDeviceLost;
	//
	// and GraphicsDevice::get_IsDeviceLost reaches D3D. CNA-Go asks CNA the
	// same question, and that call can be refused.
	// Foundation 73. RenderTargetCube's three, on RenderTarget2D's own
	// evidence: the same RenderTargetHelper, the same two field reads each, and
	// IsContentLost absent for the same reason.
	"Microsoft.Xna.Framework.Graphics.RenderTargetCube": {
		"property-get|DepthStencilFormat": true,
		"property-get|MultiSampleCount":   true,
		"property-get|RenderTargetUsage":  true,
	},
	"Microsoft.Xna.Framework.Graphics.RenderTarget2D": {
		"property-get|DepthStencilFormat": true,
		"property-get|MultiSampleCount":   true,
		"property-get|RenderTargetUsage":  true,
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
			PublicCLRMembers: publicCLRMemberCount(*t),
			SourceMembers:    len(t.Members),
		}
	}

	var allMembers []*expectedMember
	for i := range c.Types {
		t := &c.Types[i]
		owner := s.typeForXNA(t.Name)
		// The XNA-inherited members are resolved BEFORE anything is mapped,
		// because they share the derived type's overload namespace: a derived
		// class that declares one overload of an inherited name does not
		// privilege it, and both spellings have to agree about that.
		xnaInheritedSource := inheritedXNAPublicMembers(byIdentity, *t)
		groups := overloadGroupsWithXNAInherited(*t, xnaInheritedSource)
		for j := range t.Members {
			m := &t.Members[j]
			if _, blocked := blockedDeclaredMembers[memberIdentity(t.Name, *m)]; blocked {
				owner.BlockedDeclaredMembers++
				s.BlockedDeclaredMembers++
				continue
			}
			mapped := mapMember(s, byIdentity, owner, *t, *t, *m, groups)
			allMembers = append(allMembers, mapped...)
		}
		inherited := mapInheritedBaseMembers(s, byIdentity, owner, *t)
		owner.BCLInheritedCLRMembers = inheritedCLRMemberCount(*t)
		owner.BCLInheritedProjections = len(inherited)
		s.BCLInheritedCLRMembers += owner.BCLInheritedCLRMembers
		s.BCLInheritedProjections += owner.BCLInheritedProjections
		allMembers = append(allMembers, inherited...)
		xnaInherited := mapInheritedXNABaseMembers(s, byIdentity, owner, *t, xnaInheritedSource, groups)
		owner.XNAInheritedCLRMembers = len(xnaInheritedSource)
		owner.XNAInheritedProjections = len(xnaInherited)
		owner.XNAInheritedOverriddenMembers = xnaOverriddenInheritedCount(byIdentity, *t)
		s.XNAInheritedCLRMembers += owner.XNAInheritedCLRMembers
		s.XNAInheritedProjections += owner.XNAInheritedProjections
		s.XNAInheritedOverriddenMembers += owner.XNAInheritedOverriddenMembers
		allMembers = append(allMembers, xnaInherited...)
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
	buildXNABaseSubstitutability(s, c)
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

// mapMember projects one CLR member. `t` is the type the member is projected
// ONTO -- the owner of the resulting Go identity -- and `declaring` is the type
// whose body the member actually has. The two differ for an inherited member,
// and only fallibility reads `declaring`: whether a member reaches a runtime
// boundary is a property of ITS OWN body, so a base member classified managed
// stored stays managed stored on every derived type that inherits it, without
// being registered once per derived type.
func mapMember(s *expectedSurface, byIdentity map[string]*contractType, owner *expectedType, t, declaring contractType, m contractMember, groups map[string]int) []*expectedMember {
	xna := memberIdentity(t.Name, m)
	base := &expectedMember{XNA: xna, Owner: t.Name, SourceKind: m.Kind, SourceAccess: m.Access, PackagePath: owner.PackagePath, Receiver: owner.GoName}
	parameters, outResults, hasDirection := mapParametersWithGenerics(s, byIdentity, owner, m.GenericParameters, m.Parameters)
	// The direction registry is keyed by the DECLARING member's identity, for
	// the same reason fallibility is: which way a stream is used is a property
	// of the body, and an inherited SaveAsPng writes exactly as the declared one
	// does. Keying it by the derived identity would silently hand a
	// RenderTarget2D an io.Reader it cannot write to.
	base.Parameters = applyStreamDirection(memberIdentity(declaring.Name, m), parameters)
	base.Results = mapReturn(s, byIdentity, owner, m.GenericParameters, m.ReturnType)
	base.Results = append(base.Results, outResults...)
	if isFallible(declaring, m, "") {
		base.Results = append(base.Results, "error")
		base.ErrorAdded = true
	}
	shape := parameterShapeWithGenerics(m.GenericParameters, m.Parameters)

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
			// The accessor is classified from scratch, exactly as its results
			// are rebuilt from scratch just above. Inheriting the clone's flag
			// would make the accessor-level decision one-directional: a key
			// could raise fallibility on a pure-managed owner but never lower
			// it on a native-backed one, so a get-only stored property such as
			// Game::Components would keep a synthetic error its IL cannot
			// produce while its results no longer carried one.
			get.ErrorAdded = isFallible(declaring, m, "get")
			if get.ErrorAdded {
				get.Results = append(get.Results, "error")
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
			// The setter's VALUE is a parameter position, so the
			// substitutable-base rule applies to it exactly as it does to a
			// method argument: `device.Textures[0] = myTexture2D` is legal in
			// C# because the indexer's value is typed `Texture`. The GETTER
			// keeps the concrete pointer -- see Texture2DReference for why a
			// return does.
			set.Parameters = append(mapIndexerParameters(s, byIdentity, owner, m.Parameters),
				applySubstitutableParameter(s, owner, valueOrEmpty(m.Type), mappedType))
			set.Results = nil
			set.Accessor = "set"
			set.ErrorAdded = isFallible(declaring, m, "set")
			if set.ErrorAdded {
				set.Results = []string{"error"}
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
					if isFallible(declaring, m, "get") {
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
		// # The generic-method projection rule
		//
		// Go METHODS CANNOT DECLARE TYPE PARAMETERS. That is a language rule,
		// not a limitation of this binding: `func (t *Texture2D) SetData[T any]`
		// does not compile and no arrangement of receivers makes it. A CLR
		// generic instance method therefore has no method-shaped projection at
		// all, and the settled response to "this member cannot be a method
		// here" is already in the profile -- the cross-package cycle rule turns
		// such a member into a package-level function whose FIRST parameter is
		// the receiver.
		//
		// So a generic method projects as a package-level generic FUNCTION,
		// named <Owner><Member> with the usual overload suffix, taking the
		// receiver first. A generic STATIC method already had that shape and
		// keeps it.
		//
		// The type parameter itself is named by the member: `!!0` is an IL
		// token meaning "this method's first type parameter", and its name is
		// in genericParameters. Before Foundation 54 nothing resolved it and
		// the suffix builder produced `SliceOf0`, a name for a position rather
		// than a type.
		//
		// The receiver-first parameter is a PARAMETER POSITION whose CLR type is
		// the owner, so the substitutable-base rule applies to it: a CLR
		// `renderTarget.SetData(...)` is legal, and the Go function that stands
		// in for that call must accept a RenderTarget2D. Foundation 58 is what
		// made that reachable -- before it, no derived type was projected and
		// the concrete pointer was exactly sufficient.
		if len(m.GenericParameters) > 0 && !m.Static {
			base.Receiver = ""
			base.GoKind = "func"
			receiver := "*" + owner.GoName
			if name, substitutable := substitutableBases[declaring.Name]; substitutable {
				receiver = name
			}
			base.Parameters = append([]string{receiver}, base.Parameters...)
			base.GenericMethod = true
		}
		if op, ok := operatorNames[m.Name]; ok {
			base.GoName = owner.GoName + "Operator" + op + "By" + shape
			base.GoKind, base.Receiver, base.OverloadMapped = "func", "", true
		} else {
			base.GoName = m.Name
			if !base.GenericMethod {
				base.GoKind = chooseMemberKind(m.Static)
			}
			if m.Static || base.GenericMethod {
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
	return mapParametersWithGenerics(s, byIdentity, owner, nil, params)
}

func mapParametersWithGenerics(s *expectedSurface, byIdentity map[string]*contractType, owner *expectedType, generics []genericParameter, params []contractParameter) ([]string, []string, bool) {
	var inputs, outputs []string
	hasDirection := false
	for _, p := range params {
		mapped := mapTypeWithGenerics(s, byIdentity, owner, generics, p.Type)
		if p.Out {
			outputs = append(outputs, mapResultType(s, byIdentity, owner, p.Type)...)
			hasDirection = true
			continue
		}
		if p.Ref {
			mapped = "*" + strings.TrimPrefix(mapped, "*")
			hasDirection = true
		}
		inputs = append(inputs, applySubstitutableParameter(s, owner, p.Type, mapped))
	}
	return inputs, outputs, hasDirection
}

// ---------------------------------------------------------------------------
// Foundation 58 — CLR base substitutability at a Go parameter position.
// ---------------------------------------------------------------------------

// substitutableBases are the CLR classes whose PARAMETER positions accept a
// derived value, and therefore project to a reference interface rather than to
// the concrete pointer.
//
// # Why a parameter position and nothing else
//
// In C# a parameter typed `Texture2D` accepts a `RenderTarget2D`, because
// RenderTarget2D IS-A Texture2D. Go has no such relation and CNA-Go refuses to
// fake one with embedding, so a `*Texture2D` parameter would be a position no
// derived value can reach -- and there are SEVEN of them, all
// SpriteBatch.Draw's `texture`.
//
// A RETURN keeps the concrete type. `Texture2D::FromStream` returns a Texture2D
// and a caller uses every Texture2D member on it; returning an interface would
// take those members away to solve a problem returns do not have. The same
// holds for a property getter, whose value is read rather than supplied.
//
// # The mechanism
//
// One exported interface per base, `<GoName>Reference`, with an UNEXPORTED
// method. Exported so a consumer can name the parameter type; unexported method
// so only this module can satisfy it -- a consumer cannot hand SpriteBatch an
// object CNA never made.
//
// The forbidden-accessor rule is untouched. There is still no public Base,
// Parent or AsTexture2D: the interface is a position's type, not a conversion a
// consumer performs, and a call site reads exactly as the C# one does.
//
// # It is registered, not inferred
//
// A base enters this table only when the substitutability measurement reports
// its requirement as LIVE -- positions on projected carriers AND at least one
// projected derived type. measureXNABaseSubstitutability cross-checks the two,
// so a base that stops being live, or one that becomes live and is not
// recorded, is a diagnostic rather than a silent signature.
var substitutableBases = map[string]string{
	"Microsoft.Xna.Framework.Graphics.Texture2D": "Texture2DReference",
	// Foundation 61. TextureCollection's indexer setter takes a `Texture`, and
	// projecting that collection put the position on a carrier CNA-Go
	// projects; Texture already had a projected derived type, so its
	// requirement went LIVE for exactly the reason Texture2D's did.
	"Microsoft.Xna.Framework.Graphics.Texture": "TextureReference",
	// Foundation 79. BasicEffect is the first projected type deriving from
	// Effect, so Effect's requirement went LIVE the way the other two did:
	// SpriteBatch::Begin declares two `Effect effect` parameters and
	// Effect::.ctor declares a `cloneSource`, and a BasicEffect must reach all
	// three.
	//
	// Effect is also the second base -- after System.Exception in Foundation 76
	// -- whose RETURNS widen, and it is recorded in returnWideningBases for
	// that. The reason is measured rather than stylistic: Clone is declared to
	// return Effect and every stock effect OVERRIDES it to return an instance
	// of its own class, so a concrete *Effect return would hand back the base
	// HALF of a BasicEffect with no path to the object that owns it. The
	// downcast would not be lost, it would be impossible.
	"Microsoft.Xna.Framework.Graphics.Effect": "EffectReference",
	// Foundation 81. EnvironmentMapEffect::EnvironmentMap's SETTER is the only
	// TextureCube parameter position in the whole profile, and projecting that
	// effect put it on a carrier CNA-Go projects. TextureCube already had a
	// projected derived type -- RenderTargetCube, since Foundation 73 -- so its
	// requirement went LIVE for exactly the reason Texture2D's and Texture's
	// did, having been LATENT until a position appeared.
	//
	// It does NOT widen at returns: EffectParameter::GetValueTextureCube hands
	// back a value a caller uses every TextureCube member on, and no derived
	// identity is at stake there.
	"Microsoft.Xna.Framework.Graphics.TextureCube": "TextureCubeReference",
}

// returnWideningBases are the substitutable bases whose RETURN positions widen
// as well, which is the exception the rule above describes rather than its
// default.
//
// A base qualifies only when its derived types are the POINT of the returning
// member -- when returning the base would erase an identity the consumer must
// have back. Foundation 76 established it for the exception hierarchy, where
// the eight kinds a consumer catches are the whole contract; Foundation 79 adds
// Effect, where Clone's five overrides each construct their own class.
//
// The registry is closed and it is checked: a name here that is not in
// substitutableBases is a verifier failure, because a return cannot widen to an
// interface that no parameter position declares.
var returnWideningBases = map[string]struct{}{
	"Microsoft.Xna.Framework.Graphics.Effect": {},
}

// applySubstitutableParameter rewrites one mapped parameter type when its CLR
// type is a registered substitutable base.
func applySubstitutableParameter(s *expectedSurface, owner *expectedType, clrType, mapped string) string {
	name, registered := substitutableBases[clrType]
	if !registered {
		return mapped
	}
	base := s.typeForXNA(clrType)
	if base == nil {
		return mapped
	}
	if base.PackagePath == owner.PackagePath {
		return name
	}
	return canonicalPackageQualifier(base.PackagePath) + "." + name
}

// writtenStreamParameters names every CLR member whose System.IO.Stream
// parameter is WRITTEN rather than read.
//
// # Why a registry and not a rule
//
// The CLR has one Stream and Go has two interfaces, so `System.IO.Stream ->
// io.Reader` in bclTypes is a mapping that is right for most positions and
// wrong for some. It was wrong for exactly these two until Foundation 53:
// SaveAsPng and SaveAsJpeg hand the stream their encoded bytes, and projecting
// them with io.Reader would have declared a parameter a consumer cannot use for
// the only thing the member does with it.
//
// Direction cannot be derived from the signature -- both directions are spelled
// `Stream` -- so it is measured from the member's body and recorded here, one
// entry per position. The key is the full CLR member identity, so an overload
// that reads and one that writes stay distinguishable.
var writtenStreamParameters = map[string][]int{
	"Microsoft.Xna.Framework.Graphics.Texture2D::SaveAsPng(System.IO.Stream,System.Int32,System.Int32)":  {0},
	"Microsoft.Xna.Framework.Graphics.Texture2D::SaveAsJpeg(System.IO.Stream,System.Int32,System.Int32)": {0},
}

// applyStreamDirection rewrites the io.Reader default at the positions
// writtenStreamParameters names. It fails loudly rather than silently on a
// registry entry that does not describe the member it names: an index out of
// range, or a position that is not a stream at all, is a registry defect and
// would otherwise become an expected signature nobody wrote.
func applyStreamDirection(xna string, parameters []string) []string {
	positions, present := writtenStreamParameters[xna]
	if !present {
		return parameters
	}
	rewritten := append([]string(nil), parameters...)
	for _, position := range positions {
		if position < 0 || position >= len(rewritten) || rewritten[position] != "io.Reader" {
			panic(fmt.Sprintf("writtenStreamParameters names position %d of %s, which is %v",
				position, xna, parameters))
		}
		rewritten[position] = "io.Writer"
	}
	return rewritten
}

func mapIndexerParameters(s *expectedSurface, byIdentity map[string]*contractType, owner *expectedType, params []contractParameter) []string {
	inputs, _, _ := mapParameters(s, byIdentity, owner, params)
	return inputs
}

// mapReturn projects a member's return type. `generics` are the METHOD's own
// type parameters, and passing them is what stops a generic method's return
// from being reported as the IL token for a position.
//
// Foundation 54 resolved `!!0` at parameter positions and left the return
// unresolved, because the two generic members it closed both return
// System.Void. Foundation 63 found the gap the first time a generic method
// RETURNS its type parameter: `ContentManager::Load<T>(String) -> !!0` was
// projected as returning a type literally called `!!0`, which is a name for a
// position rather than a type -- the exact defect Foundation 54 named on the
// other side of the signature.
func mapReturn(s *expectedSurface, byIdentity map[string]*contractType, owner *expectedType, generics []genericParameter, raw *string) []string {
	if raw == nil || *raw == "System.Void" {
		return nil
	}
	return mapResultTypeWithGenerics(s, byIdentity, owner, generics, *raw)
}

func mapResultType(s *expectedSurface, byIdentity map[string]*contractType, owner *expectedType, raw string) []string {
	return mapResultTypeWithGenerics(s, byIdentity, owner, nil, raw)
}

func mapResultTypeWithGenerics(s *expectedSurface, byIdentity map[string]*contractType, owner *expectedType, generics []genericParameter, raw string) []string {
	if inner, ok := nullableInner(raw); ok {
		return []string{strings.TrimPrefix(mapTypeWithGenerics(s, byIdentity, owner, generics, inner), "*"), "bool"}
	}
	return []string{applyWidenedReturn(s, owner, raw, mapTypeWithGenerics(s, byIdentity, owner, generics, raw))}
}

// applyWidenedReturn rewrites one mapped RESULT type when its CLR type is a
// base recorded in returnWideningBases.
//
// It is deliberately the same shape as applySubstitutableParameter and reads
// from the same table of interface names, because a widened return and a
// widened parameter are the same interface seen from the two ends of a call.
// What differs is only which table decides: substitutableBases governs
// parameters, and returnWideningBases is the narrower set whose returns widen
// too.
func applyWidenedReturn(s *expectedSurface, owner *expectedType, clrType, mapped string) string {
	if _, widens := returnWideningBases[clrType]; !widens {
		return mapped
	}
	return applySubstitutableParameter(s, owner, clrType, mapped)
}

// mapTypeWithGenerics is mapType plus the method's own type parameters, which
// only a generic method has. Everything else calls mapType, which passes none.
func mapTypeWithGenerics(s *expectedSurface, byIdentity map[string]*contractType, owner *expectedType, generics []genericParameter, raw string) string {
	if name, ok := methodTypeParameterName(generics, strings.TrimSuffix(raw, "[]")); ok {
		if strings.HasSuffix(raw, "[]") {
			return "[]" + name
		}
		return name
	}
	return mapType(s, byIdentity, owner, raw)
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
	// Foundation 72. System.Collections.Generic.List<T>.Enumerator, which the
	// Effect cluster's four collections return DIRECTLY from GetEnumerator
	// rather than boxing as IEnumerator<T> the way DisplayModeCollection does.
	//
	// It is the same projection, because it is the same contract: List<T>'s
	// enumerator IS an IEnumerator<T>, and the settled collection rule already
	// says IEnumerator<T> is Iterator<T>. Before this it fell through to `any`,
	// which is an unmeasured degradation of exactly the kind the StringBuilder
	// position had.
	//
	// The concrete-enumerator rule does NOT apply: it is for a collection that
	// declares its OWN public enumerator type, as TouchCollection does, and
	// none of these four declares one -- they hand out the BCL list's.
	//
	// One difference is recorded rather than hidden. List<T>.Enumerator is a
	// STRUCT, so a C# caller who copies it gets an independent cursor, and a Go
	// interface value cannot express that. The four lists here are built in a
	// constructor and never mutated, so the version check that struct carries
	// has nothing to detect either.
	if inner, ok := genericTypeArgument(raw, "System.Collections.Generic.List`1+Enumerator["); ok {
		return frameworkQualified(owner, "Iterator") + "[" + mapType(s, byIdentity, owner, inner) + "]"
	}
	// Foundation 75. System.Collections.Generic.List<T> ITSELF, which the
	// pinned contract carries at exactly ONE public signature position in the
	// whole profile: GraphicsDeviceManager::RankDevices(List<GraphicsDeviceInformation>).
	//
	// The settled rule for a BCL type at a signature position is that it takes
	// the standard-library Go type whose ROLE it is, chosen from what the
	// profile's positions MEASURABLY do with it -- the rule that made
	// System.IO.Stream an io.Reader because every stream position in the
	// profile is read. RankDevices' body is `foundDevices.Sort(comparer)`, so
	// the one position sorts the list in place and reads it by index, and a Go
	// slice does exactly that.
	//
	// A slice cannot grow or shrink through the callee and a List<T> can. No
	// projected member in the profile does: the private AddDevices that appends
	// is not public surface. That difference is the reason this is a MEASURED
	// mapping of one position rather than a general claim that List<T> is a
	// slice.
	if inner, ok := genericTypeArgument(raw, "System.Collections.Generic.List`1["); ok {
		return "[]" + mapType(s, byIdentity, owner, inner)
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
	// Foundation 74. The four BCL types Dictionary<K,V>'s inherited public
	// surface carries at signature positions.
	//
	// The two nested enumerator structs take the SAME projection List<T>'s does
	// and for the same reason -- each IS an IEnumerator<T>, and IEnumerator<T>
	// is Iterator<T> -- so they are mapped before the collections that return
	// them, since their identities share a prefix.
	if arguments, ok := genericTypeArguments(raw, "System.Collections.Generic.Dictionary`2+KeyCollection+Enumerator["); ok && len(arguments) == 2 {
		return frameworkQualified(owner, "Iterator") + "[" + mapType(s, byIdentity, owner, arguments[0]) + "]"
	}
	if arguments, ok := genericTypeArguments(raw, "System.Collections.Generic.Dictionary`2+ValueCollection+Enumerator["); ok && len(arguments) == 2 {
		return frameworkQualified(owner, "Iterator") + "[" + mapType(s, byIdentity, owner, arguments[1]) + "]"
	}
	if arguments, ok := genericTypeArguments(raw, "System.Collections.Generic.Dictionary`2+Enumerator["); ok && len(arguments) == 2 {
		return frameworkQualified(owner, "Iterator") + "[" + frameworkQualified(owner, "KeyValuePair") + "[" +
			mapType(s, byIdentity, owner, arguments[0]) + ", " + mapType(s, byIdentity, owner, arguments[1]) + "]]"
	}
	if arguments, ok := genericTypeArguments(raw, "System.Collections.Generic.Dictionary`2+KeyCollection["); ok && len(arguments) == 2 {
		return "*" + frameworkQualified(owner, "DictionaryKeyCollection") + "[" +
			mapType(s, byIdentity, owner, arguments[0]) + ", " + mapType(s, byIdentity, owner, arguments[1]) + "]"
	}
	if arguments, ok := genericTypeArguments(raw, "System.Collections.Generic.Dictionary`2+ValueCollection["); ok && len(arguments) == 2 {
		return "*" + frameworkQualified(owner, "DictionaryValueCollection") + "[" +
			mapType(s, byIdentity, owner, arguments[0]) + ", " + mapType(s, byIdentity, owner, arguments[1]) + "]"
	}
	// KeyValuePair<K,V> is a CLR STRUCT, so it keeps value semantics and takes
	// no pointer -- unlike the two collection views, which are classes.
	if arguments, ok := genericTypeArguments(raw, "System.Collections.Generic.KeyValuePair`2["); ok && len(arguments) == 2 {
		return frameworkQualified(owner, "KeyValuePair") + "[" +
			mapType(s, byIdentity, owner, arguments[0]) + ", " + mapType(s, byIdentity, owner, arguments[1]) + "]"
	}
	if inner, ok := genericTypeArgument(raw, "System.Collections.Generic.IEqualityComparer`1["); ok {
		return frameworkQualified(owner, "IEqualityComparer") + "[" + mapType(s, byIdentity, owner, inner) + "]"
	}
	// System.EventArgs is a CLR class, so it keeps CLR reference semantics and
	// projects as a pointer to the framework-package language adapter.
	if raw == "System.EventArgs" {
		return "*" + frameworkQualified(owner, "EventArgs")
	}
	// Foundation 76. System.Exception widens to the exported reference
	// interface at EVERY signature position, parameter and return alike.
	//
	// The settled substitutable-base rule widens a base-typed PARAMETER and
	// leaves a base-typed RETURN as the concrete pointer, recording the lost
	// downcast as a language limitation. This family is where that trade would
	// cost the type its purpose: an exception hierarchy exists to be told apart
	// by type, and InnerException returning a concrete *Exception would erase
	// which of the eight kinds a consumer chained. The interface carries an
	// unexported method, so only this module can satisfy it.
	if raw == "System.Exception" {
		return frameworkQualified(owner, "ExceptionReference")
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

// genericTypeArguments is genericTypeArgument for a CLR identity with more than
// one type argument. It splits at top-level commas, so a nested generic
// argument survives intact.
func genericTypeArguments(raw, prefix string) ([]string, bool) {
	if !strings.HasPrefix(raw, prefix) || !strings.HasSuffix(raw, "]") {
		return nil, false
	}
	inner := raw[len(prefix) : len(raw)-1]
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
	return append(arguments, strings.TrimSpace(inner[start:])), true
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
			if witnessedInterface(contractOwner, owner, identity) {
				collectInterfaceWitnesses(surface, byIdentity, owner, interfaceType, mapped.TypeArguments, map[string]bool{})
			}
		}
	}
}

// explicitInterfaceWitnessOwners names the non-PackedVector types whose
// EXPLICIT CLR interface implementations are projected as Go witnesses.
//
// A CLR type may implement an interface member privately, and the contract's
// public member set then does not carry it. Go has no explicit implementation,
// so the method has to exist and be exported for the type to satisfy the
// interface at all -- which is what a witness is, and why a witness is not an
// unexpected member.
//
// It is a registry rather than "every mapped direct interface" because the
// second is not the same rule. Two things must both be true before a witness is
// required, and only the first is visible in the contract: the member must be
// absent from the public set, AND the Go type must be ABLE to declare it. The
// registry is where the second is recorded.
//
// GraphicsDeviceManager's OTHER direct interface,
// Microsoft.Xna.Framework.Graphics.IGraphicsDeviceService, is deliberately
// absent for exactly that reason. The manager is a framework-package type and
// that contract's GraphicsDevice accessor returns a Graphics-package type, so
// the manager cannot implement it in Go at all; the Graphics package registers
// a small adapter over it instead, and requiring witnesses here would demand
// methods no Go type in that package could ever declare.
var explicitInterfaceWitnessOwners = map[string]map[string]bool{
	// The three IGraphicsDeviceManager operations are `private hidebysig
	// newslot virtual final` with an `.override`, so the contract's public
	// member set has none of them -- yet Game resolves the interface out of
	// Services and calls all three once per frame.
	"Microsoft.Xna.Framework.GraphicsDeviceManager": {
		"Microsoft.Xna.Framework.IGraphicsDeviceManager": true,
	},
	// Foundation 77. Each of the four stock vertex structs implements
	// IVertexType::get_VertexDeclaration `private hidebysig newslot virtual
	// final` with an `.override`, so the contract's public set has none of them
	// -- and every one of them CAN declare the method in Go, because
	// VertexDeclaration is a type in their own package. Both halves of the
	// witness rule hold, unlike GraphicsDeviceManager's other interface.
	"Microsoft.Xna.Framework.Graphics.VertexPositionColor": {
		"Microsoft.Xna.Framework.Graphics.IVertexType": true,
	},
	"Microsoft.Xna.Framework.Graphics.VertexPositionColorTexture": {
		"Microsoft.Xna.Framework.Graphics.IVertexType": true,
	},
	"Microsoft.Xna.Framework.Graphics.VertexPositionNormalTexture": {
		"Microsoft.Xna.Framework.Graphics.IVertexType": true,
	},
	"Microsoft.Xna.Framework.Graphics.VertexPositionTexture": {
		"Microsoft.Xna.Framework.Graphics.IVertexType": true,
	},
	// Foundation 81. EnvironmentMapEffect and SkinnedEffect implement
	// IEffectLights::LightingEnabled EXPLICITLY, and only that member:
	//
	//	.method private ... IEffectLights.get_LightingEnabled()
	//	  ldc.i4.1; ret                       -- lighting is ALWAYS on
	//	.method private ... IEffectLights.set_LightingEnabled(bool)
	//	  if (!value) throw NotSupportedException(CantDisableLighting)
	//
	// So the contract's public member set for both types has no
	// LightingEnabled property at all, while their other six IEffectLights
	// members are ordinary public declarations. Both halves of the witness rule
	// hold: the member is absent from the public set, and both types can
	// declare it in Go -- DirectionalLight is in their own package.
	"Microsoft.Xna.Framework.Graphics.EnvironmentMapEffect": {
		"Microsoft.Xna.Framework.Graphics.IEffectLights": true,
	},
	"Microsoft.Xna.Framework.Graphics.SkinnedEffect": {
		"Microsoft.Xna.Framework.Graphics.IEffectLights": true,
	},
}

// witnessedInterface reports whether one owner/interface pair produces
// witnesses. The PackedVector structs are admitted as a family because all
// nineteen implement IPackedVector<T> the same way; everything else is named.
func witnessedInterface(contractOwner *contractType, owner *expectedType, interfaceIdentity string) bool {
	if contractOwner.Kind == "struct" && strings.HasPrefix(owner.XNA, packedVectorNamespace) {
		return true
	}
	return explicitInterfaceWitnessOwners[owner.XNA][interfaceIdentity]
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
	return parameterShapeWithGenerics(nil, params)
}

func parameterShapeWithGenerics(generics []genericParameter, params []contractParameter) string {
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
		parts = append(parts, prefix+typeShapeWithGenerics(generics, p.Type))
	}
	return strings.Join(parts, "And")
}

var nonIdentifier = regexp.MustCompile(`[^A-Za-z0-9]+`)

// methodTypeParameterName resolves a CLR method type-parameter token to the
// name the method DECLARES for it.
//
// `!!0` is an IL token, not a type name: it means "this method's first type
// parameter", whose name lives in the member's genericParameters. Before
// Foundation 54 nothing resolved it, so the shape builder stripped the
// punctuation and produced `SliceOf0` -- an overload suffix naming a position
// rather than a type, and one no consumer could have guessed.
func methodTypeParameterName(generics []genericParameter, raw string) (string, bool) {
	if !strings.HasPrefix(raw, "!!") {
		return "", false
	}
	position, err := strconv.Atoi(strings.TrimPrefix(raw, "!!"))
	if err != nil {
		return "", false
	}
	for _, parameter := range generics {
		if parameter.Position == position {
			return parameter.Name, true
		}
	}
	// A token whose position the member does not declare is a contract defect,
	// and reporting it as a name would hide it.
	return "", false
}

func typeShape(raw string) string { return typeShapeWithGenerics(nil, raw) }

func typeShapeWithGenerics(generics []genericParameter, raw string) string {
	raw = strings.TrimSuffix(raw, "&")
	if strings.HasPrefix(raw, "System.Nullable`1[") && strings.HasSuffix(raw, "]") {
		inner := strings.TrimSuffix(strings.TrimPrefix(raw, "System.Nullable`1["), "]")
		return "NullableOf" + typeShapeWithGenerics(generics, inner)
	}
	if name, ok := methodTypeParameterName(generics, strings.TrimSuffix(raw, "[]")); ok {
		if strings.HasSuffix(raw, "[]") {
			return "SliceOf" + name
		}
		return name
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
	Excluded []bclExcludedMember
	// LanguageAccessors are the exported Go members this base REQUIRES that the
	// CLR type does not declare. They are admitted on the adapter itself and on
	// every type that composes the base, and nowhere else.
	LanguageAccessors []bclLanguageAccessor
	Rationale         string
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

// bclLanguageAccessor is one EXPORTED Go member a composed base requires that
// the CLR type does not declare.
//
// Go has no explicit interface implementation and no way for a package to
// satisfy another package's unexported method, so a reference interface whose
// implementors live in OTHER packages must expose its distinguishing accessor.
// The accessor is what keeps the interface unsatisfiable from outside the
// module, because its result type is declared in an internal package -- but it
// is still an exported member the contract has no entry for, and it would
// otherwise be an unexpected one.
//
// The registry admits it by NAME, on exactly the types that compose the base,
// and requires a reason. It is not a general escape: a member on a type that
// composes no such base is still unexpected.
type bclLanguageAccessor struct {
	Name   string
	Reason string
}

// bclExcludedMember records one public-looking base member that is not
// projected, and why.
//
// Kind separates two very different claims and must never blur them.
// NOT_PUBLIC_SURFACE -- the default, and every exclusion up to Foundation 73 --
// says the member is not part of what a derived CLR caller can reach at all: a
// constructor, which the CLR does not inherit; a `family` member; or a private
// explicit interface implementation. Nothing is missing in that case.
// BCL_PROJECTION_BLOCKED_EXTERNAL says the opposite: the member IS public CLR
// surface, it IS absent from the Go projection, and Needs names the exact
// external BCL closure that would have to be projected first. It is the
// measured admission of a hole, not a permission to have one, and the verifier
// counts it.
type bclExcludedMember struct {
	CLRMember string
	Kind      string
	Needs     string
	Reason    string
}

// exclusionKind defaults an unset kind to NOT_PUBLIC_SURFACE, so the ninety
// existing entries keep meaning exactly what they meant.
func exclusionKind(excluded bclExcludedMember) string {
	if excluded.Kind == "" {
		return "NOT_PUBLIC_SURFACE"
	}
	return excluded.Kind
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

// bclOutParameter is a CLR `[out]` parameter of a BCL member, which the settled
// direction rule removes from the Go input list and appends to the results.
func bclOutParameter(name, clrType string) contractParameter {
	return contractParameter{Name: name, Type: clrType + "&", Out: true}
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

	// System.Collections.Generic.Dictionary<TKey,TValue>, read from the same
	// pinned mscorlib 4.0.30319.1.
	//
	// The class stores its entries in three parallel private fields --
	// `int32[] buckets`, `Entry[] entries` and the `count`/`freeList`/
	// `freeCount` cursor trio -- plus `version` and the `IEqualityComparer<TKey>`
	// the constructor chose. The parameterless constructor, the only one any
	// pinned XNA subclass calls, is `.ctor(0, null)`: capacity zero, so
	// Initialize is NOT called and `buckets` starts null, and a null comparer
	// argument becomes EqualityComparer<TKey>.Default.
	//
	// This is the .NET Framework 4.0 RTM Dictionary. Its Insert has no
	// collision counter and no randomised-comparer switch, so `comparer` is
	// assigned once and get_Comparer answers the same object forever.
	//
	// Its public surface is exactly the fourteen members below plus
	// GetObjectData. Everything else the class declares is one of three things
	// that are not public surface: six public constructors and one `family`
	// serialization constructor, which the CLR does not inherit; the private
	// Initialize/FindEntry/Insert/Resize helpers; and twenty-three private
	// explicit implementations of IDictionary, IDictionary<K,V>, ICollection,
	// ICollection<KeyValuePair<K,V>> and IEnumerable, which the settled
	// BCL-interface rule already excludes.
	//
	// The null-key guard that opens Insert, FindEntry and Remove is statically
	// dead for the profile's only consumer, whose key type is System.String and
	// therefore Go string, which has no null -- the same shape as
	// Collection<T>'s dead `items.IsReadOnly` guard, and recorded for the same
	// reason.
	"System.Collections.Generic.Dictionary`2": {
		GoAdapter:       "dictionaryBase[TKey, TValue]",
		AdapterField:    "base",
		GenericArity:    2,
		BehaviorLevel:   "SUPPORTED",
		Authority:       "mscorlib.dll 4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0",
		AuthoritySHA256: "5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63",
		Rationale:       "a mutable BCL hash map base projected as private composition plus measured forwarding; the adapter reproduces the entries array, the free list and the version counter rather than wrapping a Go map, because Dictionary's enumeration order is entries-array order and Go's is deliberately randomised",
		Members: []bclInheritedMember{
			{Member: bclProperty("Comparer", "System.Collections.Generic.IEqualityComparer`1[!0]", true, false),
				Rationale: "get_Comparer is one ldfld of the comparer the constructor stored; for the parameterless constructor that is EqualityComparer<TKey>.Default, a cached CLR static, so two dictionaries answer with the same object"},
			{Member: bclProperty("Count", "System.Int32", true, false),
				Rationale: "get_Count is `count - freeCount`, not the length of the entries array, so a removal reduces it without shrinking anything"},
			{Member: bclProperty("Keys", "System.Collections.Generic.Dictionary`2+KeyCollection[!0,!1]", true, false),
				Rationale: "get_Keys allocates the view once, caches it in a private field and returns the same object forever after; the view reads through to the dictionary rather than copying, so later changes are visible through an already-obtained Keys"},
			{Member: bclProperty("Values", "System.Collections.Generic.Dictionary`2+ValueCollection[!0,!1]", true, false),
				Rationale: "get_Values is get_Keys' mirror, cached the same way"},
			{Member: bclProperty("Item", "!1", true, true, bclParameter("key", "!0")),
				Rationale: "get_Item is FindEntry then either the stored value or KeyNotFoundException, which is the ONE inherited read that fails; set_Item is `Insert(key, value, false)`, which adds a missing key rather than refusing and cannot fail"},
			{Member: bclMethod("Add", "System.Void", bclParameter("key", "!0"), bclParameter("value", "!1")),
				Rationale: "Add is `Insert(key, value, true)`, whose only reachable failure is the ArgumentException a duplicate key raises"},
			{Member: bclMethod("Clear", "System.Void"),
				Rationale: "Clear's whole body is guarded by `count > 0`, so clearing an already empty dictionary does not even increment the version -- the opposite of List<T>.Clear, and a live enumerator survives it"},
			{Member: bclMethod("ContainsKey", "System.Boolean", bclParameter("key", "!0")),
				Rationale: "ContainsKey is `FindEntry(key) >= 0`, which compares the stored hash before it calls the comparer's Equals"},
			{Member: bclMethod("ContainsValue", "System.Boolean", bclParameter("value", "!1")),
				Rationale: "ContainsValue is a forward scan of the live entries over EqualityComparer<TValue>.Default; it is the one member whose comparer is the VALUE's rather than the key's"},
			{Member: bclMethod("GetEnumerator", "System.Collections.Generic.Dictionary`2+Enumerator[!0,!1]"),
				Rationale: "GetEnumerator returns the nested Enumerator STRUCT, which IS an IEnumerator<KeyValuePair<K,V>>, so the settled List<T>.Enumerator rule projects it as Iterator<T>; it walks the entries array from 0, skips freed slots, and compares the captured version BEFORE the bounds test"},
			{Member: bclMethod("OnDeserialization", "System.Void", bclParameter("sender", "System.Object")),
				Rationale: "public and not an explicit implementation, so it is genuinely inherited surface; its body opens `if (m_siInfo == null) return`, and m_siInfo's only non-null writer is the `family` .ctor(SerializationInfo, StreamingContext), which the CLR does not inherit and no consumer declares -- so the empty branch is the only reachable body and the member cannot fail"},
			{Member: bclMethod("Remove", "System.Boolean", bclParameter("key", "!0")),
				Rationale: "Remove unlinks the entry, clears both halves so neither is retained, and pushes the slot onto the HEAD of the free list, so the next insertion reuses the most recently removed index -- which is observable through enumeration order"},
			{Member: bclMethod("TryGetValue", "System.Boolean", bclParameter("key", "!0"), bclOutParameter("value", "!1")),
				Rationale: "TryGetValue is FindEntry plus an out parameter the settled direction rule turns into a second result; a miss writes default(TValue) explicitly"},
		},
		Excluded: []bclExcludedMember{
			{CLRMember: "GetObjectData", Kind: "BCL_PROJECTION_BLOCKED_EXTERNAL",
				Needs:  "System.Runtime.Serialization.SerializationInfo (43 public members) and StreamingContext (6), whose own inventory reaches System.Decimal (89) and System.DateTime (91), plus SerializationInfoEnumerator (6), SerializationEntry (3), IFormatterConverter and System.TypeCode",
				Reason: "genuinely public CLR surface that IS absent from the Go projection: the settled signature-adapter rule pins an adapter to the EXACT public member inventory of the CLR type it spells, so projecting the SerializationInfo parameter means projecting all 43 of its members, which drags System.Decimal and System.DateTime -- 180 further public members that the XNA 4.0 Windows profile names nowhere else -- in behind them. That is the mscorlib port the profile is defined to exclude, so the member stays measured-absent rather than approximated"},
			{CLRMember: ".ctor()", Reason: "the CLR does not inherit constructors; a derived type projects its own"},
			{CLRMember: ".ctor(int32)", Reason: "the CLR does not inherit constructors, and no pinned XNA subclass calls this overload"},
			{CLRMember: ".ctor(IEqualityComparer`1<TKey>)", Reason: "the CLR does not inherit constructors, and no pinned XNA subclass supplies a comparer"},
			{CLRMember: ".ctor(int32,IEqualityComparer`1<TKey>)", Reason: "the CLR does not inherit constructors"},
			{CLRMember: ".ctor(IDictionary`2<TKey,TValue>)", Reason: "the CLR does not inherit constructors"},
			{CLRMember: ".ctor(IDictionary`2<TKey,TValue>,IEqualityComparer`1<TKey>)", Reason: "the CLR does not inherit constructors"},
			{CLRMember: ".ctor(SerializationInfo,StreamingContext)", Reason: "`family`, and not inherited; it is also the only writer that could make m_siInfo non-null, which is why OnDeserialization's remaining body is unreachable"},
			{CLRMember: "Initialize", Reason: "private helper"},
			{CLRMember: "FindEntry", Reason: "private helper"},
			{CLRMember: "Insert", Reason: "private helper; Add and set_Item are its two public entry points"},
			{CLRMember: "Resize", Reason: "private helper"},
			{CLRMember: "CopyTo", Reason: "the KeyValuePair-array CopyTo is PRIVATE on Dictionary itself; the two public CopyTo members belong to KeyCollection and ValueCollection"},
			{CLRMember: "GetValueOrDefault", Reason: "`assembly`, so it is not public surface"},
			{CLRMember: "IDictionary<TKey,TValue>.Keys", Reason: "private explicit implementation returning ICollection<TKey>; the public get_Keys carries the view"},
			{CLRMember: "IDictionary<TKey,TValue>.Values", Reason: "private explicit implementation returning ICollection<TValue>"},
			{CLRMember: "ICollection<KeyValuePair<TKey,TValue>>.IsReadOnly", Reason: "private explicit implementation; the settled BCL-interface rule projects nothing for it"},
			{CLRMember: "ICollection<KeyValuePair<TKey,TValue>>.Add", Reason: "private explicit implementation; the generic Add carries the operation"},
			{CLRMember: "ICollection<KeyValuePair<TKey,TValue>>.Contains", Reason: "private explicit implementation"},
			{CLRMember: "ICollection<KeyValuePair<TKey,TValue>>.CopyTo", Reason: "private explicit implementation"},
			{CLRMember: "ICollection<KeyValuePair<TKey,TValue>>.Remove", Reason: "private explicit implementation"},
			{CLRMember: "IEnumerable<KeyValuePair<TKey,TValue>>.GetEnumerator", Reason: "private explicit implementation, and the public GetEnumerator already carries enumeration"},
			{CLRMember: "IEnumerable.GetEnumerator", Reason: "private explicit implementation"},
			{CLRMember: "ICollection.IsSynchronized", Reason: "private explicit implementation"},
			{CLRMember: "ICollection.SyncRoot", Reason: "private explicit implementation, and CNA-Go projects no CLR sync root"},
			{CLRMember: "ICollection.CopyTo", Reason: "private explicit implementation"},
			{CLRMember: "IDictionary.Item", Reason: "private explicit implementation of the non-generic indexer"},
			{CLRMember: "IDictionary.Keys", Reason: "private explicit implementation"},
			{CLRMember: "IDictionary.Values", Reason: "private explicit implementation"},
			{CLRMember: "IDictionary.IsReadOnly", Reason: "private explicit implementation"},
			{CLRMember: "IDictionary.IsFixedSize", Reason: "private explicit implementation"},
			{CLRMember: "IDictionary.Add", Reason: "private explicit implementation"},
			{CLRMember: "IDictionary.Contains", Reason: "private explicit implementation"},
			{CLRMember: "IDictionary.Remove", Reason: "private explicit implementation"},
			{CLRMember: "IDictionary.GetEnumerator", Reason: "private explicit implementation returning IDictionaryEnumerator"},
		},
	},

	// System.Exception, read from the same pinned mscorlib.
	//
	// Its eleven public instance members are the whole useful surface of the
	// profile's eight exception types, every one of which declares only
	// constructors. Eight are projected; the other three name the exact
	// external BCL closure that blocks them.
	//
	// The adapter lives in internal/bclexception rather than in the framework
	// package, because those eight derived types are in FOUR other packages and
	// an unexported framework type is unreachable from any of them. That is the
	// composition rule's own escape, and `internal/` keeps the adapter
	// unreachable from outside the module -- the property the unexported field
	// had.
	"System.Exception": {
		GoAdapter:       "bclexception.State",
		AdapterField:    "base",
		GenericArity:    0,
		BehaviorLevel:   "SUPPORTED",
		Authority:       "mscorlib.dll 4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0",
		AuthoritySHA256: "5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63",
		Rationale:       "the CLR exception base, composed privately; it is NOT a Go error, and CNA-Go's per-operation error channel is unchanged by its existence",
		Members: []bclInheritedMember{
			{Member: bclProperty("Message", "System.String", true, false),
				Rationale: "get_Message returns _message, or Environment.GetRuntimeResourceString(\"Exception_WasThrown\", GetClassName()) when the field is null -- and GetClassName is the DERIVED type's name, so a default-constructed DeviceLostException names itself. The null test is not statically dead, which is why the adapter carries the CLR field's null explicitly"},
			{Member: bclProperty("InnerException", "System.Exception", true, false),
				Rationale: "one ldfld of the exception the two-argument constructor stored"},
			{Member: bclMethod("GetBaseException", "System.Exception"),
				Rationale: "walks InnerException to the DEEPEST non-null exception, and answers `this` when there is none"},
			{Member: bclProperty("StackTrace", "System.String", true, false),
				Rationale: "the frames the CLR captured AT THROW TIME. CNA-Go throws no CLR exception, so _stackTraceString is null for every reachable state and the getter answers null, which for a Go string is empty"},
			{Member: bclProperty("HelpLink", "System.String", true, true),
				Rationale: "get and set are one ldfld and one stfld over _helpURL, with no validation"},
			{Member: bclProperty("Source", "System.String", true, true),
				Rationale: "the reference computes a default from the declaring assembly of the throwing frame when the field is null; an exception nothing threw has no such frame"},
			{Member: bclMethod("ToString", "System.String"),
				Rationale: "GetClassName(), then \": \" and the message when it is non-empty, then \" ---> \" and the inner exception's own ToString followed by NewLine, three spaces and the end-of-inner-exception marker"},
			{Member: bclMethod("GetType", "System.Type"),
				Rationale: "`virtual final`, and it answers the RUNTIME type -- the derived one. A composed base cannot see its deriver, so the derived constructor installs the CLR `this`"},
		},
		Excluded: []bclExcludedMember{
			{CLRMember: "Data", Kind: "BCL_PROJECTION_BLOCKED_EXTERNAL",
				Needs:  "System.Collections.IDictionary, the non-generic dictionary contract",
				Reason: "genuinely public CLR surface that IS absent. The non-generic IDictionary is named nowhere else in the profile except as the same blocker on the thirteen Design converters"},
			{CLRMember: "TargetSite", Kind: "BCL_PROJECTION_BLOCKED_EXTERNAL",
				Needs:  "System.Reflection.MethodBase and the reflection member model",
				Reason: "genuinely public CLR surface that IS absent; it would also be null for an exception nothing threw, which is every exception this projection can produce"},
			{CLRMember: "GetObjectData", Kind: "BCL_PROJECTION_BLOCKED_EXTERNAL",
				Needs:  "System.Runtime.Serialization.SerializationInfo and StreamingContext",
				Reason: "the same 238-member closure that blocks Dictionary`2::GetObjectData, measured in Foundation 74"},
			{CLRMember: ".ctor()", Reason: "the CLR does not inherit constructors; each derived type declares its own three"},
			{CLRMember: ".ctor(string)", Reason: "as above"},
			{CLRMember: ".ctor(string,Exception)", Reason: "as above"},
			{CLRMember: ".ctor(SerializationInfo,StreamingContext)", Reason: "`family`, and not inherited"},
			{CLRMember: "HResult", Reason: "`family`; it is not public surface. ExternalException's public ErrorCode is its one reader in the profile"},
			{CLRMember: "SerializeObjectState", Reason: "a `family` event over EventHandler<SafeSerializationEventArgs>; not public surface, and its args type is a serialization type"},
			{CLRMember: "Init", Reason: "private helper the constructors call"},
			{CLRMember: "GetClassName", Reason: "private helper Message and ToString call"},
			{CLRMember: "GetStackTrace", Reason: "private helper StackTrace and ToString call"},
			{CLRMember: "_Exception.GetType", Reason: "private explicit implementation of the COM _Exception interface"},
		},
		LanguageAccessors: []bclLanguageAccessor{
			{Name: "State", Reason: "Go has no explicit interface implementation and no way for one package to satisfy another package's unexported method, and this base's derived types live in FOUR other packages. The reference interface's distinguishing accessor therefore has to be exported -- and it stays unsatisfiable from outside the module because its result type is declared in internal/bclexception"},
		},
	},

	// System.Runtime.InteropServices.ExternalException, which three of the
	// eight XNA exception types derive from.
	//
	// It extends System.SystemException, which adds no public surface of its
	// own, so the inherited set is System.Exception's plus ONE: a public
	// ErrorCode over the protected HResult. It also OVERRIDES ToString, and the
	// override is observably different -- the HResult in parentheses as eight
	// uppercase hex digits, and no end-of-inner-exception marker.
	//
	// All three of its XNA subclasses reach E_FAIL, because the only
	// constructor that assigns another error code is the one none of them
	// declares.
	"System.Runtime.InteropServices.ExternalException": {
		GoAdapter:       "bclexception.State",
		AdapterField:    "base",
		GenericArity:    0,
		BehaviorLevel:   "SUPPORTED",
		Authority:       "mscorlib.dll 4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0",
		AuthoritySHA256: "5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63",
		Rationale:       "System.Exception's surface plus a public ErrorCode and a different ToString; the same private adapter carries both, selected by a flag the derived constructor sets",
		Members: []bclInheritedMember{
			{Member: bclProperty("Message", "System.String", true, false),
				Rationale: "get_Message returns _message, or Environment.GetRuntimeResourceString(\"Exception_WasThrown\", GetClassName()) when the field is null -- and GetClassName is the DERIVED type's name, so a default-constructed DeviceLostException names itself. The null test is not statically dead, which is why the adapter carries the CLR field's null explicitly"},
			{Member: bclProperty("InnerException", "System.Exception", true, false),
				Rationale: "one ldfld of the exception the two-argument constructor stored"},
			{Member: bclMethod("GetBaseException", "System.Exception"),
				Rationale: "walks InnerException to the DEEPEST non-null exception, and answers `this` when there is none"},
			{Member: bclProperty("StackTrace", "System.String", true, false),
				Rationale: "the frames the CLR captured AT THROW TIME. CNA-Go throws no CLR exception, so _stackTraceString is null for every reachable state and the getter answers null, which for a Go string is empty"},
			{Member: bclProperty("HelpLink", "System.String", true, true),
				Rationale: "get and set are one ldfld and one stfld over _helpURL, with no validation"},
			{Member: bclProperty("Source", "System.String", true, true),
				Rationale: "the reference computes a default from the declaring assembly of the throwing frame when the field is null; an exception nothing threw has no such frame"},
			{Member: bclMethod("ToString", "System.String"),
				Rationale: "GetClassName(), then \": \" and the message when it is non-empty, then \" ---> \" and the inner exception's own ToString followed by NewLine, three spaces and the end-of-inner-exception marker"},
			{Member: bclMethod("GetType", "System.Type"),
				Rationale: "`virtual final`, and it answers the RUNTIME type -- the derived one. A composed base cannot see its deriver, so the derived constructor installs the CLR `this`"},
			{Member: bclProperty("ErrorCode", "System.Int32", true, false),
				Rationale: "get_ErrorCode is one forwarded Exception::get_HResult, which is `family` on the base and public here"},
		},
		Excluded: []bclExcludedMember{
			{CLRMember: "Data", Kind: "BCL_PROJECTION_BLOCKED_EXTERNAL",
				Needs:  "System.Collections.IDictionary, the non-generic dictionary contract",
				Reason: "genuinely public CLR surface that IS absent. The non-generic IDictionary is named nowhere else in the profile except as the same blocker on the thirteen Design converters"},
			{CLRMember: "TargetSite", Kind: "BCL_PROJECTION_BLOCKED_EXTERNAL",
				Needs:  "System.Reflection.MethodBase and the reflection member model",
				Reason: "genuinely public CLR surface that IS absent; it would also be null for an exception nothing threw, which is every exception this projection can produce"},
			{CLRMember: "GetObjectData", Kind: "BCL_PROJECTION_BLOCKED_EXTERNAL",
				Needs:  "System.Runtime.Serialization.SerializationInfo and StreamingContext",
				Reason: "the same 238-member closure that blocks Dictionary`2::GetObjectData, measured in Foundation 74"},
			{CLRMember: ".ctor()", Reason: "the CLR does not inherit constructors; each derived type declares its own three"},
			{CLRMember: ".ctor(string)", Reason: "as above"},
			{CLRMember: ".ctor(string,Exception)", Reason: "as above"},
			{CLRMember: ".ctor(SerializationInfo,StreamingContext)", Reason: "`family`, and not inherited"},
			{CLRMember: "HResult", Reason: "`family` on System.Exception; ErrorCode above is the public projection of it"},
			{CLRMember: ".ctor(string,int32)", Reason: "the CLR does not inherit constructors, and no XNA subclass declares the errorCode overload"},
			{CLRMember: "SerializeObjectState", Reason: "a `family` event over EventHandler<SafeSerializationEventArgs>; not public surface, and its args type is a serialization type"},
			{CLRMember: "Init", Reason: "private helper the constructors call"},
			{CLRMember: "GetClassName", Reason: "private helper Message and ToString call"},
			{CLRMember: "GetStackTrace", Reason: "private helper StackTrace and ToString call"},
			{CLRMember: "_Exception.GetType", Reason: "private explicit implementation of the COM _Exception interface"},
		},
		LanguageAccessors: []bclLanguageAccessor{
			{Name: "State", Reason: "Go has no explicit interface implementation and no way for one package to satisfy another package's unexported method, and this base's derived types live in FOUR other packages. The reference interface's distinguishing accessor therefore has to be exported -- and it stays unsatisfiable from outside the module because its result type is declared in internal/bclexception"},
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
		for _, member := range mapMember(s, byIdentity, owner, synthetic, synthetic, instance, groups) {
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

	// Foundation 76. System.Exception.
	//
	// Its eleven public instance members are the whole useful surface of the
	// profile's eight exception types, which declare only constructors. Eight
	// of the eleven are projected; the other three name the exact external BCL
	// closure that blocks them.
	//
	// The registry pins the CONCRETE type, framework.Exception. Every projected
	// SIGNATURE takes framework.ExceptionReference instead -- see mapType for
	// why an exception hierarchy widens at return positions too -- and that
	// interface reproduces exactly the members below plus one unexported
	// accessor.
	"System.Exception": {
		GoAdapter: "Exception",
		LanguageAccessors: []bclLanguageAccessor{
			{Name: "State", Reason: "the reference interface's distinguishing accessor. Go has no way for one package to satisfy another package's unexported method, and this type's eight siblings live in four other packages, so it has to be exported -- and it stays unsatisfiable from outside the module because its result type is declared in internal/bclexception"},
		},
		GenericArity:    0,
		BehaviorLevel:   "SUPPORTED",
		Authority:       "mscorlib.dll 4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0",
		AuthoritySHA256: "5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63",
		Rationale:       "the CLR exception object the profile constructs and passes; it is NOT a Go error, and CNA-Go's per-operation error channel is unchanged by its existence",
		Members: []bclInheritedMember{
			{Member: bclProperty("Message", "System.String", true, false),
				Rationale: "get_Message returns _message, or Environment.GetRuntimeResourceString(\"Exception_WasThrown\", GetClassName()) when the field is null. The null test is NOT statically dead: `new Exception()` renders the default sentence and `new Exception(\"\")` renders the empty string, so the projection carries the CLR field's null explicitly"},
			{Member: bclProperty("InnerException", "System.Exception", true, false),
				Rationale: "one ldfld of the exception the two-argument constructor stored"},
			{Member: bclMethod("GetBaseException", "System.Exception"),
				Rationale: "walks InnerException to the DEEPEST non-null exception, and answers `this` when there is none"},
			{Member: bclProperty("StackTrace", "System.String", true, false),
				Rationale: "the frames the CLR captured AT THROW TIME. CNA-Go throws no CLR exception -- failure is a Go error -- so _stackTraceString is null for every reachable state and the getter answers null, which for a Go string is empty. A Go stack would be a different thing wearing the same name"},
			{Member: bclProperty("HelpLink", "System.String", true, true),
				Rationale: "get and set are one ldfld and one stfld over _helpURL, with no validation"},
			{Member: bclProperty("Source", "System.String", true, true),
				Rationale: "the reference computes a default from the declaring assembly of the throwing frame when the field is null; an exception nothing threw has no such frame, so the field is all a consumer can observe"},
			{Member: bclMethod("ToString", "System.String"),
				Rationale: "GetClassName(), then \": \" and the message when it is non-empty, then \" ---> \" and the inner exception's own ToString followed by NewLine, three spaces and the end-of-inner-exception marker, then NewLine and the stack trace when there is one"},
			{Member: bclMethod("GetType", "System.Type"),
				Rationale: "`virtual final`, and it answers the RUNTIME type -- the derived one. A composed base cannot see its deriver, so the derived constructor installs the CLR `this` and this member reflects over it"},
		},
		Excluded: []bclExcludedMember{
			{CLRMember: "Data", Kind: "BCL_PROJECTION_BLOCKED_EXTERNAL",
				Needs:  "System.Collections.IDictionary, the non-generic dictionary contract",
				Reason: "genuinely public CLR surface that IS absent from the Go projection. The non-generic IDictionary is named nowhere else in the XNA 4.0 Windows profile except as the same blocker on the thirteen Design type converters, and projecting it means projecting DictionaryEntry, IDictionaryEnumerator and the whole non-generic collection family"},
			{CLRMember: "TargetSite", Kind: "BCL_PROJECTION_BLOCKED_EXTERNAL",
				Needs:  "System.Reflection.MethodBase, and the reflection MEMBER model behind it",
				Reason: "genuinely public CLR surface that IS absent. CNA-Go maps System.Type to reflect.Type and nothing else; MethodBase reaches MethodInfo, ConstructorInfo, ParameterInfo, MemberInfo and Module, none of which the profile names. It would also be null for an exception nothing threw, which is every exception this projection can produce"},
			{CLRMember: "GetObjectData", Kind: "BCL_PROJECTION_BLOCKED_EXTERNAL",
				Needs:  "System.Runtime.Serialization.SerializationInfo (43 public members) and StreamingContext (6), whose own inventory reaches System.Decimal (89) and System.DateTime (91)",
				Reason: "the same closure that blocks Dictionary<K,V>::GetObjectData, measured in Foundation 74 and unchanged: 238 public BCL members in types the profile names nowhere else"},
			{CLRMember: ".ctor()", Reason: "the adapter's own constructors are the projection of the three public ones; no XNA member is projected from them"},
			{CLRMember: ".ctor(string)", Reason: "as above"},
			{CLRMember: ".ctor(string,Exception)", Reason: "as above"},
			{CLRMember: ".ctor(SerializationInfo,StreamingContext)", Reason: "`family`, and the CLR does not inherit constructors"},
			{CLRMember: "HResult", Reason: "`family`; it is not public surface. ExternalException::get_ErrorCode is the one public reader of it in the profile"},
			{CLRMember: "SerializeObjectState", Reason: "a `family` event over EventHandler<SafeSerializationEventArgs>; not public surface, and its args type is a serialization type"},
			{CLRMember: "Init", Reason: "private helper the constructors call"},
			{CLRMember: "GetClassName", Reason: "private helper Message and ToString call"},
			{CLRMember: "GetStackTrace", Reason: "private helper StackTrace and ToString call"},
			{CLRMember: "_Exception.GetType", Reason: "private explicit implementation of the COM _Exception interface; the settled BCL-interface rule projects nothing for it"},
			{CLRMember: "ISerializable.GetObjectData", Reason: "the public GetObjectData IS the implementation; there is no separate explicit one"},
		},
	},

	// Foundation 74. System.Collections.Generic.IEqualityComparer<T>, which
	// Dictionary<K,V>::get_Comparer returns.
	//
	// It is an interface with exactly two abstract members, so it projects as a
	// Go interface with two methods and no more. Neither is fallible: an
	// implementor's whole contract is to answer.
	"System.Collections.Generic.IEqualityComparer`1": {
		GoAdapter:       "IEqualityComparer[T]",
		GenericArity:    1,
		BehaviorLevel:   "SUPPORTED",
		Authority:       "mscorlib.dll 4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0",
		AuthoritySHA256: "5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63",
		Rationale:       "the equality contract Dictionary<K,V> stores and returns; the profile's only instance is EqualityComparer<string>.Default, whose Equals is ordinal string equality and whose GetHashCode is the pinned String::GetHashCode",
		Members: []bclInheritedMember{
			{Member: bclMethod("Equals", "System.Boolean", bclParameter("x", "!0"), bclParameter("y", "!0")),
				Rationale: "abstract; GenericEqualityComparer<string> implements it as `x != null ? x.Equals(y) : y == null`, which for Go string reduces to =="},
			{Member: bclMethod("GetHashCode", "System.Int32", bclParameter("obj", "!0")),
				Rationale: "abstract; GenericEqualityComparer<string> forwards to String::GetHashCode, the 0x15051505 two-accumulator loop over UTF-16 code units, which this mscorlib implements with no randomised-hashing branch"},
		},
		Excluded: []bclExcludedMember{},
	},

	// Foundation 74. System.Collections.Generic.KeyValuePair<TKey,TValue>, the
	// element type of Dictionary<K,V>'s enumerator.
	//
	// A struct with two private fields, so it projects as a Go struct value
	// with unexported fields. Its constructor is not inherited by anything and
	// is projected as the NewKeyValuePair function instead.
	"System.Collections.Generic.KeyValuePair`2": {
		GoAdapter:       "KeyValuePair[TKey, TValue]",
		GenericArity:    2,
		BehaviorLevel:   "SUPPORTED",
		Authority:       "mscorlib.dll 4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0",
		AuthoritySHA256: "5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63",
		Rationale:       "the immutable pair Dictionary<K,V>'s enumerator yields; both properties are get-only, which is why the Go fields are unexported rather than public",
		Members: []bclInheritedMember{
			{Member: bclProperty("Key", "!0", true, false), Rationale: "get_Key is one ldfld"},
			{Member: bclProperty("Value", "!1", true, false), Rationale: "get_Value is one ldfld"},
			{Member: bclMethod("ToString", "System.String"),
				Rationale: "`[` + Key.ToString() + `, ` + Value.ToString() + `]`, where a null half contributes NOTHING rather than a placeholder and the separator is emitted unconditionally, so a pair of nulls renders as `[, ]`"},
		},
		Excluded: []bclExcludedMember{
			{CLRMember: ".ctor(TKey,TValue)", Reason: "the constructor is projected as the NewKeyValuePair function; no XNA member is projected from it"},
		},
	},

	// Foundation 74. Dictionary<TKey,TValue>.KeyCollection, returned by
	// get_Keys.
	//
	// A sealed nested public class that holds the dictionary it was made from
	// and reads through to it. Its public surface is exactly Count, CopyTo and
	// GetEnumerator; every mutator is a private explicit implementation of
	// ICollection<TKey> or IList, so the view is read-only without a new
	// decision, exactly as ReadOnlyCollection<T> is.
	"System.Collections.Generic.Dictionary`2+KeyCollection": {
		GoAdapter:       "DictionaryKeyCollection[TKey, TValue]",
		GenericArity:    2,
		BehaviorLevel:   "SUPPORTED",
		Authority:       "mscorlib.dll 4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0",
		AuthoritySHA256: "5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63",
		Rationale:       "a live view over the dictionary's entries array, not a snapshot; the Go name is the settled nested-name rule, declaring type concatenated with nested type",
		Members: []bclInheritedMember{
			{Member: bclProperty("Count", "System.Int32", true, false),
				Rationale: "get_Count is one forwarded Dictionary::get_Count, so it tracks the dictionary rather than the view"},
			{Member: bclMethod("CopyTo", "System.Void", bclParameter("array", "!0[]"), bclParameter("index", "System.Int32")),
				Rationale: "three failures of its own, in order: a null destination, an index outside [0, array.Length], and remaining room smaller than Count; it then copies the live keys in entries-array order"},
			{Member: bclMethod("GetEnumerator", "System.Collections.Generic.Dictionary`2+KeyCollection+Enumerator[!0,!1]"),
				Rationale: "the nested KeyCollection.Enumerator struct, which IS an IEnumerator<TKey>, so the settled List<T>.Enumerator rule projects it as Iterator<T>; it is version-checked against the DICTIONARY, not the view"},
		},
		Excluded: []bclExcludedMember{
			{CLRMember: ".ctor(Dictionary`2<TKey,TValue>)", Kind: "LANGUAGE_MAPPING_LIMITATION",
				Needs:  "a public Go spelling of Dictionary`2 itself",
				Reason: "the constructor is public CLR surface, and its ONLY parameter type is the base the settled composition rule keeps private: bclBaseAdapters projects Dictionary<K,V> as the unexported dictionaryBase adapter precisely so no consumer can name it, so no exported Go function could take one. The profile's only producer of a KeyCollection is get_Keys, which hands back the cached view"},
			{CLRMember: "ICollection<TKey>.IsReadOnly", Reason: "private explicit implementation; the settled BCL-interface rule projects nothing for it"},
			{CLRMember: "ICollection<TKey>.Add", Reason: "private explicit mutator that throws NotSupportedException; read-only means it is not public surface"},
			{CLRMember: "ICollection<TKey>.Clear", Reason: "private explicit mutator"},
			{CLRMember: "ICollection<TKey>.Contains", Reason: "private explicit implementation"},
			{CLRMember: "ICollection<TKey>.Remove", Reason: "private explicit mutator"},
			{CLRMember: "IEnumerable<TKey>.GetEnumerator", Reason: "private explicit implementation; the public GetEnumerator carries enumeration"},
			{CLRMember: "IEnumerable.GetEnumerator", Reason: "private explicit implementation"},
			{CLRMember: "ICollection.IsSynchronized", Reason: "private explicit implementation"},
			{CLRMember: "ICollection.SyncRoot", Reason: "private explicit implementation, and CNA-Go projects no CLR sync root"},
			{CLRMember: "ICollection.CopyTo", Reason: "private explicit implementation"},
		},
	},

	// Foundation 74. Dictionary<TKey,TValue>.ValueCollection, KeyCollection's
	// mirror over the other half of each entry.
	"System.Collections.Generic.Dictionary`2+ValueCollection": {
		GoAdapter:       "DictionaryValueCollection[TKey, TValue]",
		GenericArity:    2,
		BehaviorLevel:   "SUPPORTED",
		Authority:       "mscorlib.dll 4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0",
		AuthoritySHA256: "5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63",
		Rationale:       "a live view over the dictionary's entries array, identical to KeyCollection over the value half",
		Members: []bclInheritedMember{
			{Member: bclProperty("Count", "System.Int32", true, false),
				Rationale: "get_Count is one forwarded Dictionary::get_Count"},
			{Member: bclMethod("CopyTo", "System.Void", bclParameter("array", "!1[]"), bclParameter("index", "System.Int32")),
				Rationale: "KeyCollection::CopyTo's three checks in the same order, over the value half"},
			{Member: bclMethod("GetEnumerator", "System.Collections.Generic.Dictionary`2+ValueCollection+Enumerator[!0,!1]"),
				Rationale: "the nested ValueCollection.Enumerator struct, projected through the same settled rule"},
		},
		Excluded: []bclExcludedMember{
			{CLRMember: ".ctor(Dictionary`2<TKey,TValue>)", Kind: "LANGUAGE_MAPPING_LIMITATION",
				Needs:  "a public Go spelling of Dictionary`2 itself",
				Reason: "public CLR surface whose only parameter type is the privately composed base; see KeyCollection's entry"},
			{CLRMember: "ICollection<TValue>.IsReadOnly", Reason: "private explicit implementation"},
			{CLRMember: "ICollection<TValue>.Add", Reason: "private explicit mutator that throws NotSupportedException"},
			{CLRMember: "ICollection<TValue>.Clear", Reason: "private explicit mutator"},
			{CLRMember: "ICollection<TValue>.Contains", Reason: "private explicit implementation"},
			{CLRMember: "ICollection<TValue>.Remove", Reason: "private explicit mutator"},
			{CLRMember: "IEnumerable<TValue>.GetEnumerator", Reason: "private explicit implementation"},
			{CLRMember: "IEnumerable.GetEnumerator", Reason: "private explicit implementation"},
			{CLRMember: "ICollection.IsSynchronized", Reason: "private explicit implementation"},
			{CLRMember: "ICollection.SyncRoot", Reason: "private explicit implementation"},
			{CLRMember: "ICollection.CopyTo", Reason: "private explicit implementation"},
		},
	},
}

// bclSignatureAdapterConstructors are the exported functions that build a
// signature adapter, declared per adapter so a stray exported constructor is
// still an unexpected member.
var bclSignatureAdapterConstructors = map[string]string{
	"NewReadOnlyCollectionOverSingles": "System.Collections.ObjectModel.ReadOnlyCollection`1",
	// Foundation 68. The reference-element counterpart, for
	// GraphicsAdapter::get_Adapters. It is a SECOND constructor over the same
	// adapter rather than a generalisation of the first because the element
	// comparers differ and the difference is observable: Single's comparer
	// finds a NaN, and a reference comparer is identity.
	"NewReadOnlyCollectionOverReferences": "System.Collections.ObjectModel.ReadOnlyCollection`1",
	// Foundation 69. The VALUE-element counterpart, for
	// SpriteFont::get_Characters, whose CLR type is ReadOnlyCollection<char>.
	// A third constructor rather than a use of the second because a font's
	// characters are values: a name saying "over references" would describe
	// the opposite of what the collection holds.
	"NewReadOnlyCollectionOverCharacters": "System.Collections.ObjectModel.ReadOnlyCollection`1",
	// Foundation 74. KeyValuePair<TKey,TValue>::.ctor, which the Dictionary
	// enumerator's Current builds with `newobj` on every step. It is the one
	// new signature adapter whose CLR constructor is public AND reachable, so
	// it gets a named Go constructor rather than an exclusion.
	"NewKeyValuePair": "System.Collections.Generic.KeyValuePair`2",
	// Foundation 76. System.Exception's three public constructors. They are
	// named constructors rather than an exclusion because a consumer really
	// does allocate one: `new Exception(message)` is what the reference writes
	// when it needs an inner exception to chain.
	"NewException":                     "System.Exception",
	"NewExceptionByString":             "System.Exception",
	"NewExceptionByStringAndException": "System.Exception",
}

// bclSignatureAdapterGoName is the Go type name of one signature adapter,
// without its type parameters.
func bclSignatureAdapterGoName(adapter bclBaseAdapter) string {
	if open := strings.Index(adapter.GoAdapter, "["); open >= 0 {
		return adapter.GoAdapter[:open]
	}
	return adapter.GoAdapter
}

// ---------------------------------------------------------------------------
// Foundation 31 — the Game base-call language adapters.
// ---------------------------------------------------------------------------

// gameBaseCallAdapter declares one package-level Go function that runs the base
// body of one protected virtual member of Microsoft.Xna.Framework.Game.
//
// The family exists because GameCallbacks projects CLR protected virtual
// OVERRIDES, and in CLR the override -- not the framework -- decides whether
// and when the base body runs. Go has no `base` keyword, so the base body needs
// a name a callback can call, and it must be a name the callback is free NOT to
// call.
type gameBaseCallAdapter struct {
	// CLRMember is the protected virtual whose base body this runs, spelled as
	// it appears in the pinned contract.
	CLRMember string
	// GoFunction is the exported package-level function in the framework
	// package. It is never a method on Game: the projected XNA member surface
	// of Game must not gain a name Microsoft never declared.
	GoFunction string
	// Parameters and Results are the exact Go signature the function must
	// have. The first parameter is always *Game, which is the `this` a CLR base
	// call passes implicitly.
	Parameters []string
	Results    []string
	// Fallibility names every reason this function can report an error, so a
	// synthetic error result cannot be added without saying why.
	Fallibility []gameBaseCallFallibility
	// ReferenceBody is what the reference base body does, in order.
	ReferenceBody []string
	// Deferred records each reference step this projection does NOT reproduce,
	// with the reason and its class. A base-call adapter that defers a step
	// without recording it is a verifier failure, exactly as a deferred BCL
	// base that records no blocker is.
	Deferred []gameBaseCallDeferral
}

type gameBaseCallFallibility struct {
	// Kind is GUARD for the Go-only nil/unconstructed-Game check, or
	// REFERENCE for a failure the reference itself can produce.
	Kind   string
	Reason string
}

type gameBaseCallDeferral struct {
	// Step is the reference step that is not reproduced.
	Step string
	// Class is SUBSYSTEM when the step needs a .NET or CNA subsystem CNA-Go
	// does not have, ARCHITECTURE when a cross-cutting decision blocks it, and
	// UNOBSERVABLE when the step has no effect any projected surface can see.
	Class string
	// Reason is the exact evidence.
	Reason string
	// Observable records whether skipping the step changes anything the
	// managed component part can observe. A deferral that IS observable is a
	// stop condition, not a deferral, so the verifier rejects one.
	Observable bool
}

// gameBaseCallAdapters is the closed registry. Its keys are exactly the five
// GameCallbacks members, which are exactly the five Game protected virtuals the
// mapper redirects to GameCallbacks: a base-call adapter for anything else
// would be an invented helper, and a GameCallbacks member without one would
// leave an override unable to reach its base.
var gameBaseCallAdapters = map[string]gameBaseCallAdapter{
	"Initialize": {
		CLRMember:  "Microsoft.Xna.Framework.Game::Initialize",
		GoFunction: "GameBaseInitialize",
		Parameters: []string{"*Game"},
		Results:    []string{"error"},
		Fallibility: []gameBaseCallFallibility{
			{Kind: "GUARD", Reason: "Go can produce a Game whose constructor never ran; Game.Run and Game.Exit already report exactly this condition"},
			{Kind: "REFERENCE", Reason: "the drain calls IGameComponent::Initialize, which the settled contract projects as fallible because DrawableGameComponent.Initialize throws when IGraphicsDeviceService is absent"},
		},
		ReferenceBody: []string{
			"HookDeviceEvents()",
			"while (notYetInitialized.Count > 0) { notYetInitialized[0].Initialize(); notYetInitialized.RemoveAt(0); }",
			"if (graphicsDeviceService != null && graphicsDeviceService.GraphicsDevice != null) LoadContent();",
		},
		Deferred: []gameBaseCallDeferral{
			{Step: "HookDeviceEvents()", Class: "ARCHITECTURE",
				Reason: "it resolves Microsoft.Xna.Framework.Graphics.IGraphicsDeviceService out of Services and subscribes four device handlers to it. The contract lives in the GRAPHICS package, which imports the framework package, so the settled cross-package cycle rule already projects Game's device-typed members into the descendant package and the framework package cannot name the type. This deferral rested on a SECOND ground until Foundation 49 -- that nothing in CNA-Go could publish the service at all -- and that ground is retired: GraphicsDeviceManager's constructor now registers an adapter under IGraphicsDeviceService, and GetService finds it. What remains is the naming ground alone"},
			{Step: "if (graphicsDeviceService != null && ...) LoadContent()", Class: "ARCHITECTURE",
				Reason: "its condition is the field HookDeviceEvents assigns, so with no device service the reference does not call LoadContent from Initialize either. CNA-Go's LoadContent arrives from the native CNA load_content callback, whose documented order is initialize, then the runtime's own component and device setup, then load_content"},
		},
	},
	"LoadContent": {
		CLRMember:  "Microsoft.Xna.Framework.Game::LoadContent",
		GoFunction: "GameBaseLoadContent",
		Parameters: []string{"*Game"},
		Results:    []string{"error"},
		Fallibility: []gameBaseCallFallibility{
			{Kind: "GUARD", Reason: "Go can produce a Game whose constructor never ran"},
		},
		ReferenceBody: []string{"ret  // code size 1"},
	},
	"Update": {
		CLRMember:  "Microsoft.Xna.Framework.Game::Update",
		GoFunction: "GameBaseUpdate",
		Parameters: []string{"*Game", "GameTime"},
		Results:    []string{"error"},
		Fallibility: []gameBaseCallFallibility{
			{Kind: "GUARD", Reason: "Go can produce a Game whose constructor never ran"},
		},
		ReferenceBody: []string{
			"Logger.BeginLogEvent(Update, \"\")",
			"for (i = 0; i < updateableComponents.Count; i++) currentlyUpdatingComponents.Add(updateableComponents[i])",
			"for (j = 0; j < currentlyUpdatingComponents.Count; j++) if (currentlyUpdatingComponents[j].Enabled) currentlyUpdatingComponents[j].Update(gameTime)",
			"currentlyUpdatingComponents.Clear()",
			"FrameworkDispatcher.Update()",
			"doneFirstUpdate = true",
			"Logger.EndLogEvent(Update, \"\")",
		},
		Deferred: []gameBaseCallDeferral{
			{Step: "FrameworkDispatcher.Update()", Class: "SUBSYSTEM",
				Reason: "it pumps the media and audio subsystems; CNA-Go has no media backend and no audio backend, so there is nothing to pump and nothing is invented in its place"},
			{Step: "Logger.BeginLogEvent / Logger.EndLogEvent", Class: "UNOBSERVABLE",
				Reason: "XNA's private profiling channel; it is not public surface and produces no effect any projected member can observe"},
		},
	},
	"Draw": {
		CLRMember:  "Microsoft.Xna.Framework.Game::Draw",
		GoFunction: "GameBaseDraw",
		Parameters: []string{"*Game", "GameTime"},
		Results:    []string{"error"},
		Fallibility: []gameBaseCallFallibility{
			{Kind: "GUARD", Reason: "Go can produce a Game whose constructor never ran"},
		},
		ReferenceBody: []string{
			"for (i = 0; i < drawableComponents.Count; i++) currentlyDrawingComponents.Add(drawableComponents[i])",
			"for (j = 0; j < currentlyDrawingComponents.Count; j++) if (currentlyDrawingComponents[j].Visible) currentlyDrawingComponents[j].Draw(gameTime)",
			"currentlyDrawingComponents.Clear()",
		},
	},
	"UnloadContent": {
		CLRMember:  "Microsoft.Xna.Framework.Game::UnloadContent",
		GoFunction: "GameBaseUnloadContent",
		Parameters: []string{"*Game"},
		Results:    []string{"error"},
		Fallibility: []gameBaseCallFallibility{
			{Kind: "GUARD", Reason: "Go can produce a Game whose constructor never ran"},
		},
		ReferenceBody: []string{"ret  // code size 1"},
	},
}

// publicCLRMemberCount is how many PUBLIC CLR members one type declares, which
// is what a derived class inherits into its own public surface.
//
// Constructors are excluded because they are not inherited. A property counts
// once when either accessor is public, which is the member the contract
// declares; the projection splits it into accessors separately. Events carry no
// accessibility in the pinned contract and every XNA event is public, so they
// count.
func publicCLRMemberCount(t contractType) int {
	total := 0
	for _, m := range t.Members {
		switch m.Kind {
		case "constructor":
			continue
		case "property":
			if valueOrEmpty(m.GetAccess) == "public" || valueOrEmpty(m.SetAccess) == "public" {
				total++
			}
		case "event":
			total++
		default:
			if m.Access == "public" {
				total++
			}
		}
	}
	return total
}

// ---------------------------------------------------------------------------
// Foundation 33 — the XNA-to-XNA base frontier.
// ---------------------------------------------------------------------------

// xnaBaseRelationship records one CLR class in the pinned profile that another
// class in the SAME profile inherits from.
//
// This is a different frontier from bclBaseRelationships and it had never been
// measured. A BCL base is a type outside the contract that CNA-Go decides how
// to model; an XNA base is a type INSIDE the contract that CNA-Go already
// projects, or will, and whose public surface a derived type inherits without
// the contract redeclaring it.
//
// The gap that makes this worth measuring: Texture2D's contract declares
// sixteen members and inherits nine more from Texture and GraphicsResource --
// Name, Tag, GraphicsDevice, IsDisposed, Dispose, ToString, Disposing,
// LevelCount and Format. CNA-Go projects Texture2D with none of them, and until
// now NOTHING recorded that. The same silence covered SpriteBatch. Recording
// every relationship with a status is what turns a silent omission into a
// measured deferral, exactly as Foundation 29 did for BCL bases.
type xnaBaseRelationship struct {
	// Status is COMPOSED when CNA-Go re-exposes the base's public surface on
	// its derived types, and DEFERRED when it does not.
	Status string
	// Blockers is what stands in the way. A DEFERRED relationship that records
	// nothing is a verifier failure.
	Blockers []xnaBaseBlocker
}

type xnaBaseBlocker struct {
	// Class is one of xnaBaseBlockerClasses.
	Class string
	// Detail is the exact evidence.
	Detail string
}

// xnaBaseBlockerClasses are the only classes a blocker may carry.
var xnaBaseBlockerClasses = map[string]string{
	// A cross-cutting projection decision no single member carries.
	"ARCHITECTURE": "a cross-cutting projection decision no single member carries",
	// The base or a member of it belongs to a .NET or CNA subsystem CNA-Go
	// does not map.
	"SUBSYSTEM": "the base reaches a subsystem CNA-Go does not map",
	// The base itself is a missing type, usually because ITS base is deferred.
	"TRANSITIVE": "the base is itself unprojected because its own base is deferred",
	// The base is owned by native code in a way CNA-Go has not decided.
	"NATIVE_OWNERSHIP": "the base is natively owned and its ownership is undecided",
}

// xnaBaseComposition is the ARCHITECTURE blocker every deferred XNA base
// carries, because it is a necessary condition for projecting any of them. It
// is stated once and shared so twelve entries cannot drift apart.
var xnaBaseComposition = xnaBaseBlocker{
	Class:  "ARCHITECTURE",
	Detail: "CNA-Go has no architecture for XNA-to-XNA class inheritance. The settled BCL base composition covers a base OUTSIDE the contract and is registered per BCL family; an XNA base is a projected type whose public surface the contract does not redeclare on the derived type, so a faithful derived projection needs (a) a composition and forwarding rule for a base that is itself an XNA identity, (b) a third provenance class beside XNA-declared and BCL-inherited so no member is counted twice, and (c) an override adapter for the base's protected virtuals, which GameCallbacks solves for Game alone. Exported Go embedding is already refused by BASE_MAPPING_MISMATCH and remains refused",
}

// xnaBaseRelationships is the closed registry. Every XNA-to-XNA base link the
// pinned contract declares must appear here; one that does not is a diagnostic
// rather than a silent omission.
var xnaBaseRelationships = map[string]xnaBaseRelationship{
	// Foundation 32's frontier. Its two derived types are the profile's only
	// XNA classes inheriting from an XNA class CNA-Go has already completed,
	// which is what makes the architecture blocker the LIVE one rather than a
	// theoretical one.
	// The first COMPOSED relationship, and the architecture proof.
	//
	// Foundation 40 measured why this family is the safe one to start with:
	// NOTHING in the whole profile names GameComponent in a public signature,
	// so no derived value can ever be required to stand in for one and private
	// named composition is exactly sufficient rather than a compromise.
	//
	// COMPOSED states that the INHERITANCE is projected -- the inherited public
	// surface is enumerated, attributed and required on the derived type. It
	// does not state that any derived type is complete, and neither is. Their
	// remaining blockers are recorded per derived type below and are entirely
	// about device and GamerServices runtime, not about inheritance.
	"Microsoft.Xna.Framework.GameComponent": {Status: "COMPOSED", Blockers: []xnaBaseBlocker{
		{Class: "SUBSYSTEM", Detail: "the inheritance is projected and ONE of the two derived types is now complete. Foundation 46 projected DrawableGameComponent: its Initialize resolves Microsoft.Xna.Framework.Graphics.IGraphicsDeviceService out of Game.Services, which the framework package cannot name because the Graphics package imports it, and internal/servicebridge resolves that with two function values installed from package inits -- no public API, no retained object, and no import cycle. GamerServicesComponent remains blocked and its blockers are not inheritance either: GamerServicesDispatcher lives in Microsoft.Xna.Framework.GamerServices.dll, which is not one of the seven pinned assemblies and has no CNA runtime behind it. Game.Window.Handle stopped being one of its blockers in Foundation 45"},
	}},

	// The one Foundation 25 measured from the other side: it alone blocked
	// seven missing types, and eleven derive from it.
	//
	// Foundation 56 composed it. The NATIVE_OWNERSHIP blocker was the real one
	// and it is now answered rather than deferred: the resource handle lives on
	// GraphicsResource, exactly where the reference's `_internalHandle` lives,
	// so there is ONE native owner per logical object and a derived wrapper
	// never creates a second. `interop.Resource` carries its own kind tag, so
	// the type-specific destruction the reference's ReleaseNativeObject
	// overrides perform is already inside `Resource.Dispose`.
	"Microsoft.Xna.Framework.Graphics.GraphicsResource": {Status: "COMPOSED", Blockers: []xnaBaseBlocker{
		{Class: "SUBSYSTEM", Detail: "the inheritance is projected and ten of the eleven derived types are complete: the four state objects, SpriteBatch, Texture and its three derived textures, VertexDeclaration, IndexBuffer and VertexBuffer. The one that remains is Effect, whose own derived family needs EffectParameter"},
	}},
	// Foundation 71 emptied this one's blocker list, which no other entry has:
	// the inheritance is projected and ALL THREE derived types are complete.
	// The blocker it used to carry -- "Texture3D and TextureCube need CNA
	// volume and cube texture routes, which the pinned 0.21.0 ABI does not
	// expose" -- was FALSE when it was written: CNA/C/texture_volume.h declares
	// sixteen of them, and Foundation 71 bound ten. An empty list is the only
	// honest state for a family with nothing left to record.
	"Microsoft.Xna.Framework.Graphics.Texture": {Status: "COMPOSED"},
	// Foundation 58 composed it, and it is the one composed base whose
	// substitutability requirement is LIVE: seven public positions name
	// Texture2D and a projected type now derives from it, so the parameter
	// positions project to Texture2DReference rather than to *Texture2D.
	"Microsoft.Xna.Framework.Graphics.Texture2D": {Status: "COMPOSED", Blockers: []xnaBaseBlocker{
		{Class: "SUBSYSTEM", Detail: "the inheritance is projected and its one derived type, RenderTarget2D, is complete. Binding a render target needs a renderer with real off-screen storage, which the qualified HEADLESS artifact does not have and the qualified SOFTWARE artifact does"},
	}},
	// Foundation 71 projected TextureCube and Foundation 73 its one derived
	// type, so this relationship joins the composed set with an empty blocker
	// list -- the second one in the registry to have nothing left to record.
	"Microsoft.Xna.Framework.Graphics.TextureCube": {Status: "COMPOSED"},
	// Foundation 79 composed it, and every blocker it carried is now false
	// rather than merely deferred. The TRANSITIVE one died in Foundation 72,
	// which projected Effect. The SUBSYSTEM one said the derived effects "reach
	// EffectParameter, which calls unmanaged D3DX": that is what the REFERENCE
	// does, and the projection reaches CNA's own stock-effect state instead --
	// the Foundation 79 probe measured CNA's BasicEffect reporting
	// PARAMETER_COUNT 0 on both qualified artifacts, so there is no parameter
	// path to map and none is needed.
	//
	// It is also the second base whose RETURNS widen. Clone is declared to
	// return Effect and all five stock effects override it to return their own
	// class, so the concrete pointer would hand back a base half with no way to
	// reach the object that owns it.
	"Microsoft.Xna.Framework.Graphics.Effect": {Status: "COMPOSED", Blockers: []xnaBaseBlocker{
		{Class: "SUBSYSTEM", Detail: "the inheritance is projected and ONE of the six derived types is complete, BasicEffect. The five that remain -- AlphaTestEffect, DualTextureEffect, EnvironmentMapEffect, SkinnedEffect and EffectMaterial -- are the same shape and are blocked by nothing but the work"},
	}},
	"Microsoft.Xna.Framework.Graphics.IndexBuffer": {Status: "DEFERRED", Blockers: []xnaBaseBlocker{
		xnaBaseComposition,
		{Class: "TRANSITIVE", Detail: "IndexBuffer extends GraphicsResource and is itself a missing type"},
	}},
	"Microsoft.Xna.Framework.Graphics.VertexBuffer": {Status: "DEFERRED", Blockers: []xnaBaseBlocker{
		xnaBaseComposition,
		{Class: "TRANSITIVE", Detail: "VertexBuffer extends GraphicsResource and is itself a missing type"},
	}},

	"Microsoft.Xna.Framework.Audio.SoundEffectInstance": {Status: "DEFERRED", Blockers: []xnaBaseBlocker{
		xnaBaseComposition,
		{Class: "SUBSYSTEM", Detail: "SoundEffectInstance is a missing type and CNA-Go has no audio backend: the qualification artifact pins a NULL audio renderer, so nothing behind it would play"},
	}},
	// Foundation 63 projected ContentManager itself. The relationship is still
	// deferred because its DERIVED type is not projected, and the blocker is
	// recorded as ResourceContentManager's rather than left as the stale claim
	// that CNA-Go maps no content subsystem -- which stopped being true then.
	"Microsoft.Xna.Framework.Content.ContentManager": {Status: "DEFERRED", Blockers: []xnaBaseBlocker{
		xnaBaseComposition,
		{Class: "SUBSYSTEM", Detail: "ContentManager is projected as of Foundation 63 and its one derived type, ResourceContentManager, is not: it loads from a System.Resources.ResourceManager, which is a BCL subsystem outside the seven pinned assemblies and has no CNA counterpart"},
	}},
	"Microsoft.Xna.Framework.Content.ContentTypeReader": {Status: "DEFERRED", Blockers: []xnaBaseBlocker{
		xnaBaseComposition,
		{Class: "SUBSYSTEM", Detail: "the same Content/XNB subsystem, and the derived type is the GENERIC ContentTypeReader`1, so the relationship would also need a rule for a generic class deriving from a non-generic one"},
	}},
	"Microsoft.Xna.Framework.Design.MathTypeConverter": {Status: "DEFERRED", Blockers: []xnaBaseBlocker{
		xnaBaseComposition,
		{Class: "SUBSYSTEM", Detail: "MathTypeConverter extends System.ComponentModel.ExpandableObjectConverter, which is already a DEFERRED BCL base with three recorded blockers, so the twelve Design converters are blocked by a BCL decision before the XNA one is reached"},
	}},
}

// ---------------------------------------------------------------------------
// Foundation 36 — the native game-signal bridge and the frame-hook frontier.
// ---------------------------------------------------------------------------

// gameNativeSignal declares one canonical CNA game signal, the CLR event it is
// bound to, and the reference raise path that event actually has.
//
// # The rule this registry enforces changed in Foundation 39
//
// It used to be "every CLR event Game declares is bound to exactly one
// canonical CNA signal, through exactly the raise path the reference uses".
// That is too weak, and it shipped a divergence: it is satisfied by binding a
// native signal to an event whose reference raise site is not the host's at
// all, which is what Foundation 34 did to Game::Disposed.
//
// The stronger rule, and the one measured now:
//
//	every projected XNA event must have its AUTHORITATIVE XNA raise path, and
//	a native signal may IMPLEMENT that path only when the semantics align
//
// Three of Game's four events are raised by GameHost, and the native CNA
// runtime plays GameHost's part, so for those three the native signal IS the
// reference's raise path. The fourth is raised by managed Dispose(bool), which
// CNA-Go now projects, so its raise path is in Go and the native signal -- which
// fires from native game destruction, a different moment entirely -- is bound
// for lifetime qualification and raises nothing public.
//
// The registry exists because binding a native signal to a projected event is a
// claim with four separable parts, and prose cannot hold any of them:
//
//  1. the CLR event really is declared by Game, and really does project to the
//     two-accessor pair the event mapping promises;
//  2. the raise site really is the protected virtual the reference routes
//     through -- or really is absent, which is a fact about Disposed rather
//     than an oversight;
//  3. the sender the projection pushes is the sender the IL pushes, which for
//     Exiting is `ldnull` and for the others is `ldarg.0`;
//  4. the runtime evidence for that signal is recorded honestly, so an event
//     the qualification environment cannot deliver says so instead of being
//     counted as verified.
type gameNativeSignal struct {
	// CNAConstant is the canonical C identity, and CNAIdentity its value. They
	// are recorded so the projection names a real signal rather than a Go
	// invention; the C-side chain from the canonical header through CNA-Go's
	// private manifest, bridge.h's mirror and the Go constants is separately
	// compiler-checked and measured by tools/native_abi.
	CNAConstant string
	CNAIdentity int
	// CLREvent is the event Game declares, spelled as the pinned contract does.
	CLREvent string
	// RaiseSite is the projected protected virtual the reference routes the
	// raise through, or empty when the reference declares none. Disposed is the
	// only one with none: Dispose(bool) invokes the delegate field directly.
	RaiseSite string
	// RaisePath says WHERE the projection actually raises the public event:
	//
	//	NATIVE_HOST_SIGNAL -- the canonical CNA signal drives the raise, because
	//	                      the reference's raise path is GameHost's and the
	//	                      native runtime plays GameHost's part
	//	MANAGED            -- the reference raises it from managed code CNA-Go
	//	                      projects, so the raise is in Go and the native
	//	                      signal MUST NOT drive it
	//
	// NativeSignalRole is the other half of the same fact, stated about the
	// signal instead of the event:
	//
	//	PUBLIC_EVENT_RAISE -- the signal drives the projected event
	//	LIFECYCLE_ONLY     -- the signal is bound and delivered for native
	//	                      lifetime qualification and raises nothing public
	//
	// The two are one decision seen from two ends and must agree.
	RaisePath        string
	NativeSignalRole string
	// ManagedRaiseSite names the projected Go member that raises the event when
	// RaisePath is MANAGED. Required then and forbidden otherwise, so a MANAGED
	// event cannot claim a raise path it does not have.
	ManagedRaiseSite string
	// NativeSignalMoment is what the CNA signal actually means, recorded for
	// every signal and load-bearing for a LIFECYCLE_ONLY one: it is the
	// statement of WHY the semantics do not align.
	NativeSignalMoment string
	// Sender is what the raise pushes: GAME for `ldarg.0`, NULL for `ldnull`.
	Sender string
	// EdgeTriggered records whether the reference's host handler suppresses a
	// repeated signal. The two activation events guard on Game::isActive;
	// Exiting and Disposed do not guard at all.
	EdgeTriggered bool
	// ReferencePath is the reference's chain from host signal to handler.
	ReferencePath []string
	// RuntimeEvidence is VERIFIED_NATIVE when the qualification artifact
	// actually delivers the signal, and NOT_RUN_ENVIRONMENT when it cannot.
	// EvidenceReason is required for the second and forbidden for the first: a
	// verified signal needs no excuse, and an unverified one must give its.
	RuntimeEvidence string
	EvidenceReason  string
}

// gameNativeSignalSenders are the only senders a raise may push. There is no
// third: every raise site in this family pushes either `this` or null.
var gameNativeSignalSenders = map[string]bool{"GAME": true, "NULL": true}

// gameEventRaisePaths and gameNativeSignalRoles are the two closed
// vocabularies Foundation 39 added, and they are two views of one decision.
var gameEventRaisePaths = map[string]string{
	"NATIVE_HOST_SIGNAL": "the canonical CNA signal drives the raise, because the reference's raise path is GameHost's and the native runtime plays GameHost's part",
	"MANAGED":            "the reference raises the event from managed code CNA-Go projects, so the raise is in Go and the native signal must not drive it",
}

var gameNativeSignalRoles = map[string]string{
	"PUBLIC_EVENT_RAISE": "the bound signal drives the projected event",
	"LIFECYCLE_ONLY":     "the bound signal is delivered and counted for native lifetime qualification and raises nothing public",
}

// gameEventRaisePathRoles pairs them. A signal cannot drive an event whose
// raise path is managed, and an event whose raise path is the host cannot have
// a signal that raises nothing.
var gameEventRaisePathRoles = map[string]string{
	"NATIVE_HOST_SIGNAL": "PUBLIC_EVENT_RAISE",
	"MANAGED":            "LIFECYCLE_ONLY",
}

// gameNativeSignalEvidence are the only runtime-evidence classes. A signal the
// environment cannot produce is NOT_RUN_ENVIRONMENT with a reason, never a
// quietly verified one.
var gameNativeSignalEvidence = map[string]bool{
	"VERIFIED_NATIVE": true, "NOT_RUN_ENVIRONMENT": true,
}

// gameNativeSignals is the closed registry, keyed by CLR event name. Its keys
// must be exactly the events Game declares: an event with no signal is a
// projected event with no raise path, and a signal for an event Game does not
// declare is an invention.
var gameNativeSignals = map[string]gameNativeSignal{
	"Activated": {
		CNAConstant: "CNA_GAME_EVENT_ACTIVATED", CNAIdentity: 0,
		CLREvent:  "Microsoft.Xna.Framework.Game::Activated",
		RaiseSite: "OnActivated", Sender: "GAME", EdgeTriggered: true,
		RaisePath: "NATIVE_HOST_SIGNAL", NativeSignalRole: "PUBLIC_EVENT_RAISE",
		NativeSignalMoment: "CNA reports the game becoming the active application, which is the moment GameHost::Activated fires and the moment HostActivated consumes",
		ReferencePath: []string{
			"EnsureHost(): host.Activated += HostActivated",
			"HostActivated: if (isActive) return; isActive = true;",
			"HostActivated: OnActivated(this, EventArgs.Empty)",
			"OnActivated: if (Activated != null) Activated(this, args)   // ldarg.0",
		},
		RuntimeEvidence: "VERIFIED_NATIVE",
	},
	"Deactivated": {
		CNAConstant: "CNA_GAME_EVENT_DEACTIVATED", CNAIdentity: 1,
		CLREvent:  "Microsoft.Xna.Framework.Game::Deactivated",
		RaiseSite: "OnDeactivated", Sender: "GAME", EdgeTriggered: true,
		RaisePath: "NATIVE_HOST_SIGNAL", NativeSignalRole: "PUBLIC_EVENT_RAISE",
		NativeSignalMoment: "CNA reports the game losing the active application, which is the moment GameHost::Deactivated fires and the moment HostDeactivated consumes",
		ReferencePath: []string{
			"EnsureHost(): host.Deactivated += HostDeactivated",
			"HostDeactivated: if (!isActive) return; isActive = false;",
			"HostDeactivated: OnDeactivated(this, EventArgs.Empty)",
			"OnDeactivated: if (Deactivated != null) Deactivated(this, args)   // ldarg.0",
		},
		RuntimeEvidence: "NOT_RUN_ENVIRONMENT",
		EvidenceReason:  "the qualification artifact runs a HEADLESS renderer with no window manager, so no focus transition away from the game can be produced and CNA_GAME_EVENT_DEACTIVATED is never delivered; the accessors, the edge-trigger guard and the raise path are proved without it and the delivery counter is left at zero",
	},
	"Exiting": {
		CNAConstant: "CNA_GAME_EVENT_EXITING", CNAIdentity: 3,
		CLREvent:  "Microsoft.Xna.Framework.Game::Exiting",
		RaiseSite: "OnExiting", Sender: "NULL", EdgeTriggered: false,
		RaisePath: "NATIVE_HOST_SIGNAL", NativeSignalRole: "PUBLIC_EVENT_RAISE",
		NativeSignalMoment: "CNA reports the run loop stopping, delivered from inside cna_game_run before it returns, which is where GameHost::Exiting fires and where HostExiting consumes it",
		ReferencePath: []string{
			"EnsureHost(): host.Exiting += HostExiting",
			"HostExiting: OnExiting(this, EventArgs.Empty)   // no guard",
			"OnExiting: if (Exiting != null) Exiting(null, args)   // ldnull, NOT ldarg.0",
		},
		RuntimeEvidence: "VERIFIED_NATIVE",
	},
	// The one event the host does not raise, and the one Foundation 39
	// corrects. Foundation 34 pointed the public event at the native disposal
	// signal because Dispose was not projected; it is now, so the event follows
	// the reference's own raise site and the signal keeps only the job it can
	// honestly do.
	"Disposed": {
		CNAConstant: "CNA_GAME_EVENT_DISPOSED", CNAIdentity: 2,
		CLREvent:  "Microsoft.Xna.Framework.Game::Disposed",
		RaiseSite: "", Sender: "GAME", EdgeTriggered: false,
		RaisePath: "MANAGED", NativeSignalRole: "LIFECYCLE_ONLY",
		ManagedRaiseSite:   "DisposeByBoolean",
		NativeSignalMoment: "CNA reports native game destruction, delivered from inside cna_game_destroy after content unloads. That is NOT where Game::Disposed is raised: the reference raises it from managed Dispose(bool), so the native signal fires for a consumer who never disposed anything and does not fire for a consumer who disposes without ever running. The signal stays bound and counted because it is what proves a registration outlives cna_game_destroy, and it raises nothing public",
		ReferencePath: []string{
			"Dispose(bool disposing): if (!disposing) return;",
			"Dispose(bool): lock (this)",
			"Dispose(bool): copy Components to an array, then Dispose() each element that is IDisposable",
			"Dispose(bool): dispose graphicsDeviceManager if it is IDisposable, then UnhookDeviceEvents()",
			"Dispose(bool): if (Disposed != null) Disposed(this, EventArgs.Empty)   // ldarg.0, no On... method",
		},
		RuntimeEvidence: "VERIFIED_NATIVE",
	},
}

// gameFrameHook declares one of Game's frame-boundary protected virtuals, the
// canonical CNA hook that sits at the same position, and whether CNA-Go
// installs it.
//
// The registry's load-bearing claim is the conditional one. CNA publishes four
// hooks that correspond position for position to these four virtuals, and
// CNA-Go installs each ONE IF AND ONLY IF the callback object handed to NewGame
// supplies the matching optional override.
//
// That is what keeps Foundation 31's rule intact while still making the four
// virtuals overridable. An unconditionally installed hook would run a base body
// at a position CNA-Go picked and make that base call mandatory; a hook
// installed only behind an override runs nothing CNA-Go chose, because the
// override IS the derived body and the base is reached only where the override
// writes the call. With no override the member stays NULL, which the canonical
// header defines as simply not called, so a consumer who never opts in observes
// exactly the native behaviour they observed before the mechanism existed.
//
// The capability is discovered once, at construction, and is a private
// structural interface rather than a public contract -- see the Capability
// fields below for why four separate unexported identities and not one.
type gameFrameHook struct {
	// CLRMember is the protected virtual, spelled as the pinned contract does.
	CLRMember string
	// GoName is the projected method on Game. These are methods rather than
	// GameCallbacks members because the mapper redirects exactly the five
	// protected virtuals GameCallbacks declares and no others.
	GoName     string
	Parameters []string
	Results    []string
	// NativeHook is the canonical CNA_GameFrameHooks member at the same
	// position in the frame, and Installation records WHEN CNA-Go installs it.
	// There are exactly two admitted classes:
	//
	//	NEVER       -- not installed at all; ReasonUninstalled is required
	//	ON_OVERRIDE -- installed exactly when the callback object supplies the
	//	               optional override; Capability is required
	//
	// There is deliberately no third. An unconditionally installed hook would
	// run a base body at a frame position CNA-Go picked and make that base call
	// mandatory, which is exactly what Foundation 31 settled against, so
	// "always" is not a class this registry can express -- declaring one is an
	// unclassified installation and a diagnostic.
	NativeHook        string
	Installation      string
	ReasonUninstalled string
	// BaseInvocation records whether CNA-Go ever runs the projected base body
	// on the consumer's behalf. EXPLICIT_ONLY is the only admitted value and
	// BaseInvocationEvidence must say how it is known.
	BaseInvocation         string
	BaseInvocationEvidence string
	// Capability names the UNEXPORTED single-method Go interface an external
	// callback object satisfies structurally to override this hook, and
	// CapabilityMethod/CapabilityParameters/CapabilityResults are the exact
	// method it has to declare. Required for ON_OVERRIDE and forbidden
	// otherwise.
	//
	// Four separate one-method identities, never one bundled contract: a CLR
	// subclass may override any SUBSET of these four virtuals, and a consumer
	// forced to supply a no-op for a virtual they did not override would be
	// installing a hook that takes the base's place. The identities are
	// unexported because Go interfaces are structural -- a consumer declares
	// the method and never names the type -- so the mechanism publishes no new
	// exported framework contract at all.
	Capability           string
	CapabilityMethod     string
	CapabilityParameters []string
	CapabilityResults    []string
	// NativeOrdering is the measured position of the native hook relative to
	// the reference's call site, so the correspondence is recorded rather than
	// asserted.
	NativeOrdering string
	// ReferenceBody is the base body, in order, and Deferred records every step
	// it does not reproduce, with the same classes and the same
	// observable-is-a-stop-condition rule the base-call adapters use.
	ReferenceBody []string
	Deferred      []gameBaseCallDeferral
}

// gameFrameHookInstallations are the only installation classes. Every hook
// declares exactly one, and there is no "always": an unconditional hook is the
// automatic base behaviour Foundation 31 refused.
var gameFrameHookInstallations = map[string]string{
	"NEVER":       "the canonical native hook is not installed at all",
	"ON_OVERRIDE": "the canonical native hook is installed if and only if the callback object supplies the optional override",
}

// gameFrameHookBaseInvocations are the only base-invocation classes. There is
// one, which is the point: CNA-Go never runs a base body on the consumer's
// behalf, at any frame position, under any installation class.
var gameFrameHookBaseInvocations = map[string]string{
	"EXPLICIT_ONLY": "the base body runs only where the override's own source calls the method on the Game, zero, one or many times",
}

// gameFrameHookCapabilityParameters is the parameter list every optional
// override method takes: the owning Game, which is the `this` a CLR base call
// passes implicitly and the receiver an explicit base call needs.
var gameFrameHookCapabilityParameters = []string{"*Game"}

// gameFrameHooks is the closed registry, keyed by Go method name. Its keys are
// exactly the four protected virtuals of Game that sit on a frame boundary and
// are NOT GameCallbacks members.
var gameFrameHooks = map[string]gameFrameHook{
	"BeginRun": {
		CLRMember: "Microsoft.Xna.Framework.Game::BeginRun",
		GoName:    "BeginRun", Parameters: nil, Results: []string{"error"},
		NativeHook: "CNA_GameFrameHooks::begin_run", Installation: "ON_OVERRIDE",
		Capability: "gameBeginRunOverride", CapabilityMethod: "BeginRun",
		BaseInvocation: "EXPLICIT_ONLY", BaseInvocationEvidence: "gameRuntimeCallbacks.BeginRun forwards to the captured override and to nothing else; Game.BeginRun is reached only if the override's own source calls it",
		CapabilityParameters: gameFrameHookCapabilityParameters, CapabilityResults: []string{"error"},
		NativeOrdering: "measured after initialize and load_content and before the first update, which is where RunGame calls BeginRun -- after Initialize returns and inRun is raised, before the priming Update",
		ReferenceBody:  []string{"IL_0000: ret   // code size 1"},
	},
	"EndRun": {
		CLRMember: "Microsoft.Xna.Framework.Game::EndRun",
		GoName:    "EndRun", Parameters: nil, Results: []string{"error"},
		NativeHook: "CNA_GameFrameHooks::end_run", Installation: "ON_OVERRIDE",
		Capability: "gameEndRunOverride", CapabilityMethod: "EndRun",
		BaseInvocation: "EXPLICIT_ONLY", BaseInvocationEvidence: "gameRuntimeCallbacks.EndRun forwards to the captured override and to nothing else; Game.EndRun is reached only if the override's own source calls it",
		CapabilityParameters: gameFrameHookCapabilityParameters, CapabilityResults: []string{"error"},
		NativeOrdering: "measured after the last frame and before cna_game_run returns, which is where RunGame calls EndRun -- immediately after the blocking host.Run() returns",
		ReferenceBody:  []string{"IL_0000: ret   // code size 1"},
	},
	"BeginDraw": {
		CLRMember: "Microsoft.Xna.Framework.Game::BeginDraw",
		GoName:    "BeginDraw", Parameters: nil, Results: []string{"bool", "error"},
		NativeHook: "CNA_GameFrameHooks::begin_draw", Installation: "ON_OVERRIDE",
		Capability: "gameBeginDrawOverride", CapabilityMethod: "BeginDraw",
		BaseInvocation: "EXPLICIT_ONLY", BaseInvocationEvidence: "the base admits every frame with no manager registered, so an override that answers false and still sees the frame skipped is a positive proof that the override's answer -- not the base's -- is what reaches CNA; measured by the native frame-hook-override scenario",
		CapabilityParameters: gameFrameHookCapabilityParameters, CapabilityResults: []string{"bool", "error"},
		NativeOrdering: "measured before each draw, and a frame whose out_should_draw is set to CNA_FALSE delivers neither draw nor end_draw -- the same shape as DrawFrame's `if (BeginDraw()) { Draw(); EndDraw(); }`",
		ReferenceBody: []string{
			"if (graphicsDeviceManager != null && !graphicsDeviceManager.BeginDraw()) return false;",
			"Logger.BeginLogEvent((LoggingEvent)4, \"\");",
			"return true;",
		},
		Deferred: []gameBaseCallDeferral{
			{Step: "graphicsDeviceManager resolution at the head of RunGame", Class: "ARCHITECTURE",
				Reason: "the statement after it calls CreateDevice on what it found, and the native CNA runtime owns and creates the device; Foundation 30 separately audited that the projected GraphicsDeviceManager satisfies neither service contract, so the reference's only registrar cannot run and the private field is permanently null -- a state the reference itself has whenever no manager is registered"},
			{Step: "Logger.BeginLogEvent((LoggingEvent)4, \"\")", Class: "UNOBSERVABLE",
				Reason: "Microsoft.Xna.Framework.Logger writes to a sink no projected member can read, so reproducing or omitting the call is indistinguishable from the public surface"},
		},
	},
	"EndDraw": {
		CLRMember: "Microsoft.Xna.Framework.Game::EndDraw",
		GoName:    "EndDraw", Parameters: nil, Results: []string{"error"},
		NativeHook: "CNA_GameFrameHooks::end_draw", Installation: "ON_OVERRIDE",
		Capability: "gameEndDrawOverride", CapabilityMethod: "EndDraw",
		BaseInvocation: "EXPLICIT_ONLY", BaseInvocationEvidence: "gameRuntimeCallbacks.EndDraw forwards to the captured override and to nothing else; Game.EndDraw is reached only if the override's own source calls it",
		CapabilityParameters: gameFrameHookCapabilityParameters, CapabilityResults: []string{"error"},
		NativeOrdering: "measured after each draw and skipped entirely on a frame begin_draw refused, which is where DrawFrame calls EndDraw -- inside the branch BeginDraw admitted",
		ReferenceBody: []string{
			"if (graphicsDeviceManager != null) graphicsDeviceManager.EndDraw();",
			"Logger.EndLogEvent((LoggingEvent)4, \"\");",
		},
		Deferred: []gameBaseCallDeferral{
			{Step: "graphicsDeviceManager resolution at the head of RunGame", Class: "ARCHITECTURE",
				Reason: "the same permanently-null private field BeginDraw records; the null branch is the one the reference itself takes whenever no manager is registered"},
			{Step: "Logger.EndLogEvent((LoggingEvent)4, \"\")", Class: "UNOBSERVABLE",
				Reason: "Microsoft.Xna.Framework.Logger writes to a sink no projected member can read"},
		},
	},
}

// ---------------------------------------------------------------------------
// Foundation 40 — the base-typed public signature inventory.
// ---------------------------------------------------------------------------

// buildXNABaseSubstitutability measures the ACTUAL CLR substitutability
// requirement of the profile, mechanically, from the pinned contract.
//
// # Why this has to be measured before any design is chosen
//
// Foundation 33 recorded twelve XNA-to-XNA base relationships and 41 derived
// types and stopped. The obvious next move is to pick a composition rule, and
// the obvious worry is that private composition cannot express CLR
// substitutability: in CLR, a DrawableGameComponent may be passed anywhere a
// GameComponent is named, and a Go struct holding a private *GameComponent
// cannot.
//
// But that worry is only real where the profile actually NAMES a base class in
// a public signature. If nothing in the whole contract takes or returns a
// GameComponent, then no consumer can ever need a DrawableGameComponent to
// stand in for one, and private composition with explicit forwarding is not a
// compromise -- it is exactly sufficient.
//
// So the question is answered by counting, not by argument. This walks every
// public member of every type in the profile and records every position -- a
// parameter, a return, a property type, a field type, or a generic argument --
// whose CLR type names a class that another class in the SAME profile derives
// from. The result is the complete requirement, with no speculation in it.
func buildXNABaseSubstitutability(s *expectedSurface, c contract) {
	byName := make(map[string]*contractType, len(c.Types))
	for i := range c.Types {
		byName[c.Types[i].Name] = &c.Types[i]
	}
	// The XNA-to-XNA base relationships, derived from the contract rather than
	// from the registry, so the inventory cannot inherit a registry mistake.
	derived := make(map[string][]string)
	for i := range c.Types {
		t := &c.Types[i]
		if t.BaseType == nil {
			continue
		}
		base := baseIdentityWithoutArguments(*t.BaseType)
		if base == t.Name {
			continue
		}
		if _, inProfile := byName[base]; inProfile {
			derived[base] = append(derived[base], t.Name)
		}
	}
	for base := range derived {
		sort.Strings(derived[base])
	}

	record := func(base, carrier, member, kind, position, clrType string) {
		s.XNABaseSubstitutability = append(s.XNABaseSubstitutability, xnaBaseSubstitutabilityRow{
			Base: base, Carrier: carrier, Member: member, MemberKind: kind,
			Position: position, CLRType: clrType,
		})
	}
	scan := func(carrier, member, kind, position, clrType string) {
		for _, named := range clrTypeIdentities(clrType) {
			if _, isBase := derived[named]; !isBase {
				continue
			}
			// A member of the base class itself, or of one of its own derived
			// types, naming the base is not a substitutability requirement on
			// an unrelated caller -- it is the family talking about itself.
			// Both are still recorded, with the position naming which it is,
			// because a projection has to satisfy them too.
			record(named, carrier, member, kind, position, clrType)
		}
	}

	for i := range c.Types {
		t := &c.Types[i]
		for j := range t.Members {
			m := &t.Members[j]
			switch m.Kind {
			case "property":
				if valueOrEmpty(m.GetAccess) != "public" && valueOrEmpty(m.SetAccess) != "public" {
					continue
				}
			case "event":
			default:
				if m.Access != "public" {
					continue
				}
			}
			// A method's type is in returnType; a property, field or event
			// carries it in `type` instead, and missing either would silently
			// shrink the inventory this whole measurement rests on.
			memberType := valueOrEmpty(m.ReturnType)
			position := "return"
			if memberType == "" {
				memberType = valueOrEmpty(m.Type)
				position = m.Kind + "-type"
				// Foundation 73. A property with a PUBLIC SETTER carries the
				// base at an assignable position, and the settled rule says a
				// setter's value IS a parameter position. Recording both as
				// "property-type" made the two indistinguishable, and the LIVE
				// measurement below could not then tell a base a consumer must
				// be able to PASS from one it can only RECEIVE.
				if m.Kind == "property" && valueOrEmpty(m.SetAccess) == "public" {
					position = "property-set"
				}
			}
			if memberType != "" {
				scan(t.Name, m.Name, m.Kind, position, memberType)
			}
			for _, p := range m.Parameters {
				scan(t.Name, m.Name, m.Kind, "parameter:"+p.Name, p.Type)
			}
		}
	}
	sort.SliceStable(s.XNABaseSubstitutability, func(i, j int) bool {
		left, right := s.XNABaseSubstitutability[i], s.XNABaseSubstitutability[j]
		if left.Base != right.Base {
			return left.Base < right.Base
		}
		if left.Carrier != right.Carrier {
			return left.Carrier < right.Carrier
		}
		if left.Member != right.Member {
			return left.Member < right.Member
		}
		return left.Position < right.Position
	})
	s.XNABaseDerivedByBase = derived
}

// clrTypeIdentities returns every named type identity a CLR type expression
// mentions, including the ones inside generic arguments and behind an array,
// pointer or by-reference marker. `Texture2D[]`, `Texture2D&` and
// `IEnumerable`1[Texture2D]` all mention Texture2D, and every one of them is a
// position a derived value would have to flow through.
func clrTypeIdentities(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	var identities []string
	var current strings.Builder
	flush := func() {
		token := strings.TrimSpace(current.String())
		current.Reset()
		token = strings.TrimRight(token, "&*")
		for strings.HasSuffix(token, "[]") {
			token = strings.TrimSuffix(token, "[]")
		}
		if token != "" {
			identities = append(identities, token)
		}
	}
	for _, character := range trimmed {
		switch character {
		case '[', ']', ',':
			flush()
		default:
			current.WriteRune(character)
		}
	}
	flush()
	return identities
}

// ---------------------------------------------------------------------------
// Foundation 41 — the XNA-to-XNA inheritance projection.
// ---------------------------------------------------------------------------

// The composition rule, stated once and measured everywhere.
//
// An XNA class that inherits another XNA class projects the base as PRIVATE
// NAMED COMPOSITION plus EXPLICIT MEASURED FORWARDING:
//
//	type DrawableGameComponent struct {
//	    base *GameComponent          // private, named, not embedded
//	    ...                          // the derived type's own state
//	}
//
// and never as Go embedding:
//
//	type DrawableGameComponent struct {
//	    *GameComponent               // REFUSED: BASE_MAPPING_MISMATCH
//	}
//
// Three reasons, in order of weight.
//
// Embedding is not inheritance. It promotes the base's method set wholesale,
// including members the derived class OVERRIDES or HIDES, and the promoted
// method would silently win wherever the derived one was not redeclared with
// exactly the right shape. Explicit forwarding makes every inherited member a
// declared, measured member of the derived type, so the override set is a fact
// the verifier can check rather than a property of Go's promotion rules.
//
// Embedding also publishes the base. An exported embedded field is public
// surface Microsoft never declared, and it hands a consumer a way to reach the
// base object and mutate it behind the derived type's back. There is no public
// Base, Parent or AsGameComponent member either, for the same reason: the
// pinned contract declares none, and a signature-projection requirement is the
// only thing that could justify one.
//
// And it is sufficient. Foundation 40 measured the actual CLR substitutability
// requirement of the whole profile: GameComponent, GraphicsResource and
// MathTypeConverter -- 25 of the 41 derived types between them -- are named in
// ZERO public signature positions, and no family is live. Private composition
// is not a compromise for such a family; there is no position in the contract
// for a derived value to flow through.

// mapInheritedXNABaseMembers synthesizes the XNA_INHERITED provenance class:
// the public members a derived XNA class inherits from an XNA base whose
// relationship is COMPOSED, transitively, minus the ones the derived class
// declares itself.
//
// # Why "minus the ones it declares itself"
//
// A derived class that redeclares an inherited member is overriding or hiding
// it, and the projected member is then the DERIVED one -- its own body, its own
// provenance, its own fallibility. Counting it twice would inflate the member
// accounting and would claim a forwarding that must not exist.
//
// # The exclusion is by CLR SIGNATURE, not by name
//
// It used to be by name and kind, on the recorded claim that "no derived type
// in the contract overloads an inherited name". Milestone 55 measured that
// claim and it is false, on the one composed family that already existed:
//
//	GameComponent        declares public Dispose() and protected Dispose(bool)
//	DrawableGameComponent declares          protected Dispose(bool)  -- an OVERRIDE
//
// The derived declaration is an override of the PROTECTED overload, which is
// not public surface and is never inherited. The PUBLIC Dispose() is a
// different slot that the derived class does not touch, so it is inherited --
// and a name-keyed exclusion dropped it, silently deleting a public member from
// DrawableGameComponent's projected surface. The same shape recurs on every
// GraphicsResource descendant, each of which overrides Dispose(bool) and
// inherits Dispose().
//
// The key is therefore the CLR slot identity: kind, name, and for methods the
// parameter type list. That is exactly what the runtime uses to decide whether
// a derived declaration occupies an inherited member's slot.
//
// # The overload namespace is shared with the derived type's own members
//
// Once an inherited public member can share a name with a declared one, the two
// are one overload group and the settled overload rule applies to the group as
// a whole: every member of a group of more than one carries By<ParameterShape>.
// DrawableGameComponent's pair therefore spells DisposeByNone and
// DisposeByBoolean -- the same two names GameComponent itself takes, which is
// the point: an inherited member must not be renamed by being inherited.
//
// That is why the caller resolves the inherited set first and passes ONE
// overload-group map through both mappings.
//
// # Transitivity
//
// The walk follows the base chain as far as COMPOSED relationships go, so a
// three-deep family projects the whole inherited surface rather than one level
// of it. It stops at the first base that is not an XNA class in the profile,
// which is where the BCL-inherited provenance class takes over.
func mapInheritedXNABaseMembers(
	s *expectedSurface, byIdentity map[string]*contractType, owner *expectedType,
	t contractType, inherited []inheritedXNAMember, groups map[string]int,
) []*expectedMember {
	if len(inherited) == 0 {
		return nil
	}
	// The inherited members form the derived type's own overload namespace
	// together with what it declares, so they are mapped through a synthetic
	// type carrying both -- otherwise an inherited overload group would be
	// renamed differently from the way the base named it.
	synthetic := contractType{Name: t.Name, Kind: t.Kind, Sealed: t.Sealed, BaseType: t.BaseType}
	synthetic.Members = append(synthetic.Members, t.Members...)
	for _, entry := range inherited {
		synthetic.Members = append(synthetic.Members, entry.Member)
	}
	var mapped []*expectedMember
	for _, entry := range inherited {
		declaring := synthetic
		if base, inProfile := byIdentity[entry.Base]; inProfile {
			declaring = *base
		}
		for _, member := range mapMember(s, byIdentity, owner, synthetic, declaring, entry.Member, groups) {
			member.XNABase = entry.Base
			member.XNABaseMember = entry.Member.Name
			member.XNA = entry.Base + "::" + strings.TrimPrefix(member.XNA, t.Name+"::")
			mapped = append(mapped, member)
		}
	}
	return mapped
}

// inheritedXNAMember is one public CLR member a derived class inherits from a
// COMPOSED XNA base, paired with the base identity that declares it.
type inheritedXNAMember struct {
	Base   string
	Member contractMember
}

// clrMemberSignature is the CLR slot identity of one contract member: the tuple
// the runtime uses to decide whether a derived declaration occupies an
// inherited member's slot rather than merely sharing its name.
//
// Methods and constructors carry their parameter type list, with ref and out
// marked, because two methods of one name and different parameters are two
// slots. Properties, events and fields cannot be overloaded in this profile --
// no type in the pinned contract declares two of one kind with one name -- so
// kind and name are their whole identity, and inheritedXNAPublicMembers asserts
// that rather than assuming it.
func clrMemberSignature(m contractMember) string {
	if m.Kind != "method" && m.Kind != "constructor" {
		return m.Kind + "|" + m.Name
	}
	shapes := make([]string, 0, len(m.Parameters))
	for _, p := range m.Parameters {
		shape := p.Type
		if p.Ref || p.Out {
			shape += "&"
		}
		shapes = append(shapes, shape)
	}
	return fmt.Sprintf("%s|%s|%d(%s)", m.Kind, m.Name, len(m.GenericParameters), strings.Join(shapes, ","))
}

// inheritedXNAPublicMembers is the public CLR members a derived class inherits
// from its COMPOSED XNA base chain, transitively, minus every slot the derived
// class -- or a nearer base -- already occupies.
func inheritedXNAPublicMembers(byIdentity map[string]*contractType, t contractType) []inheritedXNAMember {
	occupied := make(map[string]bool, len(t.Members))
	for _, m := range t.Members {
		occupied[clrMemberSignature(m)] = true
	}
	var inherited []inheritedXNAMember
	current := t
	seen := map[string]bool{t.Name: true}
	for {
		if current.BaseType == nil {
			return inherited
		}
		identity := baseIdentityWithoutArguments(*current.BaseType)
		relationship, isXNABase := xnaBaseRelationships[identity]
		if !isXNABase || relationship.Status != "COMPOSED" {
			return inherited
		}
		base, inProfile := byIdentity[identity]
		if !inProfile || seen[identity] {
			return inherited
		}
		seen[identity] = true
		for _, m := range base.Members {
			if m.Kind == "constructor" {
				// Constructors are not inherited. CLR requires a derived class
				// to declare its own, and every derived class in the profile
				// does.
				continue
			}
			if !inheritedMemberIsPublic(m) {
				continue
			}
			if inheritedMemberProjectsToPackageFunction(m) {
				continue
			}
			signature := clrMemberSignature(m)
			if occupied[signature] {
				continue
			}
			occupied[signature] = true
			inherited = append(inherited, inheritedXNAMember{Base: identity, Member: m})
		}
		current = *base
	}
}

// inheritedMemberProjectsToPackageFunction reports whether an inherited member's
// Go projection is a PACKAGE-LEVEL FUNCTION rather than a method on the derived
// type -- and is therefore already reachable from a derived value without being
// projected a second time.
//
// Two shapes qualify, and both name their DECLARING type in the Go identity:
//
//	a static method       Texture2DFromStreamByGraphicsDeviceAndStream
//	a generic instance    Texture2DSetDataBySliceOfT(Texture2DReference, []T)
//
// In CLR, `RenderTarget2D.FromStream(...)` and `Texture2D.FromStream(...)` are
// the SAME member reached through two names, and the settled static rule spells
// a static by its declaring type -- so there is no second Go identity to
// project. `renderTarget.SetData(...)` is likewise the same member, and the
// generic-method rule already turned it into a function whose first parameter
// is the receiver; the substitutable-base rule widens that parameter, so one
// function serves every derived value.
//
// Projecting them again would be duplication with a different name for the same
// member, which the overload and static rules exist to prevent.
func inheritedMemberProjectsToPackageFunction(m contractMember) bool {
	if m.Kind != "method" {
		return false
	}
	return m.Static || len(m.GenericParameters) > 0
}

// xnaOverriddenInheritedCount is how many PUBLIC members of a COMPOSED XNA base
// chain the derived class occupies with a declaration of its own. It is the
// other half of inheritedXNAPublicMembers and is measured so an exclusion is a
// reported number rather than a silent subtraction: a rule that drops a member
// it should have projected shows up here as a count that moved.
func xnaOverriddenInheritedCount(byIdentity map[string]*contractType, t contractType) int {
	declared := make(map[string]bool, len(t.Members))
	for _, m := range t.Members {
		declared[clrMemberSignature(m)] = true
	}
	overridden := 0
	current := t
	seen := map[string]bool{t.Name: true}
	counted := make(map[string]bool)
	for {
		if current.BaseType == nil {
			return overridden
		}
		identity := baseIdentityWithoutArguments(*current.BaseType)
		relationship, isXNABase := xnaBaseRelationships[identity]
		if !isXNABase || relationship.Status != "COMPOSED" {
			return overridden
		}
		base, inProfile := byIdentity[identity]
		if !inProfile || seen[identity] {
			return overridden
		}
		seen[identity] = true
		for _, m := range base.Members {
			if m.Kind == "constructor" || !inheritedMemberIsPublic(m) {
				continue
			}
			signature := clrMemberSignature(m)
			if declared[signature] && !counted[signature] {
				counted[signature] = true
				overridden++
			}
		}
		current = *base
	}
}

// overloadGroupsWithXNAInherited is overloadGroups over the derived type's
// EFFECTIVE method set: what it declares plus what it inherits from a COMPOSED
// XNA base. A derived class that declares one overload of an inherited name
// does not thereby own the name, and both spellings must agree about the group
// size or the inherited member and the declared one would be named by two
// different rules.
func overloadGroupsWithXNAInherited(t contractType, inherited []inheritedXNAMember) map[string]int {
	groups := overloadGroups(t)
	for _, entry := range inherited {
		if entry.Member.Kind != "method" {
			continue
		}
		groups[fmt.Sprintf("%t|%s", entry.Member.Static, entry.Member.Name)]++
	}
	return groups
}

// inheritedMemberIsPublic is the same accessibility test publicCLRMemberCount
// applies: a property counts when either accessor is public, an event always
// counts, and everything else needs public access. A protected member is
// inherited in CLR but is not public surface, so it is not projected onto the
// derived type.
func inheritedMemberIsPublic(m contractMember) bool {
	switch m.Kind {
	case "property":
		return valueOrEmpty(m.GetAccess) == "public" || valueOrEmpty(m.SetAccess) == "public"
	case "event":
		return true
	default:
		return m.Access == "public"
	}
}

// ---------------------------------------------------------------------------
// Foundation 75 — cross-package interface carriers.
// ---------------------------------------------------------------------------

// crossPackageInterfaceCarrier records one CLR interface a class DECLARES whose
// Go projection the class itself cannot satisfy, because the settled
// cross-package cycle rule already relocated one of the interface's members to
// a descendant package.
//
// Go has no way out of this. An interface method's signature is part of the
// interface, so an interface declared in the descendant package can name a
// descendant type in a method the ancestor class would have to implement -- and
// the ancestor package cannot name that type, because the descendant package
// imports it. The relocation rule handles a CLASS member by moving it; an
// INTERFACE member has nowhere to move to.
//
// The conformance is therefore carried by a named adapter in the interface's
// own package, which wraps the class. That is not a workaround invented here:
// it is what the projection already does, and it is what a consumer resolving
// the service out of the game's service container actually receives. The
// registry exists so the claim is MEASURED -- the verifier checks that the
// named carrier really does satisfy the interface -- rather than the class's
// failure being silently tolerated.
type crossPackageInterfaceCarrier struct {
	// Owner is the XNA class that declares the interface.
	Owner string
	// CLRInterface is the interface identity.
	CLRInterface string
	// GoCarrier is the Go type in the interface's package that satisfies it. It
	// is unexported, because nothing outside the module needs to name it.
	GoCarrier string
	// Blocker is the exact interface member the class cannot declare.
	Blocker string
	// Rationale is why the carrier is the honest answer rather than a hole.
	Rationale string
}

// crossPackageInterfaceCarriers is the closed list. A class that fails its
// declared interface without an entry here is INTERFACE_MAPPING_MISMATCH, so
// the escape hatch cannot be used by accident.
var crossPackageInterfaceCarriers = []crossPackageInterfaceCarrier{
	{
		Owner:        "Microsoft.Xna.Framework.GraphicsDeviceManager",
		CLRInterface: "Microsoft.Xna.Framework.Graphics.IGraphicsDeviceService",
		GoCarrier:    "managerDeviceService",
		Blocker:      "Microsoft.Xna.Framework.Graphics.IGraphicsDeviceService::GraphicsDevice",
		Rationale: "the interface's one non-event member returns Microsoft.Xna.Framework.Graphics.GraphicsDevice, " +
			"and GraphicsDeviceManager is declared in Microsoft.Xna.Framework, which the Graphics package imports. " +
			"The cross-package rule relocates such a member of a CLASS to the descendant package; an INTERFACE member " +
			"has nowhere to go, so the Graphics package carries the conformance in managerDeviceService, which is the " +
			"object the manager actually publishes into the game's service container and therefore the object a " +
			"consumer of IGraphicsDeviceService really holds",
	},
}

// crossPackageInterfaceCarrierFor finds the registered carrier for one
// class/interface pair.
func crossPackageInterfaceCarrierFor(owner, clrInterface string) (crossPackageInterfaceCarrier, bool) {
	for _, entry := range crossPackageInterfaceCarriers {
		if entry.Owner == owner && entry.CLRInterface == clrInterface {
			return entry, true
		}
	}
	return crossPackageInterfaceCarrier{}, false
}

// ---------------------------------------------------------------------------
// Foundation 78 — declared members blocked by an external BCL closure.
// ---------------------------------------------------------------------------

// blockedDeclaredMember records one member a pinned XNA type DECLARES that
// CNA-Go does not project, with the exact external closure that blocks it.
//
// This is the declared-member counterpart of a base adapter's
// BCL_PROJECTION_BLOCKED_EXTERNAL exclusion, and it carries the same weight: it
// is an admission that something the contract declares is absent, not a
// permission to have gaps. Three things make it measured rather than an
// allowlist:
//
//   - the identity must name a member the pinned contract really declares, or
//     the entry is a defect;
//   - the entry must state its kind, what it needs, and why;
//   - the count is reported as BLOCKED_DECLARED_MEMBERS, separately from
//     MISSING_MEMBER, so it can never be mistaken for surface that is present.
//
// It is deliberately NOT a general escape. Every entry below is one protected
// serialization constructor, and every one of them needs the same closure.
type blockedDeclaredMember struct {
	Kind   string
	Needs  string
	Reason string
}

// blockedDeclaredMembers is the closed registry, keyed by the identity
// memberIdentity produces.
var blockedDeclaredMembers = map[string]blockedDeclaredMember{
	// Foundation 78. Two of the eight XNA exception types declare a protected
	// deserialization constructor of their own. It is the same closure that
	// blocks Exception::GetObjectData and Dictionary`2::GetObjectData, measured
	// in Foundation 74 at 238 public BCL members across System.Decimal,
	// System.DateTime and System.Runtime.Serialization -- types the XNA 4.0
	// Windows profile names nowhere else.
	//
	// Neither constructor is reachable from CNA-Go for a second reason: the
	// only caller of a deserialization constructor is a CLR formatter, and
	// CNA-Go deserialises nothing. Projecting one would hand a consumer a
	// constructor whose two arguments they could not build.
	"Microsoft.Xna.Framework.Content.ContentLoadException::.ctor(System.Runtime.Serialization.SerializationInfo,System.Runtime.Serialization.StreamingContext)": {
		Kind:   "BCL_PROJECTION_BLOCKED_EXTERNAL",
		Needs:  "System.Runtime.Serialization.SerializationInfo and StreamingContext, whose own inventory reaches System.Decimal and System.DateTime",
		Reason: "a protected deserialization constructor whose two parameter types are the serialization closure Foundation 74 measured; its only CLR caller is a formatter, and CNA-Go deserialises nothing",
	},
	"Microsoft.Xna.Framework.Storage.StorageDeviceNotConnectedException::.ctor(System.Runtime.Serialization.SerializationInfo,System.Runtime.Serialization.StreamingContext)": {
		Kind:   "BCL_PROJECTION_BLOCKED_EXTERNAL",
		Needs:  "System.Runtime.Serialization.SerializationInfo and StreamingContext",
		Reason: "the same protected deserialization constructor on the other type that declares one",
	},
}
