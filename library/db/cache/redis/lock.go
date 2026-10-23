/**
 * @Author: yinwei
 * @Description:redis实现分布式锁
 * @File: lock
 * @Version: 1.0.0
 * @Date: 2021/7/19 4:46 下午
 */
package redis

import (
	"context"
	"fmt"
	"time"
	"unsafe"

	"github.com/google/uuid"
	"github.com/coderyw/gomicro-util/library/db/cache/redis/model"
	"github.com/coderyw/gomicro-util/library/db/cache/redis/prefixenum"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/skywalking"
	"go.uber.org/zap"
)

// Lock 尝试去获取分布式锁，获取失败即返回false
//
// Parameters:
//   - enum: PrefixEnum 锁的前缀
//   - k: string 锁的key
//   - v: interface{} 要锁定的值
//   - timeOut: time.Duration 锁超时时间
//
// Returns:
//   - bool 获取锁是否成功
//   - error 获取锁的错误信息
func (r *Redis) Lock(ctx context.Context, enum prefixenum.PrefixEnum, k string, v interface{}, timeOut time.Duration) (bool, error) {
	k = getKey(enum, k)
	return r.UniversalClient.SetNX(ctx, k, v, timeOut).Result()
}

// UnLock 对分布式锁解锁，解锁会判断k对应的value是否与传入的v一致，如果一致会进行删除解锁。否则不会做任何处理
//
// Parameters:
//   - enum: PrefixEnum 锁的前缀
//   - k: string 要解锁的key
//   - v: interface{} 要解锁的value
//
// Returns:
//   - bool 返回是否解锁成功
//   - error 返回解锁的错误信息
func (r *Redis) UnLock(ctx context.Context, enum prefixenum.PrefixEnum, k string, v interface{}) (bool, error) {
	skywalking.StartRedis()
	defer skywalking.StopSpan()
	log.DebugCtx(ctx, "解锁", zap.String("key", k), zap.Any("value", v))
	k = getKey(enum, k)
	sc := "if redis.call('get', KEYS[1]) == ARGV[1] then return redis.call('del', KEYS[1]) else return 0 end"
	val := fmt.Sprintf("%v", v)
	res, err := r.Eval(ctx, sc, []string{k}, []string{val})
	if err != nil {
		log.ErrorCtx(ctx, "解锁失败", zap.String("key", k), zap.Any("value", v), zap.Error(err))
		return false, err
	}
	if val, ok := res.(int64); ok {
		return val != 0, nil
	}
	return false, fmt.Errorf("lua 脚本执行结果解析失败:%v", res)
}

// LockWithRenewal 获取续约锁
//
// 尝试获取分布式锁，如果失败直接返回；如果成功会对锁续约，并执行业务逻辑。锁的释放在do函数之后进行
//
// Parameters:
//   - enum: 类型为 PrefixEnum 的key前缀值,可为空
//   - k: 类型为 string 锁的key
//   - timeOut: 类型为 time.Duration 锁的超时时间，默认为5s
//   - renewalInterval: 类型为 time.Duration 续约时间间隔，默认为3s
//   - do: 类型为 func() error 需要执行的业务逻辑
//
// Returns:
//   - bool: true=获取锁成功；false=获取锁失败
//   - error: 获取锁期间的错误信息
func (r *Redis) LockWithRenewal(ctx context.Context, enum prefixenum.PrefixEnum, k string, timeOut, renewalInterval time.Duration, do func() error) (bool, error) {
	skywalking.StartRedis()
	defer skywalking.StopSpan()
	if timeOut == 0 {
		timeOut = 5 * time.Second
	}
	if renewalInterval == 0 {
		renewalInterval = 3 * time.Second
	}

	k = getKey(enum, k)
	// 进行加锁
	v := time.Now().UnixNano()
	bo, err := r.UniversalClient.SetNX(ctx, k, v, timeOut).Result()
	if err != nil {
		return false, err
	}
	if !bo {
		return false, nil
	}
	return true, r.renewal(ctx, k, v, renewalInterval, timeOut, do)
}

