package redis

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	sync2 "sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"github.com/coderyw/gomicro-util/library/db/cache/redis/facade"
	"github.com/coderyw/gomicro-util/library/db/cache/redis/prefixenum"
	"github.com/coderyw/gomicro-util/library/db/cache/redis/sync"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/mock"
	"github.com/coderyw/gomicro-util/library/namespace"
	"github.com/coderyw/gomicro-util/library/pmicro/metrics"
	"github.com/coderyw/gomicro-util/library/trace"
	redisMock "github.com/coderyw/gomicro-util/mocks/library/db/cache/redis/facade"
	config2 "github.com/coderyw/gomicro-util/model/config"
	"github.com/coderyw/gomicro-util/model/cron"
	"github.com/coderyw/gomicro-util/model/micro"
	"github.com/coderyw/gomicro-util/util/operation"
	"go.uber.org/zap"
)

var (
	// clientMap 其他Redis客户端映射表，用于管理多个Redis实例
	redisClientMap = make(map[string]*Redis)
	// defaultName 默认客户端名称
	defaultName  = REDIS_CLIENT_NAME_CORE
	statsSyncMap = sync2.Map{}
)

const (
	// REDIS_CLIENT_NAME_CORE 核心库Redis客户端名称
	REDIS_CLIENT_NAME_CORE      = "core"
	REDIS_CLIENT_NAME_SNOWFLAKE = "snowflake"
)

// GetRedis 获取Redis实例
// 支持Mock模式，用于单元测试
// 参数:
//   - ctx: 上下文对象
//   - database: 可选的实例名称数组。如果不传会使用默认实例名称，默认实例名称可以通过链接配置设置，如果不设置则使用core
//
// 返回值:
//   - RedisInterface: Redis接口实例对象
func GetRedis(ctx context.Context, database ...string) facade.RedisInterface {
	// 判断是否开启Mock模式（用于测试）
	if t := mock.CheckAndGetMockClient[facade.RedisInterface](ctx, func(t *testing.T) facade.RedisInterface {
		var mc interface{} = redisMock.NewMockRedisInterface(t)
		if val, ok := mc.(facade.RedisInterface); ok {
			return val
		}
		return nil
	}); t != nil {
		return t
	}
	// 检查Redis连接是否已初始化
	if len(redisClientMap) == 0 {
		log.FatalCtx(ctx, fmt.Sprint("获取redis连接为空"))
		return nil
	}
	// 确定使用哪个数据库实例
	var name string
	if len(database) == 0 {
		name = defaultName
	} else {
		name = database[0]
	}
	log.Debug("获取Redis链接", zap.String("database", name))
	// 从客户端映射表中获取指定的客户端
	c, ok := redisClientMap[name]
	if !ok {
		log.FatalCtx(ctx, fmt.Sprintf("获取redis连接为空: %v", database))
	}
	return c
}

// 获取雪花 id 实例，如果没配置，则使用默认的实例
func SnowflakeOrDefault() facade.RedisInterface {
	val, ok := redisClientMap[REDIS_CLIENT_NAME_SNOWFLAKE]
	if ok {
		log.Debug("获取Redis链接", zap.String("database", REDIS_CLIENT_NAME_SNOWFLAKE))
		return val
	}
	return GetRedis(context.Background())
}

// Redis Redis缓存结构体，提供Redis服务器支持的缓存功能
// 封装了redis.UniversalClient，支持单机和集群模式
type Redis struct {
	// UniversalClient Redis通用客户端，支持单机、哨兵、集群等多种模式
	UniversalClient redis.UniversalClient
}

// InitRedis 初始化Redis连接
// 根据配置初始化多个Redis实例，支持单机和集群模式
// 参数:
//   - conf: Redis配置对象
//
// 返回值:
//   - c: 微服务关闭函数结构
//   - err: 初始化过程中的错误
func InitRedis(conf config2.RedisConfig) (c micro.CloseS, job *cron.CronJob, err error) {
	// 如果Redis未启用，直接返回
	if !conf.Enable {
		return c, nil, nil
	}
	// 配置默认连接名称
	if len(conf.Default) > 0 {
		defaultName = conf.Default
	}
	// 遍历配置中的所有Redis实例并初始化
	for k, v := range conf.Instance {
		if !v.Enable {
			continue
		}
		err := initOneRedis(v.Host, v.Password, v.Db, v.IsCluster, v.PoolSize, v.MaxIdleConns, v.Debug, k)
		if err != nil {
			return c, nil, err
		}
	}
	// 设置关闭函数和权重
	// 初始化同步
	if mgr, err := sync.NewRedisSyncMgr(); err != nil {
		return c, nil, err
	} else {
		syncMgr = mgr
	}
	c.CloseFunc = stop
	c.Weight = -1 // 权重比较低，需要最后关闭
	// 加载 redis 的所有命令 spec（SAAS 多租户 key 前缀依赖）
	if err := initKeySpec(); err != nil {
		return c, nil, err
	}
	// 返回 Redis 连接池状态监控定时任务（NS 新增，适配 redisClientMap 结构）
	// 连接池统计间隔，单位 s。默认 15s 一次，-1 表示不开启
	if conf.PoolStatsInterval == 0 {
		conf.PoolStatsInterval = 15
	}
	// 连接池统计模式：-1 不打印不上报，0 默认（等同1），1 仅日志打印，2 仅指标上报，3 都开启
	statsLogEnabled := config2.PoolStatsLogEnabled(conf.PoolStatsMode)
	statsReportEnabled := config2.PoolStatsReportEnabled(conf.PoolStatsMode)
	var cj *cron.CronJob
	if conf.PoolStatsInterval > 0 && (statsLogEnabled || statsReportEnabled) {
		cj = &cron.CronJob{
			Name: "Redis连接池状态",
			DoJob: func() {
				for k, v := range redisClientMap {
					stats := v.UniversalClient.PoolStats()
					if statsLogEnabled {
						val, ok := statsSyncMap.Load(k)
						if !ok || val.(redis.PoolStats) != *stats { // 减少打印次数
							log.Info("Redis连接监控",
								zap.String("instance", k),
								zap.Uint32("total", stats.TotalConns),                //当前连接池总连接数
								zap.Uint32("idle", stats.IdleConns),                  //当前空闲连接数
								zap.Uint32("busy", stats.TotalConns-stats.IdleConns), //正在使用的连接数 TotalConns - IdleConns
								zap.Uint32("hit", stats.Hits),                        //从池中成功获取连接次数
								zap.Uint32("miss", stats.Misses),                     //池中无空闲连接，需要创建新连接次数
								zap.Uint32("timeout", stats.Timeouts),                //获取连接超时次数
								zap.Uint32("stale", stats.StaleConns),                //被关闭的失效连接数
							)
							statsSyncMap.Store(k, *stats)
						}
					}
					if statsReportEnabled {
						// 推送到 pmicro/metrics（metrics 未开启则由其内部丢弃）
						tags := metrics.Tags{"instance": k}
						metrics.SetGauge("redis_pool_total_conns", tags, float64(stats.TotalConns))
						metrics.SetGauge("redis_pool_idle_conns", tags, float64(stats.IdleConns))
						metrics.SetGauge("redis_pool_busy_conns", tags, float64(stats.TotalConns-stats.IdleConns))
						metrics.SetGauge("redis_pool_hits", tags, float64(stats.Hits))
						metrics.SetGauge("redis_pool_misses", tags, float64(stats.Misses))
						metrics.SetGauge("redis_pool_timeouts", tags, float64(stats.Timeouts))
						metrics.SetGauge("redis_pool_stale_conns", tags, float64(stats.StaleConns))
					}
				}
			},
			CronSpc:         fmt.Sprintf("@every %ds", conf.PoolStatsInterval),
			DoBeforeDone:    false,
			DelayBeforeDone: false,
		}
	}

	return c, cj, nil
}

