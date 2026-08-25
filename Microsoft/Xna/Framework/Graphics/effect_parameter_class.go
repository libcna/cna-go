package graphics

// EffectParameterClass identifies the value class of an XNA effect parameter.
type EffectParameterClass int32

const (
	EffectParameterClassScalar EffectParameterClass = 0
	EffectParameterClassVector EffectParameterClass = 1
	EffectParameterClassMatrix EffectParameterClass = 2
	EffectParameterClassObject EffectParameterClass = 3
	EffectParameterClassStruct EffectParameterClass = 4
)
