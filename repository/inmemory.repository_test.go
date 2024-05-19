package repository_test

import (
	"sync"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/repository"
)

func TestNewMemoryRepository(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[Entity, EntityID]()
		assert.NotNil(t, repo)
	})

	t.Run("load from store", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[Entity, EntityID](repository.WithStore(testStoreSuccessEntity(t)))
		assert.NotNil(t, repo)
	})

	t.Run("load from store fails", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			repository.NewMemoryRepository[Entity, EntityID](repository.WithStore(testStoreLoadFails()))
		})
	})
}

func TestNewMemoryRepository_IntPK(t *testing.T) {
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

	repo := repository.NewMemoryRepository[entity, entityInt](repository.WithIDField("IntID"))

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

		repo := repository.NewMemoryRepository[entity, entityUint](repository.WithIDField("UintID"))

		id, err := repo.NextID(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, id)

		err = repo.Save(ctx, entity{UintID: 1337, Name: gofakeit.Name()})
		assert.NoError(t, err)
	})
}

func TestEntityWithoutID(t *testing.T) {
	t.Parallel()

	repo := repository.NewMemoryRepository[EntityWithoutID, EntityID]()

	assert.Panics(t, func() {
		repo.Save(ctx, EntityWithoutID{})
	})
}

func TestWithIDField(t *testing.T) {
	t.Parallel()

	repo := repository.NewMemoryRepository[EntityWithoutID, string](
		repository.WithIDField("Name"),
	)

	err := repo.Save(ctx, EntityWithoutID{Name: gofakeit.Name()})
	assert.NoError(t, err)
}

func TestMemoryRepository_NextID(t *testing.T) {
	t.Parallel()

	t.Run("string", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[Entity, EntityID]()
		id, err := repo.NextID(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, id)

		t.Log(id)
	})

	t.Run("int", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[Entity, int]()
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

		repo := repository.NewMemoryRepository[Entity, EntityID](repository.WithStore(testStoreSuccessEntity(t)))
		err := repo.Create(ctx, defaultEntity)
		assert.NoError(t, err)

		got, err := repo.Read(ctx, defaultEntity.ID)
		assert.NoError(t, err)
		assert.Equal(t, defaultEntity, got)
	})

	t.Run("create same again", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[Entity, EntityID]()

		err := repo.Create(ctx, defaultEntity)
		assert.NoError(t, err)

		err = repo.Create(ctx, defaultEntity)
		assert.ErrorIs(t, err, repository.ErrAlreadyExists, "already exists")
	})

	t.Run("missing id", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[Entity, EntityID]()

		err := repo.Create(ctx, Entity{})
		assert.ErrorIs(t, err, repository.ErrSaveFailed)
	})

	t.Run("store fails", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[Entity, EntityID](repository.WithStore(testStoreStoreFails()))

		err := repo.Create(ctx, defaultEntity)
		assert.Error(t, err)
	})
}

func TestMemoryRepository_Update(t *testing.T) {
	t.Parallel()

	t.Run("update", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[Entity, EntityID](repository.WithStore(testStoreSuccessEntity(t)))
		entity := testEntity()
		repo.Create(ctx, entity)

		entity.Name = gofakeit.Name()
		err := repo.Update(ctx, entity)
		assert.NoError(t, err)

		e, err := repo.FindByID(ctx, entity.ID)
		assert.NoError(t, err)
		assert.Equal(t, entity, e)
	})

	t.Run("does not exist yet", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[Entity, EntityID]()

		err := repo.Update(ctx, defaultEntity)
		assert.ErrorIs(t, err, repository.ErrSaveFailed)
	})

	t.Run("missing id", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[Entity, EntityID]()
		repo.Create(ctx, defaultEntity)

		err := repo.Update(ctx, Entity{})
		assert.ErrorIs(t, err, repository.ErrSaveFailed)
	})

	t.Run("store fails", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[Entity, EntityID](repository.WithStore(testStoreStoreFails()))

		entity := testEntity()
		repo.Create(ctx, entity)

		entity.Name = gofakeit.Name()
		err := repo.Update(ctx, entity)
		assert.Error(t, err)
	})
}

func TestMemoryRepository_Delete(t *testing.T) {
	t.Parallel()

	t.Run("delete", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[Entity, EntityID](repository.WithStore(testStoreSuccessEntity(t)))
		repo.Add(ctx, defaultEntity)

		err := repo.Delete(ctx, defaultEntity)
		assert.NoError(t, err)

		e, err := repo.FindByID(ctx, defaultEntity.ID)
		assert.ErrorIs(t, err, repository.ErrNotFound)
		assert.Empty(t, e)
	})

	t.Run("multiple delete", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[Entity, EntityID](repository.WithStore(testStoreSuccessEntity(t)))
		repo.Create(ctx, defaultEntity)

		err := repo.Delete(ctx, defaultEntity)
		assert.NoError(t, err)

		err = repo.Delete(ctx, defaultEntity)
		assert.NoError(t, err)
	})

	t.Run("store fails", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[Entity, EntityID](repository.WithStore(testStoreStoreFails()))
		repo.Add(ctx, defaultEntity)

		err := repo.Delete(ctx, defaultEntity)
		assert.Error(t, err)
	})
}

