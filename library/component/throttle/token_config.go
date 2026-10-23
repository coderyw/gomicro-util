package throttle

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"

	"strconv"
)

// TokenLimitConfig Token维度限流配置
type tokenConfig struct {
	limitEnable  bool
	black_enable bool
	timeWindow   int64
	maxCount     int64
}

// isLimitEnable 是否开启限流
func (c *tokenConfig) isLimitEnable() bool {
	return c.limitEnable
}

func (c *tokenConfig) isBlackEnable() bool {
	return c.black_enable
}

// getTimeWindow 获取限流时间窗口
func (c *tokenConfig) getTimeWindow() int64 {
	return c.timeWindow
}

// getMaxCount 获取最大请求次数
func (c *tokenConfig) getMaxCount(api ...string) int64 {
	return c.maxCount
}

// setLimitEnable 设置是否开启限流
func (c *tokenConfig) setLimitEnable(data map[string]string, key string) error {
	return c.boolSetter(data, key, func(b bool) {
		c.limitEnable = b
	})
}

// setBlackEnable 设置是否开启黑名单
func (c *tokenConfig) setBlackEnable(data map[string]string, key string) error {
	return c.boolSetter(data, key, func(b bool) {
		c.black_enable = b
	})
}

// setTimeWindow 设置限流时间窗口
func (c *tokenConfig) setTimeWindow(data map[string]string, key string) error {
	return c.intSetter(data, key, func(i int64) {
		c.timeWindow = i
	})
}

// setMaxCount 设置最大请求次数
func (c *tokenConfig) setMaxCount(data map[string]string, key string) error {
	return c.intSetter(data, key, func(i int64) {
		c.maxCount = i
	})
}

func (c *tokenConfig) intSetter(data map[string]string, key string, setter func(int64)) error {
	if v, ok := data[key]; ok {
		s, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("parse key %s value %s to int64 error %s", key, v, err)
		}
		//c.timeWindow = fmt.Sprintf("%dm", s)
		setter(s)
		return nil
	}
	return nil
}

func (c *tokenConfig) stringSetter(data map[string]string, key string, setter func(string)) error {
	if v, ok := data[key]; ok {
		setter(v)
		return nil
	}
	return nil
}

func (c *tokenConfig) boolSetter(data map[string]string, key string, setter func(bool)) error {
	if v, ok := data[key]; ok {
		s, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("parse key %s value %s to bool error %s", key, v, err)
		}
		setter(s)
		return nil
	}
	return nil
}

// setApiConfig 设置API限流配置, token限流不需实现
func (c *tokenConfig) setApiConfig(ctx context.Context, apiMap map[string]string) {}

// parseTokenLimitConfig 解析Token维度配置
func parseTokenLimitConfig(ctx context.Context, redisClient redis.UniversalClient) (configInf, error) {
	config := &tokenConfig{
		limitEnable:  false,
		black_enable: false,
		timeWindow:   1,
		maxCount:     100,
	}

	// 从Redis Hash获取所有配置
	configMap, err := redisClient.HGetAll(ctx, redisHashTokenConfig).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			log.ErrorCtx(ctx, fmt.Sprintf("hgetall %s failed", redisHashTokenConfig), zap.Error(err))
			return nil, err
		}
	}
	if err := config.setBlackEnable(configMap, "black_enable"); err != nil {
		return nil, err
	}
	if err := config.setLimitEnable(configMap, "limit_enable"); err != nil {
		return nil, err
	}
	// 解析api_time_window（默认1分钟）
	if err := config.setTimeWindow(configMap, "time_window"); err != nil {
		return nil, err
	}
	// 解析api_max_count（默认100）
	if err := config.setMaxCount(configMap, "max_count"); err != nil {
		return nil, err
	}

	return config, nil
}
