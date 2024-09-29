package hooks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/arrower/hooks"
)

func TestHooks_NamesFmt(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		hooks hooks.Hooks
		names string
	}{
		"empty":          {},
		"one hook":       {[]hooks.Hook{{Name: "hook 0"}}, "hook 0"},
		"multiple hooks": {[]hooks.Hook{{Name: "hook 0"}, {Name: "hook 1"}}, "hook 0, hook 1"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.names, tt.hooks.NamesFmt())
		})
	}
}

func TestHooks_OnConfigLoaded(t *testing.T) {
	t.Parallel()

	conf := &hooks.RunConfig{}

	hooks := hooks.Hooks{
		{
			OnConfigLoaded: func(c *hooks.RunConfig) {
				c.Port = 1
			},
		},
		{
			OnConfigLoaded: func(c *hooks.RunConfig) {
				c.Port = 2
			},
		},
	}

	hooks.OnConfigLoaded(conf)
	assert.Equal(t, 2, conf.Port)
}
