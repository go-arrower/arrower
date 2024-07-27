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

var (
	ErrNotFound      = errors.New("not found")
	ErrSaveFailed    = errors.New("save failed")
	ErrAlreadyExists = errors.New("exists already")
)

// WithStore sets a Store used to persist the Repository.
// ONLY applies to the in memory implementations.
//
// There are no transactions or any consistency guarantees at all! For example, if a store fails,
// the collection is still changed in memory of the repository.
func WithStore(store Store) Option { //nolint:revive // unexported-return is OK for this option
	return func(config *repoConfig) {
		config.store = store
	}
}

// WithStoreFilename overwrites the file name a Store should use to persist this Repository.
// ONLY applies to the in memory implementations.
func WithStoreFilename(name string) Option { //nolint:revive // unexported-return is OK for this option
	return func(config *repoConfig) {
		config.filename = name
	}
}

// NewMemoryRepository returns an implementation of Repository for the given entity E.
// It is expected that E has a field called `ID`, that is used as the primary key and can
// be overwritten by WithIDField.
// If your repository needs additional methods, you can embed this repo into our own implementation to extend
// your own repository easily to your use case. See the examples in the test files.
//
// Warning: the consistency of MemoryRepository is not on paar with ACID guarantees of a RDBMS.
// Certain steps are taken to have a minimum of consistency, but be aware that this
// it not a design goal.
func NewMemoryRepository[E any, ID id](opts ...Option) *MemoryRepository[E, ID] {
	repo := &MemoryRepository[E, ID]{
		Mutex:        &sync.Mutex{},
		Data:         make(map[ID]E),
		currentIntID: *new(ID),
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

// MemoryRepository implements Repository in a generic way. Use it to speed up your unit testing.
type MemoryRepository[E any, ID id] struct {
	// Mutex is embedded, so that repositories who extend MemoryRepository can lock the same mutex as other methods.
	*sync.Mutex

	// Data is the repository's collection. It is exposed in case you're extending the repository.
	// PREVENT using and accessing Data it directly, go through the repository methods.
	// If you write to Data, USE the Mutex to lock first.
	Data         map[ID]E
	currentIntID ID

	repoConfig
}

const panicIDNotSupported = "type of ID is not supported: "

func defaultFileName(entity any) string {
	return reflect.TypeOf(entity).Elem().Name() + ".json"
}

func (repo *MemoryRepository[E, ID]) getID(t any) ID { //nolint:dupl,ireturn,lll // needs access to the type ID and fp, as it is not recognised even with "generic" setting
	val := reflect.ValueOf(t)

	idField := val.FieldByName(repo.idFieldName)
	if reflect.DeepEqual(idField, reflect.Value{}) { //nolint:govet,lll // is a fp and will be fixed, see: https://github.com/golang/go/issues/43993
		panic("entity does not have the field with name: " + repo.idFieldName)
	}

	var id ID

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

// NextID returns a new ID. It can be of the underlying type of string or integer.
func (repo *MemoryRepository[E, ID]) NextID(_ context.Context) (ID, error) { //nolint:ireturn,lll // fp, as it is not recognised even with "generic" setting
	var id ID

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

func (repo *MemoryRepository[E, ID]) Create(_ context.Context, entity E) error {
	repo.Lock()
	defer repo.Unlock()

	id := repo.getID(entity)
	if id == *new(ID) {
		return fmt.Errorf("missing ID: %w", ErrSaveFailed)
	}

	if _, found := repo.Data[id]; found {
		return ErrAlreadyExists
	}

	repo.Data[id] = entity

	err := repo.store.Store(repo.filename, repo.Data)
	if err != nil {
		delete(repo.Data, id)
		return fmt.Errorf("could not save: %w", err)
	}

	return nil
}

func (repo *MemoryRepository[E, ID]) Read(ctx context.Context, id ID) (E, error) { //nolint:ireturn,lll // valid use of generics
	return repo.FindByID(ctx, id)
}

func (repo *MemoryRepository[E, ID]) Update(_ context.Context, entity E) error {
	repo.Lock()
	defer repo.Unlock()

	id := repo.getID(entity)
	if id == *new(ID) {
		return fmt.Errorf("missing ID: %w", ErrSaveFailed)
	}

	if _, found := repo.Data[id]; !found {
		return fmt.Errorf("entity does not exist yet: %w", ErrSaveFailed)
	}

	oldEntity := repo.Data[id]
	repo.Data[id] = entity

	err := repo.store.Store(repo.filename, repo.Data)
	if err != nil {
		repo.Data[id] = oldEntity
		return fmt.Errorf("could not save: %w", err)
	}

	return nil
}

func (repo *MemoryRepository[E, ID]) Delete(_ context.Context, entity E) error {
	repo.Lock()
	defer repo.Unlock()

	id := repo.getID(entity)
	oldEntity := repo.Data[id]

	delete(repo.Data, id)

	err := repo.store.Store(repo.filename, repo.Data)
	if err != nil {
		repo.Data[id] = oldEntity
		return fmt.Errorf("could not save: %w", err)
	}

	return nil
}

func (repo *MemoryRepository[E, ID]) All(ctx context.Context) ([]E, error) {
	return repo.FindAll(ctx)
}

func (repo *MemoryRepository[E, ID]) AllByIDs(ctx context.Context, ids []ID) ([]E, error) {
	return repo.FindByIDs(ctx, ids)
}

func (repo *MemoryRepository[E, ID]) FindAll(_ context.Context) ([]E, error) {
	repo.Lock()
	defer repo.Unlock()

	result := []E{}

	for _, v := range repo.Data {
		result = append(result, v)
	}

	return result, nil
}

func (repo *MemoryRepository[E, ID]) FindByID(_ context.Context, id ID) (E, error) { //nolint:ireturn,lll // valid use of generics
	repo.Lock()
	defer repo.Unlock()

	if e, ok := repo.Data[id]; ok {
		return e, nil
	}

	return *new(E), ErrNotFound
}

func (repo *MemoryRepository[E, ID]) FindByIDs(_ context.Context, ids []ID) ([]E, error) {
	repo.Lock()
	defer repo.Unlock()

	result := []E{}

	for _, v := range repo.Data {
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

func (repo *MemoryRepository[E, ID]) Exists(_ context.Context, id ID) (bool, error) {
	repo.Lock()
	defer repo.Unlock()

	if _, ok := repo.Data[id]; ok {
		return true, nil
	}

	return false, nil
}

func (repo *MemoryRepository[E, ID]) ExistsByID(ctx context.Context, id ID) (bool, error) {
	return repo.Exists(ctx, id)
}

func (repo *MemoryRepository[E, ID]) ExistByIDs(ctx context.Context, ids []ID) (bool, error) {
	return repo.ExistAll(ctx, ids)
}

func (repo *MemoryRepository[E, ID]) ExistAll(_ context.Context, ids []ID) (bool, error) {
	repo.Lock()
	defer repo.Unlock()

	if len(ids) == 0 {
		return false, nil
	}

	for _, id := range ids {
		if _, ok := repo.Data[id]; !ok {
			return false, nil
		}
	}

	return true, nil
}

func (repo *MemoryRepository[E, ID]) Contains(ctx context.Context, id ID) (bool, error) {
	return repo.ExistsByID(ctx, id)
}

func (repo *MemoryRepository[E, ID]) ContainsID(ctx context.Context, id ID) (bool, error) {
	return repo.ExistsByID(ctx, id)
}

func (repo *MemoryRepository[E, ID]) ContainsIDs(ctx context.Context, ids []ID) (bool, error) {
	return repo.ExistByIDs(ctx, ids)
}

func (repo *MemoryRepository[E, ID]) ContainsAll(ctx context.Context, ids []ID) (bool, error) {
	return repo.ContainsIDs(ctx, ids)
}

func (repo *MemoryRepository[E, ID]) Save(_ context.Context, entity E) error {
	repo.Lock()
	defer repo.Unlock()

	id := repo.getID(entity)
	if id == *new(ID) {
		return fmt.Errorf("missing ID: %w", ErrSaveFailed)
	}

	repo.Data[id] = entity

	err := repo.store.Store(repo.filename, repo.Data)
	if err != nil {
		delete(repo.Data, id)
		return fmt.Errorf("could not save: %w", err)
	}

	return nil
}

func (repo *MemoryRepository[E, ID]) SaveAll(_ context.Context, entities []E) error {
	repo.Lock()
	defer repo.Unlock()

	for _, e := range entities {
		if repo.getID(e) == *new(ID) {
			return fmt.Errorf("missing ID: %w", ErrSaveFailed)
		}
	}

	oldEntities := []E{}

	for _, e := range entities {
		oldEntities = append(oldEntities, repo.Data[repo.getID(e)])
		repo.Data[repo.getID(e)] = e
	}

	err := repo.store.Store(repo.filename, repo.Data)
	if err != nil {
		for _, e := range oldEntities {
			repo.Data[repo.getID(e)] = e
		}

		return fmt.Errorf("could not save: %w", err)
	}

	return nil
}

func (repo *MemoryRepository[E, ID]) UpdateAll(ctx context.Context, entities []E) error {
	repo.Lock()
	defer repo.Unlock()

	for _, e := range entities {
		if repo.getID(e) == *new(ID) {
			return fmt.Errorf("missing ID: %w", ErrSaveFailed)
		}
	}

	oldEntities := []E{}
	updatedEntities := []E{}

	for _, e := range entities {
		if _, found := repo.Data[repo.getID(e)]; !found {
			return fmt.Errorf("entity does not exist yet: %w", ErrSaveFailed)
		}

		oldEntities = append(oldEntities, repo.Data[repo.getID(e)])
		updatedEntities = append(updatedEntities, e)
	}

	for _, e := range updatedEntities {
		repo.Data[repo.getID(e)] = e
	}

	err := repo.store.Store(repo.filename, repo.Data)
	if err != nil {
		for _, e := range oldEntities {
			repo.Data[repo.getID(e)] = e
		}

		return fmt.Errorf("could not save: %w", err)
	}

	return nil
}

func (repo *MemoryRepository[E, ID]) Add(ctx context.Context, entity E) error {
	return repo.Create(ctx, entity)
}

func (repo *MemoryRepository[E, ID]) AddAll(ctx context.Context, entities []E) error {
	for _, e := range entities {
		ex, _ := repo.Exists(ctx, repo.getID(e))
		if ex {
			return ErrAlreadyExists
		}
	}

	return repo.SaveAll(ctx, entities)
}

func (repo *MemoryRepository[E, ID]) Count(_ context.Context) (int, error) {
	repo.Lock()
	defer repo.Unlock()

	return len(repo.Data), nil
}

func (repo *MemoryRepository[E, ID]) Length(ctx context.Context) (int, error) {
	return repo.Count(ctx)
}

func (repo *MemoryRepository[E, ID]) DeleteByID(ctx context.Context, id ID) error {
	return repo.DeleteByIDs(ctx, []ID{id})
}

func (repo *MemoryRepository[E, ID]) DeleteByIDs(ctx context.Context, ids []ID) error {
	repo.Lock()
	defer repo.Unlock()

	oldEntities := []E{}

	for _, id := range ids {
		oldEntities = append(oldEntities, repo.Data[id])
		delete(repo.Data, id)
	}

	err := repo.store.Store(repo.filename, repo.Data)
	if err != nil {
		for _, e := range oldEntities {
			repo.Data[repo.getID(e)] = e
		}

		return fmt.Errorf("could not delete: %w", err)
	}

	return nil
}

func (repo *MemoryRepository[E, ID]) DeleteAll(_ context.Context) error {
	repo.Lock()
	defer repo.Unlock()

	oldEntities := make(map[ID]E)
	for k, v := range repo.Data {
		oldEntities[k] = v
	}

	clear(repo.Data)

	err := repo.store.Store(repo.filename, repo.Data)
	if err != nil {
		for _, e := range oldEntities {
			repo.Data[repo.getID(e)] = e
		}

		return fmt.Errorf("could not save: %w", err)
	}

	return nil
}

func (repo *MemoryRepository[E, ID]) Clear(ctx context.Context) error {
	return repo.DeleteAll(ctx)
}

func (repo *MemoryRepository[E, ID]) AllIter(ctx context.Context) Iterator[E, ID] {
	return MemoryIterator[E, ID]{repo: repo}
}
func (repo *MemoryRepository[E, ID]) FindAllIter(ctx context.Context) Iterator[E, ID] {
	return MemoryIterator[E, ID]{repo: repo}
}

type MemoryIterator[E any, ID id] struct {
	repo *MemoryRepository[E, ID]
}

func (i MemoryIterator[E, ID]) Next() func(yield func(e E, err error) bool) {
	return func(yield func(e E, err error) bool) {
		for _, e := range i.repo.Data {
			if !yield(e, nil) {
				return
			}
		}
	}
}
