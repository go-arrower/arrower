//go:build integration

package q_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/arepo"
	"github.com/go-arrower/arrower/arepo/q"
	"github.com/go-arrower/arrower/arepo/testdata"
	"github.com/go-arrower/arrower/contexts/auth"
	"github.com/go-arrower/arrower/tests"
)

func TestExplore(t *testing.T) {
	ctx := context.Background()
	repo := arepo.NewMemoryRepository[testdata.Entity, testdata.EntityID]()

	repo.AllBy(ctx, q.Filter(auth.User{}))
	repo.AllBy(ctx, q.Filter(MyUserFilter{}))
	repo.AllBy(ctx, q.Where("Name").Is("test"))
	repo.AllBy(ctx, ActiveUsers())
	repo.AllBy(ctx, User{}.Active())
	repo.AllBy(ctx, Users().
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

	repo, err := arepo.NewPostgresRepository[Entity, string](pg.PGx())
	assert.NoError(t, err)

	found, err := repo.AllBy(ctx, q.Filter(Entity{Name: "test1", Age: 1337}))
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

func ActiveUsers() q.Query {
	return q.Query{Conditions: q.ConditionGroup{Conditions: []q.Condition{
		{
			Field:    "Active",
			Operator: "=",
			Value:    true,
		},
	}}}
}

type MyUserFilter struct{}
type User struct{}

func (u User) Active() q.Query {
	return q.Query{Conditions: q.ConditionGroup{Conditions: []q.Condition{
		{
			Field:    "Active",
			Operator: "=",
			Value:    true,
		},
	}}}
}

type UserQuery struct {
	*q.Query
}

func Users() *UserQuery {
	return &UserQuery{&q.Query{}}
}

// Now you can add model-specific helpers
func (q *UserQuery) Active() *UserQuery {
	*q.Query = q.Where("status").Is("active")
	return q
}

func (q *UserQuery) Adults() *UserQuery {
	*q.Query = q.Where("age").Is("18") // use GTE
	return q
}

func (q *UserQuery) WithVerifiedEmail() *UserQuery {
	*q.Query = q.Where("email_verified").Is(true)
	return q
}

func (q *UserQuery) Find() q.Query {
	return *q.Query
}
