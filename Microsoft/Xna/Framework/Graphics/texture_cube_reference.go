package graphics

// TextureCubeReference is the Go projection of a PARAMETER position whose CLR
// type is TextureCube.
//
// It is the FOURTH family the substitutable-base rule reaches, and its
// requirement went LIVE for exactly the reason the first three did: a position
// typed TextureCube appeared on a carrier CNA-Go projects, and TextureCube
// already had a projected derived type.
//
// The position is `EnvironmentMapEffect::EnvironmentMap`'s setter, and it is
// the only one in the whole profile -- `EffectParameter::GetValueTextureCube`
// is a RETURN and keeps the concrete pointer, because a caller uses every
// TextureCube member on what it hands back. Before Foundation 81 nothing named
// a TextureCube at a parameter position, so the requirement was LATENT and the
// measurement said so.
//
// The rule is unchanged: exported so a consumer can name the parameter type,
// unexported method so only this module can satisfy it, and no public
// conversion anywhere.
type TextureCubeReference interface {
	// textureCube is the TextureCube half of whatever the value is: the value
	// itself for a TextureCube, and the composed base for a RenderTargetCube --
	// the same logical native object either way.
	textureCube() *TextureCube
}

// textureCube makes a TextureCube its own reference.
func (t *TextureCube) textureCube() *TextureCube { return t }

// textureCube answers with the RenderTargetCube's composed base.
func (t *RenderTargetCube) textureCube() *TextureCube {
	if t == nil {
		return nil
	}
	return t.cube
}

// resolveTextureCube is the `ldarg` a CLR call site does for free. It answers
// nil for a nil interface AND for an interface holding a typed nil, because the
// reference sees one null either way.
func resolveTextureCube(reference TextureCubeReference) *TextureCube {
	if reference == nil {
		return nil
	}
	return reference.textureCube()
}
