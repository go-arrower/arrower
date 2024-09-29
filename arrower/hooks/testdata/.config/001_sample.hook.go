package main

import (
	. "github.com/go-arrower/arrower/arrower/hooks"
)

func init() {
	Register(Hook{
		Name: "sample hook",
		OnConfigLoaded: func(c *RunConfig) {
			c.Port = 1337
		},
		OnStart:    nil,
		OnChanged:  nil,
		OnShutdown: nil,
	})
}
