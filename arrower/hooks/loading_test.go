//go:build integration

package hooks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/arrower/hooks"
)

//nolint:paralleltest,tparallel // Register relies on a global variable and this makes the order of tests important.
func TestLoad(t *testing.T) {
	t.Parallel()

	t.Run("invalid parameters", func(t *testing.T) {
		tests := map[string]struct {
			path string
			err  error
		}{
			"empty path":       {"", hooks.ErrLoadHooksFailed},
			"non existing dir": {"./testdata/non-existing-dir", hooks.ErrLoadHooksFailed},
			"empty dir":        {"./testdata/.config-empty", nil},
		}

		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				plugins, err := hooks.Load(tt.path)
				assert.ErrorIs(t, err, tt.err)
				assert.Empty(t, plugins)
			})
		}
	})

	t.Run("load plugins from disc", func(t *testing.T) {
		hooks, err := hooks.Load("./testdata/.config")
		assert.NoError(t, err)
		assert.Len(t, hooks, 4, "disabled hooks should not be loaded")

		assert.Equal(t, "sample hook", hooks[0].Name, "should be sorted by hook file name")
		assert.Equal(t, "order-10", hooks[1].Name, "should be sorted by hook file name")
		assert.Equal(t, "order-12", hooks[2].Name, "should be sorted by hook file name")
	})

	t.Run("call hook", func(t *testing.T) {
		loadHooks, err := hooks.Load("./testdata/.config")
		assert.NoError(t, err)

		config := hooks.RunConfig{
			Port: 8080,
		}
		loadHooks[0].OnConfigLoaded(&config)
		assert.Equal(t, 1337, config.Port, "hook should change the port value")
	})

	t.Run("empty hook", func(t *testing.T) {
		hooks, err := hooks.Load("./testdata/.config")
		assert.NoError(t, err)

		plugin := hooks[3]
		assert.Equal(t, "<unknown>", plugin.Name)

		assert.NotPanics(t, func() {
			plugin.OnConfigLoaded(nil)
			plugin.OnStart()
			plugin.OnChanged("")
			plugin.OnShutdown()
		}, "if the method is not defined in the hook => a noop method is expected")
	})
}
