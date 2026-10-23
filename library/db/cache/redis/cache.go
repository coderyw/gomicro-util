// Package redis
// @Author: yinwei
// @File: cache
// @Version: 1.0.0
// @Date: 2023/11/14 10:54

package redis

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	facade2 "github.com/coderyw/gomicro-util/library/db/cache/redis/facade"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/skywalking"
	"github.com/coderyw/gomicro-util/model/constrict"
	"github.com/coderyw/gomicro-util/model/facade"
	"github.com/coderyw/gomicro-util/util"
	"github.com/coderyw/gomicro-util/util/mapstructure"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type LoadFunc[T any] func(context.Context) (T, error)
type SaveFunc[T any] func(context.Context, T) error

// GetIntCache 从缓存中加载整型value数据
//
// 缓存如果获取错误，则会直接返回错误。如果缓存获取为空，则会通过 load 函数加载；在加载期间会获取一个锁，锁的key是 key:lock这样格式，
// 如果锁获取失败，会重复获取，直到成功，每隔10ms获取一次锁；获取锁成功后，会去缓存继续获取一次，如果依然未存在，才可调用load函数。
//
// Parameters:
//   - rd: Redis 实例
//   - key: 缓存的key string
//   - exp: 缓存过期时间，单位秒，0表示不设置过期时间
//   - load: func() (T, error) 类型的加载函数
//
// Returns:
//   - T : 返回传入的泛型数据类型
//   - error: 错误信息
func GetIntCache[T constrict.Integer](ctx context.Context, rd facade2.RedisInterface, key string, exp time.Duration, load func() (T, error)) (T, error) {
	skywalking.StartRedis()
	defer skywalking.StopSpan()

	var (
		err error
	)

	getCache := func() (T, error) {
		// 从list左方开始读取元素
		val, err := rd.GetValueByInt64(ctx, "", key)
		if err != nil {
			if !errors.Is(err, RedisNil) {
				rd.Delete(ctx, "", key)
				log.ErrorCtx(ctx, "GetIntCache", zap.String("key", key), zap.Error(err))
			}
			return 0, err
		} else if exp > 0 {
			rd.Expire(ctx, "", key, exp)
		}
		return T(val), nil
	}
	if n, err := getCache(); err == nil {
		return n, nil
	}
	lockK := fmt.Sprintf("%v:lock", key)

	var resp T

	// 获取续约锁
	_, err = rd.LockWithRenewal(ctx, "", lockK, 5*time.Second, 3*time.Second, func() (err error) {

		//再次加载尝试,因为可能有其他锁已经在加载
		if resp, err = getCache(); err == nil {
			return nil
		}

		resp, err = load()
		if err == nil {
			if err = rd.SetValue(ctx, "", key, resp, exp); err != nil {
				log.ErrorCtx(ctx, "保存数据到列表错误", zap.Error(err))
			}
			return nil
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		} else {
			return err
		}
	})
	return resp, err
}

// GetStrCache 从缓存中加载字符串value数据
//
// 缓存如果获取错误，则会直接返回错误。如果缓存获取为空，则会通过 load 函数加载；在加载期间会获取一个锁，锁的key是 key:lock这样格式，
// 如果锁获取失败，会重复获取，直到成功，每隔10ms获取一次锁；获取锁成功后，会去缓存继续获取一次，如果依然未存在，才可调用load函数。
//
// Parameters:
//   - rd: Redis 实例
//   - key: 缓存的key string
//   - exp: 缓存过期时间，单位秒，0表示不设置过期时间
//   - load: func() (T, error) 类型的加载函数
//
// Returns:
//   - string : 返回字符
//   - error: 错误信息
func GetStrCache(ctx context.Context, rd facade2.RedisInterface, key string, exp time.Duration, load func() (string, error)) (string, error) {
	skywalking.StartRedis()
	defer skywalking.StopSpan()

	var (
		err error
	)
	getCache := func() (string, error) {
		// 从list左方开始读取元素
		val, err := rd.GetValue(ctx, "", key)
		if err != nil {
			if !errors.Is(err, RedisNil) {
				rd.Delete(ctx, "", key)
				log.ErrorCtx(ctx, "GetStrCache", zap.String("key", key), zap.Error(err))
			}
			return "", err
		} else if exp > 0 {
			rd.Expire(ctx, "", key, exp)
		}
		return val, nil
	}
	if n, err := getCache(); err == nil {
		return n, nil
	}
	lockK := fmt.Sprintf("%v:lock", key)

	var resp string

	// 获取续约锁
	_, err = rd.LockWithRenewal(ctx, "", lockK, 5*time.Second, 3*time.Second, func() (err error) {

		//再次加载尝试,因为可能有其他锁已经在加载
		if resp, err = getCache(); err == nil {
			return nil
		}

		resp, err = load()
		if err == nil {
			if err = rd.SetValue(ctx, "", key, resp, exp); err != nil {
				log.ErrorCtx(ctx, "保存数据到列表错误", zap.Error(err))
			}
			return nil
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		} else {
			return err
		}
	})
	return resp, err
}

