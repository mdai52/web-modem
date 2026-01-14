package service

import (
	"time"

	"github.com/rehiy/modem/at"
	"github.com/rehiy/web-modem/database"
	"github.com/rehiy/web-modem/models"
)

// atSmsToModelSms 将AT短信转换为数据库模型
func atSmsToModelSms(atSms at.Sms, receiveNumber string, modemName string) *models.Sms {
	return &models.Sms{
		Content:       atSms.Text,
		SmsIDs:        database.IntArrayToString(atSms.Indices),
		ReceiveTime:   parseSmsTime(atSms.Time),
		ReceiveNumber: receiveNumber,
		SendNumber:    atSms.Number,
		Direction:     "in",
		ModemName:     modemName,
	}
}

// parseSmsTime 解析短信时间字符串
func parseSmsTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Now()
	}

	// 尝试解析常见的短信时间格式
	formats := []string{
		"2006/01/02 15:04:05",
		"2006-01-02 15:04:05",
		"02/01/06 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t
		}
	}

	// 如果无法解析，返回当前时间
	return time.Now()
}
