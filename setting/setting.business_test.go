package setting_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/setting"
)

func TestNewKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		context string
		group   string
		setting string
		expKey  string
	}{
		{
			"",
			"",
			"",
			"MISSING.MISSING.MISSING",
		},
		{
			"context",
			"",
			"",
			"context.MISSING.MISSING",
		},
		{
			"",
			"group",
			"",
			"MISSING.group.MISSING",
		},
		{
			"context",
			"group",
			"",
			"context.group.MISSING",
		},
		{
			"context",
			"",
			"setting",
			"context.MISSING.setting",
		},
		{
			"",
			"group",
			"setting",
			"MISSING.group.setting",
		},
		{
			"context",
			"group",
			"setting",
			"context.group.setting",
		},
		{
			"context",
			"group",
			"setting.custom_user_extension",
			"context.group.setting.custom_user_extension",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("%s.%s.%s", tt.context, tt.group, tt.setting), func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expKey, setting.NewKey(tt.context, tt.group, tt.setting).Key())
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		v := setting.New("")
		assert.Equal(t, "", v.String())

		v = setting.New(nil)
		assert.Equal(t, "", v.String())
	})

	t.Run("string", func(t *testing.T) {
		t.Parallel()

		v := setting.New("some")
		assert.Equal(t, setting.New("some"), v)
		assert.Equal(t, "some", v.String())
	})

	t.Run("bool", func(t *testing.T) {
		t.Parallel()

		v := setting.New(true)
		assert.True(t, true, v.Bool())
		assert.Equal(t, "true", v.String())

		v = setting.New(false)
		assert.False(t, false, v.Bool())
		assert.Equal(t, "false", v.String())
	})

	t.Run("int", func(t *testing.T) {
		t.Parallel()

		val := setting.New(123)
		assert.Equal(t, 123, val.Int())
		assert.Equal(t, int8(123), val.Int8())
		assert.Equal(t, int16(123), val.Int16())
		assert.Equal(t, int32(123), val.Int32())
		assert.Equal(t, int64(123), val.Int64())
		assert.Equal(t, "123", val.String())

		val = setting.New(int32(1337))
		assert.Equal(t, int32(1337), val.Int32())
		assert.Equal(t, 1337, val.Int())
		assert.Equal(t, "1337", val.String())

		val = setting.New(uint(123))
		assert.Equal(t, uint(123), val.Uint())
		assert.Equal(t, uint8(123), val.Uint8())
		assert.Equal(t, uint16(123), val.Uint16())
		assert.Equal(t, uint32(123), val.Uint32())
		assert.Equal(t, uint64(123), val.Uint64())
		assert.Equal(t, "123", val.String())
	})

	t.Run("float", func(t *testing.T) {
		t.Parallel()

		val := setting.New(float32(0.5))
		assert.Equal(t, float32(0.5), val.Float32()) //nolint:testifylint
		assert.Equal(t, 0.5, val.Float64())          //nolint:testifylint
		assert.Equal(t, "0.50", val.String())
	})

	t.Run("complex types - json", func(t *testing.T) {
		t.Parallel()

		type someStruct struct {
			Field string
		}

		obj := someStruct{Field: "field"}
		val := setting.New(obj)
		assert.Equal(t, `{"Field":"field"}`, val.String())
		assert.Equal(t, []byte(`{"Field":"field"}`), val.Byte())

		var o someStruct
		val.Unmarshal(&o)
		assert.Equal(t, obj, o)

		t.Run("invalid json", func(t *testing.T) {
			t.Parallel()

			val = setting.New(1337)

			var o someStruct
			val.Unmarshal(&o)

			assert.Equal(t, someStruct{}, o, "invalid json should not marshal into a struct")
		})

		t.Run("slices", func(t *testing.T) {
			t.Parallel()

			val := setting.New([]int{1, 2, 3})

			var o []int
			val.Unmarshal(&o)
			assert.Equal(t, []int{1, 2, 3}, o)
			assert.Equal(t, "[1,2,3]", val.String())
		})

		t.Run("map", func(t *testing.T) {
			t.Parallel()

			mp := map[string]map[int]someStruct{
				"key": {0: {"field"}},
			}

			val := setting.New(mp)
			assert.Equal(t, `{"key":{"0":{"Field":"field"}}}`, val.String())

			var o map[string]map[int]someStruct
			val.Unmarshal(&o)
			assert.Equal(t, mp, o)
		})
	})

	t.Run("time", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		val := setting.New(now)

		assert.Equal(t, now.Format(time.RFC3339Nano), val.String())
		assert.NotEmpty(t, val.Time())
	})
}
