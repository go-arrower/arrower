package application_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"example/app"
)

func TestHelloWorldRequestHandler_H(t *testing.T) {
	t.Parallel()

	t.Run("success case", func(t *testing.T) {
		t.Parallel()

		handler := application.NewHelloWorldRequestHandler()

		res, err := handler.H(context.Background(), application.HelloWorldRequest{})
		assert.NoError(t, err)
		assert.Empty(t, res)
	})
}
