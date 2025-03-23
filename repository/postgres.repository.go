package repository

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-arrower/arrower/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidQuery = errors.New("invalid query")

	errIDFieldWrong = errors.New("the ID field used as primary key is wrong")
)

func NewPostgresRepository[E any, ID id](pgx *pgxpool.Pool, opts ...Option) *PostgresRepository[E, ID] {
	repo := &PostgresRepository[E, ID]{
		PGx: pgx,
		// logger:  alog.NewNoopLogger().WithGroup("arrower.repository"),
		Table:   tableName(*new(E)),
		Columns: columnNames(*new(E)),
		repoConfig: repoConfig{ //nolint:exhaustruct // other configs not supported by postgres implementation.
			IDFieldName: "ID",
		},
	}

	for _, opt := range opts {
		opt(&repo.repoConfig)
	}

	return repo
}

type PostgresRepository[E any, ID id] struct {
	// logger alog.Logger
	PGx *pgxpool.Pool

	repoConfig
	Table   string
	Columns []string
}

type dbInterface interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

func (repo *PostgresRepository[E, ID]) TxOrConn(ctx context.Context) dbInterface {
	tx, ok := ctx.Value(postgres.CtxTX).(pgx.Tx)
	if ok {
		return tx
	}

	return repo.PGx
}

var psql = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar) //nolint:gochecknoglobals,lll // squirrel recommends this

func (repo *PostgresRepository[E, ID]) getID(t any) (ID, error) { //nolint:ireturn,lll // needs access to the type ID and fp, as it is not recognised even with "generic" setting
	val := reflect.ValueOf(t)

	idField := val.FieldByName(repo.IDFieldName)
	if reflect.DeepEqual(idField, reflect.Value{}) { //nolint:govet,lll // is a fp and will be fixed, see: https://github.com/golang/go/issues/43993
		return *new(ID), fmt.Errorf("%w: entity does not have the field with name: %s", errIDFieldWrong, repo.IDFieldName)
	}

	var id ID

	switch idField.Kind() {
	case reflect.String:
		reflect.ValueOf(&id).Elem().SetString(idField.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		reflect.ValueOf(&id).Elem().SetInt(idField.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		reflect.ValueOf(&id).Elem().SetUint(idField.Uint())
	default:
		return *new(ID), fmt.Errorf("%w: %s: %v", errIDFieldWrong, "type of ID is not supported", idField.Kind())
	}

	if id == *new(ID) {
		return *new(ID), fmt.Errorf("%w: missing ID", errIDFieldWrong)
	}

	return id, nil
}

func (repo *PostgresRepository[E, ID]) NextID(ctx context.Context) (ID, error) {
	var id ID

	switch reflect.TypeOf(id).Kind() {
	case reflect.String:
		reflect.ValueOf(&id).Elem().SetString(uuid.New().String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var serial int64
		err := pgxscan.Get(ctx, repo.TxOrConn(ctx), &serial,
			"SELECT nextval(pg_get_serial_sequence('"+repo.Table+"', '"+strings.ToLower(repo.IDFieldName)+"'))",
		)
		if err != nil {
			return id, fmt.Errorf("could not get id from sequence: %v", err)
		}

		reflect.ValueOf(&id).Elem().SetInt(serial)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var serial int64
		err := pgxscan.Get(ctx, repo.TxOrConn(ctx), &serial,
			"SELECT nextval(pg_get_serial_sequence('"+repo.Table+"', '"+strings.ToLower(repo.IDFieldName)+"'))",
		)
		if err != nil {
			return id, fmt.Errorf("could not get id from sequence: %v", err)
		}

		reflect.ValueOf(&id).Elem().SetInt(serial)
	default:
		return id, fmt.Errorf("%w: unsupported type", errIDFieldWrong)
	}

	return id, nil
}

func (repo *PostgresRepository[E, ID]) Create(ctx context.Context, entity E) error {
	values := make([]any, len(repo.Columns))

	_, err := repo.getID(entity)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSaveFailed, err)
	}

	e := reflect.ValueOf(entity)
	for i, name := range repo.Columns {
		values[i] = e.FieldByName(name).Interface()
	}

	sql, args, err := psql.Insert(repo.Table).Columns(repo.Columns...).Values(values...).ToSql()
	if err != nil {
		return fmt.Errorf("%w: could not build query: %v", ErrInvalidQuery, err)
	}

	// repo.logger.LogAttrs(ctx, alog.LevelDebug, "create entity", slog.String("query", query), slog.Any("args", args))

	_, err = repo.TxOrConn(ctx).Exec(ctx, sql, args...)
	if err != nil && strings.Contains(err.Error(), "SQLSTATE 23505") {
		return ErrAlreadyExists
	}
	if err != nil {
		return fmt.Errorf("%w: could not insert entity: %v", ErrInvalidQuery, err)
	}

	return nil
}

func (repo *PostgresRepository[E, ID]) Read(ctx context.Context, id ID) (E, error) { //nolint:ireturn,lll // valid use of generics
	return repo.FindByID(ctx, id)
}

func (repo *PostgresRepository[E, ID]) Update(ctx context.Context, entity E) error {
	id, err := repo.getID(entity)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSaveFailed, err)
	}

	query := psql.Update(repo.Table).Where(squirrel.Eq{repo.IDFieldName: id})

	e := reflect.ValueOf(entity)
	for _, name := range repo.Columns {
		query = query.Set(name, e.FieldByName(name).Interface())
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("%w: could not build query: %v", ErrInvalidQuery, err)
	}

	res, err := repo.TxOrConn(ctx).Exec(ctx, sql, args...)
	if err == nil && res.RowsAffected() == 0 {
		return fmt.Errorf("entity does not exist yet: %w", ErrSaveFailed)
	}
	if err != nil {
		return fmt.Errorf("%w: could not update entity with id: %v", ErrSaveFailed, id)
	}

	return nil
}

