package atproxy

import "github.com/reusee/dscope"

type Global struct{}

var GlobalScope = dscope.New(dscope.Methods(new(Global))...)

type Def struct{}

type NewServerScope func(defs ...any) Scope

func (Global) NewServerScope() NewServerScope {
	return func(defs ...any) Scope {
		defs = append(defs, dscope.Methods(new(Def))...)
		return GlobalScope.Fork(defs...)
	}
}
