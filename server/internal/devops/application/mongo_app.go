package application

import (
	"context"
	"mayfly-go/internal/constant"
	"mayfly-go/internal/devops/domain/entity"
	"mayfly-go/internal/devops/domain/repository"
	"mayfly-go/internal/devops/infrastructure/machine"
	"mayfly-go/internal/devops/infrastructure/persistence"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils"
	"net"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Mongo interface {
	// 分页获取机器脚本信息列表
	GetPageList(condition *entity.Mongo, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	Count(condition *entity.Mongo) int64

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

type mongoAppImpl struct {
	mongoRepo repository.Mongo
}

var MongoApp Mongo = &mongoAppImpl{
	mongoRepo: persistence.MongoDao,
}

// 分页获取数据库信息列表
func (d *mongoAppImpl) GetPageList(condition *entity.Mongo, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return d.mongoRepo.GetList(condition, pageParam, toEntity, orderBy...)
}

func (d *mongoAppImpl) Count(condition *entity.Mongo) int64 {
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
var mongoCliCache = cache.NewTimedCache(constant.MongoConnExpireTime, 5*time.Second).
	WithUpdateAccessTime(true).
	OnEvicted(func(key interface{}, value interface{}) {
		global.Log.Info("删除mongo连接缓存: id = ", key)
		value.(*MongoInstance).Close()
	})

func init() {
	machine.AddCheckSshTunnelMachineUseFunc(func(machineId uint64) bool {
		// 遍历所有mongo连接实例，若存在redis实例使用该ssh隧道机器，则返回true，表示还在使用中...
		items := mongoCliCache.Items()
		for _, v := range items {
			if v.Value.(*MongoInstance).sshTunnelMachineId == machineId {
				return true
			}
		}
		return false
	})
}

// 获取mongo的连接实例
func GetMongoInstance(mongoId uint64, getMongoEntity func(uint64) *entity.Mongo) (*MongoInstance, error) {
	mi, err := mongoCliCache.ComputeIfAbsent(mongoId, func(_ interface{}) (interface{}, error) {
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

type MongoInstance struct {
	Id                 uint64
	ProjectId          uint64
	Cli                *mongo.Client
	sshTunnelMachineId uint64
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

	mongoInstance := &MongoInstance{Id: me.Id, ProjectId: me.ProjectId}

	mongoOptions := options.Client().ApplyURI(me.Uri).
		SetMaxPoolSize(1)
	// 启用ssh隧道则连接隧道机器
	if me.EnableSshTunnel == 1 {
		mongoInstance.sshTunnelMachineId = me.SshTunnelMachineId
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

type MongoSshDialer struct {
	machineId uint64
}

func (sd *MongoSshDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if sshConn, err := MachineApp.GetSshTunnelMachine(sd.machineId).GetDialConn(network, address); err == nil {
		// 将ssh conn包装，否则内部部设置超时会报错,ssh conn不支持设置超时会返回错误: ssh: tcpChan: deadline not supported
		return &utils.WrapSshConn{Conn: sshConn}, nil
	} else {
		return nil, err
	}
}
