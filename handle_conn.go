package atproxy

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/reusee/pr2"
)

type IdleTimeout time.Duration

func (Def) IdleTimeout() IdleTimeout {
	return IdleTimeout(time.Minute * 5)
}

type OutboundPacket struct {
	Data []byte
	Put  func() bool
}

type HandleConn func(
	parentCtx context.Context,
	conn net.Conn,
	hostPort string,
)

func (Def) HandleConn(
	dialers Dialers,
	_idleTimeout IdleTimeout,
	bytesPool BytesPool,
	onSelected OnSelected,
	onNotSelected OnNotSelected,
	getPenalty GetPenalty,
) HandleConn {

	idleTimeout := time.Duration(_idleTimeout)

	outboundsPool := pr2.NewPool(128, func() *[]chan OutboundPacket {
		slice := make([]chan OutboundPacket, 0, len(dialers))
		return &slice
	})

	return func(
		parentCtx context.Context,
		conn net.Conn,
		hostPort string,
	) {

		numDialers := len(dialers)

		// chans and contexts
		type Ctx struct {
			Context context.Context
			Cancel  func()
		}
		ctxs := make([]*Ctx, 0, numDialers)
		ptr, put := outboundsPool.Get()
		outbounds := *ptr
		defer func() {
			*ptr = outbounds[:0]
			put()
		}()
		outboundsClosed := make([]bool, 0, numDialers)
		for i := 0; i < numDialers; i++ {
			outbounds = append(outbounds, make(chan OutboundPacket, 16))
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
		closeConnRead := func() {
			once1.Do(func() {
				if c, ok := conn.(interface {
					CloseRead() error
				}); ok {
					c.CloseRead()
				} else {
					conn.Close()
				}
			})
		}
		closeConnWrite := func() {
			once2.Do(func() {
				if c, ok := conn.(interface {
					CloseWrite() error
				}); ok {
					c.CloseWrite()
				} else {
					conn.Close()
				}
				for _, ctx := range ctxs {
					ctx.Cancel()
				}
			})
		}

		// read local conn
		wg.Add(1)
		go func() {
			defer wg.Done()
			var chosenCh chan OutboundPacket
			for {
				deadline := time.Now().Add(idleTimeout)
				if err := conn.SetReadDeadline(deadline); err != nil {
					break
				}
				ptr, put, incRef := bytesPool.GetRC()
				buffer := *ptr
				n, err := conn.Read(buffer)

				if n > 0 {
					buffer = buffer[:n]

					if chosenCh != nil {
						incRef()
						chosenCh <- OutboundPacket{
							Data: buffer,
							Put:  put,
						}

					} else if i := atomic.LoadInt32(&chosen); i != -1 {
						chosenCh = outbounds[i]
						incRef()
						chosenCh <- OutboundPacket{
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
						for _, ch := range outbounds {
							incRef()
							ch <- OutboundPacket{
								Data: buffer,
								Put:  put,
							}
						}
					}

				}

				put()

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

		for i, dialer := range dialers {
			i := i
			outboundCh := outbounds[i]
			dialer := dialer
			wg.Add(1)
			go func() {
				defer wg.Done()

				noDial := false
				if dialer.Deny != nil {
					if dialer.Deny.MatchString(hostPort) {
						noDial = true
					}
				}

				// penalty
				penalty := getPenalty(dialer, hostPort)
				if penalty > 0 {
					time.Sleep(penalty)
					n := atomic.LoadInt32(&chosen)
					if n >= 0 {
						noDial = true
					}
				}

				// dial
				var upstream net.Conn
				if !noDial {
					c, err := dialer.DialContext(ctxs[i].Context, "tcp", hostPort)
					if err == nil {
						upstream = c
					}
				}

				subWg := new(sync.WaitGroup)
				var once1, once2 sync.Once
				closeUpstreamRead := func() {
					once1.Do(func() {
						if upstream != nil {
							if c, ok := upstream.(interface {
								CloseRead() error
							}); ok {
								c.CloseRead()
							} else {
								upstream.Close()
							}
							upstream.SetWriteDeadline(time.Now().Add(time.Second * 30))
						}
					})
				}
				closeUpstreamWrite := func() {
					once2.Do(func() {
						if upstream != nil {
							if c, ok := upstream.(interface {
								CloseWrite() error
							}); ok {
								c.CloseWrite()
							} else {
								upstream.Close()
							}
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
							outboundPacket, ok := <-outboundCh
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
					for outboundPacket := range outboundCh {
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
							onNotSelected(dialer, hostPort)
							closeWrite(-1)
						}
					}()
					if upstream == nil {
						return
					}
					defer closeUpstreamRead()

					ptr, put := bytesPool.Get()
					defer put()
					buffer := *ptr
					selected := false
					for {

						deadline := time.Now().Add(idleTimeout)
						if err := upstream.SetReadDeadline(deadline); err != nil {
							break
						}
						n, err := upstream.Read(buffer)

						if n > 0 {

							if !selected {
								if atomic.CompareAndSwapInt32(&chosen, -1, int32(i)) {
									onSelected(dialer, hostPort)
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
}
