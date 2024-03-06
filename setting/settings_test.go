package setting_test

import (
	"bytes"
	"log/slog"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/setting"
)

func TestNewKey(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		context string
		group   string
		setting string
		expKey  string
	}{
		"empty": {
			"",
			"",
			"",
			"MISSING.MISSING.MISSING",
		},
		"context only": {
			"context",
			"",
			"",
			"context.MISSING.MISSING",
		},
		"group only": {
			"",
			"group",
			"",
			"MISSING.group.MISSING",
		},
		"setting only": {
			"",
			"",
			"setting",
			"MISSING.MISSING.setting",
		},
		"context and group": {
			"context",
			"group",
			"",
			"context.group.MISSING",
		},
		"context and setting": {
			"context",
			"",
			"setting",
			"context.MISSING.setting",
		},
		"group and setting": {
			"",
			"group",
			"setting",
			"MISSING.group.setting",
		},
		"complete key": {
			"context",
			"group",
			"setting",
			"context.group.setting",
		},
		"custom sub key": {
			"context",
			"group",
			"setting.custom_user_extension",
			"context.group.setting.custom_user_extension",
		},
	}

	for name, tt := range tests {
		tt := tt

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expKey, setting.NewKey(tt.context, tt.group, tt.setting).Key())
		})
	}
}

