package services

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/tarm/serial"

	"modem-manager/models"
	"modem-manager/utils"
)

type SerialService struct {
	ports      map[string]*serial.Port // 所有已连接可用端口
	portLocks  map[string]*sync.Mutex  // 每个端口的读写锁，避免命令与监听竞争
	mu         sync.Mutex
	listeners  []chan string
	longSMSMap map[string]*models.LongSMS // 长短信缓存
}

var (
	instance *SerialService
	once     sync.Once
)

func GetSerialService() *SerialService {
	once.Do(func() {
		instance = &SerialService{
			listeners:  make([]chan string, 0),
			longSMSMap: make(map[string]*models.LongSMS),
			ports:      make(map[string]*serial.Port),
			portLocks:  make(map[string]*sync.Mutex),
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
		_, connected := s.ports[port]
		serialPorts = append(serialPorts, models.SerialPort{
			Name:      port,
			Path:      port,
			Connected: connected,
		})
	}

	return serialPorts, nil
}

// 扫描并连接所有可用串口（探测支持 AT 的 modem）
func (s *SerialService) ScanAndConnectAll(baudRate int) ([]models.SerialPort, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 列出候选端口
	usbPorts, _ := filepath.Glob("/dev/ttyUSB*")
	acmPorts, _ := filepath.Glob("/dev/ttyACM*")
	candidates := append(usbPorts, acmPorts...)

	for _, p := range candidates {
		if _, ok := s.ports[p]; ok {
			continue // 已连接
		}

		cfg := &serial.Config{Name: p, Baud: baudRate, ReadTimeout: 2 * time.Second, Size: 8, Parity: serial.ParityNone, StopBits: serial.Stop1}
		sp, err := serial.OpenPort(cfg)
		if err != nil {
			continue
		}

		// 快速探测
		if _, err := sp.Write([]byte("AT\r\n")); err != nil {
			_ = sp.Close()
			continue
		}
		buf := make([]byte, 128)
		resp := ""
		deadline := time.Now().Add(1 * time.Second)
		for time.Now().Before(deadline) {
			n, _ := sp.Read(buf)
			if n > 0 {
				resp += string(buf[:n])
				if strings.Contains(resp, "OK") {
					break
				}
			}
			time.Sleep(30 * time.Millisecond)
		}
		if !strings.Contains(resp, "OK") {
			_ = sp.Close()
			continue
		}

		// 视为可用，加入池并启动读取循环
		s.ports[p] = sp
		if _, ok := s.portLocks[p]; !ok {
			s.portLocks[p] = &sync.Mutex{}
		}
		go s.readLoopFor(p, sp)

		// 基本初始化
		sp.Write([]byte("ATE0\r\n"))
		time.Sleep(100 * time.Millisecond)
		sp.Write([]byte("AT+CMGF=0\r\n"))
	}

	// 仅返回已连接/可用端口，并排序
	names := make([]string, 0, len(s.ports))
	for name := range s.ports {
		names = append(names, name)
	}
	sort.Strings(names)
	var result []models.SerialPort
	for _, name := range names {
		result = append(result, models.SerialPort{Name: name, Path: name, Connected: true})
	}
	return result, nil
}

// 发送 AT 命令（指定端口），使用端口级锁避免与监听竞争
func (s *SerialService) SendATCommand(portName, command string) (string, error) {
	s.mu.Lock()
	port, ok := s.ports[portName]
	lock := s.portLocks[portName]
	s.mu.Unlock()
	if !ok {
		return "", fmt.Errorf("port not connected: %s", portName)
	}
	lock.Lock()
	defer lock.Unlock()
	return s.sendCommandPort(port, command+"\r\n")
}

// 内部发送命令方法
func (s *SerialService) sendCommandPort(port *serial.Port, command string) (string, error) {
	_, err := port.Write([]byte(command))
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
			port.Flush()
			n, err := port.Read(buf)
			if err != nil && err.Error() != "EOF" {
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

			time.Sleep(50 * time.Millisecond)
		}
	}
}

// 针对指定端口的读取循环
func (s *SerialService) readLoopFor(name string, port *serial.Port) {
	buf := make([]byte, 128)
	for {
		s.mu.Lock()
		lock := s.portLocks[name]
		s.mu.Unlock()
		if lock == nil {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		lock.Lock()
		n, err := port.Read(buf)
		lock.Unlock()
		if err != nil {
			time.Sleep(150 * time.Millisecond)
			continue
		}
		if n > 0 {
			msg := "[" + name + "] " + string(buf[:n])
			s.broadcast(msg)
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

// 移除监听器，防止 WebSocket 断开后泄漏
func (s *SerialService) RemoveListener(ch chan string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, listener := range s.listeners {
		if listener == ch {
			close(listener)
			s.listeners = append(s.listeners[:i], s.listeners[i+1:]...)
			break
		}
	}
}

// 获取 Modem 信息
func (s *SerialService) GetModemInfo(portName string) (*models.ModemInfo, error) {
	s.mu.Lock()
	_, ok := s.ports[portName]
	s.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("port not connected: %s", portName)
	}

	info := &models.ModemInfo{
		Port:      portName,
		Connected: true,
	}

	if resp, err := s.SendATCommand(portName, "AT+CGMI"); err == nil {
		info.Manufacturer = extractValue(resp)
	}

	if resp, err := s.SendATCommand(portName, "AT+CGMM"); err == nil {
		info.Model = extractValue(resp)
	}

	if resp, err := s.SendATCommand(portName, "AT+CGSN"); err == nil {
		info.IMEI = extractValue(resp)
	}

	if resp, err := s.SendATCommand(portName, "AT+CIMI"); err == nil {
		info.IMSI = extractValue(resp)
	}

	if resp, err := s.SendATCommand(portName, "AT+CNUM"); err == nil {
		info.PhoneNumber = extractPhoneNumber(resp)
	}

	if resp, err := s.SendATCommand(portName, "AT+COPS?"); err == nil {
		info.Operator = extractOperator(resp)
	}

	return info, nil
}

// 获取信号强度
func (s *SerialService) GetSignalStrength(portName string) (*models.SignalStrength, error) {
	resp, err := s.SendATCommand(portName, "AT+CSQ")
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
func (s *SerialService) ListSMS(portName string) ([]models.SMS, error) {
	// 使用 PDU 模式读取所有短信
	resp, err := s.SendATCommand(portName, "AT+CMGL=4") // 4 = 所有短信
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
					log.Println("PDU parse error:", err)
					continue
				}

				sms := models.SMS{
					Index:   index,
					Status:  "READ",
					Number:  phone,
					Time:    timestamp,
					Message: message,
				}
				smsList = append(smsList, sms)
				i++ // 跳过 PDU 数据行
			}
		}
	}

	return smsList
}

// 发送短信（支持中文和长短信）
func (s *SerialService) SendSMS(portName, number, message string) error {
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
		resp, err := s.SendATCommand(portName, cmd)
		if err != nil {
			return fmt.Errorf("failed to initiate SMS: %v", err)
		}

		// 等待 > 提示符
		if !strings.Contains(resp, ">") {
			return errors.New("modem did not respond with prompt")
		}

		time.Sleep(200 * time.Millisecond)

		// 发送 PDU + Ctrl+Z
		s.mu.Lock()
		port := s.ports[portName]
		lock := s.portLocks[portName]
		s.mu.Unlock()
		if port == nil || lock == nil {
			return fmt.Errorf("port not connected: %s", portName)
		}
		lock.Lock()
		_, err = s.sendCommandPort(port, pdu+"\x1A")
		lock.Unlock()
		if err != nil {
			return fmt.Errorf("failed to send PDU: %v", err)
		}

		// 等待发送完成
		time.Sleep(2 * time.Second)
	}

	log.Println("SMS sent successfully")
	return nil
}

// 辅助函数
func extractValue(response string) string {
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && line != "OK" && !strings.HasPrefix(line, "AT") {
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
