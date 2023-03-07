package atproxy

import (
	"os"
	"path/filepath"

	"tailscale.com/tsnet"
)

func (Def) TailscaleDial() TailscaleDial {
	exePath, err := os.Executable()
	ce(err)
	exeDir := filepath.Dir(exePath)
	hostname, err := os.Hostname()
	ce(err)
	dir := filepath.Join(exeDir, "atproxy-tsnet")
	ce(os.MkdirAll(dir, 0777))

	server := &tsnet.Server{
		Dir:      dir,
		Hostname: hostname,
		Logf: func(format string, args ...any) {
			// do nothing
		},
	}
	ce(server.Start())

	return server.Dial
}
