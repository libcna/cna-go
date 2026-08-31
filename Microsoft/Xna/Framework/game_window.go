package framework

import (
	"errors"

	"github.com/openeggbert/cna-go/internal/interop"
)

// errNoNativeWindow projects System.NullReferenceException, which is what the
// reference's three UNGUARDED window members produce when the platform window
// does not exist: WindowsGameWindow::get_ClientBounds,
// ::BeginScreenDeviceChange and ::EndScreenDeviceChange all dereference
// `mainForm` with no null check.
//
// It is unexported because the XNA public contract declares no error type at
// these positions, exactly as the graphics package's own
// InvalidOperationException projection is unexported. It is deliberately NOT
// shared with the guarded members: those return the reference's documented
// fallback and report no failure at all.
var errNoNativeWindow = errors.New("object reference is not set to an instance of an object")

// GameWindow is the Go projection of Microsoft.Xna.Framework.GameWindow.
//
// # Why it is a concrete Go type
//
// In the reference GameWindow is `public abstract` with an `assembly`
// constructor, and the selected Windows runtime profile has exactly one
// implementor: WindowsGameWindow, which GameHost creates. A consumer never
// constructs one and never subclasses one -- there is no public constructor to
// call and the class is only reachable through Game.Window.
//
// CNA-Go therefore projects the abstract base and its one implementor as a
// single concrete type with no exported constructor. The abstract members
// forward to CNA, which plays WindowsGameForm's part; the concrete ones are
// projected from the base's own IL, byte for byte.
//
// # The window exists from construction
//
// Game::.ctor calls EnsureHost(), which allocates WindowsGameHost and its
// window before the constructor's fifth statement, and the constructor then
// reads host.Window to subscribe its Paint handler. Game::get_Window is
// `host == null ? null : host.Window`, and WindowsGameHost::get_Window is one
// `ldfld` over a field its constructor assigns. So after construction the
// getter returns the same object forever.
//
// CNA-Go reproduces that identity exactly: NewGame allocates one GameWindow and
// Game.Window returns it. What it deliberately does NOT reproduce is a native
// window at construction time, because CNA-Go creates the native game inside
// Run. That difference is not hidden -- it is exactly the difference the
// reference itself already has a documented behaviour for.
//
// # The null-guard rule, read from the reference
//
// WindowsGameWindow's own members split cleanly in two, and CNA-Go's split is
// the same split rather than a policy of its own:
//
//	guarded    get_Handle             mainForm == null -> IntPtr.Zero
//	           get_AllowUserResizing  mainForm == null -> false
//	           set_AllowUserResizing  mainForm == null -> nothing happens
//	           SetTitle               mainForm == null -> nothing happens
//	           get_ScreenDeviceName   mainForm == null -> String.Empty
//
//	unguarded  get_ClientBounds            dereferences mainForm directly
//	           BeginScreenDeviceChange     dereferences mainForm directly
//	           EndScreenDeviceChange(3)    dereferences mainForm directly
//
// A guarded member with no live native game answers the reference's own
// fallback and reports no failure. An unguarded one throws
// NullReferenceException in the reference, and reports a failure here. Neither
// behaviour was chosen; both were read.
type GameWindow struct {
	// game is the owner. CNA models the window as a property of the game --
	// every cna_game_window_* route takes the GAME handle -- so this is not an
	// owned native lifetime and there is nothing here to dispose.
	game *Game

	// title is GameWindow::title, the abstract BASE's own managed field. The
	// assembly constructor stores String.Empty into it, which is Go's zero
	// value for the field as well.
	title string

	// inDeviceTransition is WindowsGameWindow::inDeviceTransition, which
	// BeginScreenDeviceChange sets and EndScreenDeviceChange clears in a
	// finally. No projected member reads it, exactly as no public member of
	// the reference does; it is kept because both projected bodies genuinely
	// assign it and omitting the assignment would make them incomplete.
	inDeviceTransition bool

	// The three PUBLIC events. Each is a private multicast delegate field in
	// the reference and a private registration list here.
	screenDeviceNameChanged EventSource[*EventArgs]
	clientSizeChanged       EventSource[*EventArgs]
	orientationChanged      EventSource[*EventArgs]

	// The three ASSEMBLY events. In the reference Activated, Deactivated and
	// Paint are `assembly`-visible: Game subscribes to them, nothing outside
	// Microsoft.Xna.Framework.Game.dll can. CLR `assembly` is Go's unexported
	// package scope, so they are unexported fields with no accessor pair --
	// which is why the public contract lists three events and this type has
	// six. Their protected On... raisers ARE public contract members and are
	// projected below; with nothing subscribed they raise nothing, exactly as
	// the reference does.
	activated   EventSource[*EventArgs]
	deactivated EventSource[*EventArgs]
	paint       EventSource[*EventArgs]
}

