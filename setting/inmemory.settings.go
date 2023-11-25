package setting

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-arrower/arrower/tests"
)

func NewInMemorySettings() *SettingsHandler {
	return &SettingsHandler{
		repo:     &inMemoryRepository{MemoryRepository: tests.NewMemoryRepository[Value, string]()},
		mu:       sync.Mutex{},
		onChange: make(map[Key][]func(Value)),
	}
}

var _ Settings = (*SettingsHandler)(nil)

type SettingsHandler struct {
	repo repository

	mu       sync.Mutex
	onChange map[Key][]func(Value)
}

func (s *SettingsHandler) Save(ctx context.Context, key Key, setting Value) error {
	current, err := s.Setting(ctx, key)
	if err != nil {
		return fmt.Errorf("could not save setting: %w", err)
	}

	settingChanged := current.String() != setting.String()
	if settingChanged {
		go s.notifyOnConfigChange(key, setting)
	}

	return s.repo.Save(ctx, key, setting)
}

func (s *SettingsHandler) Setting(ctx context.Context, key Key) (Value, error) {
	return s.repo.FindByID(ctx, key)
}

func (s *SettingsHandler) OnSettingChange(key Key, callback func(setting Value)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.onChange[key] = append(s.onChange[key], callback)
}

func (s *SettingsHandler) notifyOnConfigChange(key Key, setting Value) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, c := range s.onChange[key] {
		c(setting)
	}
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

func (repo *inMemoryRepository) FindByID(ctx context.Context, key Key) (Value, error) {
	return repo.MemoryRepository.FindByID(ctx, key.Key())
}
