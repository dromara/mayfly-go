package utils

import (
	"net"
	"time"
)

type WrapSshConn struct {
	Conn net.Conn
}

func (c *WrapSshConn) Read(b []byte) (n int, err error) {
	return c.Conn.Read(b)
}

func (c *WrapSshConn) Write(b []byte) (n int, err error) {
	return c.Conn.Write(b)
}
func (c *WrapSshConn) Close() error {
	return c.Conn.Close()
}
func (c *WrapSshConn) LocalAddr() net.Addr {
	return c.Conn.LocalAddr()
}
func (c *WrapSshConn) RemoteAddr() net.Addr {
	return c.Conn.RemoteAddr()
}
func (c *WrapSshConn) SetDeadline(t time.Time) error {
	return c.Conn.SetDeadline(t)
}
func (c *WrapSshConn) SetReadDeadline(t time.Time) error {
	return nil
}
func (c *WrapSshConn) SetWriteDeadline(t time.Time) error {
	return nil
}
