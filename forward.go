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
	incoming string
	outgoing string
}

func (p PortForwarder) listenForIncoming() (net.Listener, error) {
	return net.Listen("tcp", p.incoming)
}

func (p PortForwarder) Forward() error {
	return p.ForwardContext(context.Background())
}

func (p PortForwarder) ForwardContext(ctx context.Context) error {
	listener, err := p.listenForIncoming()
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			local, err := listener.Accept()
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

func NewPortForwarder(incoming, outgoing string, tunnel *SSHTunnel) *PortForwarder {
	return &PortForwarder{tunnel: tunnel, incoming: incoming, outgoing: outgoing}
}
