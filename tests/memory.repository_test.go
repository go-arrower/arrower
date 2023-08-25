package tests_test

import (
	"context"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/tests"
)

func TestNewMemoryRepository(t *testing.T) {
	t.Parallel()

	repo := tests.NewMemoryRepository[Entity, EntityID]()
	assert.NotNil(t, repo)
}

func TestMemoryRepository_NextID(t *testing.T) {
	t.Parallel()

	repo := tests.NewMemoryRepository[Entity, EntityID]()
	id := repo.NextID(ctx)
	assert.NotEmpty(t, id)
}

func TestMemoryRepository_Create(t *testing.T) {
	t.Parallel()

	t.Run("create", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[Entity, EntityID]()
		err := repo.Create(ctx, defaultEntity)
		assert.NoError(t, err)

		got, err := repo.Read(ctx, defaultEntity.ID)
		assert.NoError(t, err)
		assert.Equal(t, defaultEntity, got)
	})

	t.Run("create same again", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[Entity, EntityID]()

		err := repo.Create(ctx, defaultEntity)
		assert.NoError(t, err)

		err = repo.Create(ctx, defaultEntity)
		assert.ErrorIs(t, err, tests.ErrAlreadyExists, "already exists")
	})

	t.Run("missing id", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[Entity, EntityID]()

		err := repo.Create(ctx, Entity{})
		assert.ErrorIs(t, err, tests.ErrSaveFailed)
	})
}

func TestMemoryRepository_Update(t *testing.T) {
	t.Parallel()

	t.Run("update", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[Entity, EntityID]()
		entity := newEntity()
		repo.Create(ctx, entity)

		entity.Name = "updated-name"
		err := repo.Update(ctx, entity)
		assert.NoError(t, err)

		e, _ := repo.FindByID(ctx, entity.ID)
		assert.Equal(t, entity, e)
	})

	t.Run("does not exist yet", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[Entity, EntityID]()

		err := repo.Update(ctx, defaultEntity)
		assert.ErrorIs(t, err, tests.ErrSaveFailed)
	})

	t.Run("missing id", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[Entity, EntityID]()
		repo.Create(ctx, defaultEntity)

		err := repo.Update(ctx, Entity{})
		assert.ErrorIs(t, err, tests.ErrSaveFailed)
	})
}

func TestMemoryRepository_Delete(t *testing.T) {
	t.Parallel()

	repo := tests.NewMemoryRepository[Entity, EntityID]()
	repo.Add(ctx, defaultEntity)

	err := repo.Delete(ctx, defaultEntity)
	assert.NoError(t, err)

	_, err = repo.FindByID(ctx, defaultEntity.ID)
	assert.ErrorIs(t, err, tests.ErrNotFound)
}

func TestMemoryRepository_All(t *testing.T) {
	t.Parallel()

	repo := tests.NewMemoryRepository[Entity, EntityID]()

	all, err := repo.All(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, all)
	assert.Len(t, all, 0, "new repository should be empty")

	repo.Create(ctx, newEntity())
	repo.Create(ctx, newEntity())

	all, err = repo.All(ctx)
	assert.NoError(t, err)
	assert.Len(t, all, 2, "should have two entities")
}

func TestMemoryRepository_AllByIDs(t *testing.T) {
	t.Parallel()

	e0 := newEntity()
	e1 := newEntity()
	repo := tests.NewMemoryRepository[Entity, EntityID]()
	repo.Create(ctx, e0)
	repo.Create(ctx, newEntity())
	repo.Create(ctx, e1)
	repo.Create(ctx, newEntity())

	all, err := repo.AllByIDs(ctx, nil)
	assert.NoError(t, err)
	assert.Empty(t, all)

	all, err = repo.AllByIDs(ctx, []EntityID{})
	assert.NoError(t, err)
	assert.Empty(t, all)

	all, err = repo.AllByIDs(ctx, []EntityID{e0.ID, e1.ID})
	assert.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestMemoryRepository_ExistsByID(t *testing.T) {
	t.Parallel()

	repo := tests.NewMemoryRepository[Entity, EntityID]()

	ex, err := repo.ExistsByID(ctx, EntityID(uuid.New().String()))
	assert.ErrorIs(t, err, tests.ErrNotFound)
	assert.False(t, ex, "id does not exist")

	repo.Create(ctx, defaultEntity)

	ex, err = repo.ExistsByID(ctx, defaultEntity.ID)
	assert.NoError(t, err)
	assert.True(t, ex, "id does exist")
}

func TestMemoryRepository_ExistByIDs(t *testing.T) {
	t.Parallel()

	e0 := newEntity()
	e1 := newEntity()
	repo := tests.NewMemoryRepository[Entity, EntityID]()
	repo.Create(ctx, e0)
	repo.Create(ctx, newEntity())
	repo.Create(ctx, e1)
	repo.Create(ctx, newEntity())

	ex, err := repo.ExistByIDs(ctx, nil)
	assert.NoError(t, err)
	assert.False(t, ex)

	ex, err = repo.ExistByIDs(ctx, []EntityID{})
	assert.NoError(t, err)
	assert.False(t, ex)

	ex, err = repo.ExistByIDs(ctx, []EntityID{e0.ID, e1.ID})
	assert.NoError(t, err)
	assert.True(t, ex)

	ex, err = repo.ExistByIDs(ctx, []EntityID{newEntity().ID})
	assert.ErrorIs(t, err, tests.ErrNotFound)
	assert.False(t, ex)
}

func TestMemoryRepository_Length(t *testing.T) {
	t.Parallel()

	repo := tests.NewMemoryRepository[Entity, EntityID]()

	count, err := repo.Length(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "new repository should be empty")

	repo.Create(ctx, newEntity())
	repo.Create(ctx, newEntity())

	count, err = repo.Length(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 2, count, "should have two entities")
}

func TestMemoryRepository_Save(t *testing.T) {
	t.Parallel()

	t.Run("save", func(t *testing.T) {
		t.Parallel()
		repo := tests.NewMemoryRepository[Entity, EntityID]()

		entity := newEntity()
		err := repo.Save(ctx, entity)
		assert.NoError(t, err)

		c, _ := repo.Count(ctx)
		assert.Equal(t, 1, c)
	})

	t.Run("save multiple times", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[Entity, EntityID]()

		entity := newEntity()
		err := repo.Save(ctx, entity)
		assert.NoError(t, err)

		entity.Name = "new-name"
		err = repo.Save(ctx, entity)
		assert.NoError(t, err)

		c, _ := repo.Count(ctx)
		assert.Equal(t, 1, c)

		e, _ := repo.FindByID(ctx, entity.ID)
		assert.Equal(t, entity, e)
	})

	t.Run("missing id", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[Entity, EntityID]()

		err := repo.Save(ctx, Entity{})
		assert.ErrorIs(t, err, tests.ErrSaveFailed)
	})
}

