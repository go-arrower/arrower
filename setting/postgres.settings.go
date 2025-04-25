package setting

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-arrower/arrower/postgres"
	"github.com/go-arrower/arrower/setting/models"
)

var ErrOperationFailed = errors.New("operation failed")

func NewPostgresSettings(pgxPool *pgxpool.Pool) *PostgresSettings {
	return &PostgresSettings{
		repo: postgres.NewPostgresBaseRepository(models.New(pgxPool)),
	}
}

var _ Settings = (*PostgresSettings)(nil)

type PostgresSettings struct {
	repo postgres.BaseRepository[*models.Queries]
}

func (s *PostgresSettings) Save(ctx context.Context, key Key, value any) error {
	var valParam string

	if val, ok := value.(Value); ok {
		valParam = val.Raw()
	} else {
		valParam = NewValue(value).Raw()
	}

	err := s.repo.TxOrConn(ctx).UpsertSetting(ctx, models.UpsertSettingParams{
		Key:   string(key),
		Value: valParam,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrOperationFailed, err)
	}

	return nil
}

func (s *PostgresSettings) Setting(ctx context.Context, key Key) (Value, error) {
	value, err := s.repo.TxOrConn(ctx).GetSetting(ctx, string(key))
	if err != nil {
		return NewValue(nil), fmt.Errorf("%w: %v", ErrNotFound, err)
	}

	return NewValue(value), nil
}

func (s *PostgresSettings) Settings(ctx context.Context, keys []Key) (map[Key]Value, error) {
	compositeKeys := make([]string, len(keys)) // the database
	for i, k := range keys {
		compositeKeys[i] = string(k)
	}

	dbSettings, err := s.repo.TxOrConn(ctx).GetSettings(ctx, compositeKeys)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOperationFailed, err)
	}

	settings := make(map[Key]Value)

	for _, s := range dbSettings {
		k := strings.Split(s.Key, ".")
		settings[NewKey(k[0], k[1], k[2])] = NewValue(s.Value)
	}

	hasNotFound := len(keys) != len(settings)
	if hasNotFound {
		return settings, ErrNotFound
	}

	return settings, nil
}

func (s *PostgresSettings) Delete(ctx context.Context, key Key) error {
	err := s.repo.TxOrConn(ctx).DeleteSetting(ctx, string(key))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrOperationFailed, err)
	}

	return nil
}
