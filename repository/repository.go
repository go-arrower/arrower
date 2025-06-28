package repository

import "context"

// Condition is an interface that filters can implement to influence the
// selected entities that the Repository returns.
type Condition[T any] interface {
	Filter() T
	OrderBy() string
}

func Filter[T any](m T) Condition[T] {
	return filter[T]{model: m, orderBy: ""}
}

func OrderBy[T any](m string) Condition[T] {
	return filter[T]{orderBy: m, model: *new(T)}
}

type filter[T any] struct {
	model   T
	orderBy string
}

func (f filter[T]) Filter() T { //nolint:ireturn // type param of OrderBy is any, but irrelevant
	return f.model
}

func (f filter[T]) OrderBy() string {
	return f.orderBy
}

// id are the types allowed as a primary key used in the generic Repository.
type id interface {
	~string |
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// WithIDField set's the name of the field that is used as an id or primary key.
// If not set, it is assumed that the entity struct has a field with the name "ID".
func WithIDField(idFieldName string) Option {
	return func(config *repoConfig) {
		config.IDFieldName = idFieldName
	}
}

type Option func(*repoConfig)

type repoConfig struct {
	IDFieldName string
	store       Store
	filename    string
}

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
	// AllBy
	AllByIDs(ctx context.Context, ids []ID) ([]E, error)
	FindAll(ctx context.Context) ([]E, error)
	FindBy(ctx context.Context, filters ...Condition[E]) ([]E, error)
	// FindAllBy
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

	CreateAll(ctx context.Context, entities []E) error
	Save(ctx context.Context, entity E) error
	SaveAll(ctx context.Context, entities []E) error
	UpdateAll(ctx context.Context, entities []E) error
	Add(ctx context.Context, entity E) error
	AddAll(ctx context.Context, entities []E) error

	Count(ctx context.Context) (int, error)
	Length(ctx context.Context) (int, error)

	// DeleteBy
	DeleteByID(ctx context.Context, id ID) error
	DeleteByIDs(ctx context.Context, ids []ID) error
	DeleteAll(ctx context.Context) error
	Clear(ctx context.Context) error

	// AllByIter
	AllIter(ctx context.Context) Iterator[E, ID]
	// FindAllByIter
	FindAllIter(ctx context.Context) Iterator[E, ID]
}

type Iterator[E any, ID id] interface {
	Next() func(yield func(e E, err error) bool)
}
