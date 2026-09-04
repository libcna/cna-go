// Package design projects Microsoft.Xna.Framework.Design, the thirteen type
// converters that turn XNA's math types into strings and back.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//	System.dll                    c3182e40f09a8d3a...
//
// # Nothing here reaches a runtime
//
// The whole namespace is string parsing, string formatting and field
// description. No member touches CNA, a graphics device or a game.
package design

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// errDesignInvalidFormat projects the InvalidOperationException every converter
// raises for a string it cannot parse.
var errDesignInvalidFormat = errors.New("the string is not in the expected format")

// errDesignNotSupported projects System.NotSupportedException, which the base
// TypeConverter raises for a conversion it does not support.
var errDesignNotSupported = errors.New("the conversion is not supported")

// errDesignArgumentNull projects System.ArgumentNullException, which ConvertTo
// raises for a nil destinationType and CreateInstance for a nil dictionary.
var errDesignArgumentNull = errors.New("argument is null")

// invalidStringFormat is FrameworkResources::InvalidStringFormat, verified byte
// for byte against the retained assembly. Its one placeholder is the EXPECTED
// PARAMETER NAMES joined by the culture's list separator -- not the string that
// failed, which is the substitution a plausible implementation would make.
const invalidStringFormat = "Invalid string format. The correct format is \"%s\"."

// MathTypeConverter is Microsoft.Xna.Framework.Design.MathTypeConverter:
//
//	.class public auto ansi beforefieldinit MathTypeConverter
//	       extends [System]System.ComponentModel.ExpandableObjectConverter
//
// The base the other twelve converters derive from. It supplies the string
// support flag, the property descriptions, and the two generic helpers that do
// the actual splitting and joining.
type MathTypeConverter struct {
	// propertyDescriptions is the `family` field the contract carries. The
	// descriptors are what Type::GetField produced in the reference, which is a
	// FieldInfo -- reflect.StructField here.
	propertyDescriptions []framework.PropertyDescriptor
	// supportStringConvert is the other `family` field. The constructor sets it
	// TRUE and only MatrixConverter clears it.
	supportStringConvert bool
}

// NewMathTypeConverter is MathTypeConverter::.ctor(), whose body sets
// supportStringConvert before calling the base:
//
//	ldarg.0; ldc.i4.1; stfld supportStringConvert
//	ldarg.0; call ExpandableObjectConverter::.ctor()
//
// So string conversion is ON by default, exactly as ContentSerializerAttribute's
// AllowNull is.
func NewMathTypeConverter() *MathTypeConverter {
	return &MathTypeConverter{supportStringConvert: true}
}

// CanConvertFrom is MathTypeConverter::CanConvertFrom(ITypeDescriptorContext, Type):
//
//	if (this.supportStringConvert && sourceType == typeof(string)) return true;
//	return base.CanConvertFrom(context, sourceType);
//
// and the base -- TypeConverter, measured from System.dll -- is
//
//	return sourceType == typeof(InstanceDescriptor);
//
// So the answer is true for a string when the flag is set, true for
// InstanceDescriptor always, and false otherwise. The context is not read.
func (c *MathTypeConverter) CanConvertFrom(context any, sourceType reflect.Type) bool {
	if c.supportStringConvert && sourceType == stringType {
		return true
	}
	return sourceType == instanceDescriptorType
}

// CanConvertTo is MathTypeConverter::CanConvertTo(ITypeDescriptorContext, Type):
//
//	if (destinationType == typeof(InstanceDescriptor)) return true;
//	return base.CanConvertTo(context, destinationType);
//
// and the base is `destinationType == typeof(string)`. Note the ASYMMETRY with
// CanConvertFrom: the InstanceDescriptor answer here is not gated on
// supportStringConvert, and the string answer is not either -- the base's own
// string check runs whatever the flag says. Only CanConvertFrom consults it.
func (c *MathTypeConverter) CanConvertTo(context any, destinationType reflect.Type) bool {
	if destinationType == instanceDescriptorType {
		return true
	}
	return destinationType == stringType
}

// GetCreateInstanceSupported is `ldc.i4.1; ret`.
func (c *MathTypeConverter) GetCreateInstanceSupported(context any) bool { return true }

// GetPropertiesSupported is `ldc.i4.1; ret`.
func (c *MathTypeConverter) GetPropertiesSupported(context any) bool { return true }

