package rdm

import (
	"context"
	"mayfly-go/pkg/logx"

	"github.com/redis/go-redis/v9"
)

// redis连接信息
type RedisConn struct {
	Id   string
	Info *RedisInfo

	Cli        *redis.Client
	ClusterCli *redis.ClusterClient
}

// 获取命令执行接口的具体实现
func (r *RedisConn) GetCmdable() redis.Cmdable {
	redisMode := r.Info.Mode
	if redisMode == "" || redisMode == StandaloneMode || r.Info.Mode == SentinelMode {
		return r.Cli
	}
	if redisMode == ClusterMode {
		return r.ClusterCli
	}
	return nil
}

func (r *RedisConn) Scan(cursor uint64, match string, count int64) ([]string, uint64, error) {
	return r.GetCmdable().Scan(context.Background(), cursor, match, count).Result()
}

func (r *RedisConn) Close() {
	mode := r.Info.Mode
	if mode == StandaloneMode || mode == SentinelMode {
		if err := r.Cli.Close(); err != nil {
			logx.Errorf("关闭redis单机实例[%s]连接失败: %s", r.Id, err.Error())
		}
		r.Cli = nil
		return
	}

	if mode == ClusterMode {
		if err := r.ClusterCli.Close(); err != nil {
			logx.Errorf("关闭redis集群实例[%s]连接失败: %s", r.Id, err.Error())
		}
		r.ClusterCli = nil
	}
}
