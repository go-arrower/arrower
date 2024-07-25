package setting

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSuite(t *testing.T, newSettings func() Settings) { //nolint:tparallel // t.Parallel can only be called ones! The caller decides
	t.Helper()

	if newSettings == nil {
		t.Fatal("Settings constructor is nil")
	}

	var (
		ctx         = context.Background()
		key         = NewKey("arrower", "test", "setting")
		keyNotFound = NewKey("arrower", "test", "non-existing")
		value       = NewValue("setting_value")
	)

	t.Run("Save", func(t *testing.T) {
		t.Parallel()

		settings := newSettings()

		t.Run("save", func(t *testing.T) {
			t.Parallel()

			err := settings.Save(ctx, key, value)
			assert.NoError(t, err)
		})

		t.Run("update", func(t *testing.T) {
			t.Parallel()

			err := settings.Save(ctx, key, value)
			assert.NoError(t, err)

			val, err := settings.Setting(ctx, key)
			assert.NoError(t, err)
			assert.Equal(t, "setting_value", val.MustString())

			err = settings.Save(ctx, key, NewValue("setting-update"))
			assert.NoError(t, err)

			val, err = settings.Setting(ctx, key)
			assert.NoError(t, err)
			assert.Equal(t, "setting-update", val.MustString())
		})
	})

	t.Run("Setting", func(t *testing.T) {
		t.Parallel()

		settings := newSettings()

		t.Run("get setting", func(t *testing.T) {
			t.Parallel()

			err := settings.Save(ctx, key, value)
			assert.NoError(t, err)

			val, err := settings.Setting(ctx, key)
			assert.NoError(t, err)
			assert.Equal(t, "setting_value", val.MustString())
		})

		t.Run("not found", func(t *testing.T) {
			t.Parallel()

			val, err := settings.Setting(ctx, keyNotFound)
			assert.ErrorIs(t, err, ErrNotFound)
			assert.Empty(t, val.MustString())
		})
	})

	t.Run("Settings", func(t *testing.T) {
		t.Parallel()

		settings := newSettings()

		k0 := NewKey("arrower", "test", "s0")
		k1 := NewKey("arrower", "test", "s1")
		k2 := NewKey("arrower", "test", "s2")
		keys := []Key{k0, k1, k2}

		_ = settings.Save(ctx, k0, NewValue("v0"))
		_ = settings.Save(ctx, k1, NewValue("v1"))
		_ = settings.Save(ctx, k2, NewValue("v2"))

		t.Run("get settings", func(t *testing.T) {
			t.Parallel()

			s, err := settings.Settings(ctx, keys)
			assert.NoError(t, err)
			assert.Len(t, s, 3)
			assert.Equal(t, "v0", s[k0].MustString())
			assert.Equal(t, "v1", s[k1].MustString())
			assert.Equal(t, "v2", s[k2].MustString())
		})

		t.Run("not found", func(t *testing.T) {
			t.Parallel()

			s, err := settings.Settings(ctx, []Key{k0, k1, keyNotFound})
			assert.ErrorIs(t, err, ErrNotFound)

			assert.Len(t, s, 2, "error, but found settings should be returned")
			assert.Equal(t, "v0", s[k0].MustString())
			assert.Equal(t, "v1", s[k1].MustString())
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Parallel()

		settings := newSettings()

		t.Run("existing", func(t *testing.T) {
			t.Parallel()

			err := settings.Save(ctx, key, value)
			assert.NoError(t, err)

			err = settings.Delete(ctx, key)
			assert.NoError(t, err)

			val, err := settings.Setting(ctx, key)
			assert.ErrorIs(t, err, ErrNotFound)
			assert.Empty(t, val.MustString())
		})

		t.Run("non existing", func(t *testing.T) {
			t.Parallel()

			err := settings.Delete(ctx, keyNotFound)
			assert.NoError(t, err)
		})
	})

	t.Run("serialisation", func(t *testing.T) {
		t.Parallel()

		settings := newSettings()

		type someStruct struct {
			Field string
		}

		mp := map[string]map[int]someStruct{
			"key": {0: {"field"}},
		}

		t.Run("json", func(t *testing.T) {
			t.Parallel()

			key := NewKey("arrower", "test", "json")
			err := settings.Save(ctx, key, NewValue(someStruct{Field: "field"}))
			assert.NoError(t, err)

			val, err := settings.Setting(ctx, key)
			assert.NoError(t, err)
			assert.Equal(t, `{"Field":"field"}`, val.MustString())
			assert.Equal(t, []byte(`{"Field":"field"}`), val.MustByte())
		})

		t.Run("slice", func(t *testing.T) {
			t.Parallel()

			key := NewKey("arrower", "test", "slice")
			err := settings.Save(ctx, key, NewValue([]int{1, 2, 3}))
			assert.NoError(t, err)

			val, err := settings.Setting(ctx, key)
			assert.NoError(t, err)
			assert.Equal(t, "[1,2,3]", val.MustString())
			assert.Equal(t, []byte("[1,2,3]"), val.MustByte())
		})

		t.Run("map", func(t *testing.T) {
			t.Parallel()

			key := NewKey("arrower", "test", "map")
			err := settings.Save(ctx, key, NewValue(mp))
			assert.NoError(t, err)

			val, err := settings.Setting(ctx, key)
			assert.NoError(t, err)
			assert.Equal(t, `{"key":{"0":{"Field":"field"}}}`, val.MustString())
			assert.Equal(t, []byte(`{"key":{"0":{"Field":"field"}}}`), val.MustByte())

			var o map[string]map[int]someStruct
			val.MustUnmarshal(&o) //nolint:contextcheck
			assert.Equal(t, mp, o)
		})

		t.Run("time", func(t *testing.T) {
			t.Parallel()

			loc, err := time.LoadLocation("Asia/Tokyo")
			assert.NoError(t, err)
			now := time.Now().In(loc) // make the case difficult by changing the tz.

			key := NewKey("arrower", "test", "time")
			err = settings.Save(ctx, key, NewValue(now))
			assert.NoError(t, err)

			val, err := settings.Setting(ctx, key)
			assert.NoError(t, err)
			assert.Equal(t, now.Format(time.RFC3339Nano), val.MustString())
			assert.Equal(t, []byte(now.Format(time.RFC3339Nano)), val.MustByte())
			assert.Equal(t, now.Truncate(time.Nanosecond), val.MustTime())
		})
	})
}
