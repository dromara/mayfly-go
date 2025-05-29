package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/es/application/dto"
	"mayfly-go/internal/es/domain/entity"
	"mayfly-go/internal/es/domain/repository"
	"mayfly-go/internal/es/esm/esi"
	"mayfly-go/internal/es/imsg"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/internal/pkg/consts"
	tagapp "mayfly-go/internal/tag/application"
	tagdto "mayfly-go/internal/tag/application/dto"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/pool"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/utils/structx"
)

type Instance interface {
	base.App[*entity.EsInstance]
	// GetPageList 分页获取数据库实例
	GetPageList(condition *entity.InstanceQuery, orderBy ...string) (*model.PageResult[*entity.EsInstance], error)

	// DoConn 获取连接并执行函数
	DoConn(ctx context.Context, instanceId uint64, fn func(*esi.EsConn) error) error

	TestConn(ctx context.Context, instance *entity.EsInstance, ac *tagentity.ResourceAuthCert) (map[string]any, error)

	SaveInst(ctx context.Context, d *dto.SaveEsInstance) (uint64, error)

	Delete(ctx context.Context, instanceId uint64) error
}

var _ Instance = &instanceAppImpl{}

var poolGroup = pool.NewPoolGroup[*esi.EsConn]()

func init() {
	mcm.AddCheckSshTunnelMachineUseFunc(func(machineId int) bool {
		items := poolGroup.AllPool()
		for _, v := range items {
			conn, err := v.Get(context.Background(), pool.WithGetNoUpdateLastActive(), pool.WithGetNoNewConn())
			if err != nil {
				continue // 获取连接失败，跳过
			}
			if conn.Info.SshTunnelMachineId == machineId {
				return true
			}
		}
		return false
	})
}

type instanceAppImpl struct {
	base.AppImpl[*entity.EsInstance, repository.EsInstance]

	tagApp              tagapp.TagTree          `inject:"T"`
	resourceAuthCertApp tagapp.ResourceAuthCert `inject:"T"`
}

// GetPageList 分页获取数据库实例
func (app *instanceAppImpl) GetPageList(condition *entity.InstanceQuery, orderBy ...string) (*model.PageResult[*entity.EsInstance], error) {
	return app.GetRepo().GetInstanceList(condition, orderBy...)
}

func (app *instanceAppImpl) DoConn(ctx context.Context, instanceId uint64, fn func(*esi.EsConn) error) error {
	p, err := poolGroup.GetCachePool(fmt.Sprintf("es-%d", instanceId), func() (*esi.EsConn, error) {
		return app.createConn(context.Background(), instanceId)
	})

	if err != nil {
		return err
	}
	// 从连接池中获取一个可用的连接
	c, err := p.Get(ctx)
	if err != nil {
		return err
	}

	return fn(c)
}

func (app *instanceAppImpl) createConn(ctx context.Context, instanceId uint64) (*esi.EsConn, error) {
	// 缓存不存在，则重新连接
	instance, err := app.GetById(instanceId)
	if err != nil {
		return nil, errorx.NewBiz("es instance not found")
	}

	ei, err := app.ToEsInfo(instance, nil)
	if err != nil {
		return nil, err
	}
	ei.CodePath = app.tagApp.ListTagPathByTypeAndCode(int8(tagentity.TagTypeEsInstance), instance.Code)

	conn, _, err := ei.Conn(ctx)
	if err != nil {
		return nil, err
	}
	// 缓存连接信息
	return conn, nil
}

func (app *instanceAppImpl) ToEsInfo(instance *entity.EsInstance, ac *tagentity.ResourceAuthCert) (*esi.EsInfo, error) {
	ei := new(esi.EsInfo)
	ei.InstanceId = instance.Id
	structx.Copy(ei, instance)
	ei.OriginUrl = fmt.Sprintf("http://%s:%d", instance.Host, instance.Port)

	if ac != nil {
		if ac.Ciphertext == "" && ac.Name != "" {
			ac1, err := app.resourceAuthCertApp.GetAuthCert(ac.Name)
			if err == nil {
				ac = ac1
			}
		}
	} else {
		if instance.Code != "" {
			ac2, err := app.resourceAuthCertApp.GetResourceAuthCert(tagentity.TagTypeEsInstance, instance.Code)
			if err == nil {
				ac = ac2
			}
		}
	}

	if ac != nil && ac.Ciphertext != "" {
		ei.Username = ac.Username
		ei.Password = ac.Ciphertext
	}

	return ei, nil
}

