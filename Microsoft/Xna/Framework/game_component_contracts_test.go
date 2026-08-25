package framework

import (
	"errors"
	"testing"
)

// spinner is a test-only conformer. It is exactly the shape an external package
// must be able to write: private EventSource fields, projected accessors that
// delegate to them, and setters that raise only when the value actually
// changes, which is what GameComponent::set_Enabled does in the reference.
type spinner struct {
	enabled     bool
	updateOrder int32
	visible     bool
	drawOrder   int32
	updates     int
	draws       int

	enabledChanged     EventSource[*EventArgs]
	updateOrderChanged EventSource[*EventArgs]
	visibleChanged     EventSource[*EventArgs]
	drawOrderChanged   EventSource[*EventArgs]
}

func (s *spinner) Enabled() bool            { return s.enabled }
func (s *spinner) UpdateOrder() int32       { return s.updateOrder }
func (s *spinner) Visible() bool            { return s.visible }
func (s *spinner) DrawOrder() int32         { return s.drawOrder }
func (s *spinner) Update(gameTime GameTime) { s.updates++ }
func (s *spinner) Draw(gameTime GameTime)   { s.draws++ }

func (s *spinner) AddEnabledChangedHandler(h EventHandler[*EventArgs]) (EventSubscription, error) {
	return s.enabledChanged.Add(h)
}
func (s *spinner) RemoveEnabledChangedHandler(sub EventSubscription) error {
	return s.enabledChanged.Remove(sub)
}
func (s *spinner) AddUpdateOrderChangedHandler(h EventHandler[*EventArgs]) (EventSubscription, error) {
	return s.updateOrderChanged.Add(h)
}
func (s *spinner) RemoveUpdateOrderChangedHandler(sub EventSubscription) error {
	return s.updateOrderChanged.Remove(sub)
}
func (s *spinner) AddVisibleChangedHandler(h EventHandler[*EventArgs]) (EventSubscription, error) {
	return s.visibleChanged.Add(h)
}
func (s *spinner) RemoveVisibleChangedHandler(sub EventSubscription) error {
	return s.visibleChanged.Remove(sub)
}
func (s *spinner) AddDrawOrderChangedHandler(h EventHandler[*EventArgs]) (EventSubscription, error) {
	return s.drawOrderChanged.Add(h)
}
func (s *spinner) RemoveDrawOrderChangedHandler(sub EventSubscription) error {
	return s.drawOrderChanged.Remove(sub)
}

// SetEnabled mirrors GameComponent::set_Enabled, whose IL compares first and
// raises only when the stored value actually changes.
func (s *spinner) SetEnabled(value bool) error {
	if s.enabled == value {
		return nil
	}
	s.enabled = value
	return s.enabledChanged.Raise(s, EventArgsEmpty())
}

func (s *spinner) SetVisible(value bool) error {
	if s.visible == value {
		return nil
	}
	s.visible = value
	return s.visibleChanged.Raise(s, EventArgsEmpty())
}

// Both contracts are satisfied by the pointer method set.
var (
	_ IUpdateable = (*spinner)(nil)
	_ IDrawable   = (*spinner)(nil)
)

func TestComponentContractsAreSatisfiableAndInfallible(t *testing.T) {
	component := &spinner{}
	var updateable IUpdateable = component
	var drawable IDrawable = component

	// Update and Draw carry no error: both shipped implementors are a bare ret.
	updateable.Update(NewGameTimeByNone())
	drawable.Draw(NewGameTimeByNone())
	if component.updates != 1 || component.draws != 1 {
		t.Fatalf("updates=%d draws=%d", component.updates, component.draws)
	}
	if updateable.Enabled() || updateable.UpdateOrder() != 0 ||
		drawable.Visible() || drawable.DrawOrder() != 0 {
		t.Fatal("zero-valued conformer reports non-zero contract state")
	}
}

