package design

import (
	"reflect"
	"strings"
	"testing"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// TestMathTypeConverterDefaultsStringConvertToTrue is the constructor's one
// instruction before the base call, the same shape ContentSerializerAttribute's
// AllowNull has.
func TestMathTypeConverterDefaultsStringConvertToTrue(t *testing.T) {
	converter := NewMathTypeConverter()
	if !converter.supportStringConvert {
		t.Fatal("supportStringConvert defaulted to false; the constructor stores ldc.i4.1")
	}
	if converter.propertyDescriptions != nil {
		t.Fatal("the base constructor installed property descriptions; the leaves do that")
	}
	// The two `ldc.i4.1` getters.
	if !converter.GetCreateInstanceSupported(nil) || !converter.GetPropertiesSupported(nil) {
		t.Fatal("a GetSupported getter answered false")
	}
}

// TestCanConvertFromConsultsTheFlagAndCanConvertToDoesNot is the asymmetry
// between the two members, which is measured rather than tidied.
//
//	CanConvertFrom: supportStringConvert && string -> true; else == InstanceDescriptor
//	CanConvertTo:   == InstanceDescriptor -> true;          else == string
//
// So clearing the flag stops STRING PARSING and leaves string FORMATTING
// alone -- a converter that answered false to both would be wrong.
func TestCanConvertFromConsultsTheFlagAndCanConvertToDoesNot(t *testing.T) {
	stringType := reflect.TypeOf("")
	descriptorType := reflect.TypeOf(&framework.InstanceDescriptor{})
	intType := reflect.TypeOf(int32(0))

	parsing := NewVector2Converter()
	if !parsing.CanConvertFrom(nil, stringType) {
		t.Fatal("a string-converting converter refused a string source")
	}
	if !parsing.CanConvertFrom(nil, descriptorType) {
		t.Fatal("CanConvertFrom refused an InstanceDescriptor source")
	}
	if parsing.CanConvertFrom(nil, intType) {
		t.Fatal("CanConvertFrom accepted an unrelated source type")
	}

	// RectangleConverter clears the flag.
	formatting := NewRectangleConverter()
	if formatting.CanConvertFrom(nil, stringType) {
		t.Fatal("a converter that cleared supportStringConvert still accepted a string source")
	}
	if !formatting.CanConvertFrom(nil, descriptorType) {
		t.Fatal("clearing the flag also stopped the InstanceDescriptor answer, which it must not")
	}
	// ...and still formats TO a string, because CanConvertTo never reads it.
	if !formatting.CanConvertTo(nil, stringType) {
		t.Fatal("CanConvertTo consulted supportStringConvert; only CanConvertFrom does")
	}
	if !formatting.CanConvertTo(nil, descriptorType) {
		t.Fatal("CanConvertTo refused an InstanceDescriptor destination")
	}
	if formatting.CanConvertTo(nil, intType) {
		t.Fatal("CanConvertTo accepted an unrelated destination type")
	}
}

// TestTheSeparatorIsBareOnSplitAndSpacedOnJoin is the detail the two directions
// disagree about. The reference splits on the culture's list separator and
// joins with that separator PLUS ONE SPACE.
func TestTheSeparatorIsBareOnSplitAndSpacedOnJoin(t *testing.T) {
	converter := NewVector2Converter()
	formatted, err := converter.ConvertTo(nil, nil, framework.NewVector2BySingleAndSingle(1, 2), reflect.TypeOf(""))
	if err != nil {
		t.Fatalf("ConvertTo: %v", err)
	}
	text, ok := formatted.(string)
	if !ok {
		t.Fatalf("ConvertTo produced %T, want a string", formatted)
	}
	if text != "1, 2" {
		t.Fatalf("ConvertTo produced %q; the join separator is the list separator plus one space", text)
	}

	// The split takes the BARE separator, so a string with no space parses too.
	for _, spelling := range []string{"1, 2", "1,2", " 1 , 2 "} {
		value, err := converter.ConvertFrom(nil, nil, spelling)
		if err != nil {
			t.Fatalf("ConvertFrom(%q): %v", spelling, err)
		}
		if value != framework.NewVector2BySingleAndSingle(1, 2) {
			t.Fatalf("ConvertFrom(%q) = %v", spelling, value)
		}
	}
}

// TestAWrongComponentCountIsRefusedAndNamesTheParameters pins the message. It
// carries the EXPECTED PARAMETER NAMES, not the string that failed.
func TestAWrongComponentCountIsRefusedAndNamesTheParameters(t *testing.T) {
	converter := NewVector3Converter()
	_, err := converter.ConvertFrom(nil, nil, "1, 2")
	if err == nil {
		t.Fatal("a two-component string parsed as a Vector3")
	}
	message := err.Error()
	for _, name := range []string{"X", "Y", "Z"} {
		if !strings.Contains(message, name) {
			t.Fatalf("the refusal %q does not name the expected parameter %q", message, name)
		}
	}
	if strings.Contains(message, "1, 2") {
		t.Fatalf("the refusal %q names the failing string; the reference names the expected parameters", message)
	}
	// A component that is not a number is refused the same way.
	if _, err = converter.ConvertFrom(nil, nil, "1, 2, three"); err == nil {
		t.Fatal("a non-numeric component parsed")
	}
}

// TestANonStringValueFallsThroughRatherThanRefusingImmediately pins the nil
// that ConvertToValues answers for a non-string: it is what sends the caller to
// the base, and the base is what raises.
func TestANonStringValueFallsThroughRatherThanRefusingImmediately(t *testing.T) {
	converter := NewVector2Converter()
	_, err := converter.ConvertFrom(nil, nil, int32(7))
	if err == nil {
		t.Fatal("a non-string value converted")
	}
	if strings.Contains(err.Error(), "expected format") {
		t.Fatalf("a non-string value raised the PARSE failure %q; it must reach the base instead", err)
	}
}

// TestConvertToRefusesANilDestinationTypeFirst pins the ORDER of ConvertTo's
// branches: the nil check runs before the value is even looked at.
func TestConvertToRefusesANilDestinationTypeFirst(t *testing.T) {
	converter := NewVector2Converter()
	_, err := converter.ConvertTo(nil, nil, framework.NewVector2BySingleAndSingle(1, 2), nil)
	if err == nil {
		t.Fatal("a nil destinationType was accepted")
	}
	if !strings.Contains(err.Error(), "destinationType") {
		t.Fatalf("the refusal was %q; the reference names destinationType", err)
	}
	// And it fires even for a value the converter does not own, which is what
	// makes it FIRST rather than part of the value check.
	if _, err = converter.ConvertTo(nil, nil, "not a vector", nil); err == nil {
		t.Fatal("a nil destinationType was accepted for a foreign value")
	}
}

// TestConvertToBuildsAnInstanceDescriptorCarryingTheComponents pins the branch
// that exists so a designer can rebuild the value.
func TestConvertToBuildsAnInstanceDescriptorCarryingTheComponents(t *testing.T) {
	converter := NewVector2Converter()
	built, err := converter.ConvertTo(nil, nil, framework.NewVector2BySingleAndSingle(3, 4),
		reflect.TypeOf(&framework.InstanceDescriptor{}))
	if err != nil {
		t.Fatalf("ConvertTo: %v", err)
	}
	descriptor, ok := built.(*framework.InstanceDescriptor)
	if !ok {
		t.Fatalf("ConvertTo produced %T, want an *InstanceDescriptor", built)
	}
	if descriptor.MemberType() != reflect.TypeOf(framework.Vector2{}) {
		t.Fatalf("the descriptor names %v, want the Vector2 type", descriptor.MemberType())
	}
	arguments := descriptor.Arguments()
	if len(arguments) != 2 || arguments[0] != float32(3) || arguments[1] != float32(4) {
		t.Fatalf("the descriptor carries %v, want the components in constructor order", arguments)
	}
	// The arguments are a copy, so a caller cannot reach into the descriptor.
	arguments[0] = float32(99)
	if descriptor.Arguments()[0] != float32(3) {
		t.Fatal("mutating the returned arguments reached the descriptor")
	}
}

// TestCreateInstanceReadsByNameInConstructorOrder pins that the dictionary is
// read by NAME, so a caller's key order cannot change the result.
func TestCreateInstanceReadsByNameInConstructorOrder(t *testing.T) {
	converter := NewVector3Converter()
	value, err := converter.CreateInstance(nil, map[string]any{
		"Z": float32(3), "X": float32(1), "Y": float32(2),
	})
	if err != nil {
		t.Fatalf("CreateInstance: %v", err)
	}
	if value != framework.NewVector3BySingleAndSingleAndSingle(1, 2, 3) {
		t.Fatalf("CreateInstance = %v; the names decide the order, not the map", value)
	}
	if _, err = converter.CreateInstance(nil, nil); err == nil {
		t.Fatal("a nil dictionary was accepted")
	} else if !strings.Contains(err.Error(), "propertyValues") {
		t.Fatalf("the refusal was %q; the reference names propertyValues", err)
	}
	// A missing name is refused rather than defaulted to zero.
	if _, err = converter.CreateInstance(nil, map[string]any{"X": float32(1), "Y": float32(2)}); err == nil {
		t.Fatal("a dictionary missing a component produced a value")
	}
	// And so is a component of the wrong type.
	if _, err = converter.CreateInstance(nil, map[string]any{
		"X": float32(1), "Y": float32(2), "Z": int32(3),
	}); err == nil {
		t.Fatal("a component of the wrong type was accepted")
	}
}

// TestTheElementTypesAreTheMeasuredOnes pins that Point parses integers and
// Color parses bytes -- neither is a Single, which reading the value type alone
// would suggest.
func TestTheElementTypesAreTheMeasuredOnes(t *testing.T) {
	point := NewPointConverter()
	value, err := point.ConvertFrom(nil, nil, "3, 4")
	if err != nil {
		t.Fatalf("PointConverter.ConvertFrom: %v", err)
	}
	if value != framework.NewPoint(3, 4) {
		t.Fatalf("PointConverter produced %v", value)
	}
	// A fractional component is not an Int32.
	if _, err = point.ConvertFrom(nil, nil, "3.5, 4"); err == nil {
		t.Fatal("PointConverter accepted a fractional component")
	}

	color := NewColorConverter()
	value, err = color.ConvertFrom(nil, nil, "255, 128, 0, 255")
	if err != nil {
		t.Fatalf("ColorConverter.ConvertFrom: %v", err)
	}
	if value != framework.NewColorByInt32AndInt32AndInt32AndInt32(255, 128, 0, 255) {
		t.Fatalf("ColorConverter produced %v", value)
	}
	// 256 does not fit a Byte, and Byte.Parse refuses rather than truncating.
	if _, err = color.ConvertFrom(nil, nil, "256, 0, 0, 0"); err == nil {
		t.Fatal("ColorConverter accepted a channel above 255")
	}
	if _, err = color.ConvertFrom(nil, nil, "-1, 0, 0, 0"); err == nil {
		t.Fatal("ColorConverter accepted a negative channel")
	}
}

// TestColorDescribesPropertiesAndVectorsDescribeFields is the measured
// difference that decided PropertyDescriptor's shape.
func TestColorDescribesPropertiesAndVectorsDescribeFields(t *testing.T) {
	vectorProperties := NewVector2Converter().GetProperties(nil, nil, nil)
	if len(vectorProperties) != 2 {
		t.Fatalf("Vector2Converter describes %d properties, want 2", len(vectorProperties))
	}
	if vectorProperties[0].Name() != "X" || vectorProperties[1].Name() != "Y" {
		t.Fatalf("Vector2Converter's descriptors are %q, %q; want X then Y in constructor order",
			vectorProperties[0].Name(), vectorProperties[1].Name())
	}
	read, ok := vectorProperties[1].GetValue(framework.NewVector2BySingleAndSingle(5, 6))
	if !ok || read != float32(6) {
		t.Fatalf("the Y descriptor read %v, %v; want 6", read, ok)
	}

	colorProperties := NewColorConverter().GetProperties(nil, nil, nil)
	if len(colorProperties) != 4 {
		t.Fatalf("ColorConverter describes %d properties, want 4", len(colorProperties))
	}
	// Color's channels are PROPERTIES over a packed value, so the descriptor
	// must read through the accessor -- there is no struct field named "R".
	read, ok = colorProperties[0].GetValue(framework.NewColorByInt32AndInt32AndInt32AndInt32(9, 0, 0, 0))
	if !ok || read != uint8(9) {
		t.Fatalf("the R descriptor read %v, %v; want 9", read, ok)
	}
	// A value of the wrong type does not read.
	if _, ok = colorProperties[0].GetValue("not a color"); ok {
		t.Fatal("a descriptor read a value of the wrong type")
	}
}

// TestMatrixConverterDescribesSeventeen pins the one constructor that is not
// like the others: Translation FIRST, then the sixteen cells.
func TestMatrixConverterDescribesSeventeen(t *testing.T) {
	properties := NewMatrixConverter().GetProperties(nil, nil, nil)
	if len(properties) != 17 {
		t.Fatalf("MatrixConverter describes %d properties, want 17", len(properties))
	}
	if properties[0].Name() != "Translation" {
		t.Fatalf("the first descriptor is %q, want Translation", properties[0].Name())
	}
	if properties[1].Name() != "M11" || properties[16].Name() != "M44" {
		t.Fatalf("the cells run %q..%q, want M11..M44", properties[1].Name(), properties[16].Name())
	}
	// Translation is computed from M41, M42 and M43 rather than stored.
	matrix := framework.NewMatrix(
		0, 0, 0, 0,
		0, 0, 0, 0,
		0, 0, 0, 0,
		7, 8, 9, 0)
	read, ok := properties[0].GetValue(matrix)
	if !ok || read != framework.NewVector3BySingleAndSingleAndSingle(7, 8, 9) {
		t.Fatalf("Translation read %v, %v; want the M41..M43 vector", read, ok)
	}
}

// TestTheThreeForwardingConvertFromsRefuse pins that Ray, BoundingBox and
// BoundingSphere declare ConvertFrom and it does nothing but reach the base --
// which refuses.
func TestTheThreeForwardingConvertFromsRefuse(t *testing.T) {
	rayValue, err := NewRayConverter().ConvertFrom(nil, nil, "anything")
	if err == nil {
		t.Fatalf("RayConverter.ConvertFrom parsed %v; its body forwards to the base", rayValue)
	}
	if _, err = NewBoundingBoxConverter().ConvertFrom(nil, nil, "anything"); err == nil {
		t.Fatal("BoundingBoxConverter.ConvertFrom parsed a string")
	}
	if _, err = NewBoundingSphereConverter().ConvertFrom(nil, nil, "anything"); err == nil {
		t.Fatal("BoundingSphereConverter.ConvertFrom parsed a string")
	}
}

// TestTheCompositeConvertersRoundTripThroughCreateInstance pins the four whose
// components are not scalars.
func TestTheCompositeConvertersRoundTripThroughCreateInstance(t *testing.T) {
	normal := framework.NewVector3BySingleAndSingleAndSingle(0, 1, 0)
	plane, err := NewPlaneConverter().CreateInstance(nil, map[string]any{
		"Normal": normal, "D": float32(5),
	})
	if err != nil {
		t.Fatalf("PlaneConverter.CreateInstance: %v", err)
	}
	if plane != framework.NewPlaneByVector3AndSingle(normal, 5) {
		t.Fatalf("PlaneConverter produced %v", plane)
	}

	position := framework.NewVector3BySingleAndSingleAndSingle(1, 2, 3)
	direction := framework.NewVector3BySingleAndSingleAndSingle(0, 0, 1)
	ray, err := NewRayConverter().CreateInstance(nil, map[string]any{
		"Position": position, "Direction": direction,
	})
	if err != nil {
		t.Fatalf("RayConverter.CreateInstance: %v", err)
	}
	if ray != framework.NewRay(position, direction) {
		t.Fatalf("RayConverter produced %v", ray)
	}

	sphere, err := NewBoundingSphereConverter().CreateInstance(nil, map[string]any{
		"Center": position, "Radius": float32(2),
	})
	if err != nil {
		t.Fatalf("BoundingSphereConverter.CreateInstance: %v", err)
	}
	if sphere != framework.NewBoundingSphere(position, 2) {
		t.Fatalf("BoundingSphereConverter produced %v", sphere)
	}

	box, err := NewBoundingBoxConverter().CreateInstance(nil, map[string]any{
		"Min": position, "Max": direction,
	})
	if err != nil {
		t.Fatalf("BoundingBoxConverter.CreateInstance: %v", err)
	}
	if box != framework.NewBoundingBox(position, direction) {
		t.Fatalf("BoundingBoxConverter produced %v", box)
	}
}

// TestACultureSuppliesTheListSeparator pins the one thing a CultureInfo is in
// this profile, and that a nil culture means the current one.
func TestACultureSuppliesTheListSeparator(t *testing.T) {
	if got := framework.CultureInfoCurrentCulture().ListSeparator(); got != "," {
		t.Fatalf("the current culture's list separator is %q, want a comma", got)
	}
	// A nil culture is the current culture, which is what every converter's
	// `if (culture == null)` prologue does.
	converter := NewVector2Converter()
	withNil, err := converter.ConvertTo(nil, nil, framework.NewVector2BySingleAndSingle(1, 2), reflect.TypeOf(""))
	if err != nil {
		t.Fatalf("ConvertTo with a nil culture: %v", err)
	}
	withCurrent, err := converter.ConvertTo(nil, framework.CultureInfoCurrentCulture(),
		framework.NewVector2BySingleAndSingle(1, 2), reflect.TypeOf(""))
	if err != nil {
		t.Fatalf("ConvertTo with the current culture: %v", err)
	}
	if withNil != withCurrent {
		t.Fatalf("a nil culture produced %v and the current culture %v; they must agree", withNil, withCurrent)
	}
}

// TestTooManyComponentsIsRefused is the other side of the count check. A string
// with MORE components than the type takes must fail, which a `<` comparison
// would let through.
func TestTooManyComponentsIsRefused(t *testing.T) {
	converter := NewVector2Converter()
	if _, err := converter.ConvertFrom(nil, nil, "1, 2, 3"); err == nil {
		t.Fatal("a three-component string parsed as a Vector2")
	}
	if _, err := NewVector3Converter().ConvertFrom(nil, nil, "1, 2, 3, 4"); err == nil {
		t.Fatal("a four-component string parsed as a Vector3")
	}
}

// TestTheRefusalJoinsTheParametersWithTheListSeparator pins the message's exact
// shape, not just that the names appear in it.
func TestTheRefusalJoinsTheParametersWithTheListSeparator(t *testing.T) {
	_, err := NewVector3Converter().ConvertFrom(nil, nil, "1, 2")
	if err == nil {
		t.Fatal("a two-component string parsed as a Vector3")
	}
	// The reference joins with the culture's list separator and no space:
	// string.Join(culture.TextInfo.ListSeparator, expectedParams).
	if !strings.Contains(err.Error(), "X,Y,Z") {
		t.Fatalf("the refusal is %q; the expected parameters join with the bare list separator", err)
	}
}

// TestSingleParseRefusesAValueOutsideSingleRange pins that the element parse is
// a SINGLE parse and not a Double one. 1e40 is a valid Double and is not a
// Single.
func TestSingleParseRefusesAValueOutsideSingleRange(t *testing.T) {
	if _, err := NewVector2Converter().ConvertFrom(nil, nil, "1e40, 2"); err == nil {
		t.Fatal("a component outside Single's range parsed")
	}
	// A value inside the range still parses, so the check is about the range
	// and not about exponents.
	value, err := NewVector2Converter().ConvertFrom(nil, nil, "1e20, 2")
	if err != nil {
		t.Fatalf("a component inside Single's range was refused: %v", err)
	}
	if value != framework.NewVector2BySingleAndSingle(1e20, 2) {
		t.Fatalf("ConvertFrom produced %v", value)
	}
}

// TestANilDictionaryIsRefusedBeforeAnyNameIsRead separates the nil refusal from
// the missing-name one. Both mention propertyValues, so the message alone does
// not tell them apart -- the NAME in the missing-name message does.
func TestANilDictionaryIsRefusedBeforeAnyNameIsRead(t *testing.T) {
	_, err := NewVector3Converter().CreateInstance(nil, nil)
	if err == nil {
		t.Fatal("a nil dictionary was accepted")
	}
	if strings.Contains(err.Error(), "missing") {
		t.Fatalf("a nil dictionary produced the MISSING-NAME refusal %q; the nil check runs first", err)
	}
	// And an empty dictionary DOES produce the missing-name one, which is what
	// makes the assertion above about ordering rather than about wording.
	if _, err = NewVector3Converter().CreateInstance(nil, map[string]any{}); err == nil {
		t.Fatal("an empty dictionary was accepted")
	} else if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("an empty dictionary produced %q, want the missing-name refusal", err)
	}
}

