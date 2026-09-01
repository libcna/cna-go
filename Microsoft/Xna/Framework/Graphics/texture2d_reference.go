package graphics

// ---------------------------------------------------------------------------
// Foundation 58 — CLR base substitutability at a Go parameter position.
// ---------------------------------------------------------------------------

// Texture2DReference is the Go projection of a PARAMETER position whose CLR
// type is Texture2D.
//
// # Why the parameter type is not *Texture2D
//
// In C#, `spriteBatch.Draw(renderTarget, position, color)` compiles because
// RenderTarget2D IS-A Texture2D. Go has no such relation, and CNA-Go refuses to
// fake one with embedding -- embedding promotes the base's whole method set, so
// a member the derived class overrides would silently keep the base's body.
//
// Foundation 40 measured the substitutability requirement of the whole profile
// and found it LATENT everywhere: positions existed, but no derived type was
// projected, so private composition was exactly sufficient. Foundation 58
// projected RenderTarget2D and the requirement went LIVE, at seven positions,
// all of them SpriteBatch.Draw's `texture`.
//
// # Why an interface, and not a conversion
//
// The alternative is a public conversion -- `renderTarget.AsTexture2D()` -- and
// it is refused for the reason the composition rule already refuses Base and
// Parent: it hands a consumer the base object, and it makes the Go call site
// stop looking like the C# one. With this interface the call site is identical:
//
//	spriteBatch.DrawByTexture2DAndVector2AndColor(renderTarget, position, color)
//
// # Why the method is unexported
//
// Only this module can satisfy the interface, so a consumer cannot hand a
// SpriteBatch an object CNA never made. That is the same guarantee the concrete
// *Texture2D parameter gave, kept while widening the position.
//
// # Where it does NOT apply
//
// Returns and property getters keep the concrete *Texture2D.
// `Texture2D::FromStream` returns a Texture2D and a caller uses every Texture2D
// member on it; returning an interface would take those members away to solve a
// problem returns do not have. The rule is per POSITION, and the verifier
// applies it only to parameters.
type Texture2DReference interface {
	// texture2D is the Texture2D half of whatever the value is. For a Texture2D
	// it is the value itself; for a RenderTarget2D it is the composed base,
	// which is the same logical native object.
	texture2D() *Texture2D
}

// texture2D makes a Texture2D its own reference.
func (t *Texture2D) texture2D() *Texture2D { return t }

// resolveTexture2D is the `ldarg` a CLR call site does for free: it takes
// whatever satisfies the position and yields the Texture2D the callee reads.
//
// It answers nil for BOTH a nil interface and an interface holding a typed nil,
// because the reference sees one null either way and its guard reports one
// message. A Go caller can write `var rt *RenderTarget2D; Draw(rt, ...)`, which
// is a non-nil interface holding a nil pointer, and that must reach the same
// ArgumentNullException as `Draw(nil, ...)`.
func resolveTexture2D(reference Texture2DReference) *Texture2D {
	if reference == nil {
		return nil
	}
	return reference.texture2D()
}
