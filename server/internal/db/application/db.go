package application

import (
	"context"
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/utils/structx"
	"strings"
)

type Db interface {
	base.App[*entity.Db]

	// 分页获取
	GetPageList(condition *entity.DbQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	Count(condition *entity.DbQuery) int64

	SaveDb(ctx context.Context, entity *entity.Db, tagIds ...uint64) error

	// 删除数据库信息
	Delete(ctx context.Context, id uint64) error

	// 获取数据库连接实例
	// @param id 数据库id
	//
	// @param dbName 数据库名
	GetDbConn(dbId uint64, dbName string) (*dbi.DbConn, error)

	// 根据数据库实例id获取连接，随机返回该instanceId下已连接的conn，若不存在则是使用该instanceId关联的db进行连接并返回。
	GetDbConnByInstanceId(instanceId uint64) (*dbi.DbConn, error)
}

type dbAppImpl struct {
	base.AppImpl[*entity.Db, repository.Db]

	dbSqlRepo     repository.DbSql `inject:"DbSqlRepo"`
	dbInstanceApp Instance         `inject:"DbInstanceApp"`
	tagApp        tagapp.TagTree   `inject:"TagTreeApp"`
}

// 注入DbRepo
func (d *dbAppImpl) InjectDbRepo(repo repository.Db) {
	d.Repo = repo
}

// 分页获取数据库信息列表
func (d *dbAppImpl) GetPageList(condition *entity.DbQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return d.GetRepo().GetDbList(condition, pageParam, toEntity, orderBy...)
}

func (d *dbAppImpl) Count(condition *entity.DbQuery) int64 {
	return d.GetRepo().Count(condition)
}

func (d *dbAppImpl) SaveDb(ctx context.Context, dbEntity *entity.Db, tagIds ...uint64) error {
	// 查找是否存在
	oldDb := &entity.Db{Name: dbEntity.Name, InstanceId: dbEntity.InstanceId}
	err := d.GetBy(oldDb)

	if dbEntity.Id == 0 {
		if err == nil {
			return errorx.NewBiz("该实例下数据库名已存在")
		}

		resouceCode := stringx.Rand(16)
		dbEntity.Code = resouceCode

		return d.Tx(ctx, func(ctx context.Context) error {
			return d.Insert(ctx, dbEntity)
		}, func(ctx context.Context) error {
			return d.tagApp.RelateResource(ctx, resouceCode, consts.TagResourceTypeDb, tagIds)
		})
	}

	// 如果存在该库，则校验修改的库是否为该库
	if err == nil && oldDb.Id != dbEntity.Id {
		return errorx.NewBiz("该实例下数据库名已存在")
	}

	dbId := dbEntity.Id
	old, err := d.GetById(new(entity.Db), dbId)
	if err != nil {
		return errorx.NewBiz("该数据库不存在")
	}

	oldDbs := strings.Split(old.Database, " ")
	newDbs := strings.Split(dbEntity.Database, " ")
	// 比较新旧数据库列表，需要将移除的数据库相关联的信息删除
	_, delDb, _ := collx.ArrayCompare(newDbs, oldDbs)

	// 先简单关闭可能存在的旧库连接（可能改了关联标签导致DbConn.Info.TagPath与修改后的标签不一致、导致操作权限校验出错）
	for _, v := range oldDbs {
		// 关闭数据库连接
		dbm.CloseDb(dbEntity.Id, v)
	}

	for _, v := range delDb {
		// 删除该库关联的所有sql记录
		d.dbSqlRepo.DeleteByCond(ctx, &entity.DbSql{DbId: dbId, Db: v})
	}

	return d.Tx(ctx, func(ctx context.Context) error {
		return d.UpdateById(ctx, dbEntity)
	}, func(ctx context.Context) error {
		return d.tagApp.RelateResource(ctx, old.Code, consts.TagResourceTypeDb, tagIds)
	})
}

func (d *dbAppImpl) Delete(ctx context.Context, id uint64) error {
	db, err := d.GetById(new(entity.Db), id)
	if err != nil {
		return errorx.NewBiz("该数据库不存在")
	}
	dbs := strings.Split(db.Database, " ")
	for _, v := range dbs {
		// 关闭连接
		dbm.CloseDb(id, v)
	}

	return d.Tx(ctx,
		func(ctx context.Context) error {
			return d.DeleteById(ctx, id)
		},
		func(ctx context.Context) error {
			// 删除该库下用户保存的所有sql信息
			return d.dbSqlRepo.DeleteByCond(ctx, &entity.DbSql{DbId: id})
		}, func(ctx context.Context) error {
			var tagIds []uint64
			return d.tagApp.RelateResource(ctx, db.Code, consts.TagResourceTypeDb, tagIds)
		})
}

func (d *dbAppImpl) GetDbConn(dbId uint64, dbName string) (*dbi.DbConn, error) {
	return dbm.GetDbConn(dbId, dbName, func() (*dbi.DbInfo, error) {
		db, err := d.GetById(new(entity.Db), dbId)
		if err != nil {
			return nil, errorx.NewBiz("数据库信息不存在")
		}

		instance, err := d.dbInstanceApp.GetById(new(entity.DbInstance), db.InstanceId)
		if err != nil {
			return nil, errorx.NewBiz("数据库实例不存在")
		}

		checkDb := dbName
		// 兼容pgsql/dm db/schema模式
		if dbi.DbTypePostgres.Equal(instance.Type) || dbi.DbTypeGauss.Equal(instance.Type) || dbi.DbTypeDM.Equal(instance.Type) || dbi.DbTypeOracle.Equal(instance.Type) || dbi.DbTypeMssql.Equal(instance.Type) || dbi.DbTypeKingbaseEs.Equal(instance.Type) || dbi.DbTypeVastbase.Equal(instance.Type) {
			ss := strings.Split(dbName, "/")
			if len(ss) > 1 {
				checkDb = ss[0]
			}
		}
		if !strings.Contains(" "+db.Database+" ", " "+checkDb+" ") {
			return nil, errorx.NewBiz("未配置数据库【%s】的操作权限", dbName)
		}

		// 密码解密
		if err := instance.PwdDecrypt(); err != nil {
			return nil, errorx.NewBiz(err.Error())
		}
		return toDbInfo(instance, dbId, dbName, d.tagApp.ListTagPathByResource(consts.TagResourceTypeDb, db.Code)...), nil
	})
}

func (d *dbAppImpl) GetDbConnByInstanceId(instanceId uint64) (*dbi.DbConn, error) {
	conn := dbm.GetDbConnByInstanceId(instanceId)
	if conn != nil {
		return conn, nil
	}

	var dbs []*entity.Db
	if err := d.ListByCond(&entity.Db{InstanceId: instanceId}, &dbs, "id", "database"); err != nil {
		return nil, errorx.NewBiz("获取数据库列表失败")
	}
	if len(dbs) == 0 {
		return nil, errorx.NewBiz("该实例未配置数据库, 请先进行配置")
	}

	// 使用该实例关联的已配置数据库中的第一个库进行连接并返回
	firstDb := dbs[0]
	return d.GetDbConn(firstDb.Id, strings.Split(firstDb.Database, " ")[0])
}

func toDbInfo(instance *entity.DbInstance, dbId uint64, database string, tagPath ...string) *dbi.DbInfo {
	di := new(dbi.DbInfo)
	di.InstanceId = instance.Id
	di.Sid = instance.Sid
	di.Id = dbId
	di.Database = database
	di.TagPath = tagPath

	structx.Copy(di, instance)
	return di
}
