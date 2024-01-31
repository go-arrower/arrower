package setting_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/setting"
)

var ctx = context.Background()

func TestSettingsHandler_Setting(t *testing.T) {
	t.Parallel()

	settings := setting.NewInMemorySettings()
	key := setting.NewKey("c", "g", "s")

	err := settings.Save(ctx, key, setting.NewValue("setting"))
	assert.NoError(t, err)

	val, err := settings.Setting(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, "setting", val.MustString())

	err = settings.Save(ctx, key, setting.NewValue("setting-update"))
	assert.NoError(t, err)

	val, err = settings.Setting(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, "setting-update", val.MustString())

	_, err = settings.Setting(ctx, setting.NewKey("", "", ""))
	assert.ErrorIs(t, err, setting.ErrNotFound)
}

func TestSettingsHandler_Settings(t *testing.T) {
	t.Parallel()

	k0 := setting.NewKey("", "", "s0")
	k1 := setting.NewKey("", "", "s1")
	k2 := setting.NewKey("", "", "s2")
	keys := []setting.Key{k0, k1, k2}

	settings := setting.NewInMemorySettings()
	_ = settings.Save(ctx, k0, setting.NewValue(nil))
	_ = settings.Save(ctx, k1, setting.NewValue(nil))
	_ = settings.Save(ctx, k2, setting.NewValue(nil))

	s, err := settings.Settings(ctx, keys)
	assert.NoError(t, err)
	assert.Len(t, s, 3)
}

func TestInMemorySettings_Delete(t *testing.T) {
	t.Parallel()

	settings := setting.NewInMemorySettings()
	key := setting.NewKey("c", "g", "s")

	err := settings.Delete(ctx, key)
	assert.NoError(t, err)

	_ = settings.Save(ctx, key, setting.NewValue("setting"))
	err = settings.Delete(ctx, key)
	assert.NoError(t, err)

	_, err = settings.Setting(ctx, key)
	assert.ErrorIs(t, err, setting.ErrNotFound)
}
