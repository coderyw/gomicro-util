package throttle

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/coderyw/gomicro-util/library/cron"
	"github.com/coderyw/gomicro-util/library/log"
	cron2 "github.com/coderyw/gomicro-util/model/cron"
	"go.uber.org/zap"
)

// CacheManager 缓存项（带 stale-while-revalidate 策略）
type CacheManager struct {
	expireAt   time.Time
	mu         sync.RWMutex
	refreshing bool

	configMutex  sync.RWMutex
	config       configInf
	redisClient  redis.UniversalClient
	configParser ParserFunc
}

// NewCacheManager 初始化缓存配置
func NewCacheManager(cronName string, configParser ParserFunc, redisClient redis.UniversalClient) *CacheManager {
	c := &CacheManager{
		expireAt:     time.Now().Add(-1), // 初始化为已过期，触发首次加载
		redisClient:  redisClient,
		configParser: configParser,
	}
	// 添加到cron中
	cron.AddCron(cron2.CronJob{
		Name:            cronName,
		DoJob:           c.loadConfig,
		CronSpc:         "0/10 * * * * ?",
		DoBeforeDone:    false,
		DelayBeforeDone: false,
	})
	//load一次配置
	c.loadConfig()
	return c
}

func (c *CacheManager) loadConfig() {
	// 加载最新配置
	if config, err := c.configParser(context.Background(), c.redisClient); err == nil {
		c.updateCache(config)
		log.Info("load config from redis success", zap.Any("config", config))
	} else {
		log.Error("load config from redis failed", zap.Error(err))
	}
}

// updateCache 更新缓存
func (c *CacheManager) updateCache(conf configInf) {
	c.configMutex.Lock()
	defer c.configMutex.Unlock()
	c.config = conf
}

func (c *CacheManager) getCache() configInf {
	c.configMutex.RLock()
	defer c.configMutex.RUnlock() // 释放配置读锁
	// 缓存有效，直接返回
	return c.config
}
