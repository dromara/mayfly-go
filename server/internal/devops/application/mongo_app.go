package application

import (
	"context"
	"mayfly-go/internal/devops/domain/entity"
	"mayfly-go/internal/devops/domain/repository"
	"mayfly-go/internal/devops/infrastructure/persistence"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"
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
	cli, err := GetMongoCli(id, func(u uint64) string {
		mongo := d.GetById(id)
		biz.NotNil(mongo, "mongo信息不存在")
		return mongo.Uri
	})
	biz.ErrIsNilAppendErr(err, "连接mongo失败: %s")
	return cli
}

// -----------------------------------------------------------

//mongo客户端连接缓存，30分钟内没有访问则会被关闭
var mongoCliCache = cache.NewTimedCache(30*time.Minute, 5*time.Second).
	WithUpdateAccessTime(true).
	OnEvicted(func(key interface{}, value interface{}) {
		global.Log.Info("关闭mongo连接: id = ", key)
		value.(*mongo.Client).Disconnect(context.TODO())
	})

func GetMongoCli(mongoId uint64, getMongoUri func(uint64) string) (*mongo.Client, error) {
	cli, err := mongoCliCache.ComputeIfAbsent(mongoId, func(key interface{}) (interface{}, error) {
		c, err := connect(getMongoUri(mongoId))
		if err != nil {
			return nil, err
		}
		return c, nil
	})

	if cli != nil {
		return cli.(*mongo.Client), err
	}
	return nil, err
}

func DeleteMongoCache(mongoId uint64) {
	mongoCliCache.Delete(mongoId)
}

// 连接mongo，并返回client
func connect(uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri).SetMaxPoolSize(2))
	if err != nil {
		return nil, err
	}
	if err = client.Ping(context.TODO(), nil); err != nil {
		return nil, err
	}
	return client, err
}
