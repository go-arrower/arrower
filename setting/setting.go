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

func (v Value) String() (string, error) {
	i, err := strconv.ParseUint(v.v, base, 64)
	if err == nil { // match as uint or int
		return strconv.FormatUint(i, base), nil
	}

	i64, err := strconv.ParseInt(v.v, base, 64)
	if err == nil { // match as int
		return strconv.FormatInt(i64, base), nil
	}

	f, err := strconv.ParseFloat(v.v, 64)
	if err == nil { // match floats
		return fmt.Sprintf("%.2f", f), nil
	}

	return v.v, nil
}

func (v Value) MustString() string {
	s, err := v.String()
	if err != nil {
		panic(err)
	}

	return s
}

func (v Value) Byte() ([]byte, error) {
	return []byte(v.MustString()), nil
}

func (v Value) MustByte() []byte {
	b, err := v.Byte()
	if err != nil {
		panic(err)
	}

	return b
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

func (v Value) Int8() (int8, error) {
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

	if i > math.MaxInt8 || i < math.MinInt8 { // todo: do the same checks for all other methods like uint32 ...
		return 0, fmt.Errorf("uint overflow")
	}

	return int8(i), nil
}

func (v Value) MustInt8() int8 {
	i, err := v.Int8()
	if err != nil {
		panic(err)
	}

	return i
}

func (v Value) Int16() (int16, error) {
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

	return int16(i), nil //nolint:gosec,lll // accept potential integer overflow, as it is expected, that the developer knows what he is doing. //todo check lint warnings, if should panic
}

func (v Value) MustInt16() int16 {
	i, err := v.Int16()
	if err != nil {
		panic(err)
	}

	return i
}

func (v Value) Int32() (int32, error) {
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

	return int32(i), nil //nolint:gosec,lll // integer overflow: developer is responsible // TODO should smaller types just be removed to have settings more secure by default? should panic?
}

func (v Value) MustInt32() int32 {
	i, err := v.Int32()
	if err != nil {
		panic(err)
	}

	return i
}

func (v Value) Int64() (int64, error) {
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

	return int64(i), nil
}

func (v Value) MustInt64() int64 {
	i, err := v.Int64()
	if err != nil {
		panic(err)
	}

	return i
}

func (v Value) Uint() (uint, error) {
	if b, err := v.Bool(); err == nil {
		if b {
			return 1, nil
		}

		return 0, nil
	}

	i, err := strconv.ParseUint(v.v, base, 64)
	if err != nil {
		return 0, err
	}

	return uint(i), nil
}

func (v Value) MustUint() uint {
	i, err := v.Uint()
	if err != nil {
		panic(err)
	}

	return i
}

func (v Value) Uint8() (uint8, error) {
	if b, err := v.Bool(); err == nil {
		if b {
			return 1, nil
		}

		return 0, nil
	}

	i, err := strconv.ParseUint(v.v, base, 64)
	if err != nil {
		return 0, err
	}

	return uint8(i), nil
}

func (v Value) MustUint8() uint8 {
	i, err := v.Uint8()
	if err != nil {
		panic(err)
	}

	return i
}

func (v Value) Uint16() (uint16, error) {
	if b, err := v.Bool(); err == nil {
		if b {
			return 1, nil
		}

		return 0, nil
	}

	i, err := strconv.ParseUint(v.v, base, 64)
	if err != nil {
		return 0, err
	}

	return uint16(i), nil
}

func (v Value) MustUint16() uint16 {
	i, err := v.Uint16()
	if err != nil {
		panic(err)
	}

	return i
}

func (v Value) Uint32() (uint32, error) {
	if b, err := v.Bool(); err == nil {
		if b {
			return 1, nil
		}

		return 0, nil
	}

	i, err := strconv.ParseUint(v.v, base, 64)
	if err != nil {
		return 0, err
	}

	return uint32(i), nil
}

func (v Value) MustUint32() uint32 {
	i, err := v.Uint32()
	if err != nil {
		panic(err)
	}

	return i
}

func (v Value) Uint64() (uint64, error) {
	if b, err := v.Bool(); err == nil {
		if b {
			return 1, nil
		}

		return 0, nil
	}

	i, err := strconv.ParseUint(v.v, base, 64)
	if err != nil {
		return 0, err
	}

	return i, nil
}

func (v Value) MustUint64() uint64 {
	i, err := v.Uint64()
	if err != nil {
		panic(err)
	}

	return i
}

func (v Value) Float32() (float32, error) {
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

	return float32(i), nil
}

func (v Value) MustFloat32() float32 {
	f, err := v.Float32()
	if err != nil {
		panic(err)
	}

	return f
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
	f, err := v.Float64()
	if err != nil {
		panic(err)
	}

	return f
}

func (v Value) Unmarshal(o any) error {
	oKind := reflect.TypeOf(o).Elem().Kind()
	oVal := reflect.Indirect(reflect.ValueOf(o))

	//var applyValue reflect.Value // go and build rules and apply only ones in the end

	if v.kind == reflect.String {
		if v.v == "" && oKind != reflect.Bool {
			oVal.Set(reflect.ValueOf(""))

			return nil
		}

		if v.v != "" {
			if oKind == reflect.String {
				oVal.Set(reflect.ValueOf(v.v))

				return nil
			}

			err := json.Unmarshal([]byte(v.v), o)
			if err != nil {
				return err
			}

			return nil
		}
	}

	if b, err := v.Bool(); err == nil && (v.kind == reflect.Bool || v.kind == reflect.String) {
		switch oKind {
		case reflect.String:
			if b {
				oVal.Set(reflect.ValueOf("true"))

				return nil
			}

			oVal.Set(reflect.ValueOf("false"))

			return nil
		case reflect.Bool:
			if b {
				oVal.Set(reflect.ValueOf(true))

				return nil
			}

			oVal.Set(reflect.ValueOf(false))

			return nil
		default:
			return fmt.Errorf("unhandled default case")
		}
	}

	if oKind == reflect.String {
		tNow, err := v.Time()
		if err == nil {
			oVal.Set(reflect.ValueOf(tNow.Format(time.RFC3339Nano)))

			return nil
		}

		iVal, err := v.Int()
		if err == nil {
			oVal.Set(reflect.ValueOf(strconv.Itoa(iVal)))

			return nil
		}

		fVal, err := v.Float64()
		if err == nil {
			oVal.Set(reflect.ValueOf(fmt.Sprintf("%.2f", fVal)))

			return nil
		}

		oVal.Set(reflect.ValueOf(v.v))

		return nil
	}

	if oKind == reflect.Struct {
		tNow, err := v.Time()
		if err == nil {
			oVal.Set(reflect.ValueOf(tNow))

			return nil
		}
	}

	err := json.Unmarshal([]byte(v.v), o)
	if err != nil {
		return err
	}

	return nil
}

func (v Value) MustUnmarshal(o any) {
	err := v.Unmarshal(o)
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
