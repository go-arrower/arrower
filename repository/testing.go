package repository

import (
	"context"
	"sync"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/repository/testdata"
)

func TestSuite(
	t *testing.T,
	newEntityRepo func(opts ...Option) Repository[testdata.Entity, testdata.EntityID],
	newEntityRepoInt func(opts ...Option) Repository[testdata.EntityWithIntPK, testdata.EntityIDInt],
	newEntityWithoutIDRepo func(opts ...Option) Repository[testdata.EntityWithoutID, testdata.EntityID],
) { //nolint:tparallel // t.Parallel can only be called ones! The caller decides
	t.Helper()

	if newEntityRepo == nil {
		t.Fatal("entity constructor is nil")
	}

	if newEntityWithoutIDRepo == nil {
		t.Fatal("entity without id constructor is nil")
	}

	ctx := context.Background()

	t.Run("new", func(t *testing.T) {
		t.Parallel()

		repo := newEntityRepo()
		assert.NotNil(t, repo)
	})

	t.Run("IntPK", func(t *testing.T) {
		t.Parallel()

		repo := newEntityRepoInt()

		t.Run("generate IDs of int type", func(t *testing.T) {
			t.Parallel()
			id, _ := repo.NextID(ctx)
			t.Log(id)

			id, _ = repo.NextID(ctx)
			t.Log(id)

			id, _ = repo.NextID(ctx)
			t.Log(id)
			assert.Equal(t, testdata.EntityIDInt(3), id)
		})

		t.Run("access the structs ID of int field properly", func(t *testing.T) {
			t.Parallel()

			err := repo.Save(ctx, testdata.EntityWithIntPK{ID: 1337, Name: gofakeit.Name()})
			assert.NoError(t, err)

			c, _ := repo.Count(ctx)
			assert.Equal(t, 1, c)
		})
	})

	t.Run("entity without ID", func(t *testing.T) {
		t.Parallel()

		t.Run("fails", func(t *testing.T) {
			t.Parallel()

			repo := newEntityWithoutIDRepo()

			assert.Panics(t, func() {
				repo.Save(ctx, testdata.EntityWithoutID{})
			})
		})

		t.Run("with ID field option", func(t *testing.T) {
			t.Parallel()

			repo := newEntityWithoutIDRepo(
				WithIDField("Name"),
			)

			err := repo.Save(ctx, testdata.EntityWithoutID{Name: gofakeit.Name()})
			assert.NoError(t, err)
		})
	})

	t.Run("NextID", func(t *testing.T) {
		t.Parallel()

		t.Run("string", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()
			id, err := repo.NextID(ctx)
			assert.NoError(t, err)
			assert.NotEmpty(t, id)

			t.Log(id)
		})

		t.Run("int", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepoInt()
			id, err := repo.NextID(ctx)
			assert.NoError(t, err)
			assert.NotEmpty(t, id)

			t.Log(id)
		})
	})

	t.Run("Create", func(t *testing.T) {
		t.Parallel()

		t.Run("create", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()
			err := repo.Create(ctx, testdata.DefaultEntity)
			assert.NoError(t, err)

			got, err := repo.Read(ctx, testdata.DefaultEntity.ID)
			assert.NoError(t, err)
			assert.Equal(t, testdata.DefaultEntity, got)
		})

		t.Run("create same again", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()

			err := repo.Create(ctx, testdata.DefaultEntity)
			assert.NoError(t, err)

			err = repo.Create(ctx, testdata.DefaultEntity)
			assert.ErrorIs(t, err, ErrAlreadyExists, "already exists")
		})

		t.Run("missing id", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()

			err := repo.Create(ctx, testdata.Entity{})
			assert.ErrorIs(t, err, ErrSaveFailed)
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Parallel()

		t.Run("update", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()
			entity := testdata.TestEntity()
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

			repo := newEntityRepo()

			err := repo.Update(ctx, testdata.DefaultEntity)
			assert.ErrorIs(t, err, ErrSaveFailed)
		})

		t.Run("missing id", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()
			repo.Create(ctx, testdata.DefaultEntity)

			err := repo.Update(ctx, testdata.Entity{})
			assert.ErrorIs(t, err, ErrSaveFailed)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Parallel()

		t.Run("delete", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()
			repo.Add(ctx, testdata.DefaultEntity)

			err := repo.Delete(ctx, testdata.DefaultEntity)
			assert.NoError(t, err)

			e, err := repo.FindByID(ctx, testdata.DefaultEntity.ID)
			assert.ErrorIs(t, err, ErrNotFound)
			assert.Empty(t, e)
		})

		t.Run("multiple delete", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()
			repo.Create(ctx, testdata.DefaultEntity)

			err := repo.Delete(ctx, testdata.DefaultEntity)
			assert.NoError(t, err)

			err = repo.Delete(ctx, testdata.DefaultEntity)
			assert.NoError(t, err)
		})
	})

	t.Run("All", func(t *testing.T) {
		t.Parallel()

		repo := newEntityRepo()

		all, err := repo.All(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, all)
		assert.Empty(t, all, "new repository should be empty")

		repo.Create(ctx, testdata.TestEntity())
		repo.Create(ctx, testdata.TestEntity())

		all, err = repo.All(ctx)
		assert.NoError(t, err)
		assert.Len(t, all, 2, "should have two entities")
	})

	t.Run("AllByIDs", func(t *testing.T) {
		t.Parallel()

		e0 := testdata.TestEntity()
		e1 := testdata.TestEntity()
		repo := newEntityRepo()
		repo.Create(ctx, e0)
		repo.Create(ctx, testdata.TestEntity())
		repo.Create(ctx, e1)
		repo.Create(ctx, testdata.TestEntity())

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
		assert.ErrorIs(t, err, ErrNotFound, "should not find all ids")
		assert.Empty(t, all)
	})

	t.Run("FindByID", func(t *testing.T) {
		t.Parallel()

		repo := newEntityRepo()

		e, err := repo.FindByID(ctx, testdata.TestEntity().ID)
		assert.ErrorIs(t, err, ErrNotFound)
		assert.Empty(t, e)
	})

	t.Run("Contains", func(t *testing.T) {
		t.Parallel()

		repo := newEntityRepo()

		ex, err := repo.ContainsID(ctx, testdata.EntityID(uuid.New().String()))
		assert.NoError(t, err)
		assert.False(t, ex, "id should not exist")

		repo.Create(ctx, testdata.DefaultEntity)

		ex, err = repo.Contains(ctx, testdata.DefaultEntity.ID)
		assert.NoError(t, err)
		assert.True(t, ex, "id should exist")
	})

	t.Run("ContainsAll", func(t *testing.T) {
		t.Parallel()

		e0 := testdata.TestEntity()
		e1 := testdata.TestEntity()
		repo := newEntityRepo()
		repo.Create(ctx, e0)
		repo.Create(ctx, testdata.TestEntity())
		repo.Create(ctx, e1)
		repo.Create(ctx, testdata.TestEntity())

		ex, err := repo.ContainsAll(ctx, nil)
		assert.NoError(t, err)
		assert.False(t, ex)

		ex, err = repo.ContainsAll(ctx, []testdata.EntityID{})
		assert.NoError(t, err)
		assert.False(t, ex)

		ex, err = repo.ContainsAll(ctx, []testdata.EntityID{e0.ID, e1.ID})
		assert.NoError(t, err)
		assert.True(t, ex)

		ex, err = repo.ContainsAll(ctx, []testdata.EntityID{testdata.TestEntity().ID})
		assert.NoError(t, err)
		assert.False(t, ex)
	})

	t.Run("Save", func(t *testing.T) {
		t.Run("save", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()

			entity := testdata.TestEntity()
			err := repo.Save(ctx, entity)
			assert.NoError(t, err)

			c, _ := repo.Count(ctx)
			assert.Equal(t, 1, c)
		})

		t.Run("save multiple times", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()

			entity := testdata.TestEntity()
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

			repo := newEntityRepo()

			err := repo.Save(ctx, testdata.Entity{})
			assert.ErrorIs(t, err, ErrSaveFailed)
		})
	})

	t.Run("UpdateAll", func(t *testing.T) {
		t.Run("save all", func(t *testing.T) {
			t.Parallel()

			e := testdata.TestEntity()
			repo := newEntityRepo()
			repo.Create(ctx, e)

			err := repo.UpdateAll(ctx, []testdata.Entity{e, testdata.TestEntity(), testdata.TestEntity()})
			assert.NoError(t, err)

			c, _ := repo.Count(ctx)
			assert.Equal(t, 3, c)
		})

		t.Run("missing id", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()

			err := repo.UpdateAll(ctx, []testdata.Entity{testdata.TestEntity(), testdata.TestEntity(), {}, testdata.TestEntity()})
			assert.ErrorIs(t, err, ErrSaveFailed)

			empty, _ := repo.IsEmpty(ctx)
			assert.True(t, empty)
		})
	})

	t.Run("AddAll", func(t *testing.T) {
		t.Run("add all", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()

			repo.AddAll(ctx, []testdata.Entity{testdata.TestEntity(), testdata.DefaultEntity, testdata.TestEntity()})

			c, _ := repo.Count(ctx)
			assert.Equal(t, 3, c)
		})

		t.Run("entity already exists", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()
			repo.Save(ctx, testdata.DefaultEntity)

			err := repo.AddAll(ctx, []testdata.Entity{testdata.TestEntity(), testdata.DefaultEntity, testdata.TestEntity()})
			assert.ErrorIs(t, err, ErrAlreadyExists)

			c, _ := repo.Count(ctx)
			assert.Equal(t, 1, c)
		})
	})

	t.Run("Length", func(t *testing.T) {
		t.Parallel()

		repo := newEntityRepo()

		count, err := repo.Length(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 0, count, "new repository should be empty")

		repo.Create(ctx, testdata.TestEntity())
		repo.Create(ctx, testdata.TestEntity())

		count, err = repo.Length(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 2, count, "should have two entities")
	})

	t.Run("DeleteByID", func(t *testing.T) {
		t.Parallel()

		repo := newEntityRepo()
		repo.Create(ctx, testdata.DefaultEntity)

		err := repo.DeleteByID(ctx, testdata.DefaultEntity.ID)
		assert.NoError(t, err)

		empty, _ := repo.Empty(ctx)
		assert.True(t, empty)
	})

	t.Run("DeleteByIDs", func(t *testing.T) {
		t.Parallel()

		t.Run("delete", func(t *testing.T) {
			t.Parallel()

			e0 := testdata.TestEntity()
			e1 := testdata.TestEntity()
			e2 := testdata.TestEntity()
			repo := newEntityRepo()
			repo.AddAll(ctx, []testdata.Entity{e0, e1, e2})

			err := repo.DeleteByIDs(ctx, []testdata.EntityID{e0.ID, e2.ID})
			assert.NoError(t, err)

			c, _ := repo.Count(ctx)
			assert.Equal(t, 1, c)
		})
	})

	t.Run("DeleteAll", func(t *testing.T) {
		t.Parallel()

		// todo this was a subtest because the orig. test also checked for failing store => unnest the test (?)
		t.Run("delete all", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()
			repo.Create(ctx, testdata.DefaultEntity)

			err := repo.DeleteAll(ctx)
			assert.NoError(t, err)

			c, _ := repo.Count(ctx)
			assert.Equal(t, 0, c)
		})
	})

	t.Run("Clear", func(t *testing.T) {
		t.Parallel()

		repo := newEntityRepo()
		repo.Create(ctx, testdata.DefaultEntity)

		err := repo.Clear(ctx)
		assert.NoError(t, err)

		c, _ := repo.Count(ctx)
		assert.Equal(t, 0, c)
	})

	t.Run("Concurrently", func(t *testing.T) {
		t.Parallel()

		const workers = 1000

		wg := sync.WaitGroup{}
		repo := newEntityRepo()

		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func() {
				repo.Add(ctx, testdata.TestEntity())
				wg.Done()
			}()
		}

		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func() {
				repo.Read(ctx, testdata.TestEntity().ID)
				wg.Done()
			}()
		}

		wg.Wait()
	})
}
