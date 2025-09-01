package arepo_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/arepo"
	"github.com/go-arrower/arrower/arepo/testdata"
)

func TestTest(t *testing.T) {
	t.Parallel()

	t.Run("test repo", func(t *testing.T) {
		t.Parallel()

		repo := arepo.Test[testdata.Entity, testdata.EntityID](t)
		assert.NotNil(t, repo)
	})

	t.Run("nil does panic", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			arepo.Test[testdata.Entity, testdata.EntityID](nil)
		})
	})
}

func TestTestAssert(t *testing.T) {
	t.Parallel()

	repo := arepo.NewMemoryRepository[testdata.Entity, testdata.EntityID]()
	rassert := arepo.TestAssert[testdata.Entity, testdata.EntityID](t, repo)
	// rassert = arepo.TestAssert(new(testing.T), repo) // TODO see if this can be made to work

	pass := rassert.Empty()
	assert.True(t, pass)
}

func TestTestAssertions_Empty(t *testing.T) {
	t.Parallel()

	t.Run("empty repo", func(t *testing.T) {
		t.Parallel()

		repo := arepo.Test[testdata.Entity, testdata.EntityID](new(testing.T))

		pass := repo.Empty()
		assert.True(t, pass)
	})

	t.Run("not empty repo", func(t *testing.T) {
		t.Parallel()

		repo := arepo.Test[testdata.Entity, testdata.EntityID](new(testing.T))

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

		repo := arepo.Test[testdata.Entity, testdata.EntityID](new(testing.T))

		pass := repo.NotEmpty()
		assert.False(t, pass)
	})

	t.Run("not empty repo", func(t *testing.T) {
		t.Parallel()

		repo := arepo.Test[testdata.Entity, testdata.EntityID](new(testing.T))

		err := repo.Add(ctx, testdata.DefaultEntity)
		assert.NoError(t, err)

		pass := repo.NotEmpty()
		assert.True(t, pass)
	})
}

func TestTestAssertions_Total(t *testing.T) {
	t.Parallel()

	t.Run("empty repo", func(t *testing.T) {
		t.Parallel()

		repo := arepo.Test[testdata.Entity, testdata.EntityID](new(testing.T))

		pass := repo.Total(0)
		assert.True(t, pass, "empty repo -> pass")

		pass = repo.Total(1)
		assert.False(t, pass, "empty repo -> fails")
	})

	t.Run("not empty repo", func(t *testing.T) {
		t.Parallel()

		repo := arepo.Test[testdata.Entity, testdata.EntityID](new(testing.T))

		err := repo.Add(ctx, testdata.DefaultEntity)
		assert.NoError(t, err)

		pass := repo.Total(1, "not empty repo -> pass")
		assert.True(t, pass)

		pass = repo.Total(0, "not empty repo -> fails")
		assert.False(t, pass)
	})
}
