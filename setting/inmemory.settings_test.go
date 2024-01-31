package setting_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/setting"
)

func TestNewInMemorySettings(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	wg := sync.WaitGroup{}
	key := setting.NewKey("arrower", "test", "test")

	settings := setting.NewInMemorySettings()

	wg.Add(3)
	settings.OnSettingChange(key, func(s setting.Value) {
		t.Log("setting changed", key, s)

		wg.Done()
	})

	settings.Save(ctx, key, setting.NewValue(true))
	settings.Save(ctx, key, setting.NewValue(false))
	// runtime.Gosched()
	settings.Save(ctx, key, setting.NewValue(true))
	settings.Save(ctx, key, setting.NewValue(true)) // no OnSettingChange callback triggered by this change

	isTest, _ := settings.Setting(ctx, key)
	assert.True(t, isTest.MustBool())

	wg.Wait()
}
