package arepo_test

import (
	"context"
	"fmt"

	"github.com/go-arrower/arrower/arepo"
)

func Example_extendRepositoryWithNewMethods() {
	ctx := context.Background()

	repo := NewUserMemoryRepository()
	repo.Add(ctx, User{ID: 1, Login: "hello@arrower.org"})

	u, _ := repo.FindByLogin(ctx, "hello@arrower.org")
	fmt.Println(u)

	// Output: {1 hello@arrower.org}
}

type UserID int

type User struct {
	ID    UserID
	Login string
}

func NewUserMemoryRepository() *UserMemoryRepository {
	return &UserMemoryRepository{
		MemoryRepository: arepo.NewMemoryRepository[User, UserID](),
	}
}

type UserMemoryRepository struct {
	*arepo.MemoryRepository[User, UserID]
}

// FindByLogin implements a custom method, that is not supported by the Repository out of the box.
func (repo *UserMemoryRepository) FindByLogin(ctx context.Context, login string) (User, error) {
	all, _ := repo.All(ctx)

	for _, u := range all {
		if u.Login == login {
			return u, nil
		}
	}

	return User{}, arepo.ErrNotFound
}
