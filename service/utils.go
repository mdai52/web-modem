package service

import (
	"time"

	"github.com/rehiy/modem/at"
	"github.com/rehiy/web-modem/database"
	"github.com/rehiy/web-modem/models"
)

// atSMSToModelSMS 将AT短信转换为数据库模型
func atSMSToModelSMS(smsData at.SMS, receiveNumber string) *models.SMS {
	return &models.SMS{
		Content:       smsData.Text,
		SMSIDs:        database.IntArrayToString(smsData.Indices),
		ReceiveTime:   parseSMSTime(smsData.Time),
		ReceiveNumber: receiveNumber,
		SendNumber:    smsData.PhoneNumber,
		Direction:     "in",
	}
}

// parseSMSTime 解析短信时间字符串
func parseSMSTime(timeStr string) time.Time {
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
