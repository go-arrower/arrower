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

// Repository is a general purpose interface documenting which methods are available by the generic MemoryRepository.
// ID is the primary key and needs to be of one of the underlying types.
// If your repository needs additional methods, you can extend your own repository easily to tune it to your use case.
// See the examples in the test files.
type Repository[E any, ID id] interface { //nolint:interfacebloat // showcase of all methods that are possible
	NextID(ctx context.Context) (ID, error)

	Create(ctx context.Context, entity E) error
	Read(ctx context.Context, id ID) (E, error)
	Update(ctx context.Context, entity E) error
	Delete(ctx context.Context, entity E) error

	All(ctx context.Context) ([]E, error)
	AllByIDs(ctx context.Context, ids []ID) ([]E, error)
	FindAll(ctx context.Context) ([]E, error)
	FindByID(ctx context.Context, id ID) (E, error)
	FindByIDs(ctx context.Context, ids []ID) ([]E, error)
	Exists(ctx context.Context, id ID) (bool, error)
	ExistsByID(ctx context.Context, id ID) (bool, error)
	ExistByIDs(ctx context.Context, ids []ID) (bool, error)
	ExistAll(ctx context.Context, ids []ID) (bool, error)
	Contains(ctx context.Context, id ID) (bool, error)
	ContainsID(ctx context.Context, id ID) (bool, error)
	ContainsIDs(ctx context.Context, ids []ID) (bool, error)
	ContainsAll(ctx context.Context, ids []ID) (bool, error)

	Save(ctx context.Context, entity E) error
	SaveAll(ctx context.Context, entities []E) error
	UpdateAll(ctx context.Context, entities []E) error
	Add(ctx context.Context, entity E) error
	AddAll(ctx context.Context, entities []E) error

	Count(ctx context.Context) (int, error)
	Length(ctx context.Context) (int, error)
	Empty(ctx context.Context) (bool, error)
	IsEmpty(ctx context.Context) (bool, error)

	DeleteByID(ctx context.Context, id ID) error
	DeleteByIDs(ctx context.Context, ids []ID) error
	DeleteAll(ctx context.Context) error
	Clear(ctx context.Context) error
}

// WithIDField set's the name of the field that is used as an id or primary key.
// If not set, it is assumed that the entity struct has a field with the name "ID".
func WithIDField(idFieldName string) memoryRepositoryOption { //nolint:revive // unexported-return is OK for this option
	return func(config *repoConfig) {
		config.idFieldName = idFieldName
	}
}

// WithStore sets a Store used to persist the Repository.
// There are no transactions or any consistency guarantees at all! For example, if a store fails,
// the collection is still changed in memory of the repository.
func WithStore(store Store) memoryRepositoryOption { //nolint:revive // unexported-return is OK for this option
	return func(config *repoConfig) {
		config.store = store
	}
}

// WithStoreFilename overwrites the file name a Store should use to persist this Repository.
func WithStoreFilename(name string) memoryRepositoryOption { //nolint:revive // unexported-return is OK for this option
	return func(config *repoConfig) {
		config.filename = name
	}
}

// id are the types allowed as a primary key used in the generic Repository.
type id interface {
	~string |
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

type memoryRepositoryOption func(*repoConfig)

type repoConfig struct {
	idFieldName string
	store       Store
	filename    string
}

// NewMemoryRepository returns an implementation of MemoryRepository for the given entity E.
// It is expected that E has a field called `ID`, that is used as the primary key and can
// be overwritten by WithIDField.
// If your repository needs additional methods, you can embed this repo into our own implementation to extend
// your own repository easily to your use case. See the examples in the test files.
func NewMemoryRepository[E any, ID id](opts ...memoryRepositoryOption) *MemoryRepository[E, ID] {
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

func (repo *MemoryRepository[E, ID]) getID(t any) ID { //nolint:dupl,ireturn,lll // needs acces to the type ID and fp, as it is not recognised even with "generic" setting
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

	repo.Data[id] = entity

	err := repo.store.Store(repo.filename, repo.Data)
	if err != nil {
		return fmt.Errorf("could not save: %w", err)
	}

	return nil
}

func (repo *MemoryRepository[E, ID]) Delete(_ context.Context, entity E) error {
	repo.Lock()
	defer repo.Unlock()

	delete(repo.Data, repo.getID(entity))

	err := repo.store.Store(repo.filename, repo.Data)
	if err != nil {
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

	err := repo.store.Store(repo.filename, repo.Data)
	if err != nil {
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

	for _, e := range entities {
		repo.Data[repo.getID(e)] = e
	}

	err := repo.store.Store(repo.filename, repo.Data)
	if err != nil {
		return fmt.Errorf("could not save: %w", err)
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

	err := repo.store.Store(repo.filename, repo.Data)
	if err != nil {
		return fmt.Errorf("could not save: %w", err)
	}

	return nil
}

func (repo *MemoryRepository[E, ID]) DeleteAll(_ context.Context) error {
	repo.Lock()
	defer repo.Unlock()

	clear(repo.Data)

	err := repo.store.Store(repo.filename, repo.Data)
	if err != nil {
		return fmt.Errorf("could not save: %w", err)
	}

	return nil
}

func (repo *MemoryRepository[E, ID]) Clear(ctx context.Context) error {
	return repo.DeleteAll(ctx)
}
