package domain_test

import (
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
)

func TestNewUser(t *testing.T) {
	t.Parallel()

	t.Run("missing user details", func(t *testing.T) {
		t.Parallel()

		tests := map[string]struct {
			registerEmail string
			password      string
		}{
			"no register email": {
				"",
				string(strongPasswordHash),
			},
			"weak pw": {
				"",
				"123",
			},
			"invalid email": {
				"invalid-email",
				gofakeit.Password(true, true, true, true, false, 8),
			},
		}

		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				usr, err := domain.NewUser(tt.registerEmail, tt.password)
				assert.ErrorIs(t, err, domain.ErrInvalidUserDetails)
				assert.Empty(t, usr)
			})
		}
	})

	t.Run("new user", func(t *testing.T) {
		t.Parallel()

		usr, err := domain.NewUser(userLogin, rawPassword)
		assert.NoError(t, err)
		assert.NotEmpty(t, usr.ID)
		assert.Equal(t, userLogin, string(usr.Login))
		assert.True(t, usr.Verified.IsFalse())
		assert.True(t, usr.Blocked.IsFalse())
		assert.True(t, usr.Superuser.IsFalse())
	})
}

func TestUser_IsVerified(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		user     domain.User
		expected bool
	}{
		"empty time": {
			domain.User{Verified: domain.BoolFlag{}},
			false,
		},
		"user": {
			domain.User{Verified: domain.BoolFlag(time.Now().UTC())},
			true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, tt.user.IsVerified())
		})
	}
}

func TestUser_IsBlocked(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		user     domain.User
		expected bool
	}{
		"empty time": {
			domain.User{Blocked: domain.BoolFlag{}},
			false,
		},
		"user": {
			domain.User{Blocked: domain.BoolFlag(time.Now().UTC())},
			true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, tt.user.IsBlocked())
		})
	}
}

func TestUser_Block(t *testing.T) {
	t.Parallel()

	user := domain.User{}
	assert.False(t, user.IsBlocked())

	user.Block()
	assert.True(t, user.IsBlocked())

	blockedAt := user.Blocked.At()
	user.Block()

	assert.Equal(t, blockedAt, user.Blocked.At(), "if user is blocked, new calls to block will not update the time")
}

func TestUser_Unblock(t *testing.T) {
	t.Parallel()

	user := domain.User{}
	assert.False(t, user.IsBlocked())

	user.Unblock()
	assert.False(t, user.IsBlocked(), "no change on already unblocked user")

	user.Block()
	user.Unblock()
	assert.False(t, user.IsBlocked())
}

func TestUser_IsSuperuser(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		user     domain.User
		expected bool
	}{
		"empty time": {
			domain.User{Superuser: domain.BoolFlag{}},
			false,
		},
		"superuser": {
			domain.User{Superuser: domain.BoolFlag(time.Now().UTC())},
			true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, tt.user.IsSuperuser())
		})
	}
}

func TestNewPasswordHash(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		pw  string
		err error
	}{
		"empty pw": {
			"",
			nil,
		},
		"pw": {
			"some-pw",
			nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := domain.NewPasswordHash(tt.pw)
			assert.Equal(t, tt.err, err)
		})
	}
}

func TestNewStrongPasswordHash(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		password string
	}{
		"too short": {
			"123456",
		},
		"missing lower case letter": {
			"1234567890",
		},
		"missing upper case letter": {
			"123456abc",
		},
		"missing number": {
			"abcdefghi",
		},
		"missing special character": {
			"123456abCD",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := domain.NewStrongPasswordHash(tt.password)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrInvalidUserDetails)
		})
	}
}

func TestName(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		fn    string
		ln    string
		dn    string
		expFN string
		expLN string
		expDN string
	}{
		"empty name": {
			"",
			"",
			"",
			"",
			"",
			"",
		},
		"full name": {
			"Arrower",
			"Project",
			"Arrower Project",
			"Arrower",
			"Project",
			"Arrower Project",
		},
		"sanitise name": {
			" Arrower",
			"Project ",
			" Arrower Project ",
			"Arrower",
			"Project",
			"Arrower Project",
		},
		"automatic capitalise": {
			"arrower",
			"project",
			"arrower project",
			"Arrower",
			"Project",
			"Arrower Project",
		},
		"build display name": {
			"arrower",
			"project",
			"",
			"Arrower",
			"Project",
			"Arrower Project",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			name := domain.NewName(tt.fn, tt.ln, tt.dn)
			assert.Equal(t, tt.expFN, name.FirstName())
			assert.Equal(t, tt.expLN, name.LastName())
			assert.Equal(t, tt.expDN, name.DisplayName())
		})
	}
}

