package redis

import (
	"context"
	"strconv"
	"strings"

	goredis "github.com/redis/go-redis/v9"
	"github.com/coderyw/gomicro-util/library/metadata"
	"github.com/coderyw/gomicro-util/library/namespace"
)

type cmdMode uint8

const (
	modeNone cmdMode = iota
	modeFixed1
	modeAllFrom1
	modeEvery2From1
	modeTwoKeys12
	modeBlockPop
	modeStoreNumKeys
	modeEval
	modeXRead
	modeXReadGroup
	modeSort
	modePublish
	modeSubscribe
	modePSubscribe
	modeSSubscribe
	modePubSub
	modeStoreDestAllFrom2 // SUNIONSTORE dst key [key ...]
	modeNumKeysAt2        // LMPOP numkeys key [key ...]
	modeNumKeysAt3        // BLMPOP timeout numkeys key [key ...]
	modeXGroup
	modeXInfo
)

type PrefixHookV3 struct {
	EnablePubSub bool
}

func NewPrefixHookV3() *PrefixHookV3 {
	return &PrefixHookV3{
		EnablePubSub: true,
	}
}

func (h *PrefixHookV3) DialHook(next goredis.DialHook) goredis.DialHook {
	return next
}

func (h *PrefixHookV3) ProcessHook(next goredis.ProcessHook) goredis.ProcessHook {
	return func(ctx context.Context, cmd goredis.Cmder) error {
		opt := getContextOption(ctx)
		if opt == nil || !opt.withoutContext {
			h.rewriteCmd(ctx, cmd)
		}
		return next(ctx, cmd)
	}
}

func (h *PrefixHookV3) ProcessPipelineHook(next goredis.ProcessPipelineHook) goredis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []goredis.Cmder) error {
		opt := getContextOption(ctx)
		if opt == nil || !opt.withoutContext {

			prefix := metadata.Get[string](ctx, metadata.CONTEXT_NAME_SPACE)
			if prefix == "" {
				return next(ctx, cmds)
			}

			for _, cmd := range cmds {
				h.rewriteCmdWithPrefix(prefix, cmd)
			}
		}
		return next(ctx, cmds)
	}
}

func (h *PrefixHookV3) rewriteCmd(ctx context.Context, cmd goredis.Cmder) {
	prefix := metadata.Get[string](ctx, metadata.CONTEXT_NAME_SPACE)
	if prefix == "" {
		return
	}
	h.rewriteCmdWithPrefix(prefix, cmd)
}

func rewriteKey(prefix, key string) string {
	return namespace.FmtByType(namespace.NAMESPACE_TYPE_REDIS, prefix, key)
}

