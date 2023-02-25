package main

import (
	"fmt"
	"log"

	"example/cli/domain"
)

func main() {
	fmt.Printf("hello from %v\n", domain.Entity("app build with arrower"))

	fmt.Println("hello from fmt")
	log.Println("hello from log")
}
