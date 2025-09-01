package arepo_test

import (
	"context"
	"errors"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/arepo"
	"github.com/go-arrower/arrower/arepo/testdata"
)

var errStoreFailed = errors.New("store failed")

var (
	ctx = context.Background()

	defaultTenant = testTenant()
)

func testTenant() Tenant {
	return Tenant{
		ID:   TenantID(uuid.New().String()),
		Name: gofakeit.Name(),
	}
}

type (
	TenantID string
	Tenant   struct {
		ID   TenantID
		Name string
	}
)

func testEntityMemoryRepository(s arepo.Store) *entityMemoryRepository {
	return &entityMemoryRepository{
		MemoryRepository: arepo.NewMemoryRepository[testdata.Entity, testdata.EntityID](arepo.WithStore(s)),
	}
}

type entityMemoryRepository struct {
	*arepo.MemoryRepository[testdata.Entity, testdata.EntityID]
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

// testStoreStoreFails fails the call to store method,
// if it is called more than successCallsBeforeFailure times.
func testStoreStoreFails(successCallsBeforeFailure int) testStore {
	called := 0

	return testStore{
		load: func(_ string, _ any) error {
			return nil
		},
		store: func(_ string, _ any) error {
			if called >= successCallsBeforeFailure {
				return errStoreFailed
			}

			called++

			return nil
		},
	}
}

func testStoreSuccessEntity(t *testing.T, file string) testStore {
	t.Helper()

	return testStore{
		load: func(filename string, data any) error {
			assert.Equal(t, file, filename)
			assert.NotNil(t, data)

			return nil
		},
		store: func(filename string, data any) error {
			assert.Equal(t, file, filename)
			assert.NotNil(t, data)

			return nil
		},
	}
}
