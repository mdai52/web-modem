package service

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rehiy/modem/at"
	"github.com/tarm/serial"
)

var (
	modemOnce     sync.Once
	modemInstance *ModemService
	ModemEvent    = make(chan string, 100)
)

// ModemInfo 端口信息
type ModemInfo struct {
	Name        string `json:"name"`
	PhoneNumber string `json:"phoneNumber"`
	Connected   bool   `json:"connected"`
	*at.Device  `json:"-"`
}

// ModemService 管理多个串口连接
type ModemService struct {
	pool map[string]*ModemInfo
	mu   sync.Mutex
}

// GetModemService 返回单例实例
func GetModemService() *ModemService {
	modemOnce.Do(func() {
		modemInstance = &ModemService{
			pool: map[string]*ModemInfo{},
		}
	})
	return modemInstance
}

// GetModems 返回已连接的端口信息
func (m *ModemService) GetModems() []*ModemInfo {
	m.mu.Lock()
	defer m.mu.Unlock()

	var modems []*ModemInfo
	for _, model := range m.pool {
		modems = append(modems, model)
	}
	return modems
}

// ScanModems 扫描可用的调制解调器并连接到它们
func (m *ModemService) ScanModems(devs ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 环境变量
	if len(devs) == 0 {
		port := os.Getenv("MODEM_PORT")
		if port != "" {
			devs = strings.Split(port, ",")
		}
	}

	// 查找潜在设备
	switch runtime.GOOS {
	case "windows":
		if len(devs) == 0 {
			devs = []string{"COM1", "COM2", "COM3", "COM4", "COM5"}
		}
	default:
		if len(devs) == 0 {
			devs = []string{"/dev/ttyUSB*", "/dev/ttyACM*"}
		}
		pps := []string{}
		for _, p := range devs {
			matches, _ := filepath.Glob(p)
			pps = append(pps, matches...)
		}
		devs = pps
	}

	// 尝试连接到新设备
	for _, u := range devs {
		m.makeConnect(u)
	}
}

// GetConnect 返回给定端口名称的 AT 接口
func (m *ModemService) GetConnect(u string) (*ModemInfo, error) {
	n := path.Base(u)

	m.mu.Lock()
	defer m.mu.Unlock()

	modem, ok := m.pool[n]
	if !ok {
		return nil, fmt.Errorf("[%s] not found", n)
	}
	return modem, nil
}

// handleIncomingSMS 处理指定端口的新接收短信
func (m *ModemService) handleIncomingSMS(portName string, smsIndex int, webhookService *WebhookService) {
	conn, err := m.GetConnect(portName)
	if err != nil {
		log.Printf("[%s] Failed to get connection for incoming SMS: %v", portName, err)
		return
	}

	// 获取短信列表（只获取新短信）
	smsList, err := conn.ListSMSPdu(4)
	if err != nil {
		log.Printf("[%s] Failed to list SMS: %v", portName, err)
		return
	}

	// 处理每条短信
	for _, sms := range smsList {
		hasNewSMS := false
		for _, idx := range sms.Indices {
			if idx == smsIndex {
				hasNewSMS = true
				break
			}
		}
		if !hasNewSMS {
			continue
		}
		go func(smsData at.SMS) {
			modelSMS := atSMSToModelSMS(smsData, conn.PhoneNumber)
			if err := webhookService.HandleIncomingSMS(modelSMS); err != nil {
				log.Printf("[%s] Failed to handle incoming SMS: %v", portName, err)
			}
			log.Printf("[%s] New SMS from %s: %s", portName, smsData.PhoneNumber, smsData.Text)
		}(sms)
	}
}

// makeConnect 添加新的 AT 接口
func (m *ModemService) makeConnect(u string) error {
	n := path.Base(u)

	// 创建日志函数
	pf := func(s string, v ...any) {
		log.Printf(fmt.Sprintf("[%s] %s", n, s), v...)
	}

	// 检查是否已连接
	if conn, ok := m.pool[n]; ok {
		if conn.Test() == nil {
			pf("already connected")
			return nil
		}
		conn.Close()
		delete(m.pool, n)
	}

	// 创建事件处理函数，写入 ModemEvent 并处理短信
	hf := func(l string, p map[int]string) {
		ModemEvent <- fmt.Sprintf("[%s] urc:%s %v", n, l, p)
		// 处理收到的短信通知
		if l == "+CMTI" && len(p) > 0 {
			if indexStr, ok := p[1]; ok {
				if index, err := strconv.Atoi(indexStr); err == nil {
					w := NewWebhookService()
					m.handleIncomingSMS(n, index, w)
				}
			}
		}
	}

	// 打开串口
	pf("connecting")
	port, err := serial.OpenPort(&serial.Config{
		Name:        u,      // 串口完整路径
		Baud:        115200, // 波特率
		ReadTimeout: 1 * time.Second,
	})
	if err != nil {
		pf("connect failed: %v", err)
		return err
	}

	// 创建新的连接
	conn := at.New(port, hf, &at.Config{Printf: pf})
	if err := conn.Test(); err != nil {
		pf("at test failed: %v", err)
		conn.Close()
		return err
	}

	// 设置默认参数
	conn.EchoOff()     // 关闭回显
	conn.SetSMSMode(0) // PDU 模式

	// 添加到连接池
	modem := &ModemInfo{
		Name:        n,
		PhoneNumber: "unkown",
		Connected:   true,
		Device:      conn,
	}

	// 获取手机号，用于接收号码
	if phoneNum, _, err := modem.GetPhoneNumber(); err == nil {
		modem.PhoneNumber = phoneNum
	}

	// 获取并显示手机号
	pf("connected, phone number: %s", modem.PhoneNumber)
	m.pool[n] = modem

	return nil
}
