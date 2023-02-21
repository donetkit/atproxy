package atproxy

import (
	"errors"
	"fmt"

	"github.com/reusee/dscope"
	"github.com/reusee/e5"
)

var (
	ce = e5.Check
	he = e5.Handle
	we = e5.Wrap

	is = errors.Is
	pt = fmt.Printf
)

type (
	Scope = dscope.Scope
)
