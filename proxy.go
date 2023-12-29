package ws2ssh

import (
	"context"
	"net"

	"golang.org/x/crypto/ssh"

	"github.com/armon/go-socks5"
)

type Socks5Proxy struct {
	s *socks5.Server
}

func (s Socks5Proxy) ListenAndServe(addr string) error {
	return s.s.ListenAndServe("tcp", addr)
}

func NewSocks5Proxy(conn ssh.Conn) (*Socks5Proxy, error) {
	conf := &socks5.Config{
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return NewNetConnFromSSHConn(conn, addr)
		},
	}
	s, err := socks5.New(conf)
	return &Socks5Proxy{s: s}, err
}
