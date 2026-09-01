package graphics

import (
	"errors"
	"fmt"

	"github.com/openeggbert/cna-go/internal/interop"
)

// ---------------------------------------------------------------------------
// Foundation 61 — the device's two texture and two sampler collections.
// ---------------------------------------------------------------------------

// errCollectionArgumentOutOfRange projects the ArgumentOutOfRangeException both
// indexers raise. The reference's constructor takes the PARAMETER NAME and no
// message -- `newobj ArgumentOutOfRangeException::.ctor(string)` on the literal
// "index" -- so the projection carries the name and nothing else.
var errCollectionArgumentOutOfRange = errors.New("index is out of range")

// errTextureSlotNotNameable is a CNA-Go-only refusal with no reference
// counterpart, and it is the one measured divergence in this family.
//
// XNA's TextureCollection getter reads the D3D slot and recovers the managed
// wrapper with Texture3D::GetManagedObject and its two siblings -- a
// pointer-to-object map. CNA states plainly that it has no such map:
//
//	There is deliberately no route from a native object back to a handle,
//	here or anywhere else in this ABI.
//
// and tells a binding what to do instead: cache what you bind, and use the
// slot's `bound` flag to tell "something else owns this slot now" from "the
// slot is empty". A slot CNA filled itself -- a SpriteBatch flush, for example
// -- reports bound with an INVALID handle, and CNA-Go's cache holds nothing for
// it. Answering nil would say "empty", which is false; answering the stale
// cache would name the wrong texture. It reports instead.
var errTextureSlotNotNameable = errors.New(
	"the graphics device holds a texture in this slot that CNA-Go did not bind, and CNA has no route from a native texture back to a handle")

// TextureCollection is Microsoft.Xna.Framework.Graphics.TextureCollection: one
// indexed property over the device's texture sampler slots.
//
//	.class public sealed TextureCollection extends System.Object
//	  .field private GraphicsDevice _parent
//	  .field private int32 _textureOffset
//	  .field assembly int32 _maxTextures
//
// Its constructor is `assembly`, so CNA-Go declares none: only a GraphicsDevice
// hands one out, which is the same nonpublic-construction rule
// ResourceCreatedEventArgs already follows.
type TextureCollection struct {
	device *GraphicsDevice
	// stage is which of the device's two collections this is. The reference
	// spells the same distinction as `_textureOffset`; CNA spells it as
	// CNA_SHADER_STAGE_PIXEL and CNA_SHADER_STAGE_VERTEX.
	stage uint32
	// bound is the cache CNA's own documentation asks a binding to keep. The
	// reference needs none because it can recover a wrapper from a D3D pointer.
	bound [interop.MaxTextureSlots]TextureReference
}

// Item is TextureCollection::get_Item(Int32).
//
//	if (index < 0 || index >= _maxTextures)
//	    throw new ArgumentOutOfRangeException("index");
//
// then it reads the D3D slot and answers with the wrapper for whatever is
// there, or null for an empty slot.
//
// # The return type is the BASE, and that is a downcast Go cannot express
//
// The reference's getter is declared to return `Texture`, so a C# consumer who
// bound a Texture2D gets a Texture-typed reference and must downcast to use it
// as one. CNA-Go answers with the composed `*Texture` -- the same logical
// object, and the same two members `Texture` declares -- and a Go consumer has
// no downcast to recover the Texture2D with. That is a GO LANGUAGE LIMITATION
// at a base-typed RETURN, recorded rather than worked around: widening the
// return to TextureReference would take away the two members the CLR's own
// static type gives, to buy an assertion the CLR spells as a cast.
func (c *TextureCollection) Item(index int32) (*Texture, error) {
	if c == nil || c.device == nil {
		return nil, errors.New("TextureCollection is nil")
	}
	if index < 0 || index >= interop.MaxTextureSlots {
		return nil, fmt.Errorf("%w: index", errCollectionArgumentOutOfRange)
	}
	slot, err := c.device.device.TextureSlot(c.stage, uint32(index))
	if err != nil {
		return nil, err
	}
	if !slot.Bound {
		return nil, nil
	}
	cached := c.bound[index]
	if cached == nil {
		return nil, errTextureSlotNotNameable
	}
	return resolveTexture(cached), nil
}

// SetItem is TextureCollection::set_Item(Int32, Texture), whose guard is the
// getter's. A null empties the slot, which is what the reference's null branch
// does and what CNA's CNA_INVALID_HANDLE means.
func (c *TextureCollection) SetItem(index int32, value TextureReference) error {
	if c == nil || c.device == nil {
		return errors.New("TextureCollection is nil")
	}
	if index < 0 || index >= interop.MaxTextureSlots {
		return fmt.Errorf("%w: index", errCollectionArgumentOutOfRange)
	}
	texture := resolveTexture(value)
	var resource *interop.Resource
	if texture != nil {
		resource = texture.nativeResource()
		if resource == nil {
			return interop.ErrDisposed
		}
	}
	if err := c.device.device.SetTextureSlot(c.stage, uint32(index), resource); err != nil {
		return err
	}
	if texture == nil {
		c.bound[index] = nil
	} else {
		c.bound[index] = value
	}
	return nil
}