func (app *instanceAppImpl) TestConn(ctx context.Context, instance *entity.EsInstance, ac *tagentity.ResourceAuthCert) (map[string]any, error) {
	instance.Network = instance.GetNetwork()

	ei, err := app.ToEsInfo(instance, ac)
	if err != nil {
		return nil, err
	}

	_, res, err := ei.Conn(ctx)
	if err != nil {
		return nil, err
	}
	return res, nil

}

func (app *instanceAppImpl) SaveInst(ctx context.Context, instance *dto.SaveEsInstance) (uint64, error) {
	instanceEntity := instance.EsInstance
	// 默认tcp连接
	instanceEntity.Network = instanceEntity.GetNetwork()
	resourceType := consts.ResourceTypeEsInstance
	authCerts := instance.AuthCerts
	tagCodePaths := instance.TagCodePaths

	// 查找是否存在该库
	oldInstance := &entity.EsInstance{
		Host:               instanceEntity.Host,
		Port:               instanceEntity.Port,
		SshTunnelMachineId: instanceEntity.SshTunnelMachineId,
	}

	err := app.GetByCond(oldInstance)
	if instanceEntity.Id == 0 {
		if err == nil {
			return 0, errorx.NewBizI(ctx, imsg.ErrEsInstExist)
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
				ResourceTag:        app.genEsInstanceResourceTag(instanceEntity, authCerts),
				ParentTagCodePaths: tagCodePaths,
			})
		})
	}

	// 如果存在该库，则校验修改的库是否为该库
	if err == nil {
		if oldInstance.Id != instanceEntity.Id {
			return 0, errorx.NewBizI(ctx, imsg.ErrEsInstExist)
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
			ResourceTag:        app.genEsInstanceResourceTag(oldInstance, authCerts),
			ParentTagCodePaths: tagCodePaths,
		})
	})
}

func (app *instanceAppImpl) genEsInstanceResourceTag(ei *entity.EsInstance, authCerts []*tagentity.ResourceAuthCert) *tagdto.ResourceTag {

	// 授权证书对应的tag
	authCertTags := collx.ArrayMap[*tagentity.ResourceAuthCert, *tagdto.ResourceTag](authCerts, func(val *tagentity.ResourceAuthCert) *tagdto.ResourceTag {
		return &tagdto.ResourceTag{
			Code: val.Name,
			Name: val.Username,
			Type: tagentity.TagTypeAuthCert,
		}
	})

	// es实例
	return &tagdto.ResourceTag{
		Code:     ei.Code,
		Name:     ei.Name,
		Type:     tagentity.TagTypeEsInstance,
		Children: authCertTags,
	}
}

func (app *instanceAppImpl) Delete(ctx context.Context, instanceId uint64) error {
	instance, err := app.GetById(instanceId)
	if err != nil {
		return errorx.NewBiz("db instnace not found")
	}

	poolGroup.Close(fmt.Sprintf("es-%d", instanceId))

	return app.Tx(ctx, func(ctx context.Context) error {
		// 删除该实例
		return app.DeleteById(ctx, instanceId)
	}, func(ctx context.Context) error {
		// 删除该实例关联的授权凭证信息
		return app.resourceAuthCertApp.RelateAuthCert(ctx, &tagdto.RelateAuthCert{
			ResourceCode: instance.Code,
			ResourceType: tagentity.TagType(consts.ResourceTypeEsInstance),
		})
	}, func(ctx context.Context) error {
		// 删除该实例关联的tag信息
		return app.tagApp.DeleteTagByParam(ctx, &tagdto.DelResourceTag{
			ResourceCode: instance.Code,
			ResourceType: tagentity.TagType(consts.ResourceTypeEsInstance),
		})
	})
}
