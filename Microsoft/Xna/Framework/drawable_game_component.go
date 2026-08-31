package framework

import (
	"errors"
	"fmt"

	"github.com/openeggbert/cna-go/internal/servicebridge"
)

// DrawableGameComponent is the Go projection of
// Microsoft.Xna.Framework.DrawableGameComponent: a GameComponent that also
// draws, and the profile's one shipped IDrawable implementor.
//
// # It composes GameComponent, and does not embed it
//
// The settled XNA-to-XNA inheritance rule is private named composition plus
// explicit measured forwarding. The base object is private state, there is no
// Base/Parent/AsGameComponent accessor, and nothing is promoted: every
// inherited member below is a forwarding method written out, so a member this
// class overrides cannot silently keep the base's body.
//
// # What blocked it, and what unblocked it
//
// Initialize's body resolves Microsoft.Xna.Framework.Graphics.IGraphicsDeviceService
// out of Game.Services. The Graphics package imports this one, so this package
// cannot name that contract, cannot build the reflect.Type token GetService is
// keyed by, and cannot spell the return type of the service's device accessor.
// The cross-package cycle rule projects device-typed MEMBERS into the
// descendant package, which a private field resolution inside a method body
// cannot use.
//
// internal/servicebridge is the narrowest thing that resolves it: two function
// values, installed from package inits, adding no public API and holding no
// object. The Graphics package installs a resolver that knows the token and can
// compare a device with nil; this package installs a reader so the Graphics
// package can project get_GraphicsDevice over private state here.
//
// A program that never imports the Graphics package installs no resolver and
// resolves nothing -- which is correct rather than a gap, because registering
// an IGraphicsDeviceService requires naming it, which requires importing that
// package.
//
// # The virtual calls Initialize makes
//
// The reference's Initialize ends with `if (deviceService.GraphicsDevice !=
// null) this.LoadContent();`, and DeviceCreated/DeviceDisposing call
// LoadContent/UnloadContent the same way. All three are `callvirt`, so in the
// CLR a subclass override runs.
//
// Go has no virtual dispatch, and the settled rule since Foundation 31 is that
// base behaviour is explicit and never automatic. So those calls reach the
// bodies projected here, which are the reference's own: LoadContent,
// UnloadContent and Draw are each a bare `ret` of code size 1. A consumer's own
// LoadContent is not reached, exactly as a consumer's own Update is not reached
// by base Update, and for the same reason.
type DrawableGameComponent struct {
	// component is the private named GameComponent base. Foundation 41's rule
	// in one field: never embedded, never exported, never accessible.
	component *GameComponent

	// initialized is DrawableGameComponent::initialized. It gates the whole
	// resolution body; the store that sets it is deliberately OUTSIDE that
	// body, because the reference's `brtrue` jumps TO the store rather than
	// past it.
	initialized bool

	// visible is DrawableGameComponent::visible, which the constructor sets
	// to TRUE before it calls the base constructor. drawOrder is
	// ::drawOrder, which the constructor does not assign, so it starts zero.
	visible   bool
	drawOrder int32

	// deviceService is DrawableGameComponent::deviceService. It is the
	// consumer's own IGraphicsDeviceService, held through the unexported
	// structural interface below so this package can subscribe to it without
	// naming the Graphics contract.
	deviceService drawableDeviceService

	// deviceSubscriptions are the four registrations Initialize installs, in
	// the reference's own order. They are held because Go event registrations
	// are named by a token rather than matched by delegate identity, so
	// Dispose has to release the tokens it took rather than reconstruct a
	// delegate the way Delegate.Remove does.
	deviceSubscriptions [drawableDeviceEventCount]EventSubscription

	// The two events the class declares, each a private multicast delegate
	// field in the reference.
	visibleChanged   EventSource[*EventArgs]
	drawOrderChanged EventSource[*EventArgs]
}

