package application_test

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v6"

	"github.com/go-arrower/arrower/contexts/auth"
	"github.com/go-arrower/arrower/contexts/auth/internal/application"
	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
	"github.com/go-arrower/arrower/setting"
)

const (
	validUserLogin       = "0@test.com"
	notVerifiedUserLogin = "1@test.com"
	blockedUserLogin     = "2@test.com"
	newUserLogin         = "99@test.com"

	strongPassword     = "R^&npAL2iu&M6S"                                               //nolint:gosec // gosec is right, but it's testdata
	strongPasswordHash = "$2a$10$T7Bq1sNmHoGlGJUsHoF1A.S3oy.P3iLT6MoVXi6WvNdq1jbE.TnZy" // hash of strongPassword

	sessionKey = "session-key"
	userAgent  = "arrower/1"
	ip         = "127.0.0.1"
)

const (
	user0Login            = "0@test.com"
	userIDZero            = domain.ID("00000000-0000-0000-0000-000000000000")
	userNotVerifiedUserID = domain.ID("00000000-0000-0000-0000-000000000001")
	userBlockedUserID     = domain.ID("00000000-0000-0000-0000-000000000002")
)

var (
	ctx = context.Background()

	userVerified = domain.User{
		ID:           userIDZero,
		Login:        user0Login,
		PasswordHash: domain.PasswordHash(strongPasswordHash),
		Verified:     domain.BoolFlag{}.SetTrue(),
		Sessions: []domain.Session{{
			ID:        sessionKey,
			CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Now().UTC().Add(time.Hour),
			Device:    domain.Device{},
		}},
	}
	userNotVerified = domain.User{
		ID:           userNotVerifiedUserID,
		Login:        user0Login,
		PasswordHash: domain.PasswordHash(strongPasswordHash),
		Verified:     domain.BoolFlag{}.SetFalse(),
	}
	userBlocked = domain.User{
		ID:           userBlockedUserID,
		Login:        user0Login,
		PasswordHash: domain.PasswordHash(strongPasswordHash),
		Blocked:      domain.BoolFlag{}.SetTrue(),
	}

	resolvedIP = domain.ResolvedIP{
		IP:          net.ParseIP(ip),
		Country:     "-",
		CountryCode: "-",
		Region:      "-",
		City:        "-",
	}
)

func registrator(repo domain.Repository) *domain.RegistrationService {
	settings := setting.NewInMemorySettings()
	settings.Save(ctx, auth.SettingAllowRegistration, setting.NewValue(true))

	return domain.NewRegistrationService(settings, repo)
}

func authentificator() *domain.AuthenticationService {
	settings := setting.NewInMemorySettings()
	settings.Save(ctx, auth.SettingAllowLogin, setting.NewValue(true))

	return domain.NewAuthenticationService(settings)
}

func empty(field string) func(req *application.RegisterUserRequest) {
	return func(req *application.RegisterUserRequest) {
		e := reflect.TypeOf(*req)
		for i := range e.NumField() {
			if e.Field(i).Name == strings.TrimSpace(field) {
				reflect.ValueOf(req).Elem().Field(i).SetZero()
				return
			}

			if strings.ToLower(e.Field(i).Name) == strings.ToLower(strings.TrimSpace(field)) ||
				strings.Contains(strings.ToLower(e.Field(i).Name), strings.ToLower(strings.TrimSpace(field))) ||
				strings.Contains(strings.ToLower(strings.TrimSpace(field)), strings.ToLower(e.Field(i).Name)) {
				fmt.Printf("field with similar name found, did you mean: `%s`?\n", e.Field(i).Name) //nolint:forbidigo // show useful debug information to developer
				continue
			}
		}

		panic("no field `" + field + "` found in struct `" + e.Name() + "`")
	}
}

func with(field string, value string) func(req *application.RegisterUserRequest) {
	return func(req *application.RegisterUserRequest) {
		e := reflect.TypeOf(*req)
		for i := range e.NumField() {
			if e.Field(i).Name == strings.TrimSpace(field) {
				reflect.ValueOf(req).Elem().FieldByName(field).SetString(value)
				return
			}

			if strings.ToLower(e.Field(i).Name) == strings.ToLower(strings.TrimSpace(field)) ||
				strings.Contains(strings.ToLower(e.Field(i).Name), strings.ToLower(strings.TrimSpace(field))) ||
				strings.Contains(strings.ToLower(strings.TrimSpace(field)), strings.ToLower(e.Field(i).Name)) {
				fmt.Printf("field with similar name found, did you mean: `%s`?\n", e.Field(i).Name) //nolint:forbidigo // show useful debug information to developer
				continue
			}
		}

		panic("no field `" + field + "` found in struct `" + e.Name() + "`")
	}
}

func registerUserRequest(opts ...func(res *application.RegisterUserRequest)) application.RegisterUserRequest {
	password := gofakeit.Password(true, true, true, true, false, 12)
	req := application.RegisterUserRequest{
		RegisterEmail:          gofakeit.Email(),
		Password:               password,
		PasswordConfirmation:   password,
		AcceptedTermsOfService: true,
		UserAgent:              gofakeit.UserAgent(),
		IP:                     gofakeit.IPv4Address(),
		SessionKey:             gofakeit.UUID(),
	}

	for _, opt := range opts {
		opt(&req)
	}

	return req
}
