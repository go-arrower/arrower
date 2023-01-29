package main

import (
	"fmt"

	"example/cli/domain"
)

func main() {
	fmt.Printf("hello from %v", domain.Entity("app build with arrower"))
}
