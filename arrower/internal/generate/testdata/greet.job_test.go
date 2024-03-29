package application_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"example/app"
)

func TestGreetJobHandler_H(t *testing.T) {
	t.Parallel()

	t.Run("success case", func(t *testing.T) {
		t.Parallel()

		handler := application.NewGreetJobHandler()

		err := handler.H(context.Background(), application.GreetJob{})
		assert.NoError(t, err)
	})
}