// SamplerStateCollection is
// Microsoft.Xna.Framework.Graphics.SamplerStateCollection, the same shape over
// the device's sampler slots. Its constructor is `assembly` too.
type SamplerStateCollection struct {
	device *GraphicsDevice
	stage  uint32
	// bound is the same cache, and for the same reason: the reference's getter
	// answers with the state OBJECT, and CNA holds only the values.
	bound [interop.MaxSamplerSlots]*SamplerState
}

// Item is SamplerStateCollection::get_Item(Int32).
//
// The slot always holds a state -- CNA answers with a complete descriptor for
// every slot -- so there is no "empty" case here and no not-nameable one: a
// slot CNA-Go has not written answers with the state CNA reports, materialised
// as a fresh SamplerState. That is a divergence from the reference, whose
// getter answers with the object that was set, and it is confined to slots
// nothing set: once SetItem has run, the object comes back.
func (c *SamplerStateCollection) Item(index int32) (*SamplerState, error) {
	if c == nil || c.device == nil {
		return nil, errors.New("SamplerStateCollection is nil")
	}
	if index < 0 || index >= interop.MaxSamplerSlots {
		return nil, fmt.Errorf("%w: index", errCollectionArgumentOutOfRange)
	}
	if cached := c.bound[index]; cached != nil {
		return cached, nil
	}
	value, err := c.device.device.SamplerSlot(c.stage, uint32(index))
	if err != nil {
		return nil, err
	}
	state := NewSamplerState()
	state.filter = TextureFilter(value.Filter)
	state.addressU = TextureAddressMode(value.AddressU)
	state.addressV = TextureAddressMode(value.AddressV)
	state.addressW = TextureAddressMode(value.AddressW)
	state.maxAnisotropy = value.MaxAnisotropy
	state.maxMipLevel = value.MaxMipLevel
	state.mipMapLevelOfDetailBias = value.MipMapLevelOfDetailBias
	// A state the device already holds is bound by definition.
	state.resource.device = c.device
	state.isBound = true
	return state, nil
}

// SetItem is SamplerStateCollection::set_Item(Int32, SamplerState). A null is
// the reference's ArgumentNullException, not a default: SetRenderState
// substitutes LinearClamp before it reaches here, and the indexer itself
// refuses one.
func (c *SamplerStateCollection) SetItem(index int32, value *SamplerState) error {
	if c == nil || c.device == nil {
		return errors.New("SamplerStateCollection is nil")
	}
	if index < 0 || index >= interop.MaxSamplerSlots {
		return fmt.Errorf("%w: index", errCollectionArgumentOutOfRange)
	}
	if value == nil {
		return nullNotAllowedParameter("value")
	}
	if value.IsDisposed() {
		return objectDisposedState(value.clrTypeName())
	}
	if err := c.device.device.SetSamplerSlot(c.stage, uint32(index), value.interopValue()); err != nil {
		return err
	}
	value.resource.device = c.device
	value.isBound = true
	c.bound[index] = value
	return nil
}

// The four GraphicsDevice properties. Each answers with the SAME collection
// object every call, which is what the reference's fields promise, so the
// facade builds each one lazily and keeps it.

// Textures is GraphicsDevice::get_Textures, the pixel-shader collection.
func (d *GraphicsDevice) Textures() (*TextureCollection, error) {
	return d.textureCollection(&d.textures, interop.ShaderStagePixel)
}

// VertexTextures is GraphicsDevice::get_VertexTextures.
func (d *GraphicsDevice) VertexTextures() (*TextureCollection, error) {
	return d.textureCollection(&d.vertexTextures, interop.ShaderStageVertex)
}

func (d *GraphicsDevice) textureCollection(slot **TextureCollection, stage uint32) (*TextureCollection, error) {
	if d == nil || d.device == nil {
		return nil, errors.New("GraphicsDevice is nil")
	}
	if err := d.device.Live(); err != nil {
		return nil, err
	}
	if *slot == nil {
		*slot = &TextureCollection{device: d, stage: stage}
	}
	return *slot, nil
}

// SamplerStates is GraphicsDevice::get_SamplerStates, the pixel-shader
// collection.
func (d *GraphicsDevice) SamplerStates() (*SamplerStateCollection, error) {
	return d.samplerCollection(&d.samplerStates, interop.ShaderStagePixel)
}

// VertexSamplerStates is GraphicsDevice::get_VertexSamplerStates.
func (d *GraphicsDevice) VertexSamplerStates() (*SamplerStateCollection, error) {
	return d.samplerCollection(&d.vertexSamplerStates, interop.ShaderStageVertex)
}

func (d *GraphicsDevice) samplerCollection(slot **SamplerStateCollection, stage uint32) (*SamplerStateCollection, error) {
	if d == nil || d.device == nil {
		return nil, errors.New("GraphicsDevice is nil")
	}
	if err := d.device.Live(); err != nil {
		return nil, err
	}
	if *slot == nil {
		*slot = &SamplerStateCollection{device: d, stage: stage}
	}
	return *slot, nil
}
