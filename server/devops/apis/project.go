package apis

import (
	"fmt"
	"mayfly-go/base/biz"
	"mayfly-go/base/ctx"
	"mayfly-go/base/ginx"
	"mayfly-go/server/devops/apis/vo"
	"mayfly-go/server/devops/application"
	"mayfly-go/server/devops/domain/entity"
	sys_applicaiton "mayfly-go/server/sys/application"
	sys_entity "mayfly-go/server/sys/domain/entity"
)

type Project struct {
	ProjectApp application.Project
	AccountApp sys_applicaiton.Account
}

// 获取当前登录用户可以访问的项目列表
func (p *Project) GetProjectsByLoginAccount(rc *ctx.ReqCtx) {
	// 获取登录用户拥有的项目ids
	projectMembers := &[]entity.ProjectMember{}
	p.ProjectApp.ListMember(&entity.ProjectMember{AccountId: rc.LoginAccount.Id}, projectMembers)
	var pids []uint64
	for _, pm := range *projectMembers {
		pids = append(pids, pm.ProjectId)
	}

	// 获取项目信息
	projects := &vo.AccountProjects{}
	p.ProjectApp.ListProjectByIds(pids, projects)
	rc.ResData = projects
}

func (p *Project) GetProjects(rc *ctx.ReqCtx) {
	condition := &entity.Project{}
	ginx.BindQuery(rc.GinCtx, condition)
	// condition.Name = rc.GinCtx.Query("name")
	rc.ResData = p.ProjectApp.GetPageList(condition, ginx.GetPageParam(rc.GinCtx), new([]entity.Project))
}

func (p *Project) SaveProject(rc *ctx.ReqCtx) {
	project := &entity.Project{}
	ginx.BindJsonAndValid(rc.GinCtx, project)
	rc.ReqParam = project

	project.SetBaseInfo(rc.LoginAccount)
	p.ProjectApp.SaveProject(project)
}

func (p *Project) DelProject(rc *ctx.ReqCtx) {
	p.ProjectApp.DelProject(uint64(ginx.QueryInt(rc.GinCtx, "id", 0)))
}

// 获取项目下的环境信息
func (p *Project) GetProjectEnvs(rc *ctx.ReqCtx) {
	projectEnvs := &[]entity.ProjectEnv{}
	p.ProjectApp.ListEnvByProjectId(uint64(ginx.PathParamInt(rc.GinCtx, "projectId")), projectEnvs)
	rc.ResData = projectEnvs
}

//保存项目下的环境信息
func (p *Project) SaveProjectEnvs(rc *ctx.ReqCtx) {
	projectEnv := &entity.ProjectEnv{}
	ginx.BindJsonAndValid(rc.GinCtx, projectEnv)
	rc.ReqParam = projectEnv

	projectEnv.SetBaseInfo(rc.LoginAccount)
	p.ProjectApp.SaveProjectEnv(projectEnv)
}

// 获取项目下的成员信息
func (p *Project) GetProjectMembers(rc *ctx.ReqCtx) {
	projectMems := &[]entity.ProjectMember{}
	rc.ResData = p.ProjectApp.GetMemberPage(&entity.ProjectMember{ProjectId: uint64(ginx.PathParamInt(rc.GinCtx, "projectId"))},
		ginx.GetPageParam(rc.GinCtx), projectMems)
}

//保存项目的成员信息
func (p *Project) SaveProjectMember(rc *ctx.ReqCtx) {
	projectMem := &entity.ProjectMember{}
	ginx.BindJsonAndValid(rc.GinCtx, projectMem)
	rc.ReqParam = projectMem

	// 校验账号，并赋值username
	account := &sys_entity.Account{}
	account.Id = projectMem.AccountId
	biz.ErrIsNil(p.AccountApp.GetAccount(account, "Id", "Username"), "账号不存在")
	projectMem.Username = account.Username

	projectMem.SetBaseInfo(rc.LoginAccount)
	p.ProjectApp.SaveProjectMember(projectMem)
}

//删除项目成员
func (p *Project) DelProjectMember(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	pid := ginx.PathParamInt(g, "projectId")
	aid := ginx.PathParamInt(g, "accountId")
	rc.ReqParam = fmt.Sprintf("projectId: %d, accountId: %d", pid, aid)

	p.ProjectApp.DeleteMember(uint64(pid), uint64(aid))
}
