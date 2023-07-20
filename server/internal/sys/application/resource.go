package application

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/utils"
	"strings"
	"time"
)

type Resource interface {
	GetResourceList(condition *entity.Resource, toEntity any, orderBy ...string)

	GetById(id uint64, cols ...string) *entity.Resource

	Save(entity *entity.Resource)

	ChangeStatus(resourceId uint64, status int8)

	Sort(re *entity.Resource)

	Delete(id uint64)

	GetAccountResources(accountId uint64, toEntity any)
}

func newResourceApp(resourceRepo repository.Resource) Resource {
	return &resourceAppImpl{
		resourceRepo: resourceRepo,
	}
}

type resourceAppImpl struct {
	resourceRepo repository.Resource
}

func (r *resourceAppImpl) GetResourceList(condition *entity.Resource, toEntity any, orderBy ...string) {
	r.resourceRepo.GetResourceList(condition, toEntity, orderBy...)
}

func (r *resourceAppImpl) GetById(id uint64, cols ...string) *entity.Resource {
	return r.resourceRepo.GetById(id, cols...)
}

func (r *resourceAppImpl) Save(resource *entity.Resource) {
	// 更新操作
	if resource.Id != 0 {
		if resource.Code != "" {
			oldRes := r.GetById(resource.Id, "Code")
			// 如果修改了code，则校验新code是否存在
			if oldRes.Code != resource.Code {
				r.checkCode(resource.Code)
			}
		}
		gormx.UpdateById(resource)
		return
	}

	// 生成随机八位唯一标识符
	ui := utils.RandString(8)
	if pid := resource.Pid; pid != 0 {
		pResource := r.GetById(uint64(pid))
		biz.IsTrue(pResource != nil, "该父资源不存在")
		resource.UiPath = pResource.UiPath + ui + entity.ResourceUiPathSp
	} else {
		resource.UiPath = ui + entity.ResourceUiPathSp
	}
	// 默认启用状态
	if resource.Status == 0 {
		resource.Status = entity.ResourceStatusEnable
	}
	r.checkCode(resource.Code)
	resource.Weight = int(time.Now().Unix())
	gormx.Insert(resource)
}

func (r *resourceAppImpl) ChangeStatus(resourceId uint64, status int8) {
	resource := r.resourceRepo.GetById(resourceId)
	biz.NotNil(resource, "资源不存在")
	resource.Status = status
	r.resourceRepo.UpdateByUiPathLike(resource)
}

func (r *resourceAppImpl) Sort(sortResource *entity.Resource) {
	resource := r.resourceRepo.GetById(sortResource.Id)
	biz.NotNil(resource, "资源不存在")
	// 未改变父节点，则更新排序值即可
	if sortResource.Pid == resource.Pid {
		saveE := &entity.Resource{Weight: sortResource.Weight}
		saveE.Id = sortResource.Id
		r.Save(saveE)
		return
	}

	// 若资源原本唯一标识路径为：xxxx/yyyy/zzzz/，则获取其父节点路径标识 xxxx/yyyy/ 与自身节点标识 zzzz/
	splitStr := strings.Split(resource.UiPath, entity.ResourceUiPathSp)
	// 获取 zzzz/
	resourceUi := splitStr[len(splitStr)-2] + entity.ResourceUiPathSp
	// 获取父资源路径 xxxx/yyyy/
	var parentResourceUiPath string
	if len(splitStr) > 2 {
		parentResourceUiPath = strings.Split(resource.UiPath, resourceUi)[0]
	} else {
		parentResourceUiPath = resourceUi
	}

	newParentResourceUiPath := ""
	if sortResource.Pid != 0 {
		newParentResource := r.resourceRepo.GetById(uint64(sortResource.Pid))
		biz.NotNil(newParentResource, "父资源不存在")
		newParentResourceUiPath = newParentResource.UiPath
	}

	children := r.resourceRepo.GetChildren(resource.UiPath)
	for _, v := range children {
		if v.Id == sortResource.Id {
			continue
		}
		updateUiPath := &entity.Resource{}
		updateUiPath.Id = v.Id
		if parentResourceUiPath == resourceUi {
			updateUiPath.UiPath = newParentResourceUiPath + v.UiPath
		} else {
			updateUiPath.UiPath = strings.ReplaceAll(v.UiPath, parentResourceUiPath, newParentResourceUiPath)
		}
		r.Save(updateUiPath)
	}

	// 更新零值使用map，因为pid=0表示根节点
	updateMap := map[string]interface{}{
		"pid":     sortResource.Pid,
		"weight":  sortResource.Weight,
		"ui_path": newParentResourceUiPath + resourceUi,
	}
	condition := new(entity.Resource)
	condition.Id = sortResource.Id
	gormx.Updates(condition, updateMap)
}

func (r *resourceAppImpl) checkCode(code string) {
	biz.IsTrue(!strings.Contains(code, ","), "code不能包含','")
	biz.IsEquals(gormx.CountBy(&entity.Resource{Code: code}), int64(0), "该code已存在")
}

func (r *resourceAppImpl) Delete(id uint64) {
	resource := r.resourceRepo.GetById(id)
	biz.NotNil(resource, "资源不存在")

	// 删除当前节点及其所有子节点
	children := r.resourceRepo.GetChildren(resource.UiPath)
	for _, v := range children {
		r.resourceRepo.Delete(v.Id)
		// 删除角色关联的资源信息
		gormx.DeleteByCondition(&entity.RoleResource{ResourceId: v.Id})
	}
}

func (r *resourceAppImpl) GetAccountResources(accountId uint64, toEntity any) {
	r.resourceRepo.GetAccountResources(accountId, toEntity)
}
