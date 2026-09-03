package graphics

import (
	"reflect"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/bclexception"
)

// This file projects the three XNA exception types declared in
// Microsoft.Xna.Framework.Graphics.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Graphics.dll   560080fc39021c61...
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

// DeviceLostException is the XNA exception type
// Microsoft.Xna.Framework.Graphics.DeviceLostException.
type DeviceLostException struct {
	base bclexception.State
}

// NewDeviceLostExceptionByNone is DeviceLostException::.ctor(), which is
// `base..ctor(Exception::.ctor())`. The message field stays null, so Message renders the
// default sentence naming this class.
func NewDeviceLostExceptionByNone() *DeviceLostException {
	exception := &DeviceLostException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Graphics.DeviceLostException", "", false, nil, bclexception.CORException, false)
	return exception
}

// NewDeviceLostExceptionByString is DeviceLostException::.ctor(string message).
func NewDeviceLostExceptionByString(message string) *DeviceLostException {
	exception := &DeviceLostException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Graphics.DeviceLostException", message, true, nil, bclexception.CORException, false)
	return exception
}

// NewDeviceLostExceptionByStringAndException is
// DeviceLostException::.ctor(string message, Exception inner).
func NewDeviceLostExceptionByStringAndException(message string, inner framework.ExceptionReference) *DeviceLostException {
	exception := &DeviceLostException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Graphics.DeviceLostException", message, true, inner, bclexception.CORException, false)
	return exception
}

// The inherited public members, forwarded to the composed base.

func (e *DeviceLostException) Message() string { return e.base.Message() }
func (e *DeviceLostException) InnerException() framework.ExceptionReference {
	return e.base.InnerException()
}
func (e *DeviceLostException) GetBaseException() framework.ExceptionReference {
	return e.base.GetBaseException(e)
}
func (e *DeviceLostException) StackTrace() string       { return e.base.StackTrace() }
func (e *DeviceLostException) HelpLink() string         { return e.base.HelpLink() }
func (e *DeviceLostException) SetHelpLink(value string) { e.base.SetHelpLink(value) }
func (e *DeviceLostException) Source() string           { return e.base.Source() }
func (e *DeviceLostException) SetSource(value string)   { e.base.SetSource(value) }
func (e *DeviceLostException) ToString() string         { return e.base.ToString() }
func (e *DeviceLostException) GetType() reflect.Type    { return e.base.GetType() }

// State is the language accessor the reference interface requires. Its result
// type is declared in an internal package, so nothing outside this module can
// supply one.
func (e *DeviceLostException) State() *bclexception.State { return &e.base }

// DeviceNotResetException is the XNA exception type
// Microsoft.Xna.Framework.Graphics.DeviceNotResetException.
type DeviceNotResetException struct {
	base bclexception.State
}

// NewDeviceNotResetExceptionByNone is DeviceNotResetException::.ctor(), which is
// `base..ctor(Exception::.ctor())`. The message field stays null, so Message renders the
// default sentence naming this class.
func NewDeviceNotResetExceptionByNone() *DeviceNotResetException {
	exception := &DeviceNotResetException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Graphics.DeviceNotResetException", "", false, nil, bclexception.CORException, false)
	return exception
}

// NewDeviceNotResetExceptionByString is DeviceNotResetException::.ctor(string message).
func NewDeviceNotResetExceptionByString(message string) *DeviceNotResetException {
	exception := &DeviceNotResetException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Graphics.DeviceNotResetException", message, true, nil, bclexception.CORException, false)
	return exception
}

// NewDeviceNotResetExceptionByStringAndException is
// DeviceNotResetException::.ctor(string message, Exception inner).
func NewDeviceNotResetExceptionByStringAndException(message string, inner framework.ExceptionReference) *DeviceNotResetException {
	exception := &DeviceNotResetException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Graphics.DeviceNotResetException", message, true, inner, bclexception.CORException, false)
	return exception
}

// The inherited public members, forwarded to the composed base.

