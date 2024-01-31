package setting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var ErrNotFound = errors.New("not found")

type Settings interface {
	Save(ctx context.Context, key Key, value Value) error

	Setting(ctx context.Context, key Key) (Value, error)
	Settings(ctx context.Context, keys []Key) (map[Key]Value, error)

	Delete(ctx context.Context, key Key) error
}

func NewKey(context string, group string, name string) Key {
	return Key{context: context, group: group, setting: name}
}

type Key struct {
	context string
	group   string
	setting string
}

func (k Key) Key() string {
	parts := []string{"MISSING", "MISSING", "MISSING"}

	if k.context != "" {
		parts[0] = k.context
	}

	if k.group != "" {
		parts[1] = k.group
	}

	if k.setting != "" {
		parts[2] = k.setting
	}

	return strings.Join(parts, ".")
}

// NewValue returns a valid Value for val.
func NewValue(val any) Value { //nolint:gocyclo,cyclop,funlen
	if val == nil {
		return Value{v: "", kind: reflect.String}
	}

	r := reflect.TypeOf(val)

	switch r.Kind() {
	case reflect.String:
		return Value{v: val.(string), kind: reflect.String} //nolint:forcetypeassert
	case reflect.Bool:
		return Value{v: strconv.FormatBool(val.(bool)), kind: reflect.Bool} //nolint:forcetypeassert
	case reflect.Int:
		return Value{v: strconv.Itoa(val.(int)), kind: reflect.Int} //nolint:forcetypeassert
	case reflect.Int8:
		return Value{v: strconv.Itoa(int(val.(int8))), kind: reflect.Int8} //nolint:forcetypeassert
	case reflect.Int16:
		return Value{v: strconv.Itoa(int(val.(int16))), kind: reflect.Int16} //nolint:forcetypeassert
	case reflect.Int32:
		return Value{v: strconv.Itoa(int(val.(int32))), kind: reflect.Int32} //nolint:forcetypeassert
	case reflect.Int64:
		return Value{v: strconv.Itoa(int(val.(int64))), kind: reflect.Int64} //nolint:forcetypeassert
	case reflect.Uint:
		return Value{v: strconv.FormatUint(uint64(val.(uint)), base), kind: reflect.Uint} //nolint:forcetypeassert
	case reflect.Uint8:
		return Value{v: strconv.FormatUint(uint64(val.(uint8)), base), kind: reflect.Uint8} //nolint:forcetypeassert
	case reflect.Uint16:
		return Value{v: strconv.FormatUint(uint64(val.(uint16)), base), kind: reflect.Uint16} //nolint:forcetypeassert
	case reflect.Uint32:
		return Value{v: strconv.FormatUint(uint64(val.(uint32)), base), kind: reflect.Uint32} //nolint:forcetypeassert
	case reflect.Uint64:
		return Value{v: strconv.FormatUint(val.(uint64), base), kind: reflect.Uint64} //nolint:forcetypeassert
	case reflect.Float32:
		return Value{v: strconv.FormatFloat(float64(val.(float32)), 'g', -1, 64), kind: reflect.Float32} //nolint:forcetypeassert
	case reflect.Float64:
		return Value{v: strconv.FormatFloat(val.(float64), 'g', -1, 64), kind: reflect.Float64} //nolint:forcetypeassert
	case reflect.Map, reflect.Slice, reflect.Array, reflect.Struct:
		if t, ok := val.(time.Time); ok {
			return Value{v: t.Format(time.RFC3339Nano), kind: reflect.Struct}
		}

		b, err := json.Marshal(val)
		if err != nil {
			return Value{v: "", kind: reflect.Interface}
		}

		return Value{v: string(b), kind: reflect.Interface}
	default:
		return Value{v: "", kind: reflect.String}
	}
}

// base is the base used to format and parse uint values from and to strings.
const base = 10

type Value struct {
	v    string
	kind reflect.Kind
}

func (v Value) MustString() string {
	i, err := strconv.ParseUint(v.v, base, 64)
	if err == nil { // match as uint or int
		return strconv.FormatUint(i, base)
	}

	i64, err := strconv.ParseInt(v.v, base, 64)
	if err == nil { // match as int
		return strconv.FormatInt(i64, base)
	}

	f, err := strconv.ParseFloat(v.v, 64)
	if err == nil { // match floats
		return fmt.Sprintf("%.2f", f)
	}

	return v.v
}

func (v Value) MustByte() []byte {
	return []byte(v.MustString())
}

func (v Value) Bool() (bool, error) {
	if v.v == "" {
		return false, nil
	}

	return strconv.ParseBool(v.v)
}

func (v Value) MustBool() bool {
	b, err := v.Bool()
	if err != nil {
		panic(err)
	}

	return b
}

func (v Value) Int() (int, error) {
	if b, err := v.Bool(); err == nil {
		if b {
			return 1, nil
		}

		return 0, nil
	}

	i, err := strconv.Atoi(v.v)
	if err != nil {
		return 0, err
	}

	return i, nil
}

func (v Value) MustInt() int {
	i, err := v.Int()
	if err != nil {
		panic(err)
	}

	return i
}

func (v Value) MustInt8() int8 {
	if b, err := v.Bool(); err == nil {
		if b {
			return 1
		}

		return 0
	}

	i, err := strconv.Atoi(v.v)
	if err != nil {
		panic(err)
	}

	if i > math.MaxInt8 || i < math.MinInt8 { // todo: do the same checks for all other methods like uint32 ...
		panic("uint overflow")
	}

	return int8(i)
}