// GetObjCache
//
//	@Description: 从缓存获取数据实体，获取字符串然后反序列化成实体。如果获取不到则使用load函数加载
//	@Author yw 2024-07-03 10:13:55
//	@param ctx
//	@param key 缓存的key
//	@param load 加载函数，可从db中加载
//	@param target 反序列化的目标实体
//	@param exp 缓存过期时间
//	@return T
//	@return error
func GetObjCache[T any](ctx context.Context, rd facade2.RedisInterface, key string, load func(context.Context) (T, error), target T, exp time.Duration) (t T, err error) {
	skywalking.StartRedis()
	defer skywalking.StopSpan()
	lockK := fmt.Sprintf("%v:lock", key)
	var (
		valK int64
	)

	if err = getObjCache(ctx, rd, target, key, exp); err == nil {
		return target, nil
	}
	for {
		//获取加载锁
		valK = time.Now().UnixNano()
		bo, err := rd.Lock(ctx, "", lockK, valK, 5*time.Second)
		if err != nil { //加锁错误，直接调用loadFunc
			log.Error("加锁错误", zap.String("key", key), zap.Error(err))
		} else if !bo { //加锁失败，等待重试
			//等待重试
			time.Sleep(time.Millisecond * 10)
			continue
		}
		//加锁成功
		break
	}

	defer rd.UnLock(ctx, "", lockK, valK)
	//再次加载尝试,因为可能有其他锁已经在加载
	if err = getObjCache(ctx, rd, target, key, exp); err == nil {
		return target, nil
	}

	target, err = load(ctx)
	if err == nil {
		var data []byte
		if data, err = json.Marshal(target); err != nil {
			return target, err
		}
		if err = rd.SetValue(ctx, "", key, util.Bytes2Str(data), exp); err != nil {
			log.Error("保存数据到列表错误", zap.Error(err))
		}
		return target, nil
	} else {
		return t, err
	}
}

// GetObjCacheT
//
//	@Description:通过反射返回类型数据
//	@Author yw 2024-07-09 08:56:18
//	@param ctx
//	@param key
//	@param load
//	@param exp
//	@return target
//	@return err
func GetObjCacheT[T any](ctx context.Context, rd facade2.RedisInterface, key string, load func(context.Context) (T, error), exp time.Duration) (target T, err error) {
	skywalking.StartRedis()
	defer skywalking.StopSpan()
	lockK := fmt.Sprintf("%v:lock", key)
	var (
		valK int64
	)

	if target, err = getObjCacheT[T](ctx, rd, key, exp); err == nil {
		return target, nil
	}
	for {
		//获取加载锁
		valK = time.Now().UnixNano()
		bo, err := rd.Lock(ctx, "", lockK, valK, 5*time.Second)
		if err != nil { //加锁错误，直接调用loadFunc
			log.Error("加锁错误", zap.String("key", key), zap.Error(err))
		} else if !bo { //加锁失败，等待重试
			//等待重试
			time.Sleep(time.Millisecond * 10)
			continue
		}
		//加锁成功
		break
	}

	defer rd.UnLock(ctx, "", lockK, valK)
	//再次加载尝试,因为可能有其他锁已经在加载
	if target, err = getObjCacheT[T](ctx, rd, key, exp); err == nil {
		return target, nil
	}

	target, err = load(ctx)
	if err == nil {
		var data []byte
		if data, err = json.Marshal(target); err != nil {
			return target, err
		}
		if err = rd.SetValue(ctx, "", key, util.Bytes2Str(data), exp); err != nil {
			log.Error("保存数据到列表错误", zap.Error(err))
		}
		return target, nil
	} else {
		return target, err
	}
}

