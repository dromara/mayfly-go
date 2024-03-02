package application

import (
	"context"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
)

type Procdef interface {
	base.App[*entity.Procdef]

	GetPageList(condition *entity.Procdef, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	// 保存流程实例信息
	Save(ctx context.Context, def *entity.Procdef) error

	// 删除流程实例信息
	DeleteProcdef(ctx context.Context, defId uint64) error
}

type procdefAppImpl struct {
	base.AppImpl[*entity.Procdef, repository.Procdef]

	procinstApp Procinst `inject:"ProcinstApp"`
}

// 注入repo
func (p *procdefAppImpl) InjectProcdefRepo(procdefRepo repository.Procdef) {
	p.Repo = procdefRepo
}

func (p *procdefAppImpl) GetPageList(condition *entity.Procdef, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return p.Repo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

func (p *procdefAppImpl) Save(ctx context.Context, def *entity.Procdef) error {
	if err := entity.ProcdefStatusEnum.Valid(def.Status); err != nil {
		return err
	}
	if def.Id == 0 {
		if p.GetBy(&entity.Procdef{DefKey: def.DefKey}) == nil {
			return errorx.NewBiz("该流程实例key已存在")
		}
		return p.Insert(ctx, def)
	}

	// 防止误修改key
	def.DefKey = ""
	if err := p.canModify(def.Id); err != nil {
		return err
	}

	return p.UpdateById(ctx, def)
}

func (p *procdefAppImpl) DeleteProcdef(ctx context.Context, defId uint64) error {
	if err := p.canModify(defId); err != nil {
		return err
	}
	return p.DeleteById(ctx, defId)
}

// 判断该流程实例是否可以执行修改操作
func (p *procdefAppImpl) canModify(prodefId uint64) error {
	if activeInstCount := p.procinstApp.CountByCond(&entity.Procinst{ProcdefId: prodefId, Status: entity.ProcinstStatusActive}); activeInstCount > 0 {
		return errorx.NewBiz("存在运行中的流程实例，无法操作")
	}
	if suspInstCount := p.procinstApp.CountByCond(&entity.Procinst{ProcdefId: prodefId, Status: entity.ProcinstStatusSuspended}); suspInstCount > 0 {
		return errorx.NewBiz("存在挂起中的流程实例，无法操作")
	}
	return nil
}
