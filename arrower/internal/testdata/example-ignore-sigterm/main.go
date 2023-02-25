package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Println("Run example server")

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, syscall.SIGTERM, os.Interrupt)

	waitUntilShutdownFinished := make(chan struct{})

	go func() {
		<-osSignal
		fmt.Println("Shutdown signal received")

		// close(waitUntilShutdownFinished)
	}()

	fmt.Println("Waiting for shutdown")
	<-waitUntilShutdownFinished

	fmt.Println("Done")
}
