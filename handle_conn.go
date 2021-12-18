package atproxy

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

func (s *Server) handleConn(
	parentCtx context.Context,
	conn *net.TCPConn,
	hostPort string,
) {

	numDialers := len(s.dialers)

	// chans and contexts
	type Ctx struct {
		Context context.Context
		Cancel  func()
	}
	ctxs := make([]*Ctx, 0, numDialers)
	type OutboundPacket struct {
		Data []byte
		Put  func() bool
	}
	outbounds := make([]chan OutboundPacket, 0, numDialers)
	outboundsClosed := make([]bool, 0, numDialers)
	for i := 0; i < numDialers; i++ {
		outbounds = append(outbounds, make(chan OutboundPacket, 512))
		outboundsClosed = append(outboundsClosed, false)
		ctx, cancel := context.WithCancel(parentCtx)
		ctxs = append(ctxs, &Ctx{
			Context: ctx,
			Cancel:  cancel,
		})
	}

	chosen := int32(-1)
	wg := new(sync.WaitGroup)

	var once1, once2 sync.Once
	var connBytesRead int64
	var connBytesWritten int64
	closeConnRead := func() {
		once1.Do(func() {
			conn.CloseRead()
		})
	}
	closeConnWrite := func() {
		once2.Do(func() {
			conn.CloseWrite()
			for _, ctx := range ctxs {
				ctx.Cancel()
			}
		})
	}

	// read local conn
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			v, put, incRef := bytesPool.GetRC()
			buffer := v.([]byte)
			deadline := time.Now().Add(s.idleTimeout)
			if err := conn.SetReadDeadline(deadline); err != nil {
				break
			}
			n, err := conn.Read(buffer)
			if n > 0 {
				atomic.AddInt64(&connBytesRead, int64(n))
				buffer = buffer[:n]
				if i := atomic.LoadInt32(&chosen); i != -1 {
					// send to the chosen one
					outbounds[i] <- OutboundPacket{
						Data: buffer,
						Put:  put,
					}
					for n, outbound := range outbounds {
						if n == int(i) {
							continue
						}
						if !outboundsClosed[n] {
							close(outbound)
							outboundsClosed[n] = true
							ctxs[n].Cancel()
						}
					}
				} else {
					// send to all
					for _, ch := range outbounds {
						incRef()
						ch <- OutboundPacket{
							Data: buffer,
							Put:  put,
						}
					}
				}
			}
			if err != nil {
				break
			}
		}
		closeConnRead()
		// close write to chans
		for i, ch := range outbounds {
			if !outboundsClosed[i] {
				close(ch)
			}
		}
	}()

	closeReadCounter := int64(numDialers)
	closeRead := func(n int64) {
		if n := atomic.AddInt64(&closeReadCounter, n); n <= 0 {
			closeConnRead()
		}
	}
	closeWriteCounter := int64(numDialers)
	closeWrite := func(n int64) {
		if n := atomic.AddInt64(&closeWriteCounter, n); n <= 0 {
			closeConnWrite()
		}
	}

	for i, dialer := range s.dialers {
		i := i
		dialer := dialer
		wg.Add(1)
		go func() {
			defer wg.Done()

			if dialer.Deny != nil {
				if dialer.Deny.MatchString(hostPort) {
					return
				}
			}

			// dial
			var upstream *net.TCPConn
			c, err := dialer.DialContext(ctxs[i].Context, "tcp", hostPort)
			if err == nil {
				upstream = c.(*net.TCPConn)
			}

			subWg := new(sync.WaitGroup)
			var once1, once2 sync.Once
			var upstreamBytesWritten int64
			var upstreamBytesRead int64
			closeUpstreamRead := func() {
				once1.Do(func() {
					if upstream != nil {
						upstream.CloseRead()
						upstream.SetWriteDeadline(time.Now().Add(time.Second * 30))
					}
				})
			}
			closeUpstreamWrite := func() {
				once2.Do(func() {
					if upstream != nil {
						upstream.CloseWrite()
						deadline := time.Now().Add(time.Second * 30)
						upstream.SetReadDeadline(deadline)
					}
				})
			}

			// chan -> remote
			wg.Add(1)
			subWg.Add(1)
			go func() {
				defer func() {
					wg.Done()
					subWg.Done()
				}()
				if upstream != nil {
					for {
						outboundPacket, ok := <-outbounds[i]
						if !ok {
							break
						}
						n := atomic.LoadInt32(&chosen)
						if n != -1 && n != int32(i) {
							outboundPacket.Put()
							break
						}
						if err := upstream.SetWriteDeadline(time.Now().Add(time.Minute * 8)); err != nil {
							outboundPacket.Put()
							break
						}
						_, err := upstream.Write(outboundPacket.Data)
						if err != nil {
							outboundPacket.Put()
							break
						}
						atomic.AddInt64(&upstreamBytesWritten, int64(len(outboundPacket.Data)))
						outboundPacket.Put()
					}
					closeUpstreamWrite()
				}
				if n := int(atomic.LoadInt32(&chosen)); n == i {
					// chosen
					closeRead(-int64(numDialers))
				} else {
					closeRead(-1)
				}
				for outboundPacket := range outbounds[i] {
					outboundPacket.Put()
				}
			}()

			// local <- remote
			wg.Add(1)
			subWg.Add(1)
			go func() {
				defer func() {
					wg.Done()
					subWg.Done()
				}()
				defer func() {
					if n := int(atomic.LoadInt32(&chosen)); n == i {
						// chosen
						closeWrite(-int64(numDialers))
					} else {
						closeWrite(-1)
					}
				}()
				if upstream == nil {
					return
				}
				defer closeUpstreamRead()

				v, put := bytesPool.Get()
				defer put()
				buffer := v.([]byte)
				selected := false
				for {

					deadline := time.Now().Add(s.idleTimeout)
					if err := upstream.SetReadDeadline(deadline); err != nil {
						break
					}
					n, err := upstream.Read(buffer)

					if n > 0 {
						atomic.AddInt64(&upstreamBytesRead, int64(n))

						if !selected {
							if atomic.CompareAndSwapInt32(&chosen, -1, int32(i)) {
								selected = true
							} else {
								break // not selected
							}
						}

						if err := conn.SetWriteDeadline(time.Now().Add(time.Minute * 8)); err != nil {
							break
						}
						_, err := conn.Write(buffer[:n])
						if err != nil {
							break
						}
						atomic.AddInt64(&connBytesWritten, int64(n))

					}

					if err != nil {
						break
					}

				}
			}()

			subWg.Wait()
			if upstream != nil {
				upstream.Close()
			}

		}()

	}

	wg.Wait()

}
