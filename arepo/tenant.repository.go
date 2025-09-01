package arepo

import "context"

// TenantRepository is a general purpose interface documenting
// which methods are available by the generic MemoryTenantRepository.
// tID is the primary key for the tenant and eID of the entity and needs to be of one of the underlying types.
// If your repository needs additional methods, you can extend your own repository easily to tune it to your use case.
type TenantRepository[tID id, E any, eID id] interface { //nolint:interfacebloat,lll // showcase of all methods that are possible
	NextID(ctx context.Context) (eID, error)

	Create(ctx context.Context, tenantID tID, entity E) error
	Read(ctx context.Context, tenantID tID, id eID) (E, error)
	Update(ctx context.Context, tenantID tID, entity E) error
	Delete(ctx context.Context, tenantID tID, entity E) error

	All(ctx context.Context) ([]E, error)
	AllOfTenant(ctx context.Context, tenantID tID) ([]E, error) // rename AllOf
	AllByIDs(ctx context.Context, tenantID tID, ids []eID) ([]E, error)
	FindAll(ctx context.Context) ([]E, error)
	FindAllOfTenant(ctx context.Context, tenantID tID) ([]E, error)
	FindByID(ctx context.Context, tenantID tID, id eID) (E, error)
	FindByIDs(ctx context.Context, tenantID tID, ids []eID) ([]E, error)
	Exists(ctx context.Context, tenantID tID, id eID) (bool, error)
	ExistsByID(ctx context.Context, tenantID tID, id eID) (bool, error)
	ExistByIDs(ctx context.Context, tenantID tID, ids []eID) (bool, error)
	ExistAll(ctx context.Context, tenantID tID, ids []eID) (bool, error)
	Contains(ctx context.Context, tenantID tID, id eID) (bool, error)
	ContainsID(ctx context.Context, tenantID tID, id eID) (bool, error)
	ContainsIDs(ctx context.Context, tenantID tID, ids []eID) (bool, error)
	ContainsAll(ctx context.Context, tenantID tID, ids []eID) (bool, error)

	Save(ctx context.Context, tenantID tID, entity E) error
	SaveAll(ctx context.Context, tenantID tID, entities []E) error
	UpdateAll(ctx context.Context, tenantID tID, entities []E) error
	Add(ctx context.Context, tenantID tID, entity E) error
	AddAll(ctx context.Context, tenantID tID, entities []E) error

	Count(ctx context.Context) (int, error)
	CountOfTenant(ctx context.Context, tenantID tID) (int, error)
	Length(ctx context.Context) (int, error)
	LengthOfTenant(ctx context.Context, tenantID tID) (int, error)

	DeleteByID(ctx context.Context, tenantID tID, id eID) error
	DeleteByIDs(ctx context.Context, tenantID tID, ids []eID) error
	DeleteAll(ctx context.Context) error
	DeleteAllOfTenant(ctx context.Context, tenantID tID) error
	Clear(ctx context.Context) error
	ClearTenant(ctx context.Context, tenantID tID) error
}
