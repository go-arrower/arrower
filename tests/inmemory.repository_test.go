package tests_test

import (
	"context"
	"sync"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/tests"
	"github.com/go-arrower/arrower/tests/testdata"
)

var ctx = context.Background()

func TestNewMemoryRepository(t *testing.T) {
	t.Parallel()

	repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()
	assert.NotNil(t, repo)
}

func TestNewMemoryRepository_Ints(t *testing.T) {
	t.Parallel()

	type (
		entityInt  int
		entityUint uint
		entity     struct {
			IntID  entityInt
			UintID entityUint
			Name   string
		}
	)

	repo := tests.NewMemoryRepository[entity, entityInt](tests.WithIDField("IntID"))

	t.Run("generate IDs of int type", func(t *testing.T) {
		t.Parallel()
		id, _ := repo.NextID(ctx)
		t.Log(id)

		id, _ = repo.NextID(ctx)
		t.Log(id)

		id, _ = repo.NextID(ctx)
		t.Log(id)
		assert.Equal(t, entityInt(3), id)
	})

	t.Run("access the structs ID of int field properly", func(t *testing.T) {
		t.Parallel()

		err := repo.Save(ctx, entity{IntID: 1337, Name: gofakeit.Name()})
		assert.NoError(t, err)

		c, _ := repo.Count(ctx)
		assert.Equal(t, 1, c)
	})

	t.Run("use uint field", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[entity, entityUint](tests.WithIDField("UintID"))

		id, err := repo.NextID(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, id)

		err = repo.Save(ctx, entity{UintID: 1337, Name: gofakeit.Name()})
		assert.NoError(t, err)
	})
}

func TestEntityWithoutID(t *testing.T) {
	t.Parallel()

	repo := tests.NewMemoryRepository[testdata.EntityWithoutID, testdata.EntityID]()

	assert.Panics(t, func() {
		repo.Save(ctx, testdata.EntityWithoutID{})
	})
}

func TestWithIDField(t *testing.T) {
	t.Parallel()

	repo := tests.NewMemoryRepository[testdata.EntityWithoutID, string](
		tests.WithIDField("Name"),
	)

	err := repo.Save(ctx, testdata.EntityWithoutID{Name: gofakeit.Name()})
	assert.NoError(t, err)
}

func TestMemoryRepository_NextID(t *testing.T) {
	t.Parallel()

	t.Run("string", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()
		id, err := repo.NextID(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, id)

		t.Log(id)
	})

	t.Run("int", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[testdata.Entity, int]()
		id, err := repo.NextID(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, id)

		t.Log(id)
	})
}

func TestMemoryRepository_Create(t *testing.T) {
	t.Parallel()

	t.Run("create", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()
		err := repo.Create(ctx, testdata.DefaultEntity)
		assert.NoError(t, err)

		got, err := repo.Read(ctx, testdata.DefaultEntity.ID)
		assert.NoError(t, err)
		assert.Equal(t, testdata.DefaultEntity, got)
	})

	t.Run("create same again", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()

		err := repo.Create(ctx, testdata.DefaultEntity)
		assert.NoError(t, err)

		err = repo.Create(ctx, testdata.DefaultEntity)
		assert.ErrorIs(t, err, tests.ErrAlreadyExists, "already exists")
	})

	t.Run("missing id", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()

		err := repo.Create(ctx, testdata.Entity{})
		assert.ErrorIs(t, err, tests.ErrSaveFailed)
	})
}

func TestMemoryRepository_Update(t *testing.T) {
	t.Parallel()

	t.Run("update", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()
		entity := testdata.NewEntity()
		repo.Create(ctx, entity)

		entity.Name = "updated-name"
		err := repo.Update(ctx, entity)
		assert.NoError(t, err)

		e, _ := repo.FindByID(ctx, entity.ID)
		assert.Equal(t, entity, e)
	})

	t.Run("does not exist yet", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()

		err := repo.Update(ctx, testdata.DefaultEntity)
		assert.ErrorIs(t, err, tests.ErrSaveFailed)
	})

	t.Run("missing id", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()
		repo.Create(ctx, testdata.DefaultEntity)

		err := repo.Update(ctx, testdata.Entity{})
		assert.ErrorIs(t, err, tests.ErrSaveFailed)
	})
}

