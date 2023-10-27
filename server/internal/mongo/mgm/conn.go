package mgm

import (
	"context"
	"mayfly-go/pkg/logx"

	"go.mongodb.org/mongo-driver/mongo"
)

type MongoConn struct {
	Id   string
	Info *MongoInfo

	Cli *mongo.Client
}

func (mc *MongoConn) Close() {
	if mc.Cli != nil {
		if err := mc.Cli.Disconnect(context.Background()); err != nil {
			logx.Errorf("关闭mongo实例[%s]连接失败: %s", mc.Id, err)
		}
		mc.Cli = nil
	}
}
