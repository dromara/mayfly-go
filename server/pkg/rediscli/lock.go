package rediscli

import (
	"context"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/stringx"
	"time"

	"github.com/redis/go-redis/v9"
)

const LockKeyPrefix = "mayfly:lock:"

// RedisLock redis实现的分布式锁
type RedisLock struct {
	key        string
	value      string // 唯一标识,一般使用uuid
	expiration time.Duration
}

func NewLock(key string, expiration time.Duration) *RedisLock {
	if key == "" || cli == nil {
		return nil
	}
	return &RedisLock{
		key:        key,
		value:      stringx.Rand(32),
		expiration: expiration,
	}
}

// Lock 添加分布式锁,expiration过期时间,小于等于0,不过期,需要通过 UnLock方法释放锁
func (rl *RedisLock) Lock() bool {
	result, err := cli.SetNX(context.Background(), LockKeyPrefix+rl.key, rl.value, rl.expiration).Result()
	if err != nil {
		logx.Errorf("redis lock setNx fail: %s", err.Error())
		return false
	}

	return result
}

// TryLock 加锁重试五次
func (rl *RedisLock) TryLock() bool {
	var locked bool
	for index := 0; index < 5; index++ {
		locked = rl.Lock()
		if locked {
			return locked
		}
		time.Sleep(50 * time.Millisecond)
	}

	return locked
}

func (rl *RedisLock) UnLock() bool {
	script := redis.NewScript(`
	if redis.call("get", KEYS[1]) == ARGV[1] then
		return redis.call("del", KEYS[1])
	else
		return 0
	end
	`)

	result, err := script.Run(context.Background(), cli, []string{LockKeyPrefix + rl.key}, rl.value).Int64()
	if err != nil {
		logx.Errorf("redis unlock runScript fail: %s", err.Error())
		return false
	}

	return result > 0
}

// RefreshLock 存在则更新过期时间,不存在则创建key
func (rl *RedisLock) RefreshLock() bool {
	script := redis.NewScript(`
	local val = redis.call("GET", KEYS[1])
	if not val then
		redis.call("setex", KEYS[1], ARGV[2], ARGV[1])
		return 2
	elseif val == ARGV[1] then
		return redis.call("expire", KEYS[1], ARGV[2])
	else
		return 0
	end
	`)

	result, err := script.Run(context.Background(), cli, []string{LockKeyPrefix + rl.key}, rl.value, rl.expiration/time.Second).Int64()
	if err != nil {
		logx.Errorf("redis refreshLock runScript fail: %s", err.Error())
		return false
	}

	return result > 0
}
