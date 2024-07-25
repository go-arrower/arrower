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

func NewPostgresRepository[E any, ID id](pgx *pgxpool.Pool, opts ...Option) *PostgresRepository[E, ID] {
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

func (repo *PostgresRepository[E, ID]) NextID(ctx context.Context) (ID, error) {
	return *new(ID), nil
}

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

func (repo *PostgresRepository[E, ID]) Update(ctx context.Context, entity E) error {
	return nil
}

func (repo *PostgresRepository[E, ID]) Delete(ctx context.Context, entity E) error {
	return nil
}

func (repo *PostgresRepository[E, ID]) All(ctx context.Context) ([]E, error) {
	return nil, nil
}

func (repo *PostgresRepository[E, ID]) AllByIDs(ctx context.Context, ids []ID) ([]E, error) {
	return nil, nil
}

func (repo *PostgresRepository[E, ID]) FindAll(ctx context.Context) ([]E, error) {
	return nil, nil
}

func (repo *PostgresRepository[E, ID]) FindByID(ctx context.Context, id ID) (E, error) {
	return *new(E), nil
}

func (repo *PostgresRepository[E, ID]) FindByIDs(ctx context.Context, ids []ID) ([]E, error) {
	return nil, nil
}

func (repo *PostgresRepository[E, ID]) Exists(ctx context.Context, id ID) (bool, error) {
	return false, nil
}

func (repo *PostgresRepository[E, ID]) ExistsByID(ctx context.Context, id ID) (bool, error) {
	return false, nil
}

func (repo *PostgresRepository[E, ID]) ExistByIDs(ctx context.Context, ids []ID) (bool, error) {
	return false, nil
}

func (repo *PostgresRepository[E, ID]) ExistAll(ctx context.Context, ids []ID) (bool, error) {
	return false, nil
}

func (repo *PostgresRepository[E, ID]) Contains(ctx context.Context, id ID) (bool, error) {
	return false, nil
}

func (repo *PostgresRepository[E, ID]) ContainsID(ctx context.Context, id ID) (bool, error) {
	return false, nil
}

func (repo *PostgresRepository[E, ID]) ContainsIDs(ctx context.Context, ids []ID) (bool, error) {
	return false, nil
}

func (repo *PostgresRepository[E, ID]) ContainsAll(ctx context.Context, ids []ID) (bool, error) {
	return false, nil
}

func (repo *PostgresRepository[E, ID]) Save(ctx context.Context, entity E) error {
	return nil
}

func (repo *PostgresRepository[E, ID]) SaveAll(ctx context.Context, entities []E) error {
	return nil
}

func (repo *PostgresRepository[E, ID]) UpdateAll(ctx context.Context, entities []E) error {
	return nil
}

func (repo *PostgresRepository[E, ID]) Add(ctx context.Context, entity E) error {
	return nil
}

func (repo *PostgresRepository[E, ID]) AddAll(ctx context.Context, entities []E) error {
	return nil
}

func (repo *PostgresRepository[E, ID]) Count(ctx context.Context) (int, error) {
	return 0, nil
}

func (repo *PostgresRepository[E, ID]) Length(ctx context.Context) (int, error) {
	return 0, nil
}

func (repo *PostgresRepository[E, ID]) Empty(ctx context.Context) (bool, error) {
	return false, nil
}

func (repo *PostgresRepository[E, ID]) IsEmpty(ctx context.Context) (bool, error) {
	return false, nil
}

func (repo *PostgresRepository[E, ID]) DeleteByID(ctx context.Context, id ID) error {
	return nil
}

func (repo *PostgresRepository[E, ID]) DeleteByIDs(ctx context.Context, ids []ID) error {
	return nil
}

func (repo *PostgresRepository[E, ID]) DeleteAll(ctx context.Context) error {
	return nil
}

func (repo *PostgresRepository[E, ID]) Clear(ctx context.Context) error {
	return nil
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
