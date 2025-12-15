package utils

import (
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf16"
)

// SMS 编码类型
type SMSEncoding int

const (
	GSM7Bit SMSEncoding = iota
	UCS2
)

// 检测短信编码类型
func DetectEncoding(message string) SMSEncoding {
	// 检查是否包含非 GSM 7-bit 字符
	for _, r := range message {
		if r > 127 || !isGSM7BitChar(r) {
			return UCS2
		}
	}
	return GSM7Bit
}

// 判断是否为 GSM 7-bit 字符
func isGSM7BitChar(r rune) bool {
	gsmChars := "@£$¥èéùìòÇ\nØø\rÅåΔ_ΦΓΛΩΠΨΣΘΞÆæßÉ ! \"#¤%&'()*+,-./0123456789:;<=>?¡ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÑÜ§¿abcdefghijklmnopqrstuvwxyzäöñüà"
	for _, c := range gsmChars {
		if c == r {
			return true
		}
	}
	return false
}

// 将字符串转换为 UCS2 编码（UTF-16BE）
func StringToUCS2(text string) string {
	// 转换为 UTF-16 编码
	utf16Codes := utf16.Encode([]rune(text))

	// 转换为十六进制字符串
	hexStr := ""
	for _, code := range utf16Codes {
		hexStr += fmt.Sprintf("%04X", code)
	}

	return hexStr
}

// 将 UCS2 编码转换为字符串
func UCS2ToString(hexStr string) (string, error) {
	// 移除空格
	hexStr = strings.ReplaceAll(hexStr, " ", "")

	// 解码十六进制
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", err
	}

	// 转换为 UTF-16 codes
	utf16Codes := make([]uint16, len(bytes)/2)
	for i := 0; i < len(utf16Codes); i++ {
		utf16Codes[i] = uint16(bytes[i*2])<<8 | uint16(bytes[i*2+1])
	}

	// 解码 UTF-16
	runes := utf16.Decode(utf16Codes)
	return string(runes), nil
}

// PDU 编码电话号码
func EncodePDUPhone(phone string) string {
	// 移除 + 号
	phone = strings.TrimPrefix(phone, "+")

	// 如果长度为奇数，在末尾添加 F
	if len(phone)%2 != 0 {
		phone += "F"
	}

	// 交换每对数字
	result := ""
	for i := 0; i < len(phone); i += 2 {
		result += string(phone[i+1]) + string(phone[i])
	}

	return result
}

// 解码 PDU 电话号码
func DecodePDUPhone(pduPhone string) string {
	result := ""
	for i := 0; i < len(pduPhone); i += 2 {
		if i+1 < len(pduPhone) {
			result += string(pduPhone[i+1]) + string(pduPhone[i])
		}
	}
	// 移除 F
	result = strings.ReplaceAll(result, "F", "")
	return result
}

// PDU 时间戳编码
func EncodePDUTimestamp() string {
	// 简化版本，返回空字符串，让 modem 自动处理
	return ""
}

// 解码 PDU 时间戳
func DecodePDUTimestamp(pduTime string) string {
	if len(pduTime) < 14 {
		return ""
	}

	// 交换每对数字
	year := string(pduTime[1]) + string(pduTime[0])
	month := string(pduTime[3]) + string(pduTime[2])
	day := string(pduTime[5]) + string(pduTime[4])
	hour := string(pduTime[7]) + string(pduTime[6])
	minute := string(pduTime[9]) + string(pduTime[8])
	second := string(pduTime[11]) + string(pduTime[10])

	return fmt.Sprintf("20%s-%s-%s %s:%s:%s", year, month, day, hour, minute, second)
}

// 创建 PDU 格式短信（支持长短信）
func CreatePDUMessage(phone, message string) []string {
	encoding := DetectEncoding(message)

	var pdus []string
	var msgLen int
	var encodedMsg string

	if encoding == UCS2 {
		encodedMsg = StringToUCS2(message)
		msgLen = len(encodedMsg) / 2 // UCS2 每个字符2字节

		// UCS2 编码：单条短信最多 70 字符，长短信每条 67 字符
		maxChars := 70
		if msgLen > maxChars {
			maxChars = 67 // 长短信需要 6 字节的 UDH
		}

		// 分割消息
		parts := splitUCS2Message(encodedMsg, maxChars)
		totalParts := len(parts)

		// 生成随机引用号（简化版本）
		refNum := byte(0x00)

		for i, part := range parts {
			pdu := buildPDU(phone, part, UCS2, totalParts, i+1, refNum)
			pdus = append(pdus, pdu)
		}
	} else {
		// GSM 7-bit 编码（简化处理，这里用文本模式）
		pdus = append(pdus, message)
	}

	return pdus
}

