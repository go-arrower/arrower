package setting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidValue = errors.New("invalid value")
)

type Settings interface {
	Save(ctx context.Context, key Key, value Value) error

	Setting(ctx context.Context, key Key) (Value, error)
	// Settings returns a setting for each of the keys.
	// If not all keys exist, the found settings are returned anyway with an ErrNotFound error.
	Settings(ctx context.Context, keys []Key) (map[Key]Value, error)

	Delete(ctx context.Context, key Key) error
}

func NewKey(context string, group string, setting string) Key {
	return Key{context: context, group: group, setting: setting}
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
func NewValue(val any) Value { //nolint:gocyclo,cyclop,funlen,gocognit
	if val == nil {
		return Value{v: "", kind: reflect.String}
	}

	r := reflect.TypeOf(val)

	switch r.Kind() {
	case reflect.String:
		if t, err := time.Parse(time.RFC3339Nano, val.(string)); err == nil {
			return Value{
				v:    fmt.Sprintf("%s%s%s", t.Format(time.RFC3339Nano), timeLocationSeparator, t.Location().String()),
				kind: reflect.Struct,
			}
		}

		return Value{v: val.(string), kind: reflect.String} //nolint:forcetypeassert
	case reflect.Bool:
		return Value{v: strconv.FormatBool(val.(bool)), kind: reflect.Bool} //nolint:forcetypeassert
	case reflect.Int:
		valueOf := reflect.ValueOf(val)
		targetType := reflect.TypeOf(int(0))

		if valueOf.CanConvert(targetType) {
			val = valueOf.Convert(targetType).Interface()
		}

		return Value{v: strconv.Itoa(val.(int)), kind: reflect.Int} //nolint:forcetypeassert
	case reflect.Int8:
		valueOf := reflect.ValueOf(val)
		targetType := reflect.TypeOf(int8(0))

		if valueOf.CanConvert(targetType) {
			val = valueOf.Convert(targetType).Interface()
		}

		return Value{v: strconv.Itoa(int(val.(int8))), kind: reflect.Int8} //nolint:forcetypeassert
	case reflect.Int16:
		valueOf := reflect.ValueOf(val)
		targetType := reflect.TypeOf(int16(0))

		if valueOf.CanConvert(targetType) {
			val = valueOf.Convert(targetType).Interface()
		}

		return Value{v: strconv.Itoa(int(val.(int16))), kind: reflect.Int16} //nolint:forcetypeassert
	case reflect.Int32:
		valueOf := reflect.ValueOf(val)
		targetType := reflect.TypeOf(int32(0))

		if valueOf.CanConvert(targetType) {
			val = valueOf.Convert(targetType).Interface()
		}

		return Value{v: strconv.Itoa(int(val.(int32))), kind: reflect.Int32} //nolint:forcetypeassert
	case reflect.Int64:
		valueOf := reflect.ValueOf(val)
		targetType := reflect.TypeOf(int64(0))

		if valueOf.CanConvert(targetType) {
			val = valueOf.Convert(targetType).Interface()
		}

		return Value{v: strconv.Itoa(int(val.(int64))), kind: reflect.Int64} //nolint:forcetypeassert
	case reflect.Uint:
		valueOf := reflect.ValueOf(val)
		targetType := reflect.TypeOf(uint(0))

		if valueOf.CanConvert(targetType) {
			val = valueOf.Convert(targetType).Interface()
		}

		return Value{v: strconv.FormatUint(uint64(val.(uint)), base), kind: reflect.Uint} //nolint:forcetypeassert
	case reflect.Uint8:
		valueOf := reflect.ValueOf(val)
		targetType := reflect.TypeOf(uint8(0))

		if valueOf.CanConvert(targetType) {
			val = valueOf.Convert(targetType).Interface()
		}

		return Value{v: strconv.FormatUint(uint64(val.(uint8)), base), kind: reflect.Uint8} //nolint:forcetypeassert
	case reflect.Uint16:
		valueOf := reflect.ValueOf(val)
		targetType := reflect.TypeOf(uint16(0))

		if valueOf.CanConvert(targetType) {
			val = valueOf.Convert(targetType).Interface()
		}

		return Value{v: strconv.FormatUint(uint64(val.(uint16)), base), kind: reflect.Uint16} //nolint:forcetypeassert
	case reflect.Uint32:
		valueOf := reflect.ValueOf(val)
		targetType := reflect.TypeOf(uint32(0))

		if valueOf.CanConvert(targetType) {
			val = valueOf.Convert(targetType).Interface()
		}

		return Value{v: strconv.FormatUint(uint64(val.(uint32)), base), kind: reflect.Uint32} //nolint:forcetypeassert
	case reflect.Uint64:
		valueOf := reflect.ValueOf(val)
		targetType := reflect.TypeOf(uint64(0))

		if valueOf.CanConvert(targetType) {
			val = valueOf.Convert(targetType).Interface()
		}

		return Value{v: strconv.FormatUint(val.(uint64), base), kind: reflect.Uint64} //nolint:forcetypeassert
	case reflect.Float32:
		valueOf := reflect.ValueOf(val)
		targetType := reflect.TypeOf(float32(0))

		if valueOf.CanConvert(targetType) {
			val = valueOf.Convert(targetType).Interface()
		}

		return Value{
			v:    strconv.FormatFloat(float64(val.(float32)), 'g', -1, 32), //nolint:forcetypeassert
			kind: reflect.Float32,
		}
	case reflect.Float64:
		valueOf := reflect.ValueOf(val)
		targetType := reflect.TypeOf(float64(0))

		if valueOf.CanConvert(targetType) {
			val = valueOf.Convert(targetType).Interface()
		}

		return Value{
			v:    strconv.FormatFloat(val.(float64), 'g', -1, 64), //nolint:forcetypeassert
			kind: reflect.Float64,
		}
	case reflect.Map, reflect.Slice, reflect.Array, reflect.Struct:
		if t, ok := val.(time.Time); ok {
			return Value{
				v:    fmt.Sprintf("%s%s%s", t.Format(time.RFC3339Nano), timeLocationSeparator, t.Location().String()),
				kind: reflect.Struct,
			}
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

const (
	// base is the base used to format and parse uint values from and to strings.
	base = 10

	// timeLocationSeparator is used to serialise time, and it's time zone on the fly.
	timeLocationSeparator = "|:|"
)

type Value struct {
	v    string
	kind reflect.Kind
}

// Raw returns the raw string value of Value as created by NewValue.
// The main purpose is to aid when implementing persistent implementations of Settings,
// as some types like time have a composite format.
// Each part is separated by the timeLocationSeparator.
func (v Value) Raw() string {
	return v.v
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

	if isSerialisedTime(v.v) {
		return strings.Split(v.v, timeLocationSeparator)[0], nil
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

	b, err := strconv.ParseBool(v.v)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrInvalidValue, err) //nolint:errorlint // prevent err in api
	}

	return b, nil
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
		return 0, fmt.Errorf("%w: %v", ErrInvalidValue, err) //nolint:errorlint // prevent err in api
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

	i, err := strconv.ParseInt(v.v, 10, 8)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidValue, err) //nolint:errorlint // prevent err in api
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

	i, err := strconv.ParseInt(v.v, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidValue, err) //nolint:errorlint // prevent err in api
	}

	return int16(i), nil
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

	i, err := strconv.ParseInt(v.v, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidValue, err) //nolint:errorlint // prevent err in api
	}

	return int32(i), nil
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
		return 0, fmt.Errorf("%w: %v", ErrInvalidValue, err) //nolint:errorlint // prevent err in api
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
		return 0, fmt.Errorf("%w: %v", ErrInvalidValue, err) //nolint:errorlint // prevent err in api
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

	i, err := strconv.ParseUint(v.v, base, 8)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidValue, err) //nolint:errorlint // prevent err in api
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

	i, err := strconv.ParseUint(v.v, base, 16)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidValue, err) //nolint:errorlint // prevent err in api
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

	i, err := strconv.ParseUint(v.v, base, 32)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidValue, err) //nolint:errorlint // prevent err in api
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
		return 0, fmt.Errorf("%w: %v", ErrInvalidValue, err) //nolint:errorlint // prevent err in api
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

	i, err := strconv.ParseFloat(v.v, 32)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidValue, err) //nolint:errorlint // prevent err in api
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
		return 0, fmt.Errorf("%w: %v", ErrInvalidValue, err) //nolint:errorlint // prevent err in api
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

