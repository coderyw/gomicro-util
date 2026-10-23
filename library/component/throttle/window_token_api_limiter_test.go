package throttle

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// TokenApiLimiterTestSuite 固定窗口API限流器测试套件
type TokenApiLimiterTestSuite struct {
	suite.Suite
	ctx         context.Context
	limiter     *TokenApiLimiterManager
	userId      string // 测试用户ID
	defaultApi  string // 默认API
	specialApi  string // 特殊配置API
	maxCount    int64  // 默认最大请求数
	specialMax  int64  // 特殊API最大请求数
	window      int64  // 时间窗口(秒)
	prefix      string // Redis键前缀
	redisDB     int    // Redis数据库编号
	redisClient redis.UniversalClient
}

// 初始化测试环境
func (s *TokenApiLimiterTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.userId = "test_window_api_user"
	s.defaultApi = "_test_default"
	s.specialApi = "_test_special"
	s.maxCount = 5    // 默认API窗口内最大5次
	s.specialMax = 10 // 特殊API窗口内最大10次
	s.window = 60     // 30秒窗口
	s.prefix = "test_window_api"
	s.redisDB = 11

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "172.33.3.141:6379",
		Password: "",
		DB:       s.redisDB,
	})
	assert.NoError(s.T(), redisClient.Ping(s.ctx).Err())
	s.redisClient = redisClient

	// 清理残留数据
	s.cleanupTestData()

	// 写入默认限流配置
	defaultConfig := map[string]interface{}{
		"api_limit_enable": "true",
		"api_max_count":    s.maxCount,
		"api_time_window":  s.window,
	}
	err := redisClient.HMSet(s.ctx, redisHashTokenApiConfig, defaultConfig).Err()
	assert.NoError(s.T(), err)

	// 写入特殊API配置
	specialConfig := map[string]interface{}{
		s.specialApi: s.specialMax,
	}
	err = redisClient.HMSet(s.ctx, redisHashApiMaxCount, specialConfig).Err()
	assert.NoError(s.T(), err)

	// 初始化限流器
	limiter, err := NewTokenApiLimiterManager(
		s.prefix,
		1*time.Minute,
		redisClient,
	)
	assert.NoError(s.T(), err)
	s.limiter = limiter
	assert.NoError(s.T(), s.limiter.LoadConfig(s.ctx))
}

// 清理测试数据
func (s *TokenApiLimiterTestSuite) cleanupTestData() {
	//redisClient := redis.GetRedis(s.ctx).Client()
	//_ = redisClient.Del(s.ctx, redisHashTokenApiConfig).Err()
	//_ = redisClient.Del(s.ctx, redisHashApiMaxCount).Err()
	//// 清理限流器计数键
	//keys, _ := redisClient.Keys(s.ctx, fmt.Sprintf("limiter:%s:*", s.prefix)).Result()
	//if len(keys) > 0 {
	//	_ = redisClient.Del(s.ctx, keys...).Err()
	//}
}

// 测试1: 默认API限流控制
func (s *TokenApiLimiterTestSuite) TestDefaultApiLimit() {
	t := s.T()
	ctx := s.ctx

	var (
		success int64 = 0
		fail    int64 = 0
		total   int64 = s.maxCount + 3 // 超过最大限制3个
		wg      sync.WaitGroup
	)

	wg.Add(3)
	reqChan := make(chan struct{}, total)
	for i := int64(0); i < total; i++ {
		reqChan <- struct{}{}
	}
	close(reqChan)

	for i := 0; i < 3; i++ {
		go func() {
			defer wg.Done()
			for range reqChan {
				if s.limiter.Limit(ctx, s.userId, s.defaultApi) {
					atomic.AddInt64(&fail, 1)
				} else {
					atomic.AddInt64(&success, 1)
				}
			}
		}()
	}
	wg.Wait()

	s.redisClient.Del(ctx, fmt.Sprintf("limiter:%s:%s%s", s.prefix, s.userId, s.defaultApi))
	s.redisClient.Del(ctx, fmt.Sprintf("limiter:%s:%s%s", s.prefix, s.userId, s.specialApi))

	assert.Equal(t, s.maxCount, success, "默认API窗口内成功请求数不符合预期")
	assert.Equal(t, int64(3), fail, "默认API窗口内限流数不符合预期")
}

