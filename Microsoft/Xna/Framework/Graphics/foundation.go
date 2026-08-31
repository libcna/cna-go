package graphics

import (
	"errors"
	"fmt"
	"io"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
	"github.com/openeggbert/cna-go/internal/interop"
	"github.com/openeggbert/cna-go/internal/servicebridge"
)

// SpriteEffects is the XNA sprite mirroring flags type.
// xna:flags
type SpriteEffects int32

const (
	SpriteEffectsNone             SpriteEffects = 0
	SpriteEffectsFlipHorizontally SpriteEffects = 1
	SpriteEffectsFlipVertically   SpriteEffects = 2
)

// SpriteSortMode controls SpriteBatch ordering.
type SpriteSortMode int32

const (
	SpriteSortModeDeferred    SpriteSortMode = 0
	SpriteSortModeImmediate   SpriteSortMode = 1
	SpriteSortModeTexture     SpriteSortMode = 2
	SpriteSortModeBackToFront SpriteSortMode = 3
	SpriteSortModeFrontToBack SpriteSortMode = 4
)

// Viewport is the value snapshot returned by the native graphics device.
type Viewport struct {
	x, y          int32
	width, height int32
	minDepth      float32
	maxDepth      float32
}

func NewViewportByInt32AndInt32AndInt32AndInt32(x, y, width, height int32) Viewport {
	return Viewport{x: x, y: y, width: width, height: height, maxDepth: 1}
}

func NewViewportByRectangle(bounds framework.Rectangle) Viewport {
	return NewViewportByInt32AndInt32AndInt32AndInt32(bounds.X, bounds.Y, bounds.Width, bounds.Height)
}

func (v Viewport) X() int32                   { return v.x }
func (v *Viewport) SetX(value int32)          { v.x = value }
func (v Viewport) Y() int32                   { return v.y }
func (v *Viewport) SetY(value int32)          { v.y = value }
func (v Viewport) Width() int32               { return v.width }
func (v *Viewport) SetWidth(value int32)      { v.width = value }
func (v Viewport) Height() int32              { return v.height }
func (v *Viewport) SetHeight(value int32)     { v.height = value }
func (v Viewport) MinDepth() float32          { return v.minDepth }
func (v *Viewport) SetMinDepth(value float32) { v.minDepth = value }
func (v Viewport) MaxDepth() float32          { return v.maxDepth }
func (v *Viewport) SetMaxDepth(value float32) { v.maxDepth = value }
func (v Viewport) Bounds() framework.Rectangle {
	return framework.NewRectangle(v.x, v.y, v.width, v.height)
}
func (v *Viewport) SetBounds(value framework.Rectangle) {
	v.x, v.y, v.width, v.height = value.X, value.Y, value.Width, value.Height
}
func (v Viewport) AspectRatio() float32 {
	if v.height == 0 {
		return 0
	}
	return float32(v.width) / float32(v.height)
}
func (v Viewport) TitleSafeArea() framework.Rectangle { return v.Bounds() }
func (v Viewport) ToString() string {
	return fmt.Sprintf("{X:%d Y:%d Width:%d Height:%d MinDepth:%g MaxDepth:%g}", v.x, v.y, v.width, v.height, v.minDepth, v.maxDepth)
}

func (v Viewport) Project(source framework.Vector3, projection, view, world framework.Matrix) framework.Vector3 {
	matrix := framework.MatrixMultiplyByMatrixAndMatrix(world, view)
	matrix = framework.MatrixMultiplyByMatrixAndMatrix(matrix, projection)
	result := framework.Vector3TransformByVector3AndMatrix(source, matrix)
	w := source.X*matrix.M14 + source.Y*matrix.M24 + source.Z*matrix.M34 + matrix.M44
	if !withinViewportEpsilon(w, 1) {
		result = framework.Vector3DivideByVector3AndSingle(result, w)
	}
	result.X = (result.X+1)*0.5*float32(v.width) + float32(v.x)
	result.Y = (-result.Y+1)*0.5*float32(v.height) + float32(v.y)
	result.Z = result.Z*(v.maxDepth-v.minDepth) + v.minDepth
	return result
}

