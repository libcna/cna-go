package graphics

// Blend identifies an XNA colour or alpha blend term.
type Blend int32

const (
	BlendOne                     Blend = 0
	BlendZero                    Blend = 1
	BlendSourceColor             Blend = 2
	BlendInverseSourceColor      Blend = 3
	BlendSourceAlpha             Blend = 4
	BlendInverseSourceAlpha      Blend = 5
	BlendDestinationColor        Blend = 6
	BlendInverseDestinationColor Blend = 7
	BlendDestinationAlpha        Blend = 8
	BlendInverseDestinationAlpha Blend = 9
	BlendBlendFactor             Blend = 10
	BlendInverseBlendFactor      Blend = 11
	BlendSourceAlphaSaturation   Blend = 12
)
