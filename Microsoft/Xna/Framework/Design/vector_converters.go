package design

import (
	"reflect"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// The four converters over Single components, and the two over other element
// types. All six set no flag, so supportStringConvert stays TRUE from
// MathTypeConverter's constructor and CanConvertFrom answers true for a string.
//
// Each is the same four members over a different field list; what differs
// between them is the list, the element type and how a value is taken apart and
// put back together. Those are written out per converter rather than driven
// from a table, because the reference writes them out too and a table would
// hide the one thing that is not uniform: Color reflects PROPERTIES where the
// vectors reflect FIELDS.

// Vector2Converter is Microsoft.Xna.Framework.Design.Vector2Converter.
type Vector2Converter struct{ base *MathTypeConverter }

// NewVector2Converter is the .ctor(), which installs the property descriptions
// and leaves supportStringConvert at the base's true.
func NewVector2Converter() *Vector2Converter {
	converter := &Vector2Converter{base: NewMathTypeConverter()}
	converter.base.propertyDescriptions = []framework.PropertyDescriptor{
		framework.NewPropertyDescriptor("X", func(c any) (any, bool) {
			v, ok := c.(framework.Vector2)
			return v.X, ok
		}),
		framework.NewPropertyDescriptor("Y", func(c any) (any, bool) {
			v, ok := c.(framework.Vector2)
			return v.Y, ok
		}),
	}
	return converter
}

var vector2Params = []string{"X", "Y"}

// ConvertFrom is Vector2Converter::ConvertFrom, measured in full:
//
//	float[] values = ConvertToValues<float>(context, culture, value, 2, "X", "Y");
//	if (values != null) return new Vector2(values[0], values[1]);
//	return base.ConvertFrom(context, culture, value);
func (c *Vector2Converter) ConvertFrom(context any, culture *framework.CultureInfo, value any) (any, error) {
	values, err := convertToValues(culture, value, 2, vector2Params, parseSingle)
	if err != nil {
		return nil, err
	}
	if values != nil {
		return framework.NewVector2BySingleAndSingle(values[0], values[1]), nil
	}
	return nil, notSupportedError(value)
}

// ConvertTo is Vector2Converter::ConvertTo, whose four branches run in the
// reference's own order: the nil check, the string case, the InstanceDescriptor
// case, then the base.
func (c *Vector2Converter) ConvertTo(context any, culture *framework.CultureInfo, value any, destinationType reflect.Type) (any, error) {
	if destinationType == nil {
		return nil, argumentNullError("destinationType")
	}
	vector, ok := value.(framework.Vector2)
	if ok {
		components := []float32{vector.X, vector.Y}
		switch destinationType {
		case stringType:
			return convertFromValues(culture, components, formatSingle), nil
		case instanceDescriptorType:
			return framework.NewInstanceDescriptor(reflect.TypeOf(framework.Vector2{}),
				[]any{components[0], components[1]}), nil
		}
	}
	return nil, notSupportedError(destinationType)
}

// CreateInstance is Vector2Converter::CreateInstance, which reads by NAME.
func (c *Vector2Converter) CreateInstance(context any, propertyValues map[string]any) (any, error) {
	values, err := singleComponents(propertyValues, vector2Params)
	if err != nil {
		return nil, err
	}
	return framework.NewVector2BySingleAndSingle(values[0], values[1]), nil
}

// CanConvertFrom is the inherited MathTypeConverter::CanConvertFrom.
func (c *Vector2Converter) CanConvertFrom(context any, sourceType reflect.Type) bool {
	return c.base.CanConvertFrom(context, sourceType)
}

// CanConvertTo is the inherited MathTypeConverter::CanConvertTo.
func (c *Vector2Converter) CanConvertTo(context any, destinationType reflect.Type) bool {
	return c.base.CanConvertTo(context, destinationType)
}

// GetCreateInstanceSupported is the inherited getter.
func (c *Vector2Converter) GetCreateInstanceSupported(context any) bool {
	return c.base.GetCreateInstanceSupported(context)
}

// GetPropertiesSupported is the inherited getter.
func (c *Vector2Converter) GetPropertiesSupported(context any) bool {
	return c.base.GetPropertiesSupported(context)
}

// GetProperties is the inherited MathTypeConverter::GetProperties.
func (c *Vector2Converter) GetProperties(context any, value any, attributes []any) []framework.PropertyDescriptor {
	return c.base.GetProperties(context, value, attributes)
}

// Vector3Converter is Microsoft.Xna.Framework.Design.Vector3Converter.
type Vector3Converter struct{ base *MathTypeConverter }

// NewVector3Converter is the .ctor(), which installs the property descriptions and
// leaves supportStringConvert at the base's true.
func NewVector3Converter() *Vector3Converter {
	converter := &Vector3Converter{base: NewMathTypeConverter()}
	converter.base.propertyDescriptions = []framework.PropertyDescriptor{
		framework.NewPropertyDescriptor("X", func(c any) (any, bool) {
			v, ok := c.(framework.Vector3)
			return v.X, ok
		}),
		framework.NewPropertyDescriptor("Y", func(c any) (any, bool) {
			v, ok := c.(framework.Vector3)
			return v.Y, ok
		}),
		framework.NewPropertyDescriptor("Z", func(c any) (any, bool) {
			v, ok := c.(framework.Vector3)
			return v.Z, ok
		}),
	}
	return converter
}

var vector3Params = []string{"X", "Y", "Z"}

// ConvertFrom is Vector3Converter::ConvertFrom, the same shape as Vector2Converter's
// over 3 components.
func (c *Vector3Converter) ConvertFrom(context any, culture *framework.CultureInfo, value any) (any, error) {
	values, err := convertToValues(culture, value, 3, vector3Params, parseSingle)
	if err != nil {
		return nil, err
	}
	if values != nil {
		return framework.NewVector3BySingleAndSingleAndSingle(values[0], values[1], values[2]), nil
	}
	return nil, notSupportedError(value)
}

// ConvertTo is Vector3Converter::ConvertTo.
func (c *Vector3Converter) ConvertTo(context any, culture *framework.CultureInfo, value any, destinationType reflect.Type) (any, error) {
	if destinationType == nil {
		return nil, argumentNullError("destinationType")
	}
	vector, ok := value.(framework.Vector3)
	if ok {
		components := []float32{vector.X, vector.Y, vector.Z}
		switch destinationType {
		case stringType:
			return convertFromValues(culture, components, formatSingle), nil
		case instanceDescriptorType:
			return framework.NewInstanceDescriptor(reflect.TypeOf(framework.Vector3{}),
				[]any{components[0], components[1], components[2]}), nil
		}
	}
	return nil, notSupportedError(destinationType)
}

// CreateInstance is Vector3Converter::CreateInstance, which reads by NAME.
func (c *Vector3Converter) CreateInstance(context any, propertyValues map[string]any) (any, error) {
	values, err := singleComponents(propertyValues, vector3Params)
	if err != nil {
		return nil, err
	}
	return framework.NewVector3BySingleAndSingleAndSingle(values[0], values[1], values[2]), nil
}

// CanConvertFrom is the inherited MathTypeConverter::CanConvertFrom.
func (c *Vector3Converter) CanConvertFrom(context any, sourceType reflect.Type) bool {
	return c.base.CanConvertFrom(context, sourceType)
}

// CanConvertTo is the inherited MathTypeConverter::CanConvertTo.
func (c *Vector3Converter) CanConvertTo(context any, destinationType reflect.Type) bool {
	return c.base.CanConvertTo(context, destinationType)
}

// GetCreateInstanceSupported is the inherited getter.
func (c *Vector3Converter) GetCreateInstanceSupported(context any) bool {
	return c.base.GetCreateInstanceSupported(context)
}

// GetPropertiesSupported is the inherited getter.
func (c *Vector3Converter) GetPropertiesSupported(context any) bool {
	return c.base.GetPropertiesSupported(context)
}

// GetProperties is the inherited MathTypeConverter::GetProperties.
func (c *Vector3Converter) GetProperties(context any, value any, attributes []any) []framework.PropertyDescriptor {
	return c.base.GetProperties(context, value, attributes)
}

// Vector4Converter is Microsoft.Xna.Framework.Design.Vector4Converter.
type Vector4Converter struct{ base *MathTypeConverter }

// NewVector4Converter is the .ctor(), which installs the property descriptions and
// leaves supportStringConvert at the base's true.
func NewVector4Converter() *Vector4Converter {
	converter := &Vector4Converter{base: NewMathTypeConverter()}
	converter.base.propertyDescriptions = []framework.PropertyDescriptor{
		framework.NewPropertyDescriptor("X", func(c any) (any, bool) {
			v, ok := c.(framework.Vector4)
			return v.X, ok
		}),
		framework.NewPropertyDescriptor("Y", func(c any) (any, bool) {
			v, ok := c.(framework.Vector4)
			return v.Y, ok
		}),
		framework.NewPropertyDescriptor("Z", func(c any) (any, bool) {
			v, ok := c.(framework.Vector4)
			return v.Z, ok
		}),
		framework.NewPropertyDescriptor("W", func(c any) (any, bool) {
			v, ok := c.(framework.Vector4)
			return v.W, ok
		}),
	}
	return converter
}

var vector4Params = []string{"X", "Y", "Z", "W"}

// ConvertFrom is Vector4Converter::ConvertFrom, the same shape as Vector2Converter's
// over 4 components.
func (c *Vector4Converter) ConvertFrom(context any, culture *framework.CultureInfo, value any) (any, error) {
	values, err := convertToValues(culture, value, 4, vector4Params, parseSingle)
	if err != nil {
		return nil, err
	}
	if values != nil {
		return framework.NewVector4BySingleAndSingleAndSingleAndSingle(values[0], values[1], values[2], values[3]), nil
	}
	return nil, notSupportedError(value)
}

// ConvertTo is Vector4Converter::ConvertTo.
func (c *Vector4Converter) ConvertTo(context any, culture *framework.CultureInfo, value any, destinationType reflect.Type) (any, error) {
	if destinationType == nil {
		return nil, argumentNullError("destinationType")
	}
	vector, ok := value.(framework.Vector4)
	if ok {
		components := []float32{vector.X, vector.Y, vector.Z, vector.W}
		switch destinationType {
		case stringType:
			return convertFromValues(culture, components, formatSingle), nil
		case instanceDescriptorType:
			return framework.NewInstanceDescriptor(reflect.TypeOf(framework.Vector4{}),
				[]any{components[0], components[1], components[2], components[3]}), nil
		}
	}
	return nil, notSupportedError(destinationType)
}

// CreateInstance is Vector4Converter::CreateInstance, which reads by NAME.
func (c *Vector4Converter) CreateInstance(context any, propertyValues map[string]any) (any, error) {
	values, err := singleComponents(propertyValues, vector4Params)
	if err != nil {
		return nil, err
	}
	return framework.NewVector4BySingleAndSingleAndSingleAndSingle(values[0], values[1], values[2], values[3]), nil
}

// CanConvertFrom is the inherited MathTypeConverter::CanConvertFrom.
func (c *Vector4Converter) CanConvertFrom(context any, sourceType reflect.Type) bool {
	return c.base.CanConvertFrom(context, sourceType)
}

// CanConvertTo is the inherited MathTypeConverter::CanConvertTo.
func (c *Vector4Converter) CanConvertTo(context any, destinationType reflect.Type) bool {
	return c.base.CanConvertTo(context, destinationType)
}

// GetCreateInstanceSupported is the inherited getter.
func (c *Vector4Converter) GetCreateInstanceSupported(context any) bool {
	return c.base.GetCreateInstanceSupported(context)
}

// GetPropertiesSupported is the inherited getter.
func (c *Vector4Converter) GetPropertiesSupported(context any) bool {
	return c.base.GetPropertiesSupported(context)
}

// GetProperties is the inherited MathTypeConverter::GetProperties.
func (c *Vector4Converter) GetProperties(context any, value any, attributes []any) []framework.PropertyDescriptor {
	return c.base.GetProperties(context, value, attributes)
}

// QuaternionConverter is Microsoft.Xna.Framework.Design.QuaternionConverter.
type QuaternionConverter struct{ base *MathTypeConverter }

// NewQuaternionConverter is the .ctor(), which installs the property descriptions and
// leaves supportStringConvert at the base's true.
func NewQuaternionConverter() *QuaternionConverter {
	converter := &QuaternionConverter{base: NewMathTypeConverter()}
	converter.base.propertyDescriptions = []framework.PropertyDescriptor{
		framework.NewPropertyDescriptor("X", func(c any) (any, bool) {
			v, ok := c.(framework.Quaternion)
			return v.X, ok
		}),
		framework.NewPropertyDescriptor("Y", func(c any) (any, bool) {
			v, ok := c.(framework.Quaternion)
			return v.Y, ok
		}),
		framework.NewPropertyDescriptor("Z", func(c any) (any, bool) {
			v, ok := c.(framework.Quaternion)
			return v.Z, ok
		}),
		framework.NewPropertyDescriptor("W", func(c any) (any, bool) {
			v, ok := c.(framework.Quaternion)
			return v.W, ok
		}),
	}
	return converter
}

var quaternionParams = []string{"X", "Y", "Z", "W"}

// ConvertFrom is QuaternionConverter::ConvertFrom, the same shape as Vector2Converter's
// over 4 components.
func (c *QuaternionConverter) ConvertFrom(context any, culture *framework.CultureInfo, value any) (any, error) {
	values, err := convertToValues(culture, value, 4, quaternionParams, parseSingle)
	if err != nil {
		return nil, err
	}
	if values != nil {
		return framework.NewQuaternionBySingleAndSingleAndSingleAndSingle(values[0], values[1], values[2], values[3]), nil
	}
	return nil, notSupportedError(value)
}

// ConvertTo is QuaternionConverter::ConvertTo.
func (c *QuaternionConverter) ConvertTo(context any, culture *framework.CultureInfo, value any, destinationType reflect.Type) (any, error) {
	if destinationType == nil {
		return nil, argumentNullError("destinationType")
	}
	vector, ok := value.(framework.Quaternion)
	if ok {
		components := []float32{vector.X, vector.Y, vector.Z, vector.W}
		switch destinationType {
		case stringType:
			return convertFromValues(culture, components, formatSingle), nil
		case instanceDescriptorType:
			return framework.NewInstanceDescriptor(reflect.TypeOf(framework.Quaternion{}),
				[]any{components[0], components[1], components[2], components[3]}), nil
		}
	}
	return nil, notSupportedError(destinationType)
}

// CreateInstance is QuaternionConverter::CreateInstance, which reads by NAME.
func (c *QuaternionConverter) CreateInstance(context any, propertyValues map[string]any) (any, error) {
	values, err := singleComponents(propertyValues, quaternionParams)
	if err != nil {
		return nil, err
	}
	return framework.NewQuaternionBySingleAndSingleAndSingleAndSingle(values[0], values[1], values[2], values[3]), nil
}

// CanConvertFrom is the inherited MathTypeConverter::CanConvertFrom.
func (c *QuaternionConverter) CanConvertFrom(context any, sourceType reflect.Type) bool {
	return c.base.CanConvertFrom(context, sourceType)
}

// CanConvertTo is the inherited MathTypeConverter::CanConvertTo.
func (c *QuaternionConverter) CanConvertTo(context any, destinationType reflect.Type) bool {
	return c.base.CanConvertTo(context, destinationType)
}

// GetCreateInstanceSupported is the inherited getter.
func (c *QuaternionConverter) GetCreateInstanceSupported(context any) bool {
	return c.base.GetCreateInstanceSupported(context)
}

// GetPropertiesSupported is the inherited getter.
func (c *QuaternionConverter) GetPropertiesSupported(context any) bool {
	return c.base.GetPropertiesSupported(context)
}

// GetProperties is the inherited MathTypeConverter::GetProperties.
func (c *QuaternionConverter) GetProperties(context any, value any, attributes []any) []framework.PropertyDescriptor {
	return c.base.GetProperties(context, value, attributes)
}
