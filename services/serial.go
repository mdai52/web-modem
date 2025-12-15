package services

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"
	"path/filepath"

	"github.com/tarm/serial"
	"modem-manager/models"
	"modem-manager/utils"
)

type SerialService struct {
	port       *serial.Port
	connected  bool
	portName   string
	mu         sync. Mutex
	listeners  []chan string
	longSMSMap map[string]*models. LongSMS // 长短信缓存
}

var (
	instance *SerialService
	once     sync.Once
)

func GetSerialService() *SerialService {
	once.Do(func() {
		instance = &SerialService{
			connected:  false,
			listeners:  make([]chan string, 0),
			longSMSMap:  make(map[string]*models. LongSMS),
		}
	})
	return instance
}

// 列出可用串口
func (s *SerialService) ListPorts() ([]models.SerialPort, error) {
	var ports []string

	// 通过通配符列出常见串口设备
	usbPorts, _ := filepath.Glob("/dev/ttyUSB*")
	acmPorts, _ := filepath.Glob("/dev/ttyACM*")

	ports = append(ports, usbPorts...)
	ports = append(ports, acmPorts...)

	var serialPorts []models.SerialPort
	for _, port := range ports {
		serialPorts = append(serialPorts, models.SerialPort{
			Name: port,
			Path: port,
		})
	}

	return serialPorts, nil
}

// 连接到 Modem
func (s *SerialService) Connect(portName string, baudRate int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connected {
		return errors.New("already connected")
	}

	config := &serial.Config{
		Name:        portName,
		Baud:        baudRate,
		ReadTimeout: time.Second * 5,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
	}

	port, err := serial.OpenPort(config)
	if err != nil {
		return err
	}

	// 暂存端口信息用于探测
	s.port = port
	s.portName = portName

	// 探测是否为调制解调器
	if err := s.ProbeModem(); err != nil {
		_ = s.port.Close()
		s.port = nil
		s.portName = ""
		return err
	}

	// 探测通过后再标记连接状态并开启读取循环
	s.connected = true

	go s.readLoop()

	// 初始化 modem
	s.sendCommand("ATZ\r\n")
	time.Sleep(500 * time. Millisecond)
	s.sendCommand("ATE0\r\n")
	time.Sleep(200 * time.Millisecond)
	
	// 设置 PDU 模式（支持所有字符集）
	s.sendCommand("AT+CMGF=0\r\n")
	time.Sleep(200 * time.Millisecond)
	
	// 设置字符集为 UCS2
	s.sendCommand("AT+CSCS=\"UCS2\"\r\n")
	time.Sleep(200 * time.Millisecond)

	return nil
}

func (s *SerialService) IsConnected() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.connected
}

// 断开连接
func (s *SerialService) Disconnect() error {
	s.mu. Lock()
	defer s.mu.Unlock()

	if !s.connected {
		return errors.New("not connected")
	}

	err := s.port.Close()
	if err != nil {
		return err
	}

	s. connected = false
	s.portName = ""
	return nil
}

