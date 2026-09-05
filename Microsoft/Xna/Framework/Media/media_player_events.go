package media

import (
	"sync"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// The two MediaPlayer events, and the one native subscription behind each.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # Both events are static, so their registrations are process-wide
//
// XNA declares ActiveSongChanged and MediaStateChanged as `public static
// event EventHandler<EventArgs>`. There is no MediaPlayer INSTANCE to hang a
// registration list on, so the lists live here -- one per event, guarded by one
// mutex, exactly as the reference's own static delegate fields are.
//
// # The native subscription is made ONCE and shared
//
// CNA's two subscribe routes each answer a registration handle, and each raises
// through a trampoline that carries only a context. So the projection
// subscribes the first time a handler is added to an event and keeps that
// registration for as long as any handler remains -- adding a second handler
// does not subscribe twice, and removing the last one releases it.
//
// That is not the reference's shape, which has no native half at all; it is the
// shape the native half forces, and the observable behaviour is the same: a
// handler added is called, a handler removed is not.

const (
	mediaPlayerActiveSongChanged = 0
	mediaPlayerMediaStateChanged = 1
)

var mediaPlayerEvents struct {
	mu     sync.Mutex
	source [2]framework.EventSource[*framework.EventArgs]
	// registration is the native handle for each event, zero when nothing is
	// subscribed.
	registration [2]uint64
	// handlers counts what each event holds, because EventSource does not
	// expose a count and the native registration has to be released when the
	// LAST one goes.
	handlers [2]int
	// installed says whether the runtime's one media-event handler has been
	// pointed at this dispatcher yet.
	installed bool
}

// mediaPlayerAddHandler is the shared add accessor.
func mediaPlayerAddHandler(event int,
	handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	// The ARGUMENT first, for the reason every other member in this family
	// checks it first: a nil handler is a programming error either way.
	if handler == nil {
		return framework.EventSubscription{}, mediaArgumentNullError("handler")
	}
	runtime, err := mediaPlayerRuntime()
	if err != nil {
		return framework.EventSubscription{}, err
	}
	mediaPlayerEvents.mu.Lock()
	defer mediaPlayerEvents.mu.Unlock()
	if !mediaPlayerEvents.installed {
		runtime.SetMediaPlayerEventHandler(dispatchMediaPlayerEvent)
		mediaPlayerEvents.installed = true
	}
	if mediaPlayerEvents.registration[event] == 0 {
		registration, err := subscribeMediaPlayerEvent(runtime, event)
		if err != nil {
			return framework.EventSubscription{}, err
		}
		mediaPlayerEvents.registration[event] = registration
	}
	subscription, err := mediaPlayerEvents.source[event].Add(handler)
	if err != nil {
		return framework.EventSubscription{}, err
	}
	mediaPlayerEvents.handlers[event]++
	return subscription, nil
}

// mediaPlayerRemoveHandler is the shared remove accessor. It releases the
// native registration when the LAST handler goes, so a program that subscribes
// and unsubscribes repeatedly does not accumulate registrations.
func mediaPlayerRemoveHandler(event int, subscription framework.EventSubscription) error {
	runtime, err := mediaPlayerRuntime()
	if err != nil {
		return err
	}
	mediaPlayerEvents.mu.Lock()
	defer mediaPlayerEvents.mu.Unlock()
	if err := mediaPlayerEvents.source[event].Remove(subscription); err != nil {
		return err
	}
	if mediaPlayerEvents.handlers[event] > 0 {
		mediaPlayerEvents.handlers[event]--
	}
	if mediaPlayerEvents.handlers[event] != 0 {
		return nil
	}
	registration := mediaPlayerEvents.registration[event]
	if registration == 0 {
		return nil
	}
	mediaPlayerEvents.registration[event] = 0
	return runtime.MediaPlayerUnsubscribe(registration)
}

func subscribeMediaPlayerEvent(runtime *interop.Runtime, event int) (uint64, error) {
	if event == mediaPlayerActiveSongChanged {
		return runtime.MediaPlayerSubscribeActiveSongChanged()
	}
	return runtime.MediaPlayerSubscribeMediaStateChanged()
}

// dispatchMediaPlayerEvent is what the native trampoline reaches. It raises
// with EventArgs.Empty and a NIL sender, which is what a static CLR event
// raises with: there is no instance to be the sender.
func dispatchMediaPlayerEvent(event uint32) {
	if int(event) >= len(mediaPlayerEvents.source) {
		return
	}
	mediaPlayerEvents.mu.Lock()
	source := &mediaPlayerEvents.source[event]
	mediaPlayerEvents.mu.Unlock()
	source.Raise(nil, framework.EventArgsEmpty())
}