// TestConvertToNamesDestinationTypeEvenForAForeignValue pins the branch ORDER
// by its message: with the nil check second, a foreign value would reach the
// fall-through and report the wrong thing.
func TestConvertToNamesDestinationTypeEvenForAForeignValue(t *testing.T) {
	_, err := NewVector2Converter().ConvertTo(nil, nil, "not a vector", nil)
	if err == nil {
		t.Fatal("a nil destinationType was accepted")
	}
	if !strings.Contains(err.Error(), "destinationType") {
		t.Fatalf("the refusal was %q; the nil check runs FIRST, before the value is looked at", err)
	}
}

// TestEachConvertersParameterNamesAreItsOwn pins the per-converter name lists,
// which the refusal message and CreateInstance both read.
func TestEachConvertersParameterNamesAreItsOwn(t *testing.T) {
	_, err := NewVector2Converter().ConvertFrom(nil, nil, "1")
	if err == nil {
		t.Fatal("a one-component string parsed as a Vector2")
	}
	if !strings.Contains(err.Error(), "X,Y") || strings.Contains(err.Error(), "Z") {
		t.Fatalf("Vector2's expected parameters read %q, want exactly X and Y", err)
	}
	// CreateInstance reads the same names.
	value, err := NewVector2Converter().CreateInstance(nil, map[string]any{"X": float32(1), "Y": float32(2)})
	if err != nil {
		t.Fatalf("Vector2Converter.CreateInstance: %v", err)
	}
	if value != framework.NewVector2BySingleAndSingle(1, 2) {
		t.Fatalf("Vector2Converter.CreateInstance = %v", value)
	}
}

