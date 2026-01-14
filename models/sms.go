package models

import (
	"time"
)

// Sms 短信模型
type Sms struct {
	ID            int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Content       string    `json:"content" gorm:"not null;type:text"`
	SmsIDs        string    `json:"sms_ids" gorm:"not null;type:text"`
	ReceiveTime   time.Time `json:"receive_time" gorm:"not null;index:idx_sms_receive_time"`
	ReceiveNumber string    `json:"receive_number" gorm:"type:text;index:idx_sms_receive_number"`
	SendNumber    string    `json:"send_number" gorm:"type:text;index:idx_sms_send_number"`
	Direction     string    `json:"direction" gorm:"not null;type:text;check:direction IN ('in', 'out');index:idx_sms_direction"` // "in" 或 "out"
	ModemName     string    `json:"modem_name" gorm:"type:text;index:idx_sms_modem_name"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// SmsFilter 短信查询过滤器
type SmsFilter struct {
	Direction  string    `json:"direction,omitempty"`
	SendNumber string    `json:"send_number,omitempty"`
	ModemName  string    `json:"modem_name,omitempty"`
	StartTime  time.Time `json:"start_time,omitempty"`
	EndTime    time.Time `json:"end_time,omitempty"`
	Limit      int       `json:"limit,omitempty"`
	Offset     int       `json:"offset,omitempty"`
}

// Webhook Webhook配置模型
type Webhook struct {
	ID        int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string    `json:"name" gorm:"not null;unique;type:text"`
	URL       string    `json:"url" gorm:"not null;type:text"`
	Template  string    `json:"template" gorm:"type:text;default:'{}'"`
	Enabled   bool      `json:"enabled" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// Setting 系统设置模型
type Setting struct {
	Key       string    `json:"key" gorm:"primaryKey;type:text"`
	Value     string    `json:"value" gorm:"not null;type:text"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// Settings 系统设置 (DTO)
type Settings struct {
	SmsdbEnabled   bool `json:"smsdb_enabled"`
	WebhookEnabled bool `json:"webhook_enabled"`
}