// drawableDeviceService is the shape DrawableGameComponent needs from an
// IGraphicsDeviceService, and it is deliberately only the shape.
//
// All eight methods are framework-typed on both sides, so the Graphics
// package's IGraphicsDeviceService satisfies this interface with no adapter at
// all -- a consumer's own service does too, because the projected contract is
// what they implement. The ninth member of that contract, GraphicsDevice(),
// returns a type this package cannot name and is deliberately absent: the one
// place Initialize needs it is a nil comparison, and servicebridge supplies
// that as a closure.
//
// The interface is unexported, so it is not public API, and its method names
// are exported so a type declared in another package can satisfy it.
type drawableDeviceService interface {
	AddDeviceCreatedHandler(handler EventHandler[*EventArgs]) (EventSubscription, error)
	RemoveDeviceCreatedHandler(subscription EventSubscription) error
	AddDeviceResettingHandler(handler EventHandler[*EventArgs]) (EventSubscription, error)
	RemoveDeviceResettingHandler(subscription EventSubscription) error
	AddDeviceResetHandler(handler EventHandler[*EventArgs]) (EventSubscription, error)
	RemoveDeviceResetHandler(subscription EventSubscription) error
	AddDeviceDisposingHandler(handler EventHandler[*EventArgs]) (EventSubscription, error)
	RemoveDeviceDisposingHandler(subscription EventSubscription) error
}

// The four device events, in the reference's own subscription order:
// DeviceCreated, DeviceResetting, DeviceReset, DeviceDisposing. Dispose removes
// them in the SAME order rather than in reverse, which is the reference's order
// too and is preserved because Delegate.Remove is order-independent while a
// token release is not.
const (
	drawableDeviceCreated = iota
	drawableDeviceResetting
	drawableDeviceReset
	drawableDeviceDisposing

	drawableDeviceEventCount
)

// errDrawableComponentInvalidOperation projects System.InvalidOperationException,
// the only failure both Initialize and get_GraphicsDevice have. It is
// unexported because the XNA public contract declares no error type here.
var errDrawableComponentInvalidOperation = errors.New("operation is not valid")

// missingGraphicsDeviceService is the exact Resources string Initialize's throw
// site loads, read from the Microsoft.Xna.Framework.Resources.resources stream
// of the retained Microsoft.Xna.Framework.Game.dll.
//
// The OTHER string this class throws -- the one get_GraphicsDevice loads under
// the key PropertyCannotBeCalledBeforeInitialize -- lives in the Graphics
// package with the member that throws it, and is not what the key suggests.
const missingGraphicsDeviceService = "Drawable components require a graphics device service in the game service container."

func drawableComponentError(message string) error {
	return fmt.Errorf("%w: %s", errDrawableComponentInvalidOperation, message)
}

// init installs the reader the Graphics package uses to project
// get_GraphicsDevice over this package's private field. It holds nothing: the
// closure receives the component it is asked about.
func init() {
	servicebridge.SetComponentServiceReader(func(component any) (any, bool) {
		typed, ok := component.(*DrawableGameComponent)
		if !ok || typed == nil || typed.deviceService == nil {
			return nil, false
		}
		return typed.deviceService, true
	})
}

// NewDrawableGameComponent is DrawableGameComponent::.ctor(Game):
//
//	this.visible = true;
//	base..ctor(game);
//
// The field store comes BEFORE the base constructor call, which is unusual and
// is preserved. It is not observable in this profile -- GameComponent's
// constructor only stores the Game -- but the reference's order is the
// reference's order.
//
// It validates nothing, including a nil Game: GameComponent's constructor is
// `ldarg.0; ldarg.1; stfld; ret` with no check, and inventing one here would be
// a failure mode the reference does not have.
func NewDrawableGameComponent(game *Game) *DrawableGameComponent {
	return &DrawableGameComponent{visible: true, component: NewGameComponent(game)}
}

