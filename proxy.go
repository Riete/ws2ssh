package ws2ssh

import (
	"context"
	"log"
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

type ProxyOption func(*socks5.Config)

func WithResolver(resolver socks5.NameResolver) ProxyOption {
	return func(c *socks5.Config) {
		c.Resolver = resolver
	}
}

func WithCredentials(credentials map[string]string) ProxyOption {
	return func(c *socks5.Config) {
		c.Credentials = socks5.StaticCredentials(credentials)
	}
}

func WithLogger(logger *log.Logger) ProxyOption {
	return func(c *socks5.Config) {
		c.Logger = logger
	}
}

func NewSocks5ProxyServer(conn ssh.Conn, options ...ProxyOption) *Socks5ProxyServer {
	conf := &socks5.Config{
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return NewNetConnFromSSHConn(conn, addr)
		},
	}
	for _, option := range options {
		option(conf)
	}
	s, _ := socks5.New(conf)
	return &Socks5ProxyServer{s: s}
}
