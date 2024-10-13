package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sync"

	"github.com/google/uuid"
)

// NewMemoryTenantRepository returns an implementation of MemoryTenantRepository for the given entity E.
// It is expected that E have a field called `ID`, that is used as the primary key and can
// be overwritten by WithIDField.
// If your repository needs additional methods, you can embed this repo into our own implementation to extend
// your own repository easily to your use case.
func NewMemoryTenantRepository[tID id, E any, eID id](
	opts ...Option,
) *MemoryTenantRepository[tID, E, eID] {
	repo := &MemoryTenantRepository[tID, E, eID]{
		Mutex: &sync.Mutex{},
		Data:  make(map[tID]map[eID]E),
		repoConfig: repoConfig{
			idFieldName: "ID",
			store:       NoopStore{},
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
type MemoryTenantRepository[tID id, E any, eID id] struct {
	// Mutex is embedded, so that repositories who extend MemoryTenantRepository can lock the same mutex as other methods.
	*sync.Mutex

	// Data is the repository's collection. It is exposed in case you're extending the repository.
	// PREVENT using and accessing Data it directly, go through the repository methods.
	// If you write to Data, USE the Mutex to lock first.
	Data         map[tID]map[eID]E
	currentIntID eID

	repoConfig
}

// ensureTenantInitialised creates a tenant-specific map, as go does not initialise maps into an empty state by default.
func (repo *MemoryTenantRepository[tID, E, eID]) ensureTenantInitialised(id tID) {
	if repo.Data[id] == nil {
		repo.Data[id] = make(map[eID]E)
	}
}

func (repo *MemoryTenantRepository[tID, E, eID]) getID(e E) eID { //nolint:dupl,ireturn,lll // needs access to the type ID and fp, as it is not recognised even with "generic" setting
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

func (repo *MemoryTenantRepository[tid, E, eID]) NextID(ctx context.Context) (eID, error) {
	var id eID

	switch reflect.TypeOf(id).Kind() {
	case reflect.String:
		reflect.ValueOf(&id).Elem().SetString(uuid.New().String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		repo.Mutex.Lock()
		defer repo.Mutex.Unlock()

		currentIDValue := reflect.ValueOf(&repo.currentIntID).Elem().Int()

		// increment the ID: the value is stored in the repo, but it cannot be accessed because
		// the generic does not know which type it is, so that is why reflection is used.
		newID := currentIDValue + 1
		reflect.ValueOf(&repo.currentIntID).Elem().SetInt(newID)

		reflect.ValueOf(&id).Elem().SetInt(newID)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		repo.Mutex.Lock()
		defer repo.Mutex.Unlock()

		currentIDValue := reflect.ValueOf(&repo.currentIntID).Elem().Uint()

		// increment the ID: the value is stored in the repo, but it cannot be accessed because
		// the generic does not know which type it is, so that is why reflection is used.
		newID := currentIDValue + 1
		reflect.ValueOf(&repo.currentIntID).Elem().SetUint(newID)

		reflect.ValueOf(&id).Elem().SetUint(newID)
	default:
		panic(panicIDNotSupported + reflect.TypeOf(id).Kind().String())
	}

	return id, nil
}

func (repo *MemoryTenantRepository[tID, E, eID]) Create(_ context.Context, tenantID tID, entity E) error {
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

func (repo *MemoryTenantRepository[tID, E, eID]) Read(_ context.Context, tenantID tID, id eID) (E, error) { //nolint:ireturn,lll // valid use of generics
	repo.Lock()
	defer repo.Unlock()

	if e, ok := repo.Data[tenantID][id]; ok {
		return e, nil
	}

	return *new(E), ErrNotFound
}

func (repo *MemoryTenantRepository[tID, E, eID]) Update(_ context.Context, tenantID tID, entity E) error {
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

func (repo *MemoryTenantRepository[tID, E, eID]) Delete(_ context.Context, tenantID tID, entity E) error {
	repo.Lock()
	defer repo.Unlock()

	delete(repo.Data[tenantID], repo.getID(entity))

	err := repo.store.Store(repo.filename, repo.Data)
	if err != nil {
		return fmt.Errorf("could not save: %w", err)
	}

	return nil
}

func (repo *MemoryTenantRepository[tID, E, eID]) All(_ context.Context) ([]E, error) {
	repo.Lock()
	defer repo.Unlock()

	result := []E{}

	for _, t := range repo.Data {
		for _, e := range t {
			result = append(result, e)
		}
	}

	return result, nil
}

func (repo *MemoryTenantRepository[tID, E, eID]) AllOfTenant(_ context.Context, tenantID tID) ([]E, error) {
	repo.Lock()
	defer repo.Unlock()

	result := []E{}

	for _, v := range repo.Data[tenantID] {
		result = append(result, v)
	}

	return result, nil
}

func (repo *MemoryTenantRepository[tID, E, eID]) AllByIDs(_ context.Context, tenantID tID, ids []eID) ([]E, error) {
	repo.Lock()
	defer repo.Unlock()

	result := []E{}

	for _, v := range repo.Data[tenantID] {
		for _, id := range ids {
			if repo.getID(v) == id {
				result = append(result, v)
			}
		}
	}

	if len(result) != len(ids) {
		return []E(nil), ErrNotFound
	}

	return result, nil
}

func (repo *MemoryTenantRepository[tID, E, eID]) FindAll(ctx context.Context) ([]E, error) {
	return repo.All(ctx)
}

func (repo *MemoryTenantRepository[tID, E, eID]) FindAllOfTenant(ctx context.Context, tenantID tID) ([]E, error) {
	return repo.AllOfTenant(ctx, tenantID)
}

func (repo *MemoryTenantRepository[tID, E, eID]) FindByID(ctx context.Context, tenantID tID, id eID) (E, error) { //nolint:ireturn,lll // valid use of generics
	return repo.Read(ctx, tenantID, id)
}

func (repo *MemoryTenantRepository[tID, E, eID]) FindByIDs(
	ctx context.Context,
	tenantID tID,
	ids []eID,
) ([]E, error) {
	return repo.AllByIDs(ctx, tenantID, ids)
}

func (repo *MemoryTenantRepository[tID, E, eID]) Exists(_ context.Context, tenantID tID, id eID) (bool, error) {
	repo.Lock()
	defer repo.Unlock()

	if _, ok := repo.Data[tenantID][id]; ok {
		return true, nil
	}

	return false, nil
}

func (repo *MemoryTenantRepository[tID, E, eID]) ExistsByID(
	ctx context.Context,
	tenantID tID,
	id eID,
) (bool, error) {
	return repo.Exists(ctx, tenantID, id)
}

func (repo *MemoryTenantRepository[tID, E, eID]) ExistByIDs(
	ctx context.Context,
	tenantID tID,
	ids []eID,
) (bool, error) {
	return repo.ExistAll(ctx, tenantID, ids)
}

func (repo *MemoryTenantRepository[tID, E, eID]) ExistAll(_ context.Context, tenantID tID, ids []eID) (bool, error) {
	repo.Lock()
	defer repo.Unlock()

	if len(ids) == 0 {
		return false, nil
	}

	for _, id := range ids {
		if _, ok := repo.Data[tenantID][id]; !ok {
			return false, nil
		}
	}

	return true, nil
}

func (repo *MemoryTenantRepository[tID, E, eID]) Contains(ctx context.Context, tenantID tID, id eID) (bool, error) {
	return repo.Exists(ctx, tenantID, id)
}

func (repo *MemoryTenantRepository[tID, E, eID]) ContainsID(
	ctx context.Context,
	tenantID tID,
	id eID,
) (bool, error) {
	return repo.ExistsByID(ctx, tenantID, id)
}

func (repo *MemoryTenantRepository[tID, E, eID]) ContainsIDs(
	ctx context.Context,
	tenantID tID,
	ids []eID,
) (bool, error) {
	return repo.ExistByIDs(ctx, tenantID, ids)
}

func (repo *MemoryTenantRepository[tID, E, eID]) ContainsAll(
	ctx context.Context,
	tenantID tID,
	ids []eID,
) (bool, error) {
	return repo.ExistAll(ctx, tenantID, ids)
}

func (repo *MemoryTenantRepository[tID, E, eID]) Save(_ context.Context, tenantID tID, entity E) error {
	repo.Lock()
	defer repo.Unlock()

	id := repo.getID(entity)
	if id == *new(eID) {
		return fmt.Errorf("missing ID: %w", ErrSaveFailed)
	}

	repo.ensureTenantInitialised(tenantID)

	repo.Data[tenantID][id] = entity

	err := repo.store.Store(repo.filename, repo.Data)
	if err != nil {
		return fmt.Errorf("could not save: %w", err)
	}

	return nil
}

func (repo *MemoryTenantRepository[tID, E, eID]) SaveAll(_ context.Context, tenantID tID, entities []E) error {
	repo.Lock()
	defer repo.Unlock()

	for _, e := range entities {
		if repo.getID(e) == *new(eID) {
			return fmt.Errorf("missing ID: %w", ErrSaveFailed)
		}
	}

	repo.ensureTenantInitialised(tenantID)

	for _, e := range entities {
		repo.Data[tenantID][repo.getID(e)] = e
	}

	err := repo.store.Store(repo.filename, repo.Data)
	if err != nil {
		return fmt.Errorf("could not save: %w", err)
	}

	return nil
}

func (repo *MemoryTenantRepository[tID, E, eID]) UpdateAll(ctx context.Context, tenantID tID, entities []E) error {
	return repo.SaveAll(ctx, tenantID, entities)
}

func (repo *MemoryTenantRepository[tID, E, eID]) Add(ctx context.Context, tenantID tID, entity E) error {
	return repo.Create(ctx, tenantID, entity)
}

func (repo *MemoryTenantRepository[tID, E, eID]) AddAll(ctx context.Context, tenantID tID, entities []E) error {
	for _, e := range entities {
		ex, _ := repo.Exists(ctx, tenantID, repo.getID(e))
		if ex {
			return ErrAlreadyExists
		}
	}

	return repo.SaveAll(ctx, tenantID, entities)
}

func (repo *MemoryTenantRepository[tID, E, eID]) Count(_ context.Context) (int, error) {
	count := 0

	for _, t := range repo.Data {
		count += len(t)
	}

	return count, nil
}

func (repo *MemoryTenantRepository[tID, E, eID]) CountOfTenant(_ context.Context, tenantID tID) (int, error) {
	repo.Lock()
	defer repo.Unlock()

	return len(repo.Data[tenantID]), nil
}

func (repo *MemoryTenantRepository[tID, E, eID]) Length(ctx context.Context) (int, error) {
	return repo.Count(ctx)
}

func (repo *MemoryTenantRepository[tID, E, eID]) LengthOfTenant(ctx context.Context, tenantID tID) (int, error) {
	return repo.CountOfTenant(ctx, tenantID)
}

func (repo *MemoryTenantRepository[tID, E, eID]) DeleteByID(ctx context.Context, tenantID tID, id eID) error {
	return repo.DeleteByIDs(ctx, tenantID, []eID{id})
}

func (repo *MemoryTenantRepository[tID, E, eID]) DeleteByIDs(_ context.Context, tenantID tID, ids []eID) error {
	repo.Lock()
	defer repo.Unlock()

	for _, id := range ids {
		delete(repo.Data[tenantID], id)
	}

	err := repo.store.Store(repo.filename, repo.Data)
	if err != nil {
		return fmt.Errorf("could not save: %w", err)
	}

	return nil
}

func (repo *MemoryTenantRepository[tID, E, eID]) DeleteAll(_ context.Context) error {
	repo.Lock()
	defer repo.Unlock()

	clear(repo.Data)

	err := repo.store.Store(repo.filename, repo.Data)
	if err != nil {
		return fmt.Errorf("could not save: %w", err)
	}

	return nil
}

func (repo *MemoryTenantRepository[tID, E, eID]) DeleteAllOfTenant(_ context.Context, tenantID tID) error {
	repo.Lock()
	defer repo.Unlock()

	clear(repo.Data[tenantID])

	err := repo.store.Store(repo.filename, repo.Data)
	if err != nil {
		return fmt.Errorf("could not save: %w", err)
	}

	return nil
}

func (repo *MemoryTenantRepository[tID, E, eID]) Clear(ctx context.Context) error {
	return repo.DeleteAll(ctx)
}

func (repo *MemoryTenantRepository[tID, E, eID]) ClearTenant(ctx context.Context, tenantID tID) error {
	return repo.DeleteAllOfTenant(ctx, tenantID)
}
