// Package json JSON处理工具包
// @Author: yinwei
// @File: decode
// @Version: 1.0.0
// @Date: 2025/4/14 16:44

package json

// Unmarshal 将JSON数据解码为Go对象
// 这是对jsoniter库Unmarshal方法的封装，提供更好的性能
// 参数:
//   - data: 要解码的JSON字节数据
//   - v: 目标对象的指针，解码后的数据将存储到这个对象中
//
// 返回值:
//   - error: 解码过程中的错误，成功时返回nil
func Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
