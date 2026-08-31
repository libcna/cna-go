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

func (b *SpriteBatch) BeginByNone() error {
	if b == nil || b.resource == nil {
		return interop.ErrDisposed
	}
	return b.resource.BeginSpriteBatch()
}

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
	if b == nil || b.resource == nil || texture == nil || texture.resource == nil {
		return interop.ErrDisposed
	}
	command := interop.SpriteCommand{
		PositionX: position.X, PositionY: position.Y,
		Red: color.R(), Green: color.G(), Blue: color.B(), Alpha: color.A(),
		Rotation: rotation, OriginX: origin.X, OriginY: origin.Y,
		ScaleX: scale, ScaleY: scale, Effects: uint32(effects), LayerDepth: layerDepth,
	}
	if sourceRectangle != nil {
		command.SourceX = sourceRectangle.X
		command.SourceY = sourceRectangle.Y
		command.SourceWidth = sourceRectangle.Width
		command.SourceHeight = sourceRectangle.Height
	}
	return b.resource.DrawSprite(texture.resource, command)
}

func (b *SpriteBatch) End() error {
	if b == nil || b.resource == nil {
		return interop.ErrDisposed
	}
	return b.resource.EndSpriteBatch()
}

func (b *SpriteBatch) Dispose(disposing bool) error {
	_ = disposing
	if b == nil || b.resource == nil {
		return nil
	}
	return b.resource.Dispose()
}
