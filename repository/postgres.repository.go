package repository

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInvalidQuery = errors.New("invalid query")

func NewPostgresRepository[E any, ID id](pgx *pgxpool.Pool, opts ...repositoryOption) *PostgresRepository[E, ID] {
	repo := &PostgresRepository[E, ID]{
		pgx: pgx,
		// logger:  alog.NewNoopLogger().WithGroup("arrower.repository"),
		table:   tableName(*new(E)),
		columns: columnNames(*new(E)),
		repoConfig: repoConfig{ //nolint:exhaustruct // other configs not supported by postgres implementation.
			idFieldName: "ID",
		},
	}

	for _, opt := range opts {
		opt(&repo.repoConfig)
	}

	return repo
}

type PostgresRepository[E any, ID id] struct {
	// logger alog.Logger
	pgx *pgxpool.Pool

	repoConfig
	table   string
	columns []string
}

var psql = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar) //nolint:gochecknoglobals,lll // squirrel recommends this

func (repo *PostgresRepository[E, ID]) Create(ctx context.Context, entity E) error {
	values := make([]any, len(repo.columns))

	e := reflect.ValueOf(entity)
	for i, name := range repo.columns {
		values[i] = e.FieldByName(name).Interface()
	}

	query, args, err := psql.Insert(repo.table).Columns(repo.columns...).Values(values...).ToSql()
	if err != nil {
		return fmt.Errorf("%w: could not build query: %v", ErrInvalidQuery, err) //nolint:errorlint // prevent err in api
	}

	// repo.logger.LogAttrs(ctx, alog.LevelDebug, "create entity", slog.String("query", query), slog.Any("args", args))

	_, err = repo.pgx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%w: could not insert entity: %v", ErrInvalidQuery, err) //nolint:errorlint // prevent err in api
	}

	return nil
}

func (repo *PostgresRepository[E, ID]) Read(ctx context.Context, id ID) (E, error) { //nolint:ireturn,lll // valid use of generics
	query, args, err := psql.Select(repo.columns...).From(repo.table).Where(squirrel.Eq{repo.idFieldName: id}).ToSql()
	if err != nil {
		return *new(E), fmt.Errorf("%w: could not build query: %v", ErrInvalidQuery, err) //nolint:errorlint,lll // prevent err in api
	}

	// repo.logger.LogAttrs(ctx, alog.LevelDebug, "read entity", slog.String("query", query), slog.Any("args", args))

	entity := *new(E)

	err = pgxscan.Get(ctx, repo.pgx, &entity, query, args...)
	if err != nil {
		return *new(E), fmt.Errorf("%w: could not scan entity: %v", ErrInvalidQuery, err) //nolint:errorlint,lll // prevent err in api
	}

	return entity, nil
}

func tableName[E any](entity E) string {
	return strings.ToLower(reflect.TypeOf(entity).Name())
}

func columnNames[E any](entity E) []string {
	columns := []string{}

	e := reflect.TypeOf(entity)
	for i := range e.NumField() {
		columns = append(columns, e.Field(i).Name)
	}

	return columns
}