// initOneRedis 初始化单个Redis连接
// 参数:
//   - host: Redis服务器地址列表
//   - password: Redis密码
//   - db: 数据库索引
//   - isCluster: 是否为集群模式
//   - poolSize: 连接池大小
//   - maxIdleConns: 最大空闲连接数
//   - debug: 是否启用调试模式
//   - name: Redis实例名称
//
// 返回值:
//   - error: 初始化过程中的错误
func initOneRedis(host []string, password string, db int, isCluster bool, poolSize int, maxIdleConns int, debug bool, name string) error {
	// 检查实例名称是否为空
	if len(name) == 0 {
		return errors.New("redis name is nil")
	}
	addr := host
	// 检查Redis主机地址是否为空
	if len(addr) == 0 {
		return errors.New("redis host is nil")
	}
	var client redis.UniversalClient
	// 设置连接池大小默认值
	if poolSize == 0 {
		poolSize = 20
	}
	// 设置最大空闲连接数默认值
	if maxIdleConns == 0 {
		maxIdleConns = poolSize / 10
	}
	// 根据是否为集群模式创建不同类型的Redis客户端
	if isCluster {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}
		// 创建集群模式Redis客户端
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:          addr,                                            // Redis集群节点地址列表
			Password:       password,                                        // Redis密码
			DialTimeout:    time.Second * 3,                                 // 连接超时时间
			PoolSize:       poolSize,                                        // 连接池大小
			MaxIdleConns:   maxIdleConns,                                    // 最大空闲连接数
			MaxActiveConns: operation.Ternary(poolSize == 0, 100, poolSize), // 最大活跃连接数
			TLSConfig:      tlsConfig,                                       // 开启 TLS
		})
	} else {
		// 创建单机模式Redis客户端
		client = redis.NewClient(&redis.Options{
			Addr:           addr[0],                                         // Redis服务器地址
			Password:       password,                                        // Redis密码
			DB:             db,                                              // 数据库索引
			DialTimeout:    time.Second * 3,                                 // 连接超时时间
			PoolSize:       poolSize,                                        // 连接池大小
			MaxIdleConns:   maxIdleConns,                                    // 最大空闲连接数
			MaxActiveConns: operation.Ternary(poolSize == 0, 100, poolSize), // 最大活跃连接数
		})
	}
	trace.RedisTrace(client)
	// Redis日志初始化
	redisLog := log.NewRedisLog()
	// 设置Redis全局日志记录器
	redis.SetLogger(redisLog)
	// 如果启用调试模式，添加日志钩子用于记录Redis操作
	client.AddHook(&logHook{
		debug:    debug,
		logger:   redisLog.Logger, // 日志记录器
		instance: name,            // Redis实例名称
	})
	// key 格式
	//client.AddHook(&keyPrefixHook{})
	client.AddHook(NewPrefixHookV3())

	// 测试Redis连接是否正常
	if err := client.Ping(context.TODO()).Err(); err != nil {
		return errors.WithStack(err)
	}
	// 记录Redis初始化成功日志
	log.InfoCtxLogger(context.Background(), redisLog.Logger, "初始化redis成功", zap.Strings("host", host), zap.String("version", redis.Version()))

	redisClientMap[name] = &Redis{UniversalClient: client}

	return nil
}

// stop 停止所有Redis连接
// 遍历所有Redis客户端并关闭连接，释放资源
func stop() {
	// 遍历客户端映射表，关闭所有Redis连接
	for _, v := range redisClientMap {
		v.UniversalClient.Close()
	}
	log.Info("停止redis成功")
}

// CheckIsNOSCRIPT 检查错误是否是 脚本不存在错误
//
// Parameters:
//   - err: error
//
// Returns:
//   - bool true=是脚本不存在错误；false=不是
func CheckIsNOSCRIPT(err error) bool {
	if err == nil {
		return false
	}
	if strings.Contains(err.Error(), "NOSCRIPT") {
		return true
	}
	return false
}

