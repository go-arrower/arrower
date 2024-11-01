package init

import (
	"context"

	"github.com/go-arrower/arrower/contexts/auth"
	"github.com/go-arrower/arrower/repository"
)

func NewMemoryAPI(opts ...repository.Option) auth.API {
	return &MemoryAPI{
		repo: repository.NewMemoryRepository[auth.User, auth.UserID](opts...),
	}
}

type MemoryAPI struct {
	repo *repository.MemoryRepository[auth.User, auth.UserID]
}

func (api *MemoryAPI) SaveUser(ctx context.Context, user auth.User) error {
	return api.repo.Save(ctx, user)
}

func (api *MemoryAPI) FindUserByID(ctx context.Context, id auth.UserID) (auth.User, error) {
	return api.repo.FindByID(ctx, id)
}
