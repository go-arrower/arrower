package domain

import "net"

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
