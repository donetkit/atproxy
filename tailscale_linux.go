package atproxy

import (
	"os"

	"tailscale.com/tsnet"
)

func (Def) TailscaleDial() TailscaleDial {
	hostname, err := os.Hostname()
	ce(err)
	return (&tsnet.Server{
		Hostname: hostname,
		Logf: func(format string, args ...any) {
			// do nothing
		},
	}).Dial
}
