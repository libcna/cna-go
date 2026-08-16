package framework

// Color stores non-premultiplied red, green, blue, and alpha channels.
type Color struct {
	R uint8
	G uint8
	B uint8
	A uint8
}

var (
	// CornflowerBlue is the traditional XNA clear color.
	CornflowerBlue = Color{R: 100, G: 149, B: 237, A: 255}
	// White is opaque white.
	White = Color{R: 255, G: 255, B: 255, A: 255}
)

// NewColor constructs an RGBA color.
func NewColor(red, green, blue, alpha uint8) Color {
	return Color{R: red, G: green, B: blue, A: alpha}
}
