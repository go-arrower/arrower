package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sync"
)

// TenantRepository is a general purpose interface documenting
// which methods are available by the generic MemoryTenantRepository.
// tID is the primary key for the tenant and eID of the entity and needs to be of one of the underlying types.
// If your repository needs additional methods, you can extend your own repository easily to tune it to your use case.
type TenantRepository[T any, tID id, E any, eID id] interface {
	// NextID(ctx context.Context) (ID, error)

	Create(ctx context.Context, tenantID tID, entity E) error
	Read(ctx context.Context, tenantID tID, id eID) (E, error)
	Update(ctx context.Context, tenantID tID, entity E) error
	Delete(ctx context.Context, tenantID tID, entity E) error

	All(ctx context.Context, tenantID tID) ([]E, error)
	// AllByIDs(ctx context.Context, ids []ID) ([]E, error)
	// FindAll(ctx context.Context) ([]E, error)
	// FindByID(ctx context.Context, id ID) (E, error)
	// FindByIDs(ctx context.Context, ids []ID) (E, error)
	// Exists(ctx context.Context, id ID) (bool, error)
	// ExistsByID(ctx context.Context, id ID) (bool, error)
	// ExistAll(ctx context.Context, ids []ID) (bool, error)
	// ExistByIDs(ctx context.Context, ids []ID) (bool, error)
	// Contains(ctx context.Context, id ID) (bool, error)
	// ContainsID(ctx context.Context, id ID) (bool, error)
	// ContainsIDs(ctx context.Context, ids []ID) (bool, error)
	// ContainsAll(ctx context.Context, ids []ID) (bool, error)
	//
	// Save(ctx context.Context, entity E) error
	// SaveAll(ctx context.Context, entities []E) error
	// UpdateAll(ctx context.Context, entities []E) error
	// Add(ctx context.Context, entity E) error
	// AddAll(ctx context.Context, entities []E) error
	//
	// Count(ctx context.Context) (int, error)
	// Length(ctx context.Context) (int, error)
	// Empty(ctx context.Context) (bool, error)
	// IsEmpty(ctx context.Context) (bool, error)
	//
	// DeleteByID(ctx context.Context, id ID) error
	// DeleteByIDs(ctx context.Context, ids []ID) error
	// DeleteAll(ctx context.Context) error
	// Clear(ctx context.Context) error
}

// NewMemoryTenantRepository returns an implementation of MemoryTenantRepository for the given tenant T and entity E.
// It is expected that T and E have a field called `ID`, that is used as the primary key and can
// be overwritten by WithTenantID and WithIDField.
// If your repository needs additional methods, you can embed this repo into our own implementation to extend
// your own repository easily to your use case.
func NewMemoryTenantRepository[T any, tID id, E any, eID id](
	opts ...memoryRepositoryOption,
) *MemoryTenantRepository[T, tID, E, eID] {
	repo := &MemoryTenantRepository[T, tID, E, eID]{
		Mutex: &sync.Mutex{},
		Data:  make(map[tID]map[eID]E),
		repoConfig: repoConfig{
			idFieldName: "ID",
			store:       noopStore{},
			filename:    defaultFileName(new(E)),
		},
	}

	for _, opt := range opts {
		opt(&repo.repoConfig)
	}

	err := repo.store.Load(repo.filename, &repo.Data)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		panic("could not load data for memory repository from store: " + err.Error())
	}

	return repo
}

// MemoryTenantRepository implements TenantRepository in a generic way. Use it to speed up your unit testing.
type MemoryTenantRepository[T any, tID id, E any, eID id] struct {
	// Mutex is embedded, so that repositories who extend MemoryTenantRepository can lock the same mutex as other methods.
	*sync.Mutex

	// Data is the repository's collection. It is exposed in case you're extending the repository.
	// PREVENT using and accessing Data it directly, go through the repository methods.
	// If you write to Data, USE the Mutex to lock first.
	Data map[tID]map[eID]E

	repoConfig
}

// ensureTenantInitialised creates a tenant-specific map, as go does not initialise maps into an empty state by default.
func (repo *MemoryTenantRepository[T, tID, E, eID]) ensureTenantInitialised(id tID) {
	if repo.Data[id] == nil {
		repo.Data[id] = make(map[eID]E)
	}
}

func (repo *MemoryTenantRepository[T, tID, E, eID]) getID(e E) eID { //nolint:dupl,ireturn,lll // needs acces to the type ID and fp, as it is not recognised even with "generic" setting
	val := reflect.ValueOf(e)

	idField := val.FieldByName(repo.idFieldName)
	if reflect.DeepEqual(idField, reflect.Value{}) { //nolint:govet,lll // is a fp and will be fixed, see: https://github.com/golang/go/issues/43993
		panic("entity does not have the field with name: " + repo.idFieldName)
	}

	var id eID

	switch idField.Kind() {
	case reflect.String:
		reflect.ValueOf(&id).Elem().SetString(idField.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		reflect.ValueOf(&id).Elem().SetInt(idField.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		reflect.ValueOf(&id).Elem().SetUint(idField.Uint())
	default:
		panic(panicIDNotSupported + idField.Kind().String())
	}

	return id
}

func (repo *MemoryTenantRepository[T, tID, E, eID]) Create(_ context.Context, tenantID tID, entity E) error {
	repo.Lock()
	defer repo.Unlock()

	id := repo.getID(entity)
	if id == *new(eID) {
		return fmt.Errorf("missing ID: %w", ErrSaveFailed)
	}

	repo.ensureTenantInitialised(tenantID)

	if _, found := repo.Data[tenantID][id]; found {
		return ErrAlreadyExists
	}

	repo.Data[tenantID][id] = entity

	err := repo.store.Store(repo.filename, repo.Data)
	if err != nil {
		return fmt.Errorf("could not save: %w", err)
	}

	return nil
}

func (repo *MemoryTenantRepository[T, tID, E, eID]) Read(_ context.Context, tenantID tID, id eID) (E, error) { //nolint:ireturn,lll // valid use of generics
	repo.Lock()
	defer repo.Unlock()

	if e, ok := repo.Data[tenantID][id]; ok {
		return e, nil
	}

	return *new(E), ErrNotFound
}

func (repo *MemoryTenantRepository[T, tID, E, eID]) Update(_ context.Context, tenantID tID, entity E) error {
	repo.Lock()
	defer repo.Unlock()

	id := repo.getID(entity)
	if id == *new(eID) {
		return fmt.Errorf("missing ID: %w", ErrSaveFailed)
	}

	if _, found := repo.Data[tenantID][id]; !found {
		return fmt.Errorf("entity does not exist yet: %w", ErrSaveFailed)
	}

	repo.Data[tenantID][id] = entity

	err := repo.store.Store(repo.filename, repo.Data)
	if err != nil {
		return fmt.Errorf("could not save: %w", err)
	}

	return nil
}

func (repo *MemoryTenantRepository[T, tID, E, eID]) Delete(_ context.Context, tenantID tID, entity E) error {
	repo.Lock()
	defer repo.Unlock()

	delete(repo.Data[tenantID], repo.getID(entity))

	err := repo.store.Store(repo.filename, repo.Data)
	if err != nil {
		return fmt.Errorf("could not save: %w", err)
	}

	return nil
}

func (repo *MemoryTenantRepository[T, tID, E, eID]) All(_ context.Context, tenantID tID) ([]E, error) {
	repo.Lock()
	defer repo.Unlock()

	result := []E{}

	for _, v := range repo.Data[tenantID] {
		result = append(result, v)
	}

	return result, nil
}
