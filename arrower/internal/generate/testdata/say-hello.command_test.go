package application_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"example/app"
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