//nolint:dupl,maintidx // the test is cumbersome, but covers a lot of cases to be explicit in the behaviour of the type casts.
func TestNewValue(t *testing.T) {
	t.Parallel()

	// enable logger, if debugging
	// slog.SetDefault(alog.NewTest(os.Stderr))

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		value := setting.NewValue(nil)

		assert.Equal(t, "", value.MustString())
		assert.Equal(t, []byte(""), value.MustByte())

		assert.False(t, value.MustBool())

		assert.Equal(t, 0, value.MustInt())
		assert.Equal(t, int8(0), value.MustInt8())
		assert.Equal(t, int16(0), value.MustInt16())
		assert.Equal(t, int32(0), value.MustInt32())
		assert.Equal(t, int64(0), value.MustInt64())

		assert.Equal(t, uint(0), value.MustUint())
		assert.Equal(t, uint8(0), value.MustUint8())
		assert.Equal(t, uint16(0), value.MustUint16())
		assert.Equal(t, uint32(0), value.MustUint32())
		assert.Equal(t, uint64(0), value.MustUint64())

		assert.Equal(t, float32(0), value.MustFloat32()) //nolint:testifylint // fp: does not work for expected == 0
		assert.Equal(t, float64(0), value.MustFloat64()) //nolint:testifylint // fp: does not work for expected == 0

		assert.Panics(t, func() { value.MustTime() })

		var b bool
		value.MustUnmarshal(&b)
		assert.False(t, b)

		var s string
		value.MustUnmarshal(&s)
		assert.Empty(t, s)

		var buf bytes.Buffer
		assert.Panics(t, func() { value.MustUnmarshal(&buf) })
	})

	t.Run("string", func(t *testing.T) {
		t.Parallel()

		t.Run("empty", func(t *testing.T) {
			t.Parallel()

			value := setting.NewValue("")

			assert.Equal(t, "", value.MustString())
			assert.Equal(t, []byte(""), value.MustByte())

			assert.False(t, value.MustBool())

			assert.Equal(t, 0, value.MustInt())
			assert.Equal(t, int8(0), value.MustInt8())
			assert.Equal(t, int16(0), value.MustInt16())
			assert.Equal(t, int32(0), value.MustInt32())
			assert.Equal(t, int64(0), value.MustInt64())

			assert.Equal(t, uint(0), value.MustUint())
			assert.Equal(t, uint8(0), value.MustUint8())
			assert.Equal(t, uint16(0), value.MustUint16())
			assert.Equal(t, uint32(0), value.MustUint32())
			assert.Equal(t, uint64(0), value.MustUint64())

			assert.Equal(t, float32(0), value.MustFloat32()) //nolint:testifylint // fp: does not work for expected == 0
			assert.Equal(t, float64(0), value.MustFloat64()) //nolint:testifylint // fp: does not work for expected == 0

			assert.Panics(t, func() { value.MustTime() })

			var b bool
			value.MustUnmarshal(&b)
			assert.False(t, b)

			var s string
			value.MustUnmarshal(&s)
			assert.Empty(t, s)

			var buf bytes.Buffer
			assert.Panics(t, func() { value.MustUnmarshal(&buf) })
		})

		t.Run("valid", func(t *testing.T) {
			t.Parallel()

			strVal := "some string value"
			value := setting.NewValue(strVal)

			assert.Equal(t, strVal, value.MustString())
			assert.Equal(t, []byte(strVal), value.MustByte())

			assert.Panics(t, func() { value.MustBool() })

			assert.Panics(t, func() { value.MustInt() })
			assert.Panics(t, func() { value.MustInt8() })
			assert.Panics(t, func() { value.MustInt16() })
			assert.Panics(t, func() { value.MustInt32() })
			assert.Panics(t, func() { value.MustInt64() })

			assert.Panics(t, func() { value.MustUint() })
			assert.Panics(t, func() { value.MustUint8() })
			assert.Panics(t, func() { value.MustUint16() })
			assert.Panics(t, func() { value.MustUint32() })
			assert.Panics(t, func() { value.MustUint64() })

			assert.Panics(t, func() { value.MustFloat32() })
			assert.Panics(t, func() { value.MustFloat64() })

			assert.Panics(t, func() { value.MustTime() })

			var b bool
			assert.Panics(t, func() { value.MustUnmarshal(&b) })

			var s string
			value.MustUnmarshal(&s)
			assert.Equal(t, strVal, s)

			var buf bytes.Buffer
			assert.Panics(t, func() { value.MustUnmarshal(&buf) })
		})

		t.Run("time", func(t *testing.T) {
			t.Parallel()

			loc, err := time.LoadLocation("Asia/Tokyo")
			assert.NoError(t, err)
			now := time.Now().In(loc)
			strVal := now.Format(time.RFC3339Nano) // this step loses the time.Location information.
			value := setting.NewValue(strVal)

			t.Log(now)
			t.Log(strVal)
			t.Log(value)

			assert.Equal(t, strVal, value.MustString())
			assert.Equal(t, []byte(strVal), value.MustByte())

			assert.Panics(t, func() { value.MustBool() })

			assert.Panics(t, func() { value.MustInt() })
			assert.Panics(t, func() { value.MustInt8() })
			assert.Panics(t, func() { value.MustInt16() })
			assert.Panics(t, func() { value.MustInt32() })
			assert.Panics(t, func() { value.MustInt64() })

			assert.Panics(t, func() { value.MustUint() })
			assert.Panics(t, func() { value.MustUint8() })
			assert.Panics(t, func() { value.MustUint16() })
			assert.Panics(t, func() { value.MustUint32() })
			assert.Panics(t, func() { value.MustUint64() })

			assert.Panics(t, func() { value.MustFloat32() })
			assert.Panics(t, func() { value.MustFloat64() })

			assert.Equal(t, now.Truncate(time.Nanosecond).UTC(), value.MustTime().UTC())

			var b bool
			assert.Panics(t, func() { value.MustUnmarshal(&b) })

			var s string
			value.MustUnmarshal(&s)
			assert.Equal(t, strVal, s)

			var buf bytes.Buffer
			assert.Panics(t, func() { value.MustUnmarshal(&buf) })
		})

		t.Run("json", func(t *testing.T) {
			t.Parallel()

			strVal := `{"Key":"Val"}`
			value := setting.NewValue(strVal)

			assert.Equal(t, strVal, value.MustString())
			assert.Equal(t, []byte(strVal), value.MustByte())

			assert.Panics(t, func() { value.MustBool() })

			assert.Panics(t, func() { value.MustInt() })
			assert.Panics(t, func() { value.MustInt8() })
			assert.Panics(t, func() { value.MustInt16() })
			assert.Panics(t, func() { value.MustInt32() })
			assert.Panics(t, func() { value.MustInt64() })

			assert.Panics(t, func() { value.MustUint() })
			assert.Panics(t, func() { value.MustUint8() })
			assert.Panics(t, func() { value.MustUint16() })
			assert.Panics(t, func() { value.MustUint32() })
			assert.Panics(t, func() { value.MustUint64() })

			assert.Panics(t, func() { value.MustFloat32() })
			assert.Panics(t, func() { value.MustFloat64() })

			assert.Panics(t, func() { value.MustTime() })

			var b bool
			assert.Panics(t, func() { value.MustUnmarshal(&b) })

			var s string
			value.MustUnmarshal(&s)
			assert.Equal(t, strVal, s)

			var buf bytes.Buffer
			value.MustUnmarshal(&buf)
			assert.Empty(t, buf)

			type obj struct {
				Key string
			}
			var o obj
			value.MustUnmarshal(&o)
			assert.Equal(t, obj{Key: "Val"}, o)
		})

		t.Run("number", func(t *testing.T) {
			t.Parallel()

			strVal := "-1337"
			value := setting.NewValue(strVal)

			assert.Equal(t, strVal, value.MustString())
			assert.Equal(t, []byte(strVal), value.MustByte())

			assert.Panics(t, func() { value.MustBool() })

			assert.Equal(t, -1337, value.MustInt())
			assert.Panics(t, func() { value.MustInt8() })
			assert.Equal(t, int16(-1337), value.MustInt16())
			assert.Equal(t, int32(-1337), value.MustInt32())
			assert.Equal(t, int64(-1337), value.MustInt64())

			assert.Panics(t, func() { value.MustUint() })
			assert.Panics(t, func() { value.MustUint8() })
			assert.Panics(t, func() { value.MustUint16() })
			assert.Panics(t, func() { value.MustUint32() })
			assert.Panics(t, func() { value.MustUint64() })

			assert.Equal(t, float32(-1337), value.MustFloat32()) //nolint:testifylint // fp: does not work for expected == 0
			assert.Equal(t, float64(-1337), value.MustFloat64()) //nolint:testifylint // fp: does not work for expected == 0

			assert.Panics(t, func() { value.MustTime() })

			var b bool
			assert.Panics(t, func() { value.MustUnmarshal(&b) })

			var s string
			value.MustUnmarshal(&s)
			assert.Equal(t, strVal, s)

			var buf bytes.Buffer
			assert.Panics(t, func() { value.MustUnmarshal(&buf) })
		})

		t.Run("slog level", func(t *testing.T) {
			t.Parallel()

			value := setting.NewValue(slog.LevelInfo)
			assert.Equal(t, 0, value.MustInt())

			type ownUint uint8
			value = setting.NewValue(ownUint(1))
			assert.Equal(t, 1, value.MustInt())
			assert.Equal(t, uint8(1), value.MustUint8())

			type ownFloat float32
			value = setting.NewValue(ownFloat(1.0))
			assert.Equal(t, 1, value.MustInt())
			assert.InEpsilon(t, float32(1.0), value.MustFloat32(), 0.1)
		})
	})

	t.Run("bool", func(t *testing.T) {
		t.Parallel()

		t.Run("true", func(t *testing.T) {
			t.Parallel()

			value := setting.NewValue(true)

			assert.Equal(t, "true", value.MustString())
			assert.Equal(t, []byte("true"), value.MustByte())

			assert.True(t, value.MustBool())

			assert.Equal(t, 1, value.MustInt())
			assert.Equal(t, int8(1), value.MustInt8())
			assert.Equal(t, int16(1), value.MustInt16())
			assert.Equal(t, int32(1), value.MustInt32())
			assert.Equal(t, int64(1), value.MustInt64())

			assert.Equal(t, uint(1), value.MustUint())
			assert.Equal(t, uint8(1), value.MustUint8())
			assert.Equal(t, uint16(1), value.MustUint16())
			assert.Equal(t, uint32(1), value.MustUint32())
			assert.Equal(t, uint64(1), value.MustUint64())

			assert.Equal(t, float32(1), value.MustFloat32()) //nolint:testifylint // fp: does not work for expected == 0
			assert.Equal(t, float64(1), value.MustFloat64()) //nolint:testifylint // fp: does not work for expected == 0

			assert.Panics(t, func() { value.MustTime() })

			var b bool
			value.MustUnmarshal(&b)
			assert.True(t, b)

			var s string
			value.MustUnmarshal(&s)
			assert.Equal(t, "true", s)

			var buf bytes.Buffer
			assert.Panics(t, func() { value.MustUnmarshal(&buf) })
		})

		t.Run("false", func(t *testing.T) {
			t.Parallel()

			value := setting.NewValue(false)

			assert.Equal(t, "false", value.MustString())
			assert.Equal(t, []byte("false"), value.MustByte())

			assert.False(t, value.MustBool())

			assert.Equal(t, 0, value.MustInt())
			assert.Equal(t, int8(0), value.MustInt8())
			assert.Equal(t, int16(0), value.MustInt16())
			assert.Equal(t, int32(0), value.MustInt32())
			assert.Equal(t, int64(0), value.MustInt64())

			assert.Equal(t, uint(0), value.MustUint())
			assert.Equal(t, uint8(0), value.MustUint8())
			assert.Equal(t, uint16(0), value.MustUint16())
			assert.Equal(t, uint32(0), value.MustUint32())
			assert.Equal(t, uint64(0), value.MustUint64())

			assert.Equal(t, float32(0), value.MustFloat32()) //nolint:testifylint // fp: does not work for expected == 0
			assert.Equal(t, float64(0), value.MustFloat64()) //nolint:testifylint // fp: does not work for expected == 0

			assert.Panics(t, func() { value.MustTime() })

			var b bool
			value.MustUnmarshal(&b)
			assert.False(t, b)

			var s string
			value.MustUnmarshal(&s)
			assert.Equal(t, "false", s)

			var buf bytes.Buffer
			assert.Panics(t, func() { value.MustUnmarshal(&buf) })
		})
	})

	t.Run("numbers", func(t *testing.T) {
		t.Parallel()

		t.Run("pos int", func(t *testing.T) {
			t.Parallel()

			value := setting.NewValue(127)

			assert.Equal(t, "127", value.MustString())
			assert.Equal(t, []byte("127"), value.MustByte())

			assert.Panics(t, func() { value.MustBool() })

			assert.Equal(t, 127, value.MustInt())
			assert.Equal(t, int8(127), value.MustInt8())
			assert.Equal(t, int16(127), value.MustInt16())
			assert.Equal(t, int32(127), value.MustInt32())
			assert.Equal(t, int64(127), value.MustInt64())

			assert.Equal(t, uint(127), value.MustUint())
			assert.Equal(t, uint8(127), value.MustUint8())
			assert.Equal(t, uint16(127), value.MustUint16())
			assert.Equal(t, uint32(127), value.MustUint32())
			assert.Equal(t, uint64(127), value.MustUint64())

			assert.Equal(t, float32(127), value.MustFloat32()) //nolint:testifylint // fp: does not work for expected == 0
			assert.Equal(t, float64(127), value.MustFloat64()) //nolint:testifylint // fp: does not work for expected == 0

			assert.Panics(t, func() { value.MustTime() })

			var b bool
			assert.Panics(t, func() { value.MustUnmarshal(&b) })

			var s string
			value.MustUnmarshal(&s)
			assert.Equal(t, "127", s)

			var buf bytes.Buffer
			assert.Panics(t, func() { value.MustUnmarshal(&buf) })
		})

		t.Run("neg int", func(t *testing.T) {
			t.Parallel()

			value := setting.NewValue(-127)

			assert.Equal(t, "-127", value.MustString())
			assert.Equal(t, []byte("-127"), value.MustByte())

			assert.Panics(t, func() { value.MustBool() })

			assert.Equal(t, -127, value.MustInt())
			assert.Equal(t, int8(-127), value.MustInt8())
			assert.Equal(t, int16(-127), value.MustInt16())
			assert.Equal(t, int32(-127), value.MustInt32())
			assert.Equal(t, int64(-127), value.MustInt64())

			assert.Panics(t, func() { value.MustUint() })
			assert.Panics(t, func() { value.MustUint8() })
			assert.Panics(t, func() { value.MustUint16() })
			assert.Panics(t, func() { value.MustUint32() })
			assert.Panics(t, func() { value.MustUint64() })

			assert.Equal(t, float32(-127), value.MustFloat32()) //nolint:testifylint // fp: does not work for expected == 0
			assert.Equal(t, float64(-127), value.MustFloat64()) //nolint:testifylint // fp: does not work for expected == 0

			assert.Panics(t, func() { value.MustTime() })

			var b bool
			assert.Panics(t, func() { value.MustUnmarshal(&b) })

			var s string
			value.MustUnmarshal(&s)
			assert.Equal(t, "-127", s)

			var buf bytes.Buffer
			assert.Panics(t, func() { value.MustUnmarshal(&buf) })
		})

		t.Run("float", func(t *testing.T) {
			t.Parallel()

			value := setting.NewValue(-0.5)

			assert.Equal(t, "-0.50", value.MustString())
			assert.Equal(t, []byte("-0.50"), value.MustByte())

			assert.Panics(t, func() { value.MustBool() })

			assert.Panics(t, func() { value.MustInt() })
			assert.Panics(t, func() { value.MustInt8() })
			assert.Panics(t, func() { value.MustInt16() })
			assert.Panics(t, func() { value.MustInt32() })
			assert.Panics(t, func() { value.MustInt64() })

			assert.Panics(t, func() { value.MustUint() })
			assert.Panics(t, func() { value.MustUint8() })
			assert.Panics(t, func() { value.MustUint16() })
			assert.Panics(t, func() { value.MustUint32() })
			assert.Panics(t, func() { value.MustUint64() })

			assert.Equal(t, float32(-0.5), value.MustFloat32()) //nolint:testifylint // fp: does not work for expected == 0
			assert.Equal(t, float64(-0.5), value.MustFloat64()) //nolint:testifylint // fp: does not work for expected == 0

			assert.Panics(t, func() { value.MustTime() })

			var b bool
			assert.Panics(t, func() { value.MustUnmarshal(&b) })

			var s string
			value.MustUnmarshal(&s)
			assert.Equal(t, "-0.50", s)

			var buf bytes.Buffer
			assert.Panics(t, func() { value.MustUnmarshal(&buf) })
		})
	})

	t.Run("complex types", func(t *testing.T) {
		t.Parallel()

		type someStruct struct {
			Field string
		}

		t.Run("json", func(t *testing.T) {
			t.Parallel()

			obj := someStruct{Field: "field"}
			value := setting.NewValue(obj)

			assert.Equal(t, `{"Field":"field"}`, value.MustString())
			assert.Equal(t, []byte(`{"Field":"field"}`), value.MustByte())

			assert.Panics(t, func() { value.MustBool() })

			assert.Panics(t, func() { value.MustInt() })
			assert.Panics(t, func() { value.MustInt8() })
			assert.Panics(t, func() { value.MustInt16() })
			assert.Panics(t, func() { value.MustInt32() })
			assert.Panics(t, func() { value.MustInt64() })

			assert.Panics(t, func() { value.MustUint() })
			assert.Panics(t, func() { value.MustUint8() })
			assert.Panics(t, func() { value.MustUint16() })
			assert.Panics(t, func() { value.MustUint32() })
			assert.Panics(t, func() { value.MustUint64() })

			assert.Panics(t, func() { value.MustFloat32() })
			assert.Panics(t, func() { value.MustFloat64() })

			assert.Panics(t, func() { value.MustTime() })

			var b bool
			assert.Panics(t, func() { value.MustUnmarshal(&b) })

			var s string
			value.MustUnmarshal(&s)
			assert.Equal(t, `{"Field":"field"}`, s)

			var buf bytes.Buffer
			value.MustUnmarshal(&buf)
			assert.Equal(t, `{"Field":"field"}`, s)

			var o someStruct
			value.MustUnmarshal(&o)
			assert.Equal(t, obj, o)
		})

		t.Run("slice", func(t *testing.T) {
			t.Parallel()

			value := setting.NewValue([]int{1, 2, 3})

			assert.Equal(t, "[1,2,3]", value.MustString())
			assert.Equal(t, []byte("[1,2,3]"), value.MustByte())

			assert.Panics(t, func() { value.MustBool() })

			assert.Panics(t, func() { value.MustInt() })
			assert.Panics(t, func() { value.MustInt8() })
			assert.Panics(t, func() { value.MustInt16() })
			assert.Panics(t, func() { value.MustInt32() })
			assert.Panics(t, func() { value.MustInt64() })

			assert.Panics(t, func() { value.MustUint() })
			assert.Panics(t, func() { value.MustUint8() })
			assert.Panics(t, func() { value.MustUint16() })
			assert.Panics(t, func() { value.MustUint32() })
			assert.Panics(t, func() { value.MustUint64() })

			assert.Panics(t, func() { value.MustFloat32() })
			assert.Panics(t, func() { value.MustFloat64() })

			assert.Panics(t, func() { value.MustTime() })

			var b bool
			assert.Panics(t, func() { value.MustUnmarshal(&b) })

			var s string
			value.MustUnmarshal(&s)
			assert.Equal(t, "[1,2,3]", s)

			var buf bytes.Buffer
			assert.Panics(t, func() { value.MustUnmarshal(&buf) })

			var o []int
			value.MustUnmarshal(&o)
			assert.Equal(t, []int{1, 2, 3}, o)
		})

		t.Run("map", func(t *testing.T) {
			t.Parallel()

			mp := map[string]map[int]someStruct{
				"key": {0: {"field"}},
			}

			value := setting.NewValue(mp)
			assert.Equal(t, `{"key":{"0":{"Field":"field"}}}`, value.MustString())
			assert.Equal(t, []byte(`{"key":{"0":{"Field":"field"}}}`), value.MustByte())

			assert.Panics(t, func() { value.MustBool() })

			assert.Panics(t, func() { value.MustInt() })
			assert.Panics(t, func() { value.MustInt8() })
			assert.Panics(t, func() { value.MustInt16() })
			assert.Panics(t, func() { value.MustInt32() })
			assert.Panics(t, func() { value.MustInt64() })

			assert.Panics(t, func() { value.MustUint() })
			assert.Panics(t, func() { value.MustUint8() })
			assert.Panics(t, func() { value.MustUint16() })
			assert.Panics(t, func() { value.MustUint32() })
			assert.Panics(t, func() { value.MustUint64() })

			assert.Panics(t, func() { value.MustFloat32() })
			assert.Panics(t, func() { value.MustFloat64() })

			assert.Panics(t, func() { value.MustTime() })

			var b bool
			assert.Panics(t, func() { value.MustUnmarshal(&b) })

			var s string
			value.MustUnmarshal(&s)
			assert.Equal(t, `{"key":{"0":{"Field":"field"}}}`, s)

			var buf bytes.Buffer
			value.MustUnmarshal(&buf)
			assert.Empty(t, buf)

			var o map[string]map[int]someStruct
			value.MustUnmarshal(&o)
			assert.Equal(t, mp, o)
		})
	})

	t.Run("time", func(t *testing.T) {
		t.Parallel()

		loc, err := time.LoadLocation("Asia/Tokyo")
		assert.NoError(t, err)
		now := time.Now().In(loc)

		value := setting.NewValue(now)

		assert.Equal(t, now.Format(time.RFC3339Nano), value.MustString())
		assert.Equal(t, []byte(now.Format(time.RFC3339Nano)), value.MustByte())

		assert.Panics(t, func() { value.MustBool() })

		assert.Panics(t, func() { value.MustInt() })
		assert.Panics(t, func() { value.MustInt8() })
		assert.Panics(t, func() { value.MustInt16() })
		assert.Panics(t, func() { value.MustInt32() })
		assert.Panics(t, func() { value.MustInt64() })

		assert.Panics(t, func() { value.MustUint() })
		assert.Panics(t, func() { value.MustUint8() })
		assert.Panics(t, func() { value.MustUint16() })
		assert.Panics(t, func() { value.MustUint32() })
		assert.Panics(t, func() { value.MustUint64() })

		assert.Panics(t, func() { value.MustFloat32() })
		assert.Panics(t, func() { value.MustFloat64() })

		assert.Equal(t, now.Truncate(time.Nanosecond), value.MustTime())

		var b bool
		assert.Panics(t, func() { value.MustUnmarshal(&b) })

		var s string
		value.MustUnmarshal(&s)
		assert.Equal(t, now.Format(time.RFC3339Nano), s)

		var buf bytes.Buffer
		assert.Panics(t, func() { value.MustUnmarshal(&buf) })

		var vNow time.Time
		value.MustUnmarshal(&vNow)
		assert.Equal(t, now.Truncate(time.Nanosecond), vNow)
	})
}

