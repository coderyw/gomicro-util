package throttle

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// 性能对比测试套件
type LimiterPerformanceTestSuite struct {
	suite.Suite
	ctx           context.Context
	redisClient   redis.UniversalClient
	windowLimiter *TokenLimiterManager       // 窗口令牌限流器
	bucketLimiter *BucketTokenLimiterManager // 桶令牌限流器
	userId        string                     // 测试用户ID
	testDuration  time.Duration              // 测试持续时间
	concurrency   int                        // 并发协程数
}

// SetupSuite 初始化测试环境
func (s *LimiterPerformanceTestSuite) SetupSuite() {
	runtime.GOMAXPROCS(2 * runtime.NumCPU())
	s.ctx = context.Background()
	s.userId = "performance_test_user"
	s.testDuration = 30 * time.Second // 测试持续10秒
	s.concurrency = 500               // 1000个并发协程

	s.redisClient = redis.NewClient(&redis.Options{
		//Addr:     "172.33.3.141:6379",
		Addr:     "172.35.5.24:6379",
		Password: "",
		//DB:       11,
	})

	assert.NoError(s.T(), s.redisClient.Ping(s.ctx).Err())

	config := map[string]interface{}{
		"limit_enable": "true",
		"max_count":    180, // 桶容量=180
		"time_window":  60,  // 时间窗口=60秒
	}
	err := s.redisClient.HMSet(s.ctx, redisHashTokenConfig, config).Err()
	assert.NoError(s.T(), err)
	// 初始化限流器（假设已实现对应的构造函数）
	s.windowLimiter, _ = NewTokenLimiterManager("performance", 10*time.Second, s.redisClient)
	s.bucketLimiter, _ = NewBucketTokenLimiterManager("performance", 10*time.Second, s.redisClient)
	_ = s.windowLimiter.LoadConfig(s.ctx)
	_ = s.bucketLimiter.LoadConfig(s.ctx)
}

// TearDownSuite 清理测试环境
func (s *LimiterPerformanceTestSuite) TearDownSuite() {
	// 关闭Redis连接
	if err := s.redisClient.Close(); err != nil {
		s.T().Errorf("关闭Redis连接失败: %v", err)
	}
}

// TestLimiterPerformance 对比两种限流器的性能
func (s *LimiterPerformanceTestSuite) TestLimiterPerformance() {
	// 测试窗口令牌限流器
	windowCount := s.runPerformanceTest(s.windowLimiter.Limit)
	// 测试桶令牌限流器
	bucketCount := s.runPerformanceTest(s.bucketLimiter.Limit)

	// 输出结果
	s.T().Logf("窗口令牌限流器在%d秒内处理了%d次调用",
		int(s.testDuration.Seconds()), windowCount)
	s.T().Logf("桶令牌限流器在%d秒内处理了%d次调用",
		int(s.testDuration.Seconds()), bucketCount)

	// 断言性能差异（仅作日志展示，不强制失败）
	if windowCount > bucketCount {
		s.T().Log("窗口令牌限流器性能更优")
	} else {
		s.T().Log("桶令牌限流器性能更优")
	}
}

// 执行性能测试
func (s *LimiterPerformanceTestSuite) runPerformanceTest(limitFunc func(ctx context.Context, userId string) bool) int64 {
	var totalCalls int64 = 0
	var wg sync.WaitGroup
	startTime := time.Now()

	// 启动并发协程
	for i := 0; i < s.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 在测试时间内持续调用限流接口
			for time.Since(startTime) < s.testDuration {
				// 调用限流方法（忽略返回值，只关注调用次数）
				limitFunc(s.ctx, s.userId)
				atomic.AddInt64(&totalCalls, 1)
			}
		}()
	}

	wg.Wait()
	return totalCalls
}

// 启动测试套件
func TestLimiterPerformanceSuite(t *testing.T) {
	suite.Run(t, new(LimiterPerformanceTestSuite))
}
