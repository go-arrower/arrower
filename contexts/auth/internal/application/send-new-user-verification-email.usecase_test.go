package application_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/contexts/auth/internal/application"
	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/repository"
)

func TestSendNewUserVerificationEmailJobHandler_H(t *testing.T) {
	t.Parallel()

	t.Run("success case", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewUserMemoryRepository()
		repo.Save(ctx, userNotVerified)

		handler := application.NewSendNewUserVerificationEmailJobHandler(slog.New(slog.DiscardHandler), repo)

		err := handler.H(context.Background(), application.NewUserVerificationEmail{
			UserID:     userNotVerifiedUserID,
			OccurredAt: time.Now().UTC(),
			IP:         resolvedIP,
			Device:     domain.NewDevice(userAgent),
		})
		assert.NoError(t, err)
	})
}
