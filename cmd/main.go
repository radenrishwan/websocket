package main

import (
	"io"
	"log"
	"net"
	"net/http"

	"github.com/radenrishwan/websocket"
)

var ws = websocket.Websocket{}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/index.html")
	})

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := ws.Upgrade(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		defer conn.Close(nil, 1000)

		for {
			data := make([]byte, 1024)
			f, n, err := conn.Read(data)
			if err != nil {
				if err == io.EOF {
					log.Println("Look like the client disconnected (EOF)")
				} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					log.Println("Read timeout: ", err)
				} else {
					log.Println("Error reading from client: ", err)
				}

				break
			}

			if n == 0 {
				continue
			}

			log.Println(string(f.Payload))

			conn.Write([]byte("Hello, world!"), websocket.TEXT)
		}
	})

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalln(err)
	}
}
