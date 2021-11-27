package atproxy

import (
	"net"
	"time"
)

type ServerOption func(*Server)

func WithUpstream(upstream Upstream) ServerOption {
	return func(s *Server) {
		s.upstreams = append(s.upstreams, upstream)
	}
}

func WithDialer(dialer *net.Dialer) ServerOption {
	return func(s *Server) {
		s.dialer = dialer
	}
}

func WithDialTimeout(d time.Duration) ServerOption {
	return func(s *Server) {
		s.dialTimeout = d
	}
}

func WithIdleTimeout(d time.Duration) ServerOption {
	return func(s *Server) {
		s.idleTimeout = d
	}
}

func WithMaxClients(n int) ServerOption {
	return func(s *Server) {
		s.maxClients = n
	}
}

func WithDenyDirectPattern(pattern string) ServerOption {
	return func(s *Server) {
		s.denyDirectPatterns = append(s.denyDirectPatterns, pattern)
	}
}
