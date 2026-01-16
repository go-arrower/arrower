package arepo

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/dbscan"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-arrower/arrower/arepo/q"
	"github.com/go-arrower/arrower/postgres"
)

// NewPostgresRepository
//
// If using the WithIDField option, it is required that the column in postgres has a unique or exclusion constraint,
// otherwise Save will fail, because they use the ON CONFLICT statement.
//
// To map the database rows to Go structs  dbscan is used. It translates the struct field name to snake case.
// To override this behaviour, specify the column name in the `db` field tag.
//
//	type User struct {
//		ID        string `db:"user_id"`
//		FirstName string
//		Email     string
//	}
//
// In the example above User struct is mapped to the following columns: "user_id", "first_name", "email".
func NewPostgresRepository[E any, ID id](pgx *pgxpool.Pool, opts ...Option) (*PostgresRepository[E, ID], error) {
	repo := &PostgresRepository[E, ID]{
		PGx:         pgx,
		Table:       tableName(*new(E)),
		Columns:     columnNames(*new(E)),
		IDFieldName: "ID",
	}

	for _, opt := range opts {
		err := opt(repo)
		if err != nil {
			name := reflect.TypeOf(*new(E)).Name()
			return nil, fmt.Errorf("could not initialise %s postgres repository: option returned error: %w", name, err)
		}
	}

	idField := reflect.ValueOf(*new(E)).FieldByName(repo.IDFieldName)
	if reflect.DeepEqual(idField, reflect.Value{}) { //nolint:govet,lll // is a fp and will be fixed, see: https://github.com/golang/go/issues/43993
		name := reflect.TypeOf(*new(E)).Name()
		return nil, fmt.Errorf("could not initialise %s postgres repository: entity does not have the ID field with name: %v", name, repo.IDFieldName)
	}

	return repo, nil
}

// PostgresRepository implements Repository in a generic way. Use it to speed up your development.
//
// The repository exposes fields required to extend the repository with custom methods.
type PostgresRepository[E any, ID id] struct {
	PGx *pgxpool.Pool

	IDFieldName string
	Table       string
	Columns     []string
}

func (repo *PostgresRepository[E, ID]) TxOrConn(ctx context.Context) interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
} {
	tx, ok := ctx.Value(postgres.CtxTX).(pgx.Tx)
	if ok {
		return tx
	}

	return repo.PGx
}

func (repo *PostgresRepository[E, ID]) Tx(ctx context.Context) pgx.Tx {
	tx, ok := ctx.Value(postgres.CtxTX).(pgx.Tx)
	if ok {
		return tx
	}

	tx, _ = repo.PGx.Begin(ctx)

	return tx
}

var psql = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar) //nolint:gochecknoglobals,lll // squirrel recommends this

