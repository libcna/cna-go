package framework

import (
	"reflect"

	"github.com/openeggbert/cna-go/internal/bclexception"
)

// This file is CNA-Go language support, not XNA surface.
//
// It projects System.Exception in the two roles the framework package owns. The
// third -- the PRIVATE base adapter the eight XNA exception types compose --
// lives in internal/bclexception, because those eight types are in four OTHER
// packages and an unexported framework type is unreachable from any of them.
// That is the composition rule's own escape, not a departure from it.
//
//   - `Exception` is the PUBLIC concrete projection of the CLR class, because a
//     consumer constructs one: `new Exception("...")` is ordinary code.
//   - `ExceptionReference` is the interface every System.Exception SIGNATURE
//     POSITION widens to.
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
// the case where that trade would cost the type its purpose: an exception
// hierarchy exists to be told apart by type, and `InnerException` returning a
// concrete *Exception would erase which of the nine kinds a consumer chained.
//
// So System.Exception widens at EVERY position, and the downcast a C# consumer
// writes as `catch (DeviceLostException)` is the Go type assertion
// `inner.(*graphics.DeviceLostException)`. The interface's State accessor
// returns an internal-package type, so nothing outside this module can satisfy
// it: a consumer cannot invent a tenth exception type and hand it to a
// projected member.
//
// # This type is NOT a Go error
//
// Foundation 29 recorded that as the material question, and the answer is that
// the two are different contracts. A projected OPERATION reports failure through
// a Go error, as it has since Foundation 1, and none of the settled
// per-operation fallibility decisions is reopened. A CLR exception OBJECT is a
// value the profile constructs and passes -- to ShowMissingRequirementMessage,
// and through InnerException -- and it needs a Go spelling or those members
// cannot be projected at all. Giving this type an Error() method would collapse
// the two and publish a member the pinned CLR type does not declare.
//
// # Three public members are absent, and each names its closure
//
// Data needs System.Collections.IDictionary, TargetSite needs
// System.Reflection.MethodBase, and GetObjectData needs
// System.Runtime.Serialization. All three are BCL_PROJECTION_BLOCKED_EXTERNAL
// exclusions in the adapter registry, which counts them and refuses one that
// names no closure.

// ExceptionReference is the exported interface every System.Exception signature
// position takes. It is an alias for the internal package's contract, so the
// eight derived exception types in four other packages and this package's
// concrete Exception all satisfy exactly one interface.
type ExceptionReference = bclexception.Reference

// Exception is the Go spelling of System.Exception as a constructible object.
// Every projected signature takes ExceptionReference instead, which this type
// satisfies; this is what a consumer allocates where the reference writes
// `new Exception(...)`.
type Exception struct {
	base bclexception.State
}

// NewException is Exception::.ctor(), whose body is `Init()`. The message field
// stays null, which is what makes Message render the default sentence.
func NewException() *Exception {
	exception := &Exception{}
	exception.base.Init(exception, "System.Exception", "", false, nil, bclexception.CORException, false)
	return exception
}

// NewExceptionByString is Exception::.ctor(string message).
func NewExceptionByString(message string) *Exception {
	exception := &Exception{}
	exception.base.Init(exception, "System.Exception", message, true, nil, bclexception.CORException, false)
	return exception
}

// NewExceptionByStringAndException is
// Exception::.ctor(string message, Exception innerException).
func NewExceptionByStringAndException(message string, innerException ExceptionReference) *Exception {
	exception := &Exception{}
	exception.base.Init(exception, "System.Exception", message, true, innerException, bclexception.CORException, false)
	return exception
}

// The eight inherited public members, forwarded to the base adapter.

func (e *Exception) Message() string                      { return e.base.Message() }
func (e *Exception) InnerException() ExceptionReference   { return e.base.InnerException() }
func (e *Exception) GetBaseException() ExceptionReference { return e.base.GetBaseException(e) }
func (e *Exception) StackTrace() string                   { return e.base.StackTrace() }
func (e *Exception) HelpLink() string                     { return e.base.HelpLink() }
func (e *Exception) SetHelpLink(value string)             { e.base.SetHelpLink(value) }
func (e *Exception) Source() string                       { return e.base.Source() }
func (e *Exception) SetSource(value string)               { e.base.SetSource(value) }
func (e *Exception) ToString() string                     { return e.base.ToString() }
func (e *Exception) GetType() reflect.Type                { return e.base.GetType() }

// State is the accessor that keeps ExceptionReference unsatisfiable from
// outside the module. It is not projected surface: its result type is declared
// in an internal package, so no consumer can name it or supply one.
func (e *Exception) State() *bclexception.State { return &e.base }
