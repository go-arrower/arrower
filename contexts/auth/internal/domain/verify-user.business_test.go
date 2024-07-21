package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/repository"
)

func TestVerificationService_NewVerificationToken(t *testing.T) {
	t.Parallel()

	t.Run("generate new token", func(t *testing.T) {
		t.Parallel()

		usr := newUser()
		repo := repository.NewUserMemoryRepository()
		repo.Save(ctx, usr)

		// action
		verifier := domain.NewVerificationService(repo)
		token, err := verifier.NewVerificationToken(ctx, usr)
		assert.NoError(t, err)
		assert.Equal(t, usr.ID, token.UserID())
		assert.NotEmpty(t, token.Token())
		assert.NotEmpty(t, token.ValidUntilUTC())

		// assert against the db
		tok, err := repo.VerificationTokenByToken(ctx, token.Token())
		assert.NoError(t, err)
		assert.Equal(t, token.Token(), tok.Token())
	})
}

func TestVerificationService_Verify(t *testing.T) {
	t.Parallel()

	t.Run("verify token", func(t *testing.T) {
		t.Parallel()

		// setup
		usr := newUser()
		repo := repository.NewUserMemoryRepository()
		repo.Save(ctx, usr)
		verifier := domain.NewVerificationService(repo)
		token, _ := verifier.NewVerificationToken(ctx, usr)

		// action
		err := verifier.Verify(ctx, &usr, token.Token())
		assert.NoError(t, err)
		assert.True(t, usr.IsVerified())

		// assert against the db
		u, _ := repo.FindByID(ctx, usr.ID)
		assert.True(t, u.IsVerified())
	})

	t.Run("verify expired token", func(t *testing.T) {
		t.Parallel()

		// setup
		usr := newUser()
		repo := repository.NewUserMemoryRepository()
		repo.Save(ctx, usr)
		verifier := domain.NewVerificationService(
			repo,
			domain.WithValidFor(time.Nanosecond), // expire almost immediately
		)
		token, _ := verifier.NewVerificationToken(ctx, usr)

		// action
		err := verifier.Verify(ctx, &usr, token.Token())
		assert.ErrorIs(t, err, domain.ErrVerificationFailed)

		// verify against the db
		u, _ := repo.FindByID(ctx, usr.ID)
		assert.False(t, u.IsVerified())
	})

	t.Run("verify an unknown token", func(t *testing.T) {
		t.Parallel()

		usr := newUser()
		repo := repository.NewUserMemoryRepository()
		repo.Save(ctx, usr)
		verifier := domain.NewVerificationService(repo)

		err := verifier.Verify(ctx, &usr, uuid.New())
		assert.ErrorIs(t, err, domain.ErrVerificationFailed)
		assert.False(t, usr.IsVerified())
	})
}
