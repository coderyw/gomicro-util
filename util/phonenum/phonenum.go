package phonenum

import (
	error_code "github.com/coderyw/gomicro-util/model/error-code"
	"regexp"
	"strings"
)

var (
	startsWithDigitRegex = regexp.MustCompile(`^\d`)
	onlyDigitsRegex      = regexp.MustCompile(`^\d*$`)
	// ErrInvalidPhoneNumber 当手机号校验失败时返回
	ErrInvalidPhoneNumber = error_code.INVALID_PHONE_NUMBER.New()
)

// PhoneNumber 代表一个经过验证的菲律宾手机号
// 只能通过 Parse 函数创建，以确保其内部状态的有效性
type PhoneNumber struct {
	// basePhone 存储核心的10位数字，例如："9171234567"
	basePhone string
}

// Parse 根据菲律宾的自定义规则，校验并解析一个原始手机号字符串为 PhoneNumber 对象
//
// Parameters:
//   - phone: 用于解析的原始手机号字符串
//
// Returns:
//   - *PhoneNumber: 如果解析成功，返回一个有效的 PhoneNumber 对象指针
//   - error: 如果手机号无效，则返回错误
func Parse(phone string) (*PhoneNumber, error) {
	// 去除两侧空格
	phone = strings.TrimSpace(phone)
	// 去除 "+" 前缀
	phone = strings.TrimPrefix(phone, "+")
	// 规则1：必须以数字开头
	if !startsWithDigitRegex.MatchString(phone) {
		return nil, ErrInvalidPhoneNumber
	}

	// 规则2：必须只包含数字
	if !onlyDigitsRegex.MatchString(phone) {
		return nil, ErrInvalidPhoneNumber
	}

	var newPhone string // 用于存储标准化后的、以 "0" 开头的手机号

	// 将不同格式的号码标准化为 "0" + 10位数字的格式
	if strings.HasPrefix(phone, "63") {
		if len(phone) != 12 {
			return nil, ErrInvalidPhoneNumber
		}
		newPhone = "0" + phone[2:]
	} else if strings.HasPrefix(phone, "0") {
		if len(phone) != 11 {
			return nil, ErrInvalidPhoneNumber
		}
		newPhone = phone
	} else {
		if len(phone) != 10 {
			return nil, ErrInvalidPhoneNumber
		}
		newPhone = "0" + phone
	}

	// 规则3：标准化后的号码，第二位必须是 '8' 或 '9'
	secondChar := newPhone[1]
	if secondChar != '8' && secondChar != '9' {
		return nil, ErrInvalidPhoneNumber
	}

	// 所有校验通过后，创建 PhoneNumber 对象
	basePhone := newPhone[1:]
	return &PhoneNumber{basePhone: basePhone}, nil
}

// GetBasePhone 返回10位的基本手机号
// 例如："9171234567"
func (p *PhoneNumber) GetBasePhone() (basePhone string) {
	return p.basePhone
}

// GetWithCountryCode 返回带 '63' 国家代码前缀的手机号
// 例如："639171234567"
func (p *PhoneNumber) GetWithCountryCode() (normalPhone string) {
	return "63" + p.basePhone
}

// GetWithNationalCode 返回带 '0' 国内前缀的手机号
// 例如："09171234567"
func (p *PhoneNumber) GetWithNationalCode() (nationalCodePhone string) {
	return "0" + p.basePhone
}
