package repository_test

import (
	"testing"

	"github.com/go-arrower/arrower/repository/testdata"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/repository"
)

func TestInMemoryRepository(t *testing.T) {
	t.Parallel()

	repository.TestSuite(t,
		func(opts ...repository.Option) repository.Repository[testdata.Entity, testdata.EntityID] {
			return repository.NewMemoryRepository[testdata.Entity, testdata.EntityID](opts...)
		},
		func(opts ...repository.Option) repository.Repository[testdata.EntityWithIntPK, testdata.EntityIDInt] {
			return repository.NewMemoryRepository[testdata.EntityWithIntPK, testdata.EntityIDInt](opts...)
		},
		func(opts ...repository.Option) repository.Repository[testdata.EntityWithoutID, testdata.EntityID] {
			return repository.NewMemoryRepository[testdata.EntityWithoutID, testdata.EntityID](opts...)
		},
	)

	// if the TestSuite is run again with a store, does this cover all cases and this test files does not have to do many store related tests?
}

func TestNewMemoryRepository(t *testing.T) {
	t.Parallel()

	t.Run("load from store", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[testdata.Entity, testdata.EntityID](repository.WithStore(testStoreSuccessEntity(t)))
		assert.NotNil(t, repo)
	})

	t.Run("load from store fails", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			repository.NewMemoryRepository[testdata.Entity, testdata.EntityID](repository.WithStore(testStoreLoadFails()))
		})
	})
}

func TestNewMemoryRepository_IntPK(t *testing.T) {
	t.Parallel()

	type (
		entityUint uint
		entity     struct {
			UintID entityUint
			Name   string
		}
	)

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

func TestMemoryRepository_Create(t *testing.T) {
	t.Parallel()

	t.Run("create", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[testdata.Entity, testdata.EntityID](repository.WithStore(testStoreSuccessEntity(t)))
		err := repo.Create(ctx, testdata.DefaultEntity)
		assert.NoError(t, err)

		got, err := repo.Read(ctx, testdata.DefaultEntity.ID)
		assert.NoError(t, err)
		assert.Equal(t, testdata.DefaultEntity, got)
	})

	t.Run("store fails", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[testdata.Entity, testdata.EntityID](repository.WithStore(testStoreStoreFails()))

		err := repo.Create(ctx, testdata.DefaultEntity)
		assert.Error(t, err)
	})
}

func TestMemoryRepository_Update(t *testing.T) {
	t.Parallel()

	t.Run("update", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[testdata.Entity, testdata.EntityID](repository.WithStore(testStoreSuccessEntity(t)))
		entity := testdata.TestEntity()
		repo.Create(ctx, entity)

		entity.Name = gofakeit.Name()
		err := repo.Update(ctx, entity)
		assert.NoError(t, err)

		e, err := repo.FindByID(ctx, entity.ID)
		assert.NoError(t, err)
		assert.Equal(t, entity, e)
	})

	t.Run("store fails", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[testdata.Entity, testdata.EntityID](repository.WithStore(testStoreStoreFails()))

		entity := testdata.TestEntity()
		repo.Create(ctx, entity)

		entity.Name = gofakeit.Name()
		err := repo.Update(ctx, entity)
		assert.Error(t, err)
		// todo assert the old name is still valid after a read?
	})
}

func TestMemoryRepository_Delete(t *testing.T) {
	t.Parallel()

	t.Run("delete", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[testdata.Entity, testdata.EntityID](repository.WithStore(testStoreSuccessEntity(t)))
		repo.Add(ctx, testdata.DefaultEntity)

		err := repo.Delete(ctx, testdata.DefaultEntity)
		assert.NoError(t, err)

		e, err := repo.FindByID(ctx, testdata.DefaultEntity.ID)
		assert.ErrorIs(t, err, repository.ErrNotFound)
		assert.Empty(t, e)
	})

	t.Run("multiple delete", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[testdata.Entity, testdata.EntityID](repository.WithStore(testStoreSuccessEntity(t)))
		repo.Create(ctx, testdata.DefaultEntity)

		err := repo.Delete(ctx, testdata.DefaultEntity)
		assert.NoError(t, err)

		err = repo.Delete(ctx, testdata.DefaultEntity)
		assert.NoError(t, err)
	})

	t.Run("store fails", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[testdata.Entity, testdata.EntityID](repository.WithStore(testStoreStoreFails()))
		repo.Add(ctx, testdata.DefaultEntity)

		err := repo.Delete(ctx, testdata.DefaultEntity)
		assert.Error(t, err)
	})
}

