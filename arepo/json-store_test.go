//go:build integration

package arepo_test

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/arepo"
	"github.com/go-arrower/arrower/arepo/testdata"
)

func TestJSONStore_Store(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := arepo.NewJSONStore(dir)

	data := map[testdata.EntityID]testdata.Entity{}

	t.Run("empty file name", func(t *testing.T) {
		t.Parallel()

		err := store.Store("", data)
		assert.Error(t, err)
	})

	t.Run("no data", func(t *testing.T) {
		t.Parallel()

		err := store.Store("some.json", nil)
		assert.NoError(t, err)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		err := store.Store("some.json", data)
		assert.NoError(t, err)
		assert.FileExists(t, path.Join(dir, "some.json"))
	})
}

func TestJSONStore_Load(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := arepo.NewJSONStore(dir)

	t.Run("file does not exist", func(t *testing.T) {
		t.Parallel()

		err := store.Load("file-not-exists", map[testdata.EntityID]testdata.Entity{})
		assert.ErrorIs(t, err, arepo.ErrLoad)
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("empty filename", func(t *testing.T) {
		t.Parallel()

		err := store.Load("", map[testdata.EntityID]testdata.Entity{})
		assert.ErrorIs(t, err, arepo.ErrLoad)
	})
}
