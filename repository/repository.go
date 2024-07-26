package repository

import "context"

// id are the types allowed as a primary key used in the generic Repository.
type id interface {
	~string |
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
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
	Empty(ctx context.Context) (bool, error)   // todo assertion?
	IsEmpty(ctx context.Context) (bool, error) // todo assertion?

	DeleteByID(ctx context.Context, id ID) error
	DeleteByIDs(ctx context.Context, ids []ID) error
	DeleteAll(ctx context.Context) error
	Clear(ctx context.Context) error
}
