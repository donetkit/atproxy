package main

import (
	"net"
	"os"
	"path/filepath"

	"net/http"
	_ "net/http/pprof"

	"github.com/reusee/atproxy"
	"github.com/reusee/pr"
	"github.com/reusee/starlarkutil"
	"go.starlark.net/starlark"
)

var globalWaitTree = pr.NewRootWaitTree()

func main() {

	go func() {
		ce(http.ListenAndServe("localhost:8000", nil))
	}()

	if len(os.Args) > 1 {
		fn, ok := commands[os.Args[1]]
		if !ok {
			pt("no such command\n")
			os.Exit(-1)
		}
		fn()
		return
	}

	var options []atproxy.ServerOption

	options = append(options, atproxy.WithDenyDirectPattern("github"))
	var socksAddr, httpAddr string

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
				"http_addr": starlarkutil.MakeFunc("http_addr", func(addr string) {
					httpAddr = addr
					pt("http addr %s\n", addr)
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

	socksLn, err := net.Listen("tcp", socksAddr)
	ce(err)
	httpLn, err := net.Listen("tcp", httpAddr)
	ce(err)

	server, err := atproxy.NewServer(
		socksLn.(*net.TCPListener),
		httpLn.(*net.TCPListener),
		options...,
	)
	ce(err)

	server.Serve(globalWaitTree.Ctx)

}
