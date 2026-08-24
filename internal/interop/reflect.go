package interop

import "reflect"

func reflectPointer(value any) uintptr {
	if value == nil {
		return 0
	}
	reflected := reflect.ValueOf(value)
	if reflected.Kind() != reflect.Pointer || reflected.IsNil() {
		panic("interop owner association requires a non-nil pointer")
	}
	return reflected.Pointer()
}
