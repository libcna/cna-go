package design

import (
	"reflect"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// The six converters that CLEAR supportStringConvert.
//
// Measured, not assumed: each of these six stores `ldc.i4.0` into the inherited
// field in its own constructor, so CanConvertFrom stops answering true for a
// string. None of them calls ConvertToValues or ConvertFromValues at all.
//
// Three declare NO ConvertFrom -- Rectangle, Plane and Matrix -- and three
// declare one whose entire body forwards to the base: Ray, BoundingBox and
// BoundingSphere. A forwarding override and no override are the same behaviour,
// and the projection keeps the distinction because the CONTRACT does: the three
// forwarding ones declare the member and the other three do not.
//
// Their ConvertTo still answers for a string, because MathTypeConverter's
// CanConvertTo consults the base -- which compares against typeof(string)
// without looking at the flag. So these values format and do not parse, which
// is the asymmetry the flag creates.

// RectangleConverter is Microsoft.Xna.Framework.Design.RectangleConverter.
type RectangleConverter struct{ base *MathTypeConverter }

// NewRectangleConverter is the .ctor(), which CLEARS supportStringConvert.
func NewRectangleConverter() *RectangleConverter {
	converter := &RectangleConverter{base: NewMathTypeConverter()}
	converter.base.supportStringConvert = false
	converter.base.propertyDescriptions = []framework.PropertyDescriptor{
		framework.NewPropertyDescriptor("X", func(c any) (any, bool) {
			v, ok := c.(framework.Rectangle)
			return v.X, ok
		}),
		framework.NewPropertyDescriptor("Y", func(c any) (any, bool) {
			v, ok := c.(framework.Rectangle)
			return v.Y, ok
		}),
		framework.NewPropertyDescriptor("Width", func(c any) (any, bool) {
			v, ok := c.(framework.Rectangle)
			return v.Width, ok
		}),
		framework.NewPropertyDescriptor("Height", func(c any) (any, bool) {
			v, ok := c.(framework.Rectangle)
			return v.Height, ok
		}),
	}
	return converter
}

var rectangleParams = []string{"X", "Y", "Width", "Height"}

// ConvertTo is RectangleConverter::ConvertTo. It formats the four components
// with the culture's separator even though the converter refuses to PARSE one.
func (c *RectangleConverter) ConvertTo(context any, culture *framework.CultureInfo, value any, destinationType reflect.Type) (any, error) {
	if destinationType == nil {
		return nil, argumentNullError("destinationType")
	}
	rectangle, ok := value.(framework.Rectangle)
	if ok {
		components := []int32{rectangle.X, rectangle.Y, rectangle.Width, rectangle.Height}
		switch destinationType {
		case stringType:
			return convertFromValues(culture, components, formatInt32), nil
		case instanceDescriptorType:
			return framework.NewInstanceDescriptor(reflect.TypeOf(framework.Rectangle{}),
				[]any{components[0], components[1], components[2], components[3]}), nil
		}
	}
	return nil, notSupportedError(destinationType)
}

// CreateInstance is RectangleConverter::CreateInstance.
func (c *RectangleConverter) CreateInstance(context any, propertyValues map[string]any) (any, error) {
	values, err := int32Components(propertyValues, rectangleParams)
	if err != nil {
		return nil, err
	}
	return framework.NewRectangle(values[0], values[1], values[2], values[3]), nil
}

// CanConvertFrom is the inherited MathTypeConverter::CanConvertFrom. With the
// flag cleared it answers true only for an InstanceDescriptor.
func (c *RectangleConverter) CanConvertFrom(context any, sourceType reflect.Type) bool {
	return c.base.CanConvertFrom(context, sourceType)
}

// CanConvertTo is the inherited MathTypeConverter::CanConvertTo.
func (c *RectangleConverter) CanConvertTo(context any, destinationType reflect.Type) bool {
	return c.base.CanConvertTo(context, destinationType)
}

// GetCreateInstanceSupported is the inherited getter.
func (c *RectangleConverter) GetCreateInstanceSupported(context any) bool {
	return c.base.GetCreateInstanceSupported(context)
}

// GetPropertiesSupported is the inherited getter.
func (c *RectangleConverter) GetPropertiesSupported(context any) bool {
	return c.base.GetPropertiesSupported(context)
}

// GetProperties is the inherited MathTypeConverter::GetProperties.
func (c *RectangleConverter) GetProperties(context any, value any, attributes []any) []framework.PropertyDescriptor {
	return c.base.GetProperties(context, value, attributes)
}

// PlaneConverter is Microsoft.Xna.Framework.Design.PlaneConverter.
type PlaneConverter struct{ base *MathTypeConverter }

// NewPlaneConverter is the .ctor(), which CLEARS supportStringConvert.
func NewPlaneConverter() *PlaneConverter {
	converter := &PlaneConverter{base: NewMathTypeConverter()}
	converter.base.supportStringConvert = false
	converter.base.propertyDescriptions = []framework.PropertyDescriptor{
		framework.NewPropertyDescriptor("Normal", func(c any) (any, bool) {
			v, ok := c.(framework.Plane)
			return v.Normal, ok
		}),
		framework.NewPropertyDescriptor("D", func(c any) (any, bool) {
			v, ok := c.(framework.Plane)
			return v.D, ok
		}),
	}
	return converter
}

var planeParams = []string{"Normal", "D"}

// ConvertTo is PlaneConverter::ConvertTo. Its components are not Singles, so there is
// no ConvertFromValues call: the string case formats through the components'
// own converters, which for a Vector3 is its converter and not this one.
func (c *PlaneConverter) ConvertTo(context any, culture *framework.CultureInfo, value any, destinationType reflect.Type) (any, error) {
	if destinationType == nil {
		return nil, argumentNullError("destinationType")
	}
	plane, ok := value.(framework.Plane)
	if ok && destinationType == instanceDescriptorType {
		return framework.NewInstanceDescriptor(reflect.TypeOf(framework.Plane{}),
			[]any{plane.Normal, plane.D}), nil
	}
	return nil, notSupportedError(destinationType)
}

// CreateInstance is PlaneConverter::CreateInstance, which reads by NAME.
func (c *PlaneConverter) CreateInstance(context any, propertyValues map[string]any) (any, error) {
	if propertyValues == nil {
		return nil, argumentNullError("propertyValues")
	}
	normals, err := vector3Components(propertyValues, []string{"Normal"})
	if err != nil {
		return nil, err
	}
	scalars, err := singleComponents(propertyValues, []string{"D"})
	if err != nil {
		return nil, err
	}
	normal, d := normals[0], scalars[0]
	return framework.NewPlaneByVector3AndSingle(normal, d), nil
}

// CanConvertFrom is the inherited MathTypeConverter::CanConvertFrom. With the
// flag cleared it answers true only for an InstanceDescriptor.
func (c *PlaneConverter) CanConvertFrom(context any, sourceType reflect.Type) bool {
	return c.base.CanConvertFrom(context, sourceType)
}

// CanConvertTo is the inherited MathTypeConverter::CanConvertTo.
func (c *PlaneConverter) CanConvertTo(context any, destinationType reflect.Type) bool {
	return c.base.CanConvertTo(context, destinationType)
}

// GetCreateInstanceSupported is the inherited getter.
func (c *PlaneConverter) GetCreateInstanceSupported(context any) bool {
	return c.base.GetCreateInstanceSupported(context)
}

// GetPropertiesSupported is the inherited getter.
func (c *PlaneConverter) GetPropertiesSupported(context any) bool {
	return c.base.GetPropertiesSupported(context)
}

// GetProperties is the inherited MathTypeConverter::GetProperties.
func (c *PlaneConverter) GetProperties(context any, value any, attributes []any) []framework.PropertyDescriptor {
	return c.base.GetProperties(context, value, attributes)
}

// RayConverter is Microsoft.Xna.Framework.Design.RayConverter.
type RayConverter struct{ base *MathTypeConverter }

// NewRayConverter is the .ctor(), which CLEARS supportStringConvert.
func NewRayConverter() *RayConverter {
	converter := &RayConverter{base: NewMathTypeConverter()}
	converter.base.supportStringConvert = false
	converter.base.propertyDescriptions = []framework.PropertyDescriptor{
		framework.NewPropertyDescriptor("Position", func(c any) (any, bool) {
			v, ok := c.(framework.Ray)
			return v.Position, ok
		}),
		framework.NewPropertyDescriptor("Direction", func(c any) (any, bool) {
			v, ok := c.(framework.Ray)
			return v.Direction, ok
		}),
	}
	return converter
}

var rayParams = []string{"Position", "Direction"}

// ConvertFrom is RayConverter::ConvertFrom, whose ENTIRE body is
// `return base.ConvertFrom(context, culture, value)`. It overrides the member
// and changes nothing, which is why the projection declares it: the contract
// declares it too, and the three converters that do not are a measured
// difference rather than an omission.
func (c *RayConverter) ConvertFrom(context any, culture *framework.CultureInfo, value any) (any, error) {
	return nil, notSupportedError(value)
}

// ConvertTo is RayConverter::ConvertTo. Its components are not Singles, so there is
// no ConvertFromValues call: the string case formats through the components'
// own converters, which for a Vector3 is its converter and not this one.
func (c *RayConverter) ConvertTo(context any, culture *framework.CultureInfo, value any, destinationType reflect.Type) (any, error) {
	if destinationType == nil {
		return nil, argumentNullError("destinationType")
	}
	ray, ok := value.(framework.Ray)
	if ok && destinationType == instanceDescriptorType {
		return framework.NewInstanceDescriptor(reflect.TypeOf(framework.Ray{}),
			[]any{ray.Position, ray.Direction}), nil
	}
	return nil, notSupportedError(destinationType)
}

// CreateInstance is RayConverter::CreateInstance, which reads by NAME.
func (c *RayConverter) CreateInstance(context any, propertyValues map[string]any) (any, error) {
	values, err := vector3Components(propertyValues, rayParams)
	if err != nil {
		return nil, err
	}
	position, direction := values[0], values[1]
	return framework.NewRay(position, direction), nil
}

// CanConvertFrom is the inherited MathTypeConverter::CanConvertFrom. With the
// flag cleared it answers true only for an InstanceDescriptor.
func (c *RayConverter) CanConvertFrom(context any, sourceType reflect.Type) bool {
	return c.base.CanConvertFrom(context, sourceType)
}

// CanConvertTo is the inherited MathTypeConverter::CanConvertTo.
func (c *RayConverter) CanConvertTo(context any, destinationType reflect.Type) bool {
	return c.base.CanConvertTo(context, destinationType)
}

// GetCreateInstanceSupported is the inherited getter.
func (c *RayConverter) GetCreateInstanceSupported(context any) bool {
	return c.base.GetCreateInstanceSupported(context)
}

// GetPropertiesSupported is the inherited getter.
func (c *RayConverter) GetPropertiesSupported(context any) bool {
	return c.base.GetPropertiesSupported(context)
}

// GetProperties is the inherited MathTypeConverter::GetProperties.
func (c *RayConverter) GetProperties(context any, value any, attributes []any) []framework.PropertyDescriptor {
	return c.base.GetProperties(context, value, attributes)
}

// BoundingBoxConverter is Microsoft.Xna.Framework.Design.BoundingBoxConverter.
type BoundingBoxConverter struct{ base *MathTypeConverter }

// NewBoundingBoxConverter is the .ctor(), which CLEARS supportStringConvert.
func NewBoundingBoxConverter() *BoundingBoxConverter {
	converter := &BoundingBoxConverter{base: NewMathTypeConverter()}
	converter.base.supportStringConvert = false
	converter.base.propertyDescriptions = []framework.PropertyDescriptor{
		framework.NewPropertyDescriptor("Min", func(c any) (any, bool) {
			v, ok := c.(framework.BoundingBox)
			return v.Min, ok
		}),
		framework.NewPropertyDescriptor("Max", func(c any) (any, bool) {
			v, ok := c.(framework.BoundingBox)
			return v.Max, ok
		}),
	}
	return converter
}

var boundingBoxParams = []string{"Min", "Max"}

// ConvertFrom is BoundingBoxConverter::ConvertFrom, whose ENTIRE body is
// `return base.ConvertFrom(context, culture, value)`. It overrides the member
// and changes nothing, which is why the projection declares it: the contract
// declares it too, and the three converters that do not are a measured
// difference rather than an omission.
func (c *BoundingBoxConverter) ConvertFrom(context any, culture *framework.CultureInfo, value any) (any, error) {
	return nil, notSupportedError(value)
}

// ConvertTo is BoundingBoxConverter::ConvertTo. Its components are not Singles, so there is
// no ConvertFromValues call: the string case formats through the components'
// own converters, which for a Vector3 is its converter and not this one.
func (c *BoundingBoxConverter) ConvertTo(context any, culture *framework.CultureInfo, value any, destinationType reflect.Type) (any, error) {
	if destinationType == nil {
		return nil, argumentNullError("destinationType")
	}
	boundingBox, ok := value.(framework.BoundingBox)
	if ok && destinationType == instanceDescriptorType {
		return framework.NewInstanceDescriptor(reflect.TypeOf(framework.BoundingBox{}),
			[]any{boundingBox.Min, boundingBox.Max}), nil
	}
	return nil, notSupportedError(destinationType)
}

// CreateInstance is BoundingBoxConverter::CreateInstance, which reads by NAME.
func (c *BoundingBoxConverter) CreateInstance(context any, propertyValues map[string]any) (any, error) {
	values, err := vector3Components(propertyValues, boundingBoxParams)
	if err != nil {
		return nil, err
	}
	min, max := values[0], values[1]
	return framework.NewBoundingBox(min, max), nil
}

// CanConvertFrom is the inherited MathTypeConverter::CanConvertFrom. With the
// flag cleared it answers true only for an InstanceDescriptor.
func (c *BoundingBoxConverter) CanConvertFrom(context any, sourceType reflect.Type) bool {
	return c.base.CanConvertFrom(context, sourceType)
}

// CanConvertTo is the inherited MathTypeConverter::CanConvertTo.
func (c *BoundingBoxConverter) CanConvertTo(context any, destinationType reflect.Type) bool {
	return c.base.CanConvertTo(context, destinationType)
}

// GetCreateInstanceSupported is the inherited getter.
func (c *BoundingBoxConverter) GetCreateInstanceSupported(context any) bool {
	return c.base.GetCreateInstanceSupported(context)
}

// GetPropertiesSupported is the inherited getter.
func (c *BoundingBoxConverter) GetPropertiesSupported(context any) bool {
	return c.base.GetPropertiesSupported(context)
}

// GetProperties is the inherited MathTypeConverter::GetProperties.
func (c *BoundingBoxConverter) GetProperties(context any, value any, attributes []any) []framework.PropertyDescriptor {
	return c.base.GetProperties(context, value, attributes)
}

// BoundingSphereConverter is Microsoft.Xna.Framework.Design.BoundingSphereConverter.
type BoundingSphereConverter struct{ base *MathTypeConverter }

// NewBoundingSphereConverter is the .ctor(), which CLEARS supportStringConvert.
func NewBoundingSphereConverter() *BoundingSphereConverter {
	converter := &BoundingSphereConverter{base: NewMathTypeConverter()}
	converter.base.supportStringConvert = false
	converter.base.propertyDescriptions = []framework.PropertyDescriptor{
		framework.NewPropertyDescriptor("Center", func(c any) (any, bool) {
			v, ok := c.(framework.BoundingSphere)
			return v.Center, ok
		}),
		framework.NewPropertyDescriptor("Radius", func(c any) (any, bool) {
			v, ok := c.(framework.BoundingSphere)
			return v.Radius, ok
		}),
	}
	return converter
}

var boundingSphereParams = []string{"Center", "Radius"}

// ConvertFrom is BoundingSphereConverter::ConvertFrom, whose ENTIRE body is
// `return base.ConvertFrom(context, culture, value)`. It overrides the member
// and changes nothing, which is why the projection declares it: the contract
// declares it too, and the three converters that do not are a measured
// difference rather than an omission.
func (c *BoundingSphereConverter) ConvertFrom(context any, culture *framework.CultureInfo, value any) (any, error) {
	return nil, notSupportedError(value)
}

// ConvertTo is BoundingSphereConverter::ConvertTo. Its components are not Singles, so there is
// no ConvertFromValues call: the string case formats through the components'
// own converters, which for a Vector3 is its converter and not this one.
func (c *BoundingSphereConverter) ConvertTo(context any, culture *framework.CultureInfo, value any, destinationType reflect.Type) (any, error) {
	if destinationType == nil {
		return nil, argumentNullError("destinationType")
	}
	boundingSphere, ok := value.(framework.BoundingSphere)
	if ok && destinationType == instanceDescriptorType {
		return framework.NewInstanceDescriptor(reflect.TypeOf(framework.BoundingSphere{}),
			[]any{boundingSphere.Center, boundingSphere.Radius}), nil
	}
	return nil, notSupportedError(destinationType)
}

// CreateInstance is BoundingSphereConverter::CreateInstance, which reads by NAME.
func (c *BoundingSphereConverter) CreateInstance(context any, propertyValues map[string]any) (any, error) {
	if propertyValues == nil {
		return nil, argumentNullError("propertyValues")
	}
	centers, err := vector3Components(propertyValues, []string{"Center"})
	if err != nil {
		return nil, err
	}
	radii, err := singleComponents(propertyValues, []string{"Radius"})
	if err != nil {
		return nil, err
	}
	center, radius := centers[0], radii[0]
	return framework.NewBoundingSphere(center, radius), nil
}

// CanConvertFrom is the inherited MathTypeConverter::CanConvertFrom. With the
// flag cleared it answers true only for an InstanceDescriptor.
func (c *BoundingSphereConverter) CanConvertFrom(context any, sourceType reflect.Type) bool {
	return c.base.CanConvertFrom(context, sourceType)
}

// CanConvertTo is the inherited MathTypeConverter::CanConvertTo.
func (c *BoundingSphereConverter) CanConvertTo(context any, destinationType reflect.Type) bool {
	return c.base.CanConvertTo(context, destinationType)
}

// GetCreateInstanceSupported is the inherited getter.
func (c *BoundingSphereConverter) GetCreateInstanceSupported(context any) bool {
	return c.base.GetCreateInstanceSupported(context)
}

// GetPropertiesSupported is the inherited getter.
func (c *BoundingSphereConverter) GetPropertiesSupported(context any) bool {
	return c.base.GetPropertiesSupported(context)
}

// GetProperties is the inherited MathTypeConverter::GetProperties.
func (c *BoundingSphereConverter) GetProperties(context any, value any, attributes []any) []framework.PropertyDescriptor {
	return c.base.GetProperties(context, value, attributes)
}

// MatrixConverter is Microsoft.Xna.Framework.Design.MatrixConverter, and its
// constructor is the one that is not like the others.
//
// It builds SEVENTEEN descriptors, measured from `ldc.i4.s 17`:
//
//	Translation, found on TypeDescriptor.GetProperties(typeof(Matrix)) by name
//	M11 through M44, each from typeof(Matrix).GetField(name)
//
// So the collection mixes a PROPERTY with sixteen FIELDS, and Translation comes
// FIRST -- before M11, not after M44 and not in the alphabetical order a
// PropertyDescriptorCollection would otherwise take.
//
// Its Sort names only "Translation", which puts that one at the front and
// leaves the sixteen in the order they were added.
type MatrixConverter struct{ base *MathTypeConverter }

// NewMatrixConverter is the .ctor(), which CLEARS supportStringConvert.
func NewMatrixConverter() *MatrixConverter {
	converter := &MatrixConverter{base: NewMathTypeConverter()}
	converter.base.supportStringConvert = false
	descriptors := make([]framework.PropertyDescriptor, 0, 17)
	// Translation is a PROPERTY on Matrix -- a computed Vector3 over M41, M42
	// and M43 -- which is why it is found through the property collection in
	// the reference rather than through GetField.
	descriptors = append(descriptors, framework.NewPropertyDescriptor("Translation",
		func(c any) (any, bool) {
			v, ok := c.(framework.Matrix)
			return v.Translation(), ok
		}))
	for _, name := range matrixCellNames {
		cell := name
		descriptors = append(descriptors, framework.NewPropertyDescriptor(cell,
			func(c any) (any, bool) {
				v, ok := c.(framework.Matrix)
				if !ok {
					return nil, false
				}
				return matrixCell(v, cell), true
			}))
	}
	converter.base.propertyDescriptions = descriptors
	return converter
}

// matrixCellNames are M11 through M44 in the reference's own order: row by row.
var matrixCellNames = []string{
	"M11", "M12", "M13", "M14",
	"M21", "M22", "M23", "M24",
	"M31", "M32", "M33", "M34",
	"M41", "M42", "M43", "M44",
}

// matrixCell reads one cell by name. It is a switch rather than reflection
// because the sixteen names are a closed, measured list and a reflect lookup
// would turn a compile-time guarantee into a runtime one.
func matrixCell(m framework.Matrix, name string) float32 {
	switch name {
	case "M11":
		return m.M11
	case "M12":
		return m.M12
	case "M13":
		return m.M13
	case "M14":
		return m.M14
	case "M21":
		return m.M21
	case "M22":
		return m.M22
	case "M23":
		return m.M23
	case "M24":
		return m.M24
	case "M31":
		return m.M31
	case "M32":
		return m.M32
	case "M33":
		return m.M33
	case "M34":
		return m.M34
	case "M41":
		return m.M41
	case "M42":
		return m.M42
	case "M43":
		return m.M43
	default:
		return m.M44
	}
}

// ConvertTo is MatrixConverter::ConvertTo.
func (c *MatrixConverter) ConvertTo(context any, culture *framework.CultureInfo, value any, destinationType reflect.Type) (any, error) {
	if destinationType == nil {
		return nil, argumentNullError("destinationType")
	}
	matrix, ok := value.(framework.Matrix)
	if ok && destinationType == instanceDescriptorType {
		arguments := make([]any, 0, 16)
		for _, name := range matrixCellNames {
			arguments = append(arguments, matrixCell(matrix, name))
		}
		return framework.NewInstanceDescriptor(reflect.TypeOf(framework.Matrix{}), arguments), nil
	}
	return nil, notSupportedError(destinationType)
}

// CreateInstance is MatrixConverter::CreateInstance, which reads all SIXTEEN
// cells by name -- not Translation, which is computed from three of them.
func (c *MatrixConverter) CreateInstance(context any, propertyValues map[string]any) (any, error) {
	values, err := singleComponents(propertyValues, matrixCellNames)
	if err != nil {
		return nil, err
	}
	return framework.NewMatrix(
		values[0], values[1], values[2], values[3],
		values[4], values[5], values[6], values[7],
		values[8], values[9], values[10], values[11],
		values[12], values[13], values[14], values[15]), nil
}

// CanConvertFrom is the inherited MathTypeConverter::CanConvertFrom.
func (c *MatrixConverter) CanConvertFrom(context any, sourceType reflect.Type) bool {
	return c.base.CanConvertFrom(context, sourceType)
}

// CanConvertTo is the inherited MathTypeConverter::CanConvertTo.
func (c *MatrixConverter) CanConvertTo(context any, destinationType reflect.Type) bool {
	return c.base.CanConvertTo(context, destinationType)
}

// GetCreateInstanceSupported is the inherited getter.
func (c *MatrixConverter) GetCreateInstanceSupported(context any) bool {
	return c.base.GetCreateInstanceSupported(context)
}

// GetPropertiesSupported is the inherited getter.
func (c *MatrixConverter) GetPropertiesSupported(context any) bool {
	return c.base.GetPropertiesSupported(context)
}

// GetProperties is the inherited MathTypeConverter::GetProperties.
func (c *MatrixConverter) GetProperties(context any, value any, attributes []any) []framework.PropertyDescriptor {
	return c.base.GetProperties(context, value, attributes)
}
