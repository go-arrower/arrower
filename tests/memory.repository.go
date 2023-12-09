package tests

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrSaveFailed    = errors.New("save failed")
	ErrAlreadyExists = errors.New("exists already")
)

// Repository is a general purpose interface documenting which methods are available by the generic MemoryRepository.
// ID is the primary key and needs to have an underlying string type, integers are not supported.
// If your repository needs additional methods, you can extend your own repository easily to tune it to your use case.
type Repository[E any, ID id] interface { //nolint:interfacebloat // showcase of all methods that are possible
	NextID(context.Context) (ID, error)

	Create(context.Context, E) error
	Read(context.Context, ID) (E, error)
	Update(context.Context, E) error
	Delete(context.Context, E) error

	All(context.Context) ([]E, error)
	AllByIDs(context.Context, []ID) ([]E, error)
	FindAll(ctx context.Context) ([]E, error)
	FindByID(context.Context, ID) (E, error)
	FindByIDs(context.Context, []ID) (E, error)
	Exists(context.Context, ID) (bool, error)
	ExistsByID(context.Context, ID) (bool, error)
	ExistAll(context.Context, []ID) (bool, error)
	ExistByIDs(context.Context, []ID) (bool, error)
	Contains(context.Context, ID) (bool, error)
	ContainsID(context.Context, ID) (bool, error)
	ContainsIDs(context.Context, []ID) (bool, error)
	ContainsAll(context.Context, []ID) (bool, error)

	Save(context.Context, E) error
	SaveAll(context.Context, []E) error
	UpdateAll(context.Context, []E) error
	Add(context.Context, E) error
	AddAll(context.Context, []E) error

	Count(context.Context) (int, error)
	Length(context.Context) (int, error)
	Empty(context.Context) (bool, error)
	IsEmpty(context.Context) (bool, error)

	DeleteByID(context.Context, ID) error
	DeleteByIDs(context.Context, []ID) error
	DeleteAll(context.Context) error
	Clear(context.Context) error
}

// WithIDField set's the name of the field that is used a id/primary key.
// If not set, it is assumed that the entity struct has a field with the name "ID".
func WithIDField(idFieldName string) memoryRepositoryOption { //nolint:revive
	return func(repo *repoConfig) {
		repo.idFieldName = idFieldName
	}
}

