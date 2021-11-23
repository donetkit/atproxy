package atproxy

import (
	"errors"
	"fmt"

	"github.com/reusee/e4"
)

var (
	ce = e4.Check
	he = e4.Handle
	we = e4.Wrap

	is = errors.Is
	pt = fmt.Printf
)
