package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/riete/go-websocket"
	"github.com/riete/ws2ssh"
)

var forwarder *ws2ssh.SSHTunnel

func serve(w http.ResponseWriter, r *http.Request) {
	s, err := ws.NewServer(w, r, nil, ws.WithDisableCheckOrigin())
	if err != nil {
		log.Fatalln(err)
	}
	defer s.Close()
	tunnel := ws2ssh.NewSSHTunnel(s.Conn())
	err = tunnel.AsServerSide(nil, true)
	if err != nil {
		log.Fatalln(err)
	}
	go tunnel.HandleOutgoing(ws2ssh.Next(forwarder))
	log.Println(tunnel.Wait())
}

func forward(w http.ResponseWriter, r *http.Request) {
	s, err := ws.NewServer(w, r, nil, ws.WithDisableCheckOrigin())
	if err != nil {
		log.Fatalln(err)
	}
	defer s.Close()
	forwarder = ws2ssh.NewSSHTunnel(s.Conn())
	err = forwarder.AsClientSide(nil, true)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(forwarder.Wait())
}

func main() {
	listenPort := flag.String("listen-port", "8080", "ws server listen port")
	flag.Parse()
	http.HandleFunc("/", serve)
	http.HandleFunc("/forward", forward)
	log.Fatal(http.ListenAndServe(":"+*listenPort, nil).Error())
}
