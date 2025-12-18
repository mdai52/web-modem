package handlers

import (
	"encoding/json"
	"net/http"

	"modem-manager/models"
	"modem-manager/services"
)

var serialManager = services.GetSerialManager()

// 列出可用串口
func ListModems(w http.ResponseWriter, r *http.Request) {
	// 扫描并一次性连接可用串口
	ports, err := serialManager.Scan(115200)
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

	svc, ok := getServiceFromPortParam(w, cmd.Port)
	if !ok {
		return
	}

	response, err := svc.SendATCommand(cmd.Command)
	if err != nil {
		cmd.Error = err.Error()
	}
	cmd.Response = response

	respondJSON(w, http.StatusOK, cmd)
}

// 获取 Modem 信息
func GetModemInfo(w http.ResponseWriter, r *http.Request) {
	svc, ok := requireQueryService(w, r)
	if !ok {
		return
	}

	info, err := svc.GetModemInfo()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, info)
}

// 获取信号强度
func GetSignalStrength(w http.ResponseWriter, r *http.Request) {
	svc, ok := requireQueryService(w, r)
	if !ok {
		return
	}

	signal, err := svc.GetSignalStrength()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, signal)
}

// 列出短信
func ListSMS(w http.ResponseWriter, r *http.Request) {
	svc, ok := requireQueryService(w, r)
	if !ok {
		return
	}

	smsList, err := svc.ListSMS()
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

	svc, ok := getServiceFromPortParam(w, req.Port)
	if !ok {
		return
	}

	if err := svc.SendSMS(req.Number, req.Message); err != nil {
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

func requireQueryService(w http.ResponseWriter, r *http.Request) (*services.SerialService, bool) {
	return getServiceFromPortParam(w, r.URL.Query().Get("port"))
}

func getServiceFromPortParam(w http.ResponseWriter, port string) (*services.SerialService, bool) {
	if port == "" {
		respondError(w, http.StatusBadRequest, "port is required")
		return nil, false
	}

	svc, err := serialManager.GetService(port)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return nil, false
	}

	return svc, true
}
