package framework

import (
	"fmt"
	"reflect"
	"strings"
)

// This file is CNA-Go language support, not XNA surface.
//
// It projects System.Exception in THREE roles at once, which no other BCL type
// in the profile has needed:
//
//   - `exceptionBase` is the PRIVATE base adapter the eight XNA exception types
//     compose, exactly as collectionBase[T] and dictionaryBase[K,V] are;
//   - `Exception` is the PUBLIC concrete projection of the CLR class, because a
//     consumer constructs one -- `new Exception("...")` is ordinary code;
//   - `ExceptionReference` is the exported reference interface a System.Exception
//     SIGNATURE POSITION widens to.
//
// # Reference authority
//
//	mscorlib.dll  4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0
//	sha256 5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63
//
// # Why the signature position is the INTERFACE and not the concrete type
//
// The settled substitutable-base rule widens a base-typed PARAMETER to an
// exported reference interface and leaves a base-typed RETURN as the concrete
// pointer, recording the lost downcast as a language limitation. This family is
// the case where that trade would cost the type's whole purpose: an exception
// hierarchy exists to be told apart by type, and `InnerException` returning a
// concrete *Exception would erase which of the eight kinds a consumer chained.
//
// So System.Exception widens at EVERY position, parameter and return alike, and
// the downcast a C# consumer writes as `catch (DeviceLostException)` is the Go
// type assertion `inner.(*graphics.DeviceLostException)`. The interface carries
// an unexported method, so only this module can satisfy it -- a consumer cannot
// invent a ninth exception type and hand it to a projected member.
//
// # This type is NOT a Go error
//
// Foundation 29 recorded that as the material question, and the answer is that
// the two are different contracts. A projected OPERATION reports failure through
// a Go error, as it has since Foundation 1, and none of the settled
// per-operation fallibility decisions is reopened by this milestone. A CLR
// exception OBJECT is a value the profile constructs and passes -- to
// ShowMissingRequirementMessage, and through InnerException -- and it needs a Go
// spelling or those members cannot be projected at all. Giving this type an
// Error() method would collapse the two and publish a member the pinned CLR type
// does not declare, so it has none.
//
// # Three public members are absent, and each names its closure
//
// Data needs System.Collections.IDictionary, TargetSite needs
// System.Reflection.MethodBase, and GetObjectData needs
// System.Runtime.Serialization. All three are BCL_PROJECTION_BLOCKED_EXTERNAL
// exclusions in the adapter registry, which counts them and refuses one that
// names no closure.

// The exact .NET Framework 4.0 BCL messages this projection reproduces, read
// from the pinned mscorlib above and verified byte for byte by
// tools/resource_strings.
const (
	// exceptionWasThrown is Exception_WasThrown, whose {0} is the class name.
	exceptionWasThrown = "Exception of type '%s' was thrown."
	// exceptionEndOfInnerExceptionStack is what ToString puts after a nested
	// exception's own rendering.
	exceptionEndOfInnerExceptionStack = "--- End of inner exception stack trace ---"
	// argExternalException is ExternalException's parameterless message.
	argExternalException = "External component has thrown an exception."
)

const (
	// exceptionCORException is COR_E_EXCEPTION, which Exception::Init assigns.
	exceptionCORException int32 = -2146233088 // 0x80131500
	// exceptionEFail is E_FAIL, which every ExternalException constructor but
	// the errorCode overload assigns through SetErrorCode.
	exceptionEFail int32 = -2146232832 // 0x80004005
)

// exceptionNewLine is System.Environment.NewLine as the SELECTED PROFILE has
// it. The profile is the XNA 4.0 WINDOWS runtime, where NewLine is CRLF; using
// the host's separator would make the same exception render differently
// depending on where CNA-Go happens to run, which the reference never does.
const exceptionNewLine = "\r\n"

// ExceptionReference is the exported interface every System.Exception signature
// position takes.
//
// It carries the ten projected public members of System.Exception plus the
// unexported accessor that makes it unsatisfiable from outside this module.
type ExceptionReference interface {
	// Message is Exception::get_Message.
	Message() string
	// InnerException is Exception::get_InnerException.
	InnerException() ExceptionReference
	// GetBaseException is Exception::GetBaseException.
	GetBaseException() ExceptionReference
	// StackTrace is Exception::get_StackTrace.
	StackTrace() string
	// HelpLink is Exception::get_HelpLink.
	HelpLink() string
	// SetHelpLink is Exception::set_HelpLink.
	SetHelpLink(value string)
	// Source is Exception::get_Source.
	Source() string
	// SetSource is Exception::set_Source.
	SetSource(value string)
	// ToString is Exception::ToString, or the deriving type's override of it.
	ToString() string
	// GetType is Exception::GetType, the RUNTIME type.
	GetType() reflect.Type
	// exceptionState is the unexported accessor. Only a type declared in this
	// module can supply one, which is what keeps the eight XNA exception types
	// the whole of the interface's implementors.
	exceptionState() *exceptionBase
}

