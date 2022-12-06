package atproxy

import "github.com/reusee/pr"

type BytesPoolCapacity int

func (Def) BytesPoolCapacity() BytesPoolCapacity {
	return 512
}

type BytesPoolBufferSize int

func (Def) BytesPoolBufferSize() BytesPoolBufferSize {
	return 4 * 1024
}

type BytesPool struct {
	*pr.Pool[*[]byte]
}

func (Def) BytesPool(
	capacity BytesPoolCapacity,
	size BytesPoolBufferSize,
) BytesPool {
	s := int(size)
	return BytesPool{
		Pool: pr.NewPool(
			int32(capacity),
			func() *[]byte {
				bs := make([]byte, s)
				return &bs
			},
		),
	}
}