func TestMemoryRepository_All(t *testing.T) {
	t.Parallel()

	repo := repository.NewMemoryRepository[Entity, EntityID]()

	all, err := repo.All(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, all)
	assert.Empty(t, all, "new repository should be empty")

	repo.Create(ctx, testEntity())
	repo.Create(ctx, testEntity())

	all, err = repo.All(ctx)
	assert.NoError(t, err)
	assert.Len(t, all, 2, "should have two entities")
}

func TestMemoryRepository_AllByIDs(t *testing.T) {
	t.Parallel()

	e0 := testEntity()
	e1 := testEntity()
	repo := repository.NewMemoryRepository[Entity, EntityID]()
	repo.Create(ctx, e0)
	repo.Create(ctx, testEntity())
	repo.Create(ctx, e1)
	repo.Create(ctx, testEntity())

	all, err := repo.AllByIDs(ctx, nil)
	assert.NoError(t, err)
	assert.Empty(t, all)

	all, err = repo.AllByIDs(ctx, []EntityID{})
	assert.NoError(t, err, "empty list should not return an error")
	assert.Empty(t, all)

	all, err = repo.AllByIDs(ctx, []EntityID{e0.ID, e1.ID})
	assert.NoError(t, err)
	assert.Len(t, all, 2, "should find all ids")

	all, err = repo.AllByIDs(ctx, []EntityID{
		e0.ID,
		EntityID(uuid.New().String()),
	})
	assert.ErrorIs(t, err, repository.ErrNotFound, "should not find all ids")
	assert.Empty(t, all)
}

func TestMemoryRepository_FindByID(t *testing.T) {
	t.Parallel()

	repo := repository.NewMemoryRepository[Entity, EntityID]()

	e, err := repo.FindByID(ctx, testEntity().ID)
	assert.ErrorIs(t, err, repository.ErrNotFound)
	assert.Empty(t, e)
}

func TestMemoryRepository_Contains(t *testing.T) {
	t.Parallel()

	repo := repository.NewMemoryRepository[Entity, EntityID]()

	ex, err := repo.ContainsID(ctx, EntityID(uuid.New().String()))
	assert.NoError(t, err)
	assert.False(t, ex, "id should not exist")

	repo.Create(ctx, defaultEntity)

	ex, err = repo.Contains(ctx, defaultEntity.ID)
	assert.NoError(t, err)
	assert.True(t, ex, "id should exist")
}

func TestMemoryRepository_ContainsAll(t *testing.T) {
	t.Parallel()

	e0 := testEntity()
	e1 := testEntity()
	repo := repository.NewMemoryRepository[Entity, EntityID]()
	repo.Create(ctx, e0)
	repo.Create(ctx, testEntity())
	repo.Create(ctx, e1)
	repo.Create(ctx, testEntity())

	ex, err := repo.ContainsAll(ctx, nil)
	assert.NoError(t, err)
	assert.False(t, ex)

	ex, err = repo.ContainsAll(ctx, []EntityID{})
	assert.NoError(t, err)
	assert.False(t, ex)

	ex, err = repo.ContainsAll(ctx, []EntityID{e0.ID, e1.ID})
	assert.NoError(t, err)
	assert.True(t, ex)

	ex, err = repo.ContainsAll(ctx, []EntityID{testEntity().ID})
	assert.NoError(t, err)
	assert.False(t, ex)
}

func TestMemoryRepository_Save(t *testing.T) {
	t.Parallel()

	t.Run("save", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[Entity, EntityID](repository.WithStore(testStoreSuccessEntity(t)))

		entity := testEntity()
		err := repo.Save(ctx, entity)
		assert.NoError(t, err)

		c, _ := repo.Count(ctx)
		assert.Equal(t, 1, c)
	})

	t.Run("save multiple times", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[Entity, EntityID]()

		entity := testEntity()
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

		repo := repository.NewMemoryRepository[Entity, EntityID]()

		err := repo.Save(ctx, Entity{})
		assert.ErrorIs(t, err, repository.ErrSaveFailed)
	})

	t.Run("store fails", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[Entity, EntityID](repository.WithStore(testStoreStoreFails()))

		err := repo.Save(ctx, testEntity())
		assert.Error(t, err)
	})
}

