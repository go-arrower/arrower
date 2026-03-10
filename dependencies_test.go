package arrower_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower"
)

func TestInitialiseDefaultDependencies(t *testing.T) {
	t.Parallel()

	t.Run("empty config", func(t *testing.T) {
		t.Parallel()

		app, err := arrower.InitialiseDefaultDependencies(t.Context(), &arrower.Config{}, nil, nil, nil, nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, app)
		assert.Nil(t, app.PGx)
		assert.NotNil(t, app.WebRouter)
		t.Log(app.WebRouter)
	})
}
