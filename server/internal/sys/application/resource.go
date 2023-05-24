package application

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"strings"
)

type Resource interface {
	GetResourceList(condition *entity.Resource, toEntity interface{}, orderBy ...string)

	GetById(id uint64, cols ...string) *entity.Resource

	GetByIdIn(ids []uint64, toEntity interface{}, cols ...string)

	Save(entity *entity.Resource)

	Delete(id uint64)

	GetAccountResources(accountId uint64, toEntity interface{})
}

func newResourceApp(resourceRepo repository.Resource) Resource {
	return &resourceAppImpl{
		resourceRepo: resourceRepo,
	}
}

type resourceAppImpl struct {
	resourceRepo repository.Resource
}

func (r *resourceAppImpl) GetResourceList(condition *entity.Resource, toEntity interface{}, orderBy ...string) {
	r.resourceRepo.GetResourceList(condition, toEntity, orderBy...)
}

func (r *resourceAppImpl) GetById(id uint64, cols ...string) *entity.Resource {
	return r.resourceRepo.GetById(id, cols...)
}

func (r *resourceAppImpl) GetByIdIn(ids []uint64, toEntity interface{}, orderBy ...string) {
	r.resourceRepo.GetByIdIn(ids, toEntity, orderBy...)
}

func (r *resourceAppImpl) Save(resource *entity.Resource) {
	if resource.Id != 0 {
		if resource.Code != "" {
			oldRes := r.GetById(resource.Id, "Code")
			// 如果修改了code，则校验新code是否存在
			if oldRes.Code != resource.Code {
				r.checkCode(resource.Code)
			}
		}
		model.UpdateById(resource)
	} else {
		if pid := resource.Pid; pid != 0 {
			biz.IsTrue(r.GetById(uint64(pid)) != nil, "该父资源不存在")
		}
		// 默认启用状态
		if resource.Status == 0 {
			resource.Status = entity.ResourceStatusEnable
		}
		r.checkCode(resource.Code)
		model.Insert(resource)
	}
}

func (r *resourceAppImpl) checkCode(code string) {
	biz.IsTrue(!strings.Contains(code, ","), "code不能包含','")
	biz.IsEquals(model.CountBy(&entity.Resource{Code: code}), int64(0), "该code已存在")
}

func (r *resourceAppImpl) Delete(id uint64) {
	// 查找pid == id的资源
	condition := &entity.Resource{Pid: int(id)}
	var resources resourceList
	r.resourceRepo.GetResourceList(condition, &resources)

	biz.IsTrue(len(resources) == 0, "请先删除该资源的所有子资源")
	model.DeleteById(condition, id)
	// 删除角色关联的资源信息
	model.DeleteByCondition(&entity.RoleResource{ResourceId: id})
}

func (r *resourceAppImpl) GetAccountResources(accountId uint64, toEntity interface{}) {
	r.resourceRepo.GetAccountResources(accountId, toEntity)
}

type resourceList []entity.Resource
