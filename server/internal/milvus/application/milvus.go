package application

import (
	"context"
	"mayfly-go/internal/milvus/domain/entity"
	"mayfly-go/internal/milvus/domain/repository"
	"mayfly-go/internal/milvus/imsg"
	"mayfly-go/internal/milvus/mvm"
	"mayfly-go/internal/pkg/consts"
	tagapp "mayfly-go/internal/tag/application"
	tagdto "mayfly-go/internal/tag/application/dto"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
)

type Milvus interface {
	base.App[*entity.Milvus]

	// 分页获取机器脚本信息列表
	GetPageList(condition *entity.MilvusQuery, orderBy ...string) (*model.PageResult[*entity.Milvus], error)

	TestConn(ctx context.Context, entity *entity.Milvus, authCert *tagentity.ResourceAuthCert) error

	SaveMilvus(ctx context.Context, entity *entity.Milvus, authCerts []*tagentity.ResourceAuthCert, tagCodePaths ...string) error

	Delete(ctx context.Context, id uint64) error

	// 获取Milvus连接实例
	GetMilvusConn(rc *req.Ctx) (*mvm.MilvusConn, error)
}

// MilvusApp Milvus 应用服务
type milvusAppImpl struct {
	base.AppImpl[*entity.Milvus, repository.Milvus]
	tagTreeApp          tagapp.TagTreeService   `inject:"T"`
	resourceAuthCertApp tagapp.ResourceAuthCert `inject:"T"`
}

var _ Milvus = (*milvusAppImpl)(nil)

// GetPageList 获取分页列表
func (a *milvusAppImpl) GetPageList(query *entity.MilvusQuery, orderBy ...string) (*model.PageResult[*entity.Milvus], error) {
	return a.GetRepo().GetList(query, orderBy...)
}

// TestConn 测试连接
func (a *milvusAppImpl) TestConn(ctx context.Context, milvus *entity.Milvus, authCert *tagentity.ResourceAuthCert) error {
	mi := milvus.ToMilvusInfo()
	realAc, err := a.resourceAuthCertApp.GetRealAuthCert(authCert)
	if err != nil {
		return err
	}
	mi.Username = realAc.Username
	mi.Password = realAc.Ciphertext

	conn, err := mi.Conn()
	if err != nil {
		return err
	}
	// 尝试获取数据库列表
	_, err = conn.ListDatabases()
	if err != nil {
		return err
	}
	return nil
}

// SaveMilvus 保存 Milvus 实例
func (a *milvusAppImpl) SaveMilvus(ctx context.Context, m *entity.Milvus, authCerts []*tagentity.ResourceAuthCert, tagCodePaths ...string) error {
	resourceType := consts.ResourceTypeMilvus

	if len(authCerts) == 0 {
		return errorx.NewBiz("ac cannot be empty")
	}

	old := &entity.Milvus{Host: m.Host, SshTunnelMachineId: m.SshTunnelMachineId}
	err := a.GetByCond(old)

	if m.Id == 0 {
		if err == nil {
			return errorx.NewBizI(ctx, imsg.ErrInfoExist)
		}
		// 生成随机编号
		m.Code = stringx.Rand(10)

		return a.Tx(ctx,
			func(ctx context.Context) error {
				return a.Insert(ctx, m)
			},
			func(ctx context.Context) error {
				return a.resourceAuthCertApp.RelateAuthCert(ctx, &tagdto.RelateAuthCert{
					ResourceCode: m.Code,
					ResourceType: tagentity.TagType(resourceType),
					AuthCerts:    authCerts,
				})
			},
			func(ctx context.Context) error {
				return a.tagTreeApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{
					ResourceTag:        a.genMilvusResourceTag(m, authCerts),
					ParentTagCodePaths: tagCodePaths,
				})
			})
	}

	// 如果存在该库，则校验修改的库是否为该库
	if err == nil && old.Id != m.Id {
		return errorx.NewBizI(ctx, imsg.ErrInfoExist)
	}
	// 如果调整了ssh等会查不到旧数据，故需要根据id获取旧信息将code赋值给标签进行关联
	if old.Code == "" {
		old, _ = a.GetById(m.Id)
	}

	// 校验当前操作者是否有权操作该资源，防止越权修改他人资源信息
	if err := a.tagTreeApp.CanAccessByCode(ctx, resourceType, old.Code); err != nil {
		return err
	}

	// 先关闭连接
	mvm.CloseAll(m.Id)
	m.Code = ""
	return a.Tx(ctx, func(ctx context.Context) error {
		return a.UpdateById(ctx, m)
	}, func(ctx context.Context) error {
		return a.resourceAuthCertApp.RelateAuthCert(ctx, &tagdto.RelateAuthCert{
			ResourceCode: old.Code,
			ResourceType: tagentity.TagType(resourceType),
			AuthCerts:    authCerts,
		})
	}, func(ctx context.Context) error {
		if old.Name != m.Name {
			if err := a.tagTreeApp.UpdateTagName(ctx, tagentity.TagTypeMilvus, old.Code, m.Name); err != nil {
				return err
			}
		}
		return a.tagTreeApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{
			ResourceTag:        a.genMilvusResourceTag(old, authCerts),
			ParentTagCodePaths: tagCodePaths,
		})
	})
}

