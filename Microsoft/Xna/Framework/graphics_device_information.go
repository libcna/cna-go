package framework

import (
	"errors"
	"fmt"

	"github.com/openeggbert/cna-go/internal/servicebridge"
)

// The exact Microsoft resource string the reference's throw site loads, read
// from the Microsoft.Xna.Framework.Resources.resources stream of the retained
// Microsoft.Xna.Framework.Game.dll
// (sha256 b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0).
const adapterCannotBeNull = "Adapter cannot be null.  Try using GraphicsAdapter.DefaultAdapter instead."

// errGraphicsDeviceInformationNil is the failure a member reports on a nil
// receiver, which is the reference's NullReferenceException.
var errGraphicsDeviceInformationNil = errors.New("graphics device information is nil")

// GraphicsDeviceInformation is the candidate device description
// GraphicsDeviceManager builds, ranks and hands to the PreparingDeviceSettings
// event before a device is created.
//
// # Three fields, and where they live
//
// The CLR class is three private fields and nothing else:
//
//	PresentationParameters presentationParameters
//	GraphicsAdapter        adapter
//	GraphicsProfile        graphicsProfile
//
// Every one of those types is declared in Microsoft.Xna.Framework.Graphics,
// which imports THIS package, so the settled cross-package cycle rule applies:
// the three properties project as Graphics-package functions named
// GraphicsDeviceInformationAdapter and friends, and the state stays here, held
// as `any` for the two references and as the raw int32 the CLR enum is.
//
// The four members below are declared on the type because the reference
// declares them here, and their LOGIC is the reference's -- which fields, in
// which order, with which short-circuit. Only the operations they cannot spell
// -- allocating a PresentationParameters, cloning one, reading its ten values,
// asking for the default adapter -- cross to the Graphics package, through
// internal/servicebridge.
type GraphicsDeviceInformation struct {
	// presentationParameters is the *graphics.PresentationParameters the
	// constructor allocates. It is `any` because this package cannot name it.
	presentationParameters any
	// adapter is the *graphics.GraphicsAdapter, likewise.
	adapter any
	// graphicsProfile is the raw int32 of the CLR GraphicsProfile enum.
	graphicsProfile int32
}

// NewGraphicsDeviceInformation projects GraphicsDeviceInformation::.ctor:
//
//	presentationParameters = new PresentationParameters();
//	adapter                = GraphicsAdapter.DefaultAdapter;
//	base..ctor();
//
// It is fallible for one measured reason: get_DefaultAdapter is not a field
// read in CNA-Go either. It enumerates CNA's adapters, so a program with no
// live native device cannot answer it, and the reference's own
// get_DefaultAdapter is equally free to throw. Reporting that failure is the
// projection of it; inventing an adapter would be the alternative.
func NewGraphicsDeviceInformation() (*GraphicsDeviceInformation, error) {
	information := &GraphicsDeviceInformation{
		presentationParameters: servicebridge.NewPresentationParameters(),
	}
	adapter, err := servicebridge.ReadDefaultAdapter()
	if err != nil {
		return nil, err
	}
	information.adapter = adapter
	return information, nil
}

// Equals is GraphicsDeviceInformation::Equals(object).
//
// The reference's order is exactly this, and every step short-circuits to
// false:
//
//	obj as GraphicsDeviceInformation, null -> false
//	other.adapter.Equals(this.adapter)         reference identity
//	other.graphicsProfile.Equals(this.graphicsProfile)
//	then NINE PresentationParameters values, compared one at a time
//
// The nine are the presentation snapshot: BackBufferWidth, BackBufferHeight,
// BackBufferFormat, DepthStencilFormat, MultiSampleCount, DisplayOrientation,
// PresentationInterval, RenderTargetUsage, DeviceWindowHandle and IsFullScreen.
// It compares the VALUES, not the parameter objects, so two informations with
// distinct but identical PresentationParameters are equal.
//
// One measured divergence. The reference reaches `other.adapter` with a
// `callvirt`, so a null adapter on the OTHER instance is a
// NullReferenceException rather than an answer -- and set_Adapter can store one,
// because of the reference bug documented on that setter. Equals is infallible
// in the contract, so CNA-Go answers false where the reference crashes. The
// alternative would be a panic from inside an equality test.
func (g *GraphicsDeviceInformation) Equals(obj any) bool {
	other, ok := obj.(*GraphicsDeviceInformation)
	if !ok || other == nil || g == nil {
		return false
	}
	if other.adapter == nil || other.adapter != g.adapter {
		return false
	}
	if other.graphicsProfile != g.graphicsProfile {
		return false
	}
	left, leftOK := servicebridge.ReadPresentationSnapshot(other.presentationParameters)
	right, rightOK := servicebridge.ReadPresentationSnapshot(g.presentationParameters)
	if !leftOK || !rightOK {
		// The reference dereferences both parameter objects; a null one is its
		// NullReferenceException. Reporting "not equal" is the infallible
		// contract's answer.
		return false
	}
	return left == right
}

