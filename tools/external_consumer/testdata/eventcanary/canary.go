// Package eventcanary is an external conformance canary for the CNA-Go event
// architecture.
//
// It is deliberately NOT part of the cna-go module. tools/external_consumer
// materialises it as its own Go module whose only dependency is an extracted,
// audited CNA-Go source artifact, with GOWORK=off and no sibling checkout, so
// what it proves is what a real downstream user can do:
//
//   - declare their own types that satisfy IUpdateable and IDrawable;
//   - own private EventSource fields and expose only the projected accessors;
//   - raise their own events with the shared EventArgs.Empty identity;
//   - subscribe, subscribe again, and remove each registration separately.
//
// Nothing here is XNA surface and nothing here is part of the binding.
package eventcanary

import (
	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	graphics "github.com/openeggbert/cna-go/Microsoft/Xna/Framework/Graphics"
)

// Rotator is an external conformer of both component contracts.
//
// The shape is the one the contracts' documentation describes: the EventSource
// fields are unexported, so a consumer of a Rotator can add and remove handlers
// but cannot raise the events. Only Rotator's own methods can.
type Rotator struct {
	name        string
	enabled     bool
	visible     bool
	updateOrder int32
	drawOrder   int32

	Updates int
	Draws   int

	initialized bool

	enabledChanged     framework.EventSource[*framework.EventArgs]
	updateOrderChanged framework.EventSource[*framework.EventArgs]
	visibleChanged     framework.EventSource[*framework.EventArgs]
	drawOrderChanged   framework.EventSource[*framework.EventArgs]
}

func NewRotator(name string) *Rotator {
	return &Rotator{name: name}
}

func (r *Rotator) Name() string { return r.name }

// Initialized records whether the collection's owner ever initialized this
// component. Nothing in CNA-Go calls it: there is no component loop.
func (r *Rotator) Initialized() bool { return r.initialized }

// Initialize satisfies IGameComponent, which is what a GameComponentCollection
// stores. XNA's own GameComponent satisfies IGameComponent, IUpdateable and
// IDisposable on one class, and an external conformer does the same here.
//
// IGameComponent stays fallible where IUpdateable and IDrawable do not,
// because the reference implementor's Initialize can throw: DrawableGameComponent
// resolves IGraphicsDeviceService out of Game.Services and throws when it is
// absent. This conformer reaches no service and cannot fail.
func (r *Rotator) Initialize() error {
	r.initialized = true
	return nil
}

func (r *Rotator) Enabled() bool      { return r.enabled }
func (r *Rotator) UpdateOrder() int32 { return r.updateOrder }
func (r *Rotator) Visible() bool      { return r.visible }
func (r *Rotator) DrawOrder() int32   { return r.drawOrder }

func (r *Rotator) Update(gameTime framework.GameTime) { r.Updates++ }
func (r *Rotator) Draw(gameTime framework.GameTime)   { r.Draws++ }

func (r *Rotator) AddEnabledChangedHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return r.enabledChanged.Add(h)
}
func (r *Rotator) RemoveEnabledChangedHandler(s framework.EventSubscription) error {
	return r.enabledChanged.Remove(s)
}
func (r *Rotator) AddUpdateOrderChangedHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return r.updateOrderChanged.Add(h)
}
func (r *Rotator) RemoveUpdateOrderChangedHandler(s framework.EventSubscription) error {
	return r.updateOrderChanged.Remove(s)
}
func (r *Rotator) AddVisibleChangedHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return r.visibleChanged.Add(h)
}
func (r *Rotator) RemoveVisibleChangedHandler(s framework.EventSubscription) error {
	return r.visibleChanged.Remove(s)
}
func (r *Rotator) AddDrawOrderChangedHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return r.drawOrderChanged.Add(h)
}
func (r *Rotator) RemoveDrawOrderChangedHandler(s framework.EventSubscription) error {
	return r.drawOrderChanged.Remove(s)
}

// SetEnabled mirrors GameComponent::set_Enabled: store first, then raise, and
// only when the value actually changed.
func (r *Rotator) SetEnabled(value bool) error {
	if r.enabled == value {
		return nil
	}
	r.enabled = value
	return r.enabledChanged.Raise(r, framework.EventArgsEmpty())
}

func (r *Rotator) SetUpdateOrder(value int32) error {
	if r.updateOrder == value {
		return nil
	}
	r.updateOrder = value
	return r.updateOrderChanged.Raise(r, framework.EventArgsEmpty())
}

