package throttle

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	limiterRedis "github.com/ulule/limiter/v3/drivers/store/redis"
	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"
	"strings"
	"time"
)

// TokenApiLimiterManager Token维度限流器管理器
type TokenApiLimiterManager struct {
	*baseLimiterManager
	limiterStore limiter.Store // limiter的Redis存储
}

// NewTokenApiLimiterManager 创建Token+API维度限流器管理器
func NewTokenApiLimiterManager(prefix string, cacheTtl time.Duration, redisClient redis.UniversalClient) (*TokenApiLimiterManager, error) {
	// 初始化limiter的Redis存储
	store, err := limiterRedis.NewStoreWithOptions(
		redisClient,
		limiter.StoreOptions{
			Prefix: fmt.Sprintf("limiter:%s", prefix),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("init limiter store failed: %w", err)
	}

	return &TokenApiLimiterManager{
		baseLimiterManager: newBaseLimiterManager("WindowTokenApi", cacheTtl, parseTokenApiLimitConfig, redisClient),
		limiterStore:       store,
	}, nil
}

// EnableLimit 获取限流器开关状态
//
// Returns:
//   - bool true=开启限流，false=关闭限流
func (m *TokenApiLimiterManager) EnableLimit() bool {
	return m.getConfig().isLimitEnable()
}

// EnableBlack 获取黑名单开关状态
//
// Returns:
//   - bool
func (m *TokenApiLimiterManager) EnableBlack() bool {
	return m.getConfig().isBlackEnable()
}

// Limit 实现限流
func (m *TokenApiLimiterManager) Limit(ctx context.Context, userId string, api string) bool {
	// 获取配置
	config := m.getConfig()
	if config == nil {
		log.ErrorCtx(ctx, "TokenApiLimiterManager ERR: get config failed", zap.String("userId", userId), zap.String("api", api))
		return false
	}
	// 计算最终的maxCount（支持API维度覆盖）
	maxCount := config.getMaxCount(api)
	timeWindow := config.getTimeWindow()
	countPerMin := int64((float32(maxCount) / float32(timeWindow)) * 60)

	// 解析限流规则, 每分钟maxCount次, 注意这里不支持自定义的时间窗口
	rateStr := fmt.Sprintf("%d-m", countPerMin)
	rate, err := limiter.NewRateFromFormatted(rateStr)
	if err != nil {
		log.ErrorCtx(ctx, "TokenApiLimiterManager ERR: parse rate failed", zap.String("userId", userId), zap.String("api", api), zap.Error(err))
		return false
	}

	// 创建限流器实例（内部会基于limitKey隔离计数）
	limiterInstance := limiter.New(m.limiterStore, rate)
	// 如果限流器实例为空，直接放行
	if limiterInstance == nil {
		log.ErrorCtx(ctx, "TokenApiLimiterManager ERR: get limiter failed", zap.String("userId", userId), zap.String("api", api), zap.Error(err))
		return false
	}

	// 使用limiterKey进行限流检查
	limiterKey := m.getLimiterKey(userId, api)
	c, err := limiterInstance.Get(ctx, limiterKey)
	if err != nil {
		log.ErrorCtx(ctx, "TokenApiLimiterManager ERR: get limiter failed", zap.String("userId", userId), zap.String("api", api), zap.Error(err))
		return false
	}
	if c.Reached {
		log.InfoCtx(ctx, "TokenApiLimiterManager LIMIT: limiter reached", zap.String("userId", userId), zap.String("api", api), zap.String("limiterKey", limiterKey))
		if !config.isLimitEnable() { // 限流器只打印日志, 不阻塞用户
			return false
		} else {
			return true // 限流
		}
	} else {
		return false // 未限流
	}
}

// getLimiterKey 获取限流器键值
func (m *TokenApiLimiterManager) getLimiterKey(userId string, api string) string {
	return fmt.Sprintf("%s%s", userId, strings.ReplaceAll(api, "/", "_"))
}