// GetHashCode is GraphicsDeviceInformation::GetHashCode, which is one XOR chain
// over eleven values in this order:
//
//	graphicsProfile.GetHashCode()                 the enum's int32 value
//	^ adapter.GetHashCode()                       Object identity
//	^ BackBufferWidth ^ BackBufferHeight          Int32.GetHashCode is the value
//	^ BackBufferFormat ^ DepthStencilFormat       enum values
//	^ MultiSampleCount
//	^ DisplayOrientation ^ PresentationInterval ^ RenderTargetUsage
//	^ DeviceWindowHandle.GetHashCode()            (int)(ulong)value
//	^ IsFullScreen.GetHashCode()                  1 or 0
//
// Every contributor except the adapter is exactly reproducible: the pinned
// mscorlib's Int32.GetHashCode returns the value, Boolean.GetHashCode returns
// 1 or 0, IntPtr.GetHashCode is `(int)(ulong)m_value`, and Enum.GetHashCode is
// the underlying Int32's.
//
// The adapter's is NOT reproducible and cannot be: GraphicsAdapter declares
// neither Equals nor GetHashCode, so the reference's contributor is
// System.Object's identity hash, which the CLR derives from a sync-block index
// that is unspecified and differs between runs of the reference itself. CNA-Go
// contributes a per-object identity hash of its own with the same properties --
// stable for the object's lifetime, distinct between objects. A caller who
// compares a hash code against a number the reference produced is comparing two
// unspecified values, which is true of the reference against itself.
func (g *GraphicsDeviceInformation) GetHashCode() int32 {
	if g == nil {
		return 0
	}
	hash := g.graphicsProfile ^ referenceIdentityHashCode(g.adapter)
	snapshot, ok := servicebridge.ReadPresentationSnapshot(g.presentationParameters)
	if !ok {
		return hash
	}
	hash ^= snapshot.BackBufferWidth
	hash ^= snapshot.BackBufferHeight
	hash ^= snapshot.BackBufferFormat
	hash ^= snapshot.DepthStencilFormat
	hash ^= snapshot.MultiSampleCount
	hash ^= snapshot.DisplayOrientation
	hash ^= snapshot.PresentationInterval
	hash ^= snapshot.RenderTargetUsage
	hash ^= int32(uint32(snapshot.DeviceWindowHandle))
	if snapshot.IsFullScreen {
		hash ^= 1
	}
	return hash
}

// Clone is GraphicsDeviceInformation::Clone:
//
//	GraphicsDeviceInformation copy = new GraphicsDeviceInformation();
//	copy.presentationParameters = this.presentationParameters.Clone();
//	copy.adapter                = this.adapter;
//	copy.graphicsProfile        = this.graphicsProfile;
//	return copy;
//
// The asymmetry is the reference's and is load-bearing: the parameters are
// DEEP-copied, so the clone can be reconfigured without disturbing the source,
// while the adapter is ALIASED, so both informations name the same device.
//
// It is fallible because it really does call the constructor, and the
// constructor really does ask for the default adapter -- whose value it then
// throws away. That is what the reference does, and a Clone that skipped the
// call would be a different method that happened to produce the same fields.
func (g *GraphicsDeviceInformation) Clone() (*GraphicsDeviceInformation, error) {
	if g == nil {
		return nil, errGraphicsDeviceInformationNil
	}
	clone, err := NewGraphicsDeviceInformation()
	if err != nil {
		return nil, err
	}
	if cloned, ok := servicebridge.ClonePresentationParameters(g.presentationParameters); ok {
		clone.presentationParameters = cloned
	} else {
		clone.presentationParameters = nil
	}
	clone.adapter = g.adapter
	clone.graphicsProfile = g.graphicsProfile
	return clone, nil
}

