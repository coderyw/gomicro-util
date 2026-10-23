package secret

import (
	"errors"
	"regexp"

	"github.com/coderyw/gomicro-util/library/log"
)

// yaml 的类型错误会把标量原文放进反引号：cannot unmarshal !!str `VALUE` into int
var yamlLiteral = regexp.MustCompile("`[^`]*`")

// SanitizeDecodeError 把"引用解析之后"的反序列化错误转换成不含凭据值的新错误：
// 已登记到日志脱敏表的明文替换为掩码，yaml 反引号内的标量原文整体抹掉，只保留行号/字段/类型等定位信息。
// 返回的是新错误、不包装原错误，避免调用方 errors.Unwrap / %v 再把明文取回来。
// 只在本次确实替换过引用时才需要：UnmarshalYAML / UnmarshalJSON 没有替换引用时原样返回原错误，保留原文与错误类型
func SanitizeDecodeError(err error) error {
	if err == nil {
		return nil
	}
	return errors.New(yamlLiteral.ReplaceAllString(log.RedactString(err.Error()), "`***`"))
}
