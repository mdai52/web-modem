package main

import (
	"log"
	"net/http"
	"os"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"modem-manager/handlers"
)

func main() {
	router := mux.NewRouter()

	// API 路由
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/modems", handlers.ListModems).Methods("GET")
	api.HandleFunc("/modem/connect", handlers.ConnectModem).Methods("POST")
	api.HandleFunc("/modem/disconnect", handlers.DisconnectModem).Methods("POST")
	api.HandleFunc("/modem/send", handlers.SendATCommand).Methods("POST")
	api.HandleFunc("/modem/info", handlers.GetModemInfo).Methods("GET")
	api.HandleFunc("/modem/signal", handlers.GetSignalStrength).Methods("GET")
	api.HandleFunc("/modem/sms/list", handlers.ListSMS).Methods("GET")
	api.HandleFunc("/modem/sms/send", handlers.SendSMS).Methods("POST")
	
	// WebSocket 路由
	router.HandleFunc("/ws", handlers. HandleWebSocket)

	// 静态文件服务
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("frontend")))

	// CORS 配置
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials:  true,
	})

	handler := c.Handler(router)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Server starting on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}