// exceptionBase is the private Go projection of System.Exception's state and
// behaviour, held by the concrete Exception and by every derived XNA exception
// type in an unexported field.
type exceptionBase struct {
	// message is Exception::_message, and messageSet carries the null the CLR
	// field has and Go's string does not. It is the one field here that exists
	// because of a Go/CLR mismatch rather than because the reference has it;
	// see message() for why the distinction is observable.
	message    string
	messageSet bool
	// innerException is Exception::_innerException.
	innerException ExceptionReference
	// helpLink is Exception::_helpURL and source is Exception::_source.
	helpLink string
	source   string
	// hresult is Exception::_HResult, which the `family` HResult property
	// exposes and which ExternalException::get_ErrorCode forwards to.
	hresult int32
	// className is Exception::_className, which the reference computes lazily
	// from the runtime type. A composed base cannot see its deriver, so the
	// derived constructor supplies it.
	className string
	// self is the CLR `this` a composed base loses. GetType dispatches through
	// it, and the derived constructor installs it.
	self any
	// externalToString selects ExternalException's ToString override, which
	// renders quite differently from Exception's.
	externalToString bool
}

// init is the derived constructor's `base..ctor(...)`.
func (e *exceptionBase) init(self any, className, message string, messageSet bool, inner ExceptionReference, hresult int32, external bool) {
	e.self = self
	e.className = className
	e.message, e.messageSet = message, messageSet
	e.innerException = inner
	e.hresult = hresult
	e.externalToString = external
}

// message is Exception::get_Message:
//
//	if (_message == null)
//	    return Environment.GetRuntimeResourceString("Exception_WasThrown", GetClassName());
//	return _message;
//
// The null test is NOT statically dead, unlike the null-key guard the Dictionary
// base records. `new Exception()` genuinely leaves the field null and genuinely
// renders the default sentence, while `new Exception("")` renders the empty
// string, and a consumer reading Message sees the difference. That is why
// messageSet exists.
func (e *exceptionBase) messageOf() string {
	if !e.messageSet {
		return fmt.Sprintf(exceptionWasThrown, e.classNameOf())
	}
	return e.message
}

func (e *exceptionBase) innerOf() ExceptionReference { return e.innerException }

// baseExceptionOf is Exception::GetBaseException, which walks InnerException to
// the DEEPEST non-null exception, or answers `this` when there is none.
func (e *exceptionBase) baseExceptionOf(self ExceptionReference) ExceptionReference {
	result := self
	for inner := e.innerException; inner != nil; inner = inner.InnerException() {
		result = inner
	}
	return result
}

// stackTraceOf is Exception::get_StackTrace, which returns the frames the CLR
// captured AT THROW TIME.
//
// CNA-Go never throws one of these -- failure is reported through Go errors, and
// nothing in the projection raises a CLR exception -- so `_stackTraceString` is
// null for every reachable state and the getter answers null, which for a Go
// string is "". This is the member Foundation 29 named as unrepresentable, and
// it stays that: a GO stack would be a different thing wearing the same name.
func (e *exceptionBase) stackTraceOf() string { return "" }

func (e *exceptionBase) helpLinkOf() string         { return e.helpLink }
func (e *exceptionBase) setHelpLinkOf(value string) { e.helpLink = value }

// sourceOf is Exception::get_Source. The reference computes a default from the
// declaring assembly of the throwing frame when the field is null; an exception
// nothing threw has no such frame, so the field is all a consumer can observe.
func (e *exceptionBase) sourceOf() string         { return e.source }
func (e *exceptionBase) setSourceOf(value string) { e.source = value }

// typeOf is Exception::GetType, `virtual final`, which answers the RUNTIME type
// -- the derived one, not System.Exception. The composed base holds the CLR
// `this` for exactly this member.
func (e *exceptionBase) typeOf() reflect.Type {
	if e.self == nil {
		return nil
	}
	return reflect.TypeOf(e.self)
}

