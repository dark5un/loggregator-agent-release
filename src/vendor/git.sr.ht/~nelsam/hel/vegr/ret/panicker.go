package ret

import "reflect"

// PanicFieldName is the name used as a return field for causing a method to
// panic.
const PanicFieldName = "Panic_"

// PanicFieldIdx returns the index of the panic field on v. If the field is not
// found directly on the struct (even if it is on an embedded struct), it
// returns -1
func PanicFieldIdx(t reflect.Type) int {
	f, ok := t.FieldByName(PanicFieldName)
	if !ok {
		return -1
	}
	if len(f.Index) != 1 {
		return -1
	}
	return f.Index[0]
}
