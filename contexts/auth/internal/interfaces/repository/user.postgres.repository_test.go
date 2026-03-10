//go:build integration

package repository_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/repository"
	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/repository/models"
	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/repository/testdata"
	"github.com/go-arrower/arrower/tests"
)

var pgHandler *tests.PostgresDocker

func TestMain(m *testing.M) {
	pgHandler = tests.GetPostgresDockerForIntegrationTestingInstance()

	//
	// Run tests
	code := m.Run()

	pgHandler.Cleanup()
	os.Exit(code)
}

func TestNewPostgresRepository(t *testing.T) {
	t.Parallel()

	_, err := repository.NewUserPostgresRepository(nil)
	assert.ErrorIs(t, err, repository.ErrMissingConnection)
}

func TestPostgresRepository_All(t *testing.T) {
	t.Parallel()

	t.Run("all", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo, _ := repository.NewUserPostgresRepository(pg)

		all, err := repo.All(t.Context(), domain.Filter{})
		assert.NoError(t, err)
		assert.Len(t, all, 3)
		assert.Len(t, all[0].Sessions, 1, "user should have its value objects returned")
	})

	t.Run("paginate", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo, _ := repository.NewUserPostgresRepository(pg)

		// first page
		all, err := repo.All(t.Context(), domain.Filter{
			Limit:  2,
			Offset: "",
		})
		assert.NoError(t, err)
		assert.Len(t, all, 2)
		assert.Equal(t, domain.ID("00000000-0000-0000-0000-000000000000"), all[0].ID)
		assert.Equal(t, domain.ID("00000000-0000-0000-0000-000000000001"), all[1].ID)

		// next page
		all, err = repo.All(t.Context(), domain.Filter{
			Limit:  2,
			Offset: all[1].Login,
		})
		assert.NoError(t, err)
		assert.Len(t, all, 1)
		assert.Equal(t, domain.ID("00000000-0000-0000-0000-000000000002"), all[0].ID)
	})
}

func TestPostgresRepository_FindByID(t *testing.T) {
	t.Parallel()

	t.Run("valid user", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo, _ := repository.NewUserPostgresRepository(pg)

		u, err := repo.FindByID(t.Context(), testdata.UserIDOne)
		assert.NoError(t, err)
		assert.Equal(t, testdata.UserIDOne, u.ID)
	})

	t.Run("invalid user", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo, _ := repository.NewUserPostgresRepository(pg)

		_, err := repo.FindByID(t.Context(), testdata.UserIDNotValid)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestPostgresRepository_FindByLogin(t *testing.T) {
	t.Parallel()

	t.Run("valid user", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo, _ := repository.NewUserPostgresRepository(pg)

		u, err := repo.FindByLogin(t.Context(), testdata.ValidLogin)
		assert.NoError(t, err)
		assert.Equal(t, testdata.ValidLogin, u.Login)
	})

	t.Run("invalid user", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo, _ := repository.NewUserPostgresRepository(pg)

		_, err := repo.FindByLogin(t.Context(), testdata.NotExLogin)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestPostgresRepository_ExistsByID(t *testing.T) {
	t.Parallel()

	t.Run("user exists", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo, _ := repository.NewUserPostgresRepository(pg)

		ex, err := repo.ExistsByID(t.Context(), testdata.UserIDZero)
		assert.NoError(t, err)
		assert.True(t, ex)
	})

	t.Run("user does not exist", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo, _ := repository.NewUserPostgresRepository(pg)

		ex, err := repo.ExistsByID(t.Context(), testdata.UserIDNotExists)
		assert.NoError(t, err)
		assert.False(t, ex)
	})

	t.Run("invalid user id", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo, _ := repository.NewUserPostgresRepository(pg)

		_, err := repo.ExistsByID(t.Context(), testdata.UserIDNotValid)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestPostgresRepository_ExistsByLogin(t *testing.T) {
	t.Parallel()

	t.Run("user exists", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo, _ := repository.NewUserPostgresRepository(pg)

		ex, err := repo.ExistsByLogin(t.Context(), testdata.ValidLogin)
		assert.NoError(t, err)
		assert.True(t, ex)
	})

	t.Run("user does not exist", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo, _ := repository.NewUserPostgresRepository(pg)

		ex, err := repo.ExistsByLogin(t.Context(), testdata.NotExLogin)
		assert.NoError(t, err)
		assert.False(t, ex)
	})
}

func TestPostgresRepository_Count(t *testing.T) {
	t.Parallel()

	pg := pgHandler.NewTestDatabase()
	repo, _ := repository.NewUserPostgresRepository(pg)

	c, err := repo.Count(t.Context())
	assert.NoError(t, err)
	assert.Equal(t, 3, c)
}

