package throttle

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TokenLimiterTestSuite 固定窗口限流器测试套件
type TokenLimiterTestSuite struct {
	suite.Suite
	ctx         context.Context
	limiter     *TokenLimiterManager
	userId      string // 测试用户ID
	maxCount    int64  // 窗口最大请求数
	window      int64  // 时间窗口(秒)
	prefix      string // Redis键前缀
	redisDB     int    // Redis数据库编号
	redisClient redis.UniversalClient
}

// 初始化测试环境
func (s *TokenLimiterTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.userId = "test_window_token_user"
	s.maxCount = 10 // 窗口内最大10次请求
	s.window = 60   // 30秒窗口
	s.prefix = "test_window_token"
	s.redisDB = 11

	// 初始化Redis连接

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "172.33.3.141:6379",
		Password: "",
		DB:       s.redisDB,
	})

	assert.NoError(s.T(), redisClient.Ping(s.ctx).Err())

	s.redisClient = redisClient

	// 清理残留数据
	s.cleanupTestData()

	// 写入限流配置
	config := map[string]interface{}{
		"limit_enable": "true",
		"max_count":    s.maxCount,
		"time_window":  s.window,
	}
	err := redisClient.HMSet(s.ctx, redisHashTokenConfig, config).Err()
	assert.NoError(s.T(), err)

	// 初始化限流器
	limiter, err := NewTokenLimiterManager(
		s.prefix,
		1*time.Minute,
		redisClient,
	)
	assert.NoError(s.T(), err)
	s.limiter = limiter
	assert.NoError(s.T(), s.limiter.LoadConfig(s.ctx))
}

// 清理测试数据
func (s *TokenLimiterTestSuite) cleanupTestData() {
	//redisClient := redis.GetRedis(s.ctx).Client()
	//_ = redisClient.Del(s.ctx, redisHashTokenConfig).Err()
	//// 清理限流器计数键
	//keys, _ := redisClient.Keys(s.ctx, fmt.Sprintf("limiter:%s:*", s.prefix)).Result()
	//if len(keys) > 0 {
	//	_ = redisClient.Del(s.ctx, keys...).Err()
	//}
}

// 测试1: 窗口内请求数控制
func (s *TokenLimiterTestSuite) TestWindowLimit() {
	t := s.T()
	ctx := s.ctx

	var (
		success int64 = 0
		fail    int64 = 0
		total   int64 = s.maxCount + 5 // 超过最大限制5个
		wg      sync.WaitGroup
	)

	wg.Add(5)
	reqChan := make(chan struct{}, total)
	for i := int64(0); i < total; i++ {
		reqChan <- struct{}{}
	}
	close(reqChan)

	for i := 0; i < 5; i++ {
		go func() {
			defer wg.Done()
			for range reqChan {
				if s.limiter.Limit(ctx, s.userId) {
					atomic.AddInt64(&fail, 1)
				} else {
					atomic.AddInt64(&success, 1)
				}
			}
		}()
	}
	wg.Wait()

	s.redisClient.Del(ctx, fmt.Sprintf("limiter:%s:%s", s.prefix, s.userId))

	// 验证窗口内成功数=maxCount，失败数=5
	assert.Equal(t, s.maxCount, success, "窗口内成功请求数不符合预期")
	assert.Equal(t, int64(5), fail, "窗口内限流数不符合预期")
}

// 测试2: 窗口过期后恢复
func (s *TokenLimiterTestSuite) TestWindowExpire() {
	t := s.T()
	ctx := s.ctx

	// 先耗尽窗口内额度
	s.consumeAllTokens(t)

	// 等待窗口过期
	time.Sleep(time.Duration(s.window+1) * time.Second)

	// 验证可以重新请求
	success := 0
	for i := 0; i < int(s.maxCount); i++ {
		if !s.limiter.Limit(ctx, s.userId) {
			success++
		}
	}
	s.redisClient.Del(ctx, fmt.Sprintf("limiter:%s:%s", s.prefix, s.userId))

	assert.Equal(t, int(s.maxCount), success, "窗口过期后未恢复请求额度")
}

// 测试3: 限流开关控制
func (s *TokenLimiterTestSuite) TestLimitSwitch() {
	t := s.T()
	ctx := s.ctx

	// 关闭限流
	_ = s.redisClient.HSet(ctx, redisHashTokenConfig, "limit_enable", "false").Err()
	assert.NoError(t, s.limiter.LoadConfig(ctx))

	// 验证所有请求都通过
	for i := 0; i < int(s.maxCount)+5; i++ {
		assert.False(t, s.limiter.Limit(ctx, s.userId), "关闭限流后仍有限流")
	}

	s.redisClient.Del(ctx, fmt.Sprintf("limiter:%s:%s", s.prefix, s.userId))

	// 重新开启限流
	_ = s.redisClient.HSet(ctx, redisHashTokenConfig, "limit_enable", "true").Err()

	assert.NoError(t, s.limiter.LoadConfig(ctx))
}

// 辅助方法: 耗尽窗口内所有令牌
func (s *TokenLimiterTestSuite) consumeAllTokens(t *testing.T) {
	ctx := s.ctx
	count := 0
	for count < int(s.maxCount) {
		if !s.limiter.Limit(ctx, s.userId) {
			count++
		}
	}
	assert.Equal(t, int(s.maxCount), count, "未能耗尽窗口内所有令牌")
}

// 清理测试环境
func (s *TokenLimiterTestSuite) TearDownSuite() {
	s.cleanupTestData()
}

// 启动测试套件
func TestTokenLimiterSuite(t *testing.T) {
	suite.Run(t, new(TokenLimiterTestSuite))
}
