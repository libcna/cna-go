package graphics

import (
	"strings"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// The eight XNA exception types share one composed base, so the behaviour under
// test is the same for all of them and is measured here on the three this
// package declares plus one ExternalException subclass reached through the
// interface. The Audio, Content and Storage packages assert their own class
// names; what cannot be measured from one package is that a DERIVED type names
// ITSELF, which is exactly what these tests pin.

func TestGraphicsExceptionsNameThemselves(t *testing.T) {
	cases := map[string]framework.ExceptionReference{
		"Microsoft.Xna.Framework.Graphics.DeviceLostException":               NewDeviceLostExceptionByNone(),
		"Microsoft.Xna.Framework.Graphics.DeviceNotResetException":           NewDeviceNotResetExceptionByNone(),
		"Microsoft.Xna.Framework.Graphics.NoSuitableGraphicsDeviceException": NewNoSuitableGraphicsDeviceExceptionByNone(),
	}
	for name, exception := range cases {
		t.Run(name, func(t *testing.T) {
			// Exception::get_Message renders the pinned default sentence from
			// GetClassName(), which for a derived type is the DERIVED name. A
			// composed base that named System.Exception would be wrong here and
			// nowhere else.
			want := "Exception of type '" + name + "' was thrown."
			if got := exception.Message(); got != want {
				t.Fatalf("default Message = %q, want %q", got, want)
			}
			// ToString is the class name, then ": ", then that same message.
			if got := exception.ToString(); got != name+": "+want {
				t.Fatalf("ToString = %q", got)
			}
			// A supplied message replaces the default entirely.
			if exception.InnerException() != nil || exception.StackTrace() != "" {
				t.Fatal("a fresh exception reports an inner exception or a stack trace")
			}
		})
	}
}

func TestGraphicsExceptionMessageAndChaining(t *testing.T) {
	inner := NewDeviceLostExceptionByString("the device was lost")
	outer := NewDeviceNotResetExceptionByStringAndException("and could not be reset", inner)

	if outer.Message() != "and could not be reset" {
		t.Fatalf("Message = %q", outer.Message())
	}
	if outer.InnerException() != framework.ExceptionReference(inner) {
		t.Fatal("the chain did not survive")
	}
	if outer.GetBaseException() != framework.ExceptionReference(inner) {
		t.Fatal("GetBaseException did not walk to the deepest exception")
	}
	// The downcast a C# consumer writes as a catch clause is a Go type
	// assertion, and it is the whole reason System.Exception widens at RETURN
	// positions too.
	recovered, ok := outer.InnerException().(*DeviceLostException)
	if !ok || recovered != inner {
		t.Fatal("the inner exception could not be asserted back to its concrete type")
	}
	want := "Microsoft.Xna.Framework.Graphics.DeviceNotResetException: and could not be reset" +
		" ---> Microsoft.Xna.Framework.Graphics.DeviceLostException: the device was lost" +
		"\r\n   --- End of inner exception stack trace ---"
	if got := outer.ToString(); got != want {
		t.Fatalf("nested ToString =\n%q\nwant\n%q", got, want)
	}
	// An empty message is NOT the default: the field is set, so it renders
	// empty and ToString has no ": " at all.
	empty := NewDeviceLostExceptionByString("")
	if empty.Message() != "" {
		t.Fatalf("empty Message = %q", empty.Message())
	}
	if got := empty.ToString(); got != "Microsoft.Xna.Framework.Graphics.DeviceLostException" {
		t.Fatalf("empty ToString = %q", got)
	}
}

func TestGraphicsExceptionsCarryTheInheritedStoredMembers(t *testing.T) {
	exception := NewNoSuitableGraphicsDeviceExceptionByString("no device")
	exception.SetHelpLink("http://example.invalid")
	exception.SetSource("cna-go")
	if exception.HelpLink() != "http://example.invalid" || exception.Source() != "cna-go" {
		t.Fatalf("HelpLink/Source = %q/%q", exception.HelpLink(), exception.Source())
	}
	// GetType answers the RUNTIME type, which is what the composed base holds
	// the CLR `this` for.
	if got := exception.GetType(); got == nil || got.String() != "*graphics.NoSuitableGraphicsDeviceException" {
		t.Fatalf("GetType = %v", got)
	}
	// Two different XNA exception types are never the same type, which is the
	// property a catch clause depends on.
	if exception.GetType() == NewDeviceLostExceptionByNone().GetType() {
		t.Fatal("two exception types report the same runtime type")
	}
}

// TestExternalExceptionSubclassesRenderTheirErrorCode is the one behaviour the
// ExternalException base adds beyond System.Exception's, seen from a package
// that declares none of them: the ToString override.
func TestExternalExceptionSubclassesRenderTheirErrorCode(t *testing.T) {
	// Reached through the interface, because this package declares no
	// ExternalException subclass -- which is itself the point: one interface,
	// eight implementors, four packages.
	var reference framework.ExceptionReference = NewDeviceLostExceptionByString("base form")
	if strings.Contains(reference.ToString(), "0x") {
		t.Fatal("a direct System.Exception subclass rendered an HResult")
	}
}
