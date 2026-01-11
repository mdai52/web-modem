package database

import (
	"encoding/json"
	"fmt"

	"github.com/rehiy/web-modem/models"
	"gorm.io/gorm"
)

// Create 创建webhook配置
func Create(webhook *models.Webhook) error {
	result := db.Create(webhook)
	if result.Error != nil {
		return fmt.Errorf("failed to create webhook: %w", result.Error)
	}
	return nil
}

// Update 更新webhook配置
func Update(webhook *models.Webhook) error {
	result := db.Save(webhook)
	if result.Error != nil {
		return fmt.Errorf("failed to update webhook: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("webhook not found")
	}
	return nil
}

// Delete 删除webhook配置
func Delete(id int) error {
	result := db.Delete(&models.Webhook{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete webhook: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("webhook not found")
	}
	return nil
}

// Detail 根据ID获取webhook配置
func Detail(id int) (*models.Webhook, error) {
	var webhook models.Webhook
	result := db.First(&webhook, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("webhook not found")
		}
		return nil, fmt.Errorf("failed to get webhook: %w", result.Error)
	}
	return &webhook, nil
}

// GetAllWebhooks 获取所有webhook配置
func GetAllWebhooks() ([]models.Webhook, error) {
	var webhooks []models.Webhook
	result := db.Order("created_at DESC").Find(&webhooks)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to query webhooks: %w", result.Error)
	}
	return webhooks, nil
}

// GetEnabledWebhooks 获取所有启用的webhook
func GetEnabledWebhooks() ([]models.Webhook, error) {
	var webhooks []models.Webhook
	result := db.Where("enabled = ?", true).Order("created_at DESC").Find(&webhooks)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to query enabled webhooks: %w", result.Error)
	}
	return webhooks, nil
}

// DetailTemplateData 解析webhook模板数据
func DetailTemplateData(template string) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(template), &data); err != nil {
		return nil, fmt.Errorf("invalid template JSON: %w", err)
	}
	return data, nil
}

// IsWebhookEnabled 检查webhook功能是否启用
func IsWebhookEnabled() bool {
	var setting models.Setting
	result := db.Where("key = ?", "webhook_enabled").First(&setting)
	if result.Error != nil {
		return false
	}
	return setting.Value == "true"
}

// SetWebhookEnabled 设置webhook功能启用状态
func SetWebhookEnabled(enabled bool) error {
	value := "false"
	if enabled {
		value = "true"
	}

	setting := models.Setting{Key: "webhook_enabled", Value: value}
	result := db.Where(models.Setting{Key: "webhook_enabled"}).Assign(setting).FirstOrCreate(&setting)
	if result.Error != nil {
		return fmt.Errorf("failed to set webhook_enabled: %w", result.Error)
	}
	return nil
}
