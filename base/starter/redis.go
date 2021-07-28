package starter

import (
	"fmt"
	"mayfly-go/base/config"
	"mayfly-go/base/global"

	"github.com/go-redis/redis"
)

func ConnRedis() *redis.Client {
	// 设置redis客户端
	redisConf := config.Conf.Redis
	if redisConf == nil {
		global.Log.Panic("未找到redis配置信息")
	}
	global.Log.Infof("连接redis [%s:%d]", redisConf.Host, redisConf.Port)
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisConf.Host, redisConf.Port),
		Password: redisConf.Password, // no password set
		DB:       redisConf.Db,       // use default DB
	})
	// 测试连接
	_, e := rdb.Ping().Result()
	if e != nil {
		global.Log.Panic(fmt.Sprintf("连接redis失败! [%s:%d]", redisConf.Host, redisConf.Port))
	}
	return rdb
}
