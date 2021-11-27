package main

import (
	"context"
	"net"
	"os"
	"path/filepath"

	"net/http"
	_ "net/http/pprof"

	"github.com/reusee/atproxy"
	"github.com/reusee/starlarkutil"
	"go.starlark.net/starlark"
)

func main() {

	go func() {
		ce(http.ListenAndServe("localhost:8000", nil))
	}()

	var options []atproxy.ServerOption

	options = append(options, atproxy.WithDenyDirectPattern("github"))
	var socksAddr string

	// load config file
	exePath, err := os.Executable()
	ce(err)
	exeDir := filepath.Dir(exePath)
	configFileName := "atproxy.py"
	content, err := os.ReadFile(filepath.Join(exeDir, configFileName))
	if err == nil {
		pt("load %s\n", configFileName)
		_, err := starlark.ExecFile(
			new(starlark.Thread),
			configFileName,
			content,
			starlark.StringDict{
				"socks_addr": starlarkutil.MakeFunc("socks_addr", func(addr string) {
					socksAddr = addr
					pt("socks addr %s\n", addr)
				}),
				"upstream": starlarkutil.MakeFunc("upstream", func(addr string) {
					options = append(options, atproxy.WithUpstream(atproxy.Upstream{
						Addr: addr,
					}))
					pt("upstream %s\n", addr)
				}),
			},
		)
		ce(err)
	}

	ln, err := net.Listen("tcp", socksAddr)
	ce(err)

	server, err := atproxy.NewServer(
		ln.(*net.TCPListener),
		options...,
	)
	ce(err)

	server.Serve(context.Background())

}
