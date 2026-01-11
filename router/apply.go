package router

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rehiy/web-modem/handler"
)

func Apply() *mux.Router {
	r := mux.NewRouter()

	// API 路由
	api := r.PathPrefix("/api").Subrouter()
	ModemRegister(api)
	SmsdbRegister(api)
	WebhookRegister(api)

	// WebSocket
	WebSocketRegister(r)

	// 静态文件服务
	StaticServer(r)

	return r
}

func ModemRegister(r *mux.Router) {
	mh := handler.NewModemHandler()

	// 模块列表
	r.HandleFunc("/modem/list", mh.List).Methods("GET")

	// 模块操作
	r.HandleFunc("/modem/send", mh.Command).Methods("POST")
	r.HandleFunc("/modem/info", mh.BasicInfo).Methods("GET")
	r.HandleFunc("/modem/signal", mh.SignalStrength).Methods("GET")

	// 短信读写
	r.HandleFunc("/modem/sms/list", mh.ListSMS).Methods("GET")
	r.HandleFunc("/modem/sms/send", mh.SendSMS).Methods("POST")
	r.HandleFunc("/modem/sms/delete", mh.DeleteSMS).Methods("POST")
}

func SmsdbRegister(r *mux.Router) {
	dh := handler.NewSmsdbHandler()

	// 短信存储管理
	r.HandleFunc("/smsdb/list", dh.List).Methods("GET")
	r.HandleFunc("/smsdb/delete", dh.Delete).Methods("POST")
	r.HandleFunc("/smsdb/settings", dh.GetSettings).Methods("GET")
	r.HandleFunc("/smsdb/settings", dh.UpdateSettings).Methods("PUT")
}

func WebhookRegister(r *mux.Router) {
	wh := handler.NewWebhookHandler()

	// Webhook配置管理
	r.HandleFunc("/webhook", wh.Create).Methods("POST")
	r.HandleFunc("/webhook/list", wh.List).Methods("GET")
	r.HandleFunc("/webhook/get", wh.Detail).Methods("GET")
	r.HandleFunc("/webhook/update", wh.Update).Methods("PUT")
	r.HandleFunc("/webhook/delete", wh.Delete).Methods("DELETE")
	r.HandleFunc("/webhook/test", wh.Test).Methods("POST")
	r.HandleFunc("/webhook/settings", wh.DetailSettings).Methods("GET")
	r.HandleFunc("/webhook/settings", wh.UpdateSettings).Methods("PUT")
}

func WebSocketRegister(r *mux.Router) {
	ws := handler.NewWebSocketHandler()

	r.HandleFunc("/ws/modem", ws.HandleWebSocket)
}

func StaticServer(r *mux.Router) {
	fs := http.FileServer(http.Dir("./webview"))
	r.PathPrefix("/").Handler(fs)
}
