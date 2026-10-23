package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

func (r *Redis) XAdd(ctx context.Context, a *redis.XAddArgs) (string, error) {
	return r.UniversalClient.XAdd(ctx, a).Result()
}
