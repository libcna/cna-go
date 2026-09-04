package audio

import (
	"errors"
	"fmt"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// flipHandedness reproduces the assembly-private
// Microsoft.Xna.Framework.Audio.UnsafeNativeStructures::FlipHandedness. Its IL
// copies X and Y unchanged and stores `neg` of Z:
//
//	ldloca.s V_0; ldarga.s vector; ldfld X; stfld X
//	ldloca.s V_0; ldarga.s vector; ldfld Y; stfld Y
//	ldloca.s V_0; ldarga.s vector; ldfld Z; neg; stfld Z
//
// CIL `neg` and Go's unary minus are both an IEEE-754 sign-bit flip, so this
// function is its own inverse for every binary32 bit pattern: both zeros, both
// infinities, and NaN all round-trip to the exact bits they started with. That
// is what makes every public accessor pair below an identity round-trip even
// though each accessor individually changes handedness.
//
// It is not a coordinate-system helper for callers: XNA keeps it private
// because it converts between the right-handed public XNA convention and the
// left-handed XACT storage convention.
func flipHandedness(vector framework.Vector3) framework.Vector3 {
	return framework.Vector3{X: vector.X, Y: vector.Y, Z: -vector.Z}
}

// argumentOutOfRangeError projects one CLR System.ArgumentOutOfRangeException
// raised by managed argument validation into the established CNA-Go managed
// argument-failure channel. The Go error is a language projection; the XNA
// behavior it stands for is a thrown exception, and the two are deliberately
// recorded as distinct in the behavior evidence.
//
// message is the exact FrameworkResources string the reference throw site
// loads, so the projected text stays traceable to the retained assembly.
func argumentOutOfRangeError(parameter, message string) error {
	return fmt.Errorf("%w: %s: %s", errAudioArgumentOutOfRange, parameter, message)
}

// errAudioArgumentOutOfRange is the sentinel that channel wraps. It was added
// with the other three in Foundation 87: the message text is unchanged, and
// what it gains is that a caller -- and a test -- can tell a RANGE refusal
// apart from the other four kinds the family raises, which matters because
// three of the four static setters refuse a NaN and the fourth stores it.
var errAudioArgumentOutOfRange = errors.New("audio argument out of range")

// invalidEmitterDopplerScale is the exact FrameworkResources value that
// AudioEmitter::set_DopplerScale loads before it throws, read from the
// Microsoft.Xna.Framework.FrameworkResources.resources stream of the retained
// Microsoft.Xna.Framework.dll.
const invalidEmitterDopplerScale = "The doppler scale of an audio emitter must be greater than or equal to zero."

// ---------------------------------------------------------------------------
// Foundation 87 -- the audio family's three managed failure channels.
// ---------------------------------------------------------------------------

// The reference's audio members throw four CLR exception kinds and each one is
// a distinct channel here, because a consumer distinguishes them:
//
//	ArgumentOutOfRangeException  a range, with a PARAMETER NAME and usually no message
//	ArgumentException            a buffer or loop shape, with a MESSAGE and no name
//	ArgumentNullException        a null or EMPTY string argument
//	ObjectDisposedException      GetType().Name plus a fixed sentence
//	InvalidOperationException    the dispatcher precondition and the 3D pan refusal
var (
	errAudioArgument         = errors.New("audio argument is not valid")
	errAudioArgumentNull     = errors.New("audio argument must not be null")
	errAudioObjectDisposed   = errors.New("object has already been disposed")
	errAudioInvalidOperation = errors.New("operation is not valid")
)

// argumentError projects System.ArgumentException(string message), which the
// buffer and loop checks throw with a MESSAGE and NO parameter name -- so the
// message is the only thing a caller gets and it is reproduced verbatim.
func argumentError(message string) error {
	return fmt.Errorf("%w: %s", errAudioArgument, message)
}

// argumentNullError projects System.ArgumentNullException(string paramName),
// which carries a parameter name and no message of its own.
func argumentNullError(parameter string) error {
	return fmt.Errorf("%w: %s", errAudioArgumentNull, parameter)
}