func IsEmpty(v any) bool {
	return reflect.ValueOf(v).IsZero()
}

// GetFromCacheOrDb 从缓存加载数据或者从db中加载
//
// 先从 loadCache 获取数据，如果获取到数据为空，或则获取到数据是 redis.Nil ,
// 则会调用 loadFromDb 获取数据，并通过 saveCache 将数据保存到缓存中。
//
// Parameters:
//   - ctx: context.Context 上下文
//   - rd: Redis redis实例
//   - key: string 缓存的key
//   - loadCache: LoadFunc[T] 从缓存加载数据函数
//   - loadFromDb: LoadFunc[T] 从db加载数据函数
//   - saveCache: SaveFunc[T] 保存到缓存函数
//
// Returns:
//   - target T 返回数据
//   - err error 错误信息
func GetFromCacheOrDb[T any](ctx context.Context, rd facade2.RedisInterface, key string, loadCache LoadFunc[T], loadFromDb LoadFunc[T], saveCache SaveFunc[T]) (target T, err error) {
	skywalking.StartRedis()
	defer skywalking.StopSpan()

	// 先从缓存获取
	target, err = loadCache(ctx)
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			return target, err
		}
		// 错误是redis.Nil，则从数据库加载
	} else if !IsEmpty(target) { //target非空，则直接返回
		return target, nil
	}
	lockK := fmt.Sprintf("%v:lock", key)
	ok, err := rd.LockWaitTimeOutWithRenewal(ctx, "", lockK, 20*time.Second, 100*time.Millisecond, 20*time.Second, 5*time.Second, func() error {
		//再次从缓存加载,防止获取到锁前已经有协程加载了数据
		target, err = loadCache(ctx)
		if err != nil {
			if !errors.Is(err, redis.Nil) {
				return err
			}
			// 错误是redis.Nil，则从数据库加载
		} else if !IsEmpty(target) { //target非空，则直接返回
			return nil
		}

		//从缓存加载为空，从数据库加载
		target, err = loadFromDb(ctx)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// fixme 是否配置缓存表示数据不存在
				return err
			}
			return err
		}
		// 保存到缓存
		if !IsEmpty(target) {
			err = saveCache(ctx, target)
			if err != nil {
				log.ErrorCtx(ctx, "保存到缓存错误", zap.Any("data", target), zap.Error(err))
			}
		}
		return nil
	})
	if err != nil {
		return target, err
	}
	if !ok {
		log.ErrorCtx(ctx, "获取续约锁失败", zap.String("key", key))
	}
	// 重新加载
	return loadCache(ctx)

}

func GetListCacheT[T any](ctx context.Context, rd facade2.RedisInterface, key string, load func(context.Context) ([]T, error), exp time.Duration) (target []T, err error) {
	skywalking.StartRedis()
	defer skywalking.StopSpan()
	lockK := fmt.Sprintf("%v:lock", key)
	var (
		valK int64
	)

	if target, err = getObjCacheT[[]T](ctx, rd, key, exp); err == nil {
		return target, nil
	}
	for {
		//获取加载锁
		valK = time.Now().UnixNano()
		bo, err := rd.Lock(ctx, "", lockK, valK, 5*time.Second)
		if err != nil { //加锁错误，直接调用loadFunc
			log.Error("加锁错误", zap.String("key", key), zap.Error(err))
		} else if !bo { //加锁失败，等待重试
			//等待重试
			time.Sleep(time.Millisecond * 10)
			continue
		}
		//加锁成功
		break
	}

	defer rd.UnLock(ctx, "", lockK, valK)
	//再次加载尝试,因为可能有其他锁已经在加载
	if target, err = getObjCacheT[[]T](ctx, rd, key, exp); err == nil {
		return target, nil
	}

	target, err = load(ctx)
	if err == nil {
		var data []byte
		if data, err = json.Marshal(target); err != nil {
			return target, err
		}
		if err = rd.SetValue(ctx, "", key, util.Bytes2Str(data), exp); err != nil {
			log.Error("保存数据到列表错误", zap.Error(err))
		}
		return target, nil
	} else {
		return target, err
	}
}

