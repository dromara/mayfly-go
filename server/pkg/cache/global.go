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

func UseRedisCache() bool {
	return rediscli.GetCli() != nil
}
