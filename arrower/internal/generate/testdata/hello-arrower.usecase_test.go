package application_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"example/app"
)

func TestHelloArrowerRequestHandler_H(t *testing.T) {
	t.Parallel()

	t.Run("success case", func(t *testing.T) {
		t.Parallel()

		handler := application.NewHelloArrowerRequestHandler()

		res, err := handler.H(context.Background(), application.HelloArrowerRequest{})
		assert.NoError(t, err)
		assert.Empty(t, res)
	})
}
