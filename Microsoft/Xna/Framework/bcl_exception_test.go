package framework

import (
	"reflect"
	"strings"
	"testing"
)

func TestExceptionMessageDistinguishesUnsetFromEmpty(t *testing.T) {
	// `new Exception()` leaves _message null and renders the pinned default.
	if got := NewException().Message(); got != "Exception of type 'System.Exception' was thrown." {
		t.Fatalf("default Message = %q", got)
	}
	// `new Exception("")` sets the field to the empty string, which is NOT
	// null, and renders empty. This is the distinction messageSet exists for:
	// a projection that stored only the string would render the default for
	// both.
	if got := NewExceptionByString("").Message(); got != "" {
		t.Fatalf("empty Message = %q, want the empty string", got)
	}
	if got := NewExceptionByString("boom").Message(); got != "boom" {
		t.Fatalf("Message = %q", got)
	}
}

func TestExceptionInnerChainAndBaseException(t *testing.T) {
	deepest := NewExceptionByString("deepest")
	middle := NewExceptionByStringAndException("middle", deepest)
	outer := NewExceptionByStringAndException("outer", middle)

	if outer.InnerException() != ExceptionReference(middle) {
		t.Fatal("InnerException did not hand back the object the constructor stored")
	}
	if middle.InnerException() != ExceptionReference(deepest) {
		t.Fatal("the chain is not preserved")
	}
	if deepest.InnerException() != nil {
		t.Fatal("the deepest exception reports an inner one")
	}
	// GetBaseException walks to the DEEPEST, not to the immediate inner.
	if outer.GetBaseException() != ExceptionReference(deepest) {
		t.Fatal("GetBaseException did not reach the deepest exception")
	}
	// With no inner exception it answers `this`.
	if deepest.GetBaseException() != ExceptionReference(deepest) {
		t.Fatal("GetBaseException on a leaf did not answer itself")
	}
}

func TestExceptionToStringFollowsTheReferenceShape(t *testing.T) {
	// No message: the class name alone.
	if got := NewException().ToString(); !strings.HasPrefix(got, "System.Exception: Exception of type") {
		t.Fatalf("ToString of a default exception = %q", got)
	}
	if got := NewExceptionByString("boom").ToString(); got != "System.Exception: boom" {
		t.Fatalf("ToString = %q", got)
	}
	// An inner exception is embedded, followed by CRLF, three spaces and the
	// pinned end-of-stack marker.
	outer := NewExceptionByStringAndException("outer", NewExceptionByString("inner"))
	want := "System.Exception: outer ---> System.Exception: inner" +
		"\r\n   --- End of inner exception stack trace ---"
	if got := outer.ToString(); got != want {
		t.Fatalf("nested ToString =\n%q\nwant\n%q", got, want)
	}
}

func TestExceptionHelpLinkAndSourceAreStoredFields(t *testing.T) {
	exception := NewExceptionByString("boom")
	if exception.HelpLink() != "" || exception.Source() != "" {
		t.Fatal("a fresh exception reports a help link or a source")
	}
	exception.SetHelpLink("http://example.invalid/help")
	exception.SetSource("cna-go")
	if exception.HelpLink() != "http://example.invalid/help" || exception.Source() != "cna-go" {
		t.Fatalf("HelpLink/Source = %q/%q", exception.HelpLink(), exception.Source())
	}
	// Neither setter validates, so the empty string round-trips.
	exception.SetHelpLink("")
	exception.SetSource("")
	if exception.HelpLink() != "" || exception.Source() != "" {
		t.Fatal("a setter refused the empty string")
	}
}

// TestExceptionStackTraceIsEmptyBecauseNothingThrew pins the one member
// Foundation 29 named as unrepresentable, and pins WHY: the reference's own
// answer for a constructed-never-thrown exception is null.
func TestExceptionStackTraceIsEmptyBecauseNothingThrew(t *testing.T) {
	if got := NewExceptionByString("boom").StackTrace(); got != "" {
		t.Fatalf("StackTrace = %q, want the empty string a null renders as", got)
	}
}

func TestExceptionGetTypeAnswersTheRuntimeType(t *testing.T) {
	exception := NewException()
	if got := exception.GetType(); got != reflect.TypeOf(exception) {
		t.Fatalf("GetType = %v", got)
	}
}

// TestExceptionSatisfiesTheReferenceInterface is the widening claim: every
// projected System.Exception position takes ExceptionReference, and the
// concrete type must satisfy it.
func TestExceptionSatisfiesTheReferenceInterface(t *testing.T) {
	var reference ExceptionReference = NewExceptionByString("boom")
	if reference.Message() != "boom" {
		t.Fatal("the interface does not carry Message")
	}
	// The unexported accessor is what keeps the interface unsatisfiable from
	// outside the module.
	if reference.exceptionState() == nil {
		t.Fatal("the reference interface exposes no state")
	}
	// And the concrete type is recoverable, which is the downcast a C# consumer
	// writes as a catch clause.
	typed, ok := reference.(*Exception)
	if !ok || typed.Message() != "boom" {
		t.Fatal("the reference interface could not be asserted back to its concrete type")
	}
}

// TestShowMissingRequirementMessageIsTheBaseHostsAnswer pins the measured
// constant, and pins that it is a constant rather than a refusal: the member is
// infallible.
func TestShowMissingRequirementMessageIsTheBaseHostsAnswer(t *testing.T) {
	game, err := NewGame(&bareCallbacks{})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	for _, exception := range []ExceptionReference{
		nil,
		NewException(),
		NewExceptionByString("a missing requirement"),
		NewExceptionByStringAndException("outer", NewExceptionByString("inner")),
	} {
		if game.ShowMissingRequirementMessage(exception) {
			t.Fatalf("ShowMissingRequirementMessage answered true for %v", exception)
		}
	}
}
