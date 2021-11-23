package atproxy

import (
	"context"
	"net"
	"regexp"
)

type Dialer struct {
	DialContext func(ctx context.Context, network, addr string) (net.Conn, error)
	Name        string
	Deny        *regexp.Regexp
}