func (v Value) MustInt16() int16 {
	if b, err := v.Bool(); err == nil {
		if b {
			return 1
		}

		return 0
	}

	i, err := strconv.Atoi(v.v)
	if err != nil {
		panic(err)
	}

	return int16(i) //nolint:gosec,lll // accept potential integer overflow, as it is expected, that the developer knows what he is doing. //todo check lint warnings, if should panic
}

func (v Value) MustInt32() int32 {
	if b, err := v.Bool(); err == nil {
		if b {
			return 1
		}

		return 0
	}

	i, err := strconv.Atoi(v.v)
	if err != nil {
		panic(err)
	}

	return int32(i) //nolint:gosec,lll // integer overflow: developer is responsible // TODO should smaller types just be removed to have settings more secure by default? should panic?
}

func (v Value) MustInt64() int64 {
	if b, err := v.Bool(); err == nil {
		if b {
			return 1
		}

		return 0
	}

	i, err := strconv.Atoi(v.v)
	if err != nil {
		panic(err)
	}

	return int64(i)
}

func (v Value) MustUint() uint {
	if b, err := v.Bool(); err == nil {
		if b {
			return 1
		}

		return 0
	}

	i, err := strconv.ParseUint(v.v, base, 64)
	if err != nil {
		panic(err)
	}

	return uint(i)
}

func (v Value) MustUint8() uint8 {
	if b, err := v.Bool(); err == nil {
		if b {
			return 1
		}

		return 0
	}

	i, err := strconv.ParseUint(v.v, base, 64)
	if err != nil {
		panic(err)
	}

	return uint8(i)
}

func (v Value) MustUint16() uint16 {
	if b, err := v.Bool(); err == nil {
		if b {
			return 1
		}

		return 0
	}

	i, err := strconv.ParseUint(v.v, base, 64)
	if err != nil {
		panic(err)
	}

	return uint16(i)
}

func (v Value) MustUint32() uint32 {
	if b, err := v.Bool(); err == nil {
		if b {
			return 1
		}

		return 0
	}

	i, err := strconv.ParseUint(v.v, base, 64)
	if err != nil {
		panic(err)
	}

	return uint32(i)
}

func (v Value) MustUint64() uint64 {
	if b, err := v.Bool(); err == nil {
		if b {
			return 1
		}

		return 0
	}

	i, err := strconv.ParseUint(v.v, base, 64)
	if err != nil {
		panic(err)
	}

	return i
}

func (v Value) MustFloat32() float32 {
	if b, err := v.Bool(); err == nil {
		if b {
			return 1
		}

		return 0
	}

	i, err := strconv.ParseFloat(v.v, 64)
	if err != nil {
		panic(err)
	}

	return float32(i)
}

func (v Value) Float64() (float64, error) {
	if b, err := v.Bool(); err == nil {
		if b {
			return 1, nil
		}

		return 0, nil
	}

	i, err := strconv.ParseFloat(v.v, 64)
	if err != nil {
		return 0, err
	}

	return i, nil
}

func (v Value) MustFloat64() float64 {
	i, err := v.Float64()
	if err != nil {
		panic(err)
	}

	return i
}

func (v Value) MustUnmarshal(o any) {
	oKind := reflect.TypeOf(o).Elem().Kind()
	oVal := reflect.Indirect(reflect.ValueOf(o))

	//var applyValue reflect.Value // go and build rules and apply only ones in the end

	if v.kind == reflect.String {
		if v.v == "" && oKind != reflect.Bool {
			oVal.Set(reflect.ValueOf(""))

			return
		}

		if v.v != "" {
			if oKind == reflect.String {
				oVal.Set(reflect.ValueOf(v.v))

				return
			}

			err := json.Unmarshal([]byte(v.v), o)
			if err != nil {
				panic(err)
			}

			return
		}
	}

	if b, err := v.Bool(); err == nil && (v.kind == reflect.Bool || v.kind == reflect.String) {
		switch oKind {
		case reflect.String:
			if b {
				oVal.Set(reflect.ValueOf("true"))

				return
			}

			oVal.Set(reflect.ValueOf("false"))

			return
		case reflect.Bool:
			if b {
				oVal.Set(reflect.ValueOf(true))

				return
			}

			oVal.Set(reflect.ValueOf(false))

			return
		default:
			panic("unhandled default case")
		}
	}

	if oKind == reflect.String {
		tNow, err := v.Time()
		if err == nil {
			oVal.Set(reflect.ValueOf(tNow.Format(time.RFC3339Nano)))

			return
		}

		iVal, err := v.Int()
		if err == nil {
			oVal.Set(reflect.ValueOf(strconv.Itoa(iVal)))

			return
		}

		fVal, err := v.Float64()
		if err == nil {
			oVal.Set(reflect.ValueOf(fmt.Sprintf("%.2f", fVal)))

			return
		}

		oVal.Set(reflect.ValueOf(v.v))

		return
	}

	if oKind == reflect.Struct {
		tNow, err := v.Time()
		if err == nil {
			oVal.Set(reflect.ValueOf(tNow))

			return
		}
	}

	err := json.Unmarshal([]byte(v.v), o)
	if err != nil {
		panic(err)
	}
}

func (v Value) Time() (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, v.v)
	if err != nil {
		return time.Time{}, err
	}

	return t, nil
}

func (v Value) MustTime() time.Time {
	t, err := v.Time()
	if err != nil {
		panic(err)
	}

	return t
}

// v Value Any() any
// All functions with E at the end returning an error
