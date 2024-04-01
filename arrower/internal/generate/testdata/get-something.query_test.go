package application_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"example/app"
)

func TestGetSomethingQueryHandler_H(t *testing.T) {
	t.Parallel()

	t.Run("success case", func(t *testing.T) {
		t.Parallel()

		handler := application.NewGetSomethingQueryHandler()

		res, err := handler.H(context.Background(), application.GetSomethingQuery{})
		assert.NoError(t, err)
		assert.Empty(t, res)
	})
}
