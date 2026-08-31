# Foundation 40 — the base-typed public signature inventory

Foundation 33 recorded twelve XNA-to-XNA base relationships, 41 derived types,
25 blockers and 245 unprojected inherited public members, and stopped. Every
milestone since has named the same architecture decision as the largest one
outstanding.

This milestone does not make that decision. It makes the **measurement the
decision depends on**, which had never been taken.

## The question

The obvious next move is to pick a composition rule for a derived XNA class, and
the obvious worry is that private composition cannot express CLR
substitutability. In CLR a `DrawableGameComponent` may be passed anywhere a
`GameComponent` is named; a Go struct holding a private `*GameComponent` cannot
be.

But that worry is only real **where the profile actually names a base class in a
public signature**. If nothing in the whole contract takes or returns a
`GameComponent`, then no consumer can ever be handed a signature that requires a
`DrawableGameComponent` to stand in for one, and private composition with
explicit forwarding is not a compromise — it is exactly sufficient.

That is a counting question. It is answered here by counting, mechanically, from
the pinned contract, rather than by reasoning about what a framework probably
does.

## What is counted

Every **public** member of every type in the profile, at every position whose CLR
type names a class another class in the same profile derives from:

- a parameter type,
- a return type,
- a property, field or event type,
- and any of those behind an array, a by-reference marker, or inside a generic
  argument — `Texture2D[]`, `Texture2D&` and `List<Texture2D>` are all positions
  a derived value would have to flow through.

The relationships themselves are read from the contract's own `baseType` fields,
never from the hand-written registry, and the two are cross-checked: a
relationship in one and not the other is a `BASE_MAPPING_MISMATCH`, so neither
can go stale alone.

## The result

```text
XNA_BASE_TYPED_SIGNATURE_POSITIONS      51
XNA_BASE_SUBSTITUTABILITY_NONE           3
XNA_BASE_SUBSTITUTABILITY_LATENT         9
XNA_BASE_SUBSTITUTABILITY_LIVE           0
```

| family | positions | on projected carriers | derived | projected derived | requirement |
| -- | --: | --: | --: | --: | -- |
| `GameComponent` | **0** | 0 | 2 | 0 | `NONE` |
| `Graphics.GraphicsResource` | **0** | 0 | 11 | 1 | `NONE` |
| `Design.MathTypeConverter` | **0** | 0 | 12 | 0 | `NONE` |
| `Graphics.Texture2D` | 17 | 9 | 1 | 0 | `LATENT` |
| `Graphics.Effect` | 11 | 2 | 6 | 0 | `LATENT` |
| `Graphics.VertexBuffer` | 8 | 2 | 1 | 0 | `LATENT` |
| `Content.ContentTypeReader` | 5 | 0 | 1 | 0 | `LATENT` |
| `Graphics.Texture` | 3 | 0 | 3 | 1 | `LATENT` |
| `Content.ContentManager` | 2 | 1 | 1 | 0 | `LATENT` |
| `Graphics.IndexBuffer` | 2 | 1 | 1 | 0 | `LATENT` |
| `Graphics.TextureCube` | 2 | 0 | 1 | 0 | `LATENT` |
| `Audio.SoundEffectInstance` | 1 | 0 | 1 | 0 | `LATENT` |

### Three families are named in zero public positions

`GameComponent`, `GraphicsResource` and `MathTypeConverter` carry **25 of the 41
derived types** between them, and not one public signature in the entire XNA 4.0
Windows profile names any of the three.

For those families the answer is not "private composition is good enough". It is
that **no public reference abstraction can be justified by anything in the
contract**, because there is no position for a derived value to flow through.
Private named composition with explicit forwarding is exactly sufficient, and
stays sufficient unless the pinned contract itself changes.

That is the finding this milestone exists to produce, and it is what makes
`GameComponent → DrawableGameComponent` a safe first proof rather than a
gamble.

### Nine families are latent, none is live

A requirement is **LIVE** only when both ends exist: a carrier CNA-Go projects
that names the base, *and* a derived type CNA-Go projects. Today no family has
both.

`Texture2D` is the closest and is worth naming precisely. It has 17 positions,
nine of them on `SpriteBatch`, which CNA-Go projects — so half the requirement is
already real. It stays latent only because its one derived type,
`RenderTarget2D`, is not projected. **Projecting `RenderTarget2D` while
`SpriteBatch.Draw` takes a `Texture2D` is exactly what would make it live**, and
a test says so in those words, so the day it changes is the day the suite
reports it.

## What this does and does not settle

It settles that the XNA-to-XNA inheritance architecture can proceed with private
named composition plus explicit measured forwarding, starting with the family
that has no requirement at all and eleven others that have no live one.

It does not settle what a public reference abstraction would look like when a
family does go live. That decision stays deferred — and it is now deferred
against a number rather than against a hope.
