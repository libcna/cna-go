package graphics

// ResourceCreatedEventArgs carries the object a GraphicsDevice reports as newly
// created.
//
// Its CLR base is System.EventArgs, modelled as a measured relationship rather
// than as Go embedding, so this is its own reference type and inherits no
// member. Microsoft.Xna.Framework.Graphics.dll shows a sealed class over one
// `assembly` object field with get_Resource as a single ldfld, so the getter
// cannot fail.
//
// It has **no public constructor**. The reference declares the constructor
// `assembly`, so it is not part of the public contract and CNA-Go does not
// invent one: only a GraphicsDevice raising ResourceCreated can produce this
// value. GraphicsDevice is a protected partial runtime type that raises no
// events yet, so nothing in the binding constructs one today. That is the same
// nonpublic-construction rule TouchCollection's enumerator already follows.
//
// System.Object projects to any, from the declared BCL table, so Resource is
// whatever the device reported without CNA-Go narrowing it.
type ResourceCreatedEventArgs struct {
	resource any
}

// newResourceCreatedEventArgs mirrors the reference's assembly constructor. It
// is unexported because the reference constructor is not public.
func newResourceCreatedEventArgs(resource any) *ResourceCreatedEventArgs {
	return &ResourceCreatedEventArgs{resource: resource}
}

// Resource returns the created object.
func (a *ResourceCreatedEventArgs) Resource() any {
	return a.resource
}

// ResourceDestroyedEventArgs carries the name and tag of a resource a
// GraphicsDevice reports as destroyed.
//
// It reports the destroyed resource's name and tag rather than the resource
// itself, because by the time the event is raised the object is gone. The
// reference is a sealed class over two `assembly` fields with both getters a
// single ldfld each, and its constructor is `assembly`, so CNA-Go projects the
// two getters and no constructor.
//
// The constructor's parameters are (name, tag) but its IL stores the tag first
// and the name second; the observable result is identical and the order is
// preserved here only because it costs nothing to be faithful.
type ResourceDestroyedEventArgs struct {
	tag  any
	name string
}

// newResourceDestroyedEventArgs mirrors the reference's assembly constructor,
// including its store order.
func newResourceDestroyedEventArgs(name string, tag any) *ResourceDestroyedEventArgs {
	args := &ResourceDestroyedEventArgs{}
	args.tag = tag
	args.name = name
	return args
}

// Name returns the destroyed resource's name.
func (a *ResourceDestroyedEventArgs) Name() string {
	return a.name
}

// Tag returns the destroyed resource's tag.
func (a *ResourceDestroyedEventArgs) Tag() any {
	return a.tag
}
