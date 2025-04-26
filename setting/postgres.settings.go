package setting

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
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

var psql = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar) //nolint:gochecknoglobals,lll // squirrel recommends this

func (s *PostgresSettings) Register(ctx context.Context, settings map[Key]any) error { //nolint:wsl
	// sqlc does not support atomic import:
	// sqlc can do copyform, but that does not have ON CONFLICT support
	// a workaround with temporary tables, does not worj with sqlc,
	// see: https://github.com/sqlc-dev/sqlc/issues/2165
	// => use squirrel instead

	query := psql.Insert("arrower.setting").Columns("key", "value")

	for key, v := range settings {
		var valParam string

		if val, ok := v.(Value); ok {
			valParam = val.Raw()
		} else {
			valParam = NewValue(v).Raw()
		}

		query = query.Values(key, valParam)
	}

	query = query.Suffix("ON CONFLICT (key) DO NOTHING")

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("%w: could not build query: %v", ErrOperationFailed, err)
	}

	_, err = s.repo.TxOrConn(ctx).DB().Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%w: could not register settings: %v", ErrOperationFailed, err)
	}

	return nil
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