// 分割 UCS2 编码的消息
func splitUCS2Message(encodedMsg string, maxChars int) []string {
	var parts []string
	maxBytes := maxChars * 4 // 每个 UCS2 字符是 4 个十六进制字符

	for i := 0; i < len(encodedMsg); i += maxBytes {
		end := i + maxBytes
		if end > len(encodedMsg) {
			end = len(encodedMsg)
		}
		parts = append(parts, encodedMsg[i:end])
	}

	return parts
}

// 构建 PDU
func buildPDU(phone, encodedMsg string, encoding SMSEncoding, totalParts, partNum int, refNum byte) string {
	// SMSC（使用默认）
	smsc := "00"

	// PDU Type
	var pduType string
	if totalParts > 1 {
		pduType = "41" // 包含 UDH 的短信
	} else {
		pduType = "11" // 普通短信
	}

	// Message Reference
	msgRef := "00"

	// 目标号码长度
	phoneLen := fmt.Sprintf("%02X", len(strings.ReplaceAll(phone, "+", "")))

	// 号码类型（国际格式）
	phoneType := "91"
	if !strings.HasPrefix(phone, "+") {
		phoneType = "81" // 国内格式
	}

	// 编码号码
	encodedPhone := EncodePDUPhone(phone)

	// Protocol Identifier
	pid := "00"

	// Data Coding Scheme
	var dcs string
	if encoding == UCS2 {
		dcs = "08" // UCS2 编码
	} else {
		dcs = "00" // GSM 7-bit
	}

	// Validity Period（使用默认）
	vp := "AA" // 4 天

	// User Data
	var udh string
	var udl string

	if totalParts > 1 {
		// 长短信 UDH
		udh = fmt.Sprintf("05000300%02X%02X%02X", refNum, totalParts, partNum)
		udlValue := len(encodedMsg)/2 + 6 // 消息长度 + UDH 长度
		udl = fmt.Sprintf("%02X", udlValue)
	} else {
		udh = ""
		udl = fmt.Sprintf("%02X", len(encodedMsg)/2)
	}

	// 组合 PDU
	pdu := smsc + pduType + msgRef + phoneLen + phoneType + encodedPhone + pid + dcs + vp + udl + udh + encodedMsg

	return pdu
}

// 解析接收到的 PDU 短信
func ParsePDUMessage(pdu string) (phone, message, timestamp string, err error) {
	// 简化的 PDU 解析
	// 实际实现需要完整解析 PDU 格式

	// 跳过 SMSC
	smscLen := 0
	fmt.Sscanf(pdu[0:2], "%02X", &smscLen)
	offset := (smscLen + 1) * 2

	if offset >= len(pdu) {
		return "", "", "", fmt.Errorf("invalid PDU format")
	}

	// PDU Type
	pduType := pdu[offset : offset+2]
	offset += 2

	// 发送方号码长度
	phoneLen := 0
	fmt.Sscanf(pdu[offset:offset+2], "%02X", &phoneLen)
	offset += 2

	// 号码类型
	offset += 2

	// 解码号码
	phonePDU := pdu[offset : offset+phoneLen+phoneLen%2]
	phone = DecodePDUPhone(phonePDU)
	offset += phoneLen + phoneLen%2

	// PID
	offset += 2

	// DCS (编码方式)
	dcs := pdu[offset : offset+2]
	offset += 2

	// 时间戳
	timePDU := pdu[offset : offset+14]
	timestamp = DecodePDUTimestamp(timePDU)
	offset += 14

	// UDL (用户数据长度)
	udl := 0
	fmt.Sscanf(pdu[offset:offset+2], "%02X", &udl)
	offset += 2

	// 检查是否有 UDH
	hasUDH := false
	if pduType == "44" || pduType == "64" {
		hasUDH = true
	}

	var udhLen int
	if hasUDH {
		fmt.Sscanf(pdu[offset:offset+2], "%02X", &udhLen)
		offset += (udhLen + 1) * 2 // 跳过 UDH
	}

	// 用户数据
	userData := pdu[offset:]

	// 根据编码方式解码
	if dcs == "08" {
		// UCS2 编码
		message, err = UCS2ToString(userData)
	} else {
		// GSM 7-bit（简化处理）
		message = userData
	}

	return phone, message, timestamp, err
}
