package application

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type Instance interface {
	// 分页获取
	GetPageList(condition *entity.InstanceQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any]

	Count(condition *entity.InstanceQuery) int64

	// 根据条件获取
	GetInstanceBy(condition *entity.Instance, cols ...string) error

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.Instance

	Save(instanceEntity *entity.Instance)

	// 删除数据库信息
	Delete(id uint64)
}

func newInstanceApp(InstanceRepo repository.Instance) Instance {
	return &instanceAppImpl{
		instanceRepo: InstanceRepo,
	}
}

type instanceAppImpl struct {
	instanceRepo repository.Instance
}

// 分页获取数据库信息列表
func (d *instanceAppImpl) GetPageList(condition *entity.InstanceQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any] {
	return d.instanceRepo.GetInstanceList(condition, pageParam, toEntity, orderBy...)
}

func (d *instanceAppImpl) Count(condition *entity.InstanceQuery) int64 {
	return d.instanceRepo.Count(condition)
}

// 根据条件获取
func (d *instanceAppImpl) GetInstanceBy(condition *entity.Instance, cols ...string) error {
	return d.instanceRepo.GetInstance(condition, cols...)
}

// 根据id获取
func (d *instanceAppImpl) GetById(id uint64, cols ...string) *entity.Instance {
	return d.instanceRepo.GetById(id, cols...)
}

func (d *instanceAppImpl) Save(instanceEntity *entity.Instance) {
	// 默认tcp连接
	instanceEntity.Network = instanceEntity.GetNetwork()

	// 测试连接
	// todo 测试数据库连接
	//if instanceEntity.Password != "" {
	//	TestConnection(instanceEntity)
	//}

	// 查找是否存在该库
	oldInstance := &entity.Instance{Host: instanceEntity.Host, Port: instanceEntity.Port, Username: instanceEntity.Username}
	if instanceEntity.SshTunnelMachineId > 0 {
		oldInstance.SshTunnelMachineId = instanceEntity.SshTunnelMachineId
	}

	err := d.GetInstanceBy(oldInstance)
	if instanceEntity.Id == 0 {
		biz.NotEmpty(instanceEntity.Password, "密码不能为空")
		biz.IsTrue(err != nil, "该数据库实例已存在")
		instanceEntity.PwdEncrypt()
		d.instanceRepo.Insert(instanceEntity)
	} else {
		// 如果存在该库，则校验修改的库是否为该库
		if err == nil {
			biz.IsTrue(oldInstance.Id == instanceEntity.Id, "该数据库实例已存在")
		}
		instanceEntity.PwdEncrypt()
		d.instanceRepo.Update(instanceEntity)
	}
}

func (d *instanceAppImpl) Delete(id uint64) {
	// todo 删除数据库库实例前必须删除关联数据库
	d.instanceRepo.Delete(id)
}