// 测试2: 特殊API限流控制
func (s *TokenApiLimiterTestSuite) TestSpecialApiLimit() {
	t := s.T()
	ctx := s.ctx

	var (
		success int64 = 0
		fail    int64 = 0
		total   int64 = s.specialMax + 4 // 超过最大限制4个
		wg      sync.WaitGroup
	)

	wg.Add(4)
	reqChan := make(chan struct{}, total)
	for i := int64(0); i < total; i++ {
		reqChan <- struct{}{}
	}
	close(reqChan)

	for i := 0; i < 4; i++ {
		go func() {
			defer wg.Done()
			for range reqChan {
				if s.limiter.Limit(ctx, s.userId, s.specialApi) {
					atomic.AddInt64(&fail, 1)
				} else {
					atomic.AddInt64(&success, 1)
				}
			}
		}()
	}
	wg.Wait()

	s.redisClient.Del(ctx, fmt.Sprintf("limiter:%s:%s%s", s.prefix, s.userId, s.defaultApi))
	s.redisClient.Del(ctx, fmt.Sprintf("limiter:%s:%s%s", s.prefix, s.userId, s.specialApi))

	assert.Equal(t, s.specialMax, success, "特殊API窗口内成功请求数不符合预期")
	assert.Equal(t, int64(4), fail, "特殊API窗口内限流数不符合预期")
}

// 测试3: API维度隔离性
func (s *TokenApiLimiterTestSuite) TestApiIsolation() {
	t := s.T()
	ctx := s.ctx

	// 耗尽默认API额度
	s.consumeAllTokens(t, s.defaultApi, s.maxCount)

	// 验证特殊API仍可正常请求
	success := 0
	for i := 0; i < int(s.specialMax); i++ {
		if !s.limiter.Limit(ctx, s.userId, s.specialApi) {
			success++
		}
	}

	fmt.Printf("limiter:%s:%s:%s", s.prefix, s.userId, s.defaultApi)

	s.redisClient.Del(ctx, fmt.Sprintf("limiter:%s:%s%s", s.prefix, s.userId, s.defaultApi))
	s.redisClient.Del(ctx, fmt.Sprintf("limiter:%s:%s%s", s.prefix, s.userId, s.specialApi))

	assert.Equal(t, int(s.specialMax), success, "API维度隔离性验证失败")
}

// 测试4: 窗口过期恢复
func (s *TokenApiLimiterTestSuite) TestWindowExpire() {
	t := s.T()
	ctx := s.ctx

	// 耗尽特殊API额度
	s.consumeAllTokens(t, s.specialApi, s.specialMax)

	// 等待窗口过期
	time.Sleep(time.Duration(s.window+1) * time.Second)

	// 验证额度恢复
	success := 0
	for i := 0; i < int(s.specialMax); i++ {
		if !s.limiter.Limit(ctx, s.userId, s.specialApi) {
			success++
		}
	}

	s.redisClient.Del(ctx, fmt.Sprintf("limiter:%s:%s%s", s.prefix, s.userId, s.defaultApi))
	s.redisClient.Del(ctx, fmt.Sprintf("limiter:%s:%s%s", s.prefix, s.userId, s.specialApi))

	assert.Equal(t, int(s.specialMax), success, "窗口过期后额度未恢复")
}

// 辅助方法: 耗尽指定API的窗口额度
func (s *TokenApiLimiterTestSuite) consumeAllTokens(t *testing.T, api string, max int64) {
	ctx := s.ctx
	count := 0
	for count < int(max) {
		if !s.limiter.Limit(ctx, s.userId, api) {
			count++
		}
	}
	assert.Equal(t, int(max), count, fmt.Sprintf("未能耗尽API[%s]的窗口额度", api))
}

// 清理测试环境
func (s *TokenApiLimiterTestSuite) TearDownSuite() {
	s.cleanupTestData()
}

// 启动测试套件
func TestTokenApiLimiterSuite(t *testing.T) {
	suite.Run(t, new(TokenApiLimiterTestSuite))
}
