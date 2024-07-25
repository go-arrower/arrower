package repository_test

//
//import (
//	"testing"
//
//	"github.com/brianvoe/gofakeit/v6"
//	"github.com/google/uuid"
//	"github.com/stretchr/testify/assert"
//
//	"github.com/go-arrower/arrower/repository"
//	"github.com/go-arrower/arrower/repository/testdata"
//)
//
//func TestNewMemoryTenantRepository(t *testing.T) {
//	t.Parallel()
//
//	t.Run("default", func(t *testing.T) {
//		t.Parallel()
//
//		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, testdata.Entity, testdata.EntityID]()
//		assert.NotNil(t, repo)
//	})
//
//	t.Run("load from store", func(t *testing.T) {
//		t.Parallel()
//
//		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, testdata.Entity, testdata.EntityID](repository.WithStore(testStoreSuccessEntity(t)))
//		assert.NotNil(t, repo)
//	})
//
//	t.Run("load from store fails", func(t *testing.T) {
//		t.Parallel()
//
//		assert.Panics(t, func() {
//			repository.NewMemoryTenantRepository[Tenant, TenantID, testdata.Entity, testdata.EntityID](repository.WithStore(testStoreLoadFails()))
//		})
//	})
//}
//
//func TestMemoryTenantRepository_Create(t *testing.T) {
//	t.Parallel()
//
//	t.Run("create", func(t *testing.T) {
//		t.Parallel()
//
//		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, testdata.Entity, testdata.EntityID](repository.WithStore(testStoreSuccessEntity(t)))
//		err := repo.Create(ctx, defaultTenant.ID, testdata.DefaultEntity)
//		assert.NoError(t, err)
//
//		got, err := repo.Read(ctx, defaultTenant.ID, testdata.DefaultEntity.ID)
//		assert.NoError(t, err)
//		assert.Equal(t, testdata.DefaultEntity, got)
//	})
//
//	t.Run("create same again", func(t *testing.T) {
//		t.Parallel()
//
//		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, testdata.Entity, testdata.EntityID]()
//
//		err := repo.Create(ctx, defaultTenant.ID, testdata.DefaultEntity)
//		assert.NoError(t, err)
//
//		err = repo.Create(ctx, defaultTenant.ID, testdata.DefaultEntity)
//		assert.ErrorIs(t, err, repository.ErrAlreadyExists, "already exists")
//	})
//
//	t.Run("missing entity id", func(t *testing.T) {
//		t.Parallel()
//
//		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, testdata.Entity, testdata.EntityID]()
//
//		err := repo.Create(ctx, defaultTenant.ID, testdata.Entity{})
//		assert.ErrorIs(t, err, repository.ErrSaveFailed)
//	})
//
//	t.Run("store fails", func(t *testing.T) {
//		t.Parallel()
//
//		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, testdata.Entity, testdata.EntityID](repository.WithStore(testStoreStoreFails()))
//
//		err := repo.Create(ctx, defaultTenant.ID, testdata.DefaultEntity)
//		assert.Error(t, err)
//	})
//}
//
//func TestMemoryTenantRepository_Read(t *testing.T) {
//	t.Parallel()
//
//	repo := repository.NewMemoryTenantRepository[Tenant, TenantID, testdata.Entity, testdata.EntityID]()
//
//	t.Run("tenant does not exist", func(t *testing.T) {
//		t.Parallel()
//
//		e, err := repo.Read(ctx, testTenant().ID, "")
//		assert.ErrorIs(t, err, repository.ErrNotFound)
//		assert.Empty(t, e)
//	})
//
//	t.Run("entity does not exist", func(t *testing.T) {
//		t.Parallel()
//
//		repo.Create(ctx, defaultTenant.ID, testdata.TestEntity())
//
//		e, err := repo.Read(ctx, defaultTenant.ID, testdata.TestEntity().ID)
//		assert.ErrorIs(t, err, repository.ErrNotFound)
//		assert.Empty(t, e)
//	})
//}
//
//func TestMemoryTenantRepository_Update(t *testing.T) {
//	t.Parallel()
//
//	t.Run("update", func(t *testing.T) {
//		t.Parallel()
//
//		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, testdata.Entity, testdata.EntityID](repository.WithStore(testStoreSuccessEntity(t)))
//		entity := testdata.TestEntity()
//		repo.Create(ctx, defaultTenant.ID, entity)
//
//		entity.Name = gofakeit.Name()
//		err := repo.Update(ctx, defaultTenant.ID, entity)
//		assert.NoError(t, err)
//
//		e, err := repo.FindByID(ctx, defaultTenant.ID, entity.ID)
//		assert.NoError(t, err)
//		assert.Equal(t, entity, e)
//	})
//
//	t.Run("does not exist yet", func(t *testing.T) {
//		t.Parallel()
//
//		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, testdata.Entity, testdata.EntityID]()
//
//		err := repo.Update(ctx, defaultTenant.ID, testdata.DefaultEntity)
//		assert.ErrorIs(t, err, repository.ErrSaveFailed)
//	})
//
//	t.Run("missing id", func(t *testing.T) {
//		t.Parallel()
//
//		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, testdata.Entity, testdata.EntityID]()
//		repo.Create(ctx, defaultTenant.ID, testdata.DefaultEntity)
//
//		err := repo.Update(ctx, defaultTenant.ID, testdata.Entity{})
//		assert.ErrorIs(t, err, repository.ErrSaveFailed)
//	})
//
//	t.Run("store fails", func(t *testing.T) {
//		t.Parallel()
//
//		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, testdata.Entity, testdata.EntityID](repository.WithStore(testStoreStoreFails()))
//
//		entity := testdata.TestEntity()
//		repo.Create(ctx, defaultTenant.ID, entity)
//
//		entity.Name = gofakeit.Name()
//		err := repo.Update(ctx, defaultTenant.ID, entity)
//		assert.Error(t, err)
//	})
//}
//
//func TestMemoryTenantRepository_Delete(t *testing.T) {
//	t.Parallel()
//
//	t.Run("delete", func(t *testing.T) {
//		t.Parallel()
//
//		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, testdata.Entity, testdata.EntityID](repository.WithStore(testStoreSuccessEntity(t)))
//		repo.Create(ctx, defaultTenant.ID, testdata.DefaultEntity)
//
//		err := repo.Delete(ctx, defaultTenant.ID, testdata.DefaultEntity)
//		assert.NoError(t, err)
//
//		e, err := repo.Read(ctx, defaultTenant.ID, testdata.DefaultEntity.ID)
//		assert.ErrorIs(t, err, repository.ErrNotFound)
//		assert.Empty(t, e)
//	})
//
//	t.Run("delete non existing tenant", func(t *testing.T) {
//		t.Parallel()
//
//		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, testdata.Entity, testdata.EntityID](repository.WithStore(testStoreSuccessEntity(t)))
//		repo.Create(ctx, defaultTenant.ID, testdata.DefaultEntity)
//
//		err := repo.Delete(ctx, testTenant().ID, testdata.DefaultEntity)
//		assert.NoError(t, err)
//	})
//
//	t.Run("multiple delete", func(t *testing.T) {
//		t.Parallel()
//
//		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, testdata.Entity, testdata.EntityID](repository.WithStore(testStoreSuccessEntity(t)))
//		repo.Create(ctx, defaultTenant.ID, testdata.DefaultEntity)
//
//		err := repo.Delete(ctx, defaultTenant.ID, testdata.DefaultEntity)
//		assert.NoError(t, err)
//
//		err = repo.Delete(ctx, defaultTenant.ID, testdata.DefaultEntity)
//		assert.NoError(t, err)
//	})
//
//	t.Run("store fails", func(t *testing.T) {
//		t.Parallel()
//
//		repo := repository.NewMemoryTenantRepository[Tenant, TenantID, testdata.Entity, testdata.EntityID](repository.WithStore(testStoreStoreFails()))
//		repo.Create(ctx, defaultTenant.ID, testdata.DefaultEntity)
//
//		err := repo.Delete(ctx, defaultTenant.ID, testdata.DefaultEntity)
//		assert.Error(t, err)
//	})
//}
//
//func TestMemoryTenantRepository_All(t *testing.T) {
//	t.Parallel()
//
//	repo := repository.NewMemoryTenantRepository[Tenant, TenantID, testdata.Entity, testdata.EntityID]()
//	all, err := repo.AllOfTenant(ctx, defaultTenant.ID)
//	assert.NoError(t, err)
//	assert.NotNil(t, all)
//	assert.Empty(t, all, "new repository should be empty")
//
//	repo.Create(ctx, defaultTenant.ID, testdata.TestEntity())
//	repo.Create(ctx, defaultTenant.ID, testdata.TestEntity())
//	repo.Create(ctx, testTenant().ID, testdata.TestEntity())
//
//	all, err = repo.All(ctx)
//	assert.NoError(t, err)
//	assert.Len(t, all, 3, "should have three entities")
//}
//
//func TestMemoryTenantRepository_AllOfTenant(t *testing.T) {
//	t.Parallel()
//
//	repo := repository.NewMemoryTenantRepository[Tenant, TenantID, testdata.Entity, testdata.EntityID]()
//
//	all, err := repo.AllOfTenant(ctx, defaultTenant.ID)
//	assert.NoError(t, err)
//	assert.NotNil(t, all)
//	assert.Empty(t, all, "new repository should be empty")
//
//	repo.Create(ctx, defaultTenant.ID, testdata.TestEntity())
//	repo.Create(ctx, defaultTenant.ID, testdata.TestEntity())
//
//	all, err = repo.AllOfTenant(ctx, defaultTenant.ID)
//	assert.NoError(t, err)
//	assert.Len(t, all, 2, "should have two entities")
//}
//
//func TestMemoryTenantRepository_AllByIDs(t *testing.T) {
//	t.Parallel()
//
//	e0 := testdata.TestEntity()
//	e1 := testdata.TestEntity()
//	repo := repository.NewMemoryTenantRepository[Tenant, TenantID, testdata.Entity, testdata.EntityID]()
//	repo.Create(ctx, defaultTenant.ID, e0)
//	repo.Create(ctx, defaultTenant.ID, testdata.TestEntity())
//	repo.Create(ctx, defaultTenant.ID, e1)
//	repo.Create(ctx, defaultTenant.ID, testdata.TestEntity())
//
//	all, err := repo.AllByIDs(ctx, defaultTenant.ID, nil)
//	assert.NoError(t, err)
//	assert.Empty(t, all)
//
//	all, err = repo.AllByIDs(ctx, defaultTenant.ID, []testdata.EntityID{})
//	assert.NoError(t, err, "empty list should not return an error")
//	assert.Empty(t, all)
//
//	all, err = repo.AllByIDs(ctx, defaultTenant.ID, []testdata.EntityID{e0.ID, e1.ID})
//	assert.NoError(t, err)
//	assert.Len(t, all, 2, "should find all ids")
//
//	all, err = repo.AllByIDs(ctx, defaultTenant.ID, []testdata.EntityID{
//		e0.ID,
//		testdata.EntityID(uuid.New().String()),
//	})
//	assert.ErrorIs(t, err, repository.ErrNotFound, "should not find all ids")
//	assert.Empty(t, all)
//}
//
//func TestMemoryTenantRepository_Contains(t *testing.T) {
//	t.Parallel()
//
//	repo := repository.NewMemoryTenantRepository[Tenant, TenantID, testdata.Entity, testdata.EntityID]()
//
//	ex, err := repo.ContainsID(ctx, defaultTenant.ID, testdata.TestEntity().ID)
//	assert.NoError(t, err)
//	assert.False(t, ex, "id should not exist")
//
//	repo.Create(ctx, defaultTenant.ID, testdata.DefaultEntity)
//
//	ex, err = repo.Contains(ctx, defaultTenant.ID, testdata.DefaultEntity.ID)
//	assert.NoError(t, err)
//	assert.True(t, ex, "id should exist")
//}
//
//func TestMemoryTenantRepository_ContainsAll(t *testing.T) {
//	t.Parallel()
//
//	e0 := testdata.TestEntity()
//	e1 := testdata.TestEntity()
//	repo := repository.NewMemoryTenantRepository[Tenant, TenantID, testdata.Entity, testdata.EntityID]()
//	repo.Create(ctx, defaultTenant.ID, e0)
//	repo.Create(ctx, defaultTenant.ID, testdata.TestEntity())
//	repo.Create(ctx, defaultTenant.ID, e1)
//	repo.Create(ctx, defaultTenant.ID, testdata.TestEntity())
//
//	ex, err := repo.ContainsAll(ctx, defaultTenant.ID, nil)
//	assert.NoError(t, err)
//	assert.False(t, ex)
//
//	ex, err = repo.ContainsAll(ctx, defaultTenant.ID, []testdata.EntityID{})
//	assert.NoError(t, err)
//	assert.False(t, ex)
//
//	ex, err = repo.ContainsAll(ctx, defaultTenant.ID, []testdata.EntityID{e0.ID, e1.ID})
//	assert.NoError(t, err)
//	assert.True(t, ex)
//
//	ex, err = repo.ContainsAll(ctx, defaultTenant.ID, []testdata.EntityID{testdata.TestEntity().ID})
//	assert.NoError(t, err)
//	assert.False(t, ex)
//}