// Delete 删除 Milvus 实例
func (a *milvusAppImpl) Delete(ctx context.Context, id uint64) error {
	milvusEntity, err := a.GetById(id)
	if err != nil {
		return errorx.NewBiz("milvus not found")
	}

	mvm.CloseAll(milvusEntity.Id)
	return a.Tx(ctx,
		func(ctx context.Context) error {
			return a.DeleteById(ctx, id)
		},
		func(ctx context.Context) error {
			return a.resourceAuthCertApp.RelateAuthCert(ctx, &tagdto.RelateAuthCert{
				ResourceCode: milvusEntity.Code,
				ResourceType: tagentity.TagType(consts.ResourceTypeMilvus),
			})
		},
		func(ctx context.Context) error {
			return a.tagTreeApp.DeleteTagByParam(ctx, &tagdto.DelResourceTag{
				ResourceCode: milvusEntity.Code,
				ResourceType: tagentity.TagType(consts.ResourceTypeMilvus),
			})
		})
}

// GetMilvusClient 获取 Milvus 客户端
func (a *milvusAppImpl) GetMilvusConn(rc *req.Ctx) (*mvm.MilvusConn, error) {
	id := rc.PathParamInt("id")
	biz.IsTrue(id > 0, "milvusId error")

	db := rc.Query("db")
	// 连接层统一进行数据权限校验，避免各操作接口遗漏鉴权
	milvusEntity, err := a.GetById(uint64(id))
	if err != nil {
		return nil, err
	}
	if err := a.tagTreeApp.CanAccessByCode(rc.MetaCtx, consts.ResourceTypeMilvus, milvusEntity.Code); err != nil {
		return nil, err
	}

	if db == "" {
		m, err := a.GetById(uint64(id))
		if err != nil {
			return nil, err
		}
		db = m.Database
	}
	if db == "" {
		db = "default"
	}
	// 读取 ac（授权凭证名），用于缓存键区分和连接凭证选择
	acName := rc.Query("ac")
	return mvm.GetMilvusConn(rc, uint64(id), db, acName, func() (*mvm.MilvusInfo, error) {
		me, err := a.GetById(uint64(id))
		if err != nil {
			return nil, errorx.NewBiz("milvus not found")
		}
		me.Database = db
		mi := me.ToMilvusInfo()

		var ac *tagentity.ResourceAuthCert
		if acName != "" {
			ac, err = a.resourceAuthCertApp.GetAuthCert(acName)
			if err != nil {
				return nil, errorx.NewBizf("auth cert [%s] not found", acName)
			}
		} else {
			// 未指定则使用默认授权凭证
			ac, err = a.resourceAuthCertApp.GetResourceAuthCert(tagentity.TagType(consts.ResourceTypeMilvus), me.Code)
			if err != nil {
				return nil, err
			}
		}
		mi.Username = ac.Username
		mi.Password = ac.Ciphertext

		return mi, nil
	})
}

// genMilvusResourceTag 生成 Milvus 实例的资源标签
func (a *milvusAppImpl) genMilvusResourceTag(me *entity.Milvus, authCerts []*tagentity.ResourceAuthCert) *tagdto.ResourceTag {
	authCertTags := collx.ArrayMap[*tagentity.ResourceAuthCert, *tagdto.ResourceTag](authCerts, func(val *tagentity.ResourceAuthCert) *tagdto.ResourceTag {
		return &tagdto.ResourceTag{
			Code: val.Name,
			Name: val.Username,
			Type: tagentity.TagTypeAuthCert,
		}
	})

	return &tagdto.ResourceTag{
		Code:     me.Code,
		Name:     me.Name,
		Type:     tagentity.TagTypeMilvus,
		Children: authCertTags,
	}
}
