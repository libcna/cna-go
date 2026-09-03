// Package bclexception is CNA-Go language support, not XNA surface.
//
// It holds the PRIVATE base adapter for System.Exception, and the reference
// interface every System.Exception signature position widens to.
//
// # Why it is an internal package rather than framework-private
//
// The settled composition rule keeps a BCL base adapter unexported, so nothing
// outside the module can name it or reach the base state through it. That works
// while every consumer of a base lives in one package -- Collection<T> has one
// consumer and Dictionary<K,V> has one, both in the framework package.
//
// System.Exception has EIGHT, in four different packages: Audio, Content,
// Graphics and Storage. An unexported framework type is unreachable from any of
// them, so the rule's own escape applies -- "when a COMPOSED base gains a
// consumer outside the framework package the adapter moves to an internal
// package rather than becoming exported". This is that package. `internal/`
// keeps it unreachable from outside the module, which is the property the
// unexported field had.
//
// # Reference authority
//
//	mscorlib.dll  4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0
//	sha256 5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63
package bclexception

import (
	"fmt"
	"reflect"
	"strings"
)

// The exact .NET Framework 4.0 BCL messages this projection reproduces, read
// from the pinned mscorlib above and verified byte for byte by
// tools/resource_strings.
const (
	// WasThrown is Exception_WasThrown, whose {0} is the class name.
	WasThrown = "Exception of type '%s' was thrown."
	// EndOfInnerExceptionStack is what ToString puts after a nested exception's
	// own rendering.
	EndOfInnerExceptionStack = "--- End of inner exception stack trace ---"
	// ArgExternalException is ExternalException's parameterless message.
	ArgExternalException = "External component has thrown an exception."
)

const (
	// CORException is COR_E_EXCEPTION, which Exception::Init assigns.
	CORException int32 = -2146233088 // 0x80131500
	// EFail is E_FAIL, which every ExternalException constructor but the
	// errorCode overload assigns through SetErrorCode.
	EFail int32 = -2147467259 // 0x80004005
)

// NewLine is System.Environment.NewLine as the SELECTED PROFILE has it. The
// profile is the XNA 4.0 WINDOWS runtime, where NewLine is CRLF; using the
// host's separator would make the same exception render differently depending
// on where CNA-Go happens to run, which the reference never does.
const NewLine = "\r\n"

// Reference is the interface every System.Exception signature position takes.
//
// It carries the eight projected public members of System.Exception plus the
// accessor that makes it unsatisfiable from outside this module: State returns
// an internal-package type, so no consumer can implement it.
type Reference interface {
	// Message is Exception::get_Message.
	Message() string
	// InnerException is Exception::get_InnerException.
	InnerException() Reference
	// GetBaseException is Exception::GetBaseException.
	GetBaseException() Reference
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
	// ToString is Exception::ToString, or the deriving type's override.
	ToString() string
	// GetType is Exception::GetType, the RUNTIME type.
	GetType() reflect.Type
	// State is the accessor no consumer can supply, because its result type is
	// declared in an internal package.
	State() *State
}

// State is the private Go projection of System.Exception's fields and
// behaviour. Every derived XNA exception type holds one in an unexported field
// and forwards the inherited public members to it.
type State struct {
	// message is Exception::_message, and messageSet carries the null the CLR
	// field has and Go's string does not. It is the one field here that exists
	// because of a Go/CLR mismatch rather than because the reference has it.
	message    string
	messageSet bool
	// innerException is Exception::_innerException.
	innerException Reference
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
	// external selects ExternalException's ToString override.
	external bool
}

// Init is the derived constructor's `base..ctor(...)`.
func (e *State) Init(self any, className, message string, messageSet bool, inner Reference, hresult int32, external bool) {
	e.self = self
	e.className = className
	e.message, e.messageSet = message, messageSet
	e.innerException = inner
	e.hresult = hresult
	e.external = external
}

