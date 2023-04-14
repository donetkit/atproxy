package atproxy

import "github.com/reusee/pr2"

type BytesPoolCapacity int

func (Global) BytesPoolCapacity() BytesPoolCapacity {
	return 512
}

type BytesPoolBufferSize int

func (Global) BytesPoolBufferSize() BytesPoolBufferSize {
	return 4 * 1024
}

type BytesPool struct {
	*pr2.Pool[*[]byte]
}

func (Global) BytesPool(
	capacity BytesPoolCapacity,
	size BytesPoolBufferSize,
) BytesPool {
	s := int(size)
	return BytesPool{
		Pool: pr2.NewPool(
			uint32(capacity),
			func(_ pr2.PoolPutFunc) *[]byte {
				bs := make([]byte, s)
				return &bs
			},
		),
	}
}