// GetZSetList 获取升序全部有序集合数据。获取不到会从load函数中加载
//
//	请注意：只适合小的集合，因为方法是会遍历全部数据，所以请不要使用此方法获取大集合数据
//
// Parameters:
//   - ctx: context.Context 上下文
//   - rd: Redis redis实例
//   - key: string 有序集合key
//   - load: func(ctx context.Context) ([]T, error) 加载函数
//   - exp: time.Duration 有序集合过期时间
//   - vs: ...bool true=降序，false=升序，默认升序
//
// Returns:
//   - []T 返回具体数据
//   - error 错误信息
func GetZSetList[T facade2.ZSetAction](ctx context.Context, rd facade2.RedisInterface, key string, load func(ctx context.Context) ([]T, error), exp time.Duration, vs ...bool) ([]T, error) {
	getCache := func() ([]T, error) {
		// 从list左方开始读取元素
		var (
			val []redis.Z
			err error
		)
		if len(vs) > 0 && vs[0] {
			// 降序
			val, err = rd.ZRevRangeWithScores(ctx, "", key, 0, -1)
		} else {
			val, err = rd.ZRangeWithScores(ctx, "", key, 0, -1)
		}

		if err != nil {
			if !errors.Is(err, RedisNil) {
				rd.Delete(ctx, "", key)
				log.ErrorCtx(ctx, "GetIntCache", zap.String("key", key), zap.Error(err))
			}
			return nil, err
		} else if len(val) == 0 {
			return nil, RedisNil
		}
		if exp > 0 {
			rd.Expire(ctx, "", key, exp)
		}
		arr := make([]T, len(val))
		for i, v := range val {
			obj, _ := ref[T]()
			obj.Set(v.Member.(string), decimal.NewFromFloat(v.Score))
			arr[i] = obj
		}
		return arr, nil
	}
	if n, err := getCache(); err == nil {
		return n, nil
	}
	lockK := fmt.Sprintf("%v:lock", key)

	var resp []T

	// 获取续约锁
	_, err := rd.LockWithRenewal(ctx, "", lockK, 5*time.Second, 3*time.Second, func() (err error) {

		//再次加载尝试,因为可能有其他锁已经在加载
		if resp, err = getCache(); err == nil {
			return nil
		}

		resp, err = load(ctx)
		if err == nil {
			rzs := make([]redis.Z, len(resp))
			for i, v := range resp {
				key, score := v.Get()
				rzs[i] = redis.Z{
					Score:  score.InexactFloat64(),
					Member: key,
				}
			}
			rd.ZMAdd(ctx, "", key, rzs...)
			if exp > 0 {
				rd.Expire(ctx, "", key, exp)
			}
			return nil
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		} else {
			return err
		}
	})
	return resp, err
}