// GetProperties is MathTypeConverter::GetProperties, which reads the field and
// ignores all three arguments -- `ldarg.0; ldfld propertyDescriptions; ret`.
func (c *MathTypeConverter) GetProperties(context any, value any, attributes []any) []framework.PropertyDescriptor {
	return c.propertyDescriptions
}

// stringType and instanceDescriptorType are the two `ldtoken` comparisons the
// two CanConvert members make. They are package variables because a reflect
// type literal cannot be a constant, and naming them keeps the comparison
// reading like the IL it reproduces.
var (
	stringType             = reflect.TypeOf("")
	instanceDescriptorType = reflect.TypeOf(&framework.InstanceDescriptor{})
)

// convertToValues is MathTypeConverter::ConvertToValues<T>, the STRING to
// values direction despite the name. It is GENERIC because the reference's is:
// the twelve leaves instantiate it at three different element types --
// float32 for the vectors and the quaternion, int32 for Point, uint8 for Color.
//
// Measured in full:
//
//	string text = value as string;
//	if (text == null) return null;
//	text = text.Trim();
//	if (culture == null) culture = CultureInfo.CurrentCulture;
//	string[] parts = text.Split(new[]{ culture.TextInfo.ListSeparator }, StringSplitOptions.None);
//	... convert each part through TypeDescriptor.GetConverter(typeof(T)) ...
//	if (parts.Length != arrayCount)
//	    throw new InvalidOperationException(
//	        string.Format(CultureInfo.CurrentCulture, FrameworkResources.InvalidStringFormat,
//	                      string.Join(culture.TextInfo.ListSeparator, expectedParams)));
//
// Three details worth naming. The split is on the BARE list separator, not the
// separator-plus-space the join uses -- each element is trimmed by the element
// converter instead. The message names the EXPECTED PARAMETERS, not the string
// that failed. And a non-string value is answered with NIL rather than an
// error, because that is what sends the caller to the base converter.
func convertToValues[T any](culture *framework.CultureInfo, value any, arrayCount int,
	expectedParams []string, parse func(string) (T, error)) ([]T, error) {
	text, ok := value.(string)
	if !ok {
		return nil, nil
	}
	text = strings.TrimSpace(text)
	parts := strings.Split(text, cultureListSeparator(culture))
	values := make([]T, 0, len(parts))
	for _, part := range parts {
		parsed, err := parse(strings.TrimSpace(part))
		if err != nil {
			return nil, invalidFormatError(culture, expectedParams)
		}
		values = append(values, parsed)
	}
	if len(values) != arrayCount {
		return nil, invalidFormatError(culture, expectedParams)
	}
	return values, nil
}

// convertFromValues is MathTypeConverter::ConvertFromValues<T>, the values to
// STRING direction.
//
//	if (culture == null) culture = CultureInfo.CurrentCulture;
//	string separator = culture.TextInfo.ListSeparator + " ";
//	... each value through its converter's ConvertToString ...
//	return string.Join(separator, parts);
//
// The separator here carries the trailing space the split above does not.
func convertFromValues[T any](culture *framework.CultureInfo, values []T, format func(T) string) string {
	// The join separator is the culture's list separator followed by ONE space,
	// built by the reference as `String.Concat(ListSeparator, " ")`. It is
	// composed HERE rather than on CultureInfo because it is this member's
	// behaviour: the split above uses the bare separator, and a CultureInfo
	// that offered only the spaced form would make that impossible to spell.
	separator := cultureListSeparator(culture) + " "
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = format(value)
	}
	return strings.Join(parts, separator)
}

// cultureListSeparator is the BARE separator the split uses.
func cultureListSeparator(culture *framework.CultureInfo) string {
	if culture == nil {
		return framework.CultureInfoCurrentCulture().ListSeparator()
	}
	return culture.ListSeparator()
}

// invalidFormatError builds the InvalidOperationException the parse failures
// raise, with the expected parameter names joined by the culture's separator.
func invalidFormatError(culture *framework.CultureInfo, expectedParams []string) error {
	return fmt.Errorf("%w: %s", errDesignInvalidFormat,
		fmt.Sprintf(invalidStringFormat, strings.Join(expectedParams, cultureListSeparator(culture))))
}
