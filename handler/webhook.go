package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/rehiy/web-modem/database"
	"github.com/rehiy/web-modem/models"
	"github.com/rehiy/web-modem/service"
)

// WebhookHandler Webhook处理器
type WebhookHandler struct {
	ws *service.WebhookService
}

// NewWebhookHandler 创建新的Webhook处理器
func NewWebhookHandler() *WebhookHandler {
	return &WebhookHandler{
		ws: service.NewWebhookService(),
	}
}

// Create 创建Webhook配置
func (h *WebhookHandler) Create(w http.ResponseWriter, r *http.Request) {
	var webhook models.Webhook
	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		respondJSON(w, http.StatusBadRequest, H{"error": err.Error()})
		return
	}

	// 验证必填字段
	if webhook.Name == "" || webhook.URL == "" {
		respondJSON(w, http.StatusBadRequest, H{"error": "name and url are required"})
		return
	}

	// 如果模板为空，使用默认模板
	if webhook.Template == "" {
		webhook.Template = "{}"
	}

	if err := database.Create(&webhook); err != nil {
		respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusCreated, webhook)
}

// Update 更新Webhook配置
func (h *WebhookHandler) Update(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	idStr := vars.Get("id")
	if idStr == "" {
		respondJSON(w, http.StatusBadRequest, H{"error": "id is required"})
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, H{"error": "invalid id"})
		return
	}

	var webhook models.Webhook
	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		respondJSON(w, http.StatusBadRequest, H{"error": err.Error()})
		return
	}

	webhook.ID = id

	// 验证必填字段
	if webhook.Name == "" || webhook.URL == "" {
		respondJSON(w, http.StatusBadRequest, H{"error": "name and url are required"})
		return
	}

	// 如果模板为空，使用默认模板
	if webhook.Template == "" {
		webhook.Template = "{}"
	}

	if err := database.Update(&webhook); err != nil {
		respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, webhook)
}

// Delete 删除Webhook配置
func (h *WebhookHandler) Delete(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	idStr := vars.Get("id")
	if idStr == "" {
		respondJSON(w, http.StatusBadRequest, H{"error": "id is required"})
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, H{"error": "invalid id"})
		return
	}

	if err := database.Delete(id); err != nil {
		respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, H{
		"status": "deleted",
		"id":     id,
	})
}

// Detail 获取单个Webhook配置
func (h *WebhookHandler) Detail(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	idStr := vars.Get("id")
	if idStr == "" {
		respondJSON(w, http.StatusBadRequest, H{"error": "id is required"})
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, H{"error": "invalid id"})
		return
	}

	webhook, err := database.Detail(id)
	if err != nil {
		respondJSON(w, http.StatusNotFound, H{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, webhook)
}

// List 获取所有Webhook配置
func (h *WebhookHandler) List(w http.ResponseWriter, r *http.Request) {
	webhooks, err := database.GetAllWebhooks()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, webhooks)
}

// Test 测试Webhook
func (h *WebhookHandler) Test(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	idStr := vars.Get("id")
	if idStr == "" {
		respondJSON(w, http.StatusBadRequest, H{"error": "id is required"})
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, H{"error": "invalid id"})
		return
	}

	webhook, err := database.Detail(id)
	if err != nil {
		respondJSON(w, http.StatusNotFound, H{"error": err.Error()})
		return
	}

	if err := h.ws.Test(webhook); err != nil {
		respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, H{
		"status":  "success",
		"message": "Webhook test sent successfully",
	})
}

// DetailSettings 获取Webhook设置
func (h *WebhookHandler) DetailSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := database.GetSettings()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, settings)
}

// UpdateSettings 更新Webhook设置
func (h *WebhookHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WebhookEnabled bool `json:"webhook_enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, H{"error": err.Error()})
		return
	}

	if err := database.SetWebhookEnabled(req.WebhookEnabled); err != nil {
		respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, H{
		"status":          "updated",
		"webhook_enabled": req.WebhookEnabled,
	})
}
