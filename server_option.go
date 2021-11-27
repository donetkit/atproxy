package atproxy

import (
	"net"
	"time"
)

type ServerSpec struct {
	upstreams          []Upstream
	dialer             *net.Dialer
	dialTimeout        time.Duration
	idleTimeout        time.Duration
	maxClients         int
	denyDirectPatterns []string
}

type ServerOption func(*ServerSpec)

func WithUpstream(upstream Upstream) ServerOption {
	return func(s *ServerSpec) {
		s.upstreams = append(s.upstreams, upstream)
	}
}

func WithDialer(dialer *net.Dialer) ServerOption {
	return func(s *ServerSpec) {
		s.dialer = dialer
	}
}

func WithDialTimeout(d time.Duration) ServerOption {
	return func(s *ServerSpec) {
		s.dialTimeout = d
	}
}

func WithIdleTimeout(d time.Duration) ServerOption {
	return func(s *ServerSpec) {
		s.idleTimeout = d
	}
}

func WithMaxClients(n int) ServerOption {
	return func(s *ServerSpec) {
		s.maxClients = n
	}
}

func WithDenyDirectPattern(pattern string) ServerOption {
	return func(s *ServerSpec) {
		s.denyDirectPatterns = append(s.denyDirectPatterns, pattern)
	}
}
