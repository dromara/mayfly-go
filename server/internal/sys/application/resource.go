package application

import (
	"context"
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/utils/stringx"
	"strings"
	"time"
)

type Resource interface {
	base.App[*entity.Resource]

	Save(ctx context.Context, entity *entity.Resource) error

	Delete(ctx context.Context, id uint64) error

	ChangeStatus(ctx context.Context, resourceId uint64, status int8) error

	Sort(ctx context.Context, re *entity.Resource) error

	GetAccountResources(accountId uint64, toEntity any) error
}

type resourceAppImpl struct {
	base.AppImpl[*entity.Resource, repository.Resource]
}

// 注入ResourceRepo
func (r *resourceAppImpl) InjectResourceRepo(repo repository.Resource) {
	r.Repo = repo
}

func (r *resourceAppImpl) Save(ctx context.Context, resource *entity.Resource) error {
	// 更新操作
	if resource.Id != 0 {
		if resource.Code != "" {
			oldRes, err := r.GetById(new(entity.Resource), resource.Id, "Code")
			if err != nil {
				return errorx.NewBiz("更新失败, 该资源不存在")
			}
			// 如果修改了code，则校验新code是否存在
			if oldRes.Code != resource.Code {
				if err := r.checkCode(resource.Code); err != nil {
					return err
				}
			}
		}
		return r.UpdateById(ctx, resource)
	}

	// 生成随机八位唯一标识符
	ui := stringx.Rand(8)
	if pid := resource.Pid; pid != 0 {
		pResource, err := r.GetById(new(entity.Resource), uint64(pid))
		if err != nil {
			return errorx.NewBiz("该父资源不存在")
		}
		resource.UiPath = pResource.UiPath + ui + entity.ResourceUiPathSp
	} else {
		resource.UiPath = ui + entity.ResourceUiPathSp
	}
	// 默认启用状态
	if resource.Status == 0 {
		resource.Status = entity.ResourceStatusEnable
	}
	if err := r.checkCode(resource.Code); err != nil {
		return err
	}
	resource.Weight = int(time.Now().Unix())
	return r.Insert(ctx, resource)
}

func (r *resourceAppImpl) ChangeStatus(ctx context.Context, resourceId uint64, status int8) error {
	resource, err := r.GetById(new(entity.Resource), resourceId)
	if err != nil {
		return errorx.NewBiz("资源不存在")
	}
	resource.Status = status
	return r.GetRepo().UpdateByUiPathLike(resource)
}

func (r *resourceAppImpl) Sort(ctx context.Context, sortResource *entity.Resource) error {
	resource, err := r.GetById(new(entity.Resource), sortResource.Id)
	if err != nil {
		return errorx.NewBiz("资源不存在")
	}

	// 未改变父节点，则更新排序值即可
	if sortResource.Pid == resource.Pid {
		saveE := &entity.Resource{Weight: sortResource.Weight}
		saveE.Id = sortResource.Id
		return r.Save(ctx, saveE)
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
		newParentResource, err := r.GetById(new(entity.Resource), uint64(sortResource.Pid))
		if err != nil {
			return errorx.NewBiz("父资源不存在")
		}
		newParentResourceUiPath = newParentResource.UiPath
	}

	children := r.GetRepo().GetChildren(resource.UiPath)
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
		r.Save(ctx, updateUiPath)
	}

	// 更新零值使用map，因为pid=0表示根节点
	updateMap := map[string]interface{}{
		"pid":     sortResource.Pid,
		"weight":  sortResource.Weight,
		"ui_path": newParentResourceUiPath + resourceUi,
	}
	condition := new(entity.Resource)
	condition.Id = sortResource.Id
	return gormx.Updates(condition, condition, updateMap)
}

func (r *resourceAppImpl) checkCode(code string) error {
	if strings.Contains(code, ",") {
		return errorx.NewBiz("code不能包含','")
	}
	if gormx.CountBy(&entity.Resource{Code: code}) != 0 {
		return errorx.NewBiz("该code已存在")
	}
	return nil
}

func (r *resourceAppImpl) Delete(ctx context.Context, id uint64) error {
	resource, err := r.GetById(new(entity.Resource), id)
	if err != nil {
		return errorx.NewBiz("资源不存在")
	}

	// 删除当前节点及其所有子节点
	children := r.GetRepo().GetChildren(resource.UiPath)
	for _, v := range children {
		r.GetRepo().DeleteById(ctx, v.Id)
		// 删除角色关联的资源信息
		gormx.DeleteBy(&entity.RoleResource{ResourceId: v.Id})
	}
	return nil
}

func (r *resourceAppImpl) GetAccountResources(accountId uint64, toEntity any) error {
	// 超级管理员返回所有
	if accountId == consts.AdminId {
		cond := &entity.Resource{
			Status: entity.ResourceStatusEnable,
		}
		return r.ListByCondOrder(cond, toEntity, "pid asc", "weight asc")
	}

	return r.GetRepo().GetAccountResources(accountId, toEntity)
}
