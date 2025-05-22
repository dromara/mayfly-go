package rdm

import (
	"context"
	"mayfly-go/pkg/errorx"
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

/******************* pool.Conn impl *******************/

func (r *RedisConn) Close() error {
	mode := r.Info.Mode
	if mode == StandaloneMode || mode == SentinelMode {
		if err := r.Cli.Close(); err != nil {
			logx.Errorf("close redis standalone instance [%s] connection failed: %s", r.Id, err.Error())
			return err
		}
		r.Cli = nil
		return nil
	}

	if mode == ClusterMode {
		if err := r.ClusterCli.Close(); err != nil {
			logx.Errorf("close redis cluster instance [%s] connection failed: %s", r.Id, err.Error())
			return err
		}
		r.ClusterCli = nil
	}
	return nil
}

func (r *RedisConn) Ping() error {
	_, err := r.Cli.Ping(context.Background()).Result()
	return err
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

// 执行redis命令
// 如: SET str value命令则args为['SET', 'str', 'val']
func (r *RedisConn) RunCmd(ctx context.Context, args ...any) (any, error) {
	redisMode := r.Info.Mode
	if redisMode == "" || redisMode == StandaloneMode || r.Info.Mode == SentinelMode {
		return r.Cli.Do(ctx, args...).Result()
	}
	if redisMode == ClusterMode {
		return r.ClusterCli.Do(ctx, args...).Result()
	}
	return nil, errorx.NewBiz("redis mode error")
}