// LockExpectKey 获取锁，如果锁被其他客户端持有，则返回false，如果锁被自己持有，则返回true，并且延长锁的过期时间
//
//	加锁的key为k，如果k不存在，则直接加锁；如果k存在，且v值与redis内已经保存的相同，则可以获取到锁，同时延长过期时间。否则返回false
//
// Parameters:
//   - enum: 类型为 PrefixEnum 的key前缀值,可为空
//   - k: 类型为 string 锁的key
//   - v: 类型为 interface{} 锁的值，最好传递字符串、数字类型，入股传递结构体，接口等，序列化过程可能有非预期结果
//   - timeOut: 类型为 time.Duration 锁的超时时间，默认为5s
//
// Returns:
//   - bool: true=获取锁成功；false=获取锁失败
//   - error: 获取锁期间的错误信息
func (r *Redis) LockExpectKey(ctx context.Context, enum prefixenum.PrefixEnum, k string, v interface{}, timeOut time.Duration) (bool, error) {
	k = getKey(enum, k)
	//return redis.call('EXPIRE', KEYS[1], ARGV[2])
	sc := `
local val = redis.call('get', KEYS[1])
--[ 检查条件 --]
if not val then
	redis.call('set', KEYS[1], ARGV[1])
	return redis.call('EXPIRE', KEYS[1], ARGV[2])
else
  if val == ARGV[1] then
		return redis.call('EXPIRE', KEYS[1], ARGV[2])
	end
end
return 0
`
	res, err := r.Eval(ctx, sc, []string{k}, v, int64(timeOut/time.Second))
	if err != nil {
		return false, err
	}
	return res.(int64) != 0, err
}

// LockWithTimeOut 获取锁，直到超时或者成功
//
// 尝试获取锁，如果失败会重试，直到超时；
//
// Parameters:
//   - enum: 类型为 PrefixEnum 的key前缀值,可为空
//   - k: 类型为 string 锁的key
//   - timeOut: 类型为 time.Duration 锁的超时时间，默认为5s
//   - retryDur: 类型为 time.Duration 重试间隔时间,默认为10ms
//   - expire: 类型为 time.Duration 锁的过期时间，默认为5s
//
// Returns:
//   - string: 锁的key
//   - error: 获取锁期间的错误信息
func (r *Redis) LockWithTimeOut(ctx context.Context, enum prefixenum.PrefixEnum, k string, timeOut, retryDur, expire time.Duration) (key string, err error) {
	k = getKey(enum, k)
	var (
		ok bool
		t  *time.Timer
	)
	defer func() {
		if t != nil {
			t.Stop()
		}

	}()

	for {
		key = fmt.Sprint(time.Now().UnixNano())
		if ok, err = r.UniversalClient.SetNX(ctx, k, key, expire).Result(); err != nil {
			return
		}
		if ok {
			return
		}
		timeOut -= retryDur
		if timeOut <= 0 {
			err = fmt.Errorf("Redis lock timeout: %v", k)
			return
		}
		if t == nil {
			t = time.NewTimer(retryDur)
		} else {
			t.Reset(retryDur)
		}
		<-t.C
	}
}

func NewLockParam(enum prefixenum.PrefixEnum, k string) model.LockParam {
	return model.LockParam{
		Enum:            enum,
		K:               k,
		WaitTimeOut:     5 * time.Second,
		RetryInterval:   100 * time.Millisecond,
		Expire:          5 * time.Second,
		RenewalInterval: 3 * time.Second,
	}
}

func (r *Redis) LockWaitTimeOutWithRenewalV2(ctx context.Context, param model.LockParam, do func() error) (bool, error) {
	skywalking.StartRedis()
	defer skywalking.StopSpan()
	if param.Expire == 0 {
		param.Expire = 5 * time.Second
	}
	if param.RenewalInterval == 0 {
		param.RenewalInterval = 3 * time.Second
	}
	if param.WaitTimeOut == 0 {
		param.WaitTimeOut = 5 * time.Second
	}
	// 进行加锁
	lockKey, err := r.LockWithTimeOut(ctx, param.Enum, param.K, param.WaitTimeOut, param.RetryInterval, param.Expire)
	if err != nil {
		return false, err
	}
	return true, r.renewal(ctx, getKey(param.Enum, param.K), lockKey, param.RenewalInterval, param.Expire, do)
}

