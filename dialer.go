package atproxy

import (
	"context"
	"net"
	"regexp"
	"strings"
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

func (_ Def) DirectDialer(
	noDirectPatterns NoDirectPatterns,
) Dialers {

	buf := new(strings.Builder)
	for i, pattern := range noDirectPatterns {
		if i > 0 {
			buf.WriteString("|")
		}
		buf.WriteString("(")
		buf.WriteString(pattern)
		buf.WriteString(")")
	}

	return Dialers{
		{
			DialContext: new(net.Dialer).DialContext,
			Name:        "direct",
			Deny:        regexp.MustCompile(buf.String()),
		},
	}
}

type NoDirectPatterns []string

func (_ Def) NoDirectPatterns() NoDirectPatterns {
	return nil
}
