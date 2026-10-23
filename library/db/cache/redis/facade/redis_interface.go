package facade

import (
	"context"
	"time"

	redis2 "github.com/redis/go-redis/v9"
	"github.com/coderyw/gomicro-util/library/db/cache/redis/model"
	"github.com/coderyw/gomicro-util/library/db/cache/redis/prefixenum"
)

type RedisInterface interface {
	Client() redis2.UniversalClient
	Lock(ctx context.Context, enum prefixenum.PrefixEnum, k string, v interface{}, timeOut time.Duration) (bool, error)
	UnLock(ctx context.Context, enum prefixenum.PrefixEnum, k string, v interface{}) (bool, error)
	LockWithRenewal(ctx context.Context, enum prefixenum.PrefixEnum, k string, timeOut, renewalInterval time.Duration, do func() error) (bool, error)
	LockExpectKey(ctx context.Context, enum prefixenum.PrefixEnum, k string, v interface{}, timeOut time.Duration) (bool, error)
	LockWithTimeOut(ctx context.Context, enum prefixenum.PrefixEnum, k string, timeOut, retryDur, expire time.Duration) (key string, err error)
	LockWaitTimeOutWithRenewalV2(ctx context.Context, param model.LockParam, do func() error) (bool, error)
	LockWaitTimeOutWithRenewal(ctx context.Context, enum prefixenum.PrefixEnum, k string, waitTimeOut, retryInterval, timeOut, renewalInterval time.Duration, do func() error) (bool, error)
	LockUntilSuccWithRenewal(ctx context.Context, enum prefixenum.PrefixEnum, k string, retryInterval, timeOut, renewalInterval time.Duration, do func() error) error
	RenewalLock(ctx context.Context, enum prefixenum.PrefixEnum, k string, lockValue interface{}, renewalInterval, timeout time.Duration, maxRenewalTime ...time.Duration) (stopRenewal func())
	GetValue(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (string, error)
	GetValueByInt64(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (int64, error)
	GetValueBytes(ctx context.Context, prefix prefixenum.PrefixEnum, key string) ([]byte, error)
	MGetValue(ctx context.Context, prefix prefixenum.PrefixEnum, keys []string) ([]interface{}, error)
	MSetValue(ctx context.Context, prefix prefixenum.PrefixEnum, keys []string, value interface{}) (string, error)
	Setnx(ctx context.Context, prefix prefixenum.PrefixEnum, key string, value interface{}, expire time.Duration) (bool, error)
	SetValue(ctx context.Context, prefix prefixenum.PrefixEnum, key string, value interface{}, expire time.Duration) error
	SetValueExist(ctx context.Context, prefix prefixenum.PrefixEnum, key string, value interface{}, expire time.Duration) (bool, error)
	SetBit(ctx context.Context, prefix prefixenum.PrefixEnum, key string, offset int64, value int) error
	Append(ctx context.Context, prefix prefixenum.PrefixEnum, key string, value string) (int64, error)
	Incr(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (int64, error)
	Incrby(ctx context.Context, prefix prefixenum.PrefixEnum, key string, v int64) (int64, error)
	IncrWithExpire(ctx context.Context, prefix prefixenum.PrefixEnum, key string, expire time.Duration) error
	IncrbyLimit(ctx context.Context, prefix prefixenum.PrefixEnum, key string, v int64, limit int64, up bool, expire ...time.Duration) (int64, error)
	Expire(ctx context.Context, prefix prefixenum.PrefixEnum, key string, expire time.Duration) error
	Delete(ctx context.Context, prefix prefixenum.PrefixEnum, key string) error
	Exists(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (int64, error)
	Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error)
	HExists(ctx context.Context, prefix prefixenum.PrefixEnum, key, field string) (bool, error)
	HGet(ctx context.Context, prefix prefixenum.PrefixEnum, key, field string) (string, error)
	HKeys(ctx context.Context, prefix prefixenum.PrefixEnum, key string) ([]string, error)
	HGetInt64(ctx context.Context, prefix prefixenum.PrefixEnum, key, field string) (int64, error)
	HGetALLMap(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (map[string]string, error)
	HMGet(ctx context.Context, prefix prefixenum.PrefixEnum, key string, fields ...string) ([]interface{}, error)
	HSet(ctx context.Context, prefix prefixenum.PrefixEnum, key, field, value string, expire time.Duration) error
	HSetSlice(ctx context.Context, prefix prefixenum.PrefixEnum, key string, args ...interface{}) error
	HLen(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (int64, error)
	HSetNx(ctx context.Context, prefix prefixenum.PrefixEnum, key, field, value string) (bool, error)
	HMSetMap(ctx context.Context, prefix prefixenum.PrefixEnum, key string, data map[string]interface{}, expire time.Duration) error
	HMSet(ctx context.Context, prefix prefixenum.PrefixEnum, key string, expire time.Duration, data ...interface{}) error
	HScan(ctx context.Context, prefix prefixenum.PrefixEnum, key string, cursor uint64, match string, count int64) ([]string, uint64, error)
	HIncrBy(ctx context.Context, prefix prefixenum.PrefixEnum, key, field string, increase int64) (int64, error)
	HDel(ctx context.Context, prefix prefixenum.PrefixEnum, key string, field ...string) error
	ZAdd(ctx context.Context, prefix prefixenum.PrefixEnum, key string, score float64, data interface{}) error
	ZMAdd(ctx context.Context, prefix prefixenum.PrefixEnum, key string, members ...redis2.Z) error
	ZRank(ctx context.Context, prefix prefixenum.PrefixEnum, key string, member string) (int64, error)
	ZRankWithScore(ctx context.Context, prefix prefixenum.PrefixEnum, key string, member string) (redis2.RankScore, error)
	ZRangeByScore(ctx context.Context, prefix prefixenum.PrefixEnum, key string, offset, limit int64) ([]string, error)
	ZRangeByScoreH(ctx context.Context, prefix prefixenum.PrefixEnum, key string, min, max string, offset, limit int64) ([]string, error)
	ZRangeByScoreWithScores(ctx context.Context, prefix prefixenum.PrefixEnum, key string, min, max string, offset, limit int64) ([]redis2.Z, error)
	ZRange(ctx context.Context, prefix prefixenum.PrefixEnum, key string, start, stop int64) ([]string, error)
	ZRangeWithScores(ctx context.Context, prefix prefixenum.PrefixEnum, key string, start, stop int64) ([]redis2.Z, error)
	ZRangeByLex(ctx context.Context, prefix prefixenum.PrefixEnum, key string, max, min string, offset, count int64) ([]string, error)
	ZLexCount(ctx context.Context, prefix prefixenum.PrefixEnum, key string, max, min string) (int64, error)
	ZCount(ctx context.Context, prefix prefixenum.PrefixEnum, key, min, max string) (int64, error)
	ZIncrBy(ctx context.Context, prefix prefixenum.PrefixEnum, key, member string, inc float64) (float64, error)
	ZCountAll(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (int64, error)
	ZRem(ctx context.Context, prefix prefixenum.PrefixEnum, key string, members ...interface{}) error
	ZRemRangeByScore(ctx context.Context, prefix prefixenum.PrefixEnum, key, min, max string) (int64, error)
	ZRemRangeByRank(ctx context.Context, prefix prefixenum.PrefixEnum, key string, start, end int64) (int64, error)
	ZRevRank(ctx context.Context, prefix prefixenum.PrefixEnum, key string, member string) (int64, error)
	ZRevRange(ctx context.Context, prefix prefixenum.PrefixEnum, key string, start, stop int64) ([]string, error)
	ZRevRangeWithScores(ctx context.Context, prefix prefixenum.PrefixEnum, key string, start, stop int64) ([]redis2.Z, error)
	ZRevRangeByScoreWithScores(ctx context.Context, prefix prefixenum.PrefixEnum, key string, max, min string, start, stop int64) ([]redis2.Z, error)
	ZScan(ctx context.Context, prefix prefixenum.PrefixEnum, key string, cursor uint64, match string, count int64) ([]string, uint64, error)
	ZScore(ctx context.Context, prefix prefixenum.PrefixEnum, key string, member string) (float64, error)
	SAdd(ctx context.Context, prefix prefixenum.PrefixEnum, key string, members ...interface{}) error
	SAddWithExpire(ctx context.Context, prefix prefixenum.PrefixEnum, key string, expire time.Duration, members ...interface{}) error
	SPopN(ctx context.Context, prefix prefixenum.PrefixEnum, key string, count int64) ([]string, error)
	SDiff(ctx context.Context, prefix prefixenum.PrefixEnum, keys ...string) ([]string, error)
	SDiffStore(ctx context.Context, prefix prefixenum.PrefixEnum, destination string, keys ...string) error
	Smembers(ctx context.Context, prefix prefixenum.PrefixEnum, key string) ([]string, error)
	SUnionStore(ctx context.Context, prefix prefixenum.PrefixEnum, dstKey string, srcKey ...string) (int64, error)
	SCard(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (int64, error)
	SScan(ctx context.Context, prefix prefixenum.PrefixEnum, key string, cursor uint64, match string, count int64) ([]string, uint64, error)
	SRandMember(ctx context.Context, prefix prefixenum.PrefixEnum, key string, num int64) ([]string, error)
	SIsmember(ctx context.Context, prefix prefixenum.PrefixEnum, key string, member interface{}) (bool, error)
	SRem(ctx context.Context, prefix prefixenum.PrefixEnum, key string, elementKey ...string) (int64, error)
	RPush(ctx context.Context, prefix prefixenum.PrefixEnum, key string, values ...interface{}) error
	LPush(ctx context.Context, prefix prefixenum.PrefixEnum, key string, values ...interface{}) error
	LRem(ctx context.Context, prefix prefixenum.PrefixEnum, key string, count int64, value interface{}) (int64, error)
	LRange(ctx context.Context, prefix prefixenum.PrefixEnum, key string, start, end int64) ([]string, error)
	LIndex(ctx context.Context, prefix prefixenum.PrefixEnum, key string, index int64) (string, error)
	BLPop(ctx context.Context, prefix prefixenum.PrefixEnum, timeOut time.Duration, key ...string) ([]string, error)
	LPop(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (string, error)
	LPopCount(ctx context.Context, prefix prefixenum.PrefixEnum, key string, count int) ([]string, error)
	RPop(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (string, error)
	RPopCount(ctx context.Context, prefix prefixenum.PrefixEnum, key string, count int) ([]string, error)
	BRPop(ctx context.Context, prefix prefixenum.PrefixEnum, timeout time.Duration, keys ...string) ([]string, error)
	LLen(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (int64, error)
	Ltrim(ctx context.Context, prefix prefixenum.PrefixEnum, key string, start, stop int64) error
	RPopLPush(ctx context.Context, prefix prefixenum.PrefixEnum, src, dst string) (int64, error)
	Subscribe(ctx context.Context, prefix prefixenum.PrefixEnum, key string) *redis2.PubSub
	SubscribeC(ctx context.Context, key string) *redis2.PubSub
	PSubscribe(ctx context.Context, prefix prefixenum.PrefixEnum, key string) *redis2.PubSub
	PSubscribeC(ctx context.Context, key string) *redis2.PubSub
	SSubscribe(ctx context.Context, prefix prefixenum.PrefixEnum, key string) *redis2.PubSub
	SSubscribeC(ctx context.Context, key string) *redis2.PubSub
	Publish(ctx context.Context, prefix prefixenum.PrefixEnum, key string, value interface{}) (int64, error)
	TTL(ctx context.Context, prefix prefixenum.PrefixEnum, key string) (time.Duration, error)
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error)
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) (interface{}, error)
	ScriptExists(ctx context.Context, sha1 ...string) ([]bool, error)
	EvalShaWithScript(ctx context.Context, sha1 string, keys []string, args []interface{}, script ...string) (interface{}, string, error)
	ScriptLoad(script string) (string, error)
	Pipelined(ctx context.Context, fn func(pipeliner redis2.Pipeliner) error) ([]redis2.Cmder, error)
	// PutWaitSyncKey 添加待同步数据
	//
	//	使用前需要先通过 InitSyncJob 进行初始化，然后会根据分区数量，以及 field 的 hash 值进行取模计算最终待同步的 key(zset):
	//	syncKey:partition
	//
	//	zset 中 field 存储的是函数传递进来的 filed，score 传入的是时间戳
	//
	// Parameters:
	//   - syncKey string 同步 key
	//   - field string 同步字段
	//   - timestamp int64 秒时间戳
	//
	// Returns：
	//   - error
	PutWaitSyncKey(ctx context.Context, syncKey string, field string, timestamp int64) error
	// InitSyncJob 初始化同步任务
	//
	// Parameters:
	//   - opt RedisSyncOption 同步参数
	InitSyncJob(ctx context.Context, opt model.RedisSyncOption) error
	// PutWaitSyncKeys 批量添加待同步数据
	//
	//	语义与逐条调用 PutWaitSyncKey 完全一致（每个 field 独立按 hash 取模分区、独立执行写入脚本），
	//	仅将 N 次网络往返合并为一次 pipeline，供单次需投递大量 field 的调用方（如批量消费）使用
	//
	// Parameters:
	//   - syncKey string 同步 key
	//   - fields []string 同步字段列表
	//   - timestamp int64 秒时间戳
	//
	// Returns：
	//   - error
	PutWaitSyncKeys(ctx context.Context, syncKey string, fields []string, timestamp int64) error
}
