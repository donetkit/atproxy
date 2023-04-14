package atproxy

import (
	"context"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/reusee/atproxy/internal"
	"github.com/reusee/dscope"
)

type Dialer struct {
	DialContext DialContext
	Name        string
	Addr        string
	Deny        *regexp.Regexp
}

type DialContext = func(ctx context.Context, addr, network string) (net.Conn, error)

type DialTimeout time.Duration

func (Server) DialTimeout() DialTimeout {
	return DialTimeout(time.Second * 32)
}

type Dialers []*Dialer

var _ dscope.Reducer = Dialers{}

func (Dialers) IsReducer() {}

func (Server) DirectDialer(
	noDirectPatterns NoDirectPatterns,
	noDirect NoDirect,
) Dialers {

	if noDirect {
		return nil
	}

	var deny *regexp.Regexp
	if len(noDirectPatterns) > 0 {
		buf := new(strings.Builder)
		for i, pattern := range noDirectPatterns {
			if i > 0 {
				buf.WriteString("|")
			}
			buf.WriteString("(")
			buf.WriteString(pattern)
			buf.WriteString(")")
		}
		deny = regexp.MustCompile(buf.String())
	}

	return Dialers{
		{
			DialContext: new(net.Dialer).DialContext,
			Name:        "direct",
			Deny:        deny,
		},
	}
}

type NoDirect bool

func (Server) NoDirect() NoDirect {
	return false
}

func (Server) UpstreamDialers(
	upstreams Upstreams,
	noUpstreamPatterns NoUpstreamPatterns,
	tailscaleDial TailscaleDial,
) (dialers Dialers) {

	defaultDialContext := new(net.Dialer).DialContext

	var deny *regexp.Regexp
	if len(noUpstreamPatterns) > 0 {
		buf := new(strings.Builder)
		for i, pattern := range noUpstreamPatterns {
			if i > 0 {
				buf.WriteString("|")
			}
			buf.WriteString("(")
			buf.WriteString(pattern)
			buf.WriteString(")")
		}
		deny = regexp.MustCompile(buf.String())
	}

	for _, upstream := range upstreams {
		upstream := upstream

		dial := defaultDialContext
		if upstream.DialContext != nil {
			dial = upstream.DialContext
		}
		if upstream.IsTailscale && tailscaleDial != nil {
			dial = tailscaleDial
			pt("tailscale upstream: %s\n", upstream.Name)
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
			Name: upstream.Name,
			Addr: upstream.Addr,
			Deny: deny,
		})

	}

	return
}

type NoDirectPatterns []string

func (Server) NoDirectPatterns() NoDirectPatterns {
	return nil
}

type NoUpstreamPatterns []string

func (Server) NoUpstreamPatterns() NoUpstreamPatterns {
	return nil
}
