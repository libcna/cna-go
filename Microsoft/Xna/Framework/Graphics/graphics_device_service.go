package graphics

import framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"

// IGraphicsDeviceService is XNA's device-publication contract: the shape a
// component looks up in Game.Services to find the current GraphicsDevice and
// to learn when it is created, reset, or going away.
//
// It is the contract DrawableGameComponent.Initialize resolves out of
// Game.Services and throws when it is absent, which is why IGameComponent
// stays fallible while this contract does not.
//
// # Every operation is infallible
//
// The boundary is read from the reference implementor's IL in the assembly
// that consumes the contract, not from the word "device" in its name.
// Microsoft.Xna.Framework.Game.dll ships exactly one implementor,
// GraphicsDeviceManager, and its accessor is one field read:
//
//	get_GraphicsDevice
//	  ldarg.0
//	  ldfld GraphicsDevice GraphicsDeviceManager::device
//	  ret
//
// It hands over a stored reference. It does not create a device, reset one,
// query one, or reach native code, and it returns nil before a device exists
// exactly as the reference returns null. Nothing here can fail.
//
// That is deliberately a different verdict from IGraphicsDeviceManager, whose
// CreateDevice, BeginDraw and EndDraw genuinely cross into the runtime and
// which therefore stays fallible. Two contracts on the same class disagree
// because the boundary is read per contract.
//
// The four events carry an error from the settled event accessor projection
// rather than from this contract's boundary. In the reference they are the
// ordinary compiler-generated Delegate.Combine/Delegate.Remove pair over
// System.EventHandler`1<System.EventArgs>.
//
// # Declaring the contract publishes no device
//
// CNA-Go has no implementor. GraphicsDeviceManager remains a partial
// native-backed facade that raises none of these events, Game exposes no
// Services container, and nothing in the binding resolves or publishes this
// contract. The type exists so a consumer can name it and so a future
// implementation has an exact signature to satisfy.
type IGraphicsDeviceService interface {
	// GraphicsDevice is the device the service currently publishes, or nil
	// before one exists.
	GraphicsDevice() *GraphicsDevice

	AddDeviceCreatedHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error)
	RemoveDeviceCreatedHandler(subscription framework.EventSubscription) error
	AddDeviceDisposingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error)
	RemoveDeviceDisposingHandler(subscription framework.EventSubscription) error
	AddDeviceResetHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error)
	RemoveDeviceResetHandler(subscription framework.EventSubscription) error
	AddDeviceResettingHandler(handler framework.EventHandler[*framework.EventArgs]) (framework.EventSubscription, error)
	RemoveDeviceResettingHandler(subscription framework.EventSubscription) error
}
