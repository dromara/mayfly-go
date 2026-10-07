package application

import (
	"context"
	"strings"

	"mayfly-go/internal/mongo/domain/entity"
	"mayfly-go/internal/mongo/domain/repository"
	"mayfly-go/internal/mongo/imsg"
	"mayfly-go/internal/mongo/mgm"
	"mayfly-go/internal/pkg/consts"
	tagapp "mayfly-go/internal/tag/application"
	tagdto "mayfly-go/internal/tag/application/dto"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/stringx"
)

type Mongo interface {
	base.App[*entity.Mongo]

	// 分页获取机器脚本信息列表
	GetPageList(condition *entity.MongoQuery, orderBy ...string) (*model.PageResult[*entity.Mongo], error)

	SaveMongo(ctx context.Context, entity *entity.Mongo, tagCodePaths ...string) error

	// 删除数据库信息
	Delete(ctx context.Context, id uint64) error

	// 校验连接配置（含「连得上但没权限」的探测）
	TestConn(ctx context.Context, me *entity.Mongo) error

	// 获取mongo连接实例
	//  -  id mongo id
	GetMongoConn(ctx context.Context, id uint64) (*mgm.MongoConn, error)
}

type mongoAppImpl struct {
	base.AppImpl[*entity.Mongo, repository.Mongo]

	tagTreeApp tagapp.TagTreeService `inject:"T"`
}

var _ Mongo = (*mongoAppImpl)(nil)

// 分页获取数据库信息列表
func (d *mongoAppImpl) GetPageList(condition *entity.MongoQuery, orderBy ...string) (*model.PageResult[*entity.Mongo], error) {
	return d.GetRepo().GetList(condition, orderBy...)
}

func (d *mongoAppImpl) Delete(ctx context.Context, id uint64) error {
	mongoEntity, err := d.GetById(id)
	if err != nil {
		return errorx.NewBizI(ctx, imsg.ErrMongoNotFound)
	}

	mgm.CloseConn(id)
	return d.Tx(ctx,
		func(ctx context.Context) error {
			return d.DeleteById(ctx, id)
		},
		func(ctx context.Context) error {
			return d.tagTreeApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{ResourceTag: &tagdto.ResourceTag{
				Type: tagentity.TagTypeMongo,
				Code: mongoEntity.Code,
			}})
		})
}

// TestConn 校验连接配置。
//
// 除了「能不能连上」，还要能报出「连上了但没权限」：MongoDB 允许匿名 ping，只 ping 会让
// 没配凭证的连接串显示测试通过，用户点开库列表才看到 Unauthorized。
func (d *mongoAppImpl) TestConn(ctx context.Context, me *entity.Mongo) error {
	conn, err := me.ToMongoInfo().Conn()
	if err != nil {
		return err
	}
	defer conn.Close()

	if mgm.HasCredentials(me.Uri) {
		return nil
	}

	// 超时沿用数据面配置（与查询/聚合同一预算），不再另立一个写死的秒数
	probeCtx, cancel, _ := execContext(ctx)
	defer cancel()
	return conn.ProbeAuthorized(probeCtx)
}

// SaveMongo 保存实例。
//
// Uri 的语义按新建/修改区分：新建必须给出连接串；修改时留空表示保持原值不变。
// 后者是因为列表与详情已不再回传明文连接串（内含账号密码），编辑表单无法回填，
// 若仍把留空当作「清空」就会直接连不上，而当作「不传新值」才是用户预期。
func (d *mongoAppImpl) SaveMongo(ctx context.Context, m *entity.Mongo, tagCodePaths ...string) error {
	if m.Id == 0 {
		return d.createMongo(ctx, m, tagCodePaths...)
	}
	return d.updateMongo(ctx, m, tagCodePaths...)
}

