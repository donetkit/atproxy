package main

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

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

	type serverSpec struct {
		socksAddr string
		httpAddr  string
		options   []atproxy.ServerOption
	}
	var serverSpecs []serverSpec

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

				"tailscale_addr": starlarkutil.MakeFunc("tailscale_addr", func() string {
				get:
					ifaces, err := net.Interfaces()
					ce(err)
					for _, iface := range ifaces {
						if !strings.HasPrefix(iface.Name, "tailscale") {
							continue
						}
						addrs, err := iface.Addrs()
						ce(err)
						for _, addr := range addrs {
							ip := addr.(*net.IPNet).IP.To4()
							if len(ip) != 4 {
								continue
							}
							return ip.String()
						}
					}
					time.Sleep(time.Second)
					goto get
				}),

				"server": starlarkutil.MakeFunc("server", func(
					socksAddr string,
					httpAddr string,
					upstreams ...string,
				) {
					spec := serverSpec{
						socksAddr: socksAddr,
						httpAddr:  httpAddr,
					}
					for _, upstream := range upstreams {
						spec.options = append(spec.options, atproxy.WithUpstream(atproxy.Upstream{
							Addr: upstream,
						}))
					}
					serverSpecs = append(serverSpecs, spec)
				}),

				"no_direct": starlarkutil.MakeFunc("no_direct", func(pattern string) {
					options = append(options, atproxy.WithDenyDirectPattern(pattern))
				}),

				//
			},
		)
		ce(err)
	}

	for _, spec := range serverSpecs {

		socksLn, err := net.Listen("tcp", spec.socksAddr)
		ce(err)
		httpLn, err := net.Listen("tcp", spec.httpAddr)
		ce(err)

		spec.options = append(spec.options, options...)

		server, err := atproxy.NewServer(
			socksLn.(*net.TCPListener),
			httpLn.(*net.TCPListener),
			spec.options...,
		)
		ce(err)

		go server.Serve(globalWaitTree.Ctx)

	}

	select {}

}
