package atproxy

import "github.com/reusee/dscope"

type Def struct{}

func NewServerScope(defs ...any) Scope {
	defs = append(defs, dscope.Methods(new(Def))...)
	return dscope.New(defs...)
}
