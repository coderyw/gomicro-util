package redis

import (
	"context"
	"net"

	"github.com/redis/go-redis/v9"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/metadata"
	"github.com/coderyw/gomicro-util/library/namespace"
	"go.uber.org/zap"
)

// 初始化 redis 命令规则
func initKeySpec() error {
	cms, err := GetRedis(context.Background()).Client().Command(context.Background()).Result()
	if err != nil {
		log.Error("初始化 spec key 错误", zap.Error(err))
		return err
	}
	for _, v := range cms {
		commandKeySpecs[v.Name] = KeySpec{
			FirstKey: int(v.FirstKeyPos),
			LastKey:  int(v.LastKeyPos),
			KeyStep:  int(v.StepCount),
		}
	}
	return nil
}

type KeySpec struct {
	FirstKey int
	LastKey  int
	KeyStep  int
}

// redis 的命令规则
var commandKeySpecs = map[string]KeySpec{

	"get":    {1, 1, 1},
	"set":    {1, 1, 1},
	"del":    {1, -1, 1},
	"exists": {1, -1, 1},

	"mget": {1, -1, 1},
	"mset": {1, -1, 2},

	"hget": {1, 1, 1},
	"hset": {1, 1, 1},

	"lpush": {1, 1, 1},
	"rpush": {1, 1, 1},

	"sadd": {1, 1, 1},

	"zadd": {1, 1, 1},

	"zunionstore": {1, -1, 1},

	"eval":    {3, -1, 1},
	"evalsha": {3, -1, 1},
}

// 解析执行命令，获取到 key 的索引
func ParseKeys(args []interface{}, spec KeySpec) []int {

	var result []int

	n := len(args)

	first := spec.FirstKey
	last := spec.LastKey
	step := spec.KeyStep

	if first <= 0 {
		return nil
	}

	if last < 0 {
		last = n + last
	}

	for i := first; i <= last && i < n; i += step {
		result = append(result, i)
	}

	return result
}

// 添加 key 前缀
func addKeyPrefix(ctx context.Context, cmd redis.Cmder) {
	// 从上下文获取当前命名空间
	ns := metadata.Get[string](ctx, metadata.CONTEXT_NAME_SPACE)
	if ns == "" { //如果为空，则不处理，忽略
		return
	}
	args := cmd.Args()
	val, ok := commandKeySpecs[cmd.Name()]
	if !ok {
		return
	}
	indexes := ParseKeys(args, val)
	// 为每个key增加前缀
	for _, idx := range indexes {
		key, ok := args[idx].(string)
		if !ok {
			continue
		}
		// todo 高性能场景下 优化
		args[idx] = namespace.FmtByType(namespace.NAMESPACE_TYPE_REDIS, ns, key)
	}
}

type keyPrefixHook struct{}

func (h keyPrefixHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

func (h keyPrefixHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		addKeyPrefix(ctx, cmd)
		return next(ctx, cmd)
	}
}

func (h keyPrefixHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		for _, cmd := range cmds {
			addKeyPrefix(ctx, cmd)
		}
		return next(ctx, cmds)
	}
}
