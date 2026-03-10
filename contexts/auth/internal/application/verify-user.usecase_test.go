package application_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/contexts/auth/internal/application"
	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/repository"
)

func TestVerifyUserCommandHandler_H(t *testing.T) {
	t.Parallel()

	t.Run("verify user", func(t *testing.T) {
		t.Parallel()

		// setup
		repo := repository.NewUserMemoryRepository()
		repo.Save(t.Context(), userNotVerified)

		usr, _ := repo.FindByID(t.Context(), userNotVerifiedUserID)
		verify := domain.NewVerificationService(repo)
		token, _ := verify.NewVerificationToken(t.Context(), usr)

		handler := application.NewVerifyUserCommandHandler(repo)

		// action
		err := handler.H(t.Context(), application.VerifyUserCommand{
			Token:  token.Token(),
			UserID: userNotVerifiedUserID,
		})
		assert.NoError(t, err)

		usr, _ = repo.FindByID(t.Context(), userNotVerifiedUserID)
		assert.True(t, usr.IsVerified())
	})
}
