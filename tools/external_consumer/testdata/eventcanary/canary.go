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

	enabledChanged     framework.EventSource[*framework.EventArgs]
	updateOrderChanged framework.EventSource[*framework.EventArgs]
	visibleChanged     framework.EventSource[*framework.EventArgs]
	drawOrderChanged   framework.EventSource[*framework.EventArgs]
}

func NewRotator(name string) *Rotator {
	return &Rotator{name: name}
}

func (r *Rotator) Name() string { return r.name }

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
