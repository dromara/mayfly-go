package application

import (
	"context"
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/mongo/domain/entity"
	"mayfly-go/internal/mongo/domain/repository"
	"mayfly-go/internal/mongo/mgm"
	tagapp "mayfly-go/internal/tag/application"
	tagdto "mayfly-go/internal/tag/application/dto"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
)

type Mongo interface {
	base.App[*entity.Mongo]

	// 分页获取机器脚本信息列表
	GetPageList(condition *entity.MongoQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	TestConn(entity *entity.Mongo) error

	SaveMongo(ctx context.Context, entity *entity.Mongo, tagCodePaths ...string) error

	// 删除数据库信息
	Delete(ctx context.Context, id uint64) error

	// 获取mongo连接实例
	// @param id mongo id
	GetMongoConn(id uint64) (*mgm.MongoConn, error)
}

type mongoAppImpl struct {
	base.AppImpl[*entity.Mongo, repository.Mongo]

	tagApp tagapp.TagTree `inject:"TagTreeApp"`
}

// 注入MongoRepo
func (d *mongoAppImpl) InjectMongoRepo(repo repository.Mongo) {
	d.Repo = repo
}

// 分页获取数据库信息列表
func (d *mongoAppImpl) GetPageList(condition *entity.MongoQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return d.GetRepo().GetList(condition, pageParam, toEntity, orderBy...)
}

func (d *mongoAppImpl) Delete(ctx context.Context, id uint64) error {
	mongoEntity, err := d.GetById(id)
	if err != nil {
		return errorx.NewBiz("mongo信息不存在")
	}

	mgm.CloseConn(id)
	return d.Tx(ctx,
		func(ctx context.Context) error {
			return d.DeleteById(ctx, id)
		},
		func(ctx context.Context) error {
			return d.tagApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{ResourceTag: &tagdto.ResourceTag{
				Type: tagentity.TagTypeMongo,
				Code: mongoEntity.Code,
			}})
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

func (d *mongoAppImpl) SaveMongo(ctx context.Context, m *entity.Mongo, tagCodePaths ...string) error {
	oldMongo := &entity.Mongo{Name: m.Name, SshTunnelMachineId: m.SshTunnelMachineId}
	err := d.GetByCond(oldMongo)

	if m.Id == 0 {
		if err == nil {
			return errorx.NewBiz("该名称已存在")
		}
		if d.CountByCond(&entity.Mongo{Code: m.Code}) > 0 {
			return errorx.NewBiz("该编码已存在")
		}

		return d.Tx(ctx, func(ctx context.Context) error {
			return d.Insert(ctx, m)
		}, func(ctx context.Context) error {
			return d.tagApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{
				ResourceTag: &tagdto.ResourceTag{
					Type: tagentity.TagTypeMongo,
					Code: m.Code,
				},
				ParentTagCodePaths: tagCodePaths,
			})
		})
	}

	// 如果存在该库，则校验修改的库是否为该库
	if err == nil && oldMongo.Id != m.Id {
		return errorx.NewBiz("该名称已存在")
	}
	// 如果调整了ssh等会查不到旧数据，故需要根据id获取旧信息将code赋值给标签进行关联
	if oldMongo.Code == "" {
		oldMongo, _ = d.GetById(m.Id)
	}

	// 先关闭连接
	mgm.CloseConn(m.Id)
	m.Code = ""
	return d.Tx(ctx, func(ctx context.Context) error {
		return d.UpdateById(ctx, m)
	}, func(ctx context.Context) error {
		if oldMongo.Name != m.Name {
			if err := d.tagApp.UpdateTagName(ctx, tagentity.TagTypeMachine, oldMongo.Code, m.Name); err != nil {
				return err
			}
		}

		return d.tagApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{
			ResourceTag: &tagdto.ResourceTag{
				Type: tagentity.TagTypeMongo,
				Code: oldMongo.Code,
			},
			ParentTagCodePaths: tagCodePaths,
		})
	})
}

func (d *mongoAppImpl) GetMongoConn(id uint64) (*mgm.MongoConn, error) {
	return mgm.GetMongoConn(id, func() (*mgm.MongoInfo, error) {
		me, err := d.GetById(id)
		if err != nil {
			return nil, errorx.NewBiz("mongo信息不存在")
		}
		return me.ToMongoInfo(d.tagApp.ListTagPathByTypeAndCode(consts.ResourceTypeMongo, me.Code)...), nil
	})
}