func (repo *PostgresRepository[E, ID]) getID(t any) (ID, error) { //nolint:ireturn,lll // needs access to the type ID and fp, as it is not recognised even with "generic" setting
	var id ID

	val := reflect.ValueOf(t)
	idField := val.FieldByName(repo.IDFieldName)

	switch idField.Kind() {
	case reflect.String:
		reflect.ValueOf(&id).Elem().SetString(idField.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		reflect.ValueOf(&id).Elem().SetInt(idField.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		reflect.ValueOf(&id).Elem().SetUint(idField.Uint())
	default:
		// can never happen: compiler enforces valid types from the `id` interface (constraint)
	}

	if id == *new(ID) {
		return *new(ID), fmt.Errorf("%w: missing %s", errIDFailed, repo.IDFieldName)
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
			return id, fmt.Errorf("%w: could not get from sequence: %v", errIDGenerationFailed, err)
		}

		reflect.ValueOf(&id).Elem().SetInt(serial)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var serial int64

		err := pgxscan.Get(ctx, repo.TxOrConn(ctx), &serial,
			"SELECT nextval(pg_get_serial_sequence('"+repo.Table+"', '"+strings.ToLower(repo.IDFieldName)+"'))",
		)
		if err != nil {
			return id, fmt.Errorf("%w: could not get from sequence: %v", errIDGenerationFailed, err)
		}

		reflect.ValueOf(&id).Elem().SetInt(serial)
	default:
		return id, fmt.Errorf("%w: type %s of ID is not supported; only string and ints are",
			errIDGenerationFailed, reflect.TypeOf(id).Kind().String())
	}

	return id, nil
}

func (repo *PostgresRepository[E, ID]) Create(ctx context.Context, entity E) error {
	id, err := repo.getID(entity)
	if err != nil {
		return fmt.Errorf("%w: %w", errCreateFailed, err)
	}

	sql, args, err := psql.Insert(repo.Table).Columns(repo.Columns...).Values(columnValues(entity)...).ToSql()
	if err != nil {
		return fmt.Errorf("%w: could not build query: %v", errCreateFailed, err)
	}

	_, err = repo.TxOrConn(ctx).Exec(ctx, sql, args...)
	if err != nil && strings.Contains(err.Error(), "SQLSTATE 23505") {
		return ErrAlreadyExists
	}
	if err != nil { //nolint:wsl_v5 // error handling belongs together
		return fmt.Errorf("%w: could not insert entity with id %v: %v", errCreateFailed, id, err)
	}

	return nil
}

func (repo *PostgresRepository[E, ID]) Read(ctx context.Context, id ID) (E, error) { //nolint:ireturn,lll // valid use of generics
	return repo.FindByID(ctx, id)
}

func (repo *PostgresRepository[E, ID]) Update(ctx context.Context, entity E) error {
	id, err := repo.getID(entity)
	if err != nil {
		return fmt.Errorf("%w: %w", errUpdateFailed, err)
	}

	query := psql.Update(repo.Table).Where(squirrel.Eq{repo.IDFieldName: id})

	vals := columnValues(entity)
	for i, name := range repo.Columns {
		query = query.Set(name, vals[i])
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("%w: could not build query: %v", errUpdateFailed, err)
	}

	res, err := repo.TxOrConn(ctx).Exec(ctx, sql, args...)
	if err == nil && res.RowsAffected() == 0 {
		return fmt.Errorf("entity %w", ErrNotFound)
	}
	if err != nil { //nolint:wsl_v5 // error handling belongs together
		return fmt.Errorf("%w: could not update entity with id: %v: %v", errUpdateFailed, id, err)
	}

	return nil
}

func (repo *PostgresRepository[E, ID]) Delete(ctx context.Context, entity E) error {
	id, err := repo.getID(entity)
	if err != nil {
		return nil //nolint:nilerr // entity without ID does not exist in the repo; meaning it is as if it is deleted.
	}

	sql, args, err := psql.Delete(repo.Table).Where(squirrel.Eq{repo.IDFieldName: id}).ToSql()
	if err != nil {
		return fmt.Errorf("%w: could not build query: %v", errDeleteFailed, err)
	}

	_, err = repo.TxOrConn(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%w: could not delete entity with id: %v: %v", errDeleteFailed, id, err)
	}

	return nil
}

func (repo *PostgresRepository[E, ID]) All(ctx context.Context) ([]E, error) {
	return repo.FindAll(ctx)
}

func (repo *PostgresRepository[E, ID]) AllByIDs(ctx context.Context, ids []ID) ([]E, error) {
	return repo.FindByIDs(ctx, ids)
}

func (repo *PostgresRepository[E, ID]) AllBy(ctx context.Context, query q.Query) ([]E, error) {
	sql, args, err := repo.buildFilteredSQL(query)
	if err != nil {
		return []E{}, fmt.Errorf("%w: could not build query: %v", errFindFailed, err)
	}

	entities := []E{}

	err = pgxscan.Select(ctx, repo.TxOrConn(ctx), &entities, sql, args...)
	if errors.Is(err, pgx.ErrNoRows) {
		return []E{}, fmt.Errorf("entities %w: %v", ErrNotFound, err)
	}
	if err != nil { //nolint:wsl_v5 // error handling belongs together
		return []E{}, fmt.Errorf("%w: could not scan entities: %v", errFindFailed, err)
	}

	return entities, nil
}

func (repo *PostgresRepository[E, ID]) FindAll(ctx context.Context) ([]E, error) {
	query := psql.Select(repo.Columns...).From(repo.Table)

	if slices.Contains(repo.Columns, "created_at") {
		query = query.OrderBy("created_at ASC")
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return []E{}, fmt.Errorf("%w: could not build query: %v", errFindFailed, err)
	}

	entities := []E{}

	err = pgxscan.Select(ctx, repo.PGx, &entities, sql, args...)
	if errors.Is(err, pgx.ErrNoRows) {
		return []E{}, fmt.Errorf("entities %w: %v", ErrNotFound, err)
	}
	if err != nil { //nolint:wsl_v5 // error handling belongs together
		return []E{}, fmt.Errorf("%w: could not scan entities: %v", errFindFailed, err)
	}

	return entities, nil
}

func (repo *PostgresRepository[E, ID]) FindBy(ctx context.Context, query q.Query) (E, error) {
	sql, args, err := repo.buildFilteredSQL(query)
	if err != nil {
		return *new(E), fmt.Errorf("%w: could not build query: %v", errFindFailed, err)
	}

	entities := []E{}

	err = pgxscan.Select(ctx, repo.TxOrConn(ctx), &entities, sql, args...)
	if errors.Is(err, pgx.ErrNoRows) {
		return *new(E), fmt.Errorf("entities %w: %v", ErrNotFound, err)
	}
	if err != nil { //nolint:wsl_v5 // error handling belongs together
		return *new(E), fmt.Errorf("%w: could not scan entities: %v", errFindFailed, err)
	}

	// todo verify this via test
	if len(entities) != 1 {
		return *new(E), fmt.Errorf("%w: FindBy only returns one entity, but filter found: %d", ErrNotFound, len(entities))
	}

	return entities[0], nil
}

func (repo *PostgresRepository[E, ID]) FindByID(ctx context.Context, id ID) (E, error) { //nolint:ireturn,lll // valid use of generics
	sql, args, err := psql.Select(repo.Columns...).From(repo.Table).Where(squirrel.Eq{repo.IDFieldName: id}).ToSql()
	if err != nil {
		return *new(E), fmt.Errorf("%w: could not build query: %v", errFindFailed, err)
	}

	entity := new(E)

	err = pgxscan.Get(ctx, repo.TxOrConn(ctx), entity, sql, args...)
	if errors.Is(err, pgx.ErrNoRows) {
		return *new(E), fmt.Errorf("entity %w: %v", ErrNotFound, err)
	}
	if err != nil { //nolint:wsl_v5 // error handling belongs together
		return *new(E), fmt.Errorf("%w: could not scan entity: %v", errFindFailed, err)
	}

	return *entity, nil
}

func (repo *PostgresRepository[E, ID]) FindByIDs(ctx context.Context, ids []ID) ([]E, error) {
	if ids == nil || reflect.DeepEqual(ids, []ID{}) {
		return []E{}, nil
	}

	sql, args, err := psql.Select(repo.Columns...).From(repo.Table).Where(squirrel.Eq{repo.IDFieldName: ids}).ToSql()
	if err != nil {
		return []E{}, fmt.Errorf("%w: could not build query: %v", errFindFailed, err)
	}

	entities := []E{}

	err = pgxscan.Select(ctx, repo.TxOrConn(ctx), &entities, sql, args...)
	if err != nil {
		return []E{}, fmt.Errorf("%w: could not scan entities: %v", errFindFailed, err)
	}

	if len(entities) != len(ids) {
		return []E{}, fmt.Errorf("some ids: %w", ErrNotFound)
	}

	return entities, nil
}

func (repo *PostgresRepository[E, ID]) Exists(ctx context.Context, id ID) (bool, error) {
	sql, args, err := psql.Select("1").
		Prefix("SELECT EXISTS (").
		From(repo.Table).Where(squirrel.Eq{repo.IDFieldName: id}).
		Suffix(")").ToSql()
	if err != nil {
		return false, fmt.Errorf("%w: could not build query: %v", errExistsFailed, err)
	}

	var exists bool

	err = pgxscan.Get(ctx, repo.TxOrConn(ctx), &exists, sql, args...)
	if err != nil {
		return false, fmt.Errorf("%w: could not scan result: %v", errExistsFailed, err)
	}

	return exists, nil
}

func (repo *PostgresRepository[E, ID]) ExistsByID(ctx context.Context, id ID) (bool, error) {
	return repo.Exists(ctx, id)
}

func (repo *PostgresRepository[E, ID]) ExistByIDs(ctx context.Context, ids []ID) (bool, error) {
	return repo.ExistAll(ctx, ids)
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
		return false, fmt.Errorf("%w: could not build query: %v", errExistsFailed, err)
	}

	var count int

	err = pgxscan.Get(ctx, repo.TxOrConn(ctx), &count, sql, args...)
	if err != nil {
		return false, fmt.Errorf("%w: could not scan result: %v", errExistsFailed, err)
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
		return fmt.Errorf("%w: %w", errSaveFailed, err)
	}

	values := make([]any, len(repo.Columns))

	ent := reflect.ValueOf(entity)
	for i, name := range repo.Columns {
		values[i] = ent.FieldByName(entitiesFieldNameByColumnName[E](name)).Interface()
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

		sql, args, sqErr := squirrel.Expr(expr, ent.FieldByName(entitiesFieldNameByColumnName[E](name)).Interface()).ToSql()
		if sqErr != nil {
			return fmt.Errorf("%w: could not build on conflict sub-query: %v", errSaveFailed, sqErr)
		}

		query = query.Suffix(sql, args...)
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("%w: could not build query: %v", errSaveFailed, err)
	}

	_, err = repo.TxOrConn(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%w: could not insert entity: %v", errSaveFailed, err)
	}

	return nil
}

