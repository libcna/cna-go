package audio

import (
	"errors"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
)

// Microphone is Microsoft.Xna.Framework.Audio.Microphone:
//
//	.class public auto ansi sealed Microphone extends [mscorlib]System.Object
//	  .field assembly initonly uint32 Handle
//	  .field assembly initonly int32  Id
//	  .field public   initonly string Name        <-- a public FIELD
//	  .field private  AudioFormat format
//	  .field private  TimeSpan captureBufferDuration
//	  .field private  bool isHeadset
//	  .field private  EventHandler`1<EventArgs> BufferReady
//	  .field static assembly MicrophoneCollection microphones
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # It owns nothing, and neither does this projection
//
// The reference's constructor is `assembly`, so a consumer never builds one:
// they come from `Microphone.All` or `Microphone.Default`. The type declares no
// disposal at all.
//
// CNA's whole family is INDEX-addressed to match -- every route takes a game
// handle and a position, and there is no owned handle to release. So this type
// holds an index and nothing else that needs freeing, and the object a consumer
// receives stays valid for as long as the game does.
//
// # The capture members are projected and NEVER exercised
//
// `Start` and `GetData` are here because the pinned contract declares them. The
// native scenario calls neither, and the reason is a standing constraint on the
// SUITE rather than a limitation of the projection: recording from a physical
// microphone is not something a test suite does on someone's machine. What the
// scenario does exercise is the whole enumeration and description surface --
// count, default, name, buffer duration, headset flag, sample rate, state and
// the two conversions.
type Microphone struct {
	// index is the position CNA addresses this microphone by. It stands in for
	// the reference's `Handle` and `Id`, both of which are `assembly`.
	index uint64
	// Name is the reference's PUBLIC INITONLY field, projected as an exported
	// Go field because the pinned contract declares it `kind: field` and not a
	// property.
	//
	// Go has no readonly field, so the `initonly` half cannot be projected: a
	// consumer can assign to this where the reference would not compile. That
	// is a LANGUAGE limitation and is recorded rather than worked around --
	// hiding the field behind a getter would have projected a property the
	// contract does not declare, which is the larger divergence.
	//
	// It has a second consequence worth stating: every METHOD on this type
	// answers or refuses through a nil receiver, and reading this FIELD through
	// one panics. A field cannot carry a guard.
	Name string
	// captureBufferDuration is the reference's own field. set_BufferDuration
	// writes native FIRST and stores second, so this holds what is in effect.
	captureBufferDuration framework.TimeSpan
	// bufferReadyEvent is the `BufferReady` delegate field.
	bufferReadyEvent framework.EventSource[*framework.EventArgs]
}

// errMicrophoneNil is the Go-only guard for a zero value.
var errMicrophoneNil = errors.New("Microphone is nil or uninitialized")

// MicrophoneAll is Microphone::get_All, which forwards to the static
// MicrophoneCollection the class initializer builds.
//
// `MicrophoneCollection` is NOT in the pinned contract -- the property's
// declared type is `ReadOnlyCollection<Microphone>` and that is what this
// answers. The collection is built by enumerating CNA's count, which is the
// same thing the reference's collection does at startup.
//
// It is FALLIBLE where the reference's is not: the reference reads a list built
// once, and this reaches CNA for the count and each name.
func MicrophoneAll() (*framework.ReadOnlyCollection[*Microphone], error) {
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil, errSoundEffectNoRunningGame
	}
	count, err := runtime.MicrophoneCount()
	if err != nil {
		return nil, err
	}
	microphones := make([]*Microphone, 0, count)
	for index := uint64(0); index < count; index++ {
		microphone, buildErr := newMicrophone(runtime, index)
		if buildErr != nil {
			return nil, buildErr
		}
		microphones = append(microphones, microphone)
	}
	return framework.NewReadOnlyCollectionOverReferences(microphones), nil
}

// MicrophoneDefault is Microphone::get_Default.
//
// CNA reports whether there IS a default separately from the index -- "left
// unchanged when there is no default" -- and the reference's collection answers
// null in that case. So a machine with no default microphone gets a nil
// Microphone and no error, which is what `null` means there.
func MicrophoneDefault() (*Microphone, error) {
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil, errSoundEffectNoRunningGame
	}
	index, available, err := runtime.MicrophoneDefaultIndex()
	if err != nil {
		return nil, err
	}
	if !available {
		return nil, nil
	}
	return newMicrophone(runtime, index)
}

// newMicrophone reads the two values the reference's constructor stores once:
// the name, which is a public initonly field there, and the capture buffer
// duration, which its constructor seeds with TimeSpan.Zero and the device then
// reports.
func newMicrophone(runtime *interop.Runtime, index uint64) (*Microphone, error) {
	name, err := runtime.MicrophoneName(index)
	if err != nil {
		return nil, err
	}
	ticks, err := runtime.MicrophoneBufferDurationTicks(index)
	if err != nil {
		return nil, err
	}
	return &Microphone{
		index:                 index,
		Name:                  name,
		captureBufferDuration: framework.TimeSpanFromTicks(ticks),
	}, nil
}

