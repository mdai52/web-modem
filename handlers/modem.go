package handlers

import (
	"encoding/json"
	"net/http"

	"modem-manager/models"
	"modem-manager/services"
)

var serialService = services.GetSerialService()

// 列出可用串口
func ListModems(w http.ResponseWriter, r *http.Request) {
	// 扫描并一次性连接可用串口
	ports, err := serialService.ScanAndConnectAll(115200)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, ports)
}

// 发送 AT 命令
func SendATCommand(w http.ResponseWriter, r *http.Request) {
	var cmd models.ATCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if cmd.Port == "" {
		respondError(w, http.StatusBadRequest, "port is required")
		return
	}

	response, err := serialService.SendATCommand(cmd.Port, cmd.Command)
	if err != nil {
		cmd.Error = err.Error()
	}
	cmd.Response = response

	respondJSON(w, http.StatusOK, cmd)
}

// 获取 Modem 信息
func GetModemInfo(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Query().Get("port")
	if p == "" { respondError(w, http.StatusBadRequest, "port is required"); return }
	info, err := serialService.GetModemInfo(p)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, info)
}

// 获取信号强度
func GetSignalStrength(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Query().Get("port")
	if p == "" { respondError(w, http.StatusBadRequest, "port is required"); return }
	signal, err := serialService.GetSignalStrength(p)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, signal)
}

// 列出短信
func ListSMS(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Query().Get("port")
	if p == "" { respondError(w, http.StatusBadRequest, "port is required"); return }
	smsList, err := serialService.ListSMS(p)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, smsList)
}

// 发送短信
func SendSMS(w http.ResponseWriter, r *http.Request) {
	var req models.SendSMSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if req.Port == "" {
		respondError(w, http.StatusBadRequest, "port is required")
		return
	}

	err := serialService.SendSMS(req.Port, req.Number, req.Message)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

// 辅助函数：返回 JSON
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// 辅助函数：返回错误
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
