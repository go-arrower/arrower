//go:build integration

package q_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/contexts/auth"
	"github.com/go-arrower/arrower/repository"
	"github.com/go-arrower/arrower/repository/q"
	"github.com/go-arrower/arrower/repository/testdata"
	"github.com/go-arrower/arrower/tests"
)

func TestExplore(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemoryRepository[testdata.Entity, testdata.EntityID]()

	repo.AllBy(ctx, q.F(auth.User{}))
	repo.AllBy(ctx, q.F(q.MyUserFilter{}))
	repo.AllBy(ctx, q.Where("Name").Is("test"))
	repo.AllBy(ctx, q.ActiveUsers())
	repo.AllBy(ctx, q.User{}.Active())
	repo.AllBy(ctx, q.Users().
		Active().
		Adults().
		WithVerifiedEmail().
		Find())
}

func TestPG(t *testing.T) {
	t.Parallel()

	type Entity struct {
		ID   string
		Name string
		Age  int
	}

	ctx := context.Background()
	pg := tests.NewPostgresDockerForIntegrationTesting()
	_, err := pg.PGx().Exec(ctx, `CREATE TABLE IF NOT EXISTS entity(id TEXT PRIMARY KEY, name TEXT, age INTEGER);`)
	assert.NoError(t, err)

	pg.PGx().Exec(ctx, `INSERT INTO entity (id, name, age) VALUES (uuid_generate_v4(),'test0', 1337);`)
	pg.PGx().Exec(ctx, `INSERT INTO entity (id, name, age) VALUES (uuid_generate_v4(),'test0', 1338);`)
	_, err = pg.PGx().Exec(ctx, `INSERT INTO entity (id, name, age) VALUES (uuid_generate_v4(),'test1', 1337);`)
	assert.NoError(t, err)

	repo, err := repository.NewPostgresRepository[Entity, string](pg.PGx())
	assert.NoError(t, err)

	found, err := repo.AllBy(ctx, q.F(Entity{Name: "test1", Age: 1337}))
	assert.NoError(t, err)
	assert.Len(t, found, 1)
	t.Log(found)

	found, err = repo.AllBy(ctx, q.Where("Name").Is("test0").
		Or(
			q.Where("Age").Is(1337).
				Where("Age").Is(1338),
		))
	assert.NoError(t, err)
	assert.Len(t, found, 2)
	t.Log(found)

	found, err = repo.AllBy(ctx, q.Where("Name").Is("test0").
		Or(
			q.Where("Age").Is(1337).OrderBy("Name").Ascending(), // INVALID => FIXME
			q.Where("Age").Is(1338),
		))
	assert.NoError(t, err)
	assert.Len(t, found, 2)
	t.Log(found)

	found, err = repo.AllBy(ctx, q.Where("Name").Is("test0").
		Where("age").Is(1337))
	assert.NoError(t, err)
	assert.Len(t, found, 1)
	t.Log(found)

	found, err = repo.AllBy(ctx, q.Where("Name").Is("test0"))
	assert.NoError(t, err)
	assert.Len(t, found, 2)
	t.Log(found)
}

/*
Test Cases
* Where
	* conditionGroup empty
* SortBy
* Limit
* complex combinations of logical queries, e.g.
	* nested OR
	* mixed nested AND and OR
	* logical query calls limit or sortBy
*/
