package application

import (
	"mayfly-go/internal/mongo/domain/entity"
	"mayfly-go/internal/mongo/domain/repository"
	"mayfly-go/internal/mongo/mgm"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
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
	GetMongoConn(id uint64) (*mgm.MongoConn, error)
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
	mgm.CloseConn(id)
	return d.GetRepo().DeleteById(id)
}

func (d *mongoAppImpl) Save(m *entity.Mongo) error {
	if m.Id == 0 {
		return d.GetRepo().Insert(m)
	}

	// 先关闭连接
	mgm.CloseConn(m.Id)
	return d.GetRepo().UpdateById(m)
}

func (d *mongoAppImpl) GetMongoConn(id uint64) (*mgm.MongoConn, error) {
	return mgm.GetMongoConn(id, func() (*mgm.MongoInfo, error) {
		mongo, err := d.GetById(new(entity.Mongo), id)
		if err != nil {
			return nil, errorx.NewBiz("mongo信息不存在")
		}
		return mongo.ToMongoInfo(), nil
	})
}
