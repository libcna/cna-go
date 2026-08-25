package audio

import (
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
	return fmt.Errorf("audio argument out of range: %s: %s", parameter, message)
}

// invalidEmitterDopplerScale is the exact FrameworkResources value that
// AudioEmitter::set_DopplerScale loads before it throws, read from the
// Microsoft.Xna.Framework.FrameworkResources.resources stream of the retained
// Microsoft.Xna.Framework.dll.
const invalidEmitterDopplerScale = "The doppler scale of an audio emitter must be greater than or equal to zero."
