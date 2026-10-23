package env

import (
	"fmt"
	"os"

	"github.com/coderyw/gomicro-util/model/cst"
)

// 启动时读取，保证进程内所有数据中心使用相同模式，不支持热切换。
var disableEonidHashTag = os.Getenv(cst.ENV_DISABLE_EONID_HASHTAG)

// GetDisableEonidHashTag 返回是否使用旧版 eonId Redis 前缀。
// 未配置或 0 使用 {eonId}，1 使用 eonId；其他值返回配置错误。
func GetDisableEonidHashTag() (bool, error) {
	switch disableEonidHashTag {
	case "", "0":
		return false, nil
	case "1":
		return true, nil
	default:
		return false, fmt.Errorf("%s must be 0 or 1", cst.ENV_DISABLE_EONID_HASHTAG)
	}
}