func (h *PrefixHookV3) rewriteCmdWithPrefix(fullPrefix string, cmd goredis.Cmder) {
	if cmd == nil {
		return
	}

	args := cmd.Args()
	if len(args) <= 1 {
		return
	}

	name := cmd.Name()
	if name == "" && len(args) > 0 {
		if s, ok := args[0].(string); ok {
			name = s
		}
	}
	if name == "" {
		return
	}

	mode, ok := cmdModes[name]
	if !ok {
		return
	}

	if !h.EnablePubSub {
		switch mode {
		case modePublish, modeSubscribe, modePSubscribe, modeSSubscribe, modePubSub:
			return
		}
	}

	switch mode {
	case modeFixed1:
		rewriteIndex(args, 1, fullPrefix)

	case modeAllFrom1:
		rewriteRange(args, 1, len(args)-1, 1, fullPrefix)

	case modeEvery2From1:
		rewriteRange(args, 1, len(args)-1, 2, fullPrefix)

	case modeTwoKeys12:
		rewriteIndex(args, 1, fullPrefix)
		rewriteIndex(args, 2, fullPrefix)

	case modeBlockPop:
		if len(args) > 2 {
			rewriteRange(args, 1, len(args)-2, 1, fullPrefix)
		}

	case modeStoreNumKeys:
		// destination numkeys key [key ...]
		rewriteIndex(args, 1, fullPrefix)
		if len(args) < 4 {
			return
		}
		n, ok := toIntFast(args[2])
		if !ok || n <= 0 {
			return
		}
		end := 3 + n - 1
		if end >= len(args) {
			end = len(args) - 1
		}
		rewriteRange(args, 3, end, 1, fullPrefix)

	case modeEval:
		// eval script numkeys key [key ...] arg [arg ...]
		if len(args) < 4 {
			return
		}
		n, ok := toIntFast(args[2])
		if !ok || n <= 0 {
			return
		}
		end := 3 + n - 1
		if end >= len(args) {
			end = len(args) - 1
		}
		rewriteRange(args, 3, end, 1, fullPrefix)

	case modeXRead:
		rewriteXRead(args, fullPrefix)

	case modeXReadGroup:
		rewriteXReadGroup(args, fullPrefix)

	case modeSort:
		rewriteIndex(args, 1, fullPrefix)
		for i := 2; i < len(args)-1; i++ {
			s, ok := args[i].(string)
			if !ok {
				continue
			}
			if s == "store" || s == "STORE" {
				rewriteIndex(args, i+1, fullPrefix)
				i++
			}
		}

	case modePublish:
		rewriteIndex(args, 1, fullPrefix)

	case modeSubscribe, modePSubscribe, modeSSubscribe:
		rewriteRange(args, 1, len(args)-1, 1, fullPrefix)

	case modePubSub:
		// PUBSUB CHANNELS [pattern]
		// PUBSUB SHARDCHANNELS [pattern]
		if len(args) >= 3 {
			if sub, ok := args[1].(string); ok {
				if sub == "channels" || sub == "CHANNELS" || sub == "shardchannels" || sub == "SHARDCHANNELS" {
					rewriteIndex(args, 2, fullPrefix)
				}
			}
		}
	case modeStoreDestAllFrom2:
		// destination key [key ...]
		rewriteIndex(args, 1, fullPrefix)
		if len(args) >= 3 {
			rewriteRange(args, 2, len(args)-1, 1, fullPrefix)
		}
	case modeNumKeysAt2:
		// LMPOP numkeys key [key ...] LEFT|RIGHT [COUNT count]
		if len(args) < 4 {
			return
		}
		n, ok := toIntFast(args[1])
		if !ok || n <= 0 {
			return
		}
		end := 2 + n - 1
		if end >= len(args) {
			end = len(args) - 1
		}
		rewriteRange(args, 2, end, 1, fullPrefix)
	case modeNumKeysAt3:
		// BLMPOP timeout numkeys key [key ...] LEFT|RIGHT [COUNT count]
		if len(args) < 5 {
			return
		}
		n, ok := toIntFast(args[2])
		if !ok || n <= 0 {
			return
		}
		end := 3 + n - 1
		if end >= len(args) {
			end = len(args) - 1
		}
		rewriteRange(args, 3, end, 1, fullPrefix)
	case modeXGroup:
		// XGROUP <subcmd> key ...
		if len(args) >= 3 {
			rewriteIndex(args, 2, fullPrefix)
		}
	case modeXInfo:
		if len(args) >= 3 {
			rewriteIndex(args, 2, fullPrefix)
		}
	}
}

