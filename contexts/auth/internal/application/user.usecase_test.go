package application_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/contexts/auth/internal/application"
	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/repository"
)

func TestBlockUser(t *testing.T) {
	t.Parallel()

	t.Run("block user", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository()
		repo.Save(ctx, userVerified)

		cmd := application.BlockUser(repo)
		_, err := cmd(ctx, application.BlockUserRequest{UserID: userIDZero})
		assert.NoError(t, err)

		// verify
		usr, err := repo.FindByID(ctx, userIDZero)
		assert.NoError(t, err)
		assert.True(t, usr.IsBlocked())
	})
}

func TestUnblockUser(t *testing.T) {
	t.Parallel()

	t.Run("unblock user", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository()
		repo.Save(ctx, userBlocked)

		cmd := application.UnblockUser(repo)
		_, err := cmd(ctx, application.BlockUserRequest{UserID: userBlockedUserID})
		assert.NoError(t, err)

		// verify
		usr, err := repo.FindByID(ctx, userBlockedUserID)
		assert.NoError(t, err)
		assert.True(t, !usr.IsBlocked())
	})
}
