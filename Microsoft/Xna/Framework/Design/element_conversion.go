package design

import (
	"fmt"
	"strconv"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// The three element conversions the twelve leaves instantiate the generic
// helpers at, measured from which `ConvertToValues<T>` each leaf calls:
//
//	float32   Vector2, Vector3, Vector4, Quaternion
//	int32     Point
//	uint8     Color
//
// The other six leaves never call the helpers at all: three declare no
// ConvertFrom, and three declare one whose whole body forwards to the base.
//
// # Reference authority
//
//	mscorlib.dll  4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0
//	sha256 5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63
//
// The reference reaches each of these through
// `TypeDescriptor.GetConverter(typeof(T))` -- SingleConverter, Int32Converter
// and ByteConverter -- whose ConvertFromString and ConvertToString are the
// type's own Parse and ToString under the culture's number format.
//
// # The number format is NOT projected, and that is recorded
//
// This projection reaches CultureInfo for ONE thing, the list separator,
// because that is all the Design namespace's own IL touches. The number format
// is reached only indirectly, through these element converters, and the profile
// names NumberFormatInfo in no signature.
//
// So the parses and formats below are the INVARIANT ones. On a culture whose
// decimal separator is a comma this diverges from the reference. The divergence
// is stated rather than hidden: reproducing it would mean projecting a number
// format the profile never names, and inventing one would be worse than saying
// which one this is.

// parseSingle is SingleConverter::ConvertFromString for one element.
func parseSingle(text string) (float32, error) {
	value, err := strconv.ParseFloat(text, 32)
	if err != nil {
		return 0, err
	}
	return float32(value), nil
}

// formatSingle is SingleConverter::ConvertToString for one element.
//
// The 'g' verb at -1 precision prints the shortest decimal that parses back to
// the same float32, which is the round-trip Single.ToString() promises.
func formatSingle(value float32) string {
	return strconv.FormatFloat(float64(value), 'g', -1, 32)
}

// parseInt32 is Int32Converter::ConvertFromString.
func parseInt32(text string) (int32, error) {
	value, err := strconv.ParseInt(text, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(value), nil
}

// formatInt32 is Int32Converter::ConvertToString.
func formatInt32(value int32) string { return strconv.FormatInt(int64(value), 10) }

// parseByte is ByteConverter::ConvertFromString. A value outside 0..255 is a
// failure rather than a truncation, which is what Byte.Parse does.
func parseByte(text string) (uint8, error) {
	value, err := strconv.ParseUint(text, 10, 8)
	if err != nil {
		return 0, err
	}
	return uint8(value), nil
}

// formatByte is ByteConverter::ConvertToString.
func formatByte(value uint8) string { return strconv.FormatUint(uint64(value), 10) }

// notSupportedError is what the base TypeConverter raises for a conversion it
// does not know, which is what every fall-through in this package reaches.
//
// The base's ConvertFrom has ONE case it does handle -- an InstanceDescriptor,
// which it Invokes -- and that case is unreachable here: Go cannot invoke a
// described constructor, and the projected InstanceDescriptor carries the type
// rather than a ConstructorInfo for exactly that reason. A consumer holding one
// rebuilds the value itself.
func notSupportedError(what any) error {
	return fmt.Errorf("%w: %v", errDesignNotSupported, what)
}

// argumentNullError is the ArgumentNullException every ConvertTo raises for a
// nil destinationType and every CreateInstance for a nil dictionary. The
// reference names the PARAMETER, and the two names differ, so the caller
// supplies it.
func argumentNullError(parameter string) error {
	return fmt.Errorf("%w: %s", errDesignArgumentNull, parameter)
}

// The three component readers CreateInstance uses. Each reads the dictionary by
// NAME, in the constructor's order, exactly as the reference's
//
//	new Vector2((float)propertyValues["X"], (float)propertyValues["Y"])
//
// does. A name the dictionary does not hold is a REFUSAL here where the
// reference's unbox of a null raises NullReferenceException -- answering a
// documented refusal instead is the projection's own decision and is recorded
// rather than passed off as the reference's.

func singleComponents(propertyValues map[string]any, names []string) ([]float32, error) {
	return components(propertyValues, names, func(entry any) (float32, bool) {
		value, ok := entry.(float32)
		return value, ok
	})
}

func int32Components(propertyValues map[string]any, names []string) ([]int32, error) {
	return components(propertyValues, names, func(entry any) (int32, bool) {
		value, ok := entry.(int32)
		return value, ok
	})
}

func byteComponents(propertyValues map[string]any, names []string) ([]uint8, error) {
	return components(propertyValues, names, func(entry any) (uint8, bool) {
		value, ok := entry.(uint8)
		return value, ok
	})
}

func vector3Components(propertyValues map[string]any, names []string) ([]framework.Vector3, error) {
	return components(propertyValues, names, func(entry any) (framework.Vector3, bool) {
		value, ok := entry.(framework.Vector3)
		return value, ok
	})
}

func components[T any](propertyValues map[string]any, names []string, cast func(any) (T, bool)) ([]T, error) {
	if propertyValues == nil {
		return nil, argumentNullError("propertyValues")
	}
	values := make([]T, len(names))
	for index, name := range names {
		entry, present := propertyValues[name]
		if !present {
			return nil, fmt.Errorf("%w: propertyValues is missing %q", errDesignArgumentNull, name)
		}
		value, ok := cast(entry)
		if !ok {
			var want T
			return nil, fmt.Errorf("%w: propertyValues[%q] is %T, want %T", errDesignArgumentNull, name, entry, want)
		}
		values[index] = value
	}
	return values, nil
}
