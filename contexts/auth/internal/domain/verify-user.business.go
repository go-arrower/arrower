package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var ErrVerificationFailed = errors.New("verification failed")

func NewVerificationToken(token uuid.UUID, userID ID, validUntilUTC time.Time) VerificationToken {
	return VerificationToken{
		validUntil: validUntilUTC,
		userID:     userID,
		token:      token,
	}
}

// VerificationToken is a token a User receives (via email) and uses to verify his Login.
type VerificationToken struct {
	validUntil time.Time
	userID     ID
	token      uuid.UUID
}

func (t VerificationToken) Token() uuid.UUID {
	return t.token
}

func (t VerificationToken) UserID() ID {
	return t.userID
}

func (t VerificationToken) ValidUntilUTC() time.Time {
	return t.validUntil
}

func (t VerificationToken) isValid() bool {
	return !time.Now().UTC().After(t.validUntil)
}

type VerificationOpt func(vs *VerificationService)

// WithValidFor overwrites the time a VerificationToken is valid.
func WithValidFor(validTime time.Duration) VerificationOpt {
	return func(vs *VerificationService) {
		vs.validFor = validTime
	}
}

func NewVerificationService(repo Repository, opts ...VerificationOpt) *VerificationService {
	const oneWeek = time.Hour * 24 * 7 // default time a token is valid.

	verificationService := &VerificationService{
		repo:     repo,
		validFor: oneWeek,
	}

	for _, opt := range opts {
		opt(verificationService)
	}

	return verificationService
}

type VerificationService struct {
	repo     Repository
	validFor time.Duration
}

// NewVerificationToken creates a new VerificationToken and persists it.
func (s *VerificationService) NewVerificationToken(ctx context.Context, user User) (VerificationToken, error) {
	token := VerificationToken{
		token:      uuid.New(),
		validUntil: time.Now().UTC().Add(s.validFor),
		userID:     user.ID,
	}

	err := s.repo.CreateVerificationToken(ctx, token)
	if err != nil {
		return VerificationToken{}, fmt.Errorf("could not save new verification token: %w", err)
	}

	return token, nil
}

// Verify verifies a User with the given Token. If it is valid, the user is updated and persisted.
func (s *VerificationService) Verify(ctx context.Context, user *User, rawToken uuid.UUID) error {
	token, err := s.repo.VerificationTokenByToken(ctx, rawToken)
	if err != nil {
		return fmt.Errorf("%w: could not fetch verification token: %w", ErrVerificationFailed, err)
	}

	if token.UserID() != user.ID {
		return ErrVerificationFailed
	}

	if !token.isValid() {
		return ErrVerificationFailed
	}

	user.Verified = user.Verified.SetTrue()

	err = s.repo.Save(ctx, *user)
	if err != nil {
		return fmt.Errorf("%w: could not save user: %w", ErrVerificationFailed, err)
	}

	return nil
}
