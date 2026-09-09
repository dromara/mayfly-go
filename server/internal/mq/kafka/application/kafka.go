package application

import (
	"context"
	"mayfly-go/internal/mq/kafka/domain/entity"
	"mayfly-go/internal/mq/kafka/domain/repository"
	"mayfly-go/internal/mq/kafka/imsg"
	"mayfly-go/internal/mq/kafka/kfm"
	"mayfly-go/internal/pkg/consts"
	tagapp "mayfly-go/internal/tag/application"
	tagdto "mayfly-go/internal/tag/application/dto"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/stringx"
)

type Kafka interface {
	base.App[*entity.Kafka]

	// 分页获取机器脚本信息列表
	GetPageList(condition *entity.KafkaQuery, orderBy ...string) (*model.PageResult[*entity.Kafka], error)

	TestConn(entity *entity.Kafka) error

	SaveKafka(ctx context.Context, entity *entity.Kafka, tagCodePaths ...string) error

	// 删除数据库信息
	Delete(ctx context.Context, id uint64) error

	// 获取Kafka连接实例
	//  -  id Kafka id
	GetKafkaConn(ctx context.Context, id uint64) (*kfm.KafkaConn, error)
}

type kafkaAppImpl struct {
	base.AppImpl[*entity.Kafka, repository.Kafka]

	tagTreeApp tagapp.TagTreeService `inject:"T"`
}

var _ Kafka = (*kafkaAppImpl)(nil)

// 分页获取数据库信息列表
func (d *kafkaAppImpl) GetPageList(condition *entity.KafkaQuery, orderBy ...string) (*model.PageResult[*entity.Kafka], error) {
	return d.GetRepo().GetList(condition, orderBy...)
}

func (d *kafkaAppImpl) Delete(ctx context.Context, id uint64) error {
	kafkaEntity, err := d.GetById(id)
	if err != nil {
		return errorx.NewBiz("kafka not found")
	}

	kfm.CloseConn(id)
	return d.Tx(ctx,
		func(ctx context.Context) error {
			return d.DeleteById(ctx, id)
		},
		func(ctx context.Context) error {
			return d.tagTreeApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{ResourceTag: &tagdto.ResourceTag{
				Type: tagentity.TagTypeMqKafka,
				Code: kafkaEntity.Code,
			}})
		})
}

func (d *kafkaAppImpl) TestConn(me *entity.Kafka) error {
	conn, err := me.ToKafkaInfo().Conn()
	if err != nil {
		return err
	}
	_, err = conn.GetTopics()
	if err != nil {
		return err
	}
	return nil
}

func (d *kafkaAppImpl) SaveKafka(ctx context.Context, m *entity.Kafka, tagCodePaths ...string) error {
	oldKafka := &entity.Kafka{Hosts: m.Hosts, SshTunnelMachineId: m.SshTunnelMachineId}
	err := d.GetByCond(oldKafka)

	if m.Id == 0 {
		if err == nil {
			return errorx.NewBizI(ctx, imsg.ErrKafkaInfoExist)
		}
		// 生成随机编号
		m.Code = stringx.Rand(10)

		return d.Tx(ctx,
			func(ctx context.Context) error {
				return d.Insert(ctx, m)
			},
			func(ctx context.Context) error {
				return d.tagTreeApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{
					ResourceTag: &tagdto.ResourceTag{
						Type: tagentity.TagTypeMqKafka,
						Code: m.Code,
						Name: m.Name,
					},
					ParentTagCodePaths: tagCodePaths,
				})
			})
	}

	// 如果存在该库，则校验修改的库是否为该库
	if err == nil && oldKafka.Id != m.Id {
		return errorx.NewBizI(ctx, imsg.ErrKafkaInfoExist)
	}
	// 如果调整了ssh等会查不到旧数据，故需要根据id获取旧信息将code赋值给标签进行关联
	if oldKafka.Code == "" {
		oldKafka, _ = d.GetById(m.Id)
	}

	// 校验当前操作者是否有权操作该资源，防止越权修改他人资源信息
	if err := d.tagTreeApp.CanAccessByCode(ctx, consts.ResourceTypeMqKafka, oldKafka.Code); err != nil {
		return err
	}

	// 先关闭连接
	kfm.CloseConn(m.Id)
	m.Code = ""
	return d.Tx(ctx, func(ctx context.Context) error {
		return d.UpdateById(ctx, m)
	}, func(ctx context.Context) error {
		if oldKafka.Name != m.Name {
			if err := d.tagTreeApp.UpdateTagName(ctx, tagentity.TagTypeMqKafka, oldKafka.Code, m.Name); err != nil {
				return err
			}
		}

		return d.tagTreeApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{
			ResourceTag: &tagdto.ResourceTag{
				Type: tagentity.TagTypeMqKafka,
				Code: oldKafka.Code,
			},
			ParentTagCodePaths: tagCodePaths,
		})
	})
}

func (d *kafkaAppImpl) GetKafkaConn(ctx context.Context, id uint64) (*kfm.KafkaConn, error) {
	// 连接层统一进行数据权限校验，避免各操作接口遗漏鉴权
	me, err := d.GetById(id)
	if err != nil {
		return nil, errorx.NewBiz("kafka not found")
	}
	if err := d.tagTreeApp.CanAccessByCode(ctx, consts.ResourceTypeMqKafka, me.Code); err != nil {
		return nil, err
	}

	return kfm.GetKafkaConn(ctx, id, func() (*kfm.KafkaInfo, error) {
		return me.ToKafkaInfo(), nil
	})
}
