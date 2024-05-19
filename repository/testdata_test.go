package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/repository"
)

var errStoreFailed = errors.New("store failed")

var (
	ctx           = context.Background()
	defaultEntity = testEntity()
	defaultTenant = testTenant()
)

func testEntity() Entity {
	return Entity{
		ID:   EntityID(uuid.New().String()),
		Name: gofakeit.Name(),
	}
}

func testTenant() Tenant {
	return Tenant{
		ID:   TenantID(uuid.New().String()),
		Name: gofakeit.Name(),
	}
}

type (
	EntityID string
	Entity   struct {
		ID   EntityID
		Name string
	}

	TenantID string
	Tenant   struct {
		ID   TenantID
		Name string
	}
)

type EntityWithoutID struct {
	Name string
}

func testEntityMemoryRepository(s repository.Store) *entityMemoryRepository {
	return &entityMemoryRepository{
		MemoryRepository: repository.NewMemoryRepository[Entity, EntityID](repository.WithStore(s)),
	}
}

type entityMemoryRepository struct {
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
