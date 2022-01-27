package atproxy

import (
	"context"
	"io"
	"net/http"
	"sync/atomic"
)

type HandleRequest func(
	parentCtx context.Context,
	req *http.Request,
	w http.ResponseWriter,
)

func (_ Def) HandleRequest(
	transports []*http.Transport,
	bytesPool BytesPool,
) HandleRequest {
	return func(
		parentCtx context.Context,
		req *http.Request,
		w http.ResponseWriter,
	) {

		ch := make(chan *http.Response, 1)
		inflight := int64(len(transports))

		for _, transport := range transports {
			transport := transport
			go func() {
				defer func() {
					if n := atomic.AddInt64(&inflight, -1); n == 0 {
						close(ch)
					}
				}()
				resp, err := transport.RoundTrip(req)
				if err != nil {
					return
				}
				select {
				case ch <- resp:
				default:
					resp.Body.Close()
				}
			}()
		}

		var resp *http.Response
		resp, ok := <-ch
		if !ok {
			return
		}

		defer resp.Body.Close()
		header := w.Header()
		for name, h := range resp.Header {
			for _, value := range h {
				header.Add(name, value)
			}
		}
		w.WriteHeader(resp.StatusCode)

		v, put := bytesPool.Get()
		defer put()
		buf := v.([]byte)
		io.CopyBuffer(w, resp.Body, buf)

	}
}