func (v Viewport) Unproject(source framework.Vector3, projection, view, world framework.Matrix) framework.Vector3 {
	matrix := framework.MatrixMultiplyByMatrixAndMatrix(world, view)
	matrix = framework.MatrixMultiplyByMatrixAndMatrix(matrix, projection)
	matrix = framework.MatrixInvertByMatrix(matrix)
	source.X = (source.X-float32(v.x))/float32(v.width)*2 - 1
	source.Y = -((source.Y-float32(v.y))/float32(v.height)*2 - 1)
	source.Z = (source.Z - v.minDepth) / (v.maxDepth - v.minDepth)
	result := framework.Vector3TransformByVector3AndMatrix(source, matrix)
	w := source.X*matrix.M14 + source.Y*matrix.M24 + source.Z*matrix.M34 + matrix.M44
	if !withinViewportEpsilon(w, 1) {
		result = framework.Vector3DivideByVector3AndSingle(result, w)
	}
	return result
}

func withinViewportEpsilon(value1, value2 float32) bool {
	difference := value1 - value2
	const epsilon = 1.40129846e-45
	return -epsilon <= difference && difference <= epsilon
}

// GraphicsDevice is a borrowed facade over the Game-owned device. It never
// retains CNA's callback-scoped device handle.
type GraphicsDevice struct {
	device *interop.Device
}

// GraphicsDeviceManagerGraphicsDevice is the documented cross-package cycle
// cut for GraphicsDeviceManager.GraphicsDevice.
//
// # It returns ONE object per manager, per generation
//
// GraphicsDeviceManager::get_GraphicsDevice is a single `ldfld` over the
// `device` field, so in the reference every read of the property returns the
// same object until ChangeDevice replaces it. Foundation 49 made that true
// here too: a fresh facade per call passed every test written with one local
// variable, and failed the moment two callers compared what they got -- which
// is exactly what a consumer does when Game.GraphicsDevice and a
// DrawableGameComponent's device are supposed to be the same device.
//
// The facade is cached on the manager through internal/servicebridge, because
// the object is a Graphics-package type and the field belongs to a
// framework-package one. The native GENERATION is cached with it: each Run
// gets a new one, and the reference replaces its field at the equivalent
// boundary rather than handing out an object bound to a dead device.
func GraphicsDeviceManagerGraphicsDevice(manager *framework.GraphicsDeviceManager) (*GraphicsDevice, error) {
	runtime, resource, ok := interop.BindingForOwner(manager)
	if !ok || resource == nil || runtime == nil {
		return nil, errors.New("GraphicsDeviceManager is not bound to an active Game")
	}
	generation := runtime.Generation()
	if cached, cachedGeneration, present := servicebridge.ReadManagerDeviceFacade(manager); present && cachedGeneration == generation {
		if facade, typed := cached.(*GraphicsDevice); typed {
			return facade, nil
		}
	}
	device, err := interop.DeviceForManager(resource)
	if err != nil {
		return nil, err
	}
	facade := &GraphicsDevice{device: device}
	servicebridge.WriteManagerDeviceFacade(manager, facade, generation)
	return facade, nil
}

func (d *GraphicsDevice) Viewport() (Viewport, error) {
	if d == nil || d.device == nil {
		return Viewport{}, errors.New("GraphicsDevice is nil")
	}
	value, err := d.device.Viewport()
	if err != nil {
		return Viewport{}, err
	}
	return Viewport{x: value.X, y: value.Y, width: value.Width, height: value.Height, minDepth: value.MinDepth, maxDepth: value.MaxDepth}, nil
}

func (d *GraphicsDevice) ClearByColor(color framework.Color) error {
	if d == nil || d.device == nil {
		return errors.New("GraphicsDevice is nil")
	}
	return d.device.Clear(
		float32(color.R())/255,
		float32(color.G())/255,
		float32(color.B())/255,
		float32(color.A())/255,
	)
}

// Texture2D is an OWNED, generation-checked native texture.
type Texture2D struct {
	resource *interop.Resource
	info     interop.TextureInfo
}

func Texture2DFromStreamByGraphicsDeviceAndStream(device *GraphicsDevice, stream io.Reader) (*Texture2D, error) {
	if device == nil || device.device == nil {
		return nil, errors.New("GraphicsDevice is nil")
	}
	if stream == nil {
		return nil, errors.New("texture stream is nil")
	}
	data, err := io.ReadAll(stream)
	if err != nil {
		return nil, err
	}
	resource, info, err := device.device.CreateTextureFromEncoded(data)
	if err != nil {
		return nil, err
	}
	return &Texture2D{resource: resource, info: info}, nil
}

