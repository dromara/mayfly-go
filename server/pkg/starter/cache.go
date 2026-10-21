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
		// 未配置 Redis 时降级为进程内缓存：单实例可用，但以下能力**不再跨实例共享也不再重启后存活**：
		// 登录验证码、账号权限码缓存、系统公私钥、机器指标缓存（机器告警评估只读这份缓存）、
		// 以及各处用缓存做的并发去重/分布式锁语义。
		// 因此这里必须是 Warn 而不是 Info：本函数自己的注释就写明多实例部署依赖 Redis，
		// 而一个只会被当成「启动成功」的 INFO 等于把这个正确性前提完全隐藏；
		// 也不做成启动失败：本地/单实例开发不配 Redis 是合法形态，不能只因未配就直接拒起
		logx.Warnf("no redis configuration: falling back to in-process cache; " +
			"captcha / permission-code cache / system keypairs / machine stats are NOT shared across instances nor kept across restarts; configure redis for multi-instance deployment")
		return nil
	}

	if err != nil {
		return err
	}

	logx.Infof("redis connected successfully [%s:%d], using Redis for caching", redisConfig.Host, redisConfig.Port)
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