func (r *Rotator) SetVisible(value bool) error {
	if r.visible == value {
		return nil
	}
	r.visible = value
	return r.visibleChanged.Raise(r, framework.EventArgsEmpty())
}

func (r *Rotator) SetDrawOrder(value int32) error {
	if r.drawOrder == value {
		return nil
	}
	r.drawOrder = value
	return r.drawOrderChanged.Raise(r, framework.EventArgsEmpty())
}

// Compile-time conformance from outside the binding. If either contract stopped
// being satisfiable by an external type, this module would not build.
var (
	_ framework.IUpdateable = (*Rotator)(nil)
	_ framework.IDrawable   = (*Rotator)(nil)
)

// CollectionProbe is an external observer of a GameComponentCollection.
//
// It exists to prove that a caller holding nothing but the published contract
// can subscribe to both events and reconstruct the exact order in which the
// collection mutates and announces. It reaches no CNA-Go internal: the
// collection's backing store, its private Collection<T> adapter, and its
// override hooks are all unreachable from here, which is the point.
type CollectionProbe struct {
	Events     []string
	Senders    []any
	Args       []*framework.GameComponentCollectionEventArgs
	Components []framework.IGameComponent
	// Counts is what each handler observed Count to be at the moment it ran,
	// which is how an outside caller can tell "mutate then announce" from
	// "announce then mutate" without seeing any implementation.
	Counts []int32
}

// Handler returns a handler that records one raise under the given label.
func (p *CollectionProbe) Handler(label string) framework.EventHandler[*framework.GameComponentCollectionEventArgs] {
	return func(sender any, args *framework.GameComponentCollectionEventArgs) error {
		p.Events = append(p.Events, label)
		p.Senders = append(p.Senders, sender)
		p.Args = append(p.Args, args)
		p.Components = append(p.Components, args.GameComponent())
		if collection, ok := sender.(*framework.GameComponentCollection); ok {
			p.Counts = append(p.Counts, collection.Count())
		} else {
			p.Counts = append(p.Counts, -1)
		}
		return nil
	}
}

// Failing returns a handler that records the raise and then fails, so a caller
// can observe what a failed announcement leaves behind.
func (p *CollectionProbe) Failing(label string, err error) framework.EventHandler[*framework.GameComponentCollectionEventArgs] {
	inner := p.Handler(label)
	return func(sender any, args *framework.GameComponentCollectionEventArgs) error {
		_ = inner(sender, args)
		return err
	}
}

// Reset clears everything the probe has recorded.
func (p *CollectionProbe) Reset() {
	p.Events, p.Senders, p.Args, p.Components, p.Counts = nil, nil, nil, nil, nil
}

// DeviceService is an external conformer of the device-publication contract.
//
// It proves a cross-package claim the in-repo tests cannot: a type declared
// outside CNA-Go can satisfy a contract declared in the GRAPHICS package whose
// event accessors are spelled in FRAMEWORK-package types, without importing
// anything unexported and without a CNA-Go implementor to copy.
type DeviceService struct {
	device *graphics.GraphicsDevice

	created   framework.EventSource[*framework.EventArgs]
	disposing framework.EventSource[*framework.EventArgs]
	reset     framework.EventSource[*framework.EventArgs]
	resetting framework.EventSource[*framework.EventArgs]
}

func (s *DeviceService) GraphicsDevice() *graphics.GraphicsDevice { return s.device }

func (s *DeviceService) AddDeviceCreatedHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.created.Add(h)
}
func (s *DeviceService) RemoveDeviceCreatedHandler(sub framework.EventSubscription) error {
	return s.created.Remove(sub)
}
func (s *DeviceService) AddDeviceDisposingHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.disposing.Add(h)
}
func (s *DeviceService) RemoveDeviceDisposingHandler(sub framework.EventSubscription) error {
	return s.disposing.Remove(sub)
}
func (s *DeviceService) AddDeviceResetHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.reset.Add(h)
}
func (s *DeviceService) RemoveDeviceResetHandler(sub framework.EventSubscription) error {
	return s.reset.Remove(sub)
}
func (s *DeviceService) AddDeviceResettingHandler(h framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	return s.resetting.Add(h)
}
func (s *DeviceService) RemoveDeviceResettingHandler(sub framework.EventSubscription) error {
	return s.resetting.Remove(sub)
}

// RaiseDeviceCreated is how the declaring type publishes the event. A consumer
// holding only the contract has no equivalent, which is the encapsulation the
// two-accessor projection buys.
func (s *DeviceService) RaiseDeviceCreated() error {
	return s.created.Raise(s, framework.EventArgsEmpty())
}