func (r *Redis) Client() redis.UniversalClient {
	return r.UniversalClient
}

// GetValue 根据key获取字符串值
//
// Parameters:
//   - ctx: context.Context
//   - prefix: PrefixEnum key前缀
//   - key: string 要获取的key
//
// Returns:
//   - string 返回redis中的value
//   - error
func (r *Redis) GetValue(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (string, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.Get(ctx, k).Result()
}

// GetValueByInt64 获取key对应的redis int64值
//
// Parameters:
//   - ctx: context.Context 上下文
//   - prefix: PrefixEnum key前缀
//   - key: string 要获取的key
//
// Returns:
//   - int64 返回redis中的value
//   - error
func (r *Redis) GetValueByInt64(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (int64, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.Get(ctx, k).Int64()
}

// GetValueBytes 返回key对应的redis bytes值
//
// Parameters:
//   - ctx: context.Context 上下文
//   - prefix: PrefixEnum key 前缀
//   - key: string key
//
// Returns:
//   - []byte
//   - error
func (r *Redis) GetValueBytes(ctx context.Context, prefix prefixenum.PrefixEnum, key string) ([]byte, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.Get(ctx, k).Bytes()
}

// MGetValue 根据keys批量获取key对应的redis value
//
// Parameters:
//   - ctx: context.Context 上下文
//   - prefix: PrefixEnum key前缀
//   - keys: []string key 名
//
// Returns:
//   - []interface{}
//   - error
func (r *Redis) MGetValue(ctx context.Context, prefix prefixenum.PrefixEnum, keys []string) ([]interface{}, error) {
	ks := make([]string, 0, len(keys))
	for _, value := range keys {
		ks = append(ks, getKey(prefix, value))
	}
	return r.UniversalClient.MGet(ctx, ks...).Result()
}

// MSetValue 批量设置key对应的redis value
//
// Parameters:
//   - ctx: context.Context 上下文呢
//   - prefix: PrefixEnum key 前缀
//   - keys: []string 要设置的 key
//   - value: interface{} 要设置的value
//
// Returns:
//   - string
//   - error
func (r *Redis) MSetValue(ctx context.Context, prefix prefixenum.PrefixEnum, keys []string, value interface{}) (string, error) {
	msetData := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		userKey := getKey(prefix, key)
		msetData = append(msetData, userKey, value)
	}

	return r.UniversalClient.MSet(ctx, msetData).Result()
}

// Setnx 设置值，如果key不存在，则设置成功，返回true；如果key存在，则返回false。
//
// Parameters:
//   - ctx: context.Context 上下文
//   - prefix: PrefixEnum key 前缀
//   - key: string key
//   - value: interface{} 设置的 value
//   - expire: time.Duration 过期时间
//
// Returns:
//   - bool: true=设置成功，false=设置失败
//   - error 错误信息
func (r *Redis) Setnx(ctx context.Context, prefix prefixenum.PrefixEnum, key string, value interface{}, expire time.Duration) (bool, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.SetNX(ctx, k, value, expire).Result()
}

// SetValue 设置值
//
// Parameters:
//   - ctx: context.Context 上下文
//   - prefix: PrefixEnum key 前缀
//   - key: string key
//   - value: interface{} value
//   - expire: time.Duration 过期时间
//
// Returns:
//   - error 错误信息
func (r *Redis) SetValue(ctx context.Context, prefix prefixenum.PrefixEnum, key string, value interface{}, expire time.Duration) error {
	k := getKey(prefix, key)
	return r.UniversalClient.Set(ctx, k, value, expire).Err()
}

// SetValueExist 设置值
//
// 设置值，如果缓存的 value 与设置的value一致，则返回false; 否则更新key的value,然后返回true。
//
// Parameters:
//   - prefix: key前缀
//   - key: 要设置的key
//   - value: 要设置的value
//   - expire: 过期时间
//
// Returns:
//   - bool: true表示更新成功，false表示更新失败
//   - error: 如果发生错误，则返回错误信息，否则返回nil
func (r *Redis) SetValueExist(ctx context.Context, prefix prefixenum.PrefixEnum, key string, value interface{}, expire time.Duration) (bool, error) {
	key = getKey(prefix, key)
	sc := `
local key = KEYS[1]
local newValue = ARGV[1]

-- 获取当前值
local currentValue = redis.call("GET", key)

-- 比较当前值与新值
if currentValue == newValue then
    return 0
else
    -- 设置新值
    redis.call("SETEX", key, ARGV[2],newValue)
    return 1
end
`
	res, err := r.Eval(ctx, sc, []string{key}, value, int64(expire/time.Second))
	if err != nil {
		return false, err
	}
	return res.(int64) == 1, err
}

// SetBit 对 key 所储存的字符串值，设置或清除指定偏移量上的位(bit)。
//
// Parameters:
//   - ctx: context.Context 上下文
//   - prefix: PrefixEnum key前缀
//   - key: string key
//   - offset: int64 偏移量
//   - value: int 0/1
//
// Returns:
//   - error
func (r *Redis) SetBit(ctx context.Context, prefix prefixenum.PrefixEnum, key string, offset int64, value int) error {
	k := getKey(prefix, key)
	return r.UniversalClient.SetBit(ctx, k, offset, value).Err()
}

// Append 如果 key 已经存在并且是一个字符串， APPEND 命令将指定的 value 追加到该 key 原来值（value）的末尾。
//
// Parameters:
//   - ctx: context.Context 上下文
//   - prefix: PrefixEnum 前缀
//   - key: string
//   - value: string
//
// Returns:
//   - int64 追加指定值之后， key 中字符串的长度。
//   - error
func (r *Redis) Append(ctx context.Context, prefix prefixenum.PrefixEnum, key string, value string) (int64, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.Append(ctx, k, value).Result()
}

// Incr 累加1
//
// Parameters:
//   - ctx: context.Context
//   - prefix: PrefixEnum
//   - key: string
//
// Returns:
//   - int64 累加后的值
//   - error
func (r *Redis) Incr(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (int64, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.Incr(ctx, k).Result()
}

// Incrby 累加指定值
//
// Parameters:
//   - ctx: context.Context
//   - prefix: PrefixEnum
//   - key: string
//   - v: int64
//
// Returns:
//   - int64 累加后的值
//   - error
func (r *Redis) Incrby(ctx context.Context, prefix prefixenum.PrefixEnum, key string, v int64) (int64, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.IncrBy(ctx, k, v).Result()
}

// IncrWithExpire 累加 1 后设置过期时间
//
// Parameters:
//   - ctx: context.Context
//   - prefix: PrefixEnum
//   - key: string
//   - expire: time.Duration
//
// Returns:
//   - error
func (r *Redis) IncrWithExpire(ctx context.Context, prefix prefixenum.PrefixEnum, key string, expire time.Duration) error {
	k := getKey(prefix, key)
	pipe := r.UniversalClient.Pipeline()

	pipe.Incr(ctx, k).Err()
	if expire > 0 {
		pipe.Expire(ctx, k, expire)
	}
	_, err := pipe.Exec(context.TODO())
	return err
}

// IncrbyLimit 累增v数量，然后如果up是true，则判断累增后的值是否超过limit，如果超过则报错不允许更新，如果up为false，则判断累增后的值是否小于limit，如果小于则报错
//
// Parameters:
//   - ctx: context.Context 上下文
//   - prefix: PrefixEnum key前缀
//   - key: string key
//   - v: int64 累增值
//   - limit: int64 限制
//   - up: bool true=判断上限，false=判断下限
//   - expire: ...time.Duration 过期时间
//
// Returns:
//   - int64 累加后的值
//   - error
func (r *Redis) IncrbyLimit(ctx context.Context, prefix prefixenum.PrefixEnum, key string, v int64, limit int64, up bool, expire ...time.Duration) (int64, error) {
	k := getKey(prefix, key)
	sc := `
local key = KEYS[1]
local inc = tonumber(ARGV[1])
local up = tonumber(ARGV[2])
local limitVal =  tonumber(ARGV[3])
local timeout = tonumber(ARGV[4])
-- 获取当前的值
local val = tonumber(redis.call('GET', key)) or 0

local afterVal= val+inc
-- 下限
if up==0 and afterVal<limitVal then
	return {err="redis: Exceeding the limit after accumulation"}
end

if up==1 and afterVal>limitVal then
	return {err="redis: Exceeding the limit after accumulation"}
end

-- 更新值
local succ, afterRes = pcall(function()
	return redis.call('INCRBY', key, inc)
end)

if not succ then
	return afterRes
end

-- 设置超时时间
if timeout > 0 then
	redis.call('EXPIRE', key, timeout)
end


-- 返回更新后的值
return afterRes
`

	var upInt uint8
	if up {
		upInt = 1
	}
	var out int64
	if len(expire) > 0 {
		out = int64(expire[0] / time.Second)
	} else {
		out = -1
	}
	res, err := r.Eval(ctx, sc, []string{k}, v, upInt, limit, out)
	if err != nil {
		if err.Error() == RedisIncExceed.Error() {
			return 0, RedisIncExceed
		}
		return 0, err
	}
	return res.(int64), nil
}

// Expire 设置过期时间
//
// Parameters:
//   - ctx: context.Context
//   - prefix: PrefixEnum
//   - key: string
//   - expire: time.Duration
//
// Returns:
//   - error
func (r *Redis) Expire(ctx context.Context, prefix prefixenum.PrefixEnum, key string, expire time.Duration) error {
	k := getKey(prefix, key)
	return r.UniversalClient.Expire(ctx, k, expire).Err()
}

// Delete removes a specific key and its value from the Redis server.
//
// Parameters:
//   - ctx: context.Context
//   - prefix: PrefixEnum
//   - key: string
//
// Returns:
//   - error
func (r *Redis) Delete(ctx context.Context, prefix prefixenum.PrefixEnum, key string) error {
	k := getKey(prefix, key)
	return r.UniversalClient.Del(ctx, k).Err()
}

func (r *Redis) Exists(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (int64, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.Exists(ctx, k).Result()
}

// Scan  遍历 key，需要提前设置实例
//
// match 传原始 pattern，返回的也是原始 key —— 租户前缀由本方法进出各处理一次，
// 与 Get/Del 等方法的契约一致，调用方不要自己拼 tenant 前缀。
//
// 之所以要在这里手工处理而不是交给 PrefixHookV3：SCAN 没有 key 参数，需要加前缀的是
// MATCH 后面那个「可选且位置浮动」的 pattern（go-redis 只在 match != "" 时才追加
// "match" 和 pattern 两个参数），而 cmdModes 是一张「第 N 个参数起、每隔 M 个」的
// 位置表，表达不了这种形态，所以 scan 不在表里。
//
// Parameters:
//   - ctx: context.Context
//   - cursor: uint64
//   - match: string 原始 pattern，不带租户前缀
//   - count: int64
//
// Returns:
//   - []string 原始 key，不带租户前缀
//   - uint64
//   - error
func (r *Redis) Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	if r.UniversalClient == nil {
		return nil, 0, errors.New("redis: client is nil")
	}
	// match 为空时 go-redis 会整段省略 MATCH（遍历全库）。多租户下必须收敛到本租户，
	// 否则会扫出别的租户的 key；ns 为空或配了兜底租户时 FmtByType 短路，"*" 与原来的
	// 全库遍历等价，行为不变。
	if match == "" {
		match = "*"
	}
	// 用空 key 反推本次实际生效的前缀：ns 为空或配了兜底租户时 FmtByType 短路，这里得到
	// 空串，表示本次没加前缀、下面也不该剥。
	prefix := namespace.FmtByTypeOnCtx(ctx, namespace.NAMESPACE_TYPE_REDIS, "")
	keys, next, err := r.UniversalClient.Scan(ctx, cursor,
		namespace.FmtByTypeOnCtx(ctx, namespace.NAMESPACE_TYPE_REDIS, match), count).Result()
	if err != nil {
		return nil, 0, err
	}
	// 剥回原始 key：调用方拿这批 key 去 DEL/UNLINK 时会走 hook 再加一次前缀
	// （两者都是 modeAllFrom1），不剥就变成 tenantX:tenantX:…，删了个空且不报错。
	//
	// 只剥「本次自己加的那个前缀」，不要用 GetOriginKey —— 它的判据是 HasPrefix(key,
	// "tenant") 再按 ":" 切，会把业务上本来就叫 tenantxxx:… 的 key（比如 NS 线上一个
	// 名为 tenant_stats: 的裸 key）误剥成半截。
	if prefix != "" {
		for i, k := range keys {
			keys[i] = strings.TrimPrefix(k, prefix)
		}
	}
	return keys, next, nil
}

// HExists 判断 hmap 字段是否存在
//
// Parameters:
//   - ctx: context.Context
//   - prefix: PrefixEnum
//   - key: string
//   - field: string
//
// Returns:
//   - bool
//   - error
func (r *Redis) HExists(ctx context.Context, prefix prefixenum.PrefixEnum, key, field string) (bool, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.HExists(ctx, k, field).Result()
}

func (r *Redis) HGet(ctx context.Context, prefix prefixenum.PrefixEnum, key, field string) (string, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.HGet(ctx, k, field).Result()
}

func (r *Redis) HKeys(ctx context.Context, prefix prefixenum.PrefixEnum, key string) ([]string, error) {
	k := getKey(prefix, key)

	return r.UniversalClient.HKeys(ctx, k).Result()
}

func (r *Redis) HGetInt64(ctx context.Context, prefix prefixenum.PrefixEnum, key, field string) (int64, error) {
	k := getKey(prefix, key)

	return r.UniversalClient.HGet(ctx, k, field).Int64()
}

func (r *Redis) HGetALLMap(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (map[string]string, error) {
	k := getKey(prefix, key)

	return r.UniversalClient.HGetAll(ctx, k).Result()
}

func (r *Redis) HMGet(ctx context.Context, prefix prefixenum.PrefixEnum, key string, fields ...string) ([]interface{}, error) {
	k := getKey(prefix, key)

	return r.UniversalClient.HMGet(ctx, k, fields...).Result()
}

func (r *Redis) HSet(ctx context.Context, prefix prefixenum.PrefixEnum, key, field, value string, expire time.Duration) error {
	k := getKey(prefix, key)

	_, err := r.UniversalClient.Pipelined(ctx, func(pipeliner redis.Pipeliner) (err error) {
		if err = pipeliner.HSet(ctx, k, field, value).Err(); err != nil {
			return err
		}
		if expire > 0 {
			return pipeliner.Expire(ctx, k, expire).Err()
		}
		return nil
	})
	return err
}

func (r *Redis) HSetSlice(ctx context.Context, prefix prefixenum.PrefixEnum, key string, args ...interface{}) error {
	k := getKey(prefix, key)

	if err := r.UniversalClient.HSet(ctx, k, args...).Err(); err != nil {
		return err
	}

	return nil
}

func (r *Redis) HLen(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (int64, error) {
	k := getKey(prefix, key)

	return r.UniversalClient.HLen(ctx, k).Result()
}

func (r *Redis) HSetNx(ctx context.Context, prefix prefixenum.PrefixEnum, key, field, value string) (bool, error) {
	k := getKey(prefix, key)

	return r.UniversalClient.HSetNX(ctx, k, field, value).Result()

}

func (r *Redis) HMSetMap(ctx context.Context, prefix prefixenum.PrefixEnum, key string, data map[string]interface{}, expire time.Duration) error {
	k := getKey(prefix, key)

	_, err := r.UniversalClient.Pipelined(ctx, func(pipeliner redis.Pipeliner) (err error) {
		if err = pipeliner.HMSet(ctx, k, data).Err(); err != nil {
			return err
		}
		if expire > 0 {
			return pipeliner.Expire(ctx, k, expire).Err()
		}
		return nil
	})
	return err
}

func (r *Redis) HMSet(ctx context.Context, prefix prefixenum.PrefixEnum, key string, expire time.Duration, data ...interface{}) error {
	key = getKey(prefix, key)
	_, err := r.UniversalClient.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {
		if err := pipeliner.HSet(ctx, key, data...).Err(); err != nil {
			return err
		}
		if expire > 0 {
			return pipeliner.Expire(ctx, key, expire).Err()
		}
		return nil
	})
	return err
}

func (r *Redis) HScan(ctx context.Context, prefix prefixenum.PrefixEnum, key string, cursor uint64, match string, count int64) ([]string, uint64, error) {
	key = getKey(prefix, key)
	return r.UniversalClient.HScan(ctx, key, cursor, match, count).Result()
}

func (r *Redis) HIncrBy(ctx context.Context, prefix prefixenum.PrefixEnum, key, field string, increase int64) (int64, error) {
	k := getKey(prefix, key)

	return r.UniversalClient.HIncrBy(ctx, k, field, increase).Result()
}

func (r *Redis) HDel(ctx context.Context, prefix prefixenum.PrefixEnum, key string, field ...string) error {
	k := getKey(prefix, key)

	return r.UniversalClient.HDel(ctx, k, field...).Err()
}

func (r *Redis) ZAdd(ctx context.Context, prefix prefixenum.PrefixEnum, key string, score float64, data interface{}) error {
	k := getKey(prefix, key)

	return r.UniversalClient.ZAdd(ctx, k, redis.Z{Score: score, Member: data}).Err()
}

func (r *Redis) ZMAdd(ctx context.Context, prefix prefixenum.PrefixEnum, key string, members ...redis.Z) error {
	k := getKey(prefix, key)

	return r.UniversalClient.ZAdd(ctx, k, members...).Err()
}

func (r *Redis) ZRank(ctx context.Context, prefix prefixenum.PrefixEnum, key string, member string) (int64, error) {
	k := getKey(prefix, key)

	return r.UniversalClient.ZRank(ctx, k, member).Result()
}

func (r *Redis) ZRankWithScore(ctx context.Context, prefix prefixenum.PrefixEnum, key string, member string) (redis.RankScore, error) {
	k := getKey(prefix, key)

	return r.UniversalClient.ZRankWithScore(ctx, k, member).Result()
}

func (r *Redis) ZRangeByScore(ctx context.Context, prefix prefixenum.PrefixEnum, key string, offset, limit int64) ([]string, error) {
	k := getKey(prefix, key)

	return r.UniversalClient.ZRangeByScore(ctx, k, &redis.ZRangeBy{Min: "-inf", Max: "+inf", Offset: offset, Count: limit}).Result()
}

func (r *Redis) ZRangeByScoreH(ctx context.Context, prefix prefixenum.PrefixEnum, key string, min, max string, offset, limit int64) ([]string, error) {
	k := getKey(prefix, key)

	opt := &redis.ZRangeBy{Min: min, Max: max, Offset: offset, Count: limit}
	return r.UniversalClient.ZRangeByScore(ctx, k, opt).Result()
}

func (r *Redis) ZRangeByScoreWithScores(ctx context.Context, prefix prefixenum.PrefixEnum, key string, min, max string, offset, limit int64) ([]redis.Z, error) {
	k := getKey(prefix, key)

	opt := &redis.ZRangeBy{Min: min, Max: max, Offset: offset, Count: limit}
	return r.UniversalClient.ZRangeByScoreWithScores(ctx, k, opt).Result()
}

func (r *Redis) ZRange(ctx context.Context, prefix prefixenum.PrefixEnum, key string, start, stop int64) ([]string, error) {
	k := getKey(prefix, key)

	return r.UniversalClient.ZRange(ctx, k, start, stop).Result()
}

func (r *Redis) ZRangeWithScores(ctx context.Context, prefix prefixenum.PrefixEnum, key string, start, stop int64) ([]redis.Z, error) {
	k := getKey(prefix, key)

	return r.UniversalClient.ZRangeWithScores(ctx, k, start, stop).Result()
}

func (r *Redis) ZRangeByLex(ctx context.Context, prefix prefixenum.PrefixEnum, key string, max, min string, offset, count int64) ([]string, error) {
	k := getKey(prefix, key)

	z := redis.ZRangeBy{
		Max:    max,
		Min:    min,
		Offset: offset,
		Count:  count,
	}
	return r.UniversalClient.ZRangeByLex(ctx, k, &z).Result()
}

func (r *Redis) ZLexCount(ctx context.Context, prefix prefixenum.PrefixEnum, key string, max, min string) (int64, error) {
	k := getKey(prefix, key)

	return r.UniversalClient.ZLexCount(ctx, k, max, min).Result()
}

func (r *Redis) ZCount(ctx context.Context, prefix prefixenum.PrefixEnum, key, min, max string) (int64, error) {
	k := getKey(prefix, key)

	return r.UniversalClient.ZCount(ctx, k, min, max).Result()
}

func (r *Redis) ZIncrBy(ctx context.Context, prefix prefixenum.PrefixEnum, key, member string, inc float64) (float64, error) {
	k := getKey(prefix, key)

	return r.UniversalClient.ZIncrBy(ctx, k, inc, member).Result()
}

func (r *Redis) ZCountAll(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (int64, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.ZCount(ctx, k, "-inf", "+inf").Result()
}

func (r *Redis) ZRem(ctx context.Context, prefix prefixenum.PrefixEnum, key string, members ...interface{}) error {
	k := getKey(prefix, key)
	return r.UniversalClient.ZRem(ctx, k, members...).Err()
}

func (r *Redis) ZRemRangeByScore(ctx context.Context, prefix prefixenum.PrefixEnum, key, min, max string) (int64, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.ZRemRangeByScore(ctx, k, min, max).Result()
}

func (r *Redis) ZRemRangeByRank(ctx context.Context, prefix prefixenum.PrefixEnum, key string, start, end int64) (int64, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.ZRemRangeByRank(ctx, k, start, end).Result()
}

func (r *Redis) ZRevRank(ctx context.Context, prefix prefixenum.PrefixEnum, key string, member string) (int64, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.ZRevRank(ctx, k, member).Result()
}

func (r *Redis) ZRevRange(ctx context.Context, prefix prefixenum.PrefixEnum, key string, start, stop int64) ([]string, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.ZRevRange(ctx, k, start, stop).Result()
}
func (r *Redis) ZRevRangeWithScores(ctx context.Context, prefix prefixenum.PrefixEnum, key string, start, stop int64) ([]redis.Z, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.ZRevRangeWithScores(ctx, k, start, stop).Result()
}

func (r *Redis) ZRevRangeByScoreWithScores(ctx context.Context, prefix prefixenum.PrefixEnum, key string, max, min string, start, stop int64) ([]redis.Z, error) {
	k := getKey(prefix, key)
	z := redis.ZRangeBy{
		Max:    max,
		Min:    min,
		Offset: start,
		Count:  stop,
	}
	return r.UniversalClient.ZRevRangeByScoreWithScores(ctx, k, &z).Result()
}

func (r *Redis) ZScan(ctx context.Context, prefix prefixenum.PrefixEnum, key string, cursor uint64, match string, count int64) ([]string, uint64, error) {
	key = getKey(prefix, key)
	return r.UniversalClient.ZScan(ctx, key, cursor, match, count).Result()
}

func (r *Redis) ZScore(ctx context.Context, prefix prefixenum.PrefixEnum, key string, member string) (float64, error) {
	key = getKey(prefix, key)
	return r.UniversalClient.ZScore(ctx, key, member).Result()
}

func (r *Redis) SAdd(ctx context.Context, prefix prefixenum.PrefixEnum, key string, members ...interface{}) error {
	k := getKey(prefix, key)
	return r.UniversalClient.SAdd(ctx, k, members...).Err()
}

func (r *Redis) SAddWithExpire(ctx context.Context, prefix prefixenum.PrefixEnum, key string, expire time.Duration, members ...interface{}) error {
	k := getKey(prefix, key)
	err := r.UniversalClient.SAdd(ctx, k, members...).Err()
	if err != nil && expire > 0 {
		err = r.UniversalClient.Expire(ctx, k, expire).Err()
	}
	return err
}

func (r *Redis) SPopN(ctx context.Context, prefix prefixenum.PrefixEnum, key string, count int64) ([]string, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.SPopN(ctx, k, count).Result()
}

func (r *Redis) SDiff(ctx context.Context, prefix prefixenum.PrefixEnum, keys ...string) ([]string, error) {
	newKeys := make([]string, 0)
	for _, e := range keys {
		newKeys = append(newKeys, getKey(prefix, e))
	}

	return r.UniversalClient.SDiff(ctx, newKeys...).Result()
}

func (r *Redis) SDiffStore(ctx context.Context, prefix prefixenum.PrefixEnum, destination string, keys ...string) error {
	destination = getKey(prefix, destination)
	newKeys := make([]string, 0)
	for _, e := range keys {
		newKeys = append(newKeys, getKey(prefix, e))
	}
	return r.UniversalClient.SDiffStore(ctx, destination, newKeys...).Err()
}

func (r *Redis) Smembers(ctx context.Context, prefix prefixenum.PrefixEnum, key string) ([]string, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.SMembers(ctx, k).Result()
}

func (r *Redis) SUnionStore(ctx context.Context, prefix prefixenum.PrefixEnum, dstKey string, srcKey ...string) (int64, error) {
	k := getKey(prefix, dstKey)
	for i, v := range srcKey {
		srcKey[i] = getKey(prefix, v)
	}
	fmt.Println("sunio: ", srcKey)
	return r.UniversalClient.SUnionStore(ctx, k, srcKey...).Result()
}

func (r *Redis) SCard(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (int64, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.SCard(ctx, k).Result()
}

func (r *Redis) SScan(ctx context.Context, prefix prefixenum.PrefixEnum, key string, cursor uint64, match string, count int64) ([]string, uint64, error) {
	key = getKey(prefix, key)
	return r.UniversalClient.SScan(ctx, key, cursor, match, count).Result()
}

func (r *Redis) SRandMember(ctx context.Context, prefix prefixenum.PrefixEnum, key string, num int64) ([]string, error) {
	key = getKey(prefix, key)
	return r.UniversalClient.SRandMemberN(ctx, key, num).Result()
}

func (r *Redis) SIsmember(ctx context.Context, prefix prefixenum.PrefixEnum, key string, member interface{}) (bool, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.SIsMember(ctx, k, member).Result()
}

func (r *Redis) SRem(ctx context.Context, prefix prefixenum.PrefixEnum, key string, elementKey ...string) (int64, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.SRem(ctx, k, elementKey).Result()
}

func (r *Redis) RPush(ctx context.Context, prefix prefixenum.PrefixEnum, key string, values ...interface{}) error {
	k := getKey(prefix, key)
	return r.UniversalClient.RPush(ctx, k, values...).Err()
}

func (r *Redis) LPush(ctx context.Context, prefix prefixenum.PrefixEnum, key string, values ...interface{}) error {
	k := getKey(prefix, key)
	return r.UniversalClient.LPush(ctx, k, values...).Err()
}

func (r *Redis) LRem(ctx context.Context, prefix prefixenum.PrefixEnum, key string, count int64, value interface{}) (int64, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.LRem(ctx, k, count, value).Result()
}

func (r *Redis) LRange(ctx context.Context, prefix prefixenum.PrefixEnum, key string, start, end int64) ([]string, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.LRange(ctx, k, start, end).Result()
}

func (r *Redis) LIndex(ctx context.Context, prefix prefixenum.PrefixEnum, key string, index int64) (string, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.LIndex(ctx, k, index).Result()
}

func (r *Redis) BLPop(ctx context.Context, prefix prefixenum.PrefixEnum, timeOut time.Duration, key ...string) ([]string, error) {
	for i, v := range key {
		key[i] = getKey(prefix, v)
	}
	return r.UniversalClient.BLPop(ctx, timeOut, key...).Result()
}

func (r *Redis) LPop(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (string, error) {
	key = getKey(prefix, key)
	return r.UniversalClient.LPop(ctx, key).Result()
}

func (r *Redis) LPopCount(ctx context.Context, prefix prefixenum.PrefixEnum, key string, count int) ([]string, error) {
	key = getKey(prefix, key)
	return r.UniversalClient.LPopCount(ctx, key, count).Result()
}

func (r *Redis) RPopCount(ctx context.Context, prefix prefixenum.PrefixEnum, key string, count int) ([]string, error) {
	key = getKey(prefix, key)
	return r.UniversalClient.RPopCount(ctx, key, count).Result()
}

func (r *Redis) RPop(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (string, error) {
	key = getKey(prefix, key)
	return r.UniversalClient.RPop(ctx, key).Result()
}

func (r *Redis) BRPop(ctx context.Context, prefix prefixenum.PrefixEnum, timeout time.Duration, keys ...string) ([]string, error) {
	for i, v := range keys {
		keys[i] = getKey(prefix, v)
	}
	return r.UniversalClient.BRPop(ctx, timeout, keys...).Result()
}

func (r *Redis) LLen(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (int64, error) {
	key = getKey(prefix, key)
	return r.UniversalClient.LLen(ctx, key).Result()
}

func (r *Redis) Ltrim(ctx context.Context, prefix prefixenum.PrefixEnum, key string, start, stop int64) error {
	key = getKey(prefix, key)
	return r.UniversalClient.LTrim(ctx, key, start, stop).Err()
}

func (r *Redis) RPopLPush(ctx context.Context, prefix prefixenum.PrefixEnum, src, dst string) (int64, error) {
	src = getKey(prefix, src)
	dst = getKey(prefix, dst)
	return r.UniversalClient.RPopLPush(ctx, src, dst).Int64()
}

func (r *Redis) Subscribe(ctx context.Context, prefix prefixenum.PrefixEnum, key string) *redis.PubSub {
	k := getKey(prefix, key)
	return r.UniversalClient.Subscribe(ctx, k)
}
func (r *Redis) SubscribeC(ctx context.Context, key string) *redis.PubSub {
	return r.UniversalClient.Subscribe(ctx, key)
}

func (r *Redis) PSubscribe(ctx context.Context, prefix prefixenum.PrefixEnum, key string) *redis.PubSub {
	k := getKey(prefix, key)
	return r.UniversalClient.PSubscribe(ctx, k)
}

func (r *Redis) PSubscribeC(ctx context.Context, key string) *redis.PubSub {
	return r.UniversalClient.PSubscribe(ctx, key)
}

func (r *Redis) SSubscribe(ctx context.Context, prefix prefixenum.PrefixEnum, key string) *redis.PubSub {
	k := getKey(prefix, key)
	return r.UniversalClient.SSubscribe(ctx, k)
}

func (r *Redis) SSubscribeC(ctx context.Context, key string) *redis.PubSub {
	return r.UniversalClient.SSubscribe(ctx, key)
}

func (r *Redis) Publish(ctx context.Context, prefix prefixenum.PrefixEnum, key string, value interface{}) (int64, error) {
	k := getKey(prefix, key)

	return r.UniversalClient.Publish(ctx, k, value).Result()
}

func (r *Redis) TTL(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (time.Duration, error) {
	k := getKey(prefix, key)
	return r.UniversalClient.TTL(ctx, k).Result()
}

func (r *Redis) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	return r.UniversalClient.Eval(ctx, script, keys, args...).Result()
}

// EvalSha
//
//	@Description: 通过脚本的sha值执行脚本
//	@Author	yw 2024-11-04 14:16:10
//	@param	sha1	缓存脚本的sha值
//	@param	keys	脚本里需要使用的key
//	@param	args	脚本需要使用的参数
//	@return	interface{}
//	@return	error
func (r *Redis) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) (interface{}, error) {
	return r.UniversalClient.EvalSha(ctx, sha1, keys, args...).Result()
}

// ScriptExists
//
//	@Description: 判断脚本是否存在
//	@Author	yw 2024-11-04 14:17:44
//	@param	sha1	缓存脚本的sha值
//	@return	[]bool
//	@return	error
func (r *Redis) ScriptExists(ctx context.Context, sha1 ...string) ([]bool, error) {
	return r.UniversalClient.ScriptExists(ctx, sha1...).Result()
}

// EvalShaWithScript
//
//	@Description: 通过脚本的sha1值执行脚本，如果脚本不存在，根据可变参数script判断是否需要设置脚本到缓存
//	@Author	yw 2024-11-04 11:07:09
//	@param	sha1 脚本的sha1值
//	@param	keys 脚本中使用到的key
//	@param	args 脚本中使用到的参数
//	@param	script 如果脚本不存在，需要加载的脚本值
//	@return	interface{} 脚本执行后返回数据
//	@return string 脚本加载到内存后返回的sha1值
//	@return	error
func (r *Redis) EvalShaWithScript(ctx context.Context, sha1 string, keys []string, args []interface{}, script ...string) (interface{}, string, error) {
reget:
	if len(sha1) != 0 {
		// 脚本 sha 值不为空，直接去执行
		val, err := r.UniversalClient.EvalSha(ctx, sha1, keys, args...).Result()
		if err == nil { // 执行成功
			return val, sha1, nil
		}
		if errors.Is(err, RedisNoScript) {
			return nil, "", err
		}
	}
	if len(script) == 0 {
		return nil, "", errors.New("script is empty")
	}
	var err error
	// 加载脚本
	sha1, err = r.UniversalClient.ScriptLoad(ctx, script[0]).Result()
	if err != nil {
		return nil, "", err
	}
	goto reget
}

// ScriptLoad
//
//	@Description: 加载脚本到redis内存中
//	@Author	yw 2024-11-04 10:57:53
//	@param	script 脚本代码
//	@return	string 返回脚本sha1值
//	@return	error
func (r *Redis) ScriptLoad(script string) (string, error) {
	if r.UniversalClient == nil {
		return "", errors.New("UniversalClient is nil")
	}
	return r.UniversalClient.ScriptLoad(context.Background(), script).Result()
}

func (r *Redis) Pipelined(ctx context.Context, fn func(pipeliner redis.Pipeliner) error) ([]redis.Cmder, error) {
	if r.UniversalClient == nil {
		return nil, errors.New("UniversalClient is nil")
	}
	return r.UniversalClient.Pipelined(ctx, fn)
}

func getKey(prefix prefixenum.PrefixEnum, key string) string {
	if prefix == "" {
		return key
	}
	return fmt.Sprintf("%s:%s", prefix, key)
}

const (
	RedisMinVal = "-inf"
	RedisMaxVal = "+inf"
)

const RedisNil = redis.Nil
const RedisNoScript = RedisError("NOSCRIPT No matching script. Please use EVAL.")

// inc是超出限制错误
const RedisIncExceed = RedisError("redis: Exceeding the limit after accumulation")

type RedisError string

func (r RedisError) Error() string { return string(r) }
