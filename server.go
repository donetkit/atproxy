package atproxy

import (
	"context"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/reusee/atproxy/internal"
)

type Server struct {
	spec      ServerSpec
	ln        *net.TCPListener
	clientSem chan struct{}
	dialers   []Dialer
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
	var spec ServerSpec
	for _, option := range options {
		option(&spec)
	}

	// idle timeout
	if spec.idleTimeout == 0 {
		spec.idleTimeout = time.Minute * 47
	}

	// max clients
	if spec.maxClients > 0 {
		server.clientSem = make(chan struct{}, spec.maxClients)
	}

	// dial
	if spec.dialTimeout == 0 {
		spec.dialTimeout = time.Second * 16
	}
	if spec.dialer == nil {
		spec.dialer = &net.Dialer{
			Timeout: spec.dialTimeout,
		}
	}

	// direct dialer
	buf := new(strings.Builder)
	for i, pattern := range spec.denyDirectPatterns {
		if i > 0 {
			buf.WriteString("|")
		}
		buf.WriteString(pattern)
	}
	server.dialers = append(server.dialers, Dialer{
		DialContext: spec.dialer.DialContext,
		Name:        "direct",
		Deny:        regexp.MustCompile(buf.String()),
	})

	// upstream
	for _, upstream := range spec.upstreams {
		upstream := upstream
		if upstream.Network == "" {
			upstream.Network = "tcp"
		}
		server.dialers = append(server.dialers, Dialer{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				conn, err := spec.dialer.DialContext(ctx, network, upstream.Addr)
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
			},
			Name: upstream.Addr,
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

		if s.clientSem != nil {
			s.clientSem <- struct{}{}
		}
		go func() {
			if s.clientSem != nil {
				defer func() {
					<-s.clientSem
				}()
			}

			s.handleConn(ctx, conn)
		}()

	}

}
