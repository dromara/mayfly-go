package starter

import (
	"context"
	"fmt"
	"mayfly-go/internal/pkg/config"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/rediscli"

	"github.com/redis/go-redis/v9"
)

// 有配置redis信息，则初始化redis。多台机器部署需要使用redis存储验证码、权限、公私钥等信息
func initCache() {
	redisCli := connRedis()

	if redisCli == nil {
		logx.Info("no redis configuration exists, local cache is used")
		return
	}

	logx.Info("redis connection is successful, redis cache is used")
	rediscli.SetCli(connRedis())
	cache.SetCache(cache.NewRedisCache(redisCli))
}

func connRedis() *redis.Client {
	// 设置redis客户端
	redisConf := config.Conf.Redis
	if redisConf.Host == "" {
		// logx.Panic("未找到redis配置信息")
		return nil
	}
	logx.Infof("redis connecting [%s:%d]", redisConf.Host, redisConf.Port)
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisConf.Host, redisConf.Port),
		Password: redisConf.Password, // no password set
		DB:       redisConf.Db,       // use default DB
	})
	// 测试连接
	_, e := rdb.Ping(context.TODO()).Result()
	if e != nil {
		logx.Panicf("redis connection faild! [%s:%d][%s]", redisConf.Host, redisConf.Port, e.Error())
	}
	return rdb
}