// BufferDuration is Microphone::get_BufferDuration, one `ldfld` over the field
// set_BufferDuration maintains.
//
// It is INFALLIBLE, and registered as a managed stored member to say so: the
// reference's body reaches nothing, and a nil receiver answers the zero
// TimeSpan rather than an error -- the same treatment every other field read in
// this family gets.
func (m *Microphone) BufferDuration() framework.TimeSpan {
	if m == nil {
		return framework.TimeSpan{}
	}
	return m.captureBufferDuration
}

// Finalize is Microphone::Finalize, the protected finalizer. The reference's
// body stops any capture the object started; nothing calls it here, for the
// reason every projected finalizer in this module is uncalled -- Go has no CLR
// finalization and CNA-Go registers no runtime finalizer.
func (m *Microphone) Finalize() error {
	if m == nil {
		return errMicrophoneNil
	}
	return m.Stop()
}

// SetBufferDuration is Microphone::set_BufferDuration, whose one guard has
// THREE parts and a single message:
//
//	if (value.TotalMilliseconds < 100 || value.TotalMilliseconds > 1000 ||
//	    value.TotalMilliseconds % 10 != 0)
//	    throw new ArgumentOutOfRangeException("value",
//	        FrameworkResources.InvalidMicrophoneBufferDuration);
//	SetCaptureBufferDuration(Handle, (int)value.TotalMilliseconds);   // native FIRST
//	captureBufferDuration = value;                                   // store SECOND
//
// 100ms to 1s INCLUSIVE, and a multiple of 10ms. The message says exactly that,
// double space and all: "Microphone buffer duration must be between 100ms and
// 1sec and  10ms aligned."
//
// The native call runs before the store, so a refused set leaves the getter
// reporting what is actually in effect -- the same order set_Volume has and the
// opposite of set_SpeedOfSound's.
func (m *Microphone) SetBufferDuration(value framework.TimeSpan) error {
	if m == nil {
		return errMicrophoneNil
	}
	milliseconds := value.TotalMilliseconds()
	if milliseconds < 100 || milliseconds > 1000 || mathMod(milliseconds, 10) != 0 {
		return argumentOutOfRangeError("value", invalidMicrophoneBufferDuration)
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return errSoundEffectNoRunningGame
	}
	if err := runtime.MicrophoneSetBufferDurationTicks(m.index, value.Ticks()); err != nil {
		return err
	}
	m.captureBufferDuration = value
	return nil
}

// SampleRate is Microphone::get_SampleRate, which the reference reads from its
// AudioFormat -- a value the device reported when the collection was built. CNA
// answers it per index, so this is a live read.
func (m *Microphone) SampleRate() (int32, error) {
	if m == nil {
		return 0, errMicrophoneNil
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return 0, errSoundEffectNoRunningGame
	}
	return runtime.MicrophoneSampleRate(m.index)
}

// IsHeadset is Microphone::get_IsHeadset.
func (m *Microphone) IsHeadset() (bool, error) {
	if m == nil {
		return false, errMicrophoneNil
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return false, errSoundEffectNoRunningGame
	}
	return runtime.MicrophoneIsHeadset(m.index)
}

// State is Microphone::get_State, a live read of whether the device is
// capturing.
func (m *Microphone) State() (MicrophoneState, error) {
	if m == nil {
		return MicrophoneStateStopped, errMicrophoneNil
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return MicrophoneStateStopped, errSoundEffectNoRunningGame
	}
	state, err := runtime.MicrophoneState(m.index)
	if err != nil {
		return MicrophoneStateStopped, err
	}
	return microphoneStateFromNative(state), nil
}

// The two CNA_MICROPHONE_STATE_* identities. They match XNA's MicrophoneState
// literals and are mapped explicitly anyway, for the reason every other enum in
// this project is: a shared numbering is a coincidence to be checked.
const (
	nativeMicrophoneStateStarted uint32 = 0
	nativeMicrophoneStateStopped uint32 = 1
)

// microphoneStateFromNative maps CNA's answer, with STOPPED as the fallback for
// anything the enum does not name.
func microphoneStateFromNative(state uint32) MicrophoneState {
	if state == nativeMicrophoneStateStarted {
		return MicrophoneStateStarted
	}
	return MicrophoneStateStopped
}

// Start is Microphone::Start.
//
// # It is projected and the suite never calls it
//
// Starting capture opens a real recording device on whatever machine the code
// runs on. The native scenario exercises the whole enumeration and description
// surface and stops here, deliberately: what a projected member DOES is the
// contract's business, and whether a test suite performs it on someone's
// machine is not the same question.
//
// A consumer who calls it gets exactly what the reference gives them.
func (m *Microphone) Start() error {
	if m == nil {
		return errMicrophoneNil
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return errSoundEffectNoRunningGame
	}
	return runtime.MicrophoneStart(m.index)
}

