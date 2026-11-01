package api

import (
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/application/sqlexec"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"

	"github.com/spf13/cast"
)

// DbColumnAcl 列级访问控制的配置入口（与表级同一权限码 db:save）
type DbColumnAcl struct {
	columnAclApp sqlexec.TableAcl `inject:"T"`
}

func (d *DbColumnAcl) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("/column-acls", d.List).RequiredPermissionCode("db:save"),
		req.NewPost("/column-acls", d.Save).Log(req.NewLogSaveI(imsg.LogDbColumnAclSave)).RequiredPermissionCode("db:save"),
		req.NewPut("/column-acls", d.Save).Log(req.NewLogSaveI(imsg.LogDbColumnAclSave)).RequiredPermissionCode("db:save"),
		req.NewDelete("/column-acls/:id", d.Delete).Log(req.NewLogSaveI(imsg.LogDbColumnAclDelete)).RequiredPermissionCode("db:save"),
	}

	return req.NewConfs("/dbs", reqs[:]...)
}

// List 某库的列级规则
// @router /api/dbs/column-acls [get]
func (d *DbColumnAcl) List(rc *req.Ctx) {
	dbId := cast.ToUint64(rc.Query("dbId"))
	biz.IsTrue(dbId > 0, "dbId error")
	res, err := d.columnAclApp.ListColumns(dbId)
	biz.ErrIsNil(err)
	rc.ResData = res
}

// Save 保存列级规则（同库同表同列只保留一条）
// @router /api/dbs/column-acls [post]
func (d *DbColumnAcl) Save(rc *req.Ctx) {
	f := rc.BindJson[form.DbColumnAclForm]()
	rc.ReqParam = f

	acl := &entity.DbColumnAcl{
		DbId:    f.DbId,
		Table:   f.TableName,
		Column:  f.ColumnName,
		RoleIds: f.RoleIds,
		Remark:  f.Remark,
	}
	acl.Id = f.Id
	biz.ErrIsNil(d.columnAclApp.SaveColumn(rc.MetaCtx, acl))
}

// Delete 删除列级规则
// @router /api/dbs/column-acls/:id [delete]
func (d *DbColumnAcl) Delete(rc *req.Ctx) {
	id := cast.ToUint64(rc.PathParam("id"))
	biz.IsTrue(id > 0, "id error")
	rc.ReqParam = id
	biz.ErrIsNil(d.columnAclApp.DeleteColumn(rc.MetaCtx, id))
}