// LockWaitTimeOutWithRenewal 获取续约锁，直到超时或者成功
//
// 尝试获取分布式锁，如果失败会重试，直到超时；如果成功会对锁续约，并执行业务逻辑。锁的释放在do函数之后进行
//
// Parameters:
//   - enum: 类型为 PrefixEnum 的key前缀值,可为空
//   - k: 类型为 string 锁的key
//   - waitTimeOut: 类型为 time.Duration 等待锁的超时时间，默认为5s
//   - retryInterval: 类型为 time.Duration 重试获取锁的间隔时间,默认为10ms
//   - timeOut: 类型为 time.Duration 锁的超时时间，默认为5s
//   - renewalInterval: 类型为 time.Duration 续约时间间隔，默认为3s
//     -do: 类型为 func() error 需要执行的业务逻辑
//
// Returns:
//   - bool: true=获取锁成功；false=获取锁失败
//   - error: 获取锁期间的错误信息
func (r *Redis) LockWaitTimeOutWithRenewal(ctx context.Context, enum prefixenum.PrefixEnum, k string, waitTimeOut, retryInterval, timeOut, renewalInterval time.Duration, do func() error) (bool, error) {
	skywalking.StartRedis()
	defer skywalking.StopSpan()
	if timeOut == 0 {
		timeOut = 5 * time.Second
	}
	if renewalInterval == 0 {
		renewalInterval = 3 * time.Second
	}
	if waitTimeOut == 0 {
		waitTimeOut = 5 * time.Second
	}
	// 进行加锁
	lockKey, err := r.LockWithTimeOut(ctx, enum, k, waitTimeOut, retryInterval, timeOut)
	if err != nil {
		return false, err
	}
	return true, r.renewal(ctx, getKey(enum, k), lockKey, renewalInterval, timeOut, do)
}

// LockUntilSuccWithRenewal 获取续约锁，直到成功
//
// 如果获取锁失败，则会一直重试，直到成功。获取锁成功后，会执行续约操作和do这个函数，do这个函数就是需要操作的业务逻辑；解锁操作会在do执行完后执行
//
// Parameters:
//   - enum: 类型为 PrefixEnum 的key前缀值,可为空
//   - k: 类型为 string 锁的key
//   - retryInterval: 类型为 time.Duration 重试间隔时间,默认为10ms
//   - timeOut: 类型为 time.Duration 锁的超时时间，默认为5s
//   - renewalInterval: 类型为 time.Duration 续约间隔时间，默认为3s
//   - do: 类型为 func() error 需要执行的业务逻辑
//
// Returns:
//   - error: 错误信息，如果为空则表示执行成功
func (r *Redis) LockUntilSuccWithRenewal(ctx context.Context, enum prefixenum.PrefixEnum, k string, retryInterval, timeOut, renewalInterval time.Duration, do func() error) error {
	skywalking.StartRedis()
	defer skywalking.StopSpan()
	k = getKey(enum, k)
	lockKey := uuid.NewString()
	if timeOut == 0 {
		timeOut = 5 * time.Second
	}
	if renewalInterval == 0 {
		renewalInterval = 3 * time.Second
	}
	if retryInterval == 0 {
		retryInterval = time.Millisecond * 10
	}
	log.InfoCtx(ctx, "开始获取锁", zap.String("key", k))
	ok, err := r.UniversalClient.SetNX(ctx, k, lockKey, timeOut).Result()
	if err != nil {
		return err
	}
	if !ok {
		// 尝试加锁
		for {
			log.InfoCtx(ctx, "再次获取锁", zap.String("key", k), zap.String("value", lockKey))
			bo, err := r.UniversalClient.SetNX(ctx, k, lockKey, timeOut).Result()
			if err != nil { //加锁错误，直接调用loadFunc
				return err
			} else if !bo { //加锁失败，等待重试
				//等待重试
				time.Sleep(retryInterval)
				continue
			}
			//加锁成功
			break
		}
	}
	log.InfoCtx(ctx, "获取锁成功", zap.String("key", k), zap.String("value", lockKey))
	return r.renewal(ctx, k, lockKey, renewalInterval, timeOut, do)
}

// 续租操作
func (r *Redis) renewal(ctx context.Context, k string, lockValue interface{}, renewalInterval, timeOut time.Duration, do func() error) error {
	skywalking.StartRedis()
	defer skywalking.StopSpan()
	// 清理锁
	defer func() {
		// Only release after do exits. Cancellation alone does not stop the business
		// callback. Preserve tenant metadata. This deadline bounds pool/retry waits;
		// socket I/O still uses the client read/write timeouts (not a strict 2s total).
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		if bo, err := r.UnLock(cleanupCtx, "", k, lockValue); !bo {
			log.WarnCtx(cleanupCtx, "解锁失败", zap.String("key", k), zap.Any("value", lockValue), zap.Error(err))
		}
	}()
	// 监控续期分布式锁
	// Automatic renewal follows the business callback lifetime, not the request.
	// A canceled request must not let the lease expire while do is still running.
	f := r.RenewalLock(context.WithoutCancel(ctx), "", k, lockValue, renewalInterval, timeOut)
	defer f()

	if do != nil {
		return do()
	}
	return nil
}

