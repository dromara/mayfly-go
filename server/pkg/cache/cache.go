package cache

import (
	"encoding/json"
	"mayfly-go/pkg/utils/anyx"
	"time"

	"github.com/may-fly/cast"
)

type Cache interface {
	// Set 设置缓存值
	// duration == -1 则为永久缓存
	Set(key string, value any, duration time.Duration) error

	// Set2Str 将value转为字符串后设置缓存值
	Set2Str(key string, value any, duration time.Duration) error

	// Get 获取缓存值
	Get(k string) (any, bool)

	// GetJson 获取json缓存值，并将其解析到指定的结构体指针
	GetJson(k string, valPtr any) bool

	// GetStr  获取字符串缓存值
	GetStr(k string) (string, bool)

	// GetInt  获取int缓存值
	GetInt(k string) (int, bool)

	// Delete 删除缓存
	Delete(key string) error

	// DeleteByKeyPrefix 根据key前缀删除缓存
	DeleteByKeyPrefix(keyPrefix string) error

	// Count 缓存数量
	Count() int

	// Clear 清空所有缓存
	Clear()
}

type defaultCache struct {
	c Cache // 具体的缓存实现, 用于实现一些接口的通用方法
}

func (dc *defaultCache) Set2Str(key string, value any, duration time.Duration) error {
	return dc.c.Set(key, anyx.ToString(value), duration)
}

func (dc *defaultCache) GetStr(k string) (string, bool) {
	if val, ok := dc.c.Get(k); ok {
		return cast.ToString(val), true
	}
	return "", false
}

func (dc *defaultCache) GetInt(k string) (int, bool) {
	if val, ok := dc.c.Get(k); ok {
		return cast.ToInt(val), true
	}
	return 0, false
}

func (dc *defaultCache) GetJson(k string, valPtr any) bool {
	if val, ok := dc.GetStr(k); ok {
		json.Unmarshal([]byte(val), valPtr)
		return true
	}
	return false
}
