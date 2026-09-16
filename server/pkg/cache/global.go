package cache

import (
	"mayfly-go/pkg/rediscli"
	"time"
)

var c Cache = NewLocalCache()

// SetCache 设置全局缓存实现
func SetCache(cache Cache) {
	c = cache
}

func GetStr(key string) string {
	if val, ok := c.GetStr(key); ok {
		return val
	}
	return ""
}

func GetInt(key string) int {
	if val, ok := c.GetInt(key); ok {
		return val
	}
	return 0
}

// Get 获取缓存值，并使用json反序列化。返回是否获取成功。若不存在或者解析失败，则返回false
func Get[T any](key string, valPtr T) bool {
	return c.GetJson(key, valPtr)
}

// Set 设置缓存值
func Set(key string, value any, duration time.Duration) error {
	return c.Set2Str(key, value, duration)
}

// 删除指定key
func Del(key string) {
	c.Delete(key)
}

// Incr 原子自增缓存中的整数值并返回新值。
// 若 key 不存在则初始化为 0 后自增（返回 1）。
// 底层使用 Redis INCR 或本地 mutex，保证多实例并发安全。
func Incr(key string) (int64, error) {
	return c.Incr(key)
}

// IncrWithTTL 原子自增并刷新 key 的过期时间，用于需要自动回收的计数器缓存
func IncrWithTTL(key string, ttl time.Duration) (int64, error) {
	return c.IncrWithTTL(key, ttl)
}

func UseRedisCache() bool {
	return rediscli.GetCli() != nil
}
