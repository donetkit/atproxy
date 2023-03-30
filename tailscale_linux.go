package atproxy

import (
	"os"
	"path/filepath"
	"sync"

	"tailscale.com/tsnet"
)

func (Def) TsServer() *tsnet.Server {
	exePath, err := os.Executable()
	ce(err)
	exeDir := filepath.Dir(exePath)
	hostname, err := os.Hostname()
	ce(err)
	dir := filepath.Join(exeDir, "atproxy-tsnet")
	ce(os.MkdirAll(dir, 0777))

	tsServer = &tsnet.Server{
		Dir:      dir,
		Hostname: hostname,
		Logf: func(format string, args ...any) {
			// do nothing
		},
	}
	ce(tsServer.Start())

	return tsServer
}

func (Def) TailscaleDial(
	tsServer *tsnet.Server,
) TailscaleDial {
	return tsServer.Dial
}
