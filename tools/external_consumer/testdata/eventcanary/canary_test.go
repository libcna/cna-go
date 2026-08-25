package eventcanary

import (
	"errors"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

func TestExternalTypeSatisfiesBothComponentContracts(t *testing.T) {
	rotator := NewRotator("external")
	var updateable framework.IUpdateable = rotator
	var drawable framework.IDrawable = rotator

	updateable.Update(framework.NewGameTimeByNone())
	drawable.Draw(framework.NewGameTimeByNone())
	if rotator.Updates != 1 || rotator.Draws != 1 {
		t.Fatalf("Updates=%d Draws=%d", rotator.Updates, rotator.Draws)
	}
	if updateable.Enabled() || drawable.Visible() ||
		updateable.UpdateOrder() != 0 || drawable.DrawOrder() != 0 {
		t.Fatal("a zero-valued external conformer reported non-zero contract state")
	}
}

func TestExternalTypeRaisesItsOwnEvents(t *testing.T) {
	rotator := NewRotator("external")
	var updateable framework.IUpdateable = rotator

	var order []string
	var senders []any
	var args []*framework.EventArgs
	record := func(name string) framework.EventHandler[*framework.EventArgs] {
		return func(sender any, a *framework.EventArgs) error {
			order = append(order, name)
			senders = append(senders, sender)
			args = append(args, a)
			return nil
		}
	}

	first, err := updateable.AddEnabledChangedHandler(record("first"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := updateable.AddEnabledChangedHandler(record("second"))
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("two external registrations produced one token")
	}

	if err := rotator.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != "first" || order[1] != "second" {
		t.Fatalf("handler order = %v", order)
	}
	for i := range senders {
		if senders[i] != any(rotator) {
			t.Fatalf("sender[%d] is not the declaring external instance", i)
		}
		if args[i] != framework.EventArgsEmpty() {
			t.Fatalf("args[%d] is not the shared EventArgs.Empty identity", i)
		}
	}

	// Setting the same value again raises nothing.
	order = nil
	if err := rotator.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	if len(order) != 0 {
		t.Fatalf("an unchanged value raised %v", order)
	}
}

// TestExternalDuplicateRegistrationsAreRemovedSeparately is the token-identity
// projection seen from outside the binding.
func TestExternalDuplicateRegistrationsAreRemovedSeparately(t *testing.T) {
	rotator := NewRotator("external")
	runs := 0
	same := func(sender any, a *framework.EventArgs) error {
		runs++
		return nil
	}

	first, err := rotator.AddVisibleChangedHandler(same)
	if err != nil {
		t.Fatal(err)
	}
	second, err := rotator.AddVisibleChangedHandler(same)
	if err != nil {
		t.Fatal(err)
	}

	if err := rotator.SetVisible(true); err != nil {
		t.Fatal(err)
	}
	if runs != 2 {
		t.Fatalf("one handler added twice ran %d times, want 2", runs)
	}

	runs = 0
	if err := rotator.RemoveVisibleChangedHandler(first); err != nil {
		t.Fatal(err)
	}
	if err := rotator.SetVisible(false); err != nil {
		t.Fatal(err)
	}
	if runs != 1 {
		t.Fatalf("after removing one duplicate the handler ran %d times, want 1", runs)
	}

	runs = 0
	if err := rotator.RemoveVisibleChangedHandler(second); err != nil {
		t.Fatal(err)
	}
	if err := rotator.RemoveVisibleChangedHandler(second); err != nil {
		t.Fatalf("removing an already-removed token = %v", err)
	}
	if err := rotator.RemoveVisibleChangedHandler(framework.EventSubscription{}); err != nil {
		t.Fatalf("removing the zero token = %v", err)
	}
	if err := rotator.SetVisible(true); err != nil {
		t.Fatal(err)
	}
	if runs != 0 {
		t.Fatalf("after removing both registrations the handler ran %d times", runs)
	}
}

// TestExternalTokensDoNotCrossInstances proves a token from one external
// instance cannot disturb another.
func TestExternalTokensDoNotCrossInstances(t *testing.T) {
	left, right := NewRotator("left"), NewRotator("right")
	leftRuns, rightRuns := 0, 0

	leftToken, err := left.AddDrawOrderChangedHandler(func(sender any, a *framework.EventArgs) error {
		leftRuns++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := right.AddDrawOrderChangedHandler(func(sender any, a *framework.EventArgs) error {
		rightRuns++
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// Handing the left instance's token to the right instance must disturb
	// neither event.
	if err := right.RemoveDrawOrderChangedHandler(leftToken); err != nil {
		t.Fatalf("cross-instance removal = %v", err)
	}
	if err := left.SetDrawOrder(3); err != nil {
		t.Fatal(err)
	}
	if err := right.SetDrawOrder(4); err != nil {
		t.Fatal(err)
	}
	if leftRuns != 1 || rightRuns != 1 {
		t.Fatalf("leftRuns=%d rightRuns=%d, want both events intact", leftRuns, rightRuns)
	}
}

// TestExternalHandlerFailurePropagates proves the external raiser sees a
// handler's failure and that no later handler ran.
func TestExternalHandlerFailurePropagates(t *testing.T) {
	rotator := NewRotator("external")
	failure := errors.New("external observer refused")
	later := false

	if _, err := rotator.AddUpdateOrderChangedHandler(func(sender any, a *framework.EventArgs) error {
		return failure
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := rotator.AddUpdateOrderChangedHandler(func(sender any, a *framework.EventArgs) error {
		later = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if err := rotator.SetUpdateOrder(7); err != nil {
		if !errors.Is(err, failure) {
			t.Fatalf("SetUpdateOrder = %v, want the handler failure", err)
		}
	} else {
		t.Fatal("SetUpdateOrder swallowed the handler failure")
	}
	if later {
		t.Fatal("a later handler ran after an earlier one failed")
	}
	if rotator.UpdateOrder() != 7 {
		t.Fatal("the failed raise rolled back the state change")
	}
}

// TestExternalConsumerCannotRaiseThroughTheContract is the capability boundary:
// a consumer holding only the contract can add and remove handlers, and there
// is no projected way to raise. The EventSource fields are unexported, so the
// only raiser is the declaring type itself.
func TestExternalConsumerCannotRaiseThroughTheContract(t *testing.T) {
	rotator := NewRotator("external")
	var updateable framework.IUpdateable = rotator

	if _, isRaiser := any(updateable).(interface {
		Raise(any, *framework.EventArgs) error
	}); isRaiser {
		t.Fatal("the contract exposes a raise operation to plain consumers")
	}
	if _, isSource := any(updateable).(interface {
		Add(framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error)
	}); isSource {
		t.Fatal("the contract exposes the EventSource surface to plain consumers")
	}
}

// TestExternalUseOfTheEventArgsCarriers proves the three System.EventArgs
// carriers are usable from outside, including the two whose construction is
// deliberately not public.
func TestExternalUseOfTheEventArgsCarriers(t *testing.T) {
	// GameComponentCollectionEventArgs declares a public constructor.
	carrier := framework.NewGameComponentCollectionEventArgs(nil)
	if carrier == nil || carrier.GameComponent() != nil {
		t.Fatalf("carrier = %v", carrier)
	}
	// A carrier can be raised through an EventSource of its own type.
	var source framework.EventSource[*framework.GameComponentCollectionEventArgs]
	seen := 0
	if _, err := source.Add(func(sender any, args *framework.GameComponentCollectionEventArgs) error {
		if args != carrier {
			t.Fatal("the raised carrier is not the exact instance")
		}
		seen++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := source.Raise(nil, carrier); err != nil {
		t.Fatal(err)
	}
	if seen != 1 {
		t.Fatalf("handler ran %d times", seen)
	}
}
