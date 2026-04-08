package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/text/language"

	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/repository/models"
)

func usersFromModel(ctx context.Context, queries *models.Queries, dbUsers []models.AuthUser) ([]domain.User, error) {
	users := make([]domain.User, len(dbUsers))

	for i, u := range dbUsers {
		user, err := userFromModel(ctx, queries, u)
		if err != nil {
			return nil, err
		}

		users[i] = user
	}

	return users, nil
}

func userFromModel(ctx context.Context, queries *models.Queries, dbUser models.AuthUser) (domain.User, error) {
	sess, err := queries.FindSessionsByUserID(ctx, uuid.NullUUID{UUID: dbUser.ID, Valid: true})
	if err != nil {
		return domain.User{},
			fmt.Errorf("%w: could not get sessions for user: %s: %v", domain.ErrNotFound, dbUser.ID.String(), err)
	}

	return userFromModelWithSession(dbUser, sess), nil
}

func userFromModelWithSession(dbUser models.AuthUser, sessions []models.AuthSession) domain.User {
	profile := make(map[string]string)
	for k, v := range dbUser.Profile {
		profile[k] = *v
	}

	birthday, err := domain.NewBirthday(
		domain.Day(dbUser.Birthday.Time.Day()),     //nolint:gosec // assume db does always only store valid dates
		domain.Month(dbUser.Birthday.Time.Month()), //nolint:gosec // assume db does always only store valid dates
		domain.Year(dbUser.Birthday.Time.Year()),   //nolint:gosec // assume db does always only store valid dates
	)
	if err != nil {
		panic("could not map birthday to domain format: " + err.Error())
	}

	locale, err := language.Parse(dbUser.Locale)
	if err != nil {
		panic("could not map locale to domain format: " + err.Error())
	}

	return domain.User{
		ID:                domain.ID(dbUser.ID.String()),
		Login:             domain.Login(dbUser.Login),
		PasswordHash:      domain.PasswordHash(dbUser.PasswordHash),
		RegisteredAt:      dbUser.CreatedAt.Time,
		Name:              domain.NewName(dbUser.NameFirstname, dbUser.NameLastname, dbUser.NameDisplayname),
		Birthday:          birthday,
		Locale:            domain.Locale(locale),
		TimeZone:          domain.TimeZone(dbUser.TimeZone),
		ProfilePictureURL: domain.URL(dbUser.PictureUrl),
		Profile:           profile,
		Verified:          domain.BoolFlag(dbUser.VerifiedAtUtc.Time),
		Blocked:           domain.BoolFlag(dbUser.BlockedAtUtc.Time),
		Superuser:         domain.BoolFlag(dbUser.SuperuserAtUtc.Time),
		Sessions:          sessionsFromModel(sessions),
	}
}

func sessionsFromModel(sess []models.AuthSession) []domain.Session {
	if sess == nil {
		return []domain.Session{}
	}

	sessions := make([]domain.Session, len(sess))

	for i := range sess {
		sessions[i] = domain.Session{
			ID:        string(sess[i].Key),
			CreatedAt: sess[i].CreatedAt.Time,
			ExpiresAt: sess[i].ExpiresAtUtc.Time,
			Device:    domain.NewDevice(sess[i].UserAgent),
		}
	}

	return sessions
}

func userToModel(user domain.User) models.UpsertUserParams {
	verifiedAt := pgtype.Timestamptz{Time: user.Verified.At(), Valid: true, InfinityModifier: pgtype.Finite}
	if user.Verified.At().Equal((time.Time{})) {
		verifiedAt = pgtype.Timestamptz{}
	}

	blockedAt := pgtype.Timestamptz{Time: user.Blocked.At(), Valid: true, InfinityModifier: pgtype.Finite}
	if user.Blocked.At().Equal((time.Time{})) {
		blockedAt = pgtype.Timestamptz{}
	}

	superUserAt := pgtype.Timestamptz{Time: user.Superuser.At(), Valid: true, InfinityModifier: pgtype.Finite}
	if user.Superuser.At().Equal((time.Time{})) {
		superUserAt = pgtype.Timestamptz{}
	}

	birthday, err := time.Parse("2006-01-02", user.Birthday.String())
	if err != nil {
		panic("could not map birthday to database format: " + err.Error())
	}

	profile := make(map[string]*string)
	for k, v := range user.Profile {
		profile[k] = &v
	}

	return models.UpsertUserParams{
		ID: uuid.MustParse(string(user.ID)),
		// only required for insert, otherwise the time will not be updated.
		CreatedAt:       pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true, InfinityModifier: pgtype.Finite},
		Login:           string(user.Login),
		PasswordHash:    string(user.PasswordHash),
		NameFirstname:   user.Name.FirstName(),
		NameLastname:    user.Name.LastName(),
		NameDisplayname: user.Name.DisplayName(),
		Birthday: pgtype.Date{
			Time:             birthday,
			InfinityModifier: 0,
			Valid:            true,
		},
		Locale:         language.Tag(user.Locale).String(),
		TimeZone:       string(user.TimeZone),
		PictureUrl:     string(user.ProfilePictureURL),
		Profile:        profile,
		VerifiedAtUtc:  verifiedAt,
		BlockedAtUtc:   blockedAt,
		SuperuserAtUtc: superUserAt,
	}
}
