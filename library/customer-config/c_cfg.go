package customer_config

import (
	"fmt"

	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/pmicro/config"
	"go.uber.org/zap"
)

// GetCustomerConfig 获取自定义配置
//
// 获取自定义配置，T 类型必须是非指针类型否则会获取不到数据
//
// Returns：
//   - T 配置信息
func GetCustomerConfig[T any]() T {
	cfg := config.GetCustomerConfig()
	val, ok := cfg.(T)
	if !ok {
		log.Fatal("GetCustomerConfig failed type error", zap.Any("cfg type", fmt.Sprintf("%T", cfg)))
	}
	return val
}
