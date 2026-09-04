package design

import (
	"reflect"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// The two converters whose components are not Single, and the four whose
// components are not scalars at all.
//
// PointConverter and ColorConverter instantiate the generic helpers at int32
// and uint8 -- measured from the `ConvertToValues<int32>` and
// `ConvertToValues<uint8>` call sites, not assumed from the value type.
//
// ColorConverter is also the reason PropertyDescriptor is not a
// reflect.StructField: it reflects PROPERTIES, `Type::GetProperty("R")` wrapped
// in the private PropertyPropertyDescriptor, because XNA's Color exposes its
// channels over a packed value. Every other converter here reflects FIELDS.

// PointConverter is Microsoft.Xna.Framework.Design.PointConverter.
type PointConverter struct{ base *MathTypeConverter }

// NewPointConverter is the .ctor(). Point's components are FIELDS.
func NewPointConverter() *PointConverter {
	converter := &PointConverter{base: NewMathTypeConverter()}
	converter.base.propertyDescriptions = []framework.PropertyDescriptor{
		framework.NewPropertyDescriptor("X", func(c any) (any, bool) {
			v, ok := c.(framework.Point)
			return v.X, ok
		}),
		framework.NewPropertyDescriptor("Y", func(c any) (any, bool) {
			v, ok := c.(framework.Point)
			return v.Y, ok
		}),
	}
	return converter
}

var pointParams = []string{"X", "Y"}

// ConvertFrom is PointConverter::ConvertFrom, at `ConvertToValues<int32>`.
func (c *PointConverter) ConvertFrom(context any, culture *framework.CultureInfo, value any) (any, error) {
	values, err := convertToValues(culture, value, 2, pointParams, parseInt32)
	if err != nil {
		return nil, err
	}
	if values != nil {
		return framework.NewPoint(values[0], values[1]), nil
	}
	return nil, notSupportedError(value)
}

// ConvertTo is PointConverter::ConvertTo.
func (c *PointConverter) ConvertTo(context any, culture *framework.CultureInfo, value any, destinationType reflect.Type) (any, error) {
	if destinationType == nil {
		return nil, argumentNullError("destinationType")
	}
	point, ok := value.(framework.Point)
	if ok {
		components := []int32{point.X, point.Y}
		switch destinationType {
		case stringType:
			return convertFromValues(culture, components, formatInt32), nil
		case instanceDescriptorType:
			return framework.NewInstanceDescriptor(reflect.TypeOf(framework.Point{}),
				[]any{components[0], components[1]}), nil
		}
	}
	return nil, notSupportedError(destinationType)
}

// CreateInstance is PointConverter::CreateInstance.
func (c *PointConverter) CreateInstance(context any, propertyValues map[string]any) (any, error) {
	values, err := int32Components(propertyValues, pointParams)
	if err != nil {
		return nil, err
	}
	return framework.NewPoint(values[0], values[1]), nil
}

// CanConvertFrom is the inherited MathTypeConverter::CanConvertFrom.
func (c *PointConverter) CanConvertFrom(context any, sourceType reflect.Type) bool {
	return c.base.CanConvertFrom(context, sourceType)
}

// CanConvertTo is the inherited MathTypeConverter::CanConvertTo.
func (c *PointConverter) CanConvertTo(context any, destinationType reflect.Type) bool {
	return c.base.CanConvertTo(context, destinationType)
}

// GetCreateInstanceSupported is the inherited getter.
func (c *PointConverter) GetCreateInstanceSupported(context any) bool {
	return c.base.GetCreateInstanceSupported(context)
}

// GetPropertiesSupported is the inherited getter.
func (c *PointConverter) GetPropertiesSupported(context any) bool {
	return c.base.GetPropertiesSupported(context)
}

// GetProperties is the inherited MathTypeConverter::GetProperties.
func (c *PointConverter) GetProperties(context any, value any, attributes []any) []framework.PropertyDescriptor {
	return c.base.GetProperties(context, value, attributes)
}

// ColorConverter is Microsoft.Xna.Framework.Design.ColorConverter.
type ColorConverter struct{ base *MathTypeConverter }

// NewColorConverter is the .ctor(). Color's channels are PROPERTIES, which is
// why these descriptors read through the accessor rather than a field.
func NewColorConverter() *ColorConverter {
	converter := &ColorConverter{base: NewMathTypeConverter()}
	converter.base.propertyDescriptions = []framework.PropertyDescriptor{
		framework.NewPropertyDescriptor("R", func(c any) (any, bool) {
			v, ok := c.(framework.Color)
			return v.R(), ok
		}),
		framework.NewPropertyDescriptor("G", func(c any) (any, bool) {
			v, ok := c.(framework.Color)
			return v.G(), ok
		}),
		framework.NewPropertyDescriptor("B", func(c any) (any, bool) {
			v, ok := c.(framework.Color)
			return v.B(), ok
		}),
		framework.NewPropertyDescriptor("A", func(c any) (any, bool) {
			v, ok := c.(framework.Color)
			return v.A(), ok
		}),
	}
	return converter
}

var colorParams = []string{"R", "G", "B", "A"}

// ConvertFrom is ColorConverter::ConvertFrom, at `ConvertToValues<uint8>`. A
// channel outside 0..255 is a parse failure rather than a truncation, which is
// what Byte.Parse does.
func (c *ColorConverter) ConvertFrom(context any, culture *framework.CultureInfo, value any) (any, error) {
	values, err := convertToValues(culture, value, 4, colorParams, parseByte)
	if err != nil {
		return nil, err
	}
	if values != nil {
		return framework.NewColorByInt32AndInt32AndInt32AndInt32(
			int32(values[0]), int32(values[1]), int32(values[2]), int32(values[3])), nil
	}
	return nil, notSupportedError(value)
}

// ConvertTo is ColorConverter::ConvertTo.
func (c *ColorConverter) ConvertTo(context any, culture *framework.CultureInfo, value any, destinationType reflect.Type) (any, error) {
	if destinationType == nil {
		return nil, argumentNullError("destinationType")
	}
	color, ok := value.(framework.Color)
	if ok {
		components := []uint8{color.R(), color.G(), color.B(), color.A()}
		switch destinationType {
		case stringType:
			return convertFromValues(culture, components, formatByte), nil
		case instanceDescriptorType:
			return framework.NewInstanceDescriptor(reflect.TypeOf(framework.Color{}),
				[]any{components[0], components[1], components[2], components[3]}), nil
		}
	}
	return nil, notSupportedError(destinationType)
}

// CreateInstance is ColorConverter::CreateInstance.
func (c *ColorConverter) CreateInstance(context any, propertyValues map[string]any) (any, error) {
	values, err := byteComponents(propertyValues, colorParams)
	if err != nil {
		return nil, err
	}
	return framework.NewColorByInt32AndInt32AndInt32AndInt32(
		int32(values[0]), int32(values[1]), int32(values[2]), int32(values[3])), nil
}

// CanConvertFrom is the inherited MathTypeConverter::CanConvertFrom.
func (c *ColorConverter) CanConvertFrom(context any, sourceType reflect.Type) bool {
	return c.base.CanConvertFrom(context, sourceType)
}

// CanConvertTo is the inherited MathTypeConverter::CanConvertTo.
func (c *ColorConverter) CanConvertTo(context any, destinationType reflect.Type) bool {
	return c.base.CanConvertTo(context, destinationType)
}

// GetCreateInstanceSupported is the inherited getter.
func (c *ColorConverter) GetCreateInstanceSupported(context any) bool {
	return c.base.GetCreateInstanceSupported(context)
}

// GetPropertiesSupported is the inherited getter.
func (c *ColorConverter) GetPropertiesSupported(context any) bool {
	return c.base.GetPropertiesSupported(context)
}

// GetProperties is the inherited MathTypeConverter::GetProperties.
func (c *ColorConverter) GetProperties(context any, value any, attributes []any) []framework.PropertyDescriptor {
	return c.base.GetProperties(context, value, attributes)
}
