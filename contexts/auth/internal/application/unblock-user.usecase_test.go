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

		repo := repository.NewUserMemoryRepository()
		repo.Save(t.Context(), userBlocked)

		handler := application.NewUnblockUserRequestHandler(repo)
		_, err := handler.H(t.Context(), application.UnblockUserRequest{UserID: userBlockedUserID})
		assert.NoError(t, err)

		// verify
		usr, err := repo.FindByID(t.Context(), userBlockedUserID)
		assert.NoError(t, err)
		assert.False(t, usr.IsBlocked())
	})
}
