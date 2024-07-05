package application

import (
	"context"
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	tagapp "mayfly-go/internal/tag/application"
	tagdto "mayfly-go/internal/tag/application/dto"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/structx"
)

type Instance interface {
	base.App[*entity.DbInstance]

	// GetPageList 分页获取数据库实例
	GetPageList(condition *entity.InstanceQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	TestConn(instanceEntity *entity.DbInstance, authCert *tagentity.ResourceAuthCert) error

	SaveDbInstance(ctx context.Context, instance *dto.SaveDbInstance) (uint64, error)

	// Delete 删除数据库信息
	Delete(ctx context.Context, id uint64) error

	// GetDatabases 获取数据库实例的所有数据库列表
	GetDatabases(entity *entity.DbInstance, authCert *tagentity.ResourceAuthCert) ([]string, error)

	// GetDatabasesByAc 根据授权凭证名获取所有数据库名称列表
	GetDatabasesByAc(acName string) ([]string, error)

	// ToDbInfo 根据实例与授权凭证返回对应的DbInfo
	ToDbInfo(instance *entity.DbInstance, authCertName string, database string) (*dbi.DbInfo, error)
}

type instanceAppImpl struct {
	base.AppImpl[*entity.DbInstance, repository.Instance]

	tagApp              tagapp.TagTree          `inject:"TagTreeApp"`
	resourceAuthCertApp tagapp.ResourceAuthCert `inject:"ResourceAuthCertApp"`
	dbApp               Db                      `inject:"DbApp"`
	backupApp           *DbBackupApp            `inject:"DbBackupApp"`
	restoreApp          *DbRestoreApp           `inject:"DbRestoreApp"`
}

var _ (Instance) = (*instanceAppImpl)(nil)

// 注入DbInstanceRepo
func (app *instanceAppImpl) InjectDbInstanceRepo(repo repository.Instance) {
	app.Repo = repo
}

// GetPageList 分页获取数据库实例
func (app *instanceAppImpl) GetPageList(condition *entity.InstanceQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.GetRepo().GetInstanceList(condition, pageParam, toEntity, orderBy...)
}

func (app *instanceAppImpl) TestConn(instanceEntity *entity.DbInstance, authCert *tagentity.ResourceAuthCert) error {
	instanceEntity.Network = instanceEntity.GetNetwork()

	authCert, err := app.resourceAuthCertApp.GetRealAuthCert(authCert)
	if err != nil {
		return err
	}

	dbConn, err := dbm.Conn(app.toDbInfoByAc(instanceEntity, authCert, ""))
	if err != nil {
		return err
	}
	dbConn.Close()
	return nil
}

func (app *instanceAppImpl) SaveDbInstance(ctx context.Context, instance *dto.SaveDbInstance) (uint64, error) {
	instanceEntity := instance.DbInstance
	// 默认tcp连接
	instanceEntity.Network = instanceEntity.GetNetwork()
	resourceType := consts.ResourceTypeDb
	authCerts := instance.AuthCerts
	tagCodePaths := instance.TagCodePaths

	if len(authCerts) == 0 {
		return 0, errorx.NewBiz("授权凭证信息不能为空")
	}

	// 查找是否存在该库
	oldInstance := &entity.DbInstance{
		Host:               instanceEntity.Host,
		Port:               instanceEntity.Port,
		SshTunnelMachineId: instanceEntity.SshTunnelMachineId,
	}

	err := app.GetByCond(oldInstance)
	if instanceEntity.Id == 0 {
		if err == nil {
			return 0, errorx.NewBiz("该数据库实例已存在")
		}
		if app.CountByCond(&entity.DbInstance{Code: instanceEntity.Code}) > 0 {
			return 0, errorx.NewBiz("该编码已存在")
		}

		return instanceEntity.Id, app.Tx(ctx, func(ctx context.Context) error {
			return app.Insert(ctx, instanceEntity)
		}, func(ctx context.Context) error {
			return app.resourceAuthCertApp.RelateAuthCert(ctx, &tagdto.RelateAuthCert{
				ResourceCode: instanceEntity.Code,
				ResourceType: tagentity.TagType(resourceType),
				AuthCerts:    authCerts,
			})
		}, func(ctx context.Context) error {
			return app.tagApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{
				ResourceTag:        app.genDbInstanceResourceTag(instanceEntity, authCerts),
				ParentTagCodePaths: tagCodePaths,
			})
		})
	}

	// 如果存在该库，则校验修改的库是否为该库
	if err == nil {
		if oldInstance.Id != instanceEntity.Id {
			return 0, errorx.NewBiz("该数据库实例已存在")
		}
	} else {
		// 根据host等未查到旧数据，则需要根据id重新获取，因为后续需要使用到code
		oldInstance, err = app.GetById(instanceEntity.Id)
		if err != nil {
			return 0, errorx.NewBiz("该数据库实例不存在")
		}
	}

	return oldInstance.Id, app.Tx(ctx, func(ctx context.Context) error {
		return app.UpdateById(ctx, instanceEntity)
	}, func(ctx context.Context) error {
		return app.resourceAuthCertApp.RelateAuthCert(ctx, &tagdto.RelateAuthCert{
			ResourceCode: oldInstance.Code,
			ResourceType: tagentity.TagType(resourceType),
			AuthCerts:    authCerts,
		})
	}, func(ctx context.Context) error {
		if instanceEntity.Name != oldInstance.Name {
			if err := app.tagApp.UpdateTagName(ctx, tagentity.TagTypeDb, oldInstance.Code, instanceEntity.Name); err != nil {
				return err
			}
		}
		return app.tagApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{
			ResourceTag:        app.genDbInstanceResourceTag(instanceEntity, authCerts),
			ParentTagCodePaths: tagCodePaths,
		})
	})
}

