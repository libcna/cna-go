package audio

import (
	"reflect"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/bclexception"
)

// This file projects the three XNA exception types declared in
// Microsoft.Xna.Framework.Audio.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// Every one of them declares ONLY constructors: the whole useful surface is
// inherited from System.Exception -- or, for an ExternalException subclass, from
// System.Exception through it. So each type below is the settled BCL base
// composition: an unexported `base` field holding the internal/bclexception
// adapter, the derived constructors that install the CLR `this` and the derived
// class name, and the inherited public members re-exposed through measured
// forwarding.
//
// The class name matters. Exception::get_Message renders
// "Exception of type '{0}' was thrown." from GetClassName() when no message
// was given, so a default-constructed DeviceLostException names ITSELF, not
// System.Exception. A composed base cannot see its deriver, which is why the
// constructor supplies the name.
//
// # What ExternalException adds
//
// ExternalException derives from System.SystemException, which adds no public
// surface of its own, and contributes exactly two things: a public ErrorCode
// over the protected HResult, and a ToString OVERRIDE that renders
//
//	GetType() + " (0x" + HResult:X8 + ")" [+ ": " + Message] [+ " ---> " + Inner]
//
// -- with the HResult in parentheses and WITHOUT the end-of-inner-exception
// marker the base form appends. Its three constructors all assign E_FAIL
// through SetErrorCode; the errorCode overload that would assign another is not
// declared by any XNA subclass.

// InstancePlayLimitException is the XNA exception type
// Microsoft.Xna.Framework.Audio.InstancePlayLimitException.
type InstancePlayLimitException struct {
	base bclexception.State
}

// NewInstancePlayLimitExceptionByNone is InstancePlayLimitException::.ctor(), which is
// `base..ctor(SystemException::.ctor(Resources.Arg_ExternalException)` and `SetErrorCode(E_FAIL))`. The message field stays null, so Message renders the
// default sentence naming this class.
func NewInstancePlayLimitExceptionByNone() *InstancePlayLimitException {
	exception := &InstancePlayLimitException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Audio.InstancePlayLimitException", bclexception.ArgExternalException, true, nil, bclexception.EFail, true)
	return exception
}

// NewInstancePlayLimitExceptionByString is InstancePlayLimitException::.ctor(string message).
func NewInstancePlayLimitExceptionByString(message string) *InstancePlayLimitException {
	exception := &InstancePlayLimitException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Audio.InstancePlayLimitException", message, true, nil, bclexception.EFail, true)
	return exception
}

// NewInstancePlayLimitExceptionByStringAndException is
// InstancePlayLimitException::.ctor(string message, Exception inner).
func NewInstancePlayLimitExceptionByStringAndException(message string, inner framework.ExceptionReference) *InstancePlayLimitException {
	exception := &InstancePlayLimitException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Audio.InstancePlayLimitException", message, true, inner, bclexception.EFail, true)
	return exception
}

// The inherited public members, forwarded to the composed base.

func (e *InstancePlayLimitException) Message() string { return e.base.Message() }
func (e *InstancePlayLimitException) InnerException() framework.ExceptionReference {
	return e.base.InnerException()
}
func (e *InstancePlayLimitException) GetBaseException() framework.ExceptionReference {
	return e.base.GetBaseException(e)
}
func (e *InstancePlayLimitException) StackTrace() string       { return e.base.StackTrace() }
func (e *InstancePlayLimitException) HelpLink() string         { return e.base.HelpLink() }
func (e *InstancePlayLimitException) SetHelpLink(value string) { e.base.SetHelpLink(value) }
func (e *InstancePlayLimitException) Source() string           { return e.base.Source() }
func (e *InstancePlayLimitException) SetSource(value string)   { e.base.SetSource(value) }
func (e *InstancePlayLimitException) ToString() string         { return e.base.ToString() }
func (e *InstancePlayLimitException) GetType() reflect.Type    { return e.base.GetType() }

// ErrorCode is ExternalException::get_ErrorCode, one forwarded
// Exception::get_HResult. Every XNA subclass reaches it with E_FAIL, because
// the only constructor that assigns another is the one none of them declares.
func (e *InstancePlayLimitException) ErrorCode() int32 { return e.base.HResult() }

// State is the language accessor the reference interface requires. Its result
// type is declared in an internal package, so nothing outside this module can
// supply one.
func (e *InstancePlayLimitException) State() *bclexception.State { return &e.base }

// NoAudioHardwareException is the XNA exception type
// Microsoft.Xna.Framework.Audio.NoAudioHardwareException.
type NoAudioHardwareException struct {
	base bclexception.State
}

// NewNoAudioHardwareExceptionByNone is NoAudioHardwareException::.ctor(), which is
// `base..ctor(SystemException::.ctor(Resources.Arg_ExternalException)` and `SetErrorCode(E_FAIL))`. The message field stays null, so Message renders the
// default sentence naming this class.
func NewNoAudioHardwareExceptionByNone() *NoAudioHardwareException {
	exception := &NoAudioHardwareException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Audio.NoAudioHardwareException", bclexception.ArgExternalException, true, nil, bclexception.EFail, true)
	return exception
}

