package audio

import framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"

// AudioListener is the pure managed positional descriptor of the 3D audio
// listener. It is a CLR class, so it keeps CLR reference semantics: two Go
// variables holding the same *AudioListener observe each other's mutations.
//
// It owns no native object. Microsoft.Xna.Framework.dll declares exactly one
// assembly-visible field, `XACT_LISTENER_DATA listenerData`, whose four
// managed Vector3 members back the whole public surface; its fifth field
// `pCone` is private, never written by any public member, and has no public
// projection. Every public accessor is one ldfld/stfld plus flipHandedness, so
// nothing here reaches XACT, a device, or an allocation, and no member gains a
// Go error result.
//
// The four stored vectors below hold XACT's left-handed convention, exactly as
// the reference field does. The public accessors convert on the way in and on
// the way out.
type AudioListener struct {
	position framework.Vector3
	velocity framework.Vector3
	forward  framework.Vector3
	up       framework.Vector3
}

// NewAudioListener reproduces AudioListener::.ctor, which stores Vector3.Zero
// into _Position and _Velocity directly but stores FlipHandedness of
// Vector3.Forward and Vector3.Up.
//
// Because the constructor does not flip the two zero vectors while the getters
// do, Position and Velocity read back as (0, 0, -0): the exact Z bits are
// 0x80000000, not 0x00000000. Forward and Up read back bit-exactly as
// Vector3.Forward and Vector3.Up because their stored values were flipped
// once already. This asymmetry is reference behavior and is preserved.
func NewAudioListener() *AudioListener {
	return &AudioListener{
		position: framework.Vector3Zero(),
		velocity: framework.Vector3Zero(),
		forward:  flipHandedness(framework.Vector3Forward()),
		up:       flipHandedness(framework.Vector3Up()),
	}
}

// Position is AudioListener::get_Position: one ldfld through flipHandedness.
func (l *AudioListener) Position() framework.Vector3 {
	return flipHandedness(l.position)
}

// SetPosition is AudioListener::set_Position. It validates nothing.
func (l *AudioListener) SetPosition(value framework.Vector3) {
	l.position = flipHandedness(value)
}

// Velocity is AudioListener::get_Velocity.
func (l *AudioListener) Velocity() framework.Vector3 {
	return flipHandedness(l.velocity)
}

// SetVelocity is AudioListener::set_Velocity. It validates nothing.
func (l *AudioListener) SetVelocity(value framework.Vector3) {
	l.velocity = flipHandedness(value)
}

// Forward is AudioListener::get_Forward. The reference never normalizes or
// orthogonalizes the stored orientation on read.
func (l *AudioListener) Forward() framework.Vector3 {
	return flipHandedness(l.forward)
}

// SetForward is AudioListener::set_Forward. It validates nothing: the
// reference accepts a zero, denormal, infinite, or NaN orientation unchanged.
func (l *AudioListener) SetForward(value framework.Vector3) {
	l.forward = flipHandedness(value)
}

// Up is AudioListener::get_Up.
func (l *AudioListener) Up() framework.Vector3 {
	return flipHandedness(l.up)
}

// SetUp is AudioListener::set_Up. It validates nothing.
func (l *AudioListener) SetUp(value framework.Vector3) {
	l.up = flipHandedness(value)
}
