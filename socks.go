package atproxy

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/reusee/e4"
)

const (
	VERSION = byte(5)

	METHOD_NOT_REQUIRED  = byte(0)
	METHOD_NO_ACCEPTABLE = byte(0xff)

	RESERVED = byte(0)

	ADDR_TYPE_IP     = byte(1)
	ADDR_TYPE_IPV6   = byte(4)
	ADDR_TYPE_DOMAIN = byte(3)

	CMD_CONNECT       = byte(1)
	CMD_BIND          = byte(2)
	CMD_UDP_ASSOCIATE = byte(3)

	REP_SUCCEED                    = byte(0)
	REP_SERVER_FAILURE             = byte(1)
	REP_CONNECTION_NOT_ALLOW       = byte(2)
	REP_NETWORK_UNREACHABLE        = byte(3)
	REP_HOST_UNREACHABLE           = byte(4)
	REP_CONNECTION_REFUSED         = byte(5)
	REP_TTL_EXPIRED                = byte(6)
	REP_COMMAND_NOT_SUPPORTED      = byte(7)
	REP_ADDRESS_TYPE_NOT_SUPPORTED = byte(8)
)

var ErrBadHandshake = errors.New("bad handshake")

func (s *Server) socksHandshake(conn net.Conn) (hostPort string, err error) {
	defer he(&err)

	ce(conn.SetReadDeadline(time.Now().Add(time.Second * 8)))
	ce(conn.SetWriteDeadline(time.Now().Add(time.Second * 8)))

	var greetings struct {
		Version        byte
		NumAuthMethods byte
	}
	ce(binary.Read(conn, binary.BigEndian, &greetings))
	if greetings.Version != VERSION {
		err = we.With(
			e4.NewInfo("bad version"),
		)(ErrBadHandshake)
		return
	}
	authMethods := make([]byte, int(greetings.NumAuthMethods))
	_, err = io.ReadFull(conn, authMethods)
	ce(err)
	if bytes.IndexByte(authMethods, METHOD_NOT_REQUIRED) == -1 {
		err = we.With(
			e4.NewInfo("bad auth method"),
		)(ErrBadHandshake)
		return
	}
	_, err = conn.Write([]byte{
		VERSION,
		METHOD_NOT_REQUIRED,
	})
	ce(err)

	var request struct {
		Version     byte
		Command     byte
		_           byte
		AddressType byte
	}
	ce(binary.Read(conn, binary.BigEndian, &request))
	if request.Version != VERSION {
		err = we.With(
			e4.NewInfo("bad version"),
		)(ErrBadHandshake)
		return
	}
	if request.Command != CMD_CONNECT {
		err = we.With(
			e4.NewInfo("bad command"),
		)(ErrBadHandshake)
		return
	}
	if request.AddressType != ADDR_TYPE_IP &&
		request.AddressType != ADDR_TYPE_DOMAIN &&
		request.AddressType != ADDR_TYPE_IPV6 {
		err = we.With(
			e4.NewInfo("bad address type"),
		)(ErrBadHandshake)
		return
	}
	var host string
	switch request.AddressType {
	case ADDR_TYPE_IP:
		bs := make([]byte, 4)
		_, err = io.ReadFull(conn, bs)
		ce(err)
		host = net.IP(bs).String()
	case ADDR_TYPE_IPV6:
		bs := make([]byte, 16)
		_, err = io.ReadFull(conn, bs)
		ce(err)
		host = net.IP(bs).String()
	case ADDR_TYPE_DOMAIN:
		var l uint8
		ce(binary.Read(conn, binary.BigEndian, &l))
		bs := make([]byte, int(l))
		_, err = io.ReadFull(conn, bs)
		ce(err)
		host = string(bs)
	}
	var port uint16
	ce(binary.Read(conn, binary.BigEndian, &port))

	_, err = conn.Write([]byte{
		VERSION,
		REP_SUCCEED,
		RESERVED,
		ADDR_TYPE_IP,
		0, 0, 0, 0,
		0, 0,
	})
	ce(err)

	hostPort = net.JoinHostPort(host, strconv.Itoa(int(port)))

	return
}
