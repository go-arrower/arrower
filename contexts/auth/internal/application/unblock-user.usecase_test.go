package application_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/contexts/auth/internal/application"
	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/repository"
)

func TestUnblockUserRequestHandler_H(t *testing.T) {
	t.Parallel()

	t.Run("unblock user", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository()
		repo.Save(ctx, userBlocked)

		handler := application.NewUnblockUserRequestHandler(repo)
		_, err := handler.H(ctx, application.UnblockUserRequest{UserID: userBlockedUserID})
		assert.NoError(t, err)

		// verify
		usr, err := repo.FindByID(ctx, userBlockedUserID)
		assert.NoError(t, err)
		assert.False(t, usr.IsBlocked())
	})
}
