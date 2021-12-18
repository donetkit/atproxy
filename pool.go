package atproxy

import "github.com/reusee/pr"

var bytesPool = pr.NewPool(
	1024,
	func() any {
		return make([]byte, 32*1024)
	},
)
