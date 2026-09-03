package framework

// PreparingDeviceSettingsEventArgs is the argument
// GraphicsDeviceManager.PreparingDeviceSettings carries: the one
// GraphicsDeviceInformation the manager has chosen, handed to a consumer for
// inspection or mutation before the device is created.
//
// # What it is, exactly
//
// The CLR class is
//
//	.class public auto ansi beforefieldinit PreparingDeviceSettingsEventArgs
//	    extends [mscorlib]System.EventArgs
//
// with one private field, one public constructor that stores it, and one
// get-only property that returns it. There is no setter and no validation: a
// null argument is stored and handed straight back.
//
// System.EventArgs is a MAPPED base in the settled relationship table, which
// means it contributes no inherited projected surface and the derived type is
// its own Go pointer type. That is why nothing is composed here.
//
// # The information is not a copy
//
// The property is one `ldfld`, so a consumer that mutates the
// GraphicsDeviceInformation it is given is mutating the object the manager is
// about to build a device from. That is the entire purpose of the event, and it
// is what makes the args a carrier rather than a report.
type PreparingDeviceSettingsEventArgs struct {
	graphicsDeviceInformation *GraphicsDeviceInformation
}

// NewPreparingDeviceSettingsEventArgs is
// PreparingDeviceSettingsEventArgs::.ctor(GraphicsDeviceInformation), whose
// whole body is `base..ctor(); this.graphicsDeviceInformation = value;`. It
// validates nothing, so a nil information is stored exactly as the reference
// stores a null one.
func NewPreparingDeviceSettingsEventArgs(graphicsDeviceInformation *GraphicsDeviceInformation) *PreparingDeviceSettingsEventArgs {
	return &PreparingDeviceSettingsEventArgs{graphicsDeviceInformation: graphicsDeviceInformation}
}

// GraphicsDeviceInformation is
// PreparingDeviceSettingsEventArgs::get_GraphicsDeviceInformation, one field
// read of the object the constructor was given. The same object every call, and
// the manager's own.
func (a *PreparingDeviceSettingsEventArgs) GraphicsDeviceInformation() *GraphicsDeviceInformation {
	if a == nil {
		return nil
	}
	return a.graphicsDeviceInformation
}
