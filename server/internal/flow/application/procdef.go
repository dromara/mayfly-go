package application

import (
	"context"
	"mayfly-go/internal/flow/application/dto"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/internal/flow/imsg"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
)

type Procdef interface {
	base.App[*entity.Procdef]

	GetPageList(condition *entity.Procdef, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	// 保存流程实例信息
	SaveProcdef(ctx context.Context, def *dto.SaveProcdef) error

	// 删除流程实例信息
	DeleteProcdef(ctx context.Context, defId uint64) error

	// GetProcdefByCodePath 根据资源编号路径获取对应的流程定义
	GetProcdefByCodePath(ctx context.Context, codePaths ...string) *entity.Procdef

	// GetProcdefByResource 根据资源获取对应的流程定义
	GetProcdefByResource(ctx context.Context, resourceType int8, resourceCode string) *entity.Procdef
}

type procdefAppImpl struct {
	base.AppImpl[*entity.Procdef, repository.Procdef]

	procinstApp Procinst `inject:"ProcinstApp"`

	tagTreeApp       tagapp.TagTree       `inject:"TagTreeApp"`
	tagTreeRelateApp tagapp.TagTreeRelate `inject:"TagTreeRelateApp"`
}

var _ (Procdef) = (*procdefAppImpl)(nil)

// 注入repo
func (p *procdefAppImpl) InjectProcdefRepo(procdefRepo repository.Procdef) {
	p.Repo = procdefRepo
}

func (p *procdefAppImpl) GetPageList(condition *entity.Procdef, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return p.Repo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

func (p *procdefAppImpl) SaveProcdef(ctx context.Context, defParam *dto.SaveProcdef) error {
	def := defParam.Procdef
	if err := entity.ProcdefStatusEnum.Valid(def.Status); err != nil {
		return err
	}

	if def.Id == 0 {
		if p.GetByCond(&entity.Procdef{DefKey: def.DefKey}) == nil {
			return errorx.NewBizI(ctx, imsg.ErrProcdefKeyExist)
		}
	} else {
		// 防止误修改key
		def.DefKey = ""
		if err := p.canModify(ctx, def.Id); err != nil {
			return err
		}
	}

	return p.Tx(ctx, func(ctx context.Context) error {
		return p.Save(ctx, def)
	}, func(ctx context.Context) error {
		return p.tagTreeRelateApp.RelateTag(ctx, tagentity.TagRelateTypeFlowDef, def.Id, defParam.CodePaths...)
	})
}

func (p *procdefAppImpl) DeleteProcdef(ctx context.Context, defId uint64) error {
	if err := p.canModify(ctx, defId); err != nil {
		return err
	}
	return p.DeleteById(ctx, defId)
}

func (p *procdefAppImpl) GetProcdefByCodePath(ctx context.Context, codePaths ...string) *entity.Procdef {
	relateIds, err := p.tagTreeRelateApp.GetRelateIds(ctx, tagentity.TagRelateTypeFlowDef, codePaths...)
	if err != nil || len(relateIds) == 0 {
		return nil
	}

	procdefId := relateIds[len(relateIds)-1]
	procdef, err := p.GetById(procdefId)
	if err != nil {
		return nil
	}
	if procdef.Status == entity.ProcdefStatusDisable {
		return nil
	}
	return procdef
}

func (p *procdefAppImpl) GetProcdefByResource(ctx context.Context, resourceType int8, resourceCode string) *entity.Procdef {
	resourceCodePaths := p.tagTreeApp.ListTagPathByTypeAndCode(resourceType, resourceCode)
	return p.GetProcdefByCodePath(ctx, resourceCodePaths...)
}

// 判断该流程实例是否可以执行修改操作
func (p *procdefAppImpl) canModify(ctx context.Context, prodefId uint64) error {
	if activeInstCount := p.procinstApp.CountByCond(&entity.Procinst{ProcdefId: prodefId, Status: entity.ProcinstStatusActive}); activeInstCount > 0 {
		return errorx.NewBizI(ctx, imsg.ErrExistProcinstRunning)
	}
	if suspInstCount := p.procinstApp.CountByCond(&entity.Procinst{ProcdefId: prodefId, Status: entity.ProcinstStatusSuspended}); suspInstCount > 0 {
		return errorx.NewBizI(ctx, imsg.ErrExistProcinstSuspended)
	}
	return nil
}