func (e *DeviceNotResetException) Message() string { return e.base.Message() }
func (e *DeviceNotResetException) InnerException() framework.ExceptionReference {
	return e.base.InnerException()
}
func (e *DeviceNotResetException) GetBaseException() framework.ExceptionReference {
	return e.base.GetBaseException(e)
}
func (e *DeviceNotResetException) StackTrace() string       { return e.base.StackTrace() }
func (e *DeviceNotResetException) HelpLink() string         { return e.base.HelpLink() }
func (e *DeviceNotResetException) SetHelpLink(value string) { e.base.SetHelpLink(value) }
func (e *DeviceNotResetException) Source() string           { return e.base.Source() }
func (e *DeviceNotResetException) SetSource(value string)   { e.base.SetSource(value) }
func (e *DeviceNotResetException) ToString() string         { return e.base.ToString() }
func (e *DeviceNotResetException) GetType() reflect.Type    { return e.base.GetType() }

// State is the language accessor the reference interface requires. Its result
// type is declared in an internal package, so nothing outside this module can
// supply one.
func (e *DeviceNotResetException) State() *bclexception.State { return &e.base }

// NoSuitableGraphicsDeviceException is the XNA exception type
// Microsoft.Xna.Framework.Graphics.NoSuitableGraphicsDeviceException.
type NoSuitableGraphicsDeviceException struct {
	base bclexception.State
}

// NewNoSuitableGraphicsDeviceExceptionByNone is NoSuitableGraphicsDeviceException::.ctor(), which is
// `base..ctor(Exception::.ctor())`. The message field stays null, so Message renders the
// default sentence naming this class.
func NewNoSuitableGraphicsDeviceExceptionByNone() *NoSuitableGraphicsDeviceException {
	exception := &NoSuitableGraphicsDeviceException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Graphics.NoSuitableGraphicsDeviceException", "", false, nil, bclexception.CORException, false)
	return exception
}

// NewNoSuitableGraphicsDeviceExceptionByString is NoSuitableGraphicsDeviceException::.ctor(string message).
func NewNoSuitableGraphicsDeviceExceptionByString(message string) *NoSuitableGraphicsDeviceException {
	exception := &NoSuitableGraphicsDeviceException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Graphics.NoSuitableGraphicsDeviceException", message, true, nil, bclexception.CORException, false)
	return exception
}

// NewNoSuitableGraphicsDeviceExceptionByStringAndException is
// NoSuitableGraphicsDeviceException::.ctor(string message, Exception inner).
func NewNoSuitableGraphicsDeviceExceptionByStringAndException(message string, inner framework.ExceptionReference) *NoSuitableGraphicsDeviceException {
	exception := &NoSuitableGraphicsDeviceException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Graphics.NoSuitableGraphicsDeviceException", message, true, inner, bclexception.CORException, false)
	return exception
}

// The inherited public members, forwarded to the composed base.

func (e *NoSuitableGraphicsDeviceException) Message() string { return e.base.Message() }
func (e *NoSuitableGraphicsDeviceException) InnerException() framework.ExceptionReference {
	return e.base.InnerException()
}
func (e *NoSuitableGraphicsDeviceException) GetBaseException() framework.ExceptionReference {
	return e.base.GetBaseException(e)
}
func (e *NoSuitableGraphicsDeviceException) StackTrace() string       { return e.base.StackTrace() }
func (e *NoSuitableGraphicsDeviceException) HelpLink() string         { return e.base.HelpLink() }
func (e *NoSuitableGraphicsDeviceException) SetHelpLink(value string) { e.base.SetHelpLink(value) }
func (e *NoSuitableGraphicsDeviceException) Source() string           { return e.base.Source() }
func (e *NoSuitableGraphicsDeviceException) SetSource(value string)   { e.base.SetSource(value) }
func (e *NoSuitableGraphicsDeviceException) ToString() string         { return e.base.ToString() }
func (e *NoSuitableGraphicsDeviceException) GetType() reflect.Type    { return e.base.GetType() }

// State is the language accessor the reference interface requires. Its result
// type is declared in an internal package, so nothing outside this module can
// supply one.
func (e *NoSuitableGraphicsDeviceException) State() *bclexception.State { return &e.base }