// The two Texture2D constructors. Both are thirty bytes of IL over one private
// CreateTexture:
//
//	.ctor(GraphicsDevice, int width, int height)
//	  CreateTexture(device, width, height, mipMap: false, usage: 0, pool: 1,
//	                format: SurfaceFormat.Color)   // ldc.i4.0 for both
//
//	.ctor(GraphicsDevice, int width, int height, bool mipMap, SurfaceFormat format)
//	  CreateTexture(device, width, height, mipMap, usage: 0, pool: 1, format)
//
// so the three-argument overload is the five-argument one with `false` and
// `SurfaceFormat.Color`, and those two defaults are read off the IL rather than
// chosen. `usage` and `pool` are D3D concepts the reference passes as constants
// and CNA has no parameter for.
//
// Both wrap the call in a `.try/fault` that calls `GraphicsResource::Dispose(true)`
// if it throws, which is the CLR's way of not leaking a half-built object. Go
// has no half-built object to leak: a refused creation returns `(nil, err)` and
// the native texture, if one was made at all, is disposed by CreateTexture
// before it returns.
//
// The one guard reproduced here is the reference's own first check:
//
//	if (graphicsDevice == null)
//	    throw new ArgumentNullException("graphicsDevice",
//	                                    FrameworkResources.DeviceCannotBeNullOnResourceCreate);
//
// Everything after it -- a dimension of zero, a format the renderer does not
// have -- is validated by CNA, which refuses with its own result rather than
// Microsoft's sentence. That difference is recorded rather than papered over:
// reproducing those messages would mean reproducing D3D9's format-capability
// tables, and CNA-Go would then be asserting a support decision it did not make.

// NewTexture2DByGraphicsDeviceAndInt32AndInt32 is
// Texture2D::.ctor(GraphicsDevice, Int32, Int32).
func NewTexture2DByGraphicsDeviceAndInt32AndInt32(
	graphicsDevice *GraphicsDevice, width, height int32,
) (*Texture2D, error) {
	return NewTexture2DByGraphicsDeviceAndInt32AndInt32AndBooleanAndSurfaceFormat(
		graphicsDevice, width, height, false, SurfaceFormatColor)
}

// NewTexture2DByGraphicsDeviceAndInt32AndInt32AndBooleanAndSurfaceFormat is
// Texture2D::.ctor(GraphicsDevice, Int32, Int32, Boolean, SurfaceFormat).
func NewTexture2DByGraphicsDeviceAndInt32AndInt32AndBooleanAndSurfaceFormat(
	graphicsDevice *GraphicsDevice, width, height int32, mipMap bool, format SurfaceFormat,
) (*Texture2D, error) {
	if graphicsDevice == nil || graphicsDevice.device == nil {
		return nil, fmt.Errorf("%w: graphicsDevice: %s",
			errGraphicsResourceArgumentNull, deviceCannotBeNullOnResourceCreate)
	}
	if width < 0 || height < 0 {
		// CNA takes both as uint32, so a negative would arrive as an enormous
		// positive. The reference refuses these too, and this refusal exists so
		// the conversion cannot silently invent a dimension.
		return nil, fmt.Errorf("%w: a texture dimension is negative: %dx%d",
			errGraphicsResourceArgument, width, height)
	}
	resource, info, err := graphicsDevice.device.CreateTexture(
		uint32(width), uint32(height), mipMap, uint32(format))
	if err != nil {
		return nil, err
	}
	return &Texture2D{resource: resource, info: info}, nil
}

func (t *Texture2D) Width() (int32, error) {
	if t == nil || t.resource == nil {
		return 0, interop.ErrDisposed
	}
	return int32(t.info.Width), nil
}

func (t *Texture2D) Height() (int32, error) {
	if t == nil || t.resource == nil {
		return 0, interop.ErrDisposed
	}
	return int32(t.info.Height), nil
}

// Bounds is Texture2D::get_Bounds:
//
//	get_Bounds()
//	  ldarg.0; call Texture2D::get_Width
//	  ldarg.0; call Texture2D::get_Height
//	  newobj Rectangle::.ctor(0, 0, width, height)
//
// A fresh rectangle at the origin, every call. It is fallible for the reason
// Width and Height are: both getters read a disposed-checked native texture,
// and the reference's own getters read a field on a GraphicsResource whose
// disposal it checks.
func (t *Texture2D) Bounds() (framework.Rectangle, error) {
	width, err := t.Width()
	if err != nil {
		return framework.Rectangle{}, err
	}
	height, err := t.Height()
	if err != nil {
		return framework.Rectangle{}, err
	}
	return framework.NewRectangle(0, 0, width, height), nil
}

