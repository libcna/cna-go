package content

import "testing"

// TestContentLoadExceptionNamesItselfAndChains is the Content package's share
// of the exception family. ContentLoadException is one of the two types that
// also declare a protected deserialization constructor, which is recorded as a
// BLOCKED_DECLARED_MEMBER rather than projected.
func TestContentLoadExceptionNamesItselfAndChains(t *testing.T) {
	const name = "Microsoft.Xna.Framework.Content.ContentLoadException"
	if got := NewContentLoadExceptionByNone().Message(); got != "Exception of type '"+name+"' was thrown." {
		t.Fatalf("default Message = %q", got)
	}
	inner := NewContentLoadExceptionByString("the asset is missing")
	outer := NewContentLoadExceptionByStringAndException("could not load", inner)
	if outer.InnerException() != inner || outer.GetBaseException() != inner {
		t.Fatal("the chain did not survive")
	}
	if got := outer.Message(); got != "could not load" {
		t.Fatalf("Message = %q", got)
	}
	if got := outer.GetType(); got == nil || got.String() != "*content.ContentLoadException" {
		t.Fatalf("GetType = %v", got)
	}
}
