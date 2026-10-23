package throttle

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"
)

// baseLimiterManager 基础限流器管理器
type baseLimiterManager struct {
	cache        *CacheManager         // 配置缓存
	cacheTTL     time.Duration         // 缓存有效期
	configParser ParserFunc            // 配置解析函数
	redisClient  redis.UniversalClient // Redis实例
}

// newBaseLimiterManager 创建基础限流器管理器
func newBaseLimiterManager(limiterName string, cacheTTL time.Duration, configParser ParserFunc, redisClient redis.UniversalClient) *baseLimiterManager {
	return &baseLimiterManager{
		cacheTTL:     cacheTTL,
		cache:        NewCacheManager(limiterName, configParser, redisClient),
		configParser: configParser,
		redisClient:  redisClient,
	}
}

// getConfig 获取配置（自动处理缓存策略）
func (m *baseLimiterManager) getConfig() configInf {
	return m.cache.getCache()
}

// LoadConfig 从Redis加载配置并更新缓存
func (m *baseLimiterManager) LoadConfig(ctx context.Context) error {
	// 解析配置
	config, err := m.configParser(ctx, m.redisClient)
	if err != nil {
		log.ErrorCtx(ctx, "parse config failed", zap.Error(err))
		return fmt.Errorf("parse config failed: %w", err)
	}
	m.cache.updateCache(config)
	return nil
}