// NewNoAudioHardwareExceptionByString is NoAudioHardwareException::.ctor(string message).
func NewNoAudioHardwareExceptionByString(message string) *NoAudioHardwareException {
	exception := &NoAudioHardwareException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Audio.NoAudioHardwareException", message, true, nil, bclexception.EFail, true)
	return exception
}

// NewNoAudioHardwareExceptionByStringAndException is
// NoAudioHardwareException::.ctor(string message, Exception inner).
func NewNoAudioHardwareExceptionByStringAndException(message string, inner framework.ExceptionReference) *NoAudioHardwareException {
	exception := &NoAudioHardwareException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Audio.NoAudioHardwareException", message, true, inner, bclexception.EFail, true)
	return exception
}

// The inherited public members, forwarded to the composed base.

func (e *NoAudioHardwareException) Message() string { return e.base.Message() }
func (e *NoAudioHardwareException) InnerException() framework.ExceptionReference {
	return e.base.InnerException()
}
func (e *NoAudioHardwareException) GetBaseException() framework.ExceptionReference {
	return e.base.GetBaseException(e)
}
func (e *NoAudioHardwareException) StackTrace() string       { return e.base.StackTrace() }
func (e *NoAudioHardwareException) HelpLink() string         { return e.base.HelpLink() }
func (e *NoAudioHardwareException) SetHelpLink(value string) { e.base.SetHelpLink(value) }
func (e *NoAudioHardwareException) Source() string           { return e.base.Source() }
func (e *NoAudioHardwareException) SetSource(value string)   { e.base.SetSource(value) }
func (e *NoAudioHardwareException) ToString() string         { return e.base.ToString() }
func (e *NoAudioHardwareException) GetType() reflect.Type    { return e.base.GetType() }

// ErrorCode is ExternalException::get_ErrorCode, one forwarded
// Exception::get_HResult. Every XNA subclass reaches it with E_FAIL, because
// the only constructor that assigns another is the one none of them declares.
func (e *NoAudioHardwareException) ErrorCode() int32 { return e.base.HResult() }

// State is the language accessor the reference interface requires. Its result
// type is declared in an internal package, so nothing outside this module can
// supply one.
func (e *NoAudioHardwareException) State() *bclexception.State { return &e.base }

// NoMicrophoneConnectedException is the XNA exception type
// Microsoft.Xna.Framework.Audio.NoMicrophoneConnectedException.
type NoMicrophoneConnectedException struct {
	base bclexception.State
}

// NewNoMicrophoneConnectedExceptionByNone is NoMicrophoneConnectedException::.ctor(), which is
// `base..ctor(Exception::.ctor())`. The message field stays null, so Message renders the
// default sentence naming this class.
func NewNoMicrophoneConnectedExceptionByNone() *NoMicrophoneConnectedException {
	exception := &NoMicrophoneConnectedException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Audio.NoMicrophoneConnectedException", "", false, nil, bclexception.CORException, false)
	return exception
}

// NewNoMicrophoneConnectedExceptionByString is NoMicrophoneConnectedException::.ctor(string message).
func NewNoMicrophoneConnectedExceptionByString(message string) *NoMicrophoneConnectedException {
	exception := &NoMicrophoneConnectedException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Audio.NoMicrophoneConnectedException", message, true, nil, bclexception.CORException, false)
	return exception
}

// NewNoMicrophoneConnectedExceptionByStringAndException is
// NoMicrophoneConnectedException::.ctor(string message, Exception inner).
func NewNoMicrophoneConnectedExceptionByStringAndException(message string, inner framework.ExceptionReference) *NoMicrophoneConnectedException {
	exception := &NoMicrophoneConnectedException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Audio.NoMicrophoneConnectedException", message, true, inner, bclexception.CORException, false)
	return exception
}

// The inherited public members, forwarded to the composed base.

func (e *NoMicrophoneConnectedException) Message() string { return e.base.Message() }
func (e *NoMicrophoneConnectedException) InnerException() framework.ExceptionReference {
	return e.base.InnerException()
}
func (e *NoMicrophoneConnectedException) GetBaseException() framework.ExceptionReference {
	return e.base.GetBaseException(e)
}
func (e *NoMicrophoneConnectedException) StackTrace() string       { return e.base.StackTrace() }
func (e *NoMicrophoneConnectedException) HelpLink() string         { return e.base.HelpLink() }
func (e *NoMicrophoneConnectedException) SetHelpLink(value string) { e.base.SetHelpLink(value) }
func (e *NoMicrophoneConnectedException) Source() string           { return e.base.Source() }
func (e *NoMicrophoneConnectedException) SetSource(value string)   { e.base.SetSource(value) }
func (e *NoMicrophoneConnectedException) ToString() string         { return e.base.ToString() }
func (e *NoMicrophoneConnectedException) GetType() reflect.Type    { return e.base.GetType() }

// State is the language accessor the reference interface requires. Its result
// type is declared in an internal package, so nothing outside this module can
// supply one.
func (e *NoMicrophoneConnectedException) State() *bclexception.State { return &e.base }
