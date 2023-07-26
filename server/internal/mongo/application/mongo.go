package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/common/consts"
	machineapp "mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/infrastructure/machine"
	"mayfly-go/internal/mongo/domain/entity"
	"mayfly-go/internal/mongo/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/global"
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
	// 分页获取机器脚本信息列表
	GetPageList(condition *entity.MongoQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any]

	Count(condition *entity.MongoQuery) int64

	// 根据条件获取
	GetBy(condition *entity.Mongo, cols ...string) error

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.Mongo

	Save(entity *entity.Mongo)

	// 删除数据库信息
	Delete(id uint64)

	// 获取mongo连接client
	// @param id mongo id
	GetMongoCli(id uint64) *mongo.Client
}

func newMongoAppImpl(mongoRepo repository.Mongo) Mongo {
	return &mongoAppImpl{
		mongoRepo: mongoRepo,
	}
}

type mongoAppImpl struct {
	mongoRepo repository.Mongo
}

// 分页获取数据库信息列表
func (d *mongoAppImpl) GetPageList(condition *entity.MongoQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any] {
	return d.mongoRepo.GetList(condition, pageParam, toEntity, orderBy...)
}

func (d *mongoAppImpl) Count(condition *entity.MongoQuery) int64 {
	return d.mongoRepo.Count(condition)
}

// 根据条件获取
func (d *mongoAppImpl) GetBy(condition *entity.Mongo, cols ...string) error {
	return d.mongoRepo.Get(condition, cols...)
}

// 根据id获取
func (d *mongoAppImpl) GetById(id uint64, cols ...string) *entity.Mongo {
	return d.mongoRepo.GetById(id, cols...)
}

func (d *mongoAppImpl) Delete(id uint64) {
	d.mongoRepo.Delete(id)
	DeleteMongoCache(id)
}

func (d *mongoAppImpl) Save(m *entity.Mongo) {
	if m.Id == 0 {
		d.mongoRepo.Insert(m)
	} else {
		// 先关闭连接
		DeleteMongoCache(m.Id)
		d.mongoRepo.Update(m)
	}
}

func (d *mongoAppImpl) GetMongoCli(id uint64) *mongo.Client {
	mongoInstance, err := GetMongoInstance(id, func(u uint64) *entity.Mongo {
		mongo := d.GetById(u)
		biz.NotNil(mongo, "mongo信息不存在")
		return mongo
	})
	biz.ErrIsNilAppendErr(err, "连接mongo失败: %s")
	return mongoInstance.Cli
}

// -----------------------------------------------------------

// mongo客户端连接缓存，指定时间内没有访问则会被关闭
var mongoCliCache = cache.NewTimedCache(consts.MongoConnExpireTime, 5*time.Second).
	WithUpdateAccessTime(true).
	OnEvicted(func(key any, value any) {
		global.Log.Info("删除mongo连接缓存: id = ", key)
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
func GetMongoInstance(mongoId uint64, getMongoEntity func(uint64) *entity.Mongo) (*MongoInstance, error) {
	mi, err := mongoCliCache.ComputeIfAbsent(mongoId, func(_ any) (any, error) {
		c, err := connect(getMongoEntity(mongoId))
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
	Id                 uint64
	Name               string
	TagPath            string
	SshTunnelMachineId int // ssh隧道机器id
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
			global.Log.Errorf("关闭mongo实例[%d]连接失败: %s", mi.Id, err)
		}
		mi.Cli = nil
	}
}

// 连接mongo，并返回client
func connect(me *entity.Mongo) (*MongoInstance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoInstance := &MongoInstance{Id: me.Id, Info: toMongiInfo(me)}

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

	global.Log.Infof("连接mongo: %s", func(str string) string {
		reg := regexp.MustCompile(`(^mongodb://.+?:)(.+)(@.+$)`)
		return reg.ReplaceAllString(str, `${1}****${3}`)
	}(me.Uri))
	mongoInstance.Cli = client
	return mongoInstance, err
}

func toMongiInfo(me *entity.Mongo) *MongoInfo {
	mi := new(MongoInfo)
	structx.Copy(mi, me)
	return mi
}

type MongoSshDialer struct {
	machineId int
}

func (sd *MongoSshDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if sshConn, err := machineapp.GetMachineApp().GetSshTunnelMachine(sd.machineId).GetDialConn(network, address); err == nil {
		// 将ssh conn包装，否则内部部设置超时会报错,ssh conn不支持设置超时会返回错误: ssh: tcpChan: deadline not supported
		return &netx.WrapSshConn{Conn: sshConn}, nil
	} else {
		return nil, err
	}
}
