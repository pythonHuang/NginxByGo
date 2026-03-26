package http

import (
	"time"
)

// Connection 连接
type Connection struct {
	FD       int
	Conn     interface{}  // net.Conn
	ReadEvent  interface{}
	WriteEvent interface{}

	RemoteAddr string
	LocalAddr  string

	Server     *Server
	Request    *Request

	CreatedAt  time.Time
	LastActive time.Time
}

// Close 关闭连接
func (c *Connection) Close() error {
	return nil
}