package application

import (
	"context"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/internal/pkg/consts"
	tagapp "mayfly-go/internal/tag/application"
	tagdto "mayfly-go/internal/tag/application/dto"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/utils/structx"
)

type Instance interface {
	base.App[*entity.DbInstance]

	// GetPageList 分页获取数据库实例
	GetPageList(condition *entity.InstanceQuery, orderBy ...string) (*model.PageResult[*entity.DbInstance], error)

	TestConn(ctx context.Context, instanceEntity *entity.DbInstance, authCert *tagentity.ResourceAuthCert) error

	SaveDbInstance(ctx context.Context, instance *dto.SaveDbInstance) (uint64, error)

	// Delete 删除数据库信息
	Delete(ctx context.Context, id uint64) error

	// GetDatabases 获取数据库实例的所有数据库列表
	GetDatabases(ctx context.Context, entity *entity.DbInstance, authCert *tagentity.ResourceAuthCert) ([]string, error)

	// GetDatabasesByAc 根据授权凭证名获取所有数据库名称列表
	GetDatabasesByAc(ctx context.Context, acName string) ([]string, error)

	// ToDbInfo 根据实例与授权凭证返回对应的DbInfo
	ToDbInfo(instance *entity.DbInstance, authCertName string, database string) (*dbi.DbInfo, error)
}

type instanceAppImpl struct {
	base.AppImpl[*entity.DbInstance, repository.Instance]

	tagApp              tagapp.TagTree          `inject:"T"`
	resourceAuthCertApp tagapp.ResourceAuthCert `inject:"T"`
	dbApp               Db                      `inject:"T"`
}

var _ (Instance) = (*instanceAppImpl)(nil)

// GetPageList 分页获取数据库实例
func (app *instanceAppImpl) GetPageList(condition *entity.InstanceQuery, orderBy ...string) (*model.PageResult[*entity.DbInstance], error) {
	return app.GetRepo().GetInstanceList(condition, orderBy...)
}

func (app *instanceAppImpl) TestConn(ctx context.Context, instanceEntity *entity.DbInstance, authCert *tagentity.ResourceAuthCert) error {
	instanceEntity.Network = instanceEntity.GetNetwork()

	authCert, err := app.resourceAuthCertApp.GetRealAuthCert(authCert)
	if err != nil {
		return err
	}

	dbConn, err := dbm.Conn(ctx, app.toDbInfoByAc(instanceEntity, authCert, ""))
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
	resourceType := consts.ResourceTypeDbInstance
	authCerts := instance.AuthCerts
	tagCodePaths := instance.TagCodePaths

	if len(authCerts) == 0 {
		return 0, errorx.NewBiz("ac cannot be empty")
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
			return 0, errorx.NewBizI(ctx, imsg.ErrDbInstExist)
		}
		instanceEntity.Code = stringx.Rand(10)

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
			return 0, errorx.NewBizI(ctx, imsg.ErrDbInstExist)
		}
	} else {
		// 根据host等未查到旧数据，则需要根据id重新获取，因为后续需要使用到code
		oldInstance, err = app.GetById(instanceEntity.Id)
		if err != nil {
			return 0, errorx.NewBiz("db instance not found")
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
			if err := app.tagApp.UpdateTagName(ctx, tagentity.TagTypeDbInstance, oldInstance.Code, instanceEntity.Name); err != nil {
				return err
			}
		}
		return app.tagApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{
			ResourceTag:        app.genDbInstanceResourceTag(oldInstance, authCerts),
			ParentTagCodePaths: tagCodePaths,
		})
	})
}

func (app *instanceAppImpl) Delete(ctx context.Context, instanceId uint64) error {
	instance, err := app.GetById(instanceId)
	if err != nil {
		return errorx.NewBiz("db instnace not found")
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
			ResourceType: tagentity.TagType(consts.ResourceTypeDbInstance),
		})
	}, func(ctx context.Context) error {
		return app.tagApp.DeleteTagByParam(ctx, &tagdto.DelResourceTag{
			ResourceCode: instance.Code,
			ResourceType: tagentity.TagType(consts.ResourceTypeDbInstance),
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

func (app *instanceAppImpl) GetDatabases(ctx context.Context, ed *entity.DbInstance, authCert *tagentity.ResourceAuthCert) ([]string, error) {
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

	return app.getDatabases(ctx, ed, authCert)
}

func (app *instanceAppImpl) GetDatabasesByAc(ctx context.Context, acName string) ([]string, error) {
	ac, err := app.resourceAuthCertApp.GetAuthCert(acName)
	if err != nil {
		return nil, errorx.NewBiz("db ac not found")
	}

	instance := &entity.DbInstance{Code: ac.ResourceCode}
	err = app.GetByCond(instance)
	if err != nil {
		return nil, errorx.NewBiz("the db instance information for this ac does not exist")
	}

	return app.getDatabases(ctx, instance, ac)
}

func (app *instanceAppImpl) ToDbInfo(instance *entity.DbInstance, authCertName string, database string) (*dbi.DbInfo, error) {
	ac, err := app.resourceAuthCertApp.GetAuthCert(authCertName)
	if err != nil {
		return nil, err
	}

	return app.toDbInfoByAc(instance, ac, database), nil
}

func (app *instanceAppImpl) getDatabases(ctx context.Context, instance *entity.DbInstance, ac *tagentity.ResourceAuthCert) ([]string, error) {
	instance.Network = instance.GetNetwork()
	dbi := app.toDbInfoByAc(instance, ac, "")

	dbConn, err := dbm.Conn(ctx, dbi)
	if err != nil {
		return nil, err
	}
	defer dbConn.Close()

	return dbConn.GetMetadata().GetDbNames()
}

func (app *instanceAppImpl) toDbInfoByAc(instance *entity.DbInstance, ac *tagentity.ResourceAuthCert, database string) *dbi.DbInfo {
	di := new(dbi.DbInfo)
	di.InstanceId = instance.Id
	di.Database = database
	structx.Copy(di, instance)
	di.Extra = instance.Extra

	di.Username = ac.Username
	di.Password = ac.Ciphertext
	return di
}

func (m *instanceAppImpl) genDbInstanceResourceTag(me *entity.DbInstance, authCerts []*tagentity.ResourceAuthCert) *tagdto.ResourceTag {
	authCertTags := collx.ArrayMap[*tagentity.ResourceAuthCert, *tagdto.ResourceTag](authCerts, func(val *tagentity.ResourceAuthCert) *tagdto.ResourceTag {
		return &tagdto.ResourceTag{
			Code: val.Name,
			Name: val.Username,
			Type: tagentity.TagTypeAuthCert,
		}
	})

	dbs, err := m.dbApp.ListByCond(&entity.Db{
		InstanceId: me.Id,
	})
	if err != nil {
		logx.Errorf("failed to retrieve the database associated with the instance: %v", err)
	}

	authCertName2DbTags := make(map[string][]*tagdto.ResourceTag)
	for _, db := range dbs {
		authCertName2DbTags[db.AuthCertName] = append(authCertName2DbTags[db.AuthCertName], &tagdto.ResourceTag{
			Code: db.Code,
			Name: db.Name,
			Type: tagentity.TagTypeDb,
		})
	}

	// 将数据库挂至授权凭证下
	for _, ac := range authCertTags {
		ac.Children = authCertName2DbTags[ac.Code]
	}

	return &tagdto.ResourceTag{
		Code:     me.Code,
		Type:     tagentity.TagTypeDbInstance,
		Name:     me.Name,
		Children: authCertTags,
	}
}
