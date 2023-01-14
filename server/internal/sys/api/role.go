package api

import (
	"mayfly-go/internal/sys/api/form"
	"mayfly-go/internal/sys/api/vo"
	"mayfly-go/internal/sys/application"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils"
	"strconv"
	"strings"
	"time"
)

type Role struct {
	RoleApp     application.Role
	ResourceApp application.Resource
}

func (r *Role) Roles(rc *req.Ctx) {
	g := rc.GinCtx
	condition := &entity.Role{}
	rc.ResData = r.RoleApp.GetPageList(condition, ginx.GetPageParam(g), new([]entity.Role))
}

// 保存角色信息
func (r *Role) SaveRole(rc *req.Ctx) {
	g := rc.GinCtx
	form := &form.RoleForm{}
	ginx.BindJsonAndValid(g, form)
	rc.ReqParam = form

	role := new(entity.Role)
	utils.Copy(role, form)
	role.SetBaseInfo(rc.LoginAccount)

	r.RoleApp.SaveRole(role)
}

// 删除角色及其资源关联关系
func (r *Role) DelRole(rc *req.Ctx) {
	r.RoleApp.DeleteRole(uint64(ginx.PathParamInt(rc.GinCtx, "id")))
}

// 获取角色关联的资源id数组，用于分配资源时回显已拥有的资源
func (r *Role) RoleResourceIds(rc *req.Ctx) {
	rc.ResData = r.RoleApp.GetRoleResourceIds(uint64(ginx.PathParamInt(rc.GinCtx, "id")))
}

// 查看角色关联的资源树信息
func (r *Role) RoleResource(rc *req.Ctx) {
	g := rc.GinCtx

	var resources vo.ResourceManageVOList
	r.RoleApp.GetRoleResources(uint64(ginx.PathParamInt(g, "id")), &resources)

	rc.ResData = resources.ToTrees(0)
}

// 保存角色资源
func (r *Role) SaveResource(rc *req.Ctx) {
	g := rc.GinCtx

	var form form.RoleResourceForm
	ginx.BindJsonAndValid(g, &form)
	rid := uint64(form.Id)
	rc.ReqParam = form

	// 将,拼接的字符串进行切割
	idsStr := strings.Split(form.ResourceIds, ",")
	var newIds []interface{}
	for _, v := range idsStr {
		id, _ := strconv.Atoi(v)
		newIds = append(newIds, uint64(id))
	}

	// 将[]uint64转为[]interface{}
	oIds := r.RoleApp.GetRoleResourceIds(uint64(form.Id))
	var oldIds []interface{}
	for _, v := range oIds {
		oldIds = append(oldIds, v)
	}

	addIds, delIds, _ := utils.ArrayCompare(newIds, oldIds, func(i1, i2 interface{}) bool {
		return i1.(uint64) == i2.(uint64)
	})

	createTime := time.Now()
	creator := rc.LoginAccount.Username
	creatorId := rc.LoginAccount.Id
	for _, v := range addIds {
		rr := &entity.RoleResource{RoleId: rid, ResourceId: v.(uint64), CreateTime: &createTime, CreatorId: creatorId, Creator: creator}
		r.RoleApp.SaveRoleResource(rr)
	}
	for _, v := range delIds {
		r.RoleApp.DeleteRoleResource(rid, v.(uint64))
	}
}
