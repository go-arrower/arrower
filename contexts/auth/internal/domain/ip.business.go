package domain

import (
	"errors"
	"net"
)

var ErrResolveFailed = errors.New("resolving ip failed")

type IPResolver interface {
	ResolveIP(ip string) (ResolvedIP, error)
}

type ResolvedIP struct {
	Country     string
	CountryCode string
	Region      string
	City        string
	IP          net.IP
}