//nolint:gocyclo,nestif,cyclop,varnamelen,funlen,gocognit,wsl,lll // the method has to consider all the cases for auto-casting.
func (v Value) Unmarshal(o any) error {
	var applyValue reflect.Value

	oKind := reflect.TypeOf(o).Elem().Kind()

	slog.Log(context.Background(), alogLevelDebug, "Unmarshal setting.Value into object",
		slog.String("value", v.v),
		slog.Any("object_kind", oKind),
	)

	switch oKind {
	case reflect.Bool:
		if isTrue, err := v.Bool(); err == nil {
			slog.Log(context.Background(), alogLevelDebug, "is bool", slog.Bool("value", isTrue))
			applyValue = reflect.ValueOf(isTrue)
		} else {
			return fmt.Errorf("%w", ErrInvalidValue)
		}
	case reflect.String:
		if v.v == "" {
			applyValue = reflect.ValueOf("")
		} else if isTrue, err := v.Bool(); err == nil {
			if isTrue {
				applyValue = reflect.ValueOf("true")
			} else {
				applyValue = reflect.ValueOf("false")
			}
		} else if iVal, err := v.Int(); err == nil {
			slog.Log(context.Background(), alogLevelDebug, "is int")
			applyValue = reflect.ValueOf(strconv.Itoa(iVal))
		} else if fVal, err := v.Float64(); err == nil {
			slog.Log(context.Background(), alogLevelDebug, "is float")
			applyValue = reflect.ValueOf(fmt.Sprintf("%.2f", fVal))
		} else if isSerialisedTime(v.v) {
			if tNow, err := v.Time(); err == nil {
				slog.Log(context.Background(), alogLevelDebug, "is time")
				applyValue = reflect.ValueOf(tNow.Format(time.RFC3339Nano))
			}
		} else {
			slog.Log(context.Background(), alogLevelDebug, "raw string")
			applyValue = reflect.ValueOf(v.v)
		}
	case reflect.Struct, reflect.Slice, reflect.Map:
		if isSerialisedTime(v.v) {
			if tNow, err := v.Time(); err == nil {
				slog.Log(context.Background(), alogLevelDebug, "is time")

				applyValue = reflect.ValueOf(tNow)
			}
		} else {
			err := json.Unmarshal([]byte(v.v), o)
			if err != nil {
				return fmt.Errorf("%w: %v", ErrInvalidValue, err) //nolint:errorlint // prevent err in api
			}
		}
	default:
		return fmt.Errorf("%w: %s", ErrInvalidValue, "unsupported type")
	}

	if applyValue != (reflect.Value{}) { //nolint:govet,lll // I don't know how to do it without == or reflect.DeepEqual: reflectvaluecompare: avoid using != with reflect.Value
		reflect.Indirect(reflect.ValueOf(o)).Set(applyValue)
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
	if !isSerialisedTime(v.v) {
		return time.Time{}, fmt.Errorf("%w: time serialisation wrong: %v", ErrInvalidValue, v.v)
	}

	tl := strings.Split(v.v, timeLocationSeparator)

	loc, err := time.LoadLocation(tl[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %v", ErrInvalidValue, err) //nolint:errorlint // prevent err in api
	}

	t, err := time.ParseInLocation(time.RFC3339Nano, tl[0], loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %v", ErrInvalidValue, err) //nolint:errorlint // prevent err in api
	}

	return t, nil
}

func isSerialisedTime(v string) bool {
	tl := strings.Split(v, timeLocationSeparator)

	return len(tl) == 2 //nolint:mnd // 2 is just the count of time serialisation elements.
}

func (v Value) MustTime() time.Time {
	t, err := v.Time()
	if err != nil {
		panic(err)
	}

	return t
}

const alogLevelDebug = -12 // redefine the alog level to prevent import cycles