func TestPostgresRepository_Save(t *testing.T) {
	t.Parallel()

	t.Run("save new user", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo, _ := repository.NewUserPostgresRepository(pg)

		err := repo.Save(t.Context(), domain.User{
			ID: testdata.UserIDNew,
			Sessions: []domain.Session{
				{
					ID:        "some-new-session-key",
					Device:    domain.NewDevice(testdata.UserAgent),
					CreatedAt: time.Now().UTC(),
				},
			},
		})
		assert.NoError(t, err)

		c, _ := repo.Count(t.Context())
		assert.Equal(t, 4, c)

		queries := models.New(pg)
		sessions, _ := queries.AllSessions(t.Context())
		assert.Len(t, sessions, 1+1) // 1 session is already created via _common.yaml fixtures
	})

	t.Run("save existing user", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo, _ := repository.NewUserPostgresRepository(pg)

		usr, _ := repo.FindByID(t.Context(), testdata.UserIDZero)
		assert.Empty(t, usr.Name)

		usr.Name = domain.NewName("firstName", "", "")
		err := repo.Save(t.Context(), usr)
		assert.NoError(t, err)

		usr, _ = repo.FindByID(t.Context(), testdata.UserIDZero)
		assert.NotEmpty(t, usr.Name)
	})

	t.Run("save empty user", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo, _ := repository.NewUserPostgresRepository(pg)

		err := repo.Save(t.Context(), domain.User{})
		assert.ErrorIs(t, err, domain.ErrPersistenceFailed)
	})
}

func TestPostgresRepository_SaveAll(t *testing.T) {
	t.Parallel()

	t.Run("save multiple", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo, _ := repository.NewUserPostgresRepository(pg)

		newUser := domain.User{ID: testdata.UserIDNew}
		updatedUser := testdata.UserZero
		updatedUser.Name = domain.NewName("firstName", "", "")

		err := repo.SaveAll(t.Context(), []domain.User{
			newUser,
			updatedUser,
		})
		assert.NoError(t, err)

		c, _ := repo.Count(t.Context())
		assert.Equal(t, 4, c)

		u, _ := repo.FindByID(t.Context(), testdata.UserIDZero)
		assert.NotEmpty(t, u.Name)
	})
}

func TestPostgresRepository_Delete(t *testing.T) {
	t.Parallel()

	t.Run("delete user", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo, _ := repository.NewUserPostgresRepository(pg)

		err := repo.Delete(t.Context(), testdata.UserZero)
		assert.NoError(t, err)

		c, _ := repo.Count(t.Context())
		assert.Equal(t, 2, c)
	})
}

func TestPostgresRepository_DeleteByID(t *testing.T) {
	t.Parallel()

	t.Run("delete user", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo, _ := repository.NewUserPostgresRepository(pg)

		err := repo.DeleteByID(t.Context(), testdata.UserIDZero)
		assert.NoError(t, err)

		c, _ := repo.Count(t.Context())
		assert.Equal(t, 2, c)
	})

	t.Run("invalid id", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo, _ := repository.NewUserPostgresRepository(pg)

		err := repo.DeleteByID(t.Context(), testdata.UserIDNotValid)
		assert.ErrorIs(t, err, domain.ErrPersistenceFailed)
	})
}

func TestPostgresRepository_DeleteByIDs(t *testing.T) {
	t.Parallel()

	t.Run("delete users", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo, _ := repository.NewUserPostgresRepository(pg)

		err := repo.DeleteByIDs(t.Context(), []domain.ID{
			testdata.UserIDZero,
			testdata.UserIDOne,
		})
		assert.NoError(t, err)

		c, _ := repo.Count(t.Context())
		assert.Equal(t, 1, c)
	})
}

func TestPostgresRepository_DeleteAll(t *testing.T) {
	t.Parallel()

	t.Run("delete all users", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo, _ := repository.NewUserPostgresRepository(pg)

		err := repo.DeleteAll(t.Context())
		assert.NoError(t, err)

		c, _ := repo.Count(t.Context())
		assert.Equal(t, 0, c)
	})
}

func TestPostgresRepository_CreateVerificationToken(t *testing.T) {
	t.Parallel()

	t.Run("create new token", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()
		repo, _ := repository.NewUserPostgresRepository(pg)

		err := repo.CreateVerificationToken(t.Context(), testdata.ValidToken)
		assert.NoError(t, err)

		tok, err := repo.VerificationTokenByToken(t.Context(), testdata.ValidToken.Token())
		assert.NoError(t, err)
		assert.Equal(t, testdata.ValidToken.Token(), tok.Token())
	})
}
