package application

import (
	"context"
	"mayfly-go/internal/db/application/dto"
	transfer "mayfly-go/internal/db/application/transfer"
	"mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/imsg"
	tagapp "mayfly-go/internal/tag/application"
	tagdto "mayfly-go/internal/tag/application/dto"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"strings"
)

type Db interface {
	base.App[*entity.Db]

	// 分页获取
	GetPageList(condition *entity.DbQuery, orderBy ...string) (*model.PageResult[*entity.DbListPO], error)

	SaveDb(ctx context.Context, entity *entity.Db) error

	// 删除数据库信息
	Delete(ctx context.Context, id uint64) error

	// GetDbConn 获取数据库连接实例
	//    - dbId: 数据库id
	//    - dbName: 数据库名
	GetDbConn(ctx context.Context, dbId uint64, dbName string) (*dbi.DbConn, error)

	// 根据数据库实例id获取连接，随机返回该instanceId下已连接的conn，若不存在则是使用该instanceId关联的db进行连接并返回。
	GetDbConnByInstanceId(ctx context.Context, instanceId uint64) (*dbi.DbConn, error)

	// DumpDb dumpDb
	DumpDb(ctx context.Context, reqParam *dto.DumpDb) error
}

type dbAppImpl struct {
	base.AppImpl[*entity.Db, repository.Db]

	dbSqlRepo           repository.DbSql        `inject:"T"`
	dbInstanceApp       Instance                `inject:"T"`
	dbSqlExecApp        DbSqlExec               `inject:"T"`
	tagApp              tagapp.TagTree          `inject:"T"`
	resourceAuthCertApp tagapp.ResourceAuthCert `inject:"T"`
}

var _ Db = (*dbAppImpl)(nil)

var _ (Db) = (*dbAppImpl)(nil)

// 分页获取数据库信息列表
func (d *dbAppImpl) GetPageList(condition *entity.DbQuery, orderBy ...string) (*model.PageResult[*entity.DbListPO], error) {
	return d.GetRepo().GetDbList(condition, orderBy...)
}

func (d *dbAppImpl) SaveDb(ctx context.Context, dbEntity *entity.Db) error {
	// 查找是否存在
	oldDb := &entity.Db{Name: dbEntity.Name, InstanceId: dbEntity.InstanceId, AuthCertName: dbEntity.AuthCertName}

	authCert, err := d.resourceAuthCertApp.GetAuthCert(dbEntity.AuthCertName)
	if err != nil {
		return errorx.NewBiz("ac not found")
	}

	err = d.GetByCond(oldDb)
	if dbEntity.Id == 0 {
		if err == nil {
			return errorx.NewBizI(ctx, imsg.ErrDbNameExist)
		}
		dbEntity.Code = stringx.Rand(10)

		return d.Tx(ctx, func(ctx context.Context) error {
			return d.Insert(ctx, dbEntity)
		}, func(ctx context.Context) error {
			// 将库关联至指定数据库授权凭证下
			return d.tagApp.RelateTagsByCodeAndType(ctx, &tagdto.RelateTagsByCodeAndType{
				Tags: []*tagdto.ResourceTag{{
					Code: dbEntity.Code,
					Type: tagentity.TagTypeDb,
					Name: dbEntity.Name,
				}},
				ParentTagCode: authCert.Name,
				ParentTagType: tagentity.TagTypeAuthCert,
			})
		})
	}

	// 如果存在该库，则校验修改的库是否为该库
	if err == nil && oldDb.Id != dbEntity.Id {
		return errorx.NewBizI(ctx, imsg.ErrDbNameExist)
	}

	dbId := dbEntity.Id
	old, err := d.GetById(dbId)
	if err != nil {
		return errorx.NewBiz("db not found")
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

	// 防止误传修改
	dbEntity.Code = ""
	return d.Tx(ctx, func(ctx context.Context) error {
		return d.UpdateById(ctx, dbEntity)
	}, func(ctx context.Context) error {
		if old.Name != dbEntity.Name {
			if err := d.tagApp.UpdateTagName(ctx, tagentity.TagTypeDb, old.Code, dbEntity.Name); err != nil {
				return err
			}
		}
		if authCert.Name != old.AuthCertName {
			return d.tagApp.ChangeParentTag(ctx, tagentity.TagTypeDb, old.Code, tagentity.TagTypeAuthCert, authCert.Name)
		}
		return nil
	})
}

func (d *dbAppImpl) Delete(ctx context.Context, id uint64) error {
	db, err := d.GetById(id)
	if err != nil {
		return errorx.NewBiz("db not found")
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
			return d.dbSqlExecApp.DeleteBy(ctx, &entity.DbSqlExec{DbId: id})
		}, func(ctx context.Context) error {
			return d.tagApp.DeleteTagByParam(ctx, &tagdto.DelResourceTag{
				ResourceCode: db.Code,
				ResourceType: tagentity.TagTypeDb,
			})
		})
}

func (d *dbAppImpl) GetDbConn(ctx context.Context, dbId uint64, dbName string) (*dbi.DbConn, error) {
	return dbm.GetDbConn(ctx, dbId, dbName, func() (*dbi.DbInfo, error) {
		db, err := d.GetById(dbId)
		if err != nil {
			return nil, errorx.NewBiz("db not found")
		}

		instance, err := d.dbInstanceApp.GetById(db.InstanceId)
		if err != nil {
			return nil, errorx.NewBiz("db instance not found")
		}

		di, err := d.dbInstanceApp.ToDbInfo(instance, db.AuthCertName, dbName)
		if err != nil {
			return nil, err
		}
		di.CodePath = d.tagApp.ListTagPathByTypeAndCode(int8(tagentity.TagTypeDb), db.Code)
		di.Id = db.Id
		di.DbCode = db.Code

		checkDb := di.GetDatabase()
		if db.GetDatabaseMode == entity.DbGetDatabaseModeAssign && !strings.Contains(" "+db.Database+" ", " "+checkDb+" ") {
			return nil, errorx.NewBizI(context.Background(), imsg.ErrDbNotAccess, "dbName", dbName)
		}

		return di, nil
	})
}

func (d *dbAppImpl) GetDbConnByInstanceId(ctx context.Context, instanceId uint64) (*dbi.DbConn, error) {
	conn := dbm.GetDbConnByInstanceId(ctx, instanceId)
	if conn != nil {
		return conn, nil
	}

	dbs, err := d.ListByCond(&entity.Db{InstanceId: instanceId}, "id", "database")
	if err != nil {
		return nil, errorx.NewBiz("failed to get database list")
	}
	if len(dbs) == 0 {
		return nil, errorx.NewBizf("DB instance [%d] database is not configured, please configure it first", instanceId)
	}

	// 使用该实例关联的已配置数据库中的第一个库进行连接并返回
	firstDb := dbs[0]
	return d.GetDbConn(ctx, firstDb.Id, strings.Split(firstDb.Database, " ")[0])
}

func (d *dbAppImpl) DumpDb(ctx context.Context, reqParam *dto.DumpDb) error {
	dbConn, err := d.GetDbConn(ctx, reqParam.DbId, reqParam.DbName)
	if err != nil {
		return err
	}
	return transfer.DumpDbScript(ctx, dbConn, reqParam)
}