// newGameWindow is the projection of the assembly constructor:
//
//	.ctor()  base..ctor(); this.title = String.Empty
//
// It is unexported because the reference constructor is `assembly`: a consumer
// obtains the window from Game.Window and can obtain it no other way.
func newGameWindow(game *Game) *GameWindow {
	return &GameWindow{game: game}
}

// runtime is the owning Game's native runtime, or nil.
func (w *GameWindow) runtime() *interop.Runtime {
	if w == nil || w.game == nil {
		return nil
	}
	return w.game.runtime
}

// Title is GameWindow::get_Title, whose whole body is
//
//	ldarg.0; ldfld string GameWindow::title; ret
//
// It is a read of the abstract base's own managed field, so it reaches no
// window, allocates nothing and cannot fail. Binding CNA's
// cna_game_window_copy_title here would create a second source of truth that
// could disagree with the field the setter wrote, which is the same reason the
// timing getters do not bind their cna_game_get_* counterparts.
func (w *GameWindow) Title() string {
	if w == nil {
		return ""
	}
	return w.title
}

// SetTitleProperty is GameWindow::set_Title:
//
//	if (value == null)
//	    throw new ArgumentNullException("value", Resources.TitleCannotBeNull);
//	if (this.title != value) { this.title = value; this.SetTitle(this.title); }
//
// Two behaviours are load-bearing and both are preserved. An UNCHANGED value is
// suppressed: the field is not rewritten and SetTitle is not called, so a
// consumer assigning the same title in a loop makes no native calls at all. And
// SetTitle receives `this.title` -- the field just written -- rather than the
// parameter; the two are equal here, and the reference's own instruction order
// is kept anyway.
//
// The null branch is unreachable from Go: `string` is not a nullable type, so
// there is no value a caller can pass that the reference would reject. This is
// a Go language limitation, recorded rather than worked around; CNA-Go does not
// invent a pointer parameter for a position the XNA contract declares as
// System.String.
func (w *GameWindow) SetTitleProperty(value string) error {
	if w == nil {
		return nil
	}
	if w.title == value {
		return nil
	}
	w.title = value
	return w.SetTitleMethod(w.title)
}

// SetTitleMethod is the protected abstract GameWindow::SetTitle, whose Windows
// implementor is
//
//	if (mainForm != null) mainForm.Text = title;
//
// so with no window it does nothing and reports no failure. CNA's counterpart
// is cna_game_set_window_title.
//
// The Go name carries the `Method` suffix because the type also declares a
// Title property whose projected setter would otherwise collide with it; the
// two are genuinely different members in the reference and stay different here.
func (w *GameWindow) SetTitleMethod(title string) error {
	runtime := w.runtime()
	if runtime == nil {
		return nil
	}
	_, err := runtime.SetWindowTitle(title)
	return err
}

// Handle is GameWindow::get_Handle:
//
//	mainForm != null ? mainForm.Handle : IntPtr.Zero
//
// System.IntPtr projects to uintptr. The value is the opaque pointer-sized bit
// value the XNA contract carries at this position and nothing more: it must not
// be dereferenced, is not a CNA handle, and its being non-zero is not by itself
// proof that a window is on screen.
func (w *GameWindow) Handle() (uintptr, error) {
	runtime := w.runtime()
	if runtime == nil {
		return 0, nil
	}
	value, _, err := runtime.WindowHandle()
	return value, err
}

// AllowUserResizing is GameWindow::get_AllowUserResizing, whose Windows
// implementor answers false when there is no window.
func (w *GameWindow) AllowUserResizing() (bool, error) {
	runtime := w.runtime()
	if runtime == nil {
		return false, nil
	}
	value, _, err := runtime.WindowAllowUserResizing()
	return value, err
}

// SetAllowUserResizing is GameWindow::set_AllowUserResizing, whose Windows
// implementor forwards to the form when there is one and does nothing when
// there is not. It validates nothing.
func (w *GameWindow) SetAllowUserResizing(value bool) error {
	runtime := w.runtime()
	if runtime == nil {
		return nil
	}
	_, err := runtime.SetWindowAllowUserResizing(value)
	return err
}

