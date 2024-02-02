package setting_test

import (
	"testing"

	"github.com/go-arrower/arrower/setting"
)

func TestInMemorySettings(t *testing.T) {
	t.Parallel()

	setting.TestSettings(t, func() setting.Settings { return setting.NewInMemorySettings() })
}
