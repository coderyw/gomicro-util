package throttle

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// BucketTokenApiLimiterManager 专属测试套件
type TokenApiBucketPrecisionTestSuite struct {
	suite.Suite
	ctx        context.Context
	limiter    *BucketTokenApiLimiterManager
	userId     string // 测试用固定用户ID
	prefix     string
	api        string // 测试用固定API
	specialApi string
	// 限流规则常量
	burst       int64 // 桶容量（初始令牌数）
	timeWindow  int64 // 时间窗口（秒）
	rate        int64 // 每秒生成令牌数
	startTime   time.Time
	redisClient redis.UniversalClient
}

// 初始化测试环境（预置精准限流配置）
func (s *TokenApiBucketPrecisionTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.userId = "test_user_api_bucket_precision"
	s.prefix = "test_api_bucket_precision"
	s.api = "_api_precision"
	s.specialApi = "_precision"

	// 初始化限流规则常量
	s.burst = 180     // 桶容量（初始令牌数）
	s.timeWindow = 60 // 时间窗口（秒）
	s.rate = 3        // 每秒生成令牌数（180/60）

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "172.33.3.141:6379",
		Password: "",
		DB:       11,
	})
	assert.NoError(s.T(), redisClient.Ping(s.ctx).Err())

	s.redisClient = redisClient
	// 清理残留测试数据
	s.cleanupTestData()

	// 写入精准限流配置（1分钟窗口、初始180个、每秒3个）
	config := map[string]interface{}{
		"api_limit_enable": "true",
		"api_max_count":    s.burst,      // 桶容量=180
		"api_time_window":  s.timeWindow, // 时间窗口=60秒
	}
	err := redisClient.HMSet(s.ctx, redisHashTokenApiConfig, config).Err()
	assert.NoError(s.T(), err)

	// 初始化限流器
	limiter, err := NewBucketTokenApiLimiterManager(
		s.prefix,      // Redis Key前缀
		1*time.Minute, // 配置缓存TTL
		redisClient,   // Redis实例名
	)
	assert.NoError(s.T(), err)
	s.limiter = limiter
	assert.NoError(s.T(), s.limiter.LoadConfig(s.ctx)) // 加载配置
	s.startTime = time.Now()
}

// 清理测试数据（避免干扰）
func (s *TokenApiBucketPrecisionTestSuite) cleanupTestData() {
	//_ = redis.GetRedis(s.ctx).Client().Del(s.ctx, redisHashTokenApiConfig).Err()
	//limiterKey := s.limiter.getLimiterKey(s.userId, s.api)
	//_ = redis.GetRedis(s.ctx).Client().Del(s.ctx, fmt.Sprintf("limiter:bucket:test_api_bucket_precision:%s", limiterKey)).Err()
}

// 测试1：初始令牌耗尽测试（180个令牌，并发消耗验证）
func (s *TokenApiBucketPrecisionTestSuite) TestInitialTokensExhaust() {
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
				isLimited := s.limiter.Limit(ctx, s.userId, s.api)
				if isLimited {
					atomic.AddInt64(&failCnt, 1)
				} else {
					atomic.AddInt64(&successCnt, 1)
				}
			}
		}()
	}
	wg.Wait()

	diff := time.Since(s.startTime)
	fmt.Printf("并发消耗令牌耗时：%v\n", diff)

	expectSuccess := int(s.burst) + int(math.Ceil(diff.Seconds()*float64(s.rate)))
	expectFail := int(totalReq) - expectSuccess

	s.redisClient.Del(ctx, fmt.Sprintf("rate:limiter:bucket:%s:%s%s", s.prefix, s.userId, s.api))
	s.redisClient.Del(ctx, fmt.Sprintf("rate:limiter:bucket:%s:%s%s", s.prefix, s.userId, s.specialApi))

	// 断言：成功数≈180（初始令牌），失败数≈20（超出部分）
	assert.InDelta(t, expectSuccess, successCnt, 2, fmt.Sprintf("初始令牌消耗数不符合预期（期望：%d，实际：%d）", expectSuccess, successCnt))
	assert.InDelta(t, expectFail, failCnt, 2, fmt.Sprintf("超出初始令牌的限流数不符合预期（期望：%d，实际：%d）", expectFail, failCnt))
}

