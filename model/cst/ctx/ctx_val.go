// Package ctx
// @Author: yinwei
// @File: ctx_val
// @Version: 1.0.0
// @Date: 2025/5/8 17:53

package ctx_cst

// gin context key
const (
	// context 中错误信息
	RESP_KEY_ERROR = "resp_key_error"
	// context 中返回数据
	RESP_KEY_DATA = "resp_key_data"
	// 不使用默认返回结构 标识
	RESP_KEY_NO_DEFAULT = "resp_key_no_default"
	// context 中的 msgBox
	RESP_KEY_MSGBOX = "resp_key_msgbox"
	//context 中的 请求返回头 格式为 map[string]string
	RESP_KEY_HEADER = "resp_key_header"
	// 忽略响应日志输出
	RESP_NO_RESPONSE_LOG = "resp_no_response_log"
)

// gin context key
const (
	// IP地址
	HEADER_KEY_IP = "Ip"
	// token header
	HEADER_KEY_TOKEN = "Token"
	// terminal 终端类型
	HEADER_KEY_TERMINAL = "Terminal"
	// http header debug
	HEADER_KEY_DEBUG = "Debug"
)
