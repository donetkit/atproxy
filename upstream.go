package atproxy

type Upstream struct {
	Name        string
	DialContext DialContext
	Network     string
	Addr        string
	User        string
	Password    string
}

type Upstreams []*Upstream

func (_ Def) Upstreams() Upstreams {
	return nil
}
