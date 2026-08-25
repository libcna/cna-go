package audio

import framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"

// AudioEmitter is the pure managed positional descriptor of a 3D audio
// emitter. Like AudioListener it is a CLR class and keeps CLR reference
// semantics, and it owns no native object: its whole public surface is backed
// by the managed members of one assembly-visible `XACT_EMITTER_DATA
// emitterData` field.
//
// XACT_EMITTER_DATA also carries cone, curve, and channel members that the XNA
// 4.0 Windows public contract does not expose. The constructor initializes the
// three of them it touches -- ChannelCount 1, ChannelRadius 1,
// CurveDistanceScaler 1 -- and the unexported mirrors below record that, but
// no public member reads or writes them, and none of the raw pointer members
// has any managed meaning to reproduce.
//
// The four stored vectors hold XACT's left-handed convention; DopplerScale is
// stored verbatim with no conversion.
type AudioEmitter struct {
	position framework.Vector3
	velocity framework.Vector3
	forward  framework.Vector3
	up       framework.Vector3

	dopplerScale float32

	// Constructor-initialized emitterData members with no public projection
	// in the XNA 4.0 Windows profile. They are kept so this type's managed
	// state matches the reference field for field.
	channelCount        uint32
	channelRadius       float32
	curveDistanceScaler float32
}

// NewAudioEmitter reproduces AudioEmitter::.ctor. Like the listener it stores
// Vector3.Zero unflipped into _Position and _Velocity while storing
// FlipHandedness of Vector3.Forward and Vector3.Up, so Position and Velocity
// read back as (0, 0, -0) while Forward and Up read back bit-exactly as
// Vector3.Forward and Vector3.Up. _DopplerScale, ChannelCount, ChannelRadius,
// and CurveDistanceScaler all start at 1.
func NewAudioEmitter() *AudioEmitter {
	return &AudioEmitter{
		position:            framework.Vector3Zero(),
		velocity:            framework.Vector3Zero(),
		forward:             flipHandedness(framework.Vector3Forward()),
		up:                  flipHandedness(framework.Vector3Up()),
		dopplerScale:        1,
		channelCount:        1,
		channelRadius:       1,
		curveDistanceScaler: 1,
	}
}

// Position is AudioEmitter::get_Position: one ldfld through flipHandedness.
func (e *AudioEmitter) Position() framework.Vector3 {
	return flipHandedness(e.position)
}

// SetPosition is AudioEmitter::set_Position. It validates nothing.
func (e *AudioEmitter) SetPosition(value framework.Vector3) {
	e.position = flipHandedness(value)
}

// Velocity is AudioEmitter::get_Velocity.
func (e *AudioEmitter) Velocity() framework.Vector3 {
	return flipHandedness(e.velocity)
}

// SetVelocity is AudioEmitter::set_Velocity. It validates nothing.
func (e *AudioEmitter) SetVelocity(value framework.Vector3) {
	e.velocity = flipHandedness(value)
}

// Forward is AudioEmitter::get_Forward.
func (e *AudioEmitter) Forward() framework.Vector3 {
	return flipHandedness(e.forward)
}

// SetForward is AudioEmitter::set_Forward. It validates nothing.
func (e *AudioEmitter) SetForward(value framework.Vector3) {
	e.forward = flipHandedness(value)
}

// Up is AudioEmitter::get_Up.
func (e *AudioEmitter) Up() framework.Vector3 {
	return flipHandedness(e.up)
}

// SetUp is AudioEmitter::set_Up. It validates nothing.
func (e *AudioEmitter) SetUp(value framework.Vector3) {
	e.up = flipHandedness(value)
}

// DopplerScale is AudioEmitter::get_DopplerScale. Its IL is one ldflda plus
// one ldfld with no validation at all, so the getter is infallible and returns
// the stored bits verbatim -- including a negative zero or a NaN that a
// previous successful assignment stored.
func (e *AudioEmitter) DopplerScale() float32 {
	return e.dopplerScale
}

// SetDopplerScale is AudioEmitter::set_DopplerScale, the one fallible member
// of either audio descriptor. Its IL guards the store with
//
//	IL_0000: ldarg.1
//	IL_0001: ldc.r4 0.0
//	IL_0006: bge.un.s IL_0018
//	... newobj ArgumentOutOfRangeException; throw
//	IL_0018: ... stfld _DopplerScale
//
// `bge.un` branches to the store when value >= 0 *or* when the comparison is
// unordered, so the throw is reached on exactly the negative-ordered values.
// `value < 0` in Go is the exact complement of that branch:
//
//	accepted: positive finite, +0, -0, +Infinity, and every NaN
//	rejected: negative finite, negative denormal, and -Infinity
//
// NaN is accepted rather than rejected, which reads like an oversight but is
// the reference behavior and is preserved. Its getter is unaffected: only this
// setter carries the error result.
func (e *AudioEmitter) SetDopplerScale(value float32) error {
	if value < 0 {
		return argumentOutOfRangeError("value", invalidEmitterDopplerScale)
	}
	e.dopplerScale = value
	return nil
}
