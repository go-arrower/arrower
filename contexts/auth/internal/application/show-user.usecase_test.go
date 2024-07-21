package application_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/contexts/auth/internal/application"
	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/repository"
)

func TestShowUserQueryHandler_H(t *testing.T) {
	t.Parallel()

	t.Run("invalid userID", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewUserMemoryRepository()

		handler := application.NewShowUserQueryHandler(repo)
		res, err := handler.H(ctx, application.ShowUserQuery{})
		assert.Error(t, err)
		assert.Empty(t, res)
	})

	t.Run("show user", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewUserMemoryRepository()
		repo.Save(ctx, userVerified)

		handler := application.NewShowUserQueryHandler(repo)
		res, err := handler.H(ctx, application.ShowUserQuery{
			UserID: userIDZero,
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, res)

		assert.Equal(t, userIDZero, res.User.ID)
		assert.Len(t, res.User.Sessions, 1)
	})
}
