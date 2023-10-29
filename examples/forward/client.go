package main

import (
	"flag"
	"log"
	"time"

	"github.com/riete/go-websocket"
	"github.com/riete/ws2ssh"
)

func main() {
	serverAddr := flag.String("server-addr", "127.0.0.1:8080", "ws server addr")
	incomingAddr := flag.String("incoming-addr", "127.0.0.1:8888", "incoming addr")
	outgoingAddr := flag.String("outgoing-addr", "192.168.0.146:60081", "outgoing addr")

	flag.Parse()

	c, err := ws.NewClient(nil, "ws", *serverAddr, "/", nil)
	if err != nil {
		log.Fatalln(err)
	}
	c.Conn().SetPongHandler(func(appData string) error {
		log.Println("receive pong")
		return nil
	})
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for {
			<-ticker.C
			_ = c.WritePing(nil)
		}
	}()

	tunnel := ws2ssh.NewSSHTunnel(c.Conn())
	err = tunnel.AsClientSide(nil, true)
	if err != nil {
		log.Fatalln(err)
	}
	forwarder := ws2ssh.NewPortForwarder(*incomingAddr, *outgoingAddr, tunnel)
	log.Fatalln(forwarder.Forward())
}
