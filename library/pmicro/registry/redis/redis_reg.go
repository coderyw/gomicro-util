package redis

import (
	"context"
	"time"

	"github.com/coderyw/gomicro-util/library/db/cache/redis"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/model/consistent_hash"
	cron2 "github.com/coderyw/gomicro-util/model/cron"
	"go.uber.org/zap"
)

const _Redis_Reg_Key = "server_reg:zset"

type RedisReg struct {
	id string
}

func NewRedisReg() (cron2.CronJob, func()) {
	return cron2.CronJob{
			Name: "redis_reg",
			DoJob: func() {

			},
			CronSpc:         "@every 3s",
			DoBeforeDone:    false,
			DelayBeforeDone: false,
		}, func() {

		}
}

func (r *RedisReg) init(id string) {
	r.id = id
	err := redis.GetRedis(context.Background()).ZAdd(context.Background(), "", _Redis_Reg_Key, float64(time.Now().Unix()), id)
	if err != nil {
		log.Warn("Server 注册Redis错误", zap.String("id", id), zap.Error(err))
	}
}

// 停止注册 服务
func (r *RedisReg) Close() {
	redis.GetRedis(context.Background()).ZRem(context.Background(), "", _Redis_Reg_Key, r.id)
}

func (r *RedisReg) OnChanged() chan consistent_hash.Notice {

	//TODO implement me
	panic("implement me")
}
