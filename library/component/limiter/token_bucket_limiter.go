// Package limiter
// @Author: yinwei
// @File: token_bucket_limiter
// @Version: 1.0.0
// @Date: 2025/6/19 09:16

package limiter

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type TokenBucketLimiter struct {
	Redis      redis.UniversalClient
	Prefix     string
	Rate       int
	Capacity   int
	RateIp     int
	CapacityIp int
}

func NewTokenBucketLimiter(redisClient redis.UniversalClient) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		Redis:    redisClient,
		Prefix:   "rate_limit:bucket:",
		Rate:     5,
		Capacity: 100,
	}
}

// Lua 脚本：实现令牌桶限流算法
// 该脚本在 Redis 中维护一个令牌桶，用于控制访问频率。
// 令牌桶有两个关键参数：
//   - rate: 每秒新增的令牌数量（即产生速率）
//   - capacity: 令牌桶的最大容量
//
// 算法逻辑：
//  1. 获取当前桶中的令牌数和上次更新时间
//  2. 如果是第一次访问，则初始化令牌数为最大容量
//  3. 根据时间差计算应添加的令牌数，并更新当前令牌数
//  4. 若令牌数大于等于1，则允许访问并减少一个令牌；否则拒绝访问
//  5. 更新 Redis 中的令牌数和时间戳，并设置过期时间以避免内存泄漏
var luaScript = `
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

local bucket = redis.call("HMGET", key, "tokens", "last_time")
local tokens = tonumber(bucket[1])
local last_time = tonumber(bucket[2])

if tokens == nil then
    tokens = capacity
    last_time = now
end

local delta = math.max(0, now - last_time)
local add = delta * rate
tokens = math.min(capacity, tokens + add)
local allowed = tokens >= 1

if allowed then
    tokens = tokens - 1
end

redis.call("HMSET", key, "tokens", tokens, "last_time", now)
redis.call("EXPIRE", key, math.ceil(capacity / rate * 2))

if allowed then
    return 1
else
    return 0
end
`

var (
	luaSha string
	lk     sync.RWMutex
	isLoad atomic.Bool
)

// Consume 消耗令牌桶中的令牌，用于限流控制
//
// 参数:
//   - ctx: 上下文，用于控制请求的生命周期
//   - key: 限流的唯一标识键，可以是用户ID、API端点等
//   - ip: 可变参数，当传入且为true时，使用IP限流的速率和容量配置
//
// 返回值:
//   - bool: true表示允许通过(有足够的令牌)，false表示被限流
//   - error: 执行过程中的错误信息，nil表示执行成功
func (l *TokenBucketLimiter) Consume(ctx context.Context, key string, ip ...bool) (bool, error) {
	fullKey := l.Prefix + key
	now := float64(time.Now().UnixNano()) / 1e9

	// 根据是否启用IP限流来选择相应的速率和容量配置
	rate := l.Rate
	capacity := l.Capacity
	if len(ip) > 0 && ip[0] {
		rate = l.RateIp
		capacity = l.CapacityIp
	}

	// 加载Lua脚本
	if err := l.loadScript(ctx); err != nil {
		return false, err
	}

	// 执行Redis中的Lua脚本进行令牌桶算法计算
	res, err := l.Redis.EvalSha(ctx, luaSha, []string{fullKey}, rate, capacity, now).Result()
	if err != nil {
		log.Printf("Redis Eval error: %v", err)
		return false, err
	}

	// 检查并转换Redis返回的结果
	allowed, ok := res.(int64)
	if !ok {
		return false, fmt.Errorf("unexpected Redis result: %v", res)
	}
	return allowed == 1, nil
}

// loadScript 加载Lua脚本到Redis中
// 该函数确保脚本只被加载一次，使用双重检查锁定模式来避免并发问题
// 返回值：
//
//	error: 如果脚本加载失败则返回错误，否则返回nil
func (l *TokenBucketLimiter) loadScript(ctx context.Context) error {
	// 检查脚本是否已经加载，如果已加载则直接返回
	if isLoad.Load() {
		return nil
	}

	// 获取锁以确保同一时间只有一个goroutine执行加载操作
	lk.Lock()
	defer lk.Unlock()

	// 再次检查脚本是否已经加载，防止在获取锁之前其他goroutine已经完成加载
	if isLoad.Load() {
		return nil
	}

	// 执行脚本加载命令
	cmd := l.Redis.ScriptLoad(ctx, luaScript)
	res, err := cmd.Result()
	if err != nil {
		return err
	}

	// 保存脚本的SHA值并标记为已加载
	luaSha = res
	isLoad.Store(true)
	return nil
}