func (repo *PostgresRepository[E, ID]) Delete(ctx context.Context, entity E) error {
	id, err := repo.getID(entity)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSaveFailed, err)
	}

	sql, args, err := psql.Delete(repo.Table).Where(squirrel.Eq{repo.IDFieldName: id}).ToSql()
	if err != nil {
		return fmt.Errorf("%w: could not build query: %v", ErrInvalidQuery, err)
	}

	_, err = repo.TxOrConn(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%w: could not delete entity with id: %v", ErrDeleteFailed, id)
	}

	return nil
}

func (repo *PostgresRepository[E, ID]) All(ctx context.Context) ([]E, error) {
	return repo.FindAll(ctx)
}

func (repo *PostgresRepository[E, ID]) AllByIDs(ctx context.Context, ids []ID) ([]E, error) {
	return repo.FindByIDs(ctx, ids)
}

func (repo *PostgresRepository[E, ID]) FindAll(ctx context.Context) ([]E, error) {
	sql, args, err := psql.Select(repo.Columns...).From(repo.Table).ToSql()
	if err != nil {
		return []E{}, fmt.Errorf("%w: could not build query: %v", ErrInvalidQuery, err)
	}

	entities := []E{}

	err = pgxscan.Select(ctx, repo.PGx, &entities, sql, args...)
	if errors.Is(err, pgx.ErrNoRows) {
		return []E{}, fmt.Errorf("%w: entities do not exist: %v", ErrNotFound, err)
	}
	if err != nil {
		return []E{}, fmt.Errorf("%w: could not scan entities: %v", ErrInvalidQuery, err)
	}

	return entities, nil
}

func (repo *PostgresRepository[E, ID]) FindByID(ctx context.Context, id ID) (E, error) {
	sql, args, err := psql.Select(repo.Columns...).From(repo.Table).Where(squirrel.Eq{repo.IDFieldName: id}).ToSql()
	if err != nil {
		return *new(E), fmt.Errorf("%w: could not build query: %v", ErrInvalidQuery, err)
	}

	// repo.logger.LogAttrs(ctx, alog.LevelDebug, "read entity", slog.String("query", query), slog.Any("args", args))

	entity := *new(E)

	err = pgxscan.Get(ctx, repo.PGx, &entity, sql, args...)
	if errors.Is(err, pgx.ErrNoRows) {
		return *new(E), fmt.Errorf("%w: entity does not exist: %v", ErrNotFound, err)
	}
	if err != nil {
		return *new(E), fmt.Errorf("%w: could not scan entity: %v", ErrInvalidQuery, err)
	}

	return entity, nil
}

