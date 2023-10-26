package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/common/consts"
	machineapp "mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/infrastructure/machine"
	"mayfly-go/internal/mongo/domain/entity"
	"mayfly-go/internal/mongo/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/netx"
	"mayfly-go/pkg/utils/structx"
	"net"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Mongo interface {
	base.App[*entity.Mongo]

	// 分页获取机器脚本信息列表
	GetPageList(condition *entity.MongoQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	Count(condition *entity.MongoQuery) int64

	Save(entity *entity.Mongo) error

	// 删除数据库信息
	Delete(id uint64) error

	// 获取mongo连接实例
	// @param id mongo id
	GetMongoInst(id uint64) *MongoInstance
}

func newMongoAppImpl(mongoRepo repository.Mongo) Mongo {
	return &mongoAppImpl{
		base.AppImpl[*entity.Mongo, repository.Mongo]{Repo: mongoRepo},
	}
}

type mongoAppImpl struct {
	base.AppImpl[*entity.Mongo, repository.Mongo]
}

// 分页获取数据库信息列表
func (d *mongoAppImpl) GetPageList(condition *entity.MongoQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return d.GetRepo().GetList(condition, pageParam, toEntity, orderBy...)
}

func (d *mongoAppImpl) Count(condition *entity.MongoQuery) int64 {
	return d.GetRepo().Count(condition)
}

func (d *mongoAppImpl) Delete(id uint64) error {
	DeleteMongoCache(id)
	return d.GetRepo().DeleteById(id)
}

func (d *mongoAppImpl) Save(m *entity.Mongo) error {
	if m.Id == 0 {
		return d.GetRepo().Insert(m)
	}

	// 先关闭连接
	DeleteMongoCache(m.Id)
	return d.GetRepo().UpdateById(m)
}

func (d *mongoAppImpl) GetMongoInst(id uint64) *MongoInstance {
	mongoInstance, err := GetMongoInstance(id, func(u uint64) (*entity.Mongo, error) {
		mongo, err := d.GetById(new(entity.Mongo), u)
		if err != nil {
			return nil, err
		}
		return mongo, nil
	})
	biz.ErrIsNilAppendErr(err, "连接mongo失败: %s")
	return mongoInstance
}

// -----------------------------------------------------------

// mongo客户端连接缓存，指定时间内没有访问则会被关闭
var mongoCliCache = cache.NewTimedCache(consts.MongoConnExpireTime, 5*time.Second).
	WithUpdateAccessTime(true).
	OnEvicted(func(key any, value any) {
		logx.Infof("删除mongo连接缓存: id = %v", key)
		value.(*MongoInstance).Close()
	})

func init() {
	machine.AddCheckSshTunnelMachineUseFunc(func(machineId int) bool {
		// 遍历所有mongo连接实例，若存在redis实例使用该ssh隧道机器，则返回true，表示还在使用中...
		items := mongoCliCache.Items()
		for _, v := range items {
			if v.Value.(*MongoInstance).Info.SshTunnelMachineId == machineId {
				return true
			}
		}
		return false
	})
}

// 获取mongo的连接实例
func GetMongoInstance(mongoId uint64, getMongoEntity func(uint64) (*entity.Mongo, error)) (*MongoInstance, error) {
	mi, err := mongoCliCache.ComputeIfAbsent(mongoId, func(_ any) (any, error) {
		mongoEntity, err := getMongoEntity(mongoId)
		if err != nil {
			return nil, err
		}

		c, err := connect(mongoEntity)
		if err != nil {
			return nil, err
		}
		return c, nil
	})

	if mi != nil {
		return mi.(*MongoInstance), err
	}
	return nil, err
}

func DeleteMongoCache(mongoId uint64) {
	mongoCliCache.Delete(mongoId)
}

type MongoInfo struct {
	Id                 uint64 `json:"id"`
	Name               string `json:"name"`
	TagPath            string `json:"tagPath"`
	SshTunnelMachineId int    `json:"-"` // ssh隧道机器id
}

func (m *MongoInfo) GetLogDesc() string {
	return fmt.Sprintf("Mongo[id=%d, tag=%s, name=%s]", m.Id, m.TagPath, m.Name)
}

type MongoInstance struct {
	Id   uint64
	Info *MongoInfo

	Cli *mongo.Client
}

func (mi *MongoInstance) Close() {
	if mi.Cli != nil {
		if err := mi.Cli.Disconnect(context.Background()); err != nil {
			logx.Errorf("关闭mongo实例[%d]连接失败: %s", mi.Id, err)
		}
		mi.Cli = nil
	}
}

// 连接mongo，并返回client
func connect(me *entity.Mongo) (*MongoInstance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoInstance := &MongoInstance{Id: me.Id, Info: toMongoInfo(me)}

	mongoOptions := options.Client().ApplyURI(me.Uri).
		SetMaxPoolSize(1)
	// 启用ssh隧道则连接隧道机器
	if me.SshTunnelMachineId > 0 {
		mongoOptions.SetDialer(&MongoSshDialer{machineId: me.SshTunnelMachineId})
	}

	client, err := mongo.Connect(ctx, mongoOptions)
	if err != nil {
		mongoInstance.Close()
		return nil, err
	}
	if err = client.Ping(context.TODO(), nil); err != nil {
		mongoInstance.Close()
		return nil, err
	}

	logx.Infof("连接mongo: %s", func(str string) string {
		reg := regexp.MustCompile(`(^mongodb://.+?:)(.+)(@.+$)`)
		return reg.ReplaceAllString(str, `${1}****${3}`)
	}(me.Uri))
	mongoInstance.Cli = client
	return mongoInstance, err
}

func toMongoInfo(me *entity.Mongo) *MongoInfo {
	mi := new(MongoInfo)
	structx.Copy(mi, me)
	return mi
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
