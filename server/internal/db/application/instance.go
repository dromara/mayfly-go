package application

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
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

func newInstanceApp(instanceRepo repository.Instance) Instance {
	app := new(instanceAppImpl)
	app.Repo = instanceRepo
	return app
}

type instanceAppImpl struct {
	base.AppImpl[*entity.DbInstance, repository.Instance]
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
	dbConn, err := toDbInfo(instanceEntity, 0, "", "").Conn()
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
	oldInstance := &entity.DbInstance{Host: instanceEntity.Host, Port: instanceEntity.Port, Username: instanceEntity.Username}
	if instanceEntity.SshTunnelMachineId > 0 {
		oldInstance.SshTunnelMachineId = instanceEntity.SshTunnelMachineId
	}

	err := app.GetBy(oldInstance)
	if instanceEntity.Id == 0 {
		if instanceEntity.Password == "" {
			return errorx.NewBiz("密码不能为空")
		}
		if err == nil {
			return errorx.NewBiz("该数据库实例已存在")
		}
		instanceEntity.PwdEncrypt()
		return app.Insert(ctx, instanceEntity)
	}

	// 如果存在该库，则校验修改的库是否为该库
	if err == nil && oldInstance.Id != instanceEntity.Id {
		return errorx.NewBiz("该数据库实例已存在")
	}
	instanceEntity.PwdEncrypt()
	return app.UpdateById(ctx, instanceEntity)
}

func (app *instanceAppImpl) Delete(ctx context.Context, id uint64) error {
	return app.DeleteById(ctx, id)
}

func (app *instanceAppImpl) GetDatabases(ed *entity.DbInstance) ([]string, error) {
	ed.Network = ed.GetNetwork()
	metaDb := ed.Type.MetaDbName()

	dbConn, err := toDbInfo(ed, 0, metaDb, "").Conn()
	if err != nil {
		return nil, err
	}
	defer dbConn.Close()

	return dbConn.GetDialect().GetDbNames()
}
