package jobs

import (
	"context"
)

// NewNoop returns an implementation of Queue that performs no operations.
// Ideal as dependency in tests.
func NewNoop() *noopQueue { //nolint:revive // unexported struct is alright to use.
	return &noopQueue{}
}

type noopQueue struct{}

func (n noopQueue) Enqueue(_ context.Context, _ Job, _ ...JobOption) error {
	return nil
}

func (n noopQueue) Schedule(_ string, _ Job) error {
	return nil
}

func (n noopQueue) RegisterJobFunc(_ JobFunc) error {
	return nil
}

func (n noopQueue) Shutdown(_ context.Context) error {
	return nil
}

var _ Queue = (*noopQueue)(nil)
