package throttle

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// BucketTokenLimiterManager 专属测试套件
type TokenBucketPrecisionTestSuite struct {
	suite.Suite
	ctx    context.Context
	prefix string
	//redisClient *redisV9.Client
	limiter *BucketTokenLimiterManager
	userId  string // 测试用固定用户ID
	// 限流规则常量
	burst       int64 // 桶容量（初始令牌数）
	timeWindow  int64 // 时间窗口（秒）
	rate        int64 // 每秒生成令牌数
	startTime   time.Time
	redisClient redis.UniversalClient
}

// 初始化测试环境（预置精准限流配置）
func (s *TokenBucketPrecisionTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.userId = "test_user_bucket_precision"
	s.prefix = "test_bucket_precision"
	// 1. 初始化限流规则常量（替代结构体直接赋值）
	s.burst = 180     // 桶容量（初始令牌数）
	s.timeWindow = 60 // 时间窗口（秒）
	s.rate = 180      // 每秒生成令牌数

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "172.33.3.141:6379",
		Password: "",
		DB:       11,
	})
	assert.NoError(s.T(), redisClient.Ping(s.ctx).Err())
	s.redisClient = redisClient

	// 2. 清理残留测试数据
	s.cleanupTestData()

	// 3. 写入精准限流配置（1分钟窗口、初始180个、每秒3个）
	// 配置逻辑：maxCount=burst=180，timeWindow=60秒 → rate=180/60=3个/秒
	config := map[string]interface{}{
		"limit_enable": "true",
		"max_count":    s.burst,      // 桶容量=180
		"time_window":  s.timeWindow, // 时间窗口=60秒
	}
	err := redisClient.HMSet(s.ctx, redisHashTokenConfig, config).Err()
	assert.NoError(s.T(), err)

	// 4. 初始化限流器
	limiter, err := NewBucketTokenLimiterManager(
		s.prefix,      // Redis Key前缀
		1*time.Minute, // 配置缓存TTL
		redisClient,
	)
	assert.NoError(s.T(), err)
	s.limiter = limiter
	assert.NoError(s.T(), s.limiter.LoadConfig(s.ctx)) // 加载配置
	s.startTime = time.Now()
}

// 清理测试数据（避免干扰）
func (s *TokenBucketPrecisionTestSuite) cleanupTestData() {
	// 删除配置和限流Key
	//_ = redis.GetRedis(s.ctx).Client().Del(s.ctx, redisHashTokenConfig).Err()
	//_ = redis.GetRedis(s.ctx).Client().Del(s.ctx, fmt.Sprintf("test_bucket_precision:%s", s.userId)).Err()
}

// 测试1：初始令牌耗尽测试（180个令牌，并发消耗验证）
func (s *TokenBucketPrecisionTestSuite) TestInitialTokensExhaust() {
	s.startTime = time.Now()
	t := s.T()
	ctx := s.ctx
	var (
		wg          sync.WaitGroup
		successCnt  int64 = 0   // 成功获取令牌数
		failCnt     int64 = 0   // 限流数
		totalReq    int64 = 200 // 总请求数（超过初始令牌数）
		concurrency       = 50  // 并发协程数
	)

	// 并发请求生成器
	reqChan := make(chan struct{}, totalReq)
	for i := 0; i < int(totalReq); i++ {
		reqChan <- struct{}{}
	}
	close(reqChan)

	// 启动并发协程消耗令牌
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			for range reqChan {
				isLimited := s.limiter.Limit(ctx, s.userId)
				if isLimited {
					atomic.AddInt64(&failCnt, 1)
				} else {
					atomic.AddInt64(&successCnt, 1)
				}
			}
		}()
	}
	wg.Wait()

	//rate:limiter:bucket:test_bucket_precision:test_user_bucket_precision
	s.redisClient.Del(ctx, fmt.Sprintf("rate:limiter:bucket:%s:%s", s.prefix, s.userId))

	diff := time.Since(s.startTime)
	fmt.Printf("并发消耗令牌耗时：%v\n", diff)

	expectSuccess := int(s.burst) + int(math.Ceil(diff.Seconds()*(float64(s.burst)/float64(s.timeWindow))))
	expectFail := int(totalReq) - expectSuccess

	// 断言：成功数≈180（初始令牌），失败数≈20（超出部分）
	// 允许±1误差（并发时序问题）
	assert.InDelta(t, expectSuccess, successCnt, 2, fmt.Sprintf("初始令牌消耗数不符合预期（期望：%d，实际：%d）", expectSuccess, successCnt))
	assert.InDelta(t, expectFail, failCnt, 2, fmt.Sprintf("超出初始令牌的限流数不符合预期（期望：%d，实际：%d）", expectFail, failCnt))
}

