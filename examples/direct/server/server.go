package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/riete/go-websocket"
	"github.com/riete/ws2ssh"
)

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
	go tunnel.HandleOutgoing(ws2ssh.Direct)
	log.Println(tunnel.Wait())
}

func main() {
	listenPort := flag.String("listen-port", "8080", "ws server listen port")
	flag.Parse()
	http.HandleFunc("/", serve)
	log.Fatal(http.ListenAndServe(":"+*listenPort, nil).Error())
}
