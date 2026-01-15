package aassert

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// NumFields asserts that the given object struct
// does have expected number of public fields.
// Public fields of nested and embedded struct are counted as well.
func NumFields(t *testing.T, expected int, object any, msgAndArgs ...any) bool {
	t.Helper()

	if object == nil {
		return assert.Fail(t, "invalid argument, it has to be a struct")
	}

	elemType := reflect.TypeOf(object)
	elem := reflect.ValueOf(object)

	if elemType.Kind() != reflect.Struct && elemType.Kind() != reflect.Ptr {
		return assert.Fail(t, "invalid argument, it has to be a struct")
	}

	if elemType.Kind() == reflect.Ptr {
		if elemType.Elem().Kind() != reflect.Struct {
			return assert.Fail(t, "invalid argument, it has to be a struct")
		}

		elem = elem.Elem()
	}

	var fields int

	fields += getNumFields(elem)

	if fields != expected {
		t.Log(elem.Type())
		t.Log("!!! !!! !!! !!! !!! !!! !!! !!! !!!")
		t.Log("INFO: The number of public fields of the struct: `" + elem.Type().String() + "` changed.")
		t.Log("INFO: This can potentially be an issue:")
		t.Log("      - if you map the struct from one layer to another one")
		t.Log("      - ensure that all methods mapping this struct are correct")
		t.Log("      - ensure all test data has the right fields set")
		t.Log("=> Inspect the functions and factories.")
		t.Log("=> Manually correct the calling test case: `" + t.Name() + "` to the right expected count.")
		t.Log("!!! !!! !!! !!! !!! !!! !!! !!! !!!")

		return assert.Fail(t, fmt.Sprintf("struct changed, it has: %d fields, expected: %d", fields, expected), msgAndArgs...)
	}

	return true
}

func getNumFields(elem reflect.Value) int {
	if elem.Kind() == reflect.Uint ||
		elem.Kind() == reflect.Uint8 ||
		elem.Kind() == reflect.Uint16 ||
		elem.Kind() == reflect.Uint32 ||
		elem.Kind() == reflect.Uint64 {
		return 0
	}

	if elem.Kind() == reflect.Ptr {
		if elem.Elem().Kind() != reflect.Struct {
			return 0
		}

		elem = elem.Elem()
	}

	var fields int

	for i := range elem.NumField() {
		field := elem.Field(i)

		if elem.Type().Field(i).IsExported() {
			fields++

			switch field.Kind() {
			case reflect.Struct:
				fields += getNumFields(field)
			case reflect.Ptr:
				fields += getNumFields(reflect.New(field.Type().Elem()))
			case reflect.Slice:
				slice := reflect.MakeSlice(field.Type(), 1, 1)
				v := slice.Index(0)

				fields += getNumFields(v)
			case reflect.Map:
				m := reflect.MakeMapWithSize(field.Type(), 1)
				v := reflect.Value{}

				if field.Type().Kind() == reflect.Map && field.Type().Elem().Kind() == reflect.String {
					m.SetMapIndex(reflect.ValueOf(""), reflect.New(field.Type().Elem()).Elem())
					v = m.MapIndex(reflect.ValueOf(""))

					continue
				} else {
					m.SetMapIndex(reflect.ValueOf(0), reflect.New(field.Type().Elem()).Elem())
					v = m.MapIndex(reflect.ValueOf(0))
				}

				fields += getNumFields(v)
			default:
			}
		}
	}

	return fields
}
