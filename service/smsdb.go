package service

import (
	"fmt"
	"log"

	"github.com/rehiy/web-modem/database"
)

// SmsdbService 短信数据库服务
type SmsdbService struct {
	modemService *ModemService
}

// NewSmsdbService 创建短信数据库服务
func NewSmsdbService() *SmsdbService {
	return &SmsdbService{
		modemService: GetModemService(),
	}
}

// SyncSMSToDB 从指定Modem同步所有短信到数据库
func (s *SmsdbService) SyncSMSToDB(modemName string) (map[string]interface{}, error) {
	// 获取连接
	conn, err := s.modemService.GetConnect(modemName)
	if err != nil {
		return nil, fmt.Errorf("获取连接失败: %v", err)
	}

	// 列出所有短信（stat=4 表示所有短信）
	smsList, err := conn.ListSMSPdu(4)
	if err != nil {
		return nil, fmt.Errorf("读取短信失败: %v", err)
	}

	totalCount := len(smsList)
	newCount := 0

	// 同步每条短信
	for _, smsData := range smsList {
		// 转换为数据库模型
		modelSMS := atSMSToModelSMS(smsData, conn.PhoneNumber, modemName)

		// 检查是否已存在
		if res, err := database.GetSMSByIDs(smsData.Indices); err == nil && len(res) > 0 {
			log.Printf("[%s] SMS already exists in database, skipping: %s", modemName, res[0].SMSIDs)
			continue
		}

		// 保存到数据库
		if err := database.SaveSMS(modelSMS); err != nil {
			log.Printf("[%s] Failed to save SMS to database: %v", modemName, err)
			continue
		}

		newCount++
		log.Printf("[%s] Synced SMS from %s to database: %s", modemName, smsData.PhoneNumber, smsData.Text)
	}

	return map[string]interface{}{
		"modemName":  modemName,
		"totalCount": totalCount,
		"newCount":   newCount,
	}, nil
}
