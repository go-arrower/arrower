package init

import (
	"context"

	"github.com/go-arrower/arrower/arepo"
	"github.com/go-arrower/arrower/contexts/auth"
)

func NewMemoryAPI(opts ...arepo.Option) auth.API {
	return &MemoryAPI{
		repo: arepo.NewMemoryRepository[auth.User, auth.UserID](opts...),
	}
}

type MemoryAPI struct {
	repo *arepo.MemoryRepository[auth.User, auth.UserID]
}

func (api *MemoryAPI) SaveUser(ctx context.Context, user auth.User) error {
	return api.repo.Save(ctx, user)
}

func (api *MemoryAPI) FindUserByID(ctx context.Context, id auth.UserID) (auth.User, error) {
	return api.repo.FindByID(ctx, id)
}
