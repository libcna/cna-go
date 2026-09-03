package content

import (
	"reflect"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/bclexception"
)

// This file projects the one XNA exception type declared in
// Microsoft.Xna.Framework.Content.
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

// ContentLoadException is the XNA exception type
// Microsoft.Xna.Framework.Content.ContentLoadException.
type ContentLoadException struct {
	base bclexception.State
}

// NewContentLoadExceptionByNone is ContentLoadException::.ctor(), which is
// `base..ctor(Exception::.ctor())`. The message field stays null, so Message renders the
// default sentence naming this class.
func NewContentLoadExceptionByNone() *ContentLoadException {
	exception := &ContentLoadException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Content.ContentLoadException", "", false, nil, bclexception.CORException, false)
	return exception
}

// NewContentLoadExceptionByString is ContentLoadException::.ctor(string message).
func NewContentLoadExceptionByString(message string) *ContentLoadException {
	exception := &ContentLoadException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Content.ContentLoadException", message, true, nil, bclexception.CORException, false)
	return exception
}

// NewContentLoadExceptionByStringAndException is
// ContentLoadException::.ctor(string message, Exception innerException).
func NewContentLoadExceptionByStringAndException(message string, innerException framework.ExceptionReference) *ContentLoadException {
	exception := &ContentLoadException{}
	exception.base.Init(exception, "Microsoft.Xna.Framework.Content.ContentLoadException", message, true, innerException, bclexception.CORException, false)
	return exception
}

// The inherited public members, forwarded to the composed base.

func (e *ContentLoadException) Message() string { return e.base.Message() }
func (e *ContentLoadException) InnerException() framework.ExceptionReference {
	return e.base.InnerException()
}
func (e *ContentLoadException) GetBaseException() framework.ExceptionReference {
	return e.base.GetBaseException(e)
}
func (e *ContentLoadException) StackTrace() string       { return e.base.StackTrace() }
func (e *ContentLoadException) HelpLink() string         { return e.base.HelpLink() }
func (e *ContentLoadException) SetHelpLink(value string) { e.base.SetHelpLink(value) }
func (e *ContentLoadException) Source() string           { return e.base.Source() }
func (e *ContentLoadException) SetSource(value string)   { e.base.SetSource(value) }
func (e *ContentLoadException) ToString() string         { return e.base.ToString() }
func (e *ContentLoadException) GetType() reflect.Type    { return e.base.GetType() }

// State is the language accessor the reference interface requires. Its result
// type is declared in an internal package, so nothing outside this module can
// supply one.
func (e *ContentLoadException) State() *bclexception.State { return &e.base }
