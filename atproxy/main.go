package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/reusee/atproxy"
	"github.com/reusee/dscope"
	"github.com/reusee/starlarkutil"
	"go.starlark.net/starlark"
)

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

	var onExitFuncs []func()
	onExit := func(fn func()) {
		onExitFuncs = append(onExitFuncs, fn)
	}

	type serverSpec struct {
		Socks     string
		HTTP      string
		Upstreams map[string]string
		NoDirect  atproxy.NoDirect
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

				"server_spec": starlarkutil.MakeFunc("server_spec", func(spec serverSpec) {
					pt("server: %+v\n", spec)
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

				"profile": starlarkutil.MakeFunc("profile", func(path string) {
					pt("write cpu profile to %s\n", path)
					out, err := os.Create(path)
					ce(err)
					ce(pprof.StartCPUProfile(out))
					onExit(func() {
						pprof.StopCPUProfile()
						ce(out.Close())
					})
				}),

				//
			},
		)
		ce(err)
	}

	global := atproxy.NewGlobal()
	newServer := dscope.Get[atproxy.NewServer](global.Scope)

	for _, spec := range serverSpecs {
		spec := spec
		go func() {

			socksLn, err := net.Listen("tcp", spec.Socks)
			ce(err)
			httpLn, err := net.Listen("tcp", spec.HTTP)
			ce(err)

			server := newServer(defs, []any{
				&spec.NoDirect,
				&noDirectPatterns,
				&noUpstreamPattern,

				func() (upstreams atproxy.Upstreams) {
					for name, addr := range spec.Upstreams {
						upstreams = append(upstreams, &atproxy.Upstream{
							Network:     "tcp",
							Name:        name,
							Addr:        addr,
							IsTailscale: strings.HasPrefix(name, "ts:"),
						})
					}
					return
				},

				func() atproxy.OnSelected {
					return func(dialer *atproxy.Dialer, hostPort string) {
						pt("%8s: %s\n", dialer.Name, hostPort)
					}
				},

				//
			})

			serve := dscope.Get[atproxy.Serve](server.Scope)
			ctx := context.Background()
			ce(serve(
				ctx,
				socksLn.(*net.TCPListener),
				httpLn.(*net.TCPListener),
			))

		}()

	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, os.Interrupt)
	<-ch
	for _, fn := range onExitFuncs {
		fn()
	}

}
