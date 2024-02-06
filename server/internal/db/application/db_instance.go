package application

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
)

type Instance interface {
	base.App[*entity.DbInstance]

	// GetPageList 分页获取数据库实例
	GetPageList(condition *entity.InstanceQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	Count(condition *entity.InstanceQuery) int64

	TestConn(instanceEntity *entity.DbInstance) error

	Save(ctx context.Context, instanceEntity *entity.DbInstance) error

	// Delete 删除数据库信息
	Delete(ctx context.Context, id uint64) error

	// GetDatabases 获取数据库实例的所有数据库列表
	GetDatabases(entity *entity.DbInstance) ([]string, error)
}

type instanceAppImpl struct {
	base.AppImpl[*entity.DbInstance, repository.Instance]

	dbApp      Db            `inject:"DbApp"`
	backupApp  *DbBackupApp  `inject:"DbBackupApp"`
	restoreApp *DbRestoreApp `inject:"DbRestoreApp"`
}

// 注入DbInstanceRepo
func (app *instanceAppImpl) InjectDbInstanceRepo(repo repository.Instance) {
	app.Repo = repo
}

// GetPageList 分页获取数据库实例
func (app *instanceAppImpl) GetPageList(condition *entity.InstanceQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.GetRepo().GetInstanceList(condition, pageParam, toEntity, orderBy...)
}

func (app *instanceAppImpl) Count(condition *entity.InstanceQuery) int64 {
	return app.CountByCond(condition)
}

func (app *instanceAppImpl) TestConn(instanceEntity *entity.DbInstance) error {
	instanceEntity.Network = instanceEntity.GetNetwork()
	dbConn, err := dbm.Conn(toDbInfo(instanceEntity, 0, "", ""))
	if err != nil {
		return err
	}
	dbConn.Close()
	return nil
}

func (app *instanceAppImpl) Save(ctx context.Context, instanceEntity *entity.DbInstance) error {
	// 默认tcp连接
	instanceEntity.Network = instanceEntity.GetNetwork()

	// 查找是否存在该库
	oldInstance := &entity.DbInstance{
		Host:               instanceEntity.Host,
		Port:               instanceEntity.Port,
		Username:           instanceEntity.Username,
		SshTunnelMachineId: instanceEntity.SshTunnelMachineId,
	}

	err := app.GetBy(oldInstance)
	if instanceEntity.Id == 0 {

		if instanceEntity.Type != string(dbi.DbTypeSqlite) && instanceEntity.Password == "" {
			return errorx.NewBiz("密码不能为空")
		}

		if err == nil {
			return errorx.NewBiz("该数据库实例已存在")
		}
		if err := instanceEntity.PwdEncrypt(); err != nil {
			return errorx.NewBiz(err.Error())
		}
		return app.Insert(ctx, instanceEntity)
	}

	// 如果存在该库，则校验修改的库是否为该库
	if err == nil && oldInstance.Id != instanceEntity.Id {
		return errorx.NewBiz("该数据库实例已存在")
	}
	if err := instanceEntity.PwdEncrypt(); err != nil {
		return errorx.NewBiz(err.Error())
	}
	return app.UpdateById(ctx, instanceEntity)
}

func (app *instanceAppImpl) Delete(ctx context.Context, instanceId uint64) error {
	instance, err := app.GetById(new(entity.DbInstance), instanceId, "name")
	biz.ErrIsNil(err, "获取数据库实例错误，数据库实例ID为: %d", instance.Id)

	restore := &entity.DbRestore{
		DbInstanceId: instanceId,
	}
	err = app.restoreApp.restoreRepo.GetBy(restore)
	switch {
	case err == nil:
		biz.ErrNotNil(err, "不能删除数据库实例【%s】，请先删除关联的数据库恢复任务。", instance.Name)
	case errors.Is(err, gorm.ErrRecordNotFound):
		break
	default:
		biz.ErrIsNil(err, "删除数据库实例失败: %v", err)
	}

	backup := &entity.DbBackup{
		DbInstanceId: instanceId,
	}
	err = app.backupApp.backupRepo.GetBy(backup)
	switch {
	case err == nil:
		biz.ErrNotNil(err, "不能删除数据库实例【%s】，请先删除关联的数据库备份任务。", instance.Name)
	case errors.Is(err, gorm.ErrRecordNotFound):
		break
	default:
		biz.ErrIsNil(err, "删除数据库实例失败: %v", err)
	}

	db := &entity.Db{
		InstanceId: instanceId,
	}
	err = app.dbApp.GetBy(db)
	switch {
	case err == nil:
		biz.ErrNotNil(err, "不能删除数据库实例【%s】，请先删除关联的数据库资源。", instance.Name)
	case errors.Is(err, gorm.ErrRecordNotFound):
		break
	default:
		biz.ErrIsNil(err, "删除数据库实例失败: %v", err)
	}

	return app.DeleteById(ctx, instanceId)
}

func (app *instanceAppImpl) GetDatabases(ed *entity.DbInstance) ([]string, error) {
	ed.Network = ed.GetNetwork()
	metaDb := dbi.ToDbType(ed.Type).MetaDbName()

	dbConn, err := dbm.Conn(toDbInfo(ed, 0, metaDb, ""))
	if err != nil {
		return nil, err
	}
	defer dbConn.Close()

	return dbConn.GetDialect().GetDbNames()
}
