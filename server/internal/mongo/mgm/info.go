package mgm

import (
	"context"
	"fmt"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/netx"
	"net"
	"regexp"
	"time"

	machineapp "mayfly-go/internal/machine/application"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoInfo struct {
	Id   uint64 `json:"id"`
	Name string `json:"name"`

	Uri string `json:"-"`

	TagPath            []string `json:"tagPath"`
	SshTunnelMachineId int      `json:"-"` // ssh隧道机器id
}

func (mi *MongoInfo) Conn() (*MongoConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	mongoOptions := options.Client().ApplyURI(mi.Uri).
		SetMaxPoolSize(1)
	// 启用ssh隧道则连接隧道机器
	if mi.SshTunnelMachineId > 0 {
		mongoOptions.SetDialer(&MongoSshDialer{machineId: mi.SshTunnelMachineId})
	}

	client, err := mongo.Connect(ctx, mongoOptions)
	if err != nil {
		return nil, err
	}
	if err = client.Ping(ctx, nil); err != nil {
		client.Disconnect(ctx)
		return nil, err
	}

	logx.Infof("连接mongo: %s", func(str string) string {
		reg := regexp.MustCompile(`(^mongodb://.+?:)(.+)(@.+$)`)
		return reg.ReplaceAllString(str, `${1}****${3}`)
	}(mi.Uri))

	return &MongoConn{Id: getConnId(mi.Id), Info: mi, Cli: client}, nil
}

type MongoSshDialer struct {
	machineId int
}

func (sd *MongoSshDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	stm, err := machineapp.GetMachineApp().GetSshTunnelMachine(sd.machineId)
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
	return fmt.Sprintf("%d", id)
}
