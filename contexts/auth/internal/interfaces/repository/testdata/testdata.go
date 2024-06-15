package testdata

import (
	"time"

	"github.com/go-arrower/arrower/contexts/auth/internal/domain"

	"github.com/google/uuid"
)

const (
	UserIDZero = domain.ID("00000000-0000-0000-0000-000000000000")
	UserIDOne  = domain.ID("00000000-0000-0000-0000-000000000001")

	UserIDNew       = domain.ID("00000000-0000-0000-0000-000000000010")
	UserIDNotExists = domain.ID("00000000-0000-0000-0000-999999999999")
	UserIDNotValid  = domain.ID("invalid-id")

	ValidLogin = domain.Login("0@test.com")
	NotExLogin = domain.Login("invalid-login")

	SessionKey = "session-key"
	UserAgent  = "arrower/1"
)

var (
	UserZero = domain.User{
		ID:    UserIDZero,
		Login: "0@test.com",
	}

	ValidToken = domain.NewVerificationToken(
		uuid.New(),
		UserIDZero,
		time.Now().UTC().Add(time.Hour),
	)
)
