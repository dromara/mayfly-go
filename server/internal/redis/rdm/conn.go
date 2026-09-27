package rdm

import (
	"context"
	"fmt"
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
	if r.Cli == nil {
		return fmt.Errorf("redis client is nil")
	}

	stats := r.Cli.PoolStats()
	logx.Debugf("[%s] redis stats -> open: %d, idle: %d", r.Info.Name, stats.TotalConns, stats.IdleConns)
	if stats.TotalConns == 0 {
		logx.Infof("[%s] redis stats: no open connections", r.Info.Name)
	}

	cmd := r.Cli.Ping(context.Background())
	if cmd == nil {
		return fmt.Errorf("the ping cmd is nil")
	}
	_, err := cmd.Result()
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

// Pipelined 以管道方式批量执行命令，用于 key 列表的元信息读取等「一次要看很多 key」的场景。
// cluster 模式下由客户端按槽自动拆分到各节点，单条命令的失败由调用方按命令对象自行判断
func (r *RedisConn) Pipelined(ctx context.Context, fn func(redis.Pipeliner)) error {
	switch r.Info.Mode {
	case ClusterMode:
		if r.ClusterCli == nil {
			return errorx.NewBiz("redis cluster client is nil")
		}
		pipe := r.ClusterCli.Pipeline()
		fn(pipe)
		_, err := pipe.Exec(ctx)
		return err
	default:
		if r.Cli == nil {
			return errorx.NewBiz("redis client is nil")
		}
		pipe := r.Cli.Pipeline()
		fn(pipe)
		_, err := pipe.Exec(ctx)
		return err
	}
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
