package framework

import (
	"math"
	"testing"
)

func TestColorFloatPackingAndOperationsMatchXNA(t *testing.T) {
	packed := NewColorBySingleAndSingleAndSingleAndSingle(0.5, float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1)))
	if got := packed.PackedValue(); got != 0x00FF0080 {
		t.Fatalf("float packing = %08X", got)
	}
	if got := ColorLerp(NewColorByInt32AndInt32AndInt32AndInt32(0, 0, 0, 0), NewColorByInt32AndInt32AndInt32AndInt32(255, 255, 255, 255), 0.5).PackedValue(); got != 0x7F7F7F7F {
		t.Fatalf("Lerp midpoint = %08X", got)
	}
	if got := ColorMultiply(ColorWhite(), 0.5).PackedValue(); got != 0x7F7F7F7F {
		t.Fatalf("Multiply midpoint = %08X", got)
	}
	const maxInt32 = int32(1<<31 - 1)
	if got := ColorFromNonPremultipliedByInt32AndInt32AndInt32AndInt32(maxInt32, maxInt32, maxInt32, maxInt32).PackedValue(); got != 0xFFFFFFFF {
		t.Fatalf("non-premultiplied extreme = %08X", got)
	}
	if got := ColorTransparent().PackedValue(); got != 0x00FFFFFF {
		t.Fatalf("Transparent = %08X", got)
	}
}

func TestColorVectorRoundTrip(t *testing.T) {
	c := NewColorByInt32AndInt32AndInt32AndInt32(1, 2, 3, 4)
	v := c.ToVector4()
	if NewColorByVector4(v) != c {
		t.Fatalf("round trip = %v", NewColorByVector4(v))
	}
}

