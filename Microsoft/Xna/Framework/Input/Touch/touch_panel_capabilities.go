package touch

// TouchPanelCapabilities is
// Microsoft.Xna.Framework.Input.Touch.TouchPanelCapabilities:
//
//	.class public sequential ansi sealed beforefieldinit TouchPanelCapabilities
//	    extends System.ValueType
//
// Two auto-properties with PRIVATE setters and no public constructor, so the
// only way a consumer obtains one is TouchPanel.GetCapabilities.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Input.Touch.dll   b0585224c18022c3...
//
// # On the Windows runtime the answer is a constant, and the IL says so outright
//
//	.method assembly static TouchPanelCapabilities GetCaps()
//	{
//	    .locals init (TouchPanelCapabilities V_0)
//	    ldloca.s V_0
//	    initobj  TouchPanelCapabilities
//	    ldloc.0
//	    ret
//	}
//
// Ten bytes: zero a local, return it. There is no probe, no p/invoke and no
// branch. `TouchPanel.GetCapabilities()` on the XNA 4.0 WINDOWS runtime
// therefore always reports `IsConnected == false` and `MaximumTouchCount == 0`,
// on every machine, including one with a working digitizer attached.
//
// That is not an omission in this projection -- it is the shipped behaviour of
// the assembly this binding targets, measured from its IL. See TouchPanel for
// what it means for the rest of the family, and for why CNA's working touch
// routes are deliberately left unbound.
//
// The consequence for the projection is pleasant: Go's zero value for this
// struct IS the answer, so nothing has to be assigned to produce it.
type TouchPanelCapabilities struct {
	isConnected       bool
	maximumTouchCount int32
}

// IsConnected is get_IsConnected. Always false on the Windows runtime.
func (c TouchPanelCapabilities) IsConnected() bool { return c.isConnected }

// MaximumTouchCount is get_MaximumTouchCount. Always zero on the Windows
// runtime.
func (c TouchPanelCapabilities) MaximumTouchCount() int32 { return c.maximumTouchCount }
