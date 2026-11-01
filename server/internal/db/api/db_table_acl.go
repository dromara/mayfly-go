package api

import (
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/application/sqlexec"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/imsg"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"

	"github.com/spf13/cast"
)

// DbTableAcl 表访问控制的配置入口。
//
// 权限码沿用 db:save：这是「编辑某个库的配置」而不是一项独立业务能力，
// 另造代码会让同一件事在角色配置页里分成两处勾选。
type DbTableAcl struct {
	tableAclApp sqlexec.TableAcl      `inject:"T"`
	dbApp       application.Db        `inject:"T"`
	tagApp      tagapp.TagTreeService `inject:"T"`
}

// assertDbAccess 配置入口的资源可达校验。
//
// 权限码只说明「这个账号能改配置」，不说明「能改哪个库的配置」：同目录的其它接口
// 都是「权限码 + 资源可达」双轨。缺了这一步，任何持 db:save 的团队账号都能列举
// 他库的保护名单、删他库规则，甚至写一条 allow 规则把他库直接锁成白名单模式。
func (d *DbTableAcl) assertDbAccess(rc *req.Ctx, dbId uint64) {
	biz.IsTrue(dbId > 0, "dbId error")
	db, err := d.dbApp.GetById(dbId)
	biz.ErrIsNil(err)
	biz.IsTrue(db != nil && db.Code != "", "db not found")
	biz.ErrIsNilAppendErr(d.tagApp.CanAccessByCode(rc.MetaCtx, int8(tagentity.TagTypeDb), db.Code), "%s")
}

func (d *DbTableAcl) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 某库的保护名单
		req.NewGet("/table-acls", d.List).RequiredPermissionCode("db:save"),
		req.NewPost("/table-acls", d.Save).Log(req.NewLogSaveI(imsg.LogDbTableAclSave)).RequiredPermissionCode("db:save"),
		req.NewPut("/table-acls", d.Save).Log(req.NewLogSaveI(imsg.LogDbTableAclSave)).RequiredPermissionCode("db:save"),
		req.NewDelete("/table-acls/:id", d.Delete).Log(req.NewLogSaveI(imsg.LogDbTableAclDelete)).RequiredPermissionCode("db:save"),
	}

	return req.NewConfs("/dbs", reqs[:]...)
}

// List 某库的表访问控制规则
// @router /api/dbs/table-acls [get]
func (d *DbTableAcl) List(rc *req.Ctx) {
	dbId := cast.ToUint64(rc.Query("dbId"))
	d.assertDbAccess(rc, dbId)
	res, err := d.tableAclApp.ListByDb(dbId)
	biz.ErrIsNil(err)
	rc.ResData = res
}

// Save 保存表访问控制规则（同一库同一张表只保留一条）
// @router /api/dbs/table-acls [post]
func (d *DbTableAcl) Save(rc *req.Ctx) {
	f := rc.BindJson[form.DbTableAclForm]()
	rc.ReqParam = f

	d.assertDbAccess(rc, f.DbId)
	if f.Id > 0 {
		// 规则归属必须与请求里的 dbId 一致：否则可借 id 直接改到他库的规则上
		existing, err := d.tableAclApp.GetById(f.Id)
		biz.ErrIsNil(err)
		biz.IsTrue(existing != nil && existing.DbId == f.DbId, "acl rule does not belong to this db")
	}

	acl := &entity.DbTableAcl{
		DbId:    f.DbId,
		Table:   f.TableName,
		Effect:  f.Effect,
		RoleIds: f.RoleIds,
		Remark:  f.Remark,
	}
	acl.Id = f.Id
	biz.ErrIsNil(d.tableAclApp.SaveAcl(rc.MetaCtx, acl))
}

// Delete 删除表访问控制规则
// @router /api/dbs/table-acls/:id [delete]
func (d *DbTableAcl) Delete(rc *req.Ctx) {
	id := cast.ToUint64(rc.PathParam("id"))
	biz.IsTrue(id > 0, "id error")
	existing, err := d.tableAclApp.GetById(id)
	biz.ErrIsNil(err)
	biz.IsTrue(existing != nil, "acl rule not found")
	// 按规则自己的库校验：不能靠请求参数指定一个「我有权的库」来删他库规则
	d.assertDbAccess(rc, existing.DbId)
	rc.ReqParam = id
	biz.ErrIsNil(d.tableAclApp.DeleteAcl(rc.MetaCtx, id))
}
