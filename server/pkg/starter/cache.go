package starter

import (
	"context"
	"fmt"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/rediscli"

	"github.com/redis/go-redis/v9"
)

// 有配置redis信息，则初始化redis。多台机器部署需要使用redis存储验证码、权限、公私钥等信息
func initCache(redisConfig RedisConf) error {
	redisCli, err := connRedis(redisConfig)

	if redisCli == nil && err == nil {
		logx.Info("no redis configuration, using local cache")
		return nil
	}

	if err != nil {
		return err
	}

	logx.Info("redis connected successfully, using Redis for caching")
	rediscli.SetCli(redisCli)
	cache.SetCache(cache.NewRedisCache(redisCli))
	return nil
}

func connRedis(redisConfig RedisConf) (*redis.Client, error) {
	// 设置redis客户端
	if redisConfig.Host == "" {
		return nil, nil
	}
	logx.Infof("redis connecting [%s:%d]", redisConfig.Host, redisConfig.Port)
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisConfig.Host, redisConfig.Port),
		Password: redisConfig.Password, // no password set
		DB:       redisConfig.Db,       // use default DB
	})
	// 测试连接
	_, e := rdb.Ping(context.TODO()).Result()
	if e != nil {
		logx.Errorf("redis connection failed! [%s:%d][%s]", redisConfig.Host, redisConfig.Port, e.Error())
	}
	return rdb, e
}
