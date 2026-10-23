package throttle

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"
	"strconv"
)

// TokenLimitConfig Token维度限流配置
// @field limitEnable 是否开启限流
// @field blackEnable 是否开启黑名单
// @field timeWindow 限流时间窗口
// @field maxCount 默认最大请求次数
// @field apiConfig API限流配置map，key为API字符串，value为最大请求次数
type tokenApiConfig struct {
	*tokenConfig
	apiConfig map[string]int64
}

// newTokenApiConfig 创建Token API限流配置
// @return 返回Token API限流配置
func newTokenApiConfig() *tokenApiConfig {
	return &tokenApiConfig{
		tokenConfig: &tokenConfig{},
		apiConfig:   make(map[string]int64), // 初始化避免nil
	}
}

// setApiConfig 设置API限流配置
// @param ctx 上下文
// @param apiConf API限流配置map，key为API字符串，value为最大请求次数
func (c *tokenApiConfig) setApiConfig(ctx context.Context, apiConf map[string]string) {
	for k, v := range apiConf {
		vv, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			log.ErrorCtx(ctx, "tokenApiConfig setApiConfig failed", zap.String("key", k), zap.String("value", v), zap.Error(err))
			continue
		}
		c.apiConfig[k] = vv
	}

}

// getMaxCount 如果API配置中不存在该API，则返回默认配置的最大请求数
// @param api 返回API字符串
// @return 返回API最大请求数
func (c *tokenApiConfig) getMaxCount(api ...string) int64 {
	// 如果没有API参数，返回默认配置的最大请求数
	if len(api) == 0 {
		return c.tokenConfig.getMaxCount()
	}
	if apiMax, exists := c.apiConfig[api[0]]; exists {
		return apiMax
	}
	return c.tokenConfig.getMaxCount()
}

// parseTokenApiLimitConfig：返回带redisClient的配置
func parseTokenApiLimitConfig(ctx context.Context, redisClient redis.UniversalClient) (configInf, error) {
	config := newTokenApiConfig()
	// 从Redis Hash获取所有配置
	configMap, err := redisClient.HGetAll(ctx, redisHashTokenApiConfig).Result()
	if err != nil {
		log.ErrorCtx(ctx, fmt.Sprintf("parseTokenApiLimitConfig %s failed", redisHashTokenApiConfig), zap.Error(err))
		return nil, fmt.Errorf("parseTokenApiLimitConfig %s failed: %w", redisHashTokenApiConfig, err)
	}
	if err := config.setBlackEnable(configMap, "api_black_enable"); err != nil {
		return nil, err
	}
	if err := config.setLimitEnable(configMap, "api_limit_enable"); err != nil {
		return nil, err
	}
	// 解析api_time_window（默认1分钟）
	if err := config.setTimeWindow(configMap, "api_time_window"); err != nil {
		return nil, err
	}
	// 解析api_max_count（默认100）
	if err := config.setMaxCount(configMap, "api_max_count"); err != nil {
		return nil, err
	}

	// 从Redis Hash获取所有 API 限流配置
	apiMap, err := redisClient.HGetAll(ctx, redisHashApiMaxCount).Result()
	if err != nil {
		log.ErrorCtx(ctx, fmt.Sprintf("parseTokenApiLimitConfig %s failed", redisHashApiMaxCount), zap.Error(err))
		return nil, fmt.Errorf("parseTokenApiLimitConfig %s failed: %w", redisHashApiMaxCount, err)
	}
	// 解析api_max_count（默认100）
	config.setApiConfig(ctx, apiMap)

	return config, nil
}
