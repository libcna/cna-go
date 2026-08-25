package framework

import "sync"

// This file is CNA-Go language support, not XNA surface. Nothing declared here
// is a projected XNA type or a projected XNA member: the four names are
// declared language adapters in tools/api_compat/mapping-rules.json and are
// measured as adapters rather than counted as XNA identities.
//
// They exist because two BCL shapes appear all over the XNA public contract
// and neither has a faithful builtin Go spelling:
//
//	System.EventArgs         -> *EventArgs
//	System.EventHandler<T>   -> EventHandler[T]
//
// Every one of the 49 public CLR events in the pinned XNA 4.0 Windows profile
// is declared as System.EventHandler`1<T>, and 44 of them use
// System.EventHandler`1<System.EventArgs>, so these two adapters cover the
// whole event surface.

// EventArgs is the CNA-Go projection of System.EventArgs.
//
// XNA never carries information in a bare System.EventArgs. Every raise site in
// the retained assemblies pushes the one shared System.EventArgs::Empty static
// field, for example in Microsoft.Xna.Framework.Game.dll where
// GameComponent::set_Enabled runs
//
//	ldsfld [mscorlib]System.EventArgs [mscorlib]System.EventArgs::Empty
//	callvirt instance void GameComponent::OnEnabledChanged(object, EventArgs)
//
// so the projection is a reference identity with no exported state, and
// EventArgsEmpty is the one instance CNA-Go ever hands out.
//
// The single unexported field is a Go language necessity, not invented CLR
// state. The Go specification allows two distinct zero-size variables to share
// an address, and the gc runtime does exactly that: four separate
// new(struct{}) allocations measured on go1.24.4 linux/amd64 all returned
// runtime.zerobase, so a zero-size EventArgs would make every instance
// pointer-equal to every other and destroy the reference identity that
// System.EventArgs.Empty depends on. One unexported byte restores distinct
// addresses without adding any observable state.
type EventArgs struct {
	_ byte
}

// eventArgsEmpty is the single shared instance modelling the CLR
// System.EventArgs::Empty static field. It is a package-level value rather than
// an exported variable so no consumer can reassign the shared identity.
var eventArgsEmpty = &EventArgs{}

// EventArgsEmpty returns the shared instance that projects
// System.EventArgs.Empty. Repeated calls return the same pointer, so a handler
// may compare the args it receives against EventArgsEmpty by identity exactly
// as CLR code compares against EventArgs.Empty.
//
// EventArgsEmpty is deliberately a function rather than an exported variable:
// an exported variable would let one consumer replace the shared identity for
// every other consumer in the process.
func EventArgsEmpty() *EventArgs {
	return eventArgsEmpty
}

// EventHandler is the CNA-Go projection of System.EventHandler<T>.
//
// The CLR delegate returns void, but invoking it can throw, and CNA-Go reports
// runtime failure through an error result rather than a panic. The error result
// is therefore a Go language projection of the CLR exception channel, not an
// extra XNA return identity: no XNA event handler produces a value.
//
// sender is the raising instance, matching the CLR delegate's object parameter,
// and is nil only when the raiser passes nil.
type EventHandler[T any] func(sender any, args T) error

// EventSubscription is the opaque token one Add returns.
//
// It exists because Go function values are not comparable, so the CLR's
// Delegate.Remove identity match cannot be reproduced: XNA's compiler-generated
// remove_X accessors call
//
//	Delegate.Remove(existing, value)
//
// which finds the last entry equal to the delegate being removed. CNA-Go names
// the registration itself instead of the handler, which is why adding the same
// Go func value twice produces two independent registrations and two distinct
// tokens rather than one merged entry. That difference is a deliberate Go
// language projection and is documented as such.
//
// The token carries no native handle, no cgo.Handle, no unsafe.Pointer and no
// exported field, so it cannot be forged, mutated or used to reach any
// implementation state. The zero EventSubscription is valid, names no
// registration, and is ignored by every Remove.
//
// A token does not unsubscribe when it becomes unreachable. There is no
// finalizer: removal is explicit, exactly as a CLR event keeps its delegate
// until remove_X runs.
type EventSubscription struct {
	registration *eventRegistration
}

