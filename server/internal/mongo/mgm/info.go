package mgm

import (
	"context"
	"fmt"
	"mayfly-go/internal/mongo/config"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/netx"
	"net"
	"time"

	machineapp "mayfly-go/internal/machine/application"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoInfo struct {
	Id   uint64 `json:"id"`
	Name string `json:"name"`

	Uri string `json:"-"`

	CodePath           []string `json:"codePath"`
	SshTunnelMachineId int      `json:"-"` // ssh隧道机器id
}

func (mi *MongoInfo) Conn() (*MongoConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	mongoOptions := options.Client().ApplyURI(mi.Uri)
	// URI 里显式给了 maxPoolSize 时以 URI 为准：连接串是使用者对连接的调优入口，
	// 平台配置只是默认值，不能反过来覆盖掉显式设置。
	//
	// 默认值不能再是旧的 1：driver 会为每个并发操作检出一条连接，上限 1 等于把
	// 同一实例上的所有操作串行（多个 tab、多个用户都在排队）。
	if mongoOptions.MaxPoolSize == nil {
		mongoOptions.SetMaxPoolSize(uint64(config.GetMongo().PoolSize))
	}
	// 启用ssh隧道则连接隧道机器
	if mi.SshTunnelMachineId > 0 {
		mongoOptions.SetDialer(&MongoSshDialer{machineId: mi.SshTunnelMachineId})
	}

	client, err := mongo.Connect(mongoOptions)
	if err != nil {
		return nil, err
	}
	if err = client.Ping(ctx, nil); err != nil {
		client.Disconnect(ctx)
		return nil, err
	}

	logx.Infof("连接mongo: %s", MaskUri(mi.Uri))

	return &MongoConn{Id: getConnId(mi.Id), Info: mi, Cli: client}, nil
}

type MongoSshDialer struct {
	machineId int
}

func (sd *MongoSshDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	stm, err := machineapp.GetMachineApp().GetSshTunnelMachine(ctx, sd.machineId)
	if err != nil {
		return nil, err
	}
	if sshConn, err := stm.GetDialConn(network, address); err == nil {
		// 将ssh conn包装，否则内部部设置超时会报错,ssh conn不支持设置超时会返回错误: ssh: tcpChan: deadline not supported
		return &netx.WrapSshConn{Conn: sshConn}, nil
	} else {
		return nil, err
	}
}

// 生成mongo连接id
func getConnId(id uint64) string {
	if id == 0 {
		return ""
	}
	return fmt.Sprintf("mongo:%d", id)
}
