//go:build integration

package arepo_test

import (
	"path"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/arepo"
	"github.com/go-arrower/arrower/arepo/testdata"
)

//nolint:paralleltest // subtests need to execute in order
func TestStore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := arepo.NewJSONStore(dir)

	t.Run("load from empty folder", func(t *testing.T) {
		repo := testEntityMemoryRepository(store)

		count, err := repo.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("store", func(t *testing.T) {
		repo := testEntityMemoryRepository(store)

		err := repo.Save(ctx, testdata.DefaultEntity)
		assert.NoError(t, err)
		assert.FileExists(t, path.Join(dir, "Entity.json"))
	})

	t.Run("load", func(t *testing.T) {
		repo := testEntityMemoryRepository(store)

		count, err := repo.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)

		e, err := repo.FindByID(ctx, testdata.DefaultEntity.ID)
		assert.NoError(t, err)
		assert.Equal(t, testdata.DefaultEntity, e)
	})

	t.Run("second entity, different file, some store", func(t *testing.T) {
		repo := arepo.NewMemoryRepository[testdata.Entity, string](
			arepo.WithStore(store),
			arepo.WithStoreFilename("Entity2.json"),
		)
		err := repo.Save(ctx, testdata.DefaultEntity)
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
				err := repo.Save(ctx, testdata.DefaultEntity)
				assert.NoError(t, err)

				wg.Done()
			}()
		}

		wg.Wait()

		assert.FileExists(t, path.Join(dir, "Entity.json"))
	})
}
