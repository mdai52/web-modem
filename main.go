package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/rehiy/web-modem/handler"
)

const (
	defaultPort = "8080"
	apiPrefix   = "/api/v1"
)

func main() {
	// 初始化路由器
	r := mux.NewRouter()
	api := r.PathPrefix(apiPrefix).Subrouter()

	// 调制解调器路由
	api.HandleFunc("/modems", handler.ListModems).Methods("GET")
	api.HandleFunc("/modem/at", handler.SendATCommand).Methods("POST")
	api.HandleFunc("/modem/info", handler.GetModemInfo).Methods("GET")
	api.HandleFunc("/modem/signal", handler.GetSignalStrength).Methods("GET")

	// 短信读写路由
	api.HandleFunc("/modem/sms/list", handler.ListSMS).Methods("GET")
	api.HandleFunc("/modem/sms/send", handler.SendSMS).Methods("POST")
	api.HandleFunc("/modem/sms/delete", handler.DeleteSMS).Methods("POST")

	// WebSocket
	r.HandleFunc("/ws", handler.HandleWebSocket)

	// 静态文件服务
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("webview")))

	// 启动服务器
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	log.Printf("Server starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
