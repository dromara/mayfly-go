package api

import (
	"mayfly-go/internal/sys/api/form"
	"mayfly-go/internal/sys/api/vo"
	"mayfly-go/internal/sys/application"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"strings"

	"github.com/may-fly/cast"
)

type Role struct {
	roleApp application.Role `inject:"T"`
}

func (r *Role) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("", r.Roles),

		req.NewPost("", r.SaveRole).Log(req.NewLogSaveI(imsg.LogRoleSave)).RequiredPermissionCode("role:add"),

		req.NewDelete(":id", r.DelRole).Log(req.NewLogSaveI(imsg.LogRoleDelete)).RequiredPermissionCode("role:del"),

		req.NewGet(":id/resourceIds", r.RoleResourceIds),

		req.NewGet(":id/resources", r.RoleResource),

		req.NewPost(":id/resources", r.SaveResource).Log(req.NewLogSaveI(imsg.LogAssignRoleResource)).RequiredPermissionCode("role:saveResources"),

		req.NewGet(":id/accounts", r.RoleAccount),
	}

	return req.NewConfs("sys/roles", reqs[:]...)
}

func (r *Role) Roles(rc *req.Ctx) {
	cond, pageParam := req.BindQueryAndPage(rc, new(entity.RoleQuery))

	notIdsStr := rc.Query("notIds")
	if notIdsStr != "" {
		cond.NotIds = collx.ArrayMap[string, uint64](strings.Split(notIdsStr, ","), func(val string) uint64 {
			return cast.ToUint64(val)
		})
	}

	res, err := r.roleApp.GetPageList(cond, pageParam, new([]entity.Role))
	biz.ErrIsNil(err)
	rc.ResData = res
}

// 保存角色信息
func (r *Role) SaveRole(rc *req.Ctx) {
	form := &form.RoleForm{}
	role := req.BindJsonAndCopyTo(rc, form, new(entity.Role))
	rc.ReqParam = form

	r.roleApp.SaveRole(rc.MetaCtx, role)
}

// 删除角色及其资源关联关系
func (r *Role) DelRole(rc *req.Ctx) {
	idsStr := rc.PathParam("id")
	rc.ReqParam = collx.Kvs("ids", idsStr)
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		biz.ErrIsNil(r.roleApp.DeleteRole(rc.MetaCtx, cast.ToUint64(v)))
	}
}

// 获取角色关联的资源id数组，用于分配资源时回显已拥有的资源
func (r *Role) RoleResourceIds(rc *req.Ctx) {
	rc.ResData = r.roleApp.GetRoleResourceIds(uint64(rc.PathParamInt("id")))
}

// 查看角色关联的资源树信息
func (r *Role) RoleResource(rc *req.Ctx) {
	var resources vo.ResourceManageVOList
	r.roleApp.GetRoleResources(uint64(rc.PathParamInt("id")), &resources)
	rc.ResData = resources.ToTrees(0)
}

// 保存角色资源
func (r *Role) SaveResource(rc *req.Ctx) {
	var form form.RoleResourceForm
	req.BindJsonAndValid(rc, &form)
	rc.ReqParam = form

	// 将,拼接的字符串进行切割并转换
	newIds := collx.ArrayMap[string, uint64](strings.Split(form.ResourceIds, ","), func(val string) uint64 {
		return cast.ToUint64(val)
	})

	biz.ErrIsNilAppendErr(r.roleApp.SaveRoleResource(rc.MetaCtx, form.Id, newIds), "save role resource failed: %s")
}

// 查看角色关联的用户
func (r *Role) RoleAccount(rc *req.Ctx) {
	cond, pageParam := req.BindQueryAndPage[*entity.RoleAccountQuery](rc, new(entity.RoleAccountQuery))
	cond.RoleId = uint64(rc.PathParamInt("id"))
	var accounts []*vo.AccountRoleVO
	res, err := r.roleApp.GetRoleAccountPage(cond, pageParam, &accounts)
	biz.ErrIsNil(err)
	rc.ResData = res
}
