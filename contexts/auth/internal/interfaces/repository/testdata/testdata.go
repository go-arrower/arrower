package testdata

import (
	"time"

	"github.com/google/uuid"

	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
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

	RawPassword = "0Secret!"
)

var (
	Today, _ = domain.NewBirthday(
		domain.Day(time.Now().Day()),
		domain.Month(time.Now().Month()),
		domain.Year(time.Now().Year()),
	)

	UserZero = domain.User{
		ID:       UserIDZero,
		Login:    "0@test.com",
		Birthday: Today,
	}

	ValidToken = domain.NewVerificationToken(
		uuid.New(),
		UserIDZero,
		time.Now().UTC().Add(time.Hour),
	)
)
