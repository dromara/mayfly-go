package application

import (
	"context"
	"mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/structx"
	"strings"
)

type Db interface {
	base.App[*entity.Db]

	// 分页获取
	GetPageList(condition *entity.DbQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	Count(condition *entity.DbQuery) int64

	Save(ctx context.Context, entity *entity.Db) error

	// 删除数据库信息
	Delete(ctx context.Context, id uint64) error

	// 获取数据库连接实例
	// @param id 数据库id
	// @param dbName 数据库
	GetDbConn(dbId uint64, dbName string) (*dbm.DbConn, error)
}

func newDbApp(dbRepo repository.Db, dbSqlRepo repository.DbSql, dbInstanceApp Instance) Db {
	app := &dbAppImpl{
		dbSqlRepo:     dbSqlRepo,
		dbInstanceApp: dbInstanceApp,
	}
	app.Repo = dbRepo
	return app
}

type dbAppImpl struct {
	base.AppImpl[*entity.Db, repository.Db]

	dbSqlRepo     repository.DbSql
	dbInstanceApp Instance
}

// 分页获取数据库信息列表
func (d *dbAppImpl) GetPageList(condition *entity.DbQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return d.GetRepo().GetDbList(condition, pageParam, toEntity, orderBy...)
}

func (d *dbAppImpl) Count(condition *entity.DbQuery) int64 {
	return d.GetRepo().Count(condition)
}

func (d *dbAppImpl) Save(ctx context.Context, dbEntity *entity.Db) error {
	// 查找是否存在
	oldDb := &entity.Db{Name: dbEntity.Name, InstanceId: dbEntity.InstanceId}
	err := d.GetBy(oldDb)

	if dbEntity.Id == 0 {
		if err == nil {
			return errorx.NewBiz("该实例下数据库名已存在")
		}
		return d.Insert(ctx, dbEntity)
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
	_, delDb, _ := collx.ArrayCompare(newDbs, oldDbs, func(i1, i2 string) bool {
		return i1 == i2
	})

	for _, v := range delDb {
		// 关闭数据库连接
		dbm.CloseDb(dbEntity.Id, v)
		// 删除该库关联的所有sql记录
		d.dbSqlRepo.DeleteByCond(ctx, &entity.DbSql{DbId: dbId, Db: v})
	}

	return d.UpdateById(ctx, dbEntity)
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
	// 删除该库下用户保存的所有sql信息
	d.dbSqlRepo.DeleteByCond(ctx, &entity.DbSql{DbId: id})
	return d.DeleteById(ctx, id)
}

func (d *dbAppImpl) GetDbConn(dbId uint64, dbName string) (*dbm.DbConn, error) {
	return dbm.GetDbConn(dbId, dbName, func() (*dbm.DbInfo, error) {
		db, err := d.GetById(new(entity.Db), dbId)
		if err != nil {
			return nil, errorx.NewBiz("数据库信息不存在")
		}

		instance, err := d.dbInstanceApp.GetById(new(entity.DbInstance), db.InstanceId)
		if err != nil {
			return nil, errorx.NewBiz("数据库实例不存在")
		}

		checkDb := dbName
		// 兼容pgsql db/schema模式
		if instance.Type == dbm.DbTypePostgres {
			ss := strings.Split(dbName, "/")
			if len(ss) > 1 {
				checkDb = ss[0]
			}
		}
		if !strings.Contains(" "+db.Database+" ", " "+checkDb+" ") {
			return nil, errorx.NewBiz("未配置数据库【%s】的操作权限", dbName)
		}

		// 密码解密
		instance.PwdDecrypt()
		return toDbInfo(instance, dbId, dbName, db.TagPath), nil
	})
}

func toDbInfo(instance *entity.DbInstance, dbId uint64, database string, tagPath string) *dbm.DbInfo {
	di := new(dbm.DbInfo)
	di.Id = dbId
	di.Database = database
	di.TagPath = tagPath

	structx.Copy(di, instance)
	return di
}