func (repo *PostgresRepository[E, ID]) SaveAll(_ context.Context, entities []E) error {
	panic("implement me")
}

func (repo *PostgresRepository[E, ID]) UpdateAll(ctx context.Context, entities []E) error {
	var tx pgx.Tx

	conn := repo.TxOrConn(ctx)
	if conTx, ok := conn.(pgx.Tx); ok {
		tx = conTx
	} else {
		t, err := repo.PGx.Begin(ctx)
		if err != nil {
			return fmt.Errorf("%w: could not start transaction: %v", errSaveFailed, err)
		}

		tx = t
	}

	for _, entity := range entities {
		err := func() error {
			id, err := repo.getID(entity)
			if err != nil {
				return fmt.Errorf("%w: %w", errUpdateFailed, err)
			}

			query := psql.Update(repo.Table).Where(squirrel.Eq{repo.IDFieldName: id})

			e := reflect.ValueOf(entity)
			for _, name := range repo.Columns {
				query = query.Set(name, e.FieldByName(entitiesFieldNameByColumnName[E](name)).Interface())
			}

			sql, args, err := query.ToSql()
			if err != nil {
				return fmt.Errorf("%w: could not build query: %v", errUpdateFailed, err)
			}

			res, err := tx.Exec(ctx, sql, args...)
			if err == nil && res.RowsAffected() == 0 {
				return fmt.Errorf("entity %w", ErrNotFound)
			}
			if err != nil { //nolint:wsl_v5 // error handling belongs together
				return fmt.Errorf("%w: could not update entity with id: %v: %v", errUpdateFailed, id, err)
			}

			return nil
		}()
		if err != nil {
			rb := tx.Rollback(ctx)
			if rb != nil {
				return fmt.Errorf("%w: %v: could not rollback transaction: %v", errSaveFailed, err, rb)
			}

			return err
		}
	}

	rb := tx.Commit(ctx)
	if rb != nil {
		return fmt.Errorf("%w: could not commit transaction: %v", errSaveFailed, rb)
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
			return fmt.Errorf("%w", errCreateFailed)
		}

		id, err := repo.getID(entity)
		if err != nil {
			return fmt.Errorf("%w: could not get id for entity %v: %w", errCreateFailed, entity, err)
		}

		exists, err := repo.Exists(ctx, id)
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		if exists {
			return fmt.Errorf("at least one %w", ErrAlreadyExists)
		}

		vals := []any{}

		e := reflect.ValueOf(entity)
		for _, name := range repo.Columns {
			vals = append(vals, e.FieldByName(entitiesFieldNameByColumnName[E](name)).Interface())
		}

		query = query.Values(vals...)
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("%w: could not build query: %v", errCreateFailed, err)
	}

	_, err = repo.TxOrConn(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%w: could not execute query: %v", errCreateFailed, err)
	}

	return nil
}

