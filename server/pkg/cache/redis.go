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
