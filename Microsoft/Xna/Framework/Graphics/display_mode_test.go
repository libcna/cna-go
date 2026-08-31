package graphics

import (
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// DisplayMode is pure managed, so every one of its six members is measurable
// here without a device. What is NOT measurable here is where the three field
// values came from, which is the device-state stress scenario's job.

// TestAspectRatioGuardsBothDimensions pins the reference's own guard. It is
// `_height == 0 || _width == 0`, and the second half is the one a natural
// rewrite drops: a zero width divides cleanly to 0.0 anyway, so the branch
// looks redundant until you notice it is what makes the answer DEFINED rather
// than a floating point result.
func TestAspectRatioGuardsBothDimensions(t *testing.T) {
	for name, mode := range map[string]DisplayMode{
		"zero height": {width: 800, height: 0},
		"zero width":  {width: 0, height: 480},
		"both zero":   {width: 0, height: 0},
	} {
		if got := mode.AspectRatio(); got != 0 {
			t.Errorf("%s: AspectRatio = %v, want 0", name, got)
		}
	}
	mode := DisplayMode{width: 800, height: 480}
	if got, want := mode.AspectRatio(), float32(800)/float32(480); got != want {
		t.Fatalf("AspectRatio = %v, want %v", got, want)
	}
}

// TestTitleSafeAreaIsTheWholeArea measures what Viewport::GetTitleSafeArea
// actually does on this profile: it is ten bytes of IL building a Rectangle
// from its four arguments, so there is no inset. A projection that applied the
// Xbox 80% inset would be reproducing a different platform.
func TestTitleSafeAreaIsTheWholeArea(t *testing.T) {
	mode := DisplayMode{width: 1280, height: 720}
	want := framework.NewRectangle(0, 0, 1280, 720)
	if got := mode.TitleSafeArea(); got != want {
		t.Fatalf("TitleSafeArea = %+v, want %+v", got, want)
	}
	// It starts at the ORIGIN, not at the mode's own position: the reference
	// passes literal zeros for x and y.
	if got := mode.TitleSafeArea(); got.X != 0 || got.Y != 0 {
		t.Fatalf("TitleSafeArea does not start at the origin: %+v", got)
	}
}

// TestToStringRendersTheEnumByName pins the one part of ToString that a
// straightforward Go rewrite gets wrong. String.Format boxes the SurfaceFormat,
// so the CLR renders its NAME; formatting the Go value with %d or %v would
// render a number.
func TestToStringRendersTheEnumByName(t *testing.T) {
	mode := DisplayMode{width: 800, height: 480, format: SurfaceFormatColor}
	got := mode.ToString()
	want := "{Width:800 Height:480 Format:Color AspectRatio:1.6666666}"
	if got != want {
		t.Fatalf("ToString = %q, want %q", got, want)
	}
	// A value no member declares renders as its number, which is what the CLR
	// does for an undefined enum value.
	unknown := DisplayMode{width: 1, height: 1, format: SurfaceFormat(9999)}
	if got := surfaceFormatString(unknown.Format()); got != "9999" {
		t.Fatalf("an undeclared SurfaceFormat rendered as %q", got)
	}
}

// TestEveryDeclaredSurfaceFormatHasAName is the coverage proof for the name
// table: XNA declares twenty formats and a gap would render one of them as a
// number inside an otherwise correct sentence.
func TestEveryDeclaredSurfaceFormatHasAName(t *testing.T) {
	for value := SurfaceFormatColor; value <= SurfaceFormatHdrBlendable; value++ {
		if name := surfaceFormatString(value); name == "" || name[0] >= '0' && name[0] <= '9' {
			t.Errorf("SurfaceFormat(%d) rendered as %q", value, name)
		}
	}
}
