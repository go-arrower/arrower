package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/arrower/hooks"
)

func TestOnConfigLoaded(t *testing.T) {
	t.Parallel()

	c := hooks.RunConfig{Port: 8080}
	_ = c
	// toto plugin.Test(t, "001_sample.hook.go")
	// p.OnConfigLoaded(c)

	assert.Equal(t, "1337", c.Port)
}