func (repo *PostgresRepository[E, ID]) FindByIDs(ctx context.Context, ids []ID) ([]E, error) {
	if ids == nil || reflect.DeepEqual(ids, []ID{}) {
		return []E{}, nil
	}

	sql, args, err := psql.Select(repo.Columns...).From(repo.Table).Where(squirrel.Eq{repo.IDFieldName: ids}).ToSql()
	if err != nil {
		return []E{}, fmt.Errorf("%w: could not build query: %v", ErrInvalidQuery, err)
	}

	entities := []E{}

	err = pgxscan.Select(ctx, repo.PGx, &entities, sql, args...)
	if err != nil {
		return []E{}, fmt.Errorf("%w: could not scan entities: %v", ErrInvalidQuery, err)
	}

	if len(entities) != len(ids) {
		return []E{}, fmt.Errorf("%w: entities do not exist: %v", ErrNotFound, err)
	}

	return entities, nil
}

func (repo *PostgresRepository[E, ID]) Exists(ctx context.Context, id ID) (bool, error) {
	sql, args, err := psql.Select("1").
		Prefix("SELECT EXISTS (").
		From(repo.Table).Where(squirrel.Eq{repo.IDFieldName: id}).
		Suffix(")").ToSql()
	if err != nil {
		return false, fmt.Errorf("%w: could not build query: %v", ErrInvalidQuery, err) // todo add sql statement to every error that does not build for easier debugging
	}

	var exists bool

	err = pgxscan.Get(ctx, repo.PGx, &exists, sql, args...)
	if err != nil {
		return false, fmt.Errorf("%w: could not scan exists result: %v", ErrInvalidQuery, err)
	}

	return exists, nil
}

func (repo *PostgresRepository[E, ID]) ExistsByID(ctx context.Context, id ID) (bool, error) {
	return repo.Exists(ctx, id)
}

func (repo *PostgresRepository[E, ID]) ExistByIDs(_ context.Context, _ []ID) (bool, error) {
	panic("implement me")
}

func (repo *PostgresRepository[E, ID]) ExistAll(_ context.Context, _ []ID) (bool, error) {
	panic("implement me")
}

func (repo *PostgresRepository[E, ID]) Contains(ctx context.Context, id ID) (bool, error) {
	return repo.ExistsByID(ctx, id)
}

func (repo *PostgresRepository[E, ID]) ContainsID(ctx context.Context, id ID) (bool, error) {
	return repo.ContainsIDs(ctx, []ID{id})
}

func (repo *PostgresRepository[E, ID]) ContainsIDs(ctx context.Context, ids []ID) (bool, error) {
	if len(ids) == 0 {
		return false, nil
	}

	sql, args, err := psql.Select("COUNT(" + repo.IDFieldName + ")").
		From(repo.Table).Where(squirrel.Eq{repo.IDFieldName: ids}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("%w: could not build query: %v", ErrInvalidQuery, err) // todo add sql statement to every error that does not build for easier debugging
	}

	var count int

	err = pgxscan.Get(ctx, repo.PGx, &count, sql, args...)
	if err != nil {
		return false, fmt.Errorf("%w: could not scan count result: %v", ErrInvalidQuery, err)
	}

	if count == len(ids) {
		return true, nil
	}

	return false, nil
}

func (repo *PostgresRepository[E, ID]) ContainsAll(ctx context.Context, ids []ID) (bool, error) {
	return repo.ContainsIDs(ctx, ids)
}

func (repo *PostgresRepository[E, ID]) CreateAll(ctx context.Context, entities []E) error {
	return repo.AddAll(ctx, entities)
}

