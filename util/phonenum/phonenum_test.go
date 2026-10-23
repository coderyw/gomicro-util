package phonenum

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidPhoneNumber(t *testing.T) {
	// 测试有效的手机号码
	validCases := []struct {
		phone     string
		name      string
		basePhone string
	}{
		{"+639171234567", "+63开头12位有效号码", "9171234567"},
		{"639171234567", "63开头12位有效号码", "9171234567"},
		{"09171234567", "0开头11位有效号码", "9171234567"},
		{"9171234567", "无前缀10位有效号码", "9171234567"},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			phoneNum, err := Parse(tc.phone)
			assert.NoError(t, err)
			assert.True(t, phoneNum.GetBasePhone() == "9171234567")
		})
	}

	// 测试无效的手机号码
	invalidCases := []struct {
		phone   string
		name    string
		isValid bool
	}{
		{"63917123456", "63开头但长度不足12位", false},
		{"6391712345678", "63开头但长度超过12位", false},
		{"0917123456", "0开头但长度不足11位", false},
		{"091712345678", "0开头但长度超过11位", false},
		{"917123456", "无前缀但长度不足10位", false},
		{"91712345678", "无前缀但长度超过10位", false},
		{"07171234567", "0开头但第二位不是8或9", false},
		{"637171234567", "63开头但第二位不是8或9", false},
		{"7171234567", "无前缀但第二位不是8或9", false},
		{"abc9171234567", "不以数字开头", false},
		{"917123a4567", "包含非数字字符", false},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.phone)
			assert.Error(t, err)
		})
	}
}
