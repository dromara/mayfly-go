package rdm

import (
	"context"
	"fmt"
	machineapp "mayfly-go/internal/machine/application"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/netx"
	"net"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisMode string

const (
	StandaloneMode RedisMode = "standalone"
	ClusterMode    RedisMode = "cluster"
	SentinelMode   RedisMode = "sentinel"
)

type RedisInfo struct {
	Id uint64 `json:"id"`

	Host     string    `json:"host"`
	Db       int       `json:"db"` // 库号
	Mode     RedisMode `json:"-"`
	Username string    `json:"-"`
	Password string    `json:"-"`

	Name               string   `json:"-"`
	TagPath            []string `json:"tagPath"`
	SshTunnelMachineId int      `json:"-"`
}

func (r *RedisInfo) Conn() (*RedisConn, error) {
	redisMode := r.Mode
	if redisMode == StandaloneMode {
		return r.connStandalone()
	}
	if redisMode == ClusterMode {
		return r.connCluster()
	}
	if redisMode == SentinelMode {
		return r.connSentinel()
	}

	return nil, errorx.NewBiz("redis mode error")
}

func (re *RedisInfo) connStandalone() (*RedisConn, error) {
	redisOptions := &redis.Options{
		Addr:         re.Host,
		Username:     re.Username,
		Password:     re.Password, // no password set
		DB:           re.Db,       // use default DB
		DialTimeout:  8 * time.Second,
		ReadTimeout:  -1, // Disable timeouts, because SSH does not support deadlines.
		WriteTimeout: -1,
	}
	if re.SshTunnelMachineId > 0 {
		redisOptions.Dialer = getRedisDialer(re.SshTunnelMachineId)
	}

	cli := redis.NewClient(redisOptions)
	_, e := cli.Ping(context.Background()).Result()
	if e != nil {
		cli.Close()
		return nil, errorx.NewBiz("redis连接失败: %s", e.Error())
	}

	logx.Infof("连接redis standalone: %s/%d", re.Host, re.Db)

	rc := &RedisConn{Id: getConnId(re.Id, re.Db), Info: re}
	rc.Cli = cli
	return rc, nil
}

func (re *RedisInfo) connCluster() (*RedisConn, error) {
	redisClusterOptions := &redis.ClusterOptions{
		Addrs:       strings.Split(re.Host, ","),
		Username:    re.Username,
		Password:    re.Password,
		DialTimeout: 8 * time.Second,
	}
	if re.SshTunnelMachineId > 0 {
		redisClusterOptions.Dialer = getRedisDialer(re.SshTunnelMachineId)
	}
	cli := redis.NewClusterClient(redisClusterOptions)
	// 测试连接
	_, e := cli.Ping(context.Background()).Result()
	if e != nil {
		cli.Close()
		return nil, errorx.NewBiz("redis集群连接失败: %s", e.Error())
	}

	logx.Infof("连接redis cluster: %s/%d", re.Host, re.Db)

	rc := &RedisConn{Id: getConnId(re.Id, re.Db), Info: re}
	rc.ClusterCli = cli
	return rc, nil
}

func (re *RedisInfo) connSentinel() (*RedisConn, error) {
	// sentinel模式host为 masterName=host:port,host:port
	masterNameAndHosts := strings.Split(re.Host, "=")
	sentinelOptions := &redis.FailoverOptions{
		MasterName:       masterNameAndHosts[0],
		SentinelAddrs:    strings.Split(masterNameAndHosts[1], ","),
		Username:         re.Username,
		Password:         re.Password, // no password set
		SentinelPassword: re.Password, // 哨兵节点密码需与redis节点密码一致
		DB:               re.Db,       // use default DB
		DialTimeout:      8 * time.Second,
		ReadTimeout:      -1, // Disable timeouts, because SSH does not support deadlines.
		WriteTimeout:     -1,
	}
	if re.SshTunnelMachineId > 0 {
		sentinelOptions.Dialer = getRedisDialer(re.SshTunnelMachineId)
	}
	cli := redis.NewFailoverClient(sentinelOptions)

	_, e := cli.Ping(context.Background()).Result()
	if e != nil {
		cli.Close()
		return nil, errorx.NewBiz("redis sentinel连接失败: %s", e.Error())
	}

	logx.Infof("连接redis sentinel: %s/%d", re.Host, re.Db)

	rc := &RedisConn{Id: getConnId(re.Id, re.Db), Info: re}
	rc.Cli = cli
	return rc, nil
}

func getRedisDialer(machineId int) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(_ context.Context, network, addr string) (net.Conn, error) {
		sshTunnel, err := machineapp.GetMachineApp().GetSshTunnelMachine(machineId)
		if err != nil {
			return nil, err
		}

		if sshConn, err := sshTunnel.GetDialConn(network, addr); err == nil {
			// 将ssh conn包装，否则redis内部设置超时会报错,ssh conn不支持设置超时会返回错误: ssh: tcpChan: deadline not supported
			return &netx.WrapSshConn{Conn: sshConn}, nil
		} else {
			return nil, err
		}
	}
}

// 生成redis连接id
func getConnId(id uint64, db int) string {
	if id == 0 {
		return ""
	}
	return fmt.Sprintf("%d/%d", id, db)
}
