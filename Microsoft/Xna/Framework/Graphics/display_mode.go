package graphics

import (
	"fmt"
	"strconv"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// DisplayMode is Microsoft.Xna.Framework.Graphics.DisplayMode.
//
// # It is pure managed, and it has no public constructor
//
// The reference's whole type is three private fields and six members over them.
// Two of the six are not field reads:
//
//	get_AspectRatio
//	  if (_height == 0 || _width == 0) return 0f;
//	  return (float)_width / (float)_height;
//
//	get_TitleSafeArea
//	  return Viewport.GetTitleSafeArea(0, 0, _width, _height);
//
// and `Viewport::GetTitleSafeArea(x, y, w, h)` is ten bytes -- `new Rectangle(x,
// y, w, h)` -- so on this profile the title-safe area is the whole area. That is
// measured rather than assumed: the same static is what Viewport.TitleSafeArea
// has returned since Foundation 1.
//
// The contract declares SIX members and no constructor, so a consumer cannot
// build one. Every DisplayMode comes from a member that reports one, which today
// is GraphicsDevice.DisplayMode and tomorrow is GraphicsAdapter's two.
//
// # CNA reports an aspect ratio and this type does not use it
//
// CNA_DisplayMode carries `aspect_ratio` next to the two dimensions. The
// projection computes the reference's arithmetic instead, because the reference
// has a defined answer for `width == 0` -- zero, not a division -- and a second
// computed value could disagree with it. The native field is still carried
// across the boundary, so its layout stays measured.
type DisplayMode struct {
	width  int32
	height int32
	format SurfaceFormat
}

// Width is DisplayMode::get_Width, a field read.
func (m *DisplayMode) Width() int32 { return m.width }

// Height is DisplayMode::get_Height, a field read.
func (m *DisplayMode) Height() int32 { return m.height }

// Format is DisplayMode::get_Format, a field read.
func (m *DisplayMode) Format() SurfaceFormat { return m.format }

// AspectRatio is DisplayMode::get_AspectRatio.
//
// The guard is on BOTH dimensions and it is the reference's own: a zero height
// would divide by zero, and a zero width is answered with zero rather than with
// the 0.0 the division would produce anyway. Reproducing the second branch
// matters because it is the difference between a defined answer and a floating
// point one.
func (m *DisplayMode) AspectRatio() float32 {
	if m.height == 0 || m.width == 0 {
		return 0
	}
	return float32(m.width) / float32(m.height)
}

// TitleSafeArea is DisplayMode::get_TitleSafeArea.
func (m *DisplayMode) TitleSafeArea() framework.Rectangle {
	return framework.NewRectangle(0, 0, m.width, m.height)
}

// ToString is DisplayMode::ToString, whose format string is
//
//	"{{Width:{0} Height:{1} Format:{2} AspectRatio:{3}}}"
//
// formatted with CurrentCulture. The doubled braces are the CLR's escape, so the
// rendered text opens and closes with a single brace, and the four arguments are
// the two fields plus the two computed members -- Format and AspectRatio are
// read through their PROPERTIES in the IL, not through the fields.
func (m *DisplayMode) ToString() string {
	return fmt.Sprintf("{Width:%d Height:%d Format:%s AspectRatio:%v}",
		m.width, m.height, surfaceFormatString(m.Format()), m.AspectRatio())
}

// surfaceFormatString renders a SurfaceFormat the way the CLR renders a boxed
// enum inside String.Format: as its declared NAME, or as its decimal value when
// no member declares it.
//
// It is unexported, following the vertexElementFormatString convention, because
// the profile's SurfaceFormat declares no ToString of its own and an exported
// one would be public API the contract does not have.
func surfaceFormatString(value SurfaceFormat) string {
	switch value {
	case SurfaceFormatColor:
		return "Color"
	case SurfaceFormatBgr565:
		return "Bgr565"
	case SurfaceFormatBgra5551:
		return "Bgra5551"
	case SurfaceFormatBgra4444:
		return "Bgra4444"
	case SurfaceFormatDxt1:
		return "Dxt1"
	case SurfaceFormatDxt3:
		return "Dxt3"
	case SurfaceFormatDxt5:
		return "Dxt5"
	case SurfaceFormatNormalizedByte2:
		return "NormalizedByte2"
	case SurfaceFormatNormalizedByte4:
		return "NormalizedByte4"
	case SurfaceFormatRgba1010102:
		return "Rgba1010102"
	case SurfaceFormatRg32:
		return "Rg32"
	case SurfaceFormatRgba64:
		return "Rgba64"
	case SurfaceFormatAlpha8:
		return "Alpha8"
	case SurfaceFormatSingle:
		return "Single"
	case SurfaceFormatVector2:
		return "Vector2"
	case SurfaceFormatVector4:
		return "Vector4"
	case SurfaceFormatHalfSingle:
		return "HalfSingle"
	case SurfaceFormatHalfVector2:
		return "HalfVector2"
	case SurfaceFormatHalfVector4:
		return "HalfVector4"
	case SurfaceFormatHdrBlendable:
		return "HdrBlendable"
	default:
		return strconv.Itoa(int(value))
	}
}
