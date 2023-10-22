package internal_test

import (
	"bytes"
	"sync"
)

// colourful cli output of `arrower run`.
const (
	arrowerCLIBuild          = "building..."
	arrowerCLIRun            = "running..."
	arrowerCLIChangeDetected = "changed:"
)

// values printed to Stdout by the testdata/example/server.
const (
	exampleServerStart           = "Run example server"
	exampleServerWaitForShutdown = "Waiting for shutdown"
	exampleServerShutdown        = "Shutdown signal received"
	exampleServerDone            = "Done"
)

// syncBuffer is a helper implementing io.Writer, used for concurrency save testing.
type syncBuffer struct {
	b bytes.Buffer
	m sync.Mutex
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.m.Lock()
	defer b.m.Unlock()

	return b.b.Write(p) //nolint:wrapcheck
}

func (b *syncBuffer) String() string {
	b.m.Lock()
	defer b.m.Unlock()

	return b.b.String()
}

func noBrowser(_ string) error { return nil }
