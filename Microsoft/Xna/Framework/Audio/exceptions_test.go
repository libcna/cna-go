package audio

import (
	"strings"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// TestExternalExceptionSubclassesCarryErrorCodeAndTheirOwnToString is the
// ExternalException half of the family, which differs from System.Exception's
// in three observable ways.
func TestExternalExceptionSubclassesCarryErrorCodeAndTheirOwnToString(t *testing.T) {
	// Every XNA ExternalException subclass reaches E_FAIL, because the only
	// constructor that assigns another error code is the one none of them
	// declares.
	const eFail int32 = -2147467259
	for name, exception := range map[string]interface {
		framework.ExceptionReference
		ErrorCode() int32
	}{
		"Microsoft.Xna.Framework.Audio.InstancePlayLimitException": NewInstancePlayLimitExceptionByNone(),
		"Microsoft.Xna.Framework.Audio.NoAudioHardwareException":   NewNoAudioHardwareExceptionByNone(),
	} {
		t.Run(name, func(t *testing.T) {
			if got := exception.ErrorCode(); got != eFail {
				t.Fatalf("ErrorCode = %d, want E_FAIL", got)
			}
			// The parameterless constructor supplies the BCL's own
			// Arg_ExternalException message rather than leaving it null, so
			// Message is NOT the "Exception of type ..." default.
			if got := exception.Message(); got != "External component has thrown an exception." {
				t.Fatalf("default Message = %q", got)
			}
			// ToString renders the HResult in parentheses as eight uppercase
			// hex digits, which the base form never does.
			want := name + " (0x80004005): External component has thrown an exception."
			if got := exception.ToString(); got != want {
				t.Fatalf("ToString = %q, want %q", got, want)
			}
		})
	}
}

// TestExternalExceptionToStringOmitsTheEndOfStackMarker is the second
// observable difference: the override does NOT append it.
func TestExternalExceptionToStringOmitsTheEndOfStackMarker(t *testing.T) {
	inner := NewNoMicrophoneConnectedExceptionByString("no microphone")
	outer := NewNoAudioHardwareExceptionByStringAndException("no hardware", inner)
	got := outer.ToString()
	if !strings.Contains(got, " ---> ") {
		t.Fatalf("ToString did not embed the inner exception: %q", got)
	}
	if strings.Contains(got, "End of inner exception stack trace") {
		t.Fatalf("the ExternalException override appended the base form's marker: %q", got)
	}
	// A direct System.Exception subclass in the same package DOES append it.
	direct := NewNoMicrophoneConnectedExceptionByStringAndException("outer", inner)
	if !strings.Contains(direct.ToString(), "--- End of inner exception stack trace ---") {
		t.Fatalf("the base form omitted its marker: %q", direct.ToString())
	}
}

// TestAudioExceptionsNameThemselves pins the derived class name the composed
// base is told.
func TestAudioExceptionsNameThemselves(t *testing.T) {
	if got := NewNoMicrophoneConnectedExceptionByNone().Message(); got !=
		"Exception of type 'Microsoft.Xna.Framework.Audio.NoMicrophoneConnectedException' was thrown." {
		t.Fatalf("default Message = %q", got)
	}
}
