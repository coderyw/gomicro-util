// Package json JSON处理工具包
// @Author: yinwei
// @File: encode
// @Version: 1.0.0
// @Date: 2025/4/14 16:44

package json

// Marshal 将Go对象编码为JSON数据
// 这是对jsoniter库Marshal方法的封装，提供更好的性能
// 参数:
//   - v: 要编码的Go对象
//
// 返回值:
//   - []byte: 编码后的JSON字节数据
//   - error: 编码过程中的错误，成功时返回nil
func Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