func (s *TokenBucketPrecisionTestSuite) calcExpectedSuccessTokens() int {
	diff := time.Since(s.startTime)
	return int(s.burst) + int(math.Ceil(diff.Seconds()*(float64(s.burst)/float64(s.timeWindow))))
}

// 测试2：令牌生成速率验证（休眠10秒，验证生成30个令牌）
func (s *TokenBucketPrecisionTestSuite) TestTokenGenerateRate() {
	s.startTime = time.Now()
	t := s.T()
	ctx := s.ctx

	// 步骤1：先耗尽初始令牌（180个）
	s.consumeAllTokens(t)

	// 步骤2：休眠10秒 → 应生成 10*3=30 个令牌
	sleepDuration := 10 * time.Second
	time.Sleep(sleepDuration)

	// 步骤3：并发尝试获取30个令牌
	var (
		wg         sync.WaitGroup
		successCnt int64 = 0
		failCnt    int64 = 0
		testCnt    int   = 50 // 测试请求数（30个应成功，5个限流）
	)

	wg.Add(10)
	reqChan := make(chan struct{}, testCnt)
	for i := 0; i < int(testCnt); i++ {
		reqChan <- struct{}{}
	}
	close(reqChan)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			for range reqChan {
				isLimited := s.limiter.Limit(ctx, s.userId)
				if isLimited {
					atomic.AddInt64(&failCnt, 1)
				} else {
					atomic.AddInt64(&successCnt, 1)
				}
			}
		}()
	}
	wg.Wait()

	s.redisClient.Del(ctx, fmt.Sprintf("rate:limiter:bucket:%s:%s", s.prefix, s.userId))

	diff := time.Since(s.startTime)
	fmt.Printf("并发消耗令牌耗时：%v\n", diff)

	expectSuccess := int(math.Ceil(diff.Seconds() * (float64(s.burst) / float64(s.timeWindow))))
	expectFail := testCnt - expectSuccess

	// 断言：成功数≈180（初始令牌），失败数≈20（超出部分）
	// 允许±1误差（并发时序问题）
	assert.InDelta(t, expectSuccess, successCnt, 2, fmt.Sprintf("初始令牌消耗数不符合预期（期望：%d，实际：%d）", expectSuccess, successCnt))
	assert.InDelta(t, expectFail, failCnt, 2, fmt.Sprintf("超出初始令牌的限流数不符合预期（期望：%d，实际：%d）", expectFail, failCnt))

}

