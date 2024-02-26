package util

import (
	"net"

	"github.com/quic-go/quic-go"
)

type QuicConn struct {
	quic.Stream
	LAddr net.Addr
	RAddr net.Addr
}

func (c *QuicConn) LocalAddr() net.Addr {
	return c.LAddr
}

func (c *QuicConn) RemoteAddr() net.Addr {
	return c.RAddr
}
