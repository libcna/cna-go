package audio

// The FrameworkResources strings the audio family's throw sites load, byte for
// byte from the retained Microsoft.Xna.Framework.dll.
//
// Every one is verified against the assembly's `.resources` stream by
// tools/resource_strings, which is what keeps them traceable rather than
// remembered. Four of them belong to ONE method -- SoundEffect::FromBuffer --
// and each names a different way the buffer arguments can be wrong; the
// reference throws plain ArgumentException with a MESSAGE and no parameter name
// for all four, so the sentence is the only thing a caller gets.
const (
	invalidAudioBuffer       = "Ensure that the buffer length is non-zero and meets the block alignment requirements for the audio format."
	invalidAudioBufferOffset = "Offset must be within the buffer boundaries and meet the block alignment requirements for the audio format."
	invalidOffsetCountLength = "Ensure that count is valid and meets the block alignment requirements for the audio format. Offset and count must define a valid region within the buffer boundaries."
	invalidLoopRegion        = "Ensure that the loop region is defined in samples and within the buffer boundaries."
	invalidBufferSize        = "Buffer size cannot be negative."
	invalidPanCall           = "Pan cannot be set on a 3D sound. To ensure a 2D sound avoid calling Apply3D and ensure Pan is set before the first Play call."

	// invalidIsLoopedCall completes the set of three "before the first Play"
	// refusals. Looping, panning and 3D positioning are all decided before the
	// packet is submitted and fixed afterwards, and each says so in its own
	// words.
	invalidIsLoopedCall = "Loop must be set before the first Play call."

	// invalidApply3DCall is invalidPanCall's mirror: the two members are the
	// two halves of one mode latch, and each refuses when the other has won.
	invalidApply3DCall = "The sound is not a 3D sound. Call Apply3D before the first Play call to configure it to be a 3D sound."

	// callFrameworkDispatcherUpdate is the one a consumer is most likely to
	// meet: SoundEffect::Play refuses until FrameworkDispatcher.Update has run
	// at least once, and the message says why in the reference's own words.
	callFrameworkDispatcherUpdate = "FrameworkDispatcher.Update has not been called. Regular FrameworkDispatcher.Update calls are necessary for fire and forget sound effects and framework events to function correctly. See http://go.microsoft.com/fwlink/?LinkId=193853 for details."

	// objectDisposedMessage is the MESSAGE half of
	// ObjectDisposedException(objectName, message). The first argument is
	// GetType().Name, so the sentence is fixed and the type name is the
	// object's own.
	objectDisposedMessage = "This object has already been disposed."
)
