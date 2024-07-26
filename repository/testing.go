package repository

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/repository/testdata"
)

// TestSuite is a suite that ensures a Repository implementation
// adheres to the intended behaviour.
//
//nolint:maintidx,tparallel // t.Parallel can only be called ones! The caller decides
func TestSuite(
	t *testing.T,
	newEntityRepo func(opts ...Option) Repository[testdata.Entity, testdata.EntityID],
	newEntityRepoInt func(opts ...Option) Repository[testdata.EntityWithIntPK, testdata.EntityIDInt],
	newEntityWithoutIDRepo func(opts ...Option) Repository[testdata.EntityWithoutID, testdata.EntityID],
) {
	t.Helper()

	if newEntityRepo == nil {
		t.Fatal("entity constructor is nil")
	}

	if newEntityWithoutIDRepo == nil {
		t.Fatal("entity without id constructor is nil")
	}

	ctx := context.Background()

	//
	// The assertions of the Repository start below.
	// Each implementation must adhere to at least this behaviour.
	//

	t.Run("new", func(t *testing.T) {
		t.Parallel()

		repo := newEntityRepo()
		assert.NotNil(t, repo)
	})

	t.Run("struct without ID field", func(t *testing.T) {
		t.Parallel()

		t.Run("default - fails", func(t *testing.T) {
			t.Parallel()

			repo := newEntityWithoutIDRepo()

			assert.Panics(t, func() {
				repo.Save(ctx, testdata.EntityWithoutID{})
			})
		})

		t.Run("ID field overwrite", func(t *testing.T) {
			t.Parallel()

			name := gofakeit.Name()
			repo := newEntityWithoutIDRepo(WithIDField("Name"))

			err := repo.Save(ctx, testdata.EntityWithoutID{Name: name})
			assert.NoError(t, err)

			got, err := repo.FindByID(ctx, testdata.EntityID(name))
			assert.NoError(t, err)
			assert.Equal(t, name, got.Name)
		})
	})

	t.Run("NextID", func(t *testing.T) {
		t.Parallel()

		t.Run("string", func(t *testing.T) {
			t.Parallel()

			t.Run("get id", func(t *testing.T) {
				t.Parallel()

				repo := newEntityRepo()

				id, err := repo.NextID(ctx)
				assert.NoError(t, err)
				assert.NotEmpty(t, id)
				assert.True(t, reflect.TypeOf(id).ConvertibleTo(reflect.TypeOf("")), "the id should be convertible into a string")
			})

			t.Run("different values", func(t *testing.T) {
				t.Parallel()

				repo := newEntityRepo()

				id0, err := repo.NextID(ctx)
				assert.NoError(t, err)
				assert.True(t, reflect.TypeOf(id0).ConvertibleTo(reflect.TypeOf("")), "the id should be convertible into a string")

				id1, err := repo.NextID(ctx)
				assert.NoError(t, err)
				assert.True(t, reflect.TypeOf(id1).ConvertibleTo(reflect.TypeOf("")), "the id should be convertible into a string")

				id2, err := repo.NextID(ctx)
				assert.NoError(t, err)
				assert.True(t, reflect.TypeOf(id2).ConvertibleTo(reflect.TypeOf("")), "the id should be convertible into a string")

				assert.NotEqual(t, id0, id1, "IDs should be different each time they are generated")
				assert.NotEqual(t, id1, id2, "IDs should be different each time they are generated")
			})

			t.Run("works as pk", func(t *testing.T) {
				t.Parallel()

				repo := newEntityRepo()

				err := repo.Save(ctx, testdata.Entity{ID: "1337", Name: gofakeit.Name()})
				assert.NoError(t, err)

				got, err := repo.FindByID(ctx, "1337")
				assert.NoError(t, err)
				assert.NotEmpty(t, got.Name)
			})
		})

		t.Run("int", func(t *testing.T) {
			t.Parallel()

			t.Run("get id", func(t *testing.T) {
				t.Parallel()

				repo := newEntityRepoInt()

				id, err := repo.NextID(ctx)
				assert.NoError(t, err)
				assert.NotEmpty(t, id)
				assert.True(t, reflect.TypeOf(id).ConvertibleTo(reflect.TypeOf(0)), "the id should be convertible into a int")
			})

			t.Run("different values", func(t *testing.T) {
				t.Parallel()

				repo := newEntityRepoInt()

				id0, err := repo.NextID(ctx)
				assert.NoError(t, err)
				assert.True(t, reflect.TypeOf(id0).ConvertibleTo(reflect.TypeOf(0)), "the id should be convertible into a int")

				id1, err := repo.NextID(ctx)
				assert.NoError(t, err)
				assert.True(t, reflect.TypeOf(id1).ConvertibleTo(reflect.TypeOf(0)), "the id should be convertible into a int")

				id2, err := repo.NextID(ctx)
				assert.NoError(t, err)
				assert.True(t, reflect.TypeOf(id2).ConvertibleTo(reflect.TypeOf(0)), "the id should be convertible into a int")

				assert.NotEqual(t, id0, id1, "IDs should be different each time they are generated")
				assert.NotEqual(t, id1, id2, "IDs should be different each time they are generated")
			})

			t.Run("works as pk", func(t *testing.T) {
				t.Parallel()

				repo := newEntityRepoInt()

				err := repo.Save(ctx, testdata.EntityWithIntPK{ID: 1337, Name: gofakeit.Name()})
				assert.NoError(t, err)

				got, err := repo.FindByID(ctx, 1337)
				assert.NoError(t, err)
				assert.NotEmpty(t, got.Name)
			})
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

		t.Run("same entity again", func(t *testing.T) {
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
			entity := testdata.RandomEntity()
			err := repo.Create(ctx, entity)
			assert.NoError(t, err)

			entity.Name = gofakeit.Name()
			err = repo.Update(ctx, entity)
			assert.NoError(t, err)

			got, err := repo.FindByID(ctx, entity.ID)
			assert.NoError(t, err)
			assert.Equal(t, entity, got)
		})

		t.Run("does not exist yet", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()

			err := repo.Update(ctx, testdata.DefaultEntity)
			assert.ErrorIs(t, err, ErrSaveFailed, "can not update not existing entity")
		})

		t.Run("missing id", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()
			entity := testdata.RandomEntity()
			err := repo.Create(ctx, entity)
			assert.NoError(t, err)

			entity.ID = ""

			err = repo.Update(ctx, entity)
			assert.ErrorIs(t, err, ErrSaveFailed)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Parallel()

		t.Run("delete", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()
			err := repo.Add(ctx, testdata.DefaultEntity)
			assert.NoError(t, err)

			err = repo.Delete(ctx, testdata.DefaultEntity)
			assert.NoError(t, err)

			got, err := repo.FindByID(ctx, testdata.DefaultEntity.ID)
			assert.ErrorIs(t, err, ErrNotFound)
			assert.Empty(t, got)
		})

		t.Run("non-existing", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()

			err := repo.Delete(ctx, testdata.DefaultEntity)
			assert.NoError(t, err, "should succeed if entity does not exist in the repository")
		})
	})

	t.Run("All", func(t *testing.T) {
		t.Parallel()

		repo := newEntityRepo()

		all, err := repo.All(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, all)
		assert.Empty(t, all, "new repository should be empty")

		err = repo.Create(ctx, testdata.RandomEntity())
		assert.NoError(t, err)
		err = repo.Create(ctx, testdata.RandomEntity())
		assert.NoError(t, err)

		all, err = repo.All(ctx)
		assert.NoError(t, err)
		assert.Len(t, all, 2, "should have two entities")
	})

	t.Run("AllByIDs", func(t *testing.T) {
		t.Parallel()

		e0 := testdata.RandomEntity()
		e1 := testdata.RandomEntity()
		repo := newEntityRepo()

		_ = repo.Create(ctx, e0)
		_ = repo.Create(ctx, testdata.RandomEntity())
		_ = repo.Create(ctx, e1)
		_ = repo.Create(ctx, testdata.RandomEntity())

		all, err := repo.AllByIDs(ctx, nil)
		assert.NoError(t, err)
		assert.Empty(t, all, "nil is not matching any IDs")

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

		t.Run("find by id", func(t *testing.T) {
			t.Parallel()

			entity := testdata.RandomEntity()
			repo := newEntityRepo()
			err := repo.Add(ctx, entity)
			assert.NoError(t, err)

			got, err := repo.FindByID(ctx, entity.ID)
			assert.NoError(t, err)
			assert.Equal(t, entity, got)
		})

		t.Run("non-existing", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()

			e, err := repo.FindByID(ctx, testdata.RandomEntity().ID)
			assert.ErrorIs(t, err, ErrNotFound)
			assert.Empty(t, e)
		})
	})

	t.Run("Contains", func(t *testing.T) {
		t.Parallel()

		repo := newEntityRepo()

		ex, err := repo.ContainsID(ctx, testdata.RandomEntity().ID)
		assert.NoError(t, err)
		assert.False(t, ex, "id should not exist")

		err = repo.Create(ctx, testdata.DefaultEntity)
		assert.NoError(t, err)

		ex, err = repo.Contains(ctx, testdata.DefaultEntity.ID)
		assert.NoError(t, err)
		assert.True(t, ex, "id should exist")
	})

	t.Run("ContainsAll", func(t *testing.T) {
		t.Parallel()

		e0 := testdata.RandomEntity()
		e1 := testdata.RandomEntity()
		repo := newEntityRepo()
		_ = repo.Create(ctx, e0)
		_ = repo.Create(ctx, testdata.RandomEntity())
		_ = repo.Create(ctx, e1)
		_ = repo.Create(ctx, testdata.RandomEntity())

		ex, err := repo.ContainsAll(ctx, nil)
		assert.NoError(t, err)
		assert.False(t, ex, "nil is not matching any IDs")

		ex, err = repo.ContainsAll(ctx, []testdata.EntityID{})
		assert.NoError(t, err, "empty list should not return an error")
		assert.False(t, ex)

		ex, err = repo.ContainsAll(ctx, []testdata.EntityID{e0.ID, e1.ID})
		assert.NoError(t, err)
		assert.True(t, ex, "should check all IDs")

		ex, err = repo.ContainsAll(ctx, []testdata.EntityID{e0.ID, testdata.RandomEntity().ID})
		assert.NoError(t, err)
		assert.False(t, ex, "should check for all IDs")
	})

	t.Run("Save", func(t *testing.T) {
		t.Parallel()

		t.Run("save", func(t *testing.T) {
			t.Parallel()

			entity := testdata.RandomEntity()
			repo := newEntityRepo()

			err := repo.Save(ctx, entity)
			assert.NoError(t, err)

			c, err := repo.Count(ctx)
			assert.NoError(t, err)
			assert.Equal(t, 1, c)
		})

		t.Run("save multiple times", func(t *testing.T) {
			t.Parallel()

			entity := testdata.RandomEntity()
			repo := newEntityRepo()

			err := repo.Save(ctx, entity)
			assert.NoError(t, err)

			entity.Name = "new-name"
			err = repo.Save(ctx, entity)
			assert.NoError(t, err)

			c, err := repo.Count(ctx)
			assert.NoError(t, err)
			assert.Equal(t, 1, c)

			got, err := repo.FindByID(ctx, entity.ID)
			assert.NoError(t, err)
			assert.Equal(t, entity, got, "save implements upsert semantic")
		})

		t.Run("missing id", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()

			err := repo.Save(ctx, testdata.Entity{})
			assert.ErrorIs(t, err, ErrSaveFailed)
		})
	})

	t.Run("UpdateAll", func(t *testing.T) {
		t.Parallel()

		t.Run("update all", func(t *testing.T) {
			t.Parallel()

			entity0 := testdata.RandomEntity()
			oldName := entity0.Name
			entity1 := testdata.RandomEntity()
			entity2 := testdata.RandomEntity()

			repo := newEntityRepo()
			err := repo.AddAll(ctx, []testdata.Entity{entity0, entity1, entity2})
			assert.NoError(t, err)

			entity0.Name = gofakeit.Name()
			entity1.Name = gofakeit.Name()
			entity2.Name = gofakeit.Name()

			err = repo.UpdateAll(ctx, []testdata.Entity{entity0, entity1, entity2})
			assert.NoError(t, err)

			c, err := repo.Count(ctx)
			assert.NoError(t, err)
			assert.Equal(t, 3, c)

			got, err := repo.FindByID(ctx, entity0.ID)
			assert.NoError(t, err)
			assert.NotEqual(t, oldName, got.Name)
		})

		t.Run("update only existing", func(t *testing.T) {
			t.Parallel()

			entity := testdata.RandomEntity()
			oldName := entity.Name

			repo := newEntityRepo()
			err := repo.Create(ctx, entity)
			assert.NoError(t, err)

			entity.Name = gofakeit.Name()
			err = repo.UpdateAll(ctx, []testdata.Entity{entity, testdata.RandomEntity(), testdata.RandomEntity()})
			assert.Error(t, err, "can only update existing entities")

			c, err := repo.Count(ctx)
			assert.NoError(t, err)
			assert.Equal(t, 1, c)

			got, err := repo.FindByID(ctx, entity.ID)
			assert.NoError(t, err)
			assert.Equal(t, oldName, got.Name)
		})

		t.Run("missing id", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()
			err := repo.Create(ctx, testdata.DefaultEntity)
			assert.NoError(t, err)

			err = repo.UpdateAll(ctx, []testdata.Entity{{}})
			assert.ErrorIs(t, err, ErrSaveFailed)
		})
	})

	t.Run("AddAll", func(t *testing.T) {
		t.Parallel()

		t.Run("add all", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()

			err := repo.AddAll(ctx, []testdata.Entity{testdata.RandomEntity(), testdata.DefaultEntity, testdata.RandomEntity()})
			assert.NoError(t, err)

			c, err := repo.Count(ctx)
			assert.NoError(t, err)
			assert.Equal(t, 3, c)
		})

		t.Run("already exists", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()
			err := repo.Save(ctx, testdata.DefaultEntity)
			assert.NoError(t, err)

			err = repo.AddAll(ctx, []testdata.Entity{testdata.RandomEntity(), testdata.DefaultEntity, testdata.RandomEntity()})
			assert.ErrorIs(t, err, ErrAlreadyExists)

			c, err := repo.Count(ctx)
			assert.NoError(t, err)
			assert.Equal(t, 1, c)
		})
	})

	t.Run("Length", func(t *testing.T) {
		t.Parallel()

		repo := newEntityRepo()

		count, err := repo.Length(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 0, count, "new repository should be empty")

		err = repo.Create(ctx, testdata.RandomEntity())
		assert.NoError(t, err)
		err = repo.Create(ctx, testdata.RandomEntity())
		assert.NoError(t, err)

		count, err = repo.Length(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 2, count, "should have two entities")
	})

	t.Run("DeleteByID", func(t *testing.T) {
		t.Parallel()

		t.Run("delete", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()
			err := repo.Create(ctx, testdata.DefaultEntity)
			assert.NoError(t, err)

			err = repo.DeleteByID(ctx, testdata.DefaultEntity.ID)
			assert.NoError(t, err)

			empty, err := repo.Empty(ctx)
			assert.NoError(t, err)
			assert.True(t, empty)
		})

		t.Run("delete multiple times", func(t *testing.T) {
			t.Parallel()

			repo := newEntityRepo()
			err := repo.Create(ctx, testdata.DefaultEntity)
			assert.NoError(t, err)

			err = repo.DeleteByID(ctx, testdata.DefaultEntity.ID)
			assert.NoError(t, err)

			err = repo.DeleteByID(ctx, testdata.DefaultEntity.ID)
			assert.NoError(t, err)
		})
	})

	t.Run("DeleteByIDs", func(t *testing.T) {
		t.Parallel()

		t.Run("delete", func(t *testing.T) {
			t.Parallel()

			e0 := testdata.RandomEntity()
			e1 := testdata.RandomEntity()
			e2 := testdata.RandomEntity()
			repo := newEntityRepo()
			err := repo.AddAll(ctx, []testdata.Entity{e0, e1, e2})
			assert.NoError(t, err)

			err = repo.DeleteByIDs(ctx, []testdata.EntityID{e0.ID, e2.ID})
			assert.NoError(t, err)

			c, err := repo.Count(ctx)
			assert.NoError(t, err)
			assert.Equal(t, 1, c)
		})

		t.Run("delete non-existing", func(t *testing.T) {
			t.Parallel()

			e0 := testdata.RandomEntity()
			e1 := testdata.RandomEntity()
			e2 := testdata.RandomEntity()
			repo := newEntityRepo()
			err := repo.AddAll(ctx, []testdata.Entity{e0, e1, e2})
			assert.NoError(t, err)

			err = repo.DeleteByIDs(ctx, []testdata.EntityID{e0.ID, e2.ID, testdata.RandomEntity().ID})
			assert.NoError(t, err)

			c, err := repo.Count(ctx)
			assert.NoError(t, err)
			assert.Equal(t, 1, c)
		})
	})

	t.Run("DeleteAll", func(t *testing.T) {
		t.Parallel()

		repo := newEntityRepo()
		err := repo.Create(ctx, testdata.DefaultEntity)
		assert.NoError(t, err)

		err = repo.DeleteAll(ctx)
		assert.NoError(t, err)

		c, err := repo.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 0, c)

		err = repo.DeleteAll(ctx)
		assert.NoError(t, err)
	})

	t.Run("Clear", func(t *testing.T) {
		t.Parallel()

		repo := newEntityRepo()
		err := repo.Clear(ctx)
		assert.NoError(t, err, "clear empty repo is empty repo")

		err = repo.Create(ctx, testdata.DefaultEntity)
		assert.NoError(t, err)

		err = repo.Clear(ctx)
		assert.NoError(t, err)

		c, err := repo.Count(ctx)
		assert.NoError(t, err)
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
				repo.Add(ctx, testdata.RandomEntity())
				wg.Done()
			}()
		}

		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func() {
				repo.Read(ctx, testdata.RandomEntity().ID)
				wg.Done()
			}()
		}

		wg.Wait()
	})
}
