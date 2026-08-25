package graphics

// EffectParameterType identifies the value type of an XNA effect parameter.
type EffectParameterType int32

const (
	EffectParameterTypeVoid        EffectParameterType = 0
	EffectParameterTypeBool        EffectParameterType = 1
	EffectParameterTypeInt32       EffectParameterType = 2
	EffectParameterTypeSingle      EffectParameterType = 3
	EffectParameterTypeString      EffectParameterType = 4
	EffectParameterTypeTexture     EffectParameterType = 5
	EffectParameterTypeTexture1D   EffectParameterType = 6
	EffectParameterTypeTexture2D   EffectParameterType = 7
	EffectParameterTypeTexture3D   EffectParameterType = 8
	EffectParameterTypeTextureCube EffectParameterType = 9
)
