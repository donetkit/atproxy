package atproxy

import "github.com/reusee/pr"

type BytesPoolCapacity int

func (_ Def) BytesPoolCapacity() BytesPoolCapacity {
	return 512
}

type BytesPoolBufferSize int

func (_ Def) BytesPoolBufferSize() BytesPoolBufferSize {
	return 4 * 1024
}

type BytesPool struct {
	*pr.Pool
}

func (_ Def) BytesPool(
	capacity BytesPoolCapacity,
	size BytesPoolBufferSize,
) BytesPool {
	s := int(size)
	return BytesPool{
		Pool: pr.NewPool(
			int32(capacity),
			func() any {
				return make([]byte, s)
			},
		),
	}
}
