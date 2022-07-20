package application

import (
	"context"
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
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/ssh"
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

//mongo客户端连接缓存，30分钟内没有访问则会被关闭
var mongoCliCache = cache.NewTimedCache(30*time.Minute, 5*time.Second).
	WithUpdateAccessTime(true).
	OnEvicted(func(key interface{}, value interface{}) {
		global.Log.Info("删除mongo连接缓存: id = ", key)
		value.(*MongoInstance).Close()
	})

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
	Id        uint64
	ProjectId uint64
	Cli       *mongo.Client
	sshTunnel *ssh.Client
}

func (mi *MongoInstance) Close() {
	if mi.Cli != nil {
		if err := mi.Cli.Disconnect(context.Background()); err != nil {
			global.Log.Errorf("关闭mongo实例[%d]连接失败: %s", mi.Id, err)
		}
	}
	if mi.sshTunnel != nil {
		if err := mi.sshTunnel.Close(); err != nil {
			global.Log.Errorf("关闭mongo实例[%d]的ssh隧道失败: %s", mi.Id, err.Error())
		}
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
		machineEntity := MachineApp.GetById(4)
		sshClient, err := machine.GetSshClient(machineEntity)
		biz.ErrIsNilAppendErr(err, "ssh隧道连接失败: %s")
		mongoInstance.sshTunnel = sshClient

		mongoOptions.SetDialer(&MongoSshDialer{sshTunnel: sshClient})
	}

	client, err := mongo.Connect(ctx, mongoOptions)
	if err != nil {
		return nil, err
	}
	if err = client.Ping(context.TODO(), nil); err != nil {
		return nil, err
	}

	global.Log.Infof("连接mongo: %s", me.Uri)
	mongoInstance.Cli = client
	return mongoInstance, err
}

type MongoSshDialer struct {
	sshTunnel *ssh.Client
}

func (sd *MongoSshDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if sshConn, err := sd.sshTunnel.Dial(network, address); err == nil {
		// 将ssh conn包装，否则内部部设置超时会报错,ssh conn不支持设置超时会返回错误: ssh: tcpChan: deadline not supported
		return &utils.WrapSshConn{Conn: sshConn}, nil
	} else {
		return nil, err
	}
}
