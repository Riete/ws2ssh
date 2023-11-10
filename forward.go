package ws2ssh

import (
	"context"
	"net"
	"sync"
)

// PortForwarder
// incoming and outgoing is ip:port
// incoming ip:port -> ssh tunnel -> outgoing ip:port
type PortForwarder struct {
	tunnel   *SSHTunnel
	l        net.Listener
	o        sync.Once
	incoming string
	outgoing string
}

func (p *PortForwarder) listenForIncoming() error {
	var err error
	p.l, err = net.Listen("tcp", p.incoming)
	return err
}

func (p *PortForwarder) Forward() error {
	return p.ForwardContext(context.Background())
}

func (p *PortForwarder) ForwardContext(ctx context.Context) error {
	err := p.listenForIncoming()
	if err != nil {
		return err
	}
	defer p.Close()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			local, err := p.l.Accept()
			if err != nil {
				return err
			}
			ch, err := p.tunnel.HandleIncoming(p.outgoing)
			if err != nil {
				return err
			}
			go pipe(local, ch)
		}
	}
}

func (p *PortForwarder) Close() error {
	var err error
	p.o.Do(func() {
		err = p.l.Close()
	})
	return err
}

func NewPortForwarder(incoming, outgoing string, tunnel *SSHTunnel) *PortForwarder {
	return &PortForwarder{tunnel: tunnel, incoming: incoming, outgoing: outgoing}
}
