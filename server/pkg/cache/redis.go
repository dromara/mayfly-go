package cache

import (
	"context"
	"errors"
	"mayfly-go/pkg/utils/anyx"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	defaultCache

	redisCli *redis.Client
}

var _ (Cache) = (*RedisCache)(nil)

func NewRedisCache(rc *redis.Client) *RedisCache {
	rcache := &RedisCache{
		redisCli: rc,
	}
	rcache.c = rcache
	return rcache
}

var _ (Cache) = (*RedisCache)(nil)

func (rc *RedisCache) Set(key string, value any, duration time.Duration) error {
	if _, ok := value.(string); !ok {
		return errors.New("redis cache set err -> value must be string")
	}
	if duration < 0 {
		duration = 0
	}
	return rc.redisCli.Set(context.Background(), key, anyx.ToString(value), duration).Err()
}

func (rc *RedisCache) Get(k string) (any, bool) {
	if val, err := rc.redisCli.Get(context.Background(), k).Result(); err != nil {
		return "", false
	} else {
		return val, true
	}
}

// Incr 使用 Redis INCR 命令原子自增，天然支持多实例并发安全
func (rc *RedisCache) Incr(key string) (int64, error) {
	return rc.redisCli.Incr(context.Background(), key).Result()
}

// IncrWithTTL 自增后刷新过期时间。
// INCR 创建的是永不过期的 key，若不在自增时设置 TTL，
// 业务侧删除计数器 key 的逻辑一旦遗漏（规则被删除、进程重启等）就会永久泄漏。
// ttl <= 0 表示不设置过期时间，保持与 Incr 一致。
func (rc *RedisCache) IncrWithTTL(key string, ttl time.Duration) (int64, error) {
	ctx := context.Background()
	val, err := rc.redisCli.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if ttl > 0 {
		if err = rc.redisCli.Expire(ctx, key, ttl).Err(); err != nil {
			return val, err
		}
	}
	return val, nil
}

func (rc *RedisCache) Delete(k string) error {
	return rc.redisCli.Del(context.Background(), k).Err()
}

func (rc *RedisCache) DeleteByKeyPrefix(keyPrefix string) error {
	res, err := rc.redisCli.Keys(context.TODO(), keyPrefix+"*").Result()
	if err != nil {
		return err
	}
	for _, key := range res {
		Del(key)
	}
	return nil
}

func (rc *RedisCache) Count() int {
	return int(rc.redisCli.DBSize(context.Background()).Val())
}

func (rc *RedisCache) Clear() {
	rc.redisCli.FlushDB(context.Background())
}
