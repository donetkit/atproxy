package atproxy

import (
	"context"
	"net"
	"regexp"
	"time"

	"github.com/reusee/dscope"
)

type Dialer struct {
	DialContext DialContext
	Name        string
	Deny        *regexp.Regexp
}

type DialContext = func(ctx context.Context, addr, network string) (net.Conn, error)

type DialTimeout time.Duration

func (_ Def) DialTimeout() DialTimeout {
	return DialTimeout(time.Second * 32)
}

type Dialers []*Dialer

var _ dscope.Reducer = Dialers{}

func (_ Dialers) IsReducer() {}

func (_ Def) DirectDialer() Dialers {
	return Dialers{
		{
			DialContext: new(net.Dialer).DialContext,
			Name:        "direct",
		},
	}
}
