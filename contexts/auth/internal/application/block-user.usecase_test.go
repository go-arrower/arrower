package application_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/contexts/auth/internal/application"
	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/repository"
)

func TestBlockUserRequestHandler_H(t *testing.T) {
	t.Parallel()

	true := true
	false := false

	t.Run("block user", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewUserMemoryRepository()
		repo.Save(t.Context(), userVerified)

		handler := application.NewBlockUserRequestHandler(repo)
		_, err := handler.H(t.Context(), application.BlockUserRequest{
			UserID:     userIDZero,
			SetBlocked: &true,
		})
		assert.NoError(t, err)

		// verify
		usr, err := repo.FindByID(t.Context(), userIDZero)
		assert.NoError(t, err)
		assert.True(t, usr.IsBlocked())
	})

	t.Run("unblock user", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewUserMemoryRepository()
		repo.Save(t.Context(), userBlocked)

		handler := application.NewBlockUserRequestHandler(repo)
		_, err := handler.H(t.Context(), application.BlockUserRequest{
			UserID:     userBlockedUserID,
			SetBlocked: &false,
		})
		assert.NoError(t, err)

		// verify
		usr, err := repo.FindByID(t.Context(), userBlockedUserID)
		assert.NoError(t, err)
		assert.False(t, usr.IsBlocked())
	})
}
