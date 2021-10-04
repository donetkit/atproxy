package atproxy

import (
	"context"
	"net"
	"time"

	"golang.org/x/net/proxy"
)

type Server struct {
	ln          *net.TCPListener
	upstreams   []Upstream
	dialTimeout time.Duration
	dialer      *net.Dialer
	maxClients  int
	clientSem   chan struct{}
	idleTimeout time.Duration

	dialContexts []DialContext
}

type DialContext = func(ctx context.Context, addr, network string) (net.Conn, error)

func NewServer(
	ln *net.TCPListener,
	options ...ServerOption,
) (
	server *Server,
	err error,
) {
	defer he(&err)

	server = &Server{
		ln: ln,
	}
	for _, option := range options {
		option(server)
	}

	// idle timeout
	if server.idleTimeout == 0 {
		server.idleTimeout = time.Minute * 47
	}

	// max clients
	if server.maxClients > 0 {
		server.clientSem = make(chan struct{}, server.maxClients)
	}

	// dial
	if server.dialTimeout == 0 {
		server.dialTimeout = time.Second * 16
	}
	if server.dialer == nil {
		server.dialer = &net.Dialer{
			Timeout: server.dialTimeout,
		}
	}

	// direct
	server.dialContexts = append(server.dialContexts, server.dialer.DialContext)

	// upstream
	for _, upstream := range server.upstreams {
		upstream := upstream
		if upstream.Network == "" {
			upstream.Network = "tcp"
		}
		auth := &proxy.Auth{
			User:     upstream.User,
			Password: upstream.Password,
		}
		proxyDialer, err := proxy.SOCKS5(upstream.Network, upstream.Addr, auth, server.dialer)
		if err != nil {
			return nil, err
		}
		server.dialContexts = append(server.dialContexts, func(ctx context.Context, network, addr string) (net.Conn, error) {
			return proxyDialer.(proxy.ContextDialer).DialContext(ctx, network, addr)
		})
	}

	return
}

func (s *Server) Serve(
	ctx context.Context,
) (
	err error,
) {
	defer he(&err)

	for {
		conn, err := s.ln.AcceptTCP()
		ce(err)

		if s.maxClients > 0 {
			s.clientSem <- struct{}{}
		}
		go func() {
			if s.maxClients > 0 {
				defer func() {
					<-s.clientSem
				}()
			}

			s.handleConn(ctx, conn)
		}()

	}

	return
}