var cmdModes = map[string]cmdMode{
	// key
	"keys":      modeFixed1,
	"expire":    modeFixed1,
	"pexpire":   modeFixed1,
	"expireat":  modeFixed1,
	"pexpireat": modeFixed1,
	"ttl":       modeFixed1,
	"pttl":      modeFixed1,
	"persist":   modeFixed1,
	"type":      modeFixed1,
	"dump":      modeFixed1,
	"restore":   modeFixed1,

	// string
	"get":         modeFixed1,
	"set":         modeFixed1,
	"setnx":       modeFixed1,
	"setex":       modeFixed1,
	"psetex":      modeFixed1,
	"getset":      modeFixed1,
	"append":      modeFixed1,
	"strlen":      modeFixed1,
	"incr":        modeFixed1,
	"incrby":      modeFixed1,
	"incrbyfloat": modeFixed1,
	"decr":        modeFixed1,
	"decrby":      modeFixed1,
	"setbit":      modeFixed1,
	"mget":        modeAllFrom1,
	"mset":        modeEvery2From1,
	"msetnx":      modeEvery2From1,
	"getdel":      modeFixed1,
	"getex":       modeFixed1,
	"getrange":    modeFixed1,
	"setrange":    modeFixed1,
	"bitcount":    modeFixed1,
	"bitpos":      modeFixed1,
	"getbit":      modeFixed1,

	// common multi key
	"del":    modeAllFrom1,
	"unlink": modeAllFrom1,
	"exists": modeAllFrom1,
	"touch":  modeAllFrom1,
	"watch":  modeAllFrom1,

	// hash
	"hget":         modeFixed1,
	"hset":         modeFixed1,
	"hsetnx":       modeFixed1,
	"hdel":         modeFixed1,
	"hexists":      modeFixed1,
	"hlen":         modeFixed1,
	"hstrlen":      modeFixed1,
	"hgetall":      modeFixed1,
	"hkeys":        modeFixed1,
	"hvals":        modeFixed1,
	"hmget":        modeFixed1,
	"hmset":        modeFixed1,
	"hincrby":      modeFixed1,
	"hincrbyfloat": modeFixed1,
	"hscan":        modeFixed1,

	// list
	"lpush":      modeFixed1,
	"rpush":      modeFixed1,
	"lpushx":     modeFixed1,
	"rpushx":     modeFixed1,
	"lpop":       modeFixed1,
	"rpop":       modeFixed1,
	"llen":       modeFixed1,
	"lindex":     modeFixed1,
	"lset":       modeFixed1,
	"ltrim":      modeFixed1,
	"lrange":     modeFixed1,
	"linsert":    modeFixed1,
	"lrem":       modeFixed1,
	"blpop":      modeBlockPop,
	"brpop":      modeBlockPop,
	"rpoplpush":  modeTwoKeys12,
	"brpoplpush": modeTwoKeys12,
	"lmove":      modeTwoKeys12,
	"blmove":     modeTwoKeys12,
	"lmpop":      modeNumKeysAt2,
	"blmpop":     modeNumKeysAt3,

	// set
	"sadd":        modeFixed1,
	"srem":        modeFixed1,
	"scard":       modeFixed1,
	"sismember":   modeFixed1,
	"smembers":    modeFixed1,
	"spop":        modeFixed1,
	"srandmember": modeFixed1,
	"sscan":       modeFixed1,
	"smove":       modeTwoKeys12,
	"rename":      modeTwoKeys12,
	"renamenx":    modeTwoKeys12,
	"sunion":      modeAllFrom1,
	"sinter":      modeAllFrom1,
	"sdiff":       modeAllFrom1,
	"sunionstore": modeStoreDestAllFrom2,
	"sinterstore": modeStoreDestAllFrom2,
	"sdiffstore":  modeStoreDestAllFrom2,
	"smismember":  modeFixed1,

	// zset
	"zadd":             modeFixed1,
	"zrem":             modeFixed1,
	"zcard":            modeFixed1,
	"zscore":           modeFixed1,
	"zrank":            modeFixed1,
	"zrevrank":         modeFixed1,
	"zrange":           modeFixed1,
	"zrevrange":        modeFixed1,
	"zrangebyscore":    modeFixed1,
	"zrevrangebyscore": modeFixed1,
	"zremrangebyrank":  modeFixed1,
	"zremrangebyscore": modeFixed1,
	"zincrby":          modeFixed1,
	"zcount":           modeFixed1,
	"zunionstore":      modeStoreNumKeys,
	"zinterstore":      modeStoreNumKeys,
	"zdiffstore":       modeStoreNumKeys,
	"bzpopmin":         modeBlockPop,
	"bzpopmax":         modeBlockPop,
	"zlexcount":        modeFixed1,
	"zscan":            modeFixed1,
	"zmscore":          modeFixed1,
	"zpopmin":          modeFixed1,
	"zpopmax":          modeFixed1,

	// hyperloglog
	"pfadd":   modeFixed1,
	"pfcount": modeAllFrom1,
	"pfmerge": modeAllFrom1,

	// stream / script
	"xread":      modeXRead,
	"xreadgroup": modeXReadGroup,
	"eval":       modeEval,
	"evalsha":    modeEval,
	"sort":       modeSort,
	"xlen":       modeFixed1,
	"xrange":     modeFixed1,
	"xrevrange":  modeFixed1,
	"xack":       modeFixed1,
	"xdel":       modeFixed1,
	"xtrim":      modeFixed1,
	"xgroup":     modeXGroup,
	"xinfo":      modeXInfo,

	// pubsub
	"publish":      modePublish,
	"subscribe":    modeSubscribe,
	"unsubscribe":  modeSubscribe,
	"psubscribe":   modePSubscribe,
	"punsubscribe": modePSubscribe,
	"ssubscribe":   modeSSubscribe,
	"sunsubscribe": modeSSubscribe,
	"spublish":     modePublish,
	"pubsub":       modePubSub,
}

