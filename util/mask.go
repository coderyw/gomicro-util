package util

// Mask 自定义字符串脱敏
//
// Parameters:
//   - s string 原始自付出
//   - start int 前start个字符显示
//   - end int 后end个字符显示
//
// Returns：
//   - string 返回脱敏后的字符串
func Mask(s string, start, end int) string {
	n := len(s)
	if start < 0 || end < 0 || n <= start+end {
		return "***"
	}

	// 预估容量：start + "***" + end
	buf := make([]byte, 0, start+3+end)
	buf = append(buf, s[:start]...)
	buf = append(buf, "***"...)
	buf = append(buf, s[n-end:]...)
	return string(buf)
}