// TestColorFormatsItsChannelsInOrder pins that ConvertTo reads R, G, B, A in
// the constructor's order rather than any other.
func TestColorFormatsItsChannelsInOrder(t *testing.T) {
	color := framework.NewColorByInt32AndInt32AndInt32AndInt32(1, 2, 3, 4)
	formatted, err := NewColorConverter().ConvertTo(nil, nil, color, reflect.TypeOf(""))
	if err != nil {
		t.Fatalf("ColorConverter.ConvertTo: %v", err)
	}
	if formatted != "1, 2, 3, 4" {
		t.Fatalf("ColorConverter formatted %q, want the channels in R G B A order", formatted)
	}
	// Rectangle formats even though it refuses to parse, which is the flag's
	// asymmetry seen from the value side.
	formatted, err = NewRectangleConverter().ConvertTo(nil, nil,
		framework.NewRectangle(1, 2, 3, 4), reflect.TypeOf(""))
	if err != nil {
		t.Fatalf("RectangleConverter.ConvertTo: %v", err)
	}
	if formatted != "1, 2, 3, 4" {
		t.Fatalf("RectangleConverter formatted %q", formatted)
	}
}

// TestTheSixFlagClearingConvertersRefuseAStringSource pins the flag on every
// one of them, not just on the Rectangle the earlier test uses.
func TestTheSixFlagClearingConvertersRefuseAStringSource(t *testing.T) {
	stringType := reflect.TypeOf("")
	for name, canConvert := range map[string]func(any, reflect.Type) bool{
		"Rectangle":      NewRectangleConverter().CanConvertFrom,
		"Plane":          NewPlaneConverter().CanConvertFrom,
		"Ray":            NewRayConverter().CanConvertFrom,
		"BoundingBox":    NewBoundingBoxConverter().CanConvertFrom,
		"BoundingSphere": NewBoundingSphereConverter().CanConvertFrom,
		"Matrix":         NewMatrixConverter().CanConvertFrom,
	} {
		if canConvert(nil, stringType) {
			t.Errorf("%sConverter accepts a string source; its constructor clears supportStringConvert", name)
		}
	}
	// And the six that keep the flag do accept one.
	for name, canConvert := range map[string]func(any, reflect.Type) bool{
		"Vector2":    NewVector2Converter().CanConvertFrom,
		"Vector3":    NewVector3Converter().CanConvertFrom,
		"Vector4":    NewVector4Converter().CanConvertFrom,
		"Quaternion": NewQuaternionConverter().CanConvertFrom,
		"Point":      NewPointConverter().CanConvertFrom,
		"Color":      NewColorConverter().CanConvertFrom,
	} {
		if !canConvert(nil, stringType) {
			t.Errorf("%sConverter refuses a string source; it leaves supportStringConvert true", name)
		}
	}
}

