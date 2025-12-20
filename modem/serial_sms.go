package modem

import (
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xlab/at/pdu"
	"github.com/xlab/at/sms"
)

// ListSMS 获取短信列表。
func (s *SerialService) ListSMS() ([]SMS, error) {
	resp, err := s.SendATCommand("AT+CMGL=4")
	if err != nil {
		return nil, err
	}

	var parts []struct {
		SMS
		ref, total, seq int
	}

	// 按 +CMGL: 分割以处理多条消息，跳过第一个空部分
	chunks := strings.Split(resp, "+CMGL: ")
	for _, chunk := range chunks[1:] {
		lines := strings.SplitN(chunk, "\n", 2)
		if len(lines) < 2 {
			continue
		}

		// 解析元数据: index,stat,,length
		fields := strings.Split(strings.TrimSpace(lines[0]), ",")
		if len(fields) < 2 {
			continue
		}

		idx, _ := strconv.Atoi(strings.TrimSpace(fields[0]))
		stat, _ := strconv.Atoi(strings.TrimSpace(fields[1]))
		pduHex := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(lines[1]), "OK"))

		var sender, timestamp, message string
		ref, total, seq := 0, 1, 1
		if raw, decErr := hex.DecodeString(pduHex); decErr == nil {
			var msg sms.Message
			if _, rdErr := msg.ReadFrom(raw); rdErr == nil {
				sender = string(msg.Address)
				timestamp = time.Time(msg.ServiceCenterTime).Format("2006/01/02 15:04:05")
				message = msg.Text
				if msg.UserDataHeader.TotalNumber > 0 && msg.UserDataHeader.Sequence > 0 {
					total = msg.UserDataHeader.TotalNumber
					seq = msg.UserDataHeader.Sequence
					ref = msg.UserDataHeader.Tag
				}
			} else {
				message = "PDU Decode Error: " + rdErr.Error() + " Raw: " + pduHex
			}
		} else {
			message = "PDU Hex Decode Error: " + decErr.Error() + " Raw: " + pduHex
		}

		parts = append(parts, struct {
			SMS
			ref, total, seq int
		}{
			SMS: SMS{
				Index:   idx,
				Status:  getPDUStatus(stat),
				Number:  sender,
				Time:    timestamp,
				Message: message,
			},
			ref: ref, total: total, seq: seq,
		})
	}

	// 合并长短信
	var result []SMS
	merged := make(map[string][]struct {
		seq int
		msg string
	})

	for _, p := range parts {
		if p.total <= 1 {
			result = append(result, p.SMS)
			continue
		}
		key := fmt.Sprintf("%s_%d", p.Number, p.ref)
		merged[key] = append(merged[key], struct {
			seq int
			msg string
		}{p.seq, p.Message})
	}

	for key, fragments := range merged {
		sort.Slice(fragments, func(i, j int) bool { return fragments[i].seq < fragments[j].seq })
		fullMsg := ""
		for _, f := range fragments {
			fullMsg += f.msg
		}

		// 从部分中查找原始元数据（效率低但简单）
		for _, p := range parts {
			if fmt.Sprintf("%s_%d", p.Number, p.ref) == key && p.seq == 1 {
				p.SMS.Message = fullMsg
				result = append(result, p.SMS)
				break
			}
		}
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Index < result[j].Index })
	return result, nil
}

// SendSMS 发送短信。
func (s *SerialService) SendSMS(number, message string) error {
	msg := sms.Message{
		Type:    sms.MessageTypes.Submit,
		Address: sms.PhoneNumber(number),
		Text:    message,
	}
	if pdu.Is7BitEncodable(message) {
		msg.Encoding = sms.Encodings.Gsm7Bit
	} else {
		msg.Encoding = sms.Encodings.UCS2
	}

	length, octets, err := msg.PDU()
	if err != nil {
		return err
	}

	_, err = s.SendATCommand(fmt.Sprintf("AT+CMGS=%d", length))
	if err != nil {
		return err
	}

	pduHex := strings.ToUpper(hex.EncodeToString(octets))
	_, err = s.sendRawCommand(pduHex, "\x1A", 60*time.Second)
	return err
}

// DeleteSMS 删除指定索引的短信。
func (s *SerialService) DeleteSMS(index int) error {
	_, err := s.SendATCommand(fmt.Sprintf("AT+CMGD=%d", index))
	return err
}

// 辅助函数

func getPDUStatus(stat int) string {
	switch stat {
	case 0:
		return "REC UNREAD"
	case 1:
		return "REC READ"
	case 2:
		return "STO UNSENT"
	case 3:
		return "STO SENT"
	default:
		return "UNKNOWN"
	}
}
