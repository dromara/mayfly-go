package api

import (
	"mayfly-go/internal/milvus/api/form"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
)

func (c *Collection) ListUsers(rc *req.Ctx) {
	client := c.getClient(rc)
	users, err := client.ListUsers()
	biz.ErrIsNil(err)
	rc.ResData = users
}

func (c *Collection) CreateUser(rc *req.Ctx) {
	client := c.getClient(rc)
	param := rc.BindJson[form.CreateUserForm]()
	err := client.CreateUser(param.Username, param.Password)
	biz.ErrIsNil(err)
}

func (c *Collection) DeleteUser(rc *req.Ctx) {
	client := c.getClient(rc)
	username := rc.PathParam("username")
	err := client.DeleteUser(username)
	biz.ErrIsNil(err)
}

func (c *Collection) UpdatePassword(rc *req.Ctx) {
	client := c.getClient(rc)
	param := rc.BindJson[form.UpdatePasswordForm]()
	username := rc.PathParam("username")
	err := client.UpdatePassword(username, param.OldPassword, param.NewPassword)
	biz.ErrIsNil(err)
}
func (c *Collection) GrantRoleToUser(rc *req.Ctx) {
	client := c.getClient(rc)
	param := rc.BindJson[form.RoleToUserForm]()
	username := rc.PathParam("username")
	err := client.GrantRoleToUser(username, param.RoleName)
	biz.ErrIsNil(err)
}

func (c *Collection) RevokeRoleFromUser(rc *req.Ctx) {
	client := c.getClient(rc)
	param := rc.BindJson[form.RoleToUserForm]()
	username := rc.PathParam("username")
	err := client.RevokeRoleFromUser(username, param.RoleName)
	biz.ErrIsNil(err)
}