func TestNewBirthday(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		day      domain.Day
		month    domain.Month
		year     domain.Year
		expected error
	}{
		"valid": {
			1,
			1,
			2000,
			nil,
		},
		"invalid day zero": {
			0,
			1,
			2000,
			domain.ErrInvalidBirthday,
		},
		"invalid month zero": {
			1,
			0,
			2000,
			domain.ErrInvalidBirthday,
		},
		"too old": {
			1,
			1,
			1000,
			domain.ErrInvalidBirthday,
		},
		"invalid day": {
			32,
			1,
			2000,
			domain.ErrInvalidBirthday,
		},
		"invalid month": {
			1,
			13,
			2000,
			domain.ErrInvalidBirthday,
		},
		"in the future": {
			1,
			1,
			3000,
			domain.ErrInvalidBirthday,
		},
		"valid februrary": {
			29,
			2,
			2020,
			nil,
		},
		"invalid date": {
			31,
			4,
			2020,
			domain.ErrInvalidBirthday,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, got := domain.NewBirthday(tt.day, tt.month, tt.year)
			assert.ErrorIs(t, got, tt.expected)
		})
	}
}

func TestDevice(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		userAgent      string
		expectedName   string
		expectedOS     string
		expectedString string
	}{
		"empty": {
			"",
			"",
			"",
			"",
		},
		"valid": {
			"Mozilla/5.0 (Linux; Android 4.3; GT-I9300 Build/JSS15J) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.125 Mobile Safari/537.36",
			"Chrome v59.0.3071.125",
			"Android v4.3",
			"Chrome v59.0.3071.125 Android v4.3",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expectedName, domain.NewDevice(tt.userAgent).Name())
			assert.Equal(t, tt.expectedOS, domain.NewDevice(tt.userAgent).OS())
			assert.Equal(t, tt.expectedString, domain.NewDevice(tt.userAgent).String())
			assert.Equal(t, tt.userAgent, domain.NewDevice(tt.userAgent).UserAgent())
		})
	}
}

func TestBoolFlag(t *testing.T) {
	t.Parallel()

	flag := domain.BoolFlag{}
	assert.False(t, flag.IsTrue(), "empty flag is not true")
	assert.True(t, flag.IsFalse(), "empty flag is false")
	assert.Empty(t, flag.At())

	flag = domain.BoolFlag(time.Now().UTC())
	assert.True(t, flag.IsTrue())
	assert.False(t, flag.IsFalse())
	assert.NotEmpty(t, flag.At())
}

func TestBoolFlag_SetTrue(t *testing.T) {
	t.Parallel()

	t.Run("set true", func(t *testing.T) {
		t.Parallel()

		flag := domain.BoolFlag{}
		assert.False(t, flag.IsTrue())

		flag = flag.SetTrue()
		assert.True(t, flag.IsTrue())
	})

	t.Run("if flag was true, time does not change", func(t *testing.T) {
		t.Parallel()

		flag := domain.BoolFlag{}
		assert.False(t, flag.IsTrue())

		flag = flag.SetTrue()
		assert.True(t, flag.IsTrue())
		trueAt := flag.At()

		flag = flag.SetTrue()
		assert.True(t, flag.IsTrue())
		assert.Equal(t, trueAt, flag.At(), "second call does not change the time")
	})
}

func TestBoolFlag_SetFalse(t *testing.T) {
	t.Parallel()

	t.Run("set false", func(t *testing.T) {
		t.Parallel()

		flag := domain.BoolFlag(time.Now().UTC())
		assert.True(t, flag.IsTrue())

		flag = flag.SetFalse()
		assert.True(t, flag.IsFalse())

		flag = flag.SetFalse()
		assert.True(t, flag.IsFalse(), "subsequent calls stay false")
	})
}
