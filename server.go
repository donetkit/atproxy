package atproxy

import (
	"context"
	"net"
	"net/http"

	"github.com/reusee/atproxy/internal"
	"github.com/reusee/dscope"
)

type Server struct {
	Scope Scope
}

type NewServer func(defSets ...[]any) *Server

func (g *Global) NewServer() NewServer {
	return func(defSets ...[]any) *Server {
		server := new(Server)
		server.Scope = g.Scope.Fork(dscope.Methods(server)...)
		for _, defs := range defSets {
			server.Scope = server.Scope.Fork(defs...)
		}
		return server
	}
}

type MaxClients int

func (Server) MaxClients() MaxClients {
	return 0
}

type ClientSemaphore chan struct{}

func (Server) ClientSemaphore(
	max MaxClients,
) ClientSemaphore {
	if max == 0 {
		return nil
	}
	return make(chan struct{}, max)
}

type Serve func(
	ctx context.Context,
	socksLn *net.TCPListener,
	httpLn *net.TCPListener,
) (
	err error,
)

func (Server) Serve(
	clientSem ClientSemaphore,
	handleRequest HandleRequest,
	handleConn HandleConn,
) Serve {

	return func(
		ctx context.Context,
		socksLn *net.TCPListener,
		httpLn *net.TCPListener,
	) (
		err error,
	) {
		defer he(&err)

		// http
		go func() {
			server := &http.Server{
				Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					if clientSem != nil {
						clientSem <- struct{}{}
						defer func() {
							<-clientSem
						}()
					}

					if req.Method != http.MethodConnect {
						handleRequest(ctx, req, w)
						return
					}

					hostPort := req.Host
					hijacker, ok := w.(http.Hijacker)
					if !ok {
						http.Error(w, "not supported", http.StatusInternalServerError)
						return
					}
					c, _, err := hijacker.Hijack()
					if err != nil {
						http.Error(w, err.Error(), http.StatusServiceUnavailable)
						return
					}
					defer c.Close()

					_, err = c.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
					if err != nil {
						return
					}

					handleConn(ctx, c, hostPort)
				}),
			}
			server.Serve(httpLn)
		}()

		// socks
		for {
			conn, err := socksLn.AcceptTCP()
			ce(err)

			if clientSem != nil {
				clientSem <- struct{}{}
			}

			go func() {
				if clientSem != nil {
					defer func() {
						<-clientSem
					}()
				}
				defer conn.Close()

				hostPort, err := internal.Socks5ServerHandshake(conn)
				if err != nil {
					return
				}

				handleConn(ctx, conn, hostPort)
			}()

		}

	}
}
