package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"log"
	"net"
	"net/http"
)

const MAGIC_KEY = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

type Client struct {
	conn net.Conn
	brw  *bufio.ReadWriter
}

func generateWebsocketKey(key string) string {
	sha := sha1.New()
	sha.Write([]byte(key))
	sha.Write([]byte(MAGIC_KEY))

	return base64.StdEncoding.EncodeToString(sha.Sum(nil))
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("sec-websocket-key")
		if key == "" {
			http.Error(w, "missing Sec-WebSocket-Key header", http.StatusBadRequest)
			return
		}

		acceptKey := generateWebsocketKey(key)

		conn, rw, err := http.NewResponseController(w).Hijack()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if _, err := rw.WriteString("HTTP/1.1 101 Switching Protocols\r\n" +
			"Upgrade: websocket\r\n" +
			"Connection: Upgrade\r\n" +
			"Sec-WebSocket-Accept: " + acceptKey + "\r\n\r\n",
		); err != nil {
			log.Println(err)
			return
		}

		_ = Client{
			conn: conn,
			brw:  rw,
		}

		for {
			// TODO: read the message here
		}
	})

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalln(err)
	}
}
