package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/contexts/auth/internal/domain"

	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/repository"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/contexts/auth/internal/application"
)

func TestSendNewUserVerificationEmailJobHandler_H(t *testing.T) {
	t.Parallel()

	t.Run("success case", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository()
		repo.Save(ctx, userNotVerified)

		handler := application.NewSendNewUserVerificationEmailJobHandler(alog.NewTest(nil), repo)

		err := handler.H(context.Background(), application.NewUserVerificationEmail{
			UserID:     userNotVerifiedUserID,
			OccurredAt: time.Now().UTC(),
			IP:         resolvedIP,
			Device:     domain.NewDevice(userAgent),
		})
		assert.NoError(t, err)
	})
}