func TestComponentContractEventsRaiseThroughTheInterface(t *testing.T) {
	component := &spinner{}
	var updateable IUpdateable = component

	var senders []any
	var args []*EventArgs
	handler := func(sender any, a *EventArgs) error {
		senders = append(senders, sender)
		args = append(args, a)
		return nil
	}

	first, err := updateable.AddEnabledChangedHandler(handler)
	if err != nil {
		t.Fatal(err)
	}
	second, err := updateable.AddEnabledChangedHandler(handler)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("two registrations through the interface produced one token")
	}

	if err := component.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	if len(senders) != 2 {
		t.Fatalf("handlers run = %d, want both duplicates", len(senders))
	}
	for i := range senders {
		if senders[i] != any(component) {
			t.Fatalf("sender[%d] is not the declaring instance", i)
		}
		if args[i] != EventArgsEmpty() {
			t.Fatalf("args[%d] is not the shared Empty identity", i)
		}
	}

	// Setting the same value again raises nothing, as the reference does.
	senders = nil
	if err := component.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	if len(senders) != 0 {
		t.Fatalf("an unchanged value raised %d handlers", len(senders))
	}

	// Removing one duplicate leaves the other.
	senders = nil
	if err := updateable.RemoveEnabledChangedHandler(first); err != nil {
		t.Fatal(err)
	}
	if err := component.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	if len(senders) != 1 {
		t.Fatalf("handlers run after removing one duplicate = %d", len(senders))
	}
	if err := updateable.RemoveEnabledChangedHandler(second); err != nil {
		t.Fatal(err)
	}
}

func TestComponentContractEventFailurePropagates(t *testing.T) {
	component := &spinner{}
	failure := errors.New("observer refused")
	if _, err := component.AddVisibleChangedHandler(func(sender any, a *EventArgs) error {
		return failure
	}); err != nil {
		t.Fatal(err)
	}
	ran := false
	if _, err := component.AddVisibleChangedHandler(func(sender any, a *EventArgs) error {
		ran = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := component.SetVisible(true); !errors.Is(err, failure) {
		t.Fatalf("SetVisible = %v, want the handler failure", err)
	}
	if ran {
		t.Fatal("a later handler ran after an earlier one failed")
	}
	// The state change happened before the raise, exactly as the reference
	// stores the field before calling its On<Event> helper.
	if !component.Visible() {
		t.Fatal("the failed raise rolled back the state change")
	}
}

func TestGameComponentCollectionEventArgsIsAManagedCarrier(t *testing.T) {
	component := &initializable{}
	args := NewGameComponentCollectionEventArgs(component)
	if args == nil {
		t.Fatal("constructor returned nil")
	}
	if args.GameComponent() != IGameComponent(component) {
		t.Fatal("the carrier did not return the exact component it was given")
	}

	// The reference constructor validates nothing, so a nil component is stored
	// exactly as a null one is.
	empty := NewGameComponentCollectionEventArgs(nil)
	if empty.GameComponent() != nil {
		t.Fatalf("nil component read back as %v", empty.GameComponent())
	}

	// Reference semantics: two variables naming one instance see one value, and
	// two constructions are distinct instances.
	shared := args
	if shared.GameComponent() != args.GameComponent() {
		t.Fatal("aliased carrier disagreed with its origin")
	}
	if NewGameComponentCollectionEventArgs(component) == args {
		t.Fatal("two constructions produced one instance")
	}

	// The CLR base is a measured relationship, not Go structure: the carrier is
	// its own reference type and is not an EventArgs.
	var carrier any = args
	if _, isBase := carrier.(*EventArgs); isBase {
		t.Fatal("the carrier is assignable to its CLR base; the base was faked in Go")
	}
}

// spinnerImplementsIGameComponent is separate on purpose: IGameComponent is a
// different contract with a different boundary, and a conformer of the two
// component contracts is not required to implement it.
type initializable struct{ spinner }

func (i *initializable) Initialize() error { return nil }

var (
	_ IGameComponent = (*initializable)(nil)
	_ IUpdateable    = (*initializable)(nil)
	_ IDrawable      = (*initializable)(nil)
)