// Initialize is DrawableGameComponent::Initialize, reproduced statement for
// statement:
//
//	base.Initialize();
//	if (!this.initialized)
//	{
//	    this.deviceService = Game.Services.GetService(typeof(IGraphicsDeviceService))
//	                         as IGraphicsDeviceService;
//	    if (this.deviceService == null)
//	        throw new InvalidOperationException(Resources.MissingGraphicsDeviceService);
//	    deviceService.DeviceCreated   += DeviceCreated;
//	    deviceService.DeviceResetting += DeviceResetting;
//	    deviceService.DeviceReset     += DeviceReset;
//	    deviceService.DeviceDisposing += DeviceDisposing;
//	    if (this.deviceService.GraphicsDevice != null) this.LoadContent();
//	}
//	this.initialized = true;
//
// Three details are load-bearing and easy to get wrong.
//
// The `initialized` store is OUTSIDE the guard. The IL's `brtrue` at the top
// jumps to the store itself, not past it, so a second Initialize re-assigns
// true and skips only the body. Writing `if (initialized) return;` would be one
// statement shorter and would leave the flag unset when the throw path is taken
// -- which is the third detail: a component whose service is missing throws
// with `initialized` still FALSE, so a later Initialize retries the resolution.
//
// The subscription happens before the device check, so a component initialized
// while the service publishes no device is still wired for DeviceCreated and
// will be told when one appears.
func (c *DrawableGameComponent) Initialize() error {
	if c == nil {
		return errors.New("DrawableGameComponent is nil")
	}
	if err := c.component.Initialize(); err != nil {
		return err
	}
	if !c.initialized {
		service, hasDevice, resolved := c.resolveDeviceService()
		if !resolved {
			return drawableComponentError(missingGraphicsDeviceService)
		}
		c.deviceService = service
		if err := c.subscribeDeviceEvents(); err != nil {
			return err
		}
		if hasDevice() {
			c.LoadContent()
		}
	}
	c.initialized = true
	return nil
}

// resolveDeviceService is the `Services.GetService(typeof(...)) as ...` pair.
// The `as` cast is the reason a registration that is not an
// IGraphicsDeviceService produces nil rather than a failure: `isinst` yields
// null, and the null check below it is what throws.
func (c *DrawableGameComponent) resolveDeviceService() (drawableDeviceService, func() bool, bool) {
	game := c.component.Game()
	if game == nil {
		return nil, nil, false
	}
	value, hasDevice, ok := servicebridge.ResolveDeviceService(game.Services())
	if !ok || value == nil || hasDevice == nil {
		return nil, nil, false
	}
	service, structural := value.(drawableDeviceService)
	if !structural {
		return nil, nil, false
	}
	return service, hasDevice, true
}

// subscribeDeviceEvents installs the four handlers in the reference's order.
//
// A partial failure releases what it already installed and clears the field, so
// a failed Initialize leaves no half-wired component behind. The reference
// cannot fail here at all -- Delegate.Combine does not throw -- so this is the
// Go event projection's own channel rather than a reference behaviour.
func (c *DrawableGameComponent) subscribeDeviceEvents() error {
	subscriptions := [drawableDeviceEventCount]struct {
		add     func(EventHandler[*EventArgs]) (EventSubscription, error)
		handler EventHandler[*EventArgs]
	}{
		{c.deviceService.AddDeviceCreatedHandler, c.deviceCreated},
		{c.deviceService.AddDeviceResettingHandler, c.deviceResetting},
		{c.deviceService.AddDeviceResetHandler, c.deviceReset},
		{c.deviceService.AddDeviceDisposingHandler, c.deviceDisposing},
	}
	for index, entry := range subscriptions {
		token, err := entry.add(entry.handler)
		if err != nil {
			c.releaseDeviceEvents()
			c.deviceService = nil
			return err
		}
		c.deviceSubscriptions[index] = token
	}
	return nil
}

