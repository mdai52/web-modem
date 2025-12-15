package handlers

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:  func(r *http.Request) bool {
		return true
	},
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	// 创建监听通道
	ch := make(chan string, 100)
	serialService.AddListener(ch)

	// 发送实时消息
	for {
		select {
		case message := <-ch:
			err := conn.WriteMessage(websocket. TextMessage, []byte(message))
			if err != nil {
				log.Println("WebSocket write error:", err)
				return
			}
		}
	}
}