func rewriteIndex(args []interface{}, idx int, fullPrefix string) {
	if idx <= 0 || idx >= len(args) {
		return
	}
	s, ok := args[idx].(string)
	if !ok || s == "" {
		return
	}
	args[idx] = applyPrefixFast(fullPrefix, s)
}

func rewriteRange(args []interface{}, start, end, step int, fullPrefix string) {
	if start > end || step <= 0 {
		return
	}
	if end >= len(args) {
		end = len(args) - 1
	}
	for i := start; i <= end; i += step {
		s, ok := args[i].(string)
		if !ok || s == "" {
			continue
		}
		args[i] = applyPrefixFast(fullPrefix, s)
	}
}

func rewriteXRead(args []interface{}, fullPrefix string) {
	streamsIdx := findTokenFast(args, 1, "streams")
	if streamsIdx == -1 || streamsIdx+1 >= len(args) {
		return
	}
	remain := len(args) - (streamsIdx + 1)
	keyCount := remain / 2
	if keyCount <= 0 {
		return
	}
	rewriteRange(args, streamsIdx+1, streamsIdx+keyCount, 1, fullPrefix)
}

func rewriteXReadGroup(args []interface{}, fullPrefix string) {
	streamsIdx := findTokenFast(args, 1, "streams")
	if streamsIdx == -1 || streamsIdx+1 >= len(args) {
		return
	}
	remain := len(args) - (streamsIdx + 1)
	keyCount := remain / 2
	if keyCount <= 0 {
		return
	}
	rewriteRange(args, streamsIdx+1, streamsIdx+keyCount, 1, fullPrefix)
}

func applyPrefixFast(fullPrefix, raw string) string {
	if raw == "" {
		return raw
	}
	if strings.HasPrefix(raw, fullPrefix) {
		return raw
	}
	return rewriteKey(fullPrefix, raw)
}

func findTokenFast(args []interface{}, start int, token string) int {
	for i := start; i < len(args); i++ {
		s, ok := args[i].(string)
		if !ok {
			continue
		}
		if s == token || strings.EqualFold(s, token) {
			return i
		}
	}
	return -1
}

func toIntFast(v interface{}) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	case string:
		n, err := strconv.Atoi(x)
		return n, err == nil
	default:
		return 0, false
	}
}