// releaseDeviceEvents removes every installed registration once, in the
// reference's own subscription order, and reports the first failure while still
// releasing the rest.
func (c *DrawableGameComponent) releaseDeviceEvents() error {
	if c.deviceService == nil {
		return nil
	}
	removals := [drawableDeviceEventCount]func(EventSubscription) error{
		c.deviceService.RemoveDeviceCreatedHandler,
		c.deviceService.RemoveDeviceResettingHandler,
		c.deviceService.RemoveDeviceResetHandler,
		c.deviceService.RemoveDeviceDisposingHandler,
	}
	var first error
	for index, remove := range removals {
		token := c.deviceSubscriptions[index]
		c.deviceSubscriptions[index] = EventSubscription{}
		if err := remove(token); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// The four private device handlers, each exactly its reference body:
//
//	DeviceCreated    this.LoadContent();
//	DeviceDisposing  this.UnloadContent();
//	DeviceResetting  ret
//	DeviceReset      ret
//
// The two empty ones are subscribed anyway, because the reference subscribes
// them: a projection that skipped them would install two registrations where
// the reference installs four, and a service that counts its subscribers would
// see the difference.
func (c *DrawableGameComponent) deviceCreated(sender any, args *EventArgs) error {
	c.LoadContent()
	return nil
}

func (c *DrawableGameComponent) deviceResetting(sender any, args *EventArgs) error { return nil }

func (c *DrawableGameComponent) deviceReset(sender any, args *EventArgs) error { return nil }

func (c *DrawableGameComponent) deviceDisposing(sender any, args *EventArgs) error {
	c.UnloadContent()
	return nil
}

// Dispose is DrawableGameComponent::Dispose(bool):
//
//	if (disposing)
//	{
//	    this.UnloadContent();
//	    if (this.deviceService != null)
//	    {
//	        deviceService.DeviceCreated   -= DeviceCreated;
//	        deviceService.DeviceResetting -= DeviceResetting;
//	        deviceService.DeviceReset     -= DeviceReset;
//	        deviceService.DeviceDisposing -= DeviceDisposing;
//	    }
//	}
//	base.Dispose(disposing);
//
// The base call is UNGUARDED: it runs on both paths, which is what makes
// Dispose(false) still remove the component from Game.Components and raise
// Disposed. And like every Dispose in this profile, it is not idempotent --
// there is no disposed flag anywhere in either class.
func (c *DrawableGameComponent) Dispose(disposing bool) error {
	if c == nil {
		return errors.New("DrawableGameComponent is nil")
	}
	var released error
	if disposing {
		c.UnloadContent()
		released = c.releaseDeviceEvents()
	}
	baseErr := c.component.DisposeByBoolean(disposing)
	if released != nil {
		return released
	}
	return baseErr
}

// Draw is DrawableGameComponent::Draw(GameTime), whose whole body is `ret`. It
// is infallible, which IDrawable already required on this exact evidence.
func (c *DrawableGameComponent) Draw(gameTime GameTime) {}

// LoadContent is the protected DrawableGameComponent::LoadContent, one `ret`.
func (c *DrawableGameComponent) LoadContent() {}

// UnloadContent is the protected DrawableGameComponent::UnloadContent, one `ret`.
func (c *DrawableGameComponent) UnloadContent() {}

// Visible is DrawableGameComponent::get_Visible, one `ldfld`. The constructor
// sets it true.
func (c *DrawableGameComponent) Visible() bool { return c.visible }

// SetVisible is DrawableGameComponent::set_Visible:
//
//	if (this.visible != value) { this.visible = value; OnVisibleChanged(this, EventArgs.Empty); }
//
// An unchanged assignment announces nothing, which is why the setter is
// fallible and the getter is not: the failure is the consumer handler's.
func (c *DrawableGameComponent) SetVisible(value bool) error {
	if c.visible == value {
		return nil
	}
	c.visible = value
	return c.OnVisibleChanged(c, EventArgsEmpty())
}

// DrawOrder is DrawableGameComponent::get_DrawOrder, one `ldfld`. The
// constructor does not assign it, so it starts at zero.
func (c *DrawableGameComponent) DrawOrder() int32 { return c.drawOrder }

// SetDrawOrder is DrawableGameComponent::set_DrawOrder, the same
// compare-store-announce shape as SetVisible.
func (c *DrawableGameComponent) SetDrawOrder(value int32) error {
	if c.drawOrder == value {
		return nil
	}
	c.drawOrder = value
	return c.OnDrawOrderChanged(c, EventArgsEmpty())
}

// OnVisibleChanged is the protected raiser:
//
//	if (this.VisibleChanged != null) this.VisibleChanged(sender, args);
//
// It passes the caller's sender and args straight through, unlike
// Game::OnExiting, which substitutes null.
func (c *DrawableGameComponent) OnVisibleChanged(sender any, args *EventArgs) error {
	return c.visibleChanged.Raise(sender, args)
}

// OnDrawOrderChanged is the same raiser over DrawOrderChanged.
func (c *DrawableGameComponent) OnDrawOrderChanged(sender any, args *EventArgs) error {
	return c.drawOrderChanged.Raise(sender, args)
}

// AddVisibleChangedHandler registers a handler for
// DrawableGameComponent::VisibleChanged.
func (c *DrawableGameComponent) AddVisibleChangedHandler(handler EventHandler[*EventArgs]) (EventSubscription, error) {
	return c.visibleChanged.Add(handler)
}

// RemoveVisibleChangedHandler removes the registration the token names.
func (c *DrawableGameComponent) RemoveVisibleChangedHandler(subscription EventSubscription) error {
	return c.visibleChanged.Remove(subscription)
}

// AddDrawOrderChangedHandler registers a handler for
// DrawableGameComponent::DrawOrderChanged.
func (c *DrawableGameComponent) AddDrawOrderChangedHandler(handler EventHandler[*EventArgs]) (EventSubscription, error) {
	return c.drawOrderChanged.Add(handler)
}

// RemoveDrawOrderChangedHandler removes the registration the token names.
func (c *DrawableGameComponent) RemoveDrawOrderChangedHandler(subscription EventSubscription) error {
	return c.drawOrderChanged.Remove(subscription)
}

// The twelve inherited GameComponent members, forwarded explicitly.
//
// Every one is written out rather than promoted, because that is the whole
// point of composition over embedding: an inherited member that arrived by
// promotion would keep the base's body even where the derived class overrides
// it, and DrawableGameComponent overrides two of them.
//
// Initialize and Dispose are NOT here. The derived class declares its own, and
// each calls its base explicitly at the position the reference's IL calls it.

// Game is GameComponent::get_Game, one `ldfld` over what the constructor stored.
func (c *DrawableGameComponent) Game() *Game { return c.component.Game() }

// Enabled is GameComponent::get_Enabled.
func (c *DrawableGameComponent) Enabled() bool { return c.component.Enabled() }

// SetEnabled is GameComponent::set_Enabled, which announces on change.
func (c *DrawableGameComponent) SetEnabled(value bool) error { return c.component.SetEnabled(value) }

// UpdateOrder is GameComponent::get_UpdateOrder.
func (c *DrawableGameComponent) UpdateOrder() int32 { return c.component.UpdateOrder() }

// SetUpdateOrder is GameComponent::set_UpdateOrder.
func (c *DrawableGameComponent) SetUpdateOrder(value int32) error {
	return c.component.SetUpdateOrder(value)
}

// Update is GameComponent::Update(GameTime), a bare `ret`.
func (c *DrawableGameComponent) Update(gameTime GameTime) { c.component.Update(gameTime) }

// AddEnabledChangedHandler forwards to the base's own registration list, which
// is where the reference keeps it: the event is declared on GameComponent.
func (c *DrawableGameComponent) AddEnabledChangedHandler(handler EventHandler[*EventArgs]) (EventSubscription, error) {
	return c.component.AddEnabledChangedHandler(handler)
}

// RemoveEnabledChangedHandler removes the registration the token names.
func (c *DrawableGameComponent) RemoveEnabledChangedHandler(subscription EventSubscription) error {
	return c.component.RemoveEnabledChangedHandler(subscription)
}

// AddUpdateOrderChangedHandler forwards to the base's registration list.
func (c *DrawableGameComponent) AddUpdateOrderChangedHandler(handler EventHandler[*EventArgs]) (EventSubscription, error) {
	return c.component.AddUpdateOrderChangedHandler(handler)
}

// RemoveUpdateOrderChangedHandler removes the registration the token names.
func (c *DrawableGameComponent) RemoveUpdateOrderChangedHandler(subscription EventSubscription) error {
	return c.component.RemoveUpdateOrderChangedHandler(subscription)
}

// AddDisposedHandler forwards to the base's registration list. Disposed is
// declared on GameComponent and raised from its Dispose(bool), which the
// derived Dispose calls unconditionally.
func (c *DrawableGameComponent) AddDisposedHandler(handler EventHandler[*EventArgs]) (EventSubscription, error) {
	return c.component.AddDisposedHandler(handler)
}

// RemoveDisposedHandler removes the registration the token names.
func (c *DrawableGameComponent) RemoveDisposedHandler(subscription EventSubscription) error {
	return c.component.RemoveDisposedHandler(subscription)
}