func TestMemoryRepository_Delete(t *testing.T) {
	t.Parallel()

	repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()
	repo.Add(ctx, testdata.DefaultEntity)

	err := repo.Delete(ctx, testdata.DefaultEntity)
	assert.NoError(t, err)

	e, err := repo.FindByID(ctx, testdata.DefaultEntity.ID)
	assert.ErrorIs(t, err, tests.ErrNotFound)
	assert.Empty(t, e)
}

func TestMemoryRepository_All(t *testing.T) {
	t.Parallel()

	repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()

	all, err := repo.All(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, all)
	assert.Empty(t, all, "new repository should be empty")

	repo.Create(ctx, testdata.NewEntity())
	repo.Create(ctx, testdata.NewEntity())

	all, err = repo.All(ctx)
	assert.NoError(t, err)
	assert.Len(t, all, 2, "should have two entities")
}

func TestMemoryRepository_AllByIDs(t *testing.T) {
	t.Parallel()

	e0 := testdata.NewEntity()
	e1 := testdata.NewEntity()
	repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()
	repo.Create(ctx, e0)
	repo.Create(ctx, testdata.NewEntity())
	repo.Create(ctx, e1)
	repo.Create(ctx, testdata.NewEntity())

	all, err := repo.AllByIDs(ctx, nil)
	assert.NoError(t, err)
	assert.Empty(t, all)

	all, err = repo.AllByIDs(ctx, []testdata.EntityID{})
	assert.NoError(t, err, "empty list should not return an error")
	assert.Empty(t, all)

	all, err = repo.AllByIDs(ctx, []testdata.EntityID{e0.ID, e1.ID})
	assert.NoError(t, err)
	assert.Len(t, all, 2, "should find all ids")

	all, err = repo.AllByIDs(ctx, []testdata.EntityID{
		e0.ID,
		testdata.EntityID(uuid.New().String()),
	})
	assert.ErrorIs(t, err, tests.ErrNotFound, "should not find all ids")
	assert.Empty(t, all)
}

func TestMemoryRepository_FindByID(t *testing.T) {
	t.Parallel()

	repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()

	e, err := repo.FindByID(ctx, testdata.NewEntity().ID)
	assert.ErrorIs(t, err, tests.ErrNotFound)
	assert.Empty(t, e)
}

func TestMemoryRepository_Contains(t *testing.T) {
	t.Parallel()

	repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()

	ex, err := repo.ContainsID(ctx, testdata.EntityID(uuid.New().String()))
	assert.NoError(t, err)
	assert.False(t, ex, "id should not exist")

	repo.Create(ctx, testdata.DefaultEntity)

	ex, err = repo.Contains(ctx, testdata.DefaultEntity.ID)
	assert.NoError(t, err)
	assert.True(t, ex, "id should exist")
}

func TestMemoryRepository_ContainsAll(t *testing.T) {
	t.Parallel()

	e0 := testdata.NewEntity()
	e1 := testdata.NewEntity()
	repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()
	repo.Create(ctx, e0)
	repo.Create(ctx, testdata.NewEntity())
	repo.Create(ctx, e1)
	repo.Create(ctx, testdata.NewEntity())

	ex, err := repo.ContainsAll(ctx, nil)
	assert.NoError(t, err)
	assert.False(t, ex)

	ex, err = repo.ContainsAll(ctx, []testdata.EntityID{})
	assert.NoError(t, err)
	assert.False(t, ex)

	ex, err = repo.ContainsAll(ctx, []testdata.EntityID{e0.ID, e1.ID})
	assert.NoError(t, err)
	assert.True(t, ex)

	ex, err = repo.ContainsAll(ctx, []testdata.EntityID{testdata.NewEntity().ID})
	assert.NoError(t, err)
	assert.False(t, ex)
}

func TestMemoryRepository_Save(t *testing.T) {
	t.Parallel()

	t.Run("save", func(t *testing.T) {
		t.Parallel()
		repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()

		entity := testdata.NewEntity()
		err := repo.Save(ctx, entity)
		assert.NoError(t, err)

		c, _ := repo.Count(ctx)
		assert.Equal(t, 1, c)
	})

	t.Run("save multiple times", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()

		entity := testdata.NewEntity()
		err := repo.Save(ctx, entity)
		assert.NoError(t, err)

		entity.Name = "new-name"
		err = repo.Save(ctx, entity)
		assert.NoError(t, err)

		c, _ := repo.Count(ctx)
		assert.Equal(t, 1, c)

		e, _ := repo.FindByID(ctx, entity.ID)
		assert.Equal(t, entity, e)
	})

	t.Run("missing id", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()

		err := repo.Save(ctx, testdata.Entity{})
		assert.ErrorIs(t, err, tests.ErrSaveFailed)
	})
}

