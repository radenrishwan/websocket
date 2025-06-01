package websocket

import (
	"errors"
	"net"
	"net/http"
)

type Websocket struct{}

func (self *Websocket) Upgrade(w http.ResponseWriter, r *http.Request) (*Client, error) {
	if r.Header.Get("Upgrade") != "websocket" {
		return nil, errors.New("Upgrade header is not websocket")
	}

	if r.Header.Get("Connection") != "Upgrade" {
		return nil, errors.New("Connection header is not Upgrade")
	}

	key := r.Header.Get("sec-websocket-key")
	if key == "" {
		return nil, errors.New("sec-websocket-key is not set")
	}

	acceptKey := generateWebsocketKey(key)

	conn, _, err := http.NewResponseController(w).Hijack()
	if err != nil {
		return nil, err
	}

	if _, err := conn.Write([]byte("HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey + "\r\n\r\n",
	)); err != nil {
		return nil, err
	}

	return &Client{
		conn: conn,
	}, nil
}

type Client struct {
	conn net.Conn
}

func (self *Client) Read(data []byte) (Frame, int, error) {
	n, err := self.conn.Read(data)
	if err != nil {
		return Frame{}, n, err
	}

	f, err := DecodeFrame(data)
	if err != nil {
		return Frame{}, n, err
	}

	return f, n, err
}

func (self *Client) Write(data []byte, opcode Opcode) (int, error) {
	r, err := EncodeFrame(data, opcode)
	if err != nil {
		return 0, err
	}

	return self.conn.Write(r)
}

func (self *Client) Close(reason []byte, code int) error {
	payload := make([]byte, 2+len(reason))

	payload[0] = byte(code >> 0x8)
	payload[1] = byte(code & 0xFF)

	if len(reason) > 0 {
		copy(payload[2:], reason)
	}

	frame, err := EncodeFrame(payload, CLOSE)
	if err != nil {
		return err
	}

	_, err = self.conn.Write(frame)
	if err != nil {
		return err
	}

	return self.conn.Close()
}