// 测试3：长时间令牌恢复（休眠60秒，验证桶满180个）
func (s *TokenBucketPrecisionTestSuite) TestTokenFullRecovery() {
	s.startTime = time.Now()
	t := s.T()
	ctx := s.ctx

	// 步骤1：耗尽所有令牌
	s.consumeAllTokens(t)

	// 步骤2：休眠60秒 → 生成 60*3=180 个令牌（桶满）
	time.Sleep(60 * time.Second)

	// 步骤3：尝试获取180个令牌（应全部成功）
	var (
		wg         sync.WaitGroup
		successCnt int64 = 0
		failCnt    int64 = 0
	)

	wg.Add(20)
	reqChan := make(chan struct{}, s.burst)
	for i := 0; i < int(s.burst); i++ {
		reqChan <- struct{}{}
	}
	close(reqChan)

	for i := 0; i < 20; i++ {
		go func() {
			defer wg.Done()
			for range reqChan {
				isLimited := s.limiter.Limit(ctx, s.userId)
				if isLimited {
					atomic.AddInt64(&failCnt, 1)
				} else {
					atomic.AddInt64(&successCnt, 1)
				}
			}
		}()
	}
	wg.Wait()
	//rate:limiter:bucket:test_bucket_precision:test_user_bucket_precision
	s.redisClient.Del(ctx, fmt.Sprintf("rate:limiter:bucket:%s:%s", s.prefix, s.userId))

	diff := time.Since(s.startTime)
	fmt.Printf("并发消耗令牌耗时：%v\n", diff)

	expectSuccess := int(math.Ceil(diff.Seconds() * (float64(s.burst) / float64(s.timeWindow))))
	if expectSuccess > int(s.burst) {
		expectSuccess = int(s.burst)
	}
	expectFail := int(s.burst) - expectSuccess

	// 断言：成功数≈180（初始令牌），失败数≈20（超出部分）
	// 允许±1误差（并发时序问题）

	assert.InDelta(t, expectSuccess, successCnt, 2, fmt.Sprintf("初始令牌消耗数不符合预期（期望：%d，实际：%d）", expectSuccess, successCnt))
	assert.InDelta(t, expectFail, failCnt, 2, fmt.Sprintf("超出初始令牌的限流数不符合预期（期望：%d，实际：%d）", expectFail, failCnt))
}

// 测试4：混合场景（持续请求+令牌生成）
func (s *TokenBucketPrecisionTestSuite) TestContinuousRequestWithTokenGenerate() {
	s.startTime = time.Now()
	t := s.T()
	ctx := s.ctx
	var (
		wg           sync.WaitGroup
		totalSuccess int64 = 0
		duration           = 20 * time.Second // 持续请求20秒
	)

	// 启动5个协程，持续20秒请求
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			defer wg.Done()
			start := time.Now()
			for time.Since(start) < duration {
				isLimited := s.limiter.Limit(ctx, s.userId)
				if !isLimited {
					atomic.AddInt64(&totalSuccess, 1)
				}
				time.Sleep(10 * time.Millisecond) // 避免空循环占满CPU
			}
		}()
	}
	wg.Wait()

	s.redisClient.Del(ctx, fmt.Sprintf("rate:limiter:bucket:%s:%s", s.prefix, s.userId))

	diff := time.Since(s.startTime)
	expectSuccess := int(s.burst) + int(math.Ceil(diff.Seconds()*(float64(s.burst)/float64(s.timeWindow))))
	// 预期成功数：初始180 + 20秒生成60 = 240个（允许±5误差）
	//expected := s.burst + s.rate*20
	fmt.Printf("持续请求20秒耗时：%v, 预期成功数：%d, 实际成功数：%d\n", diff, expectSuccess, totalSuccess)
	assert.InDelta(t, expectSuccess, totalSuccess, 5, "持续请求20秒的成功数不符合预期")
}

// 辅助方法：耗尽所有令牌
func (s *TokenBucketPrecisionTestSuite) consumeAllTokens(t *testing.T) {
	var (
		wg         sync.WaitGroup
		successCnt int64 = 0
	)
	totalReq := s.burst
	wg.Add(30)
	reqChan := make(chan struct{}, totalReq) // 多请求10个确保耗尽
	for i := 0; i < int(totalReq); i++ {
		reqChan <- struct{}{}
	}
	close(reqChan)

	for i := 0; i < 30; i++ {
		go func() {
			defer wg.Done()
			for range reqChan {
				isLimited := s.limiter.Limit(context.Background(), s.userId)
				if !isLimited {
					atomic.AddInt64(&successCnt, 1)
				}
			}
		}()
	}
	wg.Wait()

	assert.InDelta(t, s.burst, successCnt, 1, fmt.Sprintf("初始令牌未完全耗尽（期望：%d，实际：%d）", s.burst, successCnt))
}

// 清理测试环境
func (s *TokenBucketPrecisionTestSuite) TearDownSuite() {
	s.cleanupTestData()
}

// 启动测试套件
func TestTokenBucketPrecisionSuite(t *testing.T) {
	suite.Run(t, new(TokenBucketPrecisionTestSuite))
}
