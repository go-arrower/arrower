package infrastructure

import (
	"errors"
	"fmt"
	"net"
	"path"
	"runtime"

	"github.com/ip2location/ip2location-go/v9"

	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
)

var (
	ErrInvalidIP     = errors.New("invalid ip address")
	ErrResolveFailed = errors.New("resolving ip failed")
)

// NewIP2LocationService returns a Service that can resolve ip addresses to a country, region, and city.
//
// This site or product includes IP2Location LITE data available from
// <a href="https://lite.ip2location.com">https://lite.ip2location.com</a>.
func NewIP2LocationService(dbPath string) *IP2Location {
	const defaultPath = "data/IP-COUNTRY-REGION-CITY.BIN"

	if dbPath == "" {
		dbPath = defaultPath
	}

	return &IP2Location{
		dbPath: dbPath,
	}
}

type IP2Location struct {
	dbPath string
}

func (s *IP2Location) ResolveIP(ip string) (domain.ResolvedIP, error) {
	db, err := ip2location.OpenDB(s.dbPath)
	if err != nil {
		// if not found, it might have been called from test coed from another package:
		// then get the path relative to the runtime dir.
		_, f, _, _ := runtime.Caller(0)
		searchDir := path.Join(path.Dir(f))

		db, err = ip2location.OpenDB(searchDir + "/" + s.dbPath)
		if err != nil {
			return domain.ResolvedIP{}, fmt.Errorf("%w: %v", ErrResolveFailed, err)
		}
	}

	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return domain.ResolvedIP{}, ErrInvalidIP
	}

	results, err := db.Get_all(ipAddr.String())
	if err != nil {
		return domain.ResolvedIP{}, fmt.Errorf("%w: %v", ErrResolveFailed, err)
	}

	db.Close()

	return domain.ResolvedIP{
		IP:          ipAddr,
		Country:     results.Country_long,
		CountryCode: results.Country_short,
		Region:      results.Region,
		City:        results.City,
	}, nil
}