// GetHMCacheFieldInteger
//
//	@Description: 从缓存中加载hashmap的field，不存在（field不存在也会执行）会执行load
//	@Author	yw 2024-12-02 17:44:42
//	@param	ctx	上下文
//	@param	key	redis的key
//	@param	field	hashmap中的field
//	@param	load	数据不存在时的加载函数
//	@param	exp	hashmap的过期时间
//	@return	target
//	@return	err
func GetHMCacheFieldInteger[T constrict.Integer](ctx context.Context, rd facade2.RedisInterface, key, field string, load func() (map[string]interface{}, error), exp time.Duration) (target T, err error) {
	lockK := fmt.Sprintf("%v:lock", key)
	var (
		valK int64
	)
	resp, err := rd.HGet(ctx, "", key, field)
	if err == nil {
		val, err := strconv.ParseInt(resp, 10, 64)
		if err != nil {
			log.ErrorCtx(ctx, "解析redis数据错误", zap.String("key", key), zap.String("field", field), zap.Int64("data", val))
		} else {
			return T(val), nil
		}
	} else if !errors.Is(err, RedisNil) {
		return 0, err
	}
	for {
		//获取加载锁
		valK = time.Now().UnixNano()
		bo, err := rd.Lock(ctx, "", lockK, valK, 5*time.Second)
		if err != nil { //加锁错误，直接调用loadFunc
			log.ErrorCtx(ctx, "加锁错误", zap.String("key", key), zap.Error(err))
		} else if !bo { //加锁失败，等待重试
			//等待重试
			time.Sleep(time.Millisecond * 10)
			continue
		}
		//加锁成功
		break
	}

	defer rd.UnLock(ctx, "", lockK, valK)
	//再次加载尝试,因为可能有其他锁已经在加载
	resp, err = rd.HGet(ctx, "", key, field)
	if err == nil {
		val, err := strconv.ParseInt(resp, 10, 64)
		if err != nil {
			log.ErrorCtx(ctx, "解析redis数据错误", zap.String("key", key), zap.String("field", field), zap.Int64("data", val))
		} else {
			return T(val), nil
		}
	} else if !errors.Is(err, RedisNil) {
		return 0, err
	}
	var m map[string]interface{}
	m, err = load()
	if err != nil {
		return
	}
	if err = rd.HMSetMap(ctx, "", key, m, exp); err != nil {
		return
	}
	switch val := m[field].(type) {
	case int64:
		return T(val), nil
	case uint64:
		return T(val), nil
	case int32:
		return T(val), nil
	case uint32:
		return T(val), nil
	case int16:
		return T(val), nil
	case uint16:
		return T(val), nil
	case int8:
		return T(val), nil
	case uint8:
		return T(val), nil
	case int:
		return T(val), nil
	case uint:
		return T(val), nil
	case string:
		n, err := strconv.ParseInt(val, 10, 64)
		return T(n), err
	default:
		return 0, errors.New(fmt.Sprintf("数据类型错误: val=%v, type=%T", val, val))
	}
}

// GetHMCacheField 获取hashmap的field，不存在（field不存在也会执行）会执行load
//
// 获取hashmap的field，不存在（field不存在也会执行）会执行load，load表示加载函数，load返回一个map[string]interface{}，load返回的map[string]interface{}的key为field，
// value为基础类型，如果不是基础类型，会报错.
//
// Parameters:
//   - ctx: 上下文
//   - rd: redis客户端 Redis
//   - key: redis的key
//   - field: hashmap中的field
//   - load: 数据不存在时的加载函数
//   - exp: hashmap的过期时间
//
// Returns:
//   - target: 获取到的field的值，如果field不存在，返回空字符串
//   - err: 错误信息，如果获取成功，err为nil;对于field或者load加载不到数据，会返回 gorm.NotFound 错误
func GetHMCacheField(ctx context.Context, rd facade2.RedisInterface, key, field string, load func() (map[string]interface{}, error), exp time.Duration) (target string, err error) {
	lockK := fmt.Sprintf("%v:lock", key)
	var (
		valK string
	)
	resp, err := rd.HGet(ctx, "", key, field)
	if err == nil {
		return resp, nil
	} else if !errors.Is(err, RedisNil) {
		return "", err
	}
	valK, err = rd.LockWithTimeOut(ctx, "", lockK, 5*time.Second, 100*time.Millisecond, 5*time.Second)
	if err != nil {
		return "", err
	}
	defer rd.UnLock(ctx, "", lockK, valK)

	//再次加载尝试,因为可能有其他锁已经在加载
	resp, err = rd.HGet(ctx, "", key, field)
	if err == nil {
		return resp, nil
	} else if !errors.Is(err, RedisNil) {
		return "", err
	}
	var m map[string]interface{}
	m, err = load()
	if err != nil {
		return
	}
	if err = rd.HMSetMap(ctx, "", key, m, exp); err != nil {
		return
	}
	switch val := m[field].(type) {
	case int64:
		return strconv.FormatInt(val, 10), nil
	case uint64:
		return strconv.FormatUint(val, 10), nil
	case int32:
		return strconv.FormatInt(int64(val), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(val), 10), nil
	case int16:
		return strconv.FormatInt(int64(val), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(val), 10), nil
	case int8:
		return strconv.FormatInt(int64(val), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(val), 10), nil
	case int:
		return strconv.FormatInt(int64(val), 10), nil
	case uint:
		return strconv.FormatUint(uint64(val), 10), nil
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 64), nil
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(val), nil
	case string:
		return val, nil
	default:
		return "", errors.New(fmt.Sprintf("数据类型错误: val=%v, type=%T", val, val))
	}
}

