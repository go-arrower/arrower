package main

import (
	"fmt"

	. "github.com/go-arrower/arrower/arrower/hooks"
)

func init() {
	fmt.Println("register a new hook") // force multiple hooks to import the same package, see 012_other.hook.go

	Register(Hook{
		Name: "order-10",
	})
}