func (repo *PostgresRepository[E, ID]) Count(ctx context.Context) (int, error) {
	sql, args, err := psql.Select("COUNT(*)").From(repo.Table).ToSql()
	if err != nil {
		return 0, fmt.Errorf("%w: could not build query: %v", errCountFailed, err)
	}

	var count int

	err = pgxscan.Get(ctx, repo.TxOrConn(ctx), &count, sql, args...)
	if err != nil {
		return 0, fmt.Errorf("%w: could not scan result: %v", errCountFailed, err)
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
		return fmt.Errorf("%w: could not build query: %v", errDeleteFailed, err)
	}

	_, err = repo.TxOrConn(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%w: could not execute query: %v", errDeleteFailed, err)
	}

	return nil
}

func (repo *PostgresRepository[E, ID]) DeleteAll(ctx context.Context) error {
	sql, args, err := psql.Delete(repo.Table).ToSql()
	if err != nil {
		return fmt.Errorf("%w: could not build query: %v", errDeleteFailed, err)
	}

	_, err = repo.TxOrConn(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%w: could not execute query: %v", errDeleteFailed, err)
	}

	return nil
}

func (repo *PostgresRepository[E, ID]) Clear(ctx context.Context) error {
	return repo.DeleteAll(ctx)
}

func (repo *PostgresRepository[E, ID]) AllIter(ctx context.Context) Iterator[E, ID] {
	return PostgresIterator[E, ID]{
		repo:       repo,
		ctx:        ctx,
		tx:         nil,
		sql:        "SELECT * FROM " + repo.Table,
		cursorOpen: false,
	}
}

