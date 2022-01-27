package atproxy

import (
	"context"
	"net"
	"regexp"
	"strings"

	"github.com/reusee/atproxy/internal"
)

type Upstream struct {
	DialContext DialContext
	Network     string
	Addr        string
	User        string
	Password    string
}

type Upstreams []*Upstream

func (_ Def) Upstreams() Upstreams {
	return nil
}

func (_ Def) UpstreamDialers(
	upstreams Upstreams,
	noUpstreamPatterns NoUpstreamPatterns,
) (dialers Dialers) {

	defaultDialContext := new(net.Dialer).DialContext

	buf := new(strings.Builder)
	for i, pattern := range noUpstreamPatterns {
		if i > 0 {
			buf.WriteString("|")
		}
		buf.WriteString("(")
		buf.WriteString(pattern)
		buf.WriteString(")")
	}
	noUpstreamRe := regexp.MustCompile(buf.String())

	for _, upstream := range upstreams {
		upstream := upstream

		dial := defaultDialContext
		if upstream.DialContext != nil {
			dial = upstream.DialContext
		}

		dialers = append(dialers, &Dialer{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				conn, err := dial(ctx, network, upstream.Addr)
				if err != nil {
					return nil, err
				}
				// handshake
				//TODO auth
				if err := internal.Socks5ClientHandshake(conn, addr); err != nil {
					conn.Close()
					return nil, err
				}
				return conn, err
			},
			Name: upstream.Addr,
			Deny: noUpstreamRe,
		})

	}

	return
}

type NoUpstreamPatterns []string

func (_ Def) NoUpstreamPatterns() NoUpstreamPatterns {
	return nil
}
