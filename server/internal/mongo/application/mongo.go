package application

import (
	"context"
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/mongo/domain/entity"
	"mayfly-go/internal/mongo/domain/repository"
	"mayfly-go/internal/mongo/mgm"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/stringx"
)

type Mongo interface {
	base.App[*entity.Mongo]

	// 分页获取机器脚本信息列表
	GetPageList(condition *entity.MongoQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	TestConn(entity *entity.Mongo) error

	Save(ctx context.Context, entity *entity.Mongo, tagIds ...uint64) error

	// 删除数据库信息
	Delete(ctx context.Context, id uint64) error

	// 获取mongo连接实例
	// @param id mongo id
	GetMongoConn(id uint64) (*mgm.MongoConn, error)
}

func newMongoAppImpl(mongoRepo repository.Mongo, tagApp tagapp.TagTree) Mongo {
	app := &mongoAppImpl{
		tagApp: tagApp,
	}
	app.Repo = mongoRepo
	return app
}

type mongoAppImpl struct {
	base.AppImpl[*entity.Mongo, repository.Mongo]

	tagApp tagapp.TagTree
}

// 分页获取数据库信息列表
func (d *mongoAppImpl) GetPageList(condition *entity.MongoQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return d.GetRepo().GetList(condition, pageParam, toEntity, orderBy...)
}

func (d *mongoAppImpl) Delete(ctx context.Context, id uint64) error {
	mongoEntity, err := d.GetById(new(entity.Mongo), id)
	if err != nil {
		return errorx.NewBiz("mongo信息不存在")
	}

	mgm.CloseConn(id)
	return d.Tx(ctx,
		func(ctx context.Context) error {
			return d.DeleteById(ctx, id)
		},
		func(ctx context.Context) error {
			var tagIds []uint64
			return d.tagApp.RelateResource(ctx, mongoEntity.Code, consts.TagResourceTypeMongo, tagIds)
		})
}

func (d *mongoAppImpl) TestConn(me *entity.Mongo) error {
	conn, err := me.ToMongoInfo().Conn()
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

func (d *mongoAppImpl) Save(ctx context.Context, m *entity.Mongo, tagIds ...uint64) error {
	oldMongo := &entity.Mongo{Name: m.Name}
	err := d.GetBy(oldMongo)

	if m.Id == 0 {
		if err == nil {
			return errorx.NewBiz("该名称已存在")
		}

		resouceCode := stringx.Rand(16)
		m.Code = resouceCode

		return d.Tx(ctx, func(ctx context.Context) error {
			return d.Insert(ctx, m)
		}, func(ctx context.Context) error {
			return d.tagApp.RelateResource(ctx, resouceCode, consts.TagResourceTypeMongo, tagIds)
		})
	}

	// 如果存在该库，则校验修改的库是否为该库
	if err == nil && oldMongo.Id != m.Id {
		return errorx.NewBiz("该名称已存在")
	}

	// 先关闭连接
	mgm.CloseConn(m.Id)
	return d.Tx(ctx, func(ctx context.Context) error {
		return d.UpdateById(ctx, m)
	}, func(ctx context.Context) error {
		return d.tagApp.RelateResource(ctx, oldMongo.Code, consts.TagResourceTypeMongo, tagIds)
	})
}

func (d *mongoAppImpl) GetMongoConn(id uint64) (*mgm.MongoConn, error) {
	return mgm.GetMongoConn(id, func() (*mgm.MongoInfo, error) {
		me, err := d.GetById(new(entity.Mongo), id)
		if err != nil {
			return nil, errorx.NewBiz("mongo信息不存在")
		}
		return me.ToMongoInfo(d.tagApp.ListTagPathByResource(consts.TagResourceTypeMongo, me.Code)...), nil
	})
}
