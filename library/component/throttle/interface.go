package throttle

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// ParserFunc 配置解析函数
type ParserFunc func(ctx context.Context, redisClient redis.UniversalClient) (configInf, error)

// LoadFunc 加载配置函数
type LoadFunc func(ctx context.Context) error

// configInf 配置接口
type configInf interface {
	isLimitEnable() bool
	isBlackEnable() bool
	getMaxCount(api ...string) int64
	getTimeWindow() int64

	setLimitEnable(data map[string]string, key string) error
	setBlackEnable(data map[string]string, key string) error
	setMaxCount(data map[string]string, key string) error
	setTimeWindow(data map[string]string, key string) error

	setApiConfig(ctx context.Context, apiMap map[string]string)
}

type TokenLimiterInf interface {
	Limit(ctx context.Context, userId string) bool
	LoadConfig(ctx context.Context) error
}

type TokenApiLimiterInf interface {
	Limit(ctx context.Context, userId string, api string) bool
	// EnableLimit 获取限流器开关状态
	//
	// Returns:
	//   - bool true=开启限流，false=关闭限流
	EnableLimit() bool
	// EnableBlack 获取黑名单开关状态
	//
	// Returns:
	//   - bool
	EnableBlack() bool
	LoadConfig(ctx context.Context) error
}

type LimiterExecutorInf interface {
	ExistUserBlackCache(ctx context.Context, userId string) (bool, error)
	HandleRateLimitTrigger(ctx context.Context, userId string) error
	KickOutUser(ctx context.Context, userId string) error
}
