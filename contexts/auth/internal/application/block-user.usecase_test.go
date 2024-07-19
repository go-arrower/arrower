package application_test

import (
	"testing"

	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/repository"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/contexts/auth/internal/application"
)

func TestBlockUserRequestHandler_H(t *testing.T) {
	t.Parallel()

	t.Run("block user", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository()
		repo.Save(ctx, userVerified)

		handler := application.NewBlockUserRequestHandler(repo)
		_, err := handler.H(ctx, application.BlockUserRequest{UserID: userIDZero})
		assert.NoError(t, err)

		// verify
		usr, err := repo.FindByID(ctx, userIDZero)
		assert.NoError(t, err)
		assert.True(t, usr.IsBlocked())
	})
}
