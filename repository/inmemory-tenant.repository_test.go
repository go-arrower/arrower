package repository_test

import (
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/repository"
)

func TestNewMemoryTenantRepository(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, Entity, EntityID]()
		assert.NotNil(t, repo)
	})

	t.Run("load from store", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, Entity, EntityID](repository.WithStore(testStoreSuccessEntity(t)))
		assert.NotNil(t, repo)
	})

	t.Run("load from store fails", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			repository.NewMemoryTenantRepository[Tenant, TenantID, Entity, EntityID](repository.WithStore(testStoreLoadFails()))
		})
	})
}

func TestMemoryTenantRepository_Create(t *testing.T) {
	t.Parallel()

	t.Run("create", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, Entity, EntityID](repository.WithStore(testStoreSuccessEntity(t)))
		err := repo.Create(ctx, defaultTenant.ID, defaultEntity)
		assert.NoError(t, err)

		got, err := repo.Read(ctx, defaultTenant.ID, defaultEntity.ID)
		assert.NoError(t, err)
		assert.Equal(t, defaultEntity, got)
	})

	t.Run("create same again", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, Entity, EntityID]()

		err := repo.Create(ctx, defaultTenant.ID, defaultEntity)
		assert.NoError(t, err)

		err = repo.Create(ctx, defaultTenant.ID, defaultEntity)
		assert.ErrorIs(t, err, repository.ErrAlreadyExists, "already exists")
	})

	t.Run("missing entity id", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, Entity, EntityID]()

		err := repo.Create(ctx, defaultTenant.ID, Entity{})
		assert.ErrorIs(t, err, repository.ErrSaveFailed)
	})

	t.Run("store fails", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, Entity, EntityID](repository.WithStore(testStoreStoreFails()))

		err := repo.Create(ctx, defaultTenant.ID, defaultEntity)
		assert.Error(t, err)
	})
}

func TestMemoryTenantRepository_Read(t *testing.T) {
	t.Parallel()

	repo := repository.NewMemoryTenantRepository[Tenant, TenantID, Entity, EntityID]()

	t.Run("tenant does not exist", func(t *testing.T) {
		t.Parallel()

		e, err := repo.Read(ctx, testTenant().ID, "")
		assert.ErrorIs(t, err, repository.ErrNotFound)
		assert.Empty(t, e)
	})

	t.Run("entity does not exist", func(t *testing.T) {
		t.Parallel()

		repo.Create(ctx, defaultTenant.ID, testEntity())

		e, err := repo.Read(ctx, defaultTenant.ID, testEntity().ID)
		assert.ErrorIs(t, err, repository.ErrNotFound)
		assert.Empty(t, e)
	})
}

func TestMemoryTenantRepository_Update(t *testing.T) {
	t.Parallel()

	t.Run("update", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, Entity, EntityID](repository.WithStore(testStoreSuccessEntity(t)))
		entity := testEntity()
		repo.Create(ctx, defaultTenant.ID, entity)

		entity.Name = gofakeit.Name()
		err := repo.Update(ctx, defaultTenant.ID, entity)
		assert.NoError(t, err)

		e, err := repo.Read(ctx, defaultTenant.ID, entity.ID)
		assert.NoError(t, err)
		assert.Equal(t, entity, e)
	})

	t.Run("does not exist yet", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, Entity, EntityID]()

		err := repo.Update(ctx, defaultTenant.ID, defaultEntity)
		assert.ErrorIs(t, err, repository.ErrSaveFailed)
	})

	t.Run("missing id", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, Entity, EntityID]()
		repo.Create(ctx, defaultTenant.ID, defaultEntity)

		err := repo.Update(ctx, defaultTenant.ID, Entity{})
		assert.ErrorIs(t, err, repository.ErrSaveFailed)
	})

	t.Run("store fails", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, Entity, EntityID](repository.WithStore(testStoreStoreFails()))

		entity := testEntity()
		repo.Create(ctx, defaultTenant.ID, entity)

		entity.Name = gofakeit.Name()
		err := repo.Update(ctx, defaultTenant.ID, entity)
		assert.Error(t, err)
	})
}

func TestMemoryTenantRepository_Delete(t *testing.T) {
	t.Parallel()

	t.Run("delete", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, Entity, EntityID](repository.WithStore(testStoreSuccessEntity(t)))
		repo.Create(ctx, defaultTenant.ID, defaultEntity)

		err := repo.Delete(ctx, defaultTenant.ID, defaultEntity)
		assert.NoError(t, err)

		e, err := repo.Read(ctx, defaultTenant.ID, defaultEntity.ID)
		assert.ErrorIs(t, err, repository.ErrNotFound)
		assert.Empty(t, e)
	})

	t.Run("delete non existing tenant", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, Entity, EntityID](repository.WithStore(testStoreSuccessEntity(t)))
		repo.Create(ctx, defaultTenant.ID, defaultEntity)

		err := repo.Delete(ctx, testTenant().ID, defaultEntity)
		assert.NoError(t, err)
	})

	t.Run("multiple delete", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, Entity, EntityID](repository.WithStore(testStoreSuccessEntity(t)))
		repo.Create(ctx, defaultTenant.ID, defaultEntity)

		err := repo.Delete(ctx, defaultTenant.ID, defaultEntity)
		assert.NoError(t, err)

		err = repo.Delete(ctx, defaultTenant.ID, defaultEntity)
		assert.NoError(t, err)
	})

	t.Run("store fails", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, Entity, EntityID](repository.WithStore(testStoreStoreFails()))
		repo.Create(ctx, defaultTenant.ID, defaultEntity)

		err := repo.Delete(ctx, defaultTenant.ID, defaultEntity)
		assert.Error(t, err)
	})
}

func TestMemoryTenantRepository_All(t *testing.T) {
	t.Parallel()

	repo := repository.NewMemoryTenantRepository[Tenant, TenantID, Entity, EntityID]()

	all, err := repo.All(ctx, defaultTenant.ID)
	assert.NoError(t, err)
	assert.NotNil(t, all)
	assert.Empty(t, all, "new repository should be empty")

	repo.Create(ctx, defaultTenant.ID, testEntity())
	repo.Create(ctx, defaultTenant.ID, testEntity())

	all, err = repo.All(ctx, defaultTenant.ID)
	assert.NoError(t, err)
	assert.Len(t, all, 2, "should have two entities")
}
