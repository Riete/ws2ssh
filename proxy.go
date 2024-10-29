package ws2ssh

import (
	"context"
	"net"

	"golang.org/x/crypto/ssh"

	"github.com/armon/go-socks5"
)

type Socks5ProxyServer struct {
	s *socks5.Server
}

func (s Socks5ProxyServer) ListenAndServe(addr string) error {
	return s.ListenAndServeContext(context.Background(), addr)
}

func (s Socks5ProxyServer) ListenAndServeContext(ctx context.Context, addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		_ = l.Close()
	}()
	return s.s.Serve(l)
}

type NameResolver struct {
	net  string
	ip   string
	port string
}

func (n NameResolver) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return net.Dial(n.net, n.ip+":"+n.port)
		},
	}
	ips, err := resolver.LookupIP(ctx, "ip", name)
	if err != nil {
		return ctx, nil, err
	}
	return ctx, ips[0], err
}

func NewNameResolver(net, ip, port string) socks5.NameResolver {
	return &NameResolver{net: net, ip: ip, port: port}
}

func NewSocks5ProxyServer(conn ssh.Conn, resolver socks5.NameResolver) *Socks5ProxyServer {
	conf := &socks5.Config{
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return NewNetConnFromSSHConn(conn, addr)
		},
		Resolver: resolver,
	}
	s, _ := socks5.New(conf)
	return &Socks5ProxyServer{s: s}
}