// id is the primary key used in the generic Repository.
type id interface {
	~string |
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

type memoryRepositoryOption func(*repoConfig)

type repoConfig struct {
	idFieldName string
}

// NewMemoryRepository returns an implementation of MemoryRepository for the given entity E.
// It is expected that E has a field called `ID`, that is used as the primary key and can
// be overwritten by WithIDField.
// If your repository needs additional methods, you can embed this repo into our own implementation to extend
// your own repository easily to your use case.
func NewMemoryRepository[E any, ID id](opts ...memoryRepositoryOption) *MemoryRepository[E, ID] {
	repo := &MemoryRepository[E, ID]{
		Mutex:        &sync.Mutex{},
		Data:         make(map[ID]E),
		currentIntID: *new(ID),
		repoConfig:   repoConfig{idFieldName: "ID"},
	}

	for _, opt := range opts {
		opt(&repo.repoConfig)
	}

	return repo
}

// MemoryRepository implements Repository in a generic way. Use it to speed up your unit testing.
type MemoryRepository[E any, ID id] struct {
	*sync.Mutex // The mutex is embedded, so that repositories who extend MemoryRepository can lock the same mutex.

	// Data is the repository's collection. It is exposed in case you're extending the repository, see:
	// https://www.arrower.org/docs/basics/testing#extending-the-repository
	// PREVENT using it directly and access the data through methods. If you write to it USE the Mutex to lock first.
	Data         map[ID]E
	currentIntID ID

	repoConfig
}

func (repo *MemoryRepository[E, ID]) getID(t any) ID { //nolint:ireturn,lll // fp, as it is not recognised even with "generic" setting
	val := reflect.ValueOf(t)

	idField := val.FieldByName(repo.idFieldName)
	if reflect.DeepEqual(idField, reflect.Value{}) { //nolint:govet,lll // is a fp and will be fixed, see: https://github.com/golang/go/issues/43993
		panic("entity does not have the field with name: " + repo.idFieldName)
	}

	var id ID

	switch idField.Kind() { //nolint:exhaustive
	case reflect.String:
		reflect.ValueOf(&id).Elem().SetString(idField.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		reflect.ValueOf(&id).Elem().SetInt(idField.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		reflect.ValueOf(&id).Elem().SetUint(idField.Uint())
	default:
		panic("type of ID is not supported: " + idField.Kind().String())
	}

	return id
}

// NextID returns a new ID. It can be of the underlying type of string or integer.
func (repo *MemoryRepository[E, ID]) NextID(_ context.Context) (ID, error) { //nolint:ireturn,lll // fp, as it is not recognised even with "generic" setting
	var id ID

	switch reflect.TypeOf(id).Kind() { //nolint:exhaustive
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
		panic("type of ID is not supported: " + reflect.TypeOf(id).Kind().String())
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

	repo.Data[id] = entity

	return nil
}

func (repo *MemoryRepository[E, ID]) Delete(_ context.Context, entity E) error {
	repo.Lock()
	defer repo.Unlock()

	delete(repo.Data, repo.getID(entity))

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

	if t, ok := repo.Data[id]; ok {
		return t, nil
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

	return false, ErrNotFound
}

func (repo *MemoryRepository[E, ID]) ExistsByID(ctx context.Context, id ID) (bool, error) {
	return repo.Exists(ctx, id)
}

func (repo *MemoryRepository[E, ID]) ExistAll(_ context.Context, ids []ID) (bool, error) {
	repo.Lock()
	defer repo.Unlock()

	if len(ids) == 0 {
		return false, nil
	}

	for _, id := range ids {
		if _, ok := repo.Data[id]; !ok {
			return false, ErrNotFound
		}
	}

	return true, nil
}

func (repo *MemoryRepository[E, ID]) ExistByIDs(ctx context.Context, ids []ID) (bool, error) {
	return repo.ExistAll(ctx, ids)
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

func (repo *MemoryRepository[E, ID]) Count(_ context.Context) (int, error) {
	repo.Lock()
	defer repo.Unlock()

	return len(repo.Data), nil
}

func (repo *MemoryRepository[E, ID]) Length(ctx context.Context) (int, error) {
	return repo.Count(ctx)
}

func (repo *MemoryRepository[E, ID]) Empty(ctx context.Context) (bool, error) {
	return repo.IsEmpty(ctx)
}

func (repo *MemoryRepository[E, ID]) IsEmpty(ctx context.Context) (bool, error) {
	c, err := repo.Count(ctx)

	return c == 0, err
}

func (repo *MemoryRepository[E, ID]) Save(_ context.Context, entity E) error {
	repo.Lock()
	defer repo.Unlock()

	id := repo.getID(entity)
	if id == *new(ID) {
		return fmt.Errorf("missing ID: %w", ErrSaveFailed)
	}

	repo.Data[id] = entity

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

	for _, e := range entities {
		repo.Data[repo.getID(e)] = e
	}

	return nil
}

func (repo *MemoryRepository[E, ID]) UpdateAll(ctx context.Context, entities []E) error {
	return repo.SaveAll(ctx, entities)
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

func (repo *MemoryRepository[E, ID]) DeleteByID(ctx context.Context, id ID) error {
	return repo.DeleteByIDs(ctx, []ID{id})
}

func (repo *MemoryRepository[E, ID]) DeleteByIDs(_ context.Context, ids []ID) error {
	repo.Lock()
	defer repo.Unlock()

	for _, id := range ids {
		delete(repo.Data, id)
	}

	return nil
}

func (repo *MemoryRepository[E, ID]) DeleteAll(_ context.Context) error {
	repo.Lock()
	defer repo.Unlock()

	repo.Data = make(map[ID]E)

	return nil
}

func (repo *MemoryRepository[E, ID]) Clear(ctx context.Context) error {
	return repo.DeleteAll(ctx)
}