func TestMemoryRepository_Save(t *testing.T) {
	t.Parallel()

	t.Run("save", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[testdata.Entity, testdata.EntityID](repository.WithStore(testStoreSuccessEntity(t)))

		entity := testdata.TestEntity()
		err := repo.Save(ctx, entity)
		assert.NoError(t, err)

		c, _ := repo.Count(ctx)
		assert.Equal(t, 1, c)
	})

	t.Run("store fails", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[testdata.Entity, testdata.EntityID](repository.WithStore(testStoreStoreFails()))

		err := repo.Save(ctx, testdata.TestEntity())
		assert.Error(t, err)
	})
}

func TestMemoryRepository_UpdateAll(t *testing.T) {
	t.Parallel()

	t.Run("save all", func(t *testing.T) {
		t.Parallel()

		e := testdata.TestEntity()
		repo := repository.NewMemoryRepository[testdata.Entity, testdata.EntityID](repository.WithStore(testStoreSuccessEntity(t)))
		repo.Create(ctx, e)

		err := repo.UpdateAll(ctx, []testdata.Entity{e, testdata.TestEntity(), testdata.TestEntity()})
		assert.NoError(t, err)

		c, _ := repo.Count(ctx)
		assert.Equal(t, 3, c)
	})

	t.Run("store fails", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[testdata.Entity, testdata.EntityID](repository.WithStore(testStoreStoreFails()))

		err := repo.UpdateAll(ctx, []testdata.Entity{testdata.TestEntity(), testdata.TestEntity()})
		assert.Error(t, err)
	})
}

func TestMemoryRepository_DeleteByIDs(t *testing.T) {
	t.Parallel()

	t.Run("delete", func(t *testing.T) {
		t.Parallel()

		e0 := testdata.TestEntity()
		e1 := testdata.TestEntity()
		e2 := testdata.TestEntity()
		repo := repository.NewMemoryRepository[testdata.Entity, testdata.EntityID](repository.WithStore(testStoreSuccessEntity(t)))
		repo.AddAll(ctx, []testdata.Entity{e0, e1, e2})

		err := repo.DeleteByIDs(ctx, []testdata.EntityID{e0.ID, e2.ID})
		assert.NoError(t, err)

		c, _ := repo.Count(ctx)
		assert.Equal(t, 1, c)
	})

	t.Run("store fails", func(t *testing.T) {
		t.Parallel()

		e0 := testdata.TestEntity()
		e1 := testdata.TestEntity()
		e2 := testdata.TestEntity()
		repo := repository.NewMemoryRepository[testdata.Entity, testdata.EntityID](repository.WithStore(testStoreStoreFails()))
		repo.AddAll(ctx, []testdata.Entity{e0, e1, e2})

		err := repo.DeleteByIDs(ctx, []testdata.EntityID{e0.ID, e2.ID})
		assert.Error(t, err)
	})
}

func TestMemoryRepository_DeleteAll(t *testing.T) {
	t.Parallel()

	t.Run("delete all", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[testdata.Entity, testdata.EntityID](repository.WithStore(testStoreSuccessEntity(t)))
		repo.Create(ctx, testdata.DefaultEntity)

		err := repo.DeleteAll(ctx)
		assert.NoError(t, err)

		c, _ := repo.Count(ctx)
		assert.Equal(t, 0, c)
	})

	t.Run("store fails", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository[testdata.Entity, testdata.EntityID](repository.WithStore(testStoreStoreFails()))
		repo.Create(ctx, testdata.DefaultEntity)

		err := repo.DeleteAll(ctx)
		assert.Error(t, err)
	})
}
