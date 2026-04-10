package infrastructure

import (
	"fmt"
	"net"

	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
)

// NewIPNoopResolver returns a service that does not resolve ip addresses.
func NewIPNoopResolver() *IPNoopResolver {
	return &IPNoopResolver{}
}

type IPNoopResolver struct{}

var _ domain.IPResolver = (*IPNoopResolver)(nil)

func (s *IPNoopResolver) ResolveIP(ip string) (domain.ResolvedIP, error) {
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return domain.ResolvedIP{}, fmt.Errorf("%w: invalid ip address", domain.ErrResolveFailed)
	}

	return domain.ResolvedIP{
		IP:          ipAddr,
		Country:     "-",
		CountryCode: "-",
		Region:      "-",
		City:        "-",
	}, nil
}
