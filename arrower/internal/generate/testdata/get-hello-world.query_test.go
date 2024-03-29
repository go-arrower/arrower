package application_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"example/app"
)

func TestGetHelloWorldQueryHandler_H(t *testing.T) {
	t.Parallel()

	t.Run("success case", func(t *testing.T) {
		t.Parallel()

		handler := application.NewGetHelloWorldQueryHandler()

		res, err := handler.H(context.Background(), application.GetHelloWorldQuery{})
		assert.NoError(t, err)
		assert.Empty(t, res)
	})
}
