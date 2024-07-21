package application_test

import (
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/contexts/auth/internal/application"
	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/repository"
)

func TestListUsersQueryHandler_H(t *testing.T) {
	t.Parallel()

	u0, _ := domain.NewUser(gofakeit.Email(), "abcdefA0$")
	u1, _ := domain.NewUser(gofakeit.Email(), "abcdefA0$")
	users := []domain.User{u0, u1}

	t.Run("no users", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewUserMemoryRepository()
		handler := application.NewListUsersQueryHandler(repo)

		res, err := handler.H(ctx, application.ListUsersQuery{})
		assert.NoError(t, err)
		assert.Empty(t, res.Users)
		assert.Equal(t, uint(0), res.Filtered)
		assert.Equal(t, uint(0), res.Total)
	})

	t.Run("load all", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewUserMemoryRepository()
		repo.SaveAll(ctx, users)
		handler := application.NewListUsersQueryHandler(repo)

		res, err := handler.H(ctx, application.ListUsersQuery{})
		assert.NoError(t, err)
		assert.NotEmpty(t, res.Users)
		assert.Equal(t, uint(2), res.Filtered)
		assert.Equal(t, uint(2), res.Total)
	})

	t.Run("paginate", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewUserMemoryRepository()
		repo.SaveAll(ctx, users)
		handler := application.NewListUsersQueryHandler(repo)

		res, err := handler.H(ctx, application.ListUsersQuery{Filter: domain.Filter{Limit: 1}})
		assert.NoError(t, err)
		assert.NotEmpty(t, res.Users)
		assert.Equal(t, uint(2), res.Filtered)
		assert.Equal(t, uint(2), res.Total)
	})

	t.Run("search users", func(t *testing.T) {
		t.Parallel()

		fUser, _ := domain.NewUser("search@email.com", "abcdefA0$")
		fUser.Name = domain.NewName("first", "last", "display")
		repo := repository.NewUserMemoryRepository()
		repo.SaveAll(ctx, append(users, fUser))
		handler := application.NewListUsersQueryHandler(repo)

		for _, tt := range []string{"first", "Last", "display ", "search@"} {
			tt := tt
			t.Run(tt, func(t *testing.T) {
				t.Parallel()

				res, err := handler.H(ctx, application.ListUsersQuery{Query: tt})
				assert.NoError(t, err)
				assert.NotEmpty(t, res.Users)
				assert.Equal(t, uint(1), res.Filtered)
				assert.Equal(t, uint(3), res.Total)
			})
		}
	})
}
