package model

import (
	"github.com/coderyw/gomicro-util/library/db/cache/redis/prefixenum"
	"time"
)

type LockParam struct {
	Enum prefixenum.PrefixEnum
	K    string
	// 等待超时时间 对于LockWaitTimeOutWithRenewal方法，默认为 5s
	WaitTimeOut time.Duration
	// 重试获取锁的间隔，默认为 100ms
	RetryInterval time.Duration
	// 锁的过期时间，默认为 5s
	Expire time.Duration
	// 续约时间间隔，默认为 3s
	RenewalInterval time.Duration
}