// 探测端口是否为支持 AT 的 modem
func (s *SerialService) ProbeModem() error {
	if s.port == nil {
		return errors.New("port not initialized")
	}

	// 直接对底层端口写入 AT 并读取响应，避免依赖 connected 状态
	if _, err := s.port.Write([]byte("AT\r\n")); err != nil {
		return fmt.Errorf("write AT failed: %v", err)
	}

	buf := make([]byte, 128)
	response := ""
	timeout := time.After(2 * time.Second)

	for {
		select {
		case <-timeout:
			if strings.Contains(response, "OK") {
				return nil
			}
			return errors.New("AT probe failed: timeout")
		default:
			n, err := s.port.Read(buf)
			if err != nil && err.Error() != "EOF" {
				// 继续重试直到超时
				time.Sleep(50 * time.Millisecond)
				continue
			}
			if n > 0 {
				response += string(buf[:n])
				if strings.Contains(response, "OK") {
					return nil
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// 发送 AT 命令
func (s *SerialService) SendATCommand(command string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected {
		return "", errors.New("not connected")
	}

	return s.sendCommand(command + "\r\n")
}

// 内部发送命令方法
func (s *SerialService) sendCommand(command string) (string, error) {
	_, err := s.port.Write([]byte(command))
	if err != nil {
		return "", err
	}

	response := ""
	buf := make([]byte, 128)
	timeout := time.After(5 * time.Second)
	startTime := time.Now()

	for {
		select {
		case <-timeout:
			if response == "" {
				return response, errors.New("timeout")
			}
			return response, nil
		default:
			s.port. Flush()
			n, err := s.port.Read(buf)
			if err != nil && err. Error() != "EOF" {
				if time.Since(startTime) > 5*time.Second {
					return response, errors.New("timeout")
				}
				time.Sleep(50 * time.Millisecond)
				continue
			}
			
			if n > 0 {
				response += string(buf[:n])
				if strings.Contains(response, "OK") || strings.Contains(response, "ERROR") {
					return response, nil
				}
				// PDU 模式下的提示符
				if strings.Contains(response, ">") {
					return response, nil
				}
			}
			
			time.Sleep(50 * time. Millisecond)
		}
	}
}

// 读取循环
func (s *SerialService) readLoop() {
	buf := make([]byte, 128)
	for s.connected {
		n, err := s.port.Read(buf)
		if err != nil {
			if s.connected {
				log. Println("Read error:", err)
			}
			time.Sleep(100 * time.Millisecond)
			continue
		}
		
		if n > 0 {
			line := string(buf[:n])
			log. Println("Received:", line)
			s.broadcast(line)
		}
	}
}

// 广播消息
func (s *SerialService) broadcast(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, listener := range s.listeners {
		select {
		case listener <- message:
		default: 
		}
	}
}

// 添加监听器
func (s *SerialService) AddListener(ch chan string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, ch)
}

// 获取 Modem 信息
func (s *SerialService) GetModemInfo() (*models.ModemInfo, error) {
	if !s.connected {
		return nil, errors.New("not connected")
	}

	info := &models.ModemInfo{
		Port:      s.portName,
		Connected: s.connected,
	}

	if resp, err := s.SendATCommand("AT+CGMI"); err == nil {
		info. Manufacturer = extractValue(resp)
	}

	if resp, err := s. SendATCommand("AT+CGMM"); err == nil {
		info.Model = extractValue(resp)
	}

	if resp, err := s.SendATCommand("AT+CGSN"); err == nil {
		info.IMEI = extractValue(resp)
	}

	if resp, err := s.SendATCommand("AT+CIMI"); err == nil {
		info.IMSI = extractValue(resp)
	}

	if resp, err := s.SendATCommand("AT+CNUM"); err == nil {
		info.PhoneNumber = extractPhoneNumber(resp)
	}

	if resp, err := s.SendATCommand("AT+COPS?"); err == nil {
		info.Operator = extractOperator(resp)
	}

	return info, nil
}

// 获取信号强度
func (s *SerialService) GetSignalStrength() (*models.SignalStrength, error) {
	if !s.connected {
		return nil, errors.New("not connected")
	}

	resp, err := s.SendATCommand("AT+CSQ")
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`\+CSQ:\s*(\d+),(\d+)`)
	matches := re.FindStringSubmatch(resp)

	if len(matches) < 3 {
		return nil, errors.New("invalid response")
	}

	rssi := 0
	fmt.Sscanf(matches[1], "%d", &rssi)

	quality := 0
	fmt.Sscanf(matches[2], "%d", &quality)

	dbm := fmt.Sprintf("%d dBm", -113+rssi*2)

	return &models.SignalStrength{
		RSSI:    rssi,
		Quality: quality,
		DBM:     dbm,
	}, nil
}

// 列出短信（PDU 模式）
func (s *SerialService) ListSMS() ([]models.SMS, error) {
	if !s.connected {
		return nil, errors.New("not connected")
	}

	// 使用 PDU 模式读取所有短信
	resp, err := s.SendATCommand("AT+CMGL=4") // 4 = 所有短信
	if err != nil {
		return nil, err
	}

	return s.parsePDUSMSList(resp), nil
}

// 解析 PDU 模式的短信列表
func (s *SerialService) parsePDUSMSList(response string) []models.SMS {
	smsList := []models.SMS{}
	lines := strings.Split(response, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "+CMGL:  ") {
			// 解析短信头
			re := regexp.MustCompile(`\+CMGL:\s*(\d+),(\d+),.*,(\d+)`)
			matches := re.FindStringSubmatch(line)

			if len(matches) >= 3 && i+1 < len(lines) {
				index := 0
				fmt.Sscanf(matches[1], "%d", &index)

				// 下一行是 PDU 数据
				pduData := strings.TrimSpace(lines[i+1])
				
				// 解析 PDU
				phone, message, timestamp, err := utils.ParsePDUMessage(pduData)
				if err != nil {
					log. Println("PDU parse error:", err)
					continue
				}

				sms := models.SMS{
					Index:   index,
					Status:  "READ",
					Number:  phone,
					Time:    timestamp,
					Message:  message,
				}
				smsList = append(smsList, sms)
				i++ // 跳过 PDU 数据行
			}
		}
	}

	return smsList
}

// 发送短信（支持中文和长短信）
func (s *SerialService) SendSMS(number, message string) error {
	if !s.connected {
		return errors.New("not connected")
	}

	// 创建 PDU 消息（自动处理长短信）
	pdus := utils.CreatePDUMessage(number, message)
	
	log.Printf("Sending SMS in %d part(s)", len(pdus))

	// 发送每个 PDU 片段
	for i, pdu := range pdus {
		log.Printf("Sending part %d/%d", i+1, len(pdus))
		
		// PDU 长度（不包括 SMSC）
		pduLen := (len(pdu) - 2) / 2
		
		// 发送 AT+CMGS 命令
		cmd := fmt.Sprintf("AT+CMGS=%d", pduLen)
		resp, err := s.SendATCommand(cmd)
		if err != nil {
			return fmt.Errorf("failed to initiate SMS: %v", err)
		}
		
		// 等待 > 提示符
		if ! strings.Contains(resp, ">") {
			return errors.New("modem did not respond with prompt")
		}
		
		time.Sleep(200 * time.Millisecond)
		
		// 发送 PDU + Ctrl+Z
		_, err = s.sendCommand(pdu + "\x1A")
		if err != nil {
			return fmt.Errorf("failed to send PDU: %v", err)
		}
		
		// 等待发送完成
		time.Sleep(2 * time.Second)
	}

	log. Println("SMS sent successfully")
	return nil
}

// 辅助函数
func extractValue(response string) string {
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && line != "OK" && ! strings.HasPrefix(line, "AT") {
			return line
		}
	}
	return ""
}

func extractOperator(response string) string {
	re := regexp.MustCompile(`"([^"]+)"`)
	matches := re.FindStringSubmatch(response)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// 提取 AT+CNUM 返回的手机号
func extractPhoneNumber(response string) string {
	re := regexp.MustCompile(`\+CNUM:\s*"[^"]*",\s*"([^"]+)"`)
	matches := re.FindStringSubmatch(response)
	if len(matches) > 1 {
		return matches[1]
	}

	// 回退：寻找包含数字的行
	for _, line := range strings.Split(response, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "AT") || line == "OK" {
			continue
		}
		if strings.ContainsAny(line, "0123456789") {
			return line
		}
	}

	return ""
}