// Message is Exception::get_Message:
//
//	if (_message == null)
//	    return Environment.GetRuntimeResourceString("Exception_WasThrown", GetClassName());
//	return _message;
//
// The null test is NOT statically dead. `new DeviceLostException()` genuinely
// leaves the field null and renders the default sentence naming the DERIVED
// class, while `new DeviceLostException("")` renders the empty string.
func (e *State) Message() string {
	if !e.messageSet {
		return fmt.Sprintf(WasThrown, e.ClassName())
	}
	return e.message
}

// InnerException is Exception::get_InnerException, one field read.
func (e *State) InnerException() Reference { return e.innerException }

// GetBaseException is Exception::GetBaseException, which walks InnerException to
// the DEEPEST non-null exception, or answers `this` when there is none.
func (e *State) GetBaseException(self Reference) Reference {
	result := self
	for inner := e.innerException; inner != nil; inner = inner.InnerException() {
		result = inner
	}
	return result
}

// StackTrace is Exception::get_StackTrace, which returns the frames the CLR
// captured AT THROW TIME.
//
// CNA-Go never throws one of these -- failure is reported through Go errors, and
// nothing in the projection raises a CLR exception -- so `_stackTraceString` is
// null for every reachable state and the getter answers null, which for a Go
// string is "". A GO stack would be a different thing wearing the same name.
func (e *State) StackTrace() string { return "" }

// HelpLink is Exception::get_HelpLink.
func (e *State) HelpLink() string { return e.helpLink }

// SetHelpLink is Exception::set_HelpLink, one stfld with no validation.
func (e *State) SetHelpLink(value string) { e.helpLink = value }

// Source is Exception::get_Source. The reference computes a default from the
// declaring assembly of the throwing frame when the field is null; an exception
// nothing threw has no such frame, so the field is all a consumer can observe.
func (e *State) Source() string { return e.source }

// SetSource is Exception::set_Source.
func (e *State) SetSource(value string) { e.source = value }

// GetType is Exception::GetType, `virtual final`, which answers the RUNTIME
// type -- the derived one, not System.Exception. The composed base holds the CLR
// `this` for exactly this member.
func (e *State) GetType() reflect.Type {
	if e.self == nil {
		return nil
	}
	return reflect.TypeOf(e.self)
}

// HResult is Exception::get_HResult, which is `family` and therefore projected
// surface only on a type that DECLARES it. No XNA exception type does;
// ExternalException's public ErrorCode forwards to it, and that is its one
// reader in the profile.
func (e *State) HResult() int32 { return e.hresult }

// ClassName is Exception::GetClassName, which lazily fills _className from the
// runtime type. The derived constructor supplies it, because a composed base
// cannot name its deriver.
func (e *State) ClassName() string {
	if e.className == "" {
		return "System.Exception"
	}
	return e.className
}

// ToString is Exception::ToString, which is `ToString(needFileLineInfo: true)`:
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
// and ExternalException::ToString, which OVERRIDES it:
//
//	s = GetType().ToString() + " (0x" + HResult.ToString("X8", InvariantCulture) + ")";
//	if (!string.IsNullOrEmpty(Message)) s += ": " + Message;
//	if (InnerException != null)         s += " ---> " + InnerException.ToString();
//	if (StackTrace != null)             s += Environment.NewLine + StackTrace;
//
// Three differences are observable and all reproduced: the external form shows
// the HResult as eight uppercase hex digits in parentheses; it does NOT append
// the end-of-inner-exception marker; and both append a stack trace only when
// there is one, which for a CNA-Go exception there never is.
func (e *State) ToString() string {
	var builder strings.Builder
	builder.WriteString(e.ClassName())
	if e.external {
		builder.WriteString(fmt.Sprintf(" (0x%08X)", uint32(e.hresult)))
	}
	if message := e.Message(); message != "" {
		builder.WriteString(": ")
		builder.WriteString(message)
	}
	if e.innerException != nil {
		builder.WriteString(" ---> ")
		builder.WriteString(e.innerException.ToString())
		if !e.external {
			builder.WriteString(NewLine)
			builder.WriteString("   ")
			builder.WriteString(EndOfInnerExceptionStack)
		}
	}
	return builder.String()
}
