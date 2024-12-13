package application

import (
	"context"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"time"
)

type ResourceOpLog interface {
	base.App[*entity.ResourceOpLog]

	// AddResourceOpLog 新增资源操作记录
	AddResourceOpLog(ctx context.Context, codePath string) error
}

type resourceOpLogAppImpl struct {
	base.AppImpl[*entity.ResourceOpLog, repository.ResourceOpLog]

	tagTreeApp TagTree `inject:"TagTreeApp"`
}

var _ (ResourceOpLog) = (*resourceOpLogAppImpl)(nil)

func (rol *resourceOpLogAppImpl) AddResourceOpLog(ctx context.Context, codePath string) error {
	loginAccount := contextx.GetLoginAccount(ctx)
	if loginAccount == nil {
		return errorx.NewBiz("Login information does not exist in this context")
	}

	var logs []*entity.ResourceOpLog
	qc := model.NewCond().Ge("create_time", time.Now().Add(-5*time.Minute)).Eq("creator_id", loginAccount.Id).Eq("code_path", codePath)
	logs, err := rol.ListByCond(qc)
	if err != nil {
		return err
	}
	// 指定时间内多次操作则不记录
	if len(logs) > 0 {
		return nil
	}
	tagTree := &entity.TagTree{CodePath: codePath}
	if err := rol.tagTreeApp.GetByCond(tagTree); err != nil {
		return errorx.NewBiz("tag resource not found")
	}

	resourceType := tagTree.Type
	// 获取第一段资源类型即可
	pathSections := entity.CodePath(tagTree.CodePath).GetPathSections()
	for _, ps := range pathSections {
		if ps.Type == entity.TagTypeTag {
			continue
		}
		resourceType = ps.Type
		break
	}

	return rol.Save(ctx, &entity.ResourceOpLog{
		ResourceCode: tagTree.Code,
		ResourceType: int8(resourceType),
		CodePath:     tagTree.CodePath,
	})
}