func (repo *PostgresRepository[E, ID]) FindAllIter(ctx context.Context) Iterator[E, ID] {
	return PostgresIterator[E, ID]{
		repo:       repo,
		ctx:        ctx, //nolint:containedctx,lll // context scoped to single iterator/request; enable Go 1.23+ iterator pattern without breaking loop semantics
		tx:         nil,
		sql:        "SELECT * FROM " + repo.Table,
		cursorOpen: false,
	}
}

//nolint:wrapcheck // caller wraps properly
func (repo *PostgresRepository[E, ID]) buildFilteredSQL(dataQuery q.Query) (string, []any, error) {
	query := psql.Select(repo.Columns...).From(repo.Table)

	conditions := dataQuery.Conditions.Conditions
	if len(conditions) == 0 {
		return query.ToSql()
	}

	for _, condition := range conditions {
		if condition.Value == nil {
			return "", nil, errors.New("value can not be nil")
		}

		query = query.Where(squirrel.Eq{
			pgFieldName(reflect.TypeOf(*new(E)), condition.Field): condition.Value,
		})
	}

	groups := dataQuery.Conditions.Groups
	for _, g := range groups {
		inner := squirrel.Or{}

		for _, condition := range g.Conditions {
			if condition.Value == nil {
				return "", nil, errors.New("value can not be nil")
			}

			inner = append(inner, squirrel.Eq{
				pgFieldName(reflect.TypeOf(*new(E)), condition.Field): condition.Value,
			})
		}

		query = query.Where(inner)
	}

	return query.ToSql()
}

