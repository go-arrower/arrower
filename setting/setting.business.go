package setting

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Settings interface {
	Save(ctx context.Context, key Key, setting Value) error
	Setting(ctx context.Context, key Key) (Value, error)

	OnSettingChange(key Key, callback func(setting Value))
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

func New(val any) Value {
	return NewValue(val)
}

// NewValue returns a valid Value for val.
func NewValue(val any) Value { //nolint:gocyclo,cyclop,funlen
	if val == nil {
		return Value{v: ""}
	}

	r := reflect.TypeOf(val)

	switch r.Kind() { //nolint:exhaustive
	case reflect.String:
		return Value{v: val.(string)} //nolint:forcetypeassert
	case reflect.Bool:
		return Value{v: strconv.FormatBool(val.(bool))} //nolint:forcetypeassert
	case reflect.Int:
		return Value{v: strconv.Itoa(val.(int))} //nolint:forcetypeassert
	case reflect.Int8:
		return Value{v: strconv.Itoa(int(val.(int8)))} //nolint:forcetypeassert
	case reflect.Int16:
		return Value{v: strconv.Itoa(int(val.(int16)))} //nolint:forcetypeassert
	case reflect.Int32:
		return Value{v: strconv.Itoa(int(val.(int32)))} //nolint:forcetypeassert
	case reflect.Int64:
		return Value{v: strconv.Itoa(int(val.(int64)))} //nolint:forcetypeassert
	case reflect.Uint:
		return Value{v: strconv.FormatUint(uint64(val.(uint)), base)} //nolint:forcetypeassert
	case reflect.Uint8:
		return Value{v: strconv.FormatUint(uint64(val.(uint8)), base)} //nolint:forcetypeassert
	case reflect.Uint16:
		return Value{v: strconv.FormatUint(uint64(val.(uint16)), base)} //nolint:forcetypeassert
	case reflect.Uint32:
		return Value{v: strconv.FormatUint(uint64(val.(uint32)), base)} //nolint:forcetypeassert
	case reflect.Uint64:
		return Value{v: strconv.FormatUint(val.(uint64), base)} //nolint:forcetypeassert
	case reflect.Float32:
		return Value{v: strconv.FormatFloat(float64(val.(float32)), 'g', -1, 64)} //nolint:forcetypeassert
	case reflect.Float64:
		return Value{v: strconv.FormatFloat(val.(float64), 'g', -1, 64)} //nolint:forcetypeassert
	case reflect.Map, reflect.Slice, reflect.Array, reflect.Struct:
		if t, ok := val.(time.Time); ok {
			return Value{v: t.Format(time.RFC3339Nano)}
		}

		b, err := json.Marshal(val)
		if err != nil {
			return Value{v: ""}
		}

		return Value{v: string(b)}
	default:
		return Value{v: ""}
	}
}

// base is the base used to format and parse uint values from and to strings.
const base = 10

type Value struct {
	v string
}

func (v Value) String() string {
	i, err := strconv.ParseUint(v.v, base, 64)
	if err == nil { // match as uint or int
		return strconv.FormatUint(i, base)
	}

	f, err := strconv.ParseFloat(v.v, 64)
	if err == nil { // match floats
		return fmt.Sprintf("%.2f", f)
	}

	return v.v
}

func (v Value) Byte() []byte {
	return []byte(v.String())
}

func (v Value) Bool() bool {
	b, _ := strconv.ParseBool(v.v)

	return b
}

func (v Value) Int() int {
	i, _ := strconv.Atoi(v.v)

	return i
}

func (v Value) Int8() int8 {
	i, _ := strconv.Atoi(v.v)

	return int8(i)
}

func (v Value) Int16() int16 {
	i, _ := strconv.Atoi(v.v)

	return int16(i) //nolint:gosec,lll // accept potential integer overflow, as it is expected, that the develoepr knows what he is doing.
}

func (v Value) Int32() int32 {
	i, _ := strconv.Atoi(v.v)

	return int32(i) //nolint:gosec,lll // integer overflow: developer is responsible // TODO should smaller types just be removed to have settings more secure by default?
}

func (v Value) Int64() int64 {
	i, _ := strconv.Atoi(v.v)

	return int64(i)
}

func (v Value) Uint() uint {
	i, _ := strconv.ParseUint(v.v, base, 64)

	return uint(i)
}

func (v Value) Uint8() uint8 {
	i, _ := strconv.ParseUint(v.v, base, 64)

	return uint8(i)
}

func (v Value) Uint16() uint16 {
	i, _ := strconv.ParseUint(v.v, base, 64)

	return uint16(i)
}

func (v Value) Uint32() uint32 {
	i, _ := strconv.ParseUint(v.v, base, 64)

	return uint32(i)
}

func (v Value) Uint64() uint64 {
	i, _ := strconv.ParseUint(v.v, base, 64)

	return i
}

func (v Value) Float32() float32 {
	i, _ := strconv.ParseFloat(v.v, 64)

	return float32(i)
}

func (v Value) Float64() float64 {
	i, _ := strconv.ParseFloat(v.v, 64)

	return i
}

func (v Value) Unmarshal(o any) {
	_ = json.Unmarshal([]byte(v.v), o)
}

func (v Value) Time() time.Time {
	t, _ := time.Parse(time.RFC3339Nano, v.v)

	return t
}

type repository interface {
	Save(context.Context, Key, Value) error
	FindByID(context.Context, Key) (Value, error)
}
