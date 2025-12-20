package modem

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/tarm/serial"
	"github.com/xlab/at/pdu"
)

const (
	bufferSize  = 256
	errorSleep  = 500 * time.Millisecond
	readTimeout = 500 * time.Millisecond
)

// SerialService 封装了单个串口的读取、写入和监控。
type SerialService struct {
	name      string
	port      *serial.Port
	broadcast func(string)
	sync.Mutex
}

// NewSerialService 尝试连接并初始化串口服务。
func NewSerialService(name string, Baud int, broadcast func(string)) (*SerialService, error) {
	port, err := serial.OpenPort(&serial.Config{
		Name: name, Baud: Baud, ReadTimeout: readTimeout,
	})
	if err != nil {
		return nil, err
	}

	s := &SerialService{name: name, port: port, broadcast: broadcast}
	if err := s.check(); err != nil {
		port.Close()
		return nil, err
	}
	return s, nil
}

// check 发送基本的 AT 命令以验证连接。
func (s *SerialService) check() error {
	resp, err := s.SendATCommand("AT")
	if err != nil {
		return err
	}
	if !strings.Contains(resp, "OK") {
		return fmt.Errorf("command AT failed: %s", resp)
	}
	return nil
}

// Start 开始串口服务读取循环。
func (s *SerialService) Start() {
	s.SendATCommand("ATE0")      // 关闭回显
	s.SendATCommand("AT+CMGF=0") // 短信格式
	go s.readLoop()
}

// readLoop 持续读取串口输出并广播它。
func (s *SerialService) readLoop() {
	buf := make([]byte, bufferSize)
	for {
		s.Lock()
		n, err := s.port.Read(buf)
		s.Unlock()

		if n > 0 && s.broadcast != nil {
			s.broadcast(fmt.Sprintf("[%s] %s", s.name, string(buf[:n])))
		}

		if err != nil {
			time.Sleep(errorSleep)
		}
	}
}

// SendATCommand 发送 AT 命令并读取响应。
func (s *SerialService) SendATCommand(command string) (string, error) {
	return s.sendRawCommand(command, "\r\n", 1*time.Second)
}

// sendRawCommand 发送原始命令并读取响应。
func (s *SerialService) sendRawCommand(command, suffix string, timeout time.Duration) (string, error) {
	s.Lock()
	defer s.Unlock()

	_ = s.port.Flush()
	if _, err := s.port.Write([]byte(command + suffix)); err != nil {
		return "", err
	}

	var (
		buf   = make([]byte, bufferSize)
		resp  = strings.Builder{}
		start = time.Now()
	)

	for {
		if time.Since(start) > timeout {
			return "", errors.New("command timeout")
		}

		n, err := s.port.Read(buf)
		if n > 0 {
			resp.Write(buf[:n])
			str := resp.String()
			if strings.Contains(str, "OK") || strings.Contains(str, "ERROR") || strings.Contains(str, ">") {
				return strings.TrimSpace(str), nil
			}
		}
		if err != nil {
			if err == io.EOF {
				time.Sleep(errorSleep)
				continue
			}
			if resp.Len() > 0 {
				return strings.TrimSpace(resp.String()), nil
			}
			return "", err
		}
	}
}

// GetModemInfo 获取有关当前端口的基本信息。
func (s *SerialService) GetModemInfo() (*ModemInfo, error) {
	info := &ModemInfo{Port: s.name, Connected: true}
	cmds := map[*string]string{
		&info.Manufacturer: "AT+CGMI",
		&info.Model:        "AT+CGMM",
		&info.IMEI:         "AT+CGSN",
		&info.IMSI:         "AT+CIMI",
	}

	for ptr, cmd := range cmds {
		if resp, err := s.SendATCommand(cmd); err == nil {
			*ptr = extractValue(resp)
		}
	}

	info.Operator, _ = s.getOperator()
	info.PhoneNumber, _ = s.GetPhoneNumber()
	return info, nil
}

// getOperator 查询当前运营商名称。
func (s *SerialService) getOperator() (string, error) {
	resp, err := s.SendATCommand("AT+COPS?")
	if err != nil {
		return "", err
	}

	if m := regexp.MustCompile(`"([^"]+)"`).FindStringSubmatch(resp); len(m) > 1 {
		return m[1], nil
	}
	return "", errors.New("not found")
}

// GetPhoneNumber 查询电话号码。
func (s *SerialService) GetPhoneNumber() (string, error) {
	resp, err := s.SendATCommand("AT+CNUM")
	if err != nil {
		return "", err
	}

	if m := regexp.MustCompile(`\+CNUM:.*,"([^"]+)"`).FindStringSubmatch(resp); len(m) > 1 {
		return DecodeUCS2Hex(m[1]), nil
	}
	return "", errors.New("not found")
}

// GetSignalStrength 查询信号强度。
func (s *SerialService) GetSignalStrength() (*SignalStrength, error) {
	resp, err := s.SendATCommand("AT+CSQ")
	if err != nil {
		return nil, err
	}

	var rssi, qual int = -1, -1
	if _, err := fmt.Sscanf(extractValue(resp), "+CSQ: %d,%d", &rssi, &qual); err != nil {
		return nil, err
	}

	dbm := "unknown"
	if rssi >= 0 && rssi <= 31 {
		dbm = fmt.Sprintf("%d dBm", -113+rssi*2)
	}

	return &SignalStrength{
		RSSI:    rssi,
		Quality: qual,
		DBM:     dbm,
	}, nil
}

// 辅助函数

func extractValue(response string) string {
	for _, line := range strings.Split(response, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && line != "OK" && !strings.HasPrefix(line, "AT") {
			return line
		}
	}
	return ""
}

func DecodeUCS2Hex(s string) string {
	b, err := hex.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return s
	}
	d, err := pdu.DecodeUcs2(b, false)
	if err != nil {
		return s
	}
	return d
}
