package throttle

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	limiterRedis "github.com/ulule/limiter/v3/drivers/store/redis"
	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"
	"time"
)

type TokenLimiterManager struct {
	*baseLimiterManager
	limiterStore limiter.Store // limiter的Redis存储
}

// NewTokenLimiterManager 创建基础限流器管理器
func NewTokenLimiterManager(prefix string, cacheTtl time.Duration, redisClient redis.UniversalClient) (*TokenLimiterManager, error) {
	// 初始化limiter的Redis存储
	store, err := limiterRedis.NewStoreWithOptions(
		redisClient,
		limiter.StoreOptions{
			Prefix: fmt.Sprintf("limiter:%s", prefix),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("TokenLimiterManager ERR: init limiter store failed: %w", err)
	}

	return &TokenLimiterManager{
		baseLimiterManager: newBaseLimiterManager("WindowToken", cacheTtl, parseTokenLimitConfig, redisClient),
		limiterStore:       store,
	}, nil
}

// Limit 实现限流
func (m *TokenLimiterManager) Limit(ctx context.Context, userId string) bool {
	// 获取配置
	config := m.getConfig()
	if config == nil {
		log.ErrorCtx(ctx, "TokenLimiterManager ERR: get config failed", zap.String("userId", userId))
		return false
	}

	// 计算每分钟的限流次数
	maxCount := config.getMaxCount()
	timeWindow := config.getTimeWindow()
	countPerMin := int64((float32(maxCount) / float32(timeWindow)) * 60)

	// 解析限流规则
	rateStr := fmt.Sprintf("%d-m", int64(countPerMin))
	rate, err := limiter.NewRateFromFormatted(rateStr)
	if err != nil {
		log.ErrorCtx(ctx, "TokenLimiterManager ERR: parse rate failed", zap.String("userId", userId), zap.Error(err))
		return false
	}

	// 创建限流器实例（内部会基于limitKey隔离计数）
	limiterInstance := limiter.New(m.limiterStore, rate)
	// 如果限流器实例为空，直接放行
	if limiterInstance == nil {
		log.ErrorCtx(ctx, "TokenLimiterManager ERR: limiter instance is nil", zap.String("userId", userId))
		return false
	}

	// 使用limiterKey进行限流检查
	c, err := limiterInstance.Get(ctx, userId)
	// 如果限流器实例为空，直接放行
	if err != nil {
		log.ErrorCtx(ctx, "TokenLimiterManager ERR: get limiter failed", zap.String("userId", userId), zap.Error(err))
		return false
	}
	if c.Reached {
		log.InfoCtx(ctx, "TokenLimiterManager LIMIT: limiter reached", zap.String("userId", userId))
		if !config.isLimitEnable() { // 限流器只打印日志, 不阻塞用户
			return false
		} else {
			return true // 限流
		}
	} else {
		return false // 未限流
	}
}
