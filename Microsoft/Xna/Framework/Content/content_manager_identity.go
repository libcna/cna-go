package content

import "fmt"

// The CLR `this` a composed ContentManager splits.
//
// # Three members need the object, and the IL says which
//
// ContentManager::Unload, Load<T> and ReadAsset<T> each open with
//
//	if (loadedAssets == null)
//	    throw new ObjectDisposedException(this.ToString());
//
// `ldarg.0; callvirt System.Object::ToString()` -- the receiver is the object,
// not a field of it. So a disposed ResourceContentManager must name
// ResourceContentManager, and a projection that answered ContentManager would
// tell a consumer the wrong type was disposed.
//
// That is the same shape GraphicsResource carries and it is settled: the base
// holds the outermost object in an unexported field, a derived constructor
// installs it, and every site reaches it through one accessor.

// contentManagerObject is what the CLR `this` answers with. It is unexported,
// so only this module can be one -- the contract declares no accessor for the
// base object and a consumer has no way to name the interface.
type contentManagerObject interface {
	clrTypeName() string
}

// clrTypeName answers for a bare ContentManager: the one nothing composes.
func (m *ContentManager) clrTypeName() string {
	return "Microsoft.Xna.Framework.Content.ContentManager"
}

// bindDerived is what a derived constructor calls to install the CLR `this`.
func (m *ContentManager) bindDerived(derived contentManagerObject) { m.derived = derived }

// self is `ldarg.0` as an OBJECT: the outermost object of the chain, which is
// the manager itself when nothing composes it.
//
// A nil receiver is a zero-value composed object whose base half was never
// built. It has no derived object to answer with, and the base's own
// clrTypeName reads no field, so answering with it names SOMETHING rather than
// panicking -- which is what the reference gives too, since no such half-built
// object exists in the CLR.
func (m *ContentManager) self() contentManagerObject {
	if m == nil || m.derived == nil {
		return m
	}
	return m.derived
}

// disposedError is the ObjectDisposedException the three sites raise, named
// with the RUNTIME type exactly as `this.ToString()` is.
func (m *ContentManager) disposedError() error {
	return fmt.Errorf("%w: %s", errContentManagerDisposed, m.self().clrTypeName())
}