const batchSize = 1000

type PostgresIterator[E any, ID id] struct {
	repo *PostgresRepository[E, ID]
	ctx  context.Context
	tx   pgx.Tx

	sql        string
	cursorOpen bool
}

func (i PostgresIterator[E, ID]) Next() func(yield func(e E, err error) bool) {
	// return i.nextSingleValue()
	return i.nextBatchValue()
}

func (i PostgresIterator[E, ID]) nextBatchValue() func(yield func(e E, err error) bool) {
	cursorName := "cursor_" + strconv.Itoa(rand.Int()) //nolint:gosec // get a unique name, no crypto required

	if !i.cursorOpen {
		tx := i.repo.Tx(i.ctx)
		i.tx = tx

		_, err := i.tx.Exec(i.ctx, "DECLARE "+cursorName+" CURSOR FOR "+i.sql+";")
		if err != nil {
			return func(yield func(e E, err error) bool) {
				yield(*new(E), fmt.Errorf("%w: iterator could not declare cursor: %v", errFindFailed, err))
			}
		}

		i.cursorOpen = true
	}

	// cap 2, so regular and error branches can terminate the go routine without blocking
	quit := make(chan struct{}, 2) //nolint:mnd

	ch := make(chan E, batchSize)

	var fetchErr atomic.Value

	go func() {
		entities := make([]E, batchSize)

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
		defer func() {
			quit <- struct{}{}
		}()

		var err error

		for e := range ch {
			select {
			case <-i.ctx.Done():
				yield(*new(E), i.ctx.Err())
				return
			default:
				if fe := fetchErr.Load(); fe != nil {
					err = fe.(error) //nolint:forcetypeassert // fetchErr is local and only stores errors
				}

				if !yield(e, err) {
					quit <- struct{}{}

					return
				}
			}
		}

		_, err = i.tx.Exec(i.ctx, "CLOSE "+cursorName+";")
		if err != nil {
			yield(*new(E), fmt.Errorf("%w: iterator could close cursor: %v", errFindFailed, err))
		}

		err = i.tx.Commit(i.ctx)
		if err != nil {
			yield(*new(E), fmt.Errorf("%w: iterator could close cursor: %v", errFindFailed, err))
		}
	}
}

// nextSingleValue returns always one value at a time.
//
//nolint:unused // keep it around for benchmarking
func (i PostgresIterator[E, ID]) nextSingleValue() func(yield func(e E, err error) bool) {
	cursorName := "cursor_" + strconv.Itoa(rand.Int()) //nolint:gosec // get a unique name, no crypto required

	if !i.cursorOpen {
		tx := i.repo.Tx(i.ctx)
		i.tx = tx

		_, err := i.tx.Exec(i.ctx, "DECLARE "+cursorName+" CURSOR FOR "+i.sql+";")
		if err != nil {
			return func(yield func(e E, err error) bool) {
				yield(*new(E), fmt.Errorf("%w: iterator could not declare cursor: %v", errFindFailed, err))
			}
		}

		i.cursorOpen = true
	}

	return func(yield func(e E, err error) bool) {
		var entity E

		for {
			select {
			case <-i.ctx.Done():
				yield(*new(E), i.ctx.Err())
				return
			default:
				err := pgxscan.Get(i.ctx, i.tx, &entity, "FETCH FORWARD 1 FROM "+cursorName+";")
				if err != nil {
					return
				}

				if !yield(entity, err) {
					_, err = i.tx.Exec(i.ctx, "CLOSE "+cursorName+";")
					if err != nil {
						yield(*new(E), fmt.Errorf("%w: iterator could close cursor: %v", errFindFailed, err))
						return
					}

					err = i.tx.Commit(i.ctx)
					if err != nil {
						yield(*new(E), fmt.Errorf("%w: iterator could close cursor: %v", errFindFailed, err))
						return
					}

					return
				}
			}
		}
	}
}

