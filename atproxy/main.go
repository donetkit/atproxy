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

	type serverSpec struct {
		socksAddr     string
		httpAddr      string
		upstreamAddrs []string
	}
	var serverSpecs []serverSpec

	var noDirectPatterns atproxy.NoDirectPatterns
	var noUpstreamPattern atproxy.NoUpstreamPatterns
	var defs []any

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

				"tailscale_addr": starlarkutil.MakeFunc("tailscale_addr", func() (ret string) {
					pt("get tailscale address\n")
					defer func() {
						pt("tailscale address is %v\n", ret)
					}()
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
					pt("server, socks %v, http %v\n", spec.socksAddr, spec.httpAddr)
					for _, upstream := range upstreams {
						pt("\tupstream %v\n", upstream)
						spec.upstreamAddrs = append(spec.upstreamAddrs, upstream)
					}
					serverSpecs = append(serverSpecs, spec)
				}),

				"no_direct": starlarkutil.MakeFunc("no_direct", func(pattern string) {
					pt("rule: no direct %q\n", pattern)
					noDirectPatterns = append(noDirectPatterns, pattern)
				}),

				"no_upstream": starlarkutil.MakeFunc("no_upstream", func(pattern string) {
					pt("rule: no upstream %q\n", pattern)
					noUpstreamPattern = append(noUpstreamPattern, pattern)
				}),

				"pool_capacity": starlarkutil.MakeFunc("pool_capacity", func(capacity atproxy.BytesPoolCapacity) {
					defs = append(defs, &capacity)
				}),

				"pool_buffer_size": starlarkutil.MakeFunc("pool_buffer_size", func(size atproxy.BytesPoolBufferSize) {
					defs = append(defs, &size)
				}),

				//
			},
		)
		ce(err)
	}

	for _, spec := range serverSpecs {
		spec := spec
		go func() {

			socksLn, err := net.Listen("tcp", spec.socksAddr)
			ce(err)
			httpLn, err := net.Listen("tcp", spec.httpAddr)
			ce(err)

			atproxy.NewServerScope().Fork(defs...).Fork(

				&noDirectPatterns,
				&noUpstreamPattern,

				func() (upstreams atproxy.Upstreams) {
					for _, addr := range spec.upstreamAddrs {
						upstreams = append(upstreams, &atproxy.Upstream{
							Network: "tcp",
							Addr:    addr,
						})
					}
					return
				},

				//
			).Call(func(
				serve atproxy.Serve,
			) {
				ce(serve(
					globalWaitTree.Ctx,
					socksLn.(*net.TCPListener),
					httpLn.(*net.TCPListener),
				))
			})

		}()

	}

	select {}

}
