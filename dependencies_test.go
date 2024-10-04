package arrower_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower"
)

func TestInitialiseDefaultDependencies(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("empty config", func(t *testing.T) {
		t.Parallel()

		app, err := arrower.InitialiseDefaultDependencies(ctx, &arrower.Config{}, nil, nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, app)
		assert.Nil(t, app.PGx)
		assert.NotNil(t, app.WebRouter)
		t.Log(app.WebRouter)
	})
}
