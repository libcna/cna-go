package graphics

import (
	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/servicebridge"
)

// This file projects the three GraphicsDeviceManager configuration properties
// whose types are declared in THIS package:
//
//	GraphicsProfile              -> Microsoft.Xna.Framework.Graphics.GraphicsProfile
//	PreferredBackBufferFormat    -> Microsoft.Xna.Framework.Graphics.SurfaceFormat
//	PreferredDepthStencilFormat  -> Microsoft.Xna.Framework.Graphics.DepthFormat
//
// The settled cross-package cycle rule puts an ancestor-namespace member whose
// type is a descendant-namespace type in the descendant package, named
// OwnerTypeMember. The manager itself stays in the framework package, and so
// does the state: it holds these three as the raw int32 the CLR enums are,
// because it cannot name the enums either. internal/servicebridge carries the
// value across, and the conversion happens here, where both sides are nameable.
//
// The getters are field reads and cannot fail; the setters store, raise the
// dirty flag and push to CNA's manager, exactly as their six framework-typed
// neighbours do.

// GraphicsDeviceManagerGraphicsProfile is
// GraphicsDeviceManager::get_GraphicsProfile, one `ldfld` over the field the
// constructor fills from ReadDefaultGraphicsProfile.
func GraphicsDeviceManagerGraphicsProfile(manager *framework.GraphicsDeviceManager) GraphicsProfile {
	value, _ := servicebridge.ReadManagerConfiguration(manager, servicebridge.ManagerGraphicsProfile)
	return GraphicsProfile(value)
}

// SetGraphicsDeviceManagerGraphicsProfile is set_GraphicsProfile: store, then
// isDeviceDirty = true. It validates nothing, so an undefined enum value is
// stored exactly as the reference stores it.
func SetGraphicsDeviceManagerGraphicsProfile(manager *framework.GraphicsDeviceManager, value GraphicsProfile) error {
	return servicebridge.WriteManagerConfiguration(manager, servicebridge.ManagerGraphicsProfile, int32(value))
}

// GraphicsDeviceManagerPreferredBackBufferFormat is
// get_PreferredBackBufferFormat. The constructor does not assign the field, so
// it starts at SurfaceFormat.Color, which is zero.
func GraphicsDeviceManagerPreferredBackBufferFormat(manager *framework.GraphicsDeviceManager) SurfaceFormat {
	value, _ := servicebridge.ReadManagerConfiguration(manager, servicebridge.ManagerPreferredBackBufferFormat)
	return SurfaceFormat(value)
}

// SetGraphicsDeviceManagerPreferredBackBufferFormat is
// set_PreferredBackBufferFormat.
func SetGraphicsDeviceManagerPreferredBackBufferFormat(manager *framework.GraphicsDeviceManager, value SurfaceFormat) error {
	return servicebridge.WriteManagerConfiguration(manager, servicebridge.ManagerPreferredBackBufferFormat, int32(value))
}

// GraphicsDeviceManagerPreferredDepthStencilFormat is
// get_PreferredDepthStencilFormat. The constructor stores DepthFormat.Depth24,
// which is 2 -- not the enum's zero value, and not None.
func GraphicsDeviceManagerPreferredDepthStencilFormat(manager *framework.GraphicsDeviceManager) DepthFormat {
	value, _ := servicebridge.ReadManagerConfiguration(manager, servicebridge.ManagerPreferredDepthStencilFormat)
	return DepthFormat(value)
}

// SetGraphicsDeviceManagerPreferredDepthStencilFormat is
// set_PreferredDepthStencilFormat.
func SetGraphicsDeviceManagerPreferredDepthStencilFormat(manager *framework.GraphicsDeviceManager, value DepthFormat) error {
	return servicebridge.WriteManagerConfiguration(manager, servicebridge.ManagerPreferredDepthStencilFormat, int32(value))
}