func (repo *PostgresRepository[E, ID]) Save(ctx context.Context, entity E) error {
	_, err := repo.getID(entity)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSaveFailed, err)
	}

	values := make([]any, len(repo.Columns))

	e := reflect.ValueOf(entity)
	for i, name := range repo.Columns {
		values[i] = e.FieldByName(name).Interface()
	}

	query := psql.
		Insert(repo.Table).
		Columns(repo.Columns...).
		Values(values...).
		Suffix("ON CONFLICT (" + repo.IDFieldName + ") DO UPDATE SET")

	for i, name := range repo.Columns {
		expr := name + "= ?,"
		if i == len(repo.Columns)-1 {
			expr = name + "= ?"
		}

		sql, args, err := squirrel.Expr(expr, e.FieldByName(name).Interface()).ToSql()
		if err != nil {
			return fmt.Errorf("%w: could not build on conflict sub-query: %v", ErrInvalidQuery, err)
		}

		query = query.Suffix(sql, args...)
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("%w: could not build query: %v", ErrInvalidQuery, err)
	}

	_, err = repo.TxOrConn(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%w: could not save entity: %v", ErrInvalidQuery, err)
	}

	return nil
}

func (repo *PostgresRepository[E, ID]) SaveAll(ctx context.Context, entities []E) error {
	panic("implement me")
}

func (repo *PostgresRepository[E, ID]) UpdateAll(ctx context.Context, entities []E) error {
	for _, entity := range entities {
		err := repo.Update(ctx, entity) // TODO make one operation instead of loop
		if err != nil {
			return fmt.Errorf("%w: could not update entity: %v", ErrInvalidQuery, err)
		}
	}

	return nil
}

func (repo *PostgresRepository[E, ID]) Add(ctx context.Context, entity E) error {
	return repo.Create(ctx, entity)
}

func (repo *PostgresRepository[E, ID]) AddAll(ctx context.Context, entities []E) error {
	query := psql.Insert(repo.Table).Columns(repo.Columns...)

	for _, entity := range entities {
		if reflect.DeepEqual(entity, *new(E)) {
			return fmt.Errorf("%w", ErrSaveFailed)
		}

		id, err := repo.getID(entity)
		if err != nil {
			return fmt.Errorf("could not get id for entity %v: %w", entity, err)
		}

		exists, err := repo.Exists(ctx, id)
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		if exists {
			return fmt.Errorf("%w: entity %v exists already", ErrAlreadyExists, entity)
		}

		vals := []any{}

		e := reflect.ValueOf(entity)
		for _, name := range repo.Columns {
			vals = append(vals, e.FieldByName(name).Interface())
		}

		query = query.Values(vals...)
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("%w: could not build query: %v", ErrInvalidQuery, err)
	}

	_, err = repo.TxOrConn(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%w: could not execute query: %v", ErrInvalidQuery, err)
	}

	return nil
}

func (repo *PostgresRepository[E, ID]) Count(ctx context.Context) (int, error) {
	sql, args, err := psql.Select("COUNT(*)").From(repo.Table).ToSql()
	if err != nil {
		return 0, fmt.Errorf("%w: could not build query: %v", ErrInvalidQuery, err)
	}

	var count int

	err = pgxscan.Get(ctx, repo.PGx, &count, sql, args...)
	if err != nil {
		return 0, fmt.Errorf("%w: could not scan count result: %v", ErrInvalidQuery, err)
	}

	return count, nil
}

func (repo *PostgresRepository[E, ID]) Length(ctx context.Context) (int, error) {
	return repo.Count(ctx)
}

func (repo *PostgresRepository[E, ID]) DeleteByID(ctx context.Context, id ID) error {
	return repo.DeleteByIDs(ctx, []ID{id})
}

func (repo *PostgresRepository[E, ID]) DeleteByIDs(ctx context.Context, ids []ID) error {
	sql, args, err := psql.Delete(repo.Table).Where(squirrel.Eq{repo.IDFieldName: ids}).ToSql()
	if err != nil {
		return fmt.Errorf("%w: could not build query: %v", ErrInvalidQuery, err)
	}

	_, err = repo.TxOrConn(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%w: could not delete ids from table: %s: %v", ErrDeleteFailed, repo.Table, err) // todo ids
	}

	return nil
}

