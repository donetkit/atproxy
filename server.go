package atproxy

import (
	"context"
	"net"
	"time"

	"github.com/reusee/atproxy/internal"
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
		server.dialContexts = append(server.dialContexts, func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := server.dialer.DialContext(ctx, network, upstream.Addr)
			if err != nil {
				return nil, err
			}
			// handshake
			//TODO auth
			if err := internal.Socks5ClientHandshake(conn, addr); err != nil {
				conn.Close()
				return nil, err
			}
			return conn, err
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

}