func (app *instanceAppImpl) Delete(ctx context.Context, instanceId uint64) error {
	instance, err := app.GetById(instanceId)
	if err != nil {
		return errorx.NewBiz("获取数据库实例错误，数据库实例ID为: %d", instance.Id)
	}

	restore := &entity.DbRestore{
		DbInstanceId: instanceId,
	}
	err = app.restoreApp.restoreRepo.GetByCond(restore)
	if err == nil {
		return errorx.NewBiz("不能删除数据库实例【%s】,请先删除关联的数据库恢复任务。", instance.Name)
	}

	backup := &entity.DbBackup{
		DbInstanceId: instanceId,
	}
	err = app.backupApp.backupRepo.GetByCond(backup)
	if err == nil {
		return errorx.NewBiz("不能删除数据库实例【%s】,请先删除关联的数据库备份任务。", instance.Name)
	}

	dbs, _ := app.dbApp.ListByCond(&entity.Db{
		InstanceId: instanceId,
	})

	return app.Tx(ctx, func(ctx context.Context) error {
		return app.DeleteById(ctx, instanceId)
	}, func(ctx context.Context) error {
		// 删除该实例关联的授权凭证信息
		return app.resourceAuthCertApp.RelateAuthCert(ctx, &tagdto.RelateAuthCert{
			ResourceCode: instance.Code,
			ResourceType: tagentity.TagType(consts.ResourceTypeDb),
		})
	}, func(ctx context.Context) error {
		return app.tagApp.DeleteTagByParam(ctx, &tagdto.DelResourceTag{
			ResourceCode: instance.Code,
			ResourceType: tagentity.TagType(consts.ResourceTypeDb),
		})
	}, func(ctx context.Context) error {
		// 删除所有库配置
		for _, db := range dbs {
			if err := app.dbApp.Delete(ctx, db.Id); err != nil {
				return err
			}
		}
		return nil
	})
}

func (app *instanceAppImpl) GetDatabases(ed *entity.DbInstance, authCert *tagentity.ResourceAuthCert) ([]string, error) {
	if authCert.Id != 0 {
		// 密文可能被清除，故需要重新获取
		authCert, _ = app.resourceAuthCertApp.GetAuthCert(authCert.Name)
	} else {
		if authCert.CiphertextType == tagentity.AuthCertCiphertextTypePublic {
			publicAuthCert, err := app.resourceAuthCertApp.GetAuthCert(authCert.Ciphertext)
			if err != nil {
				return nil, err
			}
			authCert = publicAuthCert
		}
	}

	return app.getDatabases(ed, authCert)
}

func (app *instanceAppImpl) GetDatabasesByAc(acName string) ([]string, error) {
	ac, err := app.resourceAuthCertApp.GetAuthCert(acName)
	if err != nil {
		return nil, errorx.NewBiz("该授权凭证不存在")
	}

	instance := &entity.DbInstance{Code: ac.ResourceCode}
	err = app.GetByCond(instance)
	if err != nil {
		return nil, errorx.NewBiz("不存在该授权凭证对应的数据库实例信息")
	}

	return app.getDatabases(instance, ac)
}

func (app *instanceAppImpl) ToDbInfo(instance *entity.DbInstance, authCertName string, database string) (*dbi.DbInfo, error) {
	ac, err := app.resourceAuthCertApp.GetAuthCert(authCertName)
	if err != nil {
		return nil, err
	}

	return app.toDbInfoByAc(instance, ac, database), nil
}

func (app *instanceAppImpl) getDatabases(instance *entity.DbInstance, ac *tagentity.ResourceAuthCert) ([]string, error) {
	instance.Network = instance.GetNetwork()
	dbi := app.toDbInfoByAc(instance, ac, "")

	dbConn, err := dbm.Conn(dbi)
	if err != nil {
		return nil, err
	}
	defer dbConn.Close()

	return dbConn.GetMetaData().GetDbNames()
}

func (app *instanceAppImpl) toDbInfoByAc(instance *entity.DbInstance, ac *tagentity.ResourceAuthCert, database string) *dbi.DbInfo {
	di := new(dbi.DbInfo)
	di.InstanceId = instance.Id
	di.Database = database
	structx.Copy(di, instance)

	di.Username = ac.Username
	di.Password = ac.Ciphertext
	return di
}

func (m *instanceAppImpl) genDbInstanceResourceTag(me *entity.DbInstance, authCerts []*tagentity.ResourceAuthCert) *tagdto.ResourceTag {
	authCertTags := collx.ArrayMap[*tagentity.ResourceAuthCert, *tagdto.ResourceTag](authCerts, func(val *tagentity.ResourceAuthCert) *tagdto.ResourceTag {
		return &tagdto.ResourceTag{
			Code: val.Name,
			Name: val.Username,
			Type: tagentity.TagTypeDbAuthCert,
		}
	})

	dbs, err := m.dbApp.ListByCond(&entity.Db{
		InstanceId: me.Id,
	})
	if err != nil {
		logx.Errorf("获取实例关联的数据库失败: %v", err)
	}

	authCertName2DbTags := make(map[string][]*tagdto.ResourceTag)
	for _, db := range dbs {
		authCertName2DbTags[db.AuthCertName] = append(authCertName2DbTags[db.AuthCertName], &tagdto.ResourceTag{
			Code: db.Code,
			Name: db.Name,
			Type: tagentity.TagTypeDbName,
		})
	}

	// 将数据库挂至授权凭证下
	for _, ac := range authCertTags {
		ac.Children = authCertName2DbTags[ac.Code]
	}

	return &tagdto.ResourceTag{
		Code:     me.Code,
		Type:     tagentity.TagTypeDb,
		Name:     me.Name,
		Children: authCertTags,
	}
}
