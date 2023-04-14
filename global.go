package atproxy

import "github.com/reusee/dscope"

type Global struct {
	Scope Scope
}

func NewGlobal(defs ...any) *Global {
	global := new(Global)
	defs = append(defs, dscope.Methods(global)...)
	global.Scope = dscope.New(defs...)
	return global
}
