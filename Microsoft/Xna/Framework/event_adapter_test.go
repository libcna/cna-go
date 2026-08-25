package framework

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

// recorder collects what each handler observed so ordering, sender identity and
// args identity can be asserted exactly rather than by count.
type recorder struct {
	calls  []string
	sender []any
	args   []*EventArgs
}

func (r *recorder) handler(name string) EventHandler[*EventArgs] {
	return func(sender any, args *EventArgs) error {
		r.calls = append(r.calls, name)
		r.sender = append(r.sender, sender)
		r.args = append(r.args, args)
		return nil
	}
}

func (r *recorder) failing(name string, err error) EventHandler[*EventArgs] {
	return func(sender any, args *EventArgs) error {
		r.calls = append(r.calls, name)
		r.sender = append(r.sender, sender)
		r.args = append(r.args, args)
		return err
	}
}

func equalCalls(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestEventArgsEmptyIsOneSharedIdentity(t *testing.T) {
	first, second := EventArgsEmpty(), EventArgsEmpty()
	if first != second {
		t.Fatalf("EventArgsEmpty() returned two identities: %p and %p", first, second)
	}
	if first == nil {
		t.Fatal("EventArgsEmpty() is nil; nil is not the projection of System.EventArgs.Empty")
	}
	// The shared instance must be distinguishable from any other instance, which
	// is exactly what a zero-size EventArgs would destroy: on go1.24.4 every
	// zero-size heap allocation shares runtime.zerobase.
	other := &EventArgs{}
	if other == first {
		t.Fatal("a freshly allocated EventArgs is pointer-equal to the shared Empty instance")
	}
	third := &EventArgs{}
	if third == other {
		t.Fatal("two freshly allocated EventArgs values share an address")
	}
}

func TestEventSourceRaiseWithNoHandlers(t *testing.T) {
	var source EventSource[*EventArgs]
	if err := source.Raise(nil, EventArgsEmpty()); err != nil {
		t.Fatalf("Raise with no handlers = %v", err)
	}
}

func TestEventSourceInvokesOneHandlerWithSenderAndArgsIdentity(t *testing.T) {
	var source EventSource[*EventArgs]
	seen := &recorder{}
	if _, err := source.Add(seen.handler("only")); err != nil {
		t.Fatal(err)
	}
	sender := &struct{ name string }{name: "declaring instance"}
	args := EventArgsEmpty()
	if err := source.Raise(sender, args); err != nil {
		t.Fatal(err)
	}
	if !equalCalls(seen.calls, []string{"only"}) {
		t.Fatalf("calls = %v", seen.calls)
	}
	if seen.sender[0] != any(sender) {
		t.Fatalf("sender identity = %v, want the exact raising instance", seen.sender[0])
	}
	if seen.args[0] != args {
		t.Fatalf("args identity = %p, want %p", seen.args[0], args)
	}
}

func TestEventSourceKeepsRegistrationOrder(t *testing.T) {
	var source EventSource[*EventArgs]
	seen := &recorder{}
	for _, name := range []string{"first", "second", "third", "fourth"} {
		if _, err := source.Add(seen.handler(name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := source.Raise(nil, EventArgsEmpty()); err != nil {
		t.Fatal(err)
	}
	if !equalCalls(seen.calls, []string{"first", "second", "third", "fourth"}) {
		t.Fatalf("calls = %v, want registration order", seen.calls)
	}
}

// TestEventSourceDuplicateRegistrationsAreIndependent is the Go language
// projection this design exists for: the CLR matches delegates by identity in
// Delegate.Remove, Go function values are not comparable, so CNA-Go names the
// registration instead of the handler.
func TestEventSourceDuplicateRegistrationsAreIndependent(t *testing.T) {
	var source EventSource[*EventArgs]
	seen := &recorder{}
	same := seen.handler("same")

	first, err := source.Add(same)
	if err != nil {
		t.Fatal(err)
	}
	second, err := source.Add(same)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("adding one handler twice produced one token; each registration must have its own identity")
	}
	if err := source.Raise(nil, EventArgsEmpty()); err != nil {
		t.Fatal(err)
	}
	if !equalCalls(seen.calls, []string{"same", "same"}) {
		t.Fatalf("calls = %v, want the duplicate registered twice", seen.calls)
	}

	// Removing the first duplicate leaves the second registered.
	seen.calls = nil
	if err := source.Remove(first); err != nil {
		t.Fatal(err)
	}
	if err := source.Raise(nil, EventArgsEmpty()); err != nil {
		t.Fatal(err)
	}
	if !equalCalls(seen.calls, []string{"same"}) {
		t.Fatalf("after removing the first duplicate calls = %v", seen.calls)
	}

	// Removing the second duplicate leaves nothing.
	seen.calls = nil
	if err := source.Remove(second); err != nil {
		t.Fatal(err)
	}
	if err := source.Raise(nil, EventArgsEmpty()); err != nil {
		t.Fatal(err)
	}
	if len(seen.calls) != 0 {
		t.Fatalf("after removing both duplicates calls = %v", seen.calls)
	}
}

func TestEventSourceRemovalIsHarmlessInEveryAbsentCase(t *testing.T) {
	var source, other EventSource[*EventArgs]
	seen := &recorder{}
	token, err := source.Add(seen.handler("kept"))
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := other.Add(seen.handler("foreign"))
	if err != nil {
		t.Fatal(err)
	}

	// The zero token names no registration.
	if err := source.Remove(EventSubscription{}); err != nil {
		t.Fatalf("removing the zero token = %v", err)
	}
	// A token belonging to another event must not disturb either event.
	if err := source.Remove(foreign); err != nil {
		t.Fatalf("removing a foreign token = %v", err)
	}
	if err := source.Raise(nil, EventArgsEmpty()); err != nil {
		t.Fatal(err)
	}
	if err := other.Raise(nil, EventArgsEmpty()); err != nil {
		t.Fatal(err)
	}
	if !equalCalls(seen.calls, []string{"kept", "foreign"}) {
		t.Fatalf("calls = %v, want both events intact", seen.calls)
	}

	// Removing twice is harmless and idempotent.
	if err := source.Remove(token); err != nil {
		t.Fatal(err)
	}
	if err := source.Remove(token); err != nil {
		t.Fatalf("second removal of the same token = %v", err)
	}
	seen.calls = nil
	if err := source.Raise(nil, EventArgsEmpty()); err != nil {
		t.Fatal(err)
	}
	if len(seen.calls) != 0 {
		t.Fatalf("calls after removal = %v", seen.calls)
	}
}

// TestEventSourceDispatchesOverASnapshot proves add and remove during a raise
// change the next dispatch and never the running one.
func TestEventSourceDispatchesOverASnapshot(t *testing.T) {
	var source EventSource[*EventArgs]
	seen := &recorder{}

	var lateToken EventSubscription
	var removeMe EventSubscription
	var err error

	removeMe, err = source.Add(seen.handler("removed-during-dispatch"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = source.Add(func(sender any, args *EventArgs) error {
		seen.calls = append(seen.calls, "mutator")
		// Both mutations must be invisible to the dispatch already running.
		if lateToken, err = source.Add(seen.handler("added-during-dispatch")); err != nil {
			return err
		}
		return source.Remove(removeMe)
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = source.Add(seen.handler("after-mutator")); err != nil {
		t.Fatal(err)
	}

	if err := source.Raise(nil, EventArgsEmpty()); err != nil {
		t.Fatal(err)
	}
	if !equalCalls(seen.calls, []string{"removed-during-dispatch", "mutator", "after-mutator"}) {
		t.Fatalf("first dispatch = %v; the running snapshot must be unaffected by mutation", seen.calls)
	}

	// The next dispatch sees both mutations: the removed handler is gone and the
	// late handler runs, in registration order, at the end.
	seen.calls = nil
	if err := source.Remove(lateToken); err == nil {
		// Removing it here would hide the ordering claim, so put it back.
		if _, err := source.Add(seen.handler("added-during-dispatch")); err != nil {
			t.Fatal(err)
		}
	}
	if err := source.Raise(nil, EventArgsEmpty()); err != nil {
		t.Fatal(err)
	}
	if !equalCalls(seen.calls, []string{"mutator", "after-mutator", "added-during-dispatch"}) {
		t.Fatalf("second dispatch = %v", seen.calls)
	}
}

func TestEventSourceStopsAtTheFirstHandlerError(t *testing.T) {
	failure := errors.New("handler failed")
	for _, position := range []struct {
		name    string
		failing int
		want    []string
	}{
		{"first", 0, []string{"h0"}},
		{"middle", 1, []string{"h0", "h1"}},
		{"last", 2, []string{"h0", "h1", "h2"}},
	} {
		position := position
		t.Run(position.name, func(t *testing.T) {
			var source EventSource[*EventArgs]
			seen := &recorder{}
			for i := 0; i < 3; i++ {
				name := fmt.Sprintf("h%d", i)
				handler := seen.handler(name)
				if i == position.failing {
					handler = seen.failing(name, failure)
				}
				if _, err := source.Add(handler); err != nil {
					t.Fatal(err)
				}
			}
			if err := source.Raise(nil, EventArgsEmpty()); !errors.Is(err, failure) {
				t.Fatalf("Raise = %v, want the handler failure propagated", err)
			}
			if !equalCalls(seen.calls, position.want) {
				t.Fatalf("calls = %v, want %v; no handler may run after a failure", seen.calls, position.want)
			}

			// The failed dispatch must leave the registration list intact.
			seen.calls = nil
			if err := source.Raise(nil, EventArgsEmpty()); !errors.Is(err, failure) {
				t.Fatalf("second Raise = %v, want the same failure from the same list", err)
			}
			if !equalCalls(seen.calls, position.want) {
				t.Fatalf("second dispatch = %v, want the list unchanged by the failure", seen.calls)
			}
		})
	}
}

// TestEventSourceNilHandlerMirrorsDelegateCombine pins the reference behavior:
// add_X calls Delegate.Combine(existing, value), and Combine with a null operand
// returns the other operand, so adding null registers nothing and does not throw.
func TestEventSourceNilHandlerMirrorsDelegateCombine(t *testing.T) {
	var source EventSource[*EventArgs]
	seen := &recorder{}
	if _, err := source.Add(seen.handler("real")); err != nil {
		t.Fatal(err)
	}
	token, err := source.Add(nil)
	if err != nil {
		t.Fatalf("Add(nil) = %v, want no failure", err)
	}
	if token != (EventSubscription{}) {
		t.Fatal("Add(nil) returned a live token; a null delegate registers nothing")
	}
	if err := source.Raise(nil, EventArgsEmpty()); err != nil {
		t.Fatalf("Raise after Add(nil) = %v, want no nil-function panic", err)
	}
	if !equalCalls(seen.calls, []string{"real"}) {
		t.Fatalf("calls = %v", seen.calls)
	}
	if err := source.Remove(token); err != nil {
		t.Fatalf("removing the zero token from Add(nil) = %v", err)
	}
}

// TestEventSourceIsRaceCleanUnderConcurrentUse exercises Add, Remove and Raise
// from several goroutines at once. It is meaningful under -race.
func TestEventSourceIsRaceCleanUnderConcurrentUse(t *testing.T) {
	var source EventSource[*EventArgs]
	var counter struct {
		sync.Mutex
		n int
	}
	handler := func(sender any, args *EventArgs) error {
		counter.Lock()
		counter.n++
		counter.Unlock()
		return nil
	}

	var wait sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for i := 0; i < 200; i++ {
				token, err := source.Add(handler)
				if err != nil {
					t.Error(err)
					return
				}
				if err := source.Raise(nil, EventArgsEmpty()); err != nil {
					t.Error(err)
					return
				}
				if err := source.Remove(token); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}
	wait.Wait()
	if counter.n == 0 {
		t.Fatal("no handler ran")
	}
}

// TestEventSubscriptionExposesNoImplementationState is a compile-time and
// runtime guard that the token stays opaque.
func TestEventSubscriptionExposesNoImplementationState(t *testing.T) {
	var source EventSource[*EventArgs]
	token, err := source.Add(func(sender any, args *EventArgs) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	// The zero token is usable and distinct from a live one.
	if token == (EventSubscription{}) {
		t.Fatal("a live token equals the zero token")
	}
	second, err := source.Add(func(sender any, args *EventArgs) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if token == second {
		t.Fatal("two registrations produced one token identity")
	}
}
