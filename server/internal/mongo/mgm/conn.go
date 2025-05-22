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

/******************* pool.Conn impl *******************/

func (mc *MongoConn) Close() error {
	if mc.Cli != nil {
		if err := mc.Cli.Disconnect(context.Background()); err != nil {
			logx.Errorf("关闭mongo实例[%s]连接失败: %s", mc.Id, err)
			return err
		}
		mc.Cli = nil
	}
	return nil
}

func (mc *MongoConn) Ping() error {
	return mc.Cli.Ping(context.Background(), nil)
}