func TestMemoryRepository_SaveAll(t *testing.T) {
	t.Parallel()

	t.Run("save all", func(t *testing.T) {
		t.Parallel()

		e := newEntity()
		repo := tests.NewMemoryRepository[Entity, EntityID]()
		repo.Create(ctx, e)

		err := repo.SaveAll(ctx, []Entity{e, newEntity(), newEntity()})
		assert.NoError(t, err)

		c, _ := repo.Count(ctx)
		assert.Equal(t, 3, c)
	})

	t.Run("missing id", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[Entity, EntityID]()

		err := repo.SaveAll(ctx, []Entity{newEntity(), newEntity(), {}, newEntity()})
		assert.ErrorIs(t, err, tests.ErrSaveFailed)

		empty, _ := repo.IsEmpty(ctx)
		assert.True(t, empty)
	})
}

func TestMemoryRepository_AddAll(t *testing.T) {
	t.Parallel()

	t.Run("add all", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[Entity, EntityID]()

		repo.AddAll(ctx, []Entity{newEntity(), defaultEntity, newEntity()})

		c, _ := repo.Count(ctx)
		assert.Equal(t, 3, c)
	})

	t.Run("entity already exists", func(t *testing.T) {
		t.Parallel()

		repo := tests.NewMemoryRepository[Entity, EntityID]()
		repo.Save(ctx, defaultEntity)

		err := repo.AddAll(ctx, []Entity{newEntity(), defaultEntity, newEntity()})
		assert.ErrorIs(t, err, tests.ErrAlreadyExists)

		c, _ := repo.Count(ctx)
		assert.Equal(t, 1, c)
	})
}

func TestMemoryRepository_DeleteByID(t *testing.T) {
	t.Parallel()

	repo := tests.NewMemoryRepository[Entity, EntityID]()
	repo.Create(ctx, defaultEntity)

	err := repo.DeleteByID(ctx, defaultEntity.ID)
	assert.NoError(t, err)

	empty, _ := repo.Empty(ctx)
	assert.True(t, empty)
}

func TestMemoryRepository_DeleteByIDs(t *testing.T) {
	t.Parallel()

	e0 := newEntity()
	e1 := newEntity()
	e2 := newEntity()
	repo := tests.NewMemoryRepository[Entity, EntityID]()
	repo.AddAll(ctx, []Entity{e0, e1, e2})

	err := repo.DeleteByIDs(ctx, []EntityID{e0.ID, e2.ID})
	assert.NoError(t, err)

	c, _ := repo.Count(ctx)
	assert.Equal(t, 1, c)
}

func TestMemoryRepository_DeleteAll(t *testing.T) {
	t.Parallel()

	repo := tests.NewMemoryRepository[Entity, EntityID]()
	repo.Create(ctx, defaultEntity)

	err := repo.DeleteAll(ctx)
	assert.NoError(t, err)

	c, _ := repo.Count(ctx)
	assert.Equal(t, 0, c)
}

func TestEntityWithoutID(t *testing.T) {
	t.Parallel()

	repo := tests.NewMemoryRepository[EntityWithoutID, EntityID]()

	assert.Panics(t, func() {
		repo.Save(ctx, EntityWithoutID{})
	})
}

// --- --- --- TEST DATA --- --- ---

var (
	ctx           = context.Background()
	defaultEntity = newEntity()
)

type EntityID string

type Entity struct {
	ID   EntityID
	Name string
}

type EntityWithoutID struct {
	Name string
}

func newEntity() Entity {
	return Entity{
		ID:   EntityID(uuid.New().String()),
		Name: gofakeit.Name(),
	}
}
