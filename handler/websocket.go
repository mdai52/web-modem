package handler

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/rehiy/web-modem/service"
)

// WebSocketHandler WebSocket处理器
type WebSocketHandler struct {
	upgrader websocket.Upgrader
}

// NewWebSocketHandler 创建新的WebSocket处理器
func NewWebSocketHandler() *WebSocketHandler {
	return &WebSocketHandler{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// HandleWebSocket 处理WebSocket连接
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("WebSocket client connected: %s", r.RemoteAddr)

	// 直接推送Modem事件到客户端
	for event := range service.ModemEvent {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(event)); err != nil {
			log.Printf("WebSocket write error: %v", err)
			return
		}
	}
}