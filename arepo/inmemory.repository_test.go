package arepo_test

import (
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/go-arrower/arrower/arepo"
	"github.com/go-arrower/arrower/arepo/testdata"
	"github.com/stretchr/testify/assert"
)

func TestMemoryRepository(t *testing.T) {
	t.Parallel()

	arepo.TestSuite(t,
		func(opts ...arepo.Option) arepo.Repository[testdata.Entity, testdata.EntityID] {
			return arepo.NewMemoryRepository[testdata.Entity, testdata.EntityID](opts...)
		},
		func(opts ...arepo.Option) arepo.Repository[testdata.EntityWithNamePK, string] {
			return arepo.NewMemoryRepository[testdata.EntityWithNamePK, string](opts...)
		},
		func(opts ...arepo.Option) arepo.Repository[testdata.EntityWithIntPK, testdata.EntityIDInt] {
			return arepo.NewMemoryRepository[testdata.EntityWithIntPK, testdata.EntityIDInt](opts...)
		},
	)

	t.Run("With Store: always success", func(t *testing.T) {
		t.Parallel()

		// Ensure that the store is called with the right filename.
		// This test is testing the happy path: store and load operations always work

		arepo.TestSuite(t,
			func(opts ...arepo.Option) arepo.Repository[testdata.Entity, testdata.EntityID] {
				opts = append(opts, arepo.WithStore(testStoreSuccessEntity(t, "Entity.json")))
				return arepo.NewMemoryRepository[testdata.Entity, testdata.EntityID](opts...)
			},
			func(opts ...arepo.Option) arepo.Repository[testdata.EntityWithNamePK, string] {
				return arepo.NewMemoryRepository[testdata.EntityWithNamePK, string](opts...)
			},
			func(opts ...arepo.Option) arepo.Repository[testdata.EntityWithIntPK, testdata.EntityIDInt] {
				opts = append(opts, arepo.WithStore(testStoreSuccessEntity(t, "EntityWithIntPK.json")))
				return arepo.NewMemoryRepository[testdata.EntityWithIntPK, testdata.EntityIDInt](opts...)
			},
		)
	})

	t.Run("With Store: load fails", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			arepo.NewMemoryRepository[testdata.Entity, testdata.EntityID](arepo.WithStore(testStoreLoadFails()))
		})
	})

	t.Run("With Store: store fails", func(t *testing.T) {
		t.Parallel()

		t.Run("Create", func(t *testing.T) {
			t.Parallel()

			repo := arepo.NewMemoryRepository[testdata.Entity, testdata.EntityID](arepo.WithStore(testStoreStoreFails(0)))

			err := repo.Create(ctx, testdata.DefaultEntity)
			assert.Error(t, err)

			c, err := repo.Count(ctx)
			assert.NoError(t, err)
			assert.Equal(t, 0, c, "create should not happen if store fails")
		})

		t.Run("Update", func(t *testing.T) {
			t.Parallel()

			repo := arepo.NewMemoryRepository[testdata.Entity, testdata.EntityID](arepo.WithStore(testStoreStoreFails(1)))

			entity := testdata.RandomEntity()
			oldEntity := entity
			err := repo.Create(ctx, entity)
			assert.NoError(t, err)

			entity.Name = gofakeit.Name()
			err = repo.Update(ctx, entity)
			assert.Error(t, err)

			entity, err = repo.FindByID(ctx, entity.ID)
			assert.NoError(t, err)
			assert.Equal(t, oldEntity, entity, "update should not happen if store fails")
		})

		t.Run("UpdateAll", func(t *testing.T) {
			t.Parallel()

			repo := arepo.NewMemoryRepository[testdata.Entity, testdata.EntityID](arepo.WithStore(testStoreStoreFails(1)))
			err := repo.AddAll(ctx, []testdata.Entity{testdata.RandomEntity(), testdata.RandomEntity()})
			assert.NoError(t, err)

			err = repo.UpdateAll(ctx, []testdata.Entity{testdata.RandomEntity(), testdata.RandomEntity()})
			assert.Error(t, err)

			c, err := repo.Count(ctx)
			assert.NoError(t, err)
			assert.Equal(t, 2, c)
		})

		t.Run("Delete", func(t *testing.T) {
			t.Parallel()

			repo := arepo.NewMemoryRepository[testdata.Entity, testdata.EntityID](arepo.WithStore(testStoreStoreFails(1)))
			err := repo.Add(ctx, testdata.DefaultEntity)
			assert.NoError(t, err)

			err = repo.Delete(ctx, testdata.DefaultEntity)
			assert.Error(t, err)

			ex, err := repo.ExistsByID(ctx, testdata.DefaultEntity.ID)
			assert.NoError(t, err)
			assert.True(t, ex, "delete should not happen if store fails")
		})

		t.Run("Save", func(t *testing.T) {
			t.Parallel()

			repo := arepo.NewMemoryRepository[testdata.Entity, testdata.EntityID](arepo.WithStore(testStoreStoreFails(0)))

			err := repo.Save(ctx, testdata.RandomEntity())
			assert.Error(t, err)

			c, err := repo.Count(ctx)
			assert.NoError(t, err)
			assert.Equal(t, 0, c, "save should not happen if store fails")
		})

		t.Run("SaveAll", func(t *testing.T) {
			t.Parallel()

			e0 := testdata.RandomEntity()
			oe0 := e0
			e1 := testdata.RandomEntity()
			oe1 := e1
			repo := arepo.NewMemoryRepository[testdata.Entity, testdata.EntityID](arepo.WithStore(testStoreStoreFails(1)))
			err := repo.SaveAll(ctx, []testdata.Entity{e0, e1})
			assert.NoError(t, err)

			e0.Name = gofakeit.Name()
			e1.Name = gofakeit.Name()
			err = repo.SaveAll(ctx, []testdata.Entity{e0, e1})
			assert.Error(t, err)

			entity, err := repo.FindByID(ctx, e0.ID)
			assert.NoError(t, err)
			assert.Equal(t, oe0, entity, "update should not happen if store fails")
			entity, err = repo.FindByID(ctx, e1.ID)
			assert.NoError(t, err)
			assert.Equal(t, oe1, entity, "update should not happen if store fails")
		})

		t.Run("DeleteByIDs", func(t *testing.T) {
			t.Parallel()

			e0 := testdata.RandomEntity()
			e1 := testdata.RandomEntity()
			e2 := testdata.RandomEntity()
			repo := arepo.NewMemoryRepository[testdata.Entity, testdata.EntityID](arepo.WithStore(testStoreStoreFails(1)))
			err := repo.AddAll(ctx, []testdata.Entity{e0, e1, e2})
			assert.NoError(t, err)

			err = repo.DeleteByIDs(ctx, []testdata.EntityID{e0.ID, e2.ID})
			assert.Error(t, err)

			c, err := repo.Count(ctx)
			assert.NoError(t, err)
			assert.Equal(t, 3, c, "delete should not happen if store fails")
		})

		t.Run("DeleteAll", func(t *testing.T) {
			t.Parallel()

			repo := arepo.NewMemoryRepository[testdata.Entity, testdata.EntityID](arepo.WithStore(testStoreStoreFails(1)))
			err := repo.Create(ctx, testdata.DefaultEntity)
			assert.NoError(t, err)

			err = repo.DeleteAll(ctx)
			assert.Error(t, err)

			c, err := repo.Count(ctx)
			assert.NoError(t, err)
			assert.Equal(t, 1, c, "delete should not happen if store fails")
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

		repo := arepo.NewMemoryRepository[entity, entityUint](arepo.WithIDField("UintID"))

		id, err := repo.NextID(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, id)

		err = repo.Save(ctx, entity{UintID: 1337, Name: gofakeit.Name()})
		assert.NoError(t, err)
	})
}
