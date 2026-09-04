package touch

import (
	"errors"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// resetTouchPanel returns the static state to what the reference's .cctor
// leaves behind, so one test cannot see another's assignment. The reference
// has no such reset -- these are process-lifetime statics there -- but Go test
// functions share a process and the fields are package variables.
func resetTouchPanel() {
	touchPanelEnabledGestures = GestureTypeNone
	touchPanelGesturesEnabled = false
	touchPanelWindowHandle = 0
	touchPanelDisplayOrientation = framework.DisplayOrientationDefault
	touchPanelDisplayWidth = 0
	touchPanelDisplayHeight = 0
	touchPanelDisplaySettingsChanged = false
	touchPanelState = TouchCollection{}
}

// TestTouchPanelCapabilitiesAreAlwaysEmpty pins the finding that shapes the
// whole family: GetCaps() is `initobj` and `ret`, so no machine reports a
// connected panel.
func TestTouchPanelCapabilitiesAreAlwaysEmpty(t *testing.T) {
	resetTouchPanel()
	capabilities := TouchPanelGetCapabilities()
	if capabilities.IsConnected() {
		t.Fatal("GetCapabilities reported a connected panel; the reference returns a zeroed struct on every machine")
	}
	if got := capabilities.MaximumTouchCount(); got != 0 {
		t.Fatalf("MaximumTouchCount = %d, want 0", got)
	}
	// Calling it twice cannot change the answer: there is no state behind it.
	if second := TouchPanelGetCapabilities(); second != capabilities {
		t.Fatal("two calls disagreed on a value the reference computes from nothing")
	}
}

// TestTouchPanelStateIsAlwaysEmptyAndConnected pins GetState's two halves. The
// count comes from a state the reference zeroes and never writes; the
// connected flag is the literal `true` it passes to Update.
func TestTouchPanelStateIsAlwaysEmptyAndConnected(t *testing.T) {
	resetTouchPanel()
	for attempt := 0; attempt < 3; attempt++ {
		state := TouchPanelGetState()
		if got := state.Count(); got != 0 {
			t.Fatalf("attempt %d: Count = %d, want 0", attempt, got)
		}
		if !state.IsConnected() {
			t.Fatalf("attempt %d: IsConnected was false; Update is called with the literal true", attempt)
		}
	}
}

// TestTouchPanelStateIsConnectedDisagreesWithCapabilities records the
// contradiction rather than smoothing it over: the shipped assembly reports a
// CONNECTED collection from a panel its own capabilities call says is absent.
func TestTouchPanelStateIsConnectedDisagreesWithCapabilities(t *testing.T) {
	resetTouchPanel()
	if TouchPanelGetCapabilities().IsConnected() == TouchPanelGetState().IsConnected() {
		t.Fatal("the two agreed; the reference has GetCaps return a zeroed struct while GetState passes Update the literal true")
	}
}

// TestTouchPanelReadGestureAlwaysFails covers both of ReadGesture's branches.
// The body has no `ret` instruction, so there is no third case to test.
func TestTouchPanelReadGestureAlwaysFails(t *testing.T) {
	resetTouchPanel()
	_, err := TouchPanelReadGesture()
	if !errors.Is(err, errTouchInvalidOperation) {
		t.Fatalf("ReadGesture before any assignment: err = %v, want an invalid-operation refusal", err)
	}
	if got := err.Error(); !contains(got, gesturesNotEnabled) {
		t.Fatalf("ReadGesture carried %q, want the GesturesNotEnabled message", got)
	}
	if err := SetTouchPanelEnabledGestures(GestureTypeTap); err != nil {
		t.Fatalf("SetEnabledGestures(Tap): %v", err)
	}
	_, err = TouchPanelReadGesture()
	if !errors.Is(err, errTouchInvalidOperation) {
		t.Fatalf("ReadGesture after enabling: err = %v, want an invalid-operation refusal", err)
	}
	if got := err.Error(); !contains(got, gesturesNotAvailable) {
		t.Fatalf("ReadGesture carried %q, want the GesturesNotAvailable message", got)
	}
}

// TestTouchPanelIsGestureAvailableThrowsThenAnswersFalse pins that the false is
// a constant. This is what keeps ReadGesture's unconditional throw from
// breaking the documented `while (IsGestureAvailable) ReadGesture()` idiom.
func TestTouchPanelIsGestureAvailableThrowsThenAnswersFalse(t *testing.T) {
	resetTouchPanel()
	if _, err := TouchPanelIsGestureAvailable(); !errors.Is(err, errTouchInvalidOperation) {
		t.Fatalf("before assignment: err = %v, want a refusal", err)
	}
	if err := SetTouchPanelEnabledGestures(GestureTypeFreeDrag); err != nil {
		t.Fatalf("SetEnabledGestures: %v", err)
	}
	available, err := TouchPanelIsGestureAvailable()
	if err != nil {
		t.Fatalf("after assignment: %v", err)
	}
	if available {
		t.Fatal("IsGestureAvailable was true; the reference returns ldc.i4.0")
	}
}

// TestTouchPanelEnabledGesturesMaskGuard walks the 0x3FF boundary the
// reference tests with `and 0xfffffc00`.
func TestTouchPanelEnabledGesturesMaskGuard(t *testing.T) {
	accepted := []GestureType{
		GestureTypeNone,
		GestureTypeTap,
		GestureTypePinchComplete,
		touchPanelAllGestureTypes,
		GestureTypeTap | GestureTypeFlick | GestureTypeHold,
	}
	for _, value := range accepted {
		resetTouchPanel()
		if err := SetTouchPanelEnabledGestures(value); err != nil {
			t.Fatalf("SetEnabledGestures(%d) refused an in-mask value: %v", value, err)
		}
		if got := TouchPanelEnabledGestures(); got != value {
			t.Fatalf("EnabledGestures = %d, want %d", got, value)
		}
	}
	// 0x400 is the first bit above PinchComplete and the lowest value the
	// guard rejects.
	refused := []GestureType{0x400, 0x401, GestureType(-1), GestureTypeTap | 0x800}
	for _, value := range refused {
		resetTouchPanel()
		err := SetTouchPanelEnabledGestures(value)
		if !errors.Is(err, errTouchArgument) {
			t.Fatalf("SetEnabledGestures(%d) accepted an out-of-mask value: %v", value, err)
		}
		if got := err.Error(); !contains(got, "EnabledGestures") {
			t.Fatalf("refusal carried %q, want the literal message \"EnabledGestures\"", got)
		}
		if got := TouchPanelEnabledGestures(); got != GestureTypeNone {
			t.Fatalf("a refused assignment still stored %d", got)
		}
		if touchPanelGesturesEnabled {
			t.Fatal("a refused assignment still raised the have-been-enabled flag")
		}
	}
}

// TestTouchPanelEnabledGesturesNoneStillCountsAsAssigned pins the quirk that
// decides which message ReadGesture carries: the flag is raised for every
// accepted value, GestureType.None included.
func TestTouchPanelEnabledGesturesNoneStillCountsAsAssigned(t *testing.T) {
	resetTouchPanel()
	if err := SetTouchPanelEnabledGestures(GestureTypeNone); err != nil {
		t.Fatalf("SetEnabledGestures(None): %v", err)
	}
	if _, err := TouchPanelIsGestureAvailable(); err != nil {
		t.Fatalf("IsGestureAvailable still refused after None was assigned: %v", err)
	}
}

// TestTouchPanelDisplayOrientationValidation walks ValidateOrientation's four
// accepted values and the combination it refuses despite both bits being
// declared.
func TestTouchPanelDisplayOrientationValidation(t *testing.T) {
	for _, value := range []framework.DisplayOrientation{
		framework.DisplayOrientationDefault,
		framework.DisplayOrientationLandscapeLeft,
		framework.DisplayOrientationLandscapeRight,
		framework.DisplayOrientationPortrait,
	} {
		resetTouchPanel()
		if err := SetTouchPanelDisplayOrientation(value); err != nil {
			t.Fatalf("SetDisplayOrientation(%d): %v", value, err)
		}
		if got := TouchPanelDisplayOrientation(); got != value {
			t.Fatalf("DisplayOrientation = %d, want %d", got, value)
		}
		if !touchPanelDisplaySettingsChanged {
			t.Fatal("an accepted assignment did not raise the display-settings flag")
		}
	}
	// 3 is LandscapeLeft|LandscapeRight. ValidateOrientation compares for
	// EQUALITY four times, so a combination is not a value.
	for _, value := range []framework.DisplayOrientation{3, 5, 6, 7, 8, -1} {
		resetTouchPanel()
		err := SetTouchPanelDisplayOrientation(value)
		if !errors.Is(err, errTouchArgument) {
			t.Fatalf("SetDisplayOrientation(%d) was accepted: %v", value, err)
		}
		if got := err.Error(); !contains(got, invalidDisplayOrientation) {
			t.Fatalf("refusal carried %q", got)
		}
		if touchPanelDisplaySettingsChanged {
			t.Fatal("a refused assignment raised the display-settings flag")
		}
	}
}

// TestTouchPanelUnvalidatedProperties pins that the three remaining setters
// store whatever they are handed. Their bodies are thirteen bytes: a store and
// a flag, with no comparison anywhere.
func TestTouchPanelUnvalidatedProperties(t *testing.T) {
	resetTouchPanel()
	// The two are assigned DIFFERENT values on purpose: with the same value in
	// both, a getter that read the other field would pass.
	for _, value := range []int32{0, 1, 1920, -1, -2147483648, 2147483647} {
		other := ^value
		SetTouchPanelDisplayWidth(value)
		SetTouchPanelDisplayHeight(other)
		if got := TouchPanelDisplayWidth(); got != value {
			t.Fatalf("DisplayWidth = %d, want %d", got, value)
		}
		if got := TouchPanelDisplayHeight(); got != other {
			t.Fatalf("DisplayHeight = %d, want %d", got, other)
		}
		// And once more the other way round, so neither getter can be the one
		// that happens to be right.
		SetTouchPanelDisplayWidth(other)
		SetTouchPanelDisplayHeight(value)
		if got := TouchPanelDisplayWidth(); got != other {
			t.Fatalf("DisplayWidth = %d, want %d", got, other)
		}
		if got := TouchPanelDisplayHeight(); got != value {
			t.Fatalf("DisplayHeight = %d, want %d", got, value)
		}
	}
	for _, handle := range []uintptr{0, 1, 0xdeadbeef} {
		SetTouchPanelWindowHandle(handle)
		if got := TouchPanelWindowHandle(); got != handle {
			t.Fatalf("WindowHandle = %#x, want %#x", got, handle)
		}
	}
}

// TestTouchPanelDisplaySettingsResetRunsOnce pins the one piece of GetState's
// control flow that is observable: the reset consumes the flag.
func TestTouchPanelDisplaySettingsResetRunsOnce(t *testing.T) {
	resetTouchPanel()
	SetTouchPanelDisplayWidth(800)
	if !touchPanelDisplaySettingsChanged {
		t.Fatal("the setter did not raise the flag")
	}
	TouchPanelGetState()
	if touchPanelDisplaySettingsChanged {
		t.Fatal("GetState did not lower the flag through OnDisplaySettingsChanged")
	}
	// The reset zeroes the collection and the update refills it, so the answer
	// is the same either way -- which is exactly why the flag is what the test
	// watches rather than the collection.
	if got := TouchPanelGetState().Count(); got != 0 {
		t.Fatalf("Count = %d after the reset, want 0", got)
	}
	// The width survives: OnDisplaySettingsChanged clears the collection and
	// the flag, NOT the display fields.
	if got := TouchPanelDisplayWidth(); got != 800 {
		t.Fatalf("DisplayWidth = %d after the reset, want 800", got)
	}
}

func contains(haystack, needle string) bool {
	return len(needle) <= len(haystack) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
