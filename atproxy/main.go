package main

import (
	"context"
	"net"
	"os"
	"path/filepath"

	"github.com/reusee/atproxy"
)

func main() {

	if len(os.Args) <= 1 {
		pt("usage: %s [listen addr] [upstream addr]...\n",
			filepath.Base(os.Args[0]),
		)
		return
	}

	addr := os.Args[1]
	pt("listen: %s\n", addr)
	ln, err := net.Listen("tcp", addr)
	ce(err)

	var options []atproxy.ServerOption
	for _, arg := range os.Args[2:] {
		pt("upstream: %s\n", arg)
		options = append(options, atproxy.WithUpstream(atproxy.Upstream{
			Addr: arg,
		}))
	}

	server, err := atproxy.NewServer(
		ln.(*net.TCPListener),
		options...,
	)
	ce(err)

	server.Serve(context.Background())

}