func tableName[E any](entity E) string {
	return strings.ToLower(reflect.TypeOf(entity).Name())
}

func entitiesFieldNameByColumnName[E any](name string) string {
	name = strings.TrimPrefix(name, `"`)
	name = strings.TrimSuffix(name, `"`)

	t := reflect.TypeOf(*new(E))

	for i := range t.NumField() {
		field := t.Field(i)
		dbTag := field.Tag.Get("db")

		if dbTag == name || dbscan.SnakeCaseMapper(field.Name) == name {
			return field.Name
		}

		if strings.Contains(name, ".") {
			embeddedName := strings.Split(name, ".")[0]
			if dbTag == embeddedName || field.Name == embeddedName {
				return field.Name
			}
		}
	}

	return ""
}

func pgFieldName(entity reflect.Type, name string) string {
	isQuoted := strings.HasPrefix(name, `"`) && strings.HasSuffix(name, `"`)
	if isQuoted {
		return strings.TrimPrefix(strings.TrimSuffix(name, `"`), `"`)
	}

	name = strings.ToLower(name)

	return name
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// columnNames gibt die Spaltennamen zurück (raw, ohne Quotes)
func columnNames[E any](entity E) []string {
	cols, _ := columnsAndValues(reflect.ValueOf(entity), "")
	names := make([]string, len(cols))
	for i, c := range cols {
		names[i] = quoteIdent(c.name)
	}
	return names
}

// columnValues gibt die Werte in derselben Reihenfolge zurück
func columnValues[E any](entity E) []any {
	cols, _ := columnsAndValues(reflect.ValueOf(entity), "")
	vals := make([]any, len(cols))
	for i, c := range cols {
		vals[i] = c.value
	}
	return vals
}

type colVal struct {
	name  string
	value any
}

func columnsAndValues(v reflect.Value, prefix string) ([]colVal, error) {
	var result []colVal

	// Deref pointer
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, nil
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, nil
	}

	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fv := v.Field(i)

		// Skip unexported
		if f.PkgPath != "" {
			continue
		}

		col := f.Tag.Get("db")
		if col == "" {
			col = dbscan.SnakeCaseMapper(f.Name)
		}

		// Deref pointer field
		ft := f.Type
		for ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
			if fv.Kind() == reflect.Pointer {
				if fv.IsNil() {
					// nil pointer -> skip oder NULL
					result = append(result, colVal{name: col, value: nil})
					continue
				}
				fv = fv.Elem()
			}
		}

		// Embedded struct ODER normales Struct-Feld -> flatten
		if ft.Kind() == reflect.Struct {
			// Bestimme Prefix
			var newPrefix string
			if f.Anonymous {
				// Embedded ohne db-Tag -> kein Prefix
				if f.Tag.Get("db") == "" {
					newPrefix = prefix
				} else {
					// Embedded mit db-Tag -> custom.xxx
					newPrefix = joinPrefix(prefix, col)
				}
			} else {
				// Normales Feld -> custom.xxx
				newPrefix = joinPrefix(prefix, col)
			}

			sub, err := columnsAndValues(fv, newPrefix)
			if err != nil {
				return nil, err
			}
			result = append(result, sub...)
			continue
		}

		// Normales Feld
		fullCol := joinPrefix(prefix, col)
		result = append(result, colVal{name: fullCol, value: fv.Interface()})
	}

	return result, nil
}

func joinPrefix(prefix, col string) string {
	if prefix == "" {
		return col
	}
	return prefix + "." + col
}
