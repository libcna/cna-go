package storage

import (
	"reflect"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/bclexception"
)

// This file projects the one XNA exception type declared in
// Microsoft.Xna.Framework.Storage.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Storage.dll   798f678e9ae3d9af...
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

// StorageDeviceNotConnectedException is the XNA exception type
// Microsoft.Xna.Framework.Storage.StorageDeviceNotConnectedException.
type StorageDeviceNotConnectedException struct {
	base bclexception.State
}

// NewStorageDeviceNotConnectedExceptionByNone is StorageDeviceNotConnectedException::.ctor(), which is
// `base..ctor(SystemException::.ctor(Resources.Arg_ExternalException)` and `SetErrorCode(E_FAIL))`. The message field stays null, so Message renders the
// default sentence naming this class.
func NewStorageDeviceNotConnectedExceptionByNone() *StorageDeviceNotConnectedException {
	exception := &StorageDeviceNotConnectedException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Storage.StorageDeviceNotConnectedException", bclexception.ArgExternalException, true, nil, bclexception.EFail, true)
	return exception
}

// NewStorageDeviceNotConnectedExceptionByString is StorageDeviceNotConnectedException::.ctor(string message).
func NewStorageDeviceNotConnectedExceptionByString(message string) *StorageDeviceNotConnectedException {
	exception := &StorageDeviceNotConnectedException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Storage.StorageDeviceNotConnectedException", message, true, nil, bclexception.EFail, true)
	return exception
}

// NewStorageDeviceNotConnectedExceptionByStringAndException is
// StorageDeviceNotConnectedException::.ctor(string message, Exception innerException).
func NewStorageDeviceNotConnectedExceptionByStringAndException(message string, innerException framework.ExceptionReference) *StorageDeviceNotConnectedException {
	exception := &StorageDeviceNotConnectedException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Storage.StorageDeviceNotConnectedException", message, true, innerException, bclexception.EFail, true)
	return exception
}

// The inherited public members, forwarded to the composed base.

func (e *StorageDeviceNotConnectedException) Message() string { return e.base.Message() }
func (e *StorageDeviceNotConnectedException) InnerException() framework.ExceptionReference {
	return e.base.InnerException()
}
func (e *StorageDeviceNotConnectedException) GetBaseException() framework.ExceptionReference {
	return e.base.GetBaseException(e)
}
func (e *StorageDeviceNotConnectedException) StackTrace() string       { return e.base.StackTrace() }
func (e *StorageDeviceNotConnectedException) HelpLink() string         { return e.base.HelpLink() }
func (e *StorageDeviceNotConnectedException) SetHelpLink(value string) { e.base.SetHelpLink(value) }
func (e *StorageDeviceNotConnectedException) Source() string           { return e.base.Source() }
func (e *StorageDeviceNotConnectedException) SetSource(value string)   { e.base.SetSource(value) }
func (e *StorageDeviceNotConnectedException) ToString() string         { return e.base.ToString() }
func (e *StorageDeviceNotConnectedException) GetType() reflect.Type    { return e.base.GetType() }

// ErrorCode is ExternalException::get_ErrorCode, one forwarded
// Exception::get_HResult. Every XNA subclass reaches it with E_FAIL, because
// the only constructor that assigns another is the one none of them declares.
func (e *StorageDeviceNotConnectedException) ErrorCode() int32 { return e.base.HResult() }

// State is the language accessor the reference interface requires. Its result
// type is declared in an internal package, so nothing outside this module can
// supply one.
func (e *StorageDeviceNotConnectedException) State() *bclexception.State { return &e.base }