func (d *mongoAppImpl) createMongo(ctx context.Context, m *entity.Mongo, tagCodePaths ...string) error {
	if strings.TrimSpace(m.Uri) == "" {
		return errorx.NewBizI(ctx, imsg.ErrMongoUriRequired)
	}

	// 相同连接串 + 相同隧道视为同一实例，避免重复登记
	duplicated := &entity.Mongo{Uri: m.Uri, SshTunnelMachineId: m.SshTunnelMachineId}
	if err := d.GetByCond(duplicated); err == nil {
		return errorx.NewBizI(ctx, imsg.ErrMongoInfoExist)
	}

	// 生成随机编号
	m.Code = stringx.Rand(10)

	return d.Tx(ctx, func(ctx context.Context) error {
		return d.Insert(ctx, m)
	}, func(ctx context.Context) error {
		return d.tagTreeApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{
			ResourceTag: &tagdto.ResourceTag{
				Type: tagentity.TagTypeMongo,
				Code: m.Code,
				Name: m.Name,
			},
			ParentTagCodePaths: tagCodePaths,
		})
	})
}

func (d *mongoAppImpl) updateMongo(ctx context.Context, m *entity.Mongo, tagCodePaths ...string) error {
	oldMongo, err := d.GetById(m.Id)
	if err != nil {
		return errorx.NewBizI(ctx, imsg.ErrMongoNotFound)
	}

	// 只在连接参数真的变了时查重：旧实现无条件按 uri+ssh 查条件，
	// uri 为空时该条件被 GORM 忽略，会把「任意其他实例」误判成重复登记
	if uri := strings.TrimSpace(m.Uri); uri == "" {
		m.Uri = oldMongo.Uri
	} else if uri != oldMongo.Uri || m.SshTunnelMachineId != oldMongo.SshTunnelMachineId {
		duplicated := &entity.Mongo{Uri: uri, SshTunnelMachineId: m.SshTunnelMachineId}
		if err := d.GetByCond(duplicated); err == nil && duplicated.Id != m.Id {
			return errorx.NewBizI(ctx, imsg.ErrMongoInfoExist)
		}
	}

	// 校验当前操作者是否有权操作该资源，防止越权修改他人资源信息
	if err := d.tagTreeApp.CanAccessByCode(ctx, consts.ResourceTypeMongo, oldMongo.Code); err != nil {
		return err
	}

	// code 是资源标签关联的锚点，更新时保持原值；旧实现把它置空后再反查旧记录，多一步且易错
	m.Code = oldMongo.Code

	// 先关闭缓存连接：连接参数可能已变，下一次操作按新配置重连
	mgm.CloseConn(m.Id)

	return d.Tx(ctx, func(ctx context.Context) error {
		return d.UpdateById(ctx, m)
	}, func(ctx context.Context) error {
		if oldMongo.Name != m.Name {
			if err := d.tagTreeApp.UpdateTagName(ctx, tagentity.TagTypeMongo, oldMongo.Code, m.Name); err != nil {
				return err
			}
		}

		return d.tagTreeApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{
			ResourceTag: &tagdto.ResourceTag{
				Type: tagentity.TagTypeMongo,
				Code: oldMongo.Code,
			},
			ParentTagCodePaths: tagCodePaths,
		})
	})
}

func (d *mongoAppImpl) GetMongoConn(ctx context.Context, id uint64) (*mgm.MongoConn, error) {
	// 连接层统一进行数据权限校验，避免各操作接口遗漏鉴权
	me, err := d.GetById(id)
	if err != nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrMongoNotFound)
	}
	if err := d.tagTreeApp.CanAccessByCode(ctx, consts.ResourceTypeMongo, me.Code); err != nil {
		return nil, err
	}

	return mgm.GetMongoConn(ctx, id, func() (*mgm.MongoInfo, error) {
		return me.ToMongoInfo(d.tagTreeApp.ListTagPathByTypeAndCode(consts.ResourceTypeMongo, me.Code)...), nil
	})
}
