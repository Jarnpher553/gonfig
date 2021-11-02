package listener

import (
	"net"
	"time"
)

type Listener struct {
	*net.TCPListener
}

func NewListener(addr string) (*Listener, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &Listener{ln.(*net.TCPListener)}, nil
}

func (l *Listener) Accept() (net.Conn, error) {
	conn, err := l.TCPListener.Accept()
	if err != nil {
		return nil, err
	}
	tcpConn := conn.(*net.TCPConn)
	err = tcpConn.SetKeepAlive(true)
	if err != nil {
		return nil, err
	}
	err = tcpConn.SetKeepAlivePeriod(time.Second * 60)
	if err != nil {
		return nil, err
	}
	return tcpConn, nil
}