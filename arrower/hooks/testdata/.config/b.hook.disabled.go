package main

import (
	. "github.com/go-arrower/arrower/arrower/hooks"
)

func init() {
	Register(Hook{
		Name: "this hook is disabled",
	})
}
