package ws2ssh

import (
	"context"
	"net"
)

// PortForwarder
// incoming and outgoing is ip:port
// incoming ip:port -> ssh tunnel -> outgoing ip:port
type PortForwarder struct {
	tunnel   *SSHTunnel
	errCh    chan error
	incoming string
	outgoing string
}

func (p *PortForwarder) listenForIncoming() (net.Listener, error) {
	return net.Listen("tcp", p.incoming)
}

func (p *PortForwarder) Forward() error {
	return p.ForwardContext(context.Background())
}

func (p *PortForwarder) ForwardContext(ctx context.Context) error {
	l, err := p.listenForIncoming()
	if err != nil {
		return err
	}
	defer l.Close()
	go func() {
		for {
			local, err := l.Accept()
			if err != nil {
				p.errCh <- err
				return
			}
			ch, err := p.tunnel.HandleIncoming(p.outgoing)
			if err != nil {
				p.errCh <- err
				return
			}
			go pipe(local, ch)
		}
	}()
	select {
	case <-ctx.Done():
		return nil
	case err := <-p.errCh:
		return err
	}
}

func NewPortForwarder(incoming, outgoing string, tunnel *SSHTunnel) *PortForwarder {
	return &PortForwarder{tunnel: tunnel, incoming: incoming, outgoing: outgoing, errCh: make(chan error)}
}