func TestMemoryRepository_UpdateAll(t *testing.T) {
	t.Parallel()

	t.Run("save all", func(t *testing.T) {
		t.Parallel()

		e := testdata.NewEntity()
		repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()
		repo.Create(ctx, e)

		err := repo.UpdateAll(ctx, []testdata.Entity{e, testdata.NewEntity(), testdata.NewEntity()})
		assert.NoError(t, err)

		c, _ := repo.Count(ctx)
		assert.Equal(t, 3, c)
	})

	t.Run("missing id", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()

		err := repo.UpdateAll(ctx, []testdata.Entity{testdata.NewEntity(), testdata.NewEntity(), {}, testdata.NewEntity()})
		assert.ErrorIs(t, err, tests.ErrSaveFailed)

		empty, _ := repo.IsEmpty(ctx)
		assert.True(t, empty)
	})
}

func TestMemoryRepository_AddAll(t *testing.T) {
	t.Parallel()

	t.Run("add all", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()

		repo.AddAll(ctx, []testdata.Entity{testdata.NewEntity(), testdata.DefaultEntity, testdata.NewEntity()})

		c, _ := repo.Count(ctx)
		assert.Equal(t, 3, c)
	})

	t.Run("entity already exists", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()
		repo.Save(ctx, testdata.DefaultEntity)

		err := repo.AddAll(ctx, []testdata.Entity{testdata.NewEntity(), testdata.DefaultEntity, testdata.NewEntity()})
		assert.ErrorIs(t, err, tests.ErrAlreadyExists)

		c, _ := repo.Count(ctx)
		assert.Equal(t, 1, c)
	})
}

func TestMemoryRepository_Length(t *testing.T) {
	t.Parallel()

	repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()

	count, err := repo.Length(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "new repository should be empty")

	repo.Create(ctx, testdata.NewEntity())
	repo.Create(ctx, testdata.NewEntity())

	count, err = repo.Length(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 2, count, "should have two entities")
}

func TestMemoryRepository_DeleteByID(t *testing.T) {
	t.Parallel()

	repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()
	repo.Create(ctx, testdata.DefaultEntity)

	err := repo.DeleteByID(ctx, testdata.DefaultEntity.ID)
	assert.NoError(t, err)

	empty, _ := repo.Empty(ctx)
	assert.True(t, empty)
}

func TestMemoryRepository_DeleteByIDs(t *testing.T) {
	t.Parallel()

	e0 := testdata.NewEntity()
	e1 := testdata.NewEntity()
	e2 := testdata.NewEntity()
	repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()
	repo.AddAll(ctx, []testdata.Entity{e0, e1, e2})

	err := repo.DeleteByIDs(ctx, []testdata.EntityID{e0.ID, e2.ID})
	assert.NoError(t, err)

	c, _ := repo.Count(ctx)
	assert.Equal(t, 1, c)
}

func TestMemoryRepository_DeleteAll(t *testing.T) {
	t.Parallel()

	repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()
	repo.Create(ctx, testdata.DefaultEntity)

	err := repo.DeleteAll(ctx)
	assert.NoError(t, err)

	c, _ := repo.Count(ctx)
	assert.Equal(t, 0, c)
}

func TestMemoryRepository_Clear(t *testing.T) {
	t.Parallel()

	repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()
	repo.Create(ctx, testdata.DefaultEntity)

	err := repo.Clear(ctx)
	assert.NoError(t, err)

	c, _ := repo.Count(ctx)
	assert.Equal(t, 0, c)
}

func TestMemoryRepository_Concurrently(t *testing.T) {
	t.Parallel()

	const workers = 1000

	wg := sync.WaitGroup{}
	repo := tests.NewMemoryRepository[testdata.Entity, testdata.EntityID]()

	wg.Add(workers) //nolint:wsl
	for i := 0; i < workers; i++ {
		go func() {
			repo.Add(ctx, testdata.NewEntity())
			wg.Done()
		}()
	}

	wg.Add(workers) //nolint:wsl
	for i := 0; i < workers; i++ {
		go func() {
			repo.Read(ctx, testdata.NewEntity().ID)
			wg.Done()
		}()
	}

	wg.Wait()
}
