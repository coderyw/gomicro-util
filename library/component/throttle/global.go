package throttle

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"sync"
	"time"

	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"
)

// 全局变量（仅本包可见，对外通过函数访问）
var (
	globalTokenLimiter    TokenLimiterInf
	globalTokenApiLimiter TokenApiLimiterInf
	once                  sync.Once
)

const (
	// cacheTtl 限流器缓存时间（1分钟）
	cacheTtl = 1 * time.Minute

	// redisHashTokenConfig Redis Hash Token限流配置
	redisHashTokenConfig = "redis_hash_token_rate_config"
	// tokenLimiterPrefix Token限流器键前缀
	tokenLimiterPrefix = "req-token-limiter"

	// redisHashTokenApiConfig Redis Hash Token+API限流器配置键
	redisHashTokenApiConfig = "redis_hash_token_api_rate_config"
	// redisHashApiMaxCount Redis Hash Token+API最大请求次数配置键
	redisHashApiMaxCount = "redis_hash_token_api_max_count"
	// tokenApiLimiterPrefix Token+API限流器键前缀
	tokenApiLimiterPrefix = "req-token-api-limiter"
)

// InitWindowLimiter 初始化基于固定时间窗口的全局限流器（对外暴露的唯一初始化入口）
func InitWindowLimiter(ctx context.Context, redisClient redis.UniversalClient) error {
	var initErr error
	once.Do(func() {
		// 初始化Token限流器, 并加载redis配置
		tokenLimiter, err := NewTokenLimiterManager(tokenLimiterPrefix, cacheTtl, redisClient)
		if err != nil {
			log.ErrorCtx(ctx, "init token limiter failed", zap.Error(err))
			initErr = fmt.Errorf("init token limiter failed: %w", err)
			return
		}
		globalTokenLimiter = tokenLimiter
		if err := globalTokenLimiter.LoadConfig(ctx); err != nil {
			log.ErrorCtx(ctx, "preload token config failed", zap.Error(err))
			initErr = fmt.Errorf("preload token config failed: %w", err)
			return
		}

		// 初始化Token+API限流器, 并加载redis配置
		tokenApiLimiter, err := NewTokenApiLimiterManager(tokenApiLimiterPrefix, cacheTtl, redisClient)
		if err != nil {
			log.ErrorCtx(ctx, "init token api limiter failed", zap.Error(err))
			initErr = fmt.Errorf("init token-api limiter failed: %w", err)
			return
		}
		globalTokenApiLimiter = tokenApiLimiter
		if err := globalTokenApiLimiter.LoadConfig(ctx); err != nil {
			log.ErrorCtx(ctx, "preload token-api config failed", zap.Error(err))
			initErr = fmt.Errorf("preload token-api config failed: %w", err)
			return
		}
	})
	return initErr
}

// InitBucketLimiter 初始基于令牌桶的全局限流器（对外暴露的唯一初始化入口）
func InitBucketLimiter(ctx context.Context, redisClient redis.UniversalClient) error {
	var initErr error
	once.Do(func() {
		// 初始化Token限流器, 并加载redis配置
		tokenLimiter, err := NewBucketTokenLimiterManager(tokenLimiterPrefix, cacheTtl, redisClient)
		if err != nil {
			log.ErrorCtx(ctx, "init token limiter failed", zap.Error(err))
			initErr = fmt.Errorf("init token limiter failed: %w", err)
			return
		}
		globalTokenLimiter = tokenLimiter
		if err := globalTokenLimiter.LoadConfig(ctx); err != nil {
			log.ErrorCtx(ctx, "preload token config failed", zap.Error(err))
			initErr = fmt.Errorf("preload token config failed: %w", err)
			return
		}

		// 初始化Token+API限流器, 并加载redis配置
		tokenApiLimiter, err := NewBucketTokenApiLimiterManager(tokenApiLimiterPrefix, cacheTtl, redisClient)
		if err != nil {
			log.ErrorCtx(ctx, "init token api limiter failed", zap.Error(err))
			initErr = fmt.Errorf("init token-api limiter failed: %w", err)
			return
		}

		globalTokenApiLimiter = tokenApiLimiter
		if err := globalTokenApiLimiter.LoadConfig(ctx); err != nil {
			log.ErrorCtx(ctx, "preload token-api config failed", zap.Error(err))
			initErr = fmt.Errorf("preload token-api config failed: %w", err)
			return
		}
	})
	return initErr
}

// GetGlobalTokenLimiter 获取全局Token限流器管理器（对外暴露）
func GetGlobalTokenLimiter() (TokenLimiterInf, error) {
	if globalTokenLimiter == nil {
		return nil, errors.New("token limiter not initialized")
	}
	return globalTokenLimiter, nil
}

// GetGlobalTokenApiLimiter 获取全局Token+API限流器管理器（对外暴露）
func GetGlobalTokenApiLimiter() (TokenApiLimiterInf, error) {
	if globalTokenApiLimiter == nil {
		return nil, errors.New("token-api limiter not initialized")
	}
	return globalTokenApiLimiter, nil
}