// TestMatrixCellDescriptorsReadTheirOwnCell pins all sixteen, which a single
// Translation read cannot: Translation is computed from three of them and would
// agree with several wrong cell mappings.
func TestMatrixCellDescriptorsReadTheirOwnCell(t *testing.T) {
	// A matrix whose every cell is distinct, so a descriptor reading the wrong
	// one is visible.
	matrix := framework.NewMatrix(
		1, 2, 3, 4,
		5, 6, 7, 8,
		9, 10, 11, 12,
		13, 14, 15, 16)
	properties := NewMatrixConverter().GetProperties(nil, nil, nil)
	for index, name := range matrixCellNames {
		descriptor := properties[index+1]
		if descriptor.Name() != name {
			t.Fatalf("descriptor %d is %q, want %q", index+1, descriptor.Name(), name)
		}
		read, ok := descriptor.GetValue(matrix)
		if !ok || read != float32(index+1) {
			t.Fatalf("the %s descriptor read %v, want %d", name, read, index+1)
		}
	}
}

// TestMatrixCreateInstanceReadsSixteenCellsAndNotTranslation pins that the
// computed property is NOT one of the constructor's inputs.
func TestMatrixCreateInstanceReadsSixteenCellsAndNotTranslation(t *testing.T) {
	values := map[string]any{}
	for index, name := range matrixCellNames {
		values[name] = float32(index + 1)
	}
	built, err := NewMatrixConverter().CreateInstance(nil, values)
	if err != nil {
		t.Fatalf("MatrixConverter.CreateInstance: %v", err)
	}
	want := framework.NewMatrix(
		1, 2, 3, 4,
		5, 6, 7, 8,
		9, 10, 11, 12,
		13, 14, 15, 16)
	if built != want {
		t.Fatalf("MatrixConverter.CreateInstance = %v", built)
	}
	// Translation is absent from the dictionary and the build still succeeds,
	// which is what says the sixteen cells are the whole input.
	if _, present := values["Translation"]; present {
		t.Fatal("the fixture supplied Translation; the point is that it is not needed")
	}
}

// TestAnInstanceDescriptorDoesNotAliasTheCallersSlice pins the copy on the way
// IN, which the copy on the way out does not cover.
func TestAnInstanceDescriptorDoesNotAliasTheCallersSlice(t *testing.T) {
	arguments := []any{float32(1), float32(2)}
	descriptor := framework.NewInstanceDescriptor(reflect.TypeOf(framework.Vector2{}), arguments)
	arguments[0] = float32(99)
	if got := descriptor.Arguments(); got[0] != float32(1) {
		t.Fatalf("the descriptor carries %v; mutating the caller's slice reached it", got)
	}
}