func TestEveryPredefinedColorMatchesXnaGoldenTable(t *testing.T) {
	tests := []struct {
		name string
		get  func() Color
		want uint32
	}{
		{"Transparent", ColorTransparent, 0x00FFFFFF},
		{"AliceBlue", ColorAliceBlue, 0xFFFFF8F0},
		{"AntiqueWhite", ColorAntiqueWhite, 0xFFD7EBFA},
		{"Aqua", ColorAqua, 0xFFFFFF00},
		{"Aquamarine", ColorAquamarine, 0xFFD4FF7F},
		{"Azure", ColorAzure, 0xFFFFFFF0},
		{"Beige", ColorBeige, 0xFFDCF5F5},
		{"Bisque", ColorBisque, 0xFFC4E4FF},
		{"Black", ColorBlack, 0xFF000000},
		{"BlanchedAlmond", ColorBlanchedAlmond, 0xFFCDEBFF},
		{"Blue", ColorBlue, 0xFFFF0000},
		{"BlueViolet", ColorBlueViolet, 0xFFE22B8A},
		{"Brown", ColorBrown, 0xFF2A2AA5},
		{"BurlyWood", ColorBurlyWood, 0xFF87B8DE},
		{"CadetBlue", ColorCadetBlue, 0xFFA09E5F},
		{"Chartreuse", ColorChartreuse, 0xFF00FF7F},
		{"Chocolate", ColorChocolate, 0xFF1E69D2},
		{"Coral", ColorCoral, 0xFF507FFF},
		{"CornflowerBlue", ColorCornflowerBlue, 0xFFED9564},
		{"Cornsilk", ColorCornsilk, 0xFFDCF8FF},
		{"Crimson", ColorCrimson, 0xFF3C14DC},
		{"Cyan", ColorCyan, 0xFFFFFF00},
		{"DarkBlue", ColorDarkBlue, 0xFF8B0000},
		{"DarkCyan", ColorDarkCyan, 0xFF8B8B00},
		{"DarkGoldenrod", ColorDarkGoldenrod, 0xFF0B86B8},
		{"DarkGray", ColorDarkGray, 0xFFA9A9A9},
		{"DarkGreen", ColorDarkGreen, 0xFF006400},
		{"DarkKhaki", ColorDarkKhaki, 0xFF6BB7BD},
		{"DarkMagenta", ColorDarkMagenta, 0xFF8B008B},
		{"DarkOliveGreen", ColorDarkOliveGreen, 0xFF2F6B55},
		{"DarkOrange", ColorDarkOrange, 0xFF008CFF},
		{"DarkOrchid", ColorDarkOrchid, 0xFFCC3299},
		{"DarkRed", ColorDarkRed, 0xFF00008B},
		{"DarkSalmon", ColorDarkSalmon, 0xFF7A96E9},
		{"DarkSeaGreen", ColorDarkSeaGreen, 0xFF8FBC8F},
		{"DarkSlateBlue", ColorDarkSlateBlue, 0xFF8B3D48},
		{"DarkSlateGray", ColorDarkSlateGray, 0xFF4F4F2F},
		{"DarkTurquoise", ColorDarkTurquoise, 0xFFD1CE00},
		{"DarkViolet", ColorDarkViolet, 0xFFD30094},
		{"DeepPink", ColorDeepPink, 0xFF9314FF},
		{"DeepSkyBlue", ColorDeepSkyBlue, 0xFFFFBF00},
		{"DimGray", ColorDimGray, 0xFF696969},
		{"DodgerBlue", ColorDodgerBlue, 0xFFFF901E},
		{"Firebrick", ColorFirebrick, 0xFF2222B2},
		{"FloralWhite", ColorFloralWhite, 0xFFF0FAFF},
		{"ForestGreen", ColorForestGreen, 0xFF228B22},
		{"Fuchsia", ColorFuchsia, 0xFFFF00FF},
		{"Gainsboro", ColorGainsboro, 0xFFDCDCDC},
		{"GhostWhite", ColorGhostWhite, 0xFFFFF8F8},
		{"Gold", ColorGold, 0xFF00D7FF},
		{"Goldenrod", ColorGoldenrod, 0xFF20A5DA},
		{"Gray", ColorGray, 0xFF808080},
		{"Green", ColorGreen, 0xFF008000},
		{"GreenYellow", ColorGreenYellow, 0xFF2FFFAD},
		{"Honeydew", ColorHoneydew, 0xFFF0FFF0},
		{"HotPink", ColorHotPink, 0xFFB469FF},
		{"IndianRed", ColorIndianRed, 0xFF5C5CCD},
		{"Indigo", ColorIndigo, 0xFF82004B},
		{"Ivory", ColorIvory, 0xFFF0FFFF},
		{"Khaki", ColorKhaki, 0xFF8CE6F0},
		{"Lavender", ColorLavender, 0xFFFAE6E6},
		{"LavenderBlush", ColorLavenderBlush, 0xFFF5F0FF},
		{"LawnGreen", ColorLawnGreen, 0xFF00FC7C},
		{"LemonChiffon", ColorLemonChiffon, 0xFFCDFAFF},
		{"LightBlue", ColorLightBlue, 0xFFE6D8AD},
		{"LightCoral", ColorLightCoral, 0xFF8080F0},
		{"LightCyan", ColorLightCyan, 0xFFFFFFE0},
		{"LightGoldenrodYellow", ColorLightGoldenrodYellow, 0xFFD2FAFA},
		{"LightGray", ColorLightGray, 0xFFD3D3D3},
		{"LightGreen", ColorLightGreen, 0xFF90EE90},
		{"LightPink", ColorLightPink, 0xFFC1B6FF},
		{"LightSalmon", ColorLightSalmon, 0xFF7AA0FF},
		{"LightSeaGreen", ColorLightSeaGreen, 0xFFAAB220},
		{"LightSkyBlue", ColorLightSkyBlue, 0xFFFACE87},
		{"LightSlateGray", ColorLightSlateGray, 0xFF998877},
		{"LightSteelBlue", ColorLightSteelBlue, 0xFFDEC4B0},
		{"LightYellow", ColorLightYellow, 0xFFE0FFFF},
		{"Lime", ColorLime, 0xFF00FF00},
		{"LimeGreen", ColorLimeGreen, 0xFF32CD32},
		{"Linen", ColorLinen, 0xFFE6F0FA},
		{"Magenta", ColorMagenta, 0xFFFF00FF},
		{"Maroon", ColorMaroon, 0xFF000080},
		{"MediumAquamarine", ColorMediumAquamarine, 0xFFAACD66},
		{"MediumBlue", ColorMediumBlue, 0xFFCD0000},
		{"MediumOrchid", ColorMediumOrchid, 0xFFD355BA},
		{"MediumPurple", ColorMediumPurple, 0xFFDB7093},
		{"MediumSeaGreen", ColorMediumSeaGreen, 0xFF71B33C},
		{"MediumSlateBlue", ColorMediumSlateBlue, 0xFFEE687B},
		{"MediumSpringGreen", ColorMediumSpringGreen, 0xFF9AFA00},
		{"MediumTurquoise", ColorMediumTurquoise, 0xFFCCD148},
		{"MediumVioletRed", ColorMediumVioletRed, 0xFF8515C7},
		{"MidnightBlue", ColorMidnightBlue, 0xFF701919},
		{"MintCream", ColorMintCream, 0xFFFAFFF5},
		{"MistyRose", ColorMistyRose, 0xFFE1E4FF},
		{"Moccasin", ColorMoccasin, 0xFFB5E4FF},
		{"NavajoWhite", ColorNavajoWhite, 0xFFADDEFF},
		{"Navy", ColorNavy, 0xFF800000},
		{"OldLace", ColorOldLace, 0xFFE6F5FD},
		{"Olive", ColorOlive, 0xFF008080},
		{"OliveDrab", ColorOliveDrab, 0xFF238E6B},
		{"Orange", ColorOrange, 0xFF00A5FF},
		{"OrangeRed", ColorOrangeRed, 0xFF0045FF},
		{"Orchid", ColorOrchid, 0xFFD670DA},
		{"PaleGoldenrod", ColorPaleGoldenrod, 0xFFAAE8EE},
		{"PaleGreen", ColorPaleGreen, 0xFF98FB98},
		{"PaleTurquoise", ColorPaleTurquoise, 0xFFEEEEAF},
		{"PaleVioletRed", ColorPaleVioletRed, 0xFF9370DB},
		{"PapayaWhip", ColorPapayaWhip, 0xFFD5EFFF},
		{"PeachPuff", ColorPeachPuff, 0xFFB9DAFF},
		{"Peru", ColorPeru, 0xFF3F85CD},
		{"Pink", ColorPink, 0xFFCBC0FF},
		{"Plum", ColorPlum, 0xFFDDA0DD},
		{"PowderBlue", ColorPowderBlue, 0xFFE6E0B0},
		{"Purple", ColorPurple, 0xFF800080},
		{"Red", ColorRed, 0xFF0000FF},
		{"RosyBrown", ColorRosyBrown, 0xFF8F8FBC},
		{"RoyalBlue", ColorRoyalBlue, 0xFFE16941},
		{"SaddleBrown", ColorSaddleBrown, 0xFF13458B},
		{"Salmon", ColorSalmon, 0xFF7280FA},
		{"SandyBrown", ColorSandyBrown, 0xFF60A4F4},
		{"SeaGreen", ColorSeaGreen, 0xFF578B2E},
		{"SeaShell", ColorSeaShell, 0xFFEEF5FF},
		{"Sienna", ColorSienna, 0xFF2D52A0},
		{"Silver", ColorSilver, 0xFFC0C0C0},
		{"SkyBlue", ColorSkyBlue, 0xFFEBCE87},
		{"SlateBlue", ColorSlateBlue, 0xFFCD5A6A},
		{"SlateGray", ColorSlateGray, 0xFF908070},
		{"Snow", ColorSnow, 0xFFFAFAFF},
		{"SpringGreen", ColorSpringGreen, 0xFF7FFF00},
		{"SteelBlue", ColorSteelBlue, 0xFFB48246},
		{"Tan", ColorTan, 0xFF8CB4D2},
		{"Teal", ColorTeal, 0xFF808000},
		{"Thistle", ColorThistle, 0xFFD8BFD8},
		{"Tomato", ColorTomato, 0xFF4763FF},
		{"Turquoise", ColorTurquoise, 0xFFD0E040},
		{"Violet", ColorViolet, 0xFFEE82EE},
		{"Wheat", ColorWheat, 0xFFB3DEF5},
		{"White", ColorWhite, 0xFFFFFFFF},
		{"WhiteSmoke", ColorWhiteSmoke, 0xFFF5F5F5},
		{"Yellow", ColorYellow, 0xFF00FFFF},
		{"YellowGreen", ColorYellowGreen, 0xFF32CD9A},
	}
	if len(tests) != 141 {
		t.Fatalf("palette has %d entries, want 141", len(tests))
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.get().PackedValue(); got != test.want {
				t.Fatalf("PackedValue = %08X, want %08X", got, test.want)
			}
		})
	}
}