func (repo *PostgresRepository[E, ID]) DeleteAll(ctx context.Context) error {
	sql, args, err := psql.Delete(repo.Table).ToSql()
	if err != nil {
		return fmt.Errorf("%w: could not build query: %v", ErrInvalidQuery, err)
	}

	_, err = repo.TxOrConn(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%w: could not delete table: %s: %v", ErrDeleteFailed, repo.Table, err)
	}

	return nil
}

func (repo *PostgresRepository[E, ID]) Clear(ctx context.Context) error {
	return repo.DeleteAll(ctx)
}

func (repo *PostgresRepository[E, ID]) AllIter(ctx context.Context) Iterator[E, ID] {
	return PostgresIterator[E, ID]{
		pgx:        repo.PGx,
		ctx:        ctx,
		tx:         nil,
		sql:        "SELECT * FROM " + repo.Table,
		cursorOpen: false,
	}
}

func (repo *PostgresRepository[E, ID]) FindAllIter(ctx context.Context) Iterator[E, ID] {
	return PostgresIterator[E, ID]{
		pgx:        repo.PGx,
		ctx:        ctx,
		tx:         nil,
		sql:        "SELECT * FROM " + repo.Table,
		cursorOpen: false,
	}
}

type PostgresIterator[E any, ID id] struct {
	pgx *pgxpool.Pool
	ctx context.Context
	tx  pgx.Tx

	sql        string
	cursorOpen bool
}

func (i PostgresIterator[E, ID]) Next() func(yield func(e E, err error) bool) {
	return i.nextBatchValue()
}

func (i PostgresIterator[E, ID]) nextBatchValue() func(yield func(e E, err error) bool) {
	cursorName := "cursor_" + strconv.Itoa(rand.Int()) //nolint:gosec // no need for security
	// todo check, if the cursor needs to be named different, if the transaction is already different

	if !i.cursorOpen {
		tx, _ := i.pgx.Begin(i.ctx)
		i.tx = tx

		_, _ = i.tx.Exec(i.ctx, "DECLARE "+cursorName+" CURSOR FOR "+i.sql+";")
		// todo err check

		i.cursorOpen = true
	}

	quit := make(chan struct{}, 2) // cap 2, so regular and error branches can terminate the go routine without blocking

	ch := make(chan E, 1000)
	var fetchErr atomic.Value

	go func() {
		var entities = make([]E, 1000)

		for {
			select { // prevent go routine leak
			case <-quit:
				close(ch)
				return
			default:
			}

			err := pgxscan.Select(i.ctx, i.tx, &entities, "FETCH FORWARD 1000 FROM "+cursorName+";")
			if err != nil {
				fetchErr.Store(err)
				break
			}

			if len(entities) == 0 {
				break
			}

			for i := range entities {
				ch <- entities[i]
			}
		}

		close(ch)
	}()

	return func(yield func(e E, err error) bool) {
		var err error
		for e := range ch {
			if fe := fetchErr.Load(); fe != nil {
				err = fe.(error)
			}

			if !yield(e, err) {
				quit <- struct{}{}

				return
			}
		}

		_, _ = i.tx.Exec(i.ctx, "CLOSE "+cursorName+";")

		_ = i.tx.Commit(i.ctx)

		quit <- struct{}{}
	}
}

func (i PostgresIterator[E, ID]) nextSingleValue() func(yield func(e E, err error) bool) {
	cursorName := "cursor_" + strconv.Itoa(rand.Int()) //nolint:gosec // no need for security
	// todo check, if the cursor needs to be named different, if the transaction is already different

	if !i.cursorOpen {
		tx, _ := i.pgx.Begin(i.ctx)
		i.tx = tx

		_, _ = i.tx.Exec(i.ctx, "DECLARE "+cursorName+" CURSOR FOR "+i.sql+";")
		// todo err check

		i.cursorOpen = true
	}

	return func(yield func(e E, err error) bool) {
		var entity E

		for {
			err := pgxscan.Get(i.ctx, i.tx, &entity, "FETCH FORWARD 1 FROM "+cursorName+";")
			if err != nil {
				return
			}

			if !yield(entity, err) {
				_, err = i.tx.Exec(i.ctx, "CLOSE "+cursorName+";")

				_ = i.tx.Commit(i.ctx)

				return
			}
		}
	}
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