// RenewalLock 续期锁
//
// 对已经加锁的key进行续期。续期期间如果发现锁的value值与传入的lockValue不一致，则会取消续期。
//
// Parameters:
//   - enum: 类型为 PrefixEnum 的key前缀值,可为空
//   - k: 类型为 string 需要续期的锁key
//   - lockValue: 类型为 interface{} 锁对应的value
//   - renewalInterval: 类型为 time.Duration 每次续期时间间隔
//   - timeout: 类型为 time.Duration 锁的超时时间
//   - maxRenewalTime: 类型为 time.Duration ,可选参数，最大的续期时间，到时间后不再续期
//
// Returns:
//   - func: 停止续期函数
func (r *Redis) RenewalLock(ctx context.Context, enum prefixenum.PrefixEnum, k string, lockValue interface{}, renewalInterval, timeout time.Duration, maxRenewalTime ...time.Duration) (stopRenewal func()) {
	skywalking.StartRedis()
	defer skywalking.StopSpan()
	k = getKey(enum, k)
	// 通知监控退出
	var stopChan = make(chan struct{})
	// 使用lua脚本判断和续期
	sc := `
local key = KEYS[1]
local expected_value = ARGV[1]
local new_ttl = tonumber(ARGV[2])

-- 获取key的当前值
local current_value = redis.call('GET', key)

-- 如果当前值与预期值相等，更新超时时间
if current_value == expected_value then
    return redis.call('EXPIRE', key, new_ttl)
else
    return 0 -- 返回0表示未进行任何处理
end
`
	if len(maxRenewalTime) > 0 {
		ref := skywalking.CaptureContext()
		// 监控续期分布式锁
		go func() {
			skywalking.ContinueContext(ref)
			t := time.NewTicker(renewalInterval)
			t2 := time.NewTimer(maxRenewalTime[0])
			defer t.Stop()
			defer t2.Stop()
			for {
				select {
				case <-t.C:

					ok, err := r.UniversalClient.Eval(ctx, sc, []string{k}, lockValue, int64(timeout/time.Second)).Bool()
					if err != nil {
						log.InfoCtx(ctx, "续期分布式锁错误", zap.String("k", k), zap.Error(err))
						return
					}
					if !ok {
						log.WarnCtx(ctx, "续期分布式锁失败，锁已经被释放", zap.String("key", k))
						return
					} else {
						log.WarnCtx(ctx, "续期分布式锁", zap.String("key", k), zap.String("续期间隔", renewalInterval.String()))
					}
					//t.Reset(renewalInterval)
				case <-stopChan:
					log.DebugCtx(ctx, "停止续约锁", zap.String("key", k), zap.Any("value", lockValue))
					return
				case <-ctx.Done():
					return
				case <-t2.C: //达到了最大的续期时间
					log.InfoCtx(ctx, "续约锁到达最大时间", zap.String("key", k), zap.Any("value", lockValue))
					return
				}
			}
		}()
	} else {
		ref := skywalking.CaptureContext()
		// 监控续期分布式锁
		go func() {
			skywalking.ContinueContext(ref)
			t := time.NewTicker(renewalInterval)
			defer t.Stop()
			for {
				select {
				case <-t.C:

					ok, err := r.UniversalClient.Eval(ctx, sc, []string{k}, lockValue, int64(timeout/time.Second)).Bool()
					if err != nil {
						log.InfoCtx(ctx, "续期分布式锁错误", zap.String("k", k), zap.Error(err))
						return
					}
					if !ok {
						log.WarnCtx(ctx, "续期分布式锁失败，锁已经被释放", zap.Any("key", k))
						return
					} else {
						log.WarnCtx(ctx, "续期分布式锁", zap.String("key", k), zap.String("续期间隔", renewalInterval.String()))
					}
					//t.Reset(renewalInterval)
				case <-stopChan:
					log.DebugCtx(ctx, "停止续约锁", zap.String("key", k), zap.Any("value", lockValue))
					return
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	return func() {
		close(stopChan)
	}
}

func Str2Bytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}

// b2s converts byte slice to a string without memory allocation.
//
// Note it may break if string and/or slice header will change
// in the future go versions.
func Bytes2Str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
