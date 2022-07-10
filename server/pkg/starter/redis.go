package starter

import (
	"context"
	"fmt"
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/global"

	"github.com/go-redis/redis/v8"
)

func InitRedis() {
	global.RedisCli = ConnRedis()
}

func ConnRedis() *redis.Client {
	// 设置redis客户端
	redisConf := config.Conf.Redis
	if redisConf == nil {
		global.Log.Panic("未找到redis配置信息")
		return nil
	}
	global.Log.Infof("连接redis [%s:%d]", redisConf.Host, redisConf.Port)
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisConf.Host, redisConf.Port),
		Password: redisConf.Password, // no password set
		DB:       redisConf.Db,       // use default DB
	})
	// 测试连接
	_, e := rdb.Ping(context.TODO()).Result()
	if e != nil {
		global.Log.Panic(fmt.Sprintf("连接redis失败! [%s:%d]", redisConf.Host, redisConf.Port))
	}
	return rdb
}
