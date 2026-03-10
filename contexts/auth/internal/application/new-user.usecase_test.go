package application_test

import (
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/contexts/auth/internal/application"
	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/repository"
)

func TestNewUserCommandHandler_H(t *testing.T) {
	t.Parallel()

	t.Run("success case", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewUserMemoryRepository()
		handler := application.NewNewUserCommandHandler(repo, registrator(t.Context(), repo))

		err := handler.H(t.Context(), application.NewUserCommand{
			Email: gofakeit.Email(),
		})
		assert.NoError(t, err)
	})
}
