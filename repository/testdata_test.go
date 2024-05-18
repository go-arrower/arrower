package repository_test

import (
	"errors"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/repository"
)

var errStoreFailed = errors.New("store failed")

var DefaultEntity = NewEntity()

func NewEntity() Entity {
	return Entity{
		ID:   EntityID(uuid.New().String()),
		Name: gofakeit.Name(),
	}
}

type (
	EntityID string
	Entity   struct {
		ID   EntityID
		Name string
	}
)

type EntityWithoutID struct {
	Name string
}

func NewEntityMemoryRepository(s repository.Store) *EntityMemoryRepository {
	return &EntityMemoryRepository{
		MemoryRepository: repository.NewMemoryRepository[Entity, EntityID](repository.WithStore(s)),
	}
}

type EntityMemoryRepository struct {
	*repository.MemoryRepository[Entity, EntityID]
}

type testStore struct {
	load  func(filename string, data any) error
	store func(filename string, data any) error
}

func (s testStore) Load(filename string, data any) error {
	return s.load(filename, data)
}

func (s testStore) Store(filename string, data any) error {
	return s.store(filename, data)
}

func testStoreLoadFails() testStore {
	return testStore{
		load: func(_ string, _ any) error {
			return errStoreFailed
		},
		store: func(_ string, _ any) error {
			return nil
		},
	}
}

func testStoreStoreFails() testStore {
	return testStore{
		load: func(_ string, _ any) error {
			return nil
		},
		store: func(_ string, _ any) error {
			return errStoreFailed
		},
	}
}

func testStoreSuccessEntity(t *testing.T) testStore {
	t.Helper()

	return testStore{
		load: func(filename string, data any) error {
			assert.Equal(t, "Entity.json", filename)
			assert.NotNil(t, data)

			return nil
		},
		store: func(filename string, data any) error {
			assert.Equal(t, "Entity.json", filename)
			assert.NotNil(t, data)

			return nil
		},
	}
}
