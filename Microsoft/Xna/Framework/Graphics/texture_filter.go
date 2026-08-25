package graphics

// TextureFilter identifies the minification, magnification, and mip filter
// combination of an XNA sampler.
type TextureFilter int32

const (
	TextureFilterLinear                     TextureFilter = 0
	TextureFilterPoint                      TextureFilter = 1
	TextureFilterAnisotropic                TextureFilter = 2
	TextureFilterLinearMipPoint             TextureFilter = 3
	TextureFilterPointMipLinear             TextureFilter = 4
	TextureFilterMinLinearMagPointMipLinear TextureFilter = 5
	TextureFilterMinLinearMagPointMipPoint  TextureFilter = 6
	TextureFilterMinPointMagLinearMipLinear TextureFilter = 7
	TextureFilterMinPointMagLinearMipPoint  TextureFilter = 8
)
