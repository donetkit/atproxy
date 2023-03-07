package atproxy

import (
	"context"
	"net"
)

type TailscaleDial func(ctx context.Context, network, addr string) (net.Conn, error)