func GetObjHMCache[T facade.HMField](ctx context.Context, rd facade2.RedisInterface, key string, load func() (map[string]interface{}, error), exp time.Duration, fields ...string) (target T, err error) {
	lockK := fmt.Sprintf("%v:lock", key)
	var (
		valK int64
	)
	target, _ = ref[T]()
	try := func() (bool, error) {
		if len(fields) > 0 { // 获取部分field
			resp, err := rd.HMGet(ctx, "", key, fields...)
			if err == nil {
				targetM := make(map[string]string, len(resp))
				for i, v := range resp {
					if v == nil { // 部分field为nil，需要重新加载
						return false, nil
					}
					targetM[fields[i]] = v.(string)
				}
				return true, mapstructure.Decode(targetM, target)
			} else if !errors.Is(err, RedisNil) {
				return false, err
			}
		} else {
			// 获取全部field
			resp, err := rd.HGetALLMap(ctx, "", key)
			if err == nil && len(resp) > 0 {
				// 校验hashmap 字段是否齐全
				if err := target.CheckField(resp); err != nil {
					log.WarnCtx(ctx, "缓存hashmap数据不全即将从db加载", zap.Error(err))
					return false, nil
				}
				return true, mapstructure.Decode(resp, target)
			} else if err != nil && !errors.Is(err, RedisNil) {
				return false, err
			}
		}
		return false, nil
	}
	got, err := try()
	if err != nil {
		return target, err
	}
	if got {
		return target, nil
	}

	// 从db加载
getFromDb:
	//获取加载锁
	valK = time.Now().UnixNano()
	bo, err := rd.Lock(ctx, "", lockK, valK, 5*time.Second)
	if err != nil { //加锁错误，直接调用loadFunc
		log.ErrorCtx(ctx, "加锁错误", zap.String("key", key), zap.Error(err))
	} else if !bo { //加锁失败，等待重试
		//等待重试
		time.Sleep(time.Millisecond * 10)
		goto getFromDb
	}
	//加锁成功
	defer rd.UnLock(ctx, "", lockK, valK)
	//再次加载尝试,因为可能有其他锁已经在加载
	got, err = try()
	if err != nil {
		return target, err
	}
	if got {
		return target, nil
	}
	var m map[string]interface{}
	m, err = load()
	if err != nil {
		return
	}
	if err = rd.HMSetMap(ctx, "", key, m, exp); err != nil {
		return
	}
	return target, mapstructure.Decode(m, target)
}

