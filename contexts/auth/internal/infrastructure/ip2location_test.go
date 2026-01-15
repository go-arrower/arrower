package infrastructure_test

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
	"github.com/go-arrower/arrower/contexts/auth/internal/infrastructure"
)

func TestNewIp2LocationService(t *testing.T) {
	t.Parallel()
	t.Skip("skip for now, ip2location is not available in this repo")

	t.Run("get with default db path", func(t *testing.T) {
		t.Parallel()

		ip := infrastructure.NewIP2LocationService("")
		_, err := ip.ResolveIP("127.0.0.1")
		assert.NoError(t, err)
	})

	t.Run("get with db path", func(t *testing.T) {
		t.Parallel()

		ip := infrastructure.NewIP2LocationService("data/IP-COUNTRY-REGION-CITY.BIN")
		_, err := ip.ResolveIP("127.0.0.1")
		assert.NoError(t, err)
	})
}

func TestIP2Location_ResolveIP(t *testing.T) {
	t.Parallel()
	t.Skip("skip for now, ip2location is not available in this repo")

	tests := []struct {
		testName string
		ip       string
		expIP    domain.ResolvedIP
		err      error
	}{
		{
			"empty ip",
			"",
			domain.ResolvedIP{},
			infrastructure.ErrInvalidIP,
		},
		{
			"invalid ip",
			"this-is-not-an-ip-address",
			domain.ResolvedIP{},
			infrastructure.ErrInvalidIP,
		},
		{
			"valid ip",
			"87.118.100.175",
			domain.ResolvedIP{
				IP:          net.ParseIP("87.118.100.175"),
				Country:     "Germany",
				CountryCode: "DE",
				Region:      "Thuringen",
				City:        "Erfurt",
			},
			nil,
		},
	}

	ip := infrastructure.NewIP2LocationService("")

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			res, err := ip.ResolveIP(tt.ip)
			assert.Equal(t, tt.err, err)
			assert.Equal(t, tt.expIP, res)
		})
	}
}
