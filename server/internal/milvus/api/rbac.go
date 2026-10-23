package api

import (
	"mayfly-go/internal/milvus/api/form"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
)

// builtinPrivilegeGroups 内置权限组，不可修改或删除
var builtinPrivilegeGroups = map[string]struct{}{
	"ClusterReadOnly":     {},
	"ClusterReadWrite":    {},
	"ClusterAdmin":        {},
	"DatabaseReadOnly":    {},
	"DatabaseReadWrite":   {},
	"DatabaseAdmin":       {},
	"CollectionReadOnly":  {},
	"CollectionReadWrite": {},
	"CollectionAdmin":     {},
}

func isBuiltinPrivilegeGroup(name string) bool {
	_, ok := builtinPrivilegeGroups[name]
	return ok
}

// ============ 用户权限 ============

func (c *Collection) ListRoles(rc *req.Ctx) {
	client := c.getClient(rc)
	roles, err := client.ListRoles()
	biz.ErrIsNil(err)
	rc.ResData = roles
}

func (c *Collection) UpdateRole(rc *req.Ctx) {
	client := c.getClient(rc)
	param := rc.BindJson[form.UpdateRoleForm]()
	err := client.UpdateRole(param)
	biz.ErrIsNil(err)
}

func (c *Collection) DropRole(rc *req.Ctx) {
	client := c.getClient(rc)
	role := rc.PathParam("role")
	err := client.DropRole(role)
	biz.ErrIsNil(err)
}
func (c *Collection) SelectRole(rc *req.Ctx) {
	client := c.getClient(rc)
	role := rc.PathParam("role")
	var err error
	rc.ResData, err = client.SelectRole(role)
	biz.ErrIsNil(err)
}
func (c *Collection) PrivilegeGroup(rc *req.Ctx) {
	client := c.getClient(rc)
	var err error
	rc.ResData, err = client.GetPrivilegeGroup()
	biz.ErrIsNil(err)
}

func (c *Collection) SavePrivilegeGroup(rc *req.Ctx) {
	client := c.getClient(rc)
	param := rc.BindJson[form.SavePrivilegeGroupForm]()
	biz.IsTrue(!isBuiltinPrivilegeGroup(param.GroupName), "builtin privilege group is read-only")

	// 检查权限组是否已存在
	groups, err := client.GetPrivilegeGroup()
	biz.ErrIsNil(err)

	exists := false
	for _, g := range groups {
		if g.GroupName == param.GroupName {
			exists = true
			break
		}
	}

	if exists {
		err = client.UpdatePrivilegeGroup(param.GroupName, param.Privileges)
	} else {
		err = client.CreatePrivilegeGroup(param.GroupName, param.Privileges)
	}
	biz.ErrIsNil(err)
}

func (c *Collection) DropPrivilegeGroup(rc *req.Ctx) {
	client := c.getClient(rc)
	name := rc.PathParam("name")
	biz.IsTrue(!isBuiltinPrivilegeGroup(name), "builtin privilege group cannot be dropped")
	err := client.DropPrivilegeGroup(name)
	biz.ErrIsNil(err)
}
