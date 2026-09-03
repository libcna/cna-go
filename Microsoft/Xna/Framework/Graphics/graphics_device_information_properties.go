package graphics

import (
	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/servicebridge"
)

// This file projects the three GraphicsDeviceInformation properties whose types
// are declared in THIS package:
//
//	Adapter                 -> Microsoft.Xna.Framework.Graphics.GraphicsAdapter
//	GraphicsProfile         -> Microsoft.Xna.Framework.Graphics.GraphicsProfile
//	PresentationParameters  -> Microsoft.Xna.Framework.Graphics.PresentationParameters
//
// The settled cross-package cycle rule puts an ancestor-namespace member whose
// type is a descendant-namespace type in the descendant package, named
// OwnerTypeMember. The type itself stays in the framework package, and so does
// the state: it holds the two references as `any` and the profile as the raw
// int32 the CLR enum is, because it cannot name any of the three.
//
// This is the same shape the three GraphicsDeviceManager configuration
// properties already take; see graphics_device_manager_configuration.go.

// GraphicsDeviceInformationAdapter is
// GraphicsDeviceInformation::get_Adapter, one `ldfld`. The constructor fills the
// field from GraphicsAdapter.DefaultAdapter, and Clone ALIASES it rather than
// copying, so a clone and its source name the same adapter object.
func GraphicsDeviceInformationAdapter(information *framework.GraphicsDeviceInformation) *GraphicsAdapter {
	adapter, _ := servicebridge.ReadDeviceInformationAdapter(information)
	typed, _ := adapter.(*GraphicsAdapter)
	return typed
}

// SetGraphicsDeviceInformationAdapter is
// GraphicsDeviceInformation::set_Adapter, and it reproduces a REFERENCE BUG.
//
// The setter's guard tests `this.adapter` -- the field it is about to overwrite
// -- and not the `value` it was given:
//
//	IL_0001: ldfld adapter
//	IL_0006: brtrue.s IL_0018
//	IL_0008: throw ArgumentNullException("value", NoNullUseDefaultAdapter)
//
// So assigning null SUCCEEDS whenever the current adapter is non-null, which it
// always is after the constructor, and the "use GraphicsAdapter.DefaultAdapter
// instead" message can only ever be raised on an information whose adapter is
// ALREADY null. Correcting it would make CNA-Go refuse an assignment the
// reference accepts.
func SetGraphicsDeviceInformationAdapter(information *framework.GraphicsDeviceInformation, value *GraphicsAdapter) error {
	if value == nil {
		return servicebridge.WriteDeviceInformationAdapter(information, nil)
	}
	return servicebridge.WriteDeviceInformationAdapter(information, value)
}

// GraphicsDeviceInformationGraphicsProfile is
// GraphicsDeviceInformation::get_GraphicsProfile, one `ldfld`. The constructor
// does not assign it, so it starts at GraphicsProfile.Reach, which is zero.
func GraphicsDeviceInformationGraphicsProfile(information *framework.GraphicsDeviceInformation) GraphicsProfile {
	return GraphicsProfile(servicebridge.ReadDeviceInformationGraphicsProfile(information))
}

// SetGraphicsDeviceInformationGraphicsProfile is set_GraphicsProfile, one
// `stfld` that validates nothing, so an undefined enum value is stored exactly
// as the reference stores it.
func SetGraphicsDeviceInformationGraphicsProfile(information *framework.GraphicsDeviceInformation, value GraphicsProfile) {
	servicebridge.WriteDeviceInformationGraphicsProfile(information, int32(value))
}

// GraphicsDeviceInformationPresentationParameters is
// GraphicsDeviceInformation::get_PresentationParameters, one `ldfld` of the
// object the constructor allocated. Clone DEEP-copies it, which is the one
// asymmetry with Adapter.
func GraphicsDeviceInformationPresentationParameters(information *framework.GraphicsDeviceInformation) *PresentationParameters {
	parameters, _ := servicebridge.ReadDeviceInformationPresentationParameters(information)
	typed, _ := parameters.(*PresentationParameters)
	return typed
}

// SetGraphicsDeviceInformationPresentationParameters is
// set_PresentationParameters, one `stfld` with no validation: a null is stored.
func SetGraphicsDeviceInformationPresentationParameters(
	information *framework.GraphicsDeviceInformation, value *PresentationParameters,
) {
	if value == nil {
		servicebridge.WriteDeviceInformationPresentationParameters(information, nil)
		return
	}
	servicebridge.WriteDeviceInformationPresentationParameters(information, value)
}
