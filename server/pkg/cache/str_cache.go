package cache

import "mayfly-go/pkg/rediscli"

var strCache map[string]string

// 如果系统有设置redis信息，则从redis获取，否则本机内存获取
func GetStr(key string) string {
	if rediscli.GetCli() == nil {
		checkStrCache()
		return strCache[key]
	}
	res, err := rediscli.Get(key)
	if err != nil {
		return ""
	}
	return res
}

// 如果系统有设置redis信息，则使用redis存，否则存于本机内存
func SetStr(key, value string) {
	if rediscli.GetCli() == nil {
		checkStrCache()
		strCache[key] = value
		return
	}
	rediscli.Set(key, value, 0)
}

// 删除指定key
func Del(key string) {
	if rediscli.GetCli() == nil {
		checkStrCache()
		delete(strCache, key)
		return
	}
	rediscli.Del(key)
}

func checkStrCache() {
	if strCache == nil {
		strCache = make(map[string]string)
	}
}
