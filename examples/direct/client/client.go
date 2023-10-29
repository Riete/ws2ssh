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
	proxyListenAddr := flag.String("proxy-listen-addr", "127.0.0.1:2222", "proxy listen addr")
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
	proxy, err := ws2ssh.NewSocks5Proxy(tunnel.SSHConn())
	if err != nil {
		log.Fatalln(err)
	}
	err = proxy.ListenAndServe(*proxyListenAddr)
	if err != nil {
		log.Fatalln(err)
	}
}