// ---------------------------------------------------------------------------
// The unexported halves of the three cross-package properties.
//
// The Graphics package reaches these through internal/servicebridge; nothing
// outside this module can, because they are unexported and the bridge takes
// `any`.
// ---------------------------------------------------------------------------

func (g *GraphicsDeviceInformation) readAdapter() any {
	if g == nil {
		return nil
	}
	return g.adapter
}

// writeAdapter is set_Adapter, and it reproduces a REFERENCE BUG rather than
// correcting it:
//
//	IL_0000: ldarg.0
//	IL_0001: ldfld  adapter          // THIS.adapter, not `value`
//	IL_0006: brtrue.s IL_0018
//	IL_0008: throw ArgumentNullException("value", NoNullUseDefaultAdapter)
//	IL_0018: this.adapter = value
//
// The guard tests the field the setter is about to overwrite, not the argument
// it was given. So assigning null SUCCEEDS whenever the current adapter is
// non-null -- which it always is after the constructor -- and the message about
// using GraphicsAdapter.DefaultAdapter can only ever be raised on an
// information whose adapter is ALREADY null. Correcting it here would make
// CNA-Go refuse an assignment the reference accepts, which is a different API.
func (g *GraphicsDeviceInformation) writeAdapter(value any) error {
	if g == nil {
		return errGraphicsDeviceInformationNil
	}
	if g.adapter == nil {
		return fmt.Errorf("%s: %s", "value", adapterCannotBeNull)
	}
	g.adapter = value
	return nil
}

func (g *GraphicsDeviceInformation) readGraphicsProfile() int32 {
	if g == nil {
		return 0
	}
	return g.graphicsProfile
}

func (g *GraphicsDeviceInformation) writeGraphicsProfile(value int32) {
	if g == nil {
		return
	}
	g.graphicsProfile = value
}

func (g *GraphicsDeviceInformation) readPresentationParameters() any {
	if g == nil {
		return nil
	}
	return g.presentationParameters
}

func (g *GraphicsDeviceInformation) writePresentationParameters(value any) {
	if g == nil {
		return
	}
	g.presentationParameters = value
}

// init installs this package's half of the GraphicsDeviceInformation field
// accessors, so the Graphics package's three projected properties can reach
// them. Nothing is retained: each hook is a field read or write on the object
// it is handed.
func init() {
	servicebridge.SetDeviceInformationAccessors(
		func(information any) (any, bool) {
			typed, ok := information.(*GraphicsDeviceInformation)
			if !ok || typed == nil {
				return nil, false
			}
			return typed.readAdapter(), true
		},
		func(information any, value any) error {
			typed, ok := information.(*GraphicsDeviceInformation)
			if !ok || typed == nil {
				return errGraphicsDeviceInformationNil
			}
			return typed.writeAdapter(value)
		},
		func(information any) (any, bool) {
			typed, ok := information.(*GraphicsDeviceInformation)
			if !ok || typed == nil {
				return nil, false
			}
			return typed.readPresentationParameters(), true
		},
		func(information any, value any) error {
			typed, ok := information.(*GraphicsDeviceInformation)
			if !ok || typed == nil {
				return errGraphicsDeviceInformationNil
			}
			typed.writePresentationParameters(value)
			return nil
		},
		func(information any) int32 {
			typed, _ := information.(*GraphicsDeviceInformation)
			return typed.readGraphicsProfile()
		},
		func(information any, value int32) {
			typed, _ := information.(*GraphicsDeviceInformation)
			typed.writeGraphicsProfile(value)
		},
	)
}
