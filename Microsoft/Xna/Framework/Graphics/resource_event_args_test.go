package graphics

import (
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// TestResourceCarriersHaveNoPublicConstructor is the point of these two types:
// the reference declares both constructors `assembly`, so they are not part of
// the public contract and CNA-Go invents neither. Only the unexported
// construction path exists, and only a GraphicsDevice raising the event would
// use it.
func TestResourceCarriersHaveNoPublicConstructor(t *testing.T) {
	created := newResourceCreatedEventArgs("the resource")
	if created.Resource() != "the resource" {
		t.Fatalf("Resource() = %v", created.Resource())
	}
	// System.Object projects to any, so nothing is narrowed and nil is carried
	// exactly as the reference carries a null.
	if newResourceCreatedEventArgs(nil).Resource() != nil {
		t.Fatal("a nil resource did not read back as nil")
	}

	destroyed := newResourceDestroyedEventArgs("name", 42)
	if destroyed.Name() != "name" || destroyed.Tag() != 42 {
		t.Fatalf("Name()=%q Tag()=%v", destroyed.Name(), destroyed.Tag())
	}
	// The reference stores the tag before the name; both read back regardless.
	if newResourceDestroyedEventArgs("", nil).Name() != "" {
		t.Fatal("an empty name did not read back")
	}
	if newResourceDestroyedEventArgs("", nil).Tag() != nil {
		t.Fatal("a nil tag did not read back as nil")
	}
}

// TestResourceCarriersKeepReferenceSemantics proves the CLR class projection:
// two variables naming one instance agree, two constructions do not alias, and
// neither type is assignable to its CLR base, because the base survives as a
// measured relationship rather than as Go embedding.
func TestResourceCarriersKeepReferenceSemantics(t *testing.T) {
	created := newResourceCreatedEventArgs("shared")
	alias := created
	if alias != created || alias.Resource() != created.Resource() {
		t.Fatal("aliasing a carrier did not observe one instance")
	}
	if newResourceCreatedEventArgs("shared") == created {
		t.Fatal("two constructions produced one instance")
	}

	for _, carrier := range []any{
		newResourceCreatedEventArgs(nil),
		newResourceDestroyedEventArgs("", nil),
	} {
		if _, isBase := carrier.(*framework.EventArgs); isBase {
			t.Fatalf("%T is assignable to its CLR base; the base was faked in Go", carrier)
		}
	}
}

// TestResourceCarriersAreTheProjectedEventArgumentTypes pins the reason these
// two types exist: they are the exact generic arguments GraphicsDevice's two
// resource events declare, so an event handler receives them by pointer.
func TestResourceCarriersAreTheProjectedEventArgumentTypes(t *testing.T) {
	var createdHandler framework.EventHandler[*ResourceCreatedEventArgs] = func(sender any, args *ResourceCreatedEventArgs) error {
		if args.Resource() != "resource" {
			t.Fatalf("Resource() = %v", args.Resource())
		}
		return nil
	}
	var destroyedHandler framework.EventHandler[*ResourceDestroyedEventArgs] = func(sender any, args *ResourceDestroyedEventArgs) error {
		if args.Name() != "gone" {
			t.Fatalf("Name() = %q", args.Name())
		}
		return nil
	}

	var createdSource framework.EventSource[*ResourceCreatedEventArgs]
	var destroyedSource framework.EventSource[*ResourceDestroyedEventArgs]
	device := &struct{ name string }{name: "a device that does not exist yet"}

	if _, err := createdSource.Add(createdHandler); err != nil {
		t.Fatal(err)
	}
	if _, err := destroyedSource.Add(destroyedHandler); err != nil {
		t.Fatal(err)
	}
	if err := createdSource.Raise(device, newResourceCreatedEventArgs("resource")); err != nil {
		t.Fatal(err)
	}
	if err := destroyedSource.Raise(device, newResourceDestroyedEventArgs("gone", nil)); err != nil {
		t.Fatal(err)
	}
}
