package setting

import (
	"context"

	"github.com/go-arrower/arrower/tests"
)

func NewInMemorySettings() *InMemorySettings {
	return &InMemorySettings{
		repo: &inMemoryRepository{MemoryRepository: tests.NewMemoryRepository[Value, string]()},
	}
}

var _ Settings = (*InMemorySettings)(nil)

type InMemorySettings struct {
	repo *inMemoryRepository
}

func (s *InMemorySettings) Save(ctx context.Context, key Key, setting Value) error {
	return s.repo.Save(ctx, key, setting)
}

func (s *InMemorySettings) Setting(ctx context.Context, key Key) (Value, error) {
	v, err := s.repo.FindByID(ctx, key.Key())
	if err != nil {
		return Value{}, ErrNotFound
	}

	return v, nil
}

func (s *InMemorySettings) Settings(ctx context.Context, keys []Key) (map[Key]Value, error) {
	settings := make(map[Key]Value, len(keys))
	hasNotFound := false

	for _, key := range keys {
		value, err := s.repo.FindByID(ctx, key.Key())
		if err != nil {
			hasNotFound = true

			continue
		}

		settings[key] = value
	}

	if hasNotFound {
		return settings, ErrNotFound
	}

	return settings, nil
}

func (s *InMemorySettings) Delete(ctx context.Context, key Key) error {
	return s.repo.DeleteByID(ctx, key.Key()) //nolint:wrapcheck
}

type inMemoryRepository struct {
	*tests.MemoryRepository[Value, string]
}

func (repo *inMemoryRepository) Save(_ context.Context, key Key, value Value) error {
	repo.Lock()
	defer repo.Unlock()

	repo.Data[key.Key()] = value

	return nil
}