// ClientBounds is GameWindow::get_ClientBounds. Its Windows implementor is
//
//	return this.mainForm.ClientBounds;
//
// with NO null check, so with no window the reference throws
// NullReferenceException. CNA-Go reports a failure there rather than inventing
// an empty rectangle: a Rectangle of zeros is a legitimate value a real window
// could report, and returning it would make a failure indistinguishable from a
// measurement.
func (w *GameWindow) ClientBounds() (Rectangle, error) {
	runtime := w.runtime()
	if runtime == nil {
		return Rectangle{}, errNoNativeWindow
	}
	x, y, width, height, err := runtime.WindowClientBounds()
	if err != nil {
		return Rectangle{}, err
	}
	return Rectangle{X: x, Y: y, Width: width, Height: height}, nil
}

// ScreenDeviceName is GameWindow::get_ScreenDeviceName. Its Windows
// implementor returns String.Empty when there is no form AND when the form has
// no device screen, and only otherwise asks the platform. Both empty answers
// are the same observable value, so CNA-Go's no-native-window answer is the
// reference's own.
func (w *GameWindow) ScreenDeviceName() (string, error) {
	runtime := w.runtime()
	if runtime == nil {
		return "", nil
	}
	value, _, err := runtime.WindowScreenDeviceName()
	return value, err
}

// CurrentOrientation is GameWindow::get_CurrentOrientation, whose Windows
// implementor is the whole of
//
//	ldc.i4.0
//	ret
//
// It is a CONSTANT. In the selected Windows runtime profile the reference never
// asks the platform what the orientation is; it answers
// DisplayOrientation.Default unconditionally, on every machine, in every
// window state.
//
// So CNA-Go answers the same constant and deliberately does not bind
// cna_game_window_get_current_orientation. Binding it would introduce a second
// source of truth that could disagree with the reference, and a projection that
// reported a rotated orientation where XNA reports Default would be wrong even
// if the platform value were right.
func (w *GameWindow) CurrentOrientation() DisplayOrientation {
	return DisplayOrientationDefault
}

// SetSupportedOrientations is the protected-internal abstract
// GameWindow::SetSupportedOrientations, whose Windows implementor is the whole
// of
//
//	ret
//
// It does nothing. This was the one genuinely open question about GameWindow,
// and the answer is neither a CNA gap nor a deferral: CNA has no window
// orientation route, and the reference does not need one, because in this
// profile the member is an empty body. Its only CNA-side neighbour,
// cna_graphics_device_manager_set_supported_orientations, takes a MANAGER
// handle and belongs to GraphicsDeviceManager -- whose CNA-Go projection
// already stores the value as managed state, exactly where the reference's
// caller stores it.
//
// Forwarding this member to that route would not be completing it; it would be
// inventing behaviour the reference does not have, on an object that does not
// own it. The empty body is the measured contract and is reproduced as an empty
// body.
func (w *GameWindow) SetSupportedOrientations(orientations DisplayOrientation) {
	_ = orientations
}

// BeginScreenDeviceChange is GameWindow::BeginScreenDeviceChange, whose
// Windows implementor is
//
//	this.mainForm.BeginScreenDeviceChange(willBeFullScreen);
//	this.inDeviceTransition = true;
//
// The form dereference is unguarded, so with no window the reference throws.
// The flag is set AFTER the call, so a failing call leaves it clear -- and that
// order is preserved.
func (w *GameWindow) BeginScreenDeviceChange(willBeFullScreen bool) error {
	runtime := w.runtime()
	if runtime == nil {
		return errNoNativeWindow
	}
	if err := runtime.BeginScreenDeviceChange(willBeFullScreen); err != nil {
		return err
	}
	w.inDeviceTransition = true
	return nil
}

// EndScreenDeviceChangeByStringAndInt32AndInt32 is
// GameWindow::EndScreenDeviceChange(string, int32, int32), whose Windows
// implementor is
//
//	try     { this.mainForm.EndScreenDeviceChange(name, width, height); }
//	finally { this.inDeviceTransition = false; }
//
// The finally is the point: the transition flag is cleared even when the call
// fails, which is the opposite of Begin's order and is reproduced as such.
func (w *GameWindow) EndScreenDeviceChangeByStringAndInt32AndInt32(screenDeviceName string, clientWidth, clientHeight int32) error {
	runtime := w.runtime()
	if runtime == nil {
		w.inDeviceTransition = false
		return errNoNativeWindow
	}
	err := runtime.EndScreenDeviceChange(screenDeviceName, clientWidth, clientHeight)
	w.inDeviceTransition = false
	return err
}

