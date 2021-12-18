package atproxy

import (
	"context"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/reusee/atproxy/internal"
)

type Server struct {
	socksLn     *net.TCPListener
	httpLn      *net.TCPListener
	upstreams   []Upstream
	dialTimeout time.Duration
	dialer      *net.Dialer
	maxClients  int
	clientSem   chan struct{}
	idleTimeout time.Duration

	dialers        []Dialer
	httpTransports []*http.Transport

	denyDirectPatterns []string
}

type DialContext = func(ctx context.Context, addr, network string) (net.Conn, error)

func NewServer(
	socksLn *net.TCPListener,
	httpLn *net.TCPListener,
	options ...ServerOption,
) (
	server *Server,
	err error,
) {
	defer he(&err)

	server = &Server{
		socksLn: socksLn,
		httpLn:  httpLn,
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

	// direct dialer
	buf := new(strings.Builder)
	for i, pattern := range server.denyDirectPatterns {
		if i > 0 {
			buf.WriteString("|")
		}
		buf.WriteString(pattern)
	}
	server.dialers = append(server.dialers, Dialer{
		DialContext: server.dialer.DialContext,
		Name:        "direct",
		Deny:        regexp.MustCompile(buf.String()),
	})

	// upstream
	for _, upstream := range server.upstreams {
		upstream := upstream
		if upstream.Network == "" {
			upstream.Network = "tcp"
		}
		server.dialers = append(server.dialers, Dialer{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
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
			},
			Name: upstream.Addr,
		})
	}

	// http transports
	for _, dial := range server.dialers {
		server.httpTransports = append(server.httpTransports, &http.Transport{
			DialContext: dial.DialContext,
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

	// http
	go func() {
		server := &http.Server{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if s.clientSem != nil {
					s.clientSem <- struct{}{}
					defer func() {
						<-s.clientSem
					}()
				}

				if req.Method != http.MethodConnect {
					s.handleRequest(ctx, req, w)
					return
				}

				hostPort := req.Host
				hijacker, ok := w.(http.Hijacker)
				if !ok {
					http.Error(w, "not supported", http.StatusInternalServerError)
					return
				}
				c, _, err := hijacker.Hijack()
				if err != nil {
					http.Error(w, err.Error(), http.StatusServiceUnavailable)
					return
				}
				conn := c.(*net.TCPConn)
				defer conn.Close()

				_, err = conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
				ce(err)

				s.handleConn(ctx, conn, hostPort)
			}),
		}
		server.Serve(s.httpLn)
	}()

	// socks
	for {
		conn, err := s.socksLn.AcceptTCP()
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
			defer conn.Close()

			hostPort, err := internal.Socks5ServerHandshake(conn)
			if err != nil {
				return
			}

			s.handleConn(ctx, conn, hostPort)
		}()

	}

}