func (e *exceptionBase) hresultOf() int32 { return e.hresult }

func (e *exceptionBase) classNameOf() string {
	if e.className == "" {
		return "System.Exception"
	}
	return e.className
}

// toStringOf is Exception::ToString, which is `ToString(needFileLineInfo: true)`:
//
//	message = Message;
//	s = (message == null || message.Length <= 0)
//	      ? GetClassName()
//	      : GetClassName() + ": " + message;
//	if (_innerException != null)
//	    s += " ---> " + _innerException.ToString(needFileLineInfo)
//	       + Environment.NewLine + "   "
//	       + "--- End of inner exception stack trace ---";
//	stackTrace = GetStackTrace(needFileLineInfo);
//	if (stackTrace != null) s += Environment.NewLine + stackTrace;
//
// and ExternalException::ToString, which OVERRIDES it and renders differently:
//
//	s = GetType().ToString() + " (0x" + HResult.ToString("X8", InvariantCulture) + ")";
//	if (!string.IsNullOrEmpty(Message)) s += ": " + Message;
//	if (InnerException != null)         s += " ---> " + InnerException.ToString();
//	if (StackTrace != null)             s += Environment.NewLine + StackTrace;
//
// Three differences between them are observable and are all reproduced: the
// HResult appears in the external form as eight uppercase hex digits in
// parentheses; the external form does NOT append the end-of-inner-exception
// marker; and both append a stack trace only when there is one, which for a
// CNA-Go exception there never is.
func (e *exceptionBase) toStringOf() string {
	var builder strings.Builder
	builder.WriteString(e.classNameOf())
	if e.externalToString {
		builder.WriteString(fmt.Sprintf(" (0x%08X)", uint32(e.hresult)))
	}
	if message := e.messageOf(); message != "" {
		builder.WriteString(": ")
		builder.WriteString(message)
	}
	if e.innerException != nil {
		builder.WriteString(" ---> ")
		builder.WriteString(e.innerException.ToString())
		if !e.externalToString {
			builder.WriteString(exceptionNewLine)
			builder.WriteString("   ")
			builder.WriteString(exceptionEndOfInnerExceptionStack)
		}
	}
	return builder.String()
}

// ---------------------------------------------------------------------------
// Exception -- the public concrete projection of the CLR class.
// ---------------------------------------------------------------------------

// Exception is the Go spelling of System.Exception as a constructible object.
// Every projected signature takes ExceptionReference instead, which this type
// satisfies; this is what a consumer allocates when the reference would write
// `new Exception(...)`.
type Exception struct {
	base exceptionBase
}

// NewException is Exception::.ctor(), whose body is `Init()`. The message field
// stays null, which is what makes Message render the default sentence.
func NewException() *Exception {
	exception := &Exception{}
	exception.base.init(exception, "System.Exception", "", false, nil, exceptionCORException, false)
	return exception
}

// NewExceptionByString is Exception::.ctor(string message).
func NewExceptionByString(message string) *Exception {
	exception := &Exception{}
	exception.base.init(exception, "System.Exception", message, true, nil, exceptionCORException, false)
	return exception
}

// NewExceptionByStringAndException is
// Exception::.ctor(string message, Exception innerException).
func NewExceptionByStringAndException(message string, innerException ExceptionReference) *Exception {
	exception := &Exception{}
	exception.base.init(exception, "System.Exception", message, true, innerException, exceptionCORException, false)
	return exception
}

func (e *Exception) Message() string                      { return e.base.messageOf() }
func (e *Exception) InnerException() ExceptionReference   { return e.base.innerOf() }
func (e *Exception) GetBaseException() ExceptionReference { return e.base.baseExceptionOf(e) }
func (e *Exception) StackTrace() string                   { return e.base.stackTraceOf() }
func (e *Exception) HelpLink() string                     { return e.base.helpLinkOf() }
func (e *Exception) SetHelpLink(value string)             { e.base.setHelpLinkOf(value) }
func (e *Exception) Source() string                       { return e.base.sourceOf() }
func (e *Exception) SetSource(value string)               { e.base.setSourceOf(value) }
func (e *Exception) ToString() string                     { return e.base.toStringOf() }
func (e *Exception) GetType() reflect.Type                { return e.base.typeOf() }
func (e *Exception) exceptionState() *exceptionBase       { return &e.base }
