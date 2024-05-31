package cache

import (
	"encoding/json"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/rediscli"
	"mayfly-go/pkg/utils/anyx"
	"strconv"
	"strings"
	"time"

	"github.com/may-fly/cast"
)

var tm *TimedCache

// 如果系统有设置redis信息，则从redis获取，否则本机内存获取
func GetStr(key string) string {
	if !UseRedisCache() {
		checkCache()
		val, _ := tm.Get(key)
		if val == nil {
			return ""
		}
		return val.(string)
	}

	if res, err := rediscli.Get(key); err == nil {
		return res
	}
	return ""
}

func GetInt(key string) int {
	val := GetStr(key)
	if val == "" {
		return 0
	}
	if intV, err := strconv.Atoi(val); err != nil {
		logx.Error("获取缓存中的int值转换失败", err)
		return 0
	} else {
		return intV
	}
}

// Get 获取缓存值，并使用json反序列化。返回是否获取成功。若不存在或者解析失败，则返回false
func Get[T any](key string, valPtr T) bool {
	strVal := GetStr(key)
	if strVal == "" {
		return false
	}
	if err := json.Unmarshal([]byte(strVal), valPtr); err != nil {
		logx.Errorf("json转换缓存中的值失败: %v", err)
		return false
	}
	return true
}

// SetStr 如果系统有设置redis信息，则使用redis存，否则存于本机内存。duration == -1则为永久缓存
func SetStr(key, value string, duration time.Duration) error {
	if !UseRedisCache() {
		checkCache()
		return tm.Add(key, value, duration)
	}
	return rediscli.Set(key, value, duration)
}

// Set 如果系统有设置redis信息，则使用redis存，否则存于本机内存。duration == -1则为永久缓存
func Set(key string, value any, duration time.Duration) error {
	strVal := anyx.ToString(value)
	if !UseRedisCache() {
		checkCache()
		return tm.Add(key, strVal, duration)
	}
	return rediscli.Set(key, strVal, duration)
}

// 删除指定key
func Del(key string) {
	if !UseRedisCache() {
		checkCache()
		tm.Delete(key)
		return
	}
	rediscli.Del(key)
}

// DelByKeyPrefix 根据key前缀删除满足前缀的所有缓存
func DelByKeyPrefix(keyPrefix string) error {
	if !UseRedisCache() {
		checkCache()
		for key := range tm.Items() {
			if strings.HasPrefix(cast.ToString(key), keyPrefix) {
				tm.Delete(key)
			}
		}
		return nil
	}
	return rediscli.DelByKeyPrefix(keyPrefix)
}

func UseRedisCache() bool {
	return rediscli.GetCli() != nil
}

func checkCache() {
	if tm == nil {
		tm = NewTimedCache(time.Minute*time.Duration(5), 30*time.Second)
	}
}
