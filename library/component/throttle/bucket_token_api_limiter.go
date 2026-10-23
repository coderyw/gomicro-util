package throttle

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"

	"github.com/go-redis/redis_rate/v10"
	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"
	"strings"
	//"github.com/coderyw/gomicro-util/library/db/cache/redis"
	"time"
)

// BucketTokenApiLimiterManager 令牌桶限流器管理器
type BucketTokenApiLimiterManager struct {
	*baseLimiterManager                     // 匿名嵌套基础管理器，复用配置/Redis客户端
	limiter             *redis_rate.Limiter // redis_rate 限流器实例
	limiterPrefix       string              // Redis key 前缀，避免冲突
}

// NewBucketTokenApiLimiterManager 创建令牌桶限流器管理器
func NewBucketTokenApiLimiterManager(prefix string, cacheTTL time.Duration, redisClient redis.UniversalClient) (*BucketTokenApiLimiterManager, error) {
	// 初始化 redis_rate 限流器
	limiter := redis_rate.NewLimiter(redisClient)

	return &BucketTokenApiLimiterManager{
		baseLimiterManager: newBaseLimiterManager("BucketTokenApiLimiter", cacheTTL, parseTokenApiLimitConfig, redisClient),
		limiter:            limiter,
		limiterPrefix:      fmt.Sprintf("limiter:bucket:%s", prefix), // 令牌桶 key 前缀
	}, nil
}

// EnableLimit 获取限流器开关状态
//
// Returns:
//   - bool true=开启限流，false=关闭限流
func (m *BucketTokenApiLimiterManager) EnableLimit() bool {
	return m.getConfig().isLimitEnable()
}

// EnableBlack 获取黑名单开关状态
//
// Returns:
//   - bool
func (m *BucketTokenApiLimiterManager) EnableBlack() bool {
	return m.getConfig().isBlackEnable()
}

// Limit 对外暴露的限流方法
func (m *BucketTokenApiLimiterManager) Limit(ctx context.Context, userId string, api string) bool {
	// 获取现有配置
	config := m.getConfig()
	if config == nil {
		log.ErrorCtx(ctx, "BucketTokenApiLimiterManager ERR: get config failed", zap.String("api", api), zap.String("userId", userId))
		return false
	}

	// 解析配置参数
	maxCount := config.getMaxCount(api) // 桶容量 = 窗口最大请求数
	timeWindow := config.getTimeWindow()
	// 构造 redis_rate 的令牌桶规则
	limitRule := &redis_rate.Limit{
		Rate:   int(maxCount),                           // 令牌生成速率
		Burst:  int(maxCount),                           // 桶容量（最大突发请求数）
		Period: time.Duration(timeWindow) * time.Second, // 窗口周期
	}

	if limitRule == nil {
		log.ErrorCtx(ctx, "BucketTokenApiLimiterManager ERR: limit rule is nil", zap.String("api", api), zap.String("userId", userId), zap.Any("limitRule", limitRule))
		return false // 配置异常/未启用限流，放行
	}

	// 生成限流 key（用户 + API 维度，复用原有 getLimiterKey 逻辑）
	limitKey := fmt.Sprintf("%s:%s", m.limiterPrefix, m.getLimiterKey(userId, api))
	// 尝试获取令牌（核心限流逻辑）
	result, err := m.limiter.Allow(ctx, limitKey, *limitRule)
	if err != nil {
		// 异常时默认放行（或根据业务需求改为限流）
		log.ErrorCtx(ctx, "BucketTokenApiLimiterManager ERR: limit check exception", zap.String("api", api), zap.String("userId", userId), zap.Error(err))
		return false
	}

	// 判断是否限流：result.Allowed == 0 表示令牌耗尽
	if result.Allowed == 0 {
		log.ErrorCtx(ctx, "BucketTokenApiLimiterManager LIMIT: limit check failed", zap.String("api", api), zap.String("userId", userId), zap.Any("result", result))
		if !config.isLimitEnable() { // 限流器只打印日志, 不阻塞用户
			return false // 黑名单用户，直接拒绝
		} else {
			return true
		}
	} else { //未触发限流
		return false
	}
}

// getLimiterKey 获取限流器键值
func (m *BucketTokenApiLimiterManager) getLimiterKey(userId string, api string) string {
	return fmt.Sprintf("%s%s", userId, strings.ReplaceAll(api, "/", "_"))
}
