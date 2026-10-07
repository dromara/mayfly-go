package mgm

import (
	"context"
	"errors"
	"fmt"
	"mayfly-go/pkg/logx"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// CodeUnauthorized 服务端错误码 13：命令需要认证或当前账号权限不足。
const CodeUnauthorized = 13

// ErrNoCredentials 连接串没配任何凭证，而服务器要求认证。
//
// 单独成一条错误而不是把 Unauthorized 原样抛出去：这个组合的成因永远是「登记实例时漏了账号密码
// 或没写 authSource」，与「账号权限不够」是两种修法，混在一起用户无从判断该改哪边。
var ErrNoCredentials = errors.New("mongo: the server requires authentication but the connection uri carries no credentials")

// IsUnauthorized 报告该错误是否为服务端的认证/权限拒绝。
func IsUnauthorized(err error) bool {
	var cmdErr mongo.CommandError
	return errors.As(err, &cmdErr) && cmdErr.Code == CodeUnauthorized
}

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

// ProbeAuthorized 探测「匿名能连上、但服务器要求认证」的配置缺陷。
//
// 只在连接串没写凭证时调用：配了凭证的账号可能只有某个库的读写权限，拿需要权限的命令去判它
// 会把正常连接误报成失败。而没凭证时 ping 必然通过（MongoDB 允许匿名 ping），不探测就会出现
// 「测试连接成功、一打开列表就 Unauthorized」这种更难排查的体验。
func (mc *MongoConn) ProbeAuthorized(ctx context.Context) error {
	err := mc.Cli.Database("admin").RunCommand(ctx, bson.D{{Key: "listDatabases", Value: int32(1)}}).Err()
	if err == nil {
		// 服务器允许匿名访问，连接串不配凭证是合理配置
		return nil
	}
	if IsUnauthorized(err) {
		return fmt.Errorf("%w: %s", ErrNoCredentials, err.Error())
	}
	// 网络、超时一类的真实故障原样上抛
	return err
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
