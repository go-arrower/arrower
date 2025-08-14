package repository

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/go-arrower/arrower/repository/q"
)

var (
	ErrStorage       = errors.New("storage error")
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)

// Option takes in a repository to set different optional properties.
// The WithOption function is required to cast the parameter,
// to ensure the option is only applied to a repository, where it is supported.
type Option func(any) error

// WithIDField set's the name of the field that is used as an id or primary key.
// If not set, it is assumed that the entity struct has a field with the name "ID".
func WithIDField(idFieldName string) Option {
	return func(repoConfig any) error {
		idField := reflect.ValueOf(repoConfig).Elem().FieldByName("IDFieldName")
		if !idField.CanSet() || idField.Kind() != reflect.String {
			return errSetIDFieldFailed
		}

		idField.SetString(idFieldName)

		return nil
	}
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
	AllBy(ctx context.Context, query q.Query) ([]E, error)
	FindAll(ctx context.Context) ([]E, error)
	// FindAllBy
	FindByID(ctx context.Context, id ID) (E, error)
	FindByIDs(ctx context.Context, ids []ID) ([]E, error)
	FindBy(ctx context.Context, query q.Query) ([]E, error)
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
	// CountBy
	Length(ctx context.Context) (int, error)

	// DeleteByID removes the entity with the given it from the repository.
	// If no ID is set, nothing happens and no error is returned.
	DeleteByID(ctx context.Context, id ID) error
	DeleteByIDs(ctx context.Context, ids []ID) error
	// DeleteBy
	DeleteAll(ctx context.Context) error
	Clear(ctx context.Context) error

	// IterBy(ctx context.Context, query q.Query) Iterator[E, ID]
	// AllByIter
	AllIter(ctx context.Context) Iterator[E, ID]
	// FindAllByIter
	// FindByIter
	FindAllIter(ctx context.Context) Iterator[E, ID]
}

type Iterator[E any, ID id] interface {
	Next() func(yield func(e E, err error) bool)
}

// id are the types allowed as a primary key used in the generic Repository.
type id interface {
	~string |
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

var (
	errInvalidOption      = fmt.Errorf("%w: invalid option", ErrStorage)
	errSetIDFieldFailed   = errors.New("cannot set IDField: repository MUST have a field `IDFieldName` of type string")
	errIDGenerationFailed = fmt.Errorf("%w: id generation failed", ErrStorage)
	errIDFailed           = fmt.Errorf("%w: accessing ID failed", ErrStorage)
	errCreateFailed       = fmt.Errorf("%w: create failed", ErrStorage)
	errFindFailed         = fmt.Errorf("%w: find failed", ErrStorage)
	errUpdateFailed       = fmt.Errorf("%w: update failed", ErrStorage)
	errDeleteFailed       = fmt.Errorf("%w: delete failed", ErrStorage)
	errExistsFailed       = fmt.Errorf("%w: exists failed", ErrStorage)
	errSaveFailed         = fmt.Errorf("%w: save failed", ErrStorage)
	errCountFailed        = fmt.Errorf("%w: count failed", ErrStorage)
	errInvalidQuery       = fmt.Errorf("%w: invalid query", ErrStorage)
)
