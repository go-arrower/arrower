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
type Repository[E any, ID ~string] interface { //nolint:interfacebloat // showcase of all methods that are possible
	NextID(context.Context) ID

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

	Count(context.Context) (int, error)
	Length(context.Context) (int, error)
	Empty(context.Context) (bool, error)
	IsEmpty(context.Context) (bool, error)

	Save(context.Context, E) error
	SaveAll(context.Context, []E) error
	UpdateAll(context.Context, []E) error
	Add(context.Context, E) error
	AddAll(context.Context, []E) error

	DeleteByID(context.Context, ID) error
	DeleteByIDs(context.Context, []ID) error
	DeleteAll(context.Context) error
}

type MemoryRepositoryOption func(*repoConfig)

type repoConfig struct {
	idFieldName string
}

func WithIDField(c string) MemoryRepositoryOption {
	return func(repo *repoConfig) {
		repo.idFieldName = c
	}
}

// NewMemoryRepository returns an implementation of MemoryRepository for the given entity E.
// It is expected that E has a field called `ID`, that is used as the primary key and can
// be overwritten by WithIDField.
// If your repository needs additional methods, you can embed this repo into our own implementation to extend
// your own repository easily to your use case.
func NewMemoryRepository[E any, ID ~string](opts ...MemoryRepositoryOption) *MemoryRepository[E, ID] {
	repo := &MemoryRepository[E, ID]{
		Mutex:      &sync.Mutex{},
		data:       make(map[ID]E),
		repoConfig: repoConfig{idFieldName: "ID"},
	}

	for _, opt := range opts {
		opt(&repo.repoConfig)
	}

	return repo
}

// MemoryRepository implements Repository in a generic way. Use it to speed up your unit testing.
type MemoryRepository[E any, ID ~string] struct {
	*sync.Mutex // The mutex is embedded, so that repositories who extend MemoryRepository can lock the same mutex.
	data        map[ID]E

	repoConfig
}

func (repo *MemoryRepository[E, ID]) getID(t any) ID { //nolint:ireturn // valid use of generics
	val := reflect.ValueOf(t)

	idField := val.FieldByName(repo.idFieldName)
	if reflect.DeepEqual(idField, reflect.Value{}) { //nolint:govet,lll // is a fp and will be fixed, see: https://github.com/golang/go/issues/43993
		panic("entity does not have the field " + repo.idFieldName)
	}

	return ID(idField.String())
}

func (repo *MemoryRepository[E, ID]) NextID(ctx context.Context) ID { //nolint:ireturn // valid use of generics
	return ID(uuid.New().String())
}

func (repo *MemoryRepository[E, ID]) Create(ctx context.Context, entity E) error {
	repo.Lock()
	defer repo.Unlock()

	id := repo.getID(entity)
	if id == "" {
		return fmt.Errorf("missing ID: %w", ErrSaveFailed)
	}

	if _, found := repo.data[id]; found {
		return ErrAlreadyExists
	}

	repo.data[id] = entity

	return nil
}

func (repo *MemoryRepository[E, ID]) Read(ctx context.Context, id ID) (E, error) { //nolint:ireturn,lll // valid use of generics
	return repo.FindByID(ctx, id)
}

func (repo *MemoryRepository[E, ID]) Update(ctx context.Context, entity E) error {
	repo.Lock()
	defer repo.Unlock()

	id := repo.getID(entity)
	if id == "" {
		return fmt.Errorf("missing ID: %w", ErrSaveFailed)
	}

	if _, found := repo.data[id]; !found {
		return fmt.Errorf("entity does not exist yet: %w", ErrSaveFailed)
	}

	repo.data[id] = entity

	return nil
}

func (repo *MemoryRepository[E, ID]) Delete(ctx context.Context, entity E) error {
	repo.Lock()
	defer repo.Unlock()

	delete(repo.data, repo.getID(entity))

	return nil
}

func (repo *MemoryRepository[E, ID]) All(ctx context.Context) ([]E, error) {
	return repo.FindAll(ctx)
}

func (repo *MemoryRepository[E, ID]) AllByIDs(ctx context.Context, ids []ID) ([]E, error) {
	return repo.FindByIDs(ctx, ids)
}

func (repo *MemoryRepository[E, ID]) FindAll(ctx context.Context) ([]E, error) {
	result := []E{}

	for _, v := range repo.data {
		result = append(result, v)
	}

	return result, nil
}

func (repo *MemoryRepository[E, ID]) FindByID(ctx context.Context, id ID) (E, error) { //nolint:ireturn,lll // valid use of generics
	if t, ok := repo.data[id]; ok {
		return t, nil
	}

	return *new(E), ErrNotFound //nolint:gocritic,lll // looks like a false positive or the linter does not deal with generics yet // validation error is returned on purpose
}

func (repo *MemoryRepository[E, ID]) FindByIDs(ctx context.Context, ids []ID) ([]E, error) {
	result := []E{}

	for _, v := range repo.data {
		for _, id := range ids {
			if repo.getID(v) == id {
				result = append(result, v)
			}
		}
	}

	return result, nil
}

func (repo *MemoryRepository[E, ID]) Exists(ctx context.Context, id ID) (bool, error) {
	if _, ok := repo.data[id]; ok {
		return true, nil
	}

	return false, ErrNotFound
}

func (repo *MemoryRepository[E, ID]) ExistsByID(ctx context.Context, id ID) (bool, error) {
	return repo.Exists(ctx, id)
}

func (repo *MemoryRepository[E, ID]) ExistAll(ctx context.Context, ids []ID) (bool, error) {
	if len(ids) == 0 {
		return false, nil
	}

	for _, id := range ids {
		if _, ok := repo.data[id]; !ok {
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
	return repo.ExistByIDs(ctx, ids)
}

func (repo *MemoryRepository[E, ID]) Count(ctx context.Context) (int, error) {
	return len(repo.data), nil
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

func (repo *MemoryRepository[E, ID]) Save(ctx context.Context, entity E) error {
	repo.Lock()
	defer repo.Unlock()

	id := repo.getID(entity)
	if id == "" {
		return fmt.Errorf("missing ID: %w", ErrSaveFailed)
	}

	repo.data[id] = entity

	return nil
}

func (repo *MemoryRepository[E, ID]) SaveAll(ctx context.Context, entities []E) error {
	repo.Lock()
	defer repo.Unlock()

	for _, e := range entities {
		if repo.getID(e) == "" {
			return fmt.Errorf("missing ID: %w", ErrSaveFailed)
		}
	}

	for _, e := range entities {
		repo.data[repo.getID(e)] = e
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
		ex, err := repo.Exists(ctx, repo.getID(e))
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}

		if ex {
			return ErrAlreadyExists
		}
	}

	return repo.SaveAll(ctx, entities)
}

func (repo *MemoryRepository[E, ID]) DeleteByID(ctx context.Context, id ID) error {
	return repo.DeleteByIDs(ctx, []ID{id})
}

func (repo *MemoryRepository[E, ID]) DeleteByIDs(ctx context.Context, ids []ID) error {
	repo.Lock()
	defer repo.Unlock()

	for _, id := range ids {
		delete(repo.data, id)
	}

	return nil
}

func (repo *MemoryRepository[E, ID]) DeleteAll(ctx context.Context) error {
	repo.Lock()
	defer repo.Unlock()

	repo.data = make(map[ID]E)

	return nil
}
