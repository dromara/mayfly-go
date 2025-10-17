package mgm

import (
	"context"
	"fmt"
	"mayfly-go/pkg/logx"

	"go.mongodb.org/mongo-driver/v2/mongo"
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
	// 首先检查mc是否为nil
	if mc == nil {
		return fmt.Errorf("mc connection is nil")
	}

	// 然后检查mc.Cli是否为nil，这是避免空指针异常的关键
	if mc.Cli == nil {
		return fmt.Errorf("mc client is nil")
	}
	return mc.Cli.Ping(context.Background(), nil)
}
