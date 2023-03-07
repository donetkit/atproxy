package atproxy

import (
	"context"
	"net"
	"os"

	"tailscale.com/tsnet"
)

func (Def) TailscaleDial() TailscaleDial {
	authKey := os.Getenv("TS_AUTHKEY")
	if authKey == "" {
		return nil
	}

	hostname, err := os.Hostname()
	ce(err)
	return &tsnet.Server{
		Hostname: hostname,
		Logf: func(format string, args ...any) {
			// do nothing
		},
		Ephemeral: true,
	}.Dial
}