// 测试2：令牌生成速率验证（休眠10秒，验证生成30个令牌）
func (s *TokenApiBucketPrecisionTestSuite) TestTokenGenerateRate() {
	s.startTime = time.Now()
	t := s.T()
	ctx := s.ctx

	// 步骤1：先耗尽初始令牌（180个）
	s.consumeAllTokens(t)

	// 步骤2：休眠10秒 → 应生成 10*3=30 个令牌
	sleepDuration := 10 * time.Second
	time.Sleep(sleepDuration)

	// 步骤3：并发尝试获取令牌
	var (
		wg         sync.WaitGroup
		successCnt int64 = 0
		failCnt    int64 = 0
		testCnt    int   = 50 // 测试请求数（30个应成功，20个限流）
	)

	wg.Add(10)
	reqChan := make(chan struct{}, testCnt)
	for i := 0; i < testCnt; i++ {
		reqChan <- struct{}{}
	}
	close(reqChan)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			for range reqChan {
				isLimited := s.limiter.Limit(ctx, s.userId, s.api)
				if isLimited {
					atomic.AddInt64(&failCnt, 1)
				} else {
					atomic.AddInt64(&successCnt, 1)
				}
			}
		}()
	}
	wg.Wait()

	diff := time.Since(s.startTime)
	fmt.Printf("并发消耗令牌耗时：%v\n", diff)

	expectSuccess := int(math.Ceil(diff.Seconds() * float64(s.rate)))
	if expectSuccess > int(s.burst) {
		expectSuccess = int(s.burst)
	}
	expectFail := testCnt - expectSuccess

	s.redisClient.Del(ctx, fmt.Sprintf("rate:limiter:bucket:%s:%s%s", s.prefix, s.userId, s.api))
	s.redisClient.Del(ctx, fmt.Sprintf("rate:limiter:bucket:%s:%s%s", s.prefix, s.userId, s.specialApi))

	assert.InDelta(t, expectSuccess, successCnt, 2, fmt.Sprintf("令牌生成速率验证失败（期望：%d，实际：%d）", expectSuccess, successCnt))
	assert.InDelta(t, expectFail, failCnt, 2, fmt.Sprintf("令牌生成速率限流数验证失败（期望：%d，实际：%d）", expectFail, failCnt))
}

// 测试3：长时间令牌恢复（休眠60秒，验证桶满180个）
func (s *TokenApiBucketPrecisionTestSuite) TestTokenFullRecovery() {
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
				isLimited := s.limiter.Limit(ctx, s.userId, s.api)
				if isLimited {
					atomic.AddInt64(&failCnt, 1)
				} else {
					atomic.AddInt64(&successCnt, 1)
				}
			}
		}()
	}
	wg.Wait()

	diff := time.Since(s.startTime)
	fmt.Printf("并发消耗令牌耗时：%v\n", diff)

	expectSuccess := int(math.Ceil(diff.Seconds() * float64(s.rate)))
	if expectSuccess > int(s.burst) {
		expectSuccess = int(s.burst)
	}
	expectFail := int(s.burst) - expectSuccess

	s.userId = "test_user_api_bucket_precision"
	s.prefix = "test_api_bucket_precision"
	s.api = "_api_precision"

	//rate:limiter:bucket:test_api_bucket_precision:test_user_api_bucket_precision_api_precision
	s.redisClient.Del(ctx, fmt.Sprintf("rate:limiter:bucket:%s:%s%s", s.prefix, s.userId, s.api))
	s.redisClient.Del(ctx, fmt.Sprintf("rate:limiter:bucket:%s:%s%s", s.prefix, s.userId, s.specialApi))

	assert.InDelta(t, expectSuccess, successCnt, 5, fmt.Sprintf("令牌恢复验证失败（期望：%d，实际：%d）", expectSuccess, successCnt))
	assert.InDelta(t, expectFail, failCnt, 5, fmt.Sprintf("令牌恢复限流数验证失败（期望：%d，实际：%d）", expectFail, failCnt))
}

