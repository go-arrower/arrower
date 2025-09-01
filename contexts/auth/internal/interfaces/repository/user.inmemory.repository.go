package repository

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"

	"github.com/go-arrower/arrower/arepo"
	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
)

func NewUserMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		MemoryRepository: arepo.NewMemoryRepository[domain.User, domain.ID](),
		tokens:           make(map[uuid.UUID]domain.VerificationToken),
	}
}

type MemoryRepository struct {
	*arepo.MemoryRepository[domain.User, domain.ID]

	tokens map[uuid.UUID]domain.VerificationToken
}

func (repo *MemoryRepository) All(ctx context.Context, filter domain.Filter) ([]domain.User, error) {
	all, err := repo.MemoryRepository.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].Login < all[j].Login
	})

	users := []domain.User{}
	limit := filter.Limit
	found := false

	for _, user := range all {
		// skip Logins before the offset
		if !found && filter.Offset != "" && user.Login != filter.Offset {
			continue
		}

		// skip the found = same as offset
		found = true

		if filter.Offset == user.Login {
			continue
		}

		// append up to the limit
		users = append(users, user)
		limit--

		if limit == 0 {
			return users, nil
		}
	}

	return users, nil
}

func (repo *MemoryRepository) FindByLogin(ctx context.Context, login domain.Login) (domain.User, error) {
	all, _ := repo.MemoryRepository.All(ctx)

	for _, u := range all {
		if u.Login == login {
			return u, nil
		}
	}

	return domain.User{}, domain.ErrNotFound
}

func (repo *MemoryRepository) ExistsByLogin(ctx context.Context, login domain.Login) (bool, error) {
	all, _ := repo.MemoryRepository.All(ctx)

	for _, u := range all {
		if u.Login == login {
			return true, nil
		}
	}

	return false, domain.ErrNotFound
}

func (repo *MemoryRepository) CreateVerificationToken(
	_ context.Context,
	token domain.VerificationToken,
) error {
	if token.Token().String() == "" {
		return fmt.Errorf("missing ID: %w", domain.ErrPersistenceFailed)
	}

	repo.Lock()
	defer repo.Unlock()

	repo.tokens[token.Token()] = token

	return nil
}

func (repo *MemoryRepository) VerificationTokenByToken(
	_ context.Context,
	tokenID uuid.UUID,
) (domain.VerificationToken, error) {
	for _, t := range repo.tokens {
		if t.Token() == tokenID {
			return t, nil
		}
	}

	return domain.VerificationToken{}, domain.ErrNotFound
}

var _ domain.Repository = (*MemoryRepository)(nil)
