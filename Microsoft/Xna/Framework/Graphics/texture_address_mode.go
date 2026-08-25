package graphics

// TextureAddressMode identifies how XNA resolves texture coordinates outside
// the zero-to-one range.
type TextureAddressMode int32

const (
	TextureAddressModeWrap   TextureAddressMode = 0
	TextureAddressModeClamp  TextureAddressMode = 1
	TextureAddressModeMirror TextureAddressMode = 2
)
