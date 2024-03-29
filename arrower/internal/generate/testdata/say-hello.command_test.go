package application_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	application "github.com/go-arrower/arrower/arrower/internal/generate/testdata"
)

func TestSayHelloCommandHandler_H(t *testing.T) {
	t.Parallel()

	t.Run("success case", func(t *testing.T) {
		t.Parallel()

		handler := application.NewSayHelloCommandHandler()

		err := handler.H(context.Background(), application.SayHelloCommand{})
		assert.NoError(t, err)
	})
}
