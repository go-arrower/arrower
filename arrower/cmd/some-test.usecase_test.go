package application_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"example/app"
)

func TestSomeTestRequestHandler_H(t *testing.T) {
	t.Parallel()

	t.Run("success case", func(t *testing.T) {
		t.Parallel()

		handler := application.NewSomeTestRequestHandler()

		res, err := handler.H(context.Background(), application.SomeTestRequest{})
		assert.NoError(t, err)
		assert.Empty(t, res)
	})
}
