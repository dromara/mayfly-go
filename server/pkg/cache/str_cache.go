package cache

import (
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/rediscli"
	"strconv"
	"time"
)

var tm *TimedCache

// 如果系统有设置redis信息，则从redis获取，否则本机内存获取
func GetStr(key string) string {
	if !useRedisCache() {
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

// 如果系统有设置redis信息，则使用redis存，否则存于本机内存。duration == -1则为永久缓存
func SetStr(key, value string, duration time.Duration) {
	if !useRedisCache() {
		checkCache()
		tm.Add(key, value, duration)
		return
	}
	biz.ErrIsNilAppendErr(rediscli.Set(key, value, duration), "redis set err: %s")
}

// 删除指定key
func Del(key string) {
	if !useRedisCache() {
		checkCache()
		tm.Delete(key)
		return
	}
	rediscli.Del(key)
}

func checkCache() {
	if tm == nil {
		tm = NewTimedCache(time.Minute*time.Duration(5), 30*time.Second)
	}
}

func useRedisCache() bool {
	return rediscli.GetCli() != nil
}
