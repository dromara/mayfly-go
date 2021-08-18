package persistence

import (
	"mayfly-go/base/biz"
	"mayfly-go/base/model"
	"mayfly-go/server/devops/domain/entity"
	"mayfly-go/server/devops/domain/repository"
)

type projectMemeberRepo struct{}

var ProjectMemberRepo repository.ProjectMemeber = &projectMemeberRepo{}

func (p *projectMemeberRepo) ListMemeber(condition *entity.ProjectMember, toEntity interface{}, orderBy ...string) {
	model.ListByOrder(condition, toEntity, orderBy...)
}

func (p *projectMemeberRepo) Save(pm *entity.ProjectMember) {
	biz.ErrIsNilAppendErr(model.Insert(pm), "保存项目成员失败：%s")
}

func (p *projectMemeberRepo) GetPageList(condition *entity.ProjectMember, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return model.GetPage(pageParam, condition, toEntity, orderBy...)
}

func (p *projectMemeberRepo) DeleteByPidMid(projectId, accountId uint64) {
	model.DeleteByCondition(&entity.ProjectMember{ProjectId: projectId, AccountId: accountId})
}

func (p *projectMemeberRepo) DeleteMems(projectId uint64) {
	model.DeleteByCondition(&entity.ProjectMember{ProjectId: projectId})
}