// eventRegistration is one registration's private identity. Each Add allocates
// a fresh one, so identity is per registration rather than per handler value.
// owner is the *EventSource[T] that created it and is compared only for
// identity, which is what makes a token from another event harmless.
type eventRegistration struct {
	owner any
}

// EventSource owns one CLR event's registration list.
//
// It is public language support so that a type declared outside CNA-Go can
// implement an event-bearing XNA contract such as IUpdateable or IDrawable
// without inventing its own incompatible EventSubscription. A declaring type
// keeps an unexported EventSource field and delegates its two projected event
// accessors to Add and Remove; only the declaring type can reach Raise.
//
// The zero value is ready to use. EventSource contains a mutex, so it must not
// be copied after first use; hold it as a field and use it through a pointer.
type EventSource[T any] struct {
	mu      sync.Mutex
	entries []eventEntry[T]
}

type eventEntry[T any] struct {
	registration *eventRegistration
	handler      EventHandler[T]
}

// Add registers handler and returns the token that removes exactly that
// registration. Registrations are kept in registration order and duplicates are
// allowed, so adding the same handler twice registers it twice and each
// registration is removed separately.
//
// A nil handler registers nothing and returns the zero token, mirroring the
// reference accessors: add_X calls Delegate.Combine(existing, value), and
// Delegate.Combine with a null operand returns the other operand unchanged, so
// adding null to a CLR event is a no-op that does not throw. Removing the
// returned zero token is likewise harmless.
//
// The error result belongs to the settled event accessor projection. Add itself
// never fails.
func (s *EventSource[T]) Add(handler EventHandler[T]) (EventSubscription, error) {
	if handler == nil {
		return EventSubscription{}, nil
	}
	registration := &eventRegistration{owner: s}
	s.mu.Lock()
	s.entries = append(s.entries, eventEntry[T]{registration: registration, handler: handler})
	s.mu.Unlock()
	return EventSubscription{registration: registration}, nil
}

// Remove removes the single registration the token names and leaves every other
// registration in place, including a duplicate registration of the same
// handler.
//
// Remove is harmless in every case where the token names nothing this event
// owns: the zero token, a token already removed, and a token belonging to
// another EventSource all leave this event's registration list untouched. That
// mirrors Delegate.Remove, which returns the original invocation list unchanged
// when the delegate it is asked to remove is absent or null.
//
// The error result belongs to the settled event accessor projection. Remove
// itself never fails.
func (s *EventSource[T]) Remove(subscription EventSubscription) error {
	registration := subscription.registration
	if registration == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if registration.owner != any(s) {
		return nil
	}
	for i, entry := range s.entries {
		if entry.registration != registration {
			continue
		}
		next := make([]eventEntry[T], 0, len(s.entries)-1)
		next = append(next, s.entries[:i]...)
		next = append(next, s.entries[i+1:]...)
		s.entries = next
		return nil
	}
	return nil
}

// Raise invokes every registered handler in registration order and reports the
// first handler failure.
//
// Dispatch runs over a snapshot taken under the lock, so a handler that adds or
// removes a registration changes what a later Raise sees and never what the
// running Raise sees. The lock is released before any handler runs, so a
// handler may call Add, Remove or Raise on the same event without deadlocking.
//
// When a handler returns a non-nil error, Raise stops and returns that error;
// later handlers are not invoked. The registration list is not modified by a
// failed dispatch, so the same handlers remain registered for the next Raise. No
// handler failure is ever discarded.
func (s *EventSource[T]) Raise(sender any, args T) error {
	s.mu.Lock()
	snapshot := make([]EventHandler[T], len(s.entries))
	for i, entry := range s.entries {
		snapshot[i] = entry.handler
	}
	s.mu.Unlock()
	for _, handler := range snapshot {
		if err := handler(sender, args); err != nil {
			return err
		}
	}
	return nil
}
