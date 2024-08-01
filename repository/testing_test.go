package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/repository"
	"github.com/go-arrower/arrower/repository/testdata"
)

func TestTest(t *testing.T) {
	t.Parallel()

	t.Run("test repo", func(t *testing.T) {
		t.Parallel()

		repo := repository.Test[testdata.Entity, testdata.EntityID](t)
		assert.NotNil(t, repo)
	})

	t.Run("nil does panic", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			repository.Test[testdata.Entity, testdata.EntityID](nil)
		})
	})
}

func TestTestAssert(t *testing.T) {
	t.Parallel()

	repo := repository.NewMemoryRepository[testdata.Entity, testdata.EntityID]()
	rassert := repository.TestAssert[testdata.Entity, testdata.EntityID](t, repo)
	//rassert = repository.TestAssert(new(testing.T), repo) // TODO see if this can be made to work

	rassert.Empty()
}

func TestTestAssertions_Empty(t *testing.T) {
	t.Parallel()

	t.Run("empty repo", func(t *testing.T) {
		t.Parallel()

		repo := repository.Test[testdata.Entity, testdata.EntityID](new(testing.T))

		pass := repo.Empty()
		assert.True(t, pass)
	})

	t.Run("not empty repo", func(t *testing.T) {
		t.Parallel()

		repo := repository.Test[testdata.Entity, testdata.EntityID](new(testing.T))

		err := repo.Add(ctx, testdata.DefaultEntity)
		assert.NoError(t, err)

		pass := repo.Empty()
		assert.False(t, pass)
	})
}

func TestTestAssertions_NotEmpty(t *testing.T) {
	t.Parallel()

	t.Run("empty repo", func(t *testing.T) {
		t.Parallel()

		repo := repository.Test[testdata.Entity, testdata.EntityID](new(testing.T))

		pass := repo.NotEmpty()
		assert.False(t, pass)
	})

	t.Run("not empty repo", func(t *testing.T) {
		t.Parallel()

		repo := repository.Test[testdata.Entity, testdata.EntityID](new(testing.T))

		err := repo.Add(ctx, testdata.DefaultEntity)
		assert.NoError(t, err)

		pass := repo.NotEmpty()
		assert.True(t, pass)
	})
}