func TestValue_CheckOverflows(t *testing.T) {
	t.Parallel()

	t.Run("int8", func(t *testing.T) {
		t.Parallel()

		value := setting.NewValue(int16(math.MaxInt16))

		i, err := value.Int8()
		assert.ErrorIs(t, err, setting.ErrInvalidValue)
		assert.Empty(t, i)
	})

	t.Run("int16", func(t *testing.T) {
		t.Parallel()

		value := setting.NewValue(int32(math.MaxInt32))

		i, err := value.Int16()
		assert.ErrorIs(t, err, setting.ErrInvalidValue)
		assert.Empty(t, i)
	})

	t.Run("int32", func(t *testing.T) {
		t.Parallel()

		value := setting.NewValue(int64(math.MaxInt64))

		i, err := value.Int32()
		assert.ErrorIs(t, err, setting.ErrInvalidValue)
		assert.Empty(t, i)
	})

	t.Run("uint8", func(t *testing.T) {
		t.Parallel()

		value := setting.NewValue(uint16(math.MaxUint16))

		i, err := value.Uint8()
		assert.ErrorIs(t, err, setting.ErrInvalidValue)
		assert.Empty(t, i)
	})

	t.Run("uint16", func(t *testing.T) {
		t.Parallel()

		value := setting.NewValue(uint32(math.MaxUint32))

		i, err := value.Uint16()
		assert.ErrorIs(t, err, setting.ErrInvalidValue)
		assert.Empty(t, i)
	})

	t.Run("uint32", func(t *testing.T) {
		t.Parallel()

		value := setting.NewValue(uint64(math.MaxUint64))

		i, err := value.Uint32()
		assert.ErrorIs(t, err, setting.ErrInvalidValue)
		assert.Empty(t, i)
	})

	t.Run("float32", func(t *testing.T) {
		t.Parallel()

		value := setting.NewValue(float64(math.MaxFloat64))

		f, err := value.Float32()
		assert.ErrorIs(t, err, setting.ErrInvalidValue)
		assert.Empty(t, f)
	})
}