//
//func GetHMStrCache(ctx context.Context, rd Redis, key string, load func() ([]interface{}, error), exp time.Duration, fields ...string) (targetM map[string]string, err error) {
//	lockK := fmt.Sprintf("%v:lock", key)
//	var (
//		valK int64
//	)
//	try := func() (got bool, err error) {
//		if len(fields) > 0 { // 获取部分field
//			resp, err := rd.HMGet("", key, fields...)
//			if err == nil {
//				targetM = make(map[string]string, len(resp))
//				for i, v := range resp {
//					if v == nil { // 部分field为nil，需要重新加载
//						return false, nil
//					}
//					targetM[fields[i]] = v.(string)
//				}
//				return true, nil
//			} else if !errors.Is(err, RedisNil) {
//				return false, err
//			}
//		} else {
//			// 获取全部field
//			targetM, err = rd.HGetALLMap("", key)
//			if err == nil && len(targetM) > 0 {
//				return true, nil
//			} else if err != nil && !errors.Is(err, RedisNil) {
//				return false, err
//			}
//		}
//		return false, nil
//	}
//	got, err := try()
//	if err != nil {
//		return nil, err
//	}
//	if got {
//		return
//	}
//
//	// 从db加载
//getFromDb:
//	//获取加载锁
//	valK = time.Now().UnixNano()
//	bo, err := rd.Lock("", lockK, valK, 5*time.Second)
//	if err != nil { //加锁错误，直接调用loadFunc
//		log.ErrorCtx(ctx, "加锁错误", zap.Any("key", key), zap.Error(err))
//	} else if !bo { //加锁失败，等待重试
//		//等待重试
//		time.Sleep(time.Millisecond * 10)
//		goto getFromDb
//	}
//	//加锁成功
//	defer rd.UnLock("", lockK, valK)
//	//再次加载尝试,因为可能有其他锁已经在加载
//	got, err = try()
//	if err != nil {
//		return nil, err
//	}
//	if got {
//		return
//	}
//	var m []interface{}
//	m, err = load()
//	if err != nil {
//		return
//	}
//	if err = rd.HSetSlice("", key, m...); err != nil {
//		return
//	}
//	targetM = make(map[string]string)
//	for i, v := range m {
//		targetM[fields[i]] = v.(string)
//	}
//
//	return m, nil
//}

func getObjCache[T any](ctx context.Context, rd facade2.RedisInterface, target T, key string, exp time.Duration) (err error) {
	skywalking.StartRedis()
	defer skywalking.StopSpan()
	// 从list左方开始读取元素
	var val string
	val, err = rd.GetValue(ctx, "", key)
	if err != nil {
		if !errors.Is(err, RedisNil) {
			rd.Delete(ctx, "", key)
			log.ErrorCtx(ctx, "GetObjCache", zap.String("key", key), zap.Error(err))
		}

		return
	} else {
		rd.Expire(ctx, "", key, exp)
		err = json.Unmarshal(util.Str2Bytes(val), target)
		return err
	}
}

// getObjCacheT[T facade.Marshal]
//
//	@Description: 通过反射返回数据
//	@Author yw 2024-07-09 08:55:28
//	@param key
//	@param exp
//	@return t
//	@return err
func getObjCacheT[T any](ctx context.Context, rd facade2.RedisInterface, key string, exp time.Duration) (t T, err error) {
	skywalking.StartRedis()
	defer skywalking.StopSpan()
	var val string
	val, err = rd.GetValue(ctx, "", key)
	if err != nil {
		if !errors.Is(err, RedisNil) {
			rd.Delete(ctx, "", key)
			log.ErrorCtx(ctx, "GetObjCache", zap.String("key", key), zap.Error(err))
		}

		return
	} else {
		if exp > 0 {
			rd.Expire(ctx, "", key, exp)
		}
		var isPtr bool
		t, isPtr = ref[T]()
		if !isPtr {
			err = json.Unmarshal(util.Str2Bytes(val), &t)
		} else {
			err = json.Unmarshal(util.Str2Bytes(val), t)
		}
		return
	}
}
func ref[T any]() (t T, isPtr bool) {
	// 获取类型的反射信息
	typ := reflect.TypeOf(t)
	if typ.Kind() == reflect.Ptr {
		isPtr = true
		typ = typ.Elem()
	}

	var nt reflect.Value
	if typ.Kind() == reflect.Ptr {
		nt = reflect.New(typ)
	} else if typ.Kind() == reflect.Struct {
		nt = reflect.New(typ).Elem()
	} else if typ.Kind() == reflect.Slice {
		return reflect.MakeSlice(typ, 0, 0).Interface().(T), isPtr
	}
	// 存入缓存
	if isPtr {
		return nt.Addr().Interface().(T), isPtr
	} else {
		return nt.Interface().(T), isPtr
	}
}