// Stop is Microphone::Stop. Unlike Start it is safe to call on a device that is
// not capturing, which is why the scenario does exercise it.
func (m *Microphone) Stop() error {
	if m == nil {
		return errMicrophoneNil
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return errSoundEffectNoRunningGame
	}
	return runtime.MicrophoneStop(m.index)
}

// GetDataBySliceOfByte is Microphone::GetData(Byte[]), which forwards with the
// whole buffer.
func (m *Microphone) GetDataBySliceOfByte(buffer []uint8) (int32, error) {
	return m.GetDataBySliceOfByteAndInt32AndInt32(buffer, 0, int32(len(buffer)))
}

// GetDataBySliceOfByteAndInt32AndInt32 is
// Microphone::GetData(Byte[], Int32, Int32), the member that reads captured
// audio.
//
// It is projected because the contract declares it, and the native scenario
// never calls it: see Start.
func (m *Microphone) GetDataBySliceOfByteAndInt32AndInt32(buffer []uint8, offset, count int32) (int32, error) {
	if m == nil {
		return 0, errMicrophoneNil
	}
	if len(buffer) == 0 {
		return 0, argumentError(invalidAudioBuffer)
	}
	if offset < 0 || offset >= int32(len(buffer)) {
		return 0, argumentError(invalidAudioBufferOffset)
	}
	if count <= 0 || int64(offset)+int64(count) > int64(len(buffer)) {
		return 0, argumentError(invalidOffsetCountLength)
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return 0, errSoundEffectNoRunningGame
	}
	written, err := runtime.MicrophoneGetData(m.index, buffer[offset:offset+count])
	if err != nil {
		return 0, err
	}
	return int32(written), nil
}

// GetSampleSizeInBytes is Microphone::GetSampleSizeInBytes(TimeSpan).
func (m *Microphone) GetSampleSizeInBytes(duration framework.TimeSpan) (int32, error) {
	if m == nil {
		return 0, errMicrophoneNil
	}
	total := duration.TotalMilliseconds()
	if !(total >= 0) || !(total <= 2147483647) {
		return 0, argumentOutOfRangeError("duration", "")
	}
	rate, err := m.SampleRate()
	if err != nil {
		return 0, err
	}
	// The reference builds `AudioFormat.Create(GetSampleRate(), 1, 16)` in its
	// constructor -- capture is MONO -- and this member reads that format. So
	// the arithmetic is managed, for the reason DynamicSoundEffectInstance's is:
	// CNA's own conversion does not reproduce XNA's float32 scale factor.
	size, err := sizeFromDuration(duration, rate, int32(AudioChannelsMono))
	if err != nil {
		return 0, argumentOutOfRangeError("duration", "")
	}
	return size, nil
}

// GetSampleDuration is Microphone::GetSampleDuration(Int32), and its guard is
// the family's one KIND surprise:
//
//	if (sizeInBytes < 0)
//	    throw new ArgumentException(FrameworkResources.InvalidBufferSize);
//
// A plain ArgumentException with a message and no parameter name -- where
// SoundEffect's near-identical static sibling throws
// ArgumentOutOfRangeException("sizeInBytes", InvalidBufferSize). Two members
// that read the same, one message, two exception kinds.
func (m *Microphone) GetSampleDuration(sizeInBytes int32) (framework.TimeSpan, error) {
	if m == nil {
		return framework.TimeSpan{}, errMicrophoneNil
	}
	if sizeInBytes < 0 {
		return framework.TimeSpan{}, argumentError(invalidBufferSize)
	}
	rate, err := m.SampleRate()
	if err != nil {
		return framework.TimeSpan{}, err
	}
	return durationFromSize(sizeInBytes, rate, int32(AudioChannelsMono))
}

// AddBufferReadyHandler is add_BufferReady, on the settled two-accessor event
// projection.
//
// The reference raises it from the capture callback when a buffer has filled.
// CNA has the counterpart -- `cna_microphone_subscribe_buffer_ready_at` -- and
// it is recorded unbound: the event cannot fire without capture running, and
// the suite does not start capture.
func (m *Microphone) AddBufferReadyHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error) {
	if m == nil {
		return framework.EventSubscription{}, errMicrophoneNil
	}
	return m.bufferReadyEvent.Add(handler)
}

// RemoveBufferReadyHandler is remove_BufferReady.
func (m *Microphone) RemoveBufferReadyHandler(subscription framework.EventSubscription) error {
	if m == nil {
		return errMicrophoneNil
	}
	return m.bufferReadyEvent.Remove(subscription)
}

// mathMod is the float64 remainder the reference's `rem` opcode computes. It is
// spelled out because Go's % does not accept a float and math.Mod's sign rule
// for a negative left operand is the same as CIL's -- both truncate toward
// zero -- which is what makes the substitution exact rather than close.
func mathMod(value, divisor float64) float64 {
	return value - divisor*float64(int64(value/divisor))
}
