package atproxy

import (
	"context"
	"net"

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

func (_ Def) UpstreamDialers(
	upstreams Upstreams,
) (dialers Dialers) {

	defaultDialContext := new(net.Dialer).DialContext

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
		})

	}

	return
}
