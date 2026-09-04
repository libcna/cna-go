package touch

import (
	"errors"
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// TouchPanel is Microsoft.Xna.Framework.Input.Touch.TouchPanel:
//
//	.class public abstract auto ansi sealed beforefieldinit TouchPanel
//
// A static class, so the identity is an empty struct and every member is a
// type-prefixed package function.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Input.Touch.dll   b0585224c18022c3...
//
// # The whole touch subsystem is a STUB on the Windows runtime
//
// This is the finding that shapes the entire type, and it is measured rather
// than assumed: the pinned Microsoft.Xna.Framework.Input.Touch.dll contains NO
// p/invoke declaration at all. Not one. Every member is managed code over
// managed static fields, and three of them are hard-coded to fail or to answer
// nothing:
//
//	GetCapabilities()      -> `initobj` a local and return it: IsConnected
//	                          false, MaximumTouchCount 0, unconditionally.
//	ReadGesture()          -> throws on EVERY path. Its 29-byte body has two
//	                          branches and both end in `throw`.
//	get_IsGestureAvailable -> throws when gestures were never enabled, and
//	                          otherwise returns `ldc.i4.0` -- constant false.
//	GetState()             -> updates the static collection from a local that
//	                          was just zeroed, so the result is always EMPTY.
//
// XNA 4.0's touch surface shipped for Windows Phone; the Windows build carries
// the same public API with the device half removed. The type is not
// unimplemented in this projection -- it is faithfully implemented, and what it
// faithfully does is nothing.
//
// # Why CNA's fourteen working touch routes are deliberately NOT bound
//
// CNA implements this family for real: it has cna_touch_get_capabilities,
// cna_touch_get_state, cna_touch_panel_read_gesture and get/set routes for all
// five properties, and they answer from an actual digitizer. Binding them was
// tried, measured and REVERTED.
//
// The reason is the standing authority rule. The pinned XNA IL is the contract
// and behaviour authority; CNA is evidence of what the native layer can do,
// never evidence of what XNA does. A consumer that polls TouchPanel.GetState in
// its update loop gets an empty collection from the reference on every machine,
// so a projection that returned real touches would be answering a question the
// reference answers differently -- and a game built against it would behave one
// way here and another way on the runtime this binding exists to match.
//
// The routes are recorded in the native census under CONTRACT_DIVERGENCE, which
// is the same class the audio sample-conversion routes were given in milestone
// 87 for the same reason.
//
// # What IS real here
//
// Five of the six properties are ordinary managed static state, and they
// round-trip: a consumer assigns DisplayWidth and reads it back. Two of them
// validate, and the validation is the only behaviour in the type worth
// planting a defect against:
//
//   - set_EnabledGestures rejects any bit outside the 0x3FF mask.
//   - set_DisplayOrientation accepts only Default, LandscapeLeft,
//     LandscapeRight and Portrait -- note 4, not 3, so the combined values are
//     invalid.
//
// # The static fields are unsynchronized, exactly as the reference leaves them
//
// The reference declares plain `static` fields and touches them without a lock.
// (Its private Touch class does hold a SyncObject, but no accessor ever enters
// it.) These package-level variables carry the same property: concurrent
// assignment from two goroutines is a data race here because it is one there.
type TouchPanel struct{}

// The static fields the reference declares, one Go package variable each.
var (
	// touchPanelEnabledGestures is `_enabledGestures`.
	touchPanelEnabledGestures GestureType
	// touchPanelGesturesEnabled is `_haveGestureBeenEnabled`, which
	// set_EnabledGestures raises and NOTHING ever lowers -- not even
	// assigning GestureType.None, which still counts as having assigned.
	touchPanelGesturesEnabled bool
	// touchPanelWindowHandle is Touch::_windowHandle, which TouchPanel's own
	// WindowHandle property forwards to in both directions.
	touchPanelWindowHandle uintptr
	// touchPanelDisplayOrientation is `displayOrientation`.
	touchPanelDisplayOrientation framework.DisplayOrientation
	// touchPanelDisplayWidth is `displayWidth`.
	touchPanelDisplayWidth int32
	// touchPanelDisplayHeight is `displayHeight`.
	touchPanelDisplayHeight int32
	// touchPanelDisplaySettingsChanged is `displaySettingsChanged`, raised by
	// all three display setters and cleared by the reset GetState performs.
	touchPanelDisplaySettingsChanged bool
	// touchPanelState is `touchState`, the static collection GetState updates
	// and returns a copy of. The .cctor `initobj`s it, so it starts empty and
	// DISCONNECTED -- which is observable only if GetState is never called.
	touchPanelState TouchCollection
)

// The two CLR exceptions TouchPanel throws that the rest of this package did
// not already need. They join the TouchCollection sentinels above and are
// unexported for the same reason: the XNA contract declares no error type here.
var (
	// errTouchArgument projects System.ArgumentException.
	errTouchArgument = errors.New("touch argument is invalid")
	// errTouchInvalidOperation projects System.InvalidOperationException.
	errTouchInvalidOperation = errors.New("touch operation is invalid")
)

// touchPanelAllGestureTypes is the `AllGestureTypes` literal the reference
// declares as `int32(0x000003FF)` -- every one of the ten GestureType bits.
const touchPanelAllGestureTypes = GestureType(0x3FF)

// The three FrameworkResources messages this type throws, verified byte for
// byte against the retained assembly. GesturesNotAvailable really does carry
// two spaces after its first sentence.
const (
	gesturesNotEnabled   = "This operation cannot be completed until TouchPanel.EnabledGestures is assigned."
	gesturesNotAvailable = "No gestures are available at this time.  TouchPanel.ReadGesture should only be called when TouchPanel.IsGestureAvailable is true."
	// invalidDisplayOrientation is thrown by Helpers::ValidateOrientation,
	// which set_DisplayOrientation calls before storing.
	invalidDisplayOrientation = "The specified DisplayOrientation is invalid."
)

// TouchPanelGetCapabilities is TouchPanel::GetCapabilities(), a six-byte body
// forwarding to TouchPanelCapabilities::GetCaps(), which returns the zeroed
// struct. It cannot fail and it cannot report a connected panel.
func TouchPanelGetCapabilities() TouchPanelCapabilities {
	return TouchPanelCapabilities{}
}

// TouchPanelGetState is TouchPanel::GetState().
//
// The reference's body reads:
//
//	XNAINPUT_TOUCH_LOCATION_STATE newState = default;   // zeroed
//	if (!nointerop) {
//	    try {
//	        if (displaySettingsChanged) OnDisplaySettingsChanged();
//	        touchState.Update(ref prevState, ref newState, true);
//	        prevState = newState;
//	    } catch (object) { nointerop = true; touchState.Update(..., false); }
//	}
//	return touchState;
//
// `newState` is zeroed and then never written, so its Count is zero and the
// collection is emptied on every call. The `nointerop` latch and its catch
// handler are unreachable, because the try block contains no call that can
// throw -- Update over an empty state does not. So the observable answer is
// invariant: an EMPTY collection whose IsConnected is true.
//
// The projection keeps the reset and the update rather than returning a
// constant, because the display-settings reset is observable through the flag
// and because the shape is what makes the invariance checkable.
func TouchPanelGetState() TouchCollection {
	if touchPanelDisplaySettingsChanged {
		touchPanelOnDisplaySettingsChanged()
	}
	// Update from a state that carries no touches. The reference passes
	// `true` for connected on this path.
	collection, err := NewTouchCollection([]TouchLocation{})
	if err != nil {
		// Unreachable: the argument is neither nil nor over the eight-slot
		// bound, which are the constructor's only two refusals.
		return touchPanelState
	}
	touchPanelState = collection
	return touchPanelState
}

// touchPanelOnDisplaySettingsChanged is OnDisplaySettingsChanged(): it zeroes
// the collection and lowers the flag.
//
// The reference also zeroes its private `prevState`, which this projection does
// not carry: `prevState` exists there only to be handed to TouchCollection's
// internal Update, and since the new state is always empty the previous one can
// never differ from it. A field nothing can observe is not projected.
func touchPanelOnDisplaySettingsChanged() {
	touchPanelState = TouchCollection{}
	touchPanelDisplaySettingsChanged = false
}

// TouchPanelReadGesture is TouchPanel::ReadGesture(), whose 29-byte body ends
// in `throw` on BOTH of its branches:
//
//	if (!_haveGestureBeenEnabled)
//	    throw new InvalidOperationException(GesturesNotEnabled);
//	throw new InvalidOperationException(GesturesNotAvailable);
//
// There is no third path and no return instruction. Which message a consumer
// gets is the only thing that varies, and it turns on whether EnabledGestures
// was ever assigned.
func TouchPanelReadGesture() (GestureSample, error) {
	if !touchPanelGesturesEnabled {
		return GestureSample{}, fmt.Errorf("%w: %s", errTouchInvalidOperation, gesturesNotEnabled)
	}
	return GestureSample{}, fmt.Errorf("%w: %s", errTouchInvalidOperation, gesturesNotAvailable)
}

// TouchPanelIsGestureAvailable is get_IsGestureAvailable:
//
//	if (!_haveGestureBeenEnabled)
//	    throw new InvalidOperationException(GesturesNotEnabled);
//	return false;                                       // ldc.i4.0
//
// The false is a CONSTANT, not a queue depth. So the documented idiom --
// `while (TouchPanel.IsGestureAvailable) ReadGesture();` -- never enters its
// body on this runtime, which is what keeps ReadGesture's unconditional throw
// from breaking a correctly written consumer.
func TouchPanelIsGestureAvailable() (bool, error) {
	if !touchPanelGesturesEnabled {
		return false, fmt.Errorf("%w: %s", errTouchInvalidOperation, gesturesNotEnabled)
	}
	return false, nil
}

// TouchPanelEnabledGestures is get_EnabledGestures, a plain field read.
func TouchPanelEnabledGestures() GestureType {
	return touchPanelEnabledGestures
}

// SetTouchPanelEnabledGestures is set_EnabledGestures:
//
//	if ((value & 0xfffffc00) != 0)
//	    throw new ArgumentException("EnabledGestures");
//	_enabledGestures = value;
//	_haveGestureBeenEnabled = true;
//
// Two things are worth naming. The mask is the COMPLEMENT of AllGestureTypes,
// so any bit above PinchComplete is refused while any combination of the ten
// declared bits is accepted. And the message is the literal string
// "EnabledGestures" -- the reference calls ArgumentException(string), whose
// single parameter is the MESSAGE, so what looks like a parameter name is
// reported as the message text. That is reproduced rather than corrected.
//
// The flag is raised AFTER the guard but for every accepted value, GestureType
// None included: assigning None still counts as having assigned, which is what
// turns ReadGesture's first message into its second.
func SetTouchPanelEnabledGestures(value GestureType) error {
	if value&^touchPanelAllGestureTypes != 0 {
		return fmt.Errorf("%w: %s", errTouchArgument, "EnabledGestures")
	}
	touchPanelEnabledGestures = value
	touchPanelGesturesEnabled = true
	return nil
}

// TouchPanelWindowHandle is get_WindowHandle, forwarding to the private
// Touch class's static field. System.IntPtr projects to uintptr, which the
// mapping already declares.
func TouchPanelWindowHandle() uintptr {
	return touchPanelWindowHandle
}

// SetTouchPanelWindowHandle is set_WindowHandle. It stores and nothing more --
// no hook is installed and no window is validated, because the assembly has no
// native side to install one into.
func SetTouchPanelWindowHandle(value uintptr) {
	touchPanelWindowHandle = value
}

// TouchPanelDisplayOrientation is get_DisplayOrientation, a plain field read.
func TouchPanelDisplayOrientation() framework.DisplayOrientation {
	return touchPanelDisplayOrientation
}

// SetTouchPanelDisplayOrientation is set_DisplayOrientation, which validates
// through Helpers::ValidateOrientation before storing and then raises the
// display-settings flag.
//
// ValidateOrientation accepts exactly four values -- 0, 1, 2 and 4 -- with an
// equality test each, NOT a mask test. So `LandscapeLeft | LandscapeRight` (3)
// is refused even though both of its bits are declared, and Portrait is 4
// rather than 3.
func SetTouchPanelDisplayOrientation(value framework.DisplayOrientation) error {
	switch value {
	case framework.DisplayOrientationDefault,
		framework.DisplayOrientationLandscapeLeft,
		framework.DisplayOrientationLandscapeRight,
		framework.DisplayOrientationPortrait:
	default:
		return fmt.Errorf("%w: %s", errTouchArgument, invalidDisplayOrientation)
	}
	touchPanelDisplayOrientation = value
	touchPanelDisplaySettingsChanged = true
	return nil
}

// TouchPanelDisplayWidth is get_DisplayWidth, a plain field read.
func TouchPanelDisplayWidth() int32 { return touchPanelDisplayWidth }

// SetTouchPanelDisplayWidth is set_DisplayWidth. Its thirteen-byte body stores
// and raises the flag -- there is NO validation, so zero and negative widths
// are accepted and read back unchanged.
func SetTouchPanelDisplayWidth(value int32) {
	touchPanelDisplayWidth = value
	touchPanelDisplaySettingsChanged = true
}

// TouchPanelDisplayHeight is get_DisplayHeight, a plain field read.
func TouchPanelDisplayHeight() int32 { return touchPanelDisplayHeight }

// SetTouchPanelDisplayHeight is set_DisplayHeight, unvalidated like its
// sibling.
func SetTouchPanelDisplayHeight(value int32) {
	touchPanelDisplayHeight = value
	touchPanelDisplaySettingsChanged = true
}
