package graphics

// TextureReference is the Go projection of a PARAMETER position whose CLR type
// is Texture.
//
// It is the SECOND family the substitutable-base rule reaches, and it exists
// for the same measured reason the first did: `TextureCollection::set_Item`
// takes a `Texture`, and in C# a Texture2D flows into that position because it
// IS-A Texture. Foundation 61 projected TextureCollection, which put a Texture
// position on a carrier CNA-Go projects, and Texture already had a projected
// derived type -- so its requirement went LIVE exactly as Texture2D's did.
//
// The rule is unchanged: exported so a consumer can name the parameter type,
// unexported method so only this module can satisfy it, and no public
// conversion anywhere.
type TextureReference interface {
	// textureBase is the Texture half of whatever the value is, which is the
	// same logical native object: CNA's texture routes accept the handle either
	// way. It is spelled `textureBase` rather than `texture` because both
	// Texture2D and RenderTarget2D already hold a FIELD of that name, and Go
	// has one identifier namespace for fields and methods on a type.
	textureBase() *Texture
}

// textureBase makes a Texture its own reference.
func (t *Texture) textureBase() *Texture { return t }

// textureBase answers with the Texture2D's composed base.
func (t *Texture2D) textureBase() *Texture {
	if t == nil {
		return nil
	}
	return t.texture
}

// textureBase answers across two composition links.
func (t *RenderTarget2D) textureBase() *Texture {
	if t == nil || t.texture == nil {
		return nil
	}
	return t.texture.textureBase()
}

// resolveTexture is the `ldarg` a CLR call site does for free. It answers nil
// for a nil interface AND for an interface holding a typed nil, because the
// reference sees one null either way.
func resolveTexture(reference TextureReference) *Texture {
	if reference == nil {
		return nil
	}
	return reference.textureBase()
}
