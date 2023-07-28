package mw_test

import (
	"context"
	"testing"

	"github.com/go-arrower/arrower/mw"
)

func TestTraced(t *testing.T) {
	t.Parallel()

	t.Run("successful command", func(t *testing.T) {
		t.Parallel()

		cmd := mw.Traced(newFakeTracer(t), func(context.Context, exampleCommand) (string, error) {
			return "", nil
		})

		_, _ = cmd(context.Background(), exampleCommand{})
	})

	t.Run("failed command", func(t *testing.T) {
		t.Parallel()

		cmd := mw.Traced(newFakeTracer(t), func(context.Context, exampleCommand) (string, error) {
			return "", errUseCaseFails
		})

		_, _ = cmd(context.Background(), exampleCommand{})
	})
}

func TestTracedU(t *testing.T) {
	t.Parallel()

	t.Run("successful command", func(t *testing.T) {
		t.Parallel()

		cmd := mw.TracedU(newFakeTracer(t), func(context.Context, exampleCommand) error {
			return nil
		})

		_ = cmd(context.Background(), exampleCommand{})
	})

	t.Run("failed command", func(t *testing.T) {
		t.Parallel()

		cmd := mw.TracedU(newFakeTracer(t), func(context.Context, exampleCommand) error {
			return errUseCaseFails
		})

		_ = cmd(context.Background(), exampleCommand{})
	})
}
