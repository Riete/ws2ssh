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
	flag.Parse()

	c, err := ws.NewClient(nil, "ws", *serverAddr, "/forward", nil)
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
	err = tunnel.AsServerSide(nil, true)
	if err != nil {
		log.Fatalln(err)
	}
	go tunnel.HandleOutgoing(ws2ssh.Direct)
	log.Println(tunnel.Wait())
}