func TestMemoryRepository_UpdateAll(t *testing.T) {
	t.Parallel()

	t.Run("save all", func(t *testing.T) {
		t.Parallel()

		e := testEntity()
		repo := repository.NewMemoryRepository[Entity, EntityID](repository.WithStore(testStoreSuccessEntity(t)))
		repo.Create(ctx, e)

		err := repo.UpdateAll(ctx, []Entity{e, testEntity(), testEntity()})
		assert.NoError(t, err)

		c, _ := repo.Count(ctx)
		assert.Equal(t, 3, c)
	})

	t.Run("missing id", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[Entity, EntityID]()

		err := repo.UpdateAll(ctx, []Entity{testEntity(), testEntity(), {}, testEntity()})
		assert.ErrorIs(t, err, repository.ErrSaveFailed)

		empty, _ := repo.IsEmpty(ctx)
		assert.True(t, empty)
	})

	t.Run("store fails", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[Entity, EntityID](repository.WithStore(testStoreStoreFails()))

		err := repo.UpdateAll(ctx, []Entity{testEntity(), testEntity()})
		assert.Error(t, err)
	})
}

func TestMemoryRepository_AddAll(t *testing.T) {
	t.Parallel()

	t.Run("add all", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[Entity, EntityID]()

		repo.AddAll(ctx, []Entity{testEntity(), defaultEntity, testEntity()})

		c, _ := repo.Count(ctx)
		assert.Equal(t, 3, c)
	})

	t.Run("entity already exists", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[Entity, EntityID]()
		repo.Save(ctx, defaultEntity)

		err := repo.AddAll(ctx, []Entity{testEntity(), defaultEntity, testEntity()})
		assert.ErrorIs(t, err, repository.ErrAlreadyExists)

		c, _ := repo.Count(ctx)
		assert.Equal(t, 1, c)
	})
}

func TestMemoryRepository_Length(t *testing.T) {
	t.Parallel()

	repo := repository.NewMemoryRepository[Entity, EntityID]()

	count, err := repo.Length(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "new repository should be empty")

	repo.Create(ctx, testEntity())
	repo.Create(ctx, testEntity())

	count, err = repo.Length(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 2, count, "should have two entities")
}

func TestMemoryRepository_DeleteByID(t *testing.T) {
	t.Parallel()

	repo := repository.NewMemoryRepository[Entity, EntityID]()
	repo.Create(ctx, defaultEntity)

	err := repo.DeleteByID(ctx, defaultEntity.ID)
	assert.NoError(t, err)

	empty, _ := repo.Empty(ctx)
	assert.True(t, empty)
}

func TestMemoryRepository_DeleteByIDs(t *testing.T) {
	t.Parallel()

	t.Run("delete", func(t *testing.T) {
		t.Parallel()

		e0 := testEntity()
		e1 := testEntity()
		e2 := testEntity()
		repo := repository.NewMemoryRepository[Entity, EntityID](repository.WithStore(testStoreSuccessEntity(t)))
		repo.AddAll(ctx, []Entity{e0, e1, e2})

		err := repo.DeleteByIDs(ctx, []EntityID{e0.ID, e2.ID})
		assert.NoError(t, err)

		c, _ := repo.Count(ctx)
		assert.Equal(t, 1, c)
	})

	t.Run("store fails", func(t *testing.T) {
		t.Parallel()

		e0 := testEntity()
		e1 := testEntity()
		e2 := testEntity()
		repo := repository.NewMemoryRepository[Entity, EntityID](repository.WithStore(testStoreStoreFails()))
		repo.AddAll(ctx, []Entity{e0, e1, e2})

		err := repo.DeleteByIDs(ctx, []EntityID{e0.ID, e2.ID})
		assert.Error(t, err)
	})
}

func TestMemoryRepository_DeleteAll(t *testing.T) {
	t.Parallel()

	t.Run("delete all", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[Entity, EntityID](repository.WithStore(testStoreSuccessEntity(t)))
		repo.Create(ctx, defaultEntity)

		err := repo.DeleteAll(ctx)
		assert.NoError(t, err)

		c, _ := repo.Count(ctx)
		assert.Equal(t, 0, c)
	})

	t.Run("store fails", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[Entity, EntityID](repository.WithStore(testStoreStoreFails()))
		repo.Create(ctx, defaultEntity)

		err := repo.DeleteAll(ctx)
		assert.Error(t, err)
	})
}

func TestMemoryRepository_Clear(t *testing.T) {
	t.Parallel()

	repo := repository.NewMemoryRepository[Entity, EntityID]()
	repo.Create(ctx, defaultEntity)

	err := repo.Clear(ctx)
	assert.NoError(t, err)

	c, _ := repo.Count(ctx)
	assert.Equal(t, 0, c)
}

func TestMemoryRepository_Concurrently(t *testing.T) {
	t.Parallel()

	const workers = 1000

	wg := sync.WaitGroup{}
	repo := repository.NewMemoryRepository[Entity, EntityID]()

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			repo.Add(ctx, testEntity())
			wg.Done()
		}()
	}

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			repo.Read(ctx, testEntity().ID)
			wg.Done()
		}()
	}

	wg.Wait()
}