// EndScreenDeviceChangeByString is the concrete one-argument overload:
//
//	this.EndScreenDeviceChange(screenDeviceName,
//	                           this.ClientBounds.Width,
//	                           this.ClientBounds.Height);
//
// ClientBounds is read TWICE, through two separate callvirts, and that is
// preserved rather than hoisted into one read: the property is a live platform
// query in every implementor, so a window resized between the two reads makes
// the reference mismatch its own width and height. Collapsing it to one read
// would be a fix, and CNA-Go does not fix the reference.
//
// Because ClientBounds is the unguarded member, this overload fails with no
// live native window before it reaches the three-argument one -- exactly where
// the reference's NullReferenceException would come from.
func (w *GameWindow) EndScreenDeviceChangeByString(screenDeviceName string) error {
	width, err := w.ClientBounds()
	if err != nil {
		return err
	}
	height, err := w.ClientBounds()
	if err != nil {
		return err
	}
	return w.EndScreenDeviceChangeByStringAndInt32AndInt32(screenDeviceName, width.Width, height.Height)
}

// The six protected raise methods. Every one is the same 26-byte body:
//
//	if (this.<Event> != null) this.<Event>(this, EventArgs.Empty);
//
// The sender is `this` in all six -- unlike Game::OnExiting, which passes null.
// EventArgs.Empty is the shared singleton, projected as EventArgsEmpty().

// OnActivated raises the assembly-visible Activated event.
func (w *GameWindow) OnActivated() error {
	return w.activated.Raise(w, EventArgsEmpty())
}

// OnDeactivated raises the assembly-visible Deactivated event.
func (w *GameWindow) OnDeactivated() error {
	return w.deactivated.Raise(w, EventArgsEmpty())
}

// OnPaint raises the assembly-visible Paint event.
func (w *GameWindow) OnPaint() error {
	return w.paint.Raise(w, EventArgsEmpty())
}

// OnScreenDeviceNameChanged raises the public ScreenDeviceNameChanged event.
func (w *GameWindow) OnScreenDeviceNameChanged() error {
	return w.screenDeviceNameChanged.Raise(w, EventArgsEmpty())
}

// OnClientSizeChanged raises the public ClientSizeChanged event.
func (w *GameWindow) OnClientSizeChanged() error {
	return w.clientSizeChanged.Raise(w, EventArgsEmpty())
}

// OnOrientationChanged raises the public OrientationChanged event.
func (w *GameWindow) OnOrientationChanged() error {
	return w.orientationChanged.Raise(w, EventArgsEmpty())
}

// AddScreenDeviceNameChangedHandler registers a handler for
// GameWindow::ScreenDeviceNameChanged.
func (w *GameWindow) AddScreenDeviceNameChangedHandler(handler EventHandler[*EventArgs]) (EventSubscription, error) {
	return w.screenDeviceNameChanged.Add(handler)
}

// RemoveScreenDeviceNameChangedHandler removes the registration the token names.
func (w *GameWindow) RemoveScreenDeviceNameChangedHandler(subscription EventSubscription) error {
	return w.screenDeviceNameChanged.Remove(subscription)
}

// AddClientSizeChangedHandler registers a handler for
// GameWindow::ClientSizeChanged.
func (w *GameWindow) AddClientSizeChangedHandler(handler EventHandler[*EventArgs]) (EventSubscription, error) {
	return w.clientSizeChanged.Add(handler)
}

// RemoveClientSizeChangedHandler removes the registration the token names.
func (w *GameWindow) RemoveClientSizeChangedHandler(subscription EventSubscription) error {
	return w.clientSizeChanged.Remove(subscription)
}

// AddOrientationChangedHandler registers a handler for
// GameWindow::OrientationChanged.
func (w *GameWindow) AddOrientationChangedHandler(handler EventHandler[*EventArgs]) (EventSubscription, error) {
	return w.orientationChanged.Add(handler)
}

// RemoveOrientationChangedHandler removes the registration the token names.
func (w *GameWindow) RemoveOrientationChangedHandler(subscription EventSubscription) error {
	return w.orientationChanged.Remove(subscription)
}

// raiseNativeWindowEvent is the private end of the native window bridge. It is
// not a projected member and is not reachable from outside this package.
//
// Each canonical signal is routed to the reference's own raise path for that
// event and to no other. There is no edge-trigger guard here, unlike the two
// activation signals: the reference's window events have no `if (already)`
// gate anywhere -- they are raised straight from the platform notification.
func (w *GameWindow) raiseNativeWindowEvent(event uint32) error {
	if w == nil {
		return nil
	}
	switch event {
	case interop.GameWindowEventClientSizeChanged:
		return w.OnClientSizeChanged()
	case interop.GameWindowEventOrientationChanged:
		return w.OnOrientationChanged()
	case interop.GameWindowEventScreenDeviceNameChanged:
		return w.OnScreenDeviceNameChanged()
	default:
		return nil
	}
}