func (t *Texture2D) Dispose(disposing bool) error {
	_ = disposing
	if t == nil || t.resource == nil {
		return nil
	}
	return t.resource.Dispose()
}

// SpriteBatch is an OWNED, generation-checked native SpriteBatch.
type SpriteBatch struct {
	resource *interop.Resource
	// inBeginEndPair is the reference's own field. It is tracked managed-side
	// because the reference's two guards report Microsoft's messages, and CNA
	// -- which refuses the same states with CNA_RESULT_INVALID_STATE -- reports
	// its own. See sprite_batch_draw.go.
	inBeginEndPair bool
}

func NewSpriteBatch(graphicsDevice *GraphicsDevice) (*SpriteBatch, error) {
	if graphicsDevice == nil || graphicsDevice.device == nil {
		return nil, errors.New("GraphicsDevice is nil")
	}
	resource, err := graphicsDevice.device.CreateSpriteBatch()
	if err != nil {
		return nil, err
	}
	return &SpriteBatch{resource: resource}, nil
}

// BeginByNone is SpriteBatch::Begin(), which forwards to the seven-argument
// Begin with (SpriteSortMode.Deferred, null, null, null, null, null,
// Matrix.Identity). That overload's FIRST statement is the guard:
//
//	if (this.inBeginEndPair)
//	    throw new InvalidOperationException(FrameworkResources.EndMustBeCalledBeforeBegin);
//
// The flag is raised only after the state is stored, so a refused Begin leaves
// the pair exactly as it was. CNA-Go raises it only after CNA accepts, for the
// same reason.
func (b *SpriteBatch) BeginByNone() error {
	if b == nil {
		return interop.ErrDisposed
	}
	// The pair guard runs before the disposal check, for the reason it runs
	// before everything else in the reference: it is the FIRST statement of
	// the seven-argument Begin, ahead of any state the object holds.
	if b.inBeginEndPair {
		return fmt.Errorf("%w: %s", errSpriteInvalidOperation, endMustBeCalledBeforeBegin)
	}
	if b.resource == nil {
		return interop.ErrDisposed
	}
	if err := b.resource.BeginSpriteBatch(); err != nil {
		return err
	}
	b.inBeginEndPair = true
	return nil
}

// DrawByTexture2DAndVector2AndNullableOfRectangleAndColorAndSingleAndVector2AndSingleAndSpriteEffectsAndSingle
// is the UNIFORM-scale overload: the reference stores its one `scale` float
// into BOTH the Vector4's Z and its W, which is what makes it the same route as
// its per-axis sibling and a different member.
//
// Foundation 50 routed it through spriteDraw with the other six. Until then it
// reported ErrDisposed for a nil texture and did not check the begin/end pair
// at all, so it reported neither of the reference's two messages.
func (b *SpriteBatch) DrawByTexture2DAndVector2AndNullableOfRectangleAndColorAndSingleAndVector2AndSingleAndSpriteEffectsAndSingle(
	texture *Texture2D,
	position framework.Vector2,
	sourceRectangle *framework.Rectangle,
	color framework.Color,
	rotation float32,
	origin framework.Vector2,
	scale float32,
	effects SpriteEffects,
	layerDepth float32,
) error {
	return b.spriteDraw(texture, position.X, position.Y, scale, scale, true,
		sourceRectangle, color, rotation, origin, effects, layerDepth)
}

// End is SpriteBatch::End, whose first statement is the mirror guard:
//
//	if (!this.inBeginEndPair)
//	    throw new InvalidOperationException(FrameworkResources.BeginMustBeCalledBeforeEnd);
//
// and whose LAST relevant statement clears the flag -- after the flush, so a
// refused End leaves the pair open, which is what lets a consumer retry.
func (b *SpriteBatch) End() error {
	if b == nil {
		return interop.ErrDisposed
	}
	if !b.inBeginEndPair {
		return fmt.Errorf("%w: %s", errSpriteInvalidOperation, beginMustBeCalledBeforeEnd)
	}
	if b.resource == nil {
		return interop.ErrDisposed
	}
	if err := b.resource.EndSpriteBatch(); err != nil {
		return err
	}
	b.inBeginEndPair = false
	return nil
}

func (b *SpriteBatch) Dispose(disposing bool) error {
	_ = disposing
	if b == nil || b.resource == nil {
		return nil
	}
	return b.resource.Dispose()
}
