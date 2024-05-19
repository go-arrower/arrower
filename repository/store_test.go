//go:build integration

package repository_test

import (
	"path"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/repository"
)

//nolint:paralleltest // subtests need to execute in order
func TestStore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := repository.NewJSONStore(dir)

	t.Run("load from empty folder", func(t *testing.T) {
		repo := testEntityMemoryRepository(store)

		count, err := repo.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("store", func(t *testing.T) {
		repo := testEntityMemoryRepository(store)

		err := repo.Save(ctx, defaultEntity)
		assert.NoError(t, err)
		assert.FileExists(t, path.Join(dir, "Entity.json"))
	})

	t.Run("load", func(t *testing.T) {
		repo := testEntityMemoryRepository(store)

		count, err := repo.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)

		e, err := repo.FindByID(ctx, defaultEntity.ID)
		assert.NoError(t, err)
		assert.Equal(t, defaultEntity, e)
	})

	t.Run("second entity, different file, some store", func(t *testing.T) {
		repo := repository.NewMemoryRepository[Entity, string](
			repository.WithStore(store),
			repository.WithStoreFilename("Entity2.json"),
		)
		err := repo.Save(ctx, defaultEntity)
		assert.NoError(t, err)

		assert.FileExists(t, path.Join(dir, "Entity.json"))
		assert.FileExists(t, path.Join(dir, "Entity2.json"))
	})

	t.Run("parallel", func(t *testing.T) {
		t.Parallel()

		repo := testEntityMemoryRepository(store)
		wg := sync.WaitGroup{}

		const routines = 15
		wg.Add(routines)

		for range routines {
			go func() {
				err := repo.Save(ctx, defaultEntity)
				assert.NoError(t, err)

				wg.Done()
			}()
		}

		wg.Wait()

		assert.FileExists(t, path.Join(dir, "Entity.json"))
	})
}
