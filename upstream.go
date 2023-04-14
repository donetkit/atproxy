package atproxy

type Upstream struct {
	Name        string
	DialContext DialContext
	Network     string
	Addr        string
	User        string
	Password    string
	IsTailscale bool
}

type Upstreams []*Upstream

func (Server) Upstreams() Upstreams {
	return nil
}
