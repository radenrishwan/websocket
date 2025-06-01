package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/radenrishwan/websocket"
	"github.com/radenrishwan/websocket/utility"
)

const (
	MESSAGE_JOIN = iota
	MESSAGE_LEAVE
	MESSAGE_MESSAGE
)

type ChatMessage struct {
	Type      int    `json:"type"`
	Username  string `json:"username"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

var ws = websocket.Websocket{}

func main() {
	chatRoom := utility.NewRoom("general", &utility.RoomOption{
		OnError: func(err error) {
			log.Println("Room error:", err)
		},
		OnEnter: func(msg utility.Message) {
			log.Println("User entered:", msg.Data)
		},
		OnLeave: func(msg utility.Message) {
			log.Println("User left:", msg.Data)
		},
		OnMessage: func(msg utility.Message) {
			log.Println("New message:", msg.Data)
		},
	})

	go chatRoom.Run()

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/index.html")
	})

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("username")
		if username == "" {
			username = fmt.Sprintf("Guest_%d", time.Now().Unix())
		}

		ws := websocket.Websocket{}
		client, err := ws.Upgrade(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer client.Close(nil, 1000)

		joinMsg := ChatMessage{
			Type:      MESSAGE_JOIN,
			Username:  username,
			Content:   fmt.Sprintf("%s joined the chat", username),
			Timestamp: time.Now().Unix(),
		}
		joinData, _ := json.Marshal(joinMsg)
		chatRoom.BroadcastEnter(joinData, client)

		for {
			data := make([]byte, 8192)
			frame, n, err := client.Read(data)
			if err != nil {
				log.Printf("Error reading from client: %v", err)
				break
			}

			if n == 0 {
				continue
			}

			var msg ChatMessage
			if err := json.Unmarshal(frame.Payload, &msg); err != nil {
				log.Printf("Invalid message format: %v", err)
				continue
			}

			msg.Username = username
			msg.Timestamp = time.Now().Unix()

			messageData, _ := json.Marshal(msg)
			chatRoom.BroadcastMessage(messageData, client)
		}

		leaveMsg := ChatMessage{
			Type:      MESSAGE_LEAVE,
			Username:  username,
			Content:   fmt.Sprintf("%s left the chat", username),
			Timestamp: time.Now().Unix(),
		}
		leaveData, _ := json.Marshal(leaveMsg)
		chatRoom.BroadcastLeave(leaveData, client)
	})

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalln(err)
	}
}
