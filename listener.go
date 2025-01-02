package jsonrpc2

import (
	"io"
	"net"
)

// Listener is an interface for accepting connections.
//
// This interface is used for `Server.Serve` method.
type Listener interface {
	Accept() (io.ReadWriter, error)
	Close() error
}

type netListener struct {
	listener net.Listener
}

func (l *netListener) Accept() (io.ReadWriter, error) {
	conn, err := l.listener.Accept()
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (l *netListener) Close() error {
	return l.listener.Close()
}

// NewTCPListener creates a new Listener for TCP connections.
func NewTCPListener(addr string) (Listener, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &netListener{listener: listener}, nil
}

// NewUnixListener creates a new Listener for Unix domain socket connections.
func NewUnixListener(addr string) (Listener, error) {
	listener, err := net.Listen("unix", addr)
	if err != nil {
		return nil, err
	}
	return &netListener{listener: listener}, nil
}
