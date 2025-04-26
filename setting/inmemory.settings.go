package setting

import (
	"context"

	"github.com/go-arrower/arrower/repository"
)

func NewInMemorySettings() *InMemorySettings {
	return &InMemorySettings{
		repo: &inMemoryRepository{MemoryRepository: repository.NewMemoryRepository[Value, Key]()},
	}
}

var _ Settings = (*InMemorySettings)(nil)

type InMemorySettings struct {
	repo *inMemoryRepository
}

func (s *InMemorySettings) Register(ctx context.Context, settings map[Key]any) error {
	for k, v := range settings {
		if exists, _ := s.repo.ExistsByID(ctx, k); !exists {
			_ = s.Save(ctx, k, v)
		}
	}

	return nil
}

func (s *InMemorySettings) Save(ctx context.Context, key Key, value any) error {
	if val, ok := value.(Value); ok {
		return s.repo.Save(ctx, key, val)
	}

	return s.repo.Save(ctx, key, NewValue(value))
}

func (s *InMemorySettings) Setting(ctx context.Context, key Key) (Value, error) {
	v, err := s.repo.FindByID(ctx, key)
	if err != nil {
		return Value{}, ErrNotFound
	}

	return v, nil
}

func (s *InMemorySettings) Settings(ctx context.Context, keys []Key) (map[Key]Value, error) {
	settings := make(map[Key]Value, len(keys))
	hasNotFound := false

	for _, key := range keys {
		value, err := s.repo.FindByID(ctx, key)
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
	return s.repo.DeleteByID(ctx, key) //nolint:wrapcheck
}

type inMemoryRepository struct {
	*repository.MemoryRepository[Value, Key]
}

func (repo *inMemoryRepository) Save(_ context.Context, key Key, value Value) error {
	repo.Lock()
	defer repo.Unlock()

	repo.Data[key] = value

	return nil
}