// 测试4：混合场景（持续请求+令牌生成）
func (s *TokenApiBucketPrecisionTestSuite) TestContinuousRequestWithTokenGenerate() {
	s.startTime = time.Now()
	t := s.T()
	ctx := s.ctx
	var (
		wg            sync.WaitGroup
		totalSuccess  int64 = 0
		totalSuccess2 int64 = 0
		duration            = 20 * time.Second // 持续请求20秒
	)

	// 启动5个协程，持续20秒请求
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			defer wg.Done()
			start := time.Now()
			for time.Since(start) < duration {
				isLimited2 := s.limiter.Limit(ctx, s.userId, s.specialApi)
				isLimited := s.limiter.Limit(ctx, s.userId, s.api)
				if !isLimited {
					atomic.AddInt64(&totalSuccess, 1)
				}
				if !isLimited2 {
					atomic.AddInt64(&totalSuccess2, 1)
				}
				time.Sleep(10 * time.Millisecond) // 避免空循环占满CPU
			}
		}()
	}
	wg.Wait()

	diff := time.Since(s.startTime)
	expectSuccess := int(s.burst) + int(math.Ceil(diff.Seconds()*float64(s.rate)))
	if expectSuccess > int(s.burst)+int(s.rate)*20 {
		expectSuccess = int(s.burst) + int(s.rate)*20
	}

	//rate:limiter:bucket:test_api_bucket_precision:test_user_api_bucket_precisionthe_second_api
	s.redisClient.Del(ctx, fmt.Sprintf("rate:limiter:bucket:%s:%s%s", s.prefix, s.userId, s.api))
	s.redisClient.Del(ctx, fmt.Sprintf("rate:limiter:bucket:%s:%s%s", s.prefix, s.userId, s.specialApi))

	fmt.Printf("持续请求20秒耗时：%v, 预期成功数：%d, 实际成功数：%d, api is %s\n", diff, expectSuccess, totalSuccess, s.api)
	assert.InDelta(t, expectSuccess, totalSuccess, 8, "持续请求20秒的成功数不符合预期")

	fmt.Printf("持续请求20秒耗时：%v, 预期成功数：%d, 实际成功数：%d, api is %s\n", diff, expectSuccess, totalSuccess2, "the_second_api")
	assert.InDelta(t, expectSuccess, totalSuccess2, 8, "持续请求20秒的成功数不符合预期")
}

// 测试5：API维度隔离性验证
func (s *TokenApiBucketPrecisionTestSuite) TestApiIsolation() {
	s.startTime = time.Now()
	t := s.T()
	ctx := s.ctx

	// 步骤1：耗尽当前API的令牌
	s.consumeAllTokens(t)

	// 步骤2：验证其他API仍可正常获取令牌
	var successCnt int64 = 0
	for i := 0; i < 50; i++ {
		if !s.limiter.Limit(ctx, s.userId, s.specialApi) {
			successCnt++
		}
	}
	s.redisClient.Del(ctx, fmt.Sprintf("rate:limiter:bucket:%s:%s%s", s.prefix, s.userId, s.api))
	s.redisClient.Del(ctx, fmt.Sprintf("rate:limiter:bucket:%s:%s%s", s.prefix, s.userId, s.specialApi))

	// 预期其他API应能获取到初始令牌
	assert.InDelta(t, 50, successCnt, 2, "API维度隔离性验证失败")
}

// 辅助方法：耗尽所有令牌
func (s *TokenApiBucketPrecisionTestSuite) consumeAllTokens(t *testing.T) {
	var (
		wg         sync.WaitGroup
		successCnt int64 = 0
	)
	totalReq := s.burst
	wg.Add(30)
	reqChan := make(chan struct{}, totalReq)
	for i := 0; i < int(totalReq); i++ {
		reqChan <- struct{}{}
	}
	close(reqChan)

	for i := 0; i < 30; i++ {
		go func() {
			defer wg.Done()
			for range reqChan {
				isLimited := s.limiter.Limit(context.Background(), s.userId, s.api)
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
func (s *TokenApiBucketPrecisionTestSuite) TearDownSuite() {
	s.cleanupTestData()
}

// 启动测试套件
func TestTokenApiBucketPrecisionSuite(t *testing.T) {
	suite.Run(t, new(TokenApiBucketPrecisionTestSuite))